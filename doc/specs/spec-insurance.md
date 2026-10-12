# Spec: Insurance: single-event cover of a cargo and of stored goods against the losses of the risk module, priced from the expected loss, paid from an underwriter's money with the unpaid part a credit claim

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-INS-NNN` and
`NFR-INS-NNN`, which tests cite (testing.md section 4). They are in section 21.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md),
decision 2), later stage (`vision.md` section 4, "Later"). It is off unless
`Config.Modules.Insurance` is present, and with it off nothing in this spec runs:
there is no policy, no premium, no payout, no position of 1.3a, 3.3c or 8.1b, no
event and no field in any other spec's record, and a loss of cargo or of stored
goods (`spec-risk.md`) is the whole of its effect, borne by whoever held the goods.
The core does not depend on it. It needs the risk module (`spec-risk.md`: the
losses it insures, the expected loss it prices from) and the credit module
(`spec-credit.md`: the claim that holds an unpaid payout, the shield, the arrears
and the waterfall). A route cover also needs the transport module
(`spec-transport.md`) and a store cover the storage module (`spec-storage.md`),
because the losses of risk rows exist only where those are on. It reads
`spec-ledger.md` (accounts, reasons, claims), `spec-money-supply.md` (why a
premium and a payout are not flows), `spec-agents.md` (the plan, the view, the
removal of 3.6), `spec-merchants.md` (the shipper and the contract), `spec-estates.md`
(what passes to the estate), `spec-welfare-and-wages.md` (income), `spec-world.md`
(the tick) and `spec-config.md` (the module switch and validation). Every change
it needs in another spec is listed in the handoff and in section 20 and is not made
here.

**Every number in this spec is a placeholder** except where a derivation is
shown. Nothing has been run in the library, bazaarBot has not been run, and no
acceptance band exists (notes N-015, N-027). The script of section 18
(`doc/specs/models/ins/cover.py`) is a stylised Docker Python model of the integer
rules and of two loops, not a run of code and not a claim that any default is
stable. No historical fact about insurance was read or used (N-018): every default
is structural (no insurance, a share of 10 000) or a placeholder. The settings of
the worked examples (section 19), a river valley with a wagon road and a granary,
are **examples**, flagged as such: `wagon`, `grain`, `cloth`, `bin`, `pit`,
`road` are data and nothing branches on them (architecture.md 9.1, N-017). The
normative text says **cover**, **policy**, **underwriter**, **insured**, **premium**
and **payout**; none of them is a setting word.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).
**Claim** is the word of `spec-ledger.md` 10 and `spec-credit.md` and keeps that
meaning: this spec never calls a demand for a payout a claim. The ledger claim
that holds the unpaid part of a payout is a **payout claim**.

## 1. Purpose and scope

`spec-risk.md` takes goods: a strike on a road destroys a share of a cargo at
its arrival, and a strike on a store destroys a share of what the owner holds.
Nothing in the library lets a household pass that loss on. This spec supplies the
one mechanism for it and nothing more. It covers:

- what a **policy** is, the two kinds, who is **insured** and what is covered
  (section 3);
- the **roles**: who underwrites, who insures, and their terms as data (section 4);
- the **request** and the **offer** a strategy appends to its plan and their
  admission (section 5);
- the **value** of a unit, the **premium**, and what the expected loss of a lane
  contributes (section 6);
- the **round** that forms policies (section 7);
- the **settlement** of a policy, the payout, the shield and the payout claim, and
  what a removal does (section 8);
- the ledger's part: two reasons, one operation, the identities (section 9);
- what a strategy sees and the default rules (section 10);
- welfare and the neighbouring specs (section 11);
- the positions in the tick, the draws (none) and the cost (section 12);
- parameters, validation, observability, saved state, the register entries, what was
  run, worked examples, notes settled, requirements, open questions and the home
  in the code (sections 13 to 23).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| What is lost, how likely, and how much: the strike, the severity, the `loss` sink | `spec-risk.md` (RSK). This spec reads its read `RouteLoss` and its events and never its draws or `LossBP` |
| A loan, interest, arrears, the waterfall at a removal | `spec-credit.md` (CRD). A payout claim uses all of it unchanged (8.4) |
| An insurer that is a company: shares, a charter, a reserve held apart from an owner's money, a licence | not in version 1: an underwriter is a household with money (position 2). A charter or a monopoly of cover is `spec-institutions.md` (ORG) |
| Reserves held in escrow, and reinsurance among underwriters | open questions 1 and 2. The ledger form of MER's escrow was compared and refused (3.5) |
| A premium that follows an underwriter's paid losses (experience rating), or an insured's own record | open question 3: it is state per agent and a feedback from an aggregate (position 10) |
| A state, a ruler or a guild that insures or compels cover | `spec-fiscal.md`, `spec-policy.md`; the seam is the levy and the grant of `spec-money-supply.md`, and a role block set by a pack |
| Cover of money, of a loan, of a title, of a facility, of a migrant's journey, of goods in a household's base room | not in version 1: RSK takes none of them (`spec-risk.md` 1, 7.4); there is nothing to insure |
| Cover of a risk that is not RSK's: a failed harvest, a death, a fire the pack writes as a levy | not in version 1: no event says what was lost |
| Payout in goods (replacement), or a salvage of the lost goods | open question 6: money is the one valuation the engine allows (3.5 table, `spec-goods.md` 7.4) |
| The decision to insure as a part of saving or of investment | `spec-saving-and-investment.md` (SAV); the request of 5.1 is what it appends |

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is cover? | A **policy**: a contract between one insured and one underwriter, formed at a round for **one event**: a route policy (one shipment) or one tick of a store. The insured pays a **premium** at once; the underwriter owes a **payout** if goods are lost | One event needs no term, no renewal, no cancellation and no state between ticks except a route policy in flight (16). A longer cover is a sequence of policies, each priced when it is made |
| 2 | Who insures? | The **households of a role with an `Underwrite` block**, each with its own money. No company, no pool, no reserve account, no separate entity | The brief and ADR 0003: no new entity. Households only is the least that works and adds no account kind, as CRD's lender is |
| 3 | What is covered? | The **loss** RSK reports: the units a strike took from a shipment's lines and from its traveller's account, and from an owner's stored goods. Nothing else: not spoilage, not wear, not diversion, not a lost sale, not money | RSK is the only source of a loss that the world fixes and tells (`spec-risk.md` 10.2.1). A cover that paid for spoilage would insure the market |
| 4 | How is a loss valued? | **Per unit, at the published price of the place of the goods, capped at a multiple of the reference price**, then a share (`CoverBP`) of it, up to the policy's **maximum payout** fixed at formation | The engine refuses a valuation of goods other than a market's (`spec-goods.md` 7.4). The cap breaks the loop "loss raises the price, raises the payout" (6.2, INS-A3). The share and the maximum are the moral-hazard dampers (INS-A1, INS-A5) |
| 5 | How is it priced? | A route cover: **the lane's expected loss** (`RouteLoss`) times the cover share times the underwriter's posted **loading**. A store cover: the underwriter's posted rate on the sum insured. No experience rating | RSK supplies the expected loss for exactly this (`spec-risk.md` 8, 10.2.1). A store has no expected-loss read (`spec-risk.md` 8.5), so its rate is data |
| 6 | Does a policy create money? | **No.** The premium is a transfer, a payout is a transfer, and an unpaid payout is a claim: three internal movements and no faucet, no sink, no boundary account | MON 3.4, 9.5; N-107: a premium that grows with a balance would be a faucet (it is not one: a premium is a fixed function of the cargo) |
| 7 | What if the underwriter cannot pay? | It pays what is above its **shield** now, pro rata over its policies of the position, and the rest is a **payout claim** that CRD holds, settles, bars on and passes to the estate | The brief's "a claim in CRD". The solvency of an insurer is then exactly the solvency of a debtor, with no second mechanism |
| 8 | Is cover reserved? | **No**: a policy takes no money from the underwriter at formation. Its capacity is capped by its own available money (`CapBP`, which a pack may set above 10 000: leverage is a choice) | A reserve in escrow is a second state and ties capacity to money exactly; compared and refused for version 1 (3.5) |
| 9 | Can the insured know a strike? | **No, and neither can the underwriter.** The cover is formed after the strike of a shipment is fixed (RSK 4.1), and this spec never reads `LossBP` or a draw. A test asserts it | `spec-risk.md` 10.2.2: an insurer that read it at dispatch would insure only what is lost |
| 10 | What is *not* modelled? | An aggregate that an underwriter reads to move its price: no experience rating, no market-wide loss ratio, no response to a lane's traffic | MON position 5, WLD 9.4: no rate from an aggregate. A consumer's strategy may do it from its own claims (10.4) |
| 11 | Randomness | **None.** Ties by the stateless key `K(t, id)` | WLD 7.4.2; CRD position 16 |
| 12 | Where is the decision? | In `Trade`: a strategy appends **cover requests** and **cover offers** to its plan, the world admits them as it does a delivery offer or a credit request | AGT 6.2; the precedent of MER and CRD |
| 13 | A slot, a call? | **None.** The premium is data of a role block; the loss is RSK's scales | INT is minimal (N-131). A pack that wants a different price changes a block, or a consumer's strategy |
| 14 | Weights, roles, relocation | A policy is the **agent's**, whatever its weight, role or region; no `region_bound` refusal | N-090: a route policy is bound by `open_shipment`, a store policy ends in its tick (9.4) |
| 15 | New loops | Yes: the recovery in a shipper's margin, the underwriter's capital against a common strike, the value following the price, thin capital, the premium as a cost, borrowing to insure (section 17) | `spec-stability.md` 9 |

## 3. The policy

### 3.1 Kinds

3.1.1. There are two kinds, and no other. Both are **single-event**.

| Kind | Insures | Formed | Ends | Settled at |
|------|---------|--------|------|------------|
| **route** (`route`) | the goods of **one shipment** dispatched in this tick by the insured: its lines when it has lines, and for a self-carried trip the traveller's own holdings except the mode's `Vehicle`, or both (6.3.2) | P3.3c of the tick of dispatch | at the arrival of its shipment, or at a removal (8.6) | P1.3a of the arrival tick |
| **store** | the **stored goods** of the insured in its region, for **this tick** | P3.3c | at P8.1b of the same tick | P8.1b |

3.1.2. **Why single-event.** A policy that lasted would need a term, a renewal, a
price that follows the risk (`RouteLoss` moves with every scale a consumer sets), a
cancellation and a rule for what a shipment in flight meant when it was bought. A
route policy ends with its shipment and a store policy with its tick, so the price is the
price of the moment and the only state is a route policy in flight. A household that wants
cover for a month asks every tick.

3.1.3. **Only goods dispatched now.** A route policy attaches to a shipment
dispatched in the tick it is formed (`Dispatched = t`): one already on the road is
not covered, whatever the insured asks. The reason is not information (nobody knows a
strike, position 9) but the shape: a cover that could be bought for a cargo in
flight would be a bet on a quantity the underwriter's `RouteLoss` no longer prices,
because the road's scale may have been set since (`spec-risk.md` 9.3: a shipment keeps
the `LossBP` it was given).

### 3.2 The insured

3.2.1. The **insured** of a route policy is the **shipper** of the shipment
(`Shipment.Shipper`, `spec-transport.md` 5.2): the agent that chose the lane and
paid the carriage, and in a delivery contract the deliverer, who bears the loss
(`spec-risk.md` 5.6). A consignee that is not the shipper has no policy on the
goods in flight: it did not choose the lane, and its loss is the shipper's choice of
what to consign. The insured of a store policy is the **owner** of the stores.

3.2.2. **An agent that is away** (a traveller on a self-carried trip) is the insured
of its trip and remains so: a policy is a transfer between accounts and needs no
agent present (CRD 9.4.3 is the same).

3.2.3. **An agent that owns a store does not travel** (`spec-storage.md` 9.2), so
one agent has either kind of exposure in a tick and never both at once, apart from a
hired shipment of a store owner, which is a route policy like any other.

### 3.3 The record

```go
// Policy is one cover. internal/insure: stdlib only, no root types; not the root
// `Policy` of PLC (`spec-policy.md` 4.7). Only a route policy is ever in the list
// between two ticks (16).
type Policy struct {
	ID         int64 // assigned in order of formation, never reused in a run
	Kind       int   // route or store
	Insured    int   // an agent ID
	Underwriter int  // an agent ID
	Shipment   int64 // the shipment ID of a route policy, -1 for a store
	Sum        int64 // the sum insured: the value at formation (6.3)
	MaxPayout  int64 // the most the policy pays: CoverBP of Sum, half to even (6.4)
	CoverBP    int64 // the share of a loss value it pays, fixed at formation
	Premium    int64 // what was paid, fixed at formation, for reports
	Formed     int64 // the tick of formation
}
```

3.3.1. A policy exists from formation until its **end** (8.5, 8.6). Nothing of it is
kept after: the cumulative totals of 9.3 are.

3.3.2. A route policy names its **shipment**, not a region: the region of the policy
is derived, as CRD's claim region is, from the shipment's origin (`Shipment.From`)
for a route policy and from the insured's account for a store.

### 3.4 What is not insured

3.4.1. Spoilage (the sinks `spoilage`, `wear`), goods diverted to an estate, goods
that did not arrive for want of a consignee that could hold them, goods consumed by
the traveller on the way, a delay, a lost sale, and money. A unit is insured only
if RSK's loss took it (`ShipmentArrived.Lost`, `AccountLost`; `StoreLoss.Lost`),
and the value of a unit is the one of 6.

3.4.2. A **self-carried trip to oneself with no lines** has no cargo as such: the
insured value is the traveller's holdings except the `Vehicle` (6.3.2), and the loss
is `AccountLost` (`spec-risk.md` 5.4.1). The vehicle is capital and is never insured
(position 3; `spec-risk.md` 5.4.1).

### 3.5 Ledger forms compared

The brief names two forms for the underwriter's obligation: a claim in the credit
list, and an escrow as in `spec-merchants.md` 9. Four mechanisms were compared.

| Mechanism | What it is | Decision and reason |
|-----------|-----------|---------------------|
| A. **Premium paid at formation; payout from the underwriter's money; the unpaid part a claim** | transfers and one claim of the credit list | **Chosen.** No new account kind, no second state of obligations (the credit list is the one), no reserve to keep true. Default, arrears and the waterfall are CRD's and are already written, tested and registered |
| B. **A reserve in escrow** | at formation the underwriter's maximum payout moves to an Escrow account of the policy; a payout is from it; the rest returns | **Refused for version 1.** It makes cover exactly funded: capacity equals money and there is no leverage, so an insurer can never fail and a "bust of all insurers" cannot exist; it adds a second money state (the escrow) and a case at every removal (to whom does the reserve go). It is open question 1, with `ReserveBP` as the shape |
| C. **A mutual fund account** | premiums pooled in an account that pays | **Refused.** A pooled account is an entity (a fund) with no owner, a new account kind and a rule for a deficit (a call on members); a company is position 2's exclusion |
| D. **Payout in goods** | the underwriter replaces the goods | **Refused.** A valuation of goods other than a market's, and a good that the underwriter does not hold (`spec-goods.md` 7.4; CRD 3.4 mechanism D) |

Mechanism A is also what makes the correlation of RSK visible as a **default**, the
way the brief asks: a common strike (`spec-risk.md` 3.3) calls every policy on its
lane at once, the underwriter pays above its shield and owes the rest as payout
claims, and an underwriter that cannot eat is removed and its claims are paid by the
waterfall of CRD 9.1 (INS-A2).

## 4. Roles, blocks and the module

### 4.1 The module

`Config.Modules.Insurance` is a pointer: nil is off, non-nil is on
(`spec-config.md` 3.2). Plain data, no maps:

```go
// Insurance is the insurance module. Its presence switches it on.
type Insurance struct {
	ValueCapBP       int64 // a unit's value is at most this share of its effective reference price (6.2)
	PayoutTicks      int64 // ticks from an unpaid payout's formation to its due tick (8.4)
	MinSum           int64 // the smallest sum insured of a policy, in minor units
	MaxPolicies      int64 // the most policies open at once (a route policy in flight and a tick's store policies)
	MaxCoverRequests int64 // the most cover requests in one plan
}
```

4.1.1. The module needs `Modules.Risk` and `Modules.Credit`; a request for a route
cover needs `Modules.Transport` and one for a store cover `Modules.Storage`. An
`Insure` block asks for route cover only with `MinLossBP` above 0, so a pack with
store rows and Storage and **no** Transport configures a store-only insured with
`MinLossBP` 0 and is not rejected (14.1). A
non-nil module with no role carrying a block is legal and does nothing (the warning
`no_cover_roles`, 14.2). It adds **no state to an agent** (N-128): everything it
keeps is its own section (16).

4.1.2. **Defaults.** `ValueCapBP` is required (no default: a pack that wants no cap
writes a number at the bound), `PayoutTicks` defaults to 1, `MinSum` and
`MaxPolicies` are required, `MaxCoverRequests` defaults to 4. There is no default
macro constant in this spec (N-027).

### 4.2 The role block

A role is a plain struct (`spec-agents.md` 3.3). This spec adds one field, a block
that is nil for a role that does not deal in cover, as `Merchant` and `Credit` are:

```go
// CoverTerms is the insurance block of a role: exactly one of Underwrite and Insure is set.
type CoverTerms struct {
	Underwrite *UnderwriteTerms // the households of the role post offers
	Insure     *InsureTerms     // the households of the role post requests
}

