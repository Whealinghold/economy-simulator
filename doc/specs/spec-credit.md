# Spec: Credit: claims, loans at posted rates matched in a round, interest as a transfer, settlement, arrears and the waterfall at a removal

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CRD-NNN` and
`NFR-CRD-NNN`, which tests cite (testing.md section 4). They are in section 22.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
later stage (`vision.md` section 4, "Later"). It is off unless configured:
`Modules.Credit` is a pointer, nil is off, and the zero configuration has no
claim, no lender, no interest and the empty claims list of `spec-ledger.md` 10,
exactly. The core does not depend on it. Needs `spec-ledger.md` (LED: the claims
list, accounts, the reason registry, the audit), `spec-money-supply.md` (MON: why
a loan is not a flow and why interest is no faucet, 3.4, 9.5), `spec-agents.md`
(AGT: the plan and the view of a strategy, the removal of 3.6, the default
strategy's budget of 7.4), `spec-estates.md` (EST: what passes to the estate, the
pool), `spec-welfare-and-wages.md` (WEL: income, 2.1), `spec-world.md` (WLD: the
tick), `spec-config.md` (CFG: the module switch and validation). It reads
`spec-merchants.md` (MER: the first-fit round and the escrow it shares a form
with, 9), `spec-employment.md`, `spec-land.md` and `spec-storage.md` (EMP, LND and
STO, written beside it), `spec-spawner.md` and `spec-population.md` (the ladder and
the weight changes), and asks each of the specs it touches for an edit (section 21).

**Every number in this spec is a placeholder** except where a derivation is shown.
Nothing has been run in the library, bazaarBot has not been run, and no acceptance
band exists (notes N-015, N-027). The scripts of section 19 are stylised Docker
Python models of the rules of this spec, not runs of code and not a claim that any
default is stable; the rules themselves are integer. No default is set from a
source: no historical fact about lending, interest or default was read or used
(N-018), and every default is structural (no credit, no interest, no write-off) or a
placeholder. The settings of the worked examples (section 20), a farming realm and
a mining colony, are **examples**, flagged as such. The core text and its
vocabulary are setting-agnostic: the words for who lends (a moneylender, a guild, a
temple) appear only in the examples. A commodity, a recipe, a role or a
region is data (architecture.md 9.1): nothing in this spec branches on one.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).
**Claim**, **creditor** and **debtor** are the words of `spec-ledger.md` 10 and are
kept. The word **lease** is not used (`spec-storage.md` keeps it), nor **escrow** for
a loan (it is MER's, for goods in flight), nor any setting's word for a lender role
(the examples use `financier` and say so).

## 1. Purpose and scope

`spec-ledger.md` 10 keeps a list of claims, empty in version 1, so that credit
could arrive without changing the checkpoint or the balance sheets. Every written
spec points here for the same few things the household cannot do without it: a
long job has no income for its length and the household can hold no more money
than it has (`spec-production.md` 5.2, `spec-agents.md` 7.4.3 and 11.5, N-032), a
hire needs the wage in the employer's hand (`spec-employment.md` 7.3), a merchant
cannot afford a lot (`spec-merchants.md` 9.10), a title costs more than the
household holds (`spec-land.md` 6.2). This spec supplies the one mechanism, and
nothing more. It covers:

- what a **claim** is in the ledger, the three internal reasons it adds, and what it
  does to money (it moves existing money and creates none) (section 3);
- the **roles**: who lends, who borrows, and their terms as data (section 4);
- the **request** and the **offer** a strategy appends to its plan, and their
  admission (section 5);
- the **round** that matches them at a posted rate (section 6), and the arithmetic
  of interest (section 7);
- **settlement**: repayment at the due tick, the shield, arrears and the optional
  write-off (section 8);
- what happens to a claim when its debtor or its creditor is **removed**, when a
  weight changes, a role changes and an account moves (section 9);
- what a strategy sees and the default rules for a borrower and a lender
  (section 10);
- how credit meets the welfare ratio and the neighbouring modules (section 11);
- the positions in the tick, the draws (none) and the cost (section 12), the slot
  and the call a consumer gets (section 13);
- parameters, validation, observability, saved state, the register entries, the
  models run, worked examples, notes settled, requirements, open questions and the
  home in the code (sections 14 to 24).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| A decision to save, to hoard or to invest | `spec-saving-and-investment.md` (SAV): its default asks for no loan for a project (6.2) and leaves the purchase of an asset open (6.3); this spec gives the request a strategy appends (5.1) |
| A lender that creates money against a reserve, a deposit that is a claim on a lender, a central issuer | not modelled; a setting that wants money created uses a grant of `spec-money-supply.md` (3.4) |
| Taxes on interest, a treasury that lends, a rate set by a ruler | `spec-fiscal.md` (FIS), `spec-policy.md` (PLC); the slot of 13.1 is the seam |
| Insurance of a loan or of a cargo | `spec-insurance.md` (INS) |
| Collateral: goods in custody, a title, a store | open question 3; the shape is in 3.5 |
| Credit across regions, between an agent and an outside node, and in kind | 6.2, 3.4 and open questions 4 and 6 |
| A claim that is traded, bought or sold between third parties | open question 7 |
| The public call and its guards as such | `spec-integration.md` (INT, section 13.2 here) |
| Prices, wages, the welfare ratio and the rule that sets days worked | `spec-clearing.md`, `spec-welfare-and-wages.md` |

## 2. Positions taken

The design pressures of the brief, each with a position and the reason. Later
sections give the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is credit? | A **claim**: a record in the ledger's list of an amount a **debtor** owes a **creditor**, formed by a **loan**, a transfer of money that already exists, with a fixed **interest** and a **due tick**. A household never holds a negative balance | `spec-ledger.md` 4.1, 10: a loan is a transfer and a claim, and the pair always balances. No overdraft: the engine refuses an operation and never lets a balance go below 0 |
| 2 | Does a loan create money? | **No.** The principal moves from the lender's account to the borrower's under the internal reason `loan_out`, repayment and interest move back, and `M` is unchanged by any step of this spec. A setting that wants money created has a MON grant with its own reason | MON 3.1.2 (the identity of `M`), 9.5 (a faucet proportional to a balance is an unconditional reinforcing loop); ADR 0001 decision 1. Created credit would be a second faucet with no bound but the author's care (3.4) |
| 3 | Who lends? | The households of a role with a **`Lend` block**: an agent that holds money and posts an offer. No deposit, no reserve, no special account. The estate inherits the claims of a creditor that is removed and never lends | Households only is the least that works and adds no account kind. A "lender role" is the same thing as a block on a role; the brief's two options are one |
| 4 | Who borrows? | The households of a role with a **`Borrow` block**. A role has one block or none, never both | A household that lends and borrows at once is a second decision and a self-dealing loop (4.2) |
| 5 | How are they matched? | A first-fit **round**, once a tick after the last agent's plan, in lanes that are regions, by posted rate; no negotiation, no key drawn from the world's source | The form of MER's contract round (ADR 0005), simpler because money is divisible; WLD 10.1.1 gives the tie rule that takes no draw |
| 6 | What is the price of credit? | A **posted rate**: yearly, in basis points, simple, applied to the principal for the term and **fixed at formation**; the lender states it, the borrower states the most it accepts, the lowest rate is served first. A consumer slot scales the posted rates of a region (13.1) | The rate is data like a rent (`spec-land.md` 5.2), not a measured aggregate fed back (WLD 9.4.1); an adaptive rate is a consumer's strategy or PLC |
| 7 | Is interest a faucet? | **No.** It is a transfer from debtor to creditor under `loan_interest`: the money the debtor loses the creditor gains, to the unit. MON's exclusion of a faucet proportional to a balance is about creation, and nothing is created | The sectoral identity of LED 9.3 holds unchanged; interest redistributes between households and, at an estate creditor, towards the pool |
| 8 | When is a claim repaid? | At its due tick, **automatically**, in phase 1: not a decision of the household. One pass per debtor, its due and overdue claims together, pro rata by what each owes, interest before principal | A debtor that chose not to pay would need a strategy method nobody has (architecture.md 4.2); automatic payment is what a contract is and removes strategic default from version 1 |
| 9 | What if it cannot pay? | A **shield** of `ShieldTicks` of the debtor's upkeep cost is out of reach; what cannot be paid is **overdue**, stays owed with no penalty rate, and is paid by the same pass each tick as money allows; an optional write-off ends a claim after `WriteOffTicks` | The household must be able to eat (AGT 7.4.3: tier 1 comes first); a penalty rate on arrears is a debt spiral with a gain above 1 (CRD-A1) |
| 10 | Is a loan renewed? | **Never.** The due claim is settled at P1.7a, before any request of the tick is admitted at P3.3b, and a debtor with an overdue claim is refused (`in_arrears`) | A rollover is a Ponzi path: new money pays old interest with no income behind it |
| 11 | What does a removal do? | The debtor's money pays its claims **pro rata** before the rest goes to the estate (a **waterfall**); the unpaid part is lost. The claims of a removed creditor **pass to the estate** of its region | EST 2: "no heirs, no wills": what a dead agent holds goes to the pool, and a claim is something it held |
| 12 | Collateral? | **Not in version 1.** A loan is unsecured, bounded by an exposure cap per borrower and by the lender's own reserve | A pledge needs custody of goods (a new account kind with ageing rules), or a reservation that outlives a tick (`spec-ledger.md` 6.5), or a title of LND; 3.5 gives the three candidates |
| 13 | Where is the decision? | In **`Trade`**: a strategy appends a request or an offer to its plan, and the world admits it, as it does a delivery offer. The default strategy of a role with a block appends one by a stated rule (10) | The brief and AGT 6.2: three methods and no fourth; MER's precedent for plan fields |
| 14 | Is this the remedy for a long job? | For the **cash gap** only: a household with a long job can pay its upkeep and inputs through it. It does not change how the welfare ratio reads the job (that is the derived window, `spec-welfare-and-wages.md` 2.3.3) | N-032, STB-L19: the ratio is a window over income and cost, and a principal is neither (11.1) |
| 15 | Weights, roles, relocation | A claim is the **agent's**, whatever the agent's weight, role or region; no `region_bound` refusal | N-090 and the reasoning of 9.4: a claim needs no account in a region |
| 16 | Randomness | **None.** Ties are broken by the stateless key of WLD 10.1.1 | WLD 7.4.2: a module that is on takes no word of the stream |
| 17 | New loops | Yes: the debt-service squeeze, a default cascade, a pulse of credit with the job cycle, debt deflation, money held by lenders, an eligibility ratchet (section 18) | `spec-stability.md` 9: a spec that adds a loop and no entry is not done |
| 18 | What does a consumer get? | The slot `SetRateScales`, the call `ForgiveClaims` (the only ways it changes a claim), and the claims of an agent in its state | N-091, N-131: INT is minimal and each row has a reason (13) |

## 3. The claim and the ledger

### 3.1 The record

3.1.1. A **claim** is one loan's (or one payout claim's, 3.1.5) record in the ledger's list (`spec-ledger.md` 10). The
ledger knows accounts and numbers and no agent (`spec-ledger.md` 2.7), so the record
is plain data over account IDs:

```go
// Claim is one loan. internal/ledger: stdlib only, no root types.
type Claim struct {
	ID            int64 // assigned in order of formation, never reused in a run
	Creditor      int   // an Agent account, or the Estate account of a region
	Debtor        int   // an Agent account
	Principal     int64 // the amount lent, fixed at formation
	Interest      int64 // fixed at formation (7.1)
	PrincipalOwed int64 // what remains of the principal, 0 to Principal
	InterestOwed  int64 // what remains of the interest, 0 to Interest
	RateBP        int64 // the effective yearly rate at formation, for reports (7.1)
	Formed        int64 // the tick of formation
	Due           int64 // the tick the whole falls due, Formed + Term
	Overdue       int64 // the tick the claim first went unpaid, or -1 while it has not
	Origin        int   // 0 a loan (formed by Lend), 1 a payout claim (formed by Obligate); fixed at formation
}
```

3.1.2. A claim exists from its formation (6.4) until its **end**: when
`PrincipalOwed` and `InterestOwed` are both 0 it is removed from the list and an
event says so (16.1). **A claim with nothing owed does not exist** (FR-CRD-005); the
list is the open claims, in ascending ID, and nothing else. A **paid** part of a
claim is not kept: the cumulative totals of 3.3 are.

3.1.3. The **owed** of a claim is `PrincipalOwed + InterestOwed`. The **region** of
a claim is the region of its debtor's account; it is derived each time it is read
and a claim has no region field, so a relocation of the debtor moves the claim's
reports and nothing else (9.4).

3.1.4. A claim is an **asset** of its creditor and a **liability** of its debtor on
their balance sheets (`spec-ledger.md` 9.2) at the amount owed, with no
discounting. It is measurement only: no rule of this spec or any other reads a
balance sheet (ADR 0001 decision 7), and the rules below read the claim's own
fields.

3.1.5. A claim has an **origin**, `loan` or `payout`, fixed at formation: a loan is formed
by `Lend` (6.4), a payout claim by the operation `Obligate` of `spec-insurance.md` 9.2,
which appends a claim with no transfer. Every operation of 3.2 works on a claim of either
origin; what differs is read by reports and by two rules: `ForgiveClaims` ends loans only
(13.2), and the principal paid on a payout claim is the insurance result of
`spec-insurance.md` 11.1 and not this spec's (11.1). `MaxClaims` counts both origins
(6.4 step 1). The instruments and the warnings of this spec count origin `loan` only
(16.3); the payout claims are counted by INS.

### 3.2 The operations of the ledger

Five operations, each atomic (`spec-ledger.md` 2.6): it applies completely or
changes nothing and returns an error. They live in `internal/ledger`; the root maps
the sentinels (N-075).

| Operation | Does | Refused with |
|-----------|------|--------------|
| `Lend(creditor, debtor, principal, interest, rate, due)` | transfers `principal` from the creditor to the debtor with the reason `loan_out`, and appends a claim with the next ID, `PrincipalOwed` = `principal`, `InterestOwed` = `interest`, `Formed` the current tick, `Overdue` -1 | `ErrInsufficientFunds` (the creditor's available money is below `principal`), `ErrBalanceOverflow` (the debtor's balance would pass `MaxMoney`, or `principal + interest` does), `ErrReasonMismatch`, `ErrUnknownAccount`, `ErrClosedAccount`, `ErrInvalidAmount`, `ErrClaimShape` (the same account on both sides, a `due` not after `Formed`) |
| `Settle(claim, interest, principal)` | transfers `interest` with the reason `loan_interest` and then `principal` with `loan_repay` from the debtor to the creditor, lowers the owed parts, and removes the claim if both are 0 | `ErrInsufficientFunds`, `ErrBalanceOverflow` (the creditor's balance would pass `MaxMoney`), `ErrUnknownClaim`, `ErrInvalidAmount` (an amount above what is owed), `ErrReasonMismatch` |
| `WriteOff(claim)` | removes the claim with no transfer and returns the principal and the interest that were still owed | `ErrUnknownClaim` |
| `Reassign(claim, creditor)` | makes `creditor` (an Estate account) the creditor, with no transfer | `ErrUnknownClaim`, `ErrClosedAccount`, `ErrReasonMismatch` |
| `Obligate(creditor, debtor, amount, due)` | appends a claim of origin `payout` with the next ID, `Principal` = `PrincipalOwed` = `amount`, `Interest` 0, `Formed` the current tick, `Overdue` -1, with no transfer (called by `spec-insurance.md` 9.2 only) | `ErrInvalidAmount` (below 1), `ErrClaimShape` (the same account on both sides, a `due` not after `Formed`), `ErrUnknownAccount`, `ErrClosedAccount`, `ErrTooManyClaims` (`MaxClaims` claims are open) |

3.2.1. An account **cannot be closed** while it is the creditor or the debtor of an
open claim: `ErrOpenClaim`. `spec-ledger.md` 3.3 already says a closed account has no
money, no goods and no reservation; the claim is the fourth thing, and the removal
steps of 9.1 and 9.2 end or pass every claim before the account closes.

3.2.2. A transfer that a claim operation makes obeys every rule of
`spec-ledger.md` 4: an amount below 1 is refused, the debtor is never below 0, and a
creditor never above `MaxMoney`. The **headroom** case of the last (the creditor's
balance cannot take the payment) is handled by the caller as 8.4 says.

### 3.3 Reasons and totals

3.3.1. Three reasons join the registry of `spec-ledger.md` 7.2, all **internal**,
money only, declared at construction when the module is on and closed for the run
(as MON declares its flows' reasons), and none a faucet or a sink:

| Reason | Of | Kind | Between | Owner |
|--------|----|------|---------|-------|
| `loan_out` | money | internal | agent (the lender) to agent (the borrower) | `spec-credit.md` 6.4 |
| `loan_repay` | money | internal | agent to agent, or agent to estate (the principal paid) | `spec-credit.md` 8 |
| `loan_interest` | money | internal | agent to agent, or agent to estate (the interest paid) | `spec-credit.md` 8 |

They collide with no flow ID (`ErrReasonCollision`, `spec-money-supply.md` 4.5.2): a
flow with one of these three IDs is rejected. Internal reasons are told by their own
events, not by `BoundaryFlow` (`spec-observability.md` 4.4.1).

3.3.2. The ledger keeps seven cumulative totals beside the list, plain `int64`s, never
reset: `Lent` (the sum of `loan_out`), `Awarded` (the sum of `Principal` of every payout
claim formed, by `Obligate`), `Repaid` (the principal paid, of either origin),
`InterestPaid`, `InterestFixed` (the sum of `Interest` of every claim formed),
`LostPrincipal` and `LostInterest` (what was written off). They are saved with the list (17) and make the
identity of 3.6 exact and checkable.

### 3.4 What credit does to money, and the mechanisms compared

The brief asks whether a loan creates money. Four mechanisms were compared; the
reason for the choice is in the right-hand column.

| Mechanism | What it is | Decision and reason |
|-----------|-----------|---------------------|
| A. A loan moves existing money | the lender's balance falls by the principal and the borrower's rises by it; the claim is the lender's asset; repayment reverses it | **Chosen.** No new account kind, no faucet, no sink; `M` and the sectors of `spec-ledger.md` 9.3 are untouched by a loan, and only interest moves money between households (and, at an estate creditor, to the pool) |
| B. A lender that creates money | the principal is a grant from a boundary, the claim's repayment a sink | **Rejected for version 1.** The principal would be a boundary-backed faucet, repayment a sink, interest a faucet or a sink by turns, and the lender's capacity would be a rule about reserves that this library has no state for. Credit expansion is a reinforcing loop whose only bound is the author's care (MON 9.5). A setting that wants it declares a grant to the lender role (MON 4.2) with its own reason and a `Limit` |
| C. The estate lends | the pool is a lender of last resort or a treasury | **Not here.** The pool has no strategy (`spec-estates.md` 2.7.4); a lender of last resort is a policy actor's decision (PLC) and a treasury is FIS's. The estate holds claims it inherits (9.2) |
| D. Credit in goods | the lender advances goods; the borrower repays goods or money | **Not here.** Goods advanced on terms are a delivery contract (`spec-merchants.md` 9) or a trade; a claim in kind needs a valuation the engine refuses (`spec-goods.md` 7.4: money is not a good, and a good has no price but the market's) |

A is also what makes the "closed economy" of MON 8 stay closed with credit on: a
world with this module and no grant or levy has `M` equal to `M_init` at every tick
boundary (FR-CRD-006).

### 3.5 Collateral: not in version 1, and the shape

3.5.1. **Position.** A loan is unsecured. The lender's protection is the rule of 10.3
(offer only what is above its own reserve), the **exposure cap** of the borrower's
block (4.3: `MaxDebtTicks`), the eligibility test (a debtor in arrears is refused)
and the waterfall (9.1). A default costs the creditor what it was owed less what the
debtor's money pays.

3.5.2. **Why not.** A pledge needs one of three things the ledger does not have:
(a) goods in the **custody** of a third account (goods in an Escrow account would
need rules for ageing and wear in custody that `spec-transport.md` 7 gives a
Transit account and an Escrow account does not have; `spec-merchants.md` 9.10 says
the same of "a custody transfer of goods into escrow"); (b) an **earmark that
outlives a tick** (`spec-ledger.md` 6.5: none exists at a boundary, and
`spec-storage.md`'s pledge is of room, not of goods); (c) a **title** of
`spec-land.md`, which exists only with the tenure module on and whose transfer is a
call between ticks. None is free, and the reference chain's long-job role has
nothing to pledge but what its job will make.

3.5.3. **The seam.** A later version adds the fields `Security` (what is pledged)
and `Recovered` to the claim and a step to 9.1 that moves the pledge to the
creditor before the money is shared. The claim record, the waterfall and the
events are written so that this is a field and a step, not a redesign
(open question 3).

### 3.6 The identities

These hold after every operation and are checked by the audit (12.4 of the ledger,
FR-CRD-082):

1. **The pair balances.** The sum over open claims of `PrincipalOwed` is the
   `Lent` plus the `Awarded` less the `Repaid` less the `LostPrincipal`, with exact equality, and the
   sum of `InterestOwed` over open claims plus `InterestPaid` plus `LostInterest`
   equals `InterestFixed`, the sum of `Interest` of every claim formed (a total, so
   the check needs no history).
2. **Money is conserved.** The total of money over non-boundary accounts is
   unchanged by `Lend`, `Settle`, `WriteOff`, `Reassign` and `Obligate` (FR-LED-024: the three
   reasons are internal, so the change of the total in a tick is faucets less sinks
   of the other reasons).
3. **No claim outlives an account.** Every open claim's creditor and debtor are
   open accounts, the creditor an Agent or an Estate account and the debtor an Agent
   account, and they differ.
4. **No empty claim and no overpaid claim**: `PrincipalOwed` and `InterestOwed` are
   in `[0, Principal]` and `[0, Interest]`, and not both 0.

## 4. Roles, terms and the module

### 4.1 The module

`Config.Modules.Credit` is a pointer: nil is off, non-nil is on (`spec-config.md`
3.2). Plain data, no maps:

```go
// Credit is the credit module. Its presence switches it on.
type Credit struct {
	MinPrincipal  int64 // the smallest principal of a claim, in minor units
	ShieldTicks   int64 // ticks of a debtor's upkeep cost kept out of reach of repayment (8.2)
	WriteOffTicks int64 // ticks after Due at which an unpaid claim is written off; 0 means never
	MaxClaims     int64 // the most open claims at once
}
```

4.1.1. The module needs nothing but the core: with no role carrying a block it is
legal and does nothing (the warning `no_credit_roles`, 15.2). It adds no state to an
agent (N-128): everything it keeps is the ledger's list and the module's own
section (17).

4.1.2. **Defaults.** Every field is a plain integer whose zero is its off value or
is rejected: `MinPrincipal` and `MaxClaims` must be at least 1 (a required field of
a pack), `ShieldTicks` 0 means no shield and `WriteOffTicks` 0 means a claim is
never written off. There is therefore no default macro constant in this spec
(N-027).

### 4.2 The role block

A role is a plain struct (`spec-agents.md` 3.3). This spec adds one field, a block
that is nil for a role that does not deal in credit, as `Merchant` is
(`spec-merchants.md` 3.2):

```go
// CreditTerms is the credit block of a role: at most one of Lend and Borrow is set.
type CreditTerms struct {
	Lend   *LendTerms   // the households of the role post offers
	Borrow *BorrowTerms // the households of the role post requests
}

