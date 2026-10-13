# Spec: Fiscal: taxes, duties, tolls, the treasury's payments and its standing orders

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-FIS-NNN` and
`NFR-FIS-NNN`, which tests cite (testing.md section 4). They are in section 23.

Layer: module, later stage ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)
decision 2, [ADR 0003](../adr/0003-version-1-scope.md)). It is off unless
`Config.Modules.Fiscal` is present, and the core does not depend on it. A world with
the module off is exactly the world without it (FR-FIS-001). It needs a polity with at
least one **governed** jurisdiction, because the account every fiscal flow pays into or
out of is the treasury of `spec-polity.md` (5). Needs `spec-polity.md` (3.5, 4, 5, 6;
the treasury, the chain, the crossing), `spec-money-supply.md` (4, 5, 6, 10: the
arithmetic of a carry, the declared reason, the flow scale, "no faucet reads an
aggregate"), `spec-ledger.md` (3, 7, 9: the account kinds, the reason registry, the
sector identity), `spec-world.md` (6, 7, 8, 10: the positions, the stream, the slots, the
rule for a new tie), `spec-clearing.md` (3, 4: admission and the one-order rule),
`spec-transport.md` (3.2, 5.4, 5.5: the carrying unit and the dispatch) and
`spec-external-sector.md` (4: the counterparty). It is read by the specs after it:
policy (PLC), institutions (ORG), and, for a holder of land and stores, land (LND) and
storage (STO).

**Every number in this spec is a placeholder** except where a derivation is shown.
Nothing has been run against the library, bazaarBot has not been run, and no
acceptance band exists (notes N-015, N-027). The spec sets no default macro constant:
a module with no flows does nothing, and every rate, amount and limit is written by the
pack. The models of section 20 are small integer or exact-rational models of the rules
of this spec in Docker Python; they are not claims about the library, and no default is
claimed to be stable.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017). The
setting words of the examples (`crown`, `inland`, `kingdom`, `province`, `cloth`,
`grain`, `west`, `east`, `poll`, `ferry`) appear only in the
**flagged examples** of section 18 and in sentences marked "for example"; they are
data, never branched on (architecture.md 9.1; `AGENTS.md`, setting-agnostic). The word
**levy** is MON's (a transfer from a class to a boundary: a sink); a transfer from a
class to a treasury is a **tax** here, and the two are never mixed (section 4).

## 1. Purpose and scope

`spec-polity.md` gave a jurisdiction an account that holds money and an agent ID that
other specs can name, and left every use of it to this spec (5.4.2, hooks H1 to H4 and
H12 of its 6.3). `spec-money-supply.md` gave a world the named faucets and sinks and
refused to tax a trade or to model a treasury (10.5), and four finished specs say "a
tax is FIS's": STO (a duty on storage, a buffer store and its release), LND (a tax on
rent or on titles), CAP (a toll, a levy on use), EXT (a tariff at a gate). This spec
decides what a fiscal flow is, who pays whom and when, and what the treasury may do
with what it holds. It covers:

- the **fiscal flow**: six kinds, each a typed list of the configuration, with an
  **authority** (a governed jurisdiction), a window in ticks and seasons, and a
  declared internal reason equal to its ID (section 3);
- the treasury as the counterparty, the regions a transfer is attributed to and the
  reconciling item that keeps the regional money identity exact (section 4);
- **taxes** on a class of households, a pool or a subordinate treasury: a fixed amount or
  a rate on the balance above an exemption (section 5);
- **duties** on trades: a rate on the value of a trade, by commodity and by scope
  (domestic, import, export), including a tariff at a gate (section 6);
- **tolls** on the crossing of a border by a shipment, at dispatch (section 7);
- **payouts**, the treasury's payments to a class, in money or in kind, and the rule
  when the treasury holds less than is due (section 8);
- **standing orders**, the treasury's bid and ask in its seat region's market at a
  fraction of the reference price, and what a buffer store of goods is made of (section 9);
- the place of each in the tick, the arithmetic, the welfare consequence and the lever
  (sections 10 to 13);
- parameters, validation, observability, checkpoint state, the warm start, the entries
  for the stability register, the models run, the worked examples, notes, requirements,
  open questions and where it lives in the code (sections 14 to 25).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The jurisdiction, the chain, the crossing, the seat and the treasury account and agent; the combining rule of rows keyed by jurisdiction | `spec-polity.md` (POL) |
| Creating or destroying money (an authority's mint, a tribute paid abroad), the flow scale, the carry | `spec-money-supply.md` (MON): a grant or a levy whose target is a treasury (section 4.2, owed edit) |
| A rule that chooses a rate, a release or a purchase from a measure; a controller or a player as the chooser | `spec-policy.md` (PLC): it sets the flow scale through a lever of kind `flow` (`spec-policy.md` 6.3; section 13) |
| A charter, a monopoly, a price control, a usury cap; who may do what where | `spec-institutions.md` (ORG) |
| A treasury that hires labour, pays officials or buys services | `spec-employment.md` (EMP); open question 2 |
| A tax on a reason (rent, a wage, interest) | not in version 1 (open question 3); a tax on the holder role is a tax of section 5 |
| A store with room and a keeper | `spec-storage.md` (STO): a treasury's goods have no room limit in version 1 (9.6) |
| Loans, interest, a public debt | `spec-credit.md` (CRD): a treasury borrows nothing in version 1 |
| The outside world and its quotes | `spec-external-sector.md` (EXT): FIS taxes the trades its counterparties make (6) and nothing else |
| Loss, seizure, conflict, a blockade | `spec-risk.md` (RSK); an embargo is `spec-polity.md` 7.2.2 |
| What a size class of settlement unlocks | `spec-settlements.md` (SET) |
| Measures: revenue, the fiscal balance, a tax share | `spec-indicators.md` (IND): preset asks in section 22 |
| The tick, the draws, the slots and the options | `spec-world.md` (WLD), `spec-integration.md` (INT) |

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is a fiscal flow? | A rule, written as data, that moves money (or goods) between a class of accounts and the **treasury** of one jurisdiction, under a declared reason equal to its ID, at a fixed position of the tick, with no reading of an aggregate. Six kinds: tax, duty, toll, payout, purchase, release | One mechanism per question: who pays, on what base, when. MON's flows are the model (4, 5, 10 of that spec) |
| 2 | Does the module create or destroy money? | **No.** Every fiscal reason is **internal**: both sides are non-boundary accounts. An authority's mint or a tribute paid abroad is a MON grant or levy with the target kind `treasury` | `spec-ledger.md` 7.3; N-021 and N-107: one place for a faucet or a sink, named. The money supply `M` is unchanged by every flow of this spec, and a tax moves money between sectors (households or pool to the treasury sector) |
| 3 | One flow type with a shape field, or six typed lists? | **Six typed lists.** A tax is paid by a class, a duty by the party of a trade, a toll by a shipper, a payout by the treasury, a standing order is an order | Each kind has its own payer, base, position and refusal. A flat struct with exclusive shapes (MON's `ErrFlowShape`) would grow to 25 fields with most of the combinations invalid |
| 4 | Who is the counterparty? | The treasury of a **governed** jurisdiction, named by the flow as its **authority** (`ErrNotGoverned` otherwise). The class it acts on is the regions that jurisdiction **covers** (`ErrNotCovered`) | POL 5.4.2. An ungoverned jurisdiction has no account; an authority acts only inside its own jurisdiction |
| 5 | Where is a treasury's money attributed? | To the region of its seat (POL 5.3.4). A tax paid by the households of another region is a transfer between regions, reported by the aggregate event `FiscalFlowApplied` (the region of the payer, the amount), which is the **reconciling item** of `spec-indicators.md` 8.2 | Answers POL-A2 and POL open question 5 without a new account region or a rule that keeps the receipt in the payer's region |
| 6 | When does each run? | Taxes and payouts at **P1.7b**, after the money flows (P1.7) and before the outside's walk (P1.8); the treasuries' standing orders at **P3.4a**, after the estates' asks (P3.4) and before the counterparties (P3.5); duties at **P4.1a**, after every market has settled (P4.1) and before the sweep (P4.2); a toll at the **dispatch** of a shipment (P3.2(c) and P3.3) | Each position is the earliest at which the base exists and no reservation is open where money moves (WLD 6.1.6) |
| 7 | What does a rule read? | The tick and its window, the account's own balance and weight, its own carry, the value of a trade, the crossing of an edge and the carrying units of a shipment, the effective reference price of its seat region and its own holdings. **Never** a price level, a money supply, a count of accounts of a class, an income or a revenue | MON position 5, ADR 0001 decision 7, `spec-world.md` 9.4. A rule that reacts to a measure is PLC's |
| 8 | A payer cannot pay? | The part it cannot pay is **forgiven**, not carried, and counted (`Short`) | MON 5.4: no arrears, so no claim and no debt loop; the ledger's claims are credit's (`spec-ledger.md` 10) |
| 9 | The treasury holds less than the payouts due? | **Cut pro rata**, with the largest remainder and the keyed tie of WLD 10.1.1, per authority and per resource, over all its payouts at once; counted (`Cut`) | WLD 10.1.1 and D87: a binding limit must not serve the lower ID. A treasury never goes negative |
| 10 | Who bears a tax? | The **statutory payer**, entirely. There is no pass-through rule: the engine does not move a duty into a price. Any shifting is the agents' own, through cash and beliefs | The engine holds no demand or supply elasticity to apply one with (`spec-stability.md` 6.1); inventing an incidence would be a result. Hazard FIS-A6 |
| 11 | Does an agent see a tax? | A household sees only the cash it has. A merchant sees the toll of a route as a field of `Routes` (7.5). No view carries a duty, a tax rate or a treasury balance | Structure is allowed; a rate is a configured number, but the view of AGT is not widened for it in version 1 (open question 4) |
| 12 | Is a tax netted from the welfare ratio? | **Yes**: a tax, duty or toll paid is subtracted from `income_t`, and a payout that is not `NotIncome` is added, as relief and a MON grant are; both are **kept out of the observed wage**, by being stored in the record's existing grant field `G_t` as a signed sum (section 12) | Disposable income is what the target-income rule and the relief threshold should see, and a wage that the cost anchor reads is the return to labour, gross of what the state takes or gives; MON left the choice to this spec (MON open question 5) |
| 13 | What does the treasury spend on? | Payouts to households, to a pool or to a subordinate treasury, in money or in kind; and standing orders in its seat region's market. Not wages, not services, not a debt | Two mechanisms that exist in the engine (a transfer; an order). EMP and CRD own the rest |
| 14 | What is a buffer store of goods? | A standing **purchase** at a limit below the reference price, a standing **release** above it, and an optional **in-kind payout**, over goods held in the treasury account, which has no room limit and spoils as any account (9) | POL open question 6, STO 10.1. The rule is fixed in the data; only a call between ticks (the scale) changes it |
| 15 | Randomness | None. Remainders come from a carry started from the account ID; ties come from the stateless key `K(t, id)`. A treasury's order adds one tie-break key at P4.1, as any order does | Determinism (architecture.md 7.3); WLD 7: a function of the configuration and the state |
| 16 | The one lever in a run | The **flow scale** of MON 4.4.3, set between ticks by `SetFlowScales`; a fiscal flow's ID is accepted. No new slot, no new call | The brief; INT 5 table is the whole list, and PLC writes into it |

## 3. The module and its six kinds

### 3.1 The module

`Config.Modules.Fiscal` is a pointer; `nil` is off (`spec-config.md` 3.2). When off,
nothing in this spec runs, no state is added to an agent or the world, no reason is
declared and no event is emitted. It needs `Modules.Polity` with at least one governed
jurisdiction (`ErrModuleDependency`): a pack with no treasury has nothing to pay into.
A non-nil module with no flow is legal and does nothing (the warning `no_fiscal_flows`). Plain
data, each list sorted by ID, no maps:

```go
// Fiscal is the module. Its presence switches it on.
type Fiscal struct {
	Taxes     []Tax      // sorted by ID
	Duties    []Duty     // sorted by ID
	Tolls     []Toll     // sorted by ID
	Payouts   []Payout   // sorted by ID
	Purchases []Purchase // sorted by ID
	Releases  []Release  // sorted by ID
}

