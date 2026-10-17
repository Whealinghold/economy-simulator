# Spec: Agents, households, upkeep and the Strategy

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-AGT-NNN` and
`NFR-AGT-NNN`, which tests cite (testing.md section 4). They are in section 20.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs `spec-ledger.md` (accounts, `Move`, the `consumption` and `disposal`
reasons), `spec-production.md` (jobs, `MaxExecutions`, `UnitCost`),
`spec-clearing.md` (orders, outcomes, the market report), `spec-beliefs.md`
(beliefs, `Draw`, `Update`, the price window), `spec-scale.md` (weights),
`spec-estates.md` (what a dead agent leaves, relief),
`spec-welfare-and-wages.md` (the welfare ratio and the target-income rule) and, for
discretionary demand, the optional module `spec-demand.md`. The spawner that removes,
reduces and re-roles agents is `spec-spawner.md`.

**Every number in sections 7, 12 and 16 is a placeholder.** Nothing has been run,
bazaarBot has not been run, and no acceptance band exists (notes N-015, N-027).
No default here is claimed to be stable. Section 12 lists them, so that none is
mistaken for a result.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

An **agent** is one household, or a cohort of identical households
(`spec-scale.md` 2). Every other spec says what a household earns, makes or owns;
this one says what a household *is*, what it must consume to stay alive, and how
it decides what to do each tick. It covers:

- the agent's state, what lives in the ledger and what in the agent, and what an
  agent is born with and what happens when it is removed (section 3);
- the order of events in a tick as an agent sees it (section 4);
- **upkeep**, the goods a household must consume every tick, and what failing to
  get them means (section 5);
- the `Strategy` extension point, its exact methods, what each receives and
  returns, and what the world checks (section 6);
- the default strategy: work, recipe choice, inventory targets, the budget and the
  rule that saves when wealth is low, order sizing and belief revision
  (section 7);
- inventory capacity and discards (section 8);
- cohorts (section 9), the player and other external agents (section 10), and how
  liquidity, relief and the target-income rule fit together (section 11).

Out of scope: when an agent is removed, replaced or changes role (`spec-spawner.md`, which
owns the failure ladder, the removal events and the role-change mechanics), and how many
are born (`spec-population.md`); what a want beyond subsistence is (`spec-demand.md`,
which adds a fourth tier to 7.4 when the module is on); the rule that sets a price belief (`spec-beliefs.md`); recipes
(`spec-production.md`); the tick phases (`spec-world.md`); the integration API
that adds a player (`spec-integration.md`); merchants, who need a strategy of their
own (`spec-merchants.md`); and the register of feedback loops
(`spec-stability.md`), to which section 18 contributes.

## 2. Positions taken

The design pressures of the brief, each with a position and the reason. Later
sections give the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is an agent, and what lives where? | A household or cohort: a role, a region, a weight, and a ledger account that holds its money and every good (tools too). Beliefs, job, residues, welfare history, trainee state, hunger and a strategy reference live in the agent. Upkeep, targets and the tradable set are derived, never saved | One owner per fact. The ledger already conserves what the agent holds (`spec-ledger.md` 2.1), so the agent holds no balance of its own |
| 2 | What is upkeep, and what is starvation? | A per-household list of (commodity, quantity) per tick, consumed in the consumption phase as a boundary flow (`consumption`). Each commodity is eaten independently, up to what is held. **Fed** households are the least over the list of consumed over per-household need, and **unfed** are the rest. Starvation is only a signal here (`Unfed`, `Hunger`); the consequences are `spec-spawner.md` and `spec-population.md` | A shortage must be a number the population specs can act on. Independent consumption wastes nothing: a household with the staple but no fuel still eats |
| 3 | What is the `Strategy`? | Three methods, the most the extension rules allow: `Work` (which recipe to start, before production), `Trade` (orders and discards, after production) and `Revise` (the new belief for one order outcome). Each gets a plain-data view and the world's random source, and returns plain data. The strategy never mutates the world | Architecture.md 4.2 limits an interface to three methods. Production runs before orders (`spec-production.md` 5.2), so the decision to work and the decision to trade are taken at different times; the outcome of a market arrives after clearing, so belief revision is a third call |
| 4 | Inventory target and capacity? | A **target** is the stock an agent wants to hold at the start of the next tick: a number of ticks of use (`CoverTicks`), plus the inputs of its next job. For a commodity it only makes, the target is a number of ticks of its output (`OutputCoverTicks`). **No inventory capacity unless the storage module is on** (section 8) | Capacity needs a rule for what to discard and a storage cost, which are `spec-storage.md`. Production never refuses output (`spec-production.md` 5.3.2). Overstock is limited by spoilage, by the cash-limited buffer rule, and by a production halt when output piles up (7.1) |
| 5 | Liquidity and saving? | A hard constraint: no order beyond available cash, no borrowing (with the credit module off; a loan of the module raises the balance before it is spent, 7.4.1), no negative balance. A **budget stack** of three tiers: upkeep first, job inputs second, buffers third. Only the third tier is subject to the saving rule: below a reserve of `ReserveTicks` of upkeep cost nothing is spent on buffers, and above it spending is a concave, bounded function of the excess | A household must be able to eat before it does anything else, and a producer must be able to buy inputs, or the model would starve productive households. A buffer is the only spending that can safely be cut when wealth is low |
| 6 | Cohorts? | Quantities are `w` times the household amounts; a limit price is never scaled; one belief and one draw per order; the work decision is proportional (`ceil(w * share)`), not all or nothing | `spec-scale.md` 4.4; the proportional response removes the all-at-once lockstep of the target-income rule for a cohort (N-014) |
| 7 | The player? | An ordinary agent of weight 1, flagged **external**. Its strategy is either a consumer-written `Strategy`, called synchronously and not allowed to block, or the library's **directed** strategy, which reads a **directive** the consumer left in a mailbox between steps. Nothing happens in the middle of a `Step` | `Step` is a synchronous batch (overview 4) and all orders of a tick are placed before any market clears (`spec-clearing.md` 7.2); a callback that waits for a human would stall the game loop. The mailbox is plain state, so a checkpoint saves it |
| 8 | Which events? | One aggregated event, `UpkeepShortfall`, per region and role per tick; new fields on `AgentSpawned`; no per-agent event | Observability rules 4.1 and 5.4.1 |
| 9 | What does the agent contribute to a checkpoint? | Its identity, weight, role, strategy reference, hunger and mailbox; the default strategy keeps no state | ADR 0002 decision 7 |
| 10 | What is price- or money-responsive, and what is a staple's demand elasticity? | Upkeep demand is **price-inelastic by construction**: elasticity 0 while cash suffices and -1 (a fixed expenditure) when cash binds, so never above 1. What does respond: buffer buying (price position and wealth, transient only), the choice among recipes by margin, a production halt on overstock (the damper D24, now specified), and exit through the spawner. The stability condition for the labour loop (demand elasticity above 1) is therefore **not met** by a staple in version 1, and this spec says so (11.6) | `spec-stability.md` 6.1 and 6.2 (STB-L14, L05, L06; N-043): with inelastic flows the belief rules have no interior attractor. Inventing an elastic subsistence demand would be a guess about households; the honest statement is that the anchor is bounds, the halt and exit. `spec-demand.md` adds an elastic demand for discretionary goods and leaves upkeep alone (11.6.1) |
| 11 | What spending exists beyond upkeep? | With the demand module off, none: upkeep, the inputs of jobs (which net out of income) and buffers of the same goods. A closed economy then has a mean welfare ratio of 1 whatever the target (`spec-viability.md` 4.7, N-054), the target-income rule never rests, and the surplus is a glut, **bounded** by the production halt. With `spec-demand.md` on, **wants** are a fourth tier funded from what the saving rule leaves (7.4.3), and a closed economy can reach a ratio above 1 (`spec-demand.md` 7). A non-producing class gets its money from a grant of `spec-money-supply.md` (4, 7.2), or from `spec-external-sector.md` | An identity, not a defect of this spec (11.7) |

## 3. The agent

### 3.1 State

An agent is plain data owned by the world. The fields marked with another spec
are *hosted* here and defined there. The shape is a sketch of the agent's state; the
exported form of an agent is a matter for `spec-integration.md` and `Snapshot`. The
type in the code is unexported (`agent`). The fields that the sketch hosts for a module
(`DeathCarry`, `OwnUse`, `OwnUseJobs`, `Journey`, `Member`, `Stores`, `Trainee`) are
kept in the module's slices by agent slot (`architecture.md` 2.10.1), not in the core
type; the checkpoint still writes them in the agent record (3.2).

```go
// Agent is one household or a cohort of identical households. Plain data.
type Agent struct {
	ID        AgentID  // never reused; the pseudo-agents come first: the estates (spec-estates.md 2.7.6), then the counterparties of the external module, then the treasuries of governed jurisdictions (spec-polity.md 5.2.2); the households follow
	Account   int      // the ledger account of kind Agent; the region lives on the account
	Role      RoleID
	Weight    int64    // households this agent stands for, at least 1 (spec-scale.md 2)
	External  bool     // added through the integration interface (section 10)
	Mortal    bool     // an external agent that the library's rules (a spawner) may remove or reduce (10.6); the consumer's own removal call needs no flag; false for every other agent, where it is not read
	Strategy  int      // index into the world's strategy table (6.6)

	Beliefs   []Belief        // spec-beliefs.md: one per commodity in the tradable set, canonical order
	Job       *Job            // spec-production.md 5.1
	Residues  []int64         // spec-production.md 6.6
	Welfare   WelfareHistory  // spec-welfare-and-wages.md: the ring buffer and the target ratio drawn at creation
	Trainee   *Traineeship    // spec-skill-barriers.md 4.1

	Unfed     int64        // households not fully fed in the last consumption phase (5.4)
	Hunger    int64        // consecutive ticks with Unfed above 0 (5.4)
	Privation int64        // the leaky count of unfed ticks that the failure ladder reads (5.4.1); computed by the world, defined by spec-spawner.md 5.1
	RoleSince int64        // the tick the agent took up its present role or place; a signed tick; defined by spec-spawner.md 5.2.2a, set by the spawner, a forced Relocate and the arrival of a journey
	DeathCarry int64       // the population module's carried remainder of deaths, in [0, 10 000 * TicksPerYear) (spec-population.md 3.8); unused when the module is off
	Directive *Directive   // the mailbox of an external agent (10), usually nil

	OwnUse     bool        // in the own-use mode (spec-subsistence.md 5.3); false when the module is off
	OwnUseJobs int64       // own-use jobs started since the last job of another recipe (spec-subsistence.md 5.5 item 5)
	Journey    *Journey    // a migration journey in progress: destination RegionID and arrival tick (spec-migration.md 8.2), usually nil

	Member     Membership  // the (BodyID, RegionID) of the body the household was admitted to, or none (spec-institutions.md 3.6, 4.2.1, 12.1); none when the module is off
	Stores     []StoreCount // the stores the agent hosts: (class, count) ascending by class (spec-storage.md 15.1); empty when the module is off
}
```

### 3.2 Where each piece of state lives

| Piece | Home | Saved in a checkpoint by |
|-------|------|--------------------------|
| Money, goods, tools, lots, reservations | the ledger account of the agent | `spec-ledger.md` 11 |
| Region | the account (`spec-ledger.md` 3.5); an agent has no region field of its own. It changes only by the operation of `spec-regions.md` 6 (relocation) | `spec-ledger.md` |
| Role, weight, external flag, strategy reference | the agent | this spec, section 15 |
| Price beliefs | the agent | `spec-beliefs.md` 13 |
| Job, residues | the agent | `spec-production.md` 15 |
| Welfare ring buffer, target ratio | the agent | `spec-welfare-and-wages.md` 13 |
| Trainee status | the agent | `spec-skill-barriers.md` |
| Hunger, last unfed count, `Privation`, `RoleSince`, `DeathCarry`, directive, `Mortal` | the agent | this spec, section 15, and `spec-checkpoint.md` 4.2 tag 3 (`RoleSince` is defined by `spec-spawner.md` 5.2.2a and `DeathCarry` by `spec-population.md` 3.8) |
| Trip state (away, shipment) | the transport module | `spec-transport.md` 14 |
| Own-use mode and counter (`OwnUse`, `OwnUseJobs`) | the agent | `spec-subsistence.md` 15 |
| Migration journey (`Journey`) | the agent | `spec-migration.md` 14 |
| Membership of a body (`Member`) and the stores of the agent (`Stores`) | the agent | `spec-institutions.md` 12.1 and `spec-storage.md` 15.1, in the agent record of `spec-checkpoint.md` 4.2 tag 3 |
| Upkeep, tradable set, targets, budgets, views, plans | derived or scratch, never state | not saved |
| Price window, cost anchors | the market, or derived | `spec-beliefs.md` 13, `spec-production.md` 9.3 |

The "home" column names the owner of a piece. A piece that a module owns (`DeathCarry`,
`OwnUse`, `OwnUseJobs`, `Journey`, `Member`, `Stores`, `Trainee`) is kept in that module's
slices by agent slot, and the "saved by" column is unchanged: the checkpoint writes it in the
agent record all the same (N-128).

3.2.1. The agent holds no balance and no quantity. A test that sums the agent's
fields cannot find money (FR-AGT-002).

3.2.2. Tools are goods in the ledger, held by the agent's account. "Tools held" is a
query of the ledger, not a field.

### 3.3 Roles

A **role** is a plain struct in the root package, an entry of `Config.Roles`
(`spec-config.md` 3.1). In a file it is `[role.<id>]`. This spec owns these fields;
`spec-welfare-and-wages.md` and `spec-skill-barriers.md` own others.

| Field | Type | Meaning | Default |
|-------|------|---------|---------|
| `ID` | string | the role's ID | required |
| `Recipes` | list of recipe IDs | the recipes an agent of the role may start (`spec-production.md` 3.3); sorted; empty means the role has no recipes | none |
| `Upkeep` | list of (commodity, quantity) | the upkeep of one household; a non-empty list replaces the world's upkeep (5.2) | none (the world's) |
| `Trades` | list of commodity IDs | commodities the role trades beyond those its upkeep and recipes imply (3.4) | none |
| `Informed` | bool | the agents of the role are shown rumours of other regions' prices in the view passed to `Trade` (owner `spec-information.md` 6.1; needs `Modules.Information`) | false |
| `StartMoney` | money | money per household for an agent created at world construction, and the amount drawn from a pool for a later agent (`spec-estates.md` 2.3) | 10 000 (placeholder) |
| `StartStock` | list of (commodity, quantity) | goods per household an initial agent starts with | none |
| `Strategy` | string | the registered strategy name (6.6) | `"default"` |
| `Label` | string | presentation only | empty |
| `Fixed` | bool | the agents of the role are never moved by the migration module (owner `spec-migration.md` 3.1, 11) | false |
| `Merchant` | the merchant block | nil means the role is not a merchant role (owner `spec-merchants.md` 3.2, 13) | nil |
| `Hireable` | bool | the households of the role may offer themselves for hire (owner `spec-employment.md` 3.1; needs `Modules.Employment`) | false |
| `Employer` | the employer block | nil means the role does not hire (owner `spec-employment.md` 3.2) | nil |
| `Credit` | the credit block | nil means the role does not deal in credit (owner `spec-credit.md` 4.2, 14) | nil |
| `Cover` | the insurance block | nil means the role does not deal in cover (owner `spec-insurance.md` 4.2) | nil |
| `Saver` | the saver block | nil means the role has no savings or investment rule (owner `spec-saving-and-investment.md` 3.2, 10.4) | nil |
| `Needs` | flag ID | the settlement-class flag a region must carry for the role to be entered (owner `spec-settlements.md` 6.4.1) | empty |
| `Ceiling` | limit ID | the settlement-class limit on the households of the role in a region (owner `spec-settlements.md` 6.4.2) | empty |

The role's own-use recipes are not a field of the role: they are the rows of the subsistence module
(`spec-subsistence.md` 3.2), and the **effective own-use list** of a role is derived (3.3.1 there).

**Core and module fields.** `ID`, `Recipes`, `Upkeep`, `Trades`, `StartMoney`, `StartStock`,
`Strategy` and `Label` are the core's: they are in the first code (`spec-config.md` 3.7.2),
with the welfare, skill-barrier and `Closed` fields of the other specs. `Informed`, `Fixed`,
`Merchant`, `Hireable`, `Employer`, `Credit`, `Cover`, `Saver`, `Needs` and `Ceiling` are
owned by the module their row names, are appended to `Role` with that module, and are
written into the configuration hash only when that module is on (`spec-config.md` 8.2.1).
3.3.1. A role with an empty `Recipes` list is legal. Its households earn nothing
from production: they are a trader, a carrier, a household that lives on relief or
on income another spec provides (`spec-estates.md`, `spec-employment.md`,
`spec-land.md`). This answers the question of N-001 at the level of this spec: a
class of non-producing households is a role with no recipes, and nothing else is
needed to represent it. `spec-demand.md` 3.3 confirms it: a class is a role, its upkeep from here and its ladder of
wants from the demand module, and nothing else is needed.

3.3.2. No code branches on a role identifier (architecture.md 9.1).

### 3.4 The tradable set

The **tradable set** of a role is the commodities an agent of the role may bid or ask
for. It is derived from data, not configured one by one:

```
tradable(role) = upkeep commodities (the effective list of 5.2, any region)
               + for each recipe on role.Recipes: its inputs, its required tools,
                 and its non-discard outputs
               + the non-discard outputs of the recipes on the role's effective own-use list
                 (`spec-subsistence.md` 3.6; a household may hold and sell what it makes)
               + role.Trades
               + when the demand module is on: the items of every want in the role's ladder
                 (`spec-demand.md` 3.2.3)
```

sorted in canonical commodity order, with no duplicates. A discard output is never
in it (`spec-production.md` 7.2). This settles what bazaarBot left implicit
(defect 5, `bazaarbot-reference.md` section 11): which goods an agent considers is
explicit, and a commodity outside the set is never traded by that agent, whatever
its stock. It also fixes the word *tradable* for validation: `spec-goods.md` 6.4 and
`spec-currency.md` 3.1 speak of "the cheapest tradable commodity". A commodity is
**tradable** if it is in the tradable set of at least one role. A commodity in no
set is never traded (it may exist only as a discard output).

The set is derived once at construction into the role table (`role.go`), not by a
function called in the tick, as a list in canonical commodity order with a membership
flag per commodity; a module that is on adds its items inside the constructor, before
it returns, and after that the table is a plain value that no code mutates, which is
all "frozen" means here (N-228).

3.4.1. A belief exists for every commodity in the set from the moment the agent is
created (`spec-beliefs.md` 3.1, FR-BEL-001), so the number of beliefs of an agent
never changes, except by a role change, which re-forms the set and the beliefs
(`spec-spawner.md` 4.2).

3.4.2. Trading a good that is neither used nor made, and not listed in `Trades`,
cannot happen. A merchant role lists its goods in `Trades`, or its strategy (a
consumer's, or `spec-merchants.md`'s) draws them from elsewhere; the tradable set
is the limit either way (6.5).

### 3.5 Birth

What an agent is born with is the same for every way of being created. Who creates
it and when is the spawner's (initial agents are derived from groups,
`spec-scale.md` 3.2).

3.5.1. **An initial agent** (created at world construction) of weight `w` receives:

| Item | Value | Ledger reason |
|------|-------|---------------|
| An account of kind Agent in its region | | |
| Money | `w * role.StartMoney` | `initial` (faucet) |
| Goods | `w * quantity` of each `StartStock` item, born in tick 0 if the commodity spoils, at full durability if a durable | `initial` (faucet) |
| Beliefs | one per commodity in the tradable set, from the seed price (`spec-beliefs.md` 8.2), width `BeliefWidthInit`; no randomness | none |
| Job | none, unless a warm start places one (19) | none |
| Welfare history | empty, padded by `DefaultRatio` (`spec-welfare-and-wages.md` 2.2) | none |
| Residues | one draw in [0, 10 000) copied to every residue (`spec-production.md` 6.6.2) | none |
| `Privation`, `Hunger` and unfed | 0, 0 and 0 | none |
| `RoleSince` | the tick of creation (`spec-spawner.md` 5.2.2a) | none |
| Death carry | `offset(ID)` of `spec-population.md` 3.8.3, a function of the ID (no draw) | none |
| Strategy | the registered strategy of the role (6.6) | none |

Initial money and goods are the only way either enters the economy at construction.
The `initial` reason is a faucet in the registry for money and for goods, into any
non-boundary account (`spec-ledger.md` 7.2, FR-LED-038); it is not added here any more
(an earlier draft of this spec asked for it). At construction a creation may state the
lot key (a negative `Born`, a worn durable) for a warm start (`spec-ledger.md` 5.2.4,
FR-LED-043); an agent added during a run takes the defaults of `spec-ledger.md` 5.2.3.

3.5.2. **The draws at creation**, in this order, one agent after another in agent-ID
order: first the target-ratio draw of `spec-welfare-and-wages.md` 5.1, then the
residue draw of `spec-production.md` 6.6.2. Each is made whatever the
configuration, so the random stream does not depend on which modules are on. A spec
that adds a creation draw appends it to this list. This is AGT's answer to the
ordering that N-036 left to `spec-world.md`; WLD adopts it (`spec-world.md` 5.7.1).

3.5.3. **A later agent** (a replacement, a newborn, an immigrant the spawner
creates) receives `min(role.StartMoney * w, pool balance)` from its region's pool
(`spec-estates.md` 2.3), reason `start_money`, and **no goods**, unless the spawner's
entry rule grants some. A later agent therefore has to buy its tool and its inputs
(`spec-skill-barriers.md` 2.3). Its beliefs start from the window mean or the market
price (`spec-beliefs.md` 8.3). The two creation draws of 3.5.2 are made for it too; its death carry starts at `offset(ID)` of `spec-population.md` 3.8.3 and takes no draw.

3.5.4. **An external agent** is born as an initial agent of weight 1, flagged
`External`, with an amount of money its creator states and an `initial` faucet for
it (section 10). The player's starting wealth is therefore a named, counted faucet.
An external agent added during a run takes its beliefs by `spec-beliefs.md` 8.3 (the
window mean or the market price, which before the first step is the seed price of
8.2), and takes the two creation draws of 3.5.2 as any later agent does (3.5.3), so
the stream of a run that adds one is a function of the call
(`spec-integration.md` 4.1.2).

### 3.6 Removal

The *decision* to remove an agent (bankruptcy, death, starvation, migration out) is
the spawner's (`spec-spawner.md` 4.1, 5) and the population module's. The *mechanics* are here,
so that conservation is one rule:

1. a job in progress is abandoned, creating nothing, with the reason `agent_removed`
   (`spec-production.md` 5.4);
2. a trainee relationship ends (`spec-skill-barriers.md` 4.6);
2a. a migration journey in progress is dropped (the households are counted in `Ended` of
   `MigrationReviewed`, `spec-migration.md` 8.3); and for each open delivery contract of which the agent is
   the deliverer, the escrow is returned in full to the payer (`escrow_refund`) and the contract ends
   (`spec-merchants.md` 9.8.2). Step 2a is numbered so that the steps keep their numbers; it runs before the money moves;
2b. each title of the agent moves to the pool of the title's region (`spec-land.md` 6.4); no money moves;
2c. each open hire of the agent ends (`spec-employment.md` 6.5.2): as a worker, its job is abandoned (`agent_removed`, step 1) and the remainder of the escrow returns to the employer; as an employer, for each hire in ascending hire ID, the worker's job is abandoned (`principal_left`), the remainder of the escrow returns to **the agent's own account**, so that the waterfall of 2d sees it, and the hire ends;
2d. for each open claim of the agent as debtor, the waterfall of `spec-credit.md` 9.1: its money pays them, pro rata, before the rest moves to the estate, and what is still owed is written off;
2e. each open claim of the agent as creditor passes to the estate of its region (`spec-credit.md` 9.2);
2f. each open cover of the agent, as insured or as underwriter, ends with no transfer (`spec-insurance.md` 8.6);
2g. each facility of the agent is handled as `spec-infrastructure.md` 7.1 says: a site, route or ground facility passes to the estate of its region and a store facility is ruined; no money moves;
3. all money that remains moves to the region's estate account with the reason `estate_money`;
   one function moves the money and the lots (step 4), the lots in key order and keeping their keys;
   the intake is counted in lots (`GoodsListed`, 14.1), and a refusal with no reservation open is
   an invariant failure, except a balance overflow of the estate account, which is returned as
   `ErrBalanceOverflow` (FR-EST-001, FR-EST-021);
4. all goods move to the estate account with the reason `estate_goods`, and the
   estate lists them at the region's effective reference price, a used durable at that price times its remaining durability over its full durability (`spec-estates.md` 2.1, 2.7.3, FR-EST-008; decision of round 3: the anchor stays the effective reference price, `spec-skill-barriers.md` 2.7 follows it);
5. the account is closed (`spec-ledger.md` 3.3); the agent's ID is never reused;
6. one event, `AgentBankrupted`, is emitted, carrying the agent, region, role, weight and
   `Cause` (`insolvent`, `starved`, `consumer` or `population`; section 14.1). It is the only
   event that removes the agent: bankruptcy, starvation and death are one event with a cause
   (`spec-spawner.md` 13.1.1). The events of the steps above accompany it (`RecipeAbandoned`,
   `TraineeshipEnded`, `ContractSettled`, and those of steps 2b to 2g: `TitleTransferred`,
   `HireEnded`, `LoanEnded`, `LoanPassed`, `CoverEnded` and `FacilityChanged`), in the order of
   `spec-world.md` 6.11 item 8: every event of a removal precedes `AgentBankrupted`, and
   `EstateOpened` follows it.

An agent that is away on a self-carried trip (`spec-transport.md` 5.9.2) is removed the same
way: its account's holdings go to the estate of the account's region, the region it departed
from, and its own shipment is dropped.

Removal runs after clearing and consumption, when no reservation exists. It never
creates or destroys money or goods. Beliefs, residues and the welfare history are
dropped with the agent; the fraction of a unit in a residue was never created
(`spec-production.md` 6.6.4).

3.6.1. **Cohort failure (N-008).** A cohort does not fail by itself. An agent whose
money is 0 and whose goods are gone still exists: it cannot bid, so it goes unfed,
and `Unfed`, `Hunger` and `Privation` say how badly. The ladder (change role, relief, then a
partial reduction of the weight by the unfed share, then removal) is `spec-spawner.md` 5; AGT
supplies the signals it reads (5.4, 5.5) and the effects of a weight change on the agent (3.7).
The mechanics of a reduction are 3.7 and `spec-spawner.md` 4.1; of a role change, 3.8.

### 3.7 What a weight change does to an agent

A weight is changed only by the spawner or the population module, by a ledger
operation with a reason (`spec-scale.md` 4.8). For the agent, from `w` to `w'`:

