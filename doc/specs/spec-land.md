# Spec: Tenure: titles to the stock of an endowment, the entitlement before the ration, rent as a transfer valued at the dual, and tenancy by let units

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-LND-NNN` and
`NFR-LND-NNN`, which tests cite (testing.md section 4). They are in section 19.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
later stage (`vision.md` section 4, "Later"). It is off unless configured:
`Modules.Tenure` is a pointer, nil is off, and the zero configuration has no
title, no rent and the open access of `spec-resources.md` 10.1, exactly. The core
does not depend on it. Needs `spec-resources.md` (RES: the endowment, the
snapshot, the ration and the three hooks of 10.2), `spec-production.md` (PRO: the
start of a job, the refusals of 4.2, the cost anchor of 9), `spec-agents.md` (AGT:
the removal of 3.6, the view of 6.4), `spec-regions.md` (REG: the region-keyed
table of 5.1, the estate pseudo-agent), `spec-estates.md` (EST: the pool),
`spec-ledger.md` (LED: the reason registry), `spec-money-supply.md` (MON: why rent
is not a flow of that module, 5.1), `spec-world.md` (WLD: the tick), and
`spec-config.md` (CFG: the module switch and validation). It reads
`spec-subsistence.md` (SUB 9.5, the commons), `spec-migration.md` (MIG, 7.2),
`spec-employment.md` and `spec-storage.md` (EMP and STO, written beside it) and
`spec-viability.md` (VIA: the dual `Rent`, 11.3 of RES), and asks each for an edit
(section 18).

**Every number in this spec is a placeholder** except where a derivation is shown.
Nothing has been run in the library, bazaarBot has not been run, and no acceptance
band exists (notes N-015, N-027). The scripts of section 17 are stylised Docker
Python models of the rules of this spec, not runs of code and not a claim that any
default is stable; the rules themselves are integer. No default is set from a
source: no historical fact about tenure, tenancy or rent was read or used
(N-018), and the defaults are structural (no title, no rent, a rent that does not
move) or placeholders. The settings of the worked examples (section 15), a farming
realm and a mining colony, are **examples**, flagged as such. The core text and its
vocabulary are setting-agnostic: the words for what a stock stands for (land,
pasture, ore, ice) appear only in the examples and in the name of the file. A
commodity, a recipe, a role or a resource is data (architecture.md 9.1): nothing
in this spec branches on one.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).
The word **claim** is not used here: it means a ledger claim to `spec-ledger.md`
10 and `spec-credit.md`. The word **lease** is not used: `spec-storage.md` keeps it
for a later contract over room.

## 1. Purpose and scope

`spec-resources.md` ends with a stock that anyone in the region may draw on, shared
by a proportional ration, and with three hooks that it leaves to tenure: a refusal
after `no_endowment`, a per-claimant entitlement applied before the ration, and a
rent valued at the dual of the resource row (RES 10.2, N-108). The open-access
draw-down of that spec (RES-A5, STB-L90) has no damper in the core: "tenure is
LND's". This spec fills the three hooks. It covers:

- what a **title** is, who may hold one, and the invariants of a holding
  (sections 3 and 6);
- the **entitlement** a title gives before the ration, and the four tiers of the
  ration with tenure: own, commons, let (section 4);
- **rent**: the posted rate, who pays and who is paid, what a tenant who cannot
  pay gets, the bounded step that moves the rate with the units unserved, the
  value at the dual of the resource row, and how rent enters the cost anchor and
  the welfare ratio (section 5);
- how a title is acquired, transferred and lost: configuration, a public call,
  inheritance by the estate (section 6);
- what tenure does to the specs around it: the commons of the subsistence floor,
  migration, hired labour, storage, the pool (section 7);
- the position in the tick, the draws (none), the slots and calls, the
  parameters, validation, viability, observability, the state saved, two worked
  examples, the register entries, the models run, the requirements, the open
  questions and the home in the code (sections 8 to 21).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The stock, its regeneration, the snapshot and the ration among requests without tenure | `spec-resources.md` (RES) |
| A purchase decision (an agent that buys a title because the rent is worth more than the price), credit to buy one, a market for titles | `spec-saving-and-investment.md` (SAV), `spec-credit.md` (CRD); this spec gives the transfer they will call (6.3) |
| Taxes on rent, a land tax, a tenure levy of a ruler | `spec-fiscal.md` (FIS): a tax on rent is not offered; a tax on the holder role is `spec-fiscal.md` 5; a money flow of `spec-money-supply.md` on a holder role is available now |
| A charter, an exclusive right, a rent control as law, a guild's claim | `spec-institutions.md` (ORG); the slot of 9.1 is the mechanism they will set |
| Fixed capital that raises an endowment's capacity (irrigation, drainage) and who owns it | `spec-infrastructure.md` (CAP), which adds a layer multiplied into the effective capacity (its 5.3), not a write to the capacity slot of RES 6.3 |
| Loss of a title by conflict, a seizure, a siege | not RSK's in version 1: RSK names no cause and has no actor that moves a title; a title is not goods; the consumer's call of 6.3 |
| A jurisdiction in which a holding is legal | `spec-polity.md` (POL) supplies the key (a jurisdiction); a table keyed by jurisdiction, with the combining modes of `spec-polity.md` 6.1.2, is this spec's or ORG's |
| Renting room to another agent, a keeper's business | `spec-storage.md` open question 2; a contract, not tenure of a stock |
| Holding a unit of the stock for the length of a job (a concurrency limit, not a flow) | open question 1 of RES; not modelled here either (20.1) |
| The public calls and queries as such | `spec-integration.md` (INT, section 9) |

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is tenure? | A **title** is a share, in basis points, of the snapshot stock of one endowment, held by one **holder**. The shares of an endowment never sum above 10 000; the untitled remainder is the **commons share**. A title changes who may draw and who is paid when another draws. It does not split the stock | RES keeps one stock per (region, resource) and the ration reads it once per tick (RES 5.2). A share of that scalar is the minimum that gives the three hooks a subject. A sub-stock per title would reimplement regeneration per title (section 20, question 2) |
| 2 | Is it a module? | Yes: `Modules.Tenure`, present means on, needs `Modules.Resources` (`ErrModuleDependency`). With it off, or with no row for an endowment, the ration is RES 5.3 and every result, event (but `WorldCreated`) and draw is that of a world without this spec (FR-LND-006) | ADR 0001 decision 2: the zero configuration is the tested baseline; the open-access result is the baseline tenure is measured against (RES 10.1) |
| 3 | Who may hold a title? | An agent (a household, a cohort or an external agent, as one holder whatever its weight) or **the pool**, the estate pseudo-agent of the endowment's region. Nothing else: no company, no association, no jurisdiction (6.3, 7.7) | The estate is the one holder the world already has that is not a household (`spec-estates.md` 2); a group holder is ORG's and POL's |
| 4 | What does a title give? | The **entitlement**: before the ration, a share of the tick's snapshot stock that only its principal may draw on first, by an exact apportionment (4.2). What the holder does not use is **let** to the requests the commons did not serve, at the posted rent, unless the endowment is `Closed` | RES 10.2 hook 2 ("a holder's share is its own and the remainder is shared as today"). Letting what is idle is what makes tenancy exist without a contract; closing it is the data switch for an exclusive right |
| 5 | In what order is a request served? | Own entitlement, then the commons (free), then the let units (paid). A request that is served in a tier pays only for what it draws in the third | A free unit is taken before a paid one, so rent arises only at the margin where the free stock is spent: the Ricardian reading of rent, and a damper on the price of access |
| 6 | What is rent? | A **transfer** of money, per unit drawn from let units, from the principal that runs the job to the lessors, with one new internal ledger reason `rent`. It is not a money flow of MON: a flow is a boundary flow on a class of households with no counterparty, and rent has a payer, a payee and a base that is a draw. It creates and destroys nothing | N-107 says to add a built-in reason "only for a mechanism that is not a grant or a levy"; this is that case. N-108 and RES 10.2 name the form "a named money flow"; 5.1.2 gives the reason it is not (MON 4.3, 10.3) |
| 7 | How is the rate chosen? | **Data**: a rate per unit drawn per (region, resource), with the solver's dual `Rent` (RES 11.3) as the guide and the warning `rent_off_dual`. An optional bounded **step** moves the posted rent with the units unserved, inside a band that is a fraction of the configured rate. Default: the rate does not move | "Valued at the dual of the resource row" (RES 10.2). A rate that follows a measured aggregate is refused by WLD 9.4.1; the units unserved at one endowment are that endowment's own market, like BEL's unmatched units |
| 8 | Does rent enter prices and choices? | Yes, in two places: the cost anchor of PRO 9.1 (the belief update) and the default strategy's `cost_r` (AGT 7.1.4, the recipe choice, through `Posted` of the view): the rent in force of a drawn unit is a cost of the recipe, for a holder's own units too (an owner could let them) | Without it rent changes who is paid and nothing else, and RES-A5 has no damper. PRO 9.1 said "endowment draws are not priced" and AGT 7.1.4 reads no draw; the edits owed there are one line each (section 18) |
| 9 | Who pays when a job is hired or a trainee works? | The **principal**: the account whose inputs the job uses. Holder status is the principal's, so a worker on the employer's draw is on the employer's title | EMP 6 and SKL 4: a hired or trained job runs on the principal's account (PRO 4.3) |
| 10 | What if the principal cannot pay? | The executions it can pay for start, and the others do not; with none the refusal is `rent_funds`. The rent is never a debt | Credit is an optional module (`spec-credit.md`) and a rent is never a debt; a start that cannot be paid is refused, as a bid is (`insufficient_funds`) |
| 11 | How is a title acquired, moved and lost? | At construction by configuration (a share per household of a role, or the pool); by the public call `TransferTitles` between ticks (a consumer, SAV, CRD, ORG); on the removal of a holder it passes to the pool of the endowment's region, and nothing hands it back in a run. There is **no in-run purchase decision** in version 1 | EST 2: "no heirs, no wills". A decision to buy is an investment decision (SAV) and needs a plan field of `Strategy`, which this spec does not add |
| 12 | Does a title bind its holder to the region? | **No `region_bound` refusal.** Nothing about a title needs the holder's account in the region: rent is a transfer between accounts, and the entitlement is read from the executor's own region where the job runs | N-090: the rule is for an obligation that needs the account in the region. A tenant has no commitment: it draws day by day (7.2) |
| 13 | How does a SUB household fare? | An own-use request is a request. It pays rent like any other unless the module field `OwnUseExempt` is set; where the endowment is closed to it, the household is `Blocked` with the reason `no_title` or `rent_funds` | SUB 9.5 states the hook; the exempt switch is the "right of the landless to draw for their own use" it names. Default: the rule applies to all alike |
| 14 | A new refusal reason? | `no_title` and `rent_funds`, after `no_endowment` and before CAP's build reasons and STO's `no_room` in the master order of PRO 4.2 | RES 10.2 hook 1; the order keeps `no_room` last (STO 5.3) |
| 15 | Does it draw from the random source? | **No.** The leftovers of an apportionment go by the stateless key of RES 5.3.3 and the ID. The stream of a world does not depend on whether the module is on | WLD 7.4.2; the rule for a new tie is WLD 10.1.1 |
| 16 | What is saved? | Per titled endowment: the holders with their shares, the posted rent when it can move, two diagnostic runs; the non-default rent scales. All plain data, sorted, no maps | ADR 0002; N-128: a title is saved once, here, and not in the agent record (6.5) |
| 17 | Does it add feedback loops? | Yes: the rent step against a delayed response, the priced-out tenant, rent and the target-income rule, erosion of private titles into the pool, concentration through transfers, enclosure and flight, rent drained into a pool (section 16) | `spec-stability.md` 9: a spec that adds a loop and no entry is not done |

## 3. Titles and holders

### 3.1 What a title is

3.1.1. A **title** is the triple (endowment, holder, `ShareBP`): the right of the holder to
the share `ShareBP` of the snapshot stock of the endowment of one resource in one region.
`ShareBP` is a whole number from 1 to 10 000. A holder has at most one title per
endowment: a second one is merged into it. The titles of an endowment are its
**holdings**, sorted by the holder's agent ID.

3.1.2. **The commons share** of an endowment is `10 000` minus the sum of its titles, and
is 10 000 when it has none. The sum never exceeds 10 000 at any observable point
(FR-LND-002); it is checked at construction, at every transfer and by the audit.

3.1.3. A title is **of a share of the stock, not of the capacity**. The stock moves with
draws and regeneration, so the units a title covers move with it (4.2). A shock slot of
RES 6.3 that scales the capacity, or an adjustment of the stock, changes the units and
never a share. This is a choice, with a cost: a pack author who thinks in units of capacity reads a
share as "the fraction of what is there", and a thin stock is a thin entitlement
(section 20, question 11 gives the alternative).

3.1.4. A title carries no term, no condition and no price. It is not a commodity, not
a ledger lot and not an account: it is a row of this module's state, as an endowment is a
row of RES's (RES 3.1.2). A title is not in any recipe's `Inputs`, `Outputs` or
`Requires`.

### 3.2 Holders

3.2.1. **An agent.** Any agent that is not a pseudo-agent: a household agent of the
configuration, one created later, or an external agent (`spec-integration.md` 4). A cohort
of weight `w` is one holder with one share; its weight does not scale the share and a
change of weight does not change it (6.5).

3.2.2. **The pool.** The estate pseudo-agent of the **endowment's** region
(`spec-regions.md` 3.6: its agent ID equals its `RegionID`; `spec-estates.md` 2.7.4). The
pool is a holder like any other with two properties: it has no strategy and no job, so all
its entitlement is unused and is let (or lies idle when `Closed`); and the rent it earns
goes to the estate account, which is the region's pool of money (M_p of
`spec-money-supply.md` 3.1). The estate of another region is not a holder of this
endowment (`ErrNotAHolder`, 9.2). A counterparty of the outside is never one.

3.2.3. **Who is not a holder.** An outside node has no household and no pool of its own
that draws (`spec-regions.md` 3.5); a row of this module that names an outside node's
region is `ErrOutsideNode`. A group, an association, a jurisdiction and a company are not
holders in version 1 (ORG, POL; position 3).

### 3.3 The module and its rows

The module is `Config.Modules.Tenure`, a pointer: nil is off, non-nil is on
(`spec-config.md` 3.2). Plain data, sorted, no maps:

```go
// TenureConfig is the tenure module. Its presence switches it on.
type TenureConfig struct {
	Terms            []TenureTerm // region-keyed table: one row per (region, resource), sorted
	Holdings         []Holding    // initial titles, sorted by (region, resource, holder)
	RentStepBP       int64        // step of the posted rent, basis points of the configured rate; 0 = the rent does not move
	RentFloorBP      int64        // lowest posted rent, basis points of the configured rate
	RentCeilingBP    int64        // highest posted rent, basis points of the configured rate
	OwnUseExempt     bool         // an own-use request (spec-subsistence.md) pays no rent
	MaxHolderShareBP int64        // the largest share one agent may hold; the pool is exempt
}

// TenureTerm is the row of one endowment: what a unit let from a holder costs and
// whether the unused entitlement is let at all.
type TenureTerm struct {
	Region, Resource string
	HasRent          bool  // Rent applies
	Rent             int64 // money, in minor units, per unit drawn from a let unit
	HasClosed        bool  // Closed applies
	Closed           bool  // true: the unused entitlement of a holder is not let to others
}

// Holding is an initial title, or the share of a class of households of a role.
type Holding struct {
	Region, Resource string
	Holder           HolderRef
	ShareBP          int64 // per household when Holder is a role; the total when it is the pool
}