// UnderwriteTerms is what an underwriter posts and keeps.
type UnderwriteTerms struct {
	LoadingBP   int64 // route cover: the premium is this share of the expected payout; 10 000 is fair
	StoreRateBP int64 // store cover: the premium for one tick, in basis points of the sum insured
	CoverBP     int64 // the share of a loss value it pays, 1 to 10 000
	CapBP       int64 // its open liability is at most this share of its available money; may exceed 10 000
	ShareBP     int64 // the share of its free capacity it offers in one tick (10.3)
}

// InsureTerms is what an insured asks for and accepts.
type InsureTerms struct {
	MinLossBP     int64 // route cover: buys on a lane whose expected loss is at least this; 0 asks for no route cover
	RuinShareBP   int64 // route cover: buys when the cargo's value is at least this share of its money
	MaxLoadingBP  int64 // route cover: the highest loading it accepts
	MinCoverBP    int64 // the least share of a loss it accepts
	StoreShareBP  int64 // store cover: the share of its held value it insures each tick; 0 buys none
	MaxStoreRateBP int64 // store cover: the highest rate per tick it accepts
}
```

In a file: `[modules.insurance]`, `[role.<id>.cover.underwrite]` and
`[role.<id>.cover.insure]`, the form of `[role.<id>.credit.lend]` (N-102), so that
layers merge by key (`spec-config.md` 6.4).

4.2.1. **Exactly one block.** A `CoverTerms` with both or with neither is
`ErrCoverShape`. The reason is position 2 and CRD's 4.2.1: an underwriter that also
insures is a household that lays off its own risk on another, which is
reinsurance, a chain whose failure is the worst case of INS-A2; it is open question 2.
A role may carry a credit block and a cover block (a merchant that borrows and
insures): they are different decisions with different cash.

### 4.3 What reads which field

| Field | Read by | Why there |
|-------|---------|-----------|
| `Insurance.*` | the world | every policy obeys them whatever the strategy |
| `UnderwriteTerms.CapBP` | the world, at the admission of an offer (5.2) | an insured relies on the cap, so a strategy cannot lift it |
| `UnderwriteTerms.CoverBP`, `LoadingBP`, `StoreRateBP` | the world, at formation (6.4), and the **default** strategy for the offer | the terms of a policy are what the offer says; a consumer's strategy posts its own inside 5.2 |
| `UnderwriteTerms.ShareBP` | the **default** strategy (10.3) | how much to offer is its decision |
| `InsureTerms.*` | the **default** strategy (10.2) | what to ask for and accept is its decision |

A consumer's own `Strategy` receives the block in its view (10.1).

## 5. Requests and offers

### 5.1 The plan fields

`TradePlan` (`spec-agents.md` 6.3) gains two fields, in the way of CRD's `Borrow`
and `Lend` (`spec-credit.md` 5.1). Both last one tick, as an order does.

```go
// LaneCover asks for a route policy on every shipment of the agent dispatched this tick
// in a mode to a destination. It names no shipment: the round finds them (7.2).
type LaneCover struct {
	Mode         ModeID
	To           RegionID
	MinCoverBP   int64 // the least share of a loss value it accepts, 1 to 10 000
	MaxLoadingBP int64 // the highest loading it accepts, 0 to MaxLoadingBound
}

// StoreCover asks for a store policy on this tick: a share of the value of the goods
// the agent holds, which the world computes (6.3.3).
type StoreCover struct {
	ShareBP        int64 // the share of its held value to insure, 1 to 10 000
	MinCoverBP     int64
	MaxStoreRateBP int64 // the highest rate per tick it accepts, 0 to MaxStoreRateBound
}

// CoverOffer offers capacity at posted terms. Both numbers of capacity are money.
type CoverOffer struct {
	MaxLiability int64 // the most it will add to its open liability this tick, at least MinSum
	LoadingBP    int64 // route cover, 0 to MaxLoadingBound
	StoreRateBP  int64 // store cover, 0 to MaxStoreRateBound
	CoverBP      int64 // 1 to 10 000
}