| State | Effect |
|-------|--------|
| Money and each good | `floor(m * w' / w)`; the difference goes to the estate when positive and is drawn from it when negative (`spec-scale.md` 4.8) |
| Job | `Executions` becomes `min(k, w')` and the pending output scales (`spec-production.md` 5.4) |
| Beliefs, residues, welfare history | unchanged: they are per household or per unit |
| `Hunger` | unchanged |
| `Privation` | unchanged (`spec-spawner.md` 5.1) |
| `DeathCarry` | unchanged (`spec-population.md` 3.8) |
| Want carries | unchanged: per household fractions (`spec-demand.md` 4.8) |
| `Unfed` | recomputed at the next consumption phase |
| Targets, upkeep, budgets | derived from the new weight at once |

3.7.1. Because the ratio, the beliefs and the draws are per household, a cohort
that loses half its households behaves from then on as the smaller cohort would
have: the weight changes the scale of the amounts and nothing else.

### 3.8 What a role change does to an agent

The mechanics are `spec-spawner.md` 4.2 and are the spawner's: the agent keeps its account,
money, welfare history (resized to the new role's effective window: the newest `min(held, T')` records are kept and the rest of the window is padded at read with the new role's `DefaultRatio`, `spec-welfare-and-wages.md` 2.3.3), `Hunger`, `Unfed`, `Privation` and `DeathCarry`; goods outside the new role's tradable
set (3.4) pass to the estate; the job is abandoned with `role_changed`; beliefs are kept for
commodities in both sets; residues reset; the strategy becomes the new role's registered one.
The carries of the demand module follow `spec-demand.md` 4.9. Nothing in this spec changes: the
upkeep, the tradable set, the targets and the budgets are derived from the new role at once.

## 4. The tick as an agent sees it

Tick phases are `spec-world.md`'s. This spec depends on the order below. The table
is a summary: the order of positions inside a phase is `spec-world.md` 6 (N-009
collects what other specs assumed) and it supersedes this table where they differ.

| # | Phase | What happens to an agent |
|---|-------|--------------------------|
| 1 | Start | In this order: the moment is computed (`spec-calendar.md` 6.5); cost anchors are computed from the previous tick's regional wage and prices (`spec-production.md` 9.3, FR-PRO-047, 048); transport arrivals, in shipment-ID order (`spec-transport.md` 5.7), with the reseed of an arriving traveller's beliefs (`spec-merchants.md` 12.2); the settlement of delivery contracts whose shipment arrived, in contract-ID order (`spec-merchants.md` 9.7); the arrivals of migration journeys, in ascending agent ID (`spec-migration.md` 8.4, 10.1); relief is paid (`spec-estates.md` 2.6; starting money moves at creation, in phase 7); money flows (`spec-money-supply.md` 5.1); the outside's walk, with the module on (`spec-external-sector.md` 5.1); an external agent's directive is read |
| 2 | Production | Jobs continue and complete; with the resources module on, the stocks are snapshotted; with the subsistence module on, the flag step (`OwnUse`) is computed for every eligible agent in agent-ID order (`spec-subsistence.md` 5.4); `Work` is called for each agent that is free and did not continue or complete a job; the draws are rationed per endowment (`spec-resources.md` 5.2, 5.3); then the jobs start in the rotation order (`spec-production.md` 5.2, 5.5.1); then `OwnUseChanged` (`spec-subsistence.md` 14.1). An agent that is away is skipped (`spec-transport.md` 5.5.2), and so is an agent on a migration journey (`spec-migration.md` 8.3) |
| 3 | Trade decision | `Trade` is called for each agent in ID order (an agent that is away or on a migration journey is skipped and gets no view); the world executes the plan: discards first, then shipments (`spec-transport.md` 5.3.2), then the orders built from the intents, in plan order; admission reserves cash and stock (`spec-clearing.md` 3). For an agent of an informed role the rumours of its view are built, with their error draws, immediately before its `Trade` (`spec-information.md` 5.5.2, 7.1). After the last agent's plan and before the estates' asks, the contract round matches the delivery offers and takes of the plans (`spec-merchants.md` 9.4), then `ContractRound` and `MerchantsPlanned`. After the last agent and the estates' asks, the counterparties of the external module top up and place their orders (`spec-external-sector.md` 5.2) |
| 4 | Clearing | Every market clears (`spec-clearing.md` 7.2); then the counterparty accounts are swept to zero (`spec-external-sector.md` 5.3) |
| 5 | Outcomes | Price windows are updated; the price record of the information module is appended after the windows are pushed (`spec-information.md` 4.3); `Revise` is called for each outcome in order of region, commodity and admission (`spec-beliefs.md` 5.1) |
| 6 | Consumption | Upkeep is consumed; `Unfed`, `Hunger` and `Privation` are set by the world; with the demand module on, the wants are then consumed (`spec-demand.md` 4.7, reason `want_consumption`); the welfare record of the tick is pushed |
| 7 | Entry and exit | The spawner and population module act (`spec-spawner.md` 10.2: it reads `Privation` and the signals after phase 6); the population module's deaths come before the budget of the entry step is taken and its births last (`spec-population.md` 4.8); removed agents' holdings pass to the estates; then, with the migration module on, the migration pass, once for the world, after the entry step of every region (`spec-migration.md` 10.2) |
| 8 | End | Ageing and spoilage (`spec-ledger.md` 5.3), flow totals, regeneration of the endowments (`spec-resources.md` 4.1), optional audit |

4.0. **Positions that more than one module claims (N-009).** Phase 1 holds, in this order, the
arrivals of transport, the settlement of contracts (which follows the arrivals it depends on),
and the arrivals of migration journeys, all before relief; each is a loop in its own ID order
and the three touch different state (shipments, contracts, journeys). In phase 2 the flag step of
the subsistence module comes after the snapshot of the endowments and before any `Work`. In
phase 3 the contract round comes after the last agent's plan and before the estates' asks.
In phase 7 the migration pass comes last, after the entry step of every region. `spec-world.md` 6
fixes the final order and supersedes this table and this paragraph where they differ.

4.1. Upkeep bought in phase 3 arrives in phase 4 and can be eaten in phase 6 of the
same tick. Money earned in phase 4 can be spent no earlier than phase 3 of the next
tick (`spec-clearing.md` 3.3.1). So an agent that is poor in the morning and sells in
the afternoon cannot eat on the sale that tick: it must already hold the food. This
is why the target of 7.3 holds `CoverTicks` of upkeep ahead, and why `CoverTicks`
has a minimum of 1.

4.2. A job started in phase 2 consumes inputs the agent held at the start of the
tick; purchases for the next job are made in phase 3 of the tick before it starts
(7.3).

## 5. Upkeep

### 5.1 Definition

The **upkeep** of a household is the list of goods it must consume every tick to
stay alive: a list of (commodity, quantity), each quantity a whole number of at least
1 (FR-GDS-003). It is the "daily consumption a role needs to stay alive" of
`spec-currency.md` 5.1, and a "day" is a tick. It is data, never code
(architecture.md 9.1). It is not rescaled when the calendar changes
(`spec-currency.md` 2.4): a pack for a different tick length restates it.

### 5.2 The effective upkeep

The world has a default upkeep list, `Config.World.Upkeep`, which is the reference
household. A role may replace it with `Role.Upkeep` (a non-empty list replaces the
world's whole list). A region may add goods with `Region.ExtraUpkeep` (a cold region
adds fuel, `spec-currency.md` 5.3). The **effective upkeep** of an agent of weight
`w` in region `r` is the merged list: quantities of one commodity add, the result is
in canonical commodity order, and every quantity is multiplied by `w`:

```
U_c = w * (u_c(role or world) + u_c(region.ExtraUpkeep))
```

All sums are checked. The merged list per household (before the multiplication by
`w`, which is the agent's and the world's) is derived once at construction into the
role table (`role.go`), in canonical commodity order, for each role and each region
that has an `ExtraUpkeep`; a region with none reads the role's base list. It is a
table, not a function called in the tick, and a module adds its items inside the
constructor, before it returns (N-228). A world with no upkeep anywhere is legal (a pure trading
toy); validation warns (`no_upkeep`, section 13) because no household can then
starve and the welfare ratio is undefined (`spec-welfare-and-wages.md` 2.2.3).

### 5.3 Consumption

In phase 6, for each agent in ID order and each upkeep commodity `c` in canonical
order:

```
consumed_c = min(held_c, U_c)
```

`held_c` is the agent's whole balance of `c` (no reservation exists in phase 6). The
goods move to the `consumption` boundary, oldest first and then most worn first
(`spec-ledger.md` 5.2.1), with the reason `consumption`, a goods sink owned by this
spec (already in the registry of `spec-ledger.md` 7.2). A shortfall is not an error
and not a warning of the ledger: it is the normal way a poor household shows that it
is poor. Consumption never takes more than is held, so it cannot fail, and a
household that cannot eat what it needs does not stop the others from eating
(bazaarBot defect 6 is not ported: over-consumption is not silently ignored, it is
cut to what exists and then counted, 5.4).

### 5.4 Fed, unfed and hunger

```
Fed   = min over upkeep commodities c of  floor(consumed_c / (U_c / w))         (an integer, 0 to w)
Unfed = w - Fed
Hunger = Hunger + 1 if Unfed > 0, else 0
```

The same phase sets `Privation` (5.4.1).

`U_c / w` is the per-household quantity `u_c`, so `Fed` is the number of whole
households that received their full basket of every good. An agent with no upkeep
has `Fed = w` and is never unfed. A cohort that is short of one good has
`Fed = floor(consumed / u)` for that good, exactly, with no rounding up in favour of
the household.

The consumption of the other goods still happens: a household eats what it has. So
`Fed` is a measure of *needs met in full*, and it does not mean that nothing was
consumed by the unfed (5.3).

5.4.1. **`Privation` and `Hunger`.** The two are related and both are computed by the world in
phase 6 from the same `Unfed`, never by a strategy:

```
Hunger    = Hunger + 1 if Unfed > 0, else 0                    consecutive ticks unfed (this spec)
Privation = Privation + 1 if Unfed > 0, else max(0, Privation - 1)     a leaky count (spec-spawner.md 5.1)
```

`Hunger` is the short memory of the household and of its strategy: it resets on one fed tick,
so it drives the survival rule (7.1.3) and the urgency bid (7.5.3), which should react to
the last tick. `Privation` is the long memory the failure ladder needs: a household fed one
tick in seven resets `Hunger` each time and would never be removed, while `Privation` climbs
(`spec-spawner.md` 5.1, model 16.2). `Hunger <= Privation` at all times (both rise together, `Hunger` falls to 0 first); neither is reset by a
weight change or a role change (3.7, 3.8). `Privation`
is agent state hosted here and defined there; it is not in the strategy's view (7.1 and 7.5 read
`Hunger`), and the spawner's view carries it. A strategy that wants it has `Hunger`.

### 5.5 Starvation as a signal

AGT defines no death and no loss of weight. It defines what the population and
spawner specs read:

| Signal | Meaning |
|--------|---------|
| `Unfed` | households not fully fed in the last tick, of `Weight` |
| `Hunger` | consecutive ticks with `Unfed` above 0 |
| `Privation` | the leaky count of unfed ticks (5.4.1); read by the failure ladder |
| `FedBP` | `Fed * 10 000 / Weight`, basis points, derived |
| the shortfall by commodity | `U_c - consumed_c` for each upkeep commodity, available to a query and to the Debug log |

The use is the spawner's and is decided: the households that leave through `starvation` or
`insolvent` removal are a function of `Unfed` once `Privation` has reached `RemovePrivation`, so a
cohort is reduced by the share it cannot feed before it is removed (`spec-spawner.md` 5.3, N-008).
Population rates other than the ladder's are `spec-population.md`'s.

### 5.6 The reference basket and the welfare ratio

5.6.1. The reference basket `B_r` of a region (`spec-currency.md` 5.1) is the cost of
the *baseline* upkeep, `Config.World.Upkeep` plus the region's `ExtraUpkeep`, at the
region's effective reference prices (`spec-goods.md` 6.3). A role with its own
`Upkeep` pays its own cost; `B_r` is a calibration anchor and the engine never reads
it (`spec-currency.md` 5.4). The configuration states `B_r` and validation checks
that it equals the cost of the baseline within `BasketTolerance`, in basis points of
`B_r` (`ErrBasketMismatch`). This names the tolerance that `spec-currency.md` 5.3
left without a name or default (N-016).

5.6.2. The `cost_t` of the welfare ratio (`spec-welfare-and-wages.md` 2.1) is the cost
of the agent's effective upkeep `U_c` at the prices of its region in tick `t`. The
welfare module reads it from this spec's derived upkeep and does not recompute it.

5.6.3. A household of a role whose upkeep is cheaper or dearer than the baseline has
a welfare ratio on its own upkeep; two roles with different upkeep are compared by
ratio, never by money (`spec-currency.md` 5.6).

### 5.7 Hooks

- **Trainees.** A trainee's upkeep is supplied by its trainer
  (`spec-skill-barriers.md` 4.2). The hook: a trainee's consumption in phase 6 is
  drawn from the trainer's account, at the trainee's own position of P6.1
  (`spec-world.md` 6.8, ruling C10), and the trainer's use in 7.3 includes its
  trainees' upkeep. The reason is `consumption`, from the trainer's account.
  `spec-skill-barriers.md` owns the rest; this is open question 8.
- **Wants beyond upkeep** are `spec-demand.md`'s, a module. When it is on, the tier structure of
  7.4 has a fourth tier for them (7.4.3), the want items join the tradable set (3.4), the want
  units join `Keep` and `Target` (7.3) and the bid quantity (7.5.4), and the wants are consumed
  after upkeep in phase 6 (4). The `Purpose` of such a bid is `Other` (7.3.6).
- **Agents in transit.** In version 1 a traveller stays in its origin region, away and in no
  market, until arrival (`spec-transport.md` 5.5.2); `NoRegion` and the `depart` and `arrive`
  hook are reserved and unused (`spec-regions.md` 6.5.3), and a shipment ties the account to its
  region by the refusal `open_shipment` (`spec-regions.md` 6.5.2). A migration journey is the
  same shape: the account stays in its region, the agent is passed to neither `Work` nor `Trade`,
  and the refusal `on_journey` ties it (`spec-migration.md` 8; `Journey` of 3.1). The refusals
  that bind an account to its region are `open_shipment` (TRN), `on_journey` (MIG), `open_contract`
  (MER) and `in_traineeship` (SKL).

## 6. The Strategy extension point

### 6.1 The contract

`Strategy` is one of the four sanctioned extension interfaces (architecture.md 4.3:
it "decides an agent's work (`Work`), its bids and asks (`Trade`) and its belief
updates (`Revise`); the world checks what each returns"). It is defined in the root
package, where it is consumed (4.1). It ships with a default implementation and a
compile-time assertion (4.5):

```go
var _ Strategy = DefaultStrategy{}
```

6.1.1. **Pure decisions.** A method reads its arguments and returns plain data. It
does not call back into the world, hold a pointer to anything in it, write to the
view it was given, or keep the random source after returning. The world does not
share live state with a strategy (6.4).

6.1.2. **Randomness (architecture.md 7.7).** Every method receives the world's
`*rand.Rand`. An implementation **MUST NOT** create its own source, use a global one,
or keep the one it was given after the call returns. The doc comment on `Strategy`
states this, because a consumer's implementation that breaks it breaks
reproducibility (architecture.md 7.4). The order of calls is fixed (section 4), so
the stream a strategy sees is a function of the world's state. This holds with the information module: the error draws of a view are taken from the same source immediately before that agent's `Trade`, in a fixed order that is a function of the configuration and the state (`spec-information.md` 5.5.2), and the ration of `spec-resources.md` 5.3 and the walk of `spec-external-sector.md` 5.1 take theirs at fixed points of the tick.

6.1.3. **Determinism of the strategy itself.** A strategy **MUST** be a function of
its view and the random source. It must not read the clock, a global, a map's
iteration order or the network. This is the consumer's responsibility and the world
cannot check it; a test with a recording strategy can.

6.1.4. **State.** The default strategy keeps none. A consumer's that keeps state must
be stateless or implement the optional save and load pair of ADR 0002 decision 7
(open question 4 there).

6.1.5. **Panics** in a strategy are not recovered; they propagate to the caller of
`Step`, as an observer's do (architecture.md 12.5).

6.1.6. **Pseudo-agents.** An estate, a counterparty of the external module and a treasury
have no strategy and are never passed to one (`spec-estates.md` 2.7.4,
`spec-external-sector.md` 4.2, `spec-polity.md` 5.3.3).

### 6.2 The methods

```go
// Strategy decides what an agent does. Plain data in, plain data out.
// rng is the world's random source; an implementation must not keep it.
type Strategy interface {
	// Work is called once per tick, before production, for each agent that has no
	// job in progress and did not continue or complete one this tick. It returns
	// the recipe to start, or a request for none.
	Work(v *AgentView, rng *rand.Rand) JobRequest

	// Trade is called once per tick for each agent, after production. It appends
	// orders and discards to the plan, in the order they are to be submitted.
	Trade(v *AgentView, rng *rand.Rand, p *TradePlan)

	// Revise is called once per admitted order, after its market has cleared. It
	// returns the agent's new belief for the commodity; returning o.Belief keeps it.
	Revise(o *OutcomeView, rng *rand.Rand) Belief
}
```

Three methods, and no fourth (architecture.md 4.2). The reasons for each:

- **`Work` and `Trade` are separate** because production runs before orders in the
  tick (`spec-production.md` 5.2) and the outputs of a job that completes this tick
  are available to the orders of this tick. A single "decide" call before production
  could not know the stock the orders will be sized from; a single call after it
  could not start a job in time.
- **`Revise` is separate** because the outcome of an order exists only after the
  market clears (`spec-clearing.md` 7.3), and because the world, not the strategy,
  takes the gate draw in a fixed global order (`spec-beliefs.md` 5.1).
- **No `Init`.** What an agent is born with is the world's (3.5) and the seed of a
  belief is `spec-beliefs.md` 8.
- **No `Consume`.** Upkeep is mechanical (section 5). Discretionary consumption is a
  want and belongs to `spec-demand.md`: its demand is part of `Trade` (the fourth tier of
  7.4.3) and the world consumes the units it bought in phase 6 (`spec-demand.md` 4.7), so
  the interface still has three methods.

### 6.3 The types

```go
type Side int
const (
	Bid Side = iota // buy
	Ask             // sell
)

// String returns "bid" or "ask", the `side` value of the log keys and of the golden
// rendering of events (spec-observability.md 4.2).
func (s Side) String() string

// Purpose says why a bid is placed. It is used only to classify spending for the
// welfare ratio (6.4.4); the market never sees it.
type Purpose int
const (
	PurposeOther Purpose = iota
	PurposeUpkeep
	PurposeInput
	PurposeTool
)

// Intent is one order a strategy wants placed.
type Intent struct {
	Commodity CommodityID
	Side      Side
	Quantity  Quantity // whole units, a total for the agent (scaled by its weight)
	Limit     Price    // per unit, never scaled by the weight
	Target    Quantity // units the agent wants to hold; 0 means none (the belief input of spec-beliefs.md 4.4)
	Reserve   Quantity // units of this commodity set aside for this tick's upkeep (see Stock in OutcomeView)
	UnitCost  Price    // cost anchor passed to the belief update; 0 means unknown
	Purpose   Purpose
}

// Discard is a quantity the strategy wants disposed of.
type Discard struct {
	Commodity CommodityID
	Quantity  Quantity
}

// TradePlan collects what Trade returns. The world gives the strategy a plan
// that is empty and reused, so a strategy appends and allocates nothing.
type TradePlan struct {
	Intents   []Intent
	Discards  []Discard
	Shipments []ShipmentRequest // spec-transport.md 5.3.1; the default strategy appends none
	Offers    []DeliveryOffer   // spec-merchants.md 9.2; the default strategy appends none
	Takes     []DeliveryTake    // spec-merchants.md 9.2; the default strategy appends none
	Notes     PlanNotes         // spec-merchants.md 15.1.2: counters a strategy fills; zero for the default; influence nothing
	Borrow    []CreditRequest   // spec-credit.md 5.1; the default appends one only for a role with a Borrow block
	Lend      []CreditOffer     // spec-credit.md 5.1; the default appends one only for a role with a Lend block
	Cover     []LaneCover       // spec-insurance.md 5.1; the default appends them only for a role with an Insure block
	Store     []StoreCover      // spec-insurance.md 5.1; at most one
	Offer     []CoverOffer      // spec-insurance.md 5.1; at most one, for a role with an Underwrite block (beside the merchants' Offers: another type)
}

// ShipmentRequest and Line are defined by spec-transport.md 5.3.1:
//
//	type ShipmentRequest struct {
//		Mode      ModeID
//		To        RegionID
//		Consignee AgentID // ignored when ToSelf is set
//		ToSelf    bool    // self-carried only
//		Lines     []Line  // empty when ToSelf
//	}
//	type Line struct {
//		Commodity CommodityID
//		Quantity  Quantity
//	}

// DeliveryOffer, DeliveryTake and PlanNotes are defined by spec-merchants.md 9.2 and 15.1.2:
// an offer is an indivisible quantity for delivery to another region at a least delivered
// price, a take asks for goods to be delivered to the taker's region at a most price, and
// the notes are fixed counters of the strategy's decisions. Offers and takes are matched
// in the contract round after the last agent's plan, not by Clearing (phase 3, section 4).

// CreditRequest and CreditOffer are defined by spec-credit.md 5.1, LaneCover, StoreCover and
// CoverOffer by spec-insurance.md 5.1, and LabourOffer and HireRequest by spec-employment.md 4.1:
// they are collected in the plan or the request and admitted and matched in the rounds of those
// modules (6.5), not by Clearing.

