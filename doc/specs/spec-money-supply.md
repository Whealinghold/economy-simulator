# Spec: Money supply: faucets, sinks and the closed economy

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-MON-NNN` and
`NFR-MON-NNN`, which tests cite (testing.md section 4). They are in section 20.

Layer: core for the statements about money (the identities of section 3, the closed
economy of section 8, the rule that a counterparty whose balance is not conserved is a
boundary, section 10) and for the primitives; module for the configured flows, which
are off unless `Config.Modules.MoneySupply` is present
([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md) decisions 1 to 4;
`spec-config.md` 3.2). Version 1 (`vision.md` section 4). Needs `spec-ledger.md` (the
reason registry 7, boundary accounts 3, the sectoral identity 9.3), `spec-currency.md`
(the minor unit, the checked helpers), `spec-agents.md` (the tick phases 4, weight,
external agents), `spec-estates.md` (the pool), `spec-calendar.md` (`TicksPerYear`) and
`spec-config.md` (the module switch, bounds, validation, the hash).

**Every number in this spec is a placeholder.** Nothing has been run in the library,
bazaarBot has not been run, and no acceptance band exists (notes N-015, N-027). No flow
exists by default, so the spec sets no default macro constant. The model of section 18
is a small integer model of the rules of this spec in Docker Python, not a claim about
the library, and no default is claimed to be stable.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

Money is not a good (`spec-goods.md` 2.4, 7.4): it is a balance, an `int64` count of
the minor unit, held in accounts and moved only by a transfer that carries a reason
(`spec-ledger.md` 2.1). The total of money over the non-boundary accounts therefore
changes in exactly two ways: a **faucet**, a transfer from a boundary, and a **sink**, a
transfer to one (`spec-ledger.md` 2.2, 7.3). The ledger enforces that every one has a
named reason, but it creates none. The one money faucet written so far is `initial`
(construction, and an external agent added during a run); the roadmap says "a closed
economy deflates" and DEM needs a recurring income for a class that produces nothing
(N-078). This spec decides how money enters and leaves a world, and says what a closed
economy is. It covers:

- the identities that define the **money supply** and what is measured (section 3);
- the **money flow**, the one configured, data-driven mechanism: its kinds (a grant is a
  faucet, a levy is a sink), its shapes, its targets, its schedule, its limit and its
  declared reason (section 4);
- the per-tick step: where it runs, in what order, how each shape is computed, what
  happens when a recipient cannot pay or cannot receive (sections 5, 6);
- what a grant is to its recipient: weight, income, the pool (section 7);
- the closed economy and what "deflation" does and does not mean here (section 8);
- faucets, sinks and balanced pairs, with the numbers that were run (section 9);
- the primitives reserved for the external sector, for fiscal specs and for a policy
  actor, and how a policy plugs in without this spec containing one (section 10);
- parameters, validation, observability, checkpoint state, the warm start, the entries
  for the stability register, worked examples, the notes settled, requirements and
  open questions (sections 11 to 22).

Out of scope: any rule that reacts to an aggregate (a policy actor, `spec-policy.md`);
the outside world's trade flows, prices and volumes (`spec-external-sector.md`); taxes,
tolls and a treasury (`spec-fiscal.md`); credit, interest and the claims of a loan
(`spec-credit.md`); saving and investment decisions (`spec-saving-and-investment.md`);
births, deaths and migration (`spec-population.md`, `spec-migration.md`); indicators such
as velocity, a price index or inequality (`spec-indicators.md`); the tick phases
(`spec-world.md`); the public surface that sets a flow's scale (`spec-integration.md`).

## 2. Positions taken

The design pressures of the brief, each with a position and the reason. Later sections
give the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What creates or destroys money? | Only a transfer with a reason of the ledger registry. In version 1 that is `initial` (construction and external agents) and the **money flows** this spec adds. A world with no `Modules.MoneySupply` has no other faucet or sink: it is the closed economy | `spec-ledger.md` 2.1, 7.3; ADR 0001 decisions 1 and 2 |
| 2 | How does money enter, concretely? | A **grant**: a recurring transfer from a boundary to a **class**, the households of a role (in one region or all) or a region's pool. One mechanism covers a stipend (N-078), an issuance into a region's pool, a bounty and a one-off endowment at a stated tick | One primitive, data-driven, with no code that names a role or a commodity (architecture.md 9.1) |
| 3 | Extraction and issuance (the roadmap's "mints, mines")? | Not as a recipe output and not as a conversion of a commodity to money. An extracted commodity is an ordinary commodity at an ordinary price; money from it is a grant, or, when EXT exists, a fixed-price buyer in the outside world. Compared in 3.4 | A money-yielding recipe would be a configured wage for the extracting role (`spec-currency.md` 2.6) and would change PRO's output type and WEL's income; a conversion at a fixed rate is a standing bid from a boundary, which the market cannot admit (a boundary cannot reserve, `spec-ledger.md` 6.1) and which is EXT's |
| 4 | How does money leave? | A **levy**: a recurring transfer from a class to a boundary, either a fixed amount per household or a rate on the balance above an exemption (a holding charge). Both are capped by what the recipient holds | The rate form gives a bound that can be derived (9.3); the fixed form is the paired sink DEM asked for (N-078) |
| 5 | Does any rule read an aggregate? | No. A flow reads the tick, its own schedule, its own totals, the **weight** of the recipient and, for a rate levy and a top-up, the recipient's own balance. It never reads the money supply, a price, an index or a count of agents of a class. A policy that wants to react to measures plugs in through the **flow scale** slot (10.4), as the yield-shock slot does for production | "Macro is measured, never set" (roadmap; ADR 0001 decision 7; `spec-stability.md` 5.1.3) |
| 6 | Which shapes are allowed, and which are excluded? | Allowed: a grant of a rate (an amount per tick or per year per household or per pool), a grant that tops a balance up to a level, a levy of a fixed amount, a levy of a rate on the balance above an exemption. Excluded: a faucet proportional to any balance or to the money supply (interest, a growth rule), because it is an unconditional positive loop (9.5). Sinks may be proportional to the balance they drain | The asymmetry is principled: a sink proportional to the balance is a balancing loop (A3), a faucet proportional to it is a reinforcing one (the exponential of 9.5) |
| 7 | Rate units | Integers: minor units per household (or per pool) per **tick** or per **year**, and basis points per tick or per year. A year is `TicksPerYear`, derived from the calendar. Amounts are spread over the ticks with a carried remainder, so the long-run total is exact and nothing is paid in lumps | Calendar-aware by data (`spec-calendar.md` 3.2); lumps would synchronise every recipient (STB-L22), and a consumer who wants a pulse has the shock slot |
| 8 | What names a flow in reports? | Each flow declares one ledger **reason**, equal to its ID, with the kind faucet or sink. The registry is closed at construction: the built-in rows plus one row per flow. `BoundaryFlow` and the boundary balance then total each flow by itself | `spec-ledger.md` 7.2 ("a reason is declared once, with its kind"); a pack should not have to edit the library to name a faucet |
| 9 | How are a faucet and its sink paired? | Not automatically. A pack declares both; validation compares the nominal yearly totals of the fixed flows over the census and warns when they differ (`pair_unbalanced`); a rate levy gives a bound whatever the balance of the pair (9.3); the **boundary balance per reason** is the measure. Money that is created is visible, not hidden | A mirror rule (a levy that retires what a grant issued) was considered and rejected: it adds a coupling and an allocation rule, and it hides the drift that the measure should show (open question 6) |
| 10 | Is a grant income? | By default yes: an unearned income of the recipient, like relief (WEL 2.1), excluded from the observed wage. A grant with `NotIncome` is not. A levy is not netted from income in version 1 | The target-income rule should not tell a household that lives on a stipend to work (and a household that is taxed is not richer for it); WEL 2.1 states the one rule per flow (7.2, open question 5) |
| 11 | The closed economy "deflates" | Stated exactly in 8: the supply is constant, money per household moves only with the households and the sectors, and a falling price level follows only if real volume per unit of money grows. The library asserts the identities and measures; it makes no claim that prices fall. A liquidity floor is derived (8.5) | A claim about prices would be a result, and no result exists (N-027) |
| 12 | Hoarding | Measured (sector shares of money, 3.1) and damped by the pool's release (D12), by the holding charge and by a cumulative **limit** on a grant. No rule reads the measure | `spec-stability.md` F2, L31, L45 |
| 13 | Randomness | None. Remainders come from a carry started from the account ID. The creation-draw list of `spec-agents.md` 3.5.2 does not grow | Determinism (architecture.md 7.3) |
| 14 | What does EXT reuse? | The declared reason, the class selector, the schedule, the top-up shape, the carry arithmetic and the rule that a counterparty whose balance is not conserved is a **Boundary** account (10.1). EXT specifies the outside world; MON does not | The brief; keeps EXT thin |

## 3. Money and where it can change

### 3.1 The money supply and its identity

3.1.1. The **money supply** `M` is the total of money over the non-boundary accounts:
the households (Agent accounts), the public sector (Estate accounts, that is the pools),
the contract sector (Escrow, `spec-ledger.md` 9.3) and the treasuries of the governed
jurisdictions (Treasury accounts, `spec-polity.md` 5; none with no governed jurisdiction).
Transit accounts hold goods only. Write `M_h`, `M_p`, `M_c` and `M_t` for the four. The Outside accounts of the external module are a further sector that is zero at every tick boundary and is not part of `M` (`spec-external-sector.md` 4.2.3).
 `M` is a measure: nothing in the engine reads
it (position 5).

3.1.2. **The identity.** By the zero total of the ledger (`spec-ledger.md` 9.1 item 1),

```
M  =  - (sum over reasons of the money boundary balances)
   =  M_init + (sum of grants applied) - (sum of levies applied) + (other faucets) - (other sinks)