// TradePlan gains:
//	Cover []LaneCover    // at most MaxCoverRequests, one per lane
//	Store []StoreCover   // at most one
//	Offer []CoverOffer   // at most one
```

5.1.1. A request names **no underwriter** and an offer no insured: the round
chooses (7.3). With the module off a request or an offer is refused `insurance_off`
and costs nothing else. A plan that uses none of the fields is unchanged.

5.1.2. **A request is for a lane, not for a shipment**, because a contract's
shipment is formed after every plan (`spec-world.md` 6.5, P3.3) and a strategy cannot
name what does not exist. The strategy appends a lane for each lane on which it has
planned a shipment or posted a delivery offer; the round covers whichever
shipments of that lane exist. A lane on which nothing was dispatched is a request
with no candidate and costs nothing (counted, `NoShipment`, 15.1).

### 5.2 Admission

At the start of the round (7.1) the world visits the requests and the offers in
agent-ID order, requests of an agent in plan order, and admits each or refuses it
**whole** with a `reason` (exported constants of the root package, `event.go`; a
refusal changes nothing, is counted in `CoverRound` (except `insurance_off`, a
`Debug` line only) and is not an error returned by `Step`). The first failing check
in this order is the reason:

| Of | # | `reason` | Condition |
|----|---|----------|-----------|
| all | 1 | `insurance_off` | `Modules.Insurance` is absent |
| request | 2 | `not_insured_role` | the agent's role has no `Insure` block |
| offer | 2 | `not_underwriter` | the agent's role has no `Underwrite` block |
| request | 3 | `too_many_requests` | this is request number `MaxCoverRequests + 1` or later of the plan, lanes and the store request together |
| request | 4 | `invalid_cover` | `MinCoverBP` outside 1 to 10 000, a limit outside its bound, `ShareBP` outside 1 to 10 000 |
| offer | 4 | `invalid_offer` | `MaxLiability` below `MinSum`, `CoverBP` outside 1 to 10 000, a rate outside its bound |
| lane | 5 | `unknown_lane` | the mode is not a configured mode, `To` is not a region, is the agent's own region, or is an outside node |
| store | 5 | `no_store` | the agent owns no store (`spec-storage.md` 8.3), or `Modules.Storage` is absent |
| all | 6 | `duplicate` | the agent already has an admitted request of the same lane (the same store request, an offer) this tick |
| offer | 7 | `in_arrears` | the agent has an overdue claim as debtor (`Overdue` at or above 0): a payout claim or a loan |
| offer | 8 | `exposure` | the agent's open liability plus `MaxLiability` exceeds `CapBP` times its available money (5.2.1) |

5.2.1. The agent's available money is its balance less what its orders of the tick
reserved (`spec-clearing.md` 3.3), as in CRD 5.2.1, and **its open liability** is
the sum of `MaxPayout` over the open policies of which it is the underwriter, the
route policies in flight and the policies of this round already formed.

5.2.2. **The world does not clamp.** An offer above its room is refused whole with
`exposure`, not cut. The default strategy sizes its offer inside the room it
computes from its balance less its own bids (10.3), so the check is the backstop for
any other strategy and a damper that is counted (`CoverRound.Exposure`,
FR-STB-013). The default can still be refused when money leaves between the plan and
the round: an underwriter whose role also has a `Lend` block may have lent at P3.3b.

5.2.3. **A request in the plan of an agent that is away** is admitted: the
traveller's plan was built at P3.2 before it left (3.2.2).

## 6. Value and price

### 6.1 What the expected loss is, and what is read

6.1.1. The **expected loss** of a lane `(from, to, mode)` is RSK's
`RouteLoss(from, to, mode)` (`spec-risk.md` 8.1.1, 8.2), a share of a cargo in basis
points, derived and cached with the graph, 0 with the module off or no row on the
path. It includes the scales a consumer has set: a shock is in the premium of the
next tick's round. Its cost class is 1 (a lookup).

6.1.2. **What is never read.** `LossBP` of a shipment before its arrival, a draw, the
`Common` flag, a probability or a scale as such (`spec-risk.md` 8.5, 10.2.2). The
module has no code path that reads `Shipment.LossBP`: the round of 7 and the
settlement of 8 take only `RouteLoss`, the shipment's lines and the arrival's lost
units. A test asserts it by running one world twice, with every strike of a lane
fixed to the opposite outcome, and comparing every policy formed (FR-INS-025).

### 6.2 The value of a unit

6.2.1. The **unit value** of commodity `c` in region `r` is

```
v(r, c) = min( published(r, c),  trunc( ValueCapBP * reference(r, c) / 10 000 ) )
```

with `published(r, c)` the price published by the market of `r` at the last clearing
**before the position that reads it** (`spec-clearing.md` 5.4: at P3.3c and at P1.3a
the price left by P4.1 of the previous tick, at P8.1b the one left by P4.1 of this
tick, the seed price before the first trade), and `reference(r, c)` the **effective reference price** of
the commodity in `r` (`spec-goods.md` FR-GDS-020: a region's override or the
commodity's `ReferencePrice`). The product is below `MaxPrice * MaxValueCapBP` and
fits `int64`.

6.2.2. **Why a cap.** A loss of goods raises a price (a shortage), and a payout
that follows the price makes the cover pay more for the same units exactly when
markets are short (INS-A3). The cap bounds a unit at a multiple of a number that is
configuration: no aggregate is fed back, and a loop of the kind "a loss raises the
published price and with it the payout" cannot gain beyond the cap. A pack that
wants the market's price at the loss sets `ValueCapBP` at its bound; one that wants
payout at no more than the reference sets 10 000.

6.2.3. **Why the published price and not the insured's.** An insured's own limit
prices are its choice, and a cover that paid at the insured's valuation would pay
for a number the insured wrote (moral hazard, INS-A1). The published price of a
market is what every agent of the region sees and none sets alone.

### 6.3 The sum insured

6.3.1. **The terms of a sum.** Two terms exist, and a shipment has the ones its
shape gives (`spec-transport.md` 5.1, `spec-risk.md` 5.4.1):

- the **lines term** `Σ q_c * v(From, c)` over the shipment's lines at the
  **origin** region `From`, read at P3.3c. The origin's price is the replacement
  cost of the goods the shipper bought (a lost cargo is rebought there); the margin
  the shipper expected at the destination is not insured. A shipment has lines
  when it is hired, or consigned to another agent (the lines are in a Transit
  account);
- the **account term** `Σ h_c * v(From, c)` over the traveller's account's holdings
  `h_c` at P3.3c except the mode's `Vehicle` commodity (3.4.2): the goods that
  travel with it. Only a **self-carried** trip has a traveller and this term; a hired
  shipment has none.

6.3.2. **`Sum` by shape.**

| Shape of the shipment | `Sum` |
|-----------------------|-------|
| hired (no traveller) | the lines term |
| self-carried to oneself (no lines) | the account term |
| self-carried with lines consigned to another agent | the lines term **plus** the account term (the rest of the account still travels with the traveller, and both are exposed, `spec-risk.md` 5.4.1) |

`MaxPayout` follows from `Sum` (6.4.1), and the loss value of 8.2.1 is over the same
two terms, so the sum insured and the loss value cover the same goods in every shape.

6.3.3. **A store policy.** `Held` is `Σ h_c * v(r, c)` over every commodity the
insured's account holds in its region `r`, and `Sum = trunc(Held * ShareBP / 10 000)`.
The sum is bounded by what is held, so the cover cannot exceed the goods at risk at
formation; the payout is on the units **actually lost** (8.2), so emptying the store
after the cover is bought gains nothing.

6.3.4. A policy with `Sum` below `MinSum` is not formed (`below_minimum`, 7.4).

6.3.5. **Overflow.** Every product `q * v` and every sum of 6.3 and 8.2 is computed
with the checked helper (architecture.md 3.5), and each has a stated outcome:

| Quantity | When it passes `MaxMoney` |
|----------|---------------------------|
| `Sum` or `Held` at formation | the candidate is not formed and is counted `overflow` (7.4); it is not scanned against offers |
| `LossValue` at settlement | it **saturates** at `MaxMoney`; `Due` is then at most `MaxPayout` (8.2.1), so nothing is lost by it |
| `CapBP * available money` and `RuinShareBP * Money` | the product is made in 128 bits and compared there (the admission of 5.2 step 8, the default insured of 10.2); the view's `Capacity` saturates at `MaxMoney` |
| the shield `ShieldTicks * upkeep` | saturates at `MaxMoney` (8.3.1) |
| the premium | the 128-bit quotient is checked against `MaxMoney` and a failure is `overflow` (6.4.1, 7.4 step 4) |

The intermediates of the three products that are not a plain `q * v` are
**not** formed in 64 bits: `CoverBP * S` and `CoverBP * LossValue` (up to
`10^4 * MaxMoney`, 6.4.1) and `Held * ShareBP` (6.3.3) each go through the same
128-bit multiply-divide helper as the premium, and the quotient is checked against
`MaxMoney`. The quotient fits by construction: `halfEven(CoverBP * S / 10 000)` and
`trunc(Held * ShareBP / 10 000)` are at most `S` and `Held`, and `MaxPayout` is at
most `Sum`, so it fits whenever `Sum` does.

### 6.4 The premium, the share and the maximum

6.4.1. **Route cover.** For an underwriter with `LoadingBP` and `CoverBP`, a lane
with expected loss `L` and a sum insured `S`:

```
premium   = halfEven( S * L * CoverBP * LoadingBP / 10^12 )
rate      = halfEven( L * CoverBP * LoadingBP / 10^8 )          basis points of S, for the reports
MaxPayout = halfEven( CoverBP * S / 10 000 )
```

The first product is at most `10^18 * 10^4 * 10^4 * 10^5 = 10^31`, which fits 128
bits and not `int64`; it is computed with the checked multiply-divide helper of
`internal/` (architecture.md 3.5, `spec-currency.md` 7), and the quotient is
checked against `MaxMoney`. The rounding is half to even, the rule of
`spec-clearing.md` 5.2 stated here because a rounded function states its rule; a
change of it is a behaviour change (architecture.md 6.6). `LoadingBP` 10 000 is a
**fair** premium for a risk-neutral underwriter: `S * L * CoverBP / 10^8` is the
expected payout of a cargo whose value does not change, when the strikes are as
`RouteLoss` says (a statement of arithmetic and not of the library: the expected
loss is of kept shares rounded, `spec-risk.md` 8.1.1).

6.4.2. **Store cover.** `premium = halfEven(S * StoreRateBP / 10 000)` for the one
tick, `MaxPayout` as above.

6.4.3. **What the premium does not include.** No margin for the correlation of a
common row, no margin for the underwriter's capital cost, no experience: the
underwriter's `LoadingBP` is the whole of its price (position 5, 10). A pack that
wants a lane with a common row dearer states a higher loading in a role of its own,
and an insured that reads the loading in an offer (it does not: 10.1) could choose
among roles.

6.4.4. **A premium below 1** (a small sum at a small loss) is not formed
(`below_minimum`): an amount below 1 is refused by the ledger (`spec-ledger.md` 4).

## 7. The round

### 7.1 When

7.1.1. Once per tick, in phase 3, after the credit round (P3.3b of `spec-credit.md`
6.1) and before the estates' asks (P3.4): the new position **P3.3c**. It runs only
with `Modules.Insurance` present and is a no-op when no request and no offer was
submitted.

7.1.2. **Why there.** (a) After the last plan and after the contract round (P3.3):
every shipment of the tick exists, a contract's too (5.1.2). (b) After the strike of
each shipment is fixed at its dispatch (`spec-risk.md` 4.1): not because the round
reads it (it does not, 6.1.2) but because the shipment must exist to be covered.
(c) After P3.2's orders: the premium is paid from available money, which is balance
less reservations, so the round cannot take what an order earmarked (as CRD 6.1.2
and MER's escrow). (d) After the credit round: a loan formed at P3.3b is already in
the balance, so **an agent can pay a premium from money it borrowed this tick**. It
is deliberate and is registered (INS-A6); the opposite order would be a rule that the
insured must have the cash before it borrows, which no strategy could plan.

### 7.2 Lanes and candidates

7.2.1. A **lane of the round** is a region: the region of the insured's account at
the round (for a traveller away on a self-carried trip, the origin: its account has
not moved). A request is matched only with offers of its own lane, visited in
ascending `RegionID`; an outside node has no agent and no lane. The reason is CRD
6.2.1's: an underwriter judges a risk it can see, and the engine gives it none of
another region.

7.2.2. The **candidates** of a request are the policies it asks for:

- a lane request: every shipment with `Shipper` the agent, `Dispatched = t`, the
  request's mode and destination and `From` the lane's region, in ascending shipment
  ID, that **has no policy yet** (a second request for the lane is `duplicate`);
- a store request: one candidate.

A route candidate whose lane has `L = 0` is skipped (`no_risk`, 7.4): there is
nothing to insure and the premium would be 0.

### 7.3 Order and matching

7.3.1. In a lane the admitted requests are ordered by their limit descending
(`MaxLoadingBP` of a lane request, `MaxStoreRateBP` of a store request, the two
compared in separate passes below), equal limits by `K(t, agent)` ascending then
agent ID then plan order. The admitted offers are held in **two orders**: by
`LoadingBP` ascending for the lane requests and by `StoreRateBP` ascending for the
store requests, equal values by `K(t, agent)` ascending then agent ID.
`K(t, id)` is `spec-world.md` 10.1.1's. It takes **no draw** (position 11).

7.3.2. The lane requests are served first, all of them, then the store requests, so
that the order of the two kinds is a rule and not a function of IDs. For each
candidate of a request, in order, with `S` its sum, the world goes through the
offers of the matching order:

1. an offer is skipped if it is the insured's own, if its `CoverBP` is below the
   request's `MinCoverBP`, or if its remaining `MaxLiability` is below the
   **pair's** maximum payout, which is `halfEven(CoverBP * S / 10 000)` of the offer's `CoverBP` and the candidate's sum `S` (6.4.1), not of any other offer's;
2. an offer whose loading (rate) exceeds the request's limit ends the scan for this
   candidate (**no underwriter at the price**: the offers are in ascending order
   of the price a lane request compares);
3. otherwise the pair is a **tentative fill** that goes through formation (7.4). If
   formation succeeds, the offer's remaining `MaxLiability` falls by `MaxPayout`;
   if it fails with a reason, the candidate is counted and the scan stops for this
   candidate (a failure is a property of the candidate, not of the offer, except
   the `overflow` of step 4, which is the underwriter's balance: the scan goes on).

7.3.3. **Discriminatory, not uniform.** Each policy is at its underwriter's own
posted terms, not at a common price: a cheaper underwriter fills first and an
underwriter has no reason to post other than its true price, as in CRD 6.3.3.
**What the order means when capacity binds:** the requests with the highest limits
are served and the others are left uncovered (`Unfilled`). It is rationing by
willingness to pay and not by ID or region (WLD 10.2: this position adds no place
to the list there).

### 7.4 Formation

The sum insured of a candidate is computed once, before any offer is scanned, with
the checked helper (6.3.5): a candidate whose sum passes `MaxMoney` is counted
`overflow` and not scanned. For a tentative fill of candidate `k` (a shipment or a
store) of the insured `I` with the underwriter `U`, the world then checks and acts in
this order. The first failure ends the attempt with its reason and changes nothing:

1. **`too_many_policies`**: `MaxPolicies` are open. The round ends for the lane
   (the count only rises in a round).
2. **`below_minimum`**: `S < MinSum` or the premium is below 1 (6.3.4, 6.4.4).
3. **`funds`**: the premium exceeds `I`'s available money.
4. **`overflow`**: the premium's quotient passes `MaxMoney` (6.4.1), or `U`'s balance
   plus the premium would exceed `MaxMoney`.
5. **The cover**: the premium moves from `I` to `U` with the reason `premium`
   (9.1), the policy is appended with the next ID, `MaxPayout` and `CoverBP`
   fixed, and `CoverFormed` is emitted in formation order.

7.4.1. **The money is spent at once.** The premium leaves `I` at P3.3c: an order of
P3.2 was admitted against its balance, so `I` had the money to pay both only if it
held the sum (the premium is paid from balance less reservations). It is
earned by `U` at once, as money it may bid with from the next tick (WLD 6.1.8).

7.5. **The cost** of the round is `O(requests + offers)` per lane for the sorts and
`O(candidates * scanned offers)` for the matching, counts bounded by the agents with
a block and the shipments of the tick, with no term that grows with households
(NFR-INS-002). The round allocates nothing beyond scratch the world holds.

## 8. Settlement

### 8.1 When

8.1.1. A route policy is settled at **P1.3a**, a new position of phase 1 after
P1.3 (the arrivals of transport, in which RSK's step 0 applied the loss) and before
P1.4 (the settlement of delivery contracts). A store policy is settled at **P8.1b**,
a new position of phase 8 after P8.1a (RSK's store strikes) and before P8.2 (the
flow totals). Both run only with the module on.

8.1.2. **Why P1.3a.** After the arrival, because the loss is applied there and its
amounts are in the scratch of the arrival (6.1.2, 20: an edit to TRN keeps them
whether or not an observer was supplied); before P1.4 so that the payout is in
the shipper's balance at the contract's settlement, which reads no balance and so
makes no difference to a result: the order is a rule, not a choice of result. The
payout moves before the welfare record of its tick is pushed (P6.3), so it enters that
record by the scratch of the tick (11.1); relief at P1.6 reads the records of earlier
ticks (`spec-world.md` 6.3) and so does not see it.

8.1.3. **Why P8.1b.** After P8.1a so that the amounts are decided; before P8.2 so
that nothing else of the tick follows a payout; the payout is an internal transfer
and is not in `BoundaryFlow`, so the position does not move a total.

### 8.2 The loss value and the due

8.2.1. For a policy at its position, with `lost_c` the units of commodity `c` that
RSK's loss took: for a route policy the units of the same two terms as the sum
(6.3.1): the `Lost` of its lines when it has lines, and the `AccountLost` amount of
the arrival when it is self-carried (`spec-risk.md` 5.2, 5.4), for a store the `Lost` amounts of
the agent's `StoreLoss` events of the tick, over every class (`spec-risk.md` 6.4):

```
LossValue = Σ_c lost_c * v(r, c)          r = the origin region of the shipment, or the store's region
Due       = min( MaxPayout,  halfEven( CoverBP * LossValue / 10 000 ) )
```

`v` is the unit value of 6.2 **at the settlement**: the published price of the last
clearing before the position (the previous tick's P4.1 at P1.3a, this tick's P4.1 at
P8.1b; a rule of 6.2.1 and not a choice of the strategy), capped as at formation. A
store policy's loss value therefore may read a price this tick's trading moved; the
cap and `MaxPayout` bound what that is worth. A price that has fallen lowers the
payout and one that has risen is cut by the cap (6.2.2); `Due` never exceeds the
maximum fixed at formation, so the underwriter's liability is known at the round
(the exposure check of 5.2 is a bound on `Σ MaxPayout`).

8.2.2. A policy with `LossValue` 0 (no strike, or a strike that rounded every line to
0, `spec-risk.md` 4.3.3) ends with the cause `no_loss`. A policy whose `Due` is above
0 ends `claimed`; one whose `Due` rounds to 0 with a `LossValue` above 0 ends
`claimed` with a payout of 0.

8.2.3. **A cohort.** The insured is the agent: its policy and its loss are the
agent's, and the cohort's weight is in neither the sum nor the value (`spec-risk.md`
6.3.4, 3.2.2 here).

### 8.3 The pass: the shield and the allocation

8.3.1. At each position the world forms, for every policy with `Due > 0`, in
ascending policy ID, the list of **payments**, and then visits the **underwriters**
in ascending agent ID. For an underwriter `U` with the balance `b` recorded at the
entry of the position (so that no payment of the position changes another's pot),
upkeep cost `u` (`spec-credit.md` 5.3.1), CRD's `ShieldTicks` `s` and the total
`owed = Σ Due` of its policies of the position:

```
shield = s * u                (checked; saturates at MaxMoney)
pot    = max(0, b - shield)
pay    = min(pot, owed)
```

8.3.2. When `pay` is less than `owed` it is shared among the underwriter's policies
of the position **pro rata by `Due`** with the largest remainder, ties by
`K(t, policy ID)` then policy ID (`spec-credit.md` 8.3, the same helper). The share
of a policy is paid to its insured with the reason `payout` (9.1). **Why pro rata:**
a policy is not senior to another because a shipment was dispatched earlier: the
only order would be an ID, and WLD 10.2's rule is that an ID gives no advantage.

8.3.3. **The shield is CRD's.** `ShieldTicks` is `Modules.Credit`'s: the underwriter
is a debtor of its payouts exactly as of its loans, and a payout that took its last
unit would leave it unfed and ruin it before the claim it owes (CRD 8.2). Its value
is a damper strength and is that spec's.

8.3.4. **Headroom.** A payment that would take the insured's balance above
`MaxMoney` is cut to its headroom, the part not paid is dropped and counted
(`Cut`, the name of 9.3, 15.1 and 15.3), and no claim is formed for it (a household at the bound has nothing
to be made whole): the case is stated so the pass is total, not because it is
expected (CRD 8.4).

### 8.4 The payout claim

8.4.1. For each policy with `short > 0`, in ascending policy ID, where `short` is
the pro rata share of `Due` that the underwriter's pot did not cover (8.3.2; the
part cut at a headroom, 8.3.4, is **not** in it, so `Due = Paid + Short + Dropped +
Cut` of 9.4 item 1 holds and `Short` is the ledger's `Awarded`), the
world calls the ledger operation **`Obligate(creditor = insured, debtor =
underwriter, amount = short, due = t + PayoutTicks)`** (9.2): a claim with
`Principal` = `PrincipalOwed` = `short`, `Interest` 0, **no transfer**, `Formed` the
tick, `Overdue` -1, and the **origin** `payout`. It then **is** a claim of the
credit list and obeys CRD unchanged: it falls due at P1.7a (CRD's position, `spec-credit.md` 8.1.1; the taxes of `spec-fiscal.md` run after it at P1.7b) of `t + PayoutTicks` and
is paid from the underwriter's balance above its shield pro rata with its other
claims; what cannot be paid is overdue, bears nothing (CRD 8.7), bars the
underwriter from a new offer (`in_arrears`, 5.2) and from a loan (CRD 5.2 step 7),
may be written off (`WriteOffTicks`), and at the underwriter's removal is paid by
the waterfall from its whole balance and then lost (CRD 9.1); at the insured's
removal it passes to the estate of its region (CRD 9.2).

8.4.2. **When the list is full** (`MaxClaims` claims are open, CRD 6.4 step 1) the
shortfall cannot be recorded: it is **dropped and counted** (`Dropped`), and the
event says so. The insured then bears it. A pack sets `MaxClaims` above
`MaxPolicies` plus its loans to make this impossible, and the warning
`claims_below_policies` says when it is not (14.2).

8.4.3. **Why a claim and not a retry by the module.** The module would need its own
list of obligations, its own pass at P1.7a, its own shield, arrears and waterfall,
and a checkpoint section for them; the credit list is the one place the ledger
keeps what one account owes another (`spec-ledger.md` 10). The price is one ledger
operation (9.2) and one field (the origin).

### 8.5 The end of a policy

At the end of the position, in ascending policy ID, a policy is removed from the
list with its cause and `CoverEnded` is emitted (15.1):

| Cause | When |
|-------|------|
| `no_loss` | the shipment arrived and nothing was lost, or the store was not struck (a store policy that ends with `no_loss` is **counted** in `CoverSettled` and emits no `CoverEnded`: 15.1.1) |
| `claimed` | a payout was due (paid, shortfall claim or dropped) |
| `insured_removed` | the insured was removed (8.6), a store policy of the tick included (8.6.3) |
| `underwriter_removed` | the underwriter was removed (8.6), a store policy of the tick included (8.6.3) |

### 8.6 A removal

8.6.1. `spec-agents.md` 3.6 gains one step, **2f**, after CRD's 2d (the waterfall) and
2e (the passing of claims): for the removed agent, in ascending policy ID, each open
policy of which it is the **insured** ends `insured_removed` and each of which it is
the **underwriter** ends `underwriter_removed`, and `CoverEnded` is emitted. **No
money moves**: the premium was earned at formation and is not refunded, and no payout
is due for a loss not yet told. Step 3 moves what is left of the balance to the
estate as before.

8.6.2. **What the insured bears.** The insured whose underwriter is removed has paid
for a cover that ended (its premium kept, 6.4.3's absence of a refund): the
counterparty risk is the insured's and is not in any view (INS-A4). A shipment **with
lines** of an insured that was removed arrives and is struck as before
(`spec-risk.md` 5.4.3), and no payout is due; a shipment to oneself is **dropped with
its traveller** (`spec-risk.md` 5.4.3, `spec-transport.md` 5.9.2): it has no arrival,
no loss and no settlement. The step orders itself against the steps of EMP and LND by
CRD 9.1.1's reasoning: it moves no money, so the order is not a matter of result.

8.6.3. **Which policies a removal at P7.1 meets.** Removal happens at P7.1 (phase 7)
and between ticks, never at P1.3a or P8.1b, and **phase 7 comes before P8.1b**
(`spec-world.md` 6.9, 6.10). A removal at P7.1 of tick `t` therefore meets:

- the route policies of **earlier** ticks whose shipments arrived at P1.3a of `t`:
  already settled, so a removed underwriter's payout claim is in the list and the
  waterfall pays it (CRD 9.1), and a removed insured's claim passes to the estate
  (CRD 9.2);
- the route policies whose shipments are still in flight: ended by step 2f with no
  payout, as 8.6.1 says;
- the **store policies of tick `t`**, formed at P3.3c and **not yet settled**, since
  the strike of P8.1a is yet to come: ended by step 2f with the cause
  `insured_removed` or `underwriter_removed` and **no payout**, and the premium of the
  tick is kept by the underwriter and not refunded. The strike of P8.1a does not
  reach a removed insured's goods (they are in the estate), and the loss is not told.
  A store policy is therefore at the counterparty risk of 8.6.2 for the part of its
  tick that precedes P8.1b, as a route policy is for its flight (INS-A4).

### 8.7 A weight change, a role change, a relocation, a trip

8.7.1. **A weight change** changes no policy (a cohort loses households and keeps
its policies, as CRD 9.3). **A role change** leaves every policy as it is: the
payout is the agent's whatever its block now is, and an agent that lost its
`Underwrite` block posts no offer (CRD 9.4.1). **A relocation, a trip and a
journey** change no policy and none is refused by one: this module registers **no
`region_bound` refusal** (`spec-regions.md` 6.5.2, N-090): a route policy is bound by
`open_shipment` (`spec-transport.md` 5.5.3), a store policy ends in its own tick,
and a payout is a transfer between accounts.

## 9. The ledger

### 9.1 Reasons

Two reasons join the registry of `spec-ledger.md` 7.2, both **internal**, money
only, declared at construction when the module is on and closed for the run (as MON
declares its flows' reasons and CRD its three), and neither a faucet nor a sink:

| Reason | Of | Kind | Between | Owner |
|--------|----|------|---------|-------|
| `premium` | money | internal | agent (the insured) to agent (the underwriter) | `spec-insurance.md` 7.4 |
| `payout` | money | internal | agent (the underwriter) to agent (the insured) | `spec-insurance.md` 8.3 |

They collide with no flow ID and no other reason (`ErrReasonCollision`,
`spec-money-supply.md` 4.5.2). Internal reasons are told by their own events, not by
`BoundaryFlow` (`spec-observability.md` 4.4.1). No account is an Escrow or a fund:
both accounts are Agent accounts, and a payout claim's later payments are CRD's
`loan_repay` (9.2).

### 9.2 The operation and the origin

9.2.1. The ledger gains one operation beside CRD's four (`spec-credit.md` 3.2):

| Operation | Does | Refused with |
|-----------|------|--------------|
| `Obligate(creditor, debtor, amount, due)` | appends a claim with the next ID, `Principal` = `PrincipalOwed` = `amount`, `Interest` 0, `Formed` the current tick, `Overdue` -1 and the origin `payout`, with **no transfer** | `ErrInvalidAmount` (below 1), `ErrClaimShape` (the same account on both sides, a `due` not after `Formed`), `ErrUnknownAccount`, `ErrClosedAccount`, `ErrTooManyClaims` (the list is full; CRD's sentinel, `spec-credit.md` 3.2) |

9.2.2. A claim gains one field, **`Origin`** (`loan` or `payout`), fixed at
formation. Every operation of CRD works on a claim of either origin unchanged:
`Settle` pays it (the reasons are `loan_interest` with 0 and `loan_repay`),
`WriteOff` ends it, `Reassign` passes it, the pass of P1.7a visits its debtor, the
waterfall of 9.1 there pays it, the shield and the arrears apply. What differs is
read only by reports and by two rules: `ForgiveClaims` ends **loans only** (an edit
owed to CRD 13.2: a decree that cancels debts does not cancel an insurer's
obligation, 22 open question 8), and the welfare record counts the principal paid on
a payout claim as income and cost (11.1).

9.2.3. **The totals.** The ledger's cumulative totals of CRD 3.3.2 gain one,
`Awarded` (the sum of `Principal` of every payout claim formed), and the identity of
CRD 3.6 item 1 becomes: the sum of `PrincipalOwed` over open claims equals `Lent +
Awarded - Repaid - LostPrincipal`, with exact equality. `Obligate` moves no money, so
it is not in the identity of money (item 2).

### 9.3 The module's totals

Cumulative, never reset, plain `int64`s, saved with the module's section (16): `Formed`
(policies), `Premiums` (the sum of `premium` paid), `Due` (the sum of `Due`), `Paid`
(paid at once), `Dropped` (shortfall not recorded,
8.4.2) and `Cut` (cut at a headroom, 8.3.4): six totals. The unpaid part formed as a payout claim, `Short`, is **not a seventh total**: it is the ledger's `Awarded` (9.2.3), the same quantity (`Obligate` is called by this module only), read from it, so a figure is saved once (N-128, `spec-checkpoint.md` 4.5).

### 9.4 The identities

These hold after every position of the module and are checked by the audit
(`spec-ledger.md` 12.4; FR-INS-081):

1. **A payout is the sum of its parts**: `Due = Paid + Short + Dropped + Cut`,
   exactly, over the run.
2. **Money is conserved.** The total of money over non-boundary accounts is unchanged
   by `premium`, `payout` and `Obligate` (FR-LED-024: the reasons are internal).
3. **Premiums are transfers**: `Premiums` equals the sum of `Premium` over every
   policy formed, and the sum of `premium` transfers over the run.
4. **A policy has a live shipment**: every open route policy names a shipment in
   flight, its insured and underwriter are open accounts and they differ.
5. **Admission is a bound at the round**: for each offer, the open liability at admission plus `MaxLiability` is at most `CapBP` times the available money at admission, and the policies formed from it sum to at most `MaxLiability`. It is checked in the round and is not an audit identity: money falls afterwards, and that fall is exactly the risk (INS-A4).

## 10. What a strategy sees, and the default rules

### 10.1 The `Cover` group of the view

`AgentView` (`spec-agents.md` 6.4.1) gains one group, present **only in the view of
an agent whose role has a cover block** (as the merchant and credit groups are).
Plain values, valid for the call only:

| Field | Meaning |
|-------|---------|
| `Underwrite`, `Insure` | a copy of the role's block (one is nil) |
| `MinSum`, `MaxCoverRequests` | the module's |
| `Open` | for an underwriter: the sum of `MaxPayout` of its open policies, its open liability |
| `Money` | the agent's whole balance at the start of phase 3: no reservation exists yet (`spec-agents.md` 6.4.1) |
| `Capacity` | for an underwriter: `max(0, CapBP * Money / 10 000 - Open)`, before the bids of the plan it is about to build; saturates at `MaxMoney` (6.3.5) |
| `Owed` | the principal of payout claims it owes as debtor; 0 when none |
| `Policies` | for an insured: the number of its open route policies |
| `LaneLoss` | for an insured: per (destination, mode) of `Routes` the expected loss, which is `Routes[].LossBP` (`spec-risk.md` 8.1.2) |
| `ValueCapBP` | the module's, so a strategy need not know the configuration |
| `UnitValue` | for an insured: per commodity of its region, the unit value `v(r, c)` of 6.2.1 (the published price capped at `ValueCapBP` of the effective reference price, FR-GDS-020), read at the start of phase 3, so that `DefaultCover` (10.2) values a plan's goods as the round will |

The group is built at the start of phase 3 and does not change within the tick. It
has **no other agent's policy, no offer, no loading, no count of underwriters and no
figure of the market for cover**: an insured cannot see which price it will get, and
an underwriter cannot see what an insured asks (N-106's reasoning, as CRD 10.1).

### 10.2 The default insured

A household of a role with an `Insure` block, running a strategy that calls the
exported `DefaultCover` (10.4), appends:

```
for each lane (mode, to) on which its plan has a shipment or an offer:
    L = Routes[to][mode].LossBP ;  if MinLossBP = 0 or L < MinLossBP: no request  (0: no route cover)
    V = value of the plan's goods on the lane at the unit values of 6.2 at its own region
        (a shipment's lines; a delivery offer's quantity of its commodity)
    if 10 000 * V < RuinShareBP * Money: no request          the cargo is small against what it holds
    request = { Mode, To, MinCoverBP, MaxLoadingBP }