// JobRequest names the recipe to start. Executions is a maximum; 0 means the most
// the agent can run. The zero value, with NoJob set, requests nothing.
type JobRequest struct {
	NoJob      bool
	Recipe     RecipeID
	Executions int64

	Offer LabourOffer // a worker's offer of its households for hire (spec-employment.md 4.1); Households 0 means none
	Hire  HireRequest // an employer's vacancies (spec-employment.md 4.1); Positions 0 means none
}
```

`Belief` is the value of `spec-beliefs.md` 3.1, exported with its two fields,
`Centre` (a `Price`) and `Width` (an `int64`, basis points). `RecipeID` is a named value type like `CommodityID` and
`RoleID` (architecture.md 3.3); it is added to the glossary.

### 6.4 The views

6.4.1. **`AgentView`.** A struct of plain values. The world fills a single instance
held in scratch, reused for every agent, and passes a pointer to it. It is valid for
the call only: a strategy **MUST NOT** keep the pointer or any slice in it, and
**MUST NOT** write to it. The slices are scratch copies, so a write cannot reach the
world even if a strategy tries; the world does not rely on that (6.5). It holds:

| Group | Fields |
|-------|--------|
| Identity | `Tick`, the `Moment` of `spec-calendar.md` 4, `Agent`, `Role`, `Region`, `Weight`, `External` |
| Money | `Money`, the agent's available cash |
| State | `Hunger`, `Unfed`, the welfare ratio and whether it exists, the agent's target ratio, the working ticks in its window and its **effective window** `T = max(WelfareWindow, WindowCycles * L)`, `L` the longest recipe, own-use recipe or merchant trip of its role (`spec-welfare-and-wages.md` 2.3.3; for the floor of 7.1.3), `StartsIn` (6.4.3) |
| Upkeep | the effective upkeep (5.2) as per-household quantities |
| Job | whether one is active, its recipe, `Remaining`, `Executions` |
| Per commodity in the tradable set | `Stock` (the whole balance; no reservation exists yet), `Expiring` (units that spoil at the end of this tick), the belief and its bounds, the effective reference price, `UnitCost` (0 if the role cannot make it, or if the region has no observed wage yet; it counts labour, 7.3.6), and the market report: price, age, best unmatched bid and ask with flags, and the price window's mean, low, high and the position of the price in it (`spec-beliefs.md` 7) |
| Per recipe of the role | the recipe's inputs, outputs, tools, duration, the effective yield `E` for this region and season (with the resources module on it includes the abundance of the tick's snapshot, `spec-resources.md` 4.7.3; the factors are site, shock, technology, the trainee or hired factor and abundance, `spec-production.md` 6.3), `MaxExecutions` (`spec-production.md` 4.2: `k0` of the world, the executions the inputs, the tools, the ration, the capacity and the limits of the facility classes the recipe needs, builds or repairs, and with the storage module on the free room allow; with the resources module on also bounded by `floor(stock / draw)` of the snapshot, reason `no_endowment`) and the reason if it is 0 (the master order of `spec-production.md` 4.2); with the modules on, also `Needs`, `Builds` and `Repairs` (`spec-infrastructure.md` 4.1.1), for a `Builds` or `Repairs` recipe `Standing` (the first refusal other than `no_input` and `no_tool` that would apply were the inputs and tools held), and for a recipe on an employer's hire list the hired yield (`spec-employment.md` 9.1) |
| Own-use (only with the subsistence module on) | `OwnUse`, `OwnUseJobs`, `Cover` and `Secure` (`spec-subsistence.md` 5.2, 5.4), and a mark on each own-use recipe in the per-recipe group, so that a consumer's strategy does not recompute them. The own-use recipes of the role's effective own-use list appear in the per-recipe group beside the role's recipes |
| Merchant (only in the view of an agent whose role has a merchant block) | `Params`, `Modes`, `Goods` (mass, bulk, shelf life, durability and the age of the oldest lot held) and `Outbound` (`spec-merchants.md` 4.1) |
| Employment (only with the module on, and only for a role that is `Hireable` or has an employer block) | `Hireable`; for an employer the hire list, `MaxHires`, `WorkerShareBP`, `HireYieldBP`, the executions and number of hires it has open (`Hired`) and the hires open in its region (`Open`) (`spec-employment.md` 9.1) |
| Credit (only in the view of an agent whose role has a credit block) | `Lend`, `Borrow`, `MinPrincipal`, `UpkeepCost`, `Debt`, `Owed`, `Overdue`, `NextDue`, `Room`, `Lent`, `Receivable`, `ScaleBP` (`spec-credit.md` 10.1) |
| Cover (only in the view of an agent whose role has a cover block) | `Underwrite`, `Insure`, `MinSum`, `MaxCoverRequests`, `Open`, `Capacity`, `Money`, `Owed`, `Policies`, `LaneLoss`, `ValueCapBP`, `UnitValue` (`spec-insurance.md` 10.1) |
| Room (only with the storage module on, and only for a limited agent) | `Limited`, `Room`, `Load`, `Pledge`, `Free` (pairs of mass and bulk), the agent's `Stores`, and `Mass` and `Bulk` per commodity of the tradable set (`spec-storage.md` 6.1.1). No other agent's room is visible |
| Saving (only for an agent whose role has a saver block, in the `Work` and the `Trade` view) | `Block`, `SafetyTicks`, `HurdleSpreadBP`, `CommitBandBP`, `Drawdown`, `Facilities`, `Classes` (`spec-saving-and-investment.md` 5.2); the per-recipe group carries `Needs`, `Builds`, `Repairs` and `Standing` |
| Facilities (only with the infrastructure module on) | in the `Work` view, the agent's own facilities in its region, ascending ID, at most `MaxFacilitiesPerAgent` (`FacilityView`, `spec-infrastructure.md` 9.1); the `Trade` view carries them only through the saving group above |
| Tunables | the values of section 12, the role's `MinWorkShare` (read here, not in section 12), and the fifteen belief parameters that `beliefs.Params` holds, of `spec-beliefs.md` 11, as `beliefs.Params` names them (the `Belief` prefix of the first eleven is dropped: `Step`, `AdjustProb`, `Significance`, `EvidenceShare`, `StockLow`, `StockHigh`, `MaxAge`, `WidthInit`, `WidthMin`, `WidthMax`, `WidthStep`, `CostFloor`, `CostCeiling`, `RefBandLow`, `RefBandHigh`): plain data, filled by the world from the beliefs parameters and converted inline at the call; not `MaxPrice` and not `PriceWindow`. The default `Revise` reads them from the view because a strategy has no handle to the world (FR-AGT-050, section 7): the alternatives fail, for the world to call `beliefs.Update` itself would take the update out of the strategy's hands, and fields on `DefaultStrategy` would put a result-affecting value where the configuration hash cannot see it |
| Mailbox | the pending `Directive`, if any (section 10) |
| Transport (only with the transport module on) | `Inbound` (per commodity on the way to the agent: the total and the earliest arrival tick, fixed at the start of phase 3) and `Routes` (per other region and mode: whether a route exists, the travel time, and for a hired mode the haul units per carrying unit) (`spec-transport.md` 9.1) and `Outbound` (the mirror of `Inbound`: per destination region and commodity with units on the way, the total quantity of the shipments the agent is the shipper of and the earliest arrival, fixed at the start of phase 3; `spec-merchants.md` 4.1). An agent that is away or on a migration journey is never passed a view |
| Endowments (only with the resources module on) | for the agent's own region only: each endowment that its role's recipes draw on, sorted by resource: `Resource`, `Stock`, `Capacity`, `AbundanceBP` (`spec-resources.md` 5.5.1) and, with the tenure module on, `Posted` (the rent in force per unit), `ShareBP`, `Entitled` and `Closed` (`spec-land.md` 4.6.1). No view carries another region's endowment or another holder's share |
| Rumours (only with the information module on, and only in the view passed to `Trade` of an agent of a role with `Informed`) | `Rumours`: the rumours of `spec-information.md` 5, sorted by (`Region`, `Commodity`), a scratch slice valid for the call only. The `Work` view and the `OutcomeView` carry none |

The Go shape of the core groups (Identity, Money, State, Upkeep, Job, per commodity, per
recipe, Tunables, Mailbox) is below. The module groups of the table are **appended with
their module** (architecture.md 2.10.5) and are not in the core shape. The type names are
exported; a field a module group adds is the module's. The shape is a sketch of the groups
and the fields of the table: the names of the element types that are not given elsewhere
(`UpkeepRow`, `RecipeLine`, `RecipeOutput`, `RecipeTool`, `Tunables`) are the code's to settle.

```go
type AgentView struct {
	// Identity
	Tick   Moment
	Agent  AgentID
	Role   RoleID
	Region RegionID
	Weight int64
	External bool
	// Money
	Money int64 // available cash
	// State
	Hunger, Unfed int64
	Ratio         int64 // the welfare ratio, basis points; read with HasRatio
	HasRatio      bool
	TargetRatio   int64
	WorkingTicks  int64 // working ticks in the effective window
	Window        int64 // the effective window T
	StartsIn      int64 // 6.4.3
	MinWorkShare  int64 // the role's floor; a Role field owned by spec-welfare-and-wages.md, read here and not in section 12
	// Upkeep, Job
	Upkeep []UpkeepRow // per household; 5.2
	HasJob bool
	Recipe RecipeID
	Remaining, Executions int64

	Commodities []CommodityView // the tradable set, canonical order
	Recipes     []RecipeView    // the role's recipes, ascending RecipeID
	Tunables    Tunables        // section 12, plus the fifteen belief parameters (BeliefParams)
	HasDirective bool
	Directive    Directive // by value, its slices copied into the scratch (6.4.2)

	scratch *viewScratch // unexported; nil in a view built by hand
}

type CommodityView struct {
	Commodity CommodityID
	Stock     int64 // the whole balance; no reservation exists yet
	Expiring  int64 // units that spoil at the end of this tick
	Belief    Belief
	Bounds    BeliefBounds
	Ref       Price // the effective reference price
	Tool      bool  // a required tool of some recipe of the configuration; derived once from the whole configuration; read by 7.3.5
	UnitCost  Price // 0 unknown; the filler zeroes it for a commodity the role has no recipe making (FR-AGT-048)
	Market    MarketReport
	Window    PriceWindowView
}

type MarketReport struct {
	Price, Age, BestBid, BestAsk int64
	HasBestBid, HasBestAsk       bool
}

type PriceWindowView struct {
	Mean, Low, High, Position int64
	HasWindow                 bool
}

type RecipeView struct {
	Recipe        RecipeID
	Duration      int64
	Inputs        []RecipeLine
	Outputs       []RecipeOutput // with CostShare and Discard
	Tools         []RecipeTool   // with WearPerUse and Durability
	Effective     int64          // the yield E
	MaxExecutions int64
	Reason        Reason // why MaxExecutions is 0
}
```

The slices of a view are scratch copies valid for the call only; the unexported `scratch`
field is set by the world's filler (a `viewScratch` sized once at construction from the
largest role) and is nil in a view a consumer builds by hand, for which the default strategy
allocates its small working arrays (NFR-AGT-001 binds the world's views, not a hand-built one).
The default strategy never writes to a view. `Tool` is derived from the whole configuration
(it is true for a commodity that is a required tool of any recipe, whatever the role), because no
other group of the view carries the fact; 7.3.5 offers a tool for sale only by a role that makes it.

No view carries a settlement class, a flag or a limit: a household reads its own state, and the
effect of a class on a role is the spawner's and the population module's, not a strategy's
(`spec-settlements.md` 2, FR-SET-018).

6.4.2. The view carries *configuration* (recipes, upkeep) by value as plain data.
Nothing in it points into live state. The reading of architecture.md 12.4 ("every
value returned across the API boundary is a deep copy") that this relies on is
open question 3: a view passed to an extension for one synchronous call is not a
value returned to a consumer who may marshal it elsewhere.

6.4.3. **`StartsIn`** is the number of ticks until the agent can next start a job:
1 if it has no job after the production phase of this tick, otherwise
`Remaining + 1` (`spec-production.md` 5.2). A purchase placed in phase 3 of tick `t`
arrives in phase 4, before the production phase of tick `t + 1`, so inputs for a job
that can start at `t + StartsIn` must be bought no later than tick
`t + StartsIn - 1`.

6.4.4. **`OutcomeView`** is what `Revise` receives for one admitted order:

| Field | Meaning |
|-------|---------|
| `Agent`, `Commodity`, `Side` | which order |
| `Limit`, `Quantity`, `Filled` | the order and its outcome (`spec-clearing.md` 7.3, FR-CLR-025) |
| `Belief`, `Bounds` | the current belief and `Floor`, `Ceiling` (`spec-beliefs.md` 3.2) |
| `Market` | the market report: price, age, best unmatched bid and ask with flags |
| `Stock` | the units held after the trades, less the units reserved for this tick's upkeep, not below 0 |
| `Target`, `UnitCost` | the values the intent carried |
| `Draw` | the one `uint64` the world drew for this outcome (`spec-beliefs.md` 4.5) |
| Belief parameters | the fifteen belief parameters of the Tunables row of 6.4.1, the same group, so that `DefaultStrategy.Revise` can call `beliefs.Update` from the view alone |

`Stock` is the stock the agent will have after eating, which is what a target means.
A buyer holding exactly its target after this tick's upkeep therefore has a stock
ratio of 10 000, in the neutral band of `spec-beliefs.md` 5.5.

### 6.5 What the world checks

The world protects itself from a strategy by checking its return values, the same
way it checks a `Clearing`'s trades (`spec-clearing.md` 8.2).

6.5.1. **`Work`.** A request for a recipe not on the role's list is refused with the
reason `not_on_role` (a refusal reason of `spec-production.md` 4.2, section 22); the
agent starts nothing, and the world logs it at `Debug`. Production checks the role's
list **before any other check** (FR-PRO-048), so a request off the list never reaches
the input, tool or yield checks. Any other refusal is production's, with its reasons: the whole order of the reasons is
`spec-production.md` 4.2's, and `not_permitted`, `no_facility`, `no_title`, `rent_funds` and
`no_room` are among them. An `Offer` or a `Hire` of the request is checked by the labour round
(`spec-employment.md` 5.2) and a refusal leaves the rest of the request alone.

6.5.2. **`Trade`.**

- Discards are executed first, in plan order, as `Move`s to the `disposal` boundary
  (`spec-production.md` 7.3). A discard of more than the agent holds is refused whole
  and logged at `Debug`.
- Shipments are then executed, in plan order, each refused whole with a `reason` if it fails
  a check (`spec-transport.md` 5.3.2, 5.4); the default strategy appends none, and with the
  transport module off every request is refused `unknown_mode`.
- Delivery offers and takes (`Offers`, `Takes`) are collected, not executed here: they are
  admitted and matched in the contract round after the last agent's plan (`spec-merchants.md`
  9.3, 9.4); the default strategy appends none, and with `Modules.Merchants` absent each is
  refused `contracts_off`. `Notes` is read by the world only to build `MerchantsPlanned`.
- Credit requests and offers (`Borrow`, `Lend`) are collected, not executed here: they are
  admitted and matched in the credit round at P3.3b after the last agent's plan
  (`spec-credit.md` 5.2, 6.1); the default appends one only for a role with a `Borrow` or a
  `Lend` block, and with `Modules.Credit` absent each is refused `credit_off`, a `Debug` line.
- Cover requests and offers (`Cover`, `Store`, `Offer`) are collected in the same way and
  admitted and matched in the cover round at P3.3c (`spec-insurance.md` 5.2, 7.1); with
  `Modules.Insurance` absent each is refused `insurance_off`, a `Debug` line.
- Intents are then turned into orders and submitted in plan order. An intent for a
  commodity outside the agent's tradable set is rejected with the reason
  `not_tradable`, an `OrderRejected` event, and never reaches clearing. In the order
  of checks of `spec-clearing.md` 3.2 the check comes **after** `unknown_commodity`
  (a commodity the configuration does not know is that reason, not this one), and it
  does not apply to a pseudo-agent, which has no role and no tradable set. A second
  intent for a commodity is rejected as `duplicate_order`. An intent to sell or buy a
  commodity under a `sell` or `buy` right of a body that the agent is not a member of is
  rejected with `not_permitted` (`spec-institutions.md` 5.6.2, FR-ORG-024); the world checks it
  while it builds the orders, **after** `not_tradable` and before the market's checks, and it
  does not apply to a pseudo-agent, which holds no membership (FR-AGT-085). Every other
  rejection is the market's (`spec-clearing.md` 3.2): `invalid_price`, `invalid_quantity`,
  `insufficient_funds`, `insufficient_stock`, `no_room` (with the storage module on),
  `overflow`, `holding_limit` and `receipt_limit`.
- An intent's `Target` and `Reserve` above `MaxQuantity`, and a `Purpose` outside the
  enumeration, are treated as 0 and `PurposeOther`, with a `Debug` log. They are
  inputs to a belief rule, not orders, so they do not reject the order. A negative `Target`,
  `Reserve` or `UnitCost` is treated as 0 in the same way, as one above `MaxQuantity` is (FR-AGT-048).
- The world clamps no quantity to a budget: a strategy that sizes an order beyond its budget
  gets it rejected whole (`spec-clearing.md` 3.2), not reduced. A legal price is the one thing
  admission may replace: a limit above a cap or below a floor in force is replaced by it
  (`spec-clearing.md` 3.2.1), and that is counted in `PriceControlled`, never silent.

6.5.3. **`Revise`.** The returned belief is clamped to the invariant and the bounds
before it is stored, and the clamps are counted in `BeliefsUpdated`
(`spec-beliefs.md` 10.3, FR-BEL-026). A strategy that returns `o.Belief` leaves the
belief as it was, apart from the re-clamp of `spec-beliefs.md` 3.3.

6.5.4. **A strategy cannot change the world** except through these return values:
the view holds copies (6.4.1), and no method receives a handle to the world.

### 6.6 Registration, identity and the hash

A strategy is code, and code cannot be in a configuration file. It is registered by
name:

```go
func WithStrategy(name string, s Strategy) Option
```

`WithStrategy` is additive. The name `"default"` is built in and is the library's
`DefaultStrategy`; the name `"directed"` is built in beside it and is
`Directed{Fallback: DefaultStrategy{}}` (10.3), the strategy that an empty strategy name
of `AddAgent` means (`spec-integration.md` 4.2.2). The `directed` policy of a policy actor
(`spec-policy.md` 8.4) is not this strategy: it is the mailbox of an actor's lever settings,
and has no `Sticky`. The world holds a strategy **directed**
when the registered value's dynamic type is `Directed`; a consumer wrapper that holds one
in a field is not, and `Direct` returns `ErrNotDirected` for it. `Role.Strategy` (3.3) is a name; an unknown name is rejected at
construction with `ErrUnknownStrategy`. The *names* are data: they are in the
effective configuration and so in its hash (`spec-config.md` 8.1), and a restore with
a different set of names fails the hash check. The *code* behind a name cannot be
hashed; a consumer who changes a strategy's behaviour under the same name gets a run
that no longer matches its checkpoints, and this is the consumer's to manage
(open question 4). An external agent names its strategy when it is added (section 10).
The standard names beyond `"default"` live in the package `strategy/`, which exists once there is a
second strategy (architecture.md 2.6): `strategy.Merchant`, registered as `"merchant"` by
`strategy.Options()` (`spec-merchants.md` 3.1.2). The root package cannot import `strategy/`,
so a configuration that names `"merchant"` needs the one Go line `economysim.NewWorld(cfg,
strategy.Options()...)`. `strategy/` is created when the first code lands.

### 6.7 What a consumer's strategy may do

A strategy may ignore beliefs, the tradable set's targets and the default rules, and
price and size however it likes, within 6.5. It may call the default strategy's
methods: `DefaultStrategy` is exported and stateless, and a consumer's type holds one
in a named field and delegates (architecture.md 4.7 forbids embedding in a public
struct). The pure rules of the saving module are exported beside it, `SavingTarget(v *AgentView)`
and `ChooseProject(v *AgentView)` (`spec-saving-and-investment.md` 4.3.3, FR-SAV-042), as are
`DefaultCover` and `DefaultOffer` of the insurance module (`spec-insurance.md` 10.4). That is the wrapper for the default belief rule that `spec-beliefs.md` 10.5
asks this spec to decide: `DefaultStrategy.Revise`. The merchant strategy of
`spec-merchants.md` is a second implementation (`strategy.Merchant`, `strategy/merchant.go`), written the same way. A
merchant in the own-use mode plans no trip and posts no take (`spec-merchants.md` 6.4).

## 7. The default strategy

`DefaultStrategy` is a zero-value struct with no fields. Its tunables arrive in the
view (6.4.1), so a consumer can call it with a view of its own making. Every rule
below uses integer arithmetic with checked multiplication (architecture.md 3.5) and
makes no choice by a commodity, recipe or role identifier. Every number is a
tunable of section 12 or derived from data.

### 7.1 `Work`

The default decides whether the agent works at all, and if so which recipe.

7.1.1. **No recipes, no job.** A role with an empty `Recipes` list and an empty effective own-use
list requests nothing. A role with an empty `Recipes` list and a non-empty effective own-use list
(`spec-subsistence.md` 3.3.3) requests nothing outside the own-use mode and an own-use job in it
(7.1.9). A pure merchant role (no recipe) takes an empty row in the subsistence table
(`spec-subsistence.md` 3.2.1) and so requests nothing.

7.1.2. **Whether to work.** If the ratio is absent (the agent has no upkeep, 5.6), the
agent works whenever a recipe is worth starting (7.1.4). If the ratio is present, the
target-income rule of `spec-welfare-and-wages.md` section 5 applies, as a **work share** `s` in basis
points:

```
floors:     if Hunger > 0 or Money < ReserveMoney (7.4):                        s = 10 000   (7.1.3)
            if working ticks in the window < MinWorkShare * T / 10 000:          s = 10 000   (7.1.3)
otherwise:  T_lo = TargetRatio of the agent
            T_hi = T_lo * (10 000 + RestBand) / 10 000
            s = 10 000                              if ratio < T_lo
            s = 0                                   if ratio >= T_hi
            s = (T_hi - ratio) * 10 000 / (T_hi - T_lo)   between
```

`T` is the agent's **effective window**, `max(WelfareWindow, WindowCycles * L)`, `L` the
longest `Duration` on its role's list or its effective own-use list, or the longest trip of a
merchant role (`spec-welfare-and-wages.md` 2.3.3), and the working
ticks are those of its welfare record (the view carries both, 6.4.1).

If `RestBand` is 0, `T_hi = T_lo` and `s` is 10 000 below the target and 0 at or
above it, which is the rule as `spec-welfare-and-wages.md` states it. The band is the
anti-chattering band that section 5.1 there asks the strategy for: an agent keeps
working until its ratio is `RestBand` above its target, then rests, and a cohort
answers in proportion (9.2). If the ratio is absent (no upkeep), `s` is 10 000.

7.1.3. **The survival rule and the floor on working ticks.** Two rules override the
work share, and both set it to 10 000.

- **The survival rule.** Survival comes before the target. An agent that was unfed
  last tick, or whose money is below its reserve, works whatever its ratio says. This is
  the reading of N-028's "a producer also works while its household's needs are short".
  `spec-stability.md` 6.1 has run it and finds that this reading is **not a damper by
  itself**: labour supply stays unit-elastic, so the loop of STB-L14 is stable only if
  demand elasticity exceeds 1, and a staple's does not (11.6). It stays because it is
  what a household does, not because it stabilises the market (STB-D26, "not a damper of
  L14").
- **The floor `MinWorkShare` (STB-D13, adopted).** If the working ticks in the agent's
  effective window `T` are fewer than `MinWorkShare * T / 10 000`, the agent works
  whatever its ratio says, and with `MinWorkShare` 0 the floor is off. The parameter and
  its reasons are `spec-welfare-and-wages.md` 5.3 (default 4 000 basis points, a
  placeholder); this spec applies it. It does not remove the instability of L14: it sets
  where a pinned market stops.

In the own-use mode (`spec-subsistence.md` 5.5) the work share is 10 000 and the request is
the own-use recipe, with the retry (7.1.9).

The floor acts silently, so by FR-STB-013 and N-065 it is **counted**. The world
computes, for each agent that is free when `Work` would be called, whether the floor
changed the outcome: the floor condition holds and the share the ratio alone gives is
below 10 000. It does this with a pure function of the agent's welfare record
(`internal/welfare`), not by asking the strategy, because a consumer's strategy may not
report it; the households of such agents are totalled per region and role for the
`Snapshot` (14.6) and for the bounded `Warn` "work floor binding" that
`spec-welfare-and-wages.md` 11.3 specifies. The snapshot count (`RoleRow.HeldByFloor`,
`spec-observability.md` 5.3.4) is evaluated over the agents that have no job in progress on
the record after P6.3 (`spec-world.md` 6.8), so it may differ from the count of the `Warn`,
which is evaluated when `Work` is called (`spec-observability.md` 5.3.4, `HeldByFloor`).
The world's count of the floor (and of the halt and the rest, 7.1.5) and the default `Work`
both read one **work report**, a pure function of the view (the work share, whether the floor
changed it, whether the agent rests, the recipes the halt skips and the reason a request would
be declined with), so the count and the default cannot drift apart and a consumer's strategy
cannot hide them; `JobRequest` is unchanged.

7.1.4. **Which recipe.** For each recipe `r` of the role for which the view reports
`MaxExecutions >= 1`, with effective yield `E_r`:

```
value_r = sum over non-discard outputs o:  floor(q_o * E_r * Centre_o / 10 000)
cost_r  = sum over inputs i:   q_i * Centre_i
        + sum over tools t:    ceil(WearPerUse_t * Centre_t / Durability_t)
        + sum over the recipe's Draws (res, d):  d * Posted_res   (the rent in force of the view, `spec-land.md` 4.6.2;
                                                  0 with the tenure module off or a rent of 0)