```

where `M_init` is the total created by `initial` and the other faucets and sinks are
those of later specs. Reading the boundary balances is how a report finds every source
and drain (`spec-ledger.md` 3.4). Nothing in this spec adds a way to change `M` outside
this identity (FR-MON-002, FR-MON-026).

3.1.3. **What is measured, and by whom.** MON defines these measures from ledger data
and the households' weights; the indicators (velocity, a price index, inequality, money
per household by role) are `spec-indicators.md`'s and are computed from them:

| Measure | Definition |
|---------|------------|
| `M`, `M_h`, `M_p`, `M_c`, `M_t` | as 3.1.1, world and per region (`M_t` of a region is the treasuries seated in it, `spec-polity.md` 5.1.1) |
| `H` | the households: the sum of agent weights, pseudo-agents excluded (`spec-scale.md` FR-SCL-015) |
| `M_h / H` | money per household, world and per region (`Sectors` of `spec-indicators.md` 9 row 7, with the velocity `V` as `Turnover`, row 8); per role it is reachable only through the household rows of `Money` (`spec-indicators.md` 9 row 7), not through `Sectors` |
| Boundary balance per reason | the cumulative net created (negative) or destroyed (positive) under each money reason, `initial` included |
| Issued and collected per flow | the cumulative `Total` of each flow (4.4) and the per-tick amounts in `BoundaryFlow` |

The Snapshot carries them (13.4); the names are `spec-observability.md` 5.3.2 and 5.3.3.

### 3.2 The sources and drains of money in version 1

| Reason | Kind | Owner | When |
|--------|------|-------|------|
| `initial` | faucet | `spec-agents.md` 3.5, `spec-viability.md` 8.3 | construction; an external agent added in a run |
| one reason per grant | faucet | this spec | each tick the flow is active (5) |
| one reason per levy | sink | this spec | each tick the flow is active (5) |
| the external module | faucets and sinks | `spec-external-sector.md` 8.1: `outside_budget` (faucet), `outside_unspent`, `outside_receipts` (sinks) for money | each tick a counterparty trades |
| later specs | internal | FIS (taxes, duties, tolls and payouts), CRD (claims and interest), RSK (`loss`, goods only), LND (rent) | a fiscal, credit or rent reason moves money between non-boundary accounts and is neither a faucet nor a sink; a mint or a tribute is a flow of this module with the target kind `treasury` (4.3.5) |

The internal money reasons (`trade`, `estate_money`, `start_money`, `relief`, `split`,
`merge`, `births`, `starvation`, the escrow reasons) move money between non-boundary
accounts and change the sectors, not `M`.

### 3.3 What this spec does not own

The faucets and sinks of other specs are theirs; this spec owns the statement that all of
them are registry rows and the measures above. It does not own: the pool's rules
(`spec-estates.md`), the amount a newcomer is given (`Role.StartMoney`, the spawner), the
outside world (EXT), saving and investment decisions (`spec-saving-and-investment.md`), or any
rule that adjusts a flow in response to a measure (PLC).

### 3.4 Mechanisms considered for extraction and issuance

The roadmap asks where money enters ("mints, mines"). Four mechanisms were compared; the
reason for the choice is in the right-hand column.

| Mechanism | What it is | Decision and reason |
|-----------|-----------|---------------------|
| A. Money as a recipe output | a source recipe whose output is money, created at completion | **Rejected.** The executor receives a fixed amount per execution, whatever the market: a configured wage for the role, which `spec-currency.md` 2.6 forbids. It changes PRO's output type and `UnitCost`, the tradable set and WEL's income |
| B. Conversion of a commodity | a standing buyer that pays a fixed price in money from a boundary for a commodity (an issuer buying a rare commodity at a fixed price) | **Not here; reserved for EXT.** It is a fixed-price buyer in the outside world. It needs a bid backed by a boundary, and the ledger refuses a boundary reservation (`spec-ledger.md` 6.1, `ErrBoundaryReserve`), so `spec-clearing.md` 3 would need a boundary-backed admission. Open question 1 |
| C. A grant to a class | a recurring transfer from a boundary to the households of a role or to a pool | **Chosen.** No new account kind, no change to PRO, CLR or WEL beyond one income term. An extracted commodity in a pack is the ordinary commodity at its market price and, if the setting wants money to come from extraction, a grant to the holder class on a stated schedule |
| D. A stipend to a non-producing class | the same transfer as C, to a role with an empty `Recipes` list | **Chosen: it is C.** This is the recurring named income DEM needs (N-078) |

C is a policy of "who gets how much, on what schedule", and the schedule is data. It does
not make money depend on how much a miner digs. A setting that needs that dependence is
describing a fixed-price buyer, which is B and EXT's.

### 3.5 Conservation, audit and the sectoral identity

3.5.1. The flows add no way to change a balance other than the transfer, so the identities
of `spec-ledger.md` 9.1 hold unchanged: the zero total, non-negativity (a levy collects at
most what is held, 5.4), conservation by reason and exact equality.

3.5.2. **The sectoral identity** (`spec-ledger.md` 9.3) takes its terms from the flows. A
grant to a household class raises the households sector by its applied amount; a grant to
a pool raises the public sector; a grant to a treasury raises the treasury sector; a levy
lowers the sector it is drawn from; and in each tick
`delta(households) + delta(public) + delta(contract) + delta(treasury) = faucets - sinks`,
where the faucets and sinks now include the flows' reasons (the treasury term is 0 with no
governed jurisdiction, `spec-ledger.md` 9.3). Money moved between sectors by relief,
starting money and escrow changes the split and not the sum.

3.5.3. **What the audit adds.** `World.Audit` and `WithAudit` (`spec-ledger.md` 12.4) also
check, for each flow, that its `Total` equals the absolute boundary balance of its reason
and has the right sign (FR-MON-026, FR-MON-027). The reason of a flow is touched by that
flow alone (4.5.2), so the equality is exact. A violation is reported with the flow ID.

## 4. The money flow

### 4.1 The module and its types

`Config.Modules.MoneySupply` is a pointer; `nil` is off (`spec-config.md` 3.2). When off,
nothing in this spec runs, no state is added to an agent or the world, no reason is
declared and no event is emitted. A non-nil module with no flows is legal and does
nothing (the warning `no_flows`). Plain data, sorted by ID, no maps:

```go
// MoneySupply is the module. Its presence switches it on.
type MoneySupply struct {
	Flows []MoneyFlow // sorted by ID
}

// MoneyFlow is one configured faucet (Kind "grant") or sink (Kind "levy").
type MoneyFlow struct {
	ID        string     // also the ledger reason name; not a built-in reason
	Kind      string     // "grant" or "levy"
	Target    FlowTarget // who receives or pays
	Amount    int64      // minor units per household (or per pool) per Per; grant rate or fixed levy
	Level     int64      // grant only: top the balance up to this, per household or pool
	RateBP    int64      // levy only: basis points per Per of the balance above Exempt
	Exempt    int64      // levy only: minor units per household (or pool) never collected
	Per       string     // "tick" (default) or "year"
	Start     int64      // first tick the flow is active (default 0)
	Until     int64      // first tick it is no longer active; 0 means never
	Limit     int64      // most the flow may ever apply, in minor units; 0 means none
	NotIncome bool       // grant to households only: not income for the welfare ratio
	Label     string     // presentation only
}