if StoreShareBP > 0 and it owns a store:
    request = { ShareBP: StoreShareBP, MinCoverBP, MaxStoreRateBP }
```

10.2.1. **Reading it.** The rule is **ruin-based and not price-based**: a household
insures what it could not afford to lose, and the limit it accepts is data. It does
not compare the premium with the expected payout (the library's household has no
risk aversion to price, and the loading is at least fair for a prudent pack). A
strategy that does compare, from its own claims, is a consumer's (10.4).

10.2.2. **The premium is not in the merchant's margin.** The merchant of
`spec-merchants.md` 5 prices the lot net of the expected loss and does not price
cover: a lot that is profitable uninsured may have a negative result insured, by
the premium less the recovery. It is the design of version 1 (INS-A5): a margin that
knew the offers would read a price the round forms after the plan (7.1.2). A pack
that wants the margin to include a premium sets `MinLossBP` and `RuinShareBP` so that
cover is bought only where the lot is large against the household, and a consumer's
strategy may net the recovery (INS-A1's third case).

### 10.3 The default underwriter

A household of a role with an `Underwrite` block appends **one offer** from what its
own bids left, as CRD's lender does (`spec-credit.md` 10.3):

```
avail = max(0, Money - Bids)                  Bids: the reserved cash of the plan just built (5.2.1)
free  = max(0, CapBP * avail / 10 000 - Open)  the room of 5.2 step 8, at the plan
offer = trunc( ShareBP * free / 10 000 )
if offer < MinSum or Owed > 0: no offer
offer = { MaxLiability: offer, LoadingBP, StoreRateBP, CoverBP }   the block's
```

with `Bids` the sum of limit times quantity of the intents of the plan it has just
built, so the offer is inside the room that the admission of 5.2 will compute from
the balance less the orders' reservations. It offers a share of its free capacity
so that it does not commit its last unit, and not at all while it owes a payout
claim (the round would refuse it, `in_arrears`). **It keeps nothing for its own
upkeep beyond what `CapBP` and `ShareBP` leave**: unlike CRD's lender there is no
`KeepTicks`, because an underwriter's liability is a possible payout and not money
lent out, and the shield of CRD (8.3) is what keeps its own tier 1 fed when a payout
falls due; a pack that wants a larger reserve lowers `CapBP` (10.3.1). **The terms
are its block's**, a function of nothing it sees (position 10).

10.3.1. **An underwriter lives on what it keeps.** An underwriter with no other
income pays its upkeep from premiums and payouts: with a loading of 10 000 and no
loss its income is the premiums, which fall to nothing when no one insures (the
warning `underwriter_without_income`, 14.2). The rule does not protect it; the
shield of CRD does, once.

### 10.4 A consumer's strategy

It may append requests and offers of its own and ignore the blocks' terms, inside 5.2.
It receives the group of 10.1. The default rules of 10.2 and 10.3 are exported as
functions of the view (`DefaultCover`, `DefaultOffer`), in the way `DefaultStrategy`
is exported (`spec-agents.md` 6.7), so a wrapper can call them (architecture.md 4.7:
held in a named field, not embedded). The `Merchant` of `strategy/` calls
`DefaultCover` for the lanes of its trips (an edit owed to MER, 20).

### 10.5 The directed agent and the player

The `Directed` strategy's plan is the fallback's changed by the directive
(`spec-agents.md` 10.4): a player whose role has a block insures or underwrites by
the fallback's rules. A consumer that wants its player to ask for a given cover needs
optional fields on the `Directive`; they are not offered in version 1 (open question
9).

### 10.6 No state

The default strategy keeps none. A request or an offer lasts one tick. Nothing of the
insured's or the underwriter's record is saved in an agent record (N-128): the
policies are the module's and the payout claims the ledger's.

## 11. Welfare and the neighbouring specs

### 11.1 The welfare record

With the module on, the insurance result `N_t` is one term of the signed unearned sum `G_t` of
`spec-welfare-and-wages.md` 2.1.9, which is inside `income_t` (WEL owns the letters and the formula,
where `G_t = (grants) + P_t + I^in_t + N_t + Q_t - X_t`), with `N_t` the **insurance result** of tick `t`: the payouts, the premiums and the
principal of payout claims the agent **received**, less the premiums, the payouts
and the principal of payout claims it **paid**, in cash, at the tick they moved.
So an insured agent's `N_t` has the premium as a cost and the payout as an income,
and an underwriter's has the premiums it receives as income and the payouts it
pays (and the principal of its payout claims) as cost. A premium is not a trade, so
the premium an underwriter receives is **not** in `R_t` (WEL 2.1: the money
received from trades) and is in `N_t` only; the term is the one place it is
counted. A premium is a cost, a payout an income: unlike a
loan's principal, which is a balance-sheet exchange (CRD 11.1), a payout replaces
goods that were lost and a premium is the price of that, so both are in
`G_t` and so in `income_t`. The principal paid on a payout claim by CRD's pass is in `N_t` of both
parties at the tick it moves (an edit to CRD 11.1: "the principal is neither" is for
a loan).

11.1.1. **Not wage.** `N_t` is in `G_t`, so it is excluded from the observed wage (WEL 4.3) as every
term of `G_t` is: an underwriter is not paid for working by it, and a payout is not an
earning. The record gains no field: `G_t` is the one signed integer that already holds the
grant (WEL 13).

11.1.2. **What it does to an insured shipper.** A lost cargo with no cover is cost
with no income in the window (RSK-A4's ruin chain). With cover the payout is income
in the arrival tick, so the household does not read as starving and relief, the
failure ladder and migration are not triggered by an insured loss. This is the
module's purpose and the link that INS-A7 registers (a balancing link of RSK-A4's chain).

11.1.2a. **What it does to an underwriter.** An underwriter's record shows the
premiums it receives as income and its payouts (and the principal of its payout
claims) as cost, in the tick they move (11.1.3). A year of no loss reads as a
positive ratio and a bad tick as a negative one, so relief, the failure ladder and
migration see an underwriter by its net result and not by payouts alone; an
underwriter with no other income lives on this term (10.3.1, INS-A2).

11.1.3. **Which record a move enters.** The record of tick `t` is pushed at P6.3
(`spec-world.md` 6.8) from the tick's flows and a per-agent scratch. A flow of `N_t`
that moves **before** P6.3 (a premium at P3.3c, a payout at P1.3a, the principal of
a payout claim paid at P1.7a) is in the scratch and so in the record of its own tick.
A flow that moves **after** P6.3 (the payout of a store policy at P8.1b, and the
principal a removal's waterfall pays on a payout claim at P7.1) has no scratch to
enter and is **added to the record of the same tick, already pushed**, by amending
its last record: `income` and `G` of that record gain the amount. The amendment
needs no state of its own (the record is in the ring, which is saved), writes no new
record and changes no window length; an agent with no record of tick `t` (created at
P7.1 of the tick) is not amended and the amount is not in any record. A reader of the
ratio at P1.6, P2.5 or P7.1 of the next tick sees the amended record (WEL 2.1.9 and
`spec-world.md` 6.8, P6.3). The reads of the same tick that came
before the move (P7.1's ladder reads the record before the waterfall at its step 2d)
see the record before the amendment: a payout of P8.1b is seen by the next tick.

### 11.2 The other specs

| Spec | What insurance means there | Where |
|------|---------------------------|-------|
| RSK | the losses it reads; the read `RouteLoss`; the rule that no `LossBP` is read; a store has no expected-loss read, so its rate is data (open question 4) | RSK 8.1, 8.5, 10.2 |
| CRD | the payout claim, the shield, arrears, the waterfall, the estate's claims; one operation, one field and one total owed (9.2) | CRD 3, 8, 9 |
| TRN, MER | a shipment's lines and traveller's account are what is insured; the contract's deliverer is the insured; the contract's table of 9.7 is unchanged (the payout is the shipper's, not the payer's) | TRN 5.2; MER 9.7 |
| STO | a store owner may insure its goods; the stored fraction is RSK's and this spec reads the held value only | STO 8.3; RSK 6.1 |
| MON | no faucet, no sink, no flow; a payout is not income from a boundary | MON 3.4, 9.5 |
| EST | a removed underwriter's money pays its claims first (CRD 9.1) and then reaches the pool; the estate inherits a removed insured's payout claim | EST 2.1; CRD 9.2 |
| SPN, POP, MIG | the ladder reads `Privation`, which an insured loss no longer raises; MIG's estimate reads no policy | SPN 5; MIG 4 |
| SAV | the decision to insure is a request this spec admits | SAV |
| FIS | a premium tax is a levy of MON on the class of underwriters; none here | FIS |
| ORG | a charter that makes cover a monopoly or a price control on premiums: a refusal `price_control` on a loading is ORG's mechanism and this spec's offer is its subject | ORG |
| EXT, INF | no cover of an outside node's goods; no rumour of a premium | EXT 4.4 |
| IND, STY | a loss ratio (payouts over premiums) and the insured share of losses are measures, asked of IND (20) | IND 9.1 |

## 12. Position in the tick, draws and cost

### 12.1 The positions

Three new positions and one removal step, in the form of `spec-world.md` 6.3, 6.5 and
6.10 (6.1.10 there). Nothing is renumbered.

| ID | What runs | Owner | Reads / writes | Draws | Events |
|----|-----------|-------|----------------|-------|--------|
| P1.3a | the settlement of the route policies of the shipments that arrived in P1.3, ascending policy ID; the pass per underwriter, the payments and the payout claims | INS 8 | the arrival's lost units (scratch), policies, balances, the upkeep cost, published prices; writes balances, claims, policies, totals | none | `CoverEnded` per policy in ascending ID, then `CoverSettled` per region ascending |
| P3.3c | the cover round: admission of the requests and offers of the plans, the order, the matching, the formation | INS 5, 7 | the plans, balances less reservations, `RouteLoss`, shipments of the tick, published prices; writes balances, policies | none | `CoverFormed` per policy in formation order, then `CoverRound` per region ascending |
| P8.1b | the settlement of the store policies of the tick, ascending policy ID | INS 8 | `StoreLoss` amounts (scratch), policies, balances, published prices; writes balances, claims, totals | none | `CoverEnded` (a store policy that was paid), then `CoverSettled` per region ascending |
| removal step 2f | the end of the open policies of a removed agent (8.6) | INS 8.6 | policies; writes policies | none | `CoverEnded` in ascending ID after CRD's `LoanPassed` and before `AgentBankrupted` |

12.1.1. The events of a step are in the order of `spec-world.md` 6.11: in item 2
(phase 1) the `CoverEnded` and `CoverSettled` events of P1.3a after the
`ShipmentArrived` and `AgentRelocated` events of P1.3 and before the
`ContractSettled` events of P1.4; in item 4 (phase 3) the `CoverFormed` events and
`CoverRound` after `CreditRound` and before the estates' `OrderPlaced`; in item 9
(phase 8) the `CoverEnded` and `CoverSettled` of P8.1b after the `StoreLoss` events
and before `BoundaryFlow`; in item 8 the policies of a removal after `LoanPassed` and
before `AgentBankrupted`.

### 12.2 Draws: none

The module takes **no word** of the world's stream and adds no row to `spec-world.md`
7.3. Every tie is the stateless key `K(t, id)` (7.3.1, 8.3.2). The count of draws of
a tick is unchanged by the module on or off and by any parameter of it, so a paired
comparison across insurance settings is aligned while no other count moves
(`spec-world.md` 7.6.1; N-124): none of the module's parameters is the flag `d` of
`spec-calibration.md` 4.2.1. WLD 7.5 gains an entry in its list of what does not
draw: the cover round's order and the settlement's remainders.

### 12.3 The cadence

The three positions run **every tick** and have no rate, window or gate of their own
(`spec-world.md` 9). The only time in the module is a shipment's flight, which is TRN's.
No decision rate is derived from an aggregate (WLD 9.4): a request reads the agent's
own plan, money and `Routes`; an offer its own money and open liability.

### 12.4 Cost

Counts, not timings (N-015):

| Position | Cost | Dominant term |
|----------|------|---------------|
| P1.3a | `O(arrived shipments with a policy + underwriters paid)`, found from the policy index by shipment | arrivals of insured shipments |
| P3.3c | `O(R + O)` per lane for the sorts, `O(candidates * scanned offers)` for the matching | agents with a block, shipments of the tick |
| P8.1b | `O(store owners struck + underwriters paid)` | strikes, not owners |
| removal 2f | `O(policies of the removed agent)`, from the by-agent index | policies of one agent |
| `Cover` group of the view | `O(1)` for an insured, `O(policies of the agent)` for an underwriter, once per `Trade` | one agent |

A world with the module off has none of it. A step of the reference chain with the
module on and no role with a block allocates nothing beyond the default (NFR-INS-001).

## 13. Parameters

All are fields of `Config` with a default (or "required"), a bound, a doc line and an
owner (architecture.md 9.5 to 9.7). **Every default is a placeholder** (N-015,
N-027). The bounds are validation bounds in `bounds.go` (`spec-config.md` 4.1).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Insurance` | the insurance module: present means on | nil (off) | |
| Insurance | `ValueCapBP` | a unit's value is at most this share of its effective reference price (6.2) | required | 1 to `MaxValueCapBP` |
| Insurance | `PayoutTicks` | ticks from an unpaid payout's formation to its due tick (8.4) | 1 | 1 to `MaxCreditTerm` |
| Insurance | `MinSum` | the smallest sum insured (7.4) | required | 1 to `MaxMoney` |
| Insurance | `MaxPolicies` | the most policies open at once (7.4) | required | 1 to `MaxOpenPolicies` |
| Insurance | `MaxCoverRequests` | the most requests in one plan (5.2) | 4 | 1 to `MaxCoverRequestsBound` |
| Role | `Cover` | the cover block; nil means the role does not deal in cover | nil | exactly one of `Underwrite`, `Insure` |
| Underwrite | `LoadingBP` | route cover: the premium as a share of the expected payout | required | 0 to `MaxLoadingBound` |
| Underwrite | `StoreRateBP` | store cover: the premium per tick on the sum insured | required | 0 to `MaxStoreRateBound` |
| Underwrite | `CoverBP` | the share of a loss value paid | required | 1 to 10 000 |
| Underwrite | `CapBP` | open liability at most this share of available money | required | 1 to `MaxCapBound` |
| Underwrite | `ShareBP` | the share of free capacity offered in one tick (10.3) | required | 1 to 10 000 |
| Insure | `MinLossBP` | route cover is bought on a lane with an expected loss at least this; 0 means no route cover (a store-only insured, which needs no transport) | required | 0 to 10 000 |
| Insure | `RuinShareBP` | route cover is bought when the cargo's value is at least this share of its money | required | 0 to `MaxRuinBound` |
| Insure | `MaxLoadingBP` | the highest loading accepted | required | 0 to `MaxLoadingBound` |
| Insure | `MinCoverBP` | the least share of a loss accepted | required | 1 to 10 000 |
| Insure | `StoreShareBP` | the share of held value insured each tick; 0 buys none | 0 | 0 to 10 000 |
| Insure | `MaxStoreRateBP` | the highest store rate accepted | required | 0 to `MaxStoreRateBound` |