net_r   = value_r - cost_r                       (per execution, in minor units)
```

`Centre` is the agent's own belief for the commodity: it values its outputs and
inputs at the prices it expects, which keeps the decision a function of the agent's
state alone. The recipe with the greatest `net_r / Duration_r` (value per tick of
labour) is chosen; two recipes are compared by cross-multiplication in 128 bits, and
a tie goes to the lower recipe ID. A recipe is eligible only if `net_r > 0` or it is a
source (no inputs): a source costs nothing but labour, and a household that must work
to eat will cut or gather even when its belief of the price is low. This answers N-034's
question of the default choice rule. A source stays eligible at `net_r <= 0` with the rent in
`cost_r`: a rent moves the choice among recipes and the eligibility of those with inputs, and
does not stop a household that must work to eat (`spec-land.md` 4.6.2). A recipe with `Builds`
or `Repairs` has at least one input and only discard outputs (`spec-infrastructure.md` 4.1.2
B2, B3), so its `value_r` is 0, it is not a source and its `net_r` is `-cost_r`: it is never
eligible here, and only the project request of 7.1.11 asks for one.

7.1.5. **The production halt.** A recipe is skipped if any of its outputs with a
positive `CostShare` is overstocked:

```
Stock_o * 10 000 >= ProduceHaltRatio * OutputTarget_o(r)
OutputTarget_o(r) = ceil(w * q_o * E_r * OutputCoverTicks / (10 000 * Duration_r))
```

A byproduct (`CostShare` 0) never halts a recipe, or a pile of waste would stop the
main product. An `OutputTarget` of 0 never halts a recipe: `OutputCoverTicks` may be 0 (`bounds.go`), and the
literal inequality `Stock * 10 000 >= Ratio * 0` is then always true, which would halt every such
recipe, so the rule is read only for an output whose target is above 0. The world's counts and the
default `Work` read the one work report of 7.1.3 for the halt as for the floor. The halt does not apply to an own-use recipe: its stop is the exit cover of
the mode (7.1.9). This is the inventory-driven part of production (a firm with too much
inventory hires fewer, in Lengnick's model): a negative feedback from stock to supply,
with the delay of the recipe's duration. It is the **production stop** that the
register proposed as STB-D24 and left without an owner (N-043); this spec adopts it,
with `ProduceHaltRatio` as its parameter. It acts on the stock, which is local
information (`spec-stability.md` 5.1.3). Its cost is that a halted role earns nothing
while the stock lasts. `spec-stability.md` 6.2 finds that **where** such a stop sits
decides whether an overhang ends at the floor of the reference band or at the ceiling
(a stop at 300% of the target still let the price fall to 128 to 138, one at 150%
sent it to 3 207 to 6 367, in a model with inelastic flows), so the default 30 000 is
a guess that the scenario S16 must set, and the halt is an entry of section 18.

The halt acts silently, so it is **counted** (FR-STB-013, N-067), and the count is the
world's, as the floor's is in 7.1.3, not the strategy's: `Work` returns the `JobRequest` of
6.3 and nothing beside it. For each agent that is free when `Work` is called, the world
evaluates the condition above for each recipe of the role from the agent's own stock and
`OutputTarget` (a pure function of the view, which a consumer's strategy cannot hide), and
the event `ProductionHalted` (`spec-observability.md` 4.4.6) carries, per region and role,
the households (the weights of the agents) that were free, the households that requested
no job and for which at least one recipe was skipped for the halt ("declined for the
halt": an agent skipped on recipe A that runs recipe B is not halted), and per recipe the
households for which that recipe was skipped. An agent whose work share is 0 is `rest`
(14.3), not `halted`: the halt is counted only for a household that would have worked.
The reason of the `Debug` line is `halted`. Households are the agent's weight, never its
`Executions`.

7.1.6. **Keep the food.** An input that is also an upkeep commodity is not used to
the point of starvation:

```
k <= floor( (Stock_c - U_c) / q_c )     for each input c that is also upkeep,  (0 if Stock_c <= U_c)
```

A recipe whose cap of this kind is 0 is **ineligible** for the request, and the next best
eligible recipe of 7.1.4 is tried. The request is never `Executions` 0: 0 means "the most the
agent can run" (6.3), and `production.Count` reads 0 as no maximum, so a 0 would start the whole
job and eat the food. `NoJob` with the reason `no_recipe` is returned only if no eligible recipe is left.

7.1.7. **How many.** The request is `Executions = min( ceil(w * s / 10 000), kcap )`
with `kcap` the least of the view's `MaxExecutions` and the caps of 7.1.6, and at
least 1 if `s > 0` and a recipe is eligible, counting only the recipes whose cap
`kcap` is at least 1 (7.1.6). Production starts `k` executions or
reports a reason (`spec-production.md` 4.2); a request above what it can run is cut
to what it can, not refused.

7.1.8. **A trainee** (`spec-skill-barriers.md` 4.1) is passed the trainer's
trainee-eligible recipes and its yield `TraineeYield`, and chooses among them by the
same rule, with the principal's stock for `MaxExecutions` (`spec-production.md` 8.1).

7.1.9. **The own-use mode.** With the subsistence module on, before the recipe choice of 7.1.4 the
default applies this rule to an agent that is free (the rule and its reasons are
`spec-subsistence.md` 5.5; the flag is computed by the world in phase 2, before any `Work`,
`spec-subsistence.md` 5.4):

1. **Not in the mode** (`OwnUse` false): nothing changes. The work share is 7.1.2's and the
   recipes are the role's. An own-use recipe is never chosen outside the mode.
2. **In the mode:** the work share is 10 000 (survival comes before the target, 7.1.3).
3. **Which recipe.** Among the own-use recipes with `MaxExecutions >= 1` and a positive sum of
   coverage over the covered commodities whose cover is below the exit cover, the one with the
   greatest sum, compared as exact fractions, ties to the lower recipe ID. The value uses no
   price and no belief.
4. **How many.** `k = min( w, MaxExecutions, max(1, ceil( w * (ExitTicks - Cover) / ExitTicks )) )`:
   a weight-1 agent runs 1, a cohort puts in the share of its households short of cover. The other
   households rest during the job (one job per agent, `spec-production.md` 5.1.2).
5. **The retry.** Every started own-use job adds 1 to `OwnUseJobs`; a job of any other recipe
   resets it to 0. If `RetryJobs > 0`, `OwnUseJobs >= RetryJobs` and a role recipe is eligible by
   7.1.4, the request is that role recipe instead (a test of the market); otherwise the own-use
   job goes on.
6. **Nothing runnable.** If no own-use recipe qualifies, or the ration granted none of its
   executions, the household is **blocked**: `Work` treats the tick as if it were not in the
   mode, the flag stays, and the household is counted in `Blocked` (`OwnUseChanged`).
7. **The halt does not apply** (7.1.5).

A consumer's strategy sees the same flag in its view (6.4.1); the `Directed` directive overrides
`Work` as for any recipe (10.4).

7.1.10. **Labour offers and vacancies** (employment module on, `spec-employment.md` 4.3). After
the choice of 7.1.2 to 7.1.7 and using its results, the default fills the two fields that
`JobRequest` gained (6.3) and changes nothing else. **A worker's offer:** for a role that is
`Hireable`, an agent that is not in the own-use mode and not a trainee, and whose work share
`s` (with the survival rule and the floor of 7.1.3 already applied) is above 0, `Offer` is
`Households = ceil(w * s / 10 000)` with `Reserve` the value per tick of its own best
executable recipe by 7.1.4 and 7.1.5, `max(0, floor(net_r / Duration_r))`, or 0 if it has none;
a household that the target-income rule says should rest offers nothing. **An employer's
vacancies:** for a role with an employer block, an agent that is free, and a recipe of its hire
list that is eligible by 7.1.4 and not halted by 7.1.5, `Hire` is for the best such recipe by
`net_r / Duration_r` at the hired yield, with `Bid = floor(net_r / Duration_r)` and
`Positions` the least of the establishment left, the stock, the tools and the cash after the
upkeep of the tick (`spec-employment.md` 4.3.3). The own request is the fallback of an offer
(`spec-employment.md` 4.1.1). The default keeps no state and draws no word for an offer or a
vacancy: the round breaks its ties by a keyed hash (`spec-employment.md` 5.3). The planned
recipe of 7.2 keeps `k* = w`, and the hires add `H*` to the planned executions of an employer
(`spec-employment.md` 9.2). An employer that is busy with its own job posts no vacancy: `Work`
is not called for it (`spec-employment.md` 4.1.3).

7.1.11. **The project request** (saving module on, `spec-saving-and-investment.md` 5.7). For a
free agent of a role with a `Saver` block, the default requests, before the choice of 7.1.4 and
independent of the work share, the project recipe that passes the test of
`spec-saving-and-investment.md` 5.5: `{Recipe: p, Executions: 1}` for a build and
`{Recipe: p, Executions: MaxExecutions}` for a repair, if its `MaxExecutions` is at least 1;
otherwise the request is that of 7.1.2 to 7.1.7. There is none while `Hunger` is above 0,
while `Money` is below `ReserveMoney` (7.4.2), in the own-use mode, or while the active job is
a project recipe (`spec-saving-and-investment.md` 5.5.5, 5.5.6), and a project whose per-recipe
`Standing` is not empty is not weighed (`spec-saving-and-investment.md` 5.5.1). The numbers
7.1.10 and 7.1.11 do not fix an order of evaluation.

### 7.2 The planned recipe

`Trade` needs to know what the agent expects to do next, so that it can buy the
inputs. The **planned recipe** `r*` is chosen as in 7.1.4 and 7.1.5 but *ignoring
stock*: among the role's recipes with positive yield, those with `net_r > 0` or a
source, not halted, the one with the greatest `net_r / Duration_r`. It may be none. The
planned executions are `k* = w`. A recipe whose input the agent lacks is still planned:
it is what the agent will try to buy for. `Work` then picks the best *executable*
recipe, which may differ (the tool-less variant of `spec-production.md` 3.3 when the
tool is missing). With the saving module on and a project funded
(`spec-saving-and-investment.md` 5.6.4), the planned recipe is the project recipe and `k*` its
executions; for an employer the hires add the planned job `r_h` and its `H*`
(`spec-employment.md` 9.2.1).

### 7.3 Needs and targets

For each commodity `c` in the tradable set, the quantities below are **totals for the
agent**, scaled by the weight, and are computed in `Trade`.

```
U_c      = w * upkeep per household of c (0 if not upkeep)                (5.2)
Reserve  = U_c                         units to be eaten at the end of this tick
Job_c    = 0                           if r* is none or StartsIn > 1
         = InputJobs * k* * q_c        if c is an input of r* and StartsIn = 1
         + k*                          if c is a tool of r*
UseTgt_c = U_c * CoverTicks + Job_c    units to hold at the start of next tick
Make_c   = k* * q_o * E / (10 000 * Duration)   if c is a non-discard output of r*
OutTgt_c = ceil(Make_c * OutputCoverTicks)   one ceiling over the whole product, Make_c the unrounded fraction
Want_c   = 0 with the demand module off; with it on, the want units funded for c (spec-demand.md 4.5, 4.6)
Target_c = UseTgt_c + Want_c   if either is above 0,   else OutTgt_c      (0 if both are 0)
Keep_c   = Reserve + UseTgt_c + Want_c   units the agent wants to hold now
```

`OutTgt_c` is one ceiling over the whole product, `ceil(k* * q_o * E * OutputCoverTicks / (10 000 *
Duration))`, with `Make_c` left unrounded: it is the same number as the `OutputTarget` of 7.1.5 (with
`w = k*`), so the target and the halt cannot disagree for one commodity. The tool term (`+ k*` for a
tool of `r*`) is part of `Job_c`.

`Want_c` is known only after tiers 1 to 3 are allocated, because the wants are paid from what
the cap leaves (7.4.3); the tiers 1 to 3 do not depend on it, so the order is: 7.3 without
`Want_c`, the tiers, the demand module's allocation, then `Keep_c`, `Target_c` and the gap are
raised by `Want_c` for the bid. An agent without the module has `Want_c` = 0 throughout. For an employer
(`spec-employment.md` 9.2.1) `Job_c` also counts the inputs and tools of its planned hires:
`Job_c = InputJobs * (k* * q_c(r*) + H* * q_c(r_h))` for an input of either recipe and
`k* + H*` for a tool, `StartsIn` being 1; the rest of 7.3 to 7.5 is unchanged.

7.3.1. **The target.** `Target_c` is the number the belief rule receives
(`spec-beliefs.md` 4.4), the units the agent wants to hold *after* this tick's
upkeep. For a commodity it uses, it is a number of ticks of use plus the inputs of
its next job (the lump). For a commodity it only makes, it is a number of ticks of its
output. For a commodity that is neither, it is 0. This answers N-037: the cover period
is `CoverTicks` for use and `OutputCoverTicks` for output, and "recent demand" of
`spec-beliefs.md` 4.4 is the *expected* use computed from data, so the target needs no
history and no state. The two readings of "a missing ideal stock as 0" (N-007, N-037)
are one rule: 0 means the agent has no use for the commodity, so it buys none, sells
any it holds, and the belief rule is told there is no stock signal.

7.3.2. **The lump.** A job consumes its inputs at its start (`spec-production.md`
4.3). An agent mid-job does not need the next job's inputs until it is free, so
`Job_c` is 0 unless `StartsIn` is 1. A job of duration 120 therefore costs its inputs
once in 120 ticks, in the tick the previous job completes, as a purchase that all
agents of a role whose jobs completed that tick make together. That synchrony is an
entry of section 18.

7.3.3. **Buy or sell.** For each commodity, at most one order, by the sign of the
difference between what is held and what is kept:

```
Gap_c     = max(0, Keep_c - Stock_c)        a bid, if > 0
Surplus_c = max(0, Stock_c - Keep_c)        an ask, if > 0 and no bid
```

7.3.4. **Tiers of a bid.** The gap splits into three parts, in priority order (a fourth, the wants of `spec-demand.md`, is added on top when that module is on, below). With
`left = max(0, Stock_c - Reserve)` and `Jobwant_c = Job_c`:

```
T1_c = max(0, Reserve - Stock_c)                       upkeep this tick
T2_c = min(Gap_c - T1_c, max(0, Jobwant_c - left))     the inputs and tools of the next job
T3_c = Gap_c - T1_c - T2_c                             buffer: the cover periods
```

The three add to the gap by construction (worked in 16.2). With the demand module on, the gap is first the gap of 7.3.3 without `Want_c`, and the bid carries a fourth part
`T4_c = Want_c`, the units the module funded in rank order from the cap that tier 3 left
(`spec-demand.md` 4.5): the total gap is `T1 + T2 + T3 + T4` and the three parts above are
unchanged.

7.3.5. **Asks.** The agent offers all of the surplus: it holds no unsold goods back
to wait for a better price. The stock rule of `spec-beliefs.md` 5.5 already lets the
price follow the stock, and holding back would turn the seller's own choice into a
signal of shortage (the unsold stock the belief rule reads would include what the
seller kept). The units that expire at the end of this tick are read with the ledger's
read-only query `Expiring(account, commodity, tick)` (`spec-ledger.md` 5.3.3), which
counts the units not earmarked that spoil by the end of the tick (before the first order of a tick
no earmark exists, so they are all of them); those not needed for this tick's upkeep
are offered even when there is no surplus above `Keep`, if no bid is
placed for the commodity. A **tool** is offered only by a role that makes it (a
non-discard output of one of its recipes): a holder's tool is bound to it
(`spec-skill-barriers.md` 2.7), and the estate is the only seller of a used one.

The **candidates** are decided by the view alone, never by affordability, so that each takes
exactly one word of the stream (7.5.3) whatever the money. A *bid candidate* is a commodity with
`Gap_c > 0`. An *ask candidate* is a commodity with `Gap_c == 0` and (`Surplus_c > 0` or
`Expiring_c - Reserve > 0`); "no bid is placed" above means `Gap_c == 0`, that is, not a bid
candidate. The ask is `max(Surplus_c, max(0, Expiring_c - Reserve))`: the expiring units are part of
the stock, so the two are not added. A bid candidate whose limit cannot buy one unit spends its word
and forms no ask. A tool that the role does not make is not a candidate and takes no word. `Tool`
(6.4.1) is derived from the whole configuration, not from the role. With the
storage module on, a limited agent first discards, before it forms the asks, the surplus units
that its tier 1 bids need room for, the minimum that lets them fit (`spec-storage.md` 6.3.2,
5.5.2.1); the discard is a `Discard` of the plan (6.5.2), counted under `disposal` (8.2), and
nothing is discarded for tier 2 or beyond.

7.3.6. **`Purpose` and `UnitCost` of an intent.** Purpose is `PurposeInput` if
`Job_c` is above 0 and `c` is an input of `r*`, `PurposeTool` if it is a tool of
`r*`, `PurposeUpkeep` if `c` is an upkeep commodity, and `PurposeOther` otherwise. A
commodity that is both an input and upkeep is classed as an input. For a merchant, `r*` is its
planned trip: the cargo, the haul units and the feed are inputs, a vehicle is a tool and the
provisions are upkeep (`spec-merchants.md` 3.4.2). `UnitCost` is the
cost anchor of the commodity in the agent's region (`spec-production.md` 9.3) if the
role has a recipe that makes it, and 0 otherwise (`spec-beliefs.md` 3.2.4). **The
anchor is the full cost, labour included**: inputs, the wear of tools and `Duration`
working ticks valued at the region's pooled observed wage of the previous tick and, with the
tenure module on, the rent of its draws at the posted rent (`spec-production.md` 9.1, 9.3,
FR-PRO-047, N-055; `spec-land.md` 5.7.1). It is **0 while the region has no
observed wage** (`spec-production.md` 9.1.1, FR-PRO-049), so in the first ticks of a
run with no starting wage only the reference band applies to beliefs
(`spec-beliefs.md` 3.2.2). The strategy does not compute the anchor: the world computes
it once per tick before `Work` is called (FR-PRO-048) and hands it to the view and to
the intent; the default `Trade` passes it through unchanged. A bid whose commodity is a want item and nothing else has `PurposeOther`: it is not netted from income (`spec-welfare-and-wages.md` 2.1).

7.3.7. **Room.** With the storage module on and the agent limited, after `UseTgt_c` and
`OutTgt_c` are computed and before the tiers, the targets are scaled to the room
(`spec-storage.md` 6.2.2): `Avail_d = max(0, Room_d - Pledge_d - ResLoad_d)` per dimension, `f`
the least `floor(Avail_d * 10 000 / TL_d)` over the dimensions where the load of the targets
`TL_d` exceeds it (10 000 if none binds), and `Target'_c = floor(Target_c * f / 10 000)`,
`UseTgt'_c = floor(UseTgt_c * f / 10 000)` and `Keep'_c = Reserve_c + UseTgt'_c + Want_c`
replace `Target_c`, `UseTgt_c` and `Keep_c` of 7.3 in the gap, the tiers and the intent. The
scaling is pro rata over every commodity of the target; it is not a ceiling on stock
(`spec-storage.md` 6.2.4). An agent that is not limited, or with the module off, is unchanged.

### 7.4 The budget stack and the saving rule

7.4.1. **The hard constraint.** The sum over the agent's bids of limit times quantity
is at most its available money. The default strategy builds its orders so that none
is rejected for funds (FR-AGT-044); the market's rejection (`insufficient_funds`) is
the backstop for any other strategy (`spec-clearing.md` 3.3). There is no negative
balance (`spec-ledger.md` 4.1); a loan of the credit module raises the balance before it is
spent, and the money in a bid is still reserved from the balance. The money in a bid is
reserved at its limit, the worst case, so the budget is computed at the limit price, not at
the expected price.

7.4.2. **The reserve.** The agent's upkeep cost for one tick, at its own beliefs, is
`UpkeepCost = sum over upkeep c of U_c * Centre_c` (checked). The **reserve money** is

```
ReserveMoney = ReserveTicks * UpkeepCost
```

`ReserveMoney` is the flat reserve: 7.1.2, 7.1.3 and the warning `reserve_exceeds_money` read
it. The excess of 7.4.4 is taken over `SavingTarget`, which is `ReserveMoney` unless
`Modules.Saving` is present and the role has a `Saver` block; then it is `max(ReserveMoney,
Precaution) + Goal` (`spec-saving-and-investment.md` 3.2). It replaces the flat reserve in
`e` only.

7.4.3. **The stack.** Money is allocated to tiers in order. Within a tier, commodities
are taken in canonical order, each at its drawn limit `L_c` (7.5):

| Tier | Spends on | Limited by |
|------|-----------|-----------|
| 1 | `T1_c`, upkeep this tick | the available money only |
| 2 | `T2_c`, the inputs and tools of the next job | the available money after tier 1 only |
| 3 | `T3_c`, buffers | the **spend cap** below |
| 4 | `T4_c`, the wants of `spec-demand.md` (module on) | the part of the same cap that tier 3 left, `cap - spent in tier 3`, in the ladder order of `spec-demand.md` 3.2.2, not canonical commodity order |

Tiers 1 and 2 are not subject to the saving rule. A household must be able to eat,
and a producer must be able to buy inputs, or a household that has fallen below the
reserve could never produce its way out; a rule that withheld inputs from the poor
would be a poverty trap (a stability loop, section 18). The cost of this choice is
that a long job can leave the household with no money for the length of the job; that
is the interaction of N-032 and open question 6.

7.4.4. **The spend cap.** After tiers 1 and 2, with `avail` the money left:

```
e   = max(0, avail - SavingTarget)                    excess over the savings target (`ReserveMoney`, 7.4.2)
H   = SpendHalfTicks * UpkeepCost                     the excess at which the share is half
cap = SpendMaxShare * ( e * H / (e + H) ) / 10 000    integer, checked, 128-bit intermediates
```

If `UpkeepCost` is 0, `cap = SpendMaxShare * e / 10 000`. The cap is the **rule that
saves when wealth is low**:

- below the reserve (`e = 0`) nothing is spent on buffers: the household rebuilds its
  savings first;
- above it, the cap rises with the excess but less than in proportion (the function
  `e * H / (e + H)` is increasing and concave, and bounded by `H`), so a richer
  household spends a smaller share of what it has, as Lengnick's consumption does
  (consumption rises with wealth at a decaying rate, section 17);
- the cap never exceeds `e`, so the buffer spending never touches the reserve.

The tier-3 want of each commodity is further scaled by favourability (7.5.2); the cap
is applied to the scaled wants, in canonical order, each at its limit. The cap bounds the quantities of tiers 3 and 4 only and never lowers a limit (7.5.4). Tier 4, when the demand module is on,
takes `max(0, cap - spent in tier 3)` and so is bounded by the same excess: tiers 3 and 4 together never
spend more than the cap and never touch the reserve (`spec-demand.md` 4.5, FR-DEM-020). Favourability
does not apply to tier 4: the wants answer to price through their schedule.

7.4.5. **How this differs from Lengnick.** His household spends `min((m/P)^alpha, m/P)`
on consumption goods, so a household with little liquidity spends *all* of it
(section 17). Here the first call on money is the upkeep, which Lengnick's
households have no analogue of, and a household with little money protects a buffer by
not building stock. The concave form is kept; the power function is not (no floating
point, architecture.md 3.5), and the exponent 0.9 is not adopted.

### 7.5 Building the orders

7.5.1. **Order of work.** For each commodity in canonical order that has a gap or a
surplus, the strategy forms an intent. With the demand module on, the want units of each commodity are determined after tiers 1 to 3 and joined to its intent (7.3.4); the intents stay in canonical commodity order. The intents are returned in canonical commodity order, one per commodity; the money is allocated tier by tier (7.4.3), so
the budget is computed here and the order of the intents matters only for
the world's check.

7.5.2. **Favourability.** The market price's position in the price window
(`spec-beliefs.md` 7.3) is `pos`, in basis points, 5 000 if the window is empty. The
share of a tier-3 want that is bought is

```
buyshare = FavourFloor + (10 000 - FavourFloor) * (10 000 - pos) / 10 000
```

so an agent buys more of its buffer when the price is low in the window and less when
it is high, and never less than `FavourFloor` of it. It is bazaarBot's favourability
(`bazaarbot-reference.md` section 5) in integer form with a floor. It applies only to
tier 3: upkeep and job inputs are bought at whatever the price. Asks are not scaled
(7.3.5): the original scales the quantity of an ask by `pos` too, which is dropped
(open question 7).

7.5.3. **The limit price.** One `uint64` is taken from the world's source for each
intent, and the limit is drawn from the belief's effective range with
`beliefs.Draw` (`spec-beliefs.md` 6). The limit is a price per unit and is the same
for a cohort of any weight (`spec-scale.md` 4.4). Two rules:

- **Urgency.** If `Hunger > 0` and `T1_c > 0`, the limit of the bid is the top of the
  range, `High`, not the draw: a household that went hungry bids what it is willing to
  pay at most. The draw is still taken, so the number of draws does not depend on
  hunger. This lets a starving household outbid its own random draw without leaving
  its belief range.