// LendTerms is what a lender posts and keeps.
type LendTerms struct {
	RateBP       int64 // the posted yearly rate, in basis points, simple interest; 0 lends at no interest
	MaxTermTicks int64 // the longest term it lends for
	KeepTicks    int64 // ticks of its own upkeep cost it keeps out of its offers
	ShareBP      int64 // the share of its lendable money it offers in one tick; 10 000 offers all
}

// BorrowTerms is what a borrower accepts and may owe.
type BorrowTerms struct {
	MaxRateBP    int64 // the highest yearly rate it accepts
	MaxDebtTicks int64 // the exposure cap: principal owed at most this many ticks of its upkeep cost
	SlackTicks   int64 // ticks added to the horizon of its gap, beyond the structural tick of 10.2: the time to sell what a job makes after it completes
}
```

In a file: `[modules.credit]`, `[role.<id>.credit.lend]` and
`[role.<id>.credit.borrow]`, the form of `[role.<id>.merchant]` (`spec-merchants.md`
19.1, N-102), so that layers merge by key (`spec-config.md` 6.4).

4.2.1. **Exactly one block.** A `CreditTerms` with both `Lend` and `Borrow`, or with
neither, is `ErrCreditShape` (15.1). The reason is position 4 of section 2: a role
that lends and borrows is a household deciding two things at once with one cash
reserve, and the loop of lending the money just borrowed is a dealer's, not a
household's. A setting that wants a middleman has two roles: money moves from the
lender to the middleman by one loan and from the middleman to the borrower by
another, each with its own rate and its own default.

### 4.3 What reads which field

The split is part of the contract, because a consumer's strategy may ignore a
block and the world may not.

| Field | Read by | Why there |
|-------|---------|-----------|
| `Credit.MinPrincipal`, `ShieldTicks`, `WriteOffTicks`, `MaxClaims` | the world (the round, settlement) | every claim obeys them whatever the strategy |
| `BorrowTerms.MaxDebtTicks` | the world, at the admission of a request (5.2 step 8) | a lender relies on the cap, so a strategy cannot lift it |
| the existence of a block | the world (5.2 step 2) | who may lend and borrow is a fact about the role |
| `LendTerms.RateBP`, `MaxTermTicks`, `KeepTicks`, `ShareBP` | the **default** strategy (10.3) | how much and at what price a lender offers is its decision |
| `BorrowTerms.MaxRateBP`, `SlackTicks` | the **default** strategy (10.2) | how much a borrower asks and what it accepts is its decision |

A consumer's own `Strategy` receives the block in its view (10.1) and may use it, or
post what it likes inside the admission of 5.2.

### 4.4 Who may hold a block

4.4.1. **Households of any weight.** A cohort of weight `w` is one agent: its request
is for the whole agent and its exposure cap is `MaxDebtTicks` of the agent's upkeep
cost, which has the weight in it (`spec-agents.md` 5.2). Nothing in this spec
splits a claim among the households of a cohort, and 9.3 says what a change of
weight does.

4.4.2. **An external agent** of a role with a block borrows and lends like any other,
through the plan of a consumer's strategy or through its directive (10.5). The
estate pseudo-agent has no role, no strategy and no block and is never a lender or
a borrower; it holds claims only by inheritance (9.2).

4.4.3. A role with a block may also have recipes. A household that produces and
lends is legal: its offer is made from what is left after its own bids (10.3), and
its claims are an asset the welfare ratio sees only as interest received (11.1).

## 5. Requests and offers

### 5.1 The two plan fields

`TradePlan` (`spec-agents.md` 6.3) gains two fields, in the way of `Offers` and
`Takes` (`spec-merchants.md` 9.2). Both last one tick, as an order does.

```go
// CreditRequest asks for a loan. The principal is a most: the round may fill it
// in parts, from several lenders.
type CreditRequest struct {
	Principal int64 // money in minor units, at least MinPrincipal
	Term      int64 // ticks until the whole falls due, 1 to MaxCreditTerm
	MaxRateBP int64 // the highest yearly rate it accepts, 0 to MaxCreditRateBP
}

// CreditOffer offers money on a term at a posted rate. The principal is a most.
type CreditOffer struct {
	Principal int64 // money in minor units, at least MinPrincipal
	MaxTerm   int64 // the longest term it lends for, 1 to MaxCreditTerm
	RateBP    int64 // the posted yearly rate, 0 to MaxCreditRateBP
}