**New bounds for `bounds.go`** (provisional, with the reasons):

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxValueCapBP` | 100 000 | ten times the reference price: a larger cap is a typing error, and the product with `MaxPrice` fits `int64` |
| `MaxLoadingBound` | 100 000 | ten times the fair premium; with `MaxMoney`, `10 000` and `10 000` the premium product fits 128 bits (6.4.1) |
| `MaxStoreRateBound` | 10 000 | a premium of the whole sum insured for one tick: above it is a typing error |
| `MaxCapBound` | 100 000 | ten times the underwriter's money: a leverage above it is a typing error |
| `MaxRuinBound` | 100 000 | ten times the household's money |
| `MaxOpenPolicies` | 1 000 000 | the policies in the list at once |
| `MaxCoverRequestsBound` | 64 | the loop of a plan; with `MaxShipmentLines` the size of a round |

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the
rounding rule half to even (6.4, 8.2); the unit value's `min` of the published price
and the capped reference (6.2); the premium formula and the maximum (6.4); the
single-event shape of the two kinds and the dispatch-tick rule (3.1); the order of
the checks of 5.2 and 7.4; the matching order and the stateless key (7.3); lanes
served before stores (7.3.2); the pot, the shield and the pro rata allocation with
its tie (8.3); the unpaid part a payout claim (8.4); the causes `no_loss`,
`claimed`, `insured_removed`, `underwriter_removed` and the refusal reasons
(exported constants of the root package, `event.go`); the two ledger reasons (9.1);
the positions P1.3a, P3.3c, P8.1b; that nothing of the module reads a draw or a
`LossBP`; that a policy ends at a removal with no refund (8.6); that the insured of a
route policy is its shipper (3.2).

**Not parameters:** a policy's fields, `Open`, `Capacity`, the unit value, the
loss value, the totals, the derived indexes.

The placeholders, so that none is mistaken for a result: every default and bound
above, the thresholds of the warnings of 14.2, and every number of the worked
examples (19) and of the models (18).

## 14. Validation

Each rule has a sentinel or a warning name and a failing-case test (testing.md 3.9).
Errors name the path, for example `role.shipper.cover.insure.min_cover_bp`. The
framework of `spec-config.md` 7 runs them.

### 14.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field of 13 outside its bound; `ValueCapBP`, `MinSum` or `MaxPolicies` below 1 |
| `ErrCoverShape` (new, a root sentinel) | a `CoverTerms` with both `Underwrite` and `Insure`, or with neither |
| `ErrModuleDependency` | `Modules.Insurance` while `Modules.Risk` or `Modules.Credit` is nil; a role with a cover block while `Modules.Insurance` is nil; an `Insure` block with `MinLossBP` above 0 and no `Modules.Transport`, or with `StoreShareBP` above 0 and no `Modules.Storage` |
| `ErrReasonCollision` | a money flow whose ID is `premium` or `payout` (`spec-money-supply.md` 4.5.2) |
| `ErrMissingField` | a field of 13 that is **required** and whose bound admits 0, absent from a file (FR-CFG-008): `loading_bp`, `store_rate_bp`, `min_loss_bp`, `ruin_share_bp`, `max_loading_bp` and `max_store_rate_bp`. Their 0 is legal and means a free cover or none (a premium below 1 is never formed, 6.4.4), so a silent 0 from an omission would be a different pack from the one written. A required field whose bound starts at 1 (`value_cap_bp`, `min_sum`, `max_policies`, `cover_bp`, `cap_bp`, `share_bp`, `min_cover_bp`) is `ErrOutOfBounds` when omitted: the absent value reads as 0, below the bound |
| `ErrUnknownRole`, `ErrDuplicateID` | as for any role row |

No overflow error is needed at validation: a premium or a payout that would pass
`MaxMoney` is refused at formation (7.4 step 4) or cut at settlement (8.3.4) and is a
counted reason, not an error. A strategy's request or offer outside its bounds is a
refusal (5.2), not an error returned by `Step`.

### 14.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at
construction, at `Warn`, with the attributes `rule` and `path`; the volume is bounded
by the configuration. The thresholds are placeholders.

| Warning | Condition |
|---------|-----------|
| `no_cover_roles` | `Modules.Insurance` is present and no role has a block |
| `cover_no_underwriter` | a role has an `Insure` block and no role has an `Underwrite` block: no request can be filled |
| `cover_no_insured` | a role has an `Underwrite` block and no role has an `Insure` block, and no consumer strategy is registered that could request: the offers are never used (informational, as CRD's) |
| `cover_without_risk` | no route row of `Modules.Risk` (for an insured with `MinLossBP` above 0) or no store row (for `StoreShareBP`) exists: nothing can be lost, so nothing is ever paid |
| `loading_above_limits` | the lowest `LoadingBP` of any underwriter exceeds the highest `MaxLoadingBP` of any insured: no route cover can form |
| `cover_below_minimum` | the sum insured of the smallest planned shipment of the reference programme is below `MinSum` for every insured role |
| `underwriter_without_income` | an underwriter role with no recipe and no money flow that targets it: its income is premiums, which need an insured |
| `underwriter_cap_above_money` | `CapBP` above 10 000 on a role whose `StartMoney` less its upkeep is the whole of its capital: it can commit more than it holds (leverage; informational) |
| `claims_below_policies` | `Modules.Credit.MaxClaims` is below `MaxPolicies`: a shortfall can be dropped (8.4.2) |
| `no_shield_underwriter` | `Modules.Credit.ShieldTicks` is 0 and a role has an `Underwrite` block: a payout can take the last unit |
| `value_cap_unbound` | `ValueCapBP` is at its bound: the payout follows any price spike (INS-A3) |

### 14.3 What is not validated here

An insured role with no shipments (legal: a store owner); an underwriter that is also
a merchant (legal: it has recipes and a block); two underwriter roles at different
loadings (they compete in the order of 7.3).

## 15. Observability

### 15.1 Events

Four events, all plain data with no map, pointer or wall-clock time, constructed only
when an observer was supplied (`observability-and-logging.md` 4.1, 4.5). Two are per
policy (class **E**, like `ContractFormed`); two are aggregates, bounded by the
regions (class **A**, like `ContractRound`). In the form of `spec-observability.md`
4.1. `Amount` is `Money`.

| Event | Owner | Position | Class | Gate | Fields (besides `Tick`) |
|-------|-------|----------|-------|------|-------------------------|
| `CoverFormed` | INS 7.4 | P3.3c, formation order | E | insurance | `Policy`, `Kind` (`route`, `store`), `Insured`, `Underwriter` (agents), `Region` (the insured's), `UnderwriterRegion` (the underwriter's; the lane of the round makes it equal to `Region`, 7.2.1), `Shipment` (`NoShipment` for a store), `Mode`, `To` (`NoMode`, `NoRegion` for a store), `Sum`, `MaxPayout`, `Premium`, `RateBP`, `CoverBP` |
| `CoverEnded` | INS 8.5, 8.6 | P1.3a; P8.1b; with a removal (2f) | E | insurance | `Policy`, `Kind`, `Insured`, `Underwriter`, `Region` (the insured's at the event), `UnderwriterRegion` (the underwriter's at the event, which migration may have changed since formation), `Cause` (`no_loss`, `claimed`, `insured_removed`, `underwriter_removed`), `LossValue`, `Due`, `Paid`, `Short`, `Claim` (the payout claim's ID, -1 when none), `Dropped`, `Cut`, `Capped` (the payout was cut at `MaxPayout`) |
| `CoverRound` | INS 7 | P3.3c, region ascending, after the last `CoverFormed` | A | insurance | `Region`, `Requests`, `Offers` (admitted), `RequestsRefused`, `OffersRefused` (any reason of 5.2 but `insurance_off`), `InArrears`, `Exposure` (the part of `OffersRefused` with those reasons: subsets), `Candidates`, `Formed`, `Sum`, `Premium`, `Unfilled` (the sum of candidates not covered at the end), and the counts of failures at the round `NoShipment` (requests with no candidate), `NoRisk`, `NoOffer`, `BelowMinimum`, `Funds`, `TooMany`, `Overflow`, `ValueCapped` (units valued at the cap) |
| `CoverSettled` | INS 8, 15.1.2 | P1.3a and P8.1b, region ascending, after the `CoverEnded` events of the position; stage `claim`: P1.7a, region ascending, after CRD's `CreditSettled` | A | insurance | `Region`, `Stage` (`arrival`, `store`, `claim`), `Ended` (policies ended at the position, `no_loss` store policies included), `Claimed`, `LossValue`, `Due`, `Paid`, `Short`, `Dropped`, `Cut`, `Capped` (policies cut at `MaxPayout`), `ValueCapped` (units valued at the cap), `Shielded` (the money the shield kept out of a payout: over the underwriters visited, `min(balance, owed) - pay`); at stage `claim` only `ClaimEnded` (payout claims ended repaid), `ClaimPaid` (principal paid by the debtors of the region) and `ClaimReceived` (principal received by the insureds of the region), the other fields being 0 |

15.1.1. **When.** `CoverFormed` is emitted per policy formed. `CoverEnded` is
emitted for every route policy and for a store policy that was claimed or ended by
a removal; a store policy that ends `no_loss` is **not** an event (a store owner that
insures every tick would otherwise emit one a tick: the count is in `CoverSettled.Ended`
and the metrics). `CoverRound` is emitted per region and tick in which a request or
an offer was submitted for it, and never when all its counts are 0. `CoverSettled` is
emitted per region, position and tick in which a policy was settled, and never when
all its counts and sums are 0. With the module off none is emitted.

15.1.2. **What an observer can rebuild.** Which route policies are open, from
`CoverFormed` (add) and `CoverEnded` (remove); and the money of the household
sector by region. A premium and a payout **are not in `BoundaryFlow`** (the reasons
are internal), so the events carry them:

1. **a premium** moves `CoverFormed.Premium` between two households of one region
   (a lane is a region, 7.2.1: `Region` equals `UnderwriterRegion`), so it changes no
   regional total;
2. **a payout** moves `CoverEnded.Paid` from the underwriter's region
   (`UnderwriterRegion`) to the insured's (`Region`). The two differ whenever a
   self-carried insured has arrived (its account moved at P1.3, before P1.3a) or an
   underwriter has relocated since formation, so the rebuilding reads the two region
   fields of `CoverEnded` and not `CoverSettled`, whose `Paid` is by the insured's
   region only. A store policy that ended `no_loss` emits no `CoverEnded` and paid 0;
3. **a payout claim** is added by `CoverEnded.Short` with its `Claim` (there is no
   `LoanFormed`); its payments at P1.7a are in `CoverSettled` of stage `claim`
   (`ClaimPaid` by the region of the debtors, `ClaimReceived` by the region of the
   insureds), and its end is a `LoanEnded` of origin `payout`, whose `PaidPrincipal` at a
   removal is by `Region` and `CreditorRegion` (FR-INS-079). CRD's `CreditSettled` and
   instruments count origin `loan` only (`spec-credit.md` 16.1.3, 16.3).

### 15.2 Logs

`Debug`, guarded by `Enabled`, constant messages: "cover formed" (`cover_policy`, `agent`
the insured, `region`, `sum`, `premium`), "cover ended" (`cover_policy`, `agent`,
`cause`, `payout`), "cover refused" (`agent`, `reason`). **No `Info` in the tick and
no `Warn`** (3.2.2): a policy and a refusal are per entity, and an underwriter's
arrears are CRD's `Warn` "claims overdue". The warnings of 14.2 are reported once at
construction, and the summary line "world created" gains `insurance` among its
`modules` and the counts `underwriters` and `insured` of agents with each block.

**New log keys** for `observability-and-logging.md` 3.3.2: `cover_policy` (a cover policy ID,
`int64`), `sum` (money, the sum insured, `int64`), `premium` (money, `int64`),
`payout` (money paid to an insured, `int64`), `underwriters` and `insured` (counts
of agents with each block, in the construction summary). The keys `region`, `agent`,
`cause`, `reason`, `shipment` and `modules` exist, and `claim` once CRD's keys are added to the rules (`spec-credit.md` 16.2). The key `amount` is not
used (the rules define it for the line of a refused transfer).

### 15.3 Metrics (proposed)

All `Int64`, each derivable from the events or, for the liability, from events plus
the snapshot (`observability-and-logging.md` 5.3):

| Name | Instrument | Unit | Attributes | Fed by |
|------|------------|------|------------|--------|
| `economysim.insurance.policies` | Counter | `{policy}` | `economysim.cover_kind` | `CoverFormed` |
| `economysim.insurance.premium` | Counter | `{minor_unit}` | none | `CoverFormed` (`Premium`) |
| `economysim.insurance.payout` | Counter | `{minor_unit}` | none | `CoverSettled` (`Paid`) |
| `economysim.insurance.shortfall` | Counter | `{minor_unit}` | none | `CoverSettled` (`Short`) |
| `economysim.insurance.unpaid` | Counter | `{minor_unit}` | `economysim.cover_loss` | `CoverSettled` (`Dropped`, `Cut`) |
| `economysim.insurance.liability` | UpDownCounter | `{minor_unit}` | none | `CoverFormed` (+ `MaxPayout`, route policies only), `CoverEnded` (- `MaxPayout` of a route policy); seeded at a restore |
| `economysim.insurance.claims` | UpDownCounter | `{minor_unit}` | none | `CoverEnded` (+ `Short`), `LoanEnded` of origin `payout` (- `Principal`, the claim at face; the part paid is `CoverSettled.ClaimPaid`); seeded at a restore |
| `economysim.insurance.refused` | Counter | `{request}` | none | `CoverRound` (`RequestsRefused` plus `OffersRefused`) |
| `economysim.insurance.limited` | Counter | `{case}` | `economysim.cover_limit` | `CoverRound` (`InArrears`, `Exposure`, `NoRisk`, `NoOffer`, `BelowMinimum`, `Funds`, `TooMany`, `Overflow`) |

The attributes are bounded by closed sets (`observability-and-logging.md` 5.4.1):
`economysim.cover_kind` (`route`, `store`), `economysim.cover_loss` (`dropped`,
`cut`) and `economysim.cover_limit` (the eight constants, the dampers of
FR-STB-013). `economysim.insurance.liability` counts the route policies only: a
store policy opens and ends inside one tick and nets to 0 at every boundary. At a
restore `liability` is **seeded** from the snapshot's `Policies` (`spec-observability.md`
6.4.2), equal to the sum of `MaxPayout` over the rows, and `claims` from the snapshot's `Claims`
of origin `payout`, equal to the sum of their `Principal`. No credit metric counts a payout
claim (`spec-credit.md` 16.3).

### 15.4 Snapshot

Deep copies, bounded by `MaxPolicies` (architecture.md 12.4,
`spec-observability.md` 5.6), cost class F. `Policies` is a sixth in-flight set
besides `Shipments`, `Away`, `Journeys`, `Contracts` and `Claims`. Names are OBS's:

| Slice or field | Row | Fields | Source |
|----------------|-----|--------|--------|
| `Policies` | by policy ID | `Policy`, `Insured`, `Underwriter`, `Region`, `Shipment`, `Sum`, `MaxPayout`, `CoverBP`, `Premium`, `Formed` | INS 3.3 |
| `Insurance` | one value | `Formed`, `Premiums`, `Due`, `Paid`, `Short`, `Dropped`, `Cut` (9.3; `Short` is `Credit.Awarded` and is read, not saved twice) | INS 9.3 |

The `Claims` rows of CRD gain `Origin`. Every field is a function of saved state
(N-125, FR-OBS-043). The expected loss of a lane is a derived read (`RouteLoss`) and
not a snapshot field.

### 15.5 The Observability entry (observability-and-logging.md 9)

1. **New failure path or rejected input?** A refused request or offer is a counted
   reason (5.2); validation errors are returned (14.1); a payout cut at a headroom or
   dropped for want of a claim slot is survived and counted.
2. **New state transition?** A policy formed, ended; a round; a settlement: four
   events, no map, fixed-point values.
3. **New decision point in a strategy?** The default insured and underwriter (10.2,
   10.3): inputs are the view's; `Debug` lines tell the result; no `Info`.
4. **New loop?** The round and the settlement are bounded by agents with a block and
   shipments of the tick and log at `Debug`, guarded by `Enabled`; no `Warn`; event
   construction is gated by the no-observer flag (NFR-INS-001).
5. **Changed event, metric, unit or attribute?** None changed except: `LoanEnded`
   and the `Claims` row gain `Origin`, the `Credit` row gains `Awarded`, and
   an observer's open claims (`spec-credit.md` 16.1.3) hold the payout claims of
   `CoverEnded.Short` beside the loans; CRD's instruments count origin `loan` only (edits
   to CRD, 20.2); nine metrics and four events are new.
6. **New metric attribute?** Three, closed sets (15.3).
7. **`Error` or `Info`?** None.
8. **Logging an error also returned?** None.
9. **Tests?** `observability-and-logging.md` 8.1 applies: the recorder over a scripted
   route policy, a store policy, a shortfall and a removal (FR-INS-070 to 079), the
   no-op allocation test, non-interference, composition and emission order, the golden
   rendering and the restore case (FR-INS-087).

## 16. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). Plain data in a fixed order, no
maps; **no per-agent field** is added (N-128).

### 16.1 What is saved

| Where | State |
|-------|-------|
| a module section, present when `Modules.Insurance` is (provisionally "Insurance"; the tag is 29 (`spec-checkpoint.md` 4.2), after Risk and Credit in dependency order) | the next policy ID; the open **route** policies in ascending ID: ID, insured and underwriter (agent IDs), shipment ID, `Sum`, `MaxPayout`, `CoverBP`, `Premium`, `Formed`; the six totals of 9.3 |
| the ledger section (tag 2) | the payout claims are claims of the list: `Origin` is one more field of each claim, and `Awarded` one more total beside CRD's six (9.2.3). A claim is saved once, in the ledger's section (N-128) |

A **store policy is never saved**: it is formed at P3.3c and ended at P8.1b of the
same tick, and a checkpoint is taken between ticks (`spec-world.md` 8.4).

### 16.2 Why it is the whole state

The round and the settlement read only the policies, the balances, the plans, the
shipments, the tick, the configuration, `RouteLoss` and the published prices, all of
which are state the other specs save; the arrival's lost units are recomputed in the
step (they are a function of the arriving shipment's `LossBP`, saved by RSK, and its
account). A world restored between ticks therefore settles the same policies at the
same P1.3a, forms the same policies at the same P3.3c and takes the same words of the
stream (none) as an uninterrupted one (FR-INS-087).

### 16.3 Derived and rebuilt at restore

| Derived | From |
|---------|------|
| the **by-shipment index** (a route policy per shipment ID) | the list |
| the **by-agent indexes** (policies of an insured, of an underwriter, ascending ID) | the list |
| `Open` (open liability) and `Capacity` of an underwriter | the list, balances |
| the open count against `MaxPolicies` | the list |
| the expected-loss table | RSK's, rebuilt from rows, scales and graph |

16.3.1. **The `Cover` group of `AgentState`** (`spec-integration.md` 4.7.2, a read at a boundary,
not the strategy's view of 10.1): present with `Modules.Insurance` on, and 0 for an agent with no
open policy. Its fields are `Policies` (the number of the agent's open route policies, as insured
or as underwriter) and `Open` (for an underwriter, the sum of `MaxPayout` of its open policies, its
open liability; 0 for an insured). Both are derived from the list (16.3) and saved nowhere.

### 16.4 Not saved

A request or an offer, the plan notes, the round's scratch, the allocation of a pass,
the arrival's lost units (recomputed).

### 16.5 What a restore checks

In the fixed order of `spec-checkpoint.md` 6.2, after the accounts, the agents and
the shipments are in: every policy's insured and underwriter are open Agent accounts
and differ; IDs ascend and are below the next ID; its shipment is a shipment in
flight whose shipper is the insured; `Sum`, `MaxPayout`, `CoverBP`, `Premium` are in
range with `MaxPayout` equal to `halfEven(CoverBP * Sum / 10 000)`; `Formed` is not
after the tick; the number of policies is at most `MaxPolicies`; the identities of 9.4
hold over the totals and the claims (`Short` is the ledger's `Awarded` and is not saved twice, so no equality is to be checked, 9.3). A failure is `ErrCheckpointCorrupt` naming the
tag and the policy. The configuration hash covers `Modules.Insurance` and every
block.

### 16.6 Format version

The claim record and the module's section change shape with the module on, so the
format version changes with the first release that has the module (`spec-checkpoint.md`
9.3), and a fixture of it joins `testdata/checkpoints/`.

## 17. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9 (FR-STB-003). Fields as in
its section 3; the register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn`
numbers and the entries here are `INS-An`, `INS-Dn` and `INS-Sn` (N-103). Nothing here
has been run in the library; the status `model` means the rules were run in
`models/ins/` (18) with an assumed income, and no default is claimed stable (N-027).
Delays were checked against the neighbours (9.6): the transit time `k` and the rumour
lag (`spec-transport.md` 4, `spec-information.md` 5.2), the payout's due tick
`PayoutTicks` and the underwriter's shield (CRD 8), the effective welfare window
`T` (`spec-welfare-and-wages.md` 2.3.3), a shelf life and the belief adjustment
(`spec-beliefs.md` 5.2); each entry names the ones that match.