- **Affordability.** A bid whose limit exceeds the money left for one unit is lowered
  to that money. If the money left cannot buy even one unit at the lowered limit
  there is no order, and the draw was spent. So a draw is taken for every intent the
  strategy considers (a candidate of 7.3.5, decided by the view alone and never by
  affordability), and the number of draws is a function of state, not of the
  outcome.

One order has one limit: `L_c` is drawn once, and each tier buys at the order's limit. **Only the
first tier to buy may have the limit lowered by `Afford`**: a first buyer whose money cannot buy one
unit at `L_c` lowers the limit to the money it has and buys one unit. A later tier that cannot pay one
unit at the order's current limit is **dropped, never lowered**. Every tier is charged at the limit it
bought at, so `limit * qty` is at most the sum charged and the market cannot refuse the order for funds
(FR-AGT-044). After a lowering the money left is 0, so at most one tier of one commodity is ever
lowered. The reason: spending is by priority, necessities first, and a lower-priority want must not
lower the limit price of a necessity. The spend cap of 7.4.4 bounds the quantities of tier 3 (and of tier 4 with the demand module) only and
never lowers a limit.

7.5.4. **The quantity.** `qty_c = q1 + q2 + q3 + q4`, each tier's quantity bounded by its
want, by what the money at the limit allows in that tier, and (for tiers 3 and 4) by the
cap, and, with the storage module on, by the free room, allocated in the order of the money
stack (`spec-storage.md` 6.3.1), each tier at the order's one limit (7.5.3: the spend cap bounds the quantities of tier 3 (and of tier 4 with the demand module) and lowers no limit); `q4` is the units of `Want_c` funded by `spec-demand.md` 4.5 (0 with the module off), at the
same limit as the rest of the bid. A commodity whose quantity is 0 gets no order. The intent carries `Target`,
`Reserve`, `UnitCost` and `Purpose` (7.3.6).

7.5.5. **The ask.** An ask's quantity is `max(Surplus_c, max(0, Expiring_c - Reserve))` (7.3.5): the
oldest lots go to the upkeep first (`spec-ledger.md` 5.2.1), and the expiring units are part of the stock,
so the two are not added. Its limit is a draw
from the belief's effective range. The cost floor of the belief (`spec-beliefs.md`
3.2.2) means a seller's limit is never below `CostFloor` of its full unit cost (70% by default, labour included, `spec-beliefs.md` 3.2.2 and 3.2.4 there), and there is no floor while the cost anchor is 0.

### 7.6 `Revise`

`DefaultStrategy.Revise` calls `beliefs.Update` (`spec-beliefs.md` 5) with the
belief, the bounds, the outcome, the market report, `o.Stock`, `o.Target`,
`o.UnitCost` and `o.Draw`, and returns the new belief. Nothing else: one update per
admitted order, with the gate draw the world took (`spec-beliefs.md` 5.1.2).

### 7.7 Credit

The default borrower and lender are the rules of `spec-credit.md` 10.2 and 10.3, evaluated in
`Trade` after the tiers of 7.4.3 are built. They append at most one request (`Borrow`) and one
offer (`Lend`) to the plan and take no draw. `Tier2` and `Late` of the borrower's rule are that
spec's (10.2, 10.2.1.2): `Tier2` is the cost of the inputs and tools of the next job at the
limits of the plan, whatever money allows. A role with no `Borrow` or `Lend` block gets neither,
and with the module absent none is possible (`credit_off`).

### 7.8 Cover

The default insured and the default underwriter are the rules of `spec-insurance.md` 10.2 and
10.3 (exported as `DefaultCover` and `DefaultOffer`, 10.4 there), evaluated in `Trade` after the
credit rule of 7.7 and filling the plan fields `Cover`, `Store` and `Offer` of 6.3. They take no
draw and keep no state.

## 8. Inventory capacity and discards

This answers N-002.

8.1. **Decision: no inventory capacity unless the storage module is on.** With it off an
agent can hold any quantity up to `MaxQuantity` of a commodity; with it on a limited agent has a
room (`spec-storage.md` 3, 4) and 8.3 says what changes. Reasons for the default:

1. bazaarBot's bound (`max_size`) is the source of two of its defects (the limit
   formula of defect 8 and the discarding of `makeRoomFor`, `bazaarbot-reference.md`
   section 10); a capacity makes a rule for what to discard, and that is a strategy
   decision, not a mechanism (N-002);
2. `spec-goods.md` provides `Mass` and `Bulk`, but the container with a capacity is
   `spec-storage.md`, an optional module (`vision.md` 4). A capacity outside it
   would be a second, ad hoc store;
3. production never refuses output for lack of room (`spec-production.md` 5.3.2), so
   a capacity would put a rule of loss in a place nothing else needs one;
4. overstock is limited in version 1 by spoilage (`spec-goods.md` 4), by the buffer
   spending cap (7.4), by the production halt (7.1.5) and by the stock rule of the
   belief update, which lowers the price of a commodity held in excess.

8.2. **Discards.** A strategy may discard (6.3). The world executes it as a `Move` to
the `disposal` boundary (`spec-production.md` 7.3). The `disposal` reason is a goods
sink in the registry, so a discard is counted in `BoundaryFlow` by region and
commodity, which is the event N-002 asked for. The default strategy never discards, except, with the storage module on, for upkeep: a limited
household discards the surplus that its tier 1 bids need room for (7.3.5,
`spec-storage.md` 6.3.2). A good it cannot sell falls in price and spoils or waits.

8.3. **With the storage module on** (`spec-storage.md`), room is decided at the decision and
not found out on the next tick: the view carries the agent's free room (6.4.1), its targets are
scaled to the room (7.3.7), a bid is sized to the free room (7.5.4), a job request beyond the
room is refused `no_room` (`spec-production.md` 4.2) and a bid beyond it is rejected `no_room`
(`spec-clearing.md` 3.2). Discarding stays a strategy decision, apart from the default's discard
for upkeep (8.2).

## 9. Cohorts

An agent of weight `w` stands for `w` identical households that act in lockstep
(`spec-scale.md` 2.2). For this spec:

9.1. **Invariance holds only where it is exact, and each claim has its condition.** A cohort
of weight `w` holds `w` times the stock and money of a weight-1 household of the same role,
with the same beliefs and the same drawn words. Then:

- **Exactly `w` times** (the quantities are linear: sums, `max` and `min` of homogeneous
  terms): the wants `T1`, `T2`, `T3` before the buy share and the cap, `Gap`, `Surplus`, the
  asks (`Surplus`, `Expiring - Reserve`), the reserves, `Keep` and `UseTgt`, and a `Target` that
  comes from `UseTgt`. The rules compare totals to totals.
- **Closed form, not `w` times:** a `Target` that comes from `OutTgt` is `ceil(w * X)` with
  `X = q_o * E * OutputCoverTicks / (10 000 * Duration)` (one ceiling over the whole product, 7.3), so
  `w * X <= OutTgt <= w * ceil(X)`; it equals `w` times the weight-1 value only where `X` is an
  integer (a recipe of `Duration` 3 and output 1 with `OutputCoverTicks` 1 gives 1 at `w` 1 and 3
  at `w` 7, against `7 * 1`). The halt `w * s * 10 000 >= Ratio * ceil(w * X)` is not
  scale-invariant at the threshold, so the planned recipe `r*`, and with it `Job_c`, `Keep` and the
  asks, is the same at both weights only away from the halt threshold.
- **Limits** are equal (the draw does not see the weight; a limit never scales) only while the
  first tier to buy is not lowered by `Afford` (7.5.3): a first buyer that cannot pay one unit at
  the limit lowers it to its money, and that can happen at weight 1 and not at weight 7 (money 50,
  limit 105: one unit at limit 50; money 350: three units at 105).
- **Allocated quantities** of tiers 1 and 2 are `w` times the weight-1 quantity only when both
  households fund the tier in full at the drawn limit (ample money, no lowering). When money
  binds the allocated quantity is `floor(w * m / L)`, which is not `w * floor(m / L)` (money 250,
  limit 105, wants 3: 2 units at weight 1; with `w` 7 and wants 21, 16 units against 14).
- **Tier 3, the cap, the executions and `Fed`** are floors: `floor(w*X*b/10 000)` is not
  `w * floor(X*b/10 000)` (16.2 at `w = 7`: tier 3 of the staple is 33 against `7 * 4 = 28`; the cap
  is `floor(e*H/(e+H))` of totals, 21 740 against `7 * 3 105 = 21 735`); the executions are
  `ceil(w*s/10 000)` (weight 7, share 6 000 gives 5, which is not `7` times the 1 of weight 1); and `Fed = floor(w*h/u)` of holdings (identical
  whole holdings of 2 against an upkeep of 3: weight 1 gives `Fed` 0, weight 40 gives 26). The tier-3
  quantity is bounded by the want rule `floor(w*T3*b/10 000)` and, for the want-bound case only,
  lies in `w * q1 <= q <= w * q1 + w - 1`; the spend cap, a cap-bound allocation and the
  executions are compared with their closed form on the totals (no `w * q1` band: with
  `SpendMaxShare` other than 5 000 the band fails, for example `e1 = 2, H1 = 3, SpendMaxShare = 9 000,
  w = 7` gives cap 0 and cap 7); the `Fed` fraction is equal only when `w*h/u` is exact, otherwise it
  is `floor(w*h/u)/w`.

A test of weight invariance builds its states inside these conditions (ample money, an output
target whose `X` is an integer or asserted in the `ceil(w * X)` form, a stock away from the halt
threshold). The arithmetic is exact because amounts are integers (FR-AGT-054).

9.2. **Prices are never scaled**: the limit price, the belief, the bounds and the unit
cost are per unit (FR-SCL-010, `spec-beliefs.md` 9.1). A cohort draws one limit price
per order and one gate per outcome, as an individual does.

9.3. **Work is proportional.** The work share of 7.1.2 gives `k = ceil(w * s / 10 000)`
executions. A cohort whose ratio is inside the band works with part of its
households, so aggregate labour supply moves smoothly as the ratio crosses the band,
and not all at once. This answers the part of N-014 that says a cohort multiplies the
lockstep of the target-income rule. It is not WEL's closed form: with `RestBand` above 0
the household works until its ratio is `RestBand` above the target, so the days it
works in steady state are slightly more than `spec-welfare-and-wages.md` 5 states,
which is exact only at `RestBand` 0 (open question 6). A weight-1 agent has
`k` in {0, 1}, so for it the rule is the plain threshold with a band.

9.4. **Partial support.** If inputs cover only `k < w` executions the agent runs `k`
(`spec-production.md` 4.2). The other households rest for the length of that job
(`spec-production.md` 5.1.2).

9.5. **Unfed households are counted exactly** (5.4): a cohort of 40 with 100 staple
against a need of 120 has 33 fed and 7 unfed (16.3).

9.6. **A cohort that fails (N-008).** AGT's part is settled by 3.6.1, 3.7 and 5.5: it
supplies `Unfed` and `Hunger`, the effect of a weight change on the agent's state, and
the rule that nothing is removed here. The failure ladder is `spec-spawner.md`'s.

9.7. **The floor `K`** (`spec-scale.md` 5.2) keeps the work decision of 9.3 spread
across at least `K` agents of a group, each with its own target drawn at creation
(`spec-welfare-and-wages.md` 5.1).

## 10. The player and other external agents

This answers N-022 and the question left open in overview section 4: how a
consumer-driven agent fits the synchronous `Step`, which bears on LED and INT.

10.1. **What an external agent is.** An agent added through the integration interface
(`spec-integration.md` 4.1) after construction; no configuration field marks an agent
external. It has weight 1 (`spec-scale.md` 4.7), an ordinary account of kind Agent, a role, a
region, and is bound by the same admission, reservation and conservation rules as any
agent (N-022, `spec-clearing.md` 3). It receives upkeep, can go unfed, has beliefs, a
welfare history and a job. The only things that distinguish it are the `External` flag
and its strategy.

10.2. **The problem.** `Step` is synchronous and single-threaded. All orders of a tick
are placed before any market clears (`spec-clearing.md` 7.2), and the determinism of a
run requires that nothing outside the world's own state changes a tick's result
(architecture.md 7.4). A `Strategy` that waited inside `Trade` for a human to click
would stall `Step` and, with it, the game loop that called it. So a consumer-driven
agent cannot act *during* a step, and it must act *between* steps through data.

10.3. **Decision: two ways, one model.**

- **A consumer-written `Strategy`**, registered under a name and given to the agent
  (`WithStrategy`, `AddAgent`). It is called synchronously like any strategy, on the
  stepping goroutine, and **MUST NOT block**: it returns from its view and the state
  it keeps (opaque bytes saved by the optional pair of ADR 0002 decision 7). This is the
  way for scripted agents, bots and AI merchants.
- **The directed strategy** for a player. The consumer leaves a **directive** in the
  agent's mailbox between steps, and `Directed` reads it:

```go
// Directive is what a consumer asks a directed agent to do in the next tick.
// Plain data. It lives in the agent (the mailbox), so a checkpoint saves it.
type Directive struct {
	Job      *JobRequest  // used by Work; nil means the directive has no job and Work is the fallback's (a non-nil request with NoJob set requests nothing)
	Intents  []Intent     // used by Trade
	Discards []Discard    // used by Trade
	Hold     []CommodityID // commodities the fallback must not trade
	Borrow   *CreditRequest // replaces the fallback's credit request (spec-credit.md 10.5); nil means the fallback's
	Lend     *CreditOffer   // replaces the fallback's credit offer (spec-credit.md 10.5); nil means the fallback's
	Sticky   bool         // keep the directive after the tick it applies to
}

// Directed is a Strategy that does what the agent's directive says and leaves the
// rest to Fallback. A nil Fallback means: nothing without a directive.
type Directed struct {
	Fallback Strategy
}
```

`Directed` implements `Stateful` (architecture.md 4.3) by delegation: its `SaveState` is the
fallback's when the fallback is `Stateful` and `(0, nil, nil)` otherwise, and its `LoadState`
loads the fallback's (`spec-checkpoint.md` 7.5). The directive is not that state: it is saved
in the agent record (15). The library registers `Directed{Fallback: DefaultStrategy{}}`
under the built-in name `directed`, beside `default` (6.6). It is not the `directed` policy of a
policy actor (`spec-policy.md` 8.4).

10.4. **The rules of `Directed`.**

1. The directive is read in phase 1 and applies to the tick that follows; the world
   discards it after that tick unless `Sticky` is set (FR-AGT-057).
2. `Work` returns the directive's `Job` if it has one (`Job` is non-nil), else the fallback's.
3. `Trade` returns the fallback's plan with two changes: an intent in the directive
   *replaces* the fallback's intent for the same commodity, and a commodity in `Hold`
   is removed from the fallback's plan. `Hold` removes only the fallback's intent, so the
   directive's own intent for a held commodity survives. The directive's discards are added. A `Borrow` or a `Lend`
   replaces the fallback's credit request or offer (`spec-credit.md` 10.5); a directive carries no
   cover field in version 1 (`spec-insurance.md` 10.5).
4. `Revise` is the fallback's (a player's price beliefs adjust by the default rule so
   its automatic orders remain sensible). With a nil `Fallback`, `Revise` returns the
   belief unchanged.
5. A directive's intents obey the same admission as any (6.5): the world does not trust
   them, and a player cannot overspend, trade a commodity outside the tradable set of
   its role or place two orders in a market.

A player that does nothing therefore eats, if its fallback is the default strategy, and
its stock and money follow the default rules for what the directive leaves out. A
player who wants to hold a good back lists it in `Hold`.

10.5. **Why not an immediate action.** A player who buys a sword "now" has an order,
placed in the next tick's trade phase and filled at that tick's clearing, one tick of
latency. An action that settles at once outside the phases would break the rule that
the order of clearing cannot change a result (FR-CLR-023), would need the world to
execute a trade between steps and would make the checkpoint depend on when the consumer
called it. `spec-integration.md` may add an explicit "act now" operation with its own
rules; this spec does not need one, and the model above works without it.

10.6. **What this asks of other specs.**

- `spec-integration.md`: the operations `AddAgent` (region, role, strategy name, the
  starting money and goods and the `initial` faucet that funds them) and `Direct(agent,
  Directive)`, which validates the agent exists, is external and uses a directed
  strategy (its strategy's dynamic type is `Directed`; any other external agent gets
  `ErrNotDirected`); an empty strategy name of `AddAgent` means the built-in `directed` (6.6).
  The mailbox is plain data, set between steps only. The API names are INT's
  (`spec-integration.md` 4.1, 4.3).
- `spec-ledger.md`: nothing. An external agent is a normal account; its starting wealth
  is a named faucet (3.5.4).
- `spec-spawner.md` and `spec-population.md`: an external agent is **not** replaced,
  merged, reduced, removed or moved to another role, and the world refuses any such decision with
  the reason `external_agent`, unless the agent is **mortal** (`spec-spawner.md` 9.2). `Mortal` is
  a flag of the agent (3.1), set by the consumer when it adds the agent, default false;
  `spec-integration.md` owns the field on `AddAgent` (`NewAgent.Mortal`, 4.1; the check
  follows). Its `Unfed`, `Hunger` and `Privation` are visible and count, but only a mortal
  external agent is subject to the ladder, through a consumer's spawner (by `Apply`) or the
  default spawner; whether a player can starve is the consumer's choice, made by setting
  `Mortal`. `Mortal` limits only what the library decides: the consumer's own removal call
  (`RemoveAgent`, `spec-integration.md` 4.4.1) applies to any external agent, mortal or not.
- `spec-observability.md`: `AgentSpawned` carries `External` (already in observability-and-logging.md 4.3).

## 11. Liquidity, relief and the target-income rule

How the three fit together, for `spec-stability.md` and for N-028:

11.1. **Relief** (`spec-estates.md` 2.6) is paid in phase 1 to agents whose welfare
ratio is below `ReliefBelow`. It counts as income in the ratio and is excluded from
the observed wage. It raises money, and tier 1 spends first on upkeep, so relief that
arrives in a hungry household is eaten. The saving rule does not reduce relief: the
relief test reads the ratio (income), not money, so an agent cannot raise its relief
by holding money back.

11.2. **Relief and the target-income rule.** Relief tops an agent up to the relief
line only, and the work share still sees it below target (`spec-estates.md` 2.6), so it
keeps working. If relief lifts the ratio above `T_hi` the agent rests; its ratio then
falls and it works again. The band of 7.1.2 makes that a slow cycle, not a flicker, and
the survival rule (7.1.3) keeps a hungry agent working through it.

11.3. **Liquidity and the target-income rule.** An agent with money below the reserve
works (7.1.3) and spends nothing on buffers (7.4.4); its money rises by the
proceeds, and the first thing it does with the rise is repay the reserve. So the two
rules agree: a poor household works more and saves, a household above its reserve and
its target rests and spends a share of the excess.

11.4. **N-028.** The shortage that the target-income rule can deepen (labour supply
`D = K / W` is unit-elastic, so a staple shortage lifts the price, cuts days worked and
deepens itself) is **not** damped by this spec. `spec-stability.md` 6.1 shows the
loop is stable only if demand elasticity exceeds 1 and that for `e = 0.5` every run
ends pinned. The rules here that touch it, the survival rule (7.1.3), the band and
proportional response (7.1.2, 9.3) and the production halt (7.1.5), soften steps and
bound the stock, and none of them changes the elasticity. The damper the register
proposes (a floor on working ticks, STB-D13) is **adopted** by `spec-welfare-and-wages.md`
5.3 as `MinWorkShare` and applied here (7.1.3): it sets where a pinned market stops and
does not change the elasticity. Nothing in this spec claims the loop is stable; scenario
S04 records the measured elasticity, with the floor on and off.

11.5. **Credit.** A household borrows only with the credit module on and a `Borrow` block on its
role (`spec-credit.md` 10.2); with the module off the claims list is empty
(`spec-ledger.md` 10) and a long job cannot be funded by credit, which is N-032's real remedy
and `spec-credit.md`'s.

11.6. **What responds to price or money, and a staple's elasticity.** This answers
the request of N-043 (`spec-stability.md` 6.2: with price-inelastic flows the belief
rules have no interior attractor, and a price goes to a bound). The default strategy's
flows, and how each responds:

| Flow | Responds to | Price elasticity | Where |
|------|-------------|------------------|-------|
| Upkeep purchase (tier 1) | cash only: the want is fixed whatever the price; the limit is drawn inside the belief range (the top of it if hungry) | 0 while cash suffices; -1 (a fixed expenditure) when cash is the limit, so at most 1 in size | 7.3, 7.4, 7.5.3 |
| Job inputs (tier 2) | price through the margin test (a recipe whose net value falls to 0 is not started) and cash | 0 inside a profitable recipe; a switch off at the margin | 7.1.4 |
| Buffer purchase (tier 3) | the price's position in the window and wealth above the reserve | elastic, but **transient**: a buffer is topped up only to its target, so in a steady state the flow is the consumption and is inelastic | 7.4.4, 7.5.2 |
| Own supply | the target-income rule: days worked fall as the wage rises (unit-elastic, backward); the choice among recipes by net value (a switch at the margin); the production halt on overstock; for a source with no inputs, only the first and the halt | negative for labour, 0 elsewhere, a stop at the halt | 7.1 |
| Entry and exit | the spawner | not here | `spec-spawner.md` |

So the **demand elasticity of a staple in version 1 is between 0 and 1 and never above
1**: it is 0 in the interior, and a staple is the main thing a household spends on, so
when cash binds a fixed share of income goes to it. The condition `e > 1` for the
labour loop (`spec-stability.md` STB-L14) and the condition `es < ed` for the cobweb
(STB-L21) therefore fail for a staple, and the stabilising force is the part that does
not depend on elasticity: the bounds of the belief rule, the halt, exit, and the cash
limit. The measured elasticity is recorded by scenario S04.

11.6.1. **Not invented.** A subsistence demand that falls as the price rises (a
rationing of upkeep above a dear price) would make `e` positive in the interior. It is
not added: it would be a guess about how a household trades food for other things, and
and `spec-demand.md` has decided it (N-052 closed): **rationing of tier 1 above a dear price is
rejected for version 1** (`spec-demand.md` 17), because it would turn the `Unfed` signal from an
outcome into a decision and rest on a claim about behaviour no source here supports. What
`spec-demand.md` adds instead is an elastic demand for discretionary goods (a want, with a
schedule and a choke price), including an elastic increment on a staple that a pack lists as an
item of a want. Upkeep itself stays inelastic. If the scenarios S04 and S16 show a pinned staple
market that the halt, the work floor and exit do not cure, the next lever is an upkeep that
accepts alternatives (`spec-demand.md` open question 1), which changes `Fed` and is this spec's
to decide; a new note then, not this paragraph.

11.7. **What spending exists beyond upkeep (N-054).** With the demand module off, none. A household
buys its upkeep, the inputs of its job (which net out of its income,
`spec-welfare-and-wages.md` 2.1) and buffers of the same goods. In a closed economy
(`spec-ledger.md` 7.3) money is conserved, so total income equals total upkeep
spending and the household-weighted mean welfare ratio is 1 whatever `TargetRatio` is
(`spec-viability.md` 4.7). This spec **accepts the glut** and states how it is
bounded: the work share is 10 000 for every role (the ratio never reaches the target),
so households work every tick, until the production halt of 7.1.5 stops a recipe whose
output piles up; a halted household earns nothing for that tick and its output stock
falls by its sales and by spoilage. The saving rule adds to the hoarding side: money in
reserves (7.4.2) is out of circulation, so more saving means less spending and less
income (a paradox of thrift, entry A4 of section 18). Spending beyond upkeep, which
lets a ratio above 1 exist, is now `spec-demand.md`'s: with that module on, the closed economy's
mean ratio is `1 + W / U` and can reach the target up to the ceiling `1 + Wref / U` with no faucet
(`spec-demand.md` 7; N-054 answered), and the warning `target_above_demand` says when a pack's
target is above it. Only a non-producing class, and growth, need a named flow of money: a grant
or a levy of `spec-money-supply.md` (4, 9.2), or `spec-external-sector.md` (`spec-demand.md` 7.3). With the module off, a pack that sets a
`TargetRatio` above 1 for every role should still expect the target-income rule never to rest.

## 12. Parameters