// TradePlan gains:
//	Borrow []CreditRequest // at most one; the default strategy appends one for a borrower role (10.2)
//	Lend   []CreditOffer   // at most one; the default strategy appends one for a lender role (10.3)
```

5.1.1. The fields are slices, as the others of the plan are, so that a plan has one
shape; at most one of each is admitted (5.2 step 6). A plan that does not use them
is unchanged, and with the module off a request or an offer is refused (5.2 step 1)
and costs nothing else.

5.1.2. A request or an offer names **no counterparty**. The round chooses (6.3). A
consumer that wants a loan between two named agents is not served by this: no such
call is offered in version 1 (13.3).

### 5.2 Admission

At the start of the round (6.1) the world visits the requests and the offers in
agent-ID order, a request before an offer of the same agent, and admits each or
refuses it **whole** with a `reason` (exported constants of the root package,
`event.go`; a refusal changes nothing, is counted in `CreditRound` (except `credit_off`, which is a `Debug` line only, 16.1.1) and logged at
`Debug`, and is not an error returned by `Step`). The first failing check in this
order is the reason:

| Of | # | `reason` | Condition |
|----|---|----------|-----------|
| both | 1 | `credit_off` | `Modules.Credit` is absent |
| request | 2 | `not_borrower` | the agent's role has no `Borrow` block |
| offer | 2 | `not_lender` | the agent's role has no `Lend` block |
| both | 3 | `invalid_principal` | `Principal` below `MinPrincipal` or above `MaxMoney` |
| both | 4 | `invalid_term` | `Term` (or `MaxTerm`) below 1 or above `MaxCreditTerm` |
| both | 5 | `invalid_rate` | the rate below 0 or above `MaxCreditRateBP` |
| both | 6 | `duplicate` | the agent already has an admitted request (an offer) this tick |
| request | 7 | `in_arrears` | the agent has an overdue claim as debtor (`Overdue` at or above 0) |
| request | 8 | `exposure` | `Debt + Principal` exceeds `MaxDebtTicks * U` (5.3) |
| offer | 7 | `insufficient_funds` | `Principal` exceeds the agent's available money (balance less what its orders of the tick reserved) |

5.2.1. The agent's available money is its balance less what its orders of the tick
reserved (`spec-clearing.md` 3.3), so an offer can never spend money that an order
has earmarked, and a strategy that double counts is refused, as `insufficient_stock`
refuses an ask (`spec-merchants.md` 9.3).

5.2.2. **The world does not clamp.** A request above its exposure room is refused
whole with `exposure`, not cut to the room, as an order beyond its budget is
(`spec-agents.md` 6.5.2); the world clamps no quantity to a room. The one value it replaces
is a rate above a legal cap (5.2.3). The default strategy sizes its request inside the room
(10.2), so the check is the backstop for any other strategy and a damper that is
counted (`CreditRound.Exposure`, FR-STB-013).

5.2.3. **The rate limit is not an admission check.** After the last check of the table, a
request's `MaxRateBP` above the cap in force for the borrower's region is **replaced** by the
cap, never refused for it, and the replacement is counted in `PriceControlled`
(`spec-institutions.md` 6.6.1; FR-CRD-028). The cap bounds what a borrower accepts, not what
a lender posts (ORG 6.6.2).

### 5.3 The quantities the checks read

5.3.1. **`U`** is the agent's **upkeep cost**, the quantity relief reads at P1.6: the
effective upkeep of the agent, weight included (`spec-agents.md` 5.2), at the prices
of its region (`spec-estates.md` 2.6, `spec-agents.md` 5.6.2). It is also the unit
of the shield (8.2) and of the lender's reserve (10.3). The default strategy has its
own `UpkeepCost` at its beliefs (`spec-agents.md` 7.4.2); the two are not the same
number and are not asked to be.

5.3.2. **`Debt`** is the sum of `PrincipalOwed` over the claims of which the agent is
debtor. The interest owed is not in it: the cap bounds the money borrowed, and the
interest is the price of it.

5.3.3. **`CreditRoom`** is `max(0, MaxDebtTicks * U - Debt)`, the principal the agent may
still ask for (the product is checked and saturates at `MaxMoney`). It is in the
view (10.1). With `U` 0 (a world with no upkeep) the room is 0 and no request is admitted (the warning `credit_no_upkeep`, 15.2).

## 6. The round

### 6.1 When

6.1.1. Once per tick, in phase 3, after the contract round and `MerchantsPlanned`
(P3.3 and P3.3a of `spec-world.md` 6.5), after every agent's plan has been executed
and before the estates' asks and the counterparties' orders: the new position
**P3.3b** (21, an edit of WLD 6.5). It runs only with `Modules.Credit` present and is
a no-op when no request and no offer was submitted.

6.1.2. **Why there.** (a) After the last plan: requests and offers are plan fields,
so they exist only when every agent's `Trade` has run. (b) After the orders of
P3.2: the lender's offer is made from what its own bids left (10.3), and the round
moves only available money, so it can never take a reservation (the transfer
of MER's escrow at P3.3 is the same case, `spec-world.md` 6.1.6). (c) Before P4.1: a loan changes
no market's orders of this tick, because the borrowed money is spendable no earlier
than P3.2 of the next tick (`spec-world.md` 6.1.8). That one-tick delay is the
phase order's (WLD-A1) and is not counted again here.

6.1.3. **After settlement.** P1.7a (8) of the same tick has already settled every
claim due, so a request of tick `t` is admitted against the debt that remains
after tick `t`'s payments. The order is the rule that makes a loan never renewed
(position 10): the household's cash at P1.7a pays the old claim, and the new loan
arrives at P3.3b and is spendable at P3.2 of `t + 1`.

### 6.2 Lanes

A **lane** of the round is a region: the region of the account of the agent at the
round. A request is matched only with offers of its own lane. The round visits the
lanes in ascending `RegionID`; an outside node has no agents and no lane
(`spec-regions.md` 3.5).

6.2.1. **Why local.** A lender judges a borrower it can see; the engine gives an
agent only its own market's report for nothing and a delayed, blurred rumour of
another's for an informed role (`spec-information.md` 3), and a borrower's standing
is not a price. A loan across regions would need a lender to know a borrower there,
which nothing in the library supplies, and would hide the region of a claim's
events behind two. It is open question 4.

### 6.3 The order and the matching

6.3.1. In a lane the admitted offers are ordered by **effective rate** ascending,
equal rates by the stateless key `K(t, agent)` ascending, then agent ID; the
admitted requests by `MaxRateBP` **descending**, equal limits by `K(t, agent)`
ascending, then agent ID. `K(t, id)` is `spec-world.md` 10.1.1's: SplitMix64 of
`t * 0x9E3779B97F4A7C15 + id` in modular `uint64` arithmetic, smaller first. It takes
**no draw** (position 16), needs no state and changes every tick, so a lower ID is
never systematically first (WLD 10.1.1, and `spec-employment.md` 5.3.1 for the same
choice). The **effective rate** of an offer is `halfEven(RateBP * ScaleBP / 10 000)`
with `ScaleBP` the lane's rate scale (13.1, 10 000 unless a consumer set one).

6.3.2. For each request in order, with `p` its remaining principal, the world goes
through the offers in order:

1. an offer is skipped if it is the request's own agent's (a guard for a role
   change in the same tick: a role has one block), if its `MaxTerm` is below the
   request's `Term`, or if its remaining principal `q` is 0;
2. since the offers are in ascending rate, the first offer whose effective rate exceeds
   the request's `MaxRateBP` ends the scan for this request (**no lender at the
   price**);
3. otherwise `x = min(p, q)`. A loan below `MinPrincipal` is not made: if `x` is
   below it and `x = p` the request ends (its remainder is too small), and if `x` is
   below it and `x = q < p` the offer is skipped (its remainder is too small);
4. the pair is a **tentative fill** of `x`, which goes through formation (6.4). If
   formation succeeds, `p` and `q` fall by `x`; if it fails with a reason the fill is
   dropped and counted, and the scan goes on.

6.3.3. **Discriminatory, not uniform.** Each fill is at the lender's own posted
effective rate, not at a common rate and not at a midpoint: a borrower served by two
lenders pays each its price, and the lender with the lower rate is filled first. It
is the simple form with no cycle: a request ends when its limit is below the next
rate or its principal is met, an offer when its principal is spent, and nothing
revisits. It gives no lender a reason to post other than its true rate, because
being cheaper fills it first and a higher rate than the limit never fills.

6.3.4. **What the order means when cash binds.** When the offers' principal falls
short of the requests', the requests with the highest limits are served and the
others are left unfilled (`Unfilled`, 16.1). It is rationing by willingness to pay,
the order of a market, and not by ID or by region (WLD 10.2: this position adds no
place to its list).

### 6.4 Formation

For a tentative fill of `x` from the lender `L` to the borrower `B`, in region `r`,
at effective rate `r_e` and term `n`, the world checks and acts in this order. The
first failure ends the attempt with its reason and changes nothing:

1. **`too_many_claims`**: the open claims, of either origin (3.1.5), are `MaxClaims`. The round ends (no later fill
   can form either, since the count only rises in the round).
2. **`overflow`**: `x + interest(x, r_e, n)` would exceed `MaxMoney`, or `B`'s balance
   plus `x` would (`ErrBalanceOverflow`).
3. **The loan**: `Lend(L, B, x, interest, r_e, t + n)` (3.2). It cannot fail for want of
   funds, because the offer was admitted against `L`'s available money (5.2 step 7)
   and the only transfers of the round are the loans of `L`'s single offer, whose
   remaining principal the matching tracks.
4. **The record and the event**: the claim is in the list with the next ID, and
   `LoanFormed` is emitted (16.1) in formation order.

6.4.1. **The deadline is the due tick** and is known at formation: `t + n`. There is
no late payment before it and no early payment, by the world or by a strategy
(open question 5).

6.4.2. **The money is spendable from the next tick.** `B`'s balance holds `x` from
the moment of the loan; its orders of this tick were placed at P3.2 and the round
is after them (6.1.2), so the first use is its next `Trade`.

6.5. **The cost** of the round is `O(requests + offers)` per lane for the sort and
`O(fills)` for the matching: counts, bounded by the agents with a block, and no
term that grows with households (NFR-CRD-002). The round allocates nothing beyond
scratch the world holds.

## 7. Interest

### 7.1 The rule

7.1.1. The interest of a claim of principal `x`, at effective yearly rate `r_e` (in
basis points) for a term of `n` ticks, is fixed at formation:

```
interest = halfEven( x * r_e * n / (10 000 * TicksPerYear) )
```

`TicksPerYear` is the derived constant of `spec-calendar.md` 3.2. The product is
computed with the checked multiply-divide helper and a 128-bit intermediate
(`spec-currency.md` 7, architecture.md 3.5): the largest numerator is
`MaxMoney * MaxCreditRateBP * MaxCreditTerm`, about `10^18 * 10^5 * 3.65 * 10^3`,
which fits 128 bits and not `int64`; the result is at most `x * MaxCreditRateBP *
MaxCreditTerm / (10^4 * TicksPerYear)` and is checked against `MaxMoney` with the
principal (6.4 step 2).

7.1.2. **Rounding.** The quotient `q` and remainder `m` of the numerator by the
denominator `D = 10 000 * TicksPerYear`: round up when `2m > D`, down when
`2m < D`, and to the even `q` when `2m = D`. It is the rule of
`spec-clearing.md` 5.2 (the midpoint of two limits), stated here because a rounded
function states its rule (architecture.md 3.5). A change of this rule is a
behaviour change (architecture.md 6.6) and moves the golden files.

7.1.3. The claim records `RateBP` = `r_e` and `Interest`; the **term** is
`Due - Formed`. All three are fixed: no rule of this spec reprices, extends or
rolls a claim.

### 7.2 Why simple, fixed and paid at the end

- **No accrual state.** Nothing has to be saved or advanced per tick: the two
  integers `PrincipalOwed` and `InterestOwed` are the whole of a claim's amount, and
  a restore has nothing to recompute.
- **Both parties know the amount at formation.** The lender sees what it is paid, the
  borrower what it owes; neither reads a balance, a price or a count of anything
  after the formation (MON position 5).
- **No compounding.** A rate on a balance that is itself growing is the loop of
  `spec-money-supply.md` 9.5. Interest here is a fixed function of the loan, so the
  total a debtor owes never grows after formation: no penalty rate on arrears
  (8.7), no interest on interest. A longer horizon is a sequence of loans, each
  priced when it is made and each subject to the eligibility test.
- **Interest is a transfer.** It moves money from the debtor to the creditor and
  creates none (position 7); the household sector's total is unchanged by it (or,
  with an estate creditor, falls by it and the pool's rises).

### 7.3 Size

For a term of `n` ticks and a rate of `r` basis points a year, the interest is a
share `r * n / (10 000 * TicksPerYear)` of the principal: 600 basis points a year
over 65 ticks of a 365-tick year is 1.07% (a placeholder pair; example A of 20).
A loan whose interest rounds to 0 is legal and interest-free, and the smallest loan
of the shortest term at a rate above 0 that does so is warned of at construction
(`interest_rounds_to_zero`, 15.2).

## 8. Settlement

### 8.1 When, and for whom

8.1.1. In phase 1 of every tick, with the module on, after the money flows and before
the outside's walk: the new position **P1.7a** (21, an edit of WLD 6.3). It runs
after relief (P1.6) and the flows (P1.7) so that a debtor's balance is what it holds
at the start of the tick, with the tick's relief and grants in it and the tick's levy
out of it, and before the tick's orders (P3.2), so that a payment comes out of money
the household would otherwise spend.

The pass visits the claims of **either origin** (3.1.5): a payout claim of `spec-insurance.md`
is settled by the same pot and the same pro rata share, and the world's event for its
payments is INS's `CoverSettled` of stage `claim` after this module's `CreditSettled`
(`spec-insurance.md` 15.1). FIS's position **P1.7b** follows this one at once
(`spec-world.md` 6.3): a debtor's balance there is what remains after the payment, and a
tax a payer cannot meet is forgiven for the shortfall (`spec-fiscal.md` 5.3), so a claim
is paid before a tax is due from the same money.
8.1.1.1. **The pass reads a snapshot.** At the entry of P1.7a the world records the
balance of each debtor to be visited, and every pot of 8.2 is computed from that
record, so no payment of the pass changes another debtor's pot: an agent that is
both a creditor and a debtor (9.4.1, FR-CRD-043) has the same pot whatever the order
in which it and its debtors are visited. The payment of a debtor never exceeds its
recorded balance, and its live balance only rises during the pass (it can receive,
and it pays once), so no transfer of the pass is refused for funds.

8.1.2. The pass visits **debtors** in ascending agent ID. A debtor is visited when at
least one of its claims is **due** this tick (`Due = t`) or **overdue** (`Overdue`
at or above 0). The claims due at `t` and the overdue claims are found from the
derived indexes of 17.3 in time proportional to their number, not to the open claims
(NFR-CRD-003). An away or travelling debtor, and one on a journey, is visited like
any other: a payment is a transfer between accounts and needs no agent to be
present (9.4).

### 8.2 The pot and the shield

For a debtor with the balance `b` recorded at the entry of the pass (8.1.1.1; no
reservation exists in phase 1, `spec-ledger.md` 6.5), upkeep cost `U` (5.3.1) and `ShieldTicks` `s`:

```
shield = s * U                 (checked; saturates at MaxMoney)
pot    = max(0, b - shield)
owed   = the sum of the owed of its due and overdue claims
pay    = min(pot, owed)
```

The **shield** is money the pass will not take: `ShieldTicks` ticks of the debtor's
own upkeep cost. It is a floor under what is left, not a share of what is paid, so a
debtor with `b` at or below the shield pays nothing this tick, and every unit above
the shield goes to its creditors until the owed is paid. It is the guard that MON
gives a levy (`Exempt`, `spec-money-supply.md` 9.4) and for the same reason: a debtor
that pays below its upkeep goes unfed, its privation rises and the failure ladder
starts (`spec-spawner.md` 5), which makes the creditor's loss certain instead of
late.

8.2.1. The shield is **not** a promise that the debtor eats. It leaves the debtor
`s * U` of money at the start of the tick, which tier 1 of its budget spends first
(`spec-agents.md` 7.4.3); whether upkeep goods are on the market at a price it can
pay is the market's.

8.2.2. **The shield trades food now for credit later, and its best value moves.** A
debtor that pays less leaves more owed and overdue, and an overdue claim bars the next
request (5.2 step 7). In the model of 19.2 (one bad cycle) the unfed borrower-ticks at
`ShieldTicks` 0, 1, 2, 3, 5 and 8 are 727, 723, 242, 238, 231 and 272 at
`SlackTicks` 0, and 165, 162, 212, 208, 200 and 235 at `SlackTicks` 5: the shield
protects the tick of the due and not the ticks after it, and the value with the
fewest unfed ticks depends on the horizon. It is a pack's choice and the reason it is
a parameter and not a rule constant (CRD-A8; a damper strength for CLB to find, not
a value this spec sets).

### 8.3 The allocation

When `pay` is less than `owed` it is shared among the debtor's due and overdue
claims **pro rata by what each owes**, with the largest remainder:

```
share_i = floor(pay * owed_i / owed)                 (128-bit intermediate)
left    = pay - sum(share_i)
```

then the `left` units go one each to the claims in descending order of the
fractional part `(pay * owed_i) mod owed`, equal parts by `K(t, claim ID)` ascending
and then claim ID. `left` is less than the number of claims with a positive
fractional part, and a share never exceeds `owed_i` (the arithmetic is in the model,
19.1). Within a claim a share pays **interest first** (`loan_interest`), then
principal (`loan_repay`): `ip = min(share, InterestOwed)`, `pp = share - ip`.

8.3.1. **Why pro rata.** A debtor's creditors are not ranked: no creditor has a
claim senior to another's, and an order by claim ID or by creditor would be a place
where an ID gives a systematic advantage (WLD 10.2, N-122). The rule uses only the
debtor's own claims and its balance recorded at the entry of the pass (8.1.1.1): each
payment depends on the claim's own owed and the totals, never on an order (the
property relief has, FR-EST-019).

8.3.2. **Why interest first.** It is the order of a loan: the creditor's price is
paid before its capital comes back. It matters only for a partial payment, where it
makes the creditor's income earlier than its principal; a default then costs the
creditor principal first, which is what a loss of capital is.

### 8.4 The creditor's headroom

A payment that would take the creditor's balance above `MaxMoney` is cut to its
headroom, the part not paid stays owed, and the claim is treated as unpaid in the
amount of the cut (`CreditSettled.Headroom` counts it, 16.1). The case is the bound
of `spec-ledger.md` 4.1 meeting a rich lender, which a pack at a sane `MaxMoney`
never reaches; it is stated so that the pass is total (no refused transfer is left
to panic) and not because it is expected.

### 8.5 After the pass

Per claim, in ascending claim ID, with the owed left after the payment:

1. **Ended: repaid.** Both owed parts are 0: the claim leaves the list, cause
   `repaid`, and `LoanEnded` is emitted.
2. **Defaulted.** Owed is above 0 and `Due = t` and `Overdue` was -1: `Overdue` is
   set to `t`. It is not an event of its own: it is counted in `CreditSettled`
   (`Defaulted`) and the claim is in `Arrears`.
3. **Written off.** `WriteOffTicks` is above 0 and `Overdue` is at or above 0 and
   `t >= Due + WriteOffTicks`: `WriteOff` removes the claim, cause `written_off`, and
   `LoanEnded` is emitted with the principal and interest lost.

A claim overdue stays in the list and is paid by the same pass on every later tick,
as money above the shield allows.

### 8.6 What the debtor and the creditor see

The debtor's money falls by the payment and its welfare record of the tick gains
the interest as a cost (`I^out_t`, 11.1). The creditor's money rises by it and the
record gains the interest as income (`I^in_t`, in the signed `G_t`). The principal is neither (11.1). The
creditor's view of its claims at the next `Trade` is the sums of 10.1; nothing in
the pass is a decision of either.

### 8.7 What is not done

No penalty rate and no interest on arrears (7.2: a rate on a growing balance is the
debt spiral of CRD-A1, and arrears must not be dearer than the loan). No
re-scheduling of a due tick, no partial forgiveness, no collection cost, no seizure
of goods (3.5), no bar on the debtor's orders: a debtor in arrears buys, sells and
works as any household, and the one thing it cannot do is borrow again (5.2 step 7).
Early payment is not offered (6.4.1, open question 5).

### 8.8 Relief money and a debt

Relief (P1.6) and a grant (P1.7) arrive before this pass and are in the balance it
reads, so money the pool pays a poor household for its upkeep can be paid on to a
creditor, up to the shield's floor. It is deliberate and is a hazard, not a bug: the
relief rule reads the household's ratio and not its debt (`spec-estates.md` 2.6), the
shield is what keeps the household fed, and a household in a long default is paying
back the pool's money to a lender. It is CRD-A10 in 18 and a scenario (CRD-S2).

## 9. Removal, weight, role and relocation

### 9.1 The removal of a debtor: the waterfall

9.1.1. `spec-agents.md` 3.6 gains one step, **2d**, after the contracts of a removed
deliverer (step 2a), after the steps that EMP (`spec-employment.md` 6.5.2: two steps,
not numbered there) and LND (`spec-land.md` 6.4: a step called 2b) ask for, and before
the agent's money moves to the estate (step 3). The place is not a matter of result:
none of those steps pays the removed agent's own balance (a removed worker's escrow
returns to its employer, a removed employer's goes to the estate, a title moves to
the pool), so 2d needs no ordering against them. It is placed after them so that a
step that did raise the balance is already in what 2d reads; the supervisor numbers
the steps of the three specs. For
the removed agent `d`, with `b` its whole balance (no reservation exists in phase 7,
and none at a call between ticks):

1. take its open claims as debtor, in ascending claim ID, with `owed_i` and the sum
   `owed`; with none, the step does nothing;
2. `pay = min(b, owed)`, **with no shield**: the agent is gone;
3. allocate `pay` among the claims as in 8.3 (pro rata by `owed_i`, the largest
   remainder, interest first within a claim) and apply each share with `Settle`,
   cut to the creditor's headroom as in 8.4;
4. each claim then **ends** with the cause `debtor_removed`: what is still owed is
   written off (`WriteOff`) and is the claim's loss (`LostPrincipal` and `LostInterest` of
   `LoanEnded`); a claim paid in full ends the same way with both 0; `LoanEnded` is emitted per claim in ascending claim
   ID.

Step 3 of AGT 3.6 then moves what is left of the balance, `b - pay`, to the estate
under `estate_money`. The estate therefore receives only what the creditors did not
take.

9.1.2. **Creditors come first, the estate second.** This is the one change this
spec makes to what the estate receives (`spec-estates.md` 2.1), and it is a
ranking: a dead agent's money pays its debts before it funds the pool, and the
goods it leaves are not touched (open question 8: they are the pool's, listed at
the reference price, and a creditor has no claim on their proceeds in version 1).

9.1.3. **Every cause of a removal.** The waterfall runs at every removal
(`insolvent`, `starved`, `consumer`, `population`, `spec-agents.md` 14.1) and at a
consumer's `RemoveAgent` (`spec-integration.md` 4.4). A **reduction** of weight is
not a removal and has no waterfall (9.3).

### 9.2 The removal of a creditor: the claims pass to the estate

9.2.1. Step **2e**, after 2d: for each open claim of which the removed agent is the
creditor, in ascending claim ID, `Reassign` makes the **Estate account** of the
region of the agent's account (`spec-estates.md` 2.2: the region it was in when it
died; a traveller away leaves its region of departure) the creditor, and
`LoanPassed` is emitted. Nothing moves: the estate has not paid for it and the
debtor owes what it owed.

9.2.2. The estate is then an ordinary creditor of the claim. The debtor's payments
at P1.7a go to the estate account under the same reasons, and are the pool's money
(`M_p`, `spec-money-supply.md` 3.1.1); a debtor's removal pays it in the waterfall.
The estate does not lend, ask, forgive or sell a claim: it has no strategy
(`spec-estates.md` 2.7.4). A claim it holds that is overdue is as any other (8.5).

9.2.3. **Both sides.** An agent that is debtor on some claims and creditor on
others (a role change moved it from a lender role to a borrower role, 9.4) goes
through 2d, then 2e: its debts are paid from its balance first, and its claims then
pass to the estate. The order cannot change a result: step 2d reads the balance and
2e moves no money.

### 9.3 A change of weight

A claim is **the agent's**, not a household's. A reduction of the weight of a debtor
or a creditor, by the failure ladder (`starvation`) or the population module
(`deaths`), moves money to the estate (`spec-scale.md` 4.8) and changes **no claim**:
the survivors owe what the agent owed and the creditor's claim is as large. A rise of
weight (entry, a birth into an existing agent) changes none either. What follows the
weight is the exposure room (5.3.3), which has `U` in it, so after a fall the room
shrinks and may be 0, and no debt is called in by it.

9.3.1. **The cost of the choice.** A cohort that loses half its households keeps
all its debt: per household it doubles, and the money of the households that left
went to the pool, not to the creditor. The alternative, a claim split in proportion
to the weights, needs a second claim for the share of the households that left, with
an ID and a creditor's share of the loss (a write-off or a payment from the money
that went to the estate), and was not taken: the first consumer's world has no
weight above 1 (N-030), and a split touches the ledger, the waterfall and the
checkpoint for a case no pack has. It is CRD-A7 (a hazard) and open question 9.

### 9.4 A change of role, a relocation, a trip and a journey

9.4.1. **A role change** (`spec-spawner.md` 4.2) leaves every claim as it is. The
agent's block is that of its new role: a household that loses its `Borrow` block
cannot post a request, and its claims settle as before; one that gains a `Lend`
block offers from the next tick. A claim has no role field, so `ForgiveClaims` by
role (13.2) reads the debtor's role at the call.

9.4.2. **A relocation** (`spec-regions.md` 6), the arrival of a trip or of a journey
(`spec-transport.md` 5.7, `spec-migration.md` 8.4) changes no claim and is not
refused by one. This module registers **no `region_bound` refusal**
(`spec-regions.md` 6.5.2). The rule of N-090 is for a module that keeps an
agent's account in a region while the agent is away or committed, so that another
caller cannot break its invariant; a claim has no such invariant: repayment is a
transfer between two accounts (3.2.2) and the lane is read at the round, from the
account's region at that moment. A debtor that moves repays from where it is, and
an event of its claim names the region it is in at the event (an event of a
transition of one agent carries the agent's region at the time,
`observability-and-logging.md` 4.1).

9.4.3. **A debtor that is away** (a trip) or **on a journey** is given no view
(`spec-world.md` 6.1.3) and so posts no request, and is settled by P1.7a like any
other. It is therefore never refused for being away.

### 9.5 What a restore does

Nothing special: the list, the totals and the module's section are saved (17), and
the derived indexes of 17.3 are rebuilt. A world restored between ticks settles the
same claims at the same ticks and forms the same loans (FR-CRD-087).

## 10. What a strategy sees, and the default rules

### 10.1 The `Credit` group of the view

`AgentView` (`spec-agents.md` 6.4.1) gains one group, present **only in the view of an
agent whose role has a credit block** (as the merchant group is,
`spec-merchants.md` 4.1). Plain values, valid for the call only:

| Field | Meaning |
|-------|---------|
| `Lend`, `Borrow` | a copy of the role's block (one is nil) |
| `MinPrincipal` | the module's, so a strategy need not know the configuration |
| `UpkeepCost` | the `U` of 5.3.1 for this agent, weight included |
| `Debt` | the principal owed as debtor (5.3.2) |
| `Owed` | principal and interest owed as debtor |
| `Overdue` | the owed on the claims that are overdue as debtor; 0 when none |
| `NextDue` | the earliest `Due` among its claims as debtor, -1 when it has none |
| `CreditRoom` | the exposure room (5.3.3); 0 for a lender |
| `Lent` | the principal owed to it as creditor |
| `Receivable` | principal and interest owed to it as creditor |
| `ScaleBP` | the rate scale of its region (13.1) |

The group is built from the ledger's list at the start of phase 3 and does not
change within the tick (the round is after every plan). It has **no other agent's
claim and no figure of the market for credit**: no rate other than the agent's own
scale, no count of lenders, no total of the lane. A strategy reads what it is owed
and what it owes (N-106's reasoning for remote prices: no aggregate is shown to an
agent that could close a loop through it).

### 10.2 The default borrower

A household of a role with a `Borrow` block, running `DefaultStrategy`
(`spec-agents.md` 7), appends **one request** when it has a **cash gap**: money it
will need before its next income and does not hold. The rule is a pure function of
the view and is evaluated in `Trade`, after the tiers of 7.4.3 are built:

```
J    = Remaining                    if a job is in progress (spec-agents.md 6.4.3)
     = Duration of the planned recipe  otherwise (spec-agents.md 7.2; the job starts next tick,
                                       StartsIn is 1); no request when there is none
if J = 0: no request
Late = 1 if no job is in progress and Money - UpkeepCost < Tier2, else 0
                                    (the inputs cannot be bought at once: 10.2.1.2)