// HolderRef names a holder in configuration: a role (every household of the role in
// the region at construction) or the pool of the region.
type HolderRef struct {
	Kind string // "role" or "pool"
	Role string // when Kind is "role"
}
```

3.3.1. **The rows are the override mechanism of `spec-regions.md` 5 (note N-104).**
`Terms` is a region-keyed table: one row per (region, resource), owned by this spec for
meaning, default and bound. `Rent` and `Closed` use the **replace** mode: the row's value,
else the default (no rent, not closed). Absence is the default, with no warning: a region
with no row for an endowment has open access there, which is the baseline, and a row
that restates the defaults changes nothing. `Holdings` is keyed by (region, resource,
holder); one row per key. Both tables are resolved once at construction into dense arrays indexed by (`RegionID`,
`ResourceID`), as RES 3.2.2 does; the canonical order is ascending `RegionID`, `ResourceID`, then (holdings) pool before
role, roles by `RoleID`; the errors are `ErrUnknownRegion`, `ErrUnknownResource`,
`ErrUnknownRole` and `ErrDuplicateID` (`spec-regions.md` 5.3.1, 5.3.2). Two rows join the
list of `spec-regions.md` 5.1.3 (section 18). A slot (9.1) edits no row
(`spec-regions.md` 5.3.6).

3.3.2. **File layout** (`spec-config.md` 6.2.1, the one rule for module tables, N-102 closed): the module is `[modules.tenure]`, a term is
`[modules.tenure.term.<region>.<resource>]` and a holding
`[modules.tenure.holding.<region>.<resource>.<holder>]` with `<holder>` a role ID or the
word `pool`, in the way of `[modules.subsistence.role.<id>]` and
`[modules.resources.endowment.<region>.<resource>]`, so that layers merge by key
(`spec-config.md` 6.4). A role whose ID is `pool` would collide with the word and is
`ErrDuplicateID` (11.1). Section 15 shows the keys.

3.3.3. **Defaults** are the zero values: no rent, not closed, `RentStepBP` 0, no
holdings, `OwnUseExempt` false and `MaxHolderShareBP` 10 000 (the cap is off). A module
that is present with nothing in it is legal and does nothing (the warning `no_titles`,
11.2). There is therefore no default macro constant in this spec (N-027).

## 4. The entitlement and the ration

### 4.1 Where it runs

Tenure acts inside the ration of RES (`spec-world.md` P2.6) and nowhere else in the
production phase. The snapshot of P2.3 gains one derived step, the entitlements, and the
ration of P2.6 gains the tiers below; the starts of P2.7 gain the payment of rent
(5.3). An endowment with no row in `Terms` and no title is rationed by RES 5.3 exactly: the
module adds nothing to it, takes nothing and reports nothing (FR-LND-006). An
endowment with a `Terms` row and no title has a commons share of 10 000: the tiers
below collapse to RES 5.3, there is no unit to let and no rent is ever charged (the
warning `rent_without_titles`).

The part of the tick that concerns tenure runs in this order, for every endowment that
has a title, in (`RegionID`, `ResourceID`) order:

1. **Entitlements** (P2.3, derived from the snapshot stock `A` and the holdings): section
   4.2.
2. **`Work`** is unchanged. A request's `k0` is computed ignoring draws, as RES 5.2 says;
   its results do not depend on tenure.
3. **The ration** (P2.6): the tiers of 4.3, in order, with the apportioned units of step 1.
4. **The start** (P2.7): `k` is the least of `k0`, the grants and the live inputs, as in RES
   5.4, and then the largest number of executions whose rent the principal can pay (5.4); the
   lessor supplies and the transfers follow on the units taken (5.3.1).

### 4.2 The entitlement

4.2.1. For an endowment with snapshot stock `A` (the stock at the start of the production
phase, RES 5.2 step 1, fixed for the tick) and holdings `s_h` (basis points, `h` a holder),
with `s_0` the commons share, the **entitlement** of each holder and of the commons is the
apportionment of `A` units by the weights `s` by the **largest remainder**:

```
base_p = floor( A * s_p / 10 000 )                    for every part p (a holder or the commons)
rem_p  = ( A * s_p ) mod 10 000
left   = A - sum of base_p                            0 <= left < number of parts with a positive share
```

then one more unit goes to each of the `left` parts with the largest `rem_p`, ties
broken by the **stateless key** `K(t, id)` of RES 5.3.3 (smaller first), then by agent ID,
the commons ranking after every agent. The result `E_p` sums to `A` exactly. The products
`A * s_p` go through the checked 128-bit helper (`A` is at most `MaxEndowment` and `s_p`
at most 10 000: the product exceeds `int64` in the extreme, architecture.md 3.5). A part
with a share of 0 is not a part.

4.2.2. **Why this rule.** The same walk as the ration of RES 5.3.2, for the same reasons:
it is exact, it takes no draw, it keeps no state, it is a total order, and the key changes
every tick, so no holder is systematically first when the share does not divide the stock.
With a share of 100 basis points of a stock of 80 each of fifty holders is owed 0.8 units;
the apportionment gives 1 unit to forty of them and none to ten, a different forty each
tick, and the units sum to the stock (the second case of 15.1, `tiers.out`).

4.2.3. **What it means for a small share.** An entitlement is a whole number of units, so
a holder whose share of the stock is below one unit has an entitlement of 0 or 1 that flickers
by the key. Its protection is statistical (in expectation exactly its share) and a
fixed-size request is not protected tick by tick. This is stated as a property and
registered (LND-A8, section 16): a pack that wants the protection to be exact keeps the
stock at or above 10 000 divided by the smallest share in basis points (the warning
`entitlement_below_unit`).

### 4.3 The four tiers

For an endowment with titles, per tick, in this order. Requests are those that draw on
it, with `w_i` executions wanted (`k0` of the request) and `d_i` units drawn per execution
(RES 5.3.1); the **principal** of a request is the account whose inputs it uses, which is
the executor itself except in a hire (EMP) or a traineeship (SKL), where it is the employer
or the trainer (position 9).

1. **Own.** For each holder `h` with at least one request whose principal is `h`: the
   units of its entitlement `E_h` are rationed among those requests by the rule of RES
   5.3.2 (all granted if they fit, else proportional with the leftovers by the largest
   fraction, then the key, then the agent ID). The grants are `o_i`. The **unused
   entitlement** of `h` is `U_h = E_h - sum o_i d_i`, and `E_h` in full when `h` has no
   request. A request whose principal holds no title has `o_i = 0`.
2. **Commons.** The residual demands `w_i - o_i`, over all requests, are rationed against
   the commons entitlement `E_0` by the rule of RES 5.3.2. The grants are `c_i`. The
   commons is **free**: no rent is charged for a unit of this tier.
3. **Let.** Unless the endowment is `Closed`, the supply is `L = sum of U_h` over all
   holders. The demand of a request for this tier is what remains, `w_i - o_i - c_i`, **clipped
   to the executions its principal can pay at the rent in force**,
   `ŵ_i = min(w_i - o_i - c_i, floor(balance / (r * d_i)))` when `r` is positive and the
   request is not exempt (5.3.4), the balance being the principal's at P2.6 (no reservation
   is open in P2, `spec-world.md` 6.1.6). The clip is per request: a principal with several
   requests, or draws on several endowments, can still be cut at the start (5.4). The units a
   request is clipped by are **priced out**: `(w_i - o_i - c_i - ŵ_i) d_i` (counted in
   `RentPaid.PricedOut`, and in no step of the posted rent, 5.5.1). The clipped demands are
   rationed against `L` by the same rule. The grants are `t_i`. Their units are **let units**,
   for which rent is charged (5.3). With `Closed` the supply is 0 and `t_i = 0`: the unused
   entitlement stays in the stock, and is counted in `RentPaid.Held` (13.1).
4. **The grant** of a request is `g_i = o_i + c_i + t_i` executions, at most `w_i`, and the
   units it draws are `g_i d_i`. The units taken in all tiers sum to at most `A`
   (FR-LND-011). Units of any tier that the starts do not take (for the reasons of RES
   5.4.2 and of 5.4 here) stay in the stock; the allocation does not iterate.

4.3.1. **Order of tiers, and why the commons is before the let.** A request is served
from the free supply before the paid one, so that rent is paid only for what the free
stock could not serve. The consequence is the one of position 5: while the commons covers
the demand no unit is let and rent is 0 whatever the posted rate says; the posted rate
bites at the margin. A pack that wants everyone to pay sets the commons share to 0 by
giving the pool the rest (position 3).

4.3.2. **Ordinary units.** Each ration is the rule of RES 5.3.2 on a different supply and
different residual demands, so every property RES proves for it holds per tier: the grant
never exceeds the request, the units never exceed the supply, the result does not depend
on how the requests were listed, and a cohort receives its proportional share (RES 5.3.5).
A cohort holder is one request of `w` executions; its own tier is its share of the stock
for all `w` households, which is what `ShareBP` per household times `w` meant
at construction (6.1).

4.3.3. **Several draws.** A recipe with two draws on endowments that have titles is in two
sets of tiers; its grants are independent and `k` is the least of them, exactly as RES
5.4.2 says. The units granted in a tier of the larger and not used stay in the stock.

### 4.4 Who is on which unit: lessors, tenants and the attribution

4.4.1. The units let in tier 3 are drawn from the unused entitlements. Let `V` be the let
units the starts **took** (P2.7, after the cut of 5.4: `V <= sum of t_i d_i <= L`). The
**supply** of each **lessor** `h` (a holder with `U_h > 0`) is the apportionment of `V` by
the weights `U_h`, by the largest remainder with the same key and ID ties as 4.2.1:
`S_h <= U_h` (for `V <= L = sum of U_h`, `ceil(V U_h / L) <= U_h`). This is exact and is
computed on what was drawn, not on what was granted: a holder whose unused entitlement is
half of the let supply supplies half of what was drawn, whatever tenant was cut, so rent is
shared in proportion to what each held idle and never in the order of the IDs (a lower ID
is not first, WLD 10.1.1).

4.4.2. **Attribution.** The tenants are the requests that took let units, in ascending agent
ID; the lessors in ascending agent ID (the pool first, its ID being a `RegionID`). Walk
both lists with two pointers, assigning the units each tenant took (the `k` of its start,
after the own and commons executions, 5.3.2, times `d_i`) to the lessors' remaining `S_h`
in order. The result is a list of triples (tenant, lessor, units). It is a
function of the units taken and the supplies, so it needs no state and no draw. A tenant
that is its own lessor (a holder whose request exceeds its entitlement by less than an
execution and which is assigned units of its own `U_h`) is assigned them like any other and
the transfer to itself is skipped (5.3.3); the units still count as let.

4.4.3. **What the attribution decides.** It fixes who pays whom and nothing about who
draws: all units are of one stock and one quality, the tenants were chosen by the ration
and the lessors' supplies by the apportionment of 4.4.1. Each lessor is paid for exactly
`S_h` units whichever tenant was cut, so the walk in ascending IDs gives no lower ID a
larger total of rent or of units; it only decides which pairs the ledger entries record.
Every tenant pays `r` times the units it took and every lessor receives `r * S_h`
whatever the pairing, so no balance and no sector total depends on it.

### 4.5 What the grants cost: the reasons

4.5.1. The grant `g_i` limits the start as RES 5.4 says. A request whose `k0` was positive
and whose start is 0 is refused with the first of the reasons of the master order of
`spec-production.md` 4.2 (which owns it): `zero_yield`, `no_site`, `no_tool`,
`no_facility`, `no_input`, `no_endowment`, **`no_title`**, **`rent_funds`**, then CAP's
`not_unlocked`, `facility_cap`, `no_target`, `store_refused` and `nothing_to_repair`
(`spec-infrastructure.md` 4.2.2), then `no_room` (with the storage module on); a request off
the role's list is `not_on_role` and one off a right's `produce` is `not_permitted`, both
before `k0` is computed. `no_endowment` and `no_title` are told apart by rule, not by
competition:

- `no_title`: the request's draw is refused because **by rule no unit of the stock is open
  to its principal**. With `A_i = E_p + E_0 + (Closed ? 0 : sum of E_h for h not p)` the
  units open to principal `p` before any competition, the reason holds when
  `floor(A / d) > 0` and `floor(A_i / d) = 0`: the stock could serve an execution but a
  closed tenure reserves every unit for others. A pack author fixes it by a title, a
  transfer or by opening the endowment.
- `no_endowment` is every other case of a zero grant: the stock is too thin (`A < d`) or
  the competition took what was open to it. This is RES's reason and is unchanged.
- `rent_funds`: the start is 0 and the principal's balance is what made it 0: the request
  was priced out of the let tier (`ŵ_i` below its residual demand, 4.3 step 3) or its let
  executions were cut for rent at the start (5.4), and nothing was taken in the other tiers.

4.5.2. The view (RES 5.5.1) gains the `Endowments` fields of 4.6 so a strategy can avoid
a request that the world would refuse. `MaxExecutions` is bounded by `floor(A_i / d)` and
carries `no_title` as its reason when that is 0 and the stock is not.

### 4.6 What a strategy sees

4.6.1. The `Endowments` group of the view of AGT 6.4 and RES 5.5.1 (the agent's own region
only, free like the local price) gains, for each endowment the agent's role draws on and
that has tenure data: `Posted` (the rent in force per unit, 5.2, 0 when none), `ShareBP`
(the principal's own share, 0 when it holds none), `Entitled` (the units of its
entitlement at the snapshot, `E_p`), and `Closed`. **No view carries another holder's
share or the list of holdings**: who holds what is not what a household knows, and a
strategy that wants it asks nothing of the library (INT's query `Holders` is for the
consumer, 9.3). This is the endowment rule of RES 5.5.1 (N-106) applied to a second kind of
local state.

4.6.2. The default strategy reads one of the new fields, `Posted`, and no other. AGT 7.1.4
chooses a recipe by `net_r = value_r - cost_r` with `cost_r` the inputs and the tool wear
at the agent's own beliefs, and does not read the cost anchor (`UnitCost` is the anchor of
the belief update, AGT 6.4). With this module on, `cost_r` gains `rent_cost_r = sum over
Draws of d * Posted` (the rent in force of the view, which is what a unit of the draw
would cost at the margin, for a holder's own unit too: 5.7.2); an edit to AGT 7.1.4 is owed
(18.2) and the choice is tested (FR-LND-020). A **source** recipe stays eligible at
`net_r <= 0` (AGT 7.1.4): rent does not stop a household that must work to eat; it moves
the choice among recipes and the eligibility of the recipes that have inputs. The strategy
asks for the executions its inputs and the view allow; a start the world cuts for rent is
counted and not hidden (13.1).

### 4.7 Cohorts, principals and hires

4.7.1. **A cohort** holder of weight `w` is one request of up to `w` executions; its own
tier is the units of its entitlement, `E_h`, and the executions it covers are
`floor(E_h / d)` for one request (or the ration of 4.3 step 1 among several). The
entitlement is a share of the stock for the agent, not per household.

4.7.2. **A hired job** (EMP) runs on the employer's account: its request's principal is
the employer, the employer's title is the entitlement of the job, and the employer pays any
rent. **A traineeship** (SKL 4) is the same with the trainer. EMP forms a hire at P2.5a,
before the ration of P2.6, so at formation the units let and the rent are not known; the
checks of FR-EMP-023 gain tenure as **bounds**, not as the outcome:

- `no_title` after `no_endowment` and before CAP's build reasons and `no_room` (the master order of `spec-production.md` 4.2), by the rule of 4.5.1 (it needs only
  the stock and the holdings, which are fixed for the tick);
- `rent_funds` only when the principal cannot pay even the **least** rent the job could
  owe: the rent of the let units that remain after its own entitlement and the whole
  commons entitlement (`max(0, m * d - E_p - E_0)` units, at the rent in force), which is 0
  when its own entitlement covers the draw. A principal that can pay that bound forms the
  hire; whether it can pay the rent the ration really leaves is decided at P2.7 (5.4).

What can still cut a hire after formation is, as for the ration and the room, a cut at the
start: EMP 5.5.1 and 6.1.3 gain "a rent cut" beside "a ration or room cut" (section 18);
the wages of the unstarted executions are refunded under `wage_refund`, and with `k` = 0 the
hire ends `not_started` with the escrow refunded. A vacancy refused at formation for tenure
counts in `LabourRound` under the fields EMP names (`NoEndowment` and its two new
neighbours, owed there).

4.7.3. **A holder who is also an executor and a principal** of several requests is
handled by 4.3 step 1: its entitlement is rationed among them like any supply.

### 4.8 An example

Ten holders of 500 basis points each, the commons share 5 000, a stock of 100, a draw of
1 unit, rent 40. Each holder is entitled to 5 units, the commons to 50. Each holder wants 3
executions: it takes 3 of its 5 and has 2 unused. Forty landless households want 2
executions each: 80 units against a commons of 50; the proportional part gives 1 each
(40 units) and the leftover 10 go to ten of them by the key: ten get 2 executions, thirty
get 1. Thirty requests remain of 1 execution each, against 20 let units (2 from each
holder): the proportional part is 0 and the leftover 20 go to twenty of them by the key:
twenty are tenants of 1 let unit, ten are unserved. Each lessor supplied 2 units and is
paid 80; 800 of rent moved; 100 units were drawn, the whole stock. Under RES 5.3 alone the 110 units asked
would have been shared with every holder getting 2 of 3 and every landless household 2 of
2: tenure moves ten units from the landless to the holders, which is the entitlement and
nothing else. Section 15.1 has the table and `tiers.out` the run.

## 5. Rent

### 5.1 What it is, and what it is not

5.1.1. **Rent** is the money a principal pays for the let units it draws. It is a **ledger
transfer** with the reason `rent`, kind internal, from an Agent account (the principal's)
to an Agent account (a lessor's) or an Estate account (the pool's), owner this spec. It
creates and destroys no money and no good: it moves money between non-boundary accounts,
so the supply identity of `spec-money-supply.md` 3.1 holds without change, and it is in no
`BoundaryFlow` row (the internal reasons are told by their own events, `spec-world.md`
P8.2). A rent to the pool moves money from households to the pool sector (M_h to M_p).

5.1.2. **Why not a MON flow** (N-107 and N-108 name the form "a named money flow"). A flow of
`spec-money-supply.md` 4 is a grant or a levy: a boundary account on one side and a class
of households of a role and region, or a pool, on the other (4.3). It has no
counterparty that is an agent, and its base is a balance or a fixed amount, never a
draw. Rent has a payer and a payee that are both accounts of the economy and a base that
is the units drawn. Forcing it into MON would need a levy on the tenants to a boundary and
a grant to the lessors, a pair that MON's own check calls `pair_unbalanced` and that
cannot be matched per draw. N-107 allows exactly this case ("a built-in ledger reason only
for a mechanism that is not a grant or a levy") and MON's positions are kept: **no faucet
reads an aggregate** (rent reads the units drawn this tick, a flow of one endowment, not a
balance or a total), and **a counterparty is never a boundary** (the lessors are agents or
the estate). A tax on a holder is a MON levy on the holder role or FIS's tax on it
(`spec-fiscal.md` 5), and is a sink; a tax on rent itself is not offered (`spec-fiscal.md` 24 question 3);
this spec adds none.

5.1.3. **What rent is not.** Not a price in the market: no order, no clearing, no belief.
Not a debt: it is paid at the start of a job or the job does not start (5.4). Not a
commitment: no lease, no term, no tenant's right of renewal (7.2.1 and
section 20, question 3).

### 5.2 The posted rent

5.2.1. Each endowment with a `Terms` row (or a title) has a **posted rent** `P`, the rate
in force before the slot scale, in money per unit drawn from a let unit. It starts at the
row's `Rent` (0 when the row has none). It is **state** when `RentStepBP` is above 0 (5.5,
saved, 14) and otherwise it is the configured value.

5.2.2. **The rent in force** in a tick is

```
r = P * S / 10 000      truncated toward zero, and at least 1 when P and S are both positive
```

with `S` the **rent scale** slot of 9.1 for the endowment, basis points, default 10 000,
0 to `MaxShockBP`. A scale of 0 makes rent free for the tick without changing `P`. It is what a consumer's
policy sets (a rent control is `S` below 10 000), and it is the only in-run lever besides
the step. `r` is read at P1.2 (the cost anchor), P2.3 (the
view), P2.6 and P2.7 (the charge) and P8.3a (the step). A call between ticks is read by
the first step after it (INT 5.2.5).

5.2.3. **Bounds.** `Rent` is at most `MaxPrice` (`spec-config.md` 4.1); the product of `r` and
the units let is computed in 128 bits and an overflow is treated as unaffordable (5.4).

### 5.3 The charge and the payee

5.3.1. **When.** At the start of the job, in P2.7: the principal pays `r` for each unit it
draws from a let unit, and the lessors are paid. P2.7 has two passes over the tick's starts,
which are one atomic step in effect (no other position runs between them, and a start of
pass 1 has reserved its rent in the principal's scratch balance, 5.4.1, so pass 2 cannot
fail for lack of funds):

1. **The starts** take the inputs and the draws, in the order of RES 5.4, each with the
   `k` of 5.4 (the least of the grants, the live inputs, the room and the rent its
   principal can still pay). The let units the starts took in all are `V` (4.4.1).
2. **The charge**: the supplies of the lessors are the apportionment of `V` (4.4.1), the
   attribution of 4.4.2 pairs tenants and lessors, and the transfers of 5.3.3 are made, in
   (`RegionID`, `ResourceID`) order, then by tenant and lessor ID.

The rent of a multi-tick job is charged once, at its start, for the units it drew, as the
inputs are. An abandoned job returns no rent (RES position 14: the draw is not returned).

5.3.2. **How much.** The executions of the job are served in the order own, commons, let
(a job of `k` executions uses `min(k, o_i)` own, then up to `c_i` from the commons, then
the rest from let units). The let units it takes are `max(0, k - o_i - c_i) * d_i`, and its
rent is `r` times that. The units are attributed to lessors in the order of the triples of
4.4.2 for this tenant, taking from the first triple; the supplies are those of the units
actually taken, so a tenant that took fewer units than it was granted (because `k` was cut)
leaves no lessor's supply above what was drawn (4.4.1). The rent paid to each lessor is
`r` times the units attributed to it.

5.3.3. **Transfers.** One transfer per (tenant, lessor) triple used, reason `rent`, from the
principal's account to the lessor's (the agent's account, or the estate account of the
endowment's region for the pool). A transfer whose payer and payee are the same account is
skipped, with the units still counted as let. The payer's side cannot fail (5.4.1 reserved
the money). The **payee's side** is bounded as MON bounds a credit
(`spec-money-supply.md` 6.4.1): a payee whose balance would pass `MaxMoney` receives only
`MaxMoney - balance`; the part that it cannot receive is **not charged** to the payer and is
counted in `RentPaid.Forfeited` (money). The units are still drawn and counted as let. This
makes the transfer total: no transfer of 5.3 can be refused by the ledger
(`ErrBalanceOverflow`, FR-LED-006), and nothing in this step panics. It is a bound that
`spec-money-supply.md` 6.4.2 does not rule out (the supply is bounded, one balance is not),
reached only by an account near `MaxMoney`. The number of transfers a tick is bounded by
tenants plus lessors per endowment.

5.3.4. **An exempt request.** With `OwnUseExempt` set, a request for a recipe of the
principal's effective own-use list (`spec-subsistence.md` 3.6) is charged **no** rent: its
let units are drawn, the lessors are paid nothing for them, and the units still come out of
their unused entitlement (the rule of 4.4 is unchanged, only the amount is 0). It is the one
switch of this spec that takes from a holder, and it is data, default off (position 13).

### 5.4 What a principal that cannot pay gets

5.4.1. At the start the principal's balance is what it can spend: no reservation is open in
P2 (`spec-world.md` 6.1.6). Rent is the only claim on it in this step, and a principal may
have several starts in the tick (several requests, several draws), so the starts of pass 1
(5.3.1) run in the order of RES 5.4 and each **reserves its rent in a scratch balance of the
principal**: the money available to a start is the balance less the rent that the earlier
starts of the same principal reserved. Let `k_other` be the executions the start would take
by RES 5.4 and `R(k)` the total rent of `k` executions over all its draws, `sum over its
draws of r * d * max(0, k - o - c)`. `R` is non-decreasing in `k`. The start uses the
**largest `k <= k_other` with `R(k) <= available`**, found by bisection over `k` (the cost is
`O(log w)` per start; `w` is at most `MaxWeight`). The executions between that `k` and
`k_other` are not started and their units stay in the stock (they are `CutUnits`, and they
are idle let units in the sense of 5.5.1). The clip of 4.3 step 3 makes the cut rare:
it acts before the ration so that a principal with no cash does not take let units from a
principal who can pay (the escrow of wages, P2.5a, is already out of the balance it reads);
the cut here is what the clip could not see: the sum over several requests, or several draws
on several endowments, of one principal.

5.4.2. If that `k` is 0 and `k_other` was positive the request is refused with `rent_funds`
and nothing is taken. No debt, no carried amount, no partial charge: a start that cannot be
paid in full does not begin. A household whose job is refused for rent is a household
that does not earn this tick, and the loop of section 16 (LND-A2) is the cost. A request
whose let demand was clipped to 0 at 4.3 step 3 and that took nothing in the other tiers is
refused with the same reason (4.5.1).

5.4.3. A principal that is a pseudo-agent is never a request's principal. A removed
holder's receipts are paid at the moment of the charge, so there is nothing pending at
removal (6.4).

### 5.5 The rent step

5.5.1. **Rule.** With `RentStepBP` above 0, at P8.3a of every tick, for each endowment that
has a title and is not `Closed`, and after the tick's starts, let

- `X`: the units that were wanted **at the rent in force** and not served in the let tier,
  `sum over requests of (ŵ_i - t_i) * d_i`, with `ŵ_i` the clipped demand of 4.3 step 3. It
  leaves out the units refused `no_title`, which no price clears (they are not asked of the
  let tier), and the units priced out by the clip (`PricedOut`): a tenant that has no cash at
  this rent is not demand that a higher rent serves, and it does not move the rent;
- `V`: the let units the starts took (4.4.1), and `Y = L - V`: the let supply that was not
  drawn, **whatever the reason**: units no request asked, a fragment smaller than one
  execution, and units granted and cut for rent at the start (5.4.1). Units cut for rent are
  idle units, which a lower rent would have cleared;
- `C`: the let units granted and cut for rent at the start, `CutUnits` of the event, a part
  of `Y`.

Let `B` be the configured rent (the row's `Rent`), `step = max(1, B * RentStepBP /
10 000)`, `floor = B * RentFloorBP / 10 000` and `ceil = B * RentCeilingBP / 10 000`. Then

```
if X > 0 and C = 0:  P' = min(ceil, P + step)
elif Y > 0:          P' = max(floor, P - step)
else:                P' = P
```

The test of `X` comes first only when no unit was cut for rent: a fragment of the supply
smaller than one execution (a draw of `d` units cannot take `d - 1` left) lies idle beside
unserved demand, and does not lower the rent; a unit cut for rent is the case that a lower
rent clears, and does.

The new `P` applies from the next tick (P1.2 of the next tick reads it; P8.3a is after the
last read of the tick). A `Closed` endowment's posted rent never moves. A configured `B` of
0 gives a ceiling of 0 and a posted rent that stays 0.

5.5.2. **What it is.** A bounded, one-number tatonnement for the price of access to one
endowment: it rises while demand for the let units is unmet and falls while let units lie
idle (including those cut for rent). A tenant priced out by the clip moves it neither way:
a pack that wants poverty to lower the rent has `PricedOut` in the event and a scenario
(LND-S3), not a rule. The units it reads are those of the endowment itself in this tick, a local market
quantity of the same kind as the unmatched units that move a belief
(`spec-beliefs.md` 5.4), and not an aggregate of the world (WLD 9.4.1). It is **default
off** (`RentStepBP` 0): the rate of a pack that does not set it is the configured one.

5.5.3. **Why a band, and why it is a damper.** The band `[floor, ceil]` is a fraction of
the configured rate, which is the guide of 5.6. The step is bounded, so the rent moves by
at most `step` a tick; the response of demand to it passes through the cost of the recipe
choice (4.6.2, 5.7.1) and the decisions of the agents, with the delays of `spec-stability.md` 4, so the loop is a
balancing loop with a delay and a limit cycle. In the model the half-span of the cycle
about the clearing rent is about `3 * step * (delay + 1) / 4` (`rent.out`, the last table:
a step of 250 basis points of a rate of 40, that is 1, with a response delayed 8 ticks
cycles in 63 to 78 about a clearing rent of 70; a step of 1 000, that is 4, with a delay of
3 reaches the ceiling of 80, 10 above the clearing rent, for an estimate of 12). The cycle
stays inside the band when that half-span is below the distance from the clearing rent to
the nearer of the floor and the ceiling, not below the width of the band. A band that
sits below the market-clearing rent pins the rent at its ceiling and leaves demand unserved:
the band is the damper and also a cap on what the posted rent can discover
(`rent.out`, last lines). It is registered as LND-A1 and LND-D2.

### 5.6 Valued at the dual

5.6.1. The steady-state solver's report `Rent[res][region]` is the value of one more unit of
draw per tick, in money, 0 when the resource row is slack (RES 11.3). A unit drawn is a
unit of that row, so the dual is **money per unit drawn**, the dimension of the posted rent.
The rent a pack sets for an endowment is meant to be that number: the scarcity price of
the stock at the plan share, so that the cost anchor carries the same signal into the choice
of a recipe that the shadow price would.

5.6.2. The validator does not read the solver. `rent_off_dual` is **this spec's** warning,
reported by the solver through VIA's list of warnings (`spec-viability.md` 10.2, which lists it
there; the figure it compares with is `Rent` of RES 11.3 in the report of VIA 10.3): it fires when the configured rent differs from the
dual by more than `RentDualTolBP` (a rule constant, placeholder 5 000: 50%) for an
endowment whose row binds (`Rent` above 0), or when the configured rent is above 0 and the
row is slack. It never fails validation. An endowment whose row is slack has a dual of 0
and rent there is a transfer that the physical programme does not ask for; a pack may still
want it (a title with rent is a distribution of income, not only a scarcity signal).

5.6.3. **What the dual is not.** It is the shadow price of the *physical* row at the plan
stock. Rent as a transfer does not change the programme (12): the flows of goods and
labour are what they were. The programme does not say how the income of the title divides
between the holder's household and the tenants'; that is the money circuit, which a run
shows and the solver does not.

### 5.7 Rent in the cost anchor and in the welfare ratio

5.7.1. **The cost anchor** (`spec-production.md` 9.1) gains a term: for each `Draws` entry
`(res, d)` of the recipe in the region,

```
rent_cost = sum over Draws of  d * r(region, res)
cost      = input_cost + wear_cost + labour_cost + rent_cost
```

with `r` the rent in force of 5.2.2 at the start of the tick (P1.2). It is counted for
**every** unit, a holder's own included: an owner could let the unit, so the rent is the
opportunity cost of using it (5.7.2). The text of PRO 9.1, "Endowment draws are not priced,
because the endowment is not bought; a scarce deposit shows as a rent in the market price,
not in the cost", is amended: with this module on a draw is priced at the posted rent, and
without it, or with a rent of 0, it is not. `unit_cost` is 0 (unknown) under the same
conditions as before; a rent of 0 leaves the cost as it was (FR-LND-020).

The rent has **two readers**, and they are different edits. The cost anchor of PRO 9.1 is
read by the belief update (a cost floor and ceiling on the output's price, `UnitCost`, AGT
6.4), and carries the rent into the price of the output. The default strategy's choice of
a recipe (AGT 7.1.4) does not read the anchor; it reads `Posted` of the view (4.6.2) and
subtracts `sum over Draws of d * Posted` in `cost_r`. Both are owed (18.2). The first raises
the price floor of the output, which if anything raises `value_r`; the second is what makes
rent lower the net value of the drawing recipe and so damp the draw-down (LND-D5, LND-A1).

5.7.2. **Own use is priced at the same rent.** A holder's own entitlement is free in money
and costly in opportunity. The cost anchor is per (region, commodity) and not per agent
(PRO 9.3), so it cannot tell a holder from a tenant, and the opportunity-cost reading is
the one a household with a title would take if it reasoned about the margin. What the
anchor leaves out is that the holder pays no cash: its *income* (the welfare ratio) is
higher than a tenant's by the rent the tenant pays, which is the measured difference that
tenure makes (5.7.3).

5.7.3. **The welfare record** (`spec-welfare-and-wages.md` 2.1) is amended in two lines.
The rent a principal **paid** in the tick is netted in `income_t` like a bid with the
purpose `Input` (it is the cost of the draw, a cost of production, not a personal
expense). The rent a holder **received** is an unearned receipt like relief and a grant
(`G_t`): it counts in `income_t`, and is excluded from the observed wage (4.3 there), which
is the return to labour. A household whose only income is rent is a non-producing
household with a recurring named transfer in the sense of `spec-demand.md` 7.3 item 2; the
warning `class_without_income` is not raised for a role that holds a title (section 18).

5.7.4. **When the commons is slack.** The cost anchor counts the rent in force whether or
not a let unit is drawn. Where the commons serves every request (4.3.1) nobody pays the
rent, and a cost that includes it overstates what a recipe costs: it lifts the belief cost
floor (`CostFloor`, `spec-beliefs.md` 3.2.2) and can freeze a market with perishables (L09,
N-039, RES-A6). Three things keep the overstatement to what a pack chooses: the rate is meant
to be the dual, which is 0 where the row is slack, so a pack that follows 5.6 sets no rent
there and `rent_off_dual` says it did; the rent step with a floor of 0 (the default) takes a
rent that no unit supports to 0 over `P / step` ticks; and the cost floor is below 10 000. A
pack with a positive rent and a slack commons has the overstatement, and `LND-S2` and
RES-S3 with perishables are the scenarios that show it (LND-A1).

### 5.8 The ledger form

5.8.1. Two rows join the registry of `spec-ledger.md` 7.2, declared at construction when
the module is on and closed with the rest of the registry:

| Reason | Of | Kind | Between | Owner |
|--------|----|------|---------|-------|
| `rent` | money | internal | agent to agent or estate | this spec, 5.3 |
| `title_sale` | money | internal | agent to agent or estate | this spec, 6.3 (the price of a title in a `TransferTitles` call) |

5.8.2. A name that collides with a built-in reason, a flow ID or another module's reason
is `ErrReasonCollision` (`spec-money-supply.md` 4.5.2): `rent` and `title_sale` are
reserved words for a flow ID when this module is on.

## 6. Holders over time

### 6.1 Construction

6.1.1. The titles are created in `NewWorld` **after the agents of the groups exist and
before the warm start is applied** (`spec-world.md` 5.1 step 8; a new step 8a, section 18),
in the order of the holdings (3.3.1): for each row, a role holder gives each household agent
of that role whose account is in the row's region, in ascending agent ID, a title of `w *
ShareBP` where `w` is its weight (checked), merged with a title it already holds on the
endowment; the pool holder gives the estate pseudo-agent of the region a title of `ShareBP`.
The sum over an endowment is at most 10 000 by validation (11.1), which computes it from
the census (`Households` of each group of the role in the region, `ShareBP` times their
number), so construction cannot fail on it. A group raised to `MinAgents` has the same
households and so the same total whatever its agents' weights (`spec-scale.md` 3.2).

6.1.2. **The cap.** `MaxHolderShareBP`, when below 10 000, bounds the share one agent may
hold, the pool excepted. At construction the largest agent of the role's group is the one
that holds the most (`w` is largest there), and `ErrTitleCap` is raised when `w_max *
ShareBP` exceeds it (11.1). It is the damper of 16 (LND-D1) and defaults to off.

6.1.3. **No event, no draw, no rent.** Construction emits no event for a title: the holdings
are a function of the configuration and the census, and a consumer reads them with the
queries of 9.3. A job that the warm start creates at construction with a draw is a virtual
past start (`spec-world.md` 5.6): no rent was charged at it and none is.

### 6.2 How a title is acquired

Three ways, and no other:

| Way | When | By whom | Effect |
|-----|------|---------|--------|
| the configuration | construction | `Holdings` rows (6.1) | per household of a role, or the pool |
| a call | between ticks | the consumer, `TransferTitles` (6.3) | a share moves from a holder to another, or between the commons and a holder, with an optional price |
| inheritance | at a removal | the world (6.4) | the removed agent's titles go to the pool of each endowment's region |

6.2.1. **No agent decides to acquire one.** No strategy, spawner or module of version 1
returns a request for a title. A household that has money and sees a good rent does nothing
about it: that is an investment decision, SAV's, and it needs a field of the plan a
`Strategy` returns (`spec-agents.md` 6) that this spec does not add (position 11). A
consumer that models a lessor or a player buying a share makes the call itself, between
ticks, with a price taken from the player's account. A later spec that adds a decision
calls the same transfer (6.3) and does not change this spec. `spec-saving-and-investment.md` 6.3
leaves the decision open and says what it would need (an observable return, a seller).

6.2.2. **Why leave the decision out.** A purchase is a choice with a price, a budget and an
expectation, and a household that buys a title forgoes consumption now for a stream of rent:
the decision belongs with saving and with credit (`vision.md` 4, "Later"). Putting a naive
rule here (buy when rent exceeds the price over a horizon) would settle SAV's questions
(hoarding against investment, the paradox of thrift, the credit cycle) by a default that a
later spec would have to undo. Without a buyer the concentration loop is a hazard of the seam
and not of the engine (LND-A5).

### 6.3 The call `TransferTitles`

The public call is INT's to name and bound (9.2). The mechanism:

```go
// TitleTransfer moves a share of one endowment between two holders.
type TitleTransfer struct {
	Region   RegionID
	Resource ResourceID
	From, To AgentID // NoAgent (the constant -1, below every agent ID and every RegionID) names the commons
	ShareBP  int64   // 1 to 10 000
	Price    Money   // paid by To to From with the reason title_sale; 0 is a gift
}
```

6.3.1. **Meaning.** A share of `ShareBP` basis points of the endowment moves from `From`
to `To`. `From` is an agent that holds at least that share, or the pool of the endowment's
region, or `NoAgent`, which takes it from the commons share (an **enclosure**: the commons
share falls). `To` is an agent, or the pool, or `NoAgent`, which gives it to the commons (a
**release**). `From` and `To` may not both be `NoAgent` and may not be equal. A holder that
ends with a share of 0 has no title.

6.3.2. **The price.** A positive `Price` is a transfer of money from `To` to `From`, reason
`title_sale`, made at the call (no reservation is open between ticks), in the same
atomic step as the move of the share. Where one side is `NoAgent` the price is paid to or by the pool of the
endowment's region: the pool is the payee of an enclosure and the payer of a release. A price moves existing money between accounts and creates none
(FR-LND-025).

6.3.3. **Atomic, one entry per key, an event per change.** The rules of `spec-integration.md`
3.3 apply: the call validates every entry in the canonical key order on a working copy and
applies none if any fails; a key is (region, resource, `From`, `To`) and may appear once
(`ErrDuplicateID`); the entries are applied in canonical order (ascending region, resource,
`From`, `To`), so that two entries on one endowment give the same result whatever the
order of the slice. The events are `TitleTransferred`, one per entry, at the call, with
`Cause` `consumer` (INT 3.3.3: the cause of every event of a call between ticks).

6.3.4. **Checks**, in the order of `spec-integration.md` 3.2: the module is on; each entry's
region, resource and agents are known and live (`ErrUnknownRegion`, `ErrUnknownResource`,
`ErrUnknownAgent`, `ErrAgentRemoved`, `ErrNotAHolder` for an agent that cannot hold (an
estate of another region, a counterparty of the outside) and `ErrOutsideNode` for an outside
node's region); the share and price are in bound (`ErrOutOfBounds`); `From` holds the share
(`ErrNoTitle`); `To`'s share after would not exceed `MaxHolderShareBP` unless `To` is the pool
(`ErrTitleCap`); `To` holds the price (the ledger's `ErrInsufficientFunds`) and the payee's
balance would not pass `MaxMoney` (`ErrBalanceOverflow`, mapped to `ErrOverflow`, AGT 13).
The sum of shares is not above 10 000 after any entry: enclosure takes from the commons, so
it is checked against `s_0` (`ErrNoTitle`).

6.3.5. **What it does not do.** No title is created from nothing: an enclosure moves part
of the commons into a title and a release moves it back. A transfer changes no stock, no
rent and no cost anchor of its own; the next tick's entitlement is the new share of the
snapshot stock (4.2). A call that moves a title away from a holder with a job in progress
changes nothing about that job: its draw was taken at the start (RES position 14) and its
rent paid.

### 6.4 How a title is lost: removal

6.4.1. At a removal (`spec-agents.md` 3.6), after the contracts of step 2a are settled and
before the money moves (step 2b of AGT 3.6, before EMP's 2c and CRD's 2d and 2e), each title of the removed agent moves to the pool of the title's
**endowment region** (not the region the agent was in): the pool's share becomes the sum
of its share and the removed one's, and the commons share is unchanged. One event
`TitleTransferred` per title with `Cause` `removal`, in ascending (`RegionID`,
`ResourceID`), precedes `AgentBankrupted` (WLD 6.11 item 8, section 18). No money moves:
nothing was owed. A pool that already holds the maximum is not special: the sum cannot exceed
10 000 because the removed share was part of it.

6.4.2. **What follows.** The pool's title is let, in full, to anyone the commons did not
serve (4.3 step 3 with `U` the whole entitlement), and its rent goes to the estate account;
or it lies idle if the endowment is `Closed`. Nothing returns it to a household in a run:
the next holder is a consumer's call (6.3) or no one. The share of titles that has passed to
the pool after `n` removals at a rate `q` of the holders is `1 - (1 - q)^n`: with an annual
rate of 1.5% half of the titles are in the pool after 46 years and 78% after a hundred
(`erosion.out`; a model of the rule, with births that are landless, and not a claim about
any population). It is a **bias with no loop**, registered as LND-A4: a world run long enough
with deaths and no re-grant is a world of public titles, which is the open-access regime
with a fee and the pool as its collector. A pack that does not want that gives its consumer
the call, or keeps the horizon short.

6.4.3. **Weight 1 and cohorts.** A cohort is removed whole or not at all: a reduction by
the failure ladder lowers its weight and leaves its titles (6.5). Only the removal of the
agent moves them.

### 6.5 What a weight change, a role change, a relocation and a restore do

| Event | Effect on the titles of the agent |
|-------|-----------------------------------|
| weight change (a reduction by the ladder, births added to the agent, a death of the population module) | none: the title belongs to the agent and not to its households; the cohort's entitlement is unchanged |
| role change (the spawner, a traineeship) | none: a title is not tied to a role |
| relocation (a migration, a transport arrival, INT's `Relocate`) | none: the holder is absent from the endowment's region, its entitlement is entirely unused there and let (4.3), and its rent keeps arriving; a household that now lives elsewhere requests in its new region where its titles are irrelevant |
| removal | all titles to the pool of each endowment's region (6.4) |
| a restore (`spec-checkpoint.md`) | the holdings are state and come back exactly (section 14) |

6.5.1. **Saved once (N-128).** A title is saved once, in this module's section, keyed by
(endowment, holder agent ID). The agent record (`spec-checkpoint.md` 4.2, tag 3) carries no
field for it, so the rule "a hosted per-agent field is saved once, in the agent record" is
met by there being no hosted field: the title is the module's table, sorted by agent
ID, and the restore checks each holder against the agents of tag 3 (14.2).

### 6.6 The refusal that is not registered (N-090)

The rule of N-090 is that a module that keeps an agent's account in a region while it
is away or committed registers a `region_bound` refusal. **This module registers none.**
A holder is not committed to a region by its title: rent is a transfer between accounts
that needs no co-location, and the entitlement is read from the executor's own region,
where its job runs. A tenant has no commitment either: it draws and pays at each start and
holds nothing between starts. The refusals that bind an agent (`open_shipment`,
`on_journey`, `open_contract`, `in_traineeship`, `open_hire`) are untouched. If a later
spec gives tenure a term (a lease with a right of renewal, ORG's charter that must be worked
in place) the term is that spec's and it registers its own refusal by the rule.

## 7. What tenure does to the specs around it

### 7.1 The subsistence floor and the commons (SUB 9.5)

7.1.1. **An own-use request is a request.** It carries its recipe and its principal and
is served in the four tiers like any other (4.3), pays rent on its let units unless
`OwnUseExempt` is set (5.3.4), and is refused with `no_title` or `rent_funds` as any request
is. SUB itself changes nothing (`spec-subsistence.md` 9.5: "SUB itself charges nothing and
reserves nothing").

7.1.2. **What a household meets.** A household in the own-use mode whose request is
closed to it, or that cannot pay the rent of what it was granted, starts nothing and is
`Blocked` in the count of SUB's step (SUB 5.5 item 6, `own_use_not_available`); the two new
reasons join `no_endowment` there (section 18). With `OwnUseExempt` false and every unit of
the endowment titled and closed, the floor under market failure is gone for a household
that holds nothing: this is **enclosure**, shown, and is the intended effect. With
`OwnUseExempt` true it draws the let units and the commons for nothing, and its principal's
lessors are paid nothing for those units.

7.1.3. **The right of the landless** SUB names is `OwnUseExempt`: one boolean for the
module, not per role or per endowment, so that a pack that wants a subsistence right on
some endowments and not others has to model them as different resources. A per-endowment
form is a row field (open question 4).

7.1.4. **Interaction with the failure ladder.** A household that cannot start its own-use
job goes unfed, its `Privation` rises, and the ladder of `spec-spawner.md` 5 acts: the
tenure shows up as a population response and not as an error (LND-A6).

### 7.2 Migration (MIG)

7.2.1. **Nothing to register** (6.6). A tenant who leaves loses nothing but the draw of its
running job, which MIG's departure already abandons with `region_changed` and does not
refund (RES position 14).

7.2.2. **The estimate does not read rent.** MIG's estimate of the income a destination
offers uses the table's yield, the upkeep and rumoured prices (`spec-migration.md` 6.3.3, N-108
second update): "a destination's stock, endowment or abundance is state and is not read".
The posted rent and who holds the land there are state too, and a rumour carries prices
only (N-110). A household that moves to a region of high rent finds out at its first start
(`rent_funds`, a lower income) and is pushed again by the settling rule after its window.
This is a cost stated and a hazard registered (LND-A6); a rumoured rent would be a second
kind of rumour (INF open question).

7.2.3. **Titles do not move with a household's decision to leave.** A holder that migrates
keeps its titles (6.5) and earns rent from a region it left; a pack that wants a title to
follow residence makes the call.

### 7.3 Hired labour and traineeships (EMP, SKL)

The principal of a request is the principal of its job (4.7.2), so tenure applies to a
hired or trained job by the employer's or trainer's titles, and rent is paid from the
principal's account. EMP's `Bid` is `floor(net_r / Duration_r)` with `net_r` of AGT 7.1.4
(EMP 4.3.3), so with the edit of 4.6.2 it carries the rent in force; a vacancy that the
principal cannot pay for even at the least rent (4.7.2) is refused at its formation with
`rent_funds`, and the check of FR-EMP-023 gains the two reasons (section 18). The wage and
the rent are two payments
from one account at two positions (`spec-employment.md`: the wage is escrowed at P2.5a,
rent is paid at P2.7); a principal that spent its balance on wages has none left for rent,
and the job is cut at the start (`not_started` when nothing starts, the escrow refunded,
EMP 6.1.3), which is the poverty trap of LND-A2 seen from the employer.

### 7.4 Storage (STO)

The order of the refusals is `no_endowment`, `no_title`, `rent_funds`, then CAP's build
reasons if the infrastructure module is on, then `no_room` (position 14; the master order of
`spec-production.md` 4.2). The start takes the least of the executions that inputs, grants, rent and
room allow; each limit depends only on `k`, so they are independent and the order only
names the reason (FR-STO-011 gains the two reasons ahead of `no_room`).

### 7.5 Population and the Malthusian loop (POP, RES 8)

Births are landless: a household born into a region where every unit is titled and closed
meets `no_title` (7.1) and a rent where it is not. The closed forms of RES 8 are about
the physical stock and are unchanged. What changes is who gets the surplus: the ratio a
household reads is its own, net of rent if it is a tenant, so the **role** of tenants has a
lower ratio than the role of holders at the same draw, its births fall and its deaths rise
(POP 3.2: births enter the parents' role). The Malthusian fixed point is then a point of
the tenant role and not of the region, and the rent is the share of the output above the
subsistence that the holders take. This is a consequence and not a mechanism of this spec,
and it is the Ricardian outcome the module exists to allow; it is registered with
LND-A2 and LND-A4.

### 7.6 The pool and relief (EST)

Rent paid to the pool is money that leaves households (M_h) for the pool sector (M_p). It
returns by the pool's uses only: relief at `PoolRelease`, starting money for entrants within
`EntryPoolShare`, and nothing else (`spec-money-supply.md` 7.4). A pool with `PoolRelease`
0 and no entrants holds the rent out of circulation, which is the hoarding of STB-L31 by
another path (LND-A7). With the estates module off, the pool has no policy and the rent
accumulates until a consumer's call or a flow moves it (the warning `rent_hoarded`, 11.2).

### 7.7 What the other later specs will use

| Spec | What it takes from this one | Where |
|------|-----------------------------|-------|
| CAP (`spec-infrastructure.md`) | A facility that raises capacity is a layer multiplied into the effective capacity (`spec-infrastructure.md` 5.3), not a write to the capacity slot of RES 6.3; the stock rises and every title's entitlement with it, so the gain of an improvement accrues to holders in proportion to their shares. Who may build on whose title is CAP's | 3.1.3 |
| FIS (`spec-fiscal.md`) | A tax on rent is not offered; a tax on the holder role is `spec-fiscal.md` 5 (a sink); rent itself is a transfer | 5.1.2 |
| ORG (`spec-institutions.md`) | A rent control is the slot of 9.1 or a price control in the sense ORG settles; a charter or monopoly is a `Closed` row with a title to the chartered agent, set by the call; a refusal reason that ORG adds follows `rent_funds` | 9.1, 4.5 |
| RSK (`spec-risk.md`) | Loss of a title by conflict is not RSK's in version 1 (RSK names no cause and has no actor); it is the consumer's call of 6.3; a cargo or a stock lost is RSK's `loss`, not tenure | 6.3 |
| SAV (`spec-saving-and-investment.md`) and CRD (`spec-credit.md`) | The decision to buy a title and the credit to do so call `TransferTitles`; the price is `title_sale`; the cap is the damper of LND-D1 | 6.2 |
| STO (`spec-storage.md`) | A lease of room, a keeper, is a contract over room; if it reuses the reason `rent` that is the owner's call and the reason is reserved (5.8) | 5.8 |
| SET (`spec-settlements.md`) | No hook: the class of a region gates no title. A pack that ties tenure to size does so by configuring titles per region | none |
| POL (`spec-polity.md`) | A jurisdiction is the key (POL 5, 6.1.2); a table keyed by jurisdiction over titles is this spec's or ORG's, with the combining modes of `spec-polity.md` 6.1.2; a holding is by (region, resource), and a region is in one jurisdiction | none |
| TEC (`spec-technology.md`) | A yield factor raises the output per drawn unit and so the rent a unit can bear; it changes no rule here | none |

### 7.8 Markets, beliefs, the clearing

None. Tenure does not touch an order, a belief or a clearing: the price of a drawn
commodity moves only through the cost anchor (5.7) and, for a staple, through the
supply the tiers allow; the choice of a recipe moves through `cost_r` (4.6.2).

## 8. Position in the tick and draws

### 8.1 The positions

The positions are those of `spec-world.md` 6 and one is new. Each row says what this
module does there, in addition to what its owner does.

| ID | What this module does | Reads / writes | Draws | Events |
|----|-----------------------|----------------|-------|--------|
| P1.2 | the cost anchor adds the rent of the draws (5.7) | reads `r` | none | none |
| P2.3 | with the snapshot of every endowment, the entitlements of the holders and the commons (4.2), for each endowment with a title | reads the stock and the holdings; scratch | none | none |
| P2.5 | the view carries `Posted`, `ShareBP`, `Entitled`, `Closed` and the new reason of `MaxExecutions` (4.6) | scratch | none | none (a refusal is a `Debug` line) |
| P2.6 | the ration is the four tiers (4.3), the let demand clipped to what each principal can pay (4.3 step 3) | requests, entitlements, balances; scratch | none | none |
| P2.7 | pass 1: the largest `k` whose rent the principal can pay, reserved against its other starts (5.4); pass 2: the lessor supply and the attribution (4.4) on the units taken and the transfers of rent (5.3) | balances; writes balances, stock | none | none |
| P7.1 | at a removal, the titles of the agent move to the pools (6.4) | holdings | none | `TitleTransferred` before `AgentBankrupted` |
| **P8.3a** (new) | after regeneration and `EndowmentChanged`: the rent step of every endowment that has one (5.5), then the event `RentPaid` | the tick's counters; writes the posted rent | none | `RentPaid` per endowment, in (`RegionID`, `ResourceID`) order |
| construction, step 8a (new) | the titles of the configuration (6.1) | writes the holdings | none | none |

8.1.1. **Why P8.3a and not P2.8.** The counts the rent step reads (units served in the
three tiers, unserved, idle) are complete only after the starts of P2.7, and the rent that
the next tick reads at P1.2 must be fixed by the end of this one. P8.3 is the regeneration
(RES 4.1) and its event; P8.3a follows it so that `RentPaid` is after `EndowmentChanged`
in an event order a recording observer sees (`RentPaid` carries the refusal counts the
other event does not). The new position takes a letter after the one it follows
(`spec-world.md` 6.1.1) and the events order of 6.11 item 9 becomes `BoundaryFlow`,
`EndowmentChanged`, `RentPaid`, then the events of CAP's P8.3b (`spec-infrastructure.md` 6.3,
which follows this step and keeps this one's name), `TickEnded`.

### 8.2 Draws

**None.** Every tie of the apportionment and of the ration goes by the stateless key of RES
5.3.3 and then by agent ID (4.2, 4.3), which is a function of the tick and the ID. The
rent step and the attribution are deterministic functions of the counters. The number of
words of a tick, the table of WLD 7.3, is unchanged, and the stream of a world does not
depend on whether the module is on (FR-LND-036; the rule of WLD 7.4.2). The order of the
stateless key is the one tie rule of WLD 10.1.1 and needs no reason written here.

### 8.3 The cadence

A rent step is every tick, with no cadence, no window and no gate; the device is a bounded
step (WLD 9.1 names four devices, and this is a rule of the posted number and not a rate at
which agents decide). The row for WLD 9.2:

| Cadence | Parameter | Owner | Default | Position | How the tick counter selects it |
|---------|-----------|-------|---------|----------|---------------------------------|
| The posted rent | `RentStepBP`, `RentFloorBP`, `RentCeilingBP` | LND 5.5 | 0 (the rent does not move); 0; 20 000 | P8.3a | every tick; a bounded step on the counters of that tick |

## 9. Slots and calls

The mechanism is this spec's; the public name, error values and bounds are INT's
(N-091; `spec-integration.md` 5). The names below are proposals for INT's table.

### 9.1 The slot: the rent scale

| Call | Entry (key; value) | Value bound; default | Needs | Owner | Read by the tick at | Event (at the call) | Saved by |
|------|--------------------|----------------------|-------|-------|---------------------|---------------------|----------|
| `SetRentScales` | `EndowmentScale{Region, Resource; BP}` | 0 to `MaxShockBP`; 10 000 (0 makes rent free) | tenure | LND 5.2 | P1.2, P2.3 (the view), P2.6, P2.7 | `RentScaled` (new) | LND 14, by (region, resource) |

The entry type is `EndowmentScale`, the one of the capacity and regeneration scales of RES 6.3.1
(`spec-integration.md` 5.1): the slot is told apart by its name and its own saved table, and
there is no type of its own. The slot is a lever kind of the policy module (`rent`,
`spec-policy.md` 6.3.2): reserved in version 1, offered when that module's slot row says so.

An entry equal to 10 000 clears the key (`spec-integration.md` 5.2.1). The slot never
edits a configured rent (5.2.2); the effective rate is the posted rent times the scale. A
scale does not move the posted rent: the step (5.5) acts on `P` and reads the units
unserved, so with a scale of 0 (free rent) demand for the let units rises and the step may
raise `P` while nobody pays it. A consumer that clears the scale gets the rent in force that
the step made of `P`, not the configured one; a pack that wants the configured rate back
sets `RentStepBP` to 0 or accepts the drift.

### 9.2 The adjustment: titles

| Call | Entry | Bound | Needs | Owner | Acts | Event | Saved by |
|------|-------|-------|-------|-------|------|-------|----------|
| `TransferTitles` | `TitleTransfer{Region, Resource, From, To; ShareBP, Price}` | `ShareBP` 1 to 10 000; `Price` 0 to `MaxMoney`; at most `MaxTitleTransfers` entries | tenure | LND 6.3 | **at the call**: the share moves now; the first step reads it at P2.3 | `TitleTransferred` (`Cause` `consumer`) | LND 14 (the holdings) |

It is an adjustment in the sense of `spec-world.md` 8.2 (as `AdjustEndowments` is), not a
slot: it has no default to clear and `Slots()` does not list it.

### 9.3 The queries

| Query | Returns | Class |
|-------|---------|-------|
| `Titles(a AgentID) ([]TitleState, error)` | the titles of an agent, or of an estate pseudo-agent (the pool of its own region) in ascending (region, resource); `ErrModuleOff`, `ErrUnknownAgent` (a counterparty, or an ID that was never an agent), `ErrNotAHolder` | 2 |
| `Holders(region RegionID, resource ResourceID) (HoldingsState, error)` | the commons share, `Closed`, the posted rent, the scale and the holdings in ascending holder ID; `ErrModuleOff`, `ErrUnknownRegion`, `ErrUnknownResource`, `ErrOutsideNode` | 2 |
| `Rent(region RegionID, resource ResourceID) (int64, error)` | the rent in force (5.2.2) | 1 |

```go
// TitleState is one title.
type TitleState struct {
	Region   RegionID
	Resource ResourceID
	Holder   AgentID // the estate pseudo-agent's ID is its RegionID
	ShareBP  int64
}