**What was looked for and not found.** The brief names **moral hazard** and **adverse
selection**. In their full forms both need a variable the library does not have:
moral hazard a probability of loss that the insured moves (RSK position 13: it is
exogenous) and adverse selection information that one side holds and the other does
not (RSK 4.4.1 and 10.2.2: nobody holds a strike, and the expected loss is shown to
all). What the module has is their **quasi forms**, entered below: the insured's
exposure and choice of lane (INS-A1), the price that the pack forgets to charge for a
correlation (INS-A2), and the underwriter's limited liability (INS-A4). The loop
"cover makes shipping on a dangerous road cheap, which raises exposure, which raises
payouts, which raises premiums" is a loop of this module only if the premium follows
paid losses; it does not (position 10), so the chain is balancing and bounded and the
reinforcing version needs experience rating (open question 3).

**Merges into existing loops** (checklist 9.9): INS-A1 into L84, L85 and L93 as a
case (with RSK-A1); INS-A2 into RSK-A3's lockstep and CRD-A3's default cascade as a
case; INS-A5 into L19 (a trip is a gap in income) and into L14 (the premium is a cost in `income_t`) as a case; INS-A6 into CRD-A1 as a
case; INS-A7 into RSK-A4 as a balancing link of its chain. INS-A3 and A4 are new.

**INS-A1. The shipper that nets the recovery** (a case of L84, L85 and L93;
RSK-A1). *Kind:* loop. *Loop:* the gap at the destination (+) shipments (+) cargo
arriving, `(1 - lambda)` of what left (-) the destination price (+) the gap: one
negative link, **B**, with the delay of the transit time plus the rumour's age, as
L84's MER case. Cover does not touch this physical link: **a payout moves money and no
goods**. What it changes is what a shipper reads: a shipper that nets the recovery
reads a loss `lambda (1 - c)` for a cover share `c` and ships further than one that
reads the whole loss, while the cargo still arrives at `(1 - lambda)`. *Delay:* `n =
Lag + k + 1` or `k` (RSK-A1). *Owner:* INS 10.2.2; RSK 8.4; MER 5.2. *Damper:*
RSK-D1 (the expected loss in the margin), MER's D38, D39, D80, the lane lock, and
INS-D1 (the cover share below 10 000: `c` < 1 leaves the shipper a stake `lambda
(1 - c)`). *Undamped:* none new: in version 1 the merchant does **not** net the
recovery (10.2.2), so cover changes no shipping decision and this loop is the
register's for the case a consumer's strategy does. *Condition:* the linearised
condition of L84 with the arrival scale `(1 - lambda)(1 - lambda_read)`:
`g_eff < 2 sin(pi / (2 (2 n + 1))) / ((1 - lambda)(1 - lambda_read))`. In the model of
18 part 7 (`n` = 3, `lambda` 2 500 bp, `c` 8 000 bp) the largest gain that decays is
0.592 when the loss is ignored, 0.790 when it is read in full, **0.624 when the
recovery is netted** (`lambda_read` 500 bp); the shipper that nets ships 59.6 units
and lands 44.7 where one that reads the loss ships 19.2, and its margin including the
cover and a premium at a loading of 13 000 is -39.3 a unit against -6.6. So a cover
nets less than it costs: the premium exceeds the recovery by the loading. *Test:*
INS-S3. *Status:* model; a linear model of an invented rule, not of MER's.

**INS-A2. A common strike and the underwriters' bust** (a case of RSK-A3 and
CRD-A3). *Kind:* loop (a hazard of a reinforcing chain through the supply of
cover). *Chain:* a **common** row strikes (`spec-risk.md` 3.3) every shipment of a
lane at once: the policies on it fall due together (+), the payouts exceed the
underwriters' money above their shields (+), the unpaid part becomes payout claims
and the underwriters' money falls (-) capacity (`CapBP * money`) falls (-) the
offers shrink (-) cover is refused (`NoOffer`, `Exposure`) for the next route policies, and
underwriters that cannot eat are removed (SPN's ladder, the waterfall of CRD 9.1):
fewer underwriters (-) less cover (+) a new strike is borne by the insured whole.
Polarity **R** through the supply of cover, the one link of the chain that closes
(the loss, the strike, is exogenous). *Delay:* `PayoutTicks`, the claim's term, plus
the underwriter's keep and the privation count. *Owner:* INS 8, 10.3; CRD 8, 9; RSK
3.3. *Damper:* **none that this module adds, in version 1.** What limits the chain
is not the cap: the model of 18 part 6 (strike probability 0.05 a tick, policies of
100 on a capital of 1 000, 300 seeds, 300 ticks) removes 210, 235, 202 and 200 of 300
underwriters under a common row at `CapBP` 5 000, 10 000, 20 000 and 40 000, so the
cap does not reduce the removals, and a pack at `CapBP` 5 000 (where every payout
is backed by money held at writing) loses 70% of its underwriters. Under unit rows the removals fall as
the cap rises (91, 25, 8, 6: more premium income on the same upkeep), the opposite
direction. What does limit it: CRD's shield (CRD-D1: what an underwriter eats is
not paid) and pro rata settlement (CRD-D6: an insured is not wiped out by an ID); and
the spread of lanes across underwriters, which a unit row gives and a common row
withholds (RSK 10.2.3). *Undamped:* every pack with a common row on a lane that
underwriters write: **the correlation is not in the price** (6.4.3, position 10), so
the loading does not pay for the bust. *Condition:* none is shown to hold. What the
model does show is the shortfall (A4): the unpaid part of the payouts is 0.0 and 0.1
a run at `CapBP` 5 000 and 10 000 and 566.5 and 704.2 at 20 000 and 40 000, so a
`CapBP` at or below 10 000 keeps the counterparty default (INS-A4) small while the
underwriters are still removed. *Neighbours (9.6):* `PayoutTicks` against the shield's
`ShieldTicks`; the belief adjustment of the insured's market after the lost cargo
(RSK-A3). *Test:* INS-S4. *Status:* model; **undamped in version 1**, with experience
rating (open question 3) and a loading that reads the common row (open question 5) as
the two candidates.

**INS-A3. The payout follows the price** (a new loop, bounded by a constant).
*Kind:* loop. *Loop:* a loss of goods (+) the published price of the goods (the
shortage) (+) the loss value (+) the payout (+) the insured's spending and the
underwriter's loss (-) the underwriter's capacity. The first links are RSK-A2's
buffer and the market; the loop closes only through the insured's demand, which
is the weak link, and through the underwriter's capacity, which is the one INS-A2
carries. Polarity **R** where it closes. *Delay:* one tick (the price is the one P4.1
left). *Owner:* INS 6.2, 8.2. *Damper:* **INS-D4**, the value cap `ValueCapBP`
(6.2.2): a unit is worth at most a multiple of the reference price, whatever the
market says (model part 5: at a reference of 10 and a cap of 12 000 bp, a published
price from 12 to 40 pays the same 96 for 10 units at a cover of 8 000). *Undamped:*
`ValueCapBP` at its bound (`value_cap_unbound`). *Condition:* the cap is below the
price at which the payout alone would make a loss profitable (moral hazard in its
plainest form: `cover share * cap` below 1 of the replacement cost). *Test:* INS-S5.
*Status:* model (part 5), hand for the closing link.

**INS-A4. Thin capital, limited liability** (a new hazard). *Kind:* hazard. *Chain:*
an underwriter writes cover up to `CapBP` of its **available** money at the round
and then spends the money (its bids, its upkeep) before the arrival: the payout
falls due against a balance below what backed the policy (money is not reserved,
3.5 B). Its downside is its balance, its upside the premium; the insured, who sees
no balance, bears the rest as a shortfall claim or a policy ended by a removal with
no refund (8.6). *Polarity:* not a loop; a **moral hazard of the underwriter**, the
one the model of 18 part 6 shows at `CapBP` 5 000 (91 of 300 removed under unit rows
by tick 300 on a book of 5 policies and a fixed upkeep). *Delay:* the flight of a
shipment, `k` ticks, during which the backing money can be spent. *Owner:* INS 5.2, 8.6;
CRD 8. *Damper:* INS-D3 (the cap: the shortfall of part 6 is 0.0 and 0.1 at `CapBP` 5 000 and 10 000 and 566.5 and 704.2 at 20 000 and 40 000), CRD-D1 (the shield: the underwriter eats first), CRD-D2 as INS-D7 (the
refusal `in_arrears`: an underwriter that owes a payout writes nothing, 5.2), CRD-D5 (the
write-off) and the waterfall of CRD 9.1. *Undamped:* the insured's counterparty risk: no view shows an
underwriter's money or arrears; an insured chooses among offers it cannot see (10.1).
*Condition:* the pack's `CapBP` and its underwriter roles' start money keep the
liability inside what the household keeps for its own bids. *Neighbours (9.6):* the
flight time `k` against the underwriter's belief and spend-cap delays (D27). *Test:*
INS-S4. *Status:* model.

**INS-A5. The premium is a cost the margin does not see** (a case of L19 and, through `N_t`, of L14). *Kind:*
hazard. *Chain:* a lot whose margin was computed net of the expected loss (RSK 8.4)
pays a premium at dispatch that is not in the margin (10.2.2): the premium extends a
trip's gap in income (L19, `trip_longer_than_window`) and turns a lot with a thin
margin negative. *Polarity:* the premium is an input to the household's welfare record
as a cost (`N_t`, 11.1), which the ladder reads; the loop is L19's with a larger gap. The premium is also a cost in `income_t`, so the target-income rule of `spec-welfare-and-wages.md` 5 works an insured merchant more days for the same income: **L14**'s lockstep, the case that CRD-A2 registers for interest (`Y_t`), with `N_t` in its place, and its dampers are those of L14 (the rest band and `MinWorkShare`).
*Delay:* the trip and the window `T`. *Owner:* INS 6.4, 10.2; MER 5.5. *Damper:*
`RuinShareBP` and `MinLossBP` (cover only where the lot is large against the
household and the lane is dangerous: the default insured does not insure a thin lot),
INS-D1 (a lower cover share is a lower premium), the fee-share margin of safety of
MER 5.5.1 (a trip must promise twice its cost at 5 000). *Undamped:* a strategy that
insures every lot. *Condition:* `premium < margin of safety`; the model of 18 part 7
shows -6.6 a unit at the loading 13 000 for a shipper that reads the loss and insures.
*Test:* INS-S3. *Status:* model.

**INS-A6. Borrowing to insure** (a case of CRD-A1). *Kind:* hazard. *Chain:* the
credit round comes before the cover round (7.1.2), so a household can pay a premium
from money it borrowed this tick: a debt that buys cover, repaid from a trip's
proceeds that the cover replaces if the cargo is lost. The debt service squeeze of
CRD-A1 is unchanged and the payout is the insured's one income if the cargo is lost.
*Delay:* the loan's term (`SlackTicks`) against the shipment's flight. *Owner:* CRD 6.1,
INS 7.1.2. *Damper:* CRD-D1 to D3 (the shield, no rollover, the exposure cap).
*Undamped:* as CRD-A1. *Condition:* CRD-A1's (the debt service below the trip's proceeds), with the premium added to what is borrowed; not run. *Test:* INS-S2. *Status:* hand.