H    = J + 1 + Late + SlackTicks    the ticks of the horizon, and the term
need = UpkeepCost * H + Tier2 - Money
ask  = min(need, CreditRoom)
if ask < MinPrincipal: no request
request = { Principal: ask, Term: H, MaxRateBP: Borrow.MaxRateBP }
```

`Tier2` is the **cost of the inputs and tools of the next job at the limits of the
plan, whatever money allows**: the sum over its commodities of the units
`min(Gap_c - T1_c, max(0, Jobwant_c - left_c))` of `spec-agents.md` 7.3.4 (which are
units, and do not depend on money) at each intent's limit price. It is not what tier 2
of 7.4.3 can afford after tier 1: that is limited by the money, and a request sized by
it would under-ask exactly when the household holds less than its inputs.

10.2.1. **Reading it.** The household holds `Money`; it buys upkeep at P3.2 of this
tick and of each of the `H - 1` that follow and, before a job starts, pays for its
inputs (`Tier2`). The gap is the difference, and the request is for the gap, capped
by the room and by the minimum. A household that holds enough asks for nothing: the
rule is **need-based, not price-based**; it does not compare the interest with
anything, because the alternative to the gap is going unfed.

10.2.1.1. **Why `+ 1`.** The `1` is a rule constant of the phase order and not a
tunable (architecture.md 9.6). A job of `J` ticks to its completion completes in
the production phase of tick `t + J` (`spec-production.md` 5.2); its output is sold
at P4.1 of that tick at the earliest, and money earned at P4.1 can be spent no
earlier than P3.2 of the next tick (`spec-world.md` 6.1.8), so it is in the balance
at P1.7a of `t + J + 1`. The claim must therefore fall due at `t + J + 1 +
SlackTicks`, and the household buys upkeep at P3.2 of the `J + 1 + SlackTicks` ticks
`t` to `t + J + SlackTicks` (`Late` 0): `H` ticks, as the term and as the horizon of the gap.
With `SlackTicks` 0 the default is exact for an output sold in its completion tick,
for a household that buys its inputs at once (`Late` 0, 10.2.1.2);
`SlackTicks` is the extra ticks it takes to sell (an output that spoils inside them
is open question 12). A horizon without the `1` (`H = J + SlackTicks`) has the claim fall due one tick
before the cash arrives at `SlackTicks` 0: it goes overdue every cycle and bars the
next request; the model of 19.2 (case B0) shows 47 unfed borrower-ticks and 40
defaults against none.

10.2.1.2. **Why `Late`.** The loan of this tick is formed at P3.3b, after the purchases
of P3.2, and is spendable from the next tick (6.4.2). A household whose money after
tier 1 (`UpkeepCost` is the bound the rule reads) is below `Tier2` therefore buys its
inputs at P3.2 of `t + 1`; they arrive at P4.1 and the job starts at P2.7 of `t + 2`
(`spec-agents.md` 6.4.3), a tick later than for a household that buys at once. The job
completes, is sold and its cash is visible one tick later, so the horizon and the term
gain the tick: `Late` is 1. The rule is a test of the view only (`Money`, `UpkeepCost`,
`Tier2`); it reads no price. At `t + 1` the household is a not-started job again with
`Money` including the loan, so `Late` is 0 and the request is the difference only. It
is a bound, not a promise: if tier 1 costs more than `UpkeepCost` the inputs are
bought later still, and `SlackTicks` is the pack's margin for that. A pack whose
households start with money below their inputs should not set `SlackTicks` to less
than its own estimate of that delay; the model's cases H0 and H1 (19.2) show 19
defaults at `Late` 0 for the borrowers that start below their inputs, and none with
the rule.

10.2.2. **It does not look at its eligibility.** A household in arrears asks and is
refused `in_arrears`, and a household at its cap asks for what fits (`CreditRoom` is in
the rule). The refusal is counted (`CreditRound`, 16.1), so the damper of 5.2
step 7 is not silent (FR-STB-013).

10.2.3. **Seasonal credit, the answer to N-032.** For a role with a job of `d` ticks
and an income at its end, the request at the start of each job covers its upkeep and
inputs for the length of the job, and the claim falls due `d + 1 + Late + SlackTicks`
ticks later, one tick after the output is sold (10.2.1.1, 10.2.1.2; example A, 20.1). It is the same
rule for any job that pays at its end, because the engine does not know what the job
is.

10.2.4. **It asks again.** The rule is evaluated every tick, so a gap that a request
left (a loan cut by a lender's remainder, or a request below `MinPrincipal`) is
asked for the next tick if it is at least `MinPrincipal`, and a household with a
claim open asks for the difference only, since the room and the `Money` already
reflect the loan. A gap below `MinPrincipal` is never asked for: that is why a pack's
`MinPrincipal` must be below the smallest gap it cares about (the model's case G4).

10.2.5. **A project of the saving module has no gap.** A household funded for a project of
`spec-saving-and-investment.md` 6.1 has `Tier2` in its hand, so this rule asks for nothing for
it; SAV's default asks for no loan for a project (6.2) and a consumer's strategy may append
a `CreditRequest` of its own (6.2.2 there).

### 10.3 The default lender

A household of a role with a `Lend` block appends **one offer** from what its own bids
left:

```
lendable = max(0, Money - Bids - KeepTicks * UpkeepCost)
offer    = floor(ShareBP * lendable / 10 000)
if offer < MinPrincipal: no offer
offer = { Principal: offer, MaxTerm: MaxTermTicks, RateBP: RateBP }
```

with `Bids` the sum of limit times quantity of the intents of the plan it has just
built (their reserved cash, 5.2.1). It keeps `KeepTicks` of its upkeep cost, so its
own tier 1 is funded next tick whatever the round does, and offers only a share of
the rest, so that it does not lend its last unit. **The posted rate is its block's**,
not a function of anything it sees (position 6).

10.3.1. **A lender lives on what it keeps.** A lender with no other income eats from
its cash until its claims fall due and interest arrives: with `KeepTicks` 3 and a
term of 65 ticks it has no money for the other 62 ticks of a loan, and the model of
19.2 shows lenders going unfed. The rule does not protect it, because what a lender
needs is a fact about its own income: a pack gives a lender role a recipe, a grant, or
a `KeepTicks` of at least its longest term (the warnings `lender_without_income` and
`lender_keep_below_term`, 15.2).

### 10.4 A consumer's strategy

It may append a request or an offer of its own and ignore the block's terms, inside
5.2. It receives the group of 10.1. It may append neither. The default strategy's
rules of 10.2 and 10.3 are exported as functions of the view, in the way
`DefaultStrategy` is exported (`spec-agents.md` 6.7), so a wrapper can call them
(architecture.md 4.7: held in a named field, not embedded).

### 10.5 The directed agent and the player

The `Directed` strategy's plan is the fallback's changed by the directive
(`spec-agents.md` 10.4). A player whose role has a block therefore borrows or lends
by the fallback's rules (the default's) with nothing asked of the consumer. A
consumer that wants its player to **ask for a given sum** needs two optional fields
on the `Directive` (`spec-agents.md` 10.3): `Borrow *CreditRequest` and `Lend
*CreditOffer`, which replace the fallback's. They are an edit owed to AGT and to INT
(21), validated at `Direct` (`ErrModuleOff`, `ErrOutOfBounds`, `ErrNotOnRole` for a role
with no block), and not a new mechanism.

### 10.6 No state

The default strategy keeps none. A request or an offer lasts one tick. Nothing of the
lender's or the borrower's history is saved in an agent record (N-128): the
claims are in the ledger.

## 11. Welfare and the neighbouring specs

### 11.1 Interest in the welfare ratio

With the module on, `income_t` of `spec-welfare-and-wages.md` 2.1 becomes

```
income_t = R_t + K_t + K^own_t + F_t + G_t - S_t(Input) - W_t - I^out_t
```

with `I^in_t` the interest the agent **received** in tick `t` (`loan_interest` to it, at
P1.7a), which is a term of the signed unearned sum `G_t` (WEL 2.1.9), and `I^out_t` the
interest it **paid** (the letters are WEL's; `Y_t` and `V_t` there are wages). **The principal is neither**: a loan is a
balance-sheet exchange, so `loan_out` and `loan_repay` are not income and not cost,
and the household that borrows to eat does not look richer, and the lender that is
repaid does not. The two interest terms follow the rule of inputs (WEL 2.1.3): a
financing cost is netted when paid, cash and not accrual, because the ledger records
cash and the accrual needs no valuation.

11.1.1. **`I^in_t` is unearned income**, like relief `F_t` and a grant: it is in
`income_t` through `G_t` (WEL 2.1.9, which states the one rule per flow), out of the earned
ratio of WEL 2.1.7 and **excluded from the observed wage** (WEL 4.3), because a lender is
not paid for working by it. It adds **no field to the record**: `G_t` is the one signed
field that holds it (WEL 13, CHK tag 7). `I^out_t` needs no field either: it is inside
`income_t`, and the observed wage nets it as it nets inputs (a wage is the profit of a
working tick). The principal paid on a **payout claim** is not this spec's: it is income and
cost of the insurance result `N_t` (`spec-insurance.md` 11.1), and the principal of a loan
stays neither.

11.1.2. **A lender's ratio.** A role with a `Lend` block and no recipe has income
`I^in_t` only, and a ratio of interest over upkeep cost. It is a non-producing class
(`spec-agents.md` 3.3.1) that lives on a grant or on interest: whether it is fed is a
question of the posted rate, the principal it has out and the borrowers' repayments,
and it is the pack author's (`lender_without_cash`, `lender_without_income`, 15.2).

### 11.2 The window and the ratio of a long job

A loan does not change how a long job reads in the ratio. The derived window of WEL
2.3.3 makes the window exceed the job; the loan makes the household able to pay its
upkeep and inputs through it. With credit, a household that would have gone unfed
at tick 20 of a 60-tick job eats, its `Privation` does not rise, relief does not draw
the pool's money, and the ladder is not called. Its ratio is still read over a
window that includes the interest as a cost. **Credit is the remedy for the cash
gap, not for the measurement** (position 14; N-032's "real remedy").

### 11.3 The target-income rule

Interest paid lowers `income_t`, so a debtor's ratio is lower than it would be and
the rule of WEL 5 works it more days: a debt is a labour supply, as a lower wage is
(CRD-A2, a case of STB-L14). With a staple the rule is unit-elastic and the loop is
the one STB-L14 already records; credit adds the interest as a second reason for a
household to work. The default strategy's band, share and work floor (D09, D13, D28)
are unchanged.

### 11.4 The other specs

| Spec | What credit means there | Where |
|------|-------------------------|-------|
| MON | no faucet, no sink, no flow; interest redistributes. A pack that wants a lender funded from outside has a grant to the lender role. The identities of `M_h`, `M_p` and `M_c` are unchanged. A lender's claims are not money, so MON's holding charge and its bound `m*` (9.3) bound a lender's money and not its claims | MON 3.1.2, 9.3, 9.5 |
| EST | claims pass to the pool when a creditor is removed (9.2); the pool receives debt service; a removed debtor's money pays creditors first (9.1); relief and debt (8.8) | EST 2.1, 2.2, 2.6 |
| SPN, POP | the ladder reads `Privation`, not debt; a household unfed by its repayment climbs it (shielded by `ShieldTicks`); POP's `deaths` and SPN's `starvation` reductions leave the claim (9.3); a born household brings its stake and no debt | SPN 5; POP 4 |
| MIG | no refusal (9.4.2); MIG's estimate reads no debt (N-110: no aggregate a household may react to; a household's own debt is its own state and the rule is MIG's to say, open question 10) | MIG 4, 7.2 |
| MER | a merchant that cannot afford its lot is limited to a smaller lot (`Unaffordable`); a merchant role with a `Borrow` block would fund it, and the lane lock and the lot cap are unchanged. A delivery contract's escrow is cash, not credit | MER 6.2.1, 9.10 |
| EMP | an employer's `employer_funds` refusal is a cash gap; a role with both an employer block and a `Borrow` block borrows by the default rule only for its upkeep and tier 2, **not** for the wage bill (EMP's cost is not in `Tier2`); a wage advance is not offered | EMP 5.5, FR-EMP-042 |
| LND | a title costs money, and a loan funds the call `TransferTitles` as any money does; no title is pledged (3.5). SAV, not this spec, is the buyer's decision | LND 6.2, 6.3 |
| STO | a store's cost and its room are not credit; a storage cost is a MON flow; a pledge of room is not a pledge for a loan | STO 4, 5 |
| SKL | a cash premium and credit (SKL 11.4 item 3): a trainee's premium is a loan the trainer makes; not offered, as no cash premium exists | SKL 7 |
| SUB, DEM | a household in the own-use mode feeds itself and borrows by the same rule; a want is funded from the spend cap, after interest was netted from income: a debtor's wants are fewer by the interest | SUB 5; DEM 4 |
| EXT, INF, TRN | an outside node is never a counterparty; no rumour of a rate; a loan has no cargo | EXT 4.4; INF 3 |
| VIA, REF | the steady state has no net debt: each loan is repaid inside a cycle; a warm start has **no claims**; the reference chain's 60-tick job role (C04) is the scenario (CRD-S1) | VIA 8; REF 3 |

## 12. Position in the tick, draws and cost

### 12.1 The positions

Two new positions and the removal steps, in the form of `spec-world.md` 6.3 and 6.5
(`6.1.10` there: a spec that needs a position states its phase, its reads, writes
and draws, and the change enters the table with the spec). Nothing is renumbered.

| ID | What runs | Owner | Reads / writes | Draws | Events |
|----|-----------|-------|----------------|-------|--------|
| P1.7a | the settlement: for each debtor with a claim due or overdue, ascending agent ID: the pot, the allocation, the payments; then the ends, the defaults and the write-offs | CRD 8 | claims, balances, the upkeep cost, the tick; writes balances, claims, totals, `Overdue` | none | `LoanEnded` per ended claim in ascending claim ID, then `CreditSettled` per region ascending |
| P3.3b | the credit round: admission of the requests and offers of the plans, the order, the matching, the formation | CRD 5, 6 | the plans, balances less reservations, the claims, the rate scales; writes balances, claims | none | `LoanFormed` per claim in formation order, then `CreditRound` per region ascending |
| removal step 2d, 2e | the waterfall and the passing of claims (9.1, 9.2), inside `spec-agents.md` 3.6 | CRD 9 | claims and the balance of the removed agent | none | `LoanEnded` in ascending claim ID (2d), then `LoanPassed` in ascending claim ID (2e), then `AgentBankrupted` and `EstateOpened`, in the order of `spec-world.md` 6.11 item 8 |
| call | `SetRateScales` (13.1) | CRD, INT | the scale | none | `CreditRateScaled`, one per changed region |
| call | `ForgiveClaims` (13.2) | CRD, INT | claims | none | `LoanEnded` (cause `forgiven`), one per claim ended, in ascending claim ID, at the call |

12.1.1. The events of a step are in the order of `spec-world.md` 6.11: in item 2
(phase 1) the `LoanEnded` and `CreditSettled` events after `MoneyFlowLimited` and
`MoneyFlowEnded` of P1.7; in item 4 (phase 3) the `LoanFormed` events then
`CreditRound` after `MerchantsPlanned` and before the estates' `OrderPlaced`; in item
8 the claims of a removal after `ContractSettled` and before `AgentBankrupted`.

### 12.2 Draws: none

The module takes **no word** of the world's stream and adds no row to `spec-world.md`
7.3. Every tie is the stateless key `K(t, id)` of WLD 10.1.1 (6.3.1, 8.3). The
count of draws of a tick is therefore unchanged by the module on or off and by any
parameter of it, so a paired comparison across credit settings is aligned while no
other count moves (`spec-world.md` 7.6.1; unlike a module that draws, a change of
`ShieldTicks` or of a posted rate moves no later word, and is not the flag `d` of
`spec-calibration.md` 4.2.1). WLD 7.5 gains an entry in its list of what does not
draw: the credit round's order and the settlement's remainders.

### 12.3 The cadence

Both positions run **every tick** and have no rate, window or gate of their own
(`spec-world.md` 9). The only time in the module is a claim's term, which is data of
the loan. No decision rate is derived from an aggregate (WLD 9.4): a request reads
the agent's own money, upkeep, job and claims; an offer its own money and bids.

### 12.4 Cost

Counts, not timings (N-015):

| Position | Cost | Dominant term |
|----------|------|---------------|
| P1.7a | `O(debtors visited + their claims)`; the visited are those with a claim due this tick or overdue, found from the derived indexes | claims falling due and overdue, not open claims |
| P3.3b | `O(Q + F)` per lane for the sort of requests and offers by rate and key, `O(fills)` for the matching | agents with a block |
| removal 2d, 2e | `O(claims of the removed agent)`, found from the derived index by account | claims of one agent |
| `Credit` group of the view | `O(claims of the agent)` for a role with a block, once per `Trade` | claims of one agent |

A world with the module off has none of it. A step of the reference chain with the
module on and no role with a block allocates nothing beyond the default (NFR-CRD-001).

## 13. The slot and the call

INT is minimal (ADR 0003 decision 2, N-131). Credit adds **one slot and one call**,
each for a reason the engine cannot do without a consumer.

### 13.1 `SetRateScales`

```go
// SetRateScales sets the scale, in basis points, of the posted rates of each region named
// (the type is INT's `RegionScale{Region; BP}`, reused). The default is 10 000.
func (w *World) SetRateScales(scales []RegionScale) error
```

| Item | Value |
|------|-------|
| Value bound; default | 0 to `MaxShockBP`; 10 000. 0 is **interest-free**, not a pause |
| Needs | `Modules.Credit` |
| Owner | this spec, 6.3.1 |
| Read by the tick at | P3.3b: the effective rate of an offer is `halfEven(RateBP * ScaleBP / 10 000)`, clamped to `MaxCreditRateBP` |
| Event (at the call) | `CreditRateScaled`: `Region`, `BeforeBP`, `AfterBP`, one per changed key, in ascending `RegionID` |
| Saved by | this spec, 17: the non-default scales, sparse; **not hashed** |
| Errors, in order | `ErrModuleOff`, `ErrUnknownRegion`, `ErrOutsideNode`, `ErrOutOfBounds`, `ErrDuplicateID` |

13.1.1. **Why a slot.** The posted rate is data (position 6) and no rule moves it.
PLC, a human at a console or a scripted shock changes the price of credit of a
region through this slot, as MON's `SetFlowScales` changes a flow
(`spec-money-supply.md` 10.4) and LND's `SetRentScales` a rent; the engine contains no
rule that moves the scale in response to a measure. With the policy module on the slot is a
lever of kind `rate` of a policy actor (`spec-policy.md` 6.3.2: reserved in version 1).

13.1.2. **What it changes.** New loans only: a claim's rate is fixed at formation
(7.1.3). A scale that raises the effective rate above every borrower's `MaxRateBP`
closes the lane without a second slot: the way to stop credit in a region is a
scale above the limits, and the way to make it free is 0. A **rate limit** (ORG 6.6) closes
a lane in the same way: a lender whose effective rate is above the cap finds no borrower
whose limit is above it.

### 13.2 `ForgiveClaims`

```go
// ClaimFilter names the claims to forgive by the debtor's region and role at the call.
type ClaimFilter struct {
	Region     RegionID // the debtor's region, when AllRegions is false
	Role       RoleID   // the debtor's role, when AllRoles is false
	AllRegions bool
	AllRoles   bool
}

// ForgiveClaims ends every open claim of origin `loan` that matches any filter, with no
// transfer, and returns how many it ended; a payout claim is never ended by it. The creditors lose what was owed.
func (w *World) ForgiveClaims(filters []ClaimFilter) (ended int64, err error)
```

| Item | Value |
|------|-------|
| Needs | `Modules.Credit` |
| Acts | **at the call**, between ticks: each matching claim **of origin `loan`** (a decree that cancels debts does not cancel an insurer's obligation, `spec-insurance.md` 9.2.2) is written off (`WriteOff`), cause `forgiven`, in ascending claim ID |
| Event (at the call) | `LoanEnded` (cause `forgiven`, `LostPrincipal` and `LostInterest` the owed), one per claim, carrying the tick the next step runs |
| Errors, in order | `ErrModuleOff`, `ErrUnknownRegion`, `ErrOutsideNode`, `ErrUnknownRole`, `ErrDuplicateID` |
| Saved by | nothing of its own: the claims it removed are not in the list; the totals `LostPrincipal` and `LostInterest` rise |

13.2.1. **Why a call and not a slot.** It is a one-off change of state with no value
to hold (`spec-world.md` 8.2 lists slots as values held until another call, 5.2.2
there); it is the one thing a game does to debt that no rule does, and the debt
cancellation of a story, a decree or a victory needs it. It moves no money
(position 2: nothing is created or destroyed), changes no balance, and the
**creditors' loss** is the cost, so the audit's identity holds with the lost totals raised by
what was owed. A filter that matches nothing is legal and returns 0.

### 13.3 What is not offered

A loan between two named agents (a story loan), early repayment, a change of a
claim's rate or due tick, a freeze that is not a scale, the cancellation of a single
claim by ID, of a payout claim (`spec-insurance.md` 9.2.2), and a call that makes the estate lend. Each is open question 11 or a
consumer's own loop outside the engine (N-131: "the engine guarantees conservation
and determinism, not its stability"). INT's table gains the two rows (21).

## 14. Parameters

All are fields of `Config` with a default (or "required"), a bound, a doc line and
an owner (architecture.md 9.5 to 9.7). **Every default is a placeholder** (N-015,
N-027). The bounds are validation bounds in `bounds.go` (`spec-config.md` 4.1).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Credit` | the credit module: present means on (`spec-config.md` 3.2) | nil (off) | |
| Credit | `MinPrincipal` | the smallest principal of a claim (4.1) | required | 1 to `MaxMoney` |
| Credit | `ShieldTicks` | ticks of the debtor's upkeep cost out of reach of repayment (8.2) | 0 (no shield; the warning `no_shield` with a borrower role) | 0 to `MaxCreditTerm` |
| Credit | `WriteOffTicks` | ticks after `Due` at which an unpaid claim is written off; 0 means never (8.5) | 0 | 0 to `MaxCreditTerm` |
| Credit | `MaxClaims` | the most open claims at once (6.4 step 1) | required | 1 to `MaxOpenClaims` |
| Role | `Credit` | the credit block; nil means the role does not deal in credit (4.2) | nil | at most one of `Lend`, `Borrow` |
| Lend | `RateBP` | the posted yearly rate, simple (7.1) | required: a file that omits it is `ErrMissingField`; 0 is interest-free and is written, never defaulted | 0 to `MaxCreditRateBP` |
| Lend | `MaxTermTicks` | the longest term the default lender lends for | required | 1 to `MaxCreditTerm` |
| Lend | `KeepTicks` | ticks of its upkeep cost the default lender keeps out of its offers (10.3) | 0 | 0 to `MaxCreditTerm` |
| Lend | `ShareBP` | the share of its lendable money offered in one tick (10.3) | required | 1 to 10 000 |
| Borrow | `MaxRateBP` | the highest yearly rate the default borrower accepts (10.2) | required: a file that omits it is `ErrMissingField`; 0 accepts only interest-free credit | 0 to `MaxCreditRateBP` |
| Borrow | `MaxDebtTicks` | the exposure cap in ticks of upkeep cost (5.3.3) | required | 1 to `MaxCreditTerm` |
| Borrow | `SlackTicks` | ticks added to the horizon of the gap, beyond the structural tick (10.2) | 0 | 0 to `MaxCreditTerm` |