// HoldingsState is the tenure of one endowment at the boundary.
type HoldingsState struct {
	CommonsBP int64
	Closed    bool
	Posted    int64 // the posted rent before the scale
	ScaleBP   int64 // the rent scale (10 000 by default)
	Titles    []TitleState
}
```

All three are queries in the sense of `spec-integration.md` 6.1: no draw, no state, no event,
a deep copy of plain data. A query is legal in the construction phase.

### 9.4 Errors

The sentinels this spec adds to the root package are `ErrNoTitle` (a `From` that does not
hold the share), `ErrTitleCap` (a holder above `MaxHolderShareBP`), `ErrTitleShares` (a
configuration whose shares of one endowment sum above 10 000, or a share outside 1 to
10 000) and `ErrNotAHolder` (an agent that cannot hold a title). Every other check
reuses an existing sentinel (`ErrModuleOff`, `ErrUnknownRegion`, `ErrUnknownResource`,
`ErrUnknownAgent`, `ErrAgentRemoved`, `ErrUnknownRole`, `ErrOutsideNode`, `ErrOutOfBounds`,
`ErrDuplicateID`, `ErrOverflow`) and the ledger's `ErrInsufficientFunds` for a price. The
root needs a mapping for none of them from an `internal/` error but those that
`internal/tenure` returns (`ErrTitleShares`), per N-075.

## 10. Parameters

All are tunables (architecture.md 9.5), fields of `Config.Modules.Tenure`, each with a
default, a bound, an owner and a doc line, and a row in the parameter-owner table
(section 18). **Every default is structural or a placeholder** (N-015, N-027); none comes
from a source, and the neutral default is the open access of RES.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Module | `RentStepBP` | the step of the posted rent each tick, basis points of the configured rate (5.5); 0 means the rent does not move | 0 | 0 to `MaxRentStepBP` |
| Module | `RentFloorBP` | the lowest posted rent, basis points of the configured rate | 0 | 0 to 10 000 |
| Module | `RentCeilingBP` | the highest posted rent, basis points of the configured rate | 20 000 | 10 000 to `MaxRentCeilingBP` |
| Module | `OwnUseExempt` | an own-use request pays no rent (5.3.4) | false | a bool |
| Module | `MaxHolderShareBP` | the most one agent may hold of one endowment, the pool exempt | 10 000 (off) | 1 to 10 000 |
| Term | `Region`, `Resource` | the key of the row | required | an existing region and resource; not an outside node |
| Term | `Rent` | money per unit drawn from a let unit (5.2) | 0 | 0 to `MaxPrice` |
| Term | `Closed` | the unused entitlement is not let (4.3) | false | a bool |
| Holding | `Region`, `Resource` | the key | required | an existing region and resource |
| Holding | `Holder` | `role` with a role ID, or `pool` | required | an existing role, or `pool` |
| Holding | `ShareBP` | per household for a role, the total for the pool | required | 1 to 10 000; the sum over the endowment at most 10 000 |

The bounds are in `bounds.go` (`spec-config.md` 4.1): `MaxRentStepBP` 10 000,
`MaxRentCeilingBP` 100 000, `MaxHoldings` 4 096 (rows) and `MaxTitleTransfers` 1 024 (entries
of one call), all placeholders. `MaxPrice`, `MaxShockBP`, `MaxMoney` and `MaxEndowment` are
the existing constants.

10.1. **Why these defaults.** No title and no rent is the open access of RES, which is the
only baseline; a rent that does not move is a datum, not a controller; a cap that is off
means the damper is not in force by default (`spec-calibration.md` 4.3.1: a calibration
moves it away from off when a fact shows the loop, never back). A band of 0 to 20 000
basis points of the configured rate is nothing to twice it: a placeholder that is wide on
purpose, because its centre is the dual and the dual is a report, not a measurement
(5.6), and whose floor is 0 so that a rent that scarcity no longer supports can fall to
nothing and leave the cost anchor (5.7.1). No default is called stable (N-027).

10.2. **The rule constants** are not fields: the key of the tiers (RES 5.3.3), the order
own, commons, let (4.3.1), the largest remainder of 4.2.1, the walk of 4.4.2, the minimum of
1 in 5.2.2, the tolerance `RentDualTolBP` of 5 000 of 5.6.2 (a warning threshold, not a
result), and the reasons `rent` and `title_sale` (5.8).

## 11. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9); errors name the path, for
example `modules.tenure.holding[2].share_bp`.

### 11.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrModuleDependency` | `Modules.Tenure` present with `Modules.Resources` nil |
| `ErrUnknownRegion`, `ErrUnknownResource`, `ErrUnknownRole` | a term or a holding that names a region, a resource or a role that is not defined |
| `ErrOutsideNode` | a term or a holding whose region is an outside node |
| `ErrDuplicateID` | two terms for one (region, resource); two holdings for one (region, resource, holder); a role whose ID is the word `pool`, which the holder namespace reserves |
| `ErrOutOfBounds` | a field outside the bound of 10; more than `MaxHoldings` holdings |
| `ErrTitleShares` | a `ShareBP` outside 1 to 10 000; holdings whose total on one endowment, with the households of each role's groups in the region, exceeds 10 000 (a role holding contributes `ShareBP` times the households of its groups there) |
| `ErrTitleCap` | a holding whose largest agent would hold more than `MaxHolderShareBP` (6.1.2); the pool is exempt |
| `ErrReasonCollision` | a money flow whose ID is `rent` or `title_sale` (5.8.2) |
| `ErrOverflow` | the derived check: a term's `Rent` times the largest draw of a recipe on its resource times the largest weight of a group in the region above `MaxMoney` |