// FlowTarget names a class of accounts. This spec owns the type and these three fields;
// the later kinds add their own (4.3.4, 4.3.5).
type FlowTarget struct {
	Kind         string // "households", "pool", "stores" (STO) or "treasury" (FIS)
	Region       string // "" means every region
	Role         string // households only
	Store        string // stores only: a store class; "" means every class (`spec-storage.md` 7.1.2)
	Jurisdiction string // treasury only: a governed jurisdiction (`spec-fiscal.md` 3.2)
}
```

4.1.1. **Exactly one shape.** A grant has either `Amount` (a rate) or `Level` (a top-up).
A levy has either `Amount` (fixed) or `RateBP` (a rate on the balance). `Exempt` belongs
to a levy only, `Level` and `NotIncome` to a grant only, and `NotIncome` to a grant to
households only. Any other combination is `ErrFlowShape` (12.1). For a target of kind `stores`, `Level` and
`RateBP` are `ErrFlowShape` too: only the fixed levy and the rate grant are defined (4.3.4).

4.1.2. **Defaults.** The zero values are the defaults: `Per` is `tick`, `Start` is 0,
`Until` is 0 (open ended), `Limit` is 0 (none), `Exempt` is 0 and `NotIncome` is false (a
grant counts as income). A flow has no default `Kind`, `Target` or shape: the author
states them. There is therefore no default macro constant in this spec (N-027).

### 4.2 The four shapes

| Shape | Kind | Fields | Recipient due, per tick (before limits) |
|-------|------|--------|-----------------------------------------|
| Rate grant | grant | `Amount`, `Per` | `w * Amount * S / (P * 10^4)`, spread with a carry (6.1) |
| Top-up | grant | `Level` | `max(0, w * Level * S / 10^4 - balance)` |
| Fixed levy | levy | `Amount`, `Per`, `Exempt` | as the rate grant, then capped by `max(0, balance - w * Exempt)` |
| Rate levy | levy | `RateBP`, `Per`, `Exempt` | `max(0, balance - w * Exempt) * RateBP * S / (P * 10^8)`, spread with a carry (6.3) |

`w` is the recipient's weight (1 for a pool), `S` the flow's scale in basis points (4.4,
default 10 000), `P` the number of ticks in the period: 1 for `tick`, `TicksPerYear` for
`year`. A top-up has no period. All division truncates toward zero on non-negative values
(architecture.md 3.5), and the remainder is carried, not lost (6.1). For a target of kind
`stores` only the rate grant and the fixed levy are defined, with `w` replaced by the number
of stores (4.3.4); `Level` and `RateBP` are `ErrFlowShape`.

### 4.3 Targets (classes)

4.3.1. **Households of a role.** Every open account of kind Agent whose agent's role is
`Target.Role` and, if `Target.Region` is not empty, whose region is that region. External
agents are agents and are included when their role matches. The estate pseudo-agents are
not households and are never in a class (`spec-estates.md` 2.7.4). An amount is per
household, so the account of an agent of weight `w` gets `w` times it (`spec-scale.md`
4.4: quantities of a cohort are `w` times the household amounts). Membership is evaluated each tick from the agent's current role and
region, so an agent that changes role or region moves between classes with its carry.

4.3.2. **A pool.** The Estate account of `Target.Region`, or of every region when it is
empty. The amount is per pool, weight 1. A pool target needs the estates module
(`ErrModuleDependency`, 12.1): the pool is the module's (`spec-estates.md` layer note).
The pool is where a grant to the region's pool goes; a grant to a jurisdiction's treasury is a
target of kind `treasury` (4.3.5). What the pool does with money is the estate's (relief,
starting money, `spec-estates.md` 2.3, 2.6).

4.3.3. **Regions** are visited in region order (ID order, `spec-config.md` 3.4) and the
accounts of a class in ascending account ID, so every total and every limit is applied
in one order that no map influences (architecture.md 7.5).

4.3.4. **Stores** (target kind `stores`; the type's field `Store`, `spec-storage.md` 7.1.2).
Every open Agent account whose agent owns at least one store of class `Target.Store` (every
class if empty) in `Target.Region` (every region if empty). **An amount is per store, not per
household**: the account of an owner of `n` matching stores is due `n` times the amount, where
4.3.1 multiplies by the weight `w`, because a store is not a household's quantity. `Exempt` is
an amount per account. Membership is read at the position of the flow (P1.7), so a store added
or removed between ticks counts from the next P1.7. Only the rate grant and the fixed levy are
defined (4.2); the kind needs the storage module (`ErrModuleDependency`) and a class that
exists (`ErrUnknownStoreClass`, 12.1). The carry is kept per account, as for households (14).
The flow is STO's first use of 10.3 (`spec-storage.md` 7).

4.3.5. **A treasury** (target kind `treasury`; the type's field `Jurisdiction`,
`spec-fiscal.md` 3.2.3, 4.2). The Treasury account of `Target.Jurisdiction`, a governed
jurisdiction (`spec-polity.md` 5): `ErrUnknownJurisdiction` if it is not defined and
`ErrNotGoverned` if it has no seat; with `Jurisdiction` empty, the treasuries of every governed
jurisdiction in ascending `JurisdictionID`. `Region` and `Role` are not read for it and must be
empty (`ErrFlowShape`): a treasury is in the region of its seat. The amount is per treasury,
weight 1, with the four shapes of 4.2 as for a pool. A grant to it is a faucet into the
treasury sector `M_t` and a levy on it a sink from it, both at P1.7, before the fiscal flows
of `spec-fiscal.md` at P1.7b, so a mint is spendable by a payout of the same tick. The treasury
is not the pool (4.3.2): a treasury is a jurisdiction's account, the pool a region's.
A tax, a duty, a toll and a payout between a treasury and the economy are not flows of this
module; they are `spec-fiscal.md`'s, and their IDs share this module's namespace (4.5.2).

### 4.4 Schedule, limit and scale

4.4.1. **Window.** A flow is active in tick `t` when `t >= Start`, and either `Until` is 0
or `t < Until`, and it has not ended (4.4.2). Outside the window it does nothing and
changes no state. A one-off endowment is a rate grant with `Until = Start + 1`: it pays
one tick's amount once. The first `Step` is tick 0, so `Start` 0 is the first step;
nothing of a flow is applied during construction, so the construction bucket
(`spec-ledger.md` 7.4.1) holds only `initial`.

4.4.2. **Limit.** `Limit`, when above 0, is the most the flow will ever apply (the sum of
its `Applied` over all ticks). The effective limit is the smaller of `Limit` (if set) and
`MaxFlowTotal`, the bound of 11.2. When an amount would take the total past it, the amount
is cut to what remains; when the total reaches it, the flow **ends** for good and
`MoneyFlowEnded` is emitted once (13.1). A flow shared by several regions spends one
limit in region order. The limit is the damper of a finite endowment (MON-D1).

4.4.3. **Scale.** Each flow has a runtime **flow scale** `Scale`, in basis points, default
10 000, bound 0 to `MaxFlowScale`. The effective `Amount`, `Level` and `RateBP` are the
configured values times `Scale / 10 000`. It is world state, not configuration: it is
changed only through the world's operation (10.4), between steps by a consumer or at
P0.3a by a policy actor (`spec-policy.md`); it is saved in a checkpoint and it is not part of
the configuration hash. No rule of this module changes it. `Scale` 0 pauses the flow.

4.4.4. **Total.** Each flow keeps its cumulative `Total` applied (grants issued, levies
collected), a non-negative `int64`. It is state, and the ledger's boundary balance of the
flow's reason equals `-Total` for a grant and `+Total` for a levy at every moment
(FR-MON-026).

### 4.5 Declared reasons

4.5.1. A flow's ID is its ledger reason. At construction the ledger's registry is the
built-in rows (`spec-ledger.md` 7.2) plus one row per flow, kind faucet for a grant (from
a boundary to an Agent or Estate account) and sink for a levy (from an Agent or Estate
account to a boundary), money only, owner this spec. The registry is then closed for the
run (ledger 2.3). The boundary accounts of the declared reasons are created at
construction in flow-ID order, after the built-in ones and before any other account.

4.5.2. An ID that equals a built-in reason, or a reason declared by another flow or by a
later spec's registration (the fiscal flows of `spec-fiscal.md` 3.4 share this namespace), is
`ErrReasonCollision`. IDs follow `spec-config.md` 3.3, so a
reason is usable as a log value and a metric attribute without escaping. The set of
reasons is bounded by the configuration (`MaxMoneyFlows`), so the `reason` attribute and
the aggregated `BoundaryFlow` keep a bounded cardinality (observability-and-logging.md
5.4).

## 5. The step

### 5.1 Where it runs

5.1.1. The flows run in phase 1 (Start) of the tick (`spec-agents.md` 4), after the relief
payment (`spec-estates.md` 2.6; starting money moves at an entrant's creation in phase 7,
`spec-world.md` 6.12, C1) and before production begins. So
a grant arrives before the recipient's `Trade` of the same tick (`spec-clearing.md`
3.3.1, `spec-demand.md` 7.3 item 2), and a levy sees the balance the household starts the
tick with, after its grants and any relief. No reservation exists in phase 1
(`spec-ledger.md` 6.5), so a balance is the amount available.

5.1.2. `spec-world.md` fixes the position: P1.7, after P1.6 (relief) and before P1.8 (the
outside's walk) and P1.9 (the directives). Relief is paid first because it draws on the
pool of the previous tick's entry step (`spec-estates.md` 2.6, N-082); a grant to a pool
therefore feeds the same tick's entry (P7.1 takes the pool budget after the deaths,
`spec-world.md` 6.12, C15) and the next tick's relief.

### 5.2 The order

For each tick, with the module on:

1. **Grants**, in ascending flow ID: for each active grant, for each region of its
   target in region order, for each account of the class in ascending account ID, compute
   the due (5.3), limit it (5.4), and transfer it from the flow's boundary to the account
   with the flow's reason.
2. **Levies**, in ascending flow ID, in the same nesting: compute the due, limit it, and
   transfer it from the account to the flow's boundary.

Grants come first so that a levy sees what the tick's grants gave; the order is a rule
constant, not a parameter. After the step the aggregates are accumulated and the events of
13.1 are emitted (architecture.md 12.5; observability-and-logging.md 4.4: after the
transitions they describe).

### 5.3 The due of one recipient

With `w` the weight, `bal` the account's money, `S` the flow's scale, `P` the period in
ticks and the arithmetic of 6:

| Shape | Due |
|-------|-----|
| Rate grant | `num = w*Amount*S + carry`; `due = num / (P*10^4)`; `carry = num mod (P*10^4)` |
| Top-up | `due = max(0, w*Level*S/10^4 - bal)` |
| Fixed levy | as the rate grant, then `avail = max(0, bal - w*Exempt)`; `collect = min(due, avail)` |
| Rate levy | `base = max(0, bal - w*Exempt)`; `num = base*RateBP*S + carry`; `due = num / (P*10^8)`; `carry = num mod (P*10^8)`; `collect = min(due, base)` |

The due of a household depends only on its own weight, balance and carry, the flow's
fields and scale, and the tick: not on any other account, any price or any total
(FR-MON-006). A test varies everything else and asserts the due does not move.

### 5.4 What is not applied

Three things can leave part of a due unapplied. None is an error; each is counted in
`MoneyFlowLimited` (13.1) and none is carried forward as a debt:

- **Short** (levies): the recipient holds less than the due above the exemption. The levy
  collects what is there. The unpaid part is forgiven: the carry has already advanced as
  if it were paid, and no arrears accrue (the ledger has no claims in version 1,
  `spec-ledger.md` 10).
- **Headroom** (grants): the credit would take the balance above `MaxMoney`
  (`spec-ledger.md` 4.1). The grant pays up to `MaxMoney` and the rest is not issued.
- **Limit** (both): the flow's total would pass its effective limit (4.4.2). The amount is
  cut, then the flow ends.

A transfer the ledger refuses for any other reason (an unknown or closed account cannot
occur here) is a programmer error and panics with a message that says so
(architecture.md 8.1; `spec-ledger.md` 12.2).

### 5.5 Atomicity

Each transfer is atomic (`spec-ledger.md` 2.6). The step is made of transfers, so a
checkpoint taken between ticks never sees half a step (ADR 0002 decision 4). No event is
emitted until the step's transfers are done (architecture.md 12.5).

## 6. Arithmetic and remainders

### 6.1 The carry

An amount per tick or per year is rarely a whole number of minor units per tick: 7 a year
over 365 ticks is 7/365 a tick. The flow keeps, for every recipient, an integer **carry**
`c` in units of `1 / (P * 10^4)` of a minor unit, with `0 <= c < P * 10^4`, and pays
`floor((w*Amount*S + c) / (P*10^4))` each tick, keeping the remainder as the next carry.
Over any run of ticks with unchanged `w` and `S` the total paid is
`floor((n*w*Amount*S + c0) / (P*10^4))`: exact, with no drift and no lump. A change of `w`
or `S` between ticks changes only the term added that tick; the carry is in units that do
not depend on either, which is why `S` is inside the numerator and not applied to the
result.

6.1.1. A cohort of weight `w` therefore receives exactly the `w` households' total, up to
less than one minor unit of carry (the model of section 18 shows weights 1, 3 and 40
paying 7, 21 and 280 over a year).

6.1.2. **Forfeit, not deferral.** The carry advances by the due, not by what was applied
(5.4). A recipient that could not receive does not accumulate a claim.

### 6.2 The initial carry

A recipient's carry starts at `(ID mod P) * 10^4`, where `ID` is the account ID (for a
pool, the region's index in region order). The offset spreads the first payments of a
small rate over the ticks of the period with no random number (the same device as
`spec-demand.md` 4.4). Example: 7 a year, `P` 365: account 0 is first paid in tick 53
(`7*53 = 371 >= 365`), account 100 in tick 38 (`100 + 7*38 = 366`); each is paid 7 in a
year. A new account starts from its own ID, whenever it is created. The carry of an account
that closes is dropped with it (14).

### 6.3 The rate levy

The levy reads the balance above the exemption at the moment it runs (5.1) and takes
`RateBP * S / 10^8 / P` of it per tick: basis points per tick when `Per` is `tick`, or
per year spread evenly over the year when `Per` is `year`. The per-tick rate is
**linear**, `rho = RateBP / (10^4 * P)`, not compounding: a rate of 5 000 basis points a
year over 365 ticks is `rho = 1/730` a tick, which over a year takes about 39% of an
untouched balance (`1 - (1 - 1/730)^365`), not 50%. A pack that wants the compounding
figure converts. The carry of a rate levy starts at 0 and is in units of `1/(P*10^8)`.

### 6.4 Bounds and checked arithmetic

6.4.1. Every product goes through the checked multiply-divide helper with a 128-bit
intermediate (`spec-currency.md` 7; architecture.md 3.5). The largest numerator is
`MaxWeight * MaxFlowAmount * MaxFlowScale = 10^6 * 10^12 * 10^5 = 10^23`, which does not fit
`int64` but fits 128 bits. A quotient above `MaxMoney` is treated as `MaxMoney` in the
comparisons of 5.4: a credit never exceeds the recipient's headroom, which is at most
`MaxMoney`, so the saturation can never produce a wrong payment, only a larger forfeited
due. The carry is the exact 128-bit remainder in every case.

6.4.2. A flow's `Total` and the boundary balance of its reason cannot overflow:
the effective limit is at most `MaxFlowTotal = MaxMoney / MaxMoneyFlows` (11.2), so all
flows together create at most `MaxMoney`, and with `initial` (at most `MaxMoney` at
construction, `spec-config.md` 4.2) the money supply stays below `2 * MaxMoney`, and below `3 * MaxMoney` with the outside (the list value of the buy sides of the external module is at most `MaxMoney`, `spec-external-sector.md` 13.2),
about `2 * 10^18`, inside `int64`.

### 6.5 Determinism

No random number is drawn. The order is fixed by 4.3.3 and 5.2. The carries are state
(14). The step is a pure function of the ledger, the agents' weights, roles and regions,
the flows' state and the tick, so the output is bit-identical across runs and platforms
(architecture.md 7.4) and a restored world continues exactly (FR-MON-040).

## 7. What a grant is to its recipient

7.1. **Weight.** An amount is per household and the account of an agent of weight `w`
receives `w` times it (4.3.1). The welfare ratio does not depend on the weight
(`spec-scale.md` 4.3), and a grant adds `w` times the household amount to `income_t`, so
it adds the same to the ratio of a cohort and of a single household.

7.2. **Income.** A grant to households that is not `NotIncome` is **unearned income** of
the recipient for the tick it is paid: it is added to `income_t` of
`spec-welfare-and-wages.md` 2.1 next to relief `F_t`, and it is excluded from the observed
wage of 4.3 there, because it is not earned by working. A grant with `NotIncome`, like a
starting-money grant (`initial`, `start_money`), is not income. A levy of this module is not
netted from `income_t` in version 1: it is not an input cost and not wear, and upkeep is not
netted either. The reason for the default is the household of a non-producing class: with its
stipend counted, its ratio is above the target, and the target-income rule (which has no
job to do for it) is not tripped. The reason for not netting a levy is that a levy is a
holding charge or a sink and not a public receipt: a tax, a duty and a toll of
`spec-fiscal.md` are netted, and `spec-welfare-and-wages.md` 2.1 states the one rule per flow
(open question 5).

7.3. **A non-producing class.** A role with an empty `Recipes` list and a grant to it
(N-078) has income, buys its upkeep and, with the demand module, its wants
(`spec-demand.md` 7.3 item 2). The warning `class_without_income` of `spec-demand.md`
10.2 is not raised for a role that a grant to households with `NotIncome` false targets
(handoff, section h).

7.4. **The pool.** A grant to a pool raises `M_p`, not `M_h`. The money reaches
households only through the pool's own uses: relief at `PoolRelease` per tick, and the
starting money of entrants, bounded by `EntryPoolShare` (`spec-estates.md` 2.3, 2.6;
`spec-spawner.md` 7.3). A pool grant is therefore the way to fund a growing population
(8.6) and a famine region's relief without touching households directly. A pool with
`PoolRelease` 0 and no entrants holds the money out of circulation (hoarding by the pool,
STB-L31): the grant does not reach anyone. Rent, interest and the claim payments that the
estate receives (`spec-land.md`, `spec-credit.md`) enter the same pool and reach households by
the same two uses.

7.5. **Relief and a levy on the pool.** A levy on a pool collects from the estate account
after relief has been paid in the same tick (5.1), and before that tick's entry, so it
never takes money relief has already committed.

## 8. The closed economy and what "deflation" means

### 8.1 Definition

A **closed economy** is a world in which no money faucet or sink acts after construction:
`Modules.MoneySupply` is nil (or has no active flow) and no later spec's faucet or sink is
on. It is ADR 0001's zero configuration and the tested baseline. `initial` still creates
the endowment at construction.

### 8.2 What is exact

In a closed economy, at every tick boundary:

1. `M` equals `M_init` exactly (3.1.2), for every seed.
2. `M_h + M_p + M_c = M`: money moves between the sectors (estates, relief, start money,
   escrow) and never leaves.
3. Money per household `M_h / H` changes only because `H` changes (births, entry, removal,
   weight changes) or because money moves between households and the pool. No price,
   volume or behaviour changes `M`.
4. The household-weighted mean welfare ratio is 1 if households buy nothing but upkeep, or
   `1 + W/U` with wants (`spec-viability.md` 4.7, `spec-demand.md` 7.1, N-054). MON adds
   nothing to that result.

All four are assertions of the invariant tests (testing.md 6.3: "money is conserved except
by explicit faucets and sinks") and need no tolerance.

### 8.3 What is not claimed

The library does not claim that a closed economy's prices fall. There is no price level
state in the engine and none is derived (roadmap, "Macro is measured, never set"). The
roadmap's phrase "a closed economy deflates" is a conditional, and 8.4 states it.

### 8.4 How deflation pressure arises

Define the turnover `V` of a window as the money value of the trades in it divided by `M`
(an indicator, IND's: `Turnover`, `spec-indicators.md` 9 row 8; the identity `sum(p*q) = M*V`
is its definition and asserts nothing).
With `M` fixed, nominal trade value can rise only if `V` rises. So the mean price falls
exactly when the real volume of trade (the quantities traded at constant relative prices)
grows by more than `V` does. In this library real volume grows when households are added
(entry or births funded from the pool, which keep `M` unchanged), when a shock is
recovered from, or, later, with technology; it falls with hoarding, which lowers `V` (the measure of what is hoarded is `IdleMoney`, `spec-saving-and-investment.md` 8.4). A
stationary closed economy (constant households, constant output, constant behaviour) has a
constant mean price. The pressure is therefore a statement about `M / (real volume)`, and
the measure is `M_h / H` together with `V`. In this design the effect is not a flow from
the identity to the price; it acts through the budgets of the households (8.5).

### 8.5 The liquidity floor

A household buys its upkeep in phase 3 with money it held at the start of the tick;
money earned in phase 4 is spendable no earlier than the next tick (`spec-agents.md` 4.1,
`spec-clearing.md` 3.3.1). So for the upkeep to be affordable out of cash on hand, a
household needs `M_h / H` of at least its **upkeep cost** `U_c` (one tick of its basket at
its beliefs): this is the **liquidity floor**, necessary and not sufficient. Above it,
tiers 1 and 2 (upkeep and the inputs of a job) are funded. Buffers and wants are funded
only above the reserve `ReserveTicks * U_c` (`spec-agents.md` 7.4.2, 7.4.4;
`spec-demand.md` 7.2). So there are two thresholds on money per household, in order of
falling `M_h / H`:

| Threshold | Money per household | What stops below it |
|-----------|--------------------|--------------------|
| Spending threshold | `ReserveTicks * U_c` | buffers (tier 3) and wants (tier 4); the sellers of want goods earn nothing (STB-L45) |
| Liquidity floor | `U_c` | upkeep cannot be bought from cash on hand; households go unfed (`Unfed`, `Hunger`) |

With the placeholder defaults (`ReserveTicks` 3, `B_r` 1 000 so `U_c` about 1 000 at the
reference prices, `RefBandLow` 1 000 basis points), the spending threshold is about 3 000
per household and the liquidity floor about 1 000 at the reference prices. Prices cannot
fall below the reference band (`spec-beliefs.md` 3.2.1), at `RefBandLow` of the reference,
so `U_c` can fall to about 100 and the floor with it; the band is a second, deeper floor
on how far deflation can go. Between the thresholds the market works at subsistence;
below the spending threshold it is a market for upkeep and inputs only. These are
consequences of the written rules, derived by hand (status `hand`) and not run in the
library; the default numbers are placeholders (N-015).

### 8.6 Holding money per household constant

A world whose households grow while `M` is fixed moves `M_h / H` down through both
thresholds. An example with the placeholder numbers: 1 000 households each holding 4 500
(`M_h = 4 500 000`, pool empty), one household added per tick. Without a grant,
`M_h / H = 4 500 000 / (1 000 + t)`: the spending threshold (3 000) is crossed at
`t = 500` and the liquidity floor (1 000) at `t = 3 500`. A pool grant of `G` a tick funds
`h` entrants a tick at `StartMoney s` when `G = h * s` (the pool pays each newcomer `s`,
`spec-estates.md` 2.3); here `G = 3 000` for `h = 1`, `s = 3 000`. Then `M` rises by `h*s`
and `H` by `h`, and `M_h / H` moves toward `s` and stays above the spending threshold when
`s` is at least `ReserveTicks * U_c`. The numbers are a derivation, not a run. Note that
the grant is a constant: it does not read `H` (a grant that scaled with `H` would be a
rule reacting to an aggregate, position 5). The newcomers' number is POP's and SPN's; a
pack picks `G` from its expected entry. If entry is faster than expected the pool runs dry
and entrants are partly funded (`spec-spawner.md` 7.3): the measure `M_p` shows it.

### 8.7 What the closed economy asserts for the tests

The invariant tests (testing.md 6.3) assert 8.2 over the reference chain and a range of
seeds, with `WithAudit`. The scenario S19 (`spec-reference-chain.md`) runs a closed economy
with upkeep only and the new scenario MON-S3 (16) a closed economy with growing
households, with and without a pool grant, measuring `M_h / H` against the two thresholds.

## 9. Faucets, sinks and pairs

### 9.1 An unbalanced faucet

A grant with no sink makes `M` grow without bound: `M(t) = M_0 + g * t` for a constant
issue `g` a tick (STB-L30). The mean price then drifts upward if real volume and `V` are
constant (8.4 read the other way): the rate of growth of `M` is `g / M(t)`, which falls
as `1/t`, so a fixed nominal faucet produces a steadily slowing inflation, not an
accelerating one. If the issue is per household and the households grow, `g` grows with
them; that coupling with population is POP's (MON-A5). In DEM's model a stipend of 300 a
tick to 50 households with no sink left household money at 47 700 against 2 700 at the
start (`spec-demand.md` 7.3 item 3; status `model`, not re-run here). The boundary balance
of the grant's reason is the measure, and nothing in the engine reacts to it.

### 9.2 A balanced pair

A recurring income for one class paid for by a levy on another (N-078) is two flows: a
grant to the recipients and a levy on the payers. The nominal yearly totals of the fixed
flows over the census are compared at validation (`pair_unbalanced`, 12.2): for each grant
and each fixed levy on households, `Amount * households * Y`, with `Y` the ticks in a year
when `Per` is `tick` and 1 when it is `year`, summed over grants and over fixed levies and
compared; a rate levy is not part of the comparison, because its yield depends on
balances (9.3). The model
of section 18 runs a pair of 50 recipients at 300 a tick against 450 payers at 33 a tick,
which are 15 000 and 14 850 a tick, for 400 ticks. The boundary balances are -6 000 000
for the grant and +5 940 000 for the levy; the net, -60 000, is the money created
(`150 * 400`), and the money in the economy is 1 410 000 against 1 350 000 at the start.
The drift of an unbalanced pair is visible and bounded by the difference of the fixed
amounts; a pack that wants none sets them equal or adds a rate levy.

### 9.3 The holding charge and its bound

A rate levy with exemption `E` per household (weight 1) on a class that also receives a
constant issue `g` a tick has, if the class never spends, the balance
`m(t+1) = m(t) - rho * (m(t) - E) + g` for `m > E`, with `rho = RateBP / (10^4 * P)` the
per-tick rate (6.3). It converges, from below, to

```
m*  =  E + g / rho
```

at the geometric rate `(1 - rho)`, a time constant of `1/rho` ticks. For `g` 300, `rho`
1/730 (5 000 basis points a year over 365 ticks) and `E` 0, `m*` is 219 000 and the time
constant 730 ticks. The model gives 28 016 at tick 100, 86 098 at tick 365, 189 143 at tick
1 460 (86% of `m*`, as `1 - e^-2` predicts) and 218 641 at tick 6 000. With `E` 1 000 the
bound is 220 000 and the model gives 219 641 at tick 6 000 (`flows.out`). So **a rate levy
bounds the money held by the worst case, a household that spends nothing**, by a number
that is derived from the flow's own fields. A household that spends, as the default
strategy does, holds less (`spec-agents.md` 7.4.4). It is the one damper in this spec that
does not need the author to match two nominal totals (MON-D2). The bound is on the class,
so the money of the whole economy is bounded only if every holder of money is in the
class of some rate levy or of a sink-matching pair; the pool is a holder too (7.4).

### 9.4 A levy that takes upkeep money

A levy runs in phase 1, before the household buys its upkeep (5.1). A fixed levy with no
exemption can take the money a household needs for tier 1 (`spec-agents.md` 7.4.3):
a household that is levied below its upkeep cost goes unfed, its privation rises
(`spec-spawner.md` 5.1) and the failure ladder starts. The exemption `Exempt` is the
guard: the levy never takes the balance below `w * Exempt`. The warning `levy_below_upkeep`
fires when `Exempt` is below one tick's upkeep cost of the targeted role at the reference
prices (12.2), and MON-A4 records the hazard. The levy is not netted from income (7.2), so
the target-income rule does not see it.

### 9.5 Why a faucet is never proportional to a balance

A faucet of `r` basis points per tick of the recipient's own balance gives
`m(t+1) = m(t) * (1 + r/10^4)`: exponential growth with no bound, the compound interest that CRD does not pay (interest in
`spec-credit.md` is a transfer between accounts, not a faucet) and a return on savings that SAV
does not pay (`spec-saving-and-investment.md` 2, 8.2). It is a reinforcing loop with gain above 1 on every
balance, and the hoarding that it rewards cannot be damped by anything local. This spec
therefore allows only fixed or levelled faucets, whose total over `n` ticks is linear in
`n` (a top-up is capped by the level). A consumer who wants interest has the credit and
saving specs, which will carry their own entries.

## 10. How EXT, FIS, STO and PLC plug in

### 10.1 A counterparty whose balance is not conserved is a boundary

**The rule.** Any account whose balance is created or destroyed outside the economy's
conservation (the outside world, an unseen source of money, a drain of money out of the realm) **MUST
be a Boundary account** in the ledger (`spec-ledger.md` 3.1), never an ordinary account
that is topped up quietly. A Boundary account may go negative and has a region only
through the flows attributed to it (`spec-ledger.md` 3.5). This keeps the identity of 3.1.2
true and every faucet and sink a named reason. EXT's counterparty is therefore an ordinary pass-through account
of its own kind, topped up from and swept to Boundary accounts under its own reasons (`spec-external-sector.md` 4.1, 8.1). This is FR-MON-043; the audit checks the identity it protects (FR-MON-026).

### 10.2 The top-up as the primitive for a pass-through counterparty

A counterparty that must place orders (an outside buyer with a budget) needs money in an
ordinary account to reserve against (`spec-ledger.md` 6.1, `spec-clearing.md` 3). The
primitive for it, in this spec, is the pair of shapes already defined: a **top-up grant**
that raises the account to `Level` at the start of the tick, and a **rate levy of 10 000
basis points per tick with `Exempt = Level`** that sweeps everything above `Level` at the
next tick's start. The first is a named faucet and the second a named sink, so an outside
buyer's purchases show as faucet money in `BoundaryFlow`, the proceeds of an outside seller
as swept sink money, and their balance never exceeds `Level` between ticks. A level of `L`
means the counterparty can spend at most `L` in a tick. EXT chose its own account kind (Outside) and its own six reasons instead of these shapes (`spec-external-sector.md` 4.1); the shapes of this section remain available to a pack for a pass-through class of its own, and the target kind `"external"` is released and defines nothing (open question 1). Example: a counterparty
at `Level` 5 000 whose balance is 3 800 after a tick is topped up by 1 200; at 6 300 the
sweep collects 1 300. Both amounts are exact (the top-up has no carry; the sweep's due is
`base * 10 000 / 10 000`).

### 10.3 Later specs add reasons through the same path

FIS (taxes, duties, tolls and payouts), EXT (imports and exports), CRD (claims and interest),
LND (rent) and RSK (loss, goods only) add rows to the ledger registry as `spec-ledger.md` 7.2
says; of these only EXT adds faucets and sinks of money (8.1 there), and the others are
internal reasons that move money between non-boundary accounts. They are not flows of this
module unless they choose to be: a spec that wants a configured, scheduled faucet or sink may
reuse `MoneyFlow` and its step (4, 5) and add a target kind. Two have: STO's `stores` (4.3.4,
the first use) and FIS's `treasury` (4.3.5). A reason a later spec adds MUST NOT collide with a
flow ID (`ErrReasonCollision`, FR-MON-036). The type `FlowTarget` and its three old fields
are this spec's; each new kind adds its field and is owned by the spec that adds it.

### 10.4 How a policy plugs in without this spec containing one

A policy actor (PLC) reads read-only measures and returns settings (ADR 0003 decision 3).
ADR 0001 decision 4 lists the actions it may take: a transfer between accounts (validated
by the ledger like any transfer, so it cannot create money without a declared faucet), a
change of a rate, and stock added to a boundary. MON supplies the seam for the second: the
**flow scale** (4.4.3). The public call is `SetFlowScales([]FlowScale{Flow; BP})`
(`spec-integration.md` 5.1): it sets `Scale` for a flow between steps; it validates the module
(`ErrModuleOff`), the flow ID (`ErrUnknownFlow`, new) and the bound (`ErrOutOfBounds`), and the
new value applies from the next P1.7. A fiscal flow's ID is accepted too (`spec-fiscal.md`
13.1). The scale is state (14), the operation emits `MoneyFlowScaled` per changed flow (with
the `Cause` `consumer` or `policy`), and a consumer's own rule, a human at a console, a
scripted schedule or a policy actor can set it. A policy actor of `spec-policy.md` writes it at
P0.3a through the internal form, with the cause `policy`; the public call stays between ticks.
Two consequences. First, no rule of this module changes a flow in response to a measure: the
writers of `Scale` are a call from outside and a policy actor, whose rule is the consumer's
or PLC's (`spec-policy.md` 3.5.1 gives the reach of a lever). Second, a policy actor that
doubles a stipend or pauses a faucet does so within the bounds of the configured flow and the
ledger's checks; it cannot create money by any path that is not a declared flow, and the audit
still holds.

### 10.5 What this spec will not do

It will not create a flow in response to a measure, add a faucet proportional to a balance
or the supply (9.5), convert a commodity into money (3.4), tax a trade (`spec-fiscal.md`), model the
outside world (EXT), or lend (`spec-credit.md`). A grant to an agent class by an event (for example a
newcomer's endowment beyond the pool's) is not provided: it would be a rule reacting to
entry, which is POP's and SPN's to decide (open question 4).

## 11. Parameters

### 11.1 The tunables

The tunables are the fields of a `MoneyFlow` (4.1). All are fields of
`Config.Modules.MoneySupply.Flows[]`, with an owner (this spec), a bound and a doc line
(architecture.md 9.5, 9.7), and each has a row in the parameter-owner table of `README.md`.
A flow has no default: the module has no flows unless the pack lists them, so there is no
default macro constant (N-027). The defaults below are the zero values of the fields, which
mean "absent".

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Module | `Flows` | the flows, sorted by ID (4.1) | none | at most `MaxMoneyFlows` |
| Flow | `ID` | the flow's name and its ledger reason (4.5) | none | an ID of `spec-config.md` 3.3, unique, not a built-in reason |
| Flow | `Kind` | `grant` (faucet) or `levy` (sink) | none | one of the two |
| Flow | `Target` | the class: kind `households`, `pool`, `stores` or `treasury`, an optional region, a role for households, a store class for stores, a jurisdiction for a treasury (4.3) | none | known region, role, store class and jurisdiction |
| Flow | `Target.Store` | stores only: a store class; "" means every class (4.3.4) | "" | a configured store class |
| Flow | `Target.Jurisdiction` | treasury only: a governed jurisdiction; "" means every governed one (4.3.5) | "" | a defined, governed jurisdiction |
| Flow | `Amount` | minor units per household (or per pool, per store, per treasury) per `Per`: a grant's rate or a fixed levy | 0 | 0 to `MaxFlowAmount` |
| Flow | `Level` | a top-up grant: the balance it raises to, per household or per pool | 0 | 0 to `MaxFlowAmount` |
| Flow | `RateBP` | a rate levy: basis points per `Per` of the balance above `Exempt` | 0 | 0 to `MaxFlowRateBP` |
| Flow | `Exempt` | a levy: minor units per household (or per pool) never collected | 0 | 0 to `MaxFlowAmount` |
| Flow | `Per` | the period of `Amount` and `RateBP`: `tick` or `year` | `tick` | one of the two |
| Flow | `Start` | the first tick the flow is active | 0 | 0 to `MaxInt64 - 1` (a tick) |
| Flow | `Until` | the first tick it is no longer active; 0 never | 0 | 0, or above `Start` |
| Flow | `Limit` | the most the flow may ever apply; 0 none | 0 | 0 to `MaxFlowTotal` |
| Flow | `NotIncome` | a grant to households is not income for the welfare ratio (7.2) | false | boolean |
| Flow | `Label` (p) | display name | "" | presentation only, not hashed |
| Runtime | `Scale` | the flow scale in basis points (4.4.3); state, not configuration | 10 000 | 0 to `MaxFlowScale` |

### 11.2 The bounds (`bounds.go`)

Bounds are constants, not tunables (`spec-config.md` 4). They are provisional until the
slice runs.

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxMoneyFlows` | 64 | the step loops over active flows and their classes each tick, so its cost is bounded by the configuration; also the divisor of `MaxFlowTotal` |
| `MaxFlowAmount` | `MaxMoney / MaxWeight` = 10^12 | `w * Amount` for the largest weight is at most `MaxMoney`, so one household's due fits |
| `MaxFlowRateBP` | 10 000 | at most the whole balance above the exemption per period |
| `MaxFlowScale` | 100 000 | ten times the configured value; the numerator then reaches 10^23, inside 128 bits (6.4) |
| `MaxFlowTotal` | `MaxMoney / MaxMoneyFlows` = 1.5625 * 10^16 | all flows together create at most `MaxMoney`, so with `initial` the supply stays inside `int64` (6.4.2) |