All are tunables with a default, a bound, a doc line and an owner (architecture.md
9.5 to 9.7). **Every default is a placeholder** (the preamble; N-015, N-027). Bounds are
validation bounds checked at construction (`spec-config.md` 4.1); `ErrOutOfBounds`
carries the path, value and bound.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| World | `Upkeep` | the baseline upkeep of one household, a list of (commodity, quantity) (5.2) | none | at most `MaxUpkeepItems`; each quantity from 1 to `MaxQuantity / MaxWeight` |
| World | `CoverTicks` | ticks of use held as the target, for upkeep and recurring use (7.3) | 3 | 1 to 1 000 |
| World | `InputJobs` | jobs' worth of inputs held when the next job is about to start (7.3) | 1 | 1 to 100 |
| World | `OutputCoverTicks` | ticks of own output regarded as the normal inventory (7.3); 0 gives a seller no target | 1 | 0 to 1 000 |
| World | `FavourFloor` | least share of a buffer want bought, in basis points (7.5.2) | 2 000 | 0 to 10 000 |
| World | `ReserveTicks` | ticks of upkeep cost kept as savings before any buffer is bought (7.4.2) | 3 | 0 to 1 000 |
| World | `SpendMaxShare` | the largest share of the concave function of the excess spent on buffers, in basis points (7.4.4) | 5 000 | 0 to 10 000 |
| World | `SpendHalfTicks` | ticks of upkeep cost at which the spend share is half its maximum (7.4.4) | 10 | 1 to 10 000 |
| World | `RestBand` | how far above its target the ratio must be before an agent rests, in basis points of the target (7.1.2) | 500 | 0 to 10 000 |
| World | `ProduceHaltRatio` | stock, as basis points of the output target, at which a recipe is not started (7.1.5) | 30 000 | 10 000 to `MaxBasisPoints` |
| World | `BasketTolerance` | how far `B_r` may differ from the cost of the baseline upkeep, in basis points of `B_r` (5.6.1) | 1 000 | 0 to 10 000 |
| Role | `Recipes` | the role's recipes (3.3) | none | at most `MaxRecipes`, each a known recipe |
| Role | `Upkeep` | the role's own upkeep, replacing the world's (5.2) | none | as `Upkeep` above |
| Role | `Trades` | extra tradable commodities (3.4) | none | known commodities |
| Role | `Informed` | shown rumours in the `Trade` view (owner `spec-information.md` 6.1) | false | a bool; needs the information module |
| Role | `StartMoney` | money per household at creation, and from the pool for a later agent (3.5) | 10 000 | 0 to `MaxMoney / MaxWeight` |
| Role | `StartStock` | goods per household for an initial agent (3.5) | none | each quantity 1 to `MaxQuantity / MaxWeight` |
| Role | `Strategy` | the registered strategy name (6.6) | `"default"` | a registered name |
| Region | `ExtraUpkeep` | goods added to every household's upkeep in the region (5.2) | none | as `Upkeep` |

Read here and owned elsewhere (so no row in this table): `MinWorkShare`, `WelfareWindow`
and `WindowCycles` (`spec-welfare-and-wages.md` 12.1, used by 7.1.2 and 7.1.3),
the role fields `Hireable`, `Employer`, `Credit`, `Cover`, `Saver`, `Needs` and `Ceiling` of
3.3 with their blocks (`spec-employment.md` 13, `spec-credit.md` 14, `spec-insurance.md` 13,
`spec-saving-and-investment.md` 10.4, `spec-settlements.md` 6.4),
`PriceWindow` and the belief parameters (`spec-beliefs.md` 11: 16 of them, with `CostFloor`
7 000 and `BeliefEvidenceShare` 5 000 as placeholders). `RestBand` and `ProduceHaltRatio`
are owned here (the rows above).

The placeholders, listed so that none is mistaken for a result: all of the above
defaults except the structural `Strategy` `"default"`; the bound `MaxUpkeepItems` of
16; the worked prices and quantities of section 16. The default `StartMoney` of 10 000
is ten ticks of the baseline upkeep at the default `B_r` of 1 000 (`spec-currency.md`
5.2) and is a guess. The default `OutputCoverTicks` of 1 is chosen small because it
sets the level of unsold inventory that a seller's price settles at: the stock rule
moves the price until the stock after sales lies in the band of 50% to 150% of the
target (`spec-beliefs.md` 5.5), so a target of three ticks of output means a seller
that settles holding one and a half to four and a half ticks of unsold goods. That is
a reading of the rules as written, **not a result**: `spec-stability.md` 6.2 finds the
stock rule alone an integrator (N-044) and the evidence rule an offset of 11% to 13%
below the clearing price in a stylised model (N-042), so the settled level of stock
and price is not exact (entry AGT-A10). The other defaults have no source: no source was opened that supplies one, and Lengnick's
numbers (`alpha` 0.9 and the others of his Table 1) are not adopted (section 17).

## 13. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). Errors name the
path, for example `roles.worker.upkeep[1].quantity`.

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a parameter outside its bound in section 12; an upkeep or start quantity below 1 (FR-GDS-003); more than `MaxUpkeepItems` upkeep items |
| `ErrDuplicateID` | a commodity twice in one upkeep, `StartStock`, `Recipes` or `Trades` list |
| `ErrUnknownCommodity`, `ErrUnknownRecipe` | a reference in an upkeep, `StartStock`, `Trades` or `Recipes` list |
| `ErrUnknownStrategy` | a role naming a strategy that is not registered |
| `ErrBasketMismatch` | a region whose `ReferenceBasket` differs from the cost of the baseline upkeep plus its extra upkeep by more than `BasketTolerance` |
| `ErrOverflow` | the largest weight times an upkeep quantity, a start quantity or a job input, summed with the targets of 7.3, above `MaxQuantity`; the start money times the largest weight above `MaxMoney`; the cost of the upkeep at the largest reference price above `MaxMoney` |
| `ErrModuleDependency` | a role with a `Hireable` or `Employer` field, a `Credit`, `Cover` or `Saver` block, or a `Needs` or `Ceiling`, while the module that owns it is off (the owner's rule: `spec-employment.md` 14.1, `spec-credit.md` 14, `spec-insurance.md` 13, `spec-saving-and-investment.md` 11.1, `spec-settlements.md` 9.1) |
| `ErrRoomShape`, `ErrStoreShape`, `ErrNotLimited` | a limited role whose base room per household cannot hold one tick of its upkeep or its start stock, and the other room rules of `spec-storage.md` 13.1 |

Warnings never fail validation (`spec-config.md` 7.2):

| Warning | Condition |
|---------|-----------|
| `no_upkeep` | no upkeep at the world, role or region level, so no household can starve (5.2) |
| `commodity_untraded` | a commodity in the tradable set of no role, that is not a discard output only |
| `upkeep_unsourced` | an upkeep commodity that is not raw, that no recipe makes and that no `StartStock` supplies: households will starve once the start goods run out (the producibility check of `spec-config.md` 7.5 sees recipes, not upkeep) |
| `start_stock_below_target` | a role's `StartStock` of an upkeep or input commodity is below one tick of its target, so the first tick is a purchase of everything (19) |
| `reserve_exceeds_money` | a role's `StartMoney` is below `ReserveTicks` times the cost of its effective upkeep at the region's reference prices: its households start below the reserve and buy no buffers until they have saved (AGT-A4, 19) |
| `strategy_default_unused_trades` | a role has a `Trades` entry but the default strategy sells it at once (3.4.2): a merchant role needs its own strategy |

The checks that a chain can sustain its households, and that a household's labour can
produce its upkeep, are `spec-viability.md`'s (`spec-config.md` 7.6).

## 14. Observability

14.1. **Events.** The table in observability-and-logging.md 4.3 has `AgentSpawned`,
`AgentBankrupted` and `UpkeepShortfall`, with their main fields in parentheses. This
spec owns the full field lists, as that document says, and they are below.

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `AgentSpawned` | an agent enters, including each initial agent after `WorldCreated` | `Agent`, `Region`, `Role`, `Weight`, `StartMoney` (what it received), `External`, `StoresLocked` (the stores of the role's row not given because the region's class lacks the flag, 0 with no gate; `spec-storage.md` 3.2.3) |
| `AgentBankrupted` | an agent is removed entirely (3.6): the only event that removes the agent, whatever the cause; the others of the removal accompany it, in the order of `spec-world.md` 6.11 item 8 | `Agent`, `Region`, `Role`, `Weight` (at removal), `Cause` (`insolvent`, `starved`, `consumer`, `population`; `spec-spawner.md` 13.1.1), `Money` and `GoodsListed` (the lots passed to the estate: one per distinct lot key, whatever the unit count; units travel in `WeightChanged.Units`) |
| `WeightChanged`, `RoleChanged`, `EntryLimited` | a weight change without removal, a role change, and the aggregate of the spawner's silent limits | defined and owned by `spec-spawner.md` 13.1; listed here only because they act on the agent state of 3.7 and 3.8 |
| `UpkeepShortfall` | once per region and role per tick in which at least one household was unfed, after the consumption phase, in order of region and role | `Region`, `Role`, `Households` (the sum of the weights of the role in the region), `Unfed` (households), `Agents` (agents with `Unfed` above 0), `MaxHunger` (the largest `Hunger`) |
| `ProductionHalted` | once per region and role per tick in which the production halt (7.1.5) declined the job request of at least one free household, at P2.8a; never when none was | `Region`, `Role`, `Free`, `Halted`, and per recipe skipped the households for which it was; defined and owned by `spec-observability.md` 4.4.6 |

Events carry no map, no pointer and no wall-clock time, only identifiers and
fixed-point values, and are emitted after the transition (observability-and-logging.md
4.1, 4.4). They are constructed only when an observer was supplied
(observability-and-logging.md 4.5). `UpkeepShortfall` is bounded by regions times
roles, so it is safe as a source of a metric attribute (observability-and-logging.md
5.4.1). It is not emitted when everyone is fed, so a healthy world costs nothing.
The goods consumed are already in `BoundaryFlow` under the reason `consumption`, by
region and commodity, the goods discarded under `disposal`, and with the demand module the wants consumed under `want_consumption`
(`spec-ledger.md` 12.1), so no event per consumption exists.

14.2. **No per-agent event.** There is no event for a plan, an intent, a discard or a
revision: the orders have `OrderPlaced` and `OrderRejected`, the beliefs have
`BeliefsUpdated`, and the volume of anything per agent would be per agent per tick.

The rejection reasons an `OrderRejected` of an agent's intent can carry, in the order
of `spec-clearing.md` 3.2, are: `unknown_agent`, `side_not_allowed` (not for a
household), `unknown_commodity`, `not_tradable` (after `unknown_commodity`, and
never for a pseudo-agent), `not_permitted` (after `not_tradable`, and never for a
pseudo-agent), `invalid_price`, `invalid_quantity`, `duplicate_order`, `overflow`,
`holding_limit`, `receipt_limit`, `insufficient_funds`, `insufficient_stock` and `no_room`. A consumer's strategy can cause any of
them; the default strategy causes none of `not_tradable`, `duplicate_order`,
`insufficient_funds`, `insufficient_stock`, `no_room`, `overflow`, `holding_limit` and `receipt_limit` (FR-AGT-044), and a right in
force can still refuse an order of the default as `not_permitted`, because no rule of the default
reads a right (`spec-institutions.md` 6.4). The refusals of the credit and cover fields of a plan
(`credit_off`, `insurance_off` and the reasons of `spec-credit.md` 5.2 and
`spec-insurance.md` 5.2) are `Debug` lines (14.3), not rejections of an order.

14.3. **Logs.** Everything per agent is `Debug`, guarded by `Enabled`, with a constant
message:

- "work declined", with `agent`, `recipe` and `reason` (`rest`, `halted` and `no_recipe`, which
  the work report of 7.1.3 gives, and the refusal reasons of `spec-production.md` 4.2, `not_on_role`
  and `not_permitted` among them). The strategy is pure and logs nothing: the world logs the line;
- "intent dropped", with `agent`, `commodity`, `side` and `reason` (`not_tradable`);
- "upkeep unmet", with `agent`, `commodity` and `quantity` (the shortfall);
- "discard refused", with `agent`, `commodity` and `quantity`;
- "request refused" and "offer refused" of the credit fields (`agent`, `reason`,
  `spec-credit.md` 16.1.1) and "cover refused" of the cover fields (`agent`, `reason`,
  `spec-insurance.md` 15).

One `Warn`, bounded by the configuration (observability-and-logging.md 3.2.2): "households unfed",
**per region and role**, never per agent, emitted when the role has had a nonzero
`Unfed` in the region for 10 consecutive ticks, at the tenth tick and every tenth after.
Attributes `region`, `role`, `households` and `unfed`. It is actionable: it names where,
who and how many. No `Info` and no `Error`: nothing here is a lifecycle or an
inconsistency the library repaired.

14.4. **Log keys.** `households` and `unfed` (integers), and `region`, are in the
table of observability-and-logging.md 3.3.2; `role` was there already. The `reason`
values above are exported constants of the root package (`event.go`, a closed set
of `spec-observability.md` 4.2.3); `internal/logkeys` holds the key strings only. `rest`, `halted`
and `no_recipe` are log-only: values of the `reason` log key, not of a metric attribute.

14.5. **Metric.** None is defined here. `economysim.household.unfed`, an up-down
counter by role fed by `UpkeepShortfall`, is in the catalogue of
observability-and-logging.md 5.3 (and 5.3.1), with `economysim.household.count`
(`spec-scale.md` 9.4). Because the event is not emitted when everyone is fed, the
adapter treats a tick with no event for a region and role as zero unfed there
(5.3.1). `economysim.production.halted`, a counter by role and recipe fed by the
rows of `ProductionHalted` (a household counts once for each recipe it skipped), is
in the same catalogue.

14.6. **Snapshot.** The fields are `spec-observability.md`'s: per region and role the
`RoleRow` (5.3.4: households and agents, households fed and unfed, the largest hunger, the
households held to work by the floor `MinWorkShare`, 7.1.3, N-065, which makes that silent
damper observable (FR-STB-013), and with the subsistence module on the households in the
own-use mode; the households blocked are the event `OwnUseChanged`, not a snapshot field).
Per agent, `Agent(a)` returns an `AgentState` (`spec-integration.md` 4.7.2) with `Unfed`,
`Hunger`, `Privation` and, with the modules on, `OwnUse`, `OwnUseJobs`, `Cover`, `Secure`
and the journey. Bounded by the configuration for the aggregates.

14.7. **Cost with no observer.** The strategy calls, the plan and the view use
scratch held by the world and allocate nothing (NFR-AGT-001). Observation never
changes results: the draws are taken whether or not an observer is attached
(FR-AGT-066).

## 15. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, all plain data in a
fixed order with no maps:

- for each agent, in agent ID order: `ID`, the ledger account reference, `Role`,
  `Weight`, `External`, `Mortal`, the strategy reference (its registered name), `Unfed`,
  `Hunger`, `Privation` (hosted here, defined in `spec-spawner.md` 5.1), `RoleSince` (hosted here, defined in `spec-spawner.md` 5.2.2a), `DeathCarry` (hosted here, defined in `spec-population.md` 3.8), `OwnUse` and `OwnUseJobs` (hosted here, defined in `spec-subsistence.md` 5.3, 15), the `Journey` of an agent on a migration journey (destination and arrival tick, hosted here, defined in `spec-migration.md` 14), the membership pair of a body (hosted here, defined in `spec-institutions.md` 12.1), the stores
  the agent hosts (hosted here, defined in `spec-storage.md` 15.1), and the directive if any (its
  job request if it has one, intents, discards, holds, credit request and offer, and `Sticky`); each is saved once, in the agent record of section 3 of `spec-checkpoint.md`
  (4.2 tag 3), and the module's own section lists only the state it keeps beside it. The
  want carries of an agent are the demand module's and are listed in `spec-demand.md` 12,
  and the rumours of the last view of an external informed agent are saved by the
  integration interface (`spec-checkpoint.md` 4.2 tag 22, `spec-integration.md` 13);
- the table of strategy names, sorted, that the agents refer to, so that the name of an
  external agent's strategy is checked at a restore as the roles' are (`spec-checkpoint.md`
  4.2 tag 3, 7.3);
- for each (region, role), in region then role order: the run of consecutive ticks with
  a nonzero `Unfed` that the `Warn` of 14.3 uses (diagnostic: it affects no result, and
  is saved so that a resumed run logs as the original did).

The beliefs, job, residues and welfare history are saved by the specs that own them
(3.2). **Not saved:** the effective upkeep, the tradable sets, the targets, the budgets,
the views, the plans, the planned recipe and the work share, all of which are derived
from the saved state and the configuration. **The default strategy has no state**
(ADR 0002 decision 7). A consumer's strategy that has state saves it through the
optional pair of that decision; the directed strategy's state is the mailbox above, in
the agent.

A world restored between ticks gives the same orders, jobs and consumption as an
uninterrupted one (FR-AGT-068).

## 16. Worked examples

All numbers are placeholders, chosen to be easy to follow, and were checked by a short
script of the spec's integer rules (not code of the library). The defaults of section
12 apply. Commodity IDs are neutral.

### 16.1 A trade phase: a poor household

One household (`w = 1`) of a role with upkeep `staple` 3 and `fuel` 1, and one recipe
`make_item`: duration 1, input `ore` 2, output `item` 1 (cost share 10 000). Beliefs
`Centre`: `staple` 300, `fuel` 100, `ore` 200, `item` 900. The agent holds `staple`
4, `fuel` 0, `ore` 0, `item` 1, and money 2 500. It is free (`StartsIn` 1). The effective
yield is 10 000.

| Step | Value |
|------|-------|
| Planned recipe | `net = 1 * 10 000 * 900 / 10 000 - 2 * 200 = 500`, positive: `r*` is `make_item`, `k*` = 1 |
| `UpkeepCost` | `3 * 300 + 1 * 100 = 1 000`; `ReserveMoney = 3 * 1 000 = 3 000` |
| Job needs | `Job(ore) = 1 * 1 * 2 = 2`, others 0 |
| Targets | `UseTgt`: staple `3 * 3 = 9`; fuel `1 * 3 = 3`; ore 2. `Item`: no use, `Make = 1`, `OutTgt = ceil(1 * 1) = 1` |
| `Keep` (reserve plus use target) | staple `3 + 9 = 12`; fuel `1 + 3 = 4`; ore 2; item 0 |
| Gap and surplus | staple `12 - 4 = 8`; fuel `4 - 0 = 4`; ore `2 - 0 = 2`; item surplus `1` |
| Tiers | staple: T1 = `max(0, 3 - 4) = 0`, T2 = 0, T3 = 8. fuel: T1 = `1 - 0 = 1`, T2 = 0, T3 = 3. ore: T1 = 0, T2 = 2, T3 = 0 |

Suppose the limit draws are `fuel` 105, `ore` 195, `staple` 310. Tier 1: `fuel` 1 at
105, money left 2 395. Tier 2: `ore` 2 at 195 = 390, left 2 005. Tier 3: the excess is
`max(0, 2 005 - 3 000) = 0`, so the cap is 0 and **no buffer is bought**. The household
places two bids, `fuel` 1 at 105 and `ore` 2 at 195, and an ask of 1 `item`. It holds 4
`staple` and eats 3 this tick; the next tick it will have 1 and must buy 2 more as
tier 1. It is saving first.

For the ask: the price window of `item` has low 800, high 1 000 and the market price
is 900, so `pos = (900 - 800) * 10 000 / 200 = 5 000`. Asks are not scaled by `pos`
(7.3.5): the quantity is the whole surplus, 1.

### 16.2 A trade phase: a richer household

The same agent with money 8 000. After tiers 1 and 2 (105 and 390), `avail = 7 505`.

| Step | Value |
|------|-------|
| `e` | `7 505 - 3 000 = 4 505` |
| `H` | `SpendHalfTicks * UpkeepCost = 10 * 1 000 = 10 000` |
| `e * H / (e + H)` | `4 505 * 10 000 / 14 505 = 3 105` (integer) |
| `cap` | `5 000 * 3 105 / 10 000 = 1 552` |
| Favourability | `pos` of the staple's price is 5 000 (an assumed neutral window), so `buyshare = 2 000 + 8 000 * 5 000 / 10 000 = 6 000` |
| Wants | fuel `floor(3 * 6 000 / 10 000) = 1`; staple `floor(8 * 6 000 / 10 000) = 4` |
| Allocation (canonical order, fuel before staple) | fuel 1 at 105 = 105, cap left 1 447; staple 4 at 310 = 1 240, cap left 207 |

It buys 1 unit of fuel for the buffer on top of its tier-1 unit (2 in all) and 4 staple.
The cap is 1 552 of an excess of 4 505: about a third. The household keeps the rest.

### 16.3 A cohort: unfed households

A cohort of `w = 40` with the same upkeep needs `U_staple = 120` and `U_fuel = 40`. It
holds 100 staple and 40 fuel. Consumption: staple `min(100, 120) = 100`, fuel
`min(40, 40) = 40`.

```
Fed   = min( floor(100 / 3), floor(40 / 1) ) = min(33, 40) = 33
Unfed = 40 - 33 = 7
```

Seven households are unfed; `Hunger` becomes 1. The 100 staple were all eaten, so 33
fed households ate 99 and the unfed ones ate the last unit, which does not complete a
basket for any of them.

### 16.4 A cohort: the work share

A cohort of 40 with `TargetRatio` 15 000 and `RestBand` 500. `T_lo = 15 000`, `T_hi =
15 000 * 10 500 / 10 000 = 15 750`. Its ratio is 15 300 (above the target, below the top
of the band): `s = (15 750 - 15 300) * 10 000 / (15 750 - 15 000) = 6 000`. It requests
`ceil(40 * 6 000 / 10 000) = 24` executions: 24 households work and 16 rest. At ratio 15
750 it rests entirely; below 15 000 all 40 work.

## 17. Prior art and differences from bazaarBot

What was opened, and what was taken. "Read" means the text was in front of the writer;
a fetch tool that summarises is marked as such. No default in this spec is set from any
of these sources.

| Source | Status | Taken | Not taken |
|--------|--------|-------|-----------|
| bazaarBot (`bazaarbot-reference.md` 3, 5, 10, 11) | the repository's record of the source; the source was not run | The shape of order generation (surplus and shortage against an ideal; favourability from the position in a range); one order per good | The 10-money fine for an idle role; inventory capacity; `max_size`; the limit formula (defect 8); roles hard-coded in `Logic*` classes |
| Lengnick (2013), "Agent-based macroeconomics: a baseline model" | the household part read from text extracted from the PDF: equations 1 to 4 (liquidity), the shopping and search part of 2.2, equations 11 and 12 (consumption), 2.3 (the daily purchase), the end of month, and Table 1 and Table 2. Greek symbols are partly lost in the extraction (`alpha` is written explicitly in equation 11 and Table 1, as 0.9). The results sections were not read in full | A hard liquidity constraint (a household can only spend what it holds); consumption rising with wealth at a decaying rate; planned demand cut to what is affordable; firms comparing inventory with demand (already in `spec-beliefs.md`) | The power function and `alpha` 0.9 (no floating point; no source for our default); the monthly cycle; the network of trading partners; the reservation wage; profit distribution in proportion to wealth |
| abcEconomics, `Household` page | opened through a summarising fetch: `consume(utility_function, input_goods)`, `create_cobb_douglas_utility_function`; the class is production-free and `NotEnoughGoods` is raised when goods are short | That a household is a pure consumer with a utility function (the shape of `spec-demand.md`'s wants); that consuming more than is held is an error, here counted | Cobb-Douglas utility (a want function belongs to `spec-demand.md`) |
| abcEconomics, `Trade` and `Firm` pages | opened through a summarising fetch: `sell` and `buy` commit goods ("the good offered is blocked"), `accept` and `reject` settle in subrounds, `produce` raises an error if goods are short | That an offered good is blocked until settled, which is our reservation (`spec-clearing.md` 3.3); that decisions are made in rounds of subrounds | Bilateral offers with a named receiver (our market is a double auction) |
| ASSUME, `assume/common/base.py` (GitHub raw source) | opened; the fetch tool reproduced the code of `BaseStrategy` and `BaseUnit`. The documentation page "bidding agents" returned HTTP 404 | One strategy interface for every kind of actor: `calculate_bids(unit, market_config, product_tuples)` returns an order book; `calculate_reward(unit, marketconfig, orderbook)` and `set_dispatch_plan(marketconfig, orderbook)` give the outcome back to the unit; a unit keeps its strategies indexed by market (`bidding_strategies`). This supports the split of `Trade` and `Revise`, and a strategy chosen per role (our names) | Reinforcement-learning strategies and the "foresight" of observations; units that dispatch on a time grid |
| Doran and Parberry (2012) | read in full in the earlier specs (`spec-beliefs.md` 19) | Nothing new here | |

**Differences from bazaarBot** (item numbers are those of `bazaarbot-reference.md`
section 11):

| Item | bazaarBot | This spec |
|------|-----------|-----------|
| 5 | A good not in the starting stock is never bid on; a missing ideal reads as null | The tradable set is derived and explicit (3.4); a missing ideal is a target of 0 (7.3.1) |
| 6 | Consuming more than is held succeeds and the shortfall is ignored | Consumption is cut to what is held and the shortfall counted (5.3, 5.4) |
| 8 | A bid's limit when space is short is `floor(space / shortage)` | No capacity (section 8); the limit is a quantity bounded by the money at the limit price |
| 13 | A fine of 2 money for an idle role destroys money | No fine; an agent that cannot work earns nothing and may starve |
| 17 | One role needs food and does not consume it | Upkeep is data, the same mechanism for every role (5) |
| ideal stock | A fixed ideal per good and role | A target computed from upkeep, the next job and cover periods (7.3) |
| quantity of an ask | Scaled by favourability | The whole surplus (7.3.5); favourability only for buffer bids (7.5.2) |

## 18. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9. Each entry has the fields
of its section 3: kind, loop with the sign of each link (R reinforcing, B balancing),
delay, owner, damper by `STB-Dnn`, what is undamped, the condition, the test (a
scenario of its section 8) and a status. The register owner assigns the `STB-Lnn`
numbers; the entries here are named `AGT-An` and say which existing entry each merges
with (its 9.9). **Status is `hand` throughout: nothing here has been run by this spec.**
Where the register has run a model (`spec-stability.md` 6), the entry cites it. No
default is claimed to be stable (N-027).

**Dampers this spec adds or specifies** (for the catalogue of `spec-stability.md` 5):

| ID | Damper | Home | Status |
|----|--------|------|--------|
| D24 | Production stop: a recipe is not started when its output stock is at or above `ProduceHaltRatio` of the output target | AGT 7.1.5 | **specified** here (was "proposed, owner undecided") |
| D09 | Hysteresis band: an agent rests only when its ratio is `RestBand` above its target, and a cohort answers in proportion | AGT 7.1.2, 9.3 (was "left to the strategy", WEL 5.1) | **specified** here for the work decision; SPN's role-change hysteresis is still open |
| D26 | Survival rule: work while unfed or below the reserve money | AGT 7.1.3 | specified (the register lists it so); `spec-stability.md` 6.1 finds it unit-elastic, so **not a damper of STB-L14**; it is a household rule |
| D27 | The budget stack and the saving rule: upkeep first, job inputs second, buffers only above a reserve and under a concave cap | AGT 7.4 | specified |
| D28 | Proportional work share for a cohort | AGT 7.1.2, 9.3 | specified (a form of D09 and D08) |

D13 (a floor on working ticks) is **adopted by WEL** as `MinWorkShare` (`spec-welfare-and-wages.md`
5.3, default 4 000 basis points, a placeholder) and **applied** by this spec in 7.1.3. D29,
the derived welfare window, gives the `T` of the floor (`spec-welfare-and-wages.md` 2.3.3).
The register (`spec-stability.md` 5) holds the rows; this section lists only what this
spec specifies.

**AGT-A1. A staple's demand is inelastic.** *Kind:* hazard (it removes the interior
anchor). *Loop:* price (+) spending on the staple at a fixed quantity and (-) cash
left; when cash binds, spending is fixed, so quantity (-) price and elasticity is 1;
otherwise 0 (11.6). *Delay:* none in the rule. *Owner:* AGT 7.3, 7.4, 11.6; merges
with STB-L05 and STB-L06 (shortage with no supply, overhang ratchet) and with the
conditions of STB-L14 (`e > 1`) and STB-L21 (`es < ed`), which a staple fails.
*Damper:* D04 (the belief bounds), D01 (the cash limit), D24 (the halt, A2), exit
(`spec-spawner.md`, whose entry and exit are slow and capped, not an anchor, N-043). *Undamped:* the interior of the price range. *Condition:* a
price-responsive flow acts before a bound is reached, that is the halt, exit or an
elastic demand: `spec-demand.md` supplies one for discretionary goods (D30), and for a pure staple the rationing of tier 1 is rejected (11.6.1). *Test:* S04 (records the
elasticity), S16. *Status:* hand; the model of `spec-stability.md` 6.2 runs 1 and 2
shows the ratchet for inelastic flows.

**AGT-A2. The production halt.** *Kind:* loop. *Loop:* output stock (+) halt of the
recipe (-) supply (-) stock; B. *Delay:* the recipe's `Duration` (a halted recipe is
not started; a started job still completes). *Owner:* AGT 7.1.5. *Damper:* D24 itself,
D07 (the belief stock rule acts at 15 000 basis points of the target, the halt at
30 000), D16 (spoilage). *Undamped:* where the threshold sits decides whether an
overhang ends at the floor or at the ceiling in a model with inelastic flows
(`spec-stability.md` 6.2, run 1), and a halt that all agents of a role reach in the
same tick stops them together (a cobweb source). *Condition:* the halt is reached
before the price reaches the floor of the reference band, and a restart is gradual
(the stock falls by sales and spoilage). *Test:* S16, S05. *Status:* hand, the
threshold is a guess.