The slot `SetRateScales` (13.1) is not a field: it is world state, not hashed, and has
no row in the owner table (as `Scale` of a money flow has none).

**New bounds for `bounds.go`** (provisional, with the reasons):

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxCreditRateBP` | 100 000 | ten times the principal a year: a rate above it is a typing error, and with `MaxCreditTerm` and `MaxMoney` the interest product fits 128 bits (7.1.1) |
| `MaxCreditTerm` | 3 650 | ten Earth years of daily ticks, the order of `MaxWelfareWindow`; a term above it is a different instrument |
| `MaxOpenClaims` | 1 000 000 | the claims in the list at once; with `MaxMoney` the owed over all claims is at most the money in existence |

The slot's bound is the existing `MaxShockBP` (`spec-resources.md`, 100 000), and
`MaxMoney` and `MaxPrice` are existing.

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the
interest formula and its half-to-even rounding (7.1); the interest-first and
pro-rata allocation with the largest remainder and its ties (8.3); the order of the
checks of 5.2 and 6.4; the matching order and the stateless key (6.3); at most one
request and one offer per plan (5.2); the reasons and causes: the refusal reasons of
5.2 and the causes `repaid`, `written_off`, `debtor_removed`, `forgiven`, exported
constants of the root package (`event.go`); the three ledger reasons (3.3); the
positions P1.7a and P3.3b; the rule that a debtor in arrears is refused (5.2); the
waterfall's ranking (9.1); the passing of claims to the estate (9.2).

**Not parameters:** a claim's fields, `CreditRoom`, `Debt`, `Owed`, the rate scale (state),
the cumulative totals, the derived indexes, the effective rate.

The placeholders, so that none is mistaken for a result: every default and bound
above, the thresholds of the warnings of 15.2, and every number of the worked
examples (20) and of the models (19).

## 15. Validation

Each rule has a sentinel or a warning name and a failing-case test (testing.md
3.9). Errors name the path, for example `role.borrower.credit.borrow.max_debt_ticks`.
The framework of `spec-config.md` 7 runs them.

### 15.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field of 14 outside its bound; `MinPrincipal` or `MaxClaims` below 1 |
| `ErrCreditShape` (new, a root sentinel) | a `CreditTerms` with both `Lend` and `Borrow`, or with neither |
| `ErrModuleDependency` | a role with a credit block while `Modules.Credit` is nil |
| `ErrReasonCollision` | a money flow whose ID is `loan_out`, `loan_repay` or `loan_interest` (`spec-money-supply.md` 4.5.2) |
| `ErrMissingField` | a `rate_bp` of a `Lend` block or a `max_rate_bp` of a `Borrow` block absent from a file (FR-CFG-008): a rate of 0 is legal and is written, so an omission is never a silent interest-free loan; a Go-built configuration has no absent value, and the warning `credit_free` covers it |
| `ErrUnknownRole`, `ErrDuplicateID` | as for any role row |
| `ErrTooManyClaims` (the ledger's, owned here) | `Obligate` of `spec-insurance.md` 9.2 with `MaxClaims` claims already open; `Lend` is checked by 6.4 step 1 before it reaches the ledger and is counted as `too_many_claims` |

No overflow error is needed at validation: a loan whose principal and interest
would pass `MaxMoney` is refused at formation (6.4 step 2) and is a counted
reason, not an error. A strategy's request or offer outside its bounds is a refusal
(5.2), not an error returned by `Step`.

### 15.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at
construction, at `Warn`, with the attributes `rule` and `path`; the volume is bounded
by the configuration. The thresholds are placeholders.

| Warning | Condition |
|---------|-----------|
| `no_credit_roles` | `Modules.Credit` is present and no role has a block |
| `credit_no_lender` | a role has a `Borrow` block and no role has a `Lend` block: no request can be filled |
| `credit_no_borrower` | a role has a `Lend` block and no role has a `Borrow` block, and no consumer strategy is registered that could request: the offers are never used (a consumer's own strategy is outside validation, so the rule reads only the blocks and is informational) |
| `rate_above_limits` | the lowest `RateBP` of any lender exceeds the highest `MaxRateBP` of any borrower at a scale of 10 000: no loan can form |
| `rate_limit_below_floor` | (ORG's, `spec-institutions.md` 10.2, the mirror of `rate_above_limits`: a rate limit below the lowest posted `RateBP` of every lender in its scope) |
| `term_exceeds_lenders` | the longest recipe `Duration` plus 1 plus `SlackTicks` (plus 1 more where `Late` can be 1, 10.2.1.2) of a borrower role exceeds every lender's `MaxTermTicks`: the default request of that role is never filled |
| `cap_below_horizon` | `MaxDebtTicks` of a borrower role is below its longest recipe `Duration` plus 1 (or 2) plus `SlackTicks`: the cap may cut the request below the gap of a job |
| `lender_without_cash` | a lender role whose `StartMoney` less `KeepTicks` of upkeep cost at the reference prices is below `MinPrincipal`: the household can never offer |
| `lender_without_income` | a lender role with no recipe and no money flow that targets it: its only income is interest, which needs a borrower to pay it |
| `lender_keep_below_term` | a lender role with no recipe and no money flow that targets it whose `KeepTicks` is below its `MaxTermTicks`: it lends money it must eat before the money returns (the lenders of cases E and F of 19.2) |
| `no_shield` | a role with a `Borrow` block and `ShieldTicks` 0: a debtor can be paid down to nothing |
| `credit_no_upkeep` | the world has no upkeep anywhere (`no_upkeep` of `spec-agents.md` 13): every `CreditRoom` is 0 |
| `credit_free` | every `Lend` block has `RateBP` 0: all credit is interest-free, which is legal and rarely meant |
| `interest_rounds_to_zero` | the interest of `MinPrincipal` at a rate above 0 and a term of 1 tick rounds to 0 (7.3) |

### 15.3 What is not validated here

A borrower role with no recipe (a household that borrows to eat, a class of
non-producing households: legal, and its repayment is whatever income another spec
gives it); a lender role with recipes (legal, 4.4.3); a role whose block is read by
a consumer's strategy only; two lender roles at different rates (they compete in
the order of 6.3).

## 16. Observability

### 16.1 Events

Six events, all plain data with no map, pointer or wall-clock time, constructed only
when an observer was supplied (`observability-and-logging.md` 4.1, 4.5). Three are per
claim (class **E**, like `ContractFormed`); two are aggregates, bounded by the regions
(class **A**, like `ContractRound`); one is the slot's (class **C**). In the form of `spec-observability.md` 4.1. `Amount` is `Money`.

| Event | Owner | Position | Class | Gate | Fields (besides `Tick`) |
|-------|-------|----------|-------|------|-------------------------|
| `LoanFormed` | CRD 6.4 | P3.3b, formation order | E | credit | `Claim`, `Creditor`, `Debtor` (agents), `Region` (the debtor's), `Principal`, `Interest`, `RateBP`, `Term`, `Due` |
| `LoanEnded` | CRD 8.5, 9.1, 13.2 | P1.7a, in ascending claim ID; with a removal (2d); at the call of `ForgiveClaims` (loans only) | E | credit | `Claim`, `Origin` (`loan` or `payout`: the event of a claim of either origin, a payout claim's end being INS's, `spec-insurance.md` 15.1.2), `Creditor`, `Debtor`, `Region` (the debtor's account at the event), `CreditorRegion` (the creditor's account at the event), `ToEstate bool` (the creditor is an estate), `Principal`, `PaidPrincipal`, `PaidInterest` (the waterfall's payment at a removal, 0 for other causes), `LostPrincipal`, `LostInterest`, `Cause` (`repaid`, `written_off`, `debtor_removed`, `forgiven`) |
| `LoanPassed` | CRD 9.2 | with a removal (2e), ascending claim ID | E | credit | `Claim`, `Creditor` (the removed agent), `Debtor`, `Region` (the estate's), `Owed` |
| `CreditRound` | CRD 6 | P3.3b, region ascending, after the last `LoanFormed` | A | credit | `Region`, `Requests`, `Offers` (admitted), `RequestsRefused`, `OffersRefused` (refused at admission, any reason of 5.2 but `credit_off`), `InArrears`, `Exposure` (the part of `RequestsRefused` refused with that reason: subsets, not additions), `Formed`, `Principal` (formed), `Interest` (fixed), `Unfilled` (principal asked by admitted requests and not filled at the end of the round), `TooMany`, `Overflow` (fills dropped at formation, 6.4 steps 1 and 2: not refusals, and not in `RequestsRefused`) |
| `CreditSettled` | CRD 8 | P1.7a, region ascending, after the `LoanEnded` events of the tick | A | credit | `Region`, then, **over claims of origin `loan` only** (the payments on a payout claim are INS's `CoverSettled` of stage `claim`, `spec-insurance.md` 15.1), for its **debtors**: `Due` (claims falling due), `Repaid` (claims ended repaid), `Defaulted` (claims newly overdue), `WrittenOff`, `PrincipalPaid`, `InterestPaid`, `Lost`, `Shielded` (the money the shield kept out of a payment: over the debtors visited, `min(balance, owed) - pay`, 0 where the shield did not bind), `Headroom` (payment cut at a creditor), `Arrears` (the owed on overdue claims at the end of the pass); and for its **creditors**: `Received` (money received by agents whose account is in the region) and `ReceivedByEstate` (money received by the region's estate) |
| `CreditRateScaled` | CRD 13.1 | call, ascending `RegionID` | C | credit | `Region`, `BeforeBP`, `AfterBP` |

16.1.1. **When.** `LoanFormed` is emitted per claim formed. `CreditRound` is emitted per
region and tick in which a request or an offer was submitted for it, and never when
all its counts are 0. `CreditSettled` is emitted per region and tick in which a
claim of a debtor of the region fell due, was overdue or was written off, or in which
a creditor of the region (an agent or its estate) received money, and never when all
its counts and sums are 0. `Arrears` is the owed on overdue claims of the region at the end of the pass, a level; it
is the counterpart of the "claims overdue" `Warn` (16.2) and is not the feed of a metric
(16.3). With the module off none is emitted, and the refusal `credit_off` (5.2) is
therefore never in an event: it is a `Debug` line only.

16.1.2. **The order.** Phase 1: after `MoneyFlowLimited` and `MoneyFlowEnded` of P1.7,
each `LoanEnded` of P1.7a in ascending claim ID, then `CreditSettled` per region
ascending. Phase 3: after `MerchantsPlanned` (P3.3a) and before the estates'
`OrderPlaced` (P3.4), each `LoanFormed` in formation order, then `CreditRound` per
region ascending. A removal: after the events of the steps before it (`ContractSettled`, the hires and
the titles of other specs) and before `AgentBankrupted`, each `LoanEnded` (2d) then each
`LoanPassed` (2e). A call
between ticks emits at the call, before the next `TickStarted`, with the tick the
next step runs (`spec-world.md` 8.1.1).

16.1.3. **What an observer can rebuild.** Which claims are open, from `LoanFormed`
(add), `LoanPassed` (change the creditor) and `LoanEnded` (remove), with each one's
creditor and the cause and the money paid and lost at its end. A **payout claim** has no
`LoanFormed`: it is added by `CoverEnded.Short` with its `Claim` (`spec-insurance.md`
15.1.2) and ends in a `LoanEnded` of origin `payout`; an observer keeps the two origins
apart by `Origin`. The money of the household sector and of the pool by region
(`spec-observability.md` 4.3.3) is rebuilt with three more readings in its list (the first
two are loans only, the third of either origin; a payout claim's payments at P1.7a are in
`CoverSettled`). **It cannot rebuild the owed of an open claim between its
formation and its end**: a `CreditSettled` row is a sum of a region with no claim ID,
and that figure is in the snapshot's `Claims` (16.4):

1. **a loan** at P3.3b moves `Principal` between two households of
   `LoanFormed.Region` (a lane is a region, so both are there) and changes the money
   of no region;
2. **a payment** at P1.7a lowers the money of the households of
   `CreditSettled.Region` by `PrincipalPaid + InterestPaid` and raises that of the
   households of the region by `Received` and its pool by `ReceivedByEstate`;
3. **a waterfall payment** is the `PaidPrincipal + PaidInterest` of a `LoanEnded`, of
   either `Origin`, with the cause `debtor_removed`, from the households of its `Region` to those of its
   `CreditorRegion`, or to that region's pool when `ToEstate`.

None of the three is in `BoundaryFlow` (the reasons are internal), so without the
fields above the money of a region could not be rebuilt from the events
(FR-CRD-079).

### 16.2 Logs

`Debug`, guarded by `Enabled`, constant messages: "loan formed" (`claim`, `agent`
the debtor, `region`, `principal`, `rate`, `term`), "loan ended" (`claim`, `agent`,
`cause`, `lost`), "claim passed" (`claim`, `agent` the removed creditor,
`region`), "claim overdue" (`claim`, `agent`, `owed`), "request refused" and
"offer refused" (`agent`, `reason`). **No `Info` in the tick and no `Warn` per claim
or per agent** (3.2.2): a claim and a refusal are per entity. The warnings of 15.2
are reported once at construction, and the summary line "world created" gains `credit`
among its `modules` and the counts `lenders` and `borrowers` of agents with each
block.

**`Warn`**, two lines, each with its counterpart (`spec-observability.md` 7.4):

| # | Message | Pattern, condition | Keys | Cells bounded by | Counterpart |
|---|---------|--------------------|------|------------------|-------------|
| 20 | "claims overdue" | B: the debtors of a region owe overdue claims of origin `loan` at the end of P1.7a in each of `WarnRun` consecutive ticks | `region`, `claims`, `arrears` | regions | `CreditSettled.Arrears`; the `Claims` rows of the snapshot (`Overdue` at or above 0) |
| 21 | "claim cap reached" | A: a request was cut because `MaxClaims` open claims (of either origin) exist | `claims` | the world | `CreditRound.TooMany`; the length of `Claims` |

**New log keys** for `observability-and-logging.md` 3.3.2: `claim` (a claim ID,
`int64`), `claims` (a count of claims, `int64`), `rate` (a yearly rate in basis
points, `int64`), `term` (a term in ticks, `int64`), `arrears` (money, the owed on
overdue claims, `int64`), `principal` (money, the principal of a claim, `int64`),
`owed` (money, the principal and interest owed on a claim, `int64`), `lost` (money,
the principal and interest lost at the end of a claim, `int64`), `lenders` and
`borrowers` (counts of agents with each block, in the construction summary). The keys
`region`, `agent`, `cause`, `reason` and `modules` exist; the key `amount` is not
used, because the rules define it for the line of a refused transfer.

### 16.3 Metrics (proposed)

All `Int64`, each derivable from the events or, for the arrears gauge, from the
snapshot (`observability-and-logging.md` 5.3):

| Name | Instrument | Unit | Attributes | Fed by |
|------|------------|------|------------|--------|
| `economysim.credit.formed` | Counter | `{claim}` | none | `LoanFormed` (a loan; a payout claim has no `LoanFormed`) |
| `economysim.credit.principal` | Counter | `{minor_unit}` | none | `LoanFormed` (`Principal`) |
| `economysim.credit.interest` | Counter | `{minor_unit}` | none | `CreditSettled` (`InterestPaid`) and `LoanEnded` of origin `loan` (`PaidInterest`) |
| `economysim.credit.outstanding` | UpDownCounter | `{minor_unit}` | none | `LoanFormed` (+ `Principal`), `CreditSettled` (- `PrincipalPaid`), `LoanEnded` of origin `loan` (- `PaidPrincipal` and `LostPrincipal`); seeded at a restore (below) |
| `economysim.credit.arrears` | Gauge | `{minor_unit}` | `economysim.region` | **the snapshot**: `Claims[]` rows of origin `loan` with `Overdue` at or above 0, summed by `Region` as `PrincipalOwed + InterestOwed` |
| `economysim.credit.defaults` | Counter | `{claim}` | none | `CreditSettled` (`Defaulted`; loans only) |
| `economysim.credit.lost` | Counter | `{minor_unit}` | `economysim.cause` | `LoanEnded` of origin `loan` (`LostPrincipal` plus `LostInterest`) |
| `economysim.credit.refused` | Counter | `{request}` | none | `CreditRound` (`RequestsRefused` plus `OffersRefused`: requests and offers, all reasons of 5.2 but `credit_off`) |
| `economysim.credit.limited` | Counter | `{case}` | `economysim.credit_limit` | `CreditRound` (`InArrears`, `Exposure`, `TooMany`, `Overflow`, by limit; the first two are also in `refused`, so the two instruments are not to be added) |

Every instrument counts claims of origin `loan` only: a payout claim is counted by INS
(`economysim.insurance.claims`, `spec-insurance.md` 15.3), so no credit metric counts one
The attributes are bounded by the configuration or by closed sets
(`observability-and-logging.md` 5.4.1): `economysim.cause` gains the four values of
`LoanEnded.Cause`, and `economysim.credit_limit` is new with the four constants
`in_arrears`, `exposure`, `too_many_claims` and `overflow` (the dampers of FR-STB-013,
as `economysim.entry_limit` is for the spawner). The series added for the reference
chain: the arrears gauge, one per region (5); the lost counter, four; the limited
counter, four; the rest one each.

16.3.1. **Which are fed by the snapshot, and the restore.** `economysim.credit.arrears`
is a level that changes with no `CreditSettled` (an overdue debtor removed in
phase 7, a claim ended by `ForgiveClaims`, a debtor that relocates), so it cannot be
fed by that event, and it is fed by the snapshot as the four instruments of
`observability-and-logging.md` 5.3.1 are: recorded once per snapshot the consumer
passes, absent until one is, holding its last value between two. It is the fifth
instrument of that kind. `economysim.credit.outstanding` is fed by events and is, like
every level instrument, **seeded at `WorldRestored`** (`spec-observability.md` 6.4.2):
an adapter built after a restore starts it at the sum of `PrincipalOwed` over the `Claims`
rows of origin `loan` of the snapshot the event carries, and then adds the events after it
(FR-CRD-077). The three totals alone no longer give it: `Repaid` and `LostPrincipal` count
claims of both origins (3.3.2), so `Lent - Repaid - LostPrincipal` is the loans' share only
when no payout claim has been paid or lost.

### 16.4 Snapshot

Deep copies, bounded by `MaxClaims` (architecture.md 12.4, `spec-observability.md`
5.6), cost class F (a copy of an in-flight set). `Claims` is a fifth in-flight set
besides `Shipments`, `Away`, `Journeys` and `Contracts`, so the snapshot stays
proportional to the agents only through those five sets (`spec-observability.md`
5.6.1, an edit owed). Names are OBS's:

| Slice or field | Row | Fields | Source |
|----------------|-----|--------|--------|
| `Claims` | by claim ID | `Claim`, `Origin`, `Creditor` (an `AgentID`, or the estate's agent ID, which is its `RegionID`), `Debtor`, `Region` (the debtor's), `Principal`, `Interest`, `PrincipalOwed`, `InterestOwed`, `RateBP`, `Formed`, `Due`, `Overdue` | CRD 3.1 |
| `Credit` | one value | `Lent`, `Awarded`, `Repaid`, `InterestPaid`, `InterestFixed`, `LostPrincipal`, `LostInterest` (the seven cumulative totals of 3.3.2) | CRD 3.3.2 |

`RateScales` are not snapshot fields: the non-default ones are INT's `Slots()`
(`spec-integration.md` 5.3), as the other slots are. The claims of one agent are in
INT's `AgentState` (the group of 10.1 without the block). Every field is a function
of saved state (N-125, FR-OBS-043).

### 16.5 The Observability entry (observability-and-logging.md 9)

1. **New failure path or rejected input?** A refused request or offer is a counted
   reason, not an error (5.2); the validation errors of 15.1 are returned; a
   payment cut at a creditor's headroom is survived and counted (`Headroom`).
2. **New state transition?** A claim formed, ended, passed; a round; a settlement; a
   scale: six events, no map, fixed-point values (16.1).
3. **New decision point in a strategy?** The default borrower and lender (10.2,
   10.3): their inputs are the view's, and the `Debug` line "loan formed" and the
   refusals tell the result; no `Info`.
4. **New loop?** The pass and the round are bounded by debtors and agents with a block
   and log at `Debug`, guarded by `Enabled`; the two `Warn`s are bounded by regions
   and by one (16.2); event construction is gated by the no-observer flag
   (NFR-CRD-001).
5. **Changed event, metric, unit or attribute?** None changed; nine metrics and six
   events are new, and one attribute gains values (16.3).
6. **New metric attribute?** `economysim.credit_limit`, four constants, bounded by the
   configuration (5.4.1 there).
7. **`Error` or `Info`?** None.
8. **Logging an error also returned?** None.
9. **Tests?** `observability-and-logging.md` 8.1 applies: the recorder over a scripted
   loan, a default and a removal (FR-CRD-070 to 079), the capturing handler for the
   two `Warn`s, the no-op allocation test, non-interference, composition and
   emission order, the golden rendering, and the restore case (FR-CRD-087).

## 17. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). Plain data in a fixed order,
no maps; **no per-agent field** is added (N-128): a claim names its two accounts
and lives in the ledger's list.

### 17.1 What is saved

| Where | State |
|-------|-------|
| the ledger section (`spec-checkpoint.md` 4.2 tag 2), replacing "the claims (a count, 0 in format 1)" | the next claim ID; the open claims in ascending ID: ID, `Origin`, creditor and debtor (each an **account ID** of the ledger's own numbering, as the claim is in 3.1.1; the account's kind says Agent or Estate and its owner reference gives the agent or the region, so the section keeps no agent ID), `Principal`, `Interest`, `PrincipalOwed`, `InterestOwed`, `RateBP`, `Formed`, `Due`, `Overdue` (signed); the seven cumulative totals of 3.3.2 (`Awarded` included) |
| a module section, present when `Modules.Credit` is (provisionally "Credit"; the tag is the supervisor's, after Merchants and Employment in dependency order) | the non-default rate scales as (region, `BP`) in ascending region, sparse; the diagnostic run of the "claims overdue" `Warn` per region, sparse; the run of "claim cap reached", one value |

A claim is saved once, in the ledger's section; the module's section lists only its
own scales and runs (N-128).

### 17.2 Why it is the whole state

The settlement and the round read only the list, the balances, the tick, the
configuration and the scales; a claim's due tick and rate are in the claim, and the
view's group is derived. A world restored between ticks therefore settles the same
claims at the same P1.7a, forms the same loans at the same P3.3b, and takes the same
words of the stream (none) as an uninterrupted one (FR-CRD-087).

### 17.3 Derived and rebuilt at restore

| Derived | From |
|---------|------|
| the **due index**: per due tick, the claim IDs ascending (a ring of at most `MaxCreditTerm + 1` buckets from the current tick) | the list |
| the **overdue list**: the IDs of claims with `Overdue` at or above 0, ascending | the list |
| the **by-account indexes**: per debtor and per creditor account, the claim IDs ascending | the list |
| `Debt`, `Owed`, `Overdue`, `Lent`, `Receivable`, `CreditRoom` of an agent | the list, the weights and the upkeep cost |
| the open count against `MaxClaims` | the list |

### 17.4 Not saved

A request or an offer (one tick, scratch), the plan notes, the view's group, the
effective rate of a round, the allocation of a pass. A call between ticks that
forgave claims left no record but the list and the totals.

### 17.5 What a restore checks

In the fixed order of `spec-checkpoint.md` 6.2, after the accounts and the agents are
in: every claim's creditor and debtor are open accounts of the kinds of 3.6 item 3
and differ; IDs ascend and are below the next ID; `PrincipalOwed` and `InterestOwed`
are in range and not both 0; `Due` is after `Formed` and `Formed` not after the tick;
`Overdue` is -1 or in `[Due, tick)`; the identity of 3.6 item 1 holds over the list and
the totals; the number of claims is at most `MaxClaims`; `Origin` is `loan` or `payout`, and the sum of `Principal` over the payout claims ever formed is `Awarded` (no history: the open ones are at most it). A failure is
`ErrCheckpointCorrupt` naming the tag and the claim. The configuration hash covers
`Modules.Credit` and every block, so a restore with another credit configuration
fails the hash (`ErrCheckpointConfig`); the rate scales are state, not hashed.

## 18. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9 (FR-STB-003). Fields as in
its section 3; the register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn`
numbers and the entries here are `CRD-An`, `CRD-Dn` and `CRD-Sn`. **Credit closes
loops of its own**: a debt is a fixed outflow at a known tick, and a default is the
loss of a lender's income; each has an entry. Nothing here has been run in the
library; the status `model` means the rules were run in `models/crd/` (19) with an
assumed income, and no default is claimed stable (N-027).