## 12. Validation

### 12.1 Errors

Each rule has a sentinel and a failing-case test (testing.md 3.9); an error names the path
`modules.money_supply.flows.<id>.<field>` (`spec-config.md` 7.1.1).

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field outside its bound in 11.1; a `Per`, a `Kind` or a target `Kind` that is not one of its values; `Until` not above `Start` when set; more than `MaxMoneyFlows` flows |
| `ErrFlowShape` (new) | a grant with `RateBP` or `Exempt`, or with both `Amount` and `Level` or neither; a levy with `Level`, or with both `Amount` and `RateBP` or neither; a household target with no role; a pool target with a role; `NotIncome` on a levy or on a target that is not households; `Level` with `Per` `year`; a stores target with `Level` or `RateBP`, a role, or a jurisdiction; a treasury target with a region, a role or a store class |
| `ErrDuplicateID` | two flows with one ID |
| `ErrReasonCollision` (new) | a flow ID that equals a built-in reason of the registry or a reason a later spec declares |
| `ErrUnknownRole`, `ErrUnknownRegion` | a target naming an undefined role or region |
| `ErrModuleDependency` | a pool target while the estates module is off; a stores target while the storage module is off |
| `ErrUnknownStoreClass` (STO) | a stores target naming a store class that is not configured |
| `ErrUnknownJurisdiction` (POL) | a treasury target naming a jurisdiction that is not defined |
| `ErrNotGoverned` (FIS) | a treasury target naming a jurisdiction that has no seat |