An unknown resource in a recipe's `Draws` is RES's (`ErrUnknownResource`, RES 13.2). A term
or holding for an endowment whose capacity is 0 is not an error: it is the warning
`holding_unused`, since a title to nothing is legal.

### 11.2 Warnings

A warning never fails validation (`spec-config.md` 7.2). Each has a rule name and a path;
the conditions that need the solver are reported through it (12).

| Warning | Condition |
|---------|-----------|
| `no_titles` | the module is present and has no holding and no term with a rent or a closing |
| `rent_without_titles` | a term with a positive `Rent` on an endowment that no holding gives a title: no unit can be let and no one is paid |
| `titles_without_rent` | an endowment with titles whose rent is 0 and that is not closed: tenure only gives priority |
| `holding_unused` | a holding for a role that has no group in the region, or a title to an endowment whose capacity is 0 |
| `role_locked_out` | an endowment whose titles sum to 10 000 and that is `Closed`, and a role with a group in the region that lists, or has on its effective own-use list, a recipe that draws on it, and that holds no title: that recipe can never start there (`no_title`). SUB's `own_use_not_available` is its own-use form; both may fire |
| `entitlement_below_unit` | the plan stock of an endowment (RES 11.1) times the smallest share of a holder is below one unit: the entitlement of that holder flickers by the key (4.2.3) |
| `rent_above_output` | a recipe whose rent per execution at the configured rate, `sum of d * Rent`, reaches the value of its first output at the reference price times the yield at the mean site factor: it cannot pay for itself; informational |
| `rent_hoarded` | the pool holds a title or may be given one by removal, and the estates module is off or `PoolRelease` is 0 in the region: rent paid to the pool is not released (7.6) |
| `rent_step_inert` | `RentStepBP` above 0 and every term is `Closed` or has no rent: the step never moves |
| `rent_off_dual` | reported by the solver (5.6.2, 12) |