**Merges into existing loops** (checklist 9.9): CRD-A2 into L14 as a case; CRD-A4
into L22; CRD-A6 into L31 and L45; CRD-A10 into L39, L63 and L40 as a case. CRD-A1,
A3, A5, A7, A8 and A9 are new.

**CRD-A1. The debt-service squeeze.** *Kind:* loop. *Loop:* the closed cash chain is
the one already registered as the poverty trap (L11, L45): cash (+) a job started and
food eaten (+) output (+) income at the next completion (+) cash. A debt adds to it
a **fixed offset**: the principal and interest due at a known tick, an outflow that
does not depend on the chain, and a bar (CRD-A8) that cuts the chain's input when it
goes unpaid. The loop is therefore the existing cash chain with a fixed outflow and a
gate, not a new closed chain of its own: a household whose income over a job is below
its upkeep, its inputs and the interest cannot clear the claim, goes overdue and is
barred from the next loan. Polarity R (that of L11). *Delay:* the term of the loan,
the job's `Duration` plus 1 (plus `Late`) and `SlackTicks` (10.2.1.1, 10.2.1.2). *Owner:* CRD 8, 10.2.
*Damper:* CRD-D2 (no rollover: no new money pays old debt), CRD-D3 (the exposure cap),
the rule that arrears bear nothing (7.2, 8.7), and CRD-D1 (the shield) **with a
caveat**: the shield moves the tick of the due and not the ticks after it, and its
best value depends on the horizon (19.2: the unfed ticks at shields of 0, 1, 2, 3, 5
and 8 are 727, 723, 242, 238, 231 and 272 at `SlackTicks` 0, and best at 1 at
`SlackTicks` 5), so it is an interior optimum and may worsen the trap outside it.
*Undamped:* a household with a persistently negative margin (19.2, case E: 16
defaults, 3 403 unfed borrower-ticks, no recovery without income). *Condition:* the
lump a job yields exceeds its upkeep, inputs and interest over the term; the default
request sizes the loan to the gap and not to the margin, so the condition is the
household's and the pack's. *Test:* CRD-S1, CRD-S2. *Status:* hand for the closing
links (a lender's loss lowers borrowers' output and food: the model assumes the
income); the rules and the shield's shape were run in `models/crd/` (19.2).

**CRD-A2. Interest and the target-income rule** (a case of L14). *Kind:* loop.
*Loop:* interest paid (+) `income_t` falls (-) the ratio falls (-) the work share
rises (+) output (+) supply (-) price (+) income. The interest is a second reason
for a debtor to work, besides the wage; with a staple the rule is unit-elastic
(STB-L14 and 6.1 there) and credit changes only the level at which it works.
*Delay:* the welfare window. *Owner:* WEL 5, CRD 11.3. *Damper:* D09, D13, D28 (the
rest band, the work floor and the proportional share). *Undamped:* as L14: nothing
changes the elasticity. *Test:* S04 with a borrower role (CRD-S1). *Status:* hand.

**CRD-A3. The default cascade.** *Kind:* loop. *Loop:* a borrower's default (+) the
lender's interest and principal lost (+) the lender's cash falls (-) its offers
fall (-) the borrowers' gaps unfunded (+) output and food fall (+) more defaults.
A lender with no other income and `KeepTicks` below the term eats the principal it
lent: its default is the loss of the supply of credit. Polarity R through the supply
of loans. *Delay:* a term, plus the lender's keep. *Owner:* CRD 6, 10.3, 9.2.
*Damper:* CRD-D4 (the lender's reserve and share), CRD-D7 (the claims of a removed
lender pass to the pool, so the debt is not forgiven by its death), D35 (exit through
the ladder), relief. *Undamped:* a lender's portfolio is not diversified by the
engine, and a lender with few borrowers and `ShareBP` 10 000 is the fragile case.
*Condition:* a lender's expected loss over a term is below its interest income, or
it has another income (`lender_without_income`, `lender_keep_below_term` of 15.2).
*Test:* CRD-S2. *Status:* hand for the closing link (the model of 19.2 assumes the
borrowers' income, so it shows the lender's loss and no feedback on borrowers'
output: in case F the lenders go unfed for 316 household-ticks and in case E for
36, and a write-off that lifts the bar makes it worse).

**CRD-A4. The credit pulse** (a case of L22). *Kind:* amplifier. A cold start has
every household of a role at the same point of its job: they ask in one tick, the
lenders' cash falls at once, and every claim falls due in one tick and is repaid in
one, so credit swings with the period of the job. *Delay:* the job's `Duration`
plus 1 and `SlackTicks`. *Owner:* CRD 6, 8; WLD-A3. *Damper:* D22 (the warm start staggers
initial jobs), `ShareBP` (a lender does not offer all in one tick). *Undamped:* a
cold pack, as L22. *Test:* S75's cold leg with a borrower role, CRD-S1.
*Status:* hand.

**CRD-A5. Debt deflation.** *Kind:* loop. *Loop:* the price level falls (-) the
real value of a fixed nominal claim rises (+) borrowers' margins fall (-) defaults
(+) the lenders' cash and offers fall (-) loans (+) households that cannot fund
their gap buy less upkeep (-) demand (+) the price level falls. Two links point
opposite ways: unfunded jobs also cut **supply**, which raises the price (the
balancing link), so the polarity depends on which dominates and is not claimed.
*Delay:* a term. *Owner:* CRD 8, 11. *Damper:* the reference band floors a price
(`spec-beliefs.md` 3.2.1), the shield, `WriteOffTicks`, the consumer's
`ForgiveClaims`. *Undamped:* the price level has no anchor in the interior
(L51). *Condition:* unknown; the model of 19.2 has an assumed income and cannot say.
*Test:* CRD-S2 with a yield shock that raises supply. *Status:* untested.

**CRD-A6. Money held by lenders** (a case of L31 and L45). *Kind:* hazard. *Loop:*
interest paid by borrowers (-) borrowers' money; (+) lenders' money, which the
default lender keeps above its reserve and `ShareBP` leaves unlent; money in the
lenders' hands is spent only as upkeep (tier 1) and a buffer above its reserve
(`spec-agents.md` 7.4.4). Interest therefore moves money from households that
spend it to households that hold it. *Owner:* CRD 7, 10.3. *Damper:* CRD-D4
(`ShareBP`: what is not offered this tick is offered the next), D42 (a rate levy on
the lender class, bound `m*`), D12. *Undamped:* a lender class with `ShareBP` low.
*Test:* S33 with a lender class; the money by role (`spec-indicators.md` 9).
*Status:* hand.

**CRD-A7. Debt stays with the survivors.** *Kind:* hazard. A reduction of weight
(`starvation`, `deaths`) leaves every claim with the agent (9.3): per household the
debt of a cohort that loses half its households doubles, and the money of those
that left went to the pool. A repeated reduction by the ladder (rung 3) is then a
spiral on the survivors. *Owner:* CRD 9.3. *Damper:* none in version 1 (the first
consumer has no weight above 1); the waterfall at a removal covers the end of the
agent. *Undamped:* a cohort. *Test:* CRD-S4. *Status:* hand.

**CRD-A8. An eligibility ratchet.** *Kind:* hazard. A debtor with an overdue claim
is refused every request (5.2 step 7) until the claim is paid or written off, so a
household that cannot fund the next job's gap cannot borrow for it, and its
shortfall of food and inputs lowers its income, which keeps the claim overdue.
*Delay:* until money above the shield pays the arrears. *Owner:* CRD 5.2, 8.5.
*Damper:* the settlement each tick (arrears are paid as soon as money exceeds the
shield), `WriteOffTicks` (a written-off claim ends the bar: **and so lets the
lender lend again**, 19.2 case F), relief. *Undamped:* a debtor whose income stays
below its upkeep and shield. *Note from the model (19.2):* the shield's effect on
unfed ticks is not monotone and its best value moves with the horizon; a write-off
lifts the bar and costs the lenders. *Test:* CRD-S1. *Status:* model for the
ratchet (the bar and the shield's shape); hand for its feedback on income.

**CRD-A9. A posted rate does not clear.** *Kind:* bias. The rate of a lender is
data and moves with nothing in the engine: when offers fall short, the
requests with the highest limits are served and the rest are `Unfilled`, tick
after tick (quantity rationing). The loop that would close it (a rate that follows
unfilled requests) is the consumer's, through `SetRateScales` (13.1), and is L121 (OBS-A1, INT-A1: a
consumer closes a loop through the library's reads and slots), not a loop of this
library. *Owner:* CRD 6.3.3. *Test:* the counters of `CreditRound`.
*Status:* hand.

**CRD-A10. Relief paid and taken back** (a case of L39 and L63). *Kind:* hazard.
Relief (P1.6) arrives before the settlement (P1.7a) and is in the balance the
settlement reads (8.8): the pool pays a household for its upkeep and the pool or a
lender takes part of it back, up to the shield. With a creditor that is the estate
(a passed claim) the money returns to the pool. *Owner:* CRD 8.8, EST 2.6.
*Damper:* the shield (CRD-D1). *Test:* CRD-S2 with `PoolRelease`. *Status:* hand.

### 18.1 Dampers this spec adds or specifies

| ID | Damper | Kind | For | Status |
|----|--------|------|-----|--------|
| CRD-D1 | the shield: `ShieldTicks` of upkeep cost out of reach of repayment (8.2) | data, default 0 | A1, A10 | specified |
| CRD-D2 | no rollover: a due claim is settled before a request is admitted, and a debtor in arrears is refused (6.1.3, 5.2) | rule | A1, A8 | specified |
| CRD-D3 | the exposure cap `MaxDebtTicks` of the borrower role (5.3.3) | data | A1, A3 | specified |
| CRD-D4 | the lender's reserve `KeepTicks` and share `ShareBP` (10.3) | data | A3, A4, A6 | specified |
| CRD-D5 | the write-off `WriteOffTicks` (8.5) | data, default off | A8 | specified |
| CRD-D6 | pro rata settlement and interest first (8.3) | rule | the bias of an order | specified |
| CRD-D7 | the claims of a removed creditor pass to the pool (9.2) | rule | A3 | specified |
| CRD-D8 | `MaxClaims`: the cap on open claims (6.4 step 1) | data | state, not a loop | specified |

### 18.2 Delays near neighbours (checklist 9.6)

The term of a default loan is `Duration + 1 + Late + SlackTicks`: it meets the **job's
duration** (L19, L21, L22), the **welfare window** `T = max(WelfareWindow,
WindowCycles * longest Duration)` (`spec-welfare-and-wages.md` 2.3.3: the interest
paid at the due tick is in a window that already holds the job, so the loop's
delay is inside `T`), a **shelf life** (a perishable output sold in `SlackTicks`
ticks: `SlackTicks` must be at most the shelf life of the output, a condition
the warning `slack_above_shelf_life` would check, open question 12) and a
**season** (a seasonal job's term may straddle a season boundary: the loan is
priced when it is made).

### 18.3 Scenarios proposed

| ID | Scenario | Measures |
|----|----------|----------|
| CRD-S1 | a role with a 60-tick job (REF C04) and a `Borrow` block, a lender role, cold and warm start, with credit on and off | unfed household-ticks, relief paid, removals by the ladder, claims formed, interest, defaults, `Unfilled`, the pulse of `CreditRound` against the job cycle |
| CRD-S2 | the same with a yield shock that cuts a cycle's output and one that raises it, with `PoolRelease` on and off | defaults, `Arrears`, lenders' loss and unfed households, the price level, lender removals and claims passed |
| CRD-S3 | the invariants over a range of seeds with `WithAudit`: `M` unchanged by a loan, the pair identity, no empty claim, no claim outliving an account | exact equality (FR-CRD-082 to 086) |
| CRD-S4 | a cohort borrower whose weight falls by the ladder | debt per household, defaults, the waterfall at its removal |
| CRD-S5 | `ForgiveClaims` and `SetRateScales` between ticks, and a checkpoint taken with open claims and restored | the identity after the call, the events at the call, the restored settlement (FR-CRD-087) |

## 19. What was run

Two scripts in `doc/specs/models/crd/`, in Docker with `PYTHONDONTWRITEBYTECODE=1`
and a read-only mount (the README of that folder has the commands). They are models
of the **rules** of this spec with an assumed income, not of the library, and the
numbers are placeholders.

### 19.1 `alloc.py`

The interest rule with its half-to-even rounding, the pro rata allocation by the
largest remainder with the stateless key, interest first, the waterfall, and the
arithmetic of section 20; and two property runs: **20 000 random allocations**
(the shares sum to the payment, no share exceeds what is owed, a claim with a zero
fractional part gets no extra unit, and no claim's share moves when the others are
reordered) and **6 000 draws of an operation on a mini ledger** (`Lend`, `Settle`,
`WriteOff`, `Reassign`; 4 128 were applicable and applied: 1 215, 1 428, 586 and 899)
with the total of money and the pair identity of 3.6 asserted after every one. All
hold.

### 19.2 `loans.py`

Sixteen borrowers with a 60-tick job (upkeep 100 a tick, inputs 200, start money
1 500, a lump of 7 000 at the end of each job, `SlackTicks` 0), four lenders (cash
50 000, 600 basis points a year, keep 3 ticks, share 5 000), 480 ticks; a market
account pays every income so that the sum of money is constant (asserted at every
tick). The model applies the horizon of 10.2 with its structural tick: the lump is
visible at P1.7a of the tick after the job's last, so a claim is due one tick after
the completion tick (parameter `plus`; 0 is the horizon without it). It also lets
a loan feed the tick's own consumption, which the library does not (6.4.2), so it is
a little kind to the borrower. The result (`loans.out`):

| Case | Unfed borrower-ticks | Claims | Interest paid | Defaults | Refused in arrears | Lost | Lenders' unfed ticks |
|------|---------------------:|-------:|--------------:|---------:|-------------------:|-----:|---------------------:|
| A no credit | 321 | 0 | 0 | 0 | 0 | 0 | 0 |
| B credit, shield 1 | 0 | 49 | 781 | 0 | 0 | 0 | 0 |
| B0 as B with the horizon without the structural tick (`H = J`) | 47 | 40 | 652 | 40 | 0 | 0 | 0 |
| C one bad cycle (income 1 500), shield 1 | 723 | 97 | 2 023 | 3 | 724 | 0 | 0 |
| D the same, shield 0 | 727 | 96 | 2 021 | 3 | 724 | 0 | 0 |
| D2 the same, shield 3 | 238 | 92 | 1 826 | 4 | 240 | 0 | 0 |
| E income 1 500 from tick 180 on, no write-off | 3 403 | 55 | 1 249 | 16 | 3 419 | 0 | 36 |
| F the same, write-off after 120 ticks | 2 962 | 94 | 1 669 | 39 | 2 282 | 36 349 | 316 |
| G0 start money 150 (below the inputs), no credit | 7 664 | 0 | 0 | 0 | 0 | 0 | 0 |
| G1 the same, credit, `Tier2` as what tier 2 can afford after tier 1 | 0 | 164 | 4 305 | 0 | 0 | 0 | 0 |
| G2 the same, `Tier2` in full (10.2) | 0 | 148 | 4 321 | 0 | 0 | 0 | 0 |
| G3 as G1 with `MinPrincipal` 200 | 35 | 144 | 4 153 | 0 | 0 | 0 | 0 |
| G4 as G2 with `MinPrincipal` 200 | 6 | 143 | 4 312 | 0 | 0 | 0 | 0 |
| H0 start money 150, inputs bought at P3.2 as the library does, horizon `J + 1` (no `Late`) | 0 | 165 | 4 832 | 19 | 0 | 0 | 0 |
| H1 the same with `Late` (10.2.1.2) | 0 | 148 | 4 846 | 0 | 0 | 0 | 0 |

The shield, on the one bad cycle (the last block of `loans.out`), in unfed
borrower-ticks:

| `ShieldTicks` | 0 | 1 | 2 | 3 | 5 | 8 |
|---------------|--:|--:|--:|--:|--:|--:|
| `SlackTicks` 0 | 727 | 723 | 242 | 238 | 231 | 272 |
| `SlackTicks` 5 | 165 | 162 | 212 | 208 | 200 | 235 |

What it shows: **(a)** the default request closes the cash gap of the first job (A
against B: 321 unfed ticks to none, the same 128 jobs, interest 0.9% of the 85 845
lent); once the household has earned a lump it can finance its own job, so credit
matters where money is short. **(b)** The horizon needs its structural tick: B0
(`H = J`, no structural tick) has every claim due one tick before the cash arrives,
40 defaults and 47 unfed ticks where B has none. **(c)** One bad
cycle puts 3 claims into default, and the households that cannot pay are barred
from borrowing for their next job (724 refusals): they are unfed for 723 ticks
and the lenders' cash falls (33 676 against 47 181) because it is out in open
claims. **(d)** The shield has no monotone effect: at `SlackTicks` 0 the unfed ticks
fall from 727 to about 240 between shields of 1 and 2 and rise again at 8, and at
`SlackTicks` 5 the best shield is 1; **the best value moves with the horizon**
(CRD-A8). **(e)** A persistent loss of income gives 16 defaults and 3 403 unfed
borrower-ticks; the lenders (no other income, keep 3 ticks against a term of 65) eat
principal and go unfed for 36 ticks. **A write-off is not free**: after 120 ticks
(F) 36 349 of owed moves from overdue to lost, the bar on those households lifts,
the lenders lend to them again, and the lenders end with 144 against 14 110 and go
unfed for 316 ticks against 36 (CRD-A3, CRD-A8). **(f)** With start money below the
inputs a `Tier2` of what tier 2 can afford (G1, G3) under-asks; when the shortfall is above
`MinPrincipal` the next tick's request closes it (G1 and G2 are both fed), and when
it is below it nothing does (G3: 35 unfed ticks against 6 for the revised rule, G4);
the revised rule is the one of 10.2.
**(g)** With the library's delay of the inputs (they are bought at P3.2 of the tick after the loan is formed) a
household that starts below its inputs is a tick late; with the horizon `J + 1` (H0) 19 claims are
due one tick before the cash and go overdue, and with `Late` (H1, 10.2.1.2) none does. No borrower is unfed in
either: the loan covers the upkeep, so the cost of the missing tick is a default, and a bar on the next loan.

What it does not show: any price (the income is assumed), the pool, relief, the
ladder, beliefs, the lender's other income or a second lane. Cases A to G let a loan
feed the tick of the job's start (6.4.2 says it is spendable one tick later); cases
H0 and H1 apply the delay. A lender that has another income (a
producing role) is not in it. The finding (e) is the reason for the warning
`lender_keep_below_term` (15.2).

## 20. Worked examples

Both are **examples**, not packs; the numbers are placeholders. The figures of
20.1, 20.2 and 20.3 (the request, the interest, the pot table, the shares and the
waterfall) are printed by `alloc.py` (`alloc.out`).

### 20.1 Example A: seasonal credit for a long job (a farming realm; an example)

A grower household (weight 1; role with a `Borrow` block: `MaxRateBP` 1 500,
`MaxDebtTicks` 90, `SlackTicks` 4) is about to start a 60-tick job. `TicksPerYear`
365, upkeep cost `U` 100, `MinPrincipal` 100, shield 1 tick. Two lender households
(`Lend` blocks, `KeepTicks` 3, `ShareBP` 5 000): A with cash 20 000, upkeep 80,
`RateBP` 600, `MaxTermTicks` 90; B with cash 6 000, upkeep 80, `RateBP` 900.

| Step | Value |
|------|-------|
| The grower's gap (10.2) | `J = 60` (the recipe's `Duration`), `H = 60 + 1 + 4 = 65`; `need = 100 * 65 + 200 (inputs) - 1 000 (money) = 5 700` |
| Its room (5.3.3) | `90 * 100 - 0 = 9 000`; `ask = min(5 700, 9 000) = 5 700`, term 65, limit 1 500 |
| A's offer (10.3) | `lendable = 20 000 - 3 * 80 = 19 760`; `5 000 * 19 760 / 10 000 = 9 880`, rate 600 |
| B's offer | `lendable = 6 000 - 240 = 5 760`; `2 880`, rate 900 |
| The round (6.3) | offers in ascending rate: A, B. The request is filled by A for 5 700 (A has 9 880); B is not touched |
| The interest (7.1) | `halfEven(5 700 * 600 * 65 / (10 000 * 365)) = halfEven(60.904) = 61`; owed 5 761, due 65 ticks later |

