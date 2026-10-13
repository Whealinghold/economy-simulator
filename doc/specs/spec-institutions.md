# Spec: Institutions: bodies, exclusive rights and legal prices

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-ORG-NNN` and
`NFR-ORG-NNN`, which tests cite (testing.md section 4). They are in section 17.

Layer: module, later stage ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md),
[ADR 0003](../adr/0003-version-1-scope.md)). It is off unless `Config.Modules.Institutions`
is present, and the core does not depend on it. A world with the module off, or with the
module on and no body, right or control declared, is exactly the world without it
(FR-ORG-001). Needs `spec-polity.md` (POL: `Chain`, `Regions`, the combining mode
**innermost**, hooks H5 and H6), `spec-skill-barriers.md` (SKL: the traineeship, its
places and its completion), `spec-settlements.md` (SET: the flag and the limit, hooks H4
and H6, the rules R1 to R7 of its 6.2), `spec-clearing.md` (CLR 2.3, 3.2: admission is a
rule of the market, the same for every `Clearing`), `spec-production.md` (PRO 4.2, the
refusal `not_on_role`), `spec-agents.md` (AGT 6.5: what the world checks), `spec-spawner.md`
and `spec-world.md` (6, 7, 8, 9.4: the tick, the draws, the slots, no rate from an
aggregate). It reads `spec-credit.md` (CRD 5.2, 13: the rate a borrower accepts) and
`spec-merchants.md` (MER 9.3, 9.4: a delivery contract's price). It is read by the
specs that follow it: fiscal (FIS), policy (PLC) and, for what a ruler may change in a run,
integration (INT).

**Every number in sections 9, 13 and 14 is a placeholder** except where a derivation is
shown. Nothing has been run against the library, bazaarBot has not been run, and no
acceptance band exists (notes N-015, N-027). `models/org/` (section 15) checks that a
clamp of one side's limit keeps every trade price inside a legal range and that the
membership round is order-independent and idempotent; it measures no speed and is not a
claim that any default is stable. No default of this spec is a macro constant: the module
sets no price, wage, rate or share of its own, and a legal price in a pack is a pack's
number.

Terms in **bold** on first use are new domain terms. They must be added to the glossary in
`doc.go` before any code uses them (architecture.md 3.2). "Consumer" means the library
consumer, never a household (N-001, N-017). The setting words of the examples (`guild`,
`charter`, `monopoly`, `assize`, `usury`, `masters`, `apprentice`, `station`, `licence`)
appear only in the **flagged examples** of section 13 and in sentences marked "for
example"; they are data, never branched on (architecture.md 9.1; `AGENTS.md`,
setting-agnostic). No name of a type or a field is a setting's word.

## 1. Purpose and scope

A market in this library is open: any household of a region whose role can trade a
commodity may bid or ask for it at any limit price its beliefs give, and any household
whose role lists a recipe may run it. Several settings restrict that: a craft that only
members of an association may practise, a market in which only members of a body may
sell one commodity, a ceiling on the price of a staple, a law against interest above a rate. Three
written specs say what these are not and leave them to this one: `spec-skill-barriers.md`
7 (association rules beyond the trainee limit), `spec-polity.md` 6.4.1 and H5 and H6 (a
law is a row of the spec that owns the thing it governs; an exclusive right and a price control
are ORG's), and `spec-settlements.md` 6.5 (which associations a region may have, whether
it may hold a reserved market). This spec fixes three kinds of data and what the world does
with them. It adds no behaviour of its own that a household reads. It covers:

- the **body**: a named set of households, selected by role and region, with a limit on
  how many (its **places**), a limit it puts on its members' trainees, and an optional
  settlement flag it needs (section 3);
- the **membership round**: how households come to be members and cease to be, by
  seniority, with no draw (section 4);
- the **right**: an exclusive right to one **activity** on one subject in one scope, held
  by a body or by nobody, and the one refusal reason `not_permitted` that it causes in
  production, in the market and in the traineeship (section 5);
- the **legal price**: a cap and a floor on the price of a commodity, and a cap on the
  yearly rate of credit, as a clamp of the limit of the side each bounds, applied in the
  admission of the market and of the credit round (section 6);
- the two **slots** by which a ruler or a consumer changes a legal level or suspends a
  right in a run (section 7);
- where in the tick the module acts, its seams, parameters, validation, observability,
  state, loops and requirements (sections 7 to 17).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The jurisdiction tree, the chain of a region, the crossing of an edge | `spec-polity.md` (POL) |
| What a tax, a toll, a tariff, a fee or dues **are**; a treasury's money | `spec-fiscal.md` (FIS), with `spec-money-supply.md` (MON) for the flow mechanism |
| How a ruler or a player chooses a legal level, or grants and withdraws a right, over time | `spec-policy.md` (PLC): its `legal` lever kind sets the levels of 7 (6.3.2 there); a suspension is a consumer's call (`SetRightSuspensions`) |
| Tools, training, the premium, the trainee limit of a role | `spec-skill-barriers.md` (SKL): a body adds a stricter limit and a gate, never a second mechanism |
| Tenure of an endowment and a rent control | `spec-land.md` (LND): its `Closed` rows and its rent scale are LND's; a rent control is LND's slot (6.8) |
| A minimum wage, a wage cap, a closed shop in hiring | a closed shop is the field `Who` of a right (5.3); a wage control is reserved with its form (6.8) |
| A body's treasury, dues, a charitable fund, an armed force | not in version 1: a body has no account (position 2, open question 1) |
| Loss, banditry, a border that is fought over, a smuggler | `spec-risk.md` (RSK): a law that is evaded is RSK's, and no trade is evaded here (6.5) |
| A store, a lease, who may keep goods for others | `spec-storage.md` (STO): reserved (5.8) |
| A facility and who may use it | `spec-infrastructure.md` (CAP): a right over the recipe that requires it works without CAP knowing (5.8) |
| How a household reacts to a refusal or a cap | the strategy (`spec-agents.md` 7) and beliefs (`spec-beliefs.md`): this spec changes what a household may do, never how it decides |
| Derived measures over bodies and legal prices | `spec-indicators.md` (IND): two preset asks (section 11.7) |
| The tick, the draws, the options | `spec-world.md` (WLD) |

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What does the module add? | Three data objects: a **body**, a **right**, a **control** (a legal price, goods or credit). Nothing else: no law table, no court, no enforcement agent | POL 6.4.1: a law is a row of the spec that owns the thing it governs, so this spec owns the rows of *rights* and *legal prices* and nothing about taxes, wages, rents or loans beyond its own bounds |
| 2 | Is a body an agent or an account? | **No.** It has no account, no agent ID, no pseudo-agent, no strategy and no money of its own. It is a key that selects households, with a limit | An agent would shift every household's ID (N-117, POL-A1) and need a rule for what it spends (FIS's). A body with dues is open question 1 |
| 3 | What is a member? | A household agent whose **role** is on the body's list, whose **region** is in its scope, and which the membership round has admitted **in the region it is in**. A household is a member of at most one body, in one region | A (role, region) is in at most one body (3.3), so the body is a function of the agent; the saved field of 12 records who was admitted and where, so a move lapses the membership whatever the scope (3.6) |
| 4 | How are members chosen? | By **seniority**: the household that has held its role the longest, then a stateless key of (tick, agent), then the agent ID; a household whole or not at all; no draw | Deterministic, no state beyond `RoleSince`, and "never the lower ID" (WLD 10) is kept: the tie is the key. Seniority is a stated rule, so its unfairness to arrivals is a registered hazard (ORG-A5) and not a hidden one |
| 5 | What is `Places`? | A **limit** in SET's sense (6.1 there): a count of households that the body's roles may hold in each region of its scope. It feeds the same headroom that the spawner and the population module read; it is not enforced on what is there | One mechanism of one kind (SET R2, R3). A body's places and a class's ceiling are two limits on one headroom, the least of them |
| 6 | What does a right do? | It reserves one **activity** (produce a recipe, sell a commodity, buy a commodity, train for a role) in a **scope** (a region or a jurisdiction) to the members of the holder body, or to nobody (a **prohibition**) | One mechanism with different data (the scope, the places): what a setting calls, for example, a charter, a monopoly or a licence is a row of it. The setting's word is a label |
| 7 | How does a refusal look? | One reason, `not_permitted`, an exported constant of the root package. In production it is checked after `not_on_role` and before anything else (as FR-PRO-048 puts the role check first); in the market it is checked beside `not_tradable`, by the world, while it builds the orders | A legal bar comes before a physical one, so nothing is reserved or consumed for a start that was never allowed; a refused order never reaches `Clearing` (CLR 2.3 untouched) |
| 8 | A price control: a clamp, a rejection or a strategy rule? | **A clamp**, in the admission of the market: a bid's limit is lowered to a cap, an ask's raised to a floor, after the checks of price, quantity, duplicate and overflow and before those of funds and stock; every order of the market, whoever places it. No new rejection reason; no rule in the strategy | A trade price lies between the two limits (CLR 5.1), so bounding one side's limit bounds the price (6.3); a rejection would remove a buyer who is willing to pay the legal price; the strategy is consumer code and cannot be trusted with a law (6.4) |
| 9 | Credit? | **The same clamp**, on a borrower's `MaxRateBP` in the admission of the credit round; a lender whose rate is above the cap finds no borrower | A rate limit is a cap on a price formed from two limits, and CRD's round is that shape (CRD 6.3.2) |
| 10 | Can the law change in a run? | **Two slots**, set between ticks: the cap and floor of a declared control, and the suspension of a declared right. Nothing else is a slot; membership is never set by a call | PLC's `legal` levers set the levels (ADR 0003 decision 3: it returns settings); a suspension is a consumer's call, PLC has no kind for it; a right granted to a named household is a policy about a household, which is a call, and no consumer asks (N-131) |
| 11 | Does the module draw or add a position of its own? | **No draw.** Two positions: the membership round at P1.9a, the aggregate events at P7.2b | A round that draws would make the stream depend on a module's switch (WLD 7.4.2); the tie key is a pure function (RES 5.3.3) |
| 12 | Is a legal price a rate from an aggregate? | **No.** A cap and a floor are in basis points of the commodity's **configured effective reference price** (GDS 6.3), never of a published price, a mean or a count | WLD 9.4.1 and ADR 0001 decision 7: no rule reads a measured aggregate to set a price or a period. A controller that does is PLC's, through the slot |
| 13 | Does a sole right or a cap bind the outside world or the estate? | **A legal price binds every order of the market**, a pseudo-agent's included; **a right does not bind a pseudo-agent** (estate, counterparty, treasury) | An estate must be able to liquidate and a gate must be able to trade; a price is a property of the market, not of an agent (5.5, 6.3) |
| 14 | A `region_bound` refusal? | **None.** Membership is a status derived from role and region and lapses on a move; there is no open account state, no good in flight, no job | N-090: a module registers one when it keeps an account in a region while the agent is away or committed. A seat in a body is neither (3.6) |
| 15 | Does a body gate SKL? | Yes, in three places and no more: a trainer must be a member where a `train` right is in force; a body may set a stricter `MaxTrainees` for its members; a completion is **held** until the target role has headroom | The three levers SKL 7 left to "association rules" (5.7). A tool is a recipe's `Requires`, so a right over the recipe is the barrier of the craft |
| 16 | What is a loop here? | The legal price (a shortage that the price cannot clear), the place (outsiders and a rent that persists), the held completion (trained households waiting), the prohibition of a staple. Each has a register entry and none is damped by the module itself | `spec-stability.md` 9. A law is a constraint, and a constraint that binds produces an unmatched quantity; the engine does not hide it, it counts it |

## 3. The body

### 3.1 What it is

3.1.1. A **body** is an entity of the configuration with an ID. It holds:

- its **roles**: the roles whose households may be members (at least one);
- its **scope**: the regions in which it operates, given as a list of regions, a list of
  jurisdictions (each stands for the regions it covers, `spec-polity.md` 4.1), or both
  (the union), outside nodes excluded;
- its **places**: the most households of its roles that each region of the scope may hold
  (3.4); 0 means no limit;
- its **trainee limit** `MaxTrainees`: a limit on the trainees one member may hold (5.7), 0
  meaning the role's own (SKL 4.4) alone;
- optionally a **need**: a settlement flag the region's class must carry for the body to
  admit anyone there (3.5).

3.1.2. A body is not an association of anything in particular. For example a craft association,
a league of traders, a licensed profession, a religious order with sole conduct of its rites, a union
that holds a closed shop: all are the same data with different roles, scope and rights.

3.1.3. A body holds no state of its own: no money, no goods, no decision, no calendar.
What it has, per region, is derived (the members, the households of its roles) and what is
saved is a field of each member (12).

### 3.2 Identity and order

3.2.1. The bodies of a `Config` are sorted by ID, bytewise, before a world is built or a
hash is taken (`spec-config.md` 3.4). The **`BodyID`** of a body is its index in that order,
from 0. It is a small named value type (architecture.md 3.3):

```go
// BodyID is the index of a body in ascending order of its ID string. It is not stable
// across configurations: adding a body can renumber the others. Use the string ID to
// persist or exchange one.
type BodyID int