There is no derived overflow check of `spec-config.md` 4.2 for this module: `MaxFlowAmount`
is itself `MaxMoney / MaxWeight`, so `w * Amount`, `w * Level` and `w * Exempt` fit
`MaxMoney` for any weight the bound allows, and the larger products go through the
128-bit helper (6.4).

`ErrFlowShape` and `ErrReasonCollision` are ledger-adjacent sentinels that the root maps to
public errors (N-075).

### 12.2 Warnings

Never fail validation; each is logged once at construction through the injected logger at
`Warn` with `rule` and `path` (`spec-config.md` 7.2).

| Warning | Condition |
|---------|-----------|
| `no_flows` | the module is on and `Flows` is empty |
| `faucet_without_sink` | at least one grant and no levy flow at all: the money supply is unbounded (STB-L30). A grant with a `Limit` is exempt |
| `pair_unbalanced` | there is a grant to households and a fixed levy on households, and their nominal yearly totals over the `Groups` census differ by more than 1 000 basis points of the larger (the tolerance is a reporting rule constant, not a tunable); rate levies are not compared (9.2) |
| `levy_below_upkeep` | a levy on households whose `Exempt` is below one tick's upkeep cost of the targeted role at the reference prices of its regions (9.4) |
| `class_empty` | a household target whose class has no households in any targeted region at construction (it may fill later) |
| `grant_to_pool_no_outlet` | a grant to a pool in a region where `PoolRelease` is 0, so the money reaches nobody except entrants (7.4) |
| `store_levy_below_upkeep` (STO) | a levy that targets stores has an `Exempt` below one tick's upkeep cost, at the reference prices, of a limited role (9.4; `spec-storage.md` 7.2) |

## 13. Observability

### 13.1 Events

All are plain data, no map, no pointer, no wall-clock time, emitted after the transitions
of the step (observability-and-logging.md 4.1, 4.4). The amounts of every transfer are
already in `BoundaryFlow` (`spec-ledger.md` 12.1) with the flow ID as the `Reason`; these
events carry what `BoundaryFlow` cannot.

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `BoundaryFlow` (existing) | once per tick per (region, reason, money) with a non-zero total | `Region`, `Reason` (the flow ID), faucet or sink, total |
| `MoneyFlowLimited` (new) | once per flow and region in a tick in which any due was not applied; never when all was applied | `Flow`, `Region`, `Due`, `Applied`, `Short` (households whose levy found too little), `Headroom` (households whose credit was cut at `MaxMoney`), `Limit` (households cut by the limit) |
| `MoneyFlowEnded` (new) | once, in the tick a flow reaches its effective limit and ends | `Flow`, `Cause` (`limit` or `cap`), `Total` |
| `MoneyFlowScaled` (new) | once per changed flow of a call of `SetFlowScales`, or of a policy actor's write at P0.3a; also for a fiscal flow | `Flow`, `From`, `To` (basis points), `Cause` (`consumer` or `policy`) |

13.1.1. **`MoneyFlowLimited` is the counter of the silent clamps of this spec** (the
shortfall of a levy, the headroom of a grant, the limit), as `EntryLimited` is the
spawner's (FR-STB-013, N-045). Its size is bounded by flows times regions. `Due` and
`Applied` are integers in minor units; the households are the sum of weights.

13.1.2. **Metrics (OBS decides).** Proposed: `economysim.money.flow`, a Counter, unit
`{minor_unit}`, attributes `economysim.reason` and `economysim.kind` (`faucet` or `sink`),
fed by `BoundaryFlow` (money); `economysim.money.flow.unapplied`, a Counter, `{minor_unit}`,
attribute `economysim.reason`, fed by `MoneyFlowLimited` (`Due` less `Applied`). The
attribute values are the flow IDs, at most `MaxMoneyFlows`, so the cardinality is bounded
by the configuration (observability-and-logging.md 5.4). A gauge of `M` is not an event
metric; the snapshot-fed gauges are N-088's.

### 13.2 Logs

Nothing at `Info`, `Warn` or `Error` from the step. At `Debug`, guarded by `Enabled`: "money
flow limited" (`flow`, `region`, `due`, `applied`), "money flow ended" (`flow`, `cause`),
"money flow scaled" (`flow`, `from`, `to`). The constant messages are lowercase phrases and
the values are attributes (observability-and-logging.md 3.3.1). New log keys, to add to
3.3.2 with the code: `flow` (the flow ID), `due` and `applied` (money, minor units, logged
with `slog.Int64`); `from` and `to` (a value before and after, in basis points) are added
by `spec-observability.md` and the rules (3.3.2). `reason` and `region` exist. The warnings of 12.2 are `Warn` at
construction with `rule` and `path`.

### 13.3 Cost with no observer

With the default logger and observer the step allocates nothing beyond scratch held by the
world (NFR-MON-001): the events are built only when an observer was supplied
(observability-and-logging.md 4.5), the aggregates are integer additions into preallocated
slots, and no log argument is built unguarded.

### 13.4 Snapshot