At the due tick the grower holds the lump, so three cases:

| Money at P1.7a | Shield 100, pot | Paid | After |
|----------------|----------------|------|-------|
| 7 500 | 7 400 | interest 61, principal 5 700 (5 761) | the claim ends, `repaid`; the grower may ask again at P3.3b |
| 3 000 | 2 900 | interest 61, principal 2 839 (2 900) | owed 2 861 left, `Overdue` set; the same pass pays what money above 100 allows each later tick; **no new loan** until it is paid (`in_arrears`) |
| 80 | 0 | nothing | the whole is overdue; the grower keeps its 80 for its upkeep |

### 20.2 Example B: two claims, one debtor, and a removal

A debtor's pot at P1.7a (its balance less the shield) is 1 001 and it has two claims
due that owe 600 and 900 (1 500 together). The shares are
`floor(1 001 * 600 / 1 500) = 400` and `floor(1 001 * 900 / 1 500) = 600`, one unit
left, which goes to the claim with the larger fractional part (`600.6` against
`400.4`): **400 and 601**. With a pot of 1 000 the shares are exactly 400 and 600.

The same debtor is **removed** with a balance of 800 and claims owing 500
(interest 50, principal 450) and 700 (interest 50, principal 650) (9.1): `pay =
min(800, 1 200) = 800`; shares `floor(800 * 500 / 1 200) = 333` and
`floor(800 * 700 / 1 200) = 466`, one unit left to the larger fraction (`466.67`):
**333 and 467**. The claims are paid interest first (50 and 50), then principal
283 and 417; the unpaid 167 and 233 are lost; the estate receives nothing. With a
balance of 1 500 both are paid in full and the estate receives 300 under
`estate_money`.

### 20.3 Example C: a second setting (a mining colony; an example)

A prospecting crew (a role with `Borrow`: `MaxRateBP` 1 200, `MaxDebtTicks` 60,
`SlackTicks` 2) leaves for a 30-tick survey and sells its find at a gate at the end;
the colony's year is 400 ticks, `U` 50, inputs 120, money 300. `J = 30`, `Late` 0 (300 - 50 is
above 120), `H = 30 + 1 + 0 + 2 = 33`,
`need = 50 * 33 + 120 - 300 = 1 470`. A financier role's household with `RateBP` 900
lends all of it: interest `halfEven(1 470 * 900 * 33 / (10 000 * 400)) =
halfEven(10.915) = 11`, owed 1 481. A crew with money 80, below its inputs: `Tier2` is
the inputs in full, 120, and `Late` is 1 (80 - 50 is below 120: the inputs are bought
next tick, the survey starts a tick later), so `H = 34` and `need = 50 * 34 + 120 - 80 =
1 740`; sized by what tier 2 can afford after tier 1 (`max(0, 80 - 50) = 30`) the
request would be 1 650, and the crew would be 90 short for its survey. Nothing in the rule names a crew, a survey, a
gate or a financier.

## 21. Notes settled, and the edits this spec owes

### 21.1 Notes

Each note tagged `CRD` or `ALL`, and the others this spec relies on. The text to
append in `doc/notes.md` is in the handoff.

| Note | Decision | Where |
|------|----------|-------|
| N-032 Welfare window against recipe duration (STB, CRD) | **Applied in part; leave open for STB.** Credit is the remedy for the household's cash gap through a long job, and it does not change how the ratio reads the job (11.2); open until S11 has run with a borrower role | 10.2.3, 11.2 |
| N-046 Register entries owed (CRD) | **Applied for CRD; remove CRD from the areas.** CRD-A1 to A10, dampers CRD-D1 to D8, scenarios CRD-S1 to S5; still owed: SAV, RSK, FIS, PLC | 18 |
| N-107 New money faucets and sinks start from MON (FIS, CRD, MON, LED) | **Applied for CRD.** No faucet, no sink, no boundary account; three internal reasons, as N-107's second sentence allows for a mechanism that is not a grant or a levy; a creating lender is a MON grant with its own reason | 3.3, 3.4 |
| N-021 Anything that creates or destroys money or goods (FIS, LED) | **Applied for CRD.** The ledger's claims list is used as it said; no faucet and no sink | 3 |
| N-008 Weight changes, cohort failure and the estate | **Update line.** A reduction leaves every claim with the agent; the waterfall is at a removal only | 9.3 |
| N-016 Keep the parameter registry true (ALL) | **Applied.** Rows below; no unnamed parameter | 14 |
| N-017 Words and names (ALL) | **Applied.** Terms in 21.3 | 21.3 |
| N-027 No default macro constant is stable (ALL) | **Stands.** Every number is a placeholder; the models show structure, not stability | 14, 19 |
| N-103 Running several writers (ALL) | **Followed**; two slips (one in the writing, one in the revision, each a host `python3` call with nothing read, written or run) reported in the handoff | |
| N-126 A new event or `Warn` adds a row (ALL) | **Applied.** Six events and two `Warn`s have their rows in 16 and the handoff | 16 |
| N-128 A hosted per-agent field is saved once (ALL, CHK) | **Applied.** No per-agent field: a claim is in the ledger's list, saved once | 17 |
| N-090 What a relocation must do (PRO, SKL, INT) | **Applied.** This module registers no `region_bound` refusal, with the reasoning; the rule stands for RSK, CAP, ORG | 9.4.2 |
| N-122 First-come caps (SPN, MIG, MON, MER, STB) | **Respected.** Settlement is pro rata and the round orders by limit then the stateless key; no place is added to WLD 10.2 | 6.3, 8.3 |
| N-124 A paired comparison is exact only while aligned (EXP, STB, CLB, WLD) | **Stated.** The module takes no word, so none of its parameters is the flag `d` | 12.2 |
| N-054 The closed economy fixes the mean welfare ratio at 1 (WEL, DEM, STB, MON) | **Update line.** Interest redistributes and creates nothing: the identity holds in the aggregate; the ratios of lenders and borrowers differ | 11.1 |
| N-096 The regional pool serves births, entry, relief and money flows (MON, POP, EST, SPN) | **Update line.** Debt service to an estate creditor is a sixth thing that pays into the pool | 9.2.2 |
| N-091 The world slots INT must expose (INT) | **Update line.** A slot, `SetRateScales`, and a call, `ForgiveClaims` | 13 |
| N-131 Not in version 1 (INT, FIS, PLC) | **Respected.** Two rows only; 13.3 lists what is not offered | 13.3 |
| N-125 A snapshot is a function of saved state (CHK, LED, WLD, CLR, OBS) | **Applied.** Every field of 16.4 is a function of the claim list and totals | 16.4 |
| N-130 What IND cannot measure (OBS, IND, ...) | **Update line.** Interest paid and received are measurable from the events of 16.1; the income fields of item 1 are still an ask | 16.1 |

**New notes to add** (the supervisor numbers them; areas in brackets):

1. **Credit asks and offers are plan fields, and the default strategy appends them for a role with a block** [AGT, CRD, INT]. Raised 2026-10-12, `spec-credit.md` 5, 10, 21.2.
2. **A warm start has no claims; a pack that wants initial debt needs a configured table** [VIA, CFG, CRD]. Raised 2026-10-12, `spec-credit.md` 11.4, open question 14.
3. **Collateral is not in version 1; the candidates are goods in custody, a title of LND and a pledge of room of STO** [CRD, LND, STO]. Raised 2026-10-12, `spec-credit.md` 3.5.
4. **The goods of a removed debtor do not pay its creditors** [EST, CRD]. Raised 2026-10-12, `spec-credit.md` 9.1.2, open question 8.
5. **A lender with no other income eats its principal while it is out** [CRD, REF, STY]. Raised 2026-10-12, `spec-credit.md` 10.3.1, 19.2: a pack that has lenders gives the role an income or a keep as long as its term.

### 21.2 Edits owed to other files

One consolidated table; the exact text of each is in the handoff. None is made here.

| File | Section | Change |
|------|---------|--------|
| `spec-ledger.md` | 3.3, FR-LED-009 | an account is not closed while it is the creditor or debtor of an open claim (`ErrOpenClaim`) |
| `spec-ledger.md` | 7.2 | rows `loan_out`, `loan_repay`, `loan_interest` (money, internal), owner `spec-credit.md` |
| `spec-ledger.md` | 10, 11, 12.4, 13 | the claim record with its `Origin`, five operations (with `Obligate`) and seven totals of CRD 3.1 to 3.3; the list saved with them; the audit identities of CRD 3.6; FR-LED-028 amended, new FR-LED-062 to 064 |
| `spec-agents.md` | 3.6, 2 (row 5), FR-AGT-008 | steps 2d and 2e (CRD 9.1, 9.2) and their events; "no order beyond available cash, no borrowing, no negative balance" gets "(no borrowing with the credit module off)"; FR-AGT-008 reads "move all its money, less what its creditors take in the waterfall of `spec-credit.md` 9.1, to the region's estate with the reason `estate_money`" |
| `spec-agents.md` | 3.3, 6.3, 6.4.1, 6.5.2, 7, 10.3, 11.5 | the role field `Credit`; `TradePlan.Borrow`, `.Lend`; the view group; admission by the round; the default rules (a new 7.8); `Directive.Borrow`, `.Lend`; the wording of 11.5 |
| `spec-welfare-and-wages.md` | 2.1, 2.1.7, 4.3, 13, 2.3.3 | `I^in_t` (in `G_t`) and `I^out_t` in `income_t`; both kept out of the earned ratio and `I^in_t` out of the wage; no new record field; the closing wording (applied by WEL 2.1.9) |
| `spec-world.md` | 6.3, 6.5, 6.11, 6.1.11, 7.5, 8.2, 8.3 (the removal row lists `LoanEnded` and `LoanPassed` among the events of a removal), 9.2, 11.1 | rows P1.7a and P3.3b; the event order; the outside-node skip; the no-draw list; the slot and call rows; the cadence row; the cost rows |
| `spec-estates.md` | 2.1 (both the "never negative, since there is no credit" clause and the next), 2.7.4, 2.6, FR-EST-001 | creditors before the estate at a removal; the balance "never negative" without the words "since there is no credit"; the estate as a creditor by inheritance; relief and debt |
| `spec-observability.md` | 4.1, 4.3.3, 5.3.3, 5.3.8, 5.6.1, 6.2, 6.4.2, 7.2, 7.3, 7.6 | the six events (count and classes), three readings, the snapshot rows (and `Claims` as a fifth in-flight set, 5.6.1), nine metrics (the arrears gauge fed by the snapshot, 5.3.8; the seed of `outstanding` at `WorldRestored`, 6.4.2), the summary keys, two `Warn` rows, the new keys |
| `rules/observability-and-logging.md` | 3.3.2, 4.3, 5.3, 5.3.1, 5.4.1 | the keys (new ones only: none widens `amount`, `account` or another existing key), the events, the nine metrics (the arrears gauge among the instruments fed by a snapshot, now five), and the attribute values, with a fresh read-receipt tag |
| `spec-checkpoint.md` | 4.2, 4.2.4, 4.3, 9.3, 6.2 | tag 2's claims; a Credit section; the creditor kind; the derived indexes; the format version; the restore checks |
| `spec-integration.md` | 4.3.2, 4.7.2, 5.1, 5.1.3, 5.3, new 4.10 Claims | `Direct` checks; the `Credit` group of `AgentState`; the row `SetRateScales` (type `RegionScale`, reused); its errors; `SlotState.RateScales`; the call `ForgiveClaims` (loans only) |
| `spec-config.md` | 3.1, 3.2, 3.7, 4.1, 7.3, 7.4, 8.1, 9.2 | the module and the block; the bounds; the sentinel and the rules and warnings; the hash; the owner rows |
| `spec-money-supply.md` | 3.2, 9.5, 10.3 | the three internal reasons and the pointer that interest is a transfer |
| `spec-spawner.md` | 13.1.1 | the events that accompany a removal |
| `spec-regions.md` | 6.5.2 | "`CRD` registers no refusal" (9.4.2) |
| `spec-stability.md` | 4.8, 4.13 (new), 5, 8, 10 | the row CRD removed from the hooks; the entries; the dampers; the scenarios; the wording about credit in L19 |
| `spec-indicators.md` | 7 or a new 7.9 | the asks of the handoff |
| `spec-calibration.md` | 6.2 | the registry rows of the handoff |
| `spec-merchants.md`, `spec-employment.md`, `spec-skill-barriers.md`, `spec-land.md`, `spec-production.md` | 9.10 and 22 item 1; 7.3; 11.4 item 3; LND 2 (decisions-table row 10: "There is no credit in version 1"), LND 16 LND-A5 and LND 18.1 (a) ("CRD's credit"); PRO 16 (the bullet "Credit (`spec-credit.md`) is the real answer for a long job; version 1 has none") | one line each: credit is now `spec-credit.md`, not "version 1 has none" |
| `spec-reference-chain.md`, `doc/notes.md` (N-092) | the overlays; N-092 | the credit overlay `testdata/chains/overlay/credit.toml` of 24: a borrower role on the 60-tick job role (C04) and a lender role on one region, with its warm start and the scenarios CRD-S1 to S5 |
| `doc/notes.md` (N-019), `architecture.md` 2.6, `AGENTS.md` | the housekeeping note; the layout | `credit.go` and `internal/credit/` are homes to add at the first code that needs them, with the other homes of N-019 |
| `README.md` (specs), `roadmap.md`, `documentation_index.md`, `notes.md`, `doc.go` | | the code row, the owner rows, the glossary, the roadmap row, the index row, the notes of 21.1, the terms of 21.3 |

### 21.3 Terms (N-017)

**credit** (the module), **loan** (a claim formed by the round), **request** (for
credit) and **offer** (of credit), both one tick long, **lender** and **borrower**
(roles with a block), **posted rate**, **effective rate** (posted times the rate
scale), **rate scale**, **term**, **principal**, **interest** (fixed at formation),
**owed**, **due tick**, **overdue**, **arrears**, **shield**, **settlement** (the
pass of P1.7a), **waterfall** (the pro rata payment at a removal), **passed**
(claim), **written off**, **forgiven**, **exposure cap**, **room**, **cash gap** and
its **horizon**, **round** (of credit; a **lane** of it is a region, the word of MER
and EMP in the qualified spelling "lane of the credit round"). The words
**moneylender**, **mortgage**, **guild** and **bank** are setting words: they appear
only in the flagged examples, in the header paragraph and in open question 1, as
words, and in no name of the core text or the API. `spec-storage.md` 4.1.3 uses "credit" for the posting of goods to an
account; that is a ledger operation and not this module, and a spec that means it says
"a credit of goods".