// Window says when a flow acts. A flow acts in tick t when t >= Start, and Until is
// 0 or t < Until, and Seasons is empty or the season of t is one of them.
type Window struct {
	Start   int64    // first tick it acts (default 0)
	Until   int64    // first tick it no longer acts; 0 means never
	Seasons []string // season names of the calendar; empty means every tick
}
```

3.1.1. **One namespace.** The IDs of all six lists, and the IDs of the money flows of MON,
are one set (the reason of a flow is its ID, 3.4): an ID repeated among the six lists is
`ErrDuplicateID`, and an ID equal to a built-in reason, to a MON flow or to a reason that
a later spec declares is `ErrReasonCollision` (MON 4.5.2: the one sentinel of a clash with
another flow's reason, whichever module declares it). The set of the six lists is
bounded: at most `MaxFiscalFlows` flows of all kinds together (`ErrTooMany`).

3.1.2. **Window.** `Start` and `Until` are ticks as in `spec-money-supply.md` 4.4.1; the
first `Step` is tick 0 and nothing is applied during construction (FR-FIS-006). `Seasons`
names seasons of `spec-calendar.md` 5 and needs a calendar that has them
(`ErrUnknownSeason`, the sentinel of `spec-production.md` 6): the season of a tick is
`Calendar.At(t).Season`, computed once at P1.1, so a season window is a function of the
tick and adds no state. A tax or a standing order that acts in one season only is a
pulse that every payer meets at once (hazard FIS-A8): the carry spreads an amount over the
ticks of a period and not over the seasons of a window, by design (open question 5).

3.1.3. **The flow scale.** Every flow of the module has a runtime **scale** in basis
points, default 10 000, bound 0 to `MaxFlowScale` (MON 4.4.3), set only between steps by
`SetFlowScales`, saved in a checkpoint, not part of the hash. It multiplies the flow's
configured **intensity**: a tax's `Amount` and `RateBP`, a
duty's `RateBP`, a toll's `Amount`, a payout's `Amount` and `Level`, and the `Quantity`
of a purchase or a release (not their limit prices, not `Exempt`, not `Hold` or
`Reserve`). `0` pauses the flow. The engine never sets a scale (FR-FIS-040).

### 3.2 The class that a flow acts on

The class of a tax or a payout is a `FlowTarget` of MON (4.1 there), extended by one
kind and one field (the edit is owed to `spec-money-supply.md`, section 22):

```go
// FlowTarget names a class of accounts.
type FlowTarget struct {
	Kind         string // "households", "pool" or (new) "treasury"
	Region       string // "" means every region the authority covers
	Role         string // households only; "" means every role (a difference from MON, 5.1)
	Jurisdiction string // treasury only: a governed jurisdiction below the authority
}
```

3.2.1. **Households.** As MON 4.3.1: every open Agent account whose agent's role is
`Role` (any role if empty) and whose region is `Region` (every region the authority
covers if empty), external agents included, estate and treasury pseudo-agents never.
An amount is per household, so an agent of weight `w` pays or receives `w` times it
(`spec-scale.md` 4.4). Membership is read at the position of the flow, so a household
that changes role or region moves between classes with its carry.

3.2.2. **A pool.** The Estate account of `Region` (every region the authority covers if
empty). It needs the estates module (`ErrModuleDependency`). Weight 1.

3.2.3. **A treasury** (new). The Treasury account of `Jurisdiction`, a **governed**
jurisdiction that is a proper descendant of the authority (`ErrNotCovered` otherwise);
with `Jurisdiction` empty, the treasuries of every governed proper descendant of the
authority, in ascending `JurisdictionID`. Weight 1. This is the **remittance**: a
subordinate authority pays a part of what it holds to its superior (tax), or the
superior pays it (payout). It uses the same code as every other class (accounts of another kind).

3.2.4. **Every region is one the authority covers.** A `Region` that the authority does
not cover is `ErrNotCovered`; with `Region` empty the class is the covered regions, so a
jurisdiction never taxes or pays outside its own (POL 6.2.1). An outside node has no
households and is never in a class.

### 3.3 The six kinds, in one table

| Kind | Type | Payer | Payee | Base or quantity | Position | Section |
|------|------|-------|-------|------------------|----------|---------|
| Tax | `Tax` | the accounts of a class (households, a pool, a subordinate treasury) | the authority's treasury | `Amount` per household per period, or `RateBP` of the balance above `Exempt` | P1.7b | 5 |
| Duty | `Duty` | the buyer or the seller of a trade (a household agent) | the authority's treasury | `RateBP` of the value of the trade | P4.1a | 6 |
| Toll | `Toll` | the shipper of a shipment that crosses a border | the authority's treasury | `Amount` per carrying unit per crossing | dispatch (P3.2(c), P3.3) | 7 |
| Payout | `Payout` | the authority's treasury | the accounts of a class | `Amount` per household per period, or a top-up to `Level`; money or one commodity | P1.7b | 8 |
| Purchase | `Purchase` | the treasury, buying | sellers in the seat region's market | `Quantity` a tick at a limit that is a share of the reference price | P3.4a | 9 |
| Release | `Release` | the treasury, selling | buyers in the seat region's market | `Quantity` a tick at a limit that is a share of the reference price | P3.4a | 9 |

### 3.4 Declared reasons

3.4.1. **A tax, a duty, a toll and a payout each is a reason of the ledger registry**, kind
**internal**, declared at construction with the flow's ID, owner this spec, between the
account kinds of its payer and payee (agent, estate or treasury to treasury for a tax,
a duty and a toll; treasury to agent, estate or treasury for a payout; money, or goods
for an in-kind payout). The registry is closed for a run (`spec-ledger.md` 2.3). **No
boundary account is created**: an internal reason has none (`spec-ledger.md` 3.4: a
boundary account is one per reason of kind faucet or sink), so a pack with fiscal flows
shifts no account ID (a treasury's account is POL's: `spec-polity.md` 5.2.1).

3.4.2. **A purchase and a release** use the existing reason `trade`: the treasury is a
party of an ordinary trade at the clearing price, and the ledger row of `trade` gains the
kind Treasury among the accounts it connects (`spec-ledger.md` 7.2, owed edit; FR-FIS-052).
Nothing is declared for them.

3.4.3. **What a reason is for.** The totals the ledger keeps per (region, reason) are what
IND and an audit read: a tax's reason shows what it moved, by the region of its payer
(`spec-ledger.md` 3.5). Because the reason is internal, `BoundaryFlow` does not report it
(`spec-observability.md` 4.4.1); the aggregate event of 16.1 does.

## 4. The treasury as a counterparty

### 4.1 What a fiscal flow changes, and what it cannot

4.1.1. **The money supply is untouched.** Every transfer of this module is between two
non-boundary accounts, so `M` of `spec-money-supply.md` 3.1.1 is the same before and
after, and the identity of 3.1.2 there holds with no new term (FR-FIS-002). What moves is
the split: a tax lowers the households sector (or the pool) and raises the **treasury
sector** that POL added (`spec-polity.md` 5.3.1); a payout does the reverse. The
sectoral identity of `spec-ledger.md` 9.3 becomes `delta(households) + delta(public) +
delta(contract) + delta(treasury) = faucets - sinks`, and every fiscal reason nets to zero
in it (FR-FIS-003).

4.1.2. **Non-negative balances, by construction.** A payer's collection is capped by its
balance above the exemption (5.4), and the treasury's payments are capped by its balance
(8.4, 9.3). No fiscal rule can overdraw an account, so the ledger's refusals
(`ErrInsufficientFunds`) are programmer errors here (`spec-ledger.md` 12.2). A credit
that would take a balance above `MaxMoney` is cut and counted (`Headroom`).

4.1.3. **Goods.** A purchase moves goods from households to the treasury by `trade`; an
in-kind payout moves them back by its own reason; a release is a `trade`. No good is
created or destroyed by this module. The treasury's goods age and spoil like any account's
(`spec-polity.md` FR-POL-034); the spoilage is the ledger's `spoilage` sink as for any
holder (`spec-ledger.md` 5.3).

### 4.2 Creating and destroying money stays MON's

An authority that mints, or a treasury that pays a tribute out of the realm, is a named faucet
or sink, so it is MON's. This spec asks one edit of `spec-money-supply.md` 4.3: a **target
kind `treasury`** for a grant (a faucet into the treasury of a governed jurisdiction) and
for a levy (a sink from it), with the field `Jurisdiction` of 3.2 and the shapes MON
already has (a rate, a top-up, a fixed levy, a rate levy), `ErrNotGoverned` for a
jurisdiction with no treasury, and the position P1.7 (before this module's P1.7b, so a
mint is spendable by a payout of the same tick). MON 4.3.2 says "the pool is where a grant
to 'the treasury' goes": since POL, "treasury" is the account of a governed jurisdiction
and the pool is the region's (the wording is edited, section 22). A pack that wants its
authority funded by issuance writes one MON grant to the treasury with a `Limit` and pairs it
with a sink, as MON 9.2 says; this module neither creates nor retires it.

### 4.3 The regions of a transfer, and the regional identity

4.3.1. A treasury's account is in the region of its seat (`spec-polity.md` 3.5.1, 5.1.1),
and a transfer between two non-boundary accounts is attributed to the region of the
**debited** account (`spec-ledger.md` 3.5, 7.4). A tax is therefore attributed to the
region of its payer and a payout to the seat's region, wherever the money lands. A tax
paid by the households of region `A` into a treasury seated in region `B` is a
**transfer between regions** that no item of `spec-indicators.md` 8.2 names (POL-A2).

4.3.2. **The reconciling item.** The aggregate event `FiscalFlowApplied` (16.1) carries,
for each flow, authority, region and class, what was applied. With the authority's seat
from the snapshot's `Jurisdictions` rows (`spec-polity.md` 10.5), IND forms
`FiscalOut_r` and `FiscalIn_r` and the money identity of 8.2 there gains the two terms:

```
d(M_h + M_p + M_c + M_t)_r  =  faucets_r - sinks_r + RelocMoneyIn_r - RelocMoneyOut_r
                               + SettledIn_r - SettledOut_r + FiscalIn_r - FiscalOut_r