### 11.3 Not validated here

Whether a configured rent is the dual of the row, whether a role's demand for a resource
fits its entitlement and the commons, and whether the economy circulates the rent at all are
the solver's and the scenarios' (12, 16). The validator sees structure, not flows.

## 12. Viability

12.1. **The programme is unchanged.** The steady-state programme (`spec-viability.md` 3)
is a linear programme of physical flows per tick, and the row of an endowment is the
sustainable draw (RES 11.1). A title does not change the sustainable draw, a rent is a
transfer of money between households of the programme and moves no good, and the
entitlement does not change who may draw in total. The solver is therefore not edited for
tenure and the check of RES 11 holds with the module on.

12.2. **What the solver reports to this spec.** The dual `Rent[res][region]` (RES 11.3) is
the guide of 5.6; the warning `rent_off_dual` is its form, and VIA reports it. An
endowment whose row binds has a positive dual; a pack that has titles there and a rent of 0
has `titles_without_rent` and `rent_off_dual` together. VIA's `Headroom` (RES 11.3) says
whether the row is slack (above 1) or binds (below 1), which is the condition under which
tenure has a margin to price at all: with the stock slack the commons serves everyone and
rent is never charged (4.3.1).

12.3. **What viability does not see.** The programme sums the draws of every role on one
row: it does not say that the draw of a role that holds no title is **zero** under a closed
tenure, so a configuration can be viable and have a role that cannot work (`role_locked_out`
is the static guard, and it is conservative: it needs the endowment fully titled and closed).
A per-role split of the row, with the entitlement of a role as an upper bound, is a second
row family that VIA does not have (open question 5). The warm start's money seeds
(`WarmStart.Groups`, `Pools`) are not computed from the rent circuit either: a pack with a
holder role and a tenant role seeds each at its table and the run finds the circulation
(open question 6; the closed-economy results of `spec-money-supply.md` 8 are unchanged,
since rent moves money and creates none).

## 13. Observability

The rules are those of `observability-and-logging.md`; the catalogue edits are owed
(section 18, N-126).

### 13.1 Events

Three events, plain data with no map and no pointer, emitted after the transition they
describe (`observability-and-logging.md` 4.1, 4.4). The typed IDs are `RegionID`,
`ResourceID` and `AgentID`; the strings are in the names table of `spec-observability.md`
3.2. An event of a module that is off is not emitted.