## 22. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Each is
derived from the section named. The IDs are grouped by area, in blocks of ten:
000s the module and the ledger, 010s roles and admission, 020s the round and interest,
030s settlement, 040s removal and the other changes, 050s the strategy and welfare, 060s
the slot, the call and the stream, 070s observability, 080s invariants and state. A
gap is a number not yet used, not a missing requirement.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CRD-001 | The module MUST run if and only if `Config.Modules.Credit` is non-nil, and a world with it nil MUST have no position P1.7a or P3.3b, no claim, none of the three reasons, no draw for it and none of its events, and MUST be bit-identical to a world built without this spec (4.1, 12). |
| FR-CRD-002 | The reasons `loan_out` (Agent account to Agent account), `loan_repay` and `loan_interest` (Agent account to Agent or Estate account) MUST be internal, money only, and every other pairing MUST be refused with `ErrReasonMismatch` (3.3). |
| FR-CRD-003 | No operation of the module MUST create or destroy money or goods: the total of money over non-boundary accounts MUST be unchanged by `Lend`, `Settle`, `WriteOff` and `Reassign`, and the module MUST add no faucet, no sink and no boundary account (3.4). |
| FR-CRD-004 | `Lend` MUST transfer the principal with `loan_out` and append the claim all or nothing, and MUST be refused with `ErrInsufficientFunds`, `ErrBalanceOverflow`, `ErrReasonMismatch`, `ErrUnknownAccount`, `ErrClosedAccount`, `ErrInvalidAmount` or `ErrClaimShape` leaving the ledger unchanged (3.2). |
| FR-CRD-005 | A claim with both `PrincipalOwed` and `InterestOwed` equal to 0 MUST NOT exist: it MUST be removed from the list in the operation that brought it to 0 (3.1.2). |
| FR-CRD-006 | A world with the module on and no other faucet or sink MUST have `M` equal to `M_init` at every tick boundary, for every seed (3.4; `spec-money-supply.md` 8.2). |
| FR-CRD-007 | An account that is the creditor or the debtor of an open claim MUST NOT be closed (`ErrOpenClaim`) (3.2.1). |
| FR-CRD-008 | Claim IDs MUST be assigned in order of formation, MUST NOT be reused in a run, and the list MUST be in ascending ID (3.1.1). |
| FR-CRD-009 | The `Principal`, `Interest`, `RateBP`, `Formed` and `Due` of a claim MUST NOT change after formation, and `PrincipalOwed` and `InterestOwed` MUST only fall (7.1.3, 7.2). |
| FR-CRD-010 | A role with a credit block MUST have exactly one of `Lend` and `Borrow`, and both or neither MUST be `ErrCreditShape` (4.2.1, 15.1). |
| FR-CRD-011 | A credit block on a role while `Modules.Credit` is nil MUST be `ErrModuleDependency`, a field outside its bound MUST be `ErrOutOfBounds`, and a `rate_bp` or `max_rate_bp` absent from a file MUST be `ErrMissingField` (14, 15.1). |
| FR-CRD-012 | The configuration of the module and of every block MUST be in the configuration hash, and the rate scales MUST NOT (14, 13.1, 17.5). |
| FR-CRD-013 | A plan MUST carry at most one admitted request and one admitted offer per agent per tick, and a second MUST be refused `duplicate` (5.2). |
| FR-CRD-014 | The admission checks of 5.2 MUST be made in the order of its table, each refusing whole with its `reason`, and a refusal MUST change nothing and MUST NOT be an error of `Step` (5.2). |
| FR-CRD-015 | A request whose `Principal` plus `Debt` exceeds `MaxDebtTicks` times the agent's upkeep cost MUST be refused `exposure` and MUST NOT be cut to the room (5.2.2, 5.3). |
| FR-CRD-016 | A request of an agent with an overdue claim as debtor MUST be refused `in_arrears` (5.2). |
| FR-CRD-017 | An offer above the agent's available money MUST be refused `insufficient_funds`, and a request from a role with no `Borrow` block (an offer, no `Lend` block) MUST be refused `not_borrower` (`not_lender`) (5.2). |
| FR-CRD-018 | With `Modules.Credit` absent a request or an offer MUST be refused `credit_off` and cost nothing else (5.1.1). |
| FR-CRD-020 | The round MUST run once per tick at P3.3b, after P3.3a and before P3.4, only with the module present, and MUST move only available money (6.1). |
| FR-CRD-021 | A request MUST be matched only with offers whose agent's account is in the same region at the round, and the lanes MUST be visited in ascending `RegionID` (6.2). |
| FR-CRD-022 | In a lane the offers MUST be ordered by effective rate ascending then `K(t, agent)` then agent ID, and the requests by `MaxRateBP` descending then `K(t, agent)` then agent ID, and the round MUST take no word of the world's stream (6.3.1, 12.2). |
| FR-CRD-023 | A fill MUST be at the lender's effective rate, MUST be skipped for an offer whose rate exceeds the request's `MaxRateBP` or whose `MaxTerm` is below its `Term`, and MUST end the scan of a request at the first offer above its limit (6.3.2). |
| FR-CRD-024 | A fill below `MinPrincipal` MUST NOT be formed, and the effective rate MUST be `halfEven(RateBP * ScaleBP / 10 000)` clamped to `MaxCreditRateBP` (6.3.1, 13.1). |
| FR-CRD-025 | Formation MUST check, in the order of 6.4, `too_many_claims` and `overflow`, MUST then lend, and MUST drop a failed fill with its reason leaving the ledger unchanged. |
| FR-CRD-026 | A claim MUST be formed with `Due` equal to the tick of formation plus the request's `Term`, and the borrower's money MUST be unchanged by the round except by the loans (6.4). |
| FR-CRD-027 | The interest of a loan MUST be `halfEven(principal * rate * term / (10 000 * TicksPerYear))` computed with a 128-bit intermediate, and a loan whose principal and interest exceed `MaxMoney` MUST be refused `overflow` (7.1). |
| FR-CRD-028 | In the admission of the round, after the last check of 5.2, a request's `MaxRateBP` above the rate cap in force for the borrower's region MUST be replaced by the cap, MUST NOT be refused for it, and the replacement MUST be counted in `PriceControlled` (`spec-institutions.md` 6.6.1; 5.2.3). |
| FR-CRD-030 | The settlement MUST run once per tick at P1.7a, after P1.7 and before P1.8, over the debtors with a claim due or overdue in ascending agent ID (8.1). |
| FR-CRD-031 | A debtor's payment MUST be `min(max(0, balance - ShieldTicks * U), owed)` over its due and overdue claims, with `balance` the one recorded at the entry of P1.7a for every debtor, so that no payment of the pass changes another debtor's pot (8.1.1.1, 8.2). |
| FR-CRD-032 | A payment MUST be shared among the claims pro rata by what each owes with the largest remainder (ties by `K(t, claim ID)` then claim ID), MUST pay interest before principal within a claim, and no share MUST exceed what its claim owes (8.3). |
| FR-CRD-033 | A payment MUST be cut to the creditor's headroom below `MaxMoney`, the part not paid MUST remain owed, and the cut MUST be counted in `CreditSettled.Headroom` (8.4). |
| FR-CRD-034 | A claim with owed left at its due tick MUST be set overdue at that tick, and every overdue claim MUST be paid by the same pass on each later tick (8.5). |
| FR-CRD-035 | With `WriteOffTicks` above 0 an overdue claim MUST be written off at the tick `Due + WriteOffTicks` with the cause `written_off`, and with 0 it MUST never be (8.5). |
| FR-CRD-036 | Arrears MUST bear no interest, no penalty and no re-scheduling, and the owed of a claim MUST NOT rise after formation (8.7, 7.2). |
| FR-CRD-037 | A due claim MUST be settled before any request of the same tick is admitted, and no loan MUST be formed that is used to pay a claim of the same tick (6.1.3). |
| FR-CRD-040 | At the removal of an agent its open claims as debtor MUST be paid from its whole balance with no shield, pro rata and interest first, before its money moves to the estate, and what is unpaid MUST be written off with the cause `debtor_removed` (9.1). |
| FR-CRD-041 | The estate MUST receive from a removed debtor only the balance left after the waterfall (9.1.1, 9.1.2). |
| FR-CRD-042 | At the removal of an agent each open claim of which it is the creditor MUST be made a claim of the Estate account of the region of its account, in ascending claim ID, with no money moved, and `LoanPassed` MUST be emitted (9.2). |
| FR-CRD-043 | An agent that is both a debtor and a creditor MUST go through the waterfall and then pass its claims, and the waterfall MUST run for every cause of a removal and for `RemoveAgent` and MUST NOT run for a reduction of weight (9.2.3, 9.1.3). |
| FR-CRD-044 | A change of weight, a change of role and a relocation, a trip's arrival and a journey's arrival MUST NOT change any claim and MUST NOT be refused because of one, and the module MUST register no `region_bound` refusal (9.3, 9.4). |
| FR-CRD-045 | A debtor that is away on a trip or on a journey MUST be settled at P1.7a like any other and MUST be given no view, and so MUST post no request (9.4.3). |
| FR-CRD-050 | The `Credit` group MUST be present in the view of an agent whose role has a block, and only there, with the fields of 10.1, and MUST carry no other agent's claim and no total of a lane (10.1). |
| FR-CRD-051 | The default strategy of a role with a `Borrow` block MUST append one request of `min(need, CreditRoom)` with the term `H` when that is at least `MinPrincipal`, `need` and `H` as in 10.2 (`H = J + 1 + Late + SlackTicks`, `Late` as in 10.2.1.2, and `Tier2` the cost of the next job's inputs in full whatever money allows), and none otherwise (10.2). |
| FR-CRD-052 | The default strategy of a role with a `Lend` block MUST append one offer of `ShareBP` of `max(0, Money - Bids - KeepTicks * UpkeepCost)` when that is at least `MinPrincipal`, at its block's rate and term, and none otherwise (10.3). |
| FR-CRD-053 | A role with no block, or a world with the module off, MUST give the default strategy a plan with no request and no offer (10.2, 10.3). |
| FR-CRD-054 | The principal of a loan MUST NOT be income and not a cost in the welfare ratio, interest received MUST be unearned income (`I^in_t`, in `G_t`) and excluded from the observed wage, and interest paid MUST be netted from `income_t` (`I^out_t`) (11.1). |
| FR-CRD-055 | A directive's `Borrow` and `Lend` MUST replace the fallback's request and offer, MUST be validated at `Direct` (`ErrOutOfBounds`, `ErrNotOnRole`), and MUST be refused with `ErrModuleOff` while the module is absent (10.5). |
| FR-CRD-060 | `SetRateScales` MUST apply from the next P3.3b to new loans only, with the errors in the order of 13.1, emit one `CreditRateScaled` per changed region at the call, and save the non-default scales (13.1). |
| FR-CRD-061 | `ForgiveClaims` MUST end at the call, in ascending claim ID, every open claim whose debtor's region and role match a filter, with the cause `forgiven`, move no money, emit one `LoanEnded` per claim at the call and return the count (13.2). |
| FR-CRD-062 | The module MUST take no word of the world's stream, and the number of words of a tick MUST NOT depend on the module being on, on any of its parameters or on a claim (12.2). |
| FR-CRD-070 | `LoanFormed` MUST be emitted per claim in formation order and `CreditRound` per region ascending after it, with the fields of 16.1 and no map (16.1). |
| FR-CRD-071 | `LoanEnded` MUST be emitted per claim ended, in ascending claim ID, at P1.7a, at a removal before `AgentBankrupted`, and at the call of `ForgiveClaims`, with the cause and the paid and lost parts (16.1). |
| FR-CRD-072 | `LoanPassed` MUST be emitted per claim passed, in ascending claim ID, after the `LoanEnded` events of the same removal and before `AgentBankrupted` (16.1.2). |
| FR-CRD-073 | `CreditSettled` MUST be emitted per region ascending after the `LoanEnded` events of P1.7a, only when a count or a sum is non-zero, with the debtors' and the creditors' fields (16.1.1). |
| FR-CRD-074 | The silent dampers MUST be counted: `InArrears`, `Exposure`, `TooMany`, `Unfilled` and `Overflow` in `CreditRound`, `Shielded` and `Headroom` in `CreditSettled` (16.1). |
| FR-CRD-075 | The events of the module MUST NOT be constructed when no observer was supplied, and a run with and without an observer and a logger MUST be identical (16.5). |
| FR-CRD-076 | The two `Warn`s of 16.2 MUST be emitted by their patterns and bounded by regions and by one, and each MUST have its counterpart (16.2). |
| FR-CRD-077 | The metrics of 16.3 MUST be derivable from the events and the snapshot fields of 16.4 (the arrears gauge from the `Claims` rows, `economysim.credit.outstanding` seeded at `WorldRestored` from the `Claims` rows of origin `loan`; every instrument counting origin `loan` only), a restored world's adapter MUST read the same values as one that saw the whole run, and the attributes MUST be bounded by the configuration or closed sets (16.3, 16.3.1). |
| FR-CRD-078 | Every field of 16.4 MUST be a function of the saved state, so that a restored world's snapshot equals the original's (16.4). |
| FR-CRD-079 | An observer MUST be able to rebuild the money of the households and of the pool of each region through a loan, a payment and a waterfall from the events of 16.1 alone (16.1.3). |
| FR-CRD-082 | With `WithAudit` the audit MUST check the identities of 3.6 and report the first that fails naming the claim, and an audit MUST NOT change any state (3.6). |
| FR-CRD-083 | Over the reference chain with a borrower role and a lender role and over a range of seeds, the identities of 3.6 and the conservation of money MUST hold after every tick with exact equality (CRD-S3). |
| FR-CRD-084 | A claim MUST NOT outlive an account: no open claim names a closed account, and the creditor is an Agent or an Estate account, the debtor an Agent account, and they differ (3.6). |
| FR-CRD-085 | A claim's owed parts MUST stay within `[0, Principal]` and `[0, Interest]` (3.6). |
| FR-CRD-086 | The sum of `PrincipalOwed` over open claims MUST equal `Lent` plus `Awarded` less `Repaid` less `LostPrincipal`, and the interest identity MUST hold (3.6). |
| FR-CRD-087 | A checkpoint taken with open claims, restored, MUST settle the same claims at the same ticks, form the same loans and give the same events and snapshot as an uninterrupted run, and a claim list that breaks 17.5 MUST be refused with `ErrCheckpointCorrupt` (17). |
| FR-CRD-088 | The checkpoint MUST save the claims once, in the ledger's section, with the next ID and the seven totals, and the module's section MUST hold only the non-default scales and the diagnostic runs (17.1). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-CRD-001 | With the default logger and observer the settlement, the round and the waterfall in a step of the reference chain MUST allocate nothing beyond scratch held by the world (`observability-and-logging.md` 4.5, 8.2). |
| NFR-CRD-002 | The cost of the round SHOULD be `O(requests + offers)` per lane for the sort and `O(fills)` for the matching, with no term in households (6.5, 12.4). |
| NFR-CRD-003 | The cost of the settlement SHOULD be proportional to the claims due, overdue and their debtors, and not to the open claims (8.1.2, 12.4). |
| NFR-CRD-004 | The interest, the allocation, the waterfall and the round SHOULD have benchmarks with `b.ReportAllocs()` (testing.md 8). |
| NFR-CRD-005 | The interest, the allocation and the waterfall MUST have fuzz tests (testing.md 7.4) asserting that they never panic for any amounts, rates, terms and claims, that the shares sum to the payment and none exceeds what is owed, that the interest is within one unit of the exact quotient, and that overflow is reported and not wrapped. |

## 23. Open questions

Each with a next step.

1. **A lender that creates money.** A setting with a bank-like issuer needs a loan
   that is a grant and a repayment that is a sink, and a rule for how much (3.4,
   mechanism B). *Next step:* when a consumer asks, a MON flow with its own reason
   and a `Limit` covers the money; a reserve rule needs state that does not exist.
2. **A rate that responds.** The posted rate moves with nothing (CRD-A9). A bounded
   step with the unfilled requests of the lane, in the form of LND's rent step, is
   the obvious module; it would be a new loop and a new position. *Next step:* run
   CRD-S1 and see whether rationing by limit matters.
3. **Collateral** (3.5). *Next step:* the title of LND is the cheapest (a call
   `TransferTitles` at default and a field `Security` of the claim); goods in custody
   need an account kind with ageing; a pledge of room (STO) is a number that already
   exists.
4. **Credit across regions** (6.2.1). *Next step:* with INF, a rumour of rates and a
   lender that reads it, and a distance that scales the rate; or a merchant's
   advance, which MER 9.10 lists.
5. **Early repayment** (6.4.1, 8.7). An idle debtor holds cash while it owes
   interest; a rule that prepays above a reserve is a decision. *Next step:* the
   lowest-cost form is an optional field of the request (`Prepay bool`); a consumer
   call for the player is the other.
6. **Credit in kind, and pay in kind** (EMP 22 item 1). *Next step:* none; goods
   advanced are a delivery contract.
7. **A claim that is traded.** A lender that needs cash sells its claim at a
   discount to another household; the engine would need a price of a claim and a
   market for it. *Next step:* SAV, with a call `TransferClaims` in the shape of
   LND's `TransferTitles`.
8. **The goods of a removed debtor** (9.1.2). The estate lists them at the reference
   price and the proceeds are the pool's; a second tier of the waterfall could pay
   creditors from the proceeds with a delay. *Next step:* the scenario CRD-S2 shows
   how much of a loss is recoverable.
9. **A cohort's debt** (9.3.1). *Next step:* a claim split in proportion to the
   weights when a reduction would leave debt above the room, if a consumer scales a
   borrower role.
10. **Debt and migration** (11.4). A household in debt that is pushed to move
    carries its claims; whether MIG's estimate should read the debt is MIG's
    (N-110: a household's own debt is its own state). *Next step:* when MIG is
    revised.
11. **What is not offered** (13.3): a story loan between two named agents, early
    repayment by call, a change of terms, a freeze that is not a scale. *Next step:*
    when a consumer asks (N-131).
12. **`SlackTicks` and shelf life.** A borrower whose output spoils inside the time
    to sell has a horizon too short. A warning `slack_above_shelf_life` would read
    the recipe's output commodity; not written because it needs the planned recipe's
    output, which is a choice among recipes (AGT 7.2).
13. **A profit test.** The default borrower asks for the gap and does not compare the
    interest with the margin of the job. A rational borrower would not ask for a loan
    whose interest exceeds the job's expected net value; the rule is a decision with
    inputs the view carries (the beliefs and the recipe's net value, AGT 7.1.4).
    *Next step:* a clause of 10.2 if CRD-S1 shows loans that lose.
14. **Initial claims.** A warm start has none (11.4). A pack that wants a realm that
    starts in debt needs a configured table of claims, with the creditor and debtor
    named by role and region, in the hash. *Next step:* when a pack asks.
15. **A lender's other income.** 10.3.1 leaves the lender's survival to the pack.
    *Next step:* the warnings of 15.2; a rule that keeps a lender's cash above its
    own horizon would be the lender's `Work`, outside this spec.
16. **An instalment claim.** A schedule of payments for a project financed by credit
    (`spec-saving-and-investment.md` 6.2.2); the claim of 3.1 falls due whole. *Next step:*
    when a consumer asks.

## 24. Where this lives in the code

| Concern | Home |
|---------|------|
| `Credit`, `CreditTerms`, `LendTerms`, `BorrowTerms`, `CreditRequest`, `CreditOffer`, `ClaimFilter`, the refusal reasons and causes, `SetRateScales`, `ForgiveClaims`, the call sites of the round and the settlement, the view group, the default rules | root package, `credit.go` (and the view and plan types in `strategy.go` of `spec-agents.md` 6) |
| `Claim`, `Lend`, `Settle`, `WriteOff`, `Reassign`, `Obligate`, `ErrOpenClaim`, `ErrTooManyClaims`, the totals, the audit of 3.6 | `internal/ledger` (stdlib only; architecture.md 2.9) |
| The interest, the allocation by the largest remainder, the order and matching of the round and the waterfall, over primitive slices | `internal/credit` (new; stdlib only; it does not import the root, `internal/ledger` or `internal/ordering`: the stateless key is passed in) |
| The stateless key `K(t, id)` | `internal/ordering` (shared with RES, `spec-world.md` 10.1.1) |
| Log key strings | `internal/logkeys` |
| Tests | `credit_test.go` (external package; the requirement tests of 22), `internal/credit` unit and fuzz tests (`FuzzInterest`, `FuzzAllocate`, `FuzzWaterfall`, `FuzzRound`), a ledger test for the identities, `testdata/chains/overlay/credit.toml` (a REF overlay: a borrower role and a lender role on `one`), and an example under `examples/` that shows a player borrowing |

The terms of 21.3 go to the glossary in `doc.go` before code uses them (N-017).
`doc.go` does not exist yet.