```

where `M_t` is the treasury sector of the region (POL 10.5), and `FiscalOut_r` is the
`Applied` of the events whose payer or payee region is `r` and whose other end is in
another region (a payout is `FiscalOut` of the seat region and `FiscalIn` of the
recipient region). Every term is an event field, so the regional identity stays exact
(FR-FIS-046). The alternative that POL listed, a treasury with no region, was rejected:
`spec-regions.md` 6.5.3 reserves `NoRegion` for an account between regions, and a
treasury is never between.

4.3.3. **Goods stay in one region.** An in-kind payout and a release act only in the seat
region (8.5, 9.1), so no good crosses a region boundary outside a shipment and the
regional goods identity of IND 8.2 has no new term.

### 4.4 What a treasury is not

It is not the pool: it has no automatic release, no relief and no funding of entrants
(`spec-polity.md` 2, position 7), and it receives nothing the pool receives (the estates
of the dead). It is not an agent: it is in no count of households and agents, has no
role, strategy or belief, and is never passed to `Work` or `Trade`
(`spec-polity.md` 5.3.3). It places orders only through the standing orders of section 9,
which are the module's code and not a strategy; and it spends only what a flow of this
module says. A treasury with money and no flow that spends it is idle for the life of the
run (the warning `treasury_idle`).

## 5. Taxes

### 5.1 What a tax is

5.1.1. A **tax** is a recurring transfer from a class of accounts to the treasury of its
authority: a **head tax** (a fixed amount a household), a **holding tax** (a rate on the
balance above an exemption), a tax on a **pool**, a **remittance** from a subordinate
treasury (3.2.3). These are words for the four ways to write one `Tax`; the library has
one mechanism, and nothing branches on a role or a region (architecture.md 9.1).

```go
// Tax is a recurring transfer from a class to the treasury of its authority.
type Tax struct {
	ID        string
	Authority string     // a governed jurisdiction: the treasury that is paid
	Target    FlowTarget // who pays (3.2); Role "" is every role
	Amount    int64      // fixed: minor units per household (or per account) per Per
	RateBP    int64      // or: basis points per Per of the balance above Exempt
	Exempt    int64      // minor units per household (or account) never collected
	Per       string     // "tick" (default) or "year"
	Window    Window
	Label     string // presentation only
}
```

5.1.2. **Exactly one of `Amount` and `RateBP`** (`ErrFiscalShape`); `Exempt` applies to
both shapes. A difference from MON, stated: MON's household target needs a role
(`ErrFlowShape`), and a tax may name none, because "every household" is the common head
tax and the alternative is one flow per role.

### 5.2 The due of one payer

With `w` the weight (1 for a pool or a treasury), `bal` the balance, `S` the flow's scale
in basis points, `P` the ticks of the period (1 for `tick`, `TicksPerYear` for `year`,
`spec-calendar.md` 3.2) and the arithmetic of 11 (a 128-bit intermediate, rounding
toward zero on non-negative values, the remainder carried):

| Shape | Due | Collected |
|-------|-----|-----------|
| Fixed | `num = w*Amount*S + carry`; `due = num / (P*10^4)`; `carry = num mod (P*10^4)` | `min(due, avail)` with `avail = max(0, bal - w*Exempt)` |
| Rate | `base = max(0, bal - w*Exempt)`; `num = base*RateBP*S + carry`; `due = num / (P*10^8)`; `carry = num mod (P*10^8)` | `min(due, base)` |

The due of a payer depends only on its own weight, balance and carry, the flow's fields
and scale, and the tick: not on any other account, a price or a total (FR-FIS-020). The
carry of a fixed tax starts at `(ID mod P) * 10^4`, `ID` the account ID, so that a small
rate is first paid at a different tick by each payer, with no random number
(`spec-money-supply.md` 6.2); that of a rate tax starts at 0, as does that of a duty
(6.3) and of a standing order (9.3.1), and a payout's rate carry starts as a fixed
tax's (8.3). The carry advances by the
**due**, not by what was collected: what a payer could not pay is forgiven, not owed
(position 8, `spec-money-supply.md` 6.1.2).

### 5.3 The order

5.3.1. At P1.7b, in this order (a rule constant, FR-FIS-022):

1. the taxes whose target is **households or a pool**, in ascending flow ID; for each,
   regions in ascending `RegionID` and accounts in ascending account ID;
2. the taxes whose target is a **treasury** (the remittances), in ascending flow ID; for
   each, payer treasuries in ascending `JurisdictionID`.

The remittances come second so that a subordinate authority remits what it collected from its own
households in the same tick, and the superior's own payouts (8) come third so that a
tick's receipts fund the same tick's payments. A payer that meets several taxes meets
them in flow ID order and each sees the balance the earlier ones left: the order within
one payer is by ID and is the rule's; nothing across payers is ordered by anything but
the tuple above (`spec-world.md` 6.1.2).

5.3.2. **The treasury at the limit.** If the amounts of one flow would take the treasury
above `MaxMoney`, the collection is cut to the headroom by the pro rata rule of 8.4 over
the amounts the payers would have paid, and the cut is counted (`Headroom`). The case is
a bound violation in practice (`MaxMoney` is about 10^18) and the rule exists so that no
branch is undefined.

### 5.4 What is not collected

- **Short.** The payer holds less than the due above its exemption: it pays what is
  there and the rest is forgiven; the households whose tax found too little are counted
  (`Short`, in the event of 16.1). No arrears accrue; the ledger has no claims in version
  1 (`spec-ledger.md` 10).
- **Headroom.** The treasury's balance would pass `MaxMoney` (5.3.2).

### 5.5 A tax that takes upkeep money

A fixed tax runs before the household buys its upkeep (P1.7b precedes P3.2), so a
household taxed below its upkeep cost goes unfed, its privation rises and the failure
ladder starts (`spec-spawner.md` 5; the hazard FIS-A1, the same as MON-A4). The
exemption is the guard: a tax never takes the balance below `w * Exempt`. The warning
`tax_below_upkeep` fires when a tax on households has an `Exempt` below one tick's
upkeep cost of the targeted role at the reference prices of its regions (as MON's
`levy_below_upkeep`). The treasury may pay the money back (a payout, 8) and the pool's
relief is separate from it; neither is a damper the engine applies on its own.

### 5.6 A tax on a pool

A tax on a pool collects from the estate account at P1.7b, after the relief of P1.6 has
been paid in the same tick and before the entry step of P7.1 (`spec-money-supply.md`
7.5), so it never takes money that relief has already committed; it can take money that
the same tick's entrants would have been funded from, and the entrants' budget
(`EntryPoolShare`) is a share of what is left (`spec-spawner.md` 7.3). A pool is a
holder too (`spec-money-supply.md` 7.4): a rate tax on it bounds what the pool keeps by
the same derivation as MON 9.3, `m* = E + g / rho`, with `g` the money that reaches the
pool per tick.

## 6. Duties

### 6.1 What a duty is

A **duty** is a tax on the **value** of trades executed in markets of the authority's
jurisdiction: the **sales tax** (the seller pays a share of what it received), the
**purchase tax** (the buyer pays a share of what it paid), and, with `Scope`, the
**tariff** (the trade has a counterparty of the outside world on the other side).

```go
// Duty is a tax on the value of trades.
type Duty struct {
	ID        string
	Authority string // a governed jurisdiction: the treasury that is paid
	Region    string // "" is every region the authority covers
	Commodity string // "" is every commodity
	Payer     string // "seller" or "buyer"
	Scope     string // "domestic", "import", "export" or "any"
	RateBP    int64  // basis points of the value of the trade, 1 to MaxDutyBP
	Window    Window
	Label     string
}
```

6.1.1. **Scope and payer.** A trade has a buyer and a seller. A party is a
**counterparty** if its agent ID is in the range of the counterparties of the external
module (`spec-world.md` 5.3). `domestic` matches a trade with neither party a
counterparty; `import` a trade whose seller is a counterparty (the realm buys from
outside); `export` one whose buyer is (the realm sells outside); `any` all of them. The
combinations that can never match are `ErrFiscalShape`: `import` with `Payer` `seller`
and `export` with `Payer` `buyer` (a counterparty never pays, 6.4). `import` and `export`
need the external module (`ErrModuleDependency`).

### 6.2 The trades and the base

6.2.1. At P4.1 the clearing returns, for each market in ascending `RegionID` then
`CommodityID`, the executed trades in trade order, each with its region, commodity,
buyer, seller, price and quantity (`TradeExecuted`, `spec-clearing.md` 9). The world
holds that list in scratch until P4.1a (the same data the events carry, kept whether or
not an observer is present). The **value** of a trade is `price * quantity`, checked.

6.2.2. A trade is in the base of a duty if its region is covered by the duty's authority
and equals `Region` if set, its commodity equals `Commodity` if set, its scope matches
(6.1.1), and the flow is in its window at tick `t`. For each (duty, payer) the **base**
is the sum of the values of its matching trades in the tick, in 128 bits.

### 6.3 The due

For one (duty, payer account), `S` the flow's scale and `carry` the payer's carry for
that duty in units of `1 / 10^8` of a minor unit:

```
num   = base * RateBP * S + carry
due   = num / 10^8              (toward zero)
carry = num mod 10^8
paid  = min(due, balance of the payer after P4.1's settlement)
```

The carry makes the total paid over any run of ticks exact: with unchanged `RateBP` and
`S` it is `floor((sum of base * RateBP * S + carry_0) / 10^8)` with no drift, however the
value splits into trades or ticks (the model of 20 checks 2 000 random runs). The base is
**summed over the tick before the carry is applied**, so the result does not depend on
the order of the trades or of the duties (FR-FIS-032). A payer that cannot pay is
forgiven the rest (5.4) and the carry advances by the due.

### 6.4 Who pays

A party that is a pseudo-agent never pays a duty: an estate (its asks are of public
money), a counterparty (its account is a pass-through account of the outside, topped up
and swept, `spec-external-sector.md` 4) and a treasury (its own purchases). The trade is
in no base for a payer of that side and is counted in the event (`Exempt`, trades). A
tariff is therefore always paid by the **domestic** party: the buyer of an import or the
seller of an export. The duty touches only domestic accounts and so does not disturb the
sweep of P4.2 (6.5).

### 6.5 Position

P4.1a is after P4.1 has cleared and settled every market (the proceeds are in the
seller's account, the reservations are released, `spec-world.md` 6.1.6) and before P4.2
(the sweep of the counterparties). A seller always holds what it received and `MaxDutyBP` is 10 000 basis points, so at a scale of
10 000 a sales tax is short only when several duties stack or the scale is above 10 000
(6.7); a buyer holds only what it did not spend, so a purchase tax can be short. The proceeds of a sale are spendable at the
next tick's P3.2 at the earliest (`spec-world.md` 6.1.8), so the duty is taken before
the seller can use the money. The duties run in ascending flow ID; for each, payers in
ascending account ID; and the events follow, per flow and region (16.1).

### 6.6 What a duty does not do

It does not change the clearing price, a limit price, a belief or an order: it is taken
**after** the trade, from the party the flow names. No agent reads it (position 11), and
the price a seller sees in its belief update is the clearing price and not the price net
of the duty. The **incidence is statutory** (position 10). A pack that wants a duty to
move prices has the agents' own response to the lower cash and income that it causes
(5.5, 12), which is a loop of the other specs; the entry FIS-A6 says so. A tariff is a
duty; a **quota** or a ban is the capacity of an outside line and the status of its gate
edge (`spec-external-sector.md` 6, `spec-regions.md` 7), not this spec's.

**Evasion** is not modelled. A payer that is short is forgiven (5.4), which is the only way
not to pay and which is counted; a household that avoids a duty by choosing a market or a
good would be a strategy reading the rate, which no view carries (position 11, open
question 4).

### 6.7 Several duties

Two duties may match one trade (an outer authority's and an inner one's): each acts for itself, the
payer's balance falls by each in ascending flow ID, and the later one finds less when the
payer is short. Nothing caps the sum: a pack that stacks duties above the value of a
trade has a payer that is short and counted (`Short`), and the warning
`duty_stack_above_whole` fires when the rates of the duties that can match one trade of
one payer sum above 10 000 basis points (the configured rates, at a scale of 10 000: the
scale is runtime state and the warning is a construction rule). The scale can lift one
duty above the whole value too (`RateBP * S / 10^4` up to `MaxDutyBP * MaxFlowScale /
10^4`, 10 times the value); the payer is then short and counted, and the call is not
refused, because the bound is on the configured rate and the scale has its own bound
(3.1.3). A consumer that wants no duty above the value keeps its scales at 10 000 or
below.

## 7. Tolls

### 7.1 What a toll is

A **toll** is a charge on a shipment for crossing a **border** (`spec-polity.md` 4.3): it
is paid by the shipper to the treasury of a jurisdiction that the shipment **leaves** or
**enters**, once for each edge of its path that crosses that border.

```go
// Toll is a charge per carrying unit for crossing a border.
type Toll struct {
	ID        string
	Authority string // a governed jurisdiction: the treasury that is paid
	On        string // "exit": the shipment leaves the authority; "enter": it enters it
	Mode      string // "" is every transport mode
	Amount    int64  // minor units per carrying unit per crossing
	Window    Window
	Label     string
}
```

It needs the transport module (`ErrModuleDependency`) and, when `Mode` is set, a mode of
it (`ErrUnknownMode`).

### 7.2 The crossings of a shipment

7.2.1. A shipment from `from` to `to` follows the shortest path of the graph (the path
whose distance gives its travel time, `spec-transport.md` 4.1), a list of edges. For
each edge `(a, b)` the **crossing** of `spec-polity.md` 4.1.1 gives `Exit` and `Enter`.
A toll acts on the edge if the flow is in its window at the tick of dispatch, its `Mode`
is empty or the shipment's, and its authority is in `Exit` (when `On` is `exit`) or in
`Enter` (when `On` is `enter`). The crossing of the whole route is **not** used: the ends
of a route omit a jurisdiction that it leaves and re-enters (`spec-polity.md` 4.3.4).

7.2.2. A jurisdiction that a shipment passes through and does not leave is not charged:
a toll on exit of an outer jurisdiction for a shipment between two regions it covers
charges nothing, since both are in its chain (`Exit` is empty). A border is charged each time it is crossed.

### 7.3 The amount

For a shipment that uses `n` carrying units (`spec-transport.md` 3.2.1, 3.2.3: the
binding fill of mass and bulk, `n` at least 1) and a crossing the toll applies to,

```
due = floor(Amount * S * max(1, n) / 10^4)
```

with one truncation per crossing and no carry (a toll has no payer state, so none is
saved). The total of a shipment is the sum over its edges, in path order, and over the
tolls, in ascending ID; it is the one number the refusal of 7.4 checks. The product is
checked and an overflow is the refusal `overflow` that TRN already has (`spec-transport.md`
5.4, row 23).

### 7.4 Dispatch and the refusal

`spec-transport.md` 5.4 gains the refusal `toll_unaffordable`, row 22a (after
`unprovisioned`, before `overflow`): the shipper's available money (its balance less
what is reserved) is below the total toll. A refusal changes nothing and emits
`ShipmentRejected` (`spec-transport.md` 5.4). `spec-transport.md` 5.5.1 gains the step
**3a, Toll**: for each crossing in path order and each toll in ascending ID with a due
above 0, `Transfer(shipper, treasury, due, reason = the toll's ID)`; the world cannot fail
here (the check ran). `ShipmentDispatched` gains the field `Toll` (the total, minor
units). The same step and refusal act in the contract round of P3.3, where the offerer is
the shipper (`spec-transport.md` 5.3.2). **The payer is the shipper**: the agent whose
account supplies the goods (hired) or the traveller (self-carried); a consignee pays
nothing.

`spec-transport.md` 3.3 and its position 3 say transport "never moves money". The toll is
the one exception, owed by a spec that said a toll would be FIS's; the sentence is edited
(section 22) and the module off leaves the position true.

### 7.5 What a strategy sees

A merchant that chooses its trips by margin (`spec-merchants.md` 5) must count a toll or
it will make trips that are refused or that lose money. The `Routes` field of the view
(`spec-transport.md` 9.1) gains `Toll`: for each other region and mode, the money one
carrying unit pays in tolls on the shortest path at the scales and in the windows of the
tick, a pure read of the configuration, the scales and the graph (cached when the graph
or a scale changes, so it is `O(1)` per read). The default strategy never ships and reads
nothing. MER's margin per haul unit subtracts `Toll * n / (n * s)` of a trip (the edit is
owed, section 22); how MER's shipper responds is its register entry's (MER-A1, the gain
of the response to a gap), not this spec's.

### 7.6 What a toll is not

It does not choose the path: REG's shortest path reads only the distance, so a toll never
reroutes a shipment (a route that avoids a border is a different pack's graph). It is
not a capacity or a queue (`spec-transport.md` 6, open question 4 of `spec-infrastructure.md`:
an edge limit is a different spec's). It is not charged by value: a value toll would read
a price of the cargo, which is a read of the market at dispatch; a per-unit toll is the
form with no aggregate (open question 6). A **border** between the realm and an outside
node is not a shipment's (a shipment to a node is refused, `spec-transport.md` 5.4 row
6a); a tariff is a duty (6).

## 8. Payouts

### 8.1 What a payout is

A **payout** is a recurring transfer from a treasury to a class: a **stipend** to a role,
**relief** in kind, **aid** to a pool or **a grant-in-aid** to a subordinate treasury. It
is the module's only way to put money or goods back into the economy, and its source is
the treasury's balance and nothing else (no faucet, position 2).

```go
// Payout is a recurring transfer from the treasury of its authority to a class.
type Payout struct {
	ID        string
	Authority string     // a governed jurisdiction: the treasury that pays
	Target    FlowTarget // who receives (3.2)
	Commodity string     // "" is money; else units of this commodity, in kind
	Amount    int64      // per household (or account) per Per: minor units, or units in kind
	Level     int64      // money only: top the balance up to Level
	Per       string     // "tick" (default) or "year"; not with Level
	NotIncome bool       // households only: not income for the welfare ratio (12)
	Window    Window
	Label     string
}
```

### 8.2 Shapes

Exactly one of `Amount` and `Level` (`ErrFiscalShape`); `Level` only for money and with
`Per` `tick`. An in-kind payout has `Amount` in whole units per household per `Per`, with
a carry like a money amount (a unit is not divisible: the carry gives whole units over
time), a target of households only, and a `Region` that is the **seat** of the authority
(`ErrFiscalShape` otherwise: a good does not cross a region outside a shipment, 4.3.3).

### 8.3 The due of one recipient

As a grant of MON (5.3 there): a rate `num = w*Amount*S + carry`, `due = num / (P*10^4)`,
carry `num mod (P*10^4)`, started at `(ID mod P) * 10^4`; a top-up
`due = max(0, w*Level*S/10^4 - bal)` with no carry. The due depends on the recipient's own
state only (FR-FIS-020), and the carry advances by the due, not by what was received.

### 8.4 When the treasury holds less than is due: the pro rata cut

8.4.1. For one authority and one **resource** (money, or one commodity) the payouts of
the tick are taken **together**: the dues of every recipient of every payout flow of that
resource are computed first, in ascending (flow ID, region, account) order, and summed:
`D = sum of due_i`. `A` is what the treasury holds of the resource (its balance, or its
unreserved units of the commodity), after the taxes and remittances of the tick.

8.4.2. If `D <= A`, every recipient receives its due. Otherwise, with products in 128 bits:

1. `q_i = floor(due_i * A / D)` and `phi_i = (due_i * A) mod D`;
2. `L = A - sum of q_i`, the leftover (`0 <= L` and `L < n`, where `n` is the number of
   recipients with `phi_i > 0`, because `sum of phi_i = L * D` and each `phi_i < D`);
3. the recipients are ordered by **descending `phi_i`**, then **ascending key**
   `K(t, id)`, then ascending account ID, where `K` is the stateless key of
   `spec-resources.md` 5.3.3 (SplitMix64 finaliser) applied to the **tick** and the
   **agent ID** of the account's owner (an estate's agent ID is its `RegionID`; a
   treasury's is POL's);
4. the first `L` of them receive one unit more: `applied_i = q_i + 1`, which is at most
   `due_i` because `due_i * A / D < due_i`.

The result depends on no list order (the order of 3 is total and ends in a unique ID,
architecture.md 7.6), uses the whole of `A` (`sum of applied_i = A` exactly), gives no
recipient more than its due, and a recipient due nothing gets nothing. It takes no draw
and keeps no state: WLD 10.1.1's rule for a new tie, so that the lower ID is never
systematically first (the model of 20: over 1 000 ticks, ten equal recipients and five
extra units a tick, each recipient received between 475 and 525 of them; "lower ID
first" would give 1 000 to five of them and none to the rest).

8.4.3. **What the cut does not do.** It does not carry the shortfall, so a recipient
that was cut is not owed the difference (the carry advances by the due). It does not
reserve anything for a later tick. After a cut the treasury holds 0 of the resource: it
spent all it had, which is the rule of a budget with no reserve, and a pack that wants a
reserve writes a smaller `Amount` and lets the balance build (open question 9). A **headroom** cut at a recipient (its balance would pass `MaxMoney`) is
applied after the pro rata: the unapplied part stays in the treasury.

8.4.4. **Cost.** The cut costs `O(n log n)` for `n` recipients and only in a tick in
which the treasury is short; with enough money, `O(n)`. The sort is the only one in the
module and has a total order.

### 8.5 In kind

An in-kind payout moves `applied_i` units of the commodity from the treasury's account
to the recipient's, oldest lots first (`spec-ledger.md` 5.2), under the payout's reason
with the lots' born tick and remaining durability preserved
(`spec-ledger.md` 5.2.1). A perishable's units that are about to spoil are the oldest and
so go first: **a payout in kind is the way a buffer store empties before its stock spoils**
(9.6). No unit is created or destroyed. The region is the seat's, so the move is within
one region. The units a household receives are counted in `P_t`, at the value WEL 6.2
gives an in-kind receipt, unless `NotIncome` (12.1).

### 8.6 A payout to a pool or to a treasury

A payout to a **pool** is money only, raises `M_p` and not `M_h`
(`spec-money-supply.md` 7.4); the money reaches households through the pool's own
relief and entry. The warning `payout_pool_no_outlet` fires when the region's
`PoolRelease` is 0 (MON's `grant_to_pool_no_outlet`). A payout to a **treasury** is a
grant-in-aid to a governed jurisdiction below the authority (3.2.3).

## 9. Standing orders: purchase and release

### 9.1 What they are

A treasury may buy and sell goods in the market of its **seat region** through two kinds
of flow. They are the module's only orders, written as data and placed by the world at
P3.4a; they are not a strategy and not an extension point (architecture.md 4.3 is
unchanged).

```go
// Purchase is a standing bid of a treasury in its seat region's market.
type Purchase struct {
	ID        string
	Authority string // a governed jurisdiction: the treasury that buys
	Commodity string
	Quantity  int64 // units a tick, at most
	LimitBP   int64 // the bid limit, as basis points of the effective reference price
	Hold      int64 // stop when the treasury holds this many units; 0 means no cap
	Window    Window
	Label     string
}

// Release is a standing ask of a treasury in its seat region's market.
type Release struct {
	ID        string
	Authority string // a governed jurisdiction: the treasury that sells
	Commodity string
	Quantity  int64 // units a tick, at most
	LimitBP   int64 // the ask limit, as basis points of the effective reference price
	Reserve   int64 // units never released
	Window    Window
	Label     string
}
```

9.1.1. **Where.** The seat region of the authority (`spec-polity.md` 5.1.1) and no other:
an order is admitted only for an agent whose account is in the market's region
(`spec-clearing.md` 3.2, `unknown_agent`). A treasury that wants goods of another region
buys them through a merchant of that region, or through its own seat's market. The
outside node of an external module is reached by its gate region's market, so a treasury
seated at a gate region may buy imports there (6.4 exempts it from the duty).

### 9.2 The price rule

The limit is a function of the configuration and of nothing that moves:

```
limit = min(MaxPrice, max(1, ref * LimitBP / 10^4))          (the scale does not act on a price)
ref   = the effective reference price of the commodity in the seat region
        (`spec-goods.md` 6.3, `spec-regions.md` 5.2)
```

truncated toward zero. A purchase with `LimitBP` 8 000 bids at 80% of the reference
price, a release with 12 000 asks 120%. The treasury reads **no** market price, no
belief and no window, so a standing order is not a controller on a measure: it is an
agent with a fixed limit, whose quantity is fixed and whose effect on the price is what
the clearing makes of it (position 7; the model of 9.7).

### 9.3 The quantity

9.3.1. **The wish.** Each tick a standing order acts, with scale `S`:
`num = Quantity * S + carry`; `wish = num / 10^4`; `carry = num mod 10^4`; one carry per
flow, saved (11, 17). A scale of 5 000 and a `Quantity` of 1 therefore sells one unit
every second tick.

9.3.2. **The caps**, taken in this order, each counted (16.2):

| Kind | Cap | Counted as |
|------|-----|------------|
| Purchase | the cash: `floor(available money / limit)` | `CutCash` |
| Purchase | the hold: `Hold - held units` (when `Hold` is above 0) | `CutHold` |
| Release | the stock: unreserved `held units - Reserve` | `CutStock` |

The order is **placed** if the capped quantity is at least 1: a bid of that many units at
the limit, or an ask of that many units at the limit, with the reservation of
`spec-clearing.md` 3.3. Otherwise none is placed, and the wish is counted as cut. `MaxQuantity`
bounds `Quantity` and `Hold`.

### 9.4 Admission, the pair and the order of the market

9.4.1. **One order per commodity** is a rule of `spec-clearing.md` 3.1.1 that exists so
that an agent never trades with itself. A treasury may need a bid and an ask for one
commodity (a band). The rule of 3.1.1 and 3.1.2 reads, with this spec: **at most one
order per agent, commodity, named lot and side**. The pair is admitted because the two
limits can never cross, which `spec-clearing.md` 4.3 defines on the **integer limit
prices** (a bid whose limit is not below the ask's trades): the limits are truncated
(9.2), so two different basis points can give equal limits (a reference price of 50 with
8 000 and 8 100 gives 40 and 40). A pack that gives one authority a `Purchase` and a
`Release` of one commodity must therefore have, **on the integers**,
`limit(Purchase) < limit(Release)`, `limit` being the formula of 9.2 with the effective
reference price of the commodity in the seat region and the clamps to 1 and `MaxPrice`
(`ErrFiscalShape`; two purchases, or two releases, of one commodity of one authority are
also `ErrFiscalShape`: the quantities add up in one flow). The reference price is
configuration (`spec-goods.md` FR-GDS-020: the region's override or the commodity's
`ReferencePrice`), so the check holds at every tick (FR-FIS-053). The clearing may
therefore assume that **a treasury never matches itself**, the identity of `spec-clearing.md`
3.1.1 that the one-order rule exists to keep.

9.4.2. **Position in the list.** In a market's input order the orders are the agents' (in
ascending agent ID), the estates' asks, **the treasuries' (ascending `JurisdictionID`,
the bid before the ask)**, then the counterparties' (`spec-world.md` 6.5.1). The orders
are placed at P3.4a in that one rule: authorities ascending `JurisdictionID`; within an
authority its purchases in ascending flow ID, then its releases in ascending flow ID (a
market has at most one of each of one authority, 9.4.1, so in a market the bid comes
before the ask); and the `OrderPlaced` events follow the placing order. The
tie-break keys of P4.1 are drawn one per order in that order, bids then asks: a
treasury's order takes one word like any, and the number of words is a function of the
configuration and the state (FR-FIS-054).

### 9.5 No outcome, no belief

A treasury's order has no outcome and takes no draw at P5.4 (the ruling C21 of
`spec-world.md` extends to it, as to an estate's ask and a counterparty's order); there is
no belief to revise. The price window and the record of INF are pushed with the clearing
price as for any trade (P5.1, P5.2): **the treasury changes the price the others see**, by
trading in the market (9.7).

### 9.6 The goods of a treasury

9.6.1. **The account is unlimited** in version 1: a treasury is a pseudo-agent with no
role and so no base room, so the storage module's limit does not apply to it
(`spec-storage.md` 3.3.3: "a pseudo-agent is never limited"; 3.5.1). `Hold` is
the pack's own cap. A store owned by a jurisdiction (H9 of `spec-polity.md` 6.3) stays
reserved: the first consumer's buffer store is the player's (`spec-storage.md` 10.1) and
the alternative of this section needs none. When STO accepts a treasury as a holder, the
`Hold` cap is replaced by its room and nothing else changes.

9.6.2. **Spoilage.** The units a treasury holds age and spoil as any (`spec-ledger.md`
5.3): a lot of a perishable commodity moves whole to the `spoilage` boundary at the end of
the tick in which `Born + ShelfLife - 1` passes. A purchase of a perishable that is not
released or paid out before its shelf life ends is a loss of money for goods that no one
ate (hazard FIS-A4). The release and the in-kind payout take the oldest lots first, so a
buffer store rotates by itself; the pack decides the rate (`Quantity`) and the season.

### 9.7 What a band can and cannot do

A purchase at 80% and a release at 120% of the reference price make a **band**: below
80% the treasury absorbs a glut, above 120% it releases against a shortage, and between
them it does nothing. The toy market of `models/fis/band.py` (linear demand and supply,
clearing price 50, band 40 to 60, 500 units a tick, a hold of 6 000, 3 000 units to start,
150 000 in cash; not the library) gives:

| Case | Without a treasury | With it |
|------|-------------------|---------|
| Shocks of -2 000 to +2 000 units (price -20 to +20), 400 ticks: ticks outside the band | 178 of 400 | 83 of 400 |
| Persistent shortage (untreated price 65), 120 ticks from the tick the shift begins | 120 of 120 | the 3 000 units are released at 500 a tick and are gone after 6 ticks; the price is then 65, 5 above the ask limit of 60; 114 of 120 ticks outside the band (the first 6 are at 60) |
| Persistent glut (untreated price 35), 120 ticks from the tick the shift begins | 120 of 120 | the hold of 6 000 is reached after 6 ticks of purchases; the price is then 35, 5 below the bid limit of 40; 114 of 120 outside |

So a band **damps shocks that reverse** and cannot defend a persistent shift: the
stock or the cash runs out in `stock / release rate` ticks (6 here) or `(Hold - start) /
purchase rate`, after which the price is what the market makes it. In the iid case the
treasury ended with more cash than it began with (it bought low and sold high on the
model's linear market, which has no spoilage and no costs); a real band has a carrying
cost and spoilage, and nothing here says it pays for itself. The entries FIS-A4 and FIS-A5
carry the hazards (exhaustion; a support price that attracts supply).

### 9.8 A buffer store

A pack builds the first consumer's buffer store from the data: a `Purchase` of the staple in
the harvest season (`Seasons`), a `Release` in the lean season with a higher limit and a
`Reserve`, and, for a famine, an in-kind `Payout` to the households of the seat region
with `Window.Seasons` the lean season, `NotIncome` false. The rules are fixed; a consumer
or a policy actor changes the **scales** between ticks (13). The first consumer's game
may instead give the player's agent a store (STO 10.1) and drive it by directives: both
exist, and a pack uses one.

## 10. The step

### 10.1 The positions

The module takes four positions of the tick of `spec-world.md` 6 and one step inside the
dispatch of `spec-transport.md` 5.5. None takes a draw. Each is added to WLD's table in
the change that accepts this spec (`spec-world.md` 6.1.10; the edits are in section 22).

| ID | What runs, in this order | Reads / writes | Draws | Events |
|----|--------------------------|----------------|-------|--------|
| P1.7b (new), after P1.7 and before P1.8 | (1) the taxes whose target is households or a pool, ascending flow ID (5.3); (2) the taxes whose target is a treasury, ascending flow ID; (3) the payouts, per authority ascending `JurisdictionID`, per resource (money first, then commodities ascending): the dues, the cut, the transfers (8.4) | flow state, balances, weights, the treasury's holdings; writes balances, lots, carries | none | after the step, per flow in the order above and per region ascending: `FiscalFlowApplied` |
| P3.2(c) and P3.3, in the dispatch of a shipment | the checks of `spec-transport.md` 5.4 with the new row 22a `toll_unaffordable`; step 3a, the toll (7.4) | the path, the crossings, the shipper's balance; writes the shipper's and the treasury's balances | none | `ShipmentDispatched` with `Toll`, or `ShipmentRejected` |
| P3.4a (new), after P3.4 and before P3.5 | for each authority ascending `JurisdictionID`, for each standing order active in its window: its purchases in ascending flow ID, then its releases in ascending flow ID (9.4.2): the price, the quantity, the caps, the order (9) | the reference price, the treasury's balance and holdings; writes reservations, the order list of the seat's market, carries | none (the order takes one tie-break key at P4.1) | per authority ascending: its `OrderPlaced` or `OrderRejected` per order in placing order, then its `TreasuryOrdered` |
| **P3.6c** (new), after P3.6a (STO's `RoomLimited`) and P3.6b (SAV's `SavingPlanned`), the aggregate events of the phase | the tolls of the tick, aggregated per flow and region | the tick's toll totals | none | `FiscalFlowApplied` for the tolls, after `WantShortfall`, `RoomLimited` and `SavingPlanned` |
| P4.1a (new), after P4.1 and before P4.2 | the duties, ascending flow ID; for each, payers in ascending account ID (6) | the trades of P4.1, balances, carries | none | per flow and region ascending: `FiscalFlowApplied` |

CRD's settlement pass is the position P1.7a (`spec-credit.md` 8.1.1), which runs just before P1.7b. A debtor
short of cash therefore pays its claim first, and the tax finds less to take: the part it cannot pay is
forgiven (5.4). This is a consequence of the order accepted for both specs, not an oversight.

### 10.2 What WLD must say (owed, section 22)

10.2.1. **Reservations** (`spec-world.md` 6.1.6): they open at P3.2, P3.4 and P3.5 and are
all settled or released by the end of P4.1. P3.4a opens reservations (a bid reserves its
cash, an ask its units) and is added to the list. P1.7b and P4.1a open none and move money
directly, which is why they are after P4.1 or before P3.2.

10.2.2. **The input order of a market** (6.5.1): the agents', the estates', **the
treasuries'**, the counterparties'.

10.2.3. **No outcome** (ruling C21): extended to a treasury's order (9.5).

10.2.4. **The draw table** (7.3): no row is added. The one place the stream is touched is
the tie-break key per order at P4.1, whose count is "one per order"; a treasury's order
is counted there, so "the number of orders" is still a function of the configuration and
the state (`spec-world.md` 7.4).

10.2.5. **A tie with no draw**: the cut of 8.4 uses `K(t, id)` of `spec-world.md` 10.1.1
and is the first user of that rule outside RES.

### 10.3 Atomicity and order of events

Each transfer is atomic (`spec-ledger.md` 2.6). The step of P1.7b is a sequence of
transfers; a checkpoint is taken between ticks and never sees half of it
(ADR 0002 decision 4). The events are emitted after the transitions they describe
(architecture.md 12.5), in the order of 10.1, and the aggregate for a flow is built from
counters kept in slots preallocated by the world (no event is constructed without an
observer, observability-and-logging.md 4.5).

## 11. Arithmetic and bounds

11.1. **Checked and 128 bits.** Every product goes through the checked multiply-divide
helper with a 128-bit intermediate (`spec-currency.md` 7; architecture.md 3.5). The
helper's home is the first-code decision of N-019 (`internal/measure`, `internal/reduce`
or a shared `internal/checked`); this spec adds no third copy. A division is toward zero
on non-negative values and the remainder is carried or stated.

11.2. **The products**, with the bounds of `bounds.go`:

| Quantity | Largest numerator | Fits |
|----------|-------------------|------|
| A fixed tax or a payout's rate, `w*Amount*S + carry` | `MaxWeight * MaxFlowAmount * MaxFlowScale = 10^6 * 10^12 * 10^5 = 10^23` | 128 bits (as MON 6.4) |
| A rate tax, `base*RateBP*S + carry` | `MaxMoney * 10^4 * 10^5`, about `10^27` | 128 bits |
| A duty, `base*RateBP*S + carry` | `base` is at most one order per commodity and side, each value checked into `int64` by the clearing (`spec-clearing.md` 3.2, `overflow`), so below `(the number of commodities) * 2^63`; times `10^4 * 10^5`: below `2^103` | 128 bits; no overflow path |
| A toll, `Amount*S*max(1,n)` | `10^12 * 10^5 * n` | 128 bits; the sum over crossings and tolls is checked into `int64` (`overflow`) |
| A pro rata share, `due*A` | `MaxMoney^2`, about `10^36` | 128 bits |
| A limit, `ref*LimitBP` | `MaxPrice * MaxLimitBP` | `int64` |

11.3. **Rounding rules** (rule constants, architecture.md 9.6): every rate is truncated
toward zero and its remainder carried (a tax, a duty, a payout, a standing order's
quantity), except the **toll** (one truncation per crossing, no carry, 7.3), the **limit
price** (truncated, then clamped to 1 and `MaxPrice`, 9.2) and the **pro rata share**
(floor with the largest remainder, 8.4). A change of any is a behaviour change
(architecture.md 6.6).

11.4. **Bounds** (placeholders, provisional until the slice runs; section 14): the module
adds `MaxFiscalFlows`, `MaxDutyBP` and `MaxLimitBP`; it reuses `MaxFlowAmount`,
`MaxFlowRateBP`, `MaxFlowScale`, `MaxMoney`, `MaxWeight`, `MaxQuantity` and `MaxPrice`.
No cumulative total is kept (a total over a long run would need the bound `MaxFlowTotal`
of MON; a consumer sums the events), so nothing in the module can overflow in time.

## 12. Income and the welfare ratio

12.1. **The edit to WEL 2.1 and 4.3.** The welfare ratio is a function of the agent's own
income over a window (`spec-welfare-and-wages.md` 2.1). WEL owns the letters and the record
(2.1.9): this module adds two terms to the signed unearned sum `G_t`, which is inside `income_t`:

```
G_t = (grants) + P_t + I^in_t + N_t + Q_t - X_t        (WEL 2.1.9; FIS owns P_t and X_t)
```

where `X_t` is the money the agent paid in tick `t` as a tax, a duty or a toll (the sum
of `Applied` of the flows whose payer it was), and `P_t` what it received from a payout
that is not `NotIncome`, like `F_t` (relief) and `G_t` (a MON grant): in money, or, for a
payout in kind, the units at the value WEL 6.2 gives an in-kind receipt in the agent's
region. `P_t` is an **unearned receipt** and `X_t` a **charge**; neither is a return to
working. The record of WEL 2.1 already holds the unearned part of `income_t` in its
field `G_t` (signed), which the earned ratio of WEL 2.1.7 keeps in and the observed wage
of WEL 4.3 takes out. This module stores `P_t - X_t` **in that field**: the stored
`G_t` is the MON grant plus `P_t` less `X_t`. So the observed wage
`sum(income_t - F_t - G_t - K^own_t) / ...` is **gross of every fiscal flow**: a payout
is not a wage, and a tax, a duty or a toll does not lower the pooled regional wage that
the cost anchor of `spec-production.md` 9.3 reads, so a tax reaches the belief cost band
by no channel (FIS-A6). A payout in kind is **not** `K_t` (WEL 6 is unchanged: `K_t` is
the receipt of an `InKind` share of earnings and a trainee's board).

12.2. **Why netted, and why kept out of the wage.** The target-income rule and the
relief threshold should see what a household has left to live on. MON left a levy
un-netted for lack of a ruling and handed the question to this spec (MON position 10 and
open question 5): a household taxed 20% that looked as rich as before would not work more
and would not be relieved. A tax paid to a treasury is netted; a **MON levy** (a sink)
stays un-netted until WEL rules on it too (a conflict logged, section 22): the two are
different things (a levy is a holding charge or a sink; a tax is a public receipt) and a
pack that wants both netted asks WEL for the one rule. The observed wage is the return to
market labour; a statutory charge is not a cost of the firm that hires, so the cost
anchor must not see it (a lower "wage" would lower a firm's cost belief and its asking
price, an incidence the engine does not model, position 10). A pack that wants the net
wage in the anchor needs a WEL decision (open question 1).

12.3. **Where it is kept.** The two sums are scratch of the agent for the tick, written at
P1.7b, P3.2(c) and P4.1a and read at P6.3, where `P_t - X_t` is added to the `G_t` of the
record that is pushed; they are cleared at P8.6 and not saved. The welfare ring gains no
field and no meaning but the one sentence of WEL 2.1 (the stored `G_t` is every unearned
receipt less every statutory charge): that is the WEL edit (FR-FIS-041).

12.4. **What it does.** The target-income rule has labour supply unit-elastic in the net
wage (`spec-stability.md` 6.1): a tax on sales at rate `tau` raises the days worked by
`1 / (1 - tau)` for the same target, which lowers the gross wage if demand is inelastic
(FIS-A3). It changes the level and not the stability condition `e > 1` of 6.1 there, and
a tax **cannot** raise a staple's price by cutting labour supply (the backward bend goes
the other way). The relief threshold sees the disposable ratio: a household taxed below
`ReliefBelow` is relieved by the pool, and the pool is funded by the dead and not by the
treasury (4.4).

## 13. The lever, and what a policy actor will do

13.1. **The one slot.** The flow scale of every fiscal flow is set by the existing call
`SetFlowScales([]FlowScale{Flow; BP})` of `spec-integration.md` 5.1, between steps,
atomic, with the event `MoneyFlowScaled` for each changed flow (the name stays: a fiscal
flow is a money flow in this sense, and a purchase's scale is the exception that is not
money). The row of INT's table is edited: **Needs** "money supply or fiscal", **Owner**
"MON 4.4.3, 10.4; FIS 3.1.3", **Read by the tick at** "P1.7, P1.7b, P3.2(c), P3.3, P3.4a,
P4.1a", **Saved by** "MON 14, FIS 17, by flow ID". The errors are `ErrModuleOff` when
neither module is on, `ErrUnknownFlow` for an ID that is neither a money flow nor a
fiscal flow, `ErrOutOfBounds` and `ErrDuplicateID` (`spec-integration.md` 5.1.3). The new
value applies from the next position that reads it (FR-FIS-043).

13.2. **What a scale multiplies** (3.1.3): a tax's `Amount` and `RateBP`; a duty's
`RateBP`; a toll's `Amount`; a payout's `Amount` and `Level`; a purchase's and a
release's `Quantity`. It never multiplies an exemption, a hold, a reserve, a limit
share or a window. `0` pauses a flow: **a flow whose scale is 0 is skipped whole, of
every kind, before its base, its due or its order is built** (a duty's `RateBP`, a toll's
`Amount` and a standing order's `Quantity` included): it changes no carry, emits no
event and counts nothing, an exempt trade of a duty included (FR-FIS-001, FIS-S1). A
flow at a scale above 0 whose due truncates to 0 is not paused: its carry advances and
the events of 16.1.1 apply.

13.3. **What there is not.** No call sets a rate directly (the rate is `RateBP` times the
scale, so a consumer's schedule of rates is a schedule of scales); no per-region scale (a
flow per region is the data form; a region row would be a table of `spec-regions.md` 5.1
and would add a mechanism for a rate that the scale already is, open question 7); no
timer (INT 5.2.2); no action inside a step (`spec-world.md` 8.1). Everything else a
authority does over time is **policy** (a rule that reads measures, a fixed schedule, a
player) and is `spec-policy.md`'s (a lever of kind `flow`, 6.3 there): it reads IND measures and the snapshot, and writes
the flow scale (at P0.3a, in the internal form of `SetFlowScales`; the public call stays between ticks).
FIS adds no extension interface and no change to the table of architecture.md 4.3.

13.4. **What a reactive rule on a scale is.** A policy that raises a duty when revenue
falls, or releases the buffer store when the price is high, is a controller: a loop with a
gain and a delay, closed through the economy by a consumer's code or by PLC. The engine
guarantees conservation and determinism and not its stability (`spec-integration.md` 3.6,
N-131; ADR 0001 consequences). PLC registers each such rule as a loop (`spec-policy.md` 15, PLC-A1; FIS-A2 and A4 are
its neighbours).

## 14. Parameters

All are fields of `Config` under the module entry `Modules.Fiscal` (a pointer; nil is
off, `spec-config.md` 3.2), each with an owner (this spec), a bound and a doc line
(architecture.md 9.5 to 9.7), and each has a row in the parameter-owner table of
`README.md` (handoff). The module has **no default macro constant**: it has no flows
unless the pack lists them, and the zero value of a field means "absent" (N-027). The
bounds are in `bounds.go` (architecture.md 9.6).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Fiscal` | the module; present means on | nil (off) | needs `Modules.Polity` with a governed jurisdiction |
| Fiscal | `Taxes`, `Duties`, `Tolls`, `Payouts`, `Purchases`, `Releases` | the flows of each kind, sorted by ID | none | all kinds together at most `MaxFiscalFlows` |
| every flow | `ID` | the flow's name; the ledger reason of a tax, a duty, a toll and a payout | required | `spec-config.md` 3.3, unique across kinds (`ErrDuplicateID`); not a built-in reason or a MON flow (`ErrReasonCollision`) |
| every flow | `Authority` | the governed jurisdiction whose treasury is the counterparty | required | a governed jurisdiction |
| every flow | `Window.Start`, `.Until`, `.Seasons` | when it acts | 0, 0 (never ends), every season | `Until` 0 or above `Start`; seasons of the calendar |
| Tax | `Target` (`Kind`, `Region`, `Role`, `Jurisdiction`) | who pays (3.2) | required | known region, role and jurisdiction; `Region` covered |
| Tax | `Amount` | fixed: minor units per household (or account) per `Per` | 0 | 0 to `MaxFlowAmount` |
| Tax | `RateBP` | rate: basis points per `Per` of the balance above `Exempt` | 0 | 0 to `MaxFlowRateBP` |
| Tax, Payout | `Per` | the period: `tick` or `year` | `tick` | one of the two |
| Tax | `Exempt` | minor units per household (or account) never collected | 0 | 0 to `MaxFlowAmount` |
| Duty | `Region`, `Commodity` | the market it acts in; every one if empty | "" | covered region; known commodity |
| Duty | `Payer` | `seller` or `buyer` | required | one of the two |
| Duty | `Scope` | `domestic`, `import`, `export` or `any` | `any` | one of the four |
| Duty | `RateBP` | basis points of the value of a trade | required | 1 to `MaxDutyBP` |
| Toll | `On` | `exit` or `enter` | required | one of the two |
| Toll | `Mode` | the transport mode; every one if empty | "" | a configured mode |
| Toll | `Amount` | minor units per carrying unit per crossing | required | 1 to `MaxFlowAmount` |
| Payout | `Target` | who receives (3.2) | required | as a tax's, `Role` and `Region` covered |
| Payout | `Commodity` | "" is money; else in kind | "" | known commodity |
| Payout | `Amount`, `Level` | a rate, or a top-up (money only) | 0 | 0 to `MaxFlowAmount` |
| Payout | `NotIncome` | not income for the welfare ratio | false | households target only |
| Purchase, Release | `Commodity` | the good | required | known commodity |
| Purchase, Release | `Quantity` | units a tick at most | required | 1 to `MaxQuantity` |
| Purchase, Release | `LimitBP` | the limit as basis points of the reference price | required | 1 to `MaxLimitBP` |
| Purchase | `Hold` | stop buying at this many units held | 0 (no cap) | 0 to `MaxQuantity` |
| Release | `Reserve` | units never released | 0 | 0 to `MaxQuantity` |
| Runtime | `Scale` | the flow scale in basis points (3.1.3); state, not configuration | 10 000 | 0 to `MaxFlowScale` |

Presentation-only, so exempt from the registry test and not hashed
(`spec-config.md` 8.3): `Label` of every flow, empty by default, at most `MaxLabelBytes`.

**Bounds** (placeholders, N-015): `MaxFiscalFlows` 64 (as `MaxMoneyFlows`: the step loops
over the active flows and their classes each tick, so its cost is bounded by the
configuration); `MaxDutyBP` 10 000 (a configured rate of the whole value of a trade at most, at a scale of 10 000; a scale above 10 000 can lift the applied rate above it, 6.7);
`MaxLimitBP` 100 000 (ten times the reference price at most). The others are the world's.

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the order of
the passes of P1.7b (5.3, 8.4.1); the rounding rules (11.3); the key `K` and the order of
the cut (8.4.2); the pair rule of 9.4.1; the position of each step (10.1); the kind and
class constants of the events (16.1).

**Not parameters:** the scales (state), the carries (state), the treasury's holdings
(ledger), the cached tolls and the lists of trades (derived).

## 15. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). An error names the
path, for example `modules.fiscal.taxes.poll.target.region` (`spec-config.md` 7.1.1).
`Validate` runs these after the polity's.

### 15.1 Errors

The rules run in this order, so that the first error is the same for the same mistake:
the module's dependencies; counts; IDs (syntax, duplicates, reasons); authority; windows;
kind-specific shapes; classes.

| Sentinel | Rejects |
|----------|---------|
| `ErrModuleDependency` | `Modules.Fiscal` while `Modules.Polity` is nil or has no governed jurisdiction; a pool target while the estates module is off; `import` or `export` scope while the external module is off; a toll while the transport module is off |
| `ErrTooMany` | more than `MaxFiscalFlows` flows of all kinds |
| `ErrInvalidID`, `ErrDuplicateID` | an ID that does not match `spec-config.md` 3.3, or that is repeated across the six kinds |
| `ErrReasonCollision` | an ID equal to a built-in reason, to the ID of a MON flow (its reason) or to a reason a later spec declares (MON 4.5.2) |
| `ErrUnknownJurisdiction` | an `Authority` or a target `Jurisdiction` that is not defined |
| `ErrNotGoverned` (new) | an `Authority` or a target treasury with no seat |
| `ErrNotCovered` (new) | a `Region` the authority does not cover; a target treasury that is not a proper descendant of the authority |
| `ErrUnknownRole`, `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrUnknownMode`, `ErrUnknownSeason` | a name that is not defined |
| `ErrOutOfBounds` | a field outside its bound of 14; a `Per`, `Payer`, `Scope`, `On` or target `Kind` that is not one of its values; `Until` not above `Start` when set |
| `ErrFiscalShape` (new) | a tax with both or neither of `Amount` and `RateBP`; `Exempt` on a payout; a payout with both or neither of `Amount` and `Level`, `Level` with `Per` `year` or with a commodity, `NotIncome` on a pool or treasury target or on an in-kind payout, an in-kind payout whose target is not households of the seat region; a pool target with a role; a treasury target with a role or region; `import` with `Payer` `seller` and `export` with `Payer` `buyer`; a purchase and a release of one commodity of one authority whose limit prices are not strictly ordered on the integers, `limit(Purchase) < limit(Release)` by 9.2 (not the basis points), or two purchases (or two releases) of one commodity of one authority; a `Hold` below `Quantity` when set; a `Reserve` above `Hold` of a purchase of the same commodity and authority when both are set |

The sentinels `ErrNotGoverned`, `ErrNotCovered` and `ErrFiscalShape` are root sentinels
that `errors.Is` matches (N-075 lists the mapping). Nothing at run time returns an error
from this module: there is no call (13).

### 15.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at
construction, at `Warn`, with the attributes `rule` and `path`, through the
"configuration warning" line of `spec-observability.md` 7.3 (row 1); the volume is
bounded by the configuration.

| Warning | Condition |
|---------|-----------|
| `no_fiscal_flows` | the module is on and has no flow |
| `treasury_unfunded` | an authority has a payout or a purchase and nothing can fund its treasury: no tax, duty or toll names it, no remittance targets it, no MON grant targets it, its `StartMoney` is 0 and no release sells for it |
| `treasury_idle` | an authority has a tax, a duty or a toll and no payout, no purchase, no remittance upward and no MON levy on it: what it collects never leaves (the hoard of STB-L31) |
| `tax_below_upkeep` | a tax on households whose `Exempt` is below one tick's upkeep cost of the targeted role at the reference prices of its regions (5.5) |
| `payout_pool_no_outlet` | a payout to a pool in a region where `PoolRelease` is 0 and no spawner funds from the pool (8.6) |
| `purchase_unbounded` | a purchase with `Hold` 0 for a commodity that no release, no in-kind payout and no recipe of the pack draws on: only the treasury's cash limits it (9.3) |
| `release_unsourced` | a release of a commodity for which the authority has no purchase: it has nothing to sell (a treasury is created with no goods) |
| `duty_stack_above_whole` | the rates of the duties that can match one trade of one payer sum above 10 000 basis points (6.7) |
| `toll_no_border` | a toll whose authority is in the `Exit` (or `Enter`) list of no edge of the graph for its mode: it can never apply |
| `fiscal_class_empty` | a household target whose class has no households in any targeted region at construction (as MON's) |

## 16. Observability

### 16.1 Events

All are plain data, no map, no pointer, no wall-clock time, fixed-point values only
(observability-and-logging.md 4.1), emitted after the transitions they describe and only
when an observer was supplied (4.5 there). Both are aggregates, bounded by flows,
regions and authorities in the configuration; none is per agent, order or trade
(N-126: a row of `spec-observability.md` 4.1 each, owed). Types as `spec-observability.md`
4.1: a region is a `RegionID`, an authority a `JurisdictionID`, a commodity a
`CommodityID` (`NoCommodity` for money), a flow its string ID.

| Event | Position | Class | Gate | Fields (besides `Tick`) |
|-------|----------|-------|------|-------------------------|
| `FiscalFlowApplied` | P1.7b (taxes, remittances, payouts), P3.6c (tolls), P4.1a (duties); in the order of 10.1, per flow and region ascending | A | fiscal | `Flow string`, `Kind` (`tax`, `duty`, `toll`, `payout`), `Authority`, `Region` (the payer's for a tax, duty or toll; the recipient's for a payout), `Class` (`households`, `pool`, `treasury`: the payer's or the recipient's), `Commodity`, `Due`, `Applied` (money, or units for an in-kind payout), `Parties` (households, as the sum of weights, or accounts with a due), `Short`, `Cut`, `Headroom` (parties, in the same unit), `Exempt` (duty: trades whose payer was a pseudo-agent) |
| `TreasuryOrdered` | P3.4a, once per authority after its orders | A | fiscal | `Authority`, `Region` (the seat), `Bids`, `Asks` (orders placed), `WantedBuy`, `PlacedBuy`, `WantedSell`, `PlacedSell` (units), `CutCash`, `CutHold`, `CutStock` (units cut by 9.3.2) |
| `ShipmentDispatched` (existing) | P3.2(c), P3.3 | per shipment | transport | gains `Toll Money`: the total toll of the shipment, 0 when none |

16.1.1. **`FiscalFlowApplied` is emitted** once per (flow, region) per tick in which the
flow was at a scale above 0 and had a due, or, for a duty, an exempt trade; never when
nothing was due and never for a flow at scale 0 (13.2). It is the
**counter of every silent rule of this module** (FR-STB-013, N-045): a payer that found
too little (`Short`), a recipient cut by an empty treasury (`Cut`), a collection cut at
`MaxMoney` (`Headroom`) and a trade no duty could touch (`Exempt`) are all in it, so a
dampening that acts is visible. `Due` less `Applied` is what did not move.

16.1.2. **`TreasuryOrdered` is the counter of the standing orders' clamps**: the wish
(`WantedBuy`, `WantedSell`) against what was placed, with the three caps of 9.3.2 each
counted. It is emitted when a standing order of the authority was in its window and its
wish was at least 1; never otherwise. The orders themselves are `OrderPlaced` and the
trades `TradeExecuted` with the treasury's agent ID, which the snapshot gives
(`spec-polity.md` 10.5).

16.1.3. **Reconciling item.** `FiscalFlowApplied` is the event IND reads for `FiscalIn_r`
and `FiscalOut_r` (4.3.2); the seat region is read from the snapshot's `Jurisdictions`
rows. With it, FR-IND-019 (the regional money identity is exact) holds with a treasury
seated in another region.

### 16.2 Logs

Nothing at `Info`, `Warn` or `Error` from a step. At construction, `Info` once, in the
existing "world created" line: the attribute `fiscal_flows` (the count) joins the
summary and `fiscal` joins the `modules` list; the `Warn`s of 15.2 with `rule` and
`path`. At `Debug`, guarded by `Enabled`: "fiscal flow applied" (`flow`, `region`, `due`,
`applied`) and "treasury ordered" (`authority`, `region`). Constant lowercase messages,
typed attributes (observability-and-logging.md 3.3). **New log keys** for 3.3.2:
`authority` (a jurisdiction ID string) and `fiscal_flows` (a count, in the construction
summary); `flow`, `due`, `applied`, `region`, `rule` and `path` exist.

### 16.3 Snapshot

The module adds **no row of its own**: a flow's scale is in `Slots().FlowScales`, the
treasury's money is POL's (`RegionRow.MoneyTreasury`, `Snapshot.Jurisdictions`,
`spec-polity.md` 10.5). One field is new, and OBS names it: `MarketRow.HeldByTreasury`,
the units of the commodity held in the treasuries seated in the region, beside
`HeldByAgents` and `HeldByEstate` (a snapshot field is a function of saved state, which
the ledger holds, N-125).

### 16.4 Metrics

Fed from `FiscalFlowApplied` and `TreasuryOrdered` by the adapter, each recorded once per
tick per series (`spec-observability.md` 6.6). Every instrument has a unit and a
description; the attributes are bounded by the configuration (observability-and-logging.md 5.4.1).

| Name | Instrument | Unit | Attributes | Fed by |
|------|------------|------|------------|--------|
| `economysim.fiscal.collected` | Counter | `{minor_unit}` | `economysim.reason` (the flow ID) | `Applied` of kinds `tax`, `duty`, `toll` |
| `economysim.fiscal.paid` | Counter | `{minor_unit}` | `economysim.reason` | `Applied` of kind `payout`, money |
| `economysim.fiscal.in_kind` | Counter | `{unit}` | `economysim.reason`, `economysim.commodity` | `Applied` of kind `payout`, in kind |
| `economysim.fiscal.unapplied` | Counter | `{minor_unit}` | `economysim.reason` | `Due` less `Applied` of the **fiscal** flows that move money (kinds `tax`, `duty`, `toll` and money `payout`; not MON's flows) |
| `economysim.fiscal.order.cut` | Counter | `{unit}` | `economysim.clamp` (`cash`, `hold`, `stock`) | `CutCash`, `CutHold`, `CutStock` |

`economysim.clamp` is a new attribute with three constants. `economysim.reason` takes at
most `MaxFiscalFlows` more values. The existing gauges gain: `economysim.money.supply` the
sector value `treasury` (POL 10.6) and `economysim.market.stock` the holder `treasury`
(16.3). No metric needs a field the events and the snapshot lack.

### 16.5 What an observer can rebuild

`spec-observability.md` 4.3.3 gains item 12 (items 8 to 11 are LND's and CRD's) and changes **item 2**, because a trade
with a treasury moves money by `TradeExecuted`, not by `FiscalFlowApplied`:

- **Item 2 gains a third class of agent ID.** One of the `g` IDs that follow the
  counterparties' range (the `Agent` of a `JurisdictionRow`) is a **treasury** (`spec-polity.md` 5.2.2, FR-POL-016), whose seat
  region is that of its row in `WorldCreated.Jurisdictions` (`JurisdictionRow{Jurisdiction,
  Seat, Agent, Money}`, `spec-polity.md` 10.7). The money of a `TradeExecuted` whose
  buyer or seller is a treasury moves between the household sector of the trade's region
  and the **treasury sector** of the seat (the treasury is in no household sector); a
  trade between a treasury and a counterparty or an estate moves between that sector and
  the treasury sector in the same way. The treasury's side is never put in the household
  sector, and a treasury's trade needs no `FiscalFlowApplied`.
- **Item 12, money of `FiscalFlowApplied`.** For each `FiscalFlowApplied` of money:
  `Applied` from the household sector or pool of `Region` (by `Class`) to the treasury
  sector of the seat of `Authority`, for kinds `tax`, `duty` and `toll`; the reverse for a
  `payout` (from the seat's treasury sector to the sector of `Class` of `Region`). The
  units of an in-kind payout move between the treasury's holdings and the households of
  the seat region.

With the two the rebuild equals the snapshot (FR-FIS-047); the test of FR-FIS-047 runs a
pack with a standing order and a tax in different regions.

### 16.6 The Observability entry (observability-and-logging.md 9)

1. **Failure paths.** Configuration errors are returned (15.1). A shortfall, a cut or a
   clamp is survived and counted by an event; there is no `Warn` per tick, because a
   persistent shortfall would log every tick for every flow (the counterpart rule of 8.1
   there is met by the events).
2. **State transitions.** The two events above; the transfers are the ledger's and appear
   in the totals by reason. No event holds a map.
3. **Decision points.** The cut and the standing orders' caps: a `Debug` line with the
   inputs, and the counts in the events.
4. **New loop.** The per-tick loop over flows and classes logs only at `Debug`, guarded;
   events are built only with an observer; no `Warn` inside it.
5. **Changed events or metrics.** `ShipmentDispatched` gains a field (allowed); no event
   or metric is renamed.
6. **Metric attributes.** `economysim.reason` (bounded by flows), `economysim.clamp` (three
   constants), `economysim.commodity` (bounded by the configuration).
7. **Levels.** No `Info` in the loop, no `Error`.
8. **Logging an error that is returned.** None.
9. **Tests.** Section 8 of that document: recorded events, captured logs, defaults,
   non-interference, cardinality, emission order, the allocation test.

## 17. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). All plain data in a fixed order,
no map:

- **a section `fiscal`**, added to the inventory of `spec-checkpoint.md` 4.2 after the
  money-supply section, written only when the module is on. For each flow in canonical
  order (taxes, duties, payouts, purchases, releases, each by ID; a toll has no carry):
  - a tax, a duty or a payout with a rate shape (a top-up has no carry): the carries as
    `(account ID, carry)` pairs in ascending account ID, **for every open account on
    which the flow has acted** (it was in the flow's window and in its class at a tick:
    its carry has been advanced at least once), **zeros included**, the carry in the units
    of 5.2, 6.3 and 8.3, as MON 14 saves every carry of an open account. An account with
    **no pair** has not been acted on and gets the flow's **initial value** (5.2: `(ID
    mod P) * 10^4` for a fixed tax and a rate payout; 0 for a rate tax and a
    duty); this is why zero is written and not skipped: a carry that has come back to 0
    is not the initial value of a fixed tax;
  - a purchase or a release: one carry, `0 <= c < 10^4`, initial 0, always written;
- **the scale** of a flow whose scale is not 10 000, as a **sparse list** of `(flow
  position, scale)` of the section `fiscal` itself, owner this spec, in ascending
  position in the canonical order of the six lists (taxes, duties, tolls, payouts,
  purchases, releases, each by ID); a row equal to 10 000 is corrupt (`spec-checkpoint.md`
  4.1.4). The scales of MON's flows stay in the money-supply section (4.2 row 13 there
  is dense, per flow of the configuration of that module, and is absent with MON off,
  so a fiscal flow is not a row of it). `Slots().FlowScales` (`spec-integration.md` 5.3)
  is read from the two owners and presented in one flow-ID order;
- the treasury's money and goods are the ledger's (`spec-polity.md` 11).

**Not saved:** the dues, the lists of trades of the tick, the cached tolls, the per-agent
tax and payout sums of the tick (12.3), and the carry of a closed account (dropped with
the account). No per-agent field is hosted (N-128). The flows themselves are
configuration and are in the hash, the scale is not (ADR 0002 decision 3); the hash
covers `Modules.Fiscal` when it is present (an off module writes nothing: `spec-config.md` 8.2.1) and then its six lists in the
order of the `Fiscal` struct (taxes, duties, tolls, payouts, purchases, releases), each
by ID, every field but `Label`, windows with their seasons in the calendar's
order. A world restored between ticks continues exactly (FR-FIS-060).

## 18. Worked examples

Five settings, flagged as **examples**; names and numbers are placeholders, and the
tables are the output of `models/fis/worked.py` and `band.py` (section 20). The tree is
that of `spec-polity.md` 12.1: regions `capital` (0), `farm` (1), `harbour` (2), `hills`
(3), `mill` (4); jurisdictions `coast`, `crown`, `inland`, `uplands`; the crown (seat
`capital`, 50 000) and the inland province (seat `mill`, 5 000) governed.

### 18.1 Example A: a head tax, a holding tax, a remittance and a cut (tick 0)

```toml
[modules.fiscal.tax.poll]       # the crown: every worker household, 60 a tick
authority = "crown"
target = { kind = "households", role = "worker" }
amount = 60

[modules.fiscal.tax.tithe]      # inland: 100 bp a tick of what is above 2 000
authority = "inland"
target = { kind = "households", role = "worker" }
rate_bp = 100
exempt = 2000

[modules.fiscal.tax.remit]      # the crown takes a quarter of inland's treasury above 1 000
authority = "crown"
target = { kind = "treasury", jurisdiction = "inland" }
rate_bp = 2500
exempt = 1000

[modules.fiscal.payout.alms]    # inland pays 500 a household to the workers it covers
authority = "inland"
target = { kind = "households", role = "worker" }
amount = 500
```

30 worker households of weight 1 (capital 4, farm 10, harbour 6, hills 5, mill 5; agent
IDs 7 to 36 after five estates and two treasuries), each holding 3 000, except two in
`harbour` that hold 40 and 25. Passes of P1.7b in the order of 5.3:

| Step | Result |
|------|--------|
| `poll` | collected 240 (capital), 600 (farm), 305 (harbour), 300 (hills), 300 (mill) = 1 745; two households short, 20 and 35 forgiven (`Short` 2); crown 50 000 to 51 745 |
| `tithe` (inland covers farm, hills, mill) | each household has 2 940 left: base 940, `940*100*10^4 = 9.4*10^8`, due 9, carry `4*10^7`; collected 90 (farm), 45 (hills), 45 (mill) = 180; inland 5 000 to 5 180 |
| `remit` | `(5 180 - 1 000) * 2 500 / 10^4 = 1 045`; inland 4 135, crown 52 790 |
| `alms` | due `20 * 500 = 10 000` against 4 135: `q = floor(500 * 4 135 / 10 000) = 206`, remainder `7 500` for each, leftover `4 135 - 20*206 = 15`; 15 recipients receive 207 and the five with the largest keys `K(0, id)` receive 206: agents 13, 19, 31, 32 and 36; inland 0 |

Regions of the transfers (the reconciling item of 4.3.2): the `poll` moves 600 from
`farm`, 305 from `harbour`, 300 from `hills` and 300 from `mill` into the treasury of
`capital`; the `tithe` moves 90 and 45 from `farm` and `hills` into `mill` (the 45 of
`mill` is within it); the `remit` moves 1 045 from `mill` to `capital`; the `alms` pay
2 068 to `farm`, 1 034 to `hills` and 1 033 to `mill`, all from `mill`. The households
sector falls by 1 745 and 180 and rises by 4 135 (net +2 210); the treasury sector rises
by 2 790 at the crown (50 000 to 52 790) and falls by 5 000 at inland (5 000 to 0), net
-2 210: the sum of the sectors is unchanged and so is `M` (FR-FIS-002, FR-FIS-003).

### 18.2 Example B: tolls on a corridor

```toml
[modules.fiscal.toll.ferry]     # inland charges 4 a carrying unit on leaving it
authority = "inland"
on = "exit"
amount = 4

[modules.fiscal.toll.gate_in]   # and 2 on entering it
authority = "inland"
on = "enter"
amount = 2
```

The crossings of the examples (`spec-polity.md` 12.1, edges 5 `hills>mill` and 8
`mill>harbour`): `hills > harbour` is two edges, the first leaving `uplands` (a grouping:
no treasury, no toll), the second leaving `inland`, so `ferry` applies once: a hired
shipment of `n = 3` carrying units pays `4 * 3 = 12` to the treasury of `inland` (region
`mill`) from the shipper's account in `hills`, a transfer from one region to another.
`harbour > mill` leaves `coast` and enters `inland`: `gate_in` applies, `2 * 3 = 6`.
`farm > mill` is not a border (the same province): 0. `capital > mill` enters `inland`: 2
a unit. With a scale of 5 000 and `Amount` 3 on 3 units the toll is
`floor(3 * 5 000 * 3 / 10^4) = 4`, one truncation, not `3 * floor(1.5) = 3`.

### 18.3 Example C: a sales duty and its carry

```toml
[modules.fiscal.duty.sales]     # the crown: 3% of the value a seller sells, any good
authority = "crown"
payer = "seller"
scope = "domestic"
rate_bp = 300
```

Three trades in `harbour` in tick 0: seller A sells 10 units at 53 and 2 units at 410,
seller B 4 units at 52. Bases: A `530 + 820 = 1 350`, B 208. A: `1 350 * 300 * 10^4 =
4.05 * 10^9`, due 40, carry `5 * 10^7`; B: `6.24 * 10^8`, due 6, carry `2.4 * 10^7`. In tick 1 A
sells 100: `100 * 300 * 10^4 + 5 * 10^7 = 3.5 * 10^8`, due 3, carry `5 * 10^7`. Over the
two ticks A paid 43 of the exact `0.03 * 1 450 = 43.5`, with the half unit in its carry.

### 18.4 Example D: a tariff and an export duty at a gate

The league of `spec-polity.md` 12.2: two roots `east` (seat `ashford`) and `west` (seat
`brook`); `delta` is in `west` and is the gate of the node `far`.

```toml
[modules.fiscal.duty.tariff]    # west: 15% of the value of cloth bought from outside
authority = "west"
region = "delta"
commodity = "cloth"
payer = "buyer"
scope = "import"
rate_bp = 1500

[modules.fiscal.duty.export_levy]   # and 10% of grain sold outside
authority = "west"
region = "delta"
commodity = "grain"
payer = "seller"
scope = "export"
rate_bp = 1000
```

A counterparty sells 5 cloth at 200 to a household of `delta`: value 1 000, the buyer pays
150 (the buyer holds 400 after settlement and 250 after the duty). A household of `delta`
sells 20 grain at 55 to a counterparty: value 1 100, the seller pays 110. The treasury of
`west` is in `brook`, so the 260 is a transfer from region `delta` to region `brook`.
The counterparty paid nothing and its sweep at P4.2 is unchanged.

### 18.5 Example E: a buffer store (a pre-industrial setting; the words are data)

```toml
[modules.fiscal.purchase.harvest_buy]    # in the harvest season, bid at 80% of reference
authority = "crown"
commodity = "grain"
quantity = 500
limit_bp = 8000
hold = 6000
window = { seasons = ["harvest"] }

[modules.fiscal.release.lean_sell]       # in the lean season, ask at 120%
authority = "crown"
commodity = "grain"
quantity = 500
limit_bp = 12000
reserve = 500
window = { seasons = ["lean"] }

[modules.fiscal.payout.dole]             # and two units a household to the seat's workers
authority = "crown"
target = { kind = "households", region = "capital", role = "worker" }
commodity = "grain"
amount = 2
window = { seasons = ["lean"] }
```

With a reference price of 50 at `capital`, the bid is 40 and the ask 60. The toy market of
9.7 (`band.py`) shows what the two standing orders do and do not do: they remove 95 of
178 ticks outside the band under reversing shocks, and a persistent shortage of 120
ticks exhausts 3 000 units in 6 ticks of release. The `dole` takes oldest units first.

## 19. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9; fields as its section 3. The
register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn` numbers (N-103); entries
here are `FIS-An`, dampers `FIS-Dn` and scenarios `FIS-Sn`, and say what they merge with.
**Status is `model` where section 20 supports it and `hand` otherwise; nothing here claims
stability** (N-027). The checklist's 9.1 question, "what does each new rule read and what
does it change", has one answer for every flow: it reads the tick, its window, the payer's
own balance, weight and carry, the value of a trade, the crossing and the carrying units
of a shipment, the reference price of its seat and its own holdings; it changes balances
between sectors and, for a standing order, the market. Only a call to `SetFlowScales`
that reads a measure (a consumer's or PLC's) closes a loop through the economy, and that
is the caller's (13.4).

**Dampers this spec adds or specifies:**

| ID | Damper | Home | Status |
|----|--------|------|--------|
| FIS-D1 | Exemption: a tax never takes the balance below `w * Exempt` | FIS 5.2 | specified; warning `tax_below_upkeep` |
| FIS-D2 | Forgiveness, not arrears: what a payer cannot pay is dropped, so no claim, no debt loop | FIS 5.4 | specified (counted by `Short`) |
| FIS-D3 | Pro rata cut with the largest remainder and the keyed tie: the treasury never goes negative and no ID is first | FIS 8.4 | specified (counted by `Cut`); the rule of WLD D87 |
| FIS-D4 | Caps of a standing order: cash, `Hold`, `Reserve`: no overdraft, no unbounded stock | FIS 9.3 | specified (counted by `TreasuryOrdered`) |
| FIS-D5 | A closed set of rules: no flow reads an aggregate; the one lever is the scale | FIS 2, 13 | specified (a rule constant) |
| FIS-D6 | Spreading by a carry started from the account ID: no lump within a period | FIS 5.2, 6.3 | specified |

**FIS-A1. The tax that takes upkeep money (neighbour MON-A4, STB-L40).** *Kind:* hazard.
*Loop:* tax (+) money taken (-) upkeep bought (-) unfed households (+) privation (+) the
ladder: role change, relief, exit (-) the taxed base. B in the end, through hunger.
A payout cut by an empty treasury has the same effect on the recipients' side (an income
that stops). *Delay:* the privation count. *Owner:* FIS 5.5. *Damper:* FIS-D1; D11 (relief
from the pool); the payouts. *Undamped:* a fixed tax with no exemption on a class whose
income is no more than its upkeep. *Condition:* `Exempt` at least one tick's upkeep cost.
*Test:* FIS-S2. *Status:* hand.

**FIS-A2. The cash channel of a duty: fiscal drag and a Laffer-like revenue (neighbours
STB-L31, STB-L45, MON-A6).** *Kind:* loop. *Loop:* households' money (+) sales (+) the
duty base (+) duty collected (+) the treasury's balance (-) households' money. B: the
more is taken, the less is left to spend. With a treasury that pays nothing back, revenue
per tick is `tau * s * M * (1 - tau * s)^t` (a share `s` of money spent each tick, `M`
the money, `tau` the rate), so the rate that maximises revenue **at tick `t`** is
`1 / (s * (t + 1))`: 0.33 at tick 5, 0.095 at tick 20 and 0.02 at tick 100 for `s` 0.5
(`laffer.py`: 0.35, 0.10 and 0.05 on its grid of steps of 0.05); the steady revenue is 0.
With a treasury that pays back a share `r` of its balance each tick the steady state is
`M_h = M / (1 + tau * s / r)` and the revenue `tau * s * M / (1 + tau * s / r)`, which
**rises** with the rate to `r * M` and has no interior peak. *Delay:* one tick (the
proceeds are spendable at the next P3.2) plus the payout lag. *Owner:* FIS 6.3, 8.
*Damper:* FIS-D5 and the lever (13); the payouts; the warning `treasury_idle`.
*Undamped:* a treasury that does not spend: households' money falls geometrically at
`tau * s` a tick. *Condition:* the payout share `r` such that
`M / (1 + tau * s / r)` per household stays above the spending threshold of MON 8.5.
*Test:* FIS-S3. *Status:* model (the cash channel only: no labour, price or entry
response, and the one-sector money of the model).

**FIS-A3. A tax and the target-income rule (an amplifier of STB-L14, not a new loop).**
*Kind:* amplifier. A tax `tau` on sales lowers the net wage to `(1 - tau) * w`; the rule
works `D = K / ((1 - tau) * w)` days, so labour supply rises by `1 / (1 - tau)` for the
same target. For a staple with demand elasticity `e` below 1 the price then falls and the
stability condition `e > 1` of `spec-stability.md` 6.1 is unchanged: it is the level that
moves. *Owner:* FIS 12.4; WEL 5. *Damper:* D13 (the work floor) and the rest band bound
the days. *Undamped:* L14 for a staple. *Condition:* none new. *Test:* S04 with a duty on
the staple. *Status:* hand.

**FIS-A4. Buffer stock exhaustion and spoilage.** *Kind:* loop with a stock. *Loop:* price
below the bid (-) purchases (+) stock (+) the price (+); above the ask the same with the
release (-) stock. B for shocks that reverse within `stock / rate` ticks, and a **step**
when the stock or the cash is at zero: the price is then the market's, which in the
model of 9.7 is 5 above the ask limit or 5 below the bid limit (15 from the reference
price; the largest price of the iid run was 9 above the ask). *Delay:* `stock / release rate` (6 ticks there). *Owner:* FIS 9.
*Damper:* FIS-D4; the in-kind payout (it empties a buffer store oldest first, before the
shelf life ends). *Undamped:* a persistent shift; a perishable bought and not released.
*Condition:* shocks reverse within the defence time; units are released or paid out
before they spoil. *Test:* FIS-S4. *Status:* model (a toy linear market, 9.7).

**FIS-A5. A purchase price that attracts supply (neighbours STB-L36, L21).** *Kind:* loop
with a delay. *Loop:* a bid above the clearing price (+) the price producers see (+) the
margin (+) entry and production (+) supply (-) the price; and the cash that buys the
surplus (-) the purchases. B with the delay of the occupation review and of a job.
*Delay:* `ReviewTicks` and a job `Duration` (`spec-spawner.md` 6.2). *Owner:* FIS 9.
*Damper:* FIS-D4 (the hold and the cash end the purchases); D24 (the production halt),
the entry caps. *Undamped:* a `Hold` and a cash far above what the market can absorb: the
scheme then ends at exhaustion, with a step down in the price. *Condition:* the
purchases are small against the supply response, or the window is short. *Test:* FIS-S4.
*Status:* hand.

**FIS-A6. A wedge no agent sees: statutory incidence.** *Kind:* hazard. A duty falls on
the payer the flow names and on no one else (position 10); no belief, limit or order
includes it, so there is no shifting except through the payer's cash (FIS-A1, A2) and
income (A3). The one other place a charge could reach a price, the observed wage that
the cost anchor reads, is closed by 12.1: the wage is gross of every fiscal flow. A pack
that expects a price effect finds a cash effect. *Owner:* FIS 6.6, 12.1.
*Damper:* none by design (the data states who pays); `FiscalFlowApplied` measures it.
*Condition:* none. *Test:* FIS-S3 (the price of a taxed good over the run, report only).
*Status:* hand.

**FIS-A7. A toll is a wedge in a merchant's margin (neighbours STB-L84, L85, MER-A1).**
*Kind:* loop. *Loop:* toll (+) cost of a trip (-) margin (-) trips (-) the arbitrage that
closes the gap (+) the price gap (+) margin. B: the gap settles at freight plus toll
(the law of one price up to cost). *Delay:* the transit delay `k` of TRN 15. *Owner:* FIS
7. *Damper:* MER's own (it counts `Inbound`, caps its dispatch); the toll adds to its
margin through `Routes.Toll` and so does not change its gain condition. *Undamped:* a
merchant that does not read the toll makes trips that are refused
(`toll_unaffordable`) or lose money. *Condition:* MER subtracts the toll (the owed
edit). *Test:* FIS-S5. *Status:* hand.

**FIS-A8. A seasonal tax is a pulse (neighbour STB-L22).** *Kind:* hazard. A tax or a
standing order that acts in one season is met by every payer in the same ticks; the
carry spreads a period and not a window (3.1.2), so the money leaves households at once
and the cash falls together (a lockstep). *Owner:* FIS 3.1.2. *Damper:* none by design;
the exemption (D1); spreading is the pack's (a flow that acts all year with a smaller
rate). *Condition:* the window is long against the carry, or the exemption holds the
reserve. *Test:* FIS-S2 with a seasonal window. *Status:* hand.

**New scenarios proposed (the register owner numbers them).** All are accounting
scenarios of the class of S77 and S78 and outside the band rule of FR-STY-098 unless they
say report-only:

- **FIS-S1. Neutrality and the identities.** *Setup:* `five` with
  `overlay/fiscal-governed.toml` and every flow at scale 0, against the same without the
  module, seeds 1 to 10, 200 ticks. *Pass:* every event, snapshot field and golden line
  identical except the configuration hash, the names table, `Snapshot.Modules`, the
  construction summary and the module's warnings; with the flows at scale 10 000 the
  audit passes, `M` equals `M_init` plus the MON flows, the sector identity holds with a
  treasury term and every fiscal reason nets to zero (FR-FIS-001, 002, 003).
- **FIS-S2. A poll tax with and without an exemption.** *Setup:* a fixed tax on a role at
  a quarter of its upkeep cost; with `Exempt` 0 and with `Exempt` of one tick's upkeep;
  a payout of the same total back to the role. *Measure:* the unfed households, `Short`,
  the treasury's path. *Pass:* exact accounting (the sum of `Applied` equals the change of
  the treasury); with the exemption no household goes unfed because of the tax; report the
  rest (FIS-A1, A8).
- **FIS-S3. A duty and the cash channel.** *Setup:* a sales duty of 3% on every trade,
  with no payout, with a payout share of 5% and of 20% of the balance a tick. *Measure:*
  revenue per tick, `M_h / H`, the price of the taxed good. *Pass:* revenue equals the sum
  of `rate * value` of the trades to the carry, exactly; the identities hold; the rest is
  report-only (FIS-A2, A6).
- **FIS-S4. A buffer-store band.** *Setup:* the pack of 18.5 on the staple of `one` with a
  harvest shortfall (the yield shock of `spec-integration.md`) held for 120 ticks.
  *Measure:* the stock, the cash, the price and the units spoiled. *Pass:* the treasury
  never holds a negative balance, never holds more than `Hold`, the units bought, sold and
  spoiled equal the change of its stock; the price path is report-only (FIS-A4, A5).
- **FIS-S5. A toll on a corridor.** *Setup:* the trade overlay of `three` with a toll on
  the border of one jurisdiction, merchants with and without the toll in their margin.
  *Measure:* the tolls paid, the shipments refused for `toll_unaffordable`, the trade
  volume. *Pass:* the toll paid equals `Amount * n` per crossing summed over the
  dispatches, exactly; the refused count is reported (FIS-A7).

## 20. What was run

`models/fis/` (Docker, standard library only, `PYTHONDONTWRITEBYTECODE=1`, a read-only
mount; the commands are in its README). They are prototypes of the rules, not of the
library, and no default is claimed stable:

- `cut.py`: the pro rata cut of 8.4 on **20 000 random instances** (1 to 120 recipients,
  dues of 0 to 10^6, a treasury of 0 to the total plus 5; 19 841 were short). In every one
  the amounts applied add up to exactly `min(A, D)`, no recipient receives more than its
  due or less than the floor of its share, none is due and paid, and the result is the
  same for any listing order; the largest leftover was 94. Over 1 000 ticks with ten equal
  recipients and 5 extra units a tick each received 475 to 525 extra units. And the **carry
  of a duty** on 2 000 random runs (rates 1 to 10 000, scales 0 to 25 000, bases 0 to
  10^7): the total paid is `floor((sum + c_0) / 10^8)` exactly in every one.
- `laffer.py`: the cash channel of FIS-A2, exact rationals: revenue per tick against the
  rate at ticks 0, 5, 20 and 100 for a hoarding treasury, and the steady state for a
  payout share of 0, 5%, 20% and 100%, against its closed form.
- `band.py`: the standing bid and ask of 9.7 on a toy linear market (400 ticks of iid
  shocks, a persistent shortage, a persistent glut), integer arithmetic.
- `worked.py`: the arithmetic of the worked examples of section 18: the carries, the cut,
  the keys, the crossings and the tolls.

**What they do not show.** They check that the rules have the properties claimed and give
the numbers of the examples. They measure no cost, say nothing about a real economy, and
have no labour supply, no beliefs, no entry and no spoilage: FIS-A3, A5 and A8 are
`hand`. They are not evidence that any rate, amount or limit is stable.

## 21. The warm start and the steady state

21.1. **The money side is unchanged.** A tax, a duty, a toll and a money payout move
money between sectors and add no real demand or supply. The reference programme and the
warm table of quantities, jobs and relative seed prices (`spec-viability.md`) stay valid,
and a balanced pair is neutral on `M` (it is always, 4.1.1).

21.2. **The real side is not.** A purchase removes units from households' supply, a
release and an in-kind payout add to it, and the programme counts none of them: a pack
with a purchase programme starts away from its own steady state. The rule is MON 15.3's:
set `Window.Start` to the end of the warm-up, so that the flow acts on a world that has
settled (scenario S22, warm against cold start). The warning `purchase_in_warm_start` is
not added: the warm start is optional and the pack knows.

21.3. **The treasury's opening money** is POL's `StartMoney` and nothing else; there is no
`WarmStart.Pools` for a treasury. A treasury with a payout and nothing to fund it draws
`treasury_unfunded` (15.2).

21.4. **Seeded money.** `MoneyBufferTicks` seeds households with money before any tax
(`spec-viability.md` 8.3); a large poll tax makes the first ticks a transient of
households' cash (FIS-A1). A pack lowers the warm money by the tax or accepts it.

## 22. Notes settled, and what this spec needs from the others

### 22.1 Notes

| Note | What this spec does |
|------|---------------------|
| N-016 (the parameter registry) | **Applied.** Section 14 adds the rows; no unnamed parameter; the registry test is still to be written |
| N-017 (words and names) | **Applied.** Terms fixed: fiscal flow, authority, tax (head tax, holding tax, remittance), duty (sales tax, purchase tax, tariff), toll, payout, standing order (purchase, release), counterparty (of a trade), base, payer, payee, crossing (POL's), cut, pro rata, window (a flow's, not WEL's), scale, treasury sector. `levy` stays MON's. Setting words only in the flagged examples |
| N-021 (anything that creates or destroys money) | **Applied for FIS.** No faucet, no sink, no reason of kind faucet or sink, no boundary account: every reason is internal (2, 3.4, 4.1). The ruler's mint is a MON grant to a treasury (4.2). The goods side: a purchase and a release are `trade`, an in-kind payout is internal |
| N-027 (no default is stable) | **Applied.** No default macro constant; the models show properties of rules and no stability |
| N-045 (silent dampers) | **Applied.** `FiscalFlowApplied` and `TreasuryOrdered` count every clamp (16.1) |
| N-046 (register entries owed) | **Applied for FIS.** FIS-A1 to A8, dampers FIS-D1 to D6 and scenarios FIS-S1 to S5 (19); remove FIS from the areas of N-046. SAV, PLC stay |
| N-090 (a module that keeps an account in a region) | **Applied, answered by no registration.** A tax keeps no agent away or committed; a treasury's kind is not relocatable (`spec-polity.md` 5.3.2) |
| N-091 (the slots) | **Applied: none.** The flow scale is the existing slot (13) |
| N-103 (running writers in parallel) | **Followed, with one slip.** Only this spec, `models/fis/` and `.supervisor/fis.*` were created; every model ran in Docker with no bytecode; the host `python3 --version` was run once by reflex (it printed a version; it read and wrote nothing); in the revision after review the host `python3 -c 1` was run once more by reflex (it printed nothing and read and wrote nothing); no model was re-run or changed in the revision |
| N-104 (region-keyed tables) | **Applied: none.** A flow names a jurisdiction and optionally one region; a region-keyed rate is open question 7 |
| N-107 (new faucets and sinks start from MON) | **Applied.** The module starts from MON's mechanism (arithmetic, carry, reason, scale, "no aggregate") and uses internal reasons where the counterparty is a treasury, as the note's second sentence allows; a creating or destroying ruler is a MON target (4.2) |
| N-108 (what RES leaves to LND) | **Left open for LND.** A tax on rent or on titles is a tax on the holder role (5) or a reason-keyed tax (open question 3); LND's rent itself is a transfer |
| N-117 (the agent ID space) | **Applied: none.** The module adds no agent and no account kind; the treasuries are POL's |
| N-122 (first-come caps) | **Applied.** The one cap that binds, the treasury's balance, is cut pro rata (8.4); a payer's rule in flow ID order is within one payer (5.3.1) |
| N-124 (a paired comparison is exact while aligned) | **Read.** A flow that moves no count of draws keeps the streams aligned; a standing order adds orders and so keys: a comparison "treasury on, off" is over seeds, not paired (FIS-S1 is exact only with every flow at scale 0) |
| N-126 (a new event or `Warn` adds a row) | **Applied.** Two events and no runtime `Warn`; the rows are in the handoff |
| N-128 (a hosted per-agent field is saved once) | **Applied: none.** The module hosts no per-agent field (17) |
| N-130 (what IND cannot measure) | **Read.** The income terms `X_t` and `P_t` are inside `income_t` (12.3); IND's ask 1 (`IncomeLast`) covers them when it is adopted |
| N-131 (not in version 1) | **Respected and extended.** Not offered: a rate slot, a timer, a per-region scale, a tax on a reason, a treasury that hires, a public debt (24) |

### 22.2 What this spec needs from the other specs

Each row is an owed edit; the consolidated table with exact text is in the handoff
(Edits owed to other specs).

| File | Section | The change |
|------|---------|------------|
| `spec-ledger.md` | 3.1, 7.2, 9.3 | the account kind Treasury is in `trade`'s row; a declared internal reason per tax, duty, toll and payout (7.2); the sector identity has the treasury term (POL's, restated) |
| `spec-money-supply.md` | 4.1, 4.3, 7.2, 12.1 | a target kind `treasury` with the field `Jurisdiction`; 4.3.2's "the pool is where a grant to the treasury goes" reworded; `ErrNotGoverned`; the flow ID namespace is shared with FIS |
| `spec-polity.md` | 5.4.2, 6.3, 17 | the FIS row of 5.4.2 and H1 to H4, H12 marked used; open questions 5 and 6 answered |
| `spec-world.md` | 6.1.6, 6.3, 6.5, 6.5.1, 6.6, 6.7, 6.11, 7.3, 8.2 | the positions P1.7b, P3.4a and P4.1a, the reservation list, the input order of a market, ruling C21, the event order, a note in the draw table |
| `spec-transport.md` | 3.3, 5.4, 5.5.1, 9.1, 13.1 | the position "never moves money" excepted for a toll; row 22a `toll_unaffordable`; step 3a; `Routes.Toll`; `ShipmentDispatched.Toll` |
| `spec-clearing.md` | 3.1.1, 3.1.2, 3.2 | one order per agent, commodity, named lot and side; the treasury's pair; `side_not_allowed` names a treasury |
| `spec-welfare-and-wages.md` | 2.1, 4.3, 6.2 (and FR-WEL-002, 018, 040) | `P_t` and `X_t` are terms of the signed `G_t` of 2.1.9 (so of `income_t`: the MON grant plus `P_t` less `X_t`, with WEL's other terms), so the observed wage and the pooled wage the cost anchor reads are gross of every fiscal flow; a payout in kind is valued as 6.2 says and is not `K_t`; 6 itself is unchanged |
| `spec-production.md` | 9.3 | no text change: the cost anchor reads the pooled wage of WEL 4.3, which is gross of fiscal flows (12.1); the row exists so that the check is made |
| `spec-demand.md` | 10.2 (`class_without_income`) | "not raised for a role that a grant to households with `NotIncome` false targets" gains "or that a payout of `spec-fiscal.md` with `NotIncome` false targets" (a stipend to a role, 8.1) |
| `spec-integration.md` | 5.1, 5.1.3, 5.3 | the `SetFlowScales` row: needs, owner, read-by, saved-by (MON 14 and the `fiscal` section of FIS 17); the errors; `Slots().FlowScales` is read from both owners in one flow-ID order |
| `spec-observability.md` | 4.1, 4.2.3, 4.3.3, 5.3.5, 6.2, 6.5, 7.3, 9.4 | two events, one field, the rebuild: item 2 classes a treasury agent ID by the `Jurisdictions` rows of `WorldCreated` and item 12 for `FiscalFlowApplied` (16.5); `Kind` of `FiscalFlowApplied` is a third set beside those of `BoundaryFlow` and `EndowmentAdjusted` (`tax`, `duty`, `toll`, `payout`) and `Class` a set (`households`, `pool`, `treasury`), with the cardinality list of 9.4; `HeldByTreasury`; five metrics (`economysim.fiscal.in_kind` among them); the construction warnings |
| `spec-checkpoint.md` | 4.2, 4.3 | the section `fiscal` after the money-supply section: carries (every open account acted on, zeros included; absent = initial value) and the sparse list of non-default scales of fiscal flows (owner FIS); the list of derived state |
| `spec-external-sector.md` | 20 item 8, 6 | answered: a tariff is a FIS duty with `Scope` `import` or `export`; EXT unchanged |
| `spec-infrastructure.md` | 21 item 4 | a border toll per carrying unit is FIS 7; a toll on a facility's use stays open |
| `spec-land.md` | 5.1.2, 7.7 | "a tax on rent is not offered; a tax on the holder role is `spec-fiscal.md` 5" |
| `spec-estates.md` | 2.6 | one sentence: a tax on a pool collects at P1.7b after relief (5.6); the treasury is not the pool |
| `spec-stability.md` | 4, 4.8, 5.1, 8, 10 | merge FIS-A1 to A8, D1 to D6, S1 to S5 with numbers; drop the FIS hook row of 4.8; the "FIS, PLC" row of 10 loses FIS |
| `spec-reference-chain.md` | 9, 10 | overlays `fiscal.toml` and `fiscal-governed.toml`; scenarios FIS-S1 to S5 |
| `README.md`, `spec-roadmap.md`, `documentation_index.md`, `notes.md` | index, code and parameter-owner tables, glossary; FIS row; rows for `spec-fiscal.md` and `models/fis/`; the notes of 22.1 and the new notes (fresh `NOTES-TAG`) | the handoff's rows 1 to 4 |
| `spec-indicators.md` | 8.2, 9 | the terms `FiscalIn_r` and `FiscalOut_r`; the preset asks below |
| `spec-config.md` | 3.2, 3.7, 4.1, 7.3 to 7.6, 8 | the module row, the bounds, the sentinels and warnings, the hash |
| `spec-merchants.md` | 5, 7 | the margin subtracts `Routes.Toll` |
| `spec-storage.md` | 3.3.3, 10.2 | 3.3.3 lists a treasury among the pseudo-agents that are never limited; FIS's row of 10.2 points to 9.6 here; H9 stays reserved |
| `spec-calibration.md` | 6 | one registry row per tunable of 14, classified (handoff) |
| `rules/observability-and-logging.md` | 3.3.2, 4.3, 5.3, 5.4.1 | the keys `authority` and `fiscal_flows`, the two events, the five metrics, the attribute `economysim.clamp` (each change a fresh read-receipt tag, made by the supervisor) |
| `rules/architecture.md` | 2.6 | `fiscal.go` and `internal/fiscal/` |

**IND preset asks** (not adopted by this spec): `Revenue` (the sum of `Applied` of the
kinds `tax`, `duty` and `toll` by authority and by flow), `FiscalBalance` (revenue less
payouts and the change of the treasury's money), `Treasury` as a row of `Sectors`, and the
regional terms of 4.3.2; none needs an engine aggregate.

## 23. Requirements

IDs are stable (testing.md 4.1). A requirement is a single testable statement. The
numbers run in groups (001 to 013 the module and its validation, 020 to 026 taxes, 030 to
039 duties, tolls and payouts, 040 to 049 the lever, welfare and observability, 050 to 056
standing orders, 060 to 063 state and determinism), and a gap is not a removed requirement.

### Functional

| ID | Requirement |
|----|-------------|
| FR-FIS-001 | With `Modules.Fiscal` nil, no result, event, snapshot field or golden line of any world MAY differ from a world without the module except the configuration hash, the names table, `Snapshot.Modules`, the construction summary and the module's warnings; with the module on and every flow at scale 0 the same holds (1, 3.1). |
| FR-FIS-002 | Every reason of the module MUST be internal: no transfer of this module MAY be a faucet or a sink, no boundary account MAY be created for it, no account ID MAY shift because of it, and the money supply `M` MUST be the same before and after every fiscal step (3.4, 4.1.1). |
| FR-FIS-003 | In every tick `delta(households) + delta(public) + delta(contract) + delta(treasury)` MUST equal faucets less sinks, and the changes under each fiscal reason MUST sum to zero (4.1.1). |
| FR-FIS-004 | The IDs of the six lists and of MON's flows MUST be one namespace: an ID repeated among the six lists MUST be `ErrDuplicateID`, an ID equal to a built-in reason, to a MON flow or to a later-declared reason `ErrReasonCollision`, and more than `MaxFiscalFlows` flows `ErrTooMany` (3.1.1). |
| FR-FIS-005 | Each list MUST be sorted by ID before a world is built or a hash is taken; the hash MUST cover `Modules.Fiscal` when it is present (and nothing when it is off, `spec-config.md` 8.2.1): the six lists in the order of the `Fiscal` struct (17), every field except `Label`, windows with seasons in the calendar's order, and MUST NOT cover a scale (17). |
| FR-FIS-006 | A flow MUST act in tick `t` exactly when `t >= Start`, `Until` is 0 or `t < Until`, and `Seasons` is empty or contains the season of `t`; it MUST apply nothing during the construction phase (3.1.2). |
| FR-FIS-007 | An `Authority` or a target `Jurisdiction` that is undefined MUST be `ErrUnknownJurisdiction`, one that has no seat `ErrNotGoverned`, and a `Region` the authority does not cover, or a target treasury that is not a proper descendant of it, `ErrNotCovered` (15.1). |
| FR-FIS-008 | `Modules.Fiscal` MUST be rejected with `ErrModuleDependency` while `Modules.Polity` is nil or has no governed jurisdiction; a pool target while the estates module is off, `import` or `export` scope while the external module is off, and a toll while the transport module is off MUST be rejected the same way (3.1, 15.1). |
| FR-FIS-009 | A household target MUST select the Agent accounts whose agent has the role (any if empty) and region (every covered region if empty), external agents included and estate and treasury pseudo-agents never; a pool target the Estate account of the region; a treasury target the Treasury account of the named governed proper descendant, or of every one (3.2). |
| FR-FIS-010 | A flow's scale MUST default to 10 000, MUST change only by `SetFlowScales` between steps within 0 to `MaxFlowScale`, MUST apply from the next position that reads it, and a fiscal flow's ID MUST be accepted with the errors of 13.1 (3.1.3, 13.1). |
| FR-FIS-011 | A season name that the calendar does not define MUST be `ErrUnknownSeason`, and the season of a tick MUST be that of the moment of P1.1 (3.1.2). |
| FR-FIS-012 | Each shape rule of the `ErrFiscalShape` row of 15.1 MUST be rejected with that sentinel and the path of the field, and a field outside its bound with `ErrOutOfBounds` (14, 15.1). |
| FR-FIS-013 | The warnings of 15.2 MUST NOT fail validation and each MUST name its rule and path. |
| FR-FIS-020 | The due of a payer or recipient MUST depend only on its own weight, balance and carry, the flow's fields and scale, and the tick: not on any other account, a price or a total (5.2, 8.3). |
| FR-FIS-021 | A fixed tax MUST be `due = (w*Amount*S + carry) / (P*10^4)` collected up to `max(0, bal - w*Exempt)` with the carry started at `(ID mod P)*10^4`; a rate tax `due = (base*RateBP*S + carry) / (P*10^8)` collected up to `base`, `base = max(0, bal - w*Exempt)`, the carry started at 0; the carry MUST advance by the due and not by the amount collected (5.2). |
| FR-FIS-022 | At P1.7b the taxes on households and pools MUST be applied first in ascending flow ID, then the taxes on treasuries in ascending flow ID, then the payouts per authority in ascending `JurisdictionID`; regions ascending and accounts ascending within a flow (5.3). |
| FR-FIS-023 | A remittance MUST be computed on the treasury's balance after every tax on households and pools of the tick has been collected (5.3.1). |
| FR-FIS-024 | What a payer cannot pay MUST be forgiven and not carried, and the payers that paid less than the due MUST be counted in `Short` (5.4). |
| FR-FIS-025 | A collection that would take the treasury above `MaxMoney` MUST be cut to the headroom by the pro rata rule of 8.4 and counted in `Headroom` (5.3.2). |
| FR-FIS-026 | A tax on a pool MUST be collected at P1.7b, after the relief of P1.6 and before the entry step of P7.1 (5.6). |
| FR-FIS-030 | The base of a duty MUST be the sum, in 128 bits, of `price * quantity` of the trades of P4.1 in the markets of regions the authority covers (and `Region` and `Commodity` if set), whose scope matches and which the flow's window includes, summed per payer (6.2). |
| FR-FIS-031 | A pseudo-agent (an estate, a counterparty, a treasury) MUST never pay a duty, and a trade whose payer side is one MUST be counted in `Exempt` and be in no base (6.4). |
| FR-FIS-032 | The due of a duty MUST be `(base*RateBP*S + carry) / 10^8` with the carry advancing by the due, so that the total paid over any run is exact and independent of the order of the trades and of the duties (6.3). |
| FR-FIS-033 | Duties MUST be applied at P4.1a, after every market has settled and before the sweep of P4.2, in ascending flow ID and payers in ascending account ID (6.5). |
| FR-FIS-034 | A toll MUST apply to an edge of the shipment's shortest path whose crossing has the authority in `Exit` (`On` `exit`) or `Enter` (`On` `enter`), in its window and mode, with `due = floor(Amount*S*max(1,n)/10^4)` per crossing and no carry (7.2, 7.3). |
| FR-FIS-035 | A shipment whose total toll exceeds the shipper's available money MUST be refused whole with `toll_unaffordable` and change nothing (7.4). |
| FR-FIS-036 | A toll MUST be paid by the shipper to the treasury of the authority under the toll's reason at dispatch, in path order and ascending toll ID, and `ShipmentDispatched.Toll` MUST be the total (7.4). |
| FR-FIS-037 | A payout MUST have exactly one of `Amount` and `Level`, `Level` only for money with `Per` `tick`, an in-kind payout only for households of the seat region, and its due MUST follow 8.3 (8.2, 8.3). |
| FR-FIS-038 | When the dues of one authority and resource exceed what the treasury holds, each recipient MUST receive `floor(due*A/D)` and the leftover units one each in the order of descending remainder, ascending `K(t, id)`, ascending account ID; the whole of `A` MUST be used, no recipient MAY receive more than its due, and the recipients cut MUST be counted in `Cut` (8.4). |
| FR-FIS-039 | An in-kind payout MUST move units oldest lots first from the treasury's account within the seat region under its reason, creating and destroying nothing (8.5). |
| FR-FIS-040 | The engine MUST NOT change a flow's scale or any other field by itself, and no rule of the module MUST read a price, a money supply, an index, an income, a revenue or a count of accounts (2, 13.3). |
| FR-FIS-041 | `G_t`, and so `income_t`, MUST gain `+P_t - X_t` (12.1), the two sums MUST be per-agent scratch of the tick cleared at P8.6, `P_t - X_t` MUST be stored in the record's existing grant field `G_t`, and the welfare ring MUST gain no field (12.3). |
| FR-FIS-042 | Neither a payout (money or in kind) nor a tax, a duty or a toll MUST enter the observed wage or the pooled regional wage that the cost anchor reads, and a payout with `NotIncome` MUST NOT enter `income_t` (12.1, 12.2). |
| FR-FIS-043 | A scale set by a call MUST apply from the next position that reads it, and `MoneyFlowScaled` MUST be emitted per changed flow at the call (13.1). |
| FR-FIS-044 | `FiscalFlowApplied` MUST be emitted once per flow and region in which a due or an exempt trade occurred, in the order of 10.1, with the fields of 16.1, and MUST NOT be emitted when nothing was due or for a flow whose scale is 0, which MUST be skipped before its base is built whatever its kind (13.2, 16.1.1). |
| FR-FIS-045 | `TreasuryOrdered` MUST be emitted once per authority per tick in which a standing order was active with a wish of at least 1, with the counts of 16.1, and MUST NOT otherwise (16.1.2). |
| FR-FIS-046 | With `FiscalFlowApplied` and the seats of the snapshot, the money of every region MUST be explained exactly, treasury sector included (4.3.2). |
| FR-FIS-047 | An observer's rebuild of the money of the household sector, the pool and the treasury sector from the events MUST equal the snapshot, the money of a `TradeExecuted` with a treasury being classed to the treasury sector of its seat by the `Jurisdictions` rows of `WorldCreated` (item 2) and that of a `FiscalFlowApplied` by item 8 (16.5). |
| FR-FIS-048 | The snapshot MUST carry `HeldByTreasury` per market row, and nothing else of the module (16.3). |
| FR-FIS-049 | A step MUST NOT log at `Info`, `Warn` or `Error`; its `Debug` lines MUST be guarded by `Enabled`, have constant messages and use the keys `flow`, `region`, `due`, `applied` and `authority` (16.2). |
| FR-FIS-050 | A standing order's limit MUST be `min(MaxPrice, max(1, ref*LimitBP/10^4))` with `ref` the effective reference price of its seat region, and MUST read no market price (9.2). |
| FR-FIS-051 | A standing order's wish MUST be `(Quantity*S + carry)/10^4` with the carry advancing, capped for a purchase by the cash and by `Hold` and for a release by the unreserved stock less `Reserve`, each cut counted; an order MUST be placed only for a capped quantity of at least 1 (9.3). |
| FR-FIS-052 | A purchase and a release MUST trade under the reason `trade`, and the ledger's row for `trade` MUST connect the kind Treasury (3.4.2). |
| FR-FIS-053 | One authority MAY have a purchase and a release of one commodity only when the integer limit of the purchase is below the integer limit of the release (9.2 with the effective reference price of the seat region; equal limits are `ErrFiscalShape` whatever their basis points), so that a treasury never matches itself; the clearing MUST admit at most one order per agent, commodity, named lot and side (9.4.1). |
| FR-FIS-054 | A treasury's orders MUST come after the estates' asks and before the counterparties' in a market's input order, ascending by `JurisdictionID`, the bid before the ask; each MUST take one tie-break key at P4.1 and no draw at P5.4 (9.4.2, 9.5). |
| FR-FIS-055 | A treasury MUST place orders only in the market of its seat region (9.1.1). |
| FR-FIS-056 | A treasury's goods MUST age, spoil and wear as in any account, and a release and an in-kind payout MUST take the oldest lots first (9.6). |
| FR-FIS-060 | The state of 17 MUST be saved in the order given: every carry of an open account on which a flow has acted, zeros included, with an account that has no pair given the flow's initial value, and the scale of a flow whose scale is not 10 000 in the sparse list of the section `fiscal` whether or not the money-supply module is on; a restored world MUST apply the same payments and place the same orders as an uninterrupted one, a paused flow (scale 0) MUST stay paused, and a carry of a closed account MUST NOT be saved. |
| FR-FIS-061 | A restore MUST reject as corrupt a section whose carry names an account that is not open or that is of the wrong kind for its flow, whose pairs are not in ascending account ID, or whose scale list names a flow that does not exist, repeats one or holds a scale of 10 000 (17). |
| FR-FIS-062 | The module MUST draw no random number from the world's source and MUST keep no state in a closure or a package variable. |
| FR-FIS-063 | A treasury balance MUST never be negative, and no fiscal step MUST overdraw any account (4.1.2). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-FIS-001 | With the default logger and observer, a step of the reference chain MUST allocate nothing for the module beyond scratch held by the world and reused across ticks (observability-and-logging.md 8.2). |
| NFR-FIS-002 | The cost of a tick SHOULD be proportional to the active flows times the accounts of their classes and the trades of the tick, with `O(n log n)` only for a cut, and SHOULD have a benchmark with `b.ReportAllocs()`. |
| NFR-FIS-003 | Validation of the module SHOULD have a fuzz test with random flow tables (never panics; a valid table gives the same effective configuration and hash built in Go or read from a file; an invalid one returns a sentinel). |
| NFR-FIS-004 | Every output MUST be bit-identical across runs and platforms: no iteration over a map and no sort without a total order influences a result. |
| NFR-FIS-005 | Observation MUST NOT change a result: the same seed and configuration give identical results with and without a logger and observer. |

## 24. Open questions

1. **Netting a MON levy.** 12.2 nets a tax and leaves a MON levy un-netted, which is two
   rules for the same effect on a household. The wage the cost anchor reads is gross of
   both (12.1); whether a net wage should reach the anchor is the same decision. *Next
   step:* WEL decides one rule for both with this spec's evidence (FIS-A3).
2. **A treasury that hires or buys services.** A payout to a role is a stipend, not pay
   for work. A treasury as an employer needs EMP's hire matched in a round and a
   pseudo-agent that is a party to a contract. *Next step:* EMP, when a consumer asks.
3. **A tax on a reason.** A tax on rent, on wages or on interest needs a hook at the
   transfer of another spec's reason, which none has. A tax on the holder role (5) is the
   version 1 form. *Next step:* LND, EMP and CRD name the hook, if a consumer needs it.
4. **Wedge awareness.** The default strategy does not include a duty in its limit price,
   so incidence is statutory (position 10). A field in the view would let it. *Next step:*
   AGT, after a run shows how large the effect of statutory incidence is.
5. **A pay day.** A pulse of a season is a lockstep (FIS-A8); a `Phase` by account ID, as
   the occupation review has (`spec-spawner.md` 6.2), would spread a window. *Next step:*
   STY, if a band needs a seasonal tax.
6. **A toll by value, per shipment or per edge.** The form with no aggregate is per
   carrying unit per border (7.6). A value toll reads a price of the cargo at dispatch.
   *Next step:* when a consumer asks, with MER's margin.
7. **Region-keyed rates.** One flow per region is the data form; a table of rates keyed by
   region and jurisdiction with the combining modes of `spec-polity.md` 6.1 would be one
   flow with many rows. *Next step:* ORG and PLC decide what they need.
8. **A treasury as a holder in STO and LND.** 9.6 says the account is unlimited. When STO
   gives a room to a treasury, `Hold` is replaced by it. *Next step:* STO (H9).
9. **A reserve in a cut.** 8.4.3: the treasury spends all it holds when short. A reserve
   share would be a field. *Next step:* the first run that shows a treasury emptied and
   refilled each tick.
10. **A public debt.** A treasury that cannot go below zero cannot smooth a bad year.
    Borrowing is CRD's claim on a treasury as a debtor. *Next step:* CRD, with the
    estate waterfall for a jurisdiction that has no estate.

## 25. Where this lives in the code

Nothing is written; this is the proposed layout for the first code, checked against
architecture.md 2.6 and `AGENTS.md`.

- `fiscal.go` (root): the types `Fiscal`, `Window`, `Tax`, `Duty`, `Toll`, `Payout`,
  `Purchase`, `Release` and the extension of `FlowTarget`; the steps P1.7b, P3.4a and P4.1a
  and the toll of dispatch; the events `FiscalFlowApplied` and `TreasuryOrdered` (in
  `event.go`); the sentinels `ErrNotGoverned`, `ErrNotCovered` and `ErrFiscalShape` (in
  `errors.go`). One file per concept.
- `internal/fiscal/` (standard library only, primitive slices, no root type,
  architecture.md 2.9): the due of a tax and of a payout with the carry, the base and due
  of a duty, the toll of a crossing, the pro rata cut with the key, and the quantity and
  caps of a standing order, using the 128-bit helper of N-019's decision and not a third
  copy. The key `K` is `internal/ordering`'s. The root converts at the call, inline.
- `config/`: the tables `[modules.fiscal.tax.<id>]`, `[modules.fiscal.duty.<id>]`,
  `[modules.fiscal.toll.<id>]`, `[modules.fiscal.payout.<id>]`,
  `[modules.fiscal.purchase.<id>]` and `[modules.fiscal.release.<id>]` (the form of
  `[modules.polity.jurisdiction.<id>]`) and the rules of 15.
- `testdata/chains/overlay/fiscal.toml` and `fiscal-governed.toml` (a proposal for REF).
- Tests: table-driven unit tests in `internal/fiscal` with the cases of the models
  (the cut, the carry, the toll); an external `_test` of the neutrality of FR-FIS-001 over
  seeds; a selection test of the three target kinds (3.2, FR-FIS-009: role and region,
  external agents in, pseudo-agents out); a failing-case test for each sentinel of 15.1
  and each warning of 15.2; the
  identities of FR-FIS-002 and 003 over a range of seeds with `WithAudit`; a checkpoint
  round trip with a carry per kind; a fuzz test on the module's tables; an `AllocsPerRun`
  test for NFR-FIS-001; and an `Example` of the cut with Example A.
- Rule files, at the first code (owed rows, with a fresh tag each): architecture.md 2.6
  (`fiscal.go`, `internal/fiscal/`); observability-and-logging.md 3.3.2 (the keys), 4.3
  (the events and the field), 5.3 (the metrics) and 5.4.1 (`economysim.clamp`). The
  glossary words go to `doc.go` (N-017).