// NoBody means "none": a household that is a member of no body, a right with no holder.
const NoBody BodyID = -1
```

3.2.2. The ID syntax is that of a region (`[a-z][a-z0-9_]*`, at most 64 bytes,
`spec-config.md` 3.3), unique among bodies. Bodies, rights and controls are separate
namespaces, except that a control and a rate limit share one (6.1).

3.2.3. Every iteration over bodies is by ascending `BodyID`, over regions by ascending
`RegionID`, over rights and controls by their ID order, and over households by ascending
agent ID, unless a rule of 4 says otherwise. No iteration over a map influences a result
(architecture.md 7.5).

### 3.3 Eligibility and the one-body rule

3.3.1. A household agent is **eligible** for a body when its role is on the body's list and
its region is in the body's scope. A pseudo-agent (an estate, a counterparty, a treasury)
is never eligible. An external agent is a household and is eligible like any other
(`spec-integration.md` 4.1): a player is admitted by the same rule and no call admits one.

3.3.2. A pair (role, region) is in at most one body (`ErrBodyOverlap`, 10.1). So a household
is eligible for at most one body, the body of a household is a function of its role and its
region, and the saved field of 12 says whether it was admitted and in which region. A setting with two
overlapping associations over one role models them as one body with two rights, or as two
bodies over two roles.

3.3.3. A role may be on the list of no body: its households are never members and never
barred by a right that names a body. A body whose scope covers a region in which none of its
roles has a group is legal and draws a warning (`body_empty`, 10.2).

### 3.4 Places

3.4.1. `Places` is a **limit** in the sense of `spec-settlements.md` 6.1 and its rule R2: a
count of households (the sum of weights, never agents), per region of the scope, never used
as a rate, a share or a period and never multiplied into one. With `Places` 0 there is no
limit: every eligible household is admitted (4.3) and the body is a privilege of the roles
and regions it lists. The count of households of the body's roles in a region, read against
`Places`, is a second reader of the one exception that `spec-settlements.md` 7 names to the
rule that no library rule reads a count (`spec-world.md` 9.4.1, ADR 0001 decision 7,
`spec-stability.md` 5.1.3 and FR-STB-020): it gates a permission and a number of places and
nothing else, never a rate, price, share, period or window. The edit that names ORG beside SET
in those texts is owed (section 8, SET and WLD rows).

3.4.2. **Headroom.** The **headroom** of a body in a region is
`max(0, Places - H)`, `H` the households of the body's roles in the region, members or not,
taken as SET 6.4.2 takes the households of a role: at the start of the region's entry step,
each admission of the plan reducing it. With `Places` 0 there is no headroom figure (the
view says `NoLimit`). The headroom is read by the spawner for an entrant and a switcher, by
the population module for births, and by SKL for a completion (4.5). It is **not enforced on
what is there**: a fall of the limit, a weight change, an arrival or an external agent can
leave the households above the places, and no household is removed for it (SET R3).

3.4.3. **A role that is in a body has the least of two limits.** The role's headroom in a
region is the least of the headroom of its class limit (`Role.Ceiling`, SET 6.4.2) and of its
body's (3.4.2). The two are not added and there is no third. This is an owed row to the
spawner and the population module (section 8).

### 3.5 The need

3.5.1. `Needs` names a settlement flag (`spec-settlements.md` 3.3). In a region whose class
does not carry it, the body **admits no one** (4.3 step 3 is skipped and the eligible
non-members are counted `Barred`). It is SET's hook H6 (an association) and, through a right
that the body holds, H4 (a reserved market), and it follows the rules of 6.2 there: R5 (a
`Needs` while the settlement module is off is `ErrModuleDependency`), R3 (a gate bars entry
and does not remove what exists: members stay, rights stay in force), R4 (every refusal is
counted: `Barred` of `MembershipChanged`), R6 (a gate that could close what a household needs
to eat is a hazard: 5.5 and ORG-A6).

3.5.2. The flag is read at P1.9a, which is before P7.2a, so the body reads the class of
SET 5.2.1 (the class set at the last review). A body's admission therefore follows a class
change by one tick.

### 3.6 No `region_bound` refusal

3.6.1. The rule of N-090 is that a module that keeps an agent's account in a region while
the agent is away or committed registers a `region_bound` refusal, or another caller (a forced
move of INT, a later module) can relocate the account and break the module's invariant.
**This module registers none.** A membership is a status: it needs no account, holds no
goods in flight and no escrow, and has no job. A relocation of a member changes its region,
and a membership is **of a (body, region)**: it lapses at the next round whatever the new
region is, even another region of the same body's scope (4.2.1), and the household is a
candidate of the body of its new region, if there is one, for the places that region has
left. The refusals that bind an agent (`open_shipment`,
`on_journey`, `open_contract`, `in_traineeship`, `open_hire`) are untouched.

3.6.2. A household that moves **loses** its place at the next round, in the same body or
another, and is a candidate in its new region with its `RoleSince` as it stands: a migrant is
the youngest (MIG sets `RoleSince` at arrival, N-090), and a household moved by a call
(`spec-integration.md` 4.5) keeps its seniority. A migration
estimate does not read a body (`spec-migration.md` 4: it reads the structure of the world and
rumoured prices), so a household may move to a region in which it will not be admitted. That
is ORG-A4, counted and bounded by MIG's settling rule.

## 4. The membership round

### 4.1 The position

4.1.1. One new position, in phase 1, after the directives. It follows the rules of
`spec-world.md` 6.1.1 and FR-WLD-040: a stable ID, one owner, and a letter after the position it
follows (nothing is renumbered).

| ID | What runs | Owner | Reads / writes | Draws | Events |
|----|-----------|-------|----------------|-------|--------|
| P1.9a | The membership round, with `Modules.Institutions` present and at least one body, for each body in ascending `BodyID`: the lapse pass over every household (4.2), then, for each region of its scope in ascending `RegionID`, admission (4.3). Scratch counts are kept for P7.2b | ORG 4 | reads the role, region, weight, `RoleSince` and the membership field of every household agent, the class flags (SET); writes the membership field | none | none (the counts are told at P7.2b) |

4.1.2. **Why at the start of the tick.** Membership is read from P2.5 (a `Work` request) and
P3.2 (an order), so the round must come before them; and it must come after the arrivals of
P1.3 and P1.5 so that a household that arrived this tick is a candidate before its first
start, and after the removals and role changes of the previous tick's phase 7. It must be
before P7.2a, because SET's rule is that a reader reads the class of 5.2.1 (R1, 6.5 there):
a body that read the class after the review would see a class change a tick before every other
reader. The round at P1.9a reads the class of the tick, which is the class after P7.2a of the
tick before.

4.1.3. **Construction.** There is no round at construction. At tick 0 no household is a
member; P1.9a of the first step admits by seniority (every initial agent has `RoleSince` 0,
so the key decides among them, 4.3.2). A job that a warm start created at construction
(`spec-viability.md` 8.5) runs to its completion whoever its household is: a right is read at
a **start**, never at a continuation or a completion (5.5).

### 4.2 Lapse

4.2.1. A household whose membership field names body `b` and region `r` loses it (the field
is cleared) at the round if it is **not eligible** for `b` (counted `LapsedRole` if its role is
not on the list and `LapsedRegion` if its region is not in the scope) or if its region is
**not `r`** (counted `LapsedRegion`: it moved, inside the scope or out of it). A role change is
at P7.1 and a relocation at P1.3, P1.5 or a call, so the lapse is seen at the next round;
**between the event and the round the field is stale, and every read tests eligibility and the
region as well as the field** (4.4), so a stale field is never a membership. A household that
lapsed and is eligible in its new region is a candidate there in the same round (4.3).

4.2.2. A removal clears the field with the agent (the estate takes the goods, `spec-agents.md`
3.6), and no lapse is counted: the removal event accompanies it (`AgentBankrupted`). A change of
weight keeps the membership (the household is a member whole).

### 4.3 Admission

4.3.1. For each (body, region), after the lapse:

1. `used` is the sum of weights of the members of the body admitted in the region (after the
   lapse every member is in the region it was admitted in); `free = Places - used` (no
   limit when `Places` is 0).
2. The **candidates** are the eligible non-members, ordered by `RoleSince` ascending, then by
   `Key(tick, agent)` ascending, then by agent ID. The three are a total order
   (architecture.md 7.6).
3. If the body has a need and the region's class lacks the flag, nobody is admitted: the
   candidates are counted `Barred` and the round ends for the region.
4. In order, a candidate is **admitted** if its weight is at most `free` (with no limit, always),
   and `free` falls by its weight. A candidate that does not fit is **skipped** (counted
   `Skipped`) and the round goes on to the next, which may fit.
5. `Waiting` is the eligible non-members left.

4.3.2. **The key** is the stateless `Key(t, id)` of `spec-resources.md` 5.3.3 (the SplitMix64
finaliser of `tick * 0x9E3779B97F4A7C15 + agent`), in `internal/ordering`, shared with the
ration of RES. It takes no draw and keeps no state, so the module is invisible in the stream
(FR-ORG-017).

4.3.3. **Seniority.** `RoleSince` is the tick at which the household took its current role
(`spec-spawner.md` 14: SPN's state; set at creation, at a role change and, by MIG, at an
arrival). A household that has worked its craft the longest is admitted first. An initial
household has `RoleSince` 0. The rule is a stated policy and not a claim of fairness: it
favours an incumbent over an arrival and a founder over a later entrant (ORG-A5).

4.3.4. **Weights.** A cohort is a member whole or not at all: places are counted in households
and a cohort of weight `w` is admitted only if `w <= free` (the rule of SET H2 for an entrant,
6.4.2 there). A cohort that does not fit is skipped and a smaller one behind it may be
admitted: that is the stated first-fit behaviour of a place, counted `Skipped`, and the first
consumer's world has weight 1 (N-030).

4.3.5. **The result is a function of the state**, not of the order of agents in a list: the
candidates are sorted by a total order and the lapse is per household. A second round with no
change in between admits and lapses no one (FR-ORG-019; `models/org/admit.py` A2 and A3).

### 4.4 The test of a membership

4.4.1. A household `h` is a **member of body `b` at a read** exactly when its membership field
is `(b, r)`, `r` is its region at that moment **and** it is eligible for `b` at that moment.
Every read of 5 uses it. The test is three integer comparisons and an array read.

4.4.2. A household that is eligible and not a member is an **outsider**; a household that is
not eligible is not an outsider (it is nobody's). An outsider is barred by a right of its body
exactly as any non-member is (5.5): it earns nothing from the activity and may leave the role
by the failure ladder (ORG-A4).

### 4.5 The headroom and the held completion

4.5.1. **Entry.** The default spawner places an entrant or a switcher into a role only while
the role's headroom (3.4.3) fits its weight, and counts a refusal in `EntryLimited.RoleCeiling`
(`spec-settlements.md` 6.4.2); births are capped at the headroom and counted in
`PopulationChanged.BirthsCeiling`. A body's places add to those counts under the same fields:
the owner of each count is SET's, and this spec adds a second **source** of a limit and no
field.

4.5.2. **A completion is held without headroom.** A trainee whose ticks served have reached
`TrainingTicks` completes at P7.1(a) step (1) (`spec-skill-barriers.md` 4.2.1) only if the
target role has **headroom of at least the trainee's weight (1)** in its region, by 3.4.3.
Otherwise it is **held**: it stays a trainee (it keeps running the trainer's eligible recipes
at `TraineeYield`, its board comes from the trainer, its ticks served stay at
`TrainingTicks`), it holds one of its trainer's places, and the tick is counted `Held`. A
place that opens (a member leaves, a removal) is taken by the first held trainee **in the entry
step in which the headroom reaches 1**, in the order of ticks served then application tick then
agent ID (the queue order of SKL 4.4; every held trainee has served `TrainingTicks`, so the
order is the application tick, then the agent ID). The household becomes a member at the next round
(P1.9a), because headroom of 1 means households at most `Places - 1`, so it fits unless an arrival or an external agent filled it first (ORG-A4, 3.4.2).

4.5.3. **Why a hold and not a lockout.** Without the hold a completion would make a skilled
household that is not admitted: it holds the role, may not practise it, earns nothing and goes
down the ladder. That wastes the whole traineeship the trainer paid for. A held trainee is
useful in the meantime (it produces at `TraineeYield` of the eligible recipes), and the
**throttle** that results (a place for each place vacated) is the one the body's places exist to
produce. The cost is that a trainer's places are blocked and the trainer's gain test
(`spec-skill-barriers.md` 4.3) reads a trainee that no longer pays its way: that is the
self-limiting loop ORG-A3.

4.5.4. **A body with no limit** (`Places` 0) never holds a completion. A role that is in no
body is untouched.

### 4.6 What changes the count of households

4.6.1. Households of the body's roles in a region change by: entry at P7.1(d), births, a
removal, an arrival at P1.3 or P1.5, a role change, a weight change, and an external agent's
call. The membership round reads the state at P1.9a; the headroom reads the live count at the
start of each region's entry step. So a household that enters at P7.1 is counted in the
headroom of the same step's later admissions, and is a candidate at the next P1.9a.

4.6.2. **What the round does not do.** It does not remove a household for exceeding the places
(SET R3, 3.4.2); it does not evict a member for a bad record; it does not change a role; it
takes nothing from a household and gives nothing (no dues, 3.1.3). It does not look at money,
stock, income, price or ratio: its inputs are a role, a region, a weight, a tick and an ID, and the one count it reads is
the households of the body's roles in a region against `Places` (3.4.1).

## 5. Rights

### 5.1 What a right is

5.1.1. A **right** is an entity of the configuration with an ID. It says that one
**activity** on one **subject** in one **scope** is reserved to the members of one body, or,
with no body, to nobody:

```go
// Right reserves an activity on a subject, in a scope, to the members of a body.
type Right struct {
	ID           string
	Body         string // the holder; empty means a prohibition: nobody may
	Activity     string // ActivityProduce, ActivitySell, ActivityBuy or ActivityTrain
	Subject      string // a recipe (produce), a commodity (sell, buy) or a role (train)
	Region       string // the scope: exactly one of Region and Jurisdiction
	Jurisdiction string
	Who          string // produce only: WhoPrincipal (default), WhoExecutor or WhoBoth
}
```

5.1.2. The four activities and the three values of `Who` are **rule constants**, exported
from the root package (`event.go`), a closed set: a new activity is an edit of this spec and
of its reader (architecture.md 9.6).

5.1.3. **A reservation to a few, to one, to a list of roles and to nobody are one mechanism.**
A right with a body that has a few places over a scope of one region is what a setting may call,
for example, a charter; with one place over the whole world, a monopoly; with unlimited places and
a list of roles, a licence of those roles; with no body, a prohibition. The setting's word is a
`Label` of the right (presentation only, not hashed).

### 5.2 The four activities

| Activity | Subject | The actor | Read at | Where the check sits | What is refused |
|----------|---------|-----------|---------|----------------------|-----------------|
| `produce` | a recipe | the **principal** of the job, the **executor**, or both, by `Who` (5.3) | P2.5, the `Work` request; EMP's formation at P2.5a | `spec-production.md` 4.2: immediately after `not_on_role`, before `k0` is derived | the **start**: nothing is consumed, reserved or worn; a `Debug` line and a count |
| `sell` | a commodity | the agent whose ask it is | P3.2(c), when the world builds the orders | `spec-agents.md` 6.5.2: after `not_tradable` | the **order**: `OrderRejected` with the reason `not_permitted` |
| `buy` | a commodity | the agent whose bid it is | P3.2(c) | as `sell` | the **order**: `OrderRejected`, `not_permitted` |
| `train` | a role | the trainer (a household that holds the role) | P7.1(a) step (2), the filling of places | `spec-skill-barriers.md` 4.4 | the **place**: it is not open for a non-member; counted |

5.2.1. **The region of the check** is the region of the actor's account at the read. For a
job it is the region of the executor, which is that of its principal (a trainee and its
trainer, a worker and its employer are in one region: `spec-skill-barriers.md` 4.6.1,
`spec-employment.md` 6.5). For a delivery offer or a take of the contract round (6.7) it is the
**destination** region: the offers name it and the takes are those of agents whose account is in
it (`spec-merchants.md` 9.4.2).

5.2.2. **`sell` and `buy` are checks on the plan, not on the goods.** A household that is
barred from selling a commodity may still hold, consume and use it as an input, and may ship
it (5.8); a household that is barred from buying may still produce it. A `buy` right on a
commodity that is some role's upkeep is a hazard of 5.5.

### 5.3 Who: principal, executor, both

5.3.1. A job has an **executor** (the household that works) and a **principal** (the account
whose inputs and tools the job uses and to which its output goes: `spec-production.md` 4.3.2,
`spec-skill-barriers.md` 4.2, `spec-employment.md` 6.1). For an ordinary job they are one
agent. `Who` says whose membership a `produce` right tests:

| `Who` | Tested | For example |
|-------|--------|-------------|
| `principal` (default) | the account whose inputs the job uses | a member may employ workers who are not members: the member is the craftsman |
| `executor` | the household that works | a closed shop: only members may be hired to this recipe |
| `both` | both, each against the same body | an association of employers and workers |

5.3.2. A **traineeship is exempt from `executor`**: a trainee is by definition not yet a
member, so the executor test is not applied to a trainee's execution of a trainee-eligible
recipe; its principal (the trainer) is tested as for `principal`. A right that wants to keep
trainees out is a `train` right (5.7).

5.3.3. `Who` is meaningful for `produce` only; on any other activity a non-default value is
`ErrRightShape` (10.1).

### 5.4 Scope and resolution

5.4.1. A right's scope is **one key**: a region or a jurisdiction. The rows are keyed by
(activity, subject, scope key): a second row with the same key is `ErrDuplicateID`. This is
the table-keyed-by-jurisdiction form of `spec-polity.md` 6.1, with the mode **innermost**.

5.4.2. **The right in force** for an activity on a subject in region `r` is, in this order:
the row keyed by `r` if there is one; else the row of the first jurisdiction of `Chain(r)`
(`spec-polity.md` 4.1.1) that has one; else none. A region with no jurisdiction and no
direct row has none: the activity is open. The result is resolved **once** at construction
(and again at a slot call, 7.2) into a dense array indexed by (activity, subject, `RegionID`),
so a read in the tick is an array read that allocates nothing and is not saved.

5.4.3. **A suspended right** (7.2) leaves its key **open**: the row is not consulted and no
outer row is looked at in its place. A suspension is a local exemption, not a delegation to
the next jurisdiction.

5.4.4. **A right over a region in which its body has no eligible households** (the body's
roles have no group there, or the region is outside the body's scope) closes the activity
to everyone there: nobody is a member. That is a prohibition by accident, so it draws the
warning `right_unreachable` (10.2); a deliberate prohibition says so with an empty `Body`.

### 5.5 What a right binds, and what it does not

5.5.1. **A right is read at a start, at an order and at a place; never at a continuation, a
completion, a holding or a consumption.** A job in progress when a right comes into force
runs to its end and its output is the principal's as always; a household that lapses keeps
its stock and may eat it; goods that a right bars its holder from selling are not
confiscated.

5.5.2. **Exempt.** (a) A **pseudo-agent** is never refused: an estate must be able to
liquidate what it inherits (`spec-estates.md` 2.7), an outside counterparty is the gate's
(`spec-external-sector.md` 4.2) and a tariff or a duty on it is FIS's, a treasury places no
order (`spec-polity.md` 5.3.3). (b) A job of an **own-use** recipe (`spec-subsistence.md`
3) is exempt from a `produce` right unless the module field `OwnUseBound` is set: a
household may feed itself (SET R6). (c) The **executor of a traineeship** (5.3.2). (d) A
household of a role that is on no body's list is not exempt: it is simply not a member, and
a right that names a body bars it.

5.5.3. **The hazard of a gate on food.** A `buy` right on a commodity of someone's upkeep, a
`sell` right on the only commodity a role makes, a `produce` right on the only recipe that
sources a staple, held by a body with places: each can close what a household needs to eat
(SET R6). The module does not prevent it, because a pack may want exactly that (a famine
scenario, a besieged town). It draws the warning `right_over_upkeep` (10.2) and the register
entry ORG-A6. The own-use exemption of 5.5.2(b) is the engine's one safeguard.

5.5.4. **A right is not a title.** It reserves an activity, not a stock: it does not give a
claim on an endowment (`spec-land.md`: a title and the ration are LND's) and it does not move
a unit of anything. A `produce` right over a recipe that draws on an endowment is read before
the ration (`not_permitted` before `no_endowment`), so a barred household takes no part in it.

### 5.6 The refusal reason

5.6.1. `not_permitted` is an exported constant of the root package (`event.go`; the key
string is in `internal/logkeys`). It is a **refusal reason of production** (joining
`spec-production.md` 4.2: `not_on_role`, then **`not_permitted`**, then, with `k` derived,
`zero_yield`, `no_site`, `no_tool`, `no_facility`, `no_input`, `no_endowment`, `no_title`,
`rent_funds`, `no_room`), a **rejection reason of an order** (joining `spec-clearing.md` 3.2,
checked by the world beside `not_tradable`) and nothing else: a place that is not open
(`train`) is counted, with no reason constant, because no one asked for it.

5.6.2. **Order of the checks of an order**, for a household agent (`spec-agents.md` 6.5.2 and
`spec-clearing.md` 3.2): `unknown_agent`, `side_not_allowed`, `unknown_commodity`,
`not_tradable`, **`not_permitted`**, then the market's `invalid_price`, `invalid_quantity`,
`duplicate_order`, `overflow`, then the **legal clamp** (6.3), then `insufficient_funds` and
`insufficient_stock`. A refused order is a rejection and has no outcome (`spec-clearing.md`
7.3): the household's beliefs are not revised for it.

5.6.3. **Counting (N-045, FR-STB-013).** Every refusal is counted per (region, right) and
activity, and told once a tick by `RightRefused` (11.1). An order refusal is also an
`OrderRejected` (per order, as any); a start refusal is a `Debug` line, guarded, per start; a
closed place is counted and logged at `Debug` ("place not open", 11.2). **One count of `Train`
is one place:** at P7.1(a) step (2) of an entry step, with a `train` right in force on the role,
`f` is the free places (under the role's own `MaxTrainees`) of the households of the role that
are not members of the holder body (all holders, for a prohibition) and `a` the applicants still
queued after the filling; `Train` is `min(f, a)`, so a closed craft with no queue counts none.

### 5.7 The traineeship: what a body does to SKL

5.7.1. **A trainer is a member where a `train` right is in force.** With a `train` right on
the role `R` in force in region `r`, a household that holds `R` has places for trainees (the
filling of `spec-skill-barriers.md` 4.4, at P7.1(a) step (2)) only if it is a member of the
holder body. A non-member trainer has none; a trainee already accepted is **not** thrown out
(the right, like the gain test of SKL 4.3.1, is read when a place opens). A prohibition (no
body) opens no place at all.

5.7.2. **The applicant counts only eligible trainers.** The count `trainers` of
`spec-skill-barriers.md` 4.4, 4.5 (the expected wait `q_w`) and 4.9 is the households of the
target role in the region that **can** take a trainee: members where a `train` right is in
force, all holders otherwise. With none, no one applies (SKL 4.5's rule), so a closed craft
does not collect a queue. The places `n_max = trainers * MaxTrainees` of 4.9, and with them the
expected wait `q_w` of 4.5, count for each trainer `min(MaxTrainees of the role, MaxTrainees of
its body)` (5.7.3), so that the wait divides by the places that exist.

5.7.3. **A body's own trainee limit.** A member holds at most
`min(MaxTrainees of the role, MaxTrainees of its body)` trainees at once, a body's 0 being
no limit of its own. It is the "association rule beyond the trainee limit" of SKL 7. It is a
**tunable of the body** and applies to a household that is a member at the moment a place
opens; it does not end a traineeship in progress.

5.7.4. **The completion is held without headroom** (4.5.2). The condition is read at
P7.1(a) step (1) for the trainee's target role and region.

5.7.5. **The premium check.** The derived check D of `spec-skill-barriers.md` 3 assumes entry
is limited by a tool and a traineeship and nothing else. For a role whose households are
limited by a body's places, the observed premium exceeds the derived one by a scarcity rent,
and the **check D is reported, not asserted**, for such a role (the check C, that the premium
is above 10 000 with a barrier, still holds). It is an owed row to SKL 3 and FR-SKL-024.

5.7.6. **A tool barrier through a body.** A right of activity `produce` on the recipe that
`Requires` the tool makes the craft the body's: the tool is bought on the market as ever
(`spec-skill-barriers.md` 2) and a non-member may hold it and may not use it. A right of
activity `sell` on the tool's commodity restricts who may sell tools. Both are rows; no
second tool mechanism exists.

### 5.8 Not offered, and why

5.8.1. **`ship`: a right over carriage.** A right to dispatch a shipment from a region or
along an edge (for example a company's sole use of a route) would be a refusal in the list of 23
checks of `spec-transport.md` 5.3.2. It is not offered in version 1: a delivery contract's
deliverer is a shipper, so the same bar must be applied there and at self-carried trips, and
the commodity of a shipment is a set of lines. The `sell` and `buy` rights of the
**destination** region already bar a stranger from the market it ships into, and an edge's
status closes a route (`spec-polity.md` 7.2.2). Open question 3.

5.8.2. **`store` and `use`: a right over a store or a facility.** Who may keep goods for
others (`spec-storage.md` 10.4) and who may use a facility (`spec-infrastructure.md` 5.1.5)
are reserved. A right over a recipe that requires a facility works without CAP knowing; a
right over a store is STO's refusal when STO exists.

5.8.3. **Naming a household.** A right is held by a body and a body is a set of households
selected by role, region and seniority. No right names an agent, because an agent's ID is
not a configuration key (N-117) and a right granted to a person is a policy decision about a
household (PLC; the player's own is a role of one household, 3.3.1).

5.8.4. **A right that depends on a measure.** A right that ends when the price rises, a
cap that follows the harvest: a function of an aggregate (WLD 9.4.1) and PLC's, through the
slots of 7.

5.8.5. **Delivery contracts and the markets of other regions** are reached through the
destination region's `sell` and `buy` rights (MER 9.3 admission, section 8) and need no
activity of their own.

## 6. Legal prices

### 6.1 Controls and rate limits

6.1.1. A **control** is a legal range for the price of one commodity in one scope:

```go
// Control is a legal range for the price of a commodity: a cap, a floor or both, in
// basis points of the commodity's effective reference price (spec-goods.md 6.3).
type Control struct {
	ID           string
	Commodity    string
	Region       string // the scope: exactly one of Region and Jurisdiction
	Jurisdiction string
	CapBP        int64 // 0: no cap
	FloorBP      int64 // 0: no floor
}