**INS-A7. A payout is income in the arrival tick** (a balancing link of RSK-A4's chain, entered as a case of it and not as a loop of its own, checklist 9.9). *Kind:*
link (a case of RSK-A4). *Chain:* a lost cargo (+) the shipper's loss of income (-) its welfare ratio (-)
the privation count (+) the ladder, relief and removal (+) the lane's carriers
(RSK-A4's selection of survivors). A payout replaces income (11.1.2): it **cuts**
the chain at its first link, so an insured lane keeps carriers that an uninsured one
loses, and the statistic over its survivors does not overstate its return (RSK-A4's
selection effect is weaker). *Polarity:* **B** (a damper of RSK-A4's ruin), with a
side effect: a lane whose losses are paid keeps its carriers where the loss would
have taken them off it, and the price signal of the loss does not reach the road.
*Delay:* the privation count, longer than the window. *Owner:* INS 11.1; RSK 5.
*Damper:* none is needed: it is itself a damper of RSK-A4's ruin, so it has no *Undamped* case. *Condition:* the payout (`CoverBP` of the loss value, at most `MaxPayout`) less the premium keeps the arrival tick's ratio above the ladder's threshold, so the retained `1 - CoverBP` of the loss and the premium must themselves be affordable. *Test:* INS-S3 over seeds (the removal rate of
shippers with and without cover). *Status:* hand.

**Dampers.** *INS-D1* the cover share `CoverBP` below 10 000: the insured keeps
`1 - CoverBP` of every loss and a premium on top, so a loss is never a gain
(the one moral-hazard damper); *INS-D2*, which **is CRD-D1**, the shield of CRD (an underwriter's payout
stops at what it needs to eat; listed here for the scenarios, not a new damper); *INS-D3* the capacity cap `CapBP` against available
money, a leverage of the pack's choosing, a damper of the **shortfall** (INS-A4) and not of the bust of INS-A2 (a number above 10 000 is a statement that
the pack accepts the shortfall of INS-A4, model part 6); *INS-D4* the value cap `ValueCapBP`; *INS-D5* the dispatch
rule (no cover for goods already on the road, 3.1.3) and the single-event shape (no
renewal, no tail); *INS-D6* no endogenous premium (position 10), a constraint and not
a stabiliser; *INS-D7*, which is **CRD-D2 applied to a payout claim**, the refusal `in_arrears` of an underwriter that owes a payout.

**Scenarios proposed** (numbered by the supervisor; each runs with the module on over
a range of seeds with `WithAudit` and asserts the identities of 9.4 at every tick):
*INS-S1* neutrality and accounting: the module on with no role block, against the
module off, bit for bit (FR-INS-005), and the reference chain with the trade overlay,
a route row and a store row: premiums and payouts against the events and the
identities of 9.4; *INS-S2* an insured merchant lane with a shortfall: an
underwriter at a capital below its book, a strike, the payout claim, its settlement
at P1.7a and a write-off, a loan to pay the premium (INS-A6); *INS-S3* a merchant lane
with and without cover (shipped, landed, the recovery and the premium, the removal
rate of shippers; INS-A1, A5, A7); *INS-S4* a common row over several underwriters
with `CapBP` at 5 000, 10 000 and 30 000 (removals, shortfall claims, cover refused;
INS-A2, A4); *INS-S5* a scripted price spike (a consumer's shock of a yield) with and
without `ValueCapBP` (INS-A3); *INS-S6* a checkpoint at a boundary with route
policies and a payout claim open: the restored run's policies, payouts and events
equal the uninterrupted run's.

## 18. What was run

One script, in Docker with `PYTHONDONTWRITEBYTECODE=1`, `doc/specs/models/ins/cover.py`
with its output `cover.out` (`doc/specs/models/README.md` gains a row, section 20).
It is a model of the **rules** of sections 6 to 8 with integers and placeholder
numbers, and of two stylised loops with floats and random numbers of Python's own
generator (a model's, not the library's). It shows arithmetic and structure. It does
**not** show that any default is stable, nor how the library behaves: nothing has run
in it.

| Part | What | Result |
|------|------|--------|
| 1 | the premium of a route cover on the lane of `spec-risk.md` 18.1 | `L` 1 305; sum 1 134; rate 1 357 bp; premium 154 against a fair 118 (the loading 13 000 is exactly 1.3 times fair; 154 over 118 is 1.305 because both are rounded to whole units); maximum payout 907 |
| 2 | the claim at the arrival of the same shipment (9, 1, 0, 2 units lost; grain's published price has risen to 14) | loss value 246 (grain valued at the cap 12); due 197 |
| 3 | one underwriter's three policies at one position: the pot, the shield, the pro rata allocation | balance 1 000: paid in full; 400: shares 135, 103, 62 and claims 62, 47, 28; 150: shares 23, 17, 10; 80: nothing paid, three claims. A removal's waterfall pays 20, 30, 100 of 40, 60, 200 |
| 4 | a store cover on the held goods of `spec-risk.md` 18.2 | held value 6 250; sum 3 125; premium 47; maximum 2 500; loss value 3 353; due 2 500 (the cap binds) |
| 5 | the value cap against a published price rising from 10 to 40 | payout 80, 96, 96, 96, 96 |
| 6 | an underwriter's capital under unit and common rows, `CapBP` 5 000 to 40 000, 300 seeds, 300 ticks | removed by 300: unit 91, 25, 8, 6; common 210, 235, 202, 200; mean shortfall a run 0.0, 0.1 (common, 10 000), 566 and 704 (common, 20 000 and 40 000) |
| 7 | the shipper that reads the loss, ignores it or nets the recovery | largest gain that decays 0.592 (ignores), 0.790 (reads), 0.624 (nets); margin with cover and premium -46.6, -6.6, -39.3 a unit |

**What the model does not show.** It does not show the library's draws (it has none),
the interaction of a payout with the belief rule, the ladder, the merchant's lot cap or
the credit round, or INS-A3's closing link, INS-A5 or INS-A6: those are hand entries.
Part 6 has one underwriter and an invented demand of at most 20 policies a tick, and
its removals include upkeep and the variance of a thin book; the contrast between
unit and common rows and the effect of leverage are what it is for. Part 7 is a linear
model of an invented rule, as RSK's and TRN's are.

## 19. Worked examples

The settings are invented **examples**. The arithmetic is `models/ins/cover.py`.

### 19.1 Example A: a route policy, a loss and a pro rata payout

The lane of `spec-risk.md` 18.1: regions `north` and `south` joined by `mid`, the hired
mode `wagon`, `RouteLoss` 1 305. The shipper holds a role with an `Insure` block
(`MinLossBP` 1 000, `RuinShareBP` 2 500, `MaxLoadingBP` 15 000, `MinCoverBP` 5 000)
and the underwriter a role with an `Underwrite` block (`LoadingBP` 13 000,
`CoverBP` 8 000, `CapBP` 10 000). `ValueCapBP` is 12 000.

1. **P3.2.** The shipper dispatches 40 `grain`, 5 `cloth`, 2 `rope` and 6 `tar` to
   `south` and its default rule appends a lane request: `L` 1 305 is above 1 000 and
   the cargo (value 1 134 at the origin) is above 25% of its money 4 000.
2. **P3.3c.** The shipment is a candidate. Unit values: `grain` min(12, 12 000 * 10
   / 10 000 = 12) = 12, `cloth` min(90, 96) = 90, `rope` 30, `tar` min(25, 24) = 24:
   sum insured `480 + 450 + 60 + 144 = 1 134`. Rate `halfEven(1 305 * 8 000 * 13 000
   / 10^8) = 1 357` bp; premium `halfEven(1 134 * 1 305 * 8 000 * 13 000 / 10^12) =
   154`; maximum payout `halfEven(8 000 * 1 134 / 10 000) = 907`. The premium moves
   from the shipper to the underwriter, whose balance is 2 000: its capacity is 10 000 *
   2 000 / 10 000 = 2 000, of which two earlier policies (maximum payouts 400 and 300)
   hold 700, so the 907 of this one fits in the 1 300 left.
3. **Dispatch (already past).** RSK fixed `LossBP` 2 500 at P3.2(c); this module read
   nothing of it.
4. **P1.3 (arrival, tick 15).** RSK's step 0 took 9 `grain`, 1 `cloth`, 0 `rope` and
   2 `tar`.
5. **P1.3a.** `grain`'s published price at `north` is now 14, valued at the cap 12:
   loss value `9 * 12 + 1 * 90 + 0 + 2 * 24 = 246`; due `min(907,
   halfEven(8 000 * 246 / 10 000) = 197) = 197`. The underwriter's two earlier policies are also due at this position with 150 and 90, so
   `owed` is 437. It has spent money since the round (INS-A4): with a balance of 1 000
   and a shield of 100 its pot is 900 and it pays all 437.
6. **A poorer underwriter.** With a balance of 400 and a shield of 100 the pot is 300,
   below 437: the shares are `floor(300 * 197 / 437) = 135`, 102 and 61 with 2 left to
   the largest remainders: 135, 103 and 62. The shortfalls 62, 47 and 28 become three
   payout claims due at tick 16, paid by P1.7a from the money above the shield and
   overdue if not, barring the underwriter's next offer.

### 19.2 Example B: a store policy at the cap

A household with 250 `grain` (published 11, reference 10) and 50 `cloth` (published
70, reference 80) and `StoreShareBP` 5 000. Held value `250 * 11 + 50 * 70 = 6 250`;
sum `3 125`; at a store rate of 150 bp the premium is `halfEven(3 125 * 150 /
10 000) = 47`; maximum `halfEven(8 000 * 3 125 / 10 000) = 2 500`. At P8.1a the
strikes of `spec-risk.md` 18.2 take 133 `grain` and 27 `cloth`: loss value `133 * 11 +
27 * 70 = 3 353`, `halfEven(8 000 * 3 353 / 10 000) = 2 682`, due `min(2 500, 2 682) =
2 500`: the household recovers 2 500 of a loss of 3 353 and paid 47.

### 19.3 Example C: a removed underwriter

An underwriter with 150 money owes payout claims of 40, 60 and 200 (a total 300) and
has an open route policy. It is removed at P7.1 (it could not eat). Step 2d of CRD
pays pro rata with no shield: 20, 30 and 100; the rest (20, 30, 100) is lost. Step 2f
ends its open route policy with the cause `underwriter_removed`: no refund, no
payout. The estate gets what is left of its balance, which is 0.

## 20. Notes settled, and what this spec needs from the other specs

### 20.1 Notes

No open note is tagged `INS`. The notes tagged `ALL` and the others read are:

| Note | Disposition | Where |
|------|-------------|-------|
| N-046 (EMP, SAV, CRD, STO, RSK, FIS, PLC, LND) | **Applied for INS** though not tagged: entries INS-A1 to A7, dampers INS-D1 to D7, scenarios INS-S1 to S6, provisional | 17 |
| N-016 (ALL, CFG, WEL) | **Applied (rows)**: the owner-table rows of 13; no unnamed parameter; the registry test is still to be written | 13 |
| N-017 (ALL) | **Applied**: terms in 20.4; setting words only in flagged examples | 20.4 |
| N-027 (ALL, REF, STY) | **Applied**: every number is a placeholder; no default is called stable | 13, 18 |
| N-103 (ALL) | **Applied**: provisional IDs; seams in the handoff; two slips recorded in `.supervisor/ins.progress.md` | 17 |
| N-126 (ALL) | **Applied**: rows for the event, key and metric tables in 15 and 20.2; no `Warn` is added | 15 |
| N-128 (ALL, CHK) | **Applied**: no per-agent field; a policy is saved once, in the module's section; a payout claim once, in the ledger's | 16 |
| Read, not tagged INS | N-011 (cargo lost is `loss`; the money form of a payout is not a faucet); N-021 and N-107 (no faucet, no sink, no boundary; two internal reasons, one operation); N-032 (credit, not insurance, is the remedy for the cash gap); N-054 (a payout and a premium redistribute and create nothing); N-090 (no `region_bound` refusal, 8.7); N-120 (a migrant is not exposed, so not insurable); N-122 (settlement pro rata and matching by limit then key: no place added to WLD 10.2); N-124 (no draw, so no knob has the flag `d`); N-125 (snapshot fields are functions of saved state); N-130 (a loss ratio is exact from `CoverSettled` and `CoverFormed`, 20.2); N-131 (a controller that sets premiums is outside the engine) | |

### 20.2 What this spec needs from the other specs

| Spec | Section | Edit |
|------|---------|------|
| `spec-world.md` | 6.3, 6.5, 6.8, 6.10, 6.11 | rows P1.3a (after P1.3, before P1.4), P3.3c (after P3.3b, before P3.4) and P8.1b (after P8.1a, before P8.2); the event order of 12.1.1; 7.5 lists "the cover round's order and the settlement's remainders" among what does not draw; 11.1 the cost rows; **6.8, row P6.3**: "a flow of the insurance result that moves after P6.3 (P7.1, P8.1b) is added to the record of its tick by amending it" |
| `spec-agents.md` | 3.6, 6.3, 6.4.1, 13, 14 | removal step **2f** (after CRD's 2d and 2e); the plan fields `Cover`, `Store`, `Offer`; the view group `Cover`; the errors and the `Debug` refusals; the welfare record's `N_t` is WEL's |
| `spec-risk.md` | 5, 10.2, 10.3 | P1.3 keeps, for each arrived shipment, the lost units by commodity (lines and account) in scratch **whether or not an observer was supplied**, and P8.1a the lost units of each struck agent; the row INS of 10.3 says "used"; 10.2.1's list of reads is the one this spec uses |
| `spec-transport.md` | 5.7, 9.2 | the arrival's scratch above; nothing else: no step of dispatch is added (5.1.2 here) |
| `spec-credit.md` | 3.1, 3.2, 3.3.2, 3.6, 6.4 step 1, 11.1, 13.2, 16.1.3, 16.3, 16.3.1, 16.4, 17 | the operation `Obligate` and the field `Origin` (3.1, 3.2); the total `Awarded` and the identity of 3.6 item 1 (3.3.2, 3.6); `MaxClaims` counts payout claims (6.4 step 1); the principal paid on a payout claim is income and cost (11.1); `ForgiveClaims` ends loans only (13.2); **16.1.3**: a payout claim is added to an observer's open claims by `CoverEnded.Short` with its `Claim`, and it ends in a `LoanEnded` of origin `payout`; its payments are INS's `CoverSettled` of stage `claim`; **16.3**: the credit instruments count origin `loan` only (`CreditSettled` counts loans only), and the claims outstanding are INS's `economysim.insurance.claims`; **16.4**: the `Credit` row gains `Awarded` and the `Claims` row `Origin`; `LoanEnded` carries `Origin`; the ledger section saves both (17). The pass of P1.7a and the waterfall of 8 and 9 need no edit: they pay a claim of either origin (8.4.1) |
| `spec-ledger.md` | 7.2, 10 | the reasons `premium` and `payout` (internal, money); the operation `Obligate`; "claims" are of two origins |
| `spec-merchants.md` | 5, 9.7 | the default `Merchant` calls `DefaultCover` for the lanes of its trips (an edit to the strategy and not to the rules); 9.7's table is unchanged (the payout is the shipper's); the margin of 5.2 stays net of the expected loss and does not include cover (INS-A5) |
| `spec-storage.md` | 8.3, 10 | the seam row for INS: the owner's held value and the stores by class are read, nothing else |
| `spec-welfare-and-wages.md` | 2.1, 4.3, 13 | `N_t` (payouts, premiums and principal of payout claims received less premiums, payouts and principal paid; an underwriter's premiums received are in `N_t` only) is a term of the signed `G_t`, so of `income_t` and out of the observed wage; the record gains no field (WEL 2.1.9, 13); a flow of `N_t` that moves after P6.3 (the payout of a store policy at P8.1b, the principal a removal's waterfall pays at P7.1) is added to the record of the same tick by amending the last record, no new state (INS 11.1.3) |
| `spec-observability.md` | 4.1, 5.6, 6.4.2, 7 | the four events in 4.1 (owner, position, class, gate); the sixth in-flight set `Policies`; `Insurance` totals; the metrics and the three attributes; the keys; the restore seeding of the liability; no `Warn` row |
| `spec-checkpoint.md` | 4.2, 6.2, 9.3 | a module section (tag 29) after Risk and Credit; the claim's `Origin` and the total `Awarded` in the ledger section; the validation of 16.5; no format bump while nothing has shipped (9.3) |
| `spec-config.md` | 3.7, 4.1, 7.3, 6 | the fields of 13; the bounds; `ErrCoverShape`; the rules and warnings of 14; the file layout `[modules.insurance]`, `[role.<id>.cover.underwrite]`, `[role.<id>.cover.insure]` |
| `spec-integration.md` | 6.2, 11.2 | `AgentState` gains the group `Cover` (`Policies`, `Open`, 16.3.1; a read); `ErrCoverShape` is not a setter's error; no slot, no call |
| `spec-indicators.md` | 7.1, 9.1 | the account `Premium` and `Payout` and a preset: premiums, payouts, shortfalls and the loss ratio by region and kind, from the events; the loss ratio is exact from `CoverSettled` and the snapshot |
| `spec-stability.md` | 4, 5, 8 | the entries, dampers and scenarios of 17, numbered after the ones already integrated and the provisional ones of the other later-stage specs |
| `spec-stylised-facts.md` | 5.1, 5.6 | none required: a delivered price converges within freight and the expected loss, which cover does not change; a note that a pack with cover is a different pack |
| `spec-calibration.md` | 6 | the registry rows of 20.3: `ValueCapBP`, `CoverBP` and `CapBP` damper strengths (INS-A3, A1, A4), `LoadingBP` and `StoreRateBP` structural choices, the rest structural |
| `spec-viability.md` | none | the steady state has no cover: a policy is a transfer and a loss is RSK's expected loss, already in the lane |
| `spec-polity.md`, `spec-settlements.md`, `spec-institutions.md` | none | ORG's price control on a loading is a refusal on the offer (11.2) |
| `spec-reference-chain.md` | 7, 9 | an overlay `overlay/cover.toml` and the scenarios of 17 |
| `specs/README.md`, `roadmap.md`, `documentation_index.md`, `models/README.md`, `notes.md` | | the index row, the code row, the owner rows, the terms, the decisions; the status; the rows; the notes of the handoff |
| `rules/observability-and-logging.md` | 3.3.2, 4.3, 5.3, 5.4.1 | the keys, the events, the metrics, the attributes (a rules change, with a fresh tag) |

### 20.3 Note on the registry of knobs (CLB)

For `spec-calibration.md` 6 (a registry row for each field of 13): `ValueCapBP`
is a **damper strength** of INS-A3, `CoverBP` of INS-A1 and A5, `CapBP` of INS-A4
(INS-A2 is undamped in version 1, 17); `LoadingBP` and `StoreRateBP` are **structural choices** (a posted price is a fact
of a pack's insurers); `PayoutTicks`, `MinSum`, `MaxPolicies`, `MaxCoverRequests` and
every field of an `Insure` block are structural. No field moves the steady state of the
viability programme (`spec-viability.md` 11). The module takes no draw, so no field has
the flag `d`.

### 20.4 Terms (N-017)

**cover** (the module and the act), **policy** (a cover that was formed, qualified "cover policy" where it could be taken for
PLC's `Policy`, the interface of a policy actor; **route** and **store** kinds; the
log key is `cover_policy`), **underwriter** and **insured** (the parties; roles with an
`Underwrite` or an `Insure` block), **premium**, **sum insured**, **maximum payout**,
**cover share** (`CoverBP`, a coinsurance), **loading** (the share of the expected
payout a premium is), **store rate**, **capacity** and **open liability** (of an
underwriter), **unit value**, **value cap**, **loss value**, **due** (of a policy),
**payout** (the money paid under a policy), **payout claim** (the ledger claim of an
unpaid part), **shortfall**, **round** (of cover; its lane is a region, the word of
MER, CRD and EMP in the qualified spelling "lane of the cover round"; a **lane
request** names a mode and a destination, which is MER's lane in the other sense).
The words **claim** (a demand for a payout) and **insurer** are not used for these;
**claim** is the ledger's. The words **wagon**, **granary** and **guild** are setting
words: they appear only in the flagged examples. `spec-storage.md` 4.1.3's "credit" of
goods and `spec-credit.md`'s "credit" are different words, and neither is used for a
premium or a payout.


## 21. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. The IDs are
grouped by area, in blocks of ten: 000s the module and the ledger, 010s roles and
admission, 020s value and price, 030s the round, 040s settlement, 050s ends and the
other changes, 060s the strategy and welfare, 070s observability, 080s state and
invariants. A gap is a number not yet used, not a missing requirement.

### Functional

| ID | Requirement |
|----|-------------|
| FR-INS-001 | The module MUST run if and only if `Config.Modules.Insurance` is non-nil and `Modules.Risk` and `Modules.Credit` are present, and a world with it nil MUST have no position P1.3a, P3.3c or P8.1b, no policy, neither of the two reasons, no draw for it and none of its events, and MUST be bit-identical to a world built without this spec (4.1, 12). |
| FR-INS-002 | The reasons `premium` and `payout` MUST be internal, money only, Agent account to Agent account, and every other pairing MUST be refused `ErrReasonMismatch` (9.1). |
| FR-INS-003 | No operation of the module MUST create or destroy money or goods: the total of money over non-boundary accounts MUST be unchanged by a premium, a payout and `Obligate`, and the module MUST add no faucet, no sink and no boundary account (9.4). |
| FR-INS-004 | `Obligate` MUST append a claim with no transfer, `Interest` 0 and the origin `payout`, and MUST be refused `ErrInvalidAmount`, `ErrClaimShape`, `ErrUnknownAccount`, `ErrClosedAccount` or `ErrTooManyClaims` leaving the ledger unchanged (9.2). |
| FR-INS-005 | A world with the module on and no role with a block MUST equal the world with it off in every `Snapshot` and every event of the other specs, apart from the fields of this module, which are 0 or empty, and from the configuration hash that `WorldCreated` and the checkpoint header carry, which covers the module's block whether or not a role has a block (FR-INS-008) (4.1.1, 12.4). |
| FR-INS-006 | The identity of the credit list MUST read `Σ PrincipalOwed = Lent + Awarded - Repaid - LostPrincipal` exactly, and the identity `Due = Paid + Short + Dropped + Cut` MUST hold over the run (9.2.3, 9.4). |
| FR-INS-007 | `ForgiveClaims` MUST end loans only and MUST NOT end a payout claim (9.2.2). |
| FR-INS-008 | The configuration of the module and of every block MUST be in the configuration hash (13, 16.5). |
| FR-INS-010 | A role with a cover block MUST have exactly one of `Underwrite` and `Insure`, and both or neither MUST be `ErrCoverShape` (4.2.1, 14.1). |
| FR-INS-011 | A cover block on a role while `Modules.Insurance` is nil, a module without risk or credit, and an `Insure` block that needs transport or storage that is absent MUST be `ErrModuleDependency`; a field outside its bound MUST be `ErrOutOfBounds`; an omitted `loading_bp`, `store_rate_bp`, `min_loss_bp`, `ruin_share_bp`, `max_loading_bp` or `max_store_rate_bp` MUST be `ErrMissingField`, and an omitted required field whose bound starts at 1 MUST be `ErrOutOfBounds` (13, 14.1). |
| FR-INS-012 | A plan MUST carry at most `MaxCoverRequests` admitted requests and at most one admitted offer, and a second offer or a repeated lane MUST be refused `duplicate` (5.2). |
| FR-INS-013 | The admission checks of 5.2 MUST be made in the order of its table, each refusing whole with its `reason`, and a refusal MUST change nothing and MUST NOT be an error of `Step` (5.2). |
| FR-INS-014 | An offer whose `MaxLiability` plus the underwriter's open liability exceeds `CapBP` times its available money MUST be refused `exposure` and MUST NOT be cut (5.2.2). |
| FR-INS-015 | An offer of an agent with an overdue claim as debtor, a loan or a payout claim, MUST be refused `in_arrears` (5.2). |
| FR-INS-016 | A store request of an agent that owns no store MUST be refused `no_store`, and with the module off every request and offer MUST be refused `insurance_off` and cost nothing else (5.2). |
| FR-INS-020 | The unit value MUST be `min(published, trunc(ValueCapBP * reference / 10 000))` of the commodity in the region, with the published price the one of the last clearing before the position (the previous tick's P4.1 at P3.3c and P1.3a, this tick's at P8.1b) and the reference price the effective one (6.2.1). |
| FR-INS-021 | The sum insured MUST be, by the shape of the shipment, the lines term `Σ q_c * v(From, c)` for a hired shipment, the account term (the traveller's holdings except the mode's `Vehicle`, at `v(From, c)`) for a self-carried trip to oneself, and **both** for a self-carried trip that consigns lines to another agent, all at P3.3c, and that of a store policy `Held * ShareBP / 10 000` truncated (6.3). |
| FR-INS-022 | The premium of a route cover MUST be `halfEven(S * L * CoverBP * LoadingBP / 10^12)` with `L` the lane's `RouteLoss`, computed with a 128-bit intermediate and checked, and that of a store cover `halfEven(S * StoreRateBP / 10 000)` (6.4). |
| FR-INS-023 | The maximum payout MUST be `halfEven(CoverBP * S / 10 000)`, fixed at formation and never raised (6.4.1). |
| FR-INS-024 | A candidate with `Sum` below `MinSum`, a premium below 1 or a lane of expected loss 0 MUST NOT be formed and MUST be counted (`below_minimum`, `no_risk`), and a candidate whose sum or premium passes `MaxMoney` MUST NOT be formed and MUST be counted `overflow` (6.3.5, 6.4.4, 7.2.2, 7.4). |
| FR-INS-025 | No rule of the module MUST read a shipment's `LossBP`, a draw, a `Common` flag, a strike probability or a scale; two worlds that differ only in the strikes of a lane MUST form the same policies (6.1.2). |
| FR-INS-030 | The round MUST run once per tick at P3.3c, after P3.3b and before P3.4, only with the module present, and MUST move only available money (7.1). |
| FR-INS-031 | A request MUST be matched only with offers whose agent's account is in the same region at the round, and the lanes MUST be visited in ascending `RegionID` (7.2.1). |
| FR-INS-032 | The candidates of a lane request MUST be the shipments of the agent dispatched in this tick on the lane that have no policy, in ascending shipment ID, and a shipment dispatched in an earlier tick MUST NOT be covered (3.1.3, 7.2.2). |
| FR-INS-033 | The requests MUST be ordered by limit descending then `K(t, agent)` then agent ID then plan order, the offers by loading (or store rate) ascending then `K(t, agent)` then agent ID, lane requests before store requests, and the round MUST take no word of the world's stream (7.3, 12.2). |
| FR-INS-034 | An offer MUST be skipped when it is the insured's own, its `CoverBP` is below the request's `MinCoverBP`, or its remaining `MaxLiability` is below the pair's maximum payout (the offer's `CoverBP` applied to the candidate's sum), and the scan MUST end at the first offer above the request's limit (7.3.2). |
| FR-INS-035 | Formation MUST check, in order, `too_many_policies`, `below_minimum`, `funds` and `overflow`, after the sum has been computed with checked arithmetic, and a failure MUST change nothing (7.4). |
| FR-INS-036 | A premium MUST be paid with the reason `premium` from the insured's available money to the underwriter in the operation that appends the policy, all or nothing (7.4). |
| FR-INS-040 | A route policy MUST be settled at P1.3a after P1.3 and before P1.4, and a store policy at P8.1b after P8.1a and before P8.2 (8.1). |
| FR-INS-041 | The loss value MUST be the sum of the units RSK took (a shipment's `Lost` lines where it has lines and its `AccountLost` where it is self-carried, a struck agent's `StoreLoss` amounts) at the unit values of the settlement, priced at the origin region or the store's, and a loss value above `MaxMoney` MUST saturate at `MaxMoney` (8.2.1, 6.3.5). |
| FR-INS-042 | The due MUST be `min(MaxPayout, halfEven(CoverBP * LossValue / 10 000))`, and a policy with no loss MUST end `no_loss` (8.2). |
| FR-INS-043 | Per underwriter and position the pot MUST be `max(0, balance recorded at entry - ShieldTicks * upkeep cost)`, the payment `min(pot, owed)`, and a payment short of `owed` MUST be shared pro rata by due with the largest remainder, ties by `K(t, policy ID)` then policy ID (8.3). |
| FR-INS-044 | A payment MUST be made with the reason `payout` to the insured, cut to its headroom with the part not paid dropped and counted (8.3.4). |
| FR-INS-045 | The unpaid part of a due MUST be recorded by `Obligate(insured, underwriter, short, t + PayoutTicks)`, and when the list is full it MUST be dropped and counted (8.4). |
| FR-INS-046 | A payout claim MUST be settled, barred, written off, passed to the estate and paid by the waterfall by the rules of the credit module without change (8.4.1). |
| FR-INS-050 | A policy MUST end with exactly one cause of `no_loss`, `claimed`, `insured_removed` and `underwriter_removed`, and a route policy MUST end at the arrival of its shipment or at a removal (8.5). |
| FR-INS-051 | The removal of an agent MUST end its open policies in step 2f after steps 2d and 2e, in ascending policy ID, with no transfer and no refund of a premium (8.6). |
| FR-INS-052 | A weight change, a role change, a relocation, a trip and a journey MUST change no policy and none MUST be refused by one, and the module MUST register no `region_bound` refusal (8.7). |
| FR-INS-053 | A shipment with lines whose shipper was removed MUST arrive and be struck as before and MUST pay nothing, and a shipment to oneself MUST be dropped with its traveller, with no loss and no settlement (8.6.2). |
| FR-INS-060 | The view of an agent with a cover block MUST carry the group of 10.1 and MUST carry no other agent's policy, no offer, no loading and no count of underwriters (10.1). |
| FR-INS-061 | The default insured MUST request a lane only when `MinLossBP` is above 0 and its expected loss is at least `MinLossBP` and the cargo's value at least `RuinShareBP` of its money, and a store cover only with `StoreShareBP` above 0 and a store owned (10.2). |
| FR-INS-062 | The default underwriter MUST offer `ShareBP` of `max(0, CapBP * max(0, Money - Bids) / 10 000 - Open)`, with `Bids` the reserved cash of its plan, and nothing below `MinSum` or while it owes a payout claim (10.3). |
| FR-INS-063 | `DefaultCover` and `DefaultOffer` MUST be exported functions of the view (10.4). |
| FR-INS-064 | With the module on `G_t`, and so `income_t`, MUST include `N_t`, the payouts, the premiums and the principal of payout claims received less the premiums, payouts and principal of payout claims paid (an underwriter's premiums received are in `N_t` and in no other term), and `N_t` MUST be excluded from the observed wage, and a flow that moves after P6.3 (P7.1, P8.1b) MUST be added to the record of the same tick already pushed, by amending it, with no state of its own (11.1, 11.1.3). |
| FR-INS-070 | `CoverFormed`, `CoverEnded`, `CoverRound` and `CoverSettled` MUST be emitted with the fields, positions and order of 15.1 and 12.1.1, and none MUST contain a map, a pointer or a wall-clock time. |
| FR-INS-071 | A store policy that ends `no_loss` MUST NOT emit `CoverEnded` and MUST be counted in `CoverSettled.Ended` (15.1.1). |
| FR-INS-072 | `CoverRound` MUST be emitted per region and tick in which a request or an offer was submitted and never when all its counts are 0, and `CoverSettled` per region and position in which a policy was settled and never when all are 0 (15.1.1). |
| FR-INS-073 | The nine metrics of 15.3 MUST be derivable from the events and the snapshot, with attributes from closed sets. |
| FR-INS-074 | The module MUST emit no `Info` and no `Warn` inside the tick (15.2). |
| FR-INS-075 | The log keys `cover_policy`, `sum`, `premium`, `payout`, `underwriters` and `insured` MUST be added to the keys of the rules (15.2). |
| FR-INS-076 | The snapshot MUST carry `Policies` and the `Insurance` totals, deep copies bounded by `MaxPolicies` and functions of saved state (15.4). |
| FR-INS-077 | An observer given to a restore MUST be able to seed `economysim.insurance.liability` from `WorldRestored` (15.3). |
| FR-INS-078 | The construction summary MUST name `insurance` in `modules` and carry `underwriters` and `insured` (15.2). |
| FR-INS-079 | The money of the household sector by region MUST be rebuildable from `CoverFormed`, `CoverEnded` (its `Paid` by `UnderwriterRegion` and `Region`) and, for a payout claim, `CoverSettled` of stage `claim` and the `LoanEnded` rows of origin `payout` (15.1.2). |
| FR-INS-080 | The module MUST save the next policy ID, the open route policies and the six totals (`Short` is the ledger's `Awarded`), MUST save no store policy and no scratch, and MUST save a claim's `Origin` in the ledger's section (16.1). |
| FR-INS-081 | The audit MUST check the identities of 9.4 with `WithAudit` (9.4). |
| FR-INS-082 | The restore MUST reject, with `ErrCheckpointCorrupt`, a policy whose accounts are not open Agent accounts or are equal, whose shipment is not in flight or has another shipper, whose `MaxPayout` is not `halfEven(CoverBP * Sum / 10 000)`, an unordered ID and a count above `MaxPolicies` (16.5). |
| FR-INS-087 | A restored world MUST form, settle and pay policies and emit events as the uninterrupted one (16.2). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-INS-001 | With the default logger and observer, P1.3a, P3.3c and P8.1b MUST allocate nothing per call beyond scratch the world holds, and a full step of the reference chain MUST allocate no more than the baseline of the module off (observability-and-logging.md 8.2). |
| NFR-INS-002 | The work of P3.3c MUST be at most `O(R + O)` per lane for the sorts and `O(candidates * scanned offers)` for the matching, and that of P1.3a at most the arrived insured shipments and the underwriters paid (12.4). |
| NFR-INS-003 | A run MUST be bit-identical for a given seed, configuration and sequence of calls across platforms: no floating point, no map order, no clock (architecture.md 7.4). |
| NFR-INS-004 | A test that runs over seeds MUST report the failing seed (testing.md 6.2). |
| NFR-INS-005 | No rule of the module MUST branch on a commodity, a mode, a region or a role identifier (architecture.md 9.1), and every product that this spec names MUST be computed with checked arithmetic, with the outcome of 6.3.5 when it passes `MaxMoney` (architecture.md 3.5). |

## 22. Open questions

1. **A reserve** (3.5 B). `ReserveBP` of the maximum payout moved to an Escrow account of
   the policy at formation, so that cover is funded and an insured's recovery does not
   depend on the underwriter's later spending (INS-A4). It adds a second money state and a
   case at every removal; reopen when a run shows INS-A4 or A2 to matter.
2. **Reinsurance** (4.2.1): an underwriter that insures with another. A chain whose failure
   is the worst case of INS-A2; a role with both blocks, and a rule against a cycle.
3. **Experience rating** (position 10): a premium that follows an underwriter's paid
   losses or an insured's record. It closes the reinforcing chain of the brief (cover,
   exposure, payouts, premium) and needs per-agent or per-lane state and a rate from an
   aggregate (WLD 9.4). Reopen with a run that shows the unpriced correlation of INS-A2.
4. **A store's expected loss** (6.4.2). A store has no read like `RouteLoss`
   (`spec-risk.md` 8.5), so its rate is data. An asked-for read of RSK would let a store
   premium follow a row and its scale as a route's does.
5. **Correlation in the price** (6.4.3): a loading that reads whether a lane crosses a
   common row, from the configuration the world holds. It would price INS-A2's cause
   and give the underwriter a reason to hold fewer policies on one lane.
6. **Payout in goods or in kind** (3.5 D): a replacement from the underwriter's stock.
   Needs a valuation the engine refuses and an obligation in goods.
7. **Cover across regions** (7.2.1): an underwriter in one region and an insured in
   another. Needs the underwriter to judge a risk it cannot see; rumours (INF) could
   supply it.
8. **A decree that cancels debts** (9.2.2): whether a payout claim is among the debts. The
   choice here is that it is not.
9. **A player's cover**: optional fields on the `Directive` (10.5).
10. **A cover on a shipment in flight** (3.1.3): needs a repricing against the scale
    set since dispatch.
11. **A longer policy** (3.1.2): a term, a renewal and a price that follows the scale.
12. **A deductible or a franchise** (INS-D1): a fixed amount before the cover pays.
    `CoverBP` is the one damper of version 1; a deductible adds a number and no
    structure.
13. **Cover of a loan** (CRD): an underwriter that pays a creditor when a debtor
    defaults. It is a payout on a claim event and not on a loss of goods; needs a
    credit event as the trigger and a moral hazard of the borrower that CRD has no
    answer for.
14. **Experience of the claim's due tick** (8.4): `PayoutTicks` of 1 matches the shield's
    bite; whether a longer term helps a thin underwriter is a number to find.

## 23. Where this lives in the code

Nothing is written yet (architecture.md 2.6). When it is: `insurance.go` in the root
package (`Insurance`, `CoverTerms`, `UnderwriteTerms`, `InsureTerms`, `LaneCover`,
`StoreCover`, `CoverOffer`, `DefaultCover`, `DefaultOffer`, the sentinels, the events
`CoverFormed`, `CoverEnded`, `CoverRound` and `CoverSettled`) and `internal/insure`
(stdlib only, over primitive slices: the unit value, the premium and the maximum with
the checked 128-bit helper, the loss value, the due, the matching and the pro rata
allocation reusing `internal/credit`'s helper). The operation `Obligate` and the field
`Origin` are `internal/ledger`'s, with CRD's. The log keys are in `internal/logkeys`.
The terms of 20.4 go to the `doc.go` glossary before code uses them. No
`Stateful`, no extension point and no new interface (architecture.md 4.2): the module is
data and functions.