**AGT-A3. The poverty trap and the reserve.** *Kind:* loop. *Loop:* money below the
reserve (+) no buffers (+) less stock (-) cash left for inputs; and money (+) inputs
bought (+) output (+) income (+) money. Tier 1 and 2 are outside the saving rule, so
the second loop is not blocked by it. R when it runs down, B when it runs up. *Delay:*
a production cycle. *Owner:* AGT 7.4; merges with STB-L11. *Damper:* D27, D11 (relief),
D15 (the floor under market failure, `spec-subsistence.md`, not written). *Undamped:*
an agent with no money, no goods and no relief. *Condition:* relief or a start grant
keeps the agent above the cost of one execution. *Test:* S14. *Status:* hand.

**AGT-A4. Saving, money out of circulation and the paradox of thrift.** *Kind:* loop.
*Loop:* households' reserve money (+) money held (-) money in circulation (-) sales
and income (-) ability to hold the reserve. R, bounded: each household holds at most
`ReserveMoney` plus what it chooses to hold above it, and the cap of 7.4.4 lets the
excess go. *Delay:* a tick. *Owner:* AGT 7.4; touches STB-L31 (wealth held out of
circulation). *Damper:* D27 (the cap is bounded by the excess, so a household above
its reserve spends), D11, D12. *Undamped:* if the sum of the households' reserves
exceeds the money in the world, no household reaches its reserve, buffers are never
bought and the work rule runs at full. *Condition:* total household money at least the
sum of the reserve money; the warning `reserve_exceeds_money` of section 13 checks it
at the reference prices for the starting money. *Case:* SAV-A1 of
`spec-saving-and-investment.md` 14, where a target above the reserve changes the excess of 7.4.4.
*Test:* S19 (new, below). *Status:* hand.

**AGT-A5. The closed economy fixes the mean welfare ratio at 1 (N-054).** *Kind:*
hazard. *Loop:* none; an identity (11.7): income equals upkeep spending, so the target
is unreachable, the work share stays at 10 000, and the surplus is unsold. *Owner:*
`spec-viability.md` 4.7; AGT 11.7 accepts the glut. *Damper:* D24 bounds the glut;
spoilage D16; spending beyond upkeep is `spec-demand.md`'s (module on: the ratio can exceed 1, ceiling
`1 + Wref / U`, DEM-A6). *Undamped:* with the module off, the glut until the halt, and the
target-income rule never rests. *Condition:* a world where upkeep is the only demand either sets
targets at 10 000 or accepts the halt as the only brake.
*Test:* S19 (new): the default target, no DEM, a bounded glut. *Status:* hand.

**AGT-A6. Food first and the urgency bid.** *Kind:* amplifier. *Loop:* hunger (+) bid
at the top of the belief range (+) staple price (+) cash spent on the staple (-) cash
for other goods; and the staple price (+) hunger. R, bounded by the cash limit and the
belief ceiling. *Delay:* a tick for the hunger counter. *Owner:* AGT 7.3, 7.5.3.
*Damper:* D04 (the ceiling), D01. *Undamped:* demand for non-subsistence goods falls in
a staple shortage, by design. *Condition:* the staple's price stays below the ceiling
in a shock. *Test:* S04, S09. *Status:* hand.

**AGT-A7. Recipe choice by margin is an occupation switch inside one agent.** *Kind:*
loop (N-034). *Loop:* the net value of a recipe (+) its choice (+) its output (-) its
price (-) its net value. B with a delay. For a role with two or more recipes the rule
is a maximum, so it switches all at once when the order of two values changes.
*Delay:* the job's duration (an agent chooses only when free). *Owner:* AGT 7.1.4;
cousin of STB-L36 (occupation cobweb), inside a role. *Damper:* D03 (agents' beliefs
move at different times, so they cross the threshold at different times), D08 (spread
of targets only changes whether they work, not what). *Undamped:* no hysteresis in the
recipe choice: two recipes of nearly equal value flip with each belief move, and a job
of long duration holds the flip for its length. A switching margin is not specified;
it would need the previous recipe as state. *Condition:* recipes differ in value by
more than the belief noise, or roles with several recipes are few. *Test:* S06 (a role
with two recipes). *Status:* hand.

**AGT-A8. The synchronised purchase.** *Kind:* amplifier. *Loop:* jobs complete in one
tick (+) the agents are free in one tick (+) inputs are bought in that tick (+) a pulse
of demand (+) a pulse of price. *Delay:* the job's duration. *Owner:* AGT 7.3.2; merges
with STB-L22 (lockstep of jobs). *Damper:* D22 (the staggered warm start of
`spec-viability.md` 8.5), different durations. *Undamped:* a world not started from the
warm start. *Condition:* completions are spread. *Test:* S18. *Status:* hand.

**AGT-A9. The buffer pulse (a bullwhip).** *Kind:* amplifier. *Loop:* a rise in use
(+) the target `UseTgt = U * CoverTicks + Job` (+) a purchase larger than the rise, once
(+) the input's price. *Delay:* one tick. *Owner:* AGT 7.3. *Damper:* D27 (the spend cap
limits tier 3), favourability (7.5.2). *Undamped:* tier 2 is the job lump and is not
capped. *Condition:* cover is small against the shock. *Test:* S01 (a step in upkeep).
*Status:* hand.

**AGT-A10. The level of unsold stock and the evidence offset.** *Kind:* bias. *Loop:*
none. A seller offers its whole surplus (7.3.5) and the belief rule moves its price
until its stock after sales lies in the 50% to 150% band of its target, so the unsold
stock settles at about `OutputCoverTicks` ticks of output (12). **Do not rely on this
as exact.** `spec-stability.md` 6.2 finds the stock rule alone is an integrator
(STB-L04, N-044) and that the evidence rule of `spec-beliefs.md` 5.4 settles the price
11% to 13% below the clearing price in a stylised model (STB-L02, N-042); the level of
unsold stock here is a property of the rules as written and not a result. *Owner:*
AGT 7.3, 7.6; BEL 5.4. *Damper:* D07, D17. *Undamped:* the offset. *Test:* S01.
*Status:* hand, with a model by `stb`.

**AGT-A11. Hunger, removal and demand.** *Kind:* loop. *Loop:* unfed households (+)
removal or loss of weight (POP/SPN) (-) demand for the staple (-) its price (-) hunger.
B, Malthusian by design, with a long delay and a source of cycles. *Owner:* AGT 5.5
supplies the signals (`Unfed`, `Hunger`, `Privation`); the removal ladder is `spec-spawner.md` 5 (its entry
SPN-A7) and the Malthusian loop is `spec-population.md`'s. *Damper:* D11, the ladder's exit cap. *Condition and
test:* in SPN and POP. *Status:* hand-off.

**AGT-A12. A job longer than the reserve.** *Kind:* hazard. *Loop:* a long job (+)
inputs paid at its start (-) money for upkeep until it completes. Tier 2 has no
reserve (7.4.3, open question 15). The household protects its food in stock (7.1.6),
not its money. *Delay:* the duration. *Owner:* AGT 7.4; merges with STB-L19 and L20
(N-032). *Damper:* `spec-viability.md` 8.3.1 starts money at `U * (d_max + buffer)`;
D11. *Undamped:* any household that starts or ends below it. *Remedy:* a household with a `Saver` block holds
its own buffer (`spec-saving-and-investment.md` 4.3.3). *Test:* S11. *Status:* hand.

**AGT-A13. Cohort work share.** *Kind:* amplifier made smaller. *Loop:* ratio crosses
the band (+) the cohort's households work or rest together. With the proportional share
the cohort moves in steps of `1 / w` as the ratio crosses the band, not in one step.
*Owner:* AGT 7.1.2, 9.3; merges with STB-L34. *Damper:* D28, D10 (the floor `K`), D08.
*Undamped:* the cohort's single gate draw and single decision (`spec-beliefs.md` 9.3).
*Test:* S10. *Status:* hand.

**Checklist items (spec-stability.md 9.6 and 9.7).** Delays that match a neighbour:
`CoverTicks` (3) equals `BeliefMaxAge` (3), so a purchase for cover is made while the
market price is still fresh evidence; `Duration` against the welfare window `T` (A12);
`ShelfLife` against `CoverTicks` (a cover longer than the shelf life of a perishable
buys goods that spoil, so validation could warn: open question 17). Every damper
parameter named here has a row in the parameter-owner table of the handoff.

**New scenarios proposed to the register.** S19: the default `TargetRatio` in a closed
economy with upkeep as the only demand, run with the halt on and off; asserts a
bounded glut and that the halt stops output. S20: starting money below `ReserveTicks`
times the upkeep cost for every household; asserts that households work and reach the
reserve, or that the world reports the unreachable reserve.

**Loops this spec creates that nothing yet dampens:** A1 (the interior of the price
range for a staple), A5 (the glut until the halt), A7 (no hysteresis in recipe
choice), and A3 for a household with no relief.

## 19. What the warm start needs (for `spec-viability.md`)

`spec-viability.md` 8 owns the warm start: the `Config.WarmStart` section, the seed
prices, the money, the stock, the stagger and the jobs. This section says only what an
agent offers it and what this spec requires of it. Nothing here restates a rule of
VIA's; where the two could differ, VIA 8 wins.

| Slot | What the agent offers | Requirement from this spec |
|------|-----------------------|----------------------------|
| Money | `Role.StartMoney` per household, replaced for a group by `WarmGroup.Money`; paid by `initial` | A group's money should be at least `ReserveTicks * UpkeepCost` at the seed prices, or its households start below the reserve and buy no buffers (the warning `reserve_exceeds_money`, section 13); `spec-viability.md` 8.3.1 sets money to `inputs + U * (d_max + buffer)`, which satisfies it when the buffer is at least `ReserveTicks` |
| Stock | `Role.StartStock` per household, replaced by `WarmGroup.Stock`; paid by `initial` | At least `Reserve + UseTgt` of each upkeep commodity and `Job` of each input of 7.3 at the group's steady beliefs, or the first tick buys it all as tier 1 and 2 (the warning `start_stock_below_target`). VIA's `StockCoverTicks` of 5 exceeds the default `CoverTicks` of 3, so its table satisfies it |
| Beliefs | one per commodity in the tradable set | Seeded by `spec-beliefs.md` 8; `WarmStart.Prices` replaces the reference price (VIA 8.2). Beliefs outside the cost band are clamped (VIA 8.2.3, N-055) |
| Jobs | a job in progress at creation | VIA 8.5, built in the construction phase (`spec-ledger.md` 5.2.4): inputs by `initial`, taken by `recipe_input`, tools worn with `WearItems` (5.4.3). VIA 8.5: `JobSpec.InProgressBP` and the spreading rule `V[floor(j * |V| / busy)]` with the virtual start; **this spec's earlier draft of a different rule is withdrawn**. The inputs and wear of a job in progress are granted by `initial` and taken by `recipe_input` in one construction step (VIA 8.5.3), so the books balance from tick 0 |
| Welfare history | empty, padded by `DefaultRatio` | `spec-welfare-and-wages.md` 2.2 |
| Starting wage | the regional wage that prices labour in the cost anchor (`spec-production.md` 9.3) | `spec-viability.md` 8 has no field for it: whenever the pooled observed wage of the region is unknown (no working household-tick in any window of the region, or a non-positive pooled figure) the anchor is 0 and only the reference band applies (FR-PRO-049); this is a pure function of the welfare rings, not a latch, and nothing is saved (`spec-checkpoint.md` 4.5 F5). A solved wage per region (the dual of the labour row, VIA 3.2) could seed it; asked of VIA in the sweep addendum |
| Creation draws | the target-ratio draw, then the residue draw (3.5.2) | After the warm start is applied (VIA 8.6 item 5, N-009) |

The targets of 7.3 are pure functions of data and beliefs, so VIA can compute `Reserve`,
`UseTgt` and `OutTgt` at its solved prices without running a world: they are exposed as
`internal/household` functions over primitive slices.

VIA's assumptions about this spec (`spec-viability.md` 3.3, and its handoff 9) hold:
a group has a household upkeep with regional additions; `Role.Recipes` is a sorted list;
an agent is born with money, stock, an optional job, beliefs at the seed price, its
weight and its creation draws; quantities are per household and multiplied by the
weight; agents are created by group and index. One difference: upkeep here is the world
list, a role's replacement, and the region's extra (5.2), and the viability solver
reads the effective list per group.