// RateLimit is a legal cap on the yearly rate of a loan: the highest effective rate
// at which a borrower may borrow in the scope (spec-credit.md 7).
type RateLimit struct {
	ID           string
	Region       string // exactly one of Region and Jurisdiction
	Jurisdiction string
	CapBP        int64 // yearly, basis points, 0 to MaxCreditRateBP; 0 is interest-free
}
```

6.1.2. Controls and rate limits share one ID namespace (a **`ControlID`**, the index in
ascending ID order over both, controls then rate limits sorted together bytewise), so that
the one slot of 7.1 names either. `ControlID` is a small named value type, as `BodyID`.

6.1.3. Rows are keyed by (commodity, scope key) for a control and by scope key for a rate
limit; a second row with the same key is `ErrDuplicateID`. **The legal range in force** for a
commodity in a region is the row keyed by the region, else the first row of the region's
chain (`spec-polity.md` 4.1.1), else none: the mode **innermost** of POL 6.1.2, as for a
right.

6.1.4. **No row is a rate of an aggregate.** A cap in basis points of a configured reference
price is a constant of the pack. A cap that follows a published price, a harvest or a count
is a feedback and is a policy actor's through the slot (position 12).

### 6.2 The range in minor units

6.2.1. Let `ref` be the effective reference price of the commodity in the region
(`spec-goods.md` 6.3: the region's override, else the commodity's), `cap_bp` and `floor_bp`
the levels in force (the configured ones, or the slot's, 7.1). Then, with truncating integer
division and checked arithmetic (architecture.md 3.5),

```
cap   = min(MaxPrice, max(1, ref * cap_bp   / 10 000))     if cap_bp   > 0, else none
floor = min(MaxPrice, max(1, ref * floor_bp / 10 000))     if floor_bp > 0, else none
```

Both are truncated and bounded by `MaxPrice` (`spec-config.md` 4.1: a limit above it is
`invalid_price`, so a floor above it would raise an ask out of the bound after that check has
passed, and the product of the limit and a quantity could leave `int64`). Equal basis points
give equal prices, and a floor never exceeds its cap (validation requires `FloorBP <= CapBP`
when both are set, 10.1; both are monotone in the basis points). The product `ref * cap_bp` is
at most `MaxPrice * MaxBasisPoints` = 10^15 and fits `int64`; it is checked. The admitted limit
is then in `[1, MaxPrice]` like any other, so the checks that follow see an ordinary order.

6.2.2. **Resolved once, and at a slot call**, into dense arrays indexed by (`RegionID`,
`CommodityID`), derived and not saved. An outside node has none (it has no market).

### 6.3 The clamp

6.3.1. **The rule.** In the admission of every order of a market (`spec-clearing.md` 3.2), after
the checks of price, quantity, duplicate and overflow and before the checks of funds and stock:

- a **bid** whose limit price exceeds the cap has its limit price **replaced** by the cap;
- an **ask** whose limit price is below the floor has its limit price **replaced** by the
  floor;
- an order inside the range is unchanged.

The admitted order carries the replaced limit: the **reservation** is `limit * quantity` at
it (3.3 there), `OrderPlaced` shows it, `Clearing` receives it, the world's validity check
of trades (`spec-clearing.md` 8.2) compares with it, and the household's `Outcome` carries it
as the order's limit. The count of the replaced orders is kept per (region, control), told
once a tick by `PriceControlled` (11.1).

6.3.2. **Why one side is enough.** The price of a trade lies between the ask's limit and
the bid's limit, inclusive (`spec-clearing.md` 5.1, 5.2: the midpoint rounded half to even
of integers `a <= b` lies in `[a, b]`). Therefore if every bid's limit is at most `cap`,
every trade's price is at most `cap`; if every ask's limit is at least `floor`, every trade's
price is at least `floor`. A bid above the cap that is lowered to it can still trade with an
ask at or below the cap, at a price between the two. `models/org/clamp.py` checks it on 20 000
random books (P1, P3) with a cap, a floor and both.

6.3.3. **Every order of the market, whoever places it.** The clamp is a **rule of admission**,
which `spec-clearing.md` 2.3 says is the same for every `Clearing`. It applies to an estate's
named-lot ask (the lot's listing price in the estate's state is unchanged, only the order is
raised to the floor), to a counterparty's order (a quote above a cap does not trade as an
import; the top-up of `spec-external-sector.md` 5.2 was computed from the quote and what the
order does not reserve is swept back unspent, P4.2) and to a household's. A custom `Clearing`
sees only clamped limits and cannot break the range: the guarantee of CLR 8.2 is unchanged.

6.3.4. **What the clamp does not do.** It does not reduce a quantity, reject an order, reorder
the book or change the tie-break key (the keys are drawn per admitted order as before, D7); it
adds no draw. A bid clamped to the cap and an unclamped bid at the cap tie and are separated by
the key, exactly as two bids at one limit are. A clamp is not silent: every replaced limit is counted
in `PriceControlled` (11.1), which is why the sentences of `spec-agents.md` 6.5.2,
`spec-clearing.md` 3.2 and `spec-credit.md` 5.2.2 that the world clamps nothing are about a
**quantity against a budget** and not about a legal price.

### 6.4 Why a clamp, and not a rejection or a strategy rule

The brief asked which of "an admission rule in the strategy" and "a rejection reason
`price_control`". The third, adopted:

| Option | Behaviour | Verdict |
|--------|-----------|---------|
| **Clamp the limit in admission (adopted)** | A buyer who would pay more still buys at the legal price; a seller who would sell lower is held to the floor. The order's quantity is intact. All of CLR 2.3 and 8.2 unchanged | A control states a price, not a ban on buying. Volume is at least what a rejection gives and is exactly the units that can cross (`clamp.py` P2: 125 849 units against 66 003 on 20 000 books) |
| A rejection reason `price_control` (any order with a limit beyond the range) | The order is refused whole and has no outcome | **Rejected.** It removes every buyer whose belief or draw is above the cap, which is most of them in a shortage, and so turns a cap into a trade ban at the very time it is binding; it gives beliefs no outcome to learn from; and it makes the volume a function of how the strategy's limits are drawn, not of the law |
| A rule in the strategy (a cap in `Trade`) | The default strategy lowers its own limit | **Rejected.** The strategy is consumer code (`spec-agents.md` 6); the world checks what it returns (6.5) and does not trust it with a law. A rule in the default would bind the default and leave a consumer's strategy free to break the cap |

6.4.1. **What the three share:** the range is read from the order's own region and commodity
and is a constant of the tick, so no order depends on another's (CLR 3.3.1).

### 6.5 What follows

6.5.1. **No trade outside the range, and so no black market.** Every trade is at a price
inside the range (6.3.2). A price above a cap is never paid; the quantity the cap cuts is
**unmatched**, not diverted: units wanted at the cap and not offered at it stay unmatched
bids, asks above the cap stay unmatched asks (`MarketCleared` carries both and the best of
each). A smuggler is RSK's.

6.5.2. **Who is served.** Among the bids clamped to one cap the book is ordered by the keyed
tie (CLR 4.1, 4.2) and the cheapest asks are matched first, so the service is fair between
agents and not a ration: the module allocates nothing (N-122 does not arise). A consumer that
wants a ration (a coupon) is a policy and a flow.

6.5.3. **The published price and its age.** A market's published price is the volume-weighted
mean of the tick's trades, or the previous price carried forward when nothing trades
(`spec-clearing.md` 5.4). A cap introduced below the last price leaves the published price
where it was until something trades, and **the published price can sit above a cap** while
`Age` counts up. The price window and the information record (`spec-beliefs.md` 7,
`spec-information.md` 4) take only what traded, so they hold prices at or below the cap.

6.5.4. **Beliefs.** A household's `Revise` sees the order as it stood: the clamped limit and the
outcome. Its belief is revised by the rules of `spec-beliefs.md` 5 unchanged: the **evidence**
rule compares the best unmatched ask with the (clamped) limit and moves the centre toward it,
and the **stock** rule raises it as stock falls. Under a binding cap a buyer's centre
therefore rises: it tracks the **shadow price**, up to `RefBandHigh`. The limit is then drawn
from the centre and clamped to the cap, so no order is hurt; when the cap is lifted the price
**jumps** to where the beliefs are. That is the economics of a price control and is registered
(ORG-A2) and tested (ORG-S1); no rule of beliefs is changed for it, because a belief that
tracks the unmet demand is the signal a controller reads.

6.5.5. **Sellers.** A seller's belief is bounded below by its cost band (`CostFloor`, 7 000 of
the full unit cost: `spec-beliefs.md` 3.2.2), so a cap below `CostFloor` times the unit cost
freezes the market (no ask reaches a bid): sellers hold their goods, the production halt
bounds the glut (D24) and the exit of the failure ladder follows its delay. The warnings
`cap_below_band` and `cap_below_seed_cost` (10.2) say so at construction. A cap between the
floor and the unit cost is a margin squeeze, not a freeze: ORG-A1.

6.5.6. **A floor** is the mirror: asks are raised to it, bids below it do not cross, and a
glut of unmatched asks builds; the halt of D24 acts on the output target and the stock. A
seller's belief is unaffected and only its order is raised. It is the same loop with the
other sign (ORG-A1).

### 6.6 Credit: the rate limit

6.6.1. The credit round of `spec-credit.md` 6 forms a loan from a request (a borrower's
`MaxRateBP`, the highest yearly rate it accepts) and an offer (a lender's effective rate, the
least it accepts). It is a double auction over a rate: the borrower is the bid, the lender the
ask. A rate limit is therefore a **cap on a bid**: in the admission of the round
(`spec-credit.md` 5.2), after the last of its checks, a request's `MaxRateBP` greater than
the cap in force in the borrower's region is **replaced** by the cap (a request is never
refused for it). A lender whose effective rate is above the cap then finds no borrower: the
first offer above the request's limit ends the scan (CRD 6.3.2), as CRD 13.1.2
says of a scale above the limits. A cap of 0 admits only interest-free credit.

6.6.2. **The region** is the region of the borrower's account at the round (`spec-credit.md`
6.2). **The rate** is the effective rate of an offer after its region's scale slot
(`SetRateScales`), so a lender cannot post below a cap and scale above it: the clamp bounds
what a borrower accepts, not what a lender posts, and the lender's effective rate is what is
compared. A pack that wants a lender never to quote above the cap sets the slot.

6.6.3. **Why no rejection** is the reason of 6.4: a borrower who would pay more should still
borrow at the legal rate if a lender offers it. **What it does not do**: it does not alter a
claim in force (a rate is fixed at formation, CRD 7.1.3), it does not forgive (the call
`ForgiveClaims` is CRD's) and it creates no credit that was not offered.

### 6.7 Contracts

6.7.1. A **delivery contract** forms a price from an offer's ask limit and a take's bid limit
in a round after the last plan of phase 3 (`spec-merchants.md` 9.4), not in the book. A legal
price that bound only the book would be evaded by a contract. The same clamp is therefore
applied in the admission of the round (MER 9.3): a **take's** bid limit above the cap in force
for the commodity in the **destination** region is replaced by it, an **offer's** ask limit
below the floor is raised to it, after the check `overflow` of a take (MER 9.3, row 9) and after
`invalid_price` of an offer (row 6): the same place as in the market. It is an owed row to MER
(section 8).

### 6.8 Wages and rents: reserved

6.8.1. **A rent control** is LND's: a rent is a posted value with a scale slot
(`SetRentScales`, `spec-land.md` 9.1) and a cap is that slot; this spec adds nothing.

6.8.2. **A wage control** (a minimum wage, a wage cap) has the same form: a hire is formed
in a first-fit round from a wage that the employer offers and a reservation of the worker
(`spec-employment.md` 5, 6), a cap on what an employer may offer is a clamp of its bid and a
floor on what a worker may be hired for a clamp of the worker's reservation. EMP's text is
under review (README: draft, revisions pending) and this spec does not write rows for it.
It records the form (a `Control` whose subject is a hireable role) and the open question 5.

### 6.9 What a legal price is not

6.9.1. It is not a tax, a subsidy or a levy (FIS), not a ration (6.5.2), not an order to
produce (no household is made to work: the unmatched quantity is a shortage), not a stock
that is released (a store of staples is STO's with FIS), and not a price the engine sets: the market
still forms every price inside the range.

## 7. In the tick, between ticks, and after a restore

### 7.1 The two slots

INT is minimal (ADR 0003 decision 2, N-131), and a module adds a slot only for something a
ruler or a consumer must change in a run. The institutions are fixed structure; two things
about them are not: the **level** of a legal price and the **force** of a right. Each is a
slot of the shape of `spec-integration.md` 5.1: a slice of entries, one per key, atomic, an
event per changed key in key order, between ticks only, never configuration, not hashed.

| Call | Entry (key; value) | Value bound; default | Needs | Owner | Read by the tick at | Event (at the call) | Saved by |
|------|--------------------|----------------------|-------|-------|---------------------|---------------------|----------|
| `SetLegalLevels` | `LegalLevel{Control; CapBP, FloorBP}` | goods: `CapBP` and `FloorBP` 0 to `MaxBasisPoints`, 0 none, `FloorBP` at most `CapBP` when both set; a rate limit: `CapBP` 0 to `MaxCreditRateBP`, `FloorBP` 0; the configured levels | institutions | this spec 6, 7.2 | the admission of the market (P3.2, P3.4, P3.5), of the contract round (P3.3) and of the credit round (P3.3b); **the resolved ranges are rebuilt at the call** | `LegalLevelSet` (new), one per changed key in ascending `ControlID` | this spec 12, by `ControlID`, sparse; **not hashed** |
| `SetRightSuspensions` | `RightSuspension{Right; Suspended}` | a bool; false | institutions | this spec 5.4.3, 7.2 | P2.5 (and EMP's formation), P3.2(c), P7.1(a); **the resolved rights are rebuilt at the call** | `RightSuspensionSet` (new), one per changed key in ascending `RightID` | this spec 12, by `RightID`, sparse; **not hashed** |

7.1.1. **Errors**, in the order checked (`spec-integration.md` 5.1.3): `ErrModuleOff`
(`Modules.Institutions` absent), then for each entry in the order given `ErrUnknownControl`
or `ErrUnknownRight` (new, root sentinels, an ID outside `[0, n)`), `ErrOutOfBounds` (a level
outside its bound, a floor above a cap, a `FloorBP` on a rate limit), `ErrDuplicateID` (two
entries for one key). A call that cannot have an effect is refused, not ignored.

7.1.2. **An entry equal to the configured level restores it**, and the sparse state then holds
no row for the key. A rate limit is switched off by setting `CapBP` to `MaxCreditRateBP` (it
would not bind any request); a goods control by setting both levels to 0.

7.1.3. **What a ruler may not do.** It may not add a control, a right or a body that the
configuration did not declare, move a right to another body, name a household, change a body's
roles, scope or places, or set a legal price in a currency: those are configuration, hashed.
A control that a pack wants available to a ruler is declared with both levels 0 (inert) and
the warning `control_inert` is informational (10.2). A right granted at run time to a named
household is an open question (4, N-131).

7.1.4. **No timer, no callback.** A level holds until another call sets it, as every slot does
(`spec-integration.md` 5.2.2). A cap that ends on a date is a consumer's calendar and a call.

7.1.5. **Who sets them.** A consumer sets either slot by its call. A policy actor (`spec-policy.md`) reaches
`SetLegalLevels` through a lever of kind `legal` (6.3.2 there: reserved, a number of basis points per
control); `SetRightSuspensions` has no lever, because a suspension is a switch and not a
basis-point number: it stays a consumer's call.

### 7.2 What acts at the call

7.2.1. The ranges of 6.2 and the resolved rights of 5.4.2 are derived arrays. A slot call
**rebuilds them at the call** for the whole batch once, so that no observer or reader sees a
half-built table (`spec-world.md` 8.2.1: the case of "acts at the call" for a derived table).
The next read of the tick uses them.

7.2.2. **A level takes effect for orders admitted from the next step.** There is no order
open between ticks (no reservation exists at a boundary, FR-LED-020), so no order is altered
and no reservation is re-priced. A loan formed under the old cap stays (a rate is fixed at
formation).

### 7.3 Positions, draws, events

7.3.1. **Positions.** The module has two, both in the table of `spec-world.md` 6 (owed rows):

| ID | What runs | Owner | Reads / writes | Draws | Events |
|----|-----------|-------|----------------|-------|--------|
| P1.9a | the membership round (4.1) | ORG 4 | the households' role, region, weight, `RoleSince`, the membership field; the class flags | none | none |
| P7.2b | after P7.2a, with `Modules.Institutions` present: the module's aggregate events of the tick, in the order of 7.3.3 | ORG 11.1 | the scratch counts of P1.9a, P2.5, P3.2 to P3.5, P7.1(a); nothing is written | none | `MembershipChanged`, `RightRefused`, `PriceControlled` |

7.3.2. **The reads of the module that are not positions**, listed so that WLD can say the
module has no other: P2.5 (the right of a `Work` request, after `not_on_role`), P2.5a (EMP's
formation, if EMP wants it), P3.2(c), P3.4 and P3.5 (the clamp of the market's admission, the
right of an order), P3.3 and P3.3b (the clamp of a contract's and a loan's admission), P7.1(a)
steps (1) and (2) (the held completion and the filling of places), and P7.1(c) and (d) (the
headroom, as SET 6.4.2 reads its limit). Each is a table read.

7.3.3. **The order of the events of the tick.** All three are at P7.2b, in this order: per
(region, body) in ascending (`RegionID`, `BodyID`) `MembershipChanged`; per (region, right) in
ascending (`RegionID`, `RightID`) `RightRefused`; per (region, control) in ascending
(`RegionID`, `ControlID`) `PriceControlled`. They come after `SettlementClassChanged` of P7.2a
and before phase 8's `BoundaryFlow` (`spec-world.md` 6.11 item 8 gains one clause). The two
slot events come at the call and carry the tick the next step will run, before its
`TickStarted` (6.11).

7.3.4. **Draws: none.** The module takes no word from the world's stream (FR-ORG-017). A
refusal removes an order and so moves the counts of D6, D7 and D8 as the removal of any order
does (ORG-N5); a clamp changes no count. The one place an order among households needs a key is the
stateless `Key(t, id)` (4.3.2). WLD's table of draws gains no row; D1 and D10 are unchanged.

### 7.4 After a restore, and a call between ticks

7.4.1. A restore builds the resolved tables from the configuration and the saved slot state,
reads the membership field from the agent record (12) and checks it (12.3). The next P1.9a is the round the uninterrupted world
would have run (FR-ORG-033).

7.4.2. **Calls on agents** (`spec-integration.md` 4): an external agent that is added is
eligible at the next P1.9a like any household (its `RoleSince` is the tick of its creation);
one that is removed leaves no trace; one that is relocated lapses or joins by the rule of 4.2;
no call admits, expels or moves a member (open question 4).

## 8. What the specs around it do

Every row is a change owed by this spec to a spec that exists or is being written, or a
promise to one that comes. None is made here (the file touches no other spec); each is a row
of the handoff. "Owned by" says whose behaviour it is.

| Spec | What ORG needs or gives | Where | Owned by |
|------|-------------------------|-------|----------|
| SKL | a trainer is a member where a `train` right is in force; a body's own `MaxTrainees`; a completion held without headroom; `trainers` counts the eligible; check D reported for a capped role; a `Debug` reason `not_member` | 4.2.1, 4.4, 4.5, 4.9, 3, 8.3, FR-SKL-015, 019, 024 | SKL |
| SET | the flag `Needs` of a body serves H4 and H6; a second limit into H2's headroom (the least of two); rows R5 to R7 stand; ORG is a second reader of the count of households that 7 names as the one exception (3.4.1), so the four texts of 7.4 name it too | 6.3 (H4, H6), 6.4.2, 6.5, 7.4 | SET |
| POL | H5 and H6 are taken by the fields of 5 and 6; open question 2 is answered: ORG keeps no second membership table of POL's kind, a body selects by role and region | 6.3 (H5, H6), 6.4.5, 17 (question 2) | POL |
| SPN | the entry view's `Headroom` and `EntryLimited.RoleCeiling` include a body's places | 6.2, 7.2, 11, 13.1 | SPN |
| POP | births are capped at the headroom of a role that is in a body | 3.7.3, 4.4 | POP |
| MIG | an arrival is an outsider if the places are full; the estimate does not read a body (ORG-A4); `RoleSince` at arrival is the seniority | 4, 8.5 | MIG (no change) |
| CLR | the clamp of 6.3 joins admission after `overflow`; the reason table is unchanged | 3.2, 3.3 | CLR |
| AGT | `not_permitted` joins the world's checks of a `Trade` plan after `not_tradable` | 6.5.2 | AGT |
| PRO | `not_permitted` joins the refusals of a start, after `not_on_role` | 4.2, 14.2, FR-PRO-008, FR-PRO-048 | PRO |
| EMP | a hire is checked as a request: the principal and, by `Who`, the executor; a closed shop is `Who`; a wage control is reserved (6.8) | 5.5, FR-EMP-023 | EMP |
| LND | a right over a recipe that draws on an endowment is read before `no_endowment`; a rent control is `SetRentScales`; the `Closed` row of LND is not this | 1, 4.5, 9.1 | LND |
| CAP, STO | a right over a recipe that requires a facility, a lease or a store: reserved (5.8.2) | 5.1.5; 10.4 | CAP, STO |
| MER | the clamp of 6.7 in the contract round's admission, after the check 9 (`overflow`) of a take and after the check 6 (`invalid_price`) of an offer; the destination's `sell` and `buy` rights bind an offer and a take, in the destination region (5.2.1) | 9.3 | MER |
| TRN | no change: `ship` is not offered (5.8.1) | | |
| CRD | the clamp of 6.6 after the request's last check; `CreditRound` is unchanged | 5.2, 6.3 | CRD |
| EXT | no change: the top-up of a counterparty is a budget and what is not reserved is swept unspent (6.3.3) | 5.2, P4.2 | EXT (note) |
| EST | no change: an estate is exempt from a right (5.5.2) and bound by a legal price (6.3.3) | 2.7 | EST (note) |
| SUB | `not_permitted` is a reason of `Blocked` only with `OwnUseBound` | 5.5, 14.1 | SUB |
| WEL | none: a premium that a body raises is measured as any | | |
| WLD | rows P1.9a and P7.2b, the two slots in the table 8.2, the clause of 6.11, no row of draws; the module in 6.1.5; a row of the cadence table 9.2 (the membership round, every tick, P1.9a); the second reader of a count in 9.4.1; a place by seniority and a headroom shared with the class limit and the entry budget among the first-come places of 10.2 | 6, 7.3, 8.2, 9.2, 9.4.1, 10.2 | WLD |
| INT | the two setters, their entries and the sentinels `ErrUnknownControl` and `ErrUnknownRight`; `SlotState` gains `LegalLevels []LegalLevel` and `RightSuspensions []RightSuspension`, in the order the checkpoint saves them (5.3, FR-INT-032); `AgentState` gains the group Membership: the `BodyID` or `NoBody`, true only for a member at the boundary (4.4.1; INT 4.7.2); the typed IDs of a call (3.4.1) gain `BodyID`, `RightID` and `ControlID`; no other query: the definitions are in `Config()` and the state in the snapshot (11.5) | 3.4.1, 4.7.2, 5.1, 5.1.3, 5.3, 11.2 | INT |
| OBS | three events, two slot events, the names table, the snapshot rows, the metrics and the `Warn` row of 11 | 4.1, 5.3, 6, 7.3 | OBS |
| CHK | the membership field of the agent record; the slot section; the restore check | 4.2, 4.3 | CHK |
| CFG | the fields and bounds of 9, the sentinels and warnings of 10, the table `[modules.institutions]` | 3.7, 4.1, 7.3, 7.4, 8 | CFG |
| IND | two preset asks (11.7) | 7 | IND |
| STB | the entries of 14 | 4, 5, 8 | STB |
| CLB | the registry rows of 9 | 6 | CLB |
| REF | the overlay and scenarios of 13.5 | 4.4 | REF |
| FIS, PLC | PLC's `legal` levers set the levels, a consumer sets the suspensions; FIS takes dues and fees if it gives a body an account (open question 1) | | FIS, PLC |

## 9. Parameters

All are fields of `Config` with a bound, a doc line and an owner (architecture.md 9.5 to 9.7),
under the module entry `Modules.Institutions` (a pointer; nil is off, `spec-config.md` 3.2).
The module has **no tunable with a default that moves a result**: the section as a whole
defaults to "no body, no right, no control", which is the world without the module. The
bounds are in `bounds.go` (architecture.md 9.6).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Institutions` | the module; present means on | nil (off) | |
| Institutions | `Bodies` | the bodies | none | at most `MaxBodies` |
| Institutions | `Rights` | the rights | none | at most `MaxRights` |
| Institutions | `Controls` | the price controls | none | at most `MaxControls` (shared with `RateLimits`) |
| Institutions | `RateLimits` | the rate limits | none | as above |
| Institutions | `OwnUseBound` | an own-use job is bound by a `produce` right (5.5.2) | false | a bool |
| Body | `ID` | the identity (3.2.2) | required | `[a-z][a-z0-9_]*`, at most 64 bytes, unique |
| Body | `Roles` | the roles that may be members | required | at least one, defined, no (role, region) in two bodies |
| Body | `Regions` | regions of the scope | empty | defined, not an outside node; with `Jurisdictions`, at least one region in all |
| Body | `Jurisdictions` | jurisdictions whose regions join the scope | empty | defined; needs `Modules.Polity` (`ErrModuleDependency`) |
| Body | `Places` | households per region of the scope; 0 is no limit | 0 | 0 to `MaxHouseholds` |
| Body | `MaxTrainees` | the member's limit on trainees; 0 is the role's alone | 0 | 0 to 100 (SKL's bound) |
| Body | `Needs` | the settlement flag the class must carry | empty | a defined flag; needs `Modules.Settlements` (`ErrModuleDependency`) |
| Right | `ID` | the identity | required | as a body |
| Right | `Body` | the holder; empty is a prohibition | empty | a defined body |
| Right | `Activity` | `produce`, `sell`, `buy` or `train` | required | one of the four |
| Right | `Subject` | a recipe, a commodity or a role | required | defined, of the kind the activity names |
| Right | `Region`, `Jurisdiction` | the scope key | required | exactly one; a region that is not an outside node; a jurisdiction needs the polity module |
| Right | `Who` | whose membership a `produce` right tests | `principal` | `principal`, `executor` or `both`; produce only |
| Control | `ID` | the identity | required | as a body, one namespace with `RateLimit` |
| Control | `Commodity` | the commodity | required | defined |
| Control | `Region`, `Jurisdiction` | the scope key | required | as a right |
| Control | `CapBP` | the cap in basis points of the effective reference price | 0 (none) | 0 to `MaxBasisPoints` |
| Control | `FloorBP` | the floor | 0 (none) | 0 to `CapBP` when a cap is set |
| RateLimit | `ID` | the identity | required | as a control |
| RateLimit | `Region`, `Jurisdiction` | the scope key | required | as a right; needs `Modules.Credit` |
| RateLimit | `CapBP` | the yearly cap, absolute | required | 0 to `MaxCreditRateBP`; 0 is interest-free |

Presentation-only, so exempt from the registry test and not hashed (`spec-config.md` 8.3):
`Label` on a body, a right, a control and a rate limit, at most `MaxLabelBytes` bytes.

**Bounds** (placeholders, N-015): `MaxBodies` 256, `MaxRights` 4 096, `MaxControls` 4 096.
The rest are the world's (`MaxBasisPoints`, `MaxHouseholds`, `MaxCreditRateBP`,
`MaxLabelBytes`).