The Snapshot carries, as plain data in a fixed order (field names are OBS's): per region,
`M_h`, `M_p`, `M_c`, `M_t`; the world's `M` and `H`; the money boundary balance per reason, in the
order of `spec-observability.md` 4.4.1.2; and per flow, its `Total`, `Scale` and whether it
has ended. `M_h / H` and velocity are computed by the consumer or by IND from these
(`Sectors` and `Turnover`, `spec-indicators.md` 9 rows 7 and 8). Nothing in the engine reads
the Snapshot (architecture.md 12.4; position 5). The field names are those of
`spec-observability.md` 5.3.2 (the regions) and 5.3.3 (the boundary and the flows).

### 13.5 The Observability entry (observability-and-logging.md 9)

1. **Failure paths.** Configuration errors are returned (12.1). A shortfall, a headroom cut
   or a limit is survived and is counted by `MoneyFlowLimited`; no `Warn` per tick, because
   a persistent shortfall would log every tick for every flow.
2. **State transitions.** The end of a flow and a change of scale have events; the
   payments are in `BoundaryFlow`. No event holds a map.
3. **Decision points.** None; a flow has no strategy and no draw.
4. **New loop.** The per-tick loop over flows and classes logs only at `Debug`, guarded by
   `Enabled`; the events are built only with an observer; no `Warn` inside it.
5. **Changed events or metrics.** None changed; three events and two metrics are added.
6. **Metric attributes.** `economysim.reason` and `economysim.kind`, bounded by
   `MaxMoneyFlows` plus the built-in reasons.
7. **Levels.** No `Info`, no `Error`.
8. **No logged returned error.**
9. **Tests.** Observability-and-logging.md 8: a recording observer for the events and their
   fields, a capturing handler for the `Debug` lines and the absence of `Info`, the no-op
   default, non-interference, the cardinality of the reasons, the emission order, and the
   allocation test (FR-MON-029 to FR-MON-033, NFR-MON-001).

## 14. State contributed to a checkpoint

All plain data in a fixed order, no maps (ADR 0002):

- per flow, in ID order: `Total` (an `int64`), `Scale` (basis points) and `Ended` (0 not
  ended, 1 limit, 2 cap);
- per flow with a household target and a carry shape (a rate grant, a fixed levy or a rate
  levy), in ID order: the carries, as (account ID, carry) pairs in ascending account ID, only
  for open accounts;
- per flow with a pool target and a carry shape: one carry per region, in region order;
- per flow with a stores target and a carry shape: the carries as (account ID, carry) pairs in
  ascending account ID, only for open accounts, one per account and not per store;
- per flow with a treasury target and a carry shape: one carry per treasury, in ascending
  `JurisdictionID`.

A top-up has no carry. The boundary balances of the declared reasons are in the ledger's
saved state (`spec-ledger.md` 11), not here. Not saved: the due, the per-tick aggregates and
`Applied` of the last tick (derived or cleared each tick), and the carry of a closed
account (dropped when the account closes, so the saved size is bounded by the open
accounts). The flows themselves are configuration and are in the hash, `Scale` is not
(ADR 0002 decision 3). A world restored between ticks continues exactly (FR-MON-040).

## 15. The warm start and the steady state with money flows

`spec-viability.md` solves the real economy: quantities, relative prices and wages, with
upkeep as final demand (4.7) and every group's upkeep counted, a non-producing class
included. This section says what that means when flows exist.

15.1. **The real side is unchanged.** A money flow adds no real demand and no real supply.
The reference programme and the warm table of quantities, jobs and relative seed prices
stay valid. A non-producing class that receives a grant adds its upkeep to the final
demand the programme already counts (`spec-viability.md` 3.3 item 1), and the labour of the
producers must cover it, which the programme checks.

15.2. **A balanced pair is neutral on `M`.** If the fixed grants and fixed levies match
(9.2), the nominal side is stationary and the warm start is as stable as without the flows.
The nominal level of the warm prices and the warm money are the solver's choice of
numeraire and of `MoneyBufferTicks` (`spec-viability.md` 8.3), both unaffected.

15.3. **An unbalanced faucet has no nominal steady state.** `M` grows by `g` a tick (9.1),
so the nominal prices drift; the warm start stands for the real state only. A pack that
needs a stationary start with an unbalanced faucet sets `Start` to the end of its warm-up
(the flow does nothing before), which is the use of the scenario S22 (warm against cold
start): the flow begins on a world that has settled. The claim of the warm start that the
world stays near the steady state (STB-L53) is a claim about real quantities in that case.

15.4. **A rate levy changes the steady money of its class.** The holdings of a class with a
rate levy are bounded by `m*` (9.3). A warm `Money` above `m*` for such a class is legal but
decays toward it; VIA MAY seed such a class at `min(Money, m*)` when the class is a
hoarder in its model (an optional edit, handoff section h). The household that spends its
income, as the default strategy does, is in the regime of `spec-agents.md` 7.4.4 and holds
about its reserve plus `e*` (`spec-demand.md` 7.2).

15.5. **Flows start at `Start`, not in construction.** Nothing of a flow is applied during
the construction phase; a flow with `Start` 0 first acts in the first `Step`, so the audit
at tick 0 (`spec-ledger.md` 12.4) checks `initial` only, and the identity of 3.1.2 holds at
every moment (FR-MON-026).

## 16. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9; fields as its section 3. The
register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn` numbers; entries here are
named `MON-An`, dampers `MON-Dn` and new scenarios `MON-Sn`, and say what they merge with.
**Status is `model` where section 18 supports it and `hand` otherwise; nothing here claims
stability** (N-027). The checklist's 9.1 question, "what does each new rule read and what
does it change", has the same answer for every flow: it reads the tick, its schedule and
totals, the weight and (rate levy, top-up) the recipient's own balance; it changes the
recipient's money and the money supply. Only a measure that feeds `Scale` from outside
(10.4) would close a loop through the economy, and that is the caller's.

**Dampers this spec adds or specifies:**

| ID | Damper | Home | Status |
|----|--------|------|--------|
| MON-D1 | Cumulative limit: `Limit`, and the effective `MaxFlowTotal`; a flow that reaches it ends and says so | MON 4.4.2 | specified (counted by `MoneyFlowEnded`, `MoneyFlowLimited.Limit`) |
| MON-D2 | Holding charge: a rate levy on the balance above an exemption, a balancing sink with a derived bound `m* = E + g/rho` | MON 6.3, 9.3 | specified; the bound is `model` for hoarders |
| MON-D3 | Exemption on a levy: the levy never takes the balance below `w * Exempt` | MON 5.3, 9.4 | specified; warning `levy_below_upkeep` |
| MON-D4 | A closed set of shapes: no faucet reads an aggregate or grows with a balance (9.5) | MON 2, 9.5 | specified (a rule constant: the shape set) |
| MON-D5 | Payment spread by a carry started from the account ID: no lump, no pulse of every recipient at one tick, no draw | MON 6.1, 6.2 | specified |

D11 and D12 (relief and the pool release) damp the pool as a holder (MON-A6).

**MON-A1. The fixed nominal faucet and the price level (neighbours STB-L05, L30).** *Kind:*
loop with drift. *Loop:* price level (-) real value of a fixed nominal grant (+) real
demand of its recipients (+) price level. One `-` link, so balancing (B). The drift is
not a loop: `M` rises by `g` a tick and, at constant real volume and turnover, the price
level rises at `g / M(t)`, a rate that falls as `1/t` (9.1). *Delay:* the belief
adjustment (`spec-beliefs.md` 5.2, a few ticks) and, for an income grant, the welfare
window `T`. *Owner:* MON 9.1. *Damper:* D04 (the reference band bounds the price),
MON-D1, MON-D2. *Undamped:* the nominal drift of an unbalanced faucet, by design, and
measured. *Condition:* real volume and turnover change slowly against `M`. *Test:* S13,
MON-S1. *Status:* hand (a quantity-theory reading, untested in the library).

**MON-A2. Money that cannot leave (merges STB-L30).** *Kind:* hazard. A grant without a
sink or a limit makes `M` unbounded; the boundary balance per reason shows it (3.1.3).
*Owner:* LED 14, now MON 9.1. *Damper:* MON-D1, MON-D2, and the warning
`faucet_without_sink`. *Undamped:* a pack that ignores the warning. *Condition:* every grant
has a limit or a sink (a rate levy on a class that holds the money, or a matched fixed
levy). *Test:* S13 (the faucet alone, report-only: `spec-stylised-facts.md` STY-S05 gives
no band, and says why: neither a source nor a model fixes the direction), MON-S1 (the sink
and the bound). *Status:* hand for the hazard, model for the bound `m*` of a hoarder.

**MON-A3. The holding charge (a balancing sink).** *Kind:* loop. *Loop:* the class's
money (+) the amount collected (+) money removed (-) the class's money. B. *Delay:* the
time constant `1/rho` ticks (730 in the example of 9.3), against the welfare window `T`
(30 to 120 ticks) and a year; a charge fast enough to be near `T` (`rho` above about 1/30)
acts within a window and can take a household's savings for a long job (L19, L20), which
the exemption is for. *Owner:* MON 6.3. *Damper:* itself; MON-D3 protects the savings.
*Undamped:* none known; a side effect is that holding money costs, so a household that
saves for a tool pays for it. *Condition:* `Exempt` covers the reserve and the longest
job's cash. *Test:* MON-S1. *Status:* model for a hoarder (the strategy that spends is
untested).

**MON-A4. A levy that takes upkeep money.** *Kind:* hazard. *Loop:* levy (+) money held
(-) upkeep bought (-) unfed households (+) privation (+) the ladder (role change, relief,
exit; `spec-spawner.md` 5) (-) the levied base. B in the end, but through hunger. *Delay:* the
privation count. *Owner:* MON 9.4. *Damper:* MON-D3, D11 (relief refills the poor from the
pool). *Undamped:* a fixed levy without an exemption on a class with no income above its
upkeep. *Condition:* `Exempt` at least one tick's upkeep cost (the warning
`levy_below_upkeep`). *Test:* MON-S2. *Status:* hand.

**MON-A5. A per-household grant and the population (a hook for POP).** *Kind:* loop,
owned jointly. *Loop:* grant (+) income and welfare ratio (+) births (POP) (+) households
(+) the issue (the grant is per household). R, bounded by POP's carrying capacity and the
food loop (L50). *Delay:* the welfare window and POP's. *Owner:* MON 4.3, 7.2; the rate is
POP's. *Damper:* `NotIncome`, MON-D1, `Scale` (10.4), or a grant to the pool instead of to
households, whose total does not depend on `H`. *Undamped:* POP's side. *Condition:* the
decision of POP on whether income from a grant raises births. *Test:* in POP. *Status:*
hand-off.

**MON-A6. Growing households in a closed economy (the liquidity drift).** *Kind:* hazard
(neighbours STB-L31, L45, regime F4). With `M` fixed and `H` rising, `M_h / H` crosses the
spending threshold and then the liquidity floor (8.5, 8.6). *Delay:* the entry rate.
*Owner:* MON 8. *Damper:* D11, D12 (the pool recycles what the dead leave), D31 (wants fall
first, by design), and a pool grant sized to the expected entry (8.6). *Undamped:* a closed
economy whose households grow past the thresholds. *Condition:* `M_h / H` at least the
spending threshold; the measure is in the Snapshot (13.4). *Test:* MON-S3. *Status:* hand.

**MON-A7. A top-up peg.** *Kind:* amplifier (for EXT). A class topped up to `Level` has a
budget fixed in money per tick, so its demand is unit-elastic in price (spending constant).
A seller facing it has the neutral case `e = 1` of `spec-stability.md` 6.1. *Owner:* MON
10.2, EXT. *Damper:* none in MON; the price bounds. *Condition:* the seller side is EXT's.
*Test:* in EXT. *Status:* hand.

**New scenarios proposed (the register owner numbers them):**

- **S13 (existing, now defined; 17.6).** A faucet and no sink on pack `one`:
  `overlay/faucet.toml`. *Measure:* the boundary balance of the flow's reason, `M`, and the
  index of the staple's price over four years. *Pass:* the boundary balance equals
  `-Total` exactly and equals `-Amount` times the household-ticks of the class, `M` equals
  `M_init` plus it, and the audit holds (FR-MON-026); the price drift is **report-only**:
  STY-S05 sets no band, because neither a source nor a model fixes the direction
  (`spec-stylised-facts.md`; MON-A1).
- **MON-S1. A rate levy bounds a hoarding class; a pair drifts by its difference.** *Setup:*
  a grant and a rate levy on one class and a grant and a fixed levy pair, as 9.2 and 9.3.
  *Measure:* the class's money against `m*`, the net boundary balance of the pair.
  *Pass:* the money converges below `m*` and never above it; the net of the pair equals
  the difference of the fixed amounts times the ticks, exactly. *Loops:* MON-A2, A3.
- **MON-S2. A levy below the upkeep line.** *Setup:* a fixed levy equal to a household's
  upkeep cost without an exemption, and the same with `Exempt`. *Measure:* `Unfed`,
  `Privation`, the exits (`EntryLimited`). *Pass:* with the exemption no household goes
  unfed because of the levy; without, the warning fires and the unfed households are
  counted. *Loop:* MON-A4.
- **MON-S3. Growing households with and without a pool grant.** *Setup:* 8.6. *Measure:*
  `M_h / H` against `ReserveTicks * U_c` and `U_c`, the unfed households and the pool.
  *Pass:* without the grant `M_h / H` crosses the thresholds at the derived ticks (500 and
  3 500 for the example), with it stays above the spending threshold. *Loop:* MON-A6.
- **MON-S4. The scale seam.** *Setup:* `SetFlowScales` between steps. *Pass:* only that
  flow's amounts change, from the next step; nothing changes without a call; the audit
  holds. *Loop:* MON-A1 (the PLC seam).

The checklist 9.6 neighbours: the time constant of A3 against `T` and the year; the
carry of D5 against `Per` (a rate per year spread over `TicksPerYear` ticks has no pulse);
none else. The register's section 10 "loops no spec dampens" MAY drop its L30 row: the
sink and the measure exist (the cost is that a pack must configure a sink).

## 17. Worked examples

All numbers are placeholders chosen to be easy to follow, and were checked by `flows.py`
(section 18).

### 17.1 One household, a grant and a holding charge

A household of weight 2 holds 1 000. Two flows act on its role: a rate grant `stipend`
(`Amount` 300 `Per` tick) and a rate levy `charge` (`RateBP` 5 000 `Per` year, `Exempt` 500),
with 365 ticks a year and `S` = 10 000.

| Tick | Grant `w*Amount` | Balance | Base `bal - w*Exempt` | Levy numerator `base*5000*10^4 + carry` | Levy due | Carry | Balance after |
|------|-----------------|---------|-----------------------|----------------------------------------|----------|-------|---------------|
| 0 | 600 | 1 600 | 600 | 3.0e10 | 0 (3.0e10 / 3.65e10) | 3.0e10 | 1 600 |
| 1 | 600 | 2 200 | 1 200 | 9.0e10 | 2 (7.3e10 taken) | 1.7e10 | 2 198 |
| 2 | 600 | 2 798 | 1 798 | 1.069e11 | 2 | 3.39e10 | 2 796 |
| 3 | 600 | 3 396 | 2 396 | 1.537e11 | 4 | 0.77e10 | 3 392 |

(The denominator is `P*10^8` = 3.65e10; the grant is paid whole every tick because `P` is 1
and the carry stays 0.) The grant is a faucet of 600 a tick under the reason `stipend`; the
levy a sink under `charge`; both appear in `BoundaryFlow` for the household's region.

### 17.2 A small rate spread by the carry

A grant of 7 a year over 365 ticks. Account 0 starts with carry 0 and is first paid in tick
53 (`7 * 53 = 371 >= 365`); account 100 starts with carry `100 * 10^4` (that is 100/365 of a
unit) and is first paid in tick 38 (`100 + 7*38 = 366`). Each is paid 7 in the year. A
cohort of weight 3 is paid 21 and one of weight 40 is paid 280 (model, `flows.out`).

### 17.3 A pair that drifts

50 households of a role `dependent` get a grant `stipend` of 300 a tick; 450 of a role
`payer` pay a fixed levy `due` of 33 a tick; the recipients spend their stipend on goods the
payers sell. Grants 15 000 a tick, levies 14 850. After 400 ticks: boundary `stipend`
-6 000 000, boundary `due` +5 940 000, net -60 000; money in the economy 1 410 000
against 1 350 000. The warning `pair_unbalanced` is not raised at the tolerance of 1 000
basis points (the totals differ by 1%); the drift is in the boundary balances.

### 17.4 A counterparty topped up and swept

Level 5 000 for a pass-through account. After a tick of trading its balance is 3 800: the
next tick's top-up is 1 200. After another tick it is 6 300: the sweep (a rate levy of
10 000 basis points a tick with `Exempt` 5 000) collects 1 300 (model, `flows.out`).

### 17.5 Growing households in a closed economy

8.6 is the example: thresholds crossed at ticks 500 and 3 500 without a grant; with a
constant pool grant of 3 000 a tick for one entrant a tick at a starting money of 3 000, the
spending threshold holds.

### 17.6 The faucet overlay of scenario S13

The overlay to pack `one` (`spec-reference-chain.md` 4.4.1), in the file layout of
`spec-config.md` 6.2 (a module is a table whose presence switches it on; entities are
tables keyed by ID). The amount is a placeholder, 2% of the basket `B_r` of 1 000:

```toml
[modules.money_supply]

[modules.money_supply.flow.stipend]
kind = "grant"
target = { kind = "households", role = "grower", region = "north" }
amount = 20
per = "tick"
```

The variant that adds a sink, `overlay/faucet-sunk.toml`, appends a rate levy on the same
role (`kind = "levy"`, `rate_bp = 3650`, `per = "year"`, `exempt = 3000`), whose bound for
a hoarder is `3000 + 20/rho` with `rho = 3650/(10000*365)` = 1/1000, that is 23 000
(9.3). Neither overlay is a variant of its own; both are layered after `one`.

## 18. What was run

One small model, `doc/specs/models/mon/flows.py`, run in Docker (`python:3.12-slim`), with
its output in `flows.out`. It is a prototype of the rules of 5 and 6 on toy accounts with
Python integers, so the 128-bit intermediates are not exercised. It shows:

18.1. **The carry** (6.1, 6.2): the first-payment ticks 53 and 38 and the totals of 7, 21
and 280 of 17.2, and the exact conservation `accounts + boundaries = 0` after every tick
of every run (an assertion in the script).

18.2. **The holding charge** (9.3): 50 households that never spend, a grant of 300 a tick
and a rate levy of 5 000 basis points a year over 365 ticks: 28 016 per household at tick
100, 86 098 at 365, 189 143 at 1 460, 214 705 at 2 920, 218 641 at 6 000, against the
derived bound `300 * 730 = 219 000`; with an exemption of 1 000 the figure at tick 6 000 is
219 641 against 220 000.

18.3. **A pair** (9.2, 17.3): the boundary balances -6 000 000 and +5 940 000, net -60 000.

18.4. **The traces** of 17.1 and 17.4.

What it does **not** show: any behaviour of the library, the price response to a faucet
(MON-A1 is derived by hand, not run), the behaviour of households that spend, or any
interaction with the demand, welfare or spawner rules. The numbers are placeholders and
the model is a check of arithmetic and conservation, not a result.

## 19. Notes settled

| Note | What this spec does |
|------|---------------------|
| N-078 | **Applied, close.** The recurring named income transfer is a grant (4.2), targeted at a role with an empty `Recipes` list (7.3), counted as income by default (7.2); the paired sink or return flow is a levy, with a validation check of the nominal pair (9.2, 12.2) and a bound for the rate form (9.3). The model of DEM (a stipend of 300 to 50 households, no sink) is the unbalanced case (9.1). The `class_without_income` warning of DEM needs its edit (handoff) |
| N-021 | **Applied for money, leave open for goods.** A new faucet or sink is a registry row (4.5, 10.3); this spec adds rows for flows declared in data and states the rule for later specs (10.1, FR-MON-043). The goods side (`Raw` needing a real source, the external sector) stays with EXT and RES |
| N-046 | **Leave open; MON's entries delivered** (16): MON-A1 to A7 for the register owner to merge. Other specs' entries remain owed |
| N-054 | **Closed note, read.** The answer stands: the mean welfare ratio is 1 with upkeep only and `1 + W/U` with wants (8.2). MON adds nothing to it and does not change the default `TargetRatio` of 15 000: a balanced pair moves money between classes and leaves the mean unchanged; an unbalanced faucet raises the mean only while `M` grows (9.1). "Revisit when DEM, MON or EXT exist" is answered for MON: no change |
| N-084 | **Applied in part (S13), leave open.** The overlay of S13 is defined (17.6, 16). The dynamic parts of X1 and X2, the coarse-tick leg of S12 and the rest wait for REG, TRN, MER, INF and STY |
| N-012, N-045, N-065 | **Leave open for OBS.** MON proposes three events, three log keys and two metrics (13) and a counter for its clamps (`MoneyFlowLimited`) |
| N-016 | **Applied.** Every new `Config` field has a row proposed for the parameter-owner table (handoff, b) and no parameter is unnamed |
| N-017, N-027 | **Applied.** Names are neutral (grant, levy, flow, pool, household); no default macro constant is stated (11.1) |
| N-020 | **Applied.** The state MON adds to a checkpoint is in 14 |
| N-075 | **Leave open for CFG.** `ErrFlowShape` and `ErrReasonCollision` need a root mapping |
| N-009 | **Applied.** `spec-world.md` fixes the position: P1.7, after relief (P1.6) and before production (5.1) |

## 20. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Every number is
a bound or a placeholder owned by this spec; none is calibrated (N-015, N-027).

### Functional

| ID | Requirement |
|----|-------------|
| FR-MON-001 | With `Config.Modules.MoneySupply` nil, the ledger's registry MUST equal the built-in rows, no flow state MUST be kept or saved, no flow event MUST be emitted, and `M` MUST equal `M_init` at every tick boundary. |
| FR-MON-002 | After construction, the money supply MUST change only through a transfer under a reason of the ledger registry, and in a world with only this module's flows only through a flow's reason. |
| FR-MON-003 | Each flow MUST declare a ledger reason equal to its ID, of kind faucet for a grant and sink for a levy, money only; the registry MUST be closed after construction. |
| FR-MON-004 | A grant MUST be a transfer from the flow's boundary account to the recipient with the flow's reason, and a levy a transfer from the recipient to the boundary; a transfer the ledger refuses MUST leave the ledger unchanged. |
| FR-MON-005 | A flow MUST have exactly one shape: a grant `Amount` or `Level`, a levy `Amount` or `RateBP`; any other combination of the fields of 4.1.1 MUST be rejected with `ErrFlowShape`. |
| FR-MON-006 | The due of a recipient MUST depend only on its weight, its balance (rate levy, top-up), its carry, the flow's fields, the flow's scale and the tick, and MUST NOT depend on any other account, price, total or count. |
| FR-MON-007 | A household class MUST be every open Agent account whose agent has the target role and, if a region is stated, that region, external agents included and estate pseudo-agents excluded, and an amount MUST be multiplied by the agent's weight. |
| FR-MON-008 | A pool class MUST be the Estate account of the stated region, or of every region when none is stated, with weight 1; a stores class MUST be every open Agent account whose agent owns a store of the stated class in the stated region, with an amount multiplied by the number of its matching stores and not by its weight; a treasury class MUST be the Treasury account of the stated governed jurisdiction, or of every governed jurisdiction in ascending ID when none is stated, with weight 1 (4.3.4, 4.3.5). |
| FR-MON-009 | Class membership MUST be evaluated each tick from the agent's current role and region. |
| FR-MON-010 | A flow MUST be active in tick `t` exactly when `t >= Start`, `Until` is 0 or `t < Until`, and it has not ended, and an inactive flow MUST change no state. |
| FR-MON-011 | The cumulative amount a flow applies MUST NOT exceed `Limit` when `Limit` is above 0, nor `MaxFlowTotal`; a flow that reaches its effective limit MUST end, and `MoneyFlowEnded` MUST be emitted once with the cause. |
| FR-MON-012 | A rate grant and a fixed levy MUST pay `floor((w*Amount*S + carry) / (P*10^4))` and keep the remainder as the carry, so the total over any run of ticks with unchanged `w` and `S` equals `floor((n*w*Amount*S + c0) / (P*10^4))`. |
| FR-MON-013 | A recipient's initial carry MUST be `(ID mod P) * 10^4` for a rate grant or a fixed levy, with `ID` the account ID (the region index for a pool, the jurisdiction's index in ascending `JurisdictionID` for a treasury; a stores class is by account ID as households are), and 0 for a rate levy. |
| FR-MON-014 | `Per` `year` MUST use the `TicksPerYear` derived from the calendar, so a change of calendar changes the per-tick amount, and `Per` `tick` MUST NOT be rescaled by a change of calendar. |
| FR-MON-015 | A rate levy MUST take `floor((base*RateBP*S + carry) / (P*10^8))` with `base = max(0, balance - w*Exempt)`, at most `base`, keeping the remainder as the carry. |
| FR-MON-016 | A top-up MUST pay `max(0, w*Level*S/10^4 - balance)`, truncated, and keep no carry. |
| FR-MON-017 | Every product of the step MUST use checked arithmetic with a 128-bit intermediate, MUST NOT wrap, and a quotient above `MaxMoney` MUST be treated as `MaxMoney` in the limits of FR-MON-020 to FR-MON-022. |
| FR-MON-018 | In each tick the grants MUST be applied before the levies, each in ascending flow ID, within a flow regions in region order and accounts in ascending account ID. |
| FR-MON-019 | The flows MUST run in phase 1 after relief is paid and before production begins, so a grant is available to the same tick's `Trade`. |
| FR-MON-020 | A levy MUST collect at most the balance above `w*Exempt` and MUST NOT take a balance below it; the uncollected part MUST be forgiven, not carried, and counted as `Short`. |
| FR-MON-021 | A grant MUST NOT take a balance above `MaxMoney`; the part not issued MUST be counted as `Headroom`. |
| FR-MON-022 | An amount cut by the limit MUST be counted as `Limit`, and the carry MUST advance by the due and not by what was applied. |
| FR-MON-023 | A flow's scale MUST default to 10 000, MUST be changed only by `SetFlowScales` between steps or by a policy actor at P0.3a within 0 to `MaxFlowScale`, MUST apply from the next P1.7, MUST be refused with `ErrModuleOff` (module off), `ErrUnknownFlow` (a flow that is not declared) or `ErrOutOfBounds`, and `MoneyFlowScaled` MUST be emitted for each changed flow with its `Cause`. |
| FR-MON-024 | The engine MUST NOT change a flow's scale or any other field of a flow by itself; no rule of this module MUST read the money supply, a price, an index or a count of accounts of a class. |
| FR-MON-025 | A grant to households that is not `NotIncome` MUST be added to the recipient's `income_t` for the tick and excluded from the observed wage, a grant with `NotIncome` MUST NOT, and a levy MUST NOT be netted from income. |
| FR-MON-026 | After every operation, `M` MUST equal `M_init` plus the grants applied minus the levies applied (plus the other faucets and sinks), and for each flow the boundary balance of its reason MUST equal `-Total` for a grant and `+Total` for a levy, with exact equality. |
| FR-MON-027 | The audit (`World.Audit`, `WithAudit`) MUST check FR-MON-026 and MUST report the first flow whose boundary balance and total disagree. |
| FR-MON-028 | A world with no active flow MUST keep `M` equal to `M_init` at every tick boundary, for every seed. |
| FR-MON-029 | `BoundaryFlow` MUST report the amounts of a flow under its ID as the `Reason`, for the region of the non-boundary side, with the kind faucet or sink. |
| FR-MON-030 | `MoneyFlowLimited` MUST be emitted once per flow and region per tick in which any due was not applied, with `Due`, `Applied`, `Short`, `Headroom` and `Limit`, and MUST NOT be emitted when everything was applied. |
| FR-MON-031 | Events MUST contain no map and no pointer, fixed-point values only, and MUST be emitted after the step's transfers. |
| FR-MON-032 | The step MUST NOT log at `Info`, `Warn` or `Error`; its `Debug` lines MUST be guarded by `Enabled`, have constant messages and use the keys `flow`, `region`, `due`, `applied`, `cause`. |
| FR-MON-033 | With the default logger and observer the step MUST build no event and allocate nothing beyond scratch held by the world. |
| FR-MON-034 | A field outside its bound, an unknown `Kind` or `Per`, `Until` not above `Start` when set, or more than `MaxMoneyFlows` flows MUST be rejected with `ErrOutOfBounds`. |
| FR-MON-035 | Two flows with one ID MUST be rejected with `ErrDuplicateID`. |
| FR-MON-036 | A flow ID equal to a built-in reason, or to a reason a later spec declares, MUST be rejected with `ErrReasonCollision`. |
| FR-MON-037 | A target naming an undefined role or region MUST be rejected with `ErrUnknownRole` or `ErrUnknownRegion`. |
| FR-MON-038 | A pool target while the estates module is off, and a stores target while the storage module is off, MUST be rejected with `ErrModuleDependency`; a stores target naming an unconfigured class MUST be `ErrUnknownStoreClass`, a treasury target naming an undefined jurisdiction `ErrUnknownJurisdiction` and one with no seat `ErrNotGoverned`. |
| FR-MON-039 | The warnings `no_flows`, `faucet_without_sink`, `pair_unbalanced`, `levy_below_upkeep`, `class_empty`, `grant_to_pool_no_outlet` and `store_levy_below_upkeep` MUST NOT fail validation, and each MUST name its path. |
| FR-MON-040 | The state of 14 MUST be saved in the order given, a restored world MUST apply the same payments as an uninterrupted one, and a carry of a closed account MUST NOT be saved. |
| FR-MON-041 | The module MUST draw no random number, and the draws made at agent creation (`spec-agents.md` 3.5.2) MUST NOT depend on whether it is on. |
| FR-MON-042 | The Snapshot MUST carry `M_h`, `M_p`, `M_c`, `M_t` per region, `M` and `H`, the money boundary balance per reason sorted by reason, and each flow's `Total`, `Scale` and whether it ended, as plain data; the boundary balances MUST be in the order of `spec-observability.md` 4.4.1.2. |
| FR-MON-043 | A faucet or sink reason MUST have a Boundary account on its boundary side, so a counterparty whose balance is created or destroyed outside conservation is a Boundary account; a transfer under such a reason between two non-boundary accounts MUST be refused with `ErrReasonMismatch` (`spec-ledger.md` FR-LED-004). |
| FR-MON-044 | A flow MUST NOT apply anything during the construction phase, and the construction bucket MUST contain only `initial` money. |
| FR-MON-045 | A flow whose scale is 0, or that has ended, MUST apply nothing and emit no `BoundaryFlow` entry. |
| FR-MON-046 | In every tick, the change of money held by the households, public, contract and treasury sectors MUST equal the grants applied minus the levies applied plus the other faucets minus the other sinks, a grant to a household class MUST change only the households sector, a grant to a pool only the public sector and a grant to a treasury only the treasury sector (`spec-ledger.md` FR-LED-025). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-MON-001 | With the default logger and observer, the step of a reference-chain run MUST allocate nothing beyond scratch held by the world and reused across ticks (observability-and-logging.md 8.2). |
| NFR-MON-002 | The cost of the step per tick SHOULD be proportional to the number of active flows times the accounts of their classes, and SHOULD have a benchmark with `b.ReportAllocs()`. |
| NFR-MON-003 | Validation of the module SHOULD have a fuzz test with random flow tables (never panics; a valid table gives the same effective configuration and hash built in Go or read from a file; an invalid one returns a sentinel). |

## 21. Open questions

1. **Conversion of a commodity into money (3.4 B), and the external account.** A fixed-price
   buyer is EXT's. It needs a bid whose cash is a boundary, which `spec-clearing.md` 3
   cannot admit (a boundary cannot reserve). *Next step:* **Answered by `spec-external-sector.md` 4.1:** a pass-through account of the kind Outside with six built-in reasons, topped up before admission and swept to zero after the markets clear; `spec-clearing.md` changes by one wording only; a boundary-backed admission was rejected. MON's top-up and sweep shapes were not used as the mechanism (a flow tops up to a level in phase 1 and sweeps in the next phase 1). The target kind `"external"` is released and defines nothing.
2. **A money-yielding recipe (3.4 A).** Rejected as a configured wage. *Next step:*
   reopen only if the fixed-price buyer of EXT proves unable to express money from extraction; then
   PRO and WEL would need an output of money and an income term.
3. **A fixed total for a household class.** A grant is per household, so its total follows
   the class's households (MON-A5). A fixed yearly total shared among the class would
   decouple the issue from population but needs a share-allocation rule with a
   deterministic remainder (largest remainder by weight, then ID). *Next step:* add as a
   `Basis` field when a pack needs it; none does yet.
4. **An endowment tied to entry.** Money for newcomers beyond the pool (a faucet per
   birth or per immigrant) reacts to entry, which is POP's and SPN's. MON offers the pool
   grant (8.6). *Next step:* POP decides whether it needs a per-entry faucet and, if so,
   whether it is a flow with a trigger or its own reason.
5. **Should a levy net out of income?** A household levied 20% looks as rich to the
   target-income rule as before (7.2). **Answered in part:** `spec-fiscal.md` 12 nets a tax,
   a duty and a toll; a levy of this module stays un-netted, and `spec-welfare-and-wages.md`
   2.1 states the rule per flow. *Next step:* reopen only if a pack needs a levy netted.
6. **A mirror levy.** A levy that retires exactly what a named grant issued would make a
   pair exactly neutral. Rejected for now (2, position 9). *Next step:* revisit if MON-S1
   shows packs routinely drifting by a nominal mismatch.
7. **Lumps and a pay day.** Payments are spread (D5). A tax day or an annual stipend is a
   pulse that would synchronise every recipient (L22). *Next step:* if STY needs a
   seasonal pulse, add a `Phase` by account ID, as the occupation review does
   (`spec-spawner.md` 6.2).
8. **The pool target and the estates module.** Whether the pool and its account exist
   without the module (`spec-estates.md` layer note). Answered (N-219): the pool's account
   exists without the module (it is core, `spec-estates.md` layer line, `spec-regions.md` 3.6.2);
   a pool target needs the module, and `ErrModuleDependency` stays.
9. **A cap on a flow per region.** `Limit` is shared by the regions of a flow (4.4.2). A
   per-region limit would need `Limit` as a table.
10. **Is the boundary account per reason created lazily or at construction?** This spec
    asks for construction (4.5.1). *Next step:* LED confirms (account IDs of regions and
    agents shift by the number of declared reasons).
11. **Interest, saving and credit** are excluded (9.5); they carry their own entries.
12. **Velocity and a price index** are IND's. The measures of 3.1.3 are the inputs.
    **Answered:** velocity is `Turnover` and money per household is `Sectors`
    (`spec-indicators.md` 9 rows 8 and 7); a price index is `PriceIndex`.

## 22. Where this lives in the code

| Concern | Home |
|---------|------|
| `MoneySupply`, `MoneyFlow`, `FlowTarget` and the `Modules.MoneySupply` pointer | root package, `config.go` (types) and `defaults.go`; bounds in `bounds.go` |
| Validation, the sentinels `ErrFlowShape`, `ErrReasonCollision` and the warnings | root package, `validate.go`, `errors.go` |
| The pure rules: the carry (6.1), the rate levy (6.3), the top-up, the limits (5.4) | `internal/moneyflow` (stdlib only, over primitive slices, converted inline at the call, architecture.md 2.9); name provisional, and it MAY live in `internal/ledger` helpers if small |
| The step (5) and the flow state | root package, `world.go` (phase 1) and `moneysupply.go` |
| `SetFlowScales` | root package, `integration.go` (`spec-integration.md` 5.1); the flow state is in `moneysupply.go` |
| The events `MoneyFlowLimited`, `MoneyFlowEnded`, `MoneyFlowScaled` | root package, `event.go` |
| The declared reasons | `internal/ledger` registry, built at construction from the flow list |
| Log keys `flow`, `due`, `applied` | `internal/logkeys` (key strings only) |

```go
// internal/moneyflow: stdlib only, no root types. A sketch, not final.
// Spread returns the payment of one tick and the new carry. num = w*amount*scale + carry.
func Spread(w, amount, scale, carry, den int64) (pay, newCarry int64, ok bool)

// RateLevy returns the amount due from a balance above the exemption.
func RateLevy(base, rateBP, scale, carry, den int64) (due, newCarry int64, ok bool)

// TopUp returns the amount that raises balance to w*level*scale/10_000.
func TopUp(w, level, scale, balance int64) (due int64, ok bool)
```