## 20. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-AGT-001 | Every agent MUST have a unique agent ID that is never reused in a run, an integer weight of at least 1, a role and a ledger account, and the pseudo-agents of the regions MUST have the lowest IDs, followed by the counterparties of the external module (`spec-external-sector.md` 4.2.2) and then the treasuries of the governed jurisdictions (`spec-polity.md` 5.2.2). |
| FR-AGT-002 | An agent MUST hold no money balance and no quantity of its own: its money, goods and tools MUST be in its ledger account, and its region MUST be that of the account. |
| FR-AGT-003 | The tradable set of a role MUST be the effective upkeep commodities, the inputs, required tools and non-discard outputs of its recipes, its `Trades`, and with the demand module on the items of the wants in its ladder, in canonical order and without duplicates, and MUST NOT contain a discard output. (changed in round 5) |
| FR-AGT-004 | An agent MUST hold one belief for every commodity in its tradable set from its creation, and MUST NOT trade a commodity outside the set. |
| FR-AGT-005 | An initial agent of weight `w` MUST receive `w` times `StartMoney` of money and `w` times each `StartStock` quantity of goods, both by the `initial` reason, a belief for each commodity in its tradable set from the seed price with width `BeliefWidthInit`, no job unless a warm start gives one, an empty welfare history, and zero hunger. |
| FR-AGT-006 | At creation the world MUST draw, per agent and in agent-ID order, first the target-ratio value and then the residue value, whatever the configuration, and MUST draw nothing else for the agent. |
| FR-AGT-007 | A later agent MUST receive the lesser of `w` times `StartMoney` and the pool balance from its region's pool by the reason `start_money`, and no goods unless the spawner's rule grants some. |
| FR-AGT-008 | Removing an agent MUST abandon its job, end its titles, hires, claims, covers and facilities in the steps 2b to 2g of 3.6, move all its money, less what its creditors take in the waterfall of `spec-credit.md` 9.1, to the region's estate with the reason `estate_money` and all its goods with `estate_goods`, close its account and emit `AgentBankrupted`, the only event that removes the agent, with the events of its job, traineeship, contracts and steps 2b to 2g preceding it and `EstateOpened` after it, in the order of `spec-world.md` 6.11 item 8, and MUST conserve money and goods exactly. |
| FR-AGT-009 | A change of weight from `w` to `w'` MUST leave each money and goods holding at `floor(h * w' / w)`, the difference passing to or from the estate, MUST leave the beliefs, residues, welfare history and hunger unchanged, and MUST apply the job rule of `spec-production.md` 5.4. |
| FR-AGT-010 | A role with an empty `Recipes` list MUST be valid, and an agent of it MUST request no job (the default requests none); the world's refusal of a request from such a role is the `not_on_role` of the production admission (FR-PRO-048), which a role with an empty list refuses for every recipe. |
| FR-AGT-011 | The effective upkeep of an agent MUST be the role's list, or the world's if the role has none, merged with the region's extra upkeep by adding quantities of one commodity, in canonical order, each quantity multiplied by the weight, with checked arithmetic. |
| FR-AGT-012 | Each upkeep quantity MUST be an integer of at least 1, and the lists MUST have no duplicate commodity, or the configuration MUST be rejected. |
| FR-AGT-013 | In the consumption phase an agent MUST consume of each upkeep commodity the lesser of what it holds and its effective upkeep quantity, as a move to the `consumption` boundary, oldest first, and MUST NOT fail or take more than it holds. |
| FR-AGT-014 | `Fed` MUST equal the least over the upkeep commodities of the integer quotient of the quantity consumed by the per-household quantity, `Unfed` MUST equal the weight minus `Fed`, and an agent with no upkeep MUST have `Unfed` 0. |
| FR-AGT-015 | `Hunger` MUST increase by 1 in each consumption phase in which `Unfed` is above 0 and reset to 0 in one in which it is 0. |
| FR-AGT-016 | `UpkeepShortfall` MUST be emitted once per region and role per tick with a nonzero total `Unfed`, after the consumption phase, in order of region and role, with the households, unfed, agents and largest hunger, and MUST contain no map. |
| FR-AGT-017 | The `Warn` "households unfed" MUST be emitted per region and role and never per agent, only after 10 consecutive ticks with a nonzero `Unfed`, at the tenth and every tenth tick after, with the region, role, households and unfed. |
| FR-AGT-018 | Validation MUST reject, with `ErrBasketMismatch`, a region whose reference basket differs from the cost of the baseline upkeep plus its extra upkeep at its effective reference prices by more than `BasketTolerance`. |
| FR-AGT-019 | The `Strategy` interface MUST have exactly the three methods `Work`, `Trade` and `Revise`, and the library MUST ship `DefaultStrategy` with a compile-time assertion. |
| FR-AGT-020 | Every strategy method MUST receive the world's random source and no other, and a run with the default strategy MUST give identical orders for the same seed, configuration and calls. |
| FR-AGT-021 | `Work` MUST be called once per tick, before the starts of the production phase, for each agent that has no active job and did not continue or complete one in that tick, and for no other agent. |
| FR-AGT-022 | `Trade` MUST be called once per tick for each non-pseudo agent after the production phase, in agent-ID order, and its intents MUST be submitted in plan order after its discards. |
| FR-AGT-023 | `Revise` MUST be called once per admitted order of a household agent after all markets have cleared (an estate's ask and a counterparty's order have no outcome and no draw), in order of region, commodity and admission, with the order, its outcome, the market report, the stock, the target, the unit cost and the one draw the world took. |
| FR-AGT-024 | The views MUST hold plain data only, and a strategy that writes to a view MUST NOT change the world's state or any later view. |
| FR-AGT-025 | An intent for a commodity outside the agent's tradable set MUST be rejected with the reason `not_tradable` and an `OrderRejected` event, and MUST NOT reach clearing. |
| FR-AGT-026 | A second intent of a plan for one commodity MUST be rejected with `duplicate_order`. |
| FR-AGT-027 | A plan's discards MUST be executed before its orders, each as a move to the `disposal` boundary, and a discard of more than the agent holds MUST be refused whole without changing the ledger. |
| FR-AGT-028 | A job request for a recipe not on the agent's role list MUST be refused with the reason `not_on_role` and MUST NOT start a job. |
| FR-AGT-029 | A belief returned by `Revise` MUST be clamped to the invariant and bounds before storage, and a return of the current belief MUST leave it unchanged apart from the re-clamp. |
| FR-AGT-030 | A role's `Strategy` MUST be a registered name (the built-in `default` and `directed` are registered beside those of `WithStrategy`), an unknown name MUST be rejected with `ErrUnknownStrategy`, and the names MUST be part of the effective configuration and its hash. |
| FR-AGT-031 | The default strategy MUST keep no state, and the same view and the same random source MUST give the same return. |
| FR-AGT-032 | A pseudo-agent (an estate, a counterparty or a treasury) MUST NOT be passed to a strategy. |
| FR-AGT-033 | If the welfare ratio is absent (the agent has no upkeep), the default `Work` MUST request a job whenever a recipe is eligible. |
| FR-AGT-034 | When the welfare ratio is present, the work share MUST be 10 000 when the agent's `Hunger` is above 0, or its money is below the reserve, or the working ticks in its effective window are fewer than `MinWorkShare * T / 10 000`; otherwise 10 000 below the target, 0 at or above the target raised by `RestBand`, and linear between; and the executions requested MUST be `ceil(w * s / 10 000)`, at least 1 when `s` is above 0. The effective window `T` MUST be `max(WelfareWindow, WindowCycles * longest Duration on the role's list)`. (changed in the sweep: the floor and the window) |
| FR-AGT-035 | The default recipe MUST be the eligible recipe with the greatest net value per tick, where the net value is the non-discard outputs at the effective yield and the agent's belief centres less the inputs and the wear of the tools, compared by cross-multiplication with ties to the lower recipe ID, and a recipe MUST be eligible only if its net value is positive or it has no inputs. |
| FR-AGT-036 | The default `Work` MUST skip a recipe one of whose outputs with a positive cost share has a stock at or above `ProduceHaltRatio` of its output target, and MUST NOT let a byproduct halt a recipe. |
| FR-AGT-037 | The default `Work` MUST NOT request executions that would use an input which is also an upkeep commodity below the agent's upkeep for the tick, MUST treat a recipe whose cap of 7.1.6 is 0 as ineligible and try the next best eligible recipe, and MUST NOT request `Executions` 0 (`NoJob` with `no_recipe` if none is left). |
| FR-AGT-038 | The reserve, use target, job need, output target, target and keep of a commodity MUST be computed as in 7.3, from data and the agent's state alone, with the job need nonzero only when the agent can start a job in the next tick. |
| FR-AGT-039 | For each commodity the default `Trade` MUST place at most one order: a bid for the gap if it is positive, otherwise an ask for the surplus if it is positive, and none otherwise. |
| FR-AGT-040 | The three tiers of a bid MUST add to its gap, with tier 1 the upkeep short for this tick, tier 2 the inputs and tools of the next job short, and tier 3 the rest. |
| FR-AGT-041 | The sum over an agent's bids of limit price times quantity MUST NOT exceed its available money, and tiers 1 and 2 MUST be allocated before tier 3, in canonical commodity order within a tier. |
| FR-AGT-042 | Tier 3 MUST be limited to the spend cap `SpendMaxShare * (e * H / (e + H)) / 10 000`, with `e` the money left after tiers 1 and 2 less the savings target (the reserve money, unless the saving module is on and the role has a `Saver` block, 7.4.2), floored at 0, and the cap MUST be 0 when `e` is 0, MUST NOT decrease as `e` rises and MUST NOT exceed `e`. |
| FR-AGT-043 | The share of a tier-3 want bought MUST be `FavourFloor + (10 000 - FavourFloor) * (10 000 - pos) / 10 000`, with `pos` the position of the market price in the price window or 5 000 when the window is empty, and MUST NOT be applied to tiers 1 and 2 or to asks. |
| FR-AGT-044 | With an accurate view the default strategy MUST NOT produce an order that the market rejects for `insufficient_funds`, `insufficient_stock`, `duplicate_order`, `not_tradable`, `no_room` (with the storage module on), `overflow`, `holding_limit` or `receipt_limit`, over a range of seeds. |
| FR-AGT-045 | One `uint64` MUST be taken from the world's source for each intent the default strategy considers, the limit MUST be drawn from the effective range of the belief with `beliefs.Draw`, and an agent with `Hunger` above 0 MUST bid the top of the range for tier 1. |
| FR-AGT-046 | Where 9.1 states a quantity exact, the quantities of an agent's orders, targets and reserves MUST be the weight times the per-household amounts (the wants before the buy share and the cap, `Gap`, `Surplus`, the asks, the reserves, `Keep` and `UseTgt`, and the allocation of a tier funded in full); a target from `OutTgt` MUST be `ceil(w * X)` of 9.1, with one ceiling over the whole product (7.3); and the limit price MUST be the same whatever the weight while the first tier to buy is not lowered by `Afford`. |
| FR-AGT-047 | An ask MUST be for the whole surplus, units that expire at the end of the tick MUST be offered if no bid is placed, and an ask for a tool MUST be placed only by a role that makes it. |
| FR-AGT-048 | Every intent MUST carry the target of 7.3, the units reserved for this tick's upkeep, a unit cost that is the cost anchor of `spec-production.md` 9.3 (which counts labour at the pooled observed wage of the previous tick and, with the tenure module on, the rent of the draws) only if the role can make the commodity and is 0 otherwise and whenever the region has no observed wage, and a purpose by the rule of 7.3.6. (changed in the sweep) |
| FR-AGT-049 | The stock passed to `Revise` MUST be the units held after the trades less the units reserved for this tick's upkeep, not below 0. |
| FR-AGT-050 | `DefaultStrategy.Revise` MUST return the result of `beliefs.Update` with the draw the world took. |
| FR-AGT-051 | With the storage module off an agent MUST have no inventory capacity other than `MaxQuantity` and the default strategy MUST NOT discard; with it on the capacity MUST be the room of `spec-storage.md`, and the default strategy MUST discard only the surplus that its tier 1 bids need room for (`spec-storage.md` 6.3.2). |
| FR-AGT-052 | An agent of weight `w` in the same per-household state as an agent of weight 1, with the same random draws, MUST produce orders whose limit prices are equal and whose quantities are `w` times as large, and the same `Fed` fraction, in the conditions of 9.1 and only there (ample money, no limit lowered by `Afford`, an output target whose `X` is an integer or compared in the `ceil(w * X)` form, a stock away from the halt threshold, `w*h/u` exact for `Fed`); outside them the quantities MUST agree with the closed forms of 9.1 on the totals. |
| FR-AGT-053 | An external agent MUST have weight 1 and the `External` flag, MUST be bound by the same admission and conservation rules as any agent, and MUST NOT be removed, reduced, replaced or moved to another role by any spawner, the world refusing the decision with the reason `external_agent`, unless the agent is mortal, a flag the consumer sets when it adds the agent (default false). (changed in round 5: `Mortal`, `spec-spawner.md` 9.2) |
| FR-AGT-054 | `Directed` MUST return the directive's job request if it has one (`Job` non-nil), otherwise the fallback's; MUST replace the fallback's intent for a commodity by the directive's intent for it; MUST remove from the fallback's plan the commodities in `Hold`; and MUST add the directive's discards; a `Borrow` or a `Lend` of the directive MUST replace the fallback's credit request or offer. |
| FR-AGT-055 | A directive MUST apply to the tick after it was set, MUST be discarded after that tick unless `Sticky`, and MUST be part of the checkpoint. |
| FR-AGT-056 | No strategy call MUST wait for the consumer: `Step` MUST NOT block on input from the consumer, and an external agent MUST act only through its directive or a non-blocking strategy. |
| FR-AGT-057 | Per tick, the world MUST total for each agent the money it received from trades, the relief it was paid, the value of its in-kind receipts (including the board a trainee takes from its trainer's account) and the money it spent on trades by purpose, and the wear cost of the tools its executions wore (`spec-welfare-and-wages.md` 2.1, 2.1.4), MUST clear the totals at the start of the tick, and the trade totals MUST equal the agent's money flows by the reason `trade` and the relief total those by the reason `relief` in the ledger. (changed in the sweep: relief, in-kind and wear added) |
| FR-AGT-058 | Each parameter of section 12 MUST have a default that passes validation, and a value outside its bound MUST be rejected with `ErrOutOfBounds` carrying its path, value and bound. |
| FR-AGT-059 | Validation MUST reject an unknown commodity or recipe in an upkeep, start stock, `Trades` or `Recipes` list, a duplicate commodity in one list, and a configuration in which the largest weight times an upkeep, start or job quantity, with the targets, exceeds `MaxQuantity`, or the start money times the largest weight exceeds `MaxMoney`. |
| FR-AGT-060 | Validation MUST warn, and MUST NOT fail, for `no_upkeep`, `commodity_untraded`, `upkeep_unsourced`, `start_stock_below_target` and `reserve_exceeds_money`. |
| FR-AGT-061 | `AgentSpawned` MUST carry the agent, region, role, weight, the starting money received, the external flag and `StoresLocked`, and MUST be emitted for each initial agent after `WorldCreated`. |
| FR-AGT-062 | Every per-agent log line of this spec MUST be at `Debug` and guarded by `Enabled`, and no `Info` MUST be emitted in the per-tick, per-agent or per-order loops. |
| FR-AGT-063 | A run with a logger and an observer attached MUST give results identical to a run without them, and the draws of section 7 MUST be taken in both. |
| FR-AGT-064 | The state of section 15 MUST be saved as plain data in a fixed order with no maps, and a world restored between ticks MUST give the same orders, jobs and consumption as an uninterrupted run. |
| FR-AGT-065 | No code of this spec MUST branch on a commodity, recipe, role or region identifier. |
| FR-AGT-066 | Goods and money MUST be conserved after every tick of a run that includes consumption, discards, removals and weight changes, checked by audit. |
| FR-AGT-067 | The upkeep consumed in a tick MUST appear in `BoundaryFlow` under the reason `consumption` by region and commodity and nowhere else, and a discard under `disposal`. |
| FR-AGT-068 | A trainee's upkeep MUST be consumed from its trainer's account with the reason `consumption` when the skill module is on, and from its own otherwise. |
| FR-AGT-069 | The quantity wanted for upkeep (tier 1) MUST NOT depend on any price or belief: for the same stock and weight it MUST be the same at every price, and the only price effect on upkeep bought MUST be through the limit price and the available money. |
| FR-AGT-070 | When every role's households are given a `TargetRatio` above 1 in a closed economy whose only demand is upkeep (the demand module off), the default `Work` MUST still stop starting a recipe whose co-product stock is at or above `ProduceHaltRatio` of its target, so that the unsold stock of a produced good is bounded. |
| FR-AGT-071 | The agent view MUST carry the working ticks in the agent's window and its effective window, and the default `Work` MUST read the floor from them and from no state of its own. (new) |
| FR-AGT-072 | For each agent that is free when `Work` would be called, the world MUST determine by a pure function of its welfare record whether the floor `MinWorkShare` changed the share, whatever the strategy, MUST total the households of such agents per region and role, MUST report the total in the `Snapshot` (evaluated at the boundary over the agents with no job in progress, so it may differ from the count of the `Warn`), and MUST emit the `Warn` "work floor binding" of `spec-welfare-and-wages.md` 11.3 per region and role only, never per agent. (new, N-065) |
| FR-AGT-073 | The cost anchor passed to the view and to the intents MUST be computed by the world once per tick before `Work` is called, and the default strategy MUST pass it through unchanged and MUST NOT compute a cost. (new) |
| FR-AGT-074 | A job request for a recipe not on the role's list or its effective own-use list MUST be refused before any other check of production, and an intent for a commodity unknown to the configuration MUST be rejected as `unknown_commodity` and not as `not_tradable`. (new, FR-PRO-048, `spec-clearing.md` 3.2) |
| FR-AGT-075 | In each consumption phase the world MUST set `Privation` to `Privation + 1` if `Unfed` is above 0 and to `max(0, Privation - 1)` otherwise, after setting `Hunger`, MUST do so in the world and not in a strategy, MUST leave it unchanged by a weight change or a role change, and `Hunger` MUST NOT exceed `Privation`. (new, `spec-spawner.md` 5.1) |
| FR-AGT-076 | With the demand module on, the want units funded by `spec-demand.md` 4.5 MUST join the agent's bid for the commodity as a fourth part `T4`, MUST add to `Keep` and `Target`, MUST be funded only from the cap that tier 3 leaves, and with the module off the bid MUST be exactly as without it. (new, `spec-demand.md` FR-DEM-001, 016, 019) |
| FR-AGT-077 | `AgentBankrupted` MUST be the only event that removes the agent (the others of the removal accompany it), carrying `Weight` at removal, `Cause` (`insolvent`, `starved`, `consumer` or `population`), the money and the goods listed passed to the estate; a weight change that leaves households and a role change MUST NOT emit it. (new, `spec-spawner.md` 13.1.1) |
| FR-AGT-078 | An agent MUST carry a `DeathCarry` in `[0, 10 000 * TicksPerYear)`, started at `offset(ID)` of `spec-population.md` 3.8.3 without a draw, unchanged by a weight or role change, saved in the checkpoint in agent ID order. (new, `spec-population.md` FR-POP-010, 024, 035) |
| FR-AGT-079 | The world MUST execute a `TradePlan` in the order discards, shipments (in plan order), then intents, MUST NOT call `Work` or `Trade` for an agent that is away or on a migration journey, and MUST pass no view to it; with the transport module off a shipment request MUST be refused `unknown_mode`. The default strategy MUST append no shipment. (new, `spec-transport.md` 5.3, 5.4; `spec-migration.md` 8.3) |
| FR-AGT-080 | With the subsistence module on, the default `Work` MUST, for an agent in the own-use mode, set the work share to 10 000 and request the own-use recipe by 7.1.9 (the greatest coverage of the short covered commodities, ties to the lower recipe ID; the retry of `RetryJobs`; blocked when nothing is runnable), MUST NOT apply the production halt to an own-use recipe, and MUST NOT choose an own-use recipe outside the mode; a role with an empty `Recipes` list and an empty effective own-use list MUST request nothing. (new, `spec-subsistence.md` FR-SUB-019 to 025) |
| FR-AGT-081 | The view MUST carry `OwnUse`, `OwnUseJobs`, `Cover` and `Secure` and a mark on each own-use recipe with the subsistence module on, and the group `Merchant` only for an agent whose role has a merchant block, and `Outbound` with the transport module on. (new, `spec-subsistence.md` FR-SUB-018, `spec-merchants.md` 4.1) |
| FR-AGT-082 | `TradePlan` MUST carry `Offers`, `Takes` and `Notes` beside `Intents`, `Discards` and `Shipments`; the default strategy MUST append no offer and no take and leave `Notes` zero; offers and takes MUST be admitted and matched in the contract round after the last agent's plan and not by `Clearing`. (new, `spec-merchants.md` 9.2, 9.4) |
| FR-AGT-083 | A strategy registered as `"merchant"` MUST be provided by `strategy.Merchant` in the package `strategy/`, registered by `strategy.Options()`; `NewWorld` MUST reject the name with `ErrUnknownStrategy` when it is not registered. (new, `spec-merchants.md` 3.1) |
| FR-AGT-084 | The removal of an agent MUST drop a migration journey in progress and end each open delivery contract of which it is the deliverer, returning the escrow in full to the payer, before its money moves to the estate. (new, 3.6 step 2a; `spec-migration.md` 8.3, `spec-merchants.md` 9.8.2) |
| FR-AGT-085 | An order for a commodity under a `sell` or a `buy` right in force of a body that the agent is not a member of MUST be rejected by the world while it builds the orders, with the reason `not_permitted`, after `not_tradable` and before the market's checks, MUST NOT reach clearing and MUST have no outcome; the check MUST NOT apply to a pseudo-agent. (new, `spec-institutions.md` 5.6.2, FR-ORG-024) |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-AGT-001 | With the default logger and observer, the `Work`, `Trade` and `Revise` calls of the default strategy, the building of the view and the plan, and the consumption phase of a step of the reference chain MUST allocate nothing beyond scratch held by the world. |
| NFR-AGT-002 | The cost of the agent logic in a tick SHOULD be proportional to the sum over agents of the size of the tradable set plus the number of recipes, and MUST NOT depend on the weights. |
| NFR-AGT-003 | The targets, the tier allocation, the spend cap, the work share, the recipe choice and `Fed` SHOULD have benchmarks that call `b.ReportAllocs()`. |
| NFR-AGT-004 | The pure functions of `internal/household` MUST have fuzz tests asserting that no input within the bounds causes a panic or an overflow and that the spend cap and the tiers keep their invariants. |

## 21. Open questions

1. **Is a per-role upkeep right, or a household class?** The baseline basket `B_r`
   assumes one reference household. A role's own upkeep (a hard-working role eats more)
   is allowed, but compared by ratio only. (Answered, N-001 closed.) Classes need nothing more:
   a class is a role, with its own upkeep and, from `spec-demand.md` 3.3, its own ladder of wants.
2. **Is the production halt enough?** Without a capacity, overstock is limited by
   the halt, the spend cap, spoilage and the belief stock rule. If a good that nobody
   wants piles up without bound, `spec-storage.md` or a discard rule may be needed. The
   halt ratio is a guess.
3. **The reading of architecture.md 12.4 for the view.** The view is a copy of
   plain data passed to an extension for one call, not a value returned across the API
   boundary. The alternative, a fresh deep copy per call, allocates per agent per tick.
   If the rule is read strictly, the view must be rebuilt as a new value, and
   NFR-AGT-001 changes.
4. **Strategy identity in a checkpoint.** Names are hashed; the code is not. A consumer
   who changes a strategy's behaviour under the same name breaks the guarantee of
   ADR 0002. A version string returned by an optional method would let the hash cover
   it. **Answered** in `spec-checkpoint.md` 7.3.3: the names are hashed, the code is not, and an extension that keeps state saves a version with it (`Stateful`).
5. **Purpose and the welfare income.** `spec-welfare-and-wages.md` 2.1 nets inputs
   out of income. A commodity that is both an input and upkeep is classed as an input
   (7.3.6), a rule by commodity rather than by use. A cleaner rule nets the inputs
   *consumed* by jobs at the prices of the day, which needs the cost of the job. WEL must
   decide, together with 4.3's working tick (N-035).
6. **The band and the closed form.** The closed form of `spec-welfare-and-wages.md` 5 is
   exact at `RestBand` 0. At 500 the days worked are slightly more. The right value, and
   whether to keep the band for weight 1, is for the numeric run of `spec-stability.md`.
7. **Favourability on asks.** The original scales an ask by the position of the price.
   Here it is dropped (7.3.5) because holding goods back reads as a shortage to the
   belief rule. If sellers need a way to wait for a better price, the stock rule is the
   place to change.
8. **Where a trainee eats.** 5.7 says the trainer's account; the skill module may also
   want a transfer in kind. Left to `spec-skill-barriers.md`.
9. **The welfare history. (Answered.)** `spec-welfare-and-wages.md` 13 lists the ring buffer and the target draw as its checkpoint state, and `spec-checkpoint.md` 4.2 tag 3 saves it with the agent record.
10. **Agents in transit.** In version 1 a traveller keeps its origin region and is away, in no
    market, until arrival (`spec-transport.md` 5.5.2, `spec-regions.md` 6.5.3); its upkeep is
    consumed from its own stock (`unprovisioned` refuses a trip it cannot feed). Whether a
    traveller should be fed by the market on the way is open.
11. **Substitute needs. (Answered.)** Upkeep stays a fixed list: a household that can eat either of
    two staples is two roles or a list that includes both, in version 1. Substitution among
    alternatives exists in `spec-demand.md` 4.2, for wants only, where it cannot change `Fed`.
    Upkeep that accepts alternatives would change `Fed` and the survival rule, and is decided
    only if S04 or S16 show a pinned staple (11.6.1); `spec-demand.md` open question 1 keeps the
    arithmetic ready.
12. **Targets from sales, not production.** A seller's target is its expected output.
    Lengnick uses recent demand. A target from a moving average of sales would follow
    the market but gives a seller that sells nothing a target of 0, which reads as "no
    signal" (`spec-beliefs.md` 4.4). Kept as production until a run shows a reason.
13. **Hunger and productivity.** An unfed household produces as usual. A penalty (hungry
    households work less) is a stronger death spiral and is not modelled; `spec-population.md`
    may add it.
14. **Draw order at creation (3.5.2). (Answered.)** `spec-world.md` 5.7.1 adopts it.
15. **The reserve for a long job.** Tier 2 has no reserve (7.4.3). Whether a job longer
    than a number of ticks should keep `min(Duration, ReserveTicks)` ticks of upkeep
    cost is the interaction of N-032 with the saving rule, and wants a run. With the saving module
    on and a `Saver` block, the household's own remedy is `spec-saving-and-investment.md` 4.3.3.
16. **An elastic subsistence demand. (Answered, N-052 closed.)** A staple's demand is inelastic by
    construction (11.6) and stays so. Rationing of tier 1 is rejected (`spec-demand.md` 17), and
    `spec-demand.md` supplies an elastic demand for discretionary goods instead. The only lever
    left for a pinned staple is upkeep alternatives (question 11).
17. **Cover against shelf life.** A `CoverTicks` longer than the shelf life of a
    perishable upkeep good buys goods that spoil. Validation could warn
    (`cover_exceeds_shelf_life`); the default strategy could instead cap the cover at
    the shelf life. Not decided.
18. **Labour in the belief a recipe is judged by.** Recipe choice values outputs at
    the agent's belief `Centre` (7.1.4). If `spec-beliefs.md` keeps a cost ceiling that
    excludes labour, a labour-dominated good's belief is clamped below its
    zero-profit price (N-055 of `spec-viability.md` 8.2.3), and the recipe can show a
    net value of 0 or less to a household that would live well from it. The fix is in
    BEL and PRO; this spec only notes that the choice rule inherits it.
19. **A switching margin for the recipe choice** (AGT-A7). A margin that keeps the
    previous recipe unless another beats it by a share needs the previous recipe as
    state and a tunable; not specified.
20. **Does `Target` count `Inbound`?** With the transport module on the view carries `Inbound`
    (6.4.1). Whether the default strategy's `Target` (7.3) should count the units arriving within
    `StartsIn`, so that a consignee does not buy again what is on its way (`spec-stability.md`
    L85), is open (`spec-transport.md` 20.7). The default strategy never ships and ignores
    `Inbound` until this is decided.

## 22. Where this lives in the code

| Concern | Home |
|---------|------|
| The struct `Role` | root package, `config.go` (architecture.md 2.6; built in Stage 0), not `agent.go` |
| The role table: the recipes, trades and base upkeep of each role by typed ID, its tradable set (3.4) and membership flag, the commodities tradable in some role, and the effective upkeep per region (5.2): derived, unexported, built once at construction | root package, `role.go` (a union of lists needs no package, architecture.md 2.1) |
| The agent type (unexported `agent`, a sketch in 3.1; the fields a module hosts are in the module's slices by agent slot, architecture.md 2.10.1), `Directive` | root package, `agent.go` |
| `Intent`, `Discard`, `TradePlan`, `JobRequest`, `Purpose`, `AgentView`, `OutcomeView`, `Strategy`, `DefaultStrategy`, `Directed` | root package, `strategy.go` (with the continuation files `strategy_default.go`, `strategy_work.go`, `strategy_view.go` when the file passes the limits of N-243); there is no `directive.go` |
| `WithStrategy` and the resolution of a strategy name (`resolveStrategy`, `ErrUnknownStrategy`) | root package, `option.go` |
| `Side` (`Bid`, `Ask`, with a `String` method for the golden rendering of events) | root package, `event.go`: it is first used by the events (`spec-observability.md` 3.2.1), so it is declared with them and `Intent` and `Order` use it from there; `RecipeID` and the other typed IDs are in `value.go` |
| The tunables of section 12 and the role and region fields of 3.3 | root package, `config.go` and `defaults.go`; bounds in `bounds.go` (`MaxUpkeepItems` joins `spec-config.md` 4.1) |
| Validation, the sentinels and the warnings of section 13 | root package, `validate.go`, `errors.go` |
| Pure rules: the targets and tiers (7.3), the spend cap (7.4.4), the recipe value (7.1.4), `Fed` (5.4), `Privation` (5.4.1) (the tradable set of 3.4 is `role.go`, above); with the demand module also its ratio, shares, schedule, carry and allocation (`spec-demand.md` 20) | `internal/household` (stdlib only), over primitive slices and indexes, converted inline at the call (architecture.md 2.9); named for what it provides, not for a layer |
| The consumption phase, the creation and removal of agents, the mailbox (the per-agent functions) | root package, `agent.go`, called by the world and calling `internal/ledger` and `internal/household` inline |
| The work share `WorkShare` (7.1.2) | `internal/welfare` (built, `share.go`), not `internal/household` |
| The events `AgentSpawned` (extended) and `UpkeepShortfall` | root package, `event.go` |
| Reason constants (`not_tradable`, `not_permitted`, `not_on_role`, `no_room`, `rest`, `halted`, `no_recipe`) | root package, `event.go` (exported constants, `spec-observability.md` 4.2.3) |
| The log keys `households` and `unfed` (key strings only) | `internal/logkeys` |
| The ledger reasons `consumption`, `disposal` and `initial` (money and goods), the query `Expiring` (5.3.3), and `WearItems` (5.4.3) | `internal/ledger` registry and operations (`spec-ledger.md` 7.2, 5.3.3, 5.4.3) |
| The strategy table | root package, `world.go` |
| The merchant strategy and its registration `strategy.Options()` | `strategy/merchant.go`, `strategy/options.go` (the package is created when the first code lands; `spec-merchants.md` 3.1.2) |
| The pure rules of the merchant (margin, fee share, lot, matching, settlement) | `internal/merchant` (stdlib only; `spec-merchants.md` 23) |
| The own-use flag step, `Cov`, the ramp | `internal/ownuse` (stdlib only; `spec-subsistence.md` 22) |
| The Weyl offset of the agent ID shared by POP, MIG and MER (`spec-population.md` 3.8.3) | `internal/ordering`, one helper (N-019) |

```go
// internal/household: stdlib only, no root types. A sketch, not final.
// WorkShare is not here: it lives in internal/welfare (share.go).
type Need struct {
	Upkeep   int64 // U_c: units eaten this tick
	Job      int64 // Job_c: units for the next job (inputs and tools)
	OutTgt   int64 // the output target OutTgt, already formed as one ceiling over the whole product (7.3); not a rounded Make
}

type Targets struct {
	Reserve, UseTgt, OutTgt, Target, Keep int64
}

func ComputeTargets(n Need, params Params) Targets
func Tiers(t Targets, stock int64) (t1, t2, t3 int64)
func SpendCap(e, upkeepCost int64, params Params) int64
func Fed(consumed, perHousehold []int64) int64
```