**Rule constants** (beside their code, architecture.md 9.6): the four activities and the three
values of `Who` (5.1.2); the order of the membership round (lapse, then admit by seniority,
then the key, then the agent ID, 4.3); the rounding of a cap and a floor (6.2.1); the order of
the clamp in admission (6.3.1); the reason `not_permitted`; the position of the round (P1.9a).

**Not parameters:** the `BodyID`, `RightID` and `ControlID`, the resolved arrays and the
ranges in minor units (derived), the members (state, 12), and the slot's levels and
suspensions (state).

The placeholders, so that none is mistaken for a result: `MaxBodies`, `MaxRights`,
`MaxControls`, the numbers of the worked examples of section 13.

## 10. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). Errors name the path, for
example `modules.institutions.rights[1].subject` or `modules.institutions.bodies[0].roles[2]`.
The framework of `spec-config.md` 7 runs them.

### 10.1 Errors

The rules run in this order, so that the first error is the same for the same mistake: syntax
and duplicates; counts; module dependencies; references (roles, regions, jurisdictions,
recipes, commodities, bodies); shapes; overlap; bounds.

| Sentinel | Rejects |
|----------|---------|
| `ErrInvalidID`, `ErrDuplicateID` | an ID that does not match `spec-config.md` 3.3, or twice; two rights with one (activity, subject, scope key); two controls with one (commodity, scope key); two rate limits with one scope key |
| `ErrTooMany` | more than `MaxBodies`, `MaxRights` or `MaxControls` |
| `ErrModuleDependency` | a `Jurisdictions` entry or a right, control or rate limit keyed by a jurisdiction while `Modules.Polity` is absent; a `Needs` while `Modules.Settlements` is absent; a rate limit while `Modules.Credit` is absent |
| `ErrUnknownRole`, `ErrUnknownRegion`, `ErrUnknownJurisdiction`, `ErrUnknownRecipe`, `ErrUnknownCommodity`, `ErrUnknownSettlementFlag` (SET's, 9.1 there) | a reference that names nothing, a `Needs` that no class carries included |
| `ErrUnknownBody` (new) | a right's `Body` that names no body |
| `ErrOutsideNode` | a body's `Regions` entry or a right's, control's or rate limit's `Region` that is an outside node |
| `ErrBodyShape` (new) | a body with no role, or with no region in its scope (no `Regions` and no covering `Jurisdictions`) |
| `ErrBodyOverlap` (new) | a (role, region) that is in the scope of two bodies (3.3.2) |
| `ErrRightShape` (new) | an activity outside the four; a subject of the wrong kind for the activity (a commodity for `produce`, a recipe for `sell`, ...); both or neither of `Region` and `Jurisdiction`; `Who` other than `principal` on an activity other than `produce` |
| `ErrControlShape` (new) | both or neither of `Region` and `Jurisdiction`; a `FloorBP` above a `CapBP` when both are set; a `FloorBP` on a rate limit |
| `ErrOutOfBounds` | a `Places` above `MaxHouseholds` or below 0; a `MaxTrainees` outside 0 to 100; a level above its bound; a label above `MaxLabelBytes` |

The sentinels are root sentinels matched with `errors.Is`; `ErrUnknownControl` and
`ErrUnknownRight` are the slot's (7.1.1). Note N-075 records the mapping for the root.

### 10.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at construction, at
`Warn`, with the attributes `rule` and `path`, through the "configuration warning" line of
`spec-observability.md` 7.3 (row 1); the volume is bounded by the configuration.

| Warning | Condition |
|---------|-----------|
| `body_empty` | none of a body's roles has a group in any region of its scope: it can never have a member (3.3.3) |
| `places_below_start` | a region of a body's scope starts with more households of its roles than `Places`: some start as outsiders (4.3, ORG-A4) |
| `right_unreachable` | a right whose holder body has no eligible household in some region of the right's scope: the activity is closed to everyone there (5.4.4) |
| `right_over_upkeep` | a `buy` right on a commodity that is in the effective upkeep of some role with a group in the scope, or a `sell` right whose subject is the only commodity a role in the scope can sell, or a `produce` right on the only recipe sourcing a staple (5.5.3, ORG-A6) |
| `train_right_no_training` | a `train` right on a role with `TrainingTicks` 0: it has no effect |
| `cap_below_band` | a control's cap in minor units is below the lowest belief of the reference band, `RefBandLow` basis points of the reference price (`spec-beliefs.md` 3.2.1): no ask can reach it, the market freezes (6.5.5) |
| `floor_above_band` | a control's floor is above `RefBandHigh` basis points of the reference price: no bid can reach it |
| `cap_below_seed_cost` | with a warm start, a cap below `CostFloor` of the full unit cost at the warm start's prices and wage of a recipe that makes the commodity in the scope (`spec-production.md` 9.4, `spec-viability.md` 8.2.3): sellers cannot ask what buyers may bid |
| `rate_limit_below_floor` | a rate limit below the lowest posted `RateBP` of every lender in its scope: no loan can form (the mirror of CRD's `rate_above_limits`) |
| `control_inert` | a control or rate limit whose levels are both 0 (or a rate limit's at `MaxCreditRateBP`): informational; it binds nothing until a slot sets it (7.1.3) |
| `body_needs_unmet_at_start` | a body with a `Needs` whose flag the class of no region of its scope carries at the start: it admits no one until a class changes (3.5); informational |

## 11. Observability

### 11.1 Events

Three aggregate events and two slot events. Every field obeys 4.1 of the rules: no map, no
pointer, fixed-point integers, typed IDs, closed constants. Each is constructed only when an
observer was supplied (rules 4.5). The class and the gate are those of `spec-observability.md`
4.1 (A: aggregate, bounded by the configuration; C: one per changed cell of a call between ticks).

| Event | Owner | Position | Class | Gate | Fields (besides `Tick`) |
|-------|-------|----------|-------|------|-------------------------|
| `MembershipChanged` | ORG 4 | P7.2b, (`RegionID`, `BodyID`) ascending | A | institutions | `Region`, `Body`, `Members` (households after the round), `Places` (0 for no limit), `Eligible` (households of the body's roles), `Admitted`, `LapsedRole`, `LapsedRegion`, `Skipped`, `Barred`, `Waiting`, `Held` (completions held at P7.1(a) this tick, 4.5.2); emitted for a (region, body) with any of `Admitted`, `LapsedRole`, `LapsedRegion`, `Skipped`, `Barred`, `Waiting`, `Held` non-zero; never otherwise |
| `RightRefused` | ORG 5.6 | P7.2b, (`RegionID`, `RightID`) ascending | A | institutions | `Region`, `Right`, `Produce` (starts refused), `Sell`, `Buy` (orders refused), `Train` (places not open to a non-member trainer: the `min(f, a)` of 5.6.3, one count a place); emitted for a (region, right) with any count non-zero |
| `PriceControlled` | ORG 6 | P7.2b, (`RegionID`, `ControlID`) ascending | A | institutions | `Region`, `Control`, `Commodity` (`NoCommodity` for a rate limit), `Bids` (orders or requests lowered to a cap), `Asks` (orders raised to a floor), `Units` (the units of the clamped orders; 0 for a rate limit); emitted for a (region, control) with `Bids` or `Asks` non-zero |
| `LegalLevelSet` | ORG 7.1 | call, `ControlID` order | C | institutions | `Control`, `CapBefore`, `CapAfter`, `FloorBefore`, `FloorAfter` (basis points), one per changed key in ascending `ControlID` |
| `RightSuspensionSet` | ORG 7.1 | call, `RightID` order | C | institutions | `Right`, `Suspended` (after), one per changed key in ascending `RightID` |

11.1.1. **What each counter is for.** `Waiting` and `Barred` are the counters of the outsiders and
of a missing flag (N-045, SET R4); `Skipped` of a cohort that did not fit; `Held` of the held
completions (the throttle); `RightRefused` of every refusal of 5.6.3; `PriceControlled` of
every clamp, and with the `MarketCleared` of the same region and commodity (unmatched bid units,
best unmatched ask) it is the shortage signal of a cap. A clamp is not an error and not a
`Warn`: it is the module doing what it is for.

11.1.2. **Counterpart of a `Warn`** (rules 2.1): every `Warn` below has an event or a snapshot
field that tells the same fact (`MembershipChanged.Waiting`, `RightRefused`).

### 11.2 Logs

Constant lowercase messages, typed attributes, `Enabled` guards (rules 3).

- `Info`, once, in the existing "world created" line: `bodies`, `rights` and `controls`
  (counts) join the summary, and `institutions` joins the `modules` list. No other `Info`.
- `Warn`, at construction, one per warning of 10.2, with `rule` and `path`.
- `Warn`, in the tick, one, bounded by the configuration (pattern B of
  `spec-observability.md` 7.3, the tenth tick and every tenth after): "body has waiting
  households", per (region, body), with `region`, `body`, `waiting` and `places`, when the
  waiting count has been above 0 for `warnRun` consecutive ticks. Counterpart:
  `MembershipChanged.Waiting`. A row owed to the table of 7.3 there.
- `Debug`, per entity, guarded: "start refused" (existing line) with the reason
  `not_permitted`; "order rejected" (existing) with `not_permitted`; "place not open"
  (`agent`, `role`, `region`, `reason` = `not_member`).
- No `Error`.

### 11.3 Log keys

Joining the table of `rules/observability-and-logging.md` 3.3.2 (owed rows):

| Key | Meaning |
|-----|---------|
| `body` | body ID |
| `right` | right ID |
| `control` | control or rate limit ID |
| `activity` | one of `produce`, `sell`, `buy`, `train` |
| `places` | a body's places, a count of households (`int64`) |
| `waiting` | eligible non-members, a count of households (`int64`) |
| `bodies`, `rights`, `controls` | counts of each, in the construction summary |

`reason` takes one more value: `not_permitted` (an order or a start) and, as a log-only value,
`not_member` (a place). `cause` of a lapse is `role_changed` or `region_changed`.

### 11.4 The names table and `WorldCreated`

11.4.1. The names table (`spec-observability.md` 3.2.4) gains three slices: `Bodies`,
`Rights` and `Controls`, each indexed by its typed ID in ascending ID order. An event or a row
that names one carries the typed ID and the string is read from the table. The cardinality test
of `spec-observability.md` 9.4 gets the range checks.

11.4.2. `WorldCreated` gains no field of its own beyond the names table it already carries
(11.4.1): the module moves no money and creates no agent, so the rebuild of 4.3.3 there is
unaffected (FR-ORG-029).

### 11.5 Snapshot

11.5.1. **`Snapshot.Bodies`**: one `BodyRow{Region, Body, Members, Eligible, Places}` per
(region, body) of the scope, in ascending (`RegionID`, `BodyID`), households. `Waiting` is
`Eligible - Members` and is derived by the reader. A count of the last tick (`Held`, `Skipped`,
`Admitted`) is an event and not a snapshot field (N-125).

11.5.2. **`Snapshot.LegalLevels`**: one `LegalRow{Control, CapBP, FloorBP}` for each control or
rate limit whose level differs from the configured one, ascending. **`Snapshot.Suspended`**: the
`RightID`s that are suspended, ascending. All are bounded by the configuration. They are the
slot's state (12) as a snapshot is a function of saved state (N-125).

### 11.6 Metrics

New instruments, in the naming of rules 5.1 (every one derivable from an event or a snapshot
field, rules 5.3.1 end):

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.body.members` | Gauge | `{household}` | `economysim.body`, `economysim.region` | `Snapshot.Bodies[].Members`, by snapshot |
| `economysim.body.waiting` | Gauge | `{household}` | `economysim.body`, `economysim.region` | `Snapshot.Bodies[]` (`Eligible - Members`), by snapshot |
| `economysim.right.refusals` | Counter | `{refusal}` | `economysim.right`, `economysim.activity` | `RightRefused` (`Produce`, `Sell`, `Buy`, `Train`, by activity) |
| `economysim.control.clamped` | Counter | `{clamp}` | `economysim.control`, `economysim.side` | `PriceControlled` (`Bids` as `bid`, `Asks` as `ask`) |

11.6.1. New attributes, each **bounded by the configuration** (rules 5.4.1): `economysim.body`
(bodies), `economysim.right` (rights), `economysim.control` (controls and rate limits),
`economysim.activity` (the four constants); `economysim.side` and `economysim.region` exist.
`economysim.reason` of `economysim.market.orders.rejected` gains `not_permitted`. Series for the
reference chain: bodies times regions, rights times four, controls times two: bounded by the
counts of the pack.

### 11.7 Indicators asked of IND

Two preset asks (`spec-indicators.md`; none is a field of the engine, N-130):

- **`BodyShare`**: per (region, body), `Members / Eligible` in basis points and `Members /
  Places`, from `Snapshot.Bodies`. It is the measure of how closed a craft is.
- **`LegalPriceGap`**: per (region, commodity) with a control in force, the ratio in basis
  points of the published price to the cap (or floor), with the price age, and the unmatched
  bid units of `MarketCleared`: the measure of a binding cap and of a stale price above it
  (6.5.3).

An aggregate over the regions of a jurisdiction is a fold of the aggregator whose input is a
set of regions, which POL asked for (`spec-polity.md` 15).

## 12. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's:

12.1. **A hosted per-agent field, saved once, in the agent record** (tag 3, N-128): the
**membership** of the household, the pair (`BodyID`, `RegionID`) of its admission (3.6,
4.2.1), saved as `BodyID + 1` and `RegionID + 1` with 0 for none, two varints per agent
(`spec-checkpoint.md` 4.2, tag 3, gains the field in the list of what the agent record
carries). The field is written **only with the module** (`Modules.Institutions`), and then for every agent, 0 and 0 for an agent with no membership, so that the record's shape depends on the module and on nothing in the run's history (FR-CHK-041, `spec-checkpoint.md` 4.5, F3).

12.2. **The slot state**, one new section of the frame, present with `Modules.Institutions`
(with zero rows when nothing differs from the configuration, so that its shape does not depend
on a run's history) and placed in the dependency order of `spec-checkpoint.md` 4.2.1, after the
sections of the modules whose IDs it names (settlements, polity, skill, and credit when a rate
limit is declared); its tag is 31 (`spec-checkpoint.md` 4.2). It holds the levels that differ from the configured ones, as (`ControlID`, `CapBP`,
`FloorBP`) in ascending `ControlID`, and the suspended rights as `RightID` in ascending order;
and, for the `Warn` of 11.2, the run of consecutive ticks with waiting households per
(region, body) that has one (diagnostic: it affects no result, and is saved so that a resumed
run logs as the original did).

12.3. **What a restore checks**, in the fixed order of `spec-checkpoint.md` 6.2, after the
agents are in: each membership field is 0 and 0 or the (`BodyID + 1`, `RegionID + 1`) of a defined body and a region of its scope (one of the two zero is corrupt); each
(`ControlID`) of the slot state is defined and its levels are in bound; each `RightID` is
defined; IDs ascend. A failure is `ErrCheckpointCorrupt` naming the tag and the entry. The
membership of a household that is no longer eligible is **legal** (the round of the next tick
lapses it, 4.2.1): it is not corrupt.

12.4. **Not saved** (derived, and a row owed to `spec-checkpoint.md` 4.3 for FR-CHK-045): the
resolved rights, the legal ranges in minor units, the scope regions of a body, the headroom, the
scratch counts of the tick. A world restored between ticks answers every read of 5 and 6 as an
uninterrupted one does. The configuration hash covers `Modules.Institutions` in canonical order,
every field except the labels, so a change of a body, a right, a control or `OwnUseBound`
changes the hash; the slot levels and the suspensions are state, **not hashed** (8.1.1 item 5 of
`spec-world.md`).

## 13. Worked examples

Four settings, flagged as **examples**. Names and numbers are placeholders; the tables are the
output of `models/org/clamp.py` and `models/org/admit.py` (section 15) or plain integer
arithmetic shown. Order of IDs is by ID string, bytewise.

### 13.1 Example A: a craft guild in a town (an early-modern-like setting)

One region `town`; a role `weaver` that holds the recipe `weave_cloth` (it requires a tool and
has `TrainingTicks` above 0); a role `spinner` that is on no body's list.

```toml
[modules.institutions.body.weavers]
roles        = ["weaver"]
regions      = ["town"]
places       = 12
max_trainees = 2

[modules.institutions.right.weave_in_town]
body     = "weavers"
activity = "produce"
subject  = "weave_cloth"
region   = "town"

[modules.institutions.right.train_weavers]
body     = "weavers"
activity = "train"
subject  = "weaver"
region   = "town"
```

**The round at tick 0.** Fourteen households hold the role: twelve founders (`RoleSince` 0) and
two arrivals of a migration (since 5 and 9; the example is staged, the later ones would arrive
at P1.5 of ticks 5 and 9). At the first P1.9a the founders are ordered by the key, then admitted
until the places are used; the arrivals are not yet households. (The model gives the arrivals
as present at tick 0 to show the order.)

| Step | Households of the role | Members | Free places | Admitted | Skipped | Waiting | Headroom |
|------|-----------------------:|--------:|------------:|---------:|--------:|--------:|---------:|
| tick 0 (P1.9a) | 14 | 12 | 0 | 12 (the founders) | 2 | 2 (since 5 and 9) | 0 |
| tick 4, a member removed at P7.1 of tick 3 (P1.9a) | 13 | 12 | 0 | 1 (the household since 5) | 1 | 1 (since 9) | 0 |

(`admit.out`: "tick 0: lapsed 0 admitted 12 skipped 2 waiting 2"; "tick 4, one member removed:
lapsed 0 admitted 1 skipped 1 waiting 1".) The household since 9 still waits: it is the
youngest. `places_below_start` is raised at construction for the 14 against 12.

**What the households meet.** A weaver outsider (waiting) asks `Work` for `weave_cloth` at P2.5
and is refused `not_permitted`: nothing is consumed, it earns nothing, its `Privation` rises and
the failure ladder may switch it to another role (rung 1) or relieve it. The 12 members start
as ever. A trainee under a member (a `train` right is in force: only members have places, 5.7.1)
runs the eligible recipes at `TraineeYield`; at the end of its `TrainingTicks` the region's
headroom is 0 (13 households against 12 places), so its completion is **held** (4.5.2): it
stays a trainee and holds one of its trainer's two places (`max_trainees` 2). When two more
households of the role leave, 11 remain, the headroom is 1 at the next entry step and the
trainee completes in step (a), before an entrant of step (d) can take the place. The
households of the role are then 12, the places: at the next P1.9a every eligible non-member
fits, the waiting household since 9 and the new weaver (its `RoleSince` is the tick of
completion, so it is the younger), and both are admitted. The numbers of waiting households
and held trainees are `MembershipChanged.Waiting` and `.Held`.

### 13.2 Example B: an assize of bread (a price cap, an early-modern-like setting)

A commodity `bread` with an effective reference price of 1 000 minor units in the region
`shire`; a control keyed by a jurisdiction that covers it:

```toml
[modules.institutions.control.bread_assize]
commodity    = "bread"
jurisdiction = "crown"
cap_bp       = 12000
```

The cap is `max(1, 1 000 * 12 000 / 10 000) = 1 200`. Six bids (limit, units) and five asks:
bids (1 500, 3), (1 400, 2), (1 250, 4), (1 200, 3), (1 100, 5), (900, 2); asks (800, 4),
(1 000, 4), (1 150, 3), (1 300, 5), (1 500, 4). The three bids above 1 200 are clamped to it
(`PriceControlled.Bids` = 3, `Units` = 9).

| | Trades (price, units) | Units | Volume-weighted price | Unmatched bid units | Unmatched ask units |
|--|-----------------------|------:|----------------------:|--------------------:|--------------------:|
| no law | (1 150, 3), (1 100, 1), (1 200, 1), (1 125, 3), (1 200, 1), (1 175, 2) | 11 | 1 152 | 8 | 9 |
| cap 12 000 bp (1 200) | (1 000, 3), (1 000, 1), (1 100, 1), (1 100, 3), (1 175, 3) | 11 | 1 084 | 8 | 9 |
| cap 9 000 bp (900) | (850, 3), (850, 1) | 4 | 850 | 15 | 16 |

(`clamp.out`; the volume-weighted price of the first row is `12 675 / 11 = 1 152`.) The cap of 1
200 moves the price from 1 152 to 1 084 and no unit is lost: 12 units are wanted at 1 200 or
above and 11 offered at or below it. The cap of 900 is below four of the five asks and the
volume falls from 11 to 4 units: 15 bid units go unmatched at the cap where 8 did with no law.
That is the shortage of ORG-A1, and `PriceControlled` with `MarketCleared` reports it.

### 13.3 Example C: a law against interest (a rate limit)

```toml
[modules.institutions.rate_limit.interest_law]
jurisdiction = "crown"
cap_bp       = 1200        # yearly, absolute
```

Three requests with `MaxRateBP` 2 000, 1 500 and 1 000, and two lenders whose effective rates
are 1 100 (principal 400) and 1 400 (principal 600). With the cap the requests carry 1 200,
1 200 and 1 000: the lender at 1 100 matches the first two (it is at or below their limits) and
**not** the third (1 100 is above 1 000); the lender at 1 400 matches none. Without the cap the
1 400 lender would match the first (2 000) and the second (1 500). The interest of a claim of
1 000 over 73 ticks at the capped rate with `TicksPerYear` 365 is
`1 000 * 1 200 * 73 / (10 000 * 365) = 87 600 000 / 3 650 000 = 24` exactly (CRD 7.1: no
rounding arises).

### 13.4 Example D: a licence and a prohibition (a space setting; a household is a crew)

A jurisdiction `station` over the regions `deck` and `dock`. A body `licensed` with the roles
`vendor` and `hauler`, `Places` 0 (no limit), scope `station`. A right `sell` on the commodity
`reactor_fuel` keyed by `station` held by `licensed`; a right `sell` on the commodity
`spirits` keyed by `station` with **no body** (a prohibition).

- A household of the role `crew` asks to sell `reactor_fuel`: refused `not_permitted` (it is
  not on the list, so it is nobody's member). It may hold and burn it.
- A `vendor` is admitted at the first P1.9a (no limit): every eligible household is a member.
- Any order to sell `spirits` is refused `not_permitted`, whoever places it; an estate's lot of
  it is exempt (5.5.2) and sells.
- A `sell` right is not a `buy` right: nothing bars a crew from buying `reactor_fuel`.

### 13.5 Use by the reference chain (a proposal for REF)

`overlay/institutions.toml` for `five` (`spec-reference-chain.md` 4.4.4): a body over the one
skilled role of the hub region with places at 80% of its starting households (so the module has
outsiders at construction and the scenarios ORG-S2 and ORG-S3 have something to count), a
`produce` right and a `train` right; and a control keyed by the jurisdiction of two provinces
over one staple, at levels given by ORG-S1's grid. A second overlay,
`overlay/institutions-inert.toml`, declares the same bodies, rights and controls with `Places`
0 and both levels 0, so that ORG-S5 asserts neutrality. REF builds them; this spec only names
them.

## 14. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9 (FR-STB-003). Fields as in its
section 3; the register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn` numbers and the
entries here are `ORG-An`, `ORG-Dn` and `ORG-Sn`. **The module adds no closed chain of its own
that a household's decision drives; it adds constraints, and a constraint that binds turns a
price or a place into a quantity that is not served.** Each has an entry. Nothing here has been
run in the library; the status `model` means the rule was run in `models/org/` (15) and no
default is claimed stable (N-027).

**Merges** (checklist 9.9): ORG-A1 shares its supply response with L36 and L21 and is a new
entry; ORG-A8 merges into L121 (a consumer closes a loop through the library's slots) as a case.
ORG-A1 to A7 are new.

Each entry gives the **links with their signs** (STB 3: `(+)` a rise in the cause raises the
effect, `(-)` lowers it, `(0)` the link is cut), the polarity as the product of the signs of a
closed chain (an odd number of `(-)` is balancing, B; even is reinforcing, R), the delay, and the
condition under which the entry matters. The signs are a hand reading of the rules named in each
entry; none has been run in the library (status).

**ORG-A1. Shortage under a legal price.** *Kind:* loop whose closing link is cut while the law
binds. *Links:* price `p` (+) margin of a seller `m = p - c`; `m` (+) income of the role `y`;
`y` (+) households that stay in the role `n` (the exit of the failure ladder and the production
halt D24 act below the margin the role needs); `n` (+) supply `S`; `S` (-) unmatched demand `U`;
`U` (+) the price the market would form `p*` (the evidence and stock rules of
`spec-beliefs.md` 5); `p*` (+) `p = min(p*, cap)`. Without a cap the product is (-), B: the
occupation cobweb of L36. **With the cap binding the last link is `(0)`**: `p` stays at the cap,
`m` is a constant, the loop is open and the outcome is a **level**, `U = D - S(cap)`, not a
cycle, with a **cliff** where `m` falls below what the role needs (its households then leave one
way: entry is slow and capped, `spec-spawner.md` 11, so the deficit outlasts the cap by the entry
delay after a lift). The floor is the mirror: asks are raised, unmatched asks build, demand
falls to the floor price (a glut) and the halt acts. *Delay:* the job's duration plus the
exit's review and privation window (`spec-spawner.md` 5, 6), near the welfare window (L36, L21).
*Condition:* a cap at or above `CostFloor` of the full unit cost at the flow-balancing prices
(`spec-viability.md`), for a good with no outside source. *Owner:* ORG 6. *Damper:* none in the
engine for a cap below the long-run cost: the guarantee is conservation and determinism, not
stability (ADR 0001). What bounds it: the clamp rather than a rejection (ORG-D1: the volume
stays the units that can cross), the exit cap `ExitShare`, the production halt (D24), the
warnings `cap_below_band` and `cap_below_seed_cost`, and the slot (7.1), which a ruler or a
controller outside the engine uses (ORG-A8, L121). *Test:* ORG-S1. *Status:* hand for the
signs and the level; `model` for the volume at a cap (`clamp.py` P2); untested for the dynamics.

**ORG-A2. The shadow price and the jump at the lift.** *Kind:* bias (a stored deviation: no
loop while the law binds). *Links:* unmatched asks above the clamped limit (+) buyers' centre
`c` (the evidence rule, `spec-beliefs.md` 5.4); stock `s` (-) `c` (the stock rule, 5.5); `c`
(+) the drawn limit `l`; `l` (+) the admitted limit `min(l, cap)` only while `l` is below the
cap and `(0)` above it; the admitted limit (+) the trade price. While the cap binds `c` has no
effect on the price: `c - p` grows to `RefBandHigh` and stays, a bounded integrator. At the lift
the `(0)` link opens and the stored deviation is released into the belief loops (L01, L03).
*Delay:* belief steps (`BeliefStep`, `2 * BeliefStep` an update). *Condition:* a cap that binds
for longer than the beliefs take to reach the band, then lifted in one step. *Owner:* ORG 6.5.4;
BEL 5. *Damper:* the step bound and the band; a ruler that raises the cap in steps lets the
beliefs follow. *Test:* ORG-S1 (the lift leg). *Status:* hand.

**ORG-A3. Rent, the queue and the held completion.** *Kind:* loop (two balancing chains and a
cliff). *Links, chain 1 (the queue, L41):* places `P` (+) the households of the role `H`;
`H` (-) the premium `pi` observed above the derived one, a scarcity rent; `pi` (+) applications `a`
(`spec-skill-barriers.md` 4.5); `a` (+) the queue ahead; the queue (+) the expected wait `q_w`;
`q_w` (-) the appeal of applying (the discount of 4.5); the appeal (+) `a`. Product (-): B.
*Links, chain 2 (the hold):* completions `c` (+) held trainees `h` when `H` is at `P`; `h` (+)
trainers' places occupied; occupied places (-) intake `i`; `i` (+) later completions `c`.
Product (-): B, with the delay `TrainingTicks`. *The cliff:* a held trainee yields only
`TraineeYield` and eats, so the trainer's gain `g` (4.3 there) (-) `h`; `g < 0` closes the places
(L37) and intake stops. The **hold is a link of chain 2**, not a damper of this loop (ORG-D3 is
a damper of ORG-A4, below). *Delay:* `TrainingTicks` plus the wait. *Condition:* a body with
places that bind and a role with a premium, with `train` rights or `MaxTrainees` set. *Owner:*
ORG 4.5; SKL 4. *Damper:* D20 (the throttle), `MaxTrainees`, the queue discount. *Test:* ORG-S2.
*Status:* hand.

**ORG-A4. Outsiders.** *Kind:* hazard. *Links:* outsiders `w` (households eligible and not
admitted) (+) privation; privation (+) switches and exits by the ladder; exits (-) `w`: B, slow.
A second chain through migration: arrivals into a full body `A` (+) `w`; `w` (+) a low earned
ratio; a low ratio (+) pushed households (MIG 5); pushed (+) departures; departures (+) arrivals
`A` of another region. It closes, with product (+), R, only if a second full body is reachable:
a household cycling between two closed bodies' regions (MIG's estimate does not read a body,
3.6.2). *Delay:* the settling rule (`MoveCooldown`), the privation window. *Condition:* the
migration module on and at least two regions with a full body. *Owner:* ORG 3.6, 4.4.2;
MIG 5.2. *Damper:* MIG's settling rule and its outflow and pipeline caps, the ladder's
`ExitShare`, and ORG-D3 (the held completion makes no skilled household that cannot work);
counted by `Waiting` and the `Warn` of 11.2. *Undamped:* an external agent that is an outsider
(the consumer decides). *Test:* ORG-S3. *Status:* untested.

**ORG-A5. A place goes to the incumbent.** *Kind:* hazard (a first-come cap of the kind of
N-122, `spec-world.md` 10.2, with a reinforcing tail). *Links:* seniority `s` (+) admission
rank; rank (+) membership; membership (+) the income of a member (the premium); income (+) the
household stays in the role (the ladder is not reached); staying (+) `s`, since `RoleSince` is
kept: product (+), R for the incumbent; among equal `RoleSince` the key breaks the tie
(WLD 10.1.1). *Delay:* the tenure itself. *Condition:* places that bind and a premium.
*Owner:* ORG 4.3.3. *Damper:* none by design (a body's admission is seniority); counted by
`Skipped` and `Waiting`, and the measure `BodyShare` (11.7) with the age of the waiting is the
report. *Test:* ORG-S3. *Status:* hand.

**ORG-A6. A gate on food.** *Kind:* hazard. *Links:* a `sell` right on a staple in force (+) sellers
barred; barred sellers (-) supply `S` of the staple; `S` (-) the staple's price `p` (less supply, a
higher price); `p` (+) unfed buyers; unfed (+) exits and deaths by the ladder and POP; exits (-)
buyers; buyers (+) demand; demand (+) `p`. The chain from `p` closes through the loss of
households: product (-), B, at a cost in households.
A `buy` right is the same with the buyers barred at once. *Delay:* the privation window and
the ladder. *Condition:* the right held by a body with places, or by nobody, over a staple or the
only recipe that sources one (SET R6). *Owner:* ORG 5.5.3. *Damper:* the own-use exemption
(`OwnUseBound` false, ORG-D4: a household may feed itself), the warning `right_over_upkeep`;
the failure regime of famine (STY F-regimes) is the detector. *Undamped:* a pack that bars a
staple on purpose. *Test:* ORG-S4. *Status:* untested.

**ORG-A7. A legal price that the book alone binds.** *Kind:* hazard (a leak, not a loop).
*Links:* a cap on the book (+) the gap between a contract's price and the cap; the gap (+)
volume moved to delivery contracts (MER 9.4); contract volume (-) book volume. No closed chain.
*Delay:* none (the same tick). *Condition:* the merchants module on and a cap in force. *Owner:*
ORG 6.7; MER 9.3. *Damper:* the clamp in the contract round's admission (owed to MER, row 15).
*Test:* ORG-S1 with the trade overlay. *Status:* hand.

**ORG-A8. A controller on the slots** (a case of L121). *Kind:* hazard. *Links:* unmatched units
`U` (+) a controller's raise of the cap `kappa`; `kappa` (+) the price `p`; `p` (+) supply `S`
after the delay of ORG-A1; `S` (-) `U`. Product (-): B, with the delay of the supply response,
so a gain high enough overshoots and cycles. *Delay:* that of ORG-A1 plus the controller's own
sampling. *Condition:* a consumer, or PLC, that reads `U` or the price age and sets
`SetLegalLevels`. *Owner:* INT 5.2; PLC. *Damper:* none in the engine (L121, OBS-A1). *Status:* hand.

### 14.1 Dampers this spec adds or specifies

| ID | Damper | Kind | For | Status |
|----|--------|------|-----|--------|
| ORG-D1 | the clamp, not a rejection: a legal price keeps every buyer and seller whose limit can still cross (6.3, 6.4) | rule | A1, A2 | specified |
| ORG-D2 | places as a ceiling on the role, so entry is limited at the source (3.4.2, 3.4.3, 4.5.1) | data | A3, A4 | specified |
| ORG-D3 | the held completion: it makes no skilled household that has no place to work by its own doing (4.5.2; an arrival or an external agent can still fill the place first, ORG-A4) | rule | A4 (a damper); a link of A3, not its damper | specified |
| ORG-D4 | `OwnUseBound` false: an own-use job is exempt from a `produce` right (5.5.2) | data, default off | A6 | specified |
| ORG-D5 | a right is read at a start, never at a continuation or a completion (5.5.1) | rule | A3, A4, A6 | specified |

### 14.2 Delays near neighbours (checklist 9.6)

The admission round is every tick and has no delay of its own beyond P1.9a's position
(one tick after a role change, WLD-A1). The delays that matter are those of the loops above:
the held completion waits for a vacancy, with no bound of its own (it is as long as the
vacancy flow, and near `TrainingTicks` times the throttle, D20); the exit of ORG-A1 is near
the welfare window `T` and `ReviewTicks` (D33), where a cycle can form (L23). A cap
introduced into a market whose beliefs are mid-swing meets the cobweb of L21.

### 14.3 Scenarios proposed

| ID | Scenario | Pass |
|----|----------|------|
| ORG-S1 | `one` or `three` with a cap on a staple at 80%, 100% and 120% of the solver's price, lifted at tick `T`; and with the trade overlay (contracts) | invariants over seeds (every trade price at or below the cap, conservation, `WithAudit`); the unmatched bid units, the volume, the price after the lift and the recovery time are **reported**, not asserted |
| ORG-S2 | the body overlay (13.5) over a long run | the observed premium is above the derived one (reported); held completions bounded by the trainers' places; no negative headroom admitted |
| ORG-S3 | a start with households above the places; migration on | `Waiting` falls to 0 or is bounded by the ladder; no household removed for exceeding the places |
| ORG-S4 | a prohibition of a staple's sale with `OwnUseBound` false and true, subsistence on | the famine regime is detected when it should and the exempt household feeds itself |
| ORG-S5 | the inert overlay against the world without the module, seeds 1 to 10 | every event, snapshot field and golden line identical but the names table, the configuration hash, `Snapshot.Modules` and the construction summary (exact); a harness test of the class of S77 and S78, outside the band rule of FR-STY-098 |

## 15. What was run

`models/org/` (Docker, standard library only, `PYTHONDONTWRITEBYTECODE=1`, a read-only mount;
the commands are in its README):

- `clamp.py`: the default clearing (sort, cross, midpoint rounded half to even) over **20 000
  random books** (1 to 12 bids and asks of 1 to 5 units, limits 1 to three times a reference
  price, a cap in 80% of them, a floor in 30%, both together in some) with the clamp of 6.3.
  Every trade price was inside the legal range (P1, P4), every trade price was between its two
  clamped limits (P3), and the volume equalled an independent count (the k-th highest clamped
  bid against the k-th lowest ask) and was never below the volume with the over-cap bids
  rejected (P2): 125 849 units with the clamp against 66 003 with rejection and 161 863 with no
  law, over 66 461 trades. P5: over 50 000 random (reference price, cap, floor) up to
  `MaxPrice` and `MaxBasisPoints`, the range of 6.2.1 was in `[1, MaxPrice]`, a floor never
  exceeded its cap and a limit times `MaxQuantity` fitted `int64` (14 424 of them would have been
  above `MaxPrice` without the `min`). The arithmetic of example B (13.2).
- `admit.py`: the membership round of 4.2 and 4.3 over **3 000 random populations of a body with
  three regions in its scope** (up to 40 households of weight 1 to 3, places 1 to 30, a fourth
  region outside the scope), eight rounds each with removals, role changes and moves of a member
  inside and out of the scope in between, with no weight change (4.2.2) (15 108 moves of a member inside the scope): from a start
  with no member the members of a region never exceeded the places, and every member was eligible
  and in the region it was admitted in (A1; with the lapse by region removed, the same check fails
  at the first case); the result did not depend on the order of the list (A2) and a second round
  changed nothing (A3), both also from arbitrary start states; the key is the SplitMix64
  finaliser (A4, the first output of the generator for seed 0). A5 runs **3 000 random runs of 40
  ticks of the held completion** (removals, arrivals that bypass the headroom, trainees due,
  completions in queue order, then entrants): 44 654 completions, each only with headroom of 1 and
  none took the households above the places, in the application order, before an entrant of the
  same entry step, and 110 708 ticks had a trainee held. The arithmetic of example A (13.1) and a
  move inside a scope of two regions (the member lapses and is skipped at the full region).

**What they do not show.** They check that the rules are what 4 and 6 say and have the
properties claimed. They measure no cost (the cost is counted in 19.1), say nothing about
any economy and are not evidence that any cap, place or right is stable: the dynamics of the entries
of 14 were not modelled and are untested.

## 16. Notes settled

| Note | What this spec does |
|------|---------------------|
| N-016 (the parameter registry) | **Applied.** Section 9 adds the rows and no unnamed parameter; each has a default or "required", a bound and an owner. The registry test is still to be written |
| N-017 (words and names) | **Applied.** Terms fixed: body, member, outsider, place (of a body), headroom (of a body), seniority, eligible, right, activity, subject, scope key, prohibition, in force, suspended, `Who`, principal, executor (as PRO), control, rate limit, legal range, cap, floor, clamp, legal price, membership round, held completion, need. Setting words appear only in the flagged examples |
| N-027 (no default is stable) | **Applied.** No default macro constant; `OwnUseBound` false and `Places` 0 are structural; the models check rules |
| N-045 (silent dampers need counters) | **Applied.** Every refusal, clamp, skip, bar, wait and hold is counted (11.1) |
| N-090 (the rule for a module that keeps an account in a region) | **Applied, answered by no registration**, with the reason (3.6): a membership is a status. The rule stands for RSK and the others; add ORG to the list of modules that registered none |
| N-091 (the slots INT must expose) | **Applied: two** (7.1), with rows owed to INT 5.1 and WLD 8.2 |
| N-103 (running writers in parallel) | **Followed, with one slip.** Only the spec, `models/org/` and `.supervisor/org.*` were created; every model ran in Docker with no bytecode; the host `python3` was invoked once, as `python3 --version` with its output discarded, by mistake in a shell line that edited the spec (nothing was read or written by it); and once more, in the revision after review, as `python3 -c pass` with its output discarded (it executed nothing and read or wrote nothing) |
| N-104 (region-keyed tables use one mechanism) | **Applied.** Rights and controls are rows keyed by (activity or commodity, subject, scope key), one row per key, replace mode, canonical order, with the sentinels of `spec-regions.md` 5.3; a row is owed to its 5.1.3 list |
| N-107 (new faucets and sinks start from MON) | **Applied: none.** The module moves no money or goods and adds no reason, flow or account |
| N-117 (the agent ID space) | **Applied.** A body is not an agent; no ID shifts (position 2) |
| N-122 (first-come caps) | **Applied** as a registered hazard (ORG-A5): admission by seniority is a first-come cap among the equal; the remedy of WLD 10.1 applies to the tie |
| N-124 (a paired comparison is exact only while aligned) | **Applied.** A module on with everything inert is aligned; a refusal removes an order and so moves D6, D7 and D8, and a clamp changes no count: a comparison with a binding right is unaligned from the first refusal (ORG-N5) |
| N-125 (a snapshot is a function of saved state) | **Applied.** `BodyRow` is derived from the agent record and the roles; `LegalLevels` and `Suspended` are the slot's saved state (11.5) |
| N-126 (a new event or `Warn` adds a row) | **Applied.** The rows are owed to `spec-observability.md` 4.1 and 7.3 (11.1, 11.2) |
| N-128 (a hosted per-agent field is saved once) | **Applied.** The membership is one field of the agent record (12.1) |
| N-131 (not in version 1) | **Extended:** a right over carriage (`ship`), over a store or a facility, a right that names a household, a wage control, an admission or an expulsion by call (5.8, 6.8, 7.1.3) |
| N-110 (a household reacts to no regional aggregate) | **Read:** nothing in the module is read by a household beyond its own refusal; a body's counts are measures |
| N-046 (register entries owed) | **Applied** for this spec (14); ORG was not in the areas of the note |
| N-015 (placeholders) | **Applied:** every bound and number of 9 and 13 is a placeholder |

**New notes** (areas in brackets; to add in the integration): (ORG-N1) a legal price is a
clamp of one side's limit in the admission of a market, and any later spec that forms a price
from two limits (a rent, a wage, a fee, a tariff) applies the same clamp at its admission
[LND, EMP, FIS, PLC, MER]; (ORG-N2) a right is read at a start, an order and a place, never at
a continuation, a completion or a holding [PRO, EMP, LND, STO, CAP]; (ORG-N3) a body's places
are a second limit into the headroom that SET's reader gives the spawner and POP [SET, SPN,
POP]; (ORG-N4) a completion of a traineeship is held without headroom [SKL, SPN]; (ORG-N5) a
paired comparison with a binding right is unaligned from the first refusal [EXP, CLB, STB];
(ORG-N6) reserved and not offered: a right over carriage, a store or a facility, a wage
control, a holder that is a named household [TRN, STO, CAP, EMP, PLC, INT].

**Asks of other specs** (not adopted by this change; in the handoff): see section 8 and the
edits table of the handoff.

## 17. Requirements

IDs are stable (testing.md 4.1). A requirement is a single testable statement.

### Functional

| ID | Requirement |
|----|-------------|
| FR-ORG-001 | With `Modules.Institutions` nil, or present with no body, right, control or rate limit and `OwnUseBound` false, no result, event or snapshot field of any world MAY differ from a world without the module except the names table, the configuration hash, `Snapshot.Modules` and the construction summary (3, 9). |
| FR-ORG-002 | The module MUST be on if and only if `Modules.Institutions` is non-nil; a jurisdiction, a `Needs` or a rate limit named while the module it needs is off MUST be `ErrModuleDependency` (9, 10.1). |
| FR-ORG-003 | Bodies MUST be sorted by ID bytewise before a world is built or a hash is taken, the `BodyID` MUST be the index in that order, and every iteration over bodies MUST be ascending by it (3.2). |
| FR-ORG-004 | A (role, region) MUST be in the scope of at most one body; two MUST be `ErrBodyOverlap` (3.3.2). |
| FR-ORG-005 | A household agent MUST be eligible for a body exactly when its role is on the body's list and its region is in its scope; a pseudo-agent MUST never be eligible, an external agent MUST be (3.3.1). |
| FR-ORG-006 | `Places` MUST count households (the sum of weights), 0 MUST mean no limit, and no household MUST be removed or demoted because the households of the body's roles exceed the places (3.4); from a start with no member, the members of a body in a region MUST never exceed `Places` after a round, whatever the removals, role changes and moves between the regions of the scope in between (3.6, 4.3; with no change of weight in between, 4.2.2; `models/org/admit.py` A1). |
| FR-ORG-007 | The headroom of a role in a region MUST be `max(0, Places - households of the body's roles in the region)`, and, for a role that also has a class limit, MUST be the least of the two (3.4.2, 3.4.3). |
| FR-ORG-008 | A body with a `Needs` whose flag the region's class lacks MUST admit no one there, MUST count the eligible non-members `Barred`, and MUST remove no member and suspend no right (3.5). |
| FR-ORG-009 | The module MUST register no `region_bound` refusal (3.6). |
| FR-ORG-010 | A household MUST be a member of body `b` at a read exactly when its membership field is `(b, r)`, `r` is its region at that moment and it is eligible for `b` (4.4.1). |
| FR-ORG-011 | The round MUST first clear the membership of every household that is not eligible for its body or is no longer in the region it was admitted in, and then admit the eligible non-members in the order of `RoleSince` ascending, then `Key(tick, agent)` ascending, then agent ID (4.2, 4.3). |
| FR-ORG-012 | A candidate MUST be admitted whole and only if its weight is at most the free places; one that does not fit MUST be skipped and counted, and the round MUST go on (4.3.4). |
| FR-ORG-013 | The round MUST run at P1.9a and nowhere else; there MUST be no round at construction, and a job in progress MUST run to its completion whoever its household is (4.1). |
| FR-ORG-014 | A right MUST be one row per (activity, subject, scope key); the right in force in a region MUST be the row keyed by the region, else the first row of its chain, else none, resolved once and again at a slot call (5.4). |
| FR-ORG-015 | A suspended right MUST leave its key open, and no outer right MUST be consulted in its place (5.4.3). |
| FR-ORG-016 | A right with no body MUST be a prohibition: nobody but a pseudo-agent MAY do the activity in its scope (5.1.1, 5.5.2). |
| FR-ORG-017 | The module MUST take no word from the world's random source; with the module on and nothing in force (the inert overlay) the stream MUST be that of the world without it; and the tie key MUST be the stateless `Key(tick, agent)` of `spec-resources.md` 5.3.3 (4.3.2, 7.3.4). |
| FR-ORG-018 | A right MUST be read only at a start of a job, at an order and at a place of a traineeship; a continuation, a completion, a holding, a consumption and a shipment MUST NOT be refused by it (5.5.1). |
| FR-ORG-019 | A second round with nothing changed in between MUST admit no one and clear no one, and the result of a round MUST NOT depend on the order of the list of agents it is given (4.3.5). |
| FR-ORG-020 | A start of a recipe with a right in force MUST be refused with `not_permitted` immediately after `not_on_role` and before `k0` is derived, nothing MUST be consumed, and the actor tested MUST be the principal, the executor or both by `Who` (5.2, 5.3). |
| FR-ORG-021 | The executor of a traineeship MUST NOT be tested by `executor` or `both`; the trainer MUST be tested as principal (5.3.2). |
| FR-ORG-022 | An own-use job MUST be exempt from a `produce` right unless `OwnUseBound` is set (5.5.2). |
| FR-ORG-023 | A pseudo-agent (an estate, a counterparty, a treasury) MUST NOT be refused by a right (5.5.2). |
| FR-ORG-024 | An ask or a bid of a household for a commodity under a `sell` or a `buy` right it does not hold MUST be rejected by the world while it builds the orders, with the reason `not_permitted`, after `not_tradable` and before the market's checks; it MUST have no outcome (5.6.2). |
| FR-ORG-025 | With a `train` right in force a household that is not a member MUST have no place for a trainee; a traineeship in progress MUST NOT end for it; the count `trainers` of the application rule MUST be the households that can take a trainee (5.7.1, 5.7.2). |
| FR-ORG-026 | A member MUST hold at most `min(MaxTrainees of its role, MaxTrainees of its body)` trainees, a body's 0 meaning the role's alone (5.7.3). |
| FR-ORG-027 | A trainee whose ticks served reached `TrainingTicks` MUST complete only if the target role has headroom of at least its weight; otherwise it MUST stay a trainee, count `Held`, and complete at the first entry step in which the headroom is at least its weight, in queue order (4.5.2). |
| FR-ORG-028 | The legal range in force for a commodity in a region MUST be the control keyed by the region, else the first of its chain, else none; the cap and the floor in minor units MUST be `min(MaxPrice, max(1, ref * bp / 10 000))`, truncated, with checked arithmetic, `ref` the effective reference price (6.1.3, 6.2). |
| FR-ORG-029 | `WorldCreated` MUST carry no field of this module beyond the names table (11.4.1); the module MUST move no money or goods and create no agent or account (11.4.2). |
| FR-ORG-030 | In the admission of every order of a market, after `overflow` and before `insufficient_funds`, a bid above the cap MUST have its limit replaced by the cap and an ask below the floor its limit replaced by the floor, whoever places it; the reservation, `OrderPlaced`, `Clearing` and the outcome MUST carry the replaced limit (6.3.1, 6.3.3). |
| FR-ORG-031 | With the clamp every trade price MUST be at or below the cap and at or above the floor, and the world's check of a `Clearing`'s trades MUST compare with the replaced limits (6.3.2). |
| FR-ORG-032 | The clamp MUST NOT reduce a quantity, reject an order, reorder the book or add a draw (6.3.4). |
| FR-ORG-033 | A world restored between ticks MUST give the same memberships, the same resolved rights and ranges and the same next round as an uninterrupted one (7.4.1). |
| FR-ORG-034 | In the admission of the credit round a request's `MaxRateBP` above the cap in force for its region MUST be replaced by the cap; a claim in force MUST NOT change (6.6). |
| FR-ORG-035 | In the admission of the contract round a take's bid limit MUST be capped and an offer's ask limit floored by the range in force for the commodity in the destination region, and the `sell` and `buy` rights of the destination MUST bind the offer and the take (6.7, 5.8.5). |
| FR-ORG-036 | `SetLegalLevels` MUST be between ticks, atomic, with the errors of 7.1.1 in that order, MUST rebuild the ranges at the call, and an entry equal to the configured level MUST restore it (7.1, 7.1.2). |
| FR-ORG-037 | `SetRightSuspensions` MUST be between ticks, atomic, with the errors of 7.1.1, and MUST rebuild the resolved rights at the call (7.1). |
| FR-ORG-038 | The levels and suspensions set by a call MUST be saved sparsely by ID in a checkpoint and MUST NOT be hashed (12.2, 12.4). |
| FR-ORG-039 | `MembershipChanged` MUST carry the fields of 11.1, MUST be emitted per (region, body) with any of its counts non-zero and never otherwise, and MUST be constructed only when an observer was supplied (11.1). |
| FR-ORG-040 | `RightRefused` MUST count every refusal of a right per (region, right) and activity and MUST be emitted only with a non-zero count (11.1). |
| FR-ORG-041 | `PriceControlled` MUST count every order or request that the clamp changed, per (region, control), and MUST be emitted only with a non-zero count (11.1). |
| FR-ORG-042 | `LegalLevelSet` and `RightSuspensionSet` MUST be emitted at the call, once per changed key in ID order, carrying the tick of the next step (11.1). |
| FR-ORG-043 | The three aggregate events MUST be emitted at P7.2b after `SettlementClassChanged` and before phase 8, in the order of 7.3.3 (7.3.3). |
| FR-ORG-044 | The names table MUST carry `Bodies`, `Rights` and `Controls` indexed by their typed IDs; the snapshot MUST carry `Bodies`, `LegalLevels` and `Suspended` as 11.5 says (11.4, 11.5). |
| FR-ORG-045 | The `Warn` "body has waiting households" MUST be emitted per (region, body), never per agent, by pattern B (the tenth consecutive tick and every tenth after), and MUST have the counterpart `MembershipChanged.Waiting` (11.2). |
| FR-ORG-046 | The construction summary MUST carry `bodies`, `rights` and `controls` and the module name `institutions` in `modules` when the module is on (11.2). |
| FR-ORG-047 | Every validation rule of 10.1 MUST have a sentinel and a failing-case test whose configuration breaks only that rule, matched with `errors.Is` (10.1). |
| FR-ORG-048 | The warnings of 10.2 MUST be raised once at construction and MUST NOT fail validation (10.2). |
| FR-ORG-049 | The membership MUST be saved once, in the agent record, as the pair `BodyID + 1` and `RegionID + 1` of its admission, 0 for none, and a world with no body MUST write 0 for both of every agent (12.1). |
| FR-ORG-050 | The slot section MUST be present with the module, with zero rows when nothing differs from the configuration; a restore MUST reject a membership that names an undefined body or a region outside its scope (or only one of the two), an undefined ID, an out-of-bound level or IDs out of order as corrupt, and MUST accept a membership of a household that is no longer eligible (12.2, 12.3). |
| FR-ORG-051 | The configuration hash MUST cover `Modules.Institutions` in canonical order, every field except the labels (12.4). |
| FR-ORG-052 | The module MUST add no ledger reason, faucet, sink or flow (16, N-107). |
| FR-ORG-053 | No iteration over a map and no sort without a total order MUST influence a result of the module (3.2.3, 4.3.1). |
| FR-ORG-054 | The module MUST read no price, stock, income, share or rate to decide a membership or a right; the one count it reads is the households of the body's roles in a region against `Places`, a limit (3.4.1, SET R2); a legal range MUST be a function of the configuration and the slot (position 12, 6.1.4). |
| FR-ORG-055 | The instruments of 11.6 MUST be derivable from the events and snapshot fields they name and each attribute MUST take values bounded by the configuration (11.6). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-ORG-001 | With the default logger and observer, the right test, the legal range read, the clamp, the headroom and the round of a tick MUST allocate nothing beyond scratch held by the world, and no event MUST be constructed (rules 8.2). |
| NFR-ORG-002 | The lapse pass MUST cost `O(households of the body's roles)` and the admission `O(E log E)` for the `E` eligible non-members of a (body, region) per tick, the right test and the clamp `O(1)` per order or request, and a slot call `O(entries)` to check and one rebuild of the derived arrays (19.1). |
| NFR-ORG-003 | Observation MUST NOT change a result: the same seed and configuration give identical results with and without a logger and observer. |
| NFR-ORG-004 | Every output MUST be bit-identical across runs and platforms. |

## 18. Open questions

1. **A body's account.** Dues, a fee for admission, a charitable fund, a hall of the body, an armed force
   all need an account and a rule that spends it. A body with an account is a pseudo-agent and
   shifts every household ID (N-117); the alternative is that dues are a MON levy on the
   members' class paid to a treasury of POL. FIS decides; the shape is a body's `Treasury`
   field naming a governed jurisdiction.
2. **More than one body per household** and overlap of scopes (3.3.2). For example a craft association and a
   league of traders over one role is a second membership field and a rule for which right
   binds. No consumer needs it.
3. **A right over carriage** (5.8.1): an activity `ship`, its subject (a mode, a commodity,
   an edge), and the refusal in TRN's list of 23. Reopen with the first consumer that asks.
4. **A right granted at run time.** A call that admits, expels or moves a member, or
   transfers a right, is a policy decision about named households (N-131). The slots of 7.1
   are the one lever until a consumer asks for more.
5. **A wage control** (6.8.2): the shape of a clamp in EMP's round, when EMP's revisions are
   in.
6. **Places per body or per region, and by class.** `Places` is per region of the scope
   (3.4.1) and a number of the body. A league with a limit over all its regions is a second
   field and a rule for who loses a place when a region is full; places that grow with the
   settlement (a town of the class that carries a flag supports more members of the craft) would name a
   **limit of SET** (`spec-settlements.md` 3.3.2) from the body, and the least of the two
   would apply.
7. **The admission order.** Seniority first (4.3.3) is one rule; a keyed lottery or a
   sponsorship by a member (a relative of a member) are others. The first is chosen because it
   needs no draw and no state.
8. **The shadow price** (6.5.4). Whether a household's belief should be bounded by the legal
   range in force, so that lifting a cap does not jump: it would change `spec-beliefs.md` 5 and
   was not adopted because the unmet demand is the signal.
9. **Imports under a legal price.** A counterparty is bound by a legal price and exempt from a
   right (6.3.3, 5.5.2). A sole right to import, or a cap that does not bind the outside, is FIS's
   tariff or a flag on the line (EXT).
10. **The bounds.** `MaxBodies` 256, `MaxRights` 4 096, `MaxControls` 4 096 are guesses; the
    first consumer's world needs a handful of each.

## 19. Cost, and where this lives in the code

### 19.1 Cost

Counts per tick, not timings (N-015; the library measures no wall-clock time), in the terms of
`spec-world.md` 11 (`R` regions, `C` commodities, `H` households):

- **The round** (P1.9a): the lapse pass is `O(households of the roles of the bodies)`, an
  array read and two comparisons each; admission is `O(E log E)` per (body, region) for the `E`
  eligible non-members (a sort by a total order), and no sort is needed with `Places` 0 and no
  need (every candidate is admitted). With no eligible non-member the round is the lapse pass.
- **A right test**, at a start or an order: one read of the dense array (activity, subject,
  region) and, with a right in force, one membership test: `O(1)`, no allocation.
- **The clamp**: one read of the range of (region, commodity) and a comparison, per order,
  request or contract limit: `O(1)`.
- **The headroom**: one subtraction per (region, role) per entry step, from the counts the
  spawner keeps.
- **A slot call**: the checks are `O(entries)`, and the derived arrays are rebuilt once per call
  (7.2.1): `O(R * (recipes + 2 * C + roles))` for the resolved rights and `O(R * C)` for the
  ranges.
- **Memory and events**: the two derived arrays, the membership pair of each agent (12.1) and the
  scratch counts per (region, body), (region, right) and (region, control), all bounded by the
  configuration and held by the world; at most `R * (bodies + rights + controls)` aggregate
  events a tick, built only with an observer (rules 4.5).

No figure here is a measurement.

### 19.2 Layout

Nothing is written; this is the proposed layout for the first code, checked against
architecture.md 2.6 and `AGENTS.md`.

- `institutions.go` (root): `BodyID`, `NoBody`, `ControlID`, `RightID`, `Body`, `Right`,
  `Control`, `RateLimit`, `LegalLevel`, `RightSuspension`, the activity and `Who` constants,
  the sentinels, the construction of the resolved tables, the round at P1.9a, the checks at P2.5
  and P3.2, and the three events. One file per concept (architecture.md 2.6).
- `internal/institution/` (standard library only, primitive slices, no root type,
  architecture.md 2.9): the resolution of a right or a control over a region and a chain (the
  innermost mode of `spec-polity.md` 6.1.2), the legal range from a reference price and basis
  points with checked arithmetic, the membership round (`Admit` over parallel slices of weight,
  `RoleSince`, key and flag), the headroom, and the clamp. It takes the stateless key from
  `internal/ordering`. The root converts at the call, inline.
- `config/`: the tables `[modules.institutions]`, `[modules.institutions.body.<id>]`, `.right.<id>`,
  `.control.<id>` and `.rate_limit.<id>`, and the rules of 10. No other package reads a file
  format.
- `testdata/chains/overlay/institutions.toml` and `institutions-inert.toml` (REF, 13.5).
- Tests: table-driven unit tests in `internal/institution` with the cases of `admit.py` and
  `clamp.py`; external `_test` tests of the neutrality of FR-ORG-001 over seeds, of a world with
  a body with a right and a cap, of the held completion with SKL on, of the credit clamp with CRD on; a
  failing-case test for each sentinel of 10.1; a checkpoint round trip with a member, a
  suspended right and a changed level; a fuzz test of the tables of the loader; an
  `AllocsPerRun` test for NFR-ORG-001; and an `Example` test of the clamp with example B.
- Rule files, at the first code (owed rows, with a fresh tag each): architecture.md 2.6
  (`institutions.go`, `internal/institution/`) and 3.3 (`BodyID`, `RightID`, `ControlID` among
  the typed IDs); observability-and-logging.md 3.3.2 (the keys of 11.3), 4.3 (the five events)
  and 5.3, 5.4.1 (the four instruments and the attributes). The glossary words go to `doc.go`
  (N-017).