| Event | Owner | Position | Class | Gate | Fields (besides `Tick`) |
|-------|-------|----------|-------|------|-------------------------|
| `RentPaid` | LND 13.1 | P8.3a, (`RegionID`, `ResourceID`) order, after `EndowmentChanged` | A (aggregate) | tenure | `Region`, `Resource`, `Rent` (the rent in force that tick, per unit), `Entitled` (units of entitlement of holders, `A - E_0`), `OwnUnits`, `CommonsUnits`, `LetUnits` (units drawn in each tier; `LetUnits` is `V` of 4.4.1), `Offered` (let supply `L`), `Idle` (`L - V`), `Held` (unused entitlement not let because the endowment is `Closed`), `Unserved` (`X`, 5.5.1), `PricedOut` (units clipped for cash, 4.3 step 3), `Tenants` (requests that took let units), `Lessors` (holders paid), `Paid` (money moved), `Forfeited` (money not charged because a payee could not receive it, 5.3.3), `NoTitle`, `RentFunds` (requests refused for each reason), `CutUnits` (let units granted and not taken for rent, `C`), `Exempt` (let units drawn rent-free), `RentAfter` (the posted rent after the step, before the scale), `ToPool` (money received by the pool of `Region`), `From` (list of `RegionMoney`: the money paid, by the region of the payer's account, ascending `RegionID`), `To` (the same, the money received by lessor households, by the region of the payee's account); `RegionMoney` is `{Region, Money}`, plain data, one entry per region with a non-zero amount |
| `TitleTransferred` | LND 6.3, 6.4 | a removal (before `AgentBankrupted`), or a call | E (per title moved) | tenure | `Region`, `Resource`, `From`, `To` (`NoAgent` for the commons), `ShareBP`, `Price Money`, `Cause` (`removal`, `consumer`), `FromAfter`, `ToAfter` (the shares after, basis points), `FromRegion`, `ToRegion` (the region of the account that pays or is paid the price: of the agent's account, or `Region` for the pool and for `NoAgent`), `FromPool`, `ToPool` (the account is the pool of that region: the pool itself, or `NoAgent`, whose price the pool pays or receives, 6.3.2) |
| `RentScaled` | LND 9.1 | call | C | tenure | `Region`, `Resource`, `From`, `To` (the scale before and after, basis points) |

13.1.1. `RentPaid` is emitted for an endowment that has a title in a tick in
which any of `OwnUnits`, `CommonsUnits`, `LetUnits`, `Idle`, `Held`, `Unserved`, `PricedOut`,
`NoTitle`, `RentFunds`, `CutUnits`, `Paid` or `Forfeited` is non-zero, or the posted rent
changed (`RentAfter` is not the posted rent at the start of the tick), and never otherwise.
In a tick in which any unit was drawn from a tier the event is therefore emitted, so the
identity `OwnUnits + CommonsUnits + LetUnits = EndowmentChanged.Drawn` of the same endowment
and tick always holds where `EndowmentChanged` is emitted with a positive `Drawn`, and the
counter `economysim.rent.units` (13.3) is complete. It is the counter that a silent damper
needs (N-045): the cut for rent, the clip, the closure and the band are all counted here, by
region and resource, with no per-agent event. Its count a tick is bounded by regions times
resources.

13.1.1a. **Money, for the rebuild of OBS 4.3.3.** A rent moves money between household
sectors of regions and to the pool of `Region`, and a title sale moves it between sectors of
two regions or a sector and a pool. The sums are in the event, by region, so that an observer
with no agent table (OBS 3.3.7) rebuilds `MoneyHouseholds` and `MoneyPool` of every region
(FR-OBS-041): for a `RentPaid`, the households of region `g` lose `From[g]` and gain `To[g]`,
and the pool of `Region` gains `ToPool`; `Paid = sum of From = sum of To + ToPool`, a pool
is never a payer (5.4.3). For a `TitleTransferred` with `Price` above 0, the payer's side
(`To` agent, or the pool when `ToPool`) of region `ToRegion` loses `Price` and the payee's
side (`From`, or the pool when `FromPool`) of region `FromRegion` gains it; the pool's side is
the pool sector and the household's side the household sector. A removal moves no money
(6.4.1) and its `Price` is 0. The edit to OBS 4.3.3 (a new item 8) is in 18.2.

13.1.2. `TitleTransferred` is per title moved: bounded by the titles a removed agent holds
(at most the endowments of the configuration) and by the entries of a call. An observer
that rebuilds the holdings has the construction state from the query of 9.3 and applies the
events after it (6.1.3).

13.1.3. The reasons and causes are constants of a closed set exported from the root package:
the refusal reasons `no_title` and `rent_funds` (joining `no_endowment` in production's
list), and the cause `removal` of `TitleTransferred` (the other, `consumer`, exists, INT 3.3.3). They are constants of
`event.go`, as every reason is (`observability-and-logging.md` 4.1).

13.1.4. **`Short` and "endowment exhausted" under tenure** (RES 14.1, 14.2). The fields of
`EndowmentChanged` keep their RES meaning with the tiers: `Requested` the units asked,
`Granted` the units the four tiers granted (the sum of `o_i + c_i + t_i`, times `d_i`),
`Short = Requested - Granted`, and `AgentsDenied` the requests granted 0, a request refused
`no_title` or priced out among them. Under a closed or titled endowment a request can be
short while the stock is idle (example B: 400 units short of the commons and 300 units of
closed entitlement idle), and RES's `Warn` "endowment exhausted" on `Short` above 0 would
say exhaustion of a stock that is not exhausted. The fact is carried by `RentPaid.Held` and
`Idle`, units of unused entitlement that were not drawn, so the owed rows (18.2) give the
`Warn` of RES 14.2 and OBS 7.3 row 12 the condition `Short - min(Short, Held + Idle) > 0`
for an endowment with tenure data (the same as RES's with the module off, where both are 0),
and leave the counter `economysim.endowment.short` as RES defines it.

### 13.2 Logs

New keys (`observability-and-logging.md` 3.3.2): `rent` (the rent in force, money per unit,
`int64`) and `share` (a share, basis points, `int64`); `region`, `resource`, `tick`,
`agents` and `reason` exist. At `Warn`: **"tenure refusal"** with `tick`, `region`,
`resource`, `rent`, `agents` (requests refused) and `reason` (`no_title` or `rent_funds`),
emitted for an endowment and a reason in a tick in which the count is above 0 and the
consecutive run of such ticks is 1 or a multiple of 10 (the run is saved, 14), so the first
refusal and then every tenth; its volume is bounded by regions times resources times two
(`observability-and-logging.md` 3.2.2). At `Debug`, guarded by `Enabled`, per endowment per
tick in which rent was paid: "rent charged" with `tick`, `region`, `resource`, `rent` and
`quantity` (the units let); the money is `Paid` of the event. No `Info`, no `Error`: no
state is repaired. A validation warning is returned (`spec-config.md` 7.2), not logged.

### 13.3 Metrics (proposed)

New metric attribute `economysim.tier` (three constants: `own`, `commons`, `let`).
`economysim.reason` gains `no_title` and `rent_funds`, and `economysim.cause` gains
`removal` (the closed sets of `observability-and-logging.md` 5.4.1).

| Name | Instrument | Unit | Attributes | Fed by |
|------|------------|------|------------|--------|
| `economysim.rent.paid` | Counter | `{minor_unit}` | `economysim.resource`, `economysim.region` | `RentPaid.Paid` |
| `economysim.rent.units` | Counter | `{unit}` | `economysim.resource`, `economysim.region`, `economysim.tier` | `RentPaid` (`OwnUnits`, `CommonsUnits`, `LetUnits`) |
| `economysim.rent.rate` | Gauge | `{minor_unit}` | `economysim.resource`, `economysim.region` | `RentPaid.RentAfter` and `RentScaled.To`: the adapter keeps the posted rent and the scale and records the rent in force (5.2.2); the last values are held |
| `economysim.rent.refusals` | Counter | `{request}` | `economysim.resource`, `economysim.region`, `economysim.reason` | `RentPaid` (`NoTitle`, `RentFunds`) |
| `economysim.title.transfers` | Counter | `{transfer}` | `economysim.resource`, `economysim.region`, `economysim.cause` | `TitleTransferred` |

The series for the five-region proposal of RES 18 with titles on two of its resources are 10
for each instrument with no extra attribute (`rent.paid`, `rent.rate`), 30 for `rent.units`
(three tiers), 20 for `rent.refusals` (two reasons) and 20 for `title.transfers` (two causes):
90 in all (10 + 10 + 30 + 20 + 20). A count of holders is a snapshot field (13.4) and not an event metric.

### 13.4 Snapshot

Per endowment with tenure data, in (`RegionID`, `ResourceID`) order, in the row of
`spec-observability.md` 5.3 that RES's `EndowmentRow` fills: `Holders` (the number of
titles), `PoolBP` (the pool's share), `CommonsBP`, `TopBP` (the largest share of an agent),
`Posted` (the posted rent), `ScaleBP`, `Closed`, and the two diagnostic runs
(`NoTitleRun`, `RentFundsRun`). The per-holder list is the query `Holders`; a snapshot
does not carry a list of the size of the households (N-125: every field is a function of
saved state, which these are). Field names are OBS's (N-094).

### 13.5 The counterpart rule, and cost with no observer

Every `Warn` has an event field that tells the same fact (`observability-and-logging.md`
8.1): "tenure refusal" has `RentPaid.NoTitle` and `RentFunds`. Event construction is guarded
by the world's "has observer" flag and allocates nothing otherwise; the tiers work in
scratch slices held by the world (NFR-LND-001).

### 13.6 The Observability entry (observability-and-logging.md 9)

1. *New failure path or rejected input?* A request refused for tenure is a refusal with a
   reason (`no_title`, `rent_funds`), counted in `RentPaid`, not an error and not a log per
   request; a call that cannot apply returns an error.
2. *New state transition?* A title moves (`TitleTransferred`), the posted rent moves
   (`RentPaid.RentAfter`), a scale is set (`RentScaled`); rent paid is an aggregate. No map,
   fixed-point values.
3. *New decision point?* The ration's tiers are a rule; their counts are in the event. No
   strategy decision is added.
4. *New loop?* The per-endowment tiers are `O(holders + requests log requests)`, only for an
   endowment with a title; one bounded `Warn` per endowment and reason; guarded.
5. *Changed event, metric?* No change to an existing one: additions only.
6. *New metric attribute?* `economysim.tier`, three constants; bounded.
7. *Used `Error` or `Info`?* No.
8. *Logging an error also returned?* No.
9. *Tests?* Section 8 of the rules applies; see 19.

## 14. State contributed to a checkpoint

### 14.1 What is saved

Every spec lists the state it contributes (ADR 0002). This module's, all plain data, sorted,
no maps. It is one section, tag assigned by the supervisor in the dependency order after
Resources (CHK 4.2; section 18), present only with `Modules.Tenure`:

| Part | State, in the order written | Dense or sparse |
|------|-----------------------------|-----------------|
| Holdings | for each endowment with at least one title, in (`RegionID`, `ResourceID`) order: the number of titles, then each as (holder agent ID, `ShareBP`) in ascending holder ID | dense per titled endowment; an endowment with no title is not written |
| Posted rent | for each endowment of a module with `RentStepBP` above 0 and a `Terms` row that is not closed: the posted rent `P` | dense for those endowments |
| Rent scale | the (region, resource, `BP`) of every scale that is not 10 000 | sparse, ascending |
| Runs | the two diagnostic runs of 13.4, `NoTitleRun` and `RentFundsRun`, per endowment with a non-zero run | sparse, ascending |

14.1.1. **Not saved**, because derived: the entitlements, the tiers, the supply and the
attribution of a tick, the rent in force (`P` and the scale), the key. The commons share is
`10 000` minus the saved titles. The rent of an endowment whose step is off is the
configured value and is not state.

### 14.2 Restore

The restore checks, in the order of `spec-checkpoint.md` 6.2, and reports `ErrCheckpointCorrupt`
for the first that fails: each holder ID is that of a live agent of the agent section, or
the pseudo-agent of the **endowment's** region; each share is 1 to 10 000; a holder appears
once in an endowment; the shares of an endowment sum to at most 10 000; a holder that is an
agent is not above `MaxHolderShareBP` (the configuration's); the posted rent lies within
`[floor, ceil]` of the configured rate; the runs are positive; no title is written for an
outside node. A saved title whose endowment is absent from the configuration is
`ErrCheckpointConfig`, as for any module's row. The audit that every restore runs
(`spec-checkpoint.md` 6.2.2) adds the identity of 5.8 and the sum of shares to its list.

### 14.3 Why this is the whole state

A tick's decisions are functions of the holdings, the posted rent, the scale, the stock
and the requests; the stock is RES's, the requests are recomputed, and the keys are
stateless. A world restored between ticks therefore ration, charges and steps exactly as
an uninterrupted one (FR-LND-034). The diagnostic runs are saved so that a resumed run
logs as the original (`spec-checkpoint.md` FR-CHK-043).

## 15. Worked examples

Two settings, both invented and **examples**: the core text and its vocabulary are
setting-agnostic, and the commodity, role and resource names below are the packs' data
(architecture.md 9.1). The arithmetic is checked in `models/lnd/tiers.py` (`tiers.out`).

### 15.1 Example A: a farming realm (an example)

One region, `vale`, with a resource `plots` that refills (RES 16.1), a recipe `grow` that
draws 1 unit of `plots` per execution and yields 10 units of `grain` at an abundance of 10 000,
a role `freeholder` of 10 households and a role `landless` of 40. Ten households hold 5% of
the stock each; the rest of the stock is the commons; rent is 40 per let unit and does not
move; every household holds at least 40 in money, so the clip of 4.3 step 3 does not bind.

```toml
# Example: the tenure part of a pack. The resource, the roles and the recipe are the pack's.
[modules.tenure]
own_use_exempt = false

[modules.tenure.term.vale.plots]
rent = 40                      # money per unit drawn from a let unit; the solver's dual is the guide

[modules.tenure.holding.vale.plots.freeholder]
share_bp = 500                 # per household: ten households hold 5 000 of 10 000
```

With the snapshot stock `A = 100` the entitlements are 5 for each holder and 50 for the
commons. Each holder wants 3 executions; each landless household wants 2 (`w` = 3 and 2,
`d` = 1):

| Class | Households | Wants | Own | Commons | Let | Units drawn | Rent paid |
|-------|-----------|-------|-----|---------|-----|-------------|-----------|
| holder | 10 | 3 | 3 | 0 | 0 | 30 | 0 |
| landless, served by the commons | 10 | 2 | 0 | 2 | 0 | 20 | 0 |
| landless, one commons unit and one let | 20 | 2 | 0 | 1 | 1 | 40 | 40 each |
| landless, one commons unit, one unserved | 10 | 2 | 0 | 1 | 0 | 10 | 0 |

The commons (50 units) serves 40 requests of 2 (80 units): the proportional part
is 1 each, the 10 left over go to ten by the key. Of the 30 requests that remain, 20 are
served by the let supply (each holder let its 2 unused units: `L` = 20) and ten are not.
The lessors are paid `20 * 40 = 800`, 80 each. The units drawn are 100, the whole stock;
`Unserved` is 10 and `Idle` is 0, so a rent step of 500 basis points (`step = 40 * 500 /
10 000 = 2`) takes the posted rent to 42 for the next tick. Under RES 5.3 alone the 110
units asked would have been shared with holders getting 2 of 3 and every landless household
2 of 2; tenure moves ten units from the landless to the holders, which is the entitlement
and nothing else.

The cost anchor: `grow` draws 1 unit and makes 10 units of grain, so the rent adds
`1 * 40 / 10 = 4` to the unit cost of grain (5.7.1), for a holder too.

A smaller case, for the flicker of 4.2.3: fifty holders of 100 basis points each and a stock
of 80 (`A * s = 8 000` of 10 000 for each, a base of 0 and a remainder of 8 000). The
commons' base is 40; the left over is 40, and forty of the fifty holders are entitled to one
unit, a different forty each tick by the key.

### 15.2 Example B: a mining colony (an example)

One region, `ridge`, with a finite `seam` of 600 units (`constant`, rate 0), a role
`syndicate` of 4 households that hold 2 500 basis points each, a role `digger` of 20
households with a recipe `cut_seam` that draws 5 units per execution, and the endowment is
`Closed`.

```toml
# Example: an exclusive right, as a closed endowment with every unit titled.
[modules.tenure.term.ridge.seam]
closed = true

[modules.tenure.holding.ridge.seam.syndicate]
share_bp = 2500               # per household: four households hold 10 000
```

The validator reports `role_locked_out` for `digger` (11.2): the endowment is closed, every
unit is titled and the role holds none. At `A = 600` each holder is entitled to 150 and the
commons to 0. Each syndicate household wants 10 executions of 5 units, 50 units, and takes
them from its 150; the 100 left are not let, for the endowment is closed. A digger's
request is open to `A_i = 0 + 0 + 0` and is refused with `no_title` although the stock is
600.

The consumer then calls `TransferTitles{From: s1, To: NoAgent, ShareBP: 2 500, Price: 0}`, a
release of one household's share to the commons. The commons share is 2 500: the entitlement
of the commons is 150, of each of the three holders left 150. Household `s1` now holds
nothing and still works: it asks for 10 executions of 5 units (50 units) of the commons like
the twenty diggers, who want 5 executions of 5 units each, 25 units each and 500 in
all: 550 units against 150. The proportional part is 1 execution for each digger (5 units)
and 2 for `s1` (10 units), 110 units; the 40 units left over are 8 executions, which go by
the largest fraction (`s1` first, its fraction is the largest) then the key: `s1` gets 3,
seven diggers get 2 and thirteen get 1, 30 executions and 150 units in all. `Unserved` is
400 units, and the three holders' unused 100 units each (300 in all) lie idle: the
endowment is closed (`tiers.out`, `worked_b`). The `Warn` "endowment exhausted" of RES
sees 100 units of shortage (400 short less the 300 held), which is the demand of 700 units
against a stock of 600, and not 400 (13.1.4). A second call sells a digger a title of 1 000 basis
points for 300 (`Price`), and the digger pays 300 to the seller by `title_sale`: its
entitlement is 60 units, enough for the 5 executions it wants, and its unused 35 units are
not let.

## 16. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9; fields as its section 3. The
register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn` numbers (the next free are L143,
D99 and S91 at the time of the written specs). The entries here are `LND-An`, the dampers
`LND-Dn` and the scenarios `LND-Sn`, so that specs written at once do not collide. **Status
is `model` where section 17 supports it and `hand` otherwise; nothing here claims stability.**

### 16.1 Dampers this spec adds or specifies

| ID | Damper | Home | Status |
|----|--------|------|--------|
| LND-D1 | Holder share cap: no agent holds more than `MaxHolderShareBP` of an endowment, by construction and by every transfer; the pool is exempt. Cost: a legitimate large holding is refused | LND 6.1.2, 6.3.4 | specified (data; default off, 10 000) |
| LND-D2 | Rent band and bounded step: the posted rent moves by at most `step` a tick inside `[floor, ceil]` of the configured rate. Cost: the band can sit below the clearing rent and leave demand unserved | LND 5.5 | specified (data; default off, `RentStepBP` 0) |
| LND-D3 | The commons before the let: a free unit is taken before a paid one, so rent is charged only at the margin where the free stock is spent. A rule constant | LND 4.3.1 | specified |
| LND-D4 | No debt: a start that cannot be paid in full does not begin, and a rent is never carried. A rule constant | LND 5.4 | specified |
| LND-D5 | Rent in the choice of a recipe and in the cost anchor: the rent in force of a drawn unit is subtracted in the default strategy's `cost_r` (AGT 7.1.4, an edit owed) and added to the anchor of PRO 9.1, which damps open-access draw-down (RES-A5, STB-L90) where tenure has a positive rent. Cost: a recipe that draws looks dearer to holders who pay nothing; a source recipe is still eligible, so the damper does not stop a household that must work to eat (a role whose only drawing recipe is a source has no alternative to move to) | LND 4.6.2, 5.7 | specified (a rule when the module is on, with the owed AGT edit) |
| LND-D6 | `OwnUseExempt`: the subsistence floor survives a closed tenure. Cost: lessors are not paid for those units | LND 5.3.4, 7.1 | specified (data; default off) |

### 16.2 Loops, hazards and amplifiers

**LND-A1. The rent step against a delayed response.** *Kind:* loop. *Loop:* units unserved
(+) the posted rent (+) the rent in `cost_r` of the choice of a recipe (AGT 7.1.4, owed) and so
the net value of the drawing recipe (-) the units asked of the let tier (+) units unserved. One negative link, so
balancing (B) with a delay. *Delay:* one tick (P1.2 reads the rent set at P8.3a), the job
`Duration` of the recipe, the belief step of the output's price and the review of the
occupation (`spec-spawner.md` 6). *Owner:* LND 5.5; AGT 7.1.4 (the edit owed in 18.2: without it the choice is rent-blind
and the link marked (-) does not exist); PRO 9.1. *Damper:* LND-D2 (a bounded step inside a
band), D5 (the signal itself). *Undamped:* a band below the
market-clearing rent pins the rent at its ceiling and leaves demand unserved, which is a
cap on discovery and not a cycle; a role whose only drawing recipe is a source (the usual
form of a gathering role), for which the link marked (-) does not act; a positive rent with a slack commons overstates the cost
anchor (5.7.4). *Condition:* in the model the half-span of the limit cycle about the clearing rent is about
`3 * step * (delay + 1) / 4` (a rule of thumb within a factor of 1.5, overstating a cycle that lands on the clearing rent in one step: `rent.out`, last table), and a
step of 1 000 basis points (4 on a rate of 40) with a delay of 3 reaches the ceiling of 80; the
cycle is bounded inside the band when that half-span is below the distance from the clearing
rent to the nearer of floor and ceiling (10 in the model), not below the width of the band. *Test:* LND-S2. *Status:* model (`rent.out`): demand is a step function
of the rent seen `delay` ticks ago, no recipe choice, no price; it does not show the
library.

**LND-A2. The priced-out tenant.** *Kind:* loop. *Loop:* a tenant's cash (+) the executions
whose rent it can pay (+) the jobs started and so its income (+) its cash; with a refusal
for `rent_funds` the income falls, the cash falls and the next refusal follows: two
negative links in the chain refusal to income to cash, so reinforcing (R). *Delay:* the job
`Duration` and the welfare window `T` (`spec-welfare-and-wages.md` 2.3). *Owner:* LND 4.3 step 3, 5.4;
EST 2.6; SPN 5. *Damper:* LND-D3 (the commons first: a household served by the free tier
pays nothing), D4 (no debt: the loss is bounded to one start), D6, relief of the pool
(`PoolRelease`), the failure ladder. *Undamped:* a household that holds nothing, draws only
let units and has no relief. *Condition:* the commons or relief covers the cash a start
needs; the warning `role_locked_out` and the counters `RentFunds` and `PricedOut` are its
static and dynamic forms (a tenant with no cash is clipped out of the let ration before it, so
it takes no unit from a tenant that can pay, and it does not lower the rent, 5.5.1). *Test:* LND-S3. *Status:* hand.

**LND-A3. Rent and the target-income rule.** *Kind:* case of L14. A tenant's income is net
of rent (5.7.3), so the target-income rule of `spec-welfare-and-wages.md` 5 works against a
fixed outgoing: a higher rent lowers the ratio, raises the days it works to reach its
target (unit-elastic labour supply, N-028), raises the output of the staple, lowers its price
and the income that the rent was paid from. For a staple the demand elasticity is below 1
(`spec-agents.md` 11.6), so the loop is reinforcing (R) as L14 is. *Owner:* LND 5.7; WEL 5.
*Damper:* the work floor `MinWorkShare` and the rest band (D13, D26, D28), the production
halt (D24), LND-D2. *Undamped:* the elasticity. *Condition:* as L14. *Test:* S04 with a rent
(LND-S3). *Status:* hand. Merges into L14 as a case.

**LND-A4. Erosion of private titles into the pool.** *Kind:* bias with no loop. A removed
holder's titles go to the pool and nothing returns them, and births are landless, so the
pool's share rises towards 10 000 at the rate of removals: `1 - (1 - q)^n`; half in 46 years
at 1.5% a year (`erosion.out`). *Owner:* LND 6.4. *Damper:* a consumer's call (6.3); none in
the engine. *Undamped:* by design. *Condition:* the run is short against `ln 2 / q`, or the
consumer re-grants. *Test:* LND-S4. *Status:* model (`erosion.py`: a rate and births that are
landless, not a population).

**LND-A5. Concentration through transfers.** *Kind:* hazard of the seam. *Loop:* a holder's
rent income (+) its purchasing power for titles (+) its share (+) its rent income. No buyer
exists in version 1, so the loop is closed only by a consumer's calls or by SAV's decision
and CRD's credit (a loan funds the money of the call; no title is pledged in version 1, `spec-credit.md` 3.5); it is stated here because the module offers the transfer that closes it.
*Owner:* LND 6.2, 6.3. *Damper:* LND-D1. *Undamped:* a consumer that sells every title to one
agent with the cap off. *Condition:* the cap is below the share at which one holder can
set the let supply (above half of the stock). *Test:* LND-S5. *Status:* hand.

**LND-A6. Enclosure and flight.** *Kind:* hazard through other loops. A tenure that closes
the commons to a role moves its households from the own-use floor to unfed (L104) and the
ladder, and so to migration (L68, L76): the pressure of the commons loop is moved to the
regions they can reach, whose commons are open access and whose draw-down (L90) it joins.
MIG's estimate does not read rent or title (7.2.2), so the move is to a destination that
looks as it did. *Owner:* LND 7.1, 7.2; SUB 9.4; MIG 5. *Damper:* D53 (the ration at the
destination), the migration caps (D72 to D78), LND-D6. *Undamped:* a destination whose
commons is as thin. *Condition:* the destination's commons regenerates the draw of the
arrivals. *Test:* LND-S3 with MIG. *Status:* hand.

**LND-A7. Rent drained into the pool.** *Kind:* amplifier. Rent paid to the pool moves money
from households to the pool sector; the pool returns it at `PoolRelease` and through
entrants, so with a low release the pool hoards it and household money per household falls
(`spec-money-supply.md` 8.5, STB-L31, L45). *Owner:* LND 7.6; EST 2.6. *Damper:* D42 style
holding charge on the pool is MON's; `PoolRelease`; the warning `rent_hoarded`. *Undamped:*
`PoolRelease` 0. *Condition:* the pool's release matches the rent it collects. *Test:*
LND-S4 with the pool as the sole holder. *Status:* hand. Merges into L31 as a case.

**LND-A8. A small share is a flicker.** *Kind:* bias. An entitlement is a whole number of
units by the largest remainder with a key that changes every tick (4.2), so a holder whose
share of the stock is below one unit is entitled to 0 or 1 by the key: exact in expectation,
not in a tick. Its protection against the commons is statistical. *Owner:* LND 4.2.3.
*Damper:* the key (a different holder each tick); the warning `entitlement_below_unit`.
*Undamped:* a stock smaller than the number of holders. *Condition:* the stock times the
smallest share is at least 10 000. *Test:* LND-S1. *Status:* model (`tiers.out`, the second
case).

### 16.3 Delays near neighbours (checklist 9.6)

The rent step reads the tick's counters and acts at the next P1.2, a delay of one tick; the
response passes through a recipe choice made when a household is free (a job's `Duration`)
and through the belief step of the output's price (`BeliefStep`, a few ticks). The
regeneration timescale of a refilling stock is a year (RES 19.3) and the welfare window is
`T` (30 ticks by default): the posted rent is far faster than the stock and nearer the
window, so a rent that moves every tick and a ratio that is read over 30 ticks are two
rates that do not match, and the window is what the households' response reads. A pack whose
`Duration` is near the window should set the step smaller.

### 16.4 Scenarios proposed

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| LND-S1 | A pack with titles on a refilling endowment, the apportionment and the tiers, over a range of seeds, with a mix of weight 1 and cohorts, with `WithAudit` | the units drawn never exceed the snapshot stock; the grants never exceed the requests; the entitlements sum to the stock; the money paid equals the rent times the let units; the sum of shares is at most 10 000 at every tick; a world with no title equals RES exactly | A8, D3 |
| LND-S2 | A binding endowment with a fixed demand and a posted rent with a step, a yield shock halfway | the posted rent stays in its band; the cycle amplitude is within a stated multiple of the step; `Unserved` falls to 0 or the rent is at its ceiling | A1, D2 |
| LND-S3 | The enclosure case of 15.2 with the subsistence module on, relief on and off, `OwnUseExempt` off and on, and with migration on | the refusals `no_title` and `rent_funds` are counted and warned boundedly; `Blocked` rises with a closed tenure and falls with the exemption; the households that cannot pay are in the ladder; the identity of RES 6.4 holds | A2, A3, A6, D6 |
| LND-S4 | The population module on, a long run with deaths and no re-grant, the pool as a holder, relief on and off | the pool's share of titles follows `1 - (1 - q)^n` within a band; the sum of shares stays at most 10 000; the rent reaches the pool; the pool's balance rises without a release and is bounded with one | A4, A7 |
| LND-S5 | A consumer's calls that move every title to one agent, with the cap off and on | with the cap the calls past it are refused with `ErrTitleCap` and the largest share stays at the cap; without it the agent holds all and the let supply is its own; money is conserved by `title_sale` | A5, D1 |

## 17. What was run

Three small Python models in `doc/specs/models/lnd/`, run only in Docker with
`PYTHONDONTWRITEBYTECODE=1` (README there). They model the **rules** of this spec in
integers, with an assumed economy around them. They are not runs of the library and show no
default to be stable (N-015, N-027).

### 17.1 The tiers (`tiers.py`, `tiers.out`)

The entitlement by the largest remainder, the four tiers with the ration of RES 5.3.2, the
lessor supply and the attribution of 4.4 on the units taken, over 20 000 random scenarios (0
to 6 holders with shares summing below 10 000, 0 to 12 requests of weight 1 to 5 and draw 1
to 3, a stock of 0 to 300, closed and open, a rent of 0 to 50), each run twice: with every
principal able to pay, and with a balance of 0 to 60 each (the let demand clipped, the starts
cut by the money left). In every scenario: the units drawn do not exceed the stock; every
grant is between 0 and the request; the entitlements sum to the stock; no unused entitlement
is negative; the rent paid equals the rent times the let units taken; no principal pays more
than its balance; each lessor's supply is at most its unused entitlement and the supplies sum
to the units taken; when units are unserved, an idle let unit is a cut unit or a fragment smaller than one execution;
a closed endowment lets none; and the result does not depend on how the requests are listed.
The worked example of 15.1, the second case of 4.2.2, example B before and after the release
(15.2, `worked_b`) and a case of ten tenants with no cash against ten let units (priced out,
not unserved) are in the same output. *What it does not show:* the tiers with more than one
draw per recipe (the grants are independent, 4.3.3), a principal with several draws, a cohort
that changes weight, and any economy.

### 17.2 The rent step (`rent.py`, `rent.out`)

Sixty tenants with a value per unit spread from 10 to 100, a let supply of 20 units, a
posted rent of 40, a step of 0 to 2 500 basis points, a response delayed 0 to 8 ticks, a
band of 5 000 to 20 000 basis points. The market-clearing rent is 70. With no step the rent
stays at 40 and 20 units of demand are unserved; with a step it converges to 70 with a delay of 0 and
cycles with amplitude growing with the step and the delay (a step of 250 and a delay of 8
cycles in 63 to 78; a step of 1 000 with a delay of 3 hits the ceiling of 80); a band of 8 000
to 12 000 pins the rent at 48, below the clearing rent, with demand unserved. The last table
of `rent.out` sets the observed half-span of the cycle about 70 against
`3 * step * (delay + 1) / 4`: the cycle reaches the nearer bound (80) in exactly the rows where
the estimate is 12 or more, and in none where it is 7 or less. *What it does
not show:* any response of a recipe choice, a price or an entry; the demand is a function of
the rent seen with a delay.

### 17.3 The erosion (`erosion.py`, `erosion.out`)

Five hundred titles, an annual removal rate of 1.5% of the holders, every removal replaced by
a landless birth, nothing returned: the mean over 40 seeds of the titles still held, against
the closed form `N e^{-qt}` (428.7 against 430.4 at ten years, 110.2 against 111.6 at a
hundred) and a half-life of 46.2 years. *What it does not show:* a population, a spawner or
the estate's use of the titles.

### 17.4 What was not run

The library has no code; nothing here is a run of it. The cost anchor with rent, the choice
of a recipe by net value under rent, the interaction with the ladder and with migration, and
the circulation of the rent through the pool are the scenarios of 16.4, which wait for a
first slice. The rise-and-fall rule of 5.5.1 on the tier output (the units unserved, idle and cut
together) was not run either: `tiers.py` returns the counters and `rent.py` has no cut;
whether the rule can flip-flop when one unit cut for rent lowers the rent and un-cuts it
is derived by hand only and is a case for LND-S2.

## 18. Notes settled, and the edits this spec owes

### 18.1 Notes

| Note | Resolution |
|------|-----------|
| N-046 | Applied for LND: LND-A1 to LND-A8, dampers LND-D1 to LND-D6 and scenarios LND-S1 to LND-S5 (16). Remove LND from the areas; SAV, CRD, RSK, FIS and PLC stay (STO, EMP and SET have theirs) |
| N-104 | Applied for LND: `Terms` and `Holdings` are region-keyed tables of the one mechanism, with replace modes, one row per key, the errors of `spec-regions.md` 5.3 (3.3.1); two rows join the list of 5.1.3. Remove LND from the areas; CAP stays |
| N-108 | Applied for LND: the three hooks of RES 10.2 are the refusal (4.5), the entitlement (4.2, 4.3) and the rent (5). The commons of SUB (7.1) and MIG (7.2) are answered. CAP and EXT stay |
| N-107 | Applied: rent is not a grant or a levy and so adds two built-in internal reasons, `rent` and `title_sale`; no faucet and no sink (5.1, 5.8). A tax on rent is a MON levy or FIS's |
| N-090 | Applied: this module registers no `region_bound` refusal, with the reason (6.6). The rule stands for RSK, CAP and ORG |
| N-091 | Applied: a slot `SetRentScales` and an adjustment `TransferTitles` for INT (9) |
| N-016 | The rows of 10 are owed to the parameter-owner table; no unnamed parameter; the warning tolerance `RentDualTolBP` is a rule constant (5.6.2) |
| N-017 | Terms fixed in 21.2 |
| N-027 | Every number is a placeholder or structural; no default is called stable |
| N-045 | The silent dampers of this spec (the closure, the cut for rent, the band) are counted in `RentPaid` (13.1.1) |
| N-103 | Followed: the spec and its models only, the edits handed over |
| N-122 | Respected: no first-come cap; every tie of a tier goes by the stateless key (WLD 10.1.1) |
| N-125 | Every snapshot field of 13.4 is a function of saved state (14) |
| N-126 | The rows owed to the event and `Warn` tables are in 18.2 |
| N-128 | A title is saved once, in the module's section (6.5.1, 14) |
| N-054 | Rent moves money between households and the pool and creates none, so the mean welfare ratio of a closed economy is unchanged by it; the ratio of a holder and of a tenant differ (7.5) |

New notes to add (the supervisor numbers them): (a) **a decision to acquire a title is
SAV's, with CRD's credit (a loan funds the money of the call; no title is pledged in version 1, `spec-credit.md` 3.5), and calls `TransferTitles`; it needs a field in the plan of a
`Strategy`** (areas SAV, CRD, AGT); (b) **MIG's estimate of a destination's income does not
read rent or title; a rumoured rent would be a second kind of rumour** (areas MIG, INF); (c)
**the steady-state programme has one row per endowment; the split of that row among roles and
the money circuit of a rent are not in it** (areas VIA, CLB).

### 18.2 Edits owed to other specs

One consolidated table, in the order of the files. The supervisor applies them; this spec
edits none (the process of N-103).

| File | Section | Exact change |
|------|---------|--------------|
| `spec-world.md` | 5.1 table, step 8 | add a step 8a after the agents are created and before the warm start: "the titles of `Modules.Tenure`, `spec-land.md` 6.1; no draw" |
| `spec-world.md` | 6.4 rows P2.3, P2.6, P2.7 | P2.3: "and, for an endowment with a title, the entitlements (LND 4.2)"; P2.6: "the four tiers: own, commons, let (LND 4.3), each by RES 5.3.2; reads the principals' balances (the clip of LND 4.3 step 3; no reservation is open in P2, WLD 6.1.6)"; P2.7: "the largest `k` whose rent the principal can pay and the transfers of rent (LND 5.3, 5.4)" |
| `spec-world.md` | 6.10 | a row **P8.3a**: "the rent step of every endowment with a step, then `RentPaid` per endowment in (`RegionID`, `ResourceID`) order; owner LND 5.5; reads the counters of the tick; draws none" |
| `spec-world.md` | 6.11 items 8, 9 | item 8 (one removal): `TitleTransferred` per title, ascending (region, resource), after `ContractSettled` and before `AgentBankrupted`; item 9: `BoundaryFlow`, `EndowmentChanged`, `RentPaid`, then CAP's P8.3b events, `TickEnded` |
| `spec-world.md` | 9.4.1 | a carve-out sentence: a quantity local to the entity a price belongs to (the units unserved or idle at one endowment, as BEL's unmatched units at one market) is not an aggregate in the sense of this rule; the step of LND 5.5 reads only that and is off by default (see also SET's ask) |
| `spec-world.md` | 7.5 | add "the tiers of tenure, the apportionment and the rent step (LND 8.2)" to what does not draw |
| `spec-world.md` | 8.2 table | a row "Rent scale: by (region, resource), 0 to `MaxShockBP`, default 10 000; mechanism LND 5.2; read at P1.2, P2.3, P2.6, P2.7; `RentScaled`; saved by LND 14", and the adjustment `TransferTitles` beside `AdjustEndowments` (at the call; `TitleTransferred`) |
| `spec-world.md` | 9.2 table | the row of 8.3 above |
| `spec-resources.md` | 5.2 step 3, 5.3, 5.5.1, 10.2, 14.1, 14.2, 23 (1) | 14.2 and FR-RES-028: for an endowment with tenure data the `Warn` "endowment exhausted" fires on `Short - min(Short, Held + Idle) > 0` (LND 13.1.4; unchanged with the module off); 5.2 step 3 and 5.3: "with the module of tenure on the ration is the four tiers of `spec-land.md` 4.3, each by this rule"; 5.5.1: the view fields of LND 4.6 and the reason `no_title` of `MaxExecutions`; 10.2: "the three hooks are used by `spec-land.md` 4.5, 4.2 and 5"; 14.1: `Granted` is the sum of the tiers; 23 (1): a pointer to LND 20 (1) |
| `spec-production.md` | 4.2 | the reasons: `no_title` and `rent_funds` after `no_endowment` and before CAP's build reasons and `no_room`, in the master order of 4.2 (PRO owns it) |
| `spec-production.md` | 9.1 | add `rent_cost = sum over Draws of d * r` (LND 5.7.1) to `cost`; replace "Endowment draws are not priced, ..." by "A draw is priced at the posted rent of `spec-land.md` 5.2.2 when the tenure module is on, and is not priced otherwise" |
| `spec-production.md` | 14.2 | the two reasons join the refusal table |
| `spec-agents.md` | 3.6, 6.4, 6.5.1, 7.1.4 | 3.6: a step 2b "each title of the agent moves to the pool of the title's region, `spec-land.md` 6.4"; 6.4: the fields of LND 4.6; 6.5.1: the two reasons; **7.1.4: `cost_r` gains `sum over the recipe's Draws of d * Posted` (the rent in force of the view, LND 4.6.2), 0 when the module is off or the rent is 0; a source recipe stays eligible at `net_r <= 0`; FR-AGT-048 and the `UnitCost` text are unchanged (the anchor is the belief update's)** |
| `spec-ledger.md` | 7.2 | the two rows of LND 5.8.1 |
| `spec-welfare-and-wages.md` | 2.1, 4.3 | 2.1: `income_t` gains the rent received as an unearned receipt (as `G_t`) and nets the rent paid as `S_t(Input)`; 4.3: the observed wage excludes rent received |
| `spec-demand.md` | 10.2 | `class_without_income` is not raised for a role that holds a title (a holding row names it) |
| `spec-subsistence.md` | 5.5 item 6, 9.5, 14.1 | 5.5 and 14.1: `no_title` and `rent_funds` join `no_endowment` as reasons for `Blocked`; 9.5: "applied by `spec-land.md` 7.1, with `OwnUseExempt`" |
| `spec-migration.md` | 6.3 (a sentence) | the estimate reads no rent and no title (`spec-land.md` 7.2.2) |
| `spec-employment.md` | FR-EMP-023, 5.5.1, 6.1.3, `LabourRound` fields | FR-EMP-023: the order `no_endowment`, `no_title`, `rent_funds`, `no_room`, with `rent_funds` at formation a **bound** (the principal cannot pay even the least rent, LND 4.7.2), 0 when its own entitlement covers the draw; 5.5.1: the cuts after formation are the ration, the room and **the rent** at the start; 6.1.3: "A ration, room or rent cut": `k = min(m, grant, live inputs, k_room, k_rent)`, the refund and `not_started` follow; `LabourRound` gains `NoTitle` and `RentFunds` |
| `spec-storage.md` | 4.2 reasons list, FR-STO-011 | `no_room` comes after `no_title` and `rent_funds` |
| `spec-regions.md` | 5.1.3, 6.5.2 | 5.1.3: two rows (tenure terms, tenure holdings; both replace); 6.5.2: the list of refusals records that LND registers none |
| `spec-integration.md` | 3.6 first row, 5.1, 5.1.3, 5.3, 6.2, 7.1, 3.4.3 | 3.6: one sentence on the row "An act-now operation": `TransferTitles` with a `Price` moves money between two agents at the call and is not an act-now trade of the book (it takes no order, clears nothing and cannot change the order of a clearing; a title sale is outside the book, and the reasons of 3.6 do not apply); and a row `SetRentScales` (the type `EndowmentScale`, reused); its check row (`ErrModuleOff`, `ErrUnknownRegion`, `ErrUnknownResource`, `ErrOutsideNode`, `ErrOutOfBounds`, `ErrDuplicateID`); `SlotState.RentScales`; the queries `Titles`, `Holders`, `Rent`; the sentinels of 9.4; the adjustment `TransferTitles` in a new 5.4 with its checks of 6.3.4; 3.4.3: `Titles` accepts the estate pseudo-agent of a region |
| `spec-checkpoint.md` | 4.2, 4.2.3, 4.2.5, 4.3, 6.2 | 4.3: a row LND: the entitlements, the tiers, the supplies and attribution, the rent in force and the key, the commons share (10 000 less the titles), the scratch of the reservation (FR-CHK-045); a section row for Tenure in the module order after Resources (the tag is the supervisor's); 4.2.3: a title's holder is a live agent or the endowment region's estate; 4.2.5: the sparse parts of 14.1; 6.2: the checks of 14.2 |
| `spec-observability.md` | 3.3.9, 4.1, 4.3.3, 4.4.8, 4.5, 5.3, 6, 7.3, 7.6 | the three events with their class, position and gate; **4.3.3: an item 8 for rent and for the price of a title (LND 13.1.1a): at a `RentPaid`, households of region `g` lose `From[g]` and gain `To[g]` and the pool of `Region` gains `ToPool`; at a `TitleTransferred` with `Price` above 0 the payer's side of `ToRegion` loses it and the payee's side of `FromRegion` gains it, the pool's side when `ToPool` or `FromPool`; FR-OBS-041 unchanged**; 4.4.8: the removal order, `TitleTransferred` per title before `AgentBankrupted`, after `ContractSettled`; 4.5: the order of `RentPaid` after `EndowmentChanged`; the snapshot fields of 13.4; the metrics and attribute values of 13.3; the `Warn` row "tenure refusal" with its counterpart; **7.3 row 12: "endowment exhausted" with tenure data is `Short - min(Short, Held + Idle) > 0` (LND 13.1.4)**; 7.6: the keys `rent`, `share` and the values `removal` (`economysim.cause`), `no_title`, `rent_funds` (`economysim.reason`) |
| `rules/observability-and-logging.md` | 3.3.2, 4.3 (the catalogue and the money-rebuild paragraph: the fields of LND 13.1.1a), 5.3, 5.3.1, 5.4.1 | keys `rent`, `share`; the three events; the five instruments and their feeders; the attribute `economysim.tier`, and the constants `no_title`, `rent_funds` (reasons) and `removal` (a cause; `consumer` exists); a fresh read-receipt tag |
| `spec-config.md` | 3.2, 3.7, 4.1, 6.2, 7.3, 7.4, 8.1 | the module in 3.2 and the fields in 3.7; bounds `MaxRentStepBP`, `MaxRentCeilingBP`, `MaxHoldings`, `MaxTitleTransfers`; the keys of 3.3.2 here; the sentinels `ErrTitleShares`, `ErrTitleCap`, `ErrNoTitle`, `ErrNotAHolder`; the rules and warnings of 11; the terms and holdings in the hash (`Label`-like fields none) |
| `spec-viability.md` | 10.2, 10.3 | 10.2: a row for the warning `rent_off_dual` (LND 5.6.2: a configured rent of an endowment that is more than `RentDualTolBP` from the dual `Rent` of its row, or above 0 where the row is slack; never an error); 10.3: `Headroom` and `Rent` are the guide of LND 5.6; the programme is not edited for tenure |
| `spec-stability.md` | 4, 5, 8; 4.8 | the entries, dampers and scenarios of 16 with their final numbers; remove the hook row LND (4.8, line "LND: rent") |
| `spec-calibration.md` | 6 | registry rows: `RentStepBP`, `RentFloorBP`, `RentCeilingBP` (damper strengths of LND-D2 on LND-A1), `MaxHolderShareBP` (damper strength of LND-D1 on LND-A5), `OwnUseExempt` (structural choice), `Terms.Rent` (knob; the dual is its guide), `Terms.Closed` (structural), `Holdings.ShareBP` (knob) |
| `spec-indicators.md` | presets | asks: `RentShare` (rent paid over value added of the drawing recipes), `LandlessShare` (households with no title in a region that draws), `TitleConcentration` (the share of the largest holder and a Gini of shares, from the snapshot rows and `TitleTransferred`) |
| `spec-reference-chain.md` | overlays | an `overlay/tenure.toml` for LND-S1 to LND-S5 (a titled refill endowment on `one` and `five`; the closed case of 15.2) |
| `README.md` (specs) | index, code table, owner table, glossary | the index row; `LND` exists with FR-LND-001 to 046 and NFR-LND-001 to 004; the rows of section 10; the terms of 21.2 |
| `spec-roadmap.md`, `documentation_index.md` | LND rows | LND `exists`; the files `spec-land.md` and `models/lnd/` |
| `models/README.md` | Contents table | a row for `lnd/`: `spec-land.md` sections 4, 5.5, 6.4, 15, 16 and 17; `tiers.py`, `rent.py`, `erosion.py` with outputs |
| `notes.md` | N-046, N-104, N-108, N-107, N-090, N-091 and the new notes | updates and the three new notes of 18.1 |
| `AGENTS.md`, `architecture.md` 2.6 | layout | `tenure.go` and `internal/tenure/` at the first code (N-019) |

## 19. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-LND-001 | The module MUST be on if and only if `Config.Modules.Tenure` is non-nil; with it off no title, rent, event, log, ledger reason or state of this spec MUST exist; with it on and `Modules.Resources` nil the configuration MUST be rejected with `ErrModuleDependency` (2, 3.3). |
| FR-LND-002 | A title MUST be a share of 1 to 10 000 basis points of the snapshot stock of one endowment held by one holder, a holder MUST hold at most one title per endowment, and the shares of an endowment MUST sum to at most 10 000 at every observable point (3.1). |
| FR-LND-003 | A holder MUST be an agent that is not a pseudo-agent, or the estate pseudo-agent of the endowment's region; any other agent MUST be refused with `ErrNotAHolder`, and a row naming an outside node's region with `ErrOutsideNode` (3.2, 11.1). |
| FR-LND-004 | `Terms` and `Holdings` MUST be region-keyed tables of the mechanism of `spec-regions.md` 5, with one row per key; `Rent` and `Closed` MUST be replace; absence MUST be open access without a warning; an unknown region, resource or role MUST be rejected with `ErrUnknownRegion`, `ErrUnknownResource` or `ErrUnknownRole` and a second row for one key with `ErrDuplicateID` (3.3.1). |
| FR-LND-005 | The titles of the configuration MUST be created after the agents exist and before the warm start, in the order of the holdings, with a role holder giving each household agent of the role in the region a share of `w * ShareBP`, and construction MUST emit no event, take no draw and charge no rent (6.1). |
| FR-LND-006 | An endowment with no title MUST be rationed by the rule of RES 5.3 exactly, and a world with the module on and no title MUST give the same results, the same events other than `WorldCreated` (whose `ConfigHash` differs), and the same random stream as the same world with the module off, its snapshot differing only by the tenure rows of 13.4 (4.1). |
| FR-LND-007 | The entitlements MUST be the apportionment of the snapshot stock by the shares of the holders and the commons share by the largest remainder, ties by the key of RES 5.3.3 then by agent ID with the commons last, and MUST sum to the stock exactly (4.2.1). |
| FR-LND-008 | A holder's entitlement MUST first serve the requests whose principal is that holder, by the ration of RES 5.3.2, and what it leaves MUST be its unused entitlement (4.3 step 1). |
| FR-LND-009 | The commons entitlement MUST serve the residual demands before the let tier, and a unit of the commons MUST be free (4.3 steps 2, 4.3.1). |
| FR-LND-010 | The let supply MUST be the sum of the unused entitlements and MUST be 0 for a `Closed` endowment; the demand of a request for the let tier MUST be clipped to the executions its principal can pay at the rent in force, the units clipped being counted as `PricedOut`; a unit of the let tier MUST be the only unit for which rent is charged (4.3 step 3, 5.3). |
| FR-LND-011 | The grant of a request MUST NOT exceed the request, the units taken in all tiers MUST NOT exceed the snapshot stock, and the grants MUST NOT depend on the order in which the requests were listed (4.3, 4.3.2). |
| FR-LND-012 | The principal of a request MUST be the account whose inputs the job uses, and the holder status, the entitlement and the rent of a hired or trained job MUST be the principal's (4.7.2, position 9). |
| FR-LND-013 | The supply of each lessor MUST be the apportionment of the let units the starts took by the weights of the unused entitlements by the largest remainder, MUST NOT exceed the lessor's unused entitlement, and the units MUST be attributed to tenants in ascending agent ID against lessors in ascending agent ID (4.4). |
| FR-LND-014 | A request whose start is 0 MUST be refused with the first of the master order of `spec-production.md` 4.2 (its slice here: `no_endowment`, `no_title`, `rent_funds`, after `no_input` and before `no_room`, with CAP's build reasons between), with `no_title` only when the stock could serve an execution and no unit is open to the principal by rule (4.5.1); at the formation of a hire `rent_funds` MUST be refused only when the principal cannot pay the least rent the job could owe, 0 when its own entitlement covers the draw (4.7.2). |
| FR-LND-015 | The rent in force MUST be the posted rent times the scale over 10 000, truncated toward zero and at least 1 when both are positive, and a scale of 0 MUST make it 0 (5.2.2). |
| FR-LND-016 | The rent of a job MUST be paid at its start, inside the atomic step, for the let units it draws; the start MUST use the largest number of executions whose total rent the principal's balance covers; and when that is 0 the request MUST be refused with `rent_funds`, with nothing taken and nothing carried; the rent MUST be reserved against the principal's other starts of the tick, and a payee that cannot receive the whole rent MUST receive what it can, the rest being forfeited and counted, so that no transfer of rent is refused by the ledger (5.3, 5.4). |
| FR-LND-017 | Rent MUST be a ledger transfer of the reason `rent` between non-boundary accounts, MUST create and destroy no money, MUST be in no `BoundaryFlow`, and a transfer whose payer and payee are one account MUST be skipped with the units still counted as let (5.1, 5.3.3). |
| FR-LND-018 | With `OwnUseExempt` set, a request for a recipe of the principal's effective own-use list MUST be charged no rent and its lessors MUST be paid nothing for those units (5.3.4). |
| FR-LND-019 | With `RentStepBP` above 0 the posted rent of an endowment that is not closed MUST, at P8.3a, rise by `step` bounded by the ceiling when units were unserved at the rent in force and no let unit was cut for rent, fall by `step` bounded by the floor when otherwise let units were not drawn (units cut for rent included), and MUST NOT move otherwise or with `RentStepBP` 0; units priced out MUST move it neither way (5.5.1). |
| FR-LND-020 | The cost anchor MUST add `sum of d * r` over the draws of a recipe to its cost when the module is on, for a holder's units too, and the default strategy's `cost_r` MUST add `sum of d * Posted`, with a source recipe still eligible; a rent of 0 MUST leave both as they were (5.7.1, 4.6.2). |
| FR-LND-021 | The rent a principal paid MUST be netted in `income_t` like an input, the rent a holder received MUST be an unearned receipt that is excluded from the observed wage, and a holder role MUST NOT draw `class_without_income` (5.7.3). |
| FR-LND-022 | The view MUST carry, for the agent's own region only, `Posted`, `ShareBP`, `Entitled` and `Closed` of each endowment its role draws on, and no view MUST carry another holder's share or the holdings of an endowment (4.6.1). |
| FR-LND-023 | At a removal each title of the agent MUST move to the pool of the title's region, before `AgentBankrupted`, one `TitleTransferred` with the cause `removal` each in ascending (region, resource), and no money MUST move (6.4.1). |
| FR-LND-024 | A weight change, a role change and a relocation MUST leave an agent's titles unchanged, and the module MUST register no `region_bound` refusal (6.5, 6.6). |
| FR-LND-025 | `TransferTitles` MUST validate every entry in canonical key order on a working copy and apply none if any fails, MUST move a share between two holders or between the commons and a holder, MUST move the price from `To` to `From` by the reason `title_sale` and create no money, and MUST emit one `TitleTransferred` with the cause `consumer` per entry, carrying the regions and pools of its price (6.3, 13.1.1a). |
| FR-LND-026 | `SetRentScales` MUST set a scale of 0 to `MaxShockBP` per endowment, default 10 000, atomically, one entry per key, with an event per changed key, read from the first step after the call, and MUST NOT edit a configured rent (9.1). |
| FR-LND-027 | The queries `Titles`, `Holders` and `Rent` MUST draw nothing, emit nothing, change nothing and return a deep copy of plain data in ascending order (9.3). |
| FR-LND-028 | `RentPaid` MUST be emitted once per endowment with a title per tick, after `EndowmentChanged`, in (`RegionID`, `ResourceID`) order, if and only if a tier count, `Idle`, `Held`, `Unserved`, `PricedOut`, a refusal count, `CutUnits`, `Paid` or `Forfeited` is non-zero or the posted rent changed, with the fields of 13.1 and no map, and its `From`, `To` and `ToPool` MUST rebuild the money of the household sectors and the pools by region (13.1, 13.1.1a). |
| FR-LND-029 | `TitleTransferred` and `RentScaled` MUST carry the fields of 13.1, contain no map and no pointer, and be emitted at the transition or call after it is complete (13.1). |
| FR-LND-030 | The `Warn` "tenure refusal" MUST be emitted for an endowment and a reason in a tick with a positive count and a run of 1 or a multiple of 10, at most once per endowment and reason per tick, and the run MUST be saved (13.2, 14.1). |
| FR-LND-031 | Each metric attribute of 13.3 MUST take only values present in the configuration or in its closed set, and no metric MUST carry an agent ID, a price or a tick (13.3). |
| FR-LND-032 | The snapshot MUST report per endowment with tenure data `Holders`, `PoolBP`, `CommonsBP`, `TopBP`, `Posted`, `ScaleBP`, `Closed` and the two runs, each a function of saved state (13.4). |
| FR-LND-033 | The state of 14.1 MUST be saved as plain data in a fixed order with no maps, a title MUST be saved once and not in the agent record, and the entitlements, tiers and rent in force MUST NOT be saved (14.1, 6.5.1). |
| FR-LND-034 | A world restored between ticks MUST apportion, ration, charge and step exactly as an uninterrupted one (14.3). |
| FR-LND-035 | A restore MUST reject, with `ErrCheckpointCorrupt`, a holder that is not a live agent or the endowment region's estate, a share outside 1 to 10 000, a repeated holder, shares summing above 10 000, a holder above the cap, a posted rent outside its band or a title of an outside node (14.2). |
| FR-LND-036 | The module MUST draw nothing from the random source, and the random stream of a world MUST NOT depend on whether the module is on (8.2). |
| FR-LND-037 | With `WithAudit`, the audit MUST check that the shares of every endowment sum to at most 10 000, that every holder is live, and that the money moved by rent in a tick equals the `Rent` of the tick's `RentPaid` (the rent in force at P2.7, not the posted rent after the step of P8.3a) times the units charged, less the exempt units and the forfeited money (5.8, 14.2). |
| FR-LND-038 | The errors of 11.1 MUST be reported with the sentinel and the path of the field, and `ErrTitleShares`, `ErrTitleCap`, `ErrNoTitle` and `ErrNotAHolder` MUST be defined as root sentinels (11.1, 9.4). |
| FR-LND-039 | The warnings of 11.2 MUST be reported and MUST NOT fail validation (11.2). |
| FR-LND-040 | `rent_off_dual` MUST be reported through the solver, which MUST report `Rent` per (resource, region); the programme MUST NOT be changed by a title or a rent (5.6.2, 12). |
| FR-LND-041 | With `MaxHolderShareBP` below 10 000 no agent MUST hold more than it at construction or after any transfer, the pool excepted, and a transfer that would MUST be refused with `ErrTitleCap` (6.1.2, 6.3.4). |
| FR-LND-042 | No code MUST branch on a resource or role identifier, and a pack for a new setting MUST load, validate and run with tenure and no Go code specific to it, the two examples of 15 completed with the elided commodities included (architecture.md 9.1, 9.4). |
| FR-LND-043 | Observation MUST NOT change results: a run with a logger and an observer attached MUST give the same holdings, grants, rent and stock as one without. |
| FR-LND-044 | The count of `RentPaid` events a tick MUST be bounded by regions times resources and that of `TitleTransferred` by the titles of a removal and the entries of a call (13.1). |
| FR-LND-045 | Rent MUST leave the total money supply and every sectoral identity of `spec-money-supply.md` 3 unchanged, and the boundary balance of every reason MUST be what it would be without it (5.1.1). |
| FR-LND-046 | A request refused for `no_title` or `rent_funds` MUST be counted in `RentPaid` and in the `Blocked` count of the subsistence module where it is an own-use request (7.1.2, 13.1). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-LND-001 | With the default logger and observer, the entitlements, the tiers, the charge and the step of a `Step` of a pack with titles MUST allocate nothing beyond scratch held by the world (observability-and-logging.md 4.5, 8.2). |
| NFR-LND-002 | The cost of a tick for this module SHOULD be proportional to the holders and the requests of the endowments that have a title, `O(H + R log R)` each and one sum when the tiers are not short, and not to the households or the weights. |
| NFR-LND-003 | The apportionment, the tiers, the attribution and the step MUST have property and fuzz tests (testing.md 7.4) asserting that they never panic, that the units drawn never exceed the stock, that the grants never exceed the requests, that the rent paid equals the rent times the units let, and that the result does not depend on the listing order. |
| NFR-LND-004 | The apportionment and the tiers SHOULD have benchmarks that call `b.ReportAllocs()`. |

## 20. Open questions

1. **Occupying draws.** A draw consumes; a unit held for the length of a job and then
   released is a concurrency limit, not a flow (RES open question 1). Tenure does not
   change it, and a refill with a fallow approximates it. A stock held would need a count of
   units in use per endowment and per title. Next step: the first pack that needs it, with CAP.
2. **A share of one stock, or a stock per title.** A title is a share of the snapshot of one
   stock (position 1), so a holder's draw lowers every share's units and the externality of
   open access is **partly** internalised (by the rent and the priority, and by D5), not
   removed. A parcel with its own stock, carry and regeneration would isolate holders at the
   price of the regeneration of RES per title and a split of the capacity. It is the cleaner
   model for a bounded parcel and the more expensive one. Next step: measure, with LND-S2 and RES-S2,
   whether the externality matters at the pack's holder counts.
3. **A term and security of tenure.** A tenant has no commitment, so no eviction and no
   renewal, and the "rent ratchet" of a lessor who raises rent at renewal is not
   modelled; the posted rent is one rate for everyone. A lease with a term and an
   obligation, with a `region_bound` refusal by the rule of N-090, is a second mechanism of
   the kind of the delivery contract. Next step: a pack that shows it, and ORG.
4. **`OwnUseExempt` by endowment or by role.** One boolean for the module (7.1.3). A pack
   that wants the right of the landless on one resource only models the resources apart.
   A per-term field is the form.
5. **A per-role split of the resource row.** The programme sums the draws of all roles on a
   row (12.3). A role that cannot draw under a closed tenure is not seen by it. A second
   family of rows with the entitlement of each role as a bound would make `role_locked_out`
   exact. VIA's.
6. **The warm start of the rent circuit.** A holder role's money and the pool are seeded
   from the table and the run finds the circulation (12.3). Whether the solver should add a
   rent column to the warm start is VIA's and open.
7. **Rent in a household's estimate.** MIG's estimate does not read rent or title (7.2.2).
   A rumoured rent is a second kind of rumour (INF). Next step: LND-S3 with migration shows
   whether the surprise matters.
8. **Who decides to buy, and a market for titles.** A title as a commodity traded in the
   book was considered and not taken: a title is a share of a regional stock with no mass or
   spoilage, a deed held as a durable tool could not be let (no rent), and a market that
   prices a share would need the beliefs and the unit of a commodity of which there is one
   per endowment. The decision to buy is SAV's, the credit CRD's, and the transfer is the
   call of 6.3.
9. **Re-granting from the pool.** The pool's titles are let and never given back
   (6.4.2). A rule that returns them to households of a role (a homestead) is a policy and
   is PLC's or FIS's, through the call.
10. **Holders other than agents.** A company, an association or a jurisdiction as a holder
    (position 3). ORG and POL.
11. **A share of capacity.** The entitlement moves with the stock, so a thin stock is a thin
    right (3.1.3). A title in units of capacity (`Units`, share `Units / C`) would read the
    same at a full stock and differently under a shock; the field is a presentation of the
    share and could be offered by the loader.
12. **A rent per period.** A fixed rent per tick for the use of a title, owed whether it
    is used or not, needs a term and an obligation (3 above). The rent here is a price per
    unit drawn.
13. **The leftover bias of the ration** (RES open question 4) applies to each tier: a fixed
    mix of request sizes meets the same order of fractions every tick. Next step: LND-S1 with
    a mixed mix.

## 21. Where this lives in the code

### 21.1 Homes

| Concern | Home |
|---------|------|
| `TenureConfig`, `TenureTerm`, `Holding`, `HolderRef`, `TitleTransfer`, `TitleState`, `HoldingsState`, `RegionMoney`, the constant `NoAgent AgentID = -1` (as `NoRegion RegionID = -1`) | root package, `tenure.go`; defaults in `defaults.go`; bounds in `bounds.go` |
| Validation, the sentinels `ErrTitleShares`, `ErrTitleCap`, `ErrNoTitle`, `ErrNotAHolder`, and the warnings | root package, `validate.go`, `errors.go` |
| The apportionment, the tiers, the lessor supply and the attribution, the step and the rent charge | `internal/tenure`: pure functions over primitive slices and indexes (architecture.md 2.9); the standard library, `internal/ordering` for the key and `internal/endowment` for the ration |
| The holdings, the posted rent, the scales, the diagnostic runs and the removal step | root package, `world.go` and `tenure.go` |
| The events `RentPaid`, `TitleTransferred`, `RentScaled` | root package, `event.go` |
| The log keys `rent`, `share` | `internal/logkeys` (key strings only) |
| The reasons `no_title`, `rent_funds`, `removal` | root package, `event.go` (exported constants) |
| The ledger reasons `rent`, `title_sale` | `internal/ledger` registry, declared with the module |

```go
// internal/tenure: stdlib only, no root types. A sketch, not final.

// Part is a holder (ID is the agent ID) or the commons (ID -1) with its share.
type Part struct {
	ID      int64
	ShareBP int64
}

// Apportion splits total units by the shares of parts by the largest remainder; ties
// go to the smaller key(tick, ID), then the smaller ID, the commons last. out[i] is
// the units of parts[i].
func Apportion(total int64, parts []Part, tick int64, out []int64)

// Request is one agent's demand on one endowment.
type Request struct {
	Ref        int   // index into the caller's slice, returned with the grant
	Principal  int64 // the agent whose entitlement serves it
	Executions int64
	Draw       int64
	Exempt     bool  // an own-use request when the module is exempt
	Cash       int64 // the principal's balance at P2.6 (the starts reserve against it, 5.4.1)
}

// Result holds, per request, the executions of each tier and, per lessor, the units
// supplied; and the counters of the tick.
type Result struct { /* own, commons, let by request; pays (tenant, lessor, units); counters */ }

// Tiers runs the four tiers of 4.3 (the let demand clipped by Request.Cash), the starts of
// 5.4 (rent reserved per principal) and the attribution of 4.4 on the units taken. A pure
// function of its arguments: no state, no draw.
func Tiers(stock int64, parts []Part, reqs []Request, closed bool, rent, tick int64, res *Result)

// Step returns the posted rent after one tick (5.5.1): unserved is X, idle is Y = L - V and
// cut is C.
func Step(posted, base, stepBP, floorBP, ceilBP, unserved, idle, cut int64) int64
```

### 21.2 Terms (N-017)

To add to the glossary in `specs/README.md` and `doc.go` before code uses them:
**tenure** (this module), **title** (a claim of ownership of a share of an endowment, never a rank or an honorific), **holder**, **commons share**, **entitlement**
(units of a title at a snapshot; not the entitlement of other specs), **unused entitlement**,
**let** (a unit of an unused entitlement offered to others; the verb is not "lease"),
**lessor**, **tenant** (a request served from let units), **tier** (of the ration: own,
commons, let; a different word from STY's tier of tests), **posted rent**, **rent in force**,
**rent scale**, **closed** (an endowment that lets nothing), **enclosure** and **release**
(the two directions of a transfer to and from the commons), **principal** (PRO's, used
here for the holder status of a job), **apportionment** (by the largest remainder).
**no agent** (`NoAgent`, the constant -1 of `AgentID`: the commons, as a party of `TransferTitles`; `NoRegion` of REG is the model), **priced out** (units of a request clipped by what its principal can pay at the rent in force, 4.3 step 3), **forfeited** (rent that a payee at `MaxMoney` could not receive and was not charged, 5.3.3), **held** (unused entitlement of a closed endowment).
"Land" is not a term: it appears only in the name of the file and in the examples.
