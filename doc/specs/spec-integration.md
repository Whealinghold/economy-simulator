# Spec: Integration: the player and other external agents, the between-ticks slots, the queries and the shape of a consumer's loop

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-INT-NNN` and
`NFR-INT-NNN`, which tests cite (testing.md section 4). They are in section 18.

Layer: core, and minimal. [ADR 0003](../adr/0003-version-1-scope.md) decision 2 puts INT
in version 1 only as external shocks and a player agent. This spec is the public
face of every call a consumer makes between ticks and the queries it makes
instead of reading a snapshot. It invents no mechanism: each call names the
mechanism, the slot and the position of the tick that another spec owns
(`spec-world.md` 8). It designs no policy actor, no tax and no scripting layer.

This spec adds no `Config` field, no default and no bound of its own: every bound
it applies is owned by the spec whose slot it guards, and it names the owner.
The only numbers in it are the example numbers of section 14, which are
labelled. Nothing has run in the library and no acceptance band exists (notes
N-015, N-027), and no model was needed (section 15). The settings of the worked
examples are invented **examples**: the words in them (`alpha`, `beta`, `grow`,
`player`) are data, never branched on (`AGENTS.md`, N-017).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).
"Consumer" means the library consumer (a game or a server), never a household
(N-001, N-017). The player is an agent of the consumer's world.

## 1. Purpose and scope

A game that embeds the library does five things the library cannot do for it:
it puts a player into the world, it tells the player what is going on, it
disturbs the world (a failed harvest, a closed road), it asks questions of it,
and it keeps the loop going. The first four are calls on the `World` between
two steps. This spec fixes, for each of them, the public name, the arguments,
the errors, the bound, whether it is atomic, the event it emits, the position at
which the tick reads it and how it is cleared. It covers:

- **external agents**: adding one, directing it, removing it, moving it, what it
  is told, and what the library does and never does with it (section 4);
- **the slots**: one table of every world slot a consumer may set, with the rules
  they share (section 5);
- **queries**: what a consumer may read without a snapshot, its cost class and
  that none of it draws or changes state (section 6);
- **errors and guards**: the sentinels, the order of checks, a failed world
  (section 7);
- **the shape of a consumer's loop**, in prose (section 8), and **what INT hands
  to the specs that come after it** (section 9).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The mechanism and the arithmetic of any slot (what a yield shock multiplies, how a scale applies) | the spec that owns the slot (section 5 names it in each row) |
| The position of a call in the tick, and the rule that a call acts between ticks only | `spec-world.md` 3.6, 8 |
| The event each call emits, its fields and its order | `spec-observability.md` (OBS) and the rules' table 4.3; this spec names what it needs (section 12) |
| The checkpoint, its encoding, restore and its errors | `spec-checkpoint.md` (CHK); this spec lists what it contributes (section 13) |
| Measures over many agents, indices, inequality | `spec-indicators.md` (IND) |
| A policy actor, taxes, tolls, a ruler's treasury | PLC, FIS: later (ADR 0003 decision 2) |
| Scripting, a rule language, a timer for a shock | not provided (ADR 0001 alternatives; 3.6) |
| Storage and transport of a consumer's own data (a save file, a socket) | the consumer; the library never persists (ADR 0002) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | Every call that changes the world acts between ticks, on a healthy world, is **atomic** (it validates in full and then applies, or returns an error and changes nothing) and returns `ErrInStep` from inside a step | 3.1, 3.3 |
| 2 | A call checks what is a property of its argument and of the configuration. It does **not** check what depends on state that a tick can change (affordability, stock, a reservation): the tick checks that and reports it as an event | 3.2 |
| 3 | A slot setter takes a **slice of entries**, one entry per key, and an entry replaces the key's whole value. An entry equal to the value the key already holds is accepted and is not an event. Setting a key to its default clears it | 3.3, 5.2 |
| 4 | The cause that an event of a consumer's call carries is fixed: `consumer`. The consumer supplies no cause | 3.3.3, 5.1.2 |
| 5 | A call names things by the library's typed IDs. There is no resolver in version 1: a typed ID is the position of the entity in the corresponding slice of `Config()`, and `Graph.Lookup` converts regions | 3.4 |
| 6 | An external agent is added with weight 1, an `External` flag, a `Mortal` flag (default false) and a strategy that is the library's `Directed` (the default) or a consumer's non-blocking one. It is created at the call with the next agent ID and its two creation draws, and acts from the next step | 4.1, 4.2 |
| 7 | A player acts by a **directive** left in a mailbox between steps. There is **no act-now operation**: an order is placed in the next step's trade phase and is filled at that step's clearing, one tick of latency | 4.3, 3.6 |
| 8 | The library never removes, reduces, replaces, switches or moves an external agent unless it is mortal, and the default modules never move one. Only the consumer removes one, by `RemoveAgent` | 4.4, 4.6 |
| 9 | A forced move is `Relocate`: the guarded relocation of `spec-regions.md` 6, which first cancels a migration journey **atomically** with the move. There is no separate public cancel | 4.5 |
| 10 | What a player is told is `AgentState`: its state at the tick boundary, the market reports of its region and the rumours of its last view. It is not the `AgentView` of a strategy, which is built inside a step | 4.7 |
| 11 | The slots are nine setters and one adjustment, listed in one table with the position at which the tick reads each. A shock has no timer: the consumer sets it and clears it | 5, 3.6 |
| 12 | A query takes no draw, changes no state, emits no event and returns a deep copy of plain data. Its cost is stated by class | 6 |

## 3. The rules common to every call

### 3.1 When a call may be made

3.1.1. A call that changes the world is made **between ticks**: after `NewWorld`
has returned or a restore has returned, and between two `Step` or `Run` calls. It
is also legal in the construction phase, between `NewWorld` and the first `Step`,
and its moves then go to the construction bucket (`spec-world.md` 5.5.2,
`spec-ledger.md` 7.4.1); a move made later is totalled in the tick the next step
runs (`spec-world.md` 8.1.3). The library has no timer, no callback and no queue
of calls: a call is applied when it is made.

3.1.2. A call made from inside a step (from an observer, a strategy, a spawner or
a clearing) returns `ErrInStep` and changes nothing; a call on a failed world
returns `ErrWorldFailed` wrapping the cause (`spec-world.md` 3.5, 3.6). The readers
that have no error result return their value in both cases, from the state
section 6.1.2 names for each.

3.1.3. A call runs on the caller's goroutine and returns when it is complete. The
world is not safe for concurrent use (architecture.md 12.1): a server that takes
calls from many clients serialises them itself, and the order in which it applies
them is part of the sequence of calls that fixes the run (3.5).

### 3.2 The order of checks

A call makes its checks in this order and reports the first failure, wrapped with
the call's name and, for a slice, the index of the entry:

| # | Check | Error |
|---|-------|-------|
| 1 | a step is running | `ErrInStep` |
| 2 | the world is failed | `ErrWorldFailed` |
| 3 | the module the call needs is off (section 5 says which calls need which) | `ErrModuleOff` |
| 4 | each argument, in the order given: an ID is known, an enumeration is valid, a number is within its bound, a key is not repeated | the sentinel of the row of the owner (sections 4, 5) |
| 5 | the agent, where the call names one: it exists, it is live, and it is of the kind the call needs (**for `Direct` only, this check comes before check 4**: the agent is found before its directive is read; `Relocate` keeps the order of `spec-regions.md` 6.1.3: region, then agent, then the account's kind) | `ErrUnknownAgent`, `ErrAgentRemoved`, `ErrNotExternal`, `ErrNotDirected` |
| 6 | what the owner of the mechanism refuses (the refusals of a relocation, `ErrRegionBound`) | the owner's sentinel |

3.2.1. **What a call does not check.** A call does not check what a tick can
change between the call and the position that reads it: whether the agent can
afford an order, holds the stock it asks to discard, has a job it may start or a
reservation (none exists between ticks). Those checks belong to the position
that acts, which reports a refusal as an event and a `reason`
(`spec-agents.md` 6.5, `spec-clearing.md` 3.2). A call that told the consumer
"affordable now" would be a promise that the tick would not keep.

3.2.2. **Argument errors are typed.** An error of check 4 is a `*ValidationError`
of `spec-config.md` 7.1.1 with a `Path` that names the argument (`changes[2].bp`),
the sentinel as its `Rule`, the offending value and, for a bound, the bound.
`errors.Is` matches the sentinel and `errors.As` finds the typed error
(architecture.md 8.4, testing.md 3.9). The library does not log an error it
returns (architecture.md 8.3).

### 3.3 Atomic, one entry per key, an event per change

3.3.1. **Atomic.** A call validates every entry (3.2 checks 3 to 6) and only then
applies any. If any check fails, nothing is applied, no event is emitted and no
random number is drawn (`spec-world.md` 8.1.1 item 2; `spec-regions.md` 7.3.1).

3.3.2. **The shape of a setter.** Each slot setter of section 5 takes a slice of
entries. An entry names a key and gives the key's **whole value**; a key may
appear once in a call, else `ErrDuplicateID` and nothing is applied. An empty or
nil slice is accepted and is a no-op with no event. An entry that equals the value
the key already holds is accepted and is **not** a change: it emits no event and
leaves the saved state as it was (as `spec-regions.md` 7.3.1). An entry equal to
the key's default clears the key: the owner then holds the default, and the saved state is what a world never shocked at that key would save (an owner that
saves a value for every key, as MON saves each flow's `Scale`, saves its default).

3.3.3. **Events.** A call emits one event per key that **changed**, at the call,
after the whole call is applied, with the `Tick` that the next step will run, and
before that step's `TickStarted` (`spec-world.md` 6.11). Within a call the events
are in the canonical order of the key (ascending, as each owner states), not in
the order of the slice, so that the order of a consumer's slice cannot change an
observer's record. The cause of an event that carries one is `consumer`.

3.3.4. **A call is not a transaction across calls.** Two calls are two
independent atomic changes. A consumer that needs a disaster that sets a yield
shock and closes a road, and wants both or neither, validates the second call's
arguments first or accepts that the first stays if the second fails. A bundle that
applies several slots atomically was considered and is not provided in version 1
(open question 4).

### 3.4 Identifiers

3.4.1. A call takes the typed IDs the library already exports (`RegionID`,
`CommodityID`, `RoleID`, `RecipeID`, `ModeID`, `ResourceID`, `AgentID`;
architecture.md 3.3). A **flow** is the exception: it is named by its string ID,
which is also its ledger reason (`spec-money-supply.md` 4.5), because it has no
typed ID.

3.4.2. **A typed ID is not stable across configurations** (`spec-regions.md` 3.3.1;
`spec-world.md` 5.3.2). It is the position of the entity in the corresponding slice
of the effective configuration, which is in canonical order (`spec-config.md` 3.4):
the `RegionID` of a region is its index in `Config().Regions`, and so on for the
others. A consumer that persists an entity persists its string ID and resolves it
once after `NewWorld` from `Config()` (and a region by `Graph.Lookup`,
`spec-regions.md` 4.3.2). No resolver type is added in version 1: it would be a
second surface over data the consumer already holds (open question 1). An
`AgentID` has no string form; it is never reused and it is returned by `AddAgent`.

3.4.3. A pseudo-agent (an estate, a counterparty of the outside) has an `AgentID`
and is **not an agent**: `Agent`, `Direct` and `RemoveAgent` return `ErrUnknownAgent`
for it (`spec-world.md` 5.3.1). `Relocate` and the query `RelocationRefusal` are REG's and
agree for every input (FR-REG-059): they return `ErrNotRelocatable` for an estate's ID
(`spec-regions.md` 6.1.3 row 4, after the region and agent checks).

### 3.5 What a call does to the run

3.5.1. **The sequence of calls is an input.** A run is a function of the
configuration, the seed and the sequence of calls (architecture.md 7.4;
`spec-world.md` 8.1.1 item 5). A call is not configuration and is not hashed. Two
consumers that make the same calls in the same order between the same steps get
the same run on every platform.

3.5.2. **The draws.** Exactly one call draws: `AddAgent` takes the two creation
draws of the new agent at the call, in call order (`spec-world.md` 7.3, D12). No
other call draws, and no query draws. A call that fails draws nothing.

3.5.3. **Money and goods.** A call creates money or goods in one place only:
`AddAgent`, through the `initial` faucet, which is counted in the construction
bucket or in the next tick's `BoundaryFlow` (4.1.4). No slot creates or destroys
money or goods: an endowment adjustment changes a stock in the ground, which is in
no account (`spec-resources.md` 6.2.1), and a flow scale only scales a declared
flow. The conservation identities of `spec-ledger.md` 9 hold after every call.

### 3.6 What version 1 does not offer

Each is a decision, with the reason, so that a consumer does not look for it.

| Not provided | Why | Instead |
|--------------|-----|---------|
| An **act-now** operation (buy now, settle now) | it breaks the rule that the order of clearing cannot change a result (FR-CLR-023), would execute a trade between steps and would make a checkpoint depend on when the consumer called (`spec-agents.md` 10.5) | a directive: the order is placed in the next step and filled in its clearing |
| A **timer** on a shock (a duration, an expiry) | a timer is state, saved and read by a position, and a rule in the tick; the roadmap holds the engine to measuring and not setting (ADR 0001 decision 7) | the consumer sets the slot and clears it at the tick it chooses; `Calendar().At(t)` gives the season to time it by (N-031) |
| A **bundle** of slots applied as one | a second surface, and a first consumer has not asked (3.3.4) | two calls |
| **Goods injected** into a region or an estate | a goods faucet needs a ledger reason and a named flow (N-021, N-107); the money flows are the supported faucets | an external agent added with `Stock`, which is a named, counted `initial` |
| A **capacity multiplier** of an outside line (a partial embargo) | N-116: no consumer has asked; a closed edge cuts the line entirely | `SetEdgeStatuses` on the gate edge |
| A public **cancel** of a migration journey | a cancelled journey that leaves the household in its origin, its job abandoned, is a state with no use of its own (4.5) | `Relocate`, which cancels atomically with the move |
| **Removal of a household** (a non-external agent) | a plague or a battle is a rate or a decision of a spawner, and both are already data or an extension: `DeathScale`, a consumer's `Spawner` with cause `consumer` (`spec-population.md` 6.6) | those |
| Editing a **configured value** | a slot never edits configuration (`spec-world.md` 8.2.1); the hash would no longer name the run | a new `Config` and a new world |
| A **shock by season** (a yield shock keyed by season) | the yield is read at job start, so a shock set for a window covers the starts in it; a season key would be a second axis on a slot with none (`spec-production.md` 6.8) | set before the season, clear after |

## 4. External agents: the player, bots and other agents of the consumer

This section answers N-022, N-079 and the public half of N-090. The mechanism is
`spec-agents.md` 10 (the directed strategy, the mailbox, the `External` and
`Mortal` flags); the rule that the spawner and the population module leave an
external agent alone is `spec-spawner.md` 9 and `spec-population.md` 3.7.2; the
relocation is `spec-regions.md` 6.

### 4.1 AddAgent

```go
// NewAgent describes an external agent that a consumer adds to a world.
type NewAgent struct {
	Region   RegionID    // an ordinary region, not an outside node
	Role     RoleID      // any role of the configuration
	Strategy string      // a registered name; "" means "directed"
	Money    Money       // money it starts with, 0 to MaxMoney, created by `initial`
	Stock    []StockItem // goods it starts with, one entry per commodity, created by `initial`
	Mortal   bool        // the failure ladder may act on it (4.6); default false
}

// StockItem is a quantity of one commodity.
type StockItem struct {
	Commodity CommodityID
	Quantity  Quantity // 1 to MaxQuantity
}

// AddAgent adds an external agent of weight 1 and returns its ID. It acts between
// ticks only (3.1). The agent exists when the call returns and takes part in
// the next step.
func (w *World) AddAgent(n NewAgent) (AgentID, error)
```

4.1.1. **Checks**, in the order of 3.2: the region is known (`ErrUnknownRegion`)
and is not an outside node (`ErrOutsideNode`: it has no market and no agents,
`spec-regions.md` 3.5); the role is known (`ErrUnknownRole`); the strategy name is
`default`, `directed` (4.2) or registered, else `ErrUnknownStrategy`; `Money` is in
`[0, MaxMoney]` and each `Quantity` in `[1, MaxQuantity]` (`ErrOutOfBounds`); a
commodity is known (`ErrUnknownCommodity`) and appears once (`ErrDuplicateID`); and
the creation does not take a balance or a total above its bound (`ErrOverflow` of `spec-agents.md` 13, to which the call maps the ledger's
`ErrBalanceOverflow` of `spec-ledger.md` 4.2; the mapping joins N-075). The amount of `Money` is **stated** by the
consumer: `Role.StartMoney` is the default of an agent the configuration or the
spawner creates and is not applied here (`spec-agents.md` 3.5.4), so a player may
start with nothing. A consumer that wants the role's figure reads it from
`Config()`.

4.1.2. **What the call does**, in this order, atomically:

1. the next `AgentID` is taken (`spec-world.md` 5.3), and an account of kind Agent
   is opened in the region (`spec-ledger.md` 3.1);
2. the money and the goods are created by the reason `initial` (a faucet,
   `spec-ledger.md` 7.2.1) into the account, with the default lot key: a good that
   spoils is born at `Tick()` and a durable has full durability
   (`spec-ledger.md` 5.2.3);
3. the agent is given: weight 1; `External` true; `Mortal` as asked; its strategy;
   one belief for each commodity of its role's tradable set, by the rule of
   `spec-beliefs.md` 8.3 (the clamp of the window mean of its market, or the market
   price if the window is empty, with `BeliefWidthInit`; no draw; before the first
   step this is the seed price, which is `spec-beliefs.md` 8.2); no job; an empty
   welfare record padded by `DefaultRatio`; `Hunger`, `Unfed` and `Privation` 0; a
   `DeathCarry` of `offset(ID)` (no draw); `RoleSince` the current tick;
4. the world takes its two creation draws, the target ratio and then the residue
   (`spec-world.md` 5.7.1, D12);
5. `AgentSpawned` is emitted with `External` true and `StartMoney` the money
   given, at the call, with the `Tick` the next step will run
   (`spec-world.md` 6.11).

4.1.3. **The new agent is last.** It takes the highest ID, so no other agent's ID,
review bucket or tie-break key changes (`spec-world.md` 5.3.2 is about adding a
region, not an agent). The **start rotation** of its region does change: it begins at
`t mod n_r` of the region's agent list and `n_r` grows by one, so who goes first at
P2.1, P2.5 and P2.7 changes for every agent of the region (`spec-world.md` 6.1.4,
8.3.1); a removal and a relocation do the same in the regions they leave and join.
What else the call changes for the others is the stream (two
words, then the words the new agent's own orders take in the steps that follow) and
the orders in the markets it enters.

4.1.4. **Its wealth is a named faucet** (`spec-agents.md` 3.5.4). The money and the
goods appear in `BoundaryFlow` under the reason `initial`, by region and
commodity, in the construction bucket if the call was made before the first step
and in the totals of the tick the next step runs otherwise (`spec-world.md`
8.1.3). The conservation identities hold (`spec-ledger.md` 9.1). A consumer that
adds many agents with money is a faucet by another name: the money supply grows
with the calls (INT-A2).

4.1.5. **It counts.** An external agent is counted in the households of its group
and in the role's wage and welfare like any agent (`spec-spawner.md` 9.3). It is
not exposed to the population schedules (4.6), and the spawner's decisions on it,
an `Apply` for a traineeship among them, are refused (`spec-spawner.md` 9.2). A mortal external agent may be entered as a trainee by a consumer's spawner's `Apply` (`spec-skill-barriers.md` 4.1.1); a non-mortal one is never entered.

### 4.2 The strategy of an external agent, and when it acts

4.2.1. **Two ways, one model** (`spec-agents.md` 10.3). An external agent has
either the library's `Directed` strategy, which a consumer steers by a directive
(4.3), or a consumer-written `Strategy` registered with `WithStrategy`, which is
called like any strategy, on the stepping goroutine, **must not block** and
decides from its view. A scripted merchant or a bot is the second kind; a player
is the first.

4.2.2. **The name `directed`.** This spec asks that a second name be built in
beside `default`: `directed`, the library's `Directed` with `DefaultStrategy` as
its fallback (`spec-agents.md` 10.3). It is the strategy an empty `Strategy` field
of `NewAgent` means. A consumer that wants another fallback registers its own
`Directed{Fallback: x}` under a name of its own. The world holds a strategy
**directed** when the registered value's dynamic type is `Directed`; a consumer
wrapper that holds one in a field is not (the world cannot see through it), and
`Direct` returns `ErrNotDirected` for it. The edits to `spec-agents.md` 6.6 and
`spec-world.md` 4.1 are in section 19.

4.2.3. **When each acts**, from the positions of `spec-world.md` 6:

| Position | What the world does for an external agent |
|----------|-------------------------------------------|
| at the call | the agent is created (4.1.2); nothing else |
| P1.9 | the mailbox is read, and a non-`Sticky` directive is taken out of it into the tick's scratch: the directive of the tick is fixed |
| P2.5 | `Work`, if the agent is free: a directed agent returns the directive's job, else the fallback's (`spec-agents.md` 10.4) |
| P3.2 | the view is built (with rumours if the role is `Informed`, 4.7.3) and `Trade` runs; a directed agent's plan is the fallback's changed by the directive; the plan is executed and each order is admitted or rejected like any |
| P4.1 | its orders clear with every other; an unmatched order is released |
| P5.4 | `Revise` is called for each of its admitted orders with the world's draw |
| P6.1 | its upkeep is consumed; `Unfed`, `Hunger` and `Privation` are set |
| P7.1 | the spawner's view lists it (`External` true); a decision on it is refused unless it is mortal (4.6) |
| P7.2 | the migration pass does not review it |

An agent added at the boundary of tick `t` is free at P2.5 of tick `t` and trades
at P3.2 of the same tick. A directive given before that step is read at its P1.9.

### 4.3 Direct and the mailbox

```go
// Direct leaves a directive in the mailbox of a directed external agent.
// It replaces what the mailbox held and applies to the next step only,
// unless the directive is Sticky.
func (w *World) Direct(a AgentID, d Directive) error
```

`Directive` is the type of `spec-agents.md` 10.3: a job request, intents,
discards, a hold list and `Sticky`. This spec adds validation and nothing to the
type.

4.3.1. **Semantics.** The mailbox holds one directive. A second call before the
step **replaces** the first: it does not merge. A directive with nothing in it
(no job request, no intents, no discards, no hold) empties the mailbox. The
directive is read at P1.9 of the next step and applies to that step
(`spec-agents.md` 10.4 rule 1, FR-AGT-055); the world drops it afterwards unless
`Sticky`, in which case it stays until another call replaces or empties it. A
sticky directive is a standing order: it does not follow the market, so a limit
price that has become wrong is the consumer's to change. A directive that names a
recipe off the role's list, or a commodity outside its tradable set, is refused
here (4.3.2); a directive that is sound when given and cannot be carried out when
the step comes (the agent no longer holds the stock, the order is not
affordable) is refused by the tick, as any order is (3.2.1).

4.3.2. **Checks**, agent first (the exception of 3.2 check 5), then the arguments in the order given. The agent has been found (`ErrUnknownAgent`, `ErrAgentRemoved`), found
external (`ErrNotExternal`) and found directed (`ErrNotDirected`):

| # | Check | Error |
|---|-------|-------|
| 1 | a job request names a known recipe (unless it asks for no job) and `Executions` is 0 or at most the agent's weight | `ErrUnknownRecipe`, `ErrOutOfBounds` |
| 2 | the recipe is on the role's list or its effective own-use list | `ErrNotOnRole` |
| 3 | each intent names a known commodity, has a valid `Side` and `Purpose`, `Quantity` in `[1, MaxQuantity]`, `Limit` in `[1, MaxPrice]`, and `Target`, `Reserve` and `UnitCost` in their bounds; a commodity appears in at most one intent | `ErrUnknownCommodity`, `ErrOutOfBounds`, `ErrDuplicateID` |
| 4 | each intent's commodity is in the role's tradable set (`spec-agents.md` 3.4) | `ErrNotTradable` |
| 5 | each discard names a known commodity, once, with `Quantity` in `[1, MaxQuantity]`; each hold names a known commodity, once | as 3 |

The two checks that read the role (2 and 4) are the only state-like ones, and they
are made here because a player's only feedback on a request that can never be
honoured is the error of the call; the world checks the same at admission as it
does for every strategy (`spec-agents.md` 6.5), so a change of role after the call
is still caught. The directive's `Limit` is checked against `[1, MaxPrice]` and
not against the belief band or the resolution floor: whether the price is
admissible is `spec-clearing.md` 3.2's `invalid_price` at the step.

4.3.3. **A job request carries a presence.** `spec-agents.md` 10.3 gives
`Directive.Job` as a value, and a zero `JobRequest` (`NoJob` false, recipe 0) would
read as a request for recipe 0. This spec requires that a directive carry a job
request **only when the consumer sets one**, so that a directive with intents and
no job leaves `Work` to the fallback. The representation is AGT's (recommended: a
pointer, `Job *JobRequest`); see the conflict in the handoff.

4.3.4. **No event; a `Debug` line.** A directive is state saved with the agent
(`spec-agents.md` 15) and is not a transition the world reports: `Direct` emits
no event (`spec-world.md` 8.3). It logs "directive set" at `Debug` with `agent`,
guarded by `Enabled` (observability-and-logging.md 3.1.3).

### 4.4 RemoveAgent

```go
// RemoveAgent removes an external agent the consumer no longer wants.
func (w *World) RemoveAgent(a AgentID) error
```

4.4.1. **Only an external agent.** The call returns `ErrNotExternal` for a
household of the configuration or the spawner (3.6). Whether the agent is
`Mortal` is irrelevant: `Mortal` limits what the **library** decides, not what the
consumer asks.

4.4.2. **What it does.** The removal mechanics of `spec-agents.md` 3.6, in their
order: its job is abandoned (`agent_removed`); a traineeship ends if it was in one (`spec-skill-barriers.md` 4.6); a journey is dropped (none: it is never migrated); the escrow of
a contract of which it is the deliverer returns to the payer and the contract
ends (`spec-merchants.md` 9.8.2); a contract of which it is the payer is settled
by the estate standing in for it (`spec-merchants.md` 9.7); a shipment it owns or
receives is diverted or dropped as `spec-transport.md` 5.8 and 5.9 say; its money
and goods go to the estate of its region (`estate_money`, `estate_goods`) and the
estate lists the goods; its account is closed; its ID is never reused. The call
never refuses for an obligation, because every obligation has a defined end on
removal. It draws nothing.

4.4.3. **Events**, at the call, with the `Tick` the next step will run, in the
order of `spec-world.md` 6.11 item 8: `RecipeAbandoned` if it had a job,
`TraineeshipEnded` if it was in a traineeship, the `ContractSettled` of each contract it was the deliverer of, `AgentBankrupted` with
`Cause` `consumer`, then `EstateOpened`. `AgentBankrupted` is the event that
removes the agent. A pending directive and the rumours kept for it (4.7.3) go with
the agent.

4.4.4. **After it.** The ID names a removed agent: `Agent`, `Direct`,
`RemoveAgent` and `Relocate` return `ErrAgentRemoved` (which wraps
`ErrUnknownAgent`: `errors.Is` matches both). A consumer that wants the player back
adds a new agent and takes a new ID.

### 4.5 Relocate: the forced move

```go
// Relocate moves an agent, with all the households it stands for, to another
// region at once. It is the guarded relocation of spec-regions.md 6.
func (w *World) Relocate(a AgentID, to RegionID) error
```

This is the public face of `relocate` that `spec-regions.md` 6.6 and
`spec-migration.md` 10.3 leave to INT, for the one use a consumer has for it: a
**forced move**, which is a decision of the consumer and not of the agent (a
flight, an expulsion, a player's journey that takes no time).

4.5.1. **Any agent, not a pseudo-agent.** An external agent or a household of the
configuration: the consumer asked, so the default modules' rule that they never
move an external agent (`spec-migration.md` 3.3) is not in the way. A cohort moves
whole and lockstep (`spec-regions.md` 6.1.4).

4.5.2. **What it does**, in this order, atomically:

1. the dry run `RelocationRefusal(a, to, "on_journey")` of
   `spec-regions.md` 6.1.6, in REG's order of 6.1.3: the first refusal other than
   `on_journey` is returned (`ErrUnknownRegion`, `ErrUnknownAgent` (`ErrAgentRemoved` for a
   removed ID), `ErrNotRelocatable` for an estate's ID, `ErrSameRegion`, `ErrOutsideNode`, `ErrOpenReservation`, or `ErrRegionBound`, which names the obligation:
   `open_shipment`, `open_contract`, `in_traineeship`). If one is returned nothing
   has changed. From here on the call cannot fail;
2. if the agent is on a migration journey the journey is dropped by the module's
   own cancel (`spec-migration.md` 8.4.1): the record goes, the ledger is not
   touched (the account never left the origin), and the households are counted in
   `Ended` of the next `MigrationReviewed`;
3. a job in progress is abandoned first, with the reason `region_changed`
   (`spec-production.md` 5.4), and the own-use flag and counter are dropped
   (`spec-subsistence.md` 5.8): the same two steps a departing migrant takes
   (`spec-migration.md` 8.2);
4. the account's region changes by the guarded relocation with the cause
   `consumer` (`spec-regions.md` 6.2): the relocation totals take what the account
   holds, and `AgentRelocated` is emitted;
5. the beliefs are reseeded by the arrival rule of `spec-beliefs.md` 8.3, with no
   draw, by the one helper that MIG and MER use (N-038); `RoleSince` becomes the
   current tick, which gives the agent the grace of `spec-spawner.md` 5.2.

What is left as it was: money and goods, role, weight, strategy, the welfare
record, `Hunger`, `Privation`, the target ratio, residues, want carries, and the
mailbox (`spec-regions.md` 6.3). A directive's intents name commodities and not
regions, so a pending directive stays; its limits are the consumer's to review.

4.5.3. **Why there is no public cancel.** MIG gave itself an internal
`cancelJourney` and left its public face to INT (`spec-migration.md` 8.4.1, open
question 16). A cancel on its own would leave a household in its origin with the
job it abandoned at departure and a record of a journey that did not happen, and a
consumer has no use for that state. The one need is to move a household that is on
a journey, and `Relocate` does that with the cancel inside it: the cancel and the
move happen together or not at all, so a refusal by another obligation cannot leave
the journey cancelled and the household not moved. If a consumer asks for "turn the
migrants back" the call is added then (open question 5).

4.5.4. **What it does not do.** It does not carry goods in transit
(`spec-regions.md` 6.5.1), does not check a capacity (a relocation has none; N-120
stays open), moves no money between accounts and has no reason in the registry.
It is not a travel time: an agent that must be a week away is an agent the consumer
removes from play itself, or a shipment of a self-carried mode that
`spec-transport.md` 5 owns.

### 4.6 What the library does with an external agent

| Module or rule | What it does | Owner |
|----------------|--------------|-------|
| Upkeep, consumption, relief, the welfare record | as for any agent: it eats, may go unfed, may be paid relief when its ratio is below `ReliefBelow`; its `Unfed`, `Hunger` and `Privation` are visible and count | AGT 10.1, EST 2.6 |
| The spawner (default or a consumer's) | no decision on it is carried out: the world refuses a reduction, removal, role change or application with the reason `external_agent`, **unless it is mortal**, and then only if a consumer's spawner or the consumer's call decides; the default spawner never does | SPN 9, FR-SPN-010 |
| The population module | it is not exposed: no births or deaths of it | POP 3.7.2 |
| The migration module | never reviews, moves or draws for it | MIG 3.3, FR-MIG-045 |
| The failure ladder | its signals are visible; rung 1 and rung 3 do not act unless mortal | SPN 5, 9.2 |
| Traineeships | a mortal external agent may be a trainee through a consumer's spawner's `Apply`; any other `Apply` is refused like any decision on it | SKL 4.1.1, SPN 9.2 |
| Delivery contracts, shipments, journeys of the consumer's own | it may be a payer, a deliverer, a shipper or a consignee through its strategy; the refusals of a relocation apply | TRN, MER |

4.6.1. **A mortal external agent** is one the consumer lets the library's rules
act on. It is not a promise that the default spawner will: the default spawner
decides nothing about an external agent (SPN 9.1). A mortal player can be removed
by a consumer's own spawner (cause `consumer`) or by its own call. Whether a
player can starve is the consumer's choice, made by this flag.

### 4.7 What the player is told

4.7.1. **Three channels.** A consumer tells its player what the world tells the
consumer: (a) a **query**, `Agent(id)`, `Market`, `Region` and the others of
section 6, read at the boundary; (b) the **events** of the step just run, in which
the player's own orders, rejections, fills and upkeep appear under its agent ID
(`OrderPlaced`, `OrderRejected`, `TradeExecuted`, `RecipeExecuted`,
`ShipmentArrived`, and the others of the rules' table 4.3; `UpkeepShortfall` is per
region and role and the player's unfed households are in `Agent`), filtered by the
consumer's observer; (c) the **rumours** of the player's last view (4.7.3). The
library decides nothing about what the consumer shows: `Agent(id)` answers for
any live agent, for a designer's tool as much as for a player, and the consumer
decides which agents' states its player may see.

4.7.2. **`AgentState`, not `AgentView`.** The view of a strategy
(`spec-agents.md` 6.4) is built inside a step, after production, from a state that
no consumer can see at the boundary (the stock after this tick's completions, the
refusal reasons of the starts, the draws of the rumours). `AgentState` is what is
true at the boundary and is a function of the state of the world alone: a
deep-copied plain value, no pointer, no function, no map.

| Group | Fields | Source |
|-------|--------|--------|
| Identity | `ID`, `Region`, `Role`, `Weight`, `External`, `Mortal`, the strategy's registered name, `RoleSince` | AGT 3.1, SPN 14 |
| Standing | `Money` (the available balance; no reservation exists), `Hunger`, `Unfed`, `Privation`, the welfare ratio and whether it exists, **the earned ratio** (the ratio without relief, `spec-welfare-and-wages.md` 2.1.7) and whether it exists, the target ratio, the working ticks in the window and the effective window | AGT 5.4, WEL 2 |
| Holdings | per commodity held: `Quantity`, and `Expiring`, the units that spoil at the end of the tick the next step runs (`Expiring(account, commodity, tick)`, `spec-ledger.md` 5.3.3) | LED |
| Job | whether one is active, its recipe, `Remaining`, `Executions`, and the pending output | PRO 5.1 |
| Beliefs | per commodity of the tradable set: the centre, the width and the bounds | BEL 3 |
| Markets | per commodity of the tradable set: the market report of its region, as `Market` returns it (6.3) | CLR 5.4, 5.5 |
| Upkeep | the effective upkeep per household (`spec-agents.md` 5.2) | AGT |
| Away, journey | whether it is away on a trip (its shipment), and its journey (destination, arrival tick) | TRN 5.5, MIG 14 |
| Own use | with the subsistence module on: `OwnUse`, `OwnUseJobs`, `Cover`, `Secure` | SUB 14.4 |
| Wants | with the demand module on: the carry of each want of the agent's ladder (`spec-demand.md` 4.4, 12) | DEM 4.4, 12 |
| Mailbox | the directive pending, if any | AGT 10 |
| Rumours | the rumours of the last view built for it, and the tick of that view (4.7.3) | INF 5, 6 |

The cost of building it is `O(|tradable set| + |recipes| + |lots|)`, an
allocation; it is a query of the second class (6.1). It reports the agent as of
the end of the last step and says nothing of what its production of the next step
will do: `MaxExecutions`, the refusal reason of a recipe and the stock after the
next completions are not in it, because they are not a function of the boundary
state without running a position of the tick.

4.7.3. **Rumours are kept for an external agent.** A view carries rumours
(`spec-information.md` 6) with error draws taken at P3.2(a) from the world's
stream, so a query cannot build them: it would take a draw (6.1.1). What a player
hears of other regions' prices is therefore **the rumours of the last view that
was built for it**, with the tick of that view: the world keeps, for an external
agent whose role is `Informed` and with the information module on, a copy of the
rumours it gave it at its last P3.2(a), at most `MaxRumoursPerView`
(`spec-information.md` 9.2, `spec-config.md` 4.1). They are one tick old at the
boundary. Without this a
player of an informed role would be told nothing of other regions, or the true
prices, and `spec-information.md` and N-106 forbid the second. This is a new
piece of state, per external agent, saved in a checkpoint (13); it affects no
result: the strategy sees what it saw. The edit it asks of INF, AGT and CHK is in
section 19; if it is refused the field is empty and `AgentState` says so with a
tick of -1.

### 4.8 What is saved

The mailbox, the `External` and `Mortal` flags and the strategy's registered name
are the agent's state and are saved by `spec-agents.md` 15. The one thing this
spec adds is the rumours of 4.7.3. A restored world answers `Direct` and `Agent`
as the original did (FR-AGT-064).

## 5. The slots

A **slot** is a piece of world state that a consumer sets from outside, between
ticks, over a configured value, and that a position of the tick reads
(`spec-world.md` 8.2). The mechanism, the arithmetic and the saved state of each
belong to the spec that owns it; this section gives the public call. It answers
N-091 and N-116 and settles the naming that the owners left provisional. The
table is the whole list: a spec that adds a slot adds a row here and one to
`spec-world.md` 8.2 in the same change (`spec-world.md` 8.2.1).

### 5.1 The table

Every setter has the shape of 3.3: a slice of entries, one per key, atomic, an
event per changed key in the canonical key order. The value column is the whole
value an entry gives its key. The default is what an unset key holds and what an
entry equal to it restores.

| Call | Entry (key; value) | Value bound; default | Needs | Owner | Read by the tick at | Event (at the call) | Saved by |
|------|--------------------|----------------------|-------|-------|---------------------|---------------------|----------|
| `SetYieldShocks` | `YieldShock{Region, Recipe; BP}` | 0 to `MaxYieldBP`; 10 000 | | PRO 6.8 | P1.2 (cost anchor), P2.5 (`k0`), P2.7 (the yield read at a job's start; a running job keeps its pending output) | `YieldShocked` (new) | PRO 15, by (region, recipe) |
| `SetEdgeStatuses` | `EdgeStatus{From, To; Closed, FactorBP}` | `FactorBP` 1 to `MaxEdgeFactorBP`; open at 10 000 | | REG 7 | **at the call**: the graph and lag tables are rebuilt; then P3.2 and P3.3 (route, travel time), P3.5 (a closed gate edge cuts the outside), P7.2 (a journey's distance); a shipment in flight keeps its arrival tick | `EdgeStatusChanged` | REG 11, by `EdgeID` |
| `SetTransitFactors` | `TransitFactor{From, To, Mode; BP}` | 0 to `MaxTransitFactorBP`; 10 000 (0 blocks new dispatches) | transport | TRN 10.3 | P3.2 and P3.3, at dispatch; new dispatches only | `TransitFactorChanged` (new) | TRN 14, by (from, to, mode) |
| `SetFlowScales` | `FlowScale{Flow; BP}` | 0 to `MaxFlowScale`; 10 000 (0 pauses the flow) | money supply | MON 4.4.3, 10.4 | P1.7 | `MoneyFlowScaled` | MON 14, by flow ID |
| `SetBirthScales` | `RegionScale{Region; BP}` | 0 to `MaxBasisPoints`; the region's configured `BirthScale`, 10 000 with no row | population | POP 3.6, 6.6 | P7.1(b) | `PopulationScaled` (new) | POP 11, by region |
| `SetDeathScales` | `RegionScale{Region; BP}` | as above, with `DeathScale` | population | POP 3.6, 6.6 | P7.1(b) | `PopulationScaled` (new) | POP 11, by region |
| `SetCapacityScales` | `EndowmentScale{Region, Resource; BP}` | 0 to `MaxShockBP`; 10 000 | resources | RES 6.3.1 | **at the call**: the effective capacity is recomputed and a stock above it is clamped, the excess lost | `EndowmentAdjusted` (kind `shock`) | RES 15, by (region, resource) |
| `SetRegenScales` | `EndowmentScale{Region, Resource; BP}` | 0 to `MaxShockBP`; 10 000 | resources | RES 6.3.1 | P8.3 | `EndowmentAdjusted` (kind `shock`) | RES 15, by (region, resource) |
| `AdjustEndowments` | `EndowmentAdjustment{Region, Resource; Delta}` | `Delta` signed, at most `MaxEndowment` in magnitude; the stock is clamped to [0, capacity] | resources | RES 6.3.2 | **at the call**: the stock changes now | `EndowmentAdjusted` (kind `adjust`) | RES 15 (the stock) |
| `SetOutsideShocks` | `OutsideShock{Node, Commodity; BP}` | 1 to `MaxShockBP`; 10 000 | external | EXT 6.5 | P1.8 (the walk) and the quotes of P3.5 | `OutsideShocked` | EXT 16, by (node, commodity); not hashed |

5.1.1. **The directive is the one slot of an agent**, set by `Direct` (4.3): read at
P1.9, saved by AGT 15, no event. It is in this list of slots by
`spec-world.md` 8.2 and has its own section because it names an agent.

5.1.2. **Naming.** The owners wrote `SetFlowScale(flow, bp)` (MON 10.4),
`SetOutsideShock(node, commodity, bp)` (EXT 6.5) and `SetEdgeStatuses(cause,
changes)` (REG 7.3) and said each name was provisional. The public names are the
plural slice forms above, which is the shape REG gave its call; the singular form
is an entry of one. The edits are in section 19. The slices' entry types
(`YieldShock`, `TransitFactor`, `FlowScale`, `RegionScale`, `EndowmentScale`,
`EndowmentAdjustment`, `OutsideShock`) are plain structs of the root package with
exported fields and no embedded type (architecture.md 4.7). `EdgeStatus` is REG's
type (`spec-regions.md` 7.3) and its fields are as there: the zero `EdgeStatus`
has a `FactorBP` of 0, which is out of bound, so an entry that opens an edge says
`Closed: false, FactorBP: 10000`.

5.1.3. **What the key checks are**, applied in the order of 3.2 (module, then each
entry in the order given):

| Slot | Errors, in the order checked |
|------|------------------------------|
| yield shock | `ErrUnknownRegion` (a `RegionID` outside `[0, n)`, `NoRegion` included), `ErrUnknownRecipe`, `ErrOutOfBounds`, `ErrDuplicateID` |
| edge status | `ErrUnknownEdge` (a pair that is not a configured edge), `ErrOutOfBounds`, `ErrDuplicateID` (REG FR-REG-050) |
| transit factor | `ErrModuleOff`, `ErrUnknownRegion`, `ErrOutsideNode`, `ErrSameRegion` (from equals to), `ErrUnknownMode`, `ErrOutOfBounds`, `ErrDuplicateID` |
| flow scale | `ErrModuleOff`, `ErrUnknownFlow` (new), `ErrOutOfBounds`, `ErrDuplicateID` |
| birth and death scale | `ErrModuleOff`, `ErrUnknownRegion`, `ErrOutsideNode`, `ErrOutOfBounds`, `ErrDuplicateID` |
| capacity and regeneration scale, adjustment | `ErrModuleOff`, `ErrUnknownRegion`, `ErrUnknownResource`, `ErrOutOfBounds`, `ErrDuplicateID` |
| outside shock | `ErrModuleOff`, `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrUnknownLine` (new: the node has no line for the commodity; an ordinary region has no lines, so it returns this too), `ErrOutOfBounds`, `ErrDuplicateID` |

A call that cannot have an effect is refused rather than accepted and ignored:
with the module off the call returns `ErrModuleOff` naming the module; for a key
that cannot exist (an outside node as a region of households) it returns the
sentinel above. A consumer's mistake is then found at the call and not by a run
that never moved.

### 5.2 The rules of a slot

5.2.1. **Default and clearing.** Every slot has a default (the table). A key set to
its default is **cleared**: the owner holds the default, the checkpoint saves what a never-shocked key would save, and `Slots()` (5.3) does not list it. A pack with no shock never
has a non-default entry, and the graph, for one, is then built once and never
again (`spec-regions.md` 7.1.2). The default of a birth or death scale is the
**configured** value of the region's row, not 10 000, because the configured value
is what the module uses when no shock has set it (`spec-population.md` 6.6): a birth or
death scale **replaces** the configured scale, whereas an endowment scale multiplies
the configured row and defaults to 10 000 (`spec-resources.md` 6.3.1).

5.2.2. **No timer.** A slot holds its value until another call sets it. There is no
duration, no expiry and no decay: the consumer sets the shock and clears it at the
tick it chooses (3.6). That is the answer to "its expiry is the integration spec's"
(`spec-production.md` 6.8).

5.2.3. **No cascade and no memory of the past.** Setting a slot does not undo a
past effect: a stock clamped by a capacity scale does not come back when the scale
is cleared (it regenerates, RES 4); a job started under a yield shock keeps its
output; a shipment dispatched under a factor keeps its arrival tick
(FR-TRN-007). Setting a slot does not make a decision for an agent either: every
agent's response is its own rule at the position that reads the slot.

5.2.4. **A slot never edits a configured value** (`spec-world.md` 8.2.1,
`spec-regions.md` 5.3.6). The configured row is the default and stays; a slot is
the difference over it, and the hash names the configuration, not the difference.

5.2.5. **A position reads the value as set at the last call.** A call between two
steps is read by the first step after it; a call made after a step is read by the
next. Two calls to one slot between two steps: the later one wins, with an event
for each change it made (the first call's event is emitted too: it was a change).

5.2.6. **The bounds are the owners'.** `MaxYieldBP`, `MaxEdgeFactorBP`,
`MaxTransitFactorBP`, `MaxFlowScale`, `MaxBasisPoints`, `MaxShockBP` and
`MaxEndowment` are constants of `bounds.go` (`spec-config.md` 4.1). This spec adds
none, and no slot has a number of keys of its own to bound: a key names an entity
of the configuration and appears once in a call, so the size of a slot and of a
call is bounded by the configuration.

### 5.3 Reading the slots back

```go
// SlotState lists every slot value that differs from its default, in the
// canonical order of its owner (the order a checkpoint saves it in).
type SlotState struct {
	YieldShocks    []YieldShock
	EdgeStatuses   []EdgeStatus
	TransitFactors []TransitFactor
	FlowScales     []FlowScale
	BirthScales    []RegionScale
	DeathScales    []RegionScale
	CapacityScales []EndowmentScale
	RegenScales    []EndowmentScale
	OutsideShocks  []OutsideShock
}

// Slots returns a deep copy of the world's slots. It changes nothing.
func (w *World) Slots() (SlotState, error)
```

5.3.1. **What it is for.** A consumer that sets a shock and a scenario that
restores a world to a state (STY, EXP) can read what is set, without a snapshot,
and an empty `SlotState` says that the world holds no shock. It lists the same
keys the owners save, so a restored world reports the list the original did
(FR-INT-032). The one-off adjustments of an endowment are not listed: they are
not a held value, and their effect is the stock (RES 6.4).

5.3.2. **Cost.** `O(non-default keys)`, an allocation. It is a query of the second
class (6.1).

## 6. Queries

A **query** is a read of the world that a consumer makes between ticks without
asking for a snapshot. The first consumer needs prices a designer can read and
explain, a household's state, a region's summary and the calendar
(`vision.md` 2); a player needs the same.

### 6.1 What every query guarantees

6.1.1. **No draw, no state, no event.** A query takes no random number (the draw
count is a function of the configuration and of the calls, `spec-world.md` 7.4.1),
changes no state, emits no event, writes no log above `Debug` and returns a
**deep copy of plain data**: no pointer into live state, no function, no map
(architecture.md 12.4). A consumer may marshal the result on another goroutine
(and may keep it as long as it likes: it does not follow the world).
`FR-WLD-010` is the requirement and this section lists the queries.

6.1.2. **When.** At a tick boundary, on a healthy world. A query with an error
result returns `ErrInStep` from inside a step and `ErrWorldFailed` on a failed
world. A reader with no error result returns its value in both cases
(`spec-world.md` 3.5.1, 3.6.1), and the state it reads is, by kind:

| Readers | Read | In a step | On a failed world |
|---------|------|-----------|-------------------|
| `Tick`, `Err`, `ConfigHash`, `Config` | the clock, the failure, the configuration and its hash | the value (`Tick` advances at P8.6, after the last event) | the value: the tick of the failed step, the failure |
| `Calendar`, `ReferencePrice`, `Warnings`, `Graph`, `Travel` | tables derived from the configuration and the edge statuses, which a step does not change | the value as of the last boundary | the same |
| `Endowment`, `InTransit`, `Away`, `RelocationRefusal` | live state | the value of the state as the running step has left it, which an observer sees complete up to the event it was given (events follow the transition they describe); the value is not a boundary value | the state the failed step left, which is **not** a boundary state: a consumer reads `Err()` first |

Nothing that follows architecture.md 12.5 (an observer, a strategy, a spawner or a
clearing does not call back into the world) and 12.1 (the world is single-threaded)
calls any of these readers from inside a step, so the "in a step" column describes a
call no conforming code makes; it says what the value would be, and the value is the
state the step has left, not a boundary value, because version 1 keeps no copy. A
reader with no error result is listed beside `Tick`, `Config` and `ConfigHash`; this
is not a change of the guard of `spec-world.md` 3.6.1 for methods with an error result.
The `error` of `RelocationRefusal` is the refusal it reports and not an error of the
call. WLD 3.5.1 and 3.6.1 name only four readers (`Err`, `Tick`, `ConfigHash`, `Config`); the edit that lists the others
is in section 19.
A query is legal in the construction phase.

6.1.3. **Cost classes.** The class is a promise about what a consumer can afford
to call per frame, and is stated per query. Counts, not timings (N-015).

| Class | Cost | Allocates |
|-------|------|-----------|
| 1 | `O(1)` | no |
| 2 | `O(the entity)`: the tradable set and recipes of an agent, the non-default keys of a slot, the agents of a region, the warnings | yes, the result |
| 3 | `O(the world)`: the configuration, the graph, the whole snapshot, the checkpoint, the audit | yes |

### 6.2 The list

| Query | Returns | Owner | Class |
|-------|---------|-------|-------|
| `Tick() int64` | the tick the next step will run | WLD 3.7.2 | 1 |
| `Err() error` | the failure that failed the world, nil on a healthy one | WLD 3.5.1 | 1 |
| `ConfigHash() [32]byte` | the hash of the effective configuration | CFG 8.2 | 1 |
| `Config() Config` | a deep copy of the effective configuration, in canonical order (3.4.2) | CFG 3.5 | 3 |
| `Warnings() []Warning` | the warnings of the construction, sorted by path then rule (6.4) | CFG 7.2 | 2 |
| `Calendar() Calendar` | the calendar; `At(tick)` gives the moment, `SeasonAt(tick)` the periodic season | CAL 4 | 1 for `At` |
| `Graph() Graph` | the region graph with the current edge statuses, and its queries | REG 4.3 | 3 |
| `ReferencePrice(r, c) Price` | the effective reference price | REG 5.2.4 | 1 |
| `Market(r, c) (MarketState, error)` | the market report of a region and commodity (6.3); the window's low and high are `O(PriceWindow)`, a bound of the configuration | CLR 5.4, 5.5 | 2 |
| `Region(r) (RegionState, error)` | the region's row of the snapshot (6.5) | OBS | 2 |
| `Agents(r) ([]AgentID, error)` | the live agents whose account is in region `r`, ascending, no pseudo-agent | REG 3.1.1 | 2 |
| `Agent(a) (AgentState, error)` | the state of one agent at the boundary (4.7.2) | AGT 14.6 | 2 |
| `Endowment(r, res) int64` | the units of an endowment available | RES 5.1 | 1 |
| `InTransit(a)`, `Travel(from, to, mode)`, `Away(a)` | the lines of the shipments of an agent; the travel time and haul units of a route; whether an agent is away | TRN 9.2 | 2, 1, 1 |
| `RelocationRefusal(a, to, skip...)` | the reason and error that a relocation would meet, `("", nil)` if none | REG 6.1.6 | 1 |
| `Slots() (SlotState, error)` | the non-default slots (5.3) | this spec | 2 |
| `Audit() error` | nil if every identity of the ledger and the modules holds | LED 12.4 | 3 |
| `Snapshot() (Snapshot, error)` | the aggregates and the sets in flight | OBS | 3 |
| `Checkpoint() (Checkpoint, error)` | the state to resume a run; `RestoreWorld(cfg, c, opts...)` builds a world from it and is not a method of a world | CHK | 3 |

The readers of other specs in the table are listed so that the consumer's
surface is in one place; each is defined, bounded and tested by its owner. This
spec defines `Market`, `Region`, `Agents`, `Agent`, `Warnings` and `Slots`, and
asks the owners for nothing else.

### 6.3 Market

```go
// MarketState is the report of one market at the boundary: what the last step
// published, and the recent window. Plain data; amounts in minor units.
type MarketState struct {
	Region           RegionID
	Commodity        CommodityID
	Price            Price    // the published price (spec-clearing.md 5.4)
	Age              int64    // ticks since a unit last traded; 0 if units traded in the last step
	Volume           Quantity // units traded in the last step
	UnmatchedBid     Quantity // units of bids left unmatched, as MarketCleared
	UnmatchedAsk     Quantity
	BestBid, BestAsk Price    // highest unmatched bid, lowest unmatched ask
	HasBid, HasAsk   bool     // whether each side had an unmatched order
	WindowLen        int64    // ticks of trading held in the price window
	WindowMean       Price    // the window's statistics (spec-beliefs.md 7.2)
	WindowLow        Price
	WindowHigh       Price
}
```

6.3.1. **Source.** It is the record the market keeps for the step just run: the
fields of `MarketCleared` and the window of `spec-beliefs.md` 7. Before the first
step the price is the seed price with age 0 and the sides are empty
(`spec-clearing.md` 5.4). With the number of units traded 0 the price is the
price carried forward and the age says how stale it is; a consumer that shows a
price to a player shows the age with it.

6.3.1a. **An empty window.** With `WindowLen` 0 the mean, low and high are 0 and mean
nothing (`spec-beliefs.md` 7.2: absent); a consumer reads `WindowLen` first. `Market`
reports the published price of any region as it is: which agent may be shown it, and
which price a player of an informed role may be told of another region (4.7.3), is the
consumer's filter, and the library does not apply one.

6.3.2. **Errors.** `ErrUnknownRegion`, `ErrUnknownCommodity`, and `ErrOutsideNode`
for an outside node, which has no market (`spec-regions.md` 3.5).

6.3.3. **Honest about a price.** It is the volume-weighted mean of the trades of
the last step, not a quote and not a promise: an order placed at it may not fill.
The "price a designer can read and explain" is this price with `BestBid`,
`BestAsk` and the window, and the events of the step that made it (`TradeExecuted`,
`MarketCleared`); why a price moved is read from those, not from this query.

### 6.4 Warnings

`Warnings()` returns what `Config.Warnings` returned at construction, so a
consumer with no logger sees them (`spec-config.md` 7.2): a `Warning` is a `Path`
and a `Rule` and plain data, sorted by path then rule. A restored world builds its
tables again from the configuration and holds the same list. The validation
**report** of a configuration that is not yet a world is `Config.Validate()`
(`spec-config.md` 7.1): every error, joined, at most 100, each a
`*ValidationError`; it needs no world and a consumer calls it to check a pack.

### 6.5 Region

`Region(r)` returns the **region state of the snapshot** (`RegionState`, `spec-observability.md` 5.3.5a): the region's row, its role rows, markets, jobs, endowments and the shipments and journeys that touch it; the households unfed and the largest hunger are fields of its role rows. On an outside node it returns the node's row (OBS 5.3.2) and `Agents` an empty list. This spec adds no field: a region summary that the snapshot and this query computed twice would be two truths. The query exists so that a frame that needs one region does not build the whole snapshot. Its cost is the region's share of the scans of the snapshot (`spec-observability.md` 5.5: the agents of the region, its lots and its beliefs), the second class of 6.1.3, and the result equals the filter of a full snapshot by region (`spec-observability.md` 5.3.5a).

### 6.6 What a consumer reads, by need

| Need | Read |
|------|------|
| A household's state, a player's view | `Agent(a)`, with `Market` per commodity and `Region` |
| A published price and how fresh it is | `Market(r, c)` |
| A region at a glance | `Region(r)` |
| Which agents are in a region | `Agents(r)`, then `Agent` |
| The date, the season | `Calendar().At(Tick())` (the moment the next step runs), `At(Tick() - 1)` (the last, once a step has run: a negative tick is refused) |
| What is shocked | `Slots()` |
| Whether the configuration is sound | `Config.Validate()`, `Warnings()` |
| Where a road leads, how far | `Graph()`, `Travel` |
| Everything, for a dashboard | `Snapshot()` |
| Everything, to resume | `Checkpoint()` |

## 7. Errors and guards

### 7.1 The sentinels

Every error is a sentinel matched with `errors.Is` (architecture.md 8.4,
testing.md 3.9) and wrapped with `%w` and the call's name (architecture.md 8.2);
the library does not log an error it returns. The sentinels the calls of this spec
return, and who defines each:

| Sentinel | Meaning | Defined by |
|----------|---------|------------|
| `ErrInStep` | the call was made from inside a step | WLD 3.6 |
| `ErrWorldFailed` | the world failed an earlier step; wraps the cause | WLD 3.5 |
| `ErrModuleOff` | the module the call needs is off | this spec (new) |
| `ErrOutOfBounds` | a number is outside its bound; carries path, value, bound | CFG 7 |
| `ErrDuplicateID` | a key appears twice in a call | CFG 7 |
| `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrUnknownRole`, `ErrUnknownRecipe`, `ErrUnknownMode` (defined with the merchant block, MER; `SetTransitFactors` needs it with only transport on, so it is a root sentinel: edit row), `ErrUnknownResource`, `ErrUnknownEdge`, `ErrUnknownStrategy` | an ID names nothing | the specs that own the entity |
| `ErrUnknownFlow` | a flow ID names no flow of the configuration | this spec (new) |
| `ErrUnknownLine` | no outside node has a line for the (node, commodity) | this spec (new) |
| `ErrOutsideNode` | the region is an outside node: it has no agents and no market (`Region` is not refused: it returns the node's row, 6.5) | REG 6.1.3, EXT |
| `ErrUnknownAgent` | the ID names no agent: never issued, or a pseudo-agent (but `Relocate` returns `ErrNotRelocatable` for a pseudo-agent, 7.1.1) | REG 6.1.3 (one sentinel, the same meaning, N-075) |
| `ErrAgentRemoved` | the ID was an agent and has been removed; wraps `ErrUnknownAgent` | this spec (new) |
| `ErrNotExternal` | the call needs an external agent and the agent is not | this spec (new) |
| `ErrNotDirected` | the agent's strategy is not the library's `Directed` | this spec (new) |
| `ErrNotOnRole` | a directive's recipe is not on the role's list | this spec (new; the reason `not_on_role` of PRO 4.2 as an error) |
| `ErrNotTradable` | a directive's commodity is not in the role's tradable set | this spec (new; the reason `not_tradable` of CLR 3.2 as an error) |
| `ErrOverflow` | creating the agent's money or goods would take a balance or a total above its bound (the ledger's `ErrBalanceOverflow`, mapped) | AGT 13; the ledger's is LED 4.2 |
| `ErrSameRegion`, `ErrOutsideNode`, `ErrOpenReservation`, `ErrRegionBound` (`ErrClosedAccount` cannot occur for a live agent; `ErrNotRelocatable` for an estate's ID) | the other refusals of a relocation (`spec-regions.md` 6.1.3); `ErrRegionBound` names the obligation | REG |

7.1.1. **One sentinel for an unknown agent (N-075).** REG's `relocate` and this
spec's calls both say "the ID names no agent". It is one root sentinel,
`ErrUnknownAgent`, with one meaning, so that `errors.Is` works across calls. A
removed agent's ID is the more specific `ErrAgentRemoved`, which the world can
tell because an ID is issued once: an ID below the next ID that is not live and
is not a pseudo-agent was an agent that has been removed. It needs no table of
dead IDs. The ID of a pseudo-agent (an estate, a counterparty of the outside) is not
an unknown agent to `Relocate`: REG owns the refusals of a relocation, and
`Relocate` and `RelocationRefusal` return `ErrNotRelocatable` for it (3.4.3,
`spec-regions.md` 6.1.3), while `Agent`, `Direct` and `RemoveAgent` return
`ErrUnknownAgent`.

### 7.2 The guards, in order

The order of checks of every call is 3.2. Two things follow from it. A call on a
failed world is refused before its arguments are read, so a consumer that retries
a call gets `ErrWorldFailed` and not a misleading `ErrOutOfBounds`. A call made
from inside a step is refused before it can read a state that a step is changing.

### 7.3 A failed world

On a failed world every call of this spec returns `ErrWorldFailed` wrapping the
failure and changes nothing, **including `Direct`, `RemoveAgent` and `Relocate`**:
the world is not continued, it is restored from a checkpoint the consumer took at
an earlier boundary (`spec-world.md` 3.5.2). The four readers of
`spec-world.md` 3.5.1 still answer. A consumer that wants to know why calls its
`Err()`.

### 7.4 What is a programmer error

An argument that can be wrong is an error that is returned (architecture.md 8.1).
A violated internal invariant found while a call applies (an account that cannot
be opened for an ID that passed every check) is a programmer error and panics with
a message that says so; the call is atomic, so a panic from a check that passed
means a defect in the library, not in the consumer's call.

## 8. The shape of a consumer's loop

No code: the calls are in sections 4 to 6 and the names of construction, `Step`,
`Run`, `Checkpoint` and restore are WLD's and CHK's. In prose, a game that runs a
realm with one player does this.

It **constructs** the world: it loads a pack with `config/`, passes the options it
wants (a seed, an observer that keeps the events of the player's agent, a logger,
`strategy.Options()...` if a role names `merchant`) and reads `Warnings()`. From
`Config()` it builds, once, the table from its own names to typed IDs (3.4.2).

It **adds the player**: `AddAgent` with a region, a role, the money and the goods
the player starts with and `Mortal` as the game wants. The ID it gets back is the
only handle on the player. It may do this before the first step or at any later
boundary; the player's money and goods are a named faucet either way (4.1.4).

It may **run in** a number of ticks with `Run(ctx, n)`, so that the player meets a
market with a history and not a seed price.

Then it loops, once per game time unit. It **reads**: `Agent(player)` for what the
player has, `Market` for the prices of the commodities it can trade and how fresh
they are, `Region` for the town's summary, and the events of the last step for what
happened to its orders. The player **decides**, and the consumer **directs**: one
`Direct` with the orders, discards and job the player chose, which applies to the
next step and is gone after it unless the player made it a standing order. The game
may **shock** the world as its story needs: a failed harvest is `SetYieldShocks`
before the season's starts, a closed road `SetEdgeStatuses`, a plague
`SetDeathScales`, and each is cleared by the call that sets the key back to its
default. Then it **steps**.

The player's order is placed in the trade phase of that step and filled, or not,
in its clearing: one tick of latency, by design (3.6). What the player reads after
the step is the outcome of the directive it gave before it.

Every so often it **saves**: `Checkpoint()` at a boundary. A restore brings back
the world, the mailbox and every slot; the consumer re-registers its extensions by
name and its observers, as they are not saved. If a `Step` returns an error the
world is failed, and the consumer restores the last checkpoint and may choose
another extension (`spec-world.md` 3.5.2). The consumer's own state (what the
player was shown, its quest log) is its own and is not in the checkpoint.

## 9. What INT hands to EXP, STY, CHK, OBS and IND

### 9.1 To STY and EXP: injecting a shock

A scenario of STY and a run of EXP drive a world through exactly this surface and
no other: `NewWorld`, `AddAgent`, `Direct`, the slot setters, `Relocate`,
`RemoveAgent`, `Step` and `Run`, and the queries of section 6. A scenario script
is a list of (tick, call); the harness applies each call at the boundary before
the tick it names, with its own tick counter. Run with the same seed and the same
list it gives the same result on every platform (3.5.1). Two calls that touch
different keys leave the same state in either order (their events come in the
order of the calls); two calls to one key do not commute, nor do
`AdjustEndowments` and `SetCapacityScales` on one endowment (the clamp), nor two
calls of `AddAgent`, which take their draws in call order (D12).

| A shock in a scenario | The call | Notes |
|-----------------------|----------|-------|
| a failed harvest | `SetYieldShocks` on (region, recipe) with a multiplier below 10 000 | read at job start; clear it to end |
| a bumper harvest | the same, above 10 000 | bounded by `MaxYieldBP` |
| a blight that kills the stock | `AdjustEndowments` with a negative delta | a one-off |
| a drought that slows recovery | `SetRegenScales` | the stock is untouched; clear it to end |
| a drought that shrinks the land | `SetCapacityScales` | a stock above the new capacity is lost at the call |
| a discovery | `AdjustEndowments` with a positive delta | creates no goods; a stock in the ground |
| a closed road, a flood | `SetEdgeStatuses` with `Closed` | a shipment in flight keeps its arrival tick |
| a damaged or improved road | `SetEdgeStatuses` with a `FactorBP` | the effective length of REG 7.2 |
| a blocked trade route for carriers only | `SetTransitFactors` with 0, or a slower or faster factor | new dispatches only |
| a plague, a famine death rate | `SetDeathScales` | a schedule's multiplier, not a removal |
| a baby boom, a ban on formation | `SetBirthScales` | |
| a stipend raised, paused or cut | `SetFlowScales` | a declared flow; 0 pauses it |
| a foreign supplier fails, a rival opens | `SetOutsideShocks` | a factor on the outside price |
| an embargo | `SetEdgeStatuses` on the gate edge | a node's line is cut by a closed edge |
| a refugee household, an expulsion | `Relocate` | a forced move |
| a player trades, a merchant is added | `AddAgent`, `Direct` | |
| a tax | a levy flow in the configuration, scaled by `SetFlowScales` | FIS is later |

What is not expressible, from 3.6: an act-now trade, goods dropped on a market, a
partial embargo, a shock that ends by itself, a shock on a cohort of households
only (a flight of part of a cohort).

### 9.2 To CHK

INT contributes one piece of state, the rumours of 4.7.3; it saves nothing else.
The mailbox, the flags and the slots are saved by the specs that own them (AGT 15
and the column "Saved by" of 5.1); a restored world answers `Slots()`, `Agent` and
`Direct` as the original did. The names of the calls that save and restore are
CHK's and INT lists them in its surface (6.2) and adds no second path. The
**sequence of calls is not saved**: it is the history, not the state, and a
restored world has none. A consumer that wants a replay keeps its own log of calls
with the seed and the configuration (ADR 0002 decision 9).

### 9.3 To OBS

The events each call needs, with the owner and the fields the call knows
(section 12 lists them). The player's per-agent record is a query, `Agent(a)`
(4.7.2), and the snapshot holds aggregates; the region row of the snapshot is what
`Region` returns. Three events are new and are OBS's to define: `YieldShocked`,
`TransitFactorChanged` and `PopulationScaled` (WLD 16.2 asked for them).

### 9.4 To IND

A measure may exclude external agents by the `External` flag that every agent and
`AgentSpawned` carries: a player is a household in the counts (4.1.5), and a
measure that does not want the player in a regional average filters it. IND
decides: the household rows of a measure are built from `Agents(r)` and `Agent(a)`
(`spec-indicators.md` 4.5.2), `RowOpt.ExcludeExternal` is the filter, and a bulk
read of one column is IND's open question 21 (1) and is not offered. Whether a regional difference is real before a cap or a shock is checked
is WLD-A2's warning, and a shock the consumer set is visible in `Slots()` and in
the events of the call.

## 10. Parameters

This spec adds **no `Config` field, no default and no row** to the parameter-owner
table (architecture.md 9.5, 9.7; N-016). A consumer's call is a datum of the run,
not of the configuration (3.5.1), and every bound a call applies belongs to the
slot's owner (5.2.6). A test of the parameter registry (testing.md 6.7) needs
nothing from this spec.

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the
order of checks of a call (3.2); the cause `consumer` (3.3.3); the shape of a
setter, one entry per key (3.3.2); the fixed order of the effects of `AddAgent`
(4.1.2) and of `Relocate` (4.5.2); the built-in strategy name `directed` (4.2.2);
the names of the calls and of the slot entry types, which are provisional until
the code, as the names `Err`, `Tick`, `ConfigHash` and `Config` are in
`spec-world.md` 3.7.2.

**Not parameters:** the mailbox, a slot's value, the next agent ID.

## 11. Validation

11.1. **Configuration.** This spec adds no rule to `Validate` and no warning: it
has no configuration. A `Role.Strategy` may name `directed` (it is a registered
name like `default`); a role whose every household is directed has no mailbox that
anyone fills, which is harmless and is not warned about.

11.2. **Errors this spec adds** (sentinels at the root, mapped as N-075 asks; each
has a failing-case test, testing.md 3.9):

| Sentinel | When | Where |
|----------|------|-------|
| `ErrModuleOff` | a call that needs a module that is off (5.1.3) | the setters of 5.1 |
| `ErrUnknownFlow` | `SetFlowScales` names a flow the configuration does not have | `SetFlowScales` |
| `ErrUnknownLine` | `SetOutsideShocks` names a (node, commodity) with no line | `SetOutsideShocks` |
| `ErrAgentRemoved` | an agent call names the ID of a removed agent; wraps `ErrUnknownAgent` | the agent calls and `Agent` |
| `ErrNotExternal` | `Direct` or `RemoveAgent` on an agent that is not external | those |
| `ErrNotDirected` | `Direct` on an external agent whose strategy is not `Directed` | `Direct` |
| `ErrNotOnRole` | a directive's recipe is not on the role's list (nor its effective own-use list) | `Direct` |
| `ErrNotTradable` | a directive's commodity is not in the role's tradable set | `Direct` |

The other errors the calls return are existing sentinels (7.1).

11.3. **What is not validated here:** that a shock is sensible (a yield of 100 000
for every recipe is legal), that a slot's value is a change a story needs, and
that a directive is a good idea. The library guards conservation and determinism,
not the consumer's design (ADR 0001, consequences).

## 12. Observability

12.1. **Events**, at the call, with the `Tick` the next step will run, before its
`TickStarted` (`spec-world.md` 6.11), only when an observer was supplied
(observability-and-logging.md 4.5). The fields are the owner's and OBS's; this
table says what each call needs and where an event is **new** and so asked of OBS.

| Call | Event(s) | New? | Fields the call knows |
|------|----------|------|-----------------------|
| `AddAgent` | `AgentSpawned` | no | agent, region, role, weight 1, `StartMoney`, `External` true |
| `Direct` | none | | (`spec-world.md` 8.3) |
| `RemoveAgent` | `RecipeAbandoned`, `TraineeshipEnded`, `ContractSettled`, `AgentBankrupted` (`Cause` `consumer`), `EstateOpened` | no | as the removal of AGT 3.6 |
| `Relocate` | `RecipeAbandoned` (`region_changed`), `AgentRelocated` (`Cause` `consumer`) | no | as REG 10.1 |
| `SetYieldShocks` | `YieldShocked` | **yes** | region, recipe, before and after (basis points) |
| `SetEdgeStatuses` | `EdgeStatusChanged` | no | from, to, closed, factor, cause `consumer` |
| `SetTransitFactors` | `TransitFactorChanged` | **yes** | from, to, mode, before and after (basis points) |
| `SetFlowScales` | `MoneyFlowScaled` | no | flow, from, to |
| `SetBirthScales`, `SetDeathScales` | `PopulationScaled` | **yes** | region, which (`birth` or `death`), before and after (basis points) |
| `SetCapacityScales`, `SetRegenScales`, `AdjustEndowments` | `EndowmentAdjusted` | no | kind (`shock` or `adjust`), the scales after, stock before and after, units lost (RES 14) |
| `SetOutsideShocks` | `OutsideShocked` | no | node, commodity, from, to |

The three new names follow the proposals of OBS's decisions so far
(`YieldShocked`, `TransitFactorChanged`, `PopulationScaled`); the final names and
fields are OBS's, which has adopted them (`spec-observability.md` 4.1.8). Every event is plain data with no map, a past-tense name and
fixed-point values (observability-and-logging.md 4.1). A call that changes `n` keys
emits `n` events, bounded by the configuration (3.3.2); none is per agent except
those of an agent call.

12.2. **Logs.** Constant lowercase messages, the keys of the rules' table 3.3.2 (no
new key), all at `Debug` and guarded by `Enabled`: "agent added" (`agent`,
`region`, `role`), "directive set" (`agent`), "agent removed" (`agent`, `cause`),
one constant message per setter, "yield shocks set", "edge statuses set" and so on (`tick`, `cause` with the value `consumer`; the number of keys changed is the number of events, so no key is added). No `Info`: a
consumer's calls are not a lifecycle of the library, and a server may make many. No
`Warn` or `Error`: a refused call is a returned error, handled once by the caller
(architecture.md 8.3, observability-and-logging.md 2.2). The value `consumer` joins
the constants of the key `cause`: it is an exported constant of the root package
(`event.go`), and `internal/logkeys` holds the key strings only.

12.3. **Metrics.** None are new. `economysim.agent.count` and
`economysim.household.count` count an external agent (it is an `AgentSpawned`
event with weight 1), and `economysim.household.relocations` counts a forced move
(an `AgentRelocated`), so the catalogue needs nothing for this spec
(observability-and-logging.md 5.3.1). A shock is not a metric: it is state, read
from `Slots()` and the snapshot. A consumer that wants a metric of "players" filters
`External`.

12.4. **Snapshot.** Whether the snapshot lists the non-default slots is OBS's;
`Slots()` is the query. The per-agent record is `Agent(a)`, not a snapshot field
(OBS's decision, 4.7.2).

12.5. **Cost with no observer.** No event of this spec is constructed without an
observer. The calls allocate (an account, a belief table, the result of a query):
they are between ticks and outside the allocation test of a step
(observability-and-logging.md 8.2). What a step with an external agent adds is
the agent's own orders and the draws of 7.3, with the scratch the world already
reuses; an allocation test over a `Step` of the reference chain with one external
agent holds the baseline constant plus nothing per call (NFR-INT-002).

12.6. **The Observability entry** (observability-and-logging.md 9), in short: new
failure paths are returned (the sentinels of 11.2) and not logged; three state
transitions had no event and OBS has adopted them (`YieldShocked`,
`TransitFactorChanged`, `PopulationScaled`), the others reuse an event of the
rules; no new decision point of a strategy; no new loop; nothing is logged above
`Debug`; no new metric and no new metric attribute; tests of
observability-and-logging.md 8.1 are FR-INT-050 to 054.

## 13. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's:

| State | Why it cannot be derived | Order |
|-------|--------------------------|-------|
| For each external agent whose role is `Informed`, with the information module on: the **tick of its last view** and the **rumours of that view** as they were given to it (region, commodity, price, mean, lag, source age), at most `MaxRumoursPerView`, in the order the view had (ascending region, then commodity) | they were built with error draws from the stream and cannot be rebuilt without draws (4.7.3) | section 22 of the checkpoint (`spec-checkpoint.md` 4.2, tag 22), in ascending agent ID, rows only for an external agent that has had a view |

13.1. **What is not saved by this spec.** The mailbox, the flags and the strategy
name are AGT's (15). The slots are the owners' (5.1, "Saved by"). The sequence of
calls is not state (9.2). No call is pending: a call is applied when it is made.

13.2. **Effect on results.** None: the strategy that was given these rumours has
used them, and no position reads them again. They are mandatory in a
checkpoint when `Modules.Information` is on: a restore without them would answer
`Agent` differently (`spec-checkpoint.md` 5.3.2, FR-CHK-052), although they do not
affect a result (FR-INT-049).

13.3. **A restore before the first step.** A checkpoint taken after `NewWorld` and
before the first `Step` stays in the construction phase after a restore, and a
call made then is still inside it (`spec-checkpoint.md`; the wording of
`spec-world.md` 5.5.3 and 12.2 is CHK's seam ask), so 3.1.1 holds for a restored
world as for a new one.

## 14. Worked examples

All settings are invented **examples**: the words in them (`alpha`, `beta`,
`grain`, `loaf`, `baker`, `grow`, `cart`) are flagged as example data, never
branched on (N-017), and every number is a placeholder for illustration
(N-027). The arithmetic was done by hand and re-checked; no model was run (15).

### 14.1 Example A: a player added before the first step

The world of `spec-world.md` 19.1: three regions `alpha` (RegionID 0), `beta` (1)
and `edge` (2, an outside node), commodities `grain` (0) and `loaf` (1), agents 4
to 10 (households), construction taking 14 words (7 agents, 2 draws each). The
consumer adds a player:

```
AddAgent(NewAgent{Region: 0, Role: baker, Strategy: "", Money: 2 000,
                  Stock: [{grain, 10}], Mortal: false})
```

| What | Result | Why |
|------|--------|-----|
| the ID | 11 | the next ID after the households 4 to 10 (`spec-world.md` 5.3); no other ID changes |
| the strategy | `directed` | the empty name (4.2.2) |
| the account | the next account ID after the agent accounts `B+11 .. B+17`, that is `B+18` | `spec-world.md` 5.2 |
| money and goods | 2 000 and 10 `grain` by `initial`, in the construction bucket | before the first step (3.1.1) |
| the draws | words 14 and 15 (the target ratio, then the residue) | D12; the first word of tick 0 is then word 16, not 14 |
| the event | `AgentSpawned{Tick 0, Agent 11, Region 0, Role baker, Weight 1, StartMoney 2 000, External true}` | emitted at the call, after the 7 initial `AgentSpawned` events |
| `Agent(11)` | `Money` 2 000, holdings `grain` 10 (`Expiring` as the shelf life says), no job, `Hunger` 0, beliefs at the seed prices, `Mortal` false, rumours empty (no view yet) | 4.7.2 |

Then `Direct(11, Directive{Intents: [{loaf, Bid, 3, 40}]})` (three units of `loaf`
at a limit of 40) is accepted: `loaf` is in the tradable set of `baker`
(`ErrNotTradable` would say otherwise), the quantity and the limit are in bound.
The first `Step` reads it at P1.9, the fallback's plan has its intent for `loaf`
replaced by this one at P3.2, and the order is admitted if the player holds 120
units of money and rejected with `insufficient_funds` otherwise (3.2.1). With 2 000
it is admitted and appears as `OrderPlaced{Agent 11, ...}` in tick 0.

Without the call the first word of tick 0 is word 14: **adding an external agent
changes the stream from the call onward**, which is why the call is part of the
sequence that fixes a run (3.5.1).

### 14.2 Example B: an agent added mid-run

At the boundary after tick 6, `Tick()` is 7. `AddAgent` as above returns the next
ID. Its events and positions, in time order:

1. at the call: the two draws; `AgentSpawned{Tick 7, ...}`; the money 2 000 and 10
   `grain` are by `initial` into the totals of tick 7, because a call between
   ticks is totalled in the tick the next step runs (`spec-world.md` 8.1.3);
2. `Step` runs tick 7: `TickStarted{7}`; at P1.9 its directive is read; at P2.5
   `Work` is called (it is free); at P3.2 `Trade`; at P4.1 its order clears;
3. `BoundaryFlow` of tick 7 at P8.2 reports `initial` for 2 000 money in region
   `alpha` and 10 `grain`; with `WithAudit` the audit at P8.4 passes (boundary
   balance = construction bucket + the totals of every tick).

### 14.3 Example C: a failed harvest and a damaged road

Region `alpha` (0), recipe `grow` (0) with a base output of 100 units per
execution and a duration of 20, a road `alpha` to `beta` of distance 30, a mode
`cart` of speed 10.

| Tick (boundary before it) | Call | What the tick then reads |
|---------------------------|------|--------------------------|
| 40 | `SetYieldShocks([{Region 0, Recipe 0, BP 3 000}])` | event `YieldShocked{Tick 40, Region 0, Recipe 0, BeforeBP 10 000, AfterBP 3 000}`. A job of `grow` started at tick 41 at P2.7 has its output fixed then: `1 * 100 * 3 000 / 10 000 = 30` units. A job started at tick 30 and completing at tick 49 keeps its output of 100: the pending output was fixed at its start (`spec-production.md` 6.4) |
| 40 | `SetEdgeStatuses([{From 0, To 1, Closed true, FactorBP 10 000}])` | `EdgeStatusChanged{Tick 40, From 0, To 1, Closed true, FactorBP 10 000, Cause consumer}`; the graph tables are rebuilt at the call; a shipment dispatched at tick 38 with `Arrives` 42 still arrives at 42 (FR-TRN-007); a request at tick 40 for `alpha` to `beta` is refused `no_route` at P3.2 if no other path exists |
| 50 | `SetEdgeStatuses([{From 0, To 1, Closed false, FactorBP 15 000}])` | the road is open and damaged: effective length `max(1, ceil(30 * 15 000 / 10 000)) = 45` (`spec-regions.md` 7.2); the travel time of a `cart` is `max(1, ceil(45 * 10 000 / (10 * 10 000))) = ceil(4.5) = 5` ticks, against 3 at the configured distance (30 * 10 000 / 100 000 = 3) |
| 50 | `SetTransitFactors([{From 0, To 1, Mode cart, BP 5 000}])` | new dispatches only: `ceil(45 * 10 000 / (10 * 5 000)) = 9` ticks |
| 55 | `SetYieldShocks([{0, 0, 10 000}])` | the key is back at its default: `YieldShocked{BeforeBP 3 000, AfterBP 10 000}`; `Slots().YieldShocks` is empty and a checkpoint saves nothing for it |

An entry equal to the value a key already holds (the same `SetYieldShocks` made
twice) is accepted and emits no event (3.3.2). A call with `BP: 100 001` is
`ErrOutOfBounds` with path `changes[0].bp` and the bound `MaxYieldBP`, and nothing
changes.

### 14.4 Example D: a forced move of a migrant

Household 7 (role `grower`, region `alpha`) left on a migration journey to `beta`
that arrives at tick 20. At the boundary before tick 17 the consumer calls
`Relocate(7, beta)`:

1. the dry run `RelocationRefusal(7, beta, "on_journey")` finds nothing else;
2. the journey record is dropped; its household is counted in `Ended` of the next
   `MigrationReviewed`, emitted at P7.2 of tick 17;
3. its job was abandoned at departure; none is running; the own-use flag is
   cleared;
4. the account's region becomes `beta`; `AgentRelocated{Tick 17, Agent 7, From
   alpha, To beta, Cause consumer, Weight 1, Money, Units}`;
5. its beliefs are reseeded from the window of `beta`; `RoleSince` is 17.

At tick 17 it is passed to `Work` and `Trade` in `beta`. Had the agent been the
payer of an open delivery contract, step 1 would have returned `ErrRegionBound`
naming `open_contract` and nothing would have changed. Had the call named the
outside node `edge`, step 1 returns `ErrOutsideNode`.

## 15. What was run

Nothing. This spec has no number of its own to derive: its bounds are the owners'
(5.2.6) and the numbers of section 14 are arithmetic of one line each, which were
checked by hand. No script is archived under `doc/specs/models/int/`. No
default is claimed stable (N-027).

## 16. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. The checklist
of `spec-stability.md` 9 was followed: 9.1 each call was read for the chain it
creates. **This spec adds no feedback loop of its own**: every call is a step
input from outside, and nothing in the library reads a measure and changes a rate
through it (the slots are written only by the consumer, `spec-money-supply.md`
10.4). It adds two hazards and one bias, and one damper, because the surface is
where a consumer can build a loop the engine does not contain.

**INT-A1 The consumer closes a loop through the surface.** *Kind:* hazard.
*Loop:* a rule of the consumer reads `Market`, `Region`, `Agent` or an indicator
and sets a slot or a directive each tick or every few: price level to `FlowScale`
(a mint), unemployment to a yield shock (a story), a price to a player's order.
That is a policy actor, built outside the engine, and ADR 0001 decision 4 accepts
it. *Delay:* at least one tick: it reads at the boundary and the tick reads the
slot at its position (5.1); a slot read at P8.3 or at job start adds that
position's own delay. *Owner:* INT 5, ADR 0001, `spec-world.md` 9.4.
*Damper:* the bound of each slot (INT-D1) and the one-tick delay; the engine adds
no controller and does not know the consumer has one. *Undamped:* the consumer's
own gain. *Condition:* the gain of such a rule times its delay must obey the
bounds of the loops it feeds (`spec-stability.md` 6; STB-L84 and L85 give the
form of a delayed gain): a rule that sets a flow scale at a gain above the
condition of the loop it acts on cycles, and conservation and determinism still
hold. *Test:* INT-S2 (the surface works as a controller input, not a claim that
any controller is stable). *Status:* hand.

**INT-A2 A call is a faucet by another name.** *Kind:* bias. *Loop:* none:
`AddAgent` creates money and goods by `initial` outside any declared flow, so a
consumer that adds agents with money raises the money supply by the calls. It is
visible: `BoundaryFlow` reports it under `initial` in the tick it falls in, and
the identities hold. *Owner:* INT 4.1.4, `spec-money-supply.md` 9. *Damper:*
none; the stated, counted amount (`MaxMoney`, the overflow check). *Undamped:* by
design. *Condition:* a measure of price drift in a run with `AddAgent` calls
reads the `initial` flow before it reads an inflation (`spec-money-supply.md` 8.2,
9.1). *Test:* INT-S3. *Status:* hand.

**INT-A3 An agent the library will not remove.** *Kind:* bias. *Loop:* none: a
non-mortal external agent is outside the failure ladder, so it can stay unfed for
as long as it holds out, draw relief on a ratio that stays low, and keep goods and
money out of circulation. It counts in the households of its group and in the
role's wage and welfare (`spec-spawner.md` 9.3). *Owner:* INT 4.6, SPN 9.
*Damper:* the `Mortal` flag; the counters of `UpkeepShortfall` and `ReliefPaid`
include it. *Undamped:* by design. *Condition:* a regional mean that includes a
player that cannot die is read with the `External` flag in view (9.4); a consumer
that wants a player that can fail sets `Mortal` and provides the spawner or the
call. *Test:* INT-S4. *Status:* hand.

**Damper INT-D1** *A call is validated, bounded and atomic* (3.3, 5.2.6): every
value is checked against the owner's bound before any is applied, a key appears
once, and a failed call changes nothing. It is what keeps a shock from
overflowing a product (`MaxYieldBP` against `MaxQuantity`, `spec-config.md` 4.2),
a half-applied disaster from being observed and a rejected call from drawing. It
adds no parameter: the bounds are the owners'. *Status:* specified here.

**Scenarios** (for `spec-stylised-facts.md` and `spec-experiments.md`, which own
the bands; this spec gives the calls). **INT-S1** *Failed harvest and recovery*:
`SetYieldShocks` of 3 000 on the staple recipe of one region of `three` or `five`
for `k` ticks, then cleared; pass when conservation holds each tick, prices stay
inside the belief bounds and the region recovers within a time STY sets.
**INT-S2** *A closed road and its reopening*: `SetEdgeStatuses` closed on a trade
edge, then open; the price gap of the two regions opens and then closes to the
freight (STY's band); also a consumer rule that scales a flow from a published
price, to show the surface works as a controller input (INT-A1). **INT-S3** *The
sequence of calls is the input*: two runs with the same seed, configuration and
calls give the same state at every tick and the same events (golden); two orders
of calls that touch different keys give the same state; a run with `AddAgent` in
its list shows `initial` in `BoundaryFlow` and passes the audit. **INT-S4** *A
player, a checkpoint and a restore*: add a player and a directive and a shock,
checkpoint, restore, run; the mailbox, the slots and `Agent` equal the original's
and the result equals an uninterrupted run; a non-mortal player at hunger stays and
a mortal one is removed by a consumer's spawner. **INT-S5** *A forced move*:
`Relocate` of a household on a journey and of one with an open contract (refused);
conservation holds and `Ended` counts the household.

## 17. Notes settled

Each open note tagged `INT` or `ALL`, and the others that name the surface, is
applied, rejected or left open here, with the text to append in the handoff.

| Note | Decision | Where |
|------|----------|-------|
| N-022 The player and other external agents | **Applied; close.** `AddAgent` (weight 1, `External`, `Mortal`, the strategy), `Direct` and the mailbox, when each acts, removal by the consumer only, the refusal of the spawner and of MIG, what the player is told, the directive in the checkpoint | 4 |
| N-031 What the calendar hands to other specs | **Applied for INT:** a consumer converts ticks to dates with `Calendar().At`; a shock by season is set before and cleared after the window (there is no season key, because the yield is read at job start, 3.6). **Leave open** for OBS and GDS as before | 6.2, 3.6 |
| N-036 Production's draws, phases and shocks in the world | **Applied for INT; close its INT part:** the yield-shock slot's public call is `SetYieldShocks`, read at P1.2, P2.5 and P2.7, cleared by setting 10 000; it has no timer | 5.1, 5.2.2 |
| N-051 What the Strategy contract needs from other specs | **Applied for INT:** `AddAgent`, `Direct` and the mailbox. **Leave open** for CHK (the strategy names, the directive as state) | 4 |
| N-075 Mapping the ledger's sentinels | **Leave open**, with this added: INT adds `ErrModuleOff`, `ErrUnknownFlow`, `ErrUnknownLine`, `ErrAgentRemoved`, `ErrNotExternal`, `ErrNotDirected`, `ErrNotOnRole` and `ErrNotTradable`; `ErrUnknownAgent` is one sentinel of one meaning for REG and INT (7.1.1) | 7.1, 11.2 |
| N-079 External agents are mortal only when the consumer says so | **Applied; close.** `NewAgent.Mortal`, default false; the library acts on a mortal external agent only through a consumer's spawner or call; the consumer's `RemoveAgent` ignores the flag | 4.1, 4.4, 4.6 |
| N-090 What MIG and the specs that tie obligations to a region must do with a relocation | **Applied for INT:** the public face of `relocate` is `Relocate`, for any agent, with the journey's cancel inside it (4.5). **Leave open** for SKL (to register `in_traineeship`) and for the later modules that keep an account in a region (the rule of 2026-10-10) | 4.5 |
| N-091 The world slots INT must expose | **Applied; close.** The table of 5.1: nine setters and one adjustment, with public name, arguments, errors, bound, event, position and saved state; a later spec that adds a slot adds a row (5.1) | 5 |
| N-116 The outside's slots | **Applied; close.** `SetOutsideShocks`; `SetEdgeStatuses` for an embargo; a capacity multiplier is not provided (3.6) | 5.1, 3.6 |
| N-016 Keep the parameter registry true | **Applied:** this spec adds no field and no row (10) | 10 |
| N-017 Words and names | **Applied:** the terms of the handoff go to the glossary (agent state, forced move, market state, slot state, query class, entry, key); the examples are flagged setting words | 14 |
| N-027 No default macro constant is stable | **Applied:** no default and no number of its own; the numbers of 14 are placeholders | 15 |
| N-103 Running several spec writers in parallel | **Applied:** no shared file was edited; the proposals are in the handoff with provisional IDs (`INT-A1` to `A3`, `INT-D1`, `INT-S1` to `S5`); no model was run; the host `python3` was never called | 16 |
| N-012 Events, metrics and log keys to define | **Leave open**, with this added: three events asked of OBS (`YieldShocked`, `TransitFactorChanged`, `PopulationScaled`), the value `consumer` of the key `cause` (12) | 12 |
| N-020 What each spec contributes to a checkpoint | **Leave open**, with this added: INT contributes the rumours of the last view of an external agent (13) | 13 |
| N-120 A migrant carries its whole account for free | **Leave open**, with this added: a forced `Relocate` carries the whole account too and has no capacity (4.5.4) | 4.5.4 |

Proposed new notes are in the handoff.

## 18. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. This
spec has no default or bound of its own.

### Functional

| ID | Requirement |
|----|-------------|
| FR-INT-001 | A call that changes the world MUST return `ErrInStep` when made from inside a step and `ErrWorldFailed` (wrapping the cause) on a failed world, before it reads its arguments, and MUST change nothing (3.1.2, 3.2, 7.3). |
| FR-INT-002 | A call MUST validate every entry before it applies any; on any error no state MUST change, no event MUST be emitted and no random number MUST be drawn (3.3.1). |
| FR-INT-003 | A slot setter MUST reject a key that appears twice with `ErrDuplicateID`, MUST accept an empty or nil slice as a no-op with no event, and MUST treat an entry equal to the key's current value as no change and no event (3.3.2). |
| FR-INT-004 | A call MUST emit one event per key that changed, at the call, carrying the `Tick` the next step will run, before that step's `TickStarted`, in the canonical order of the key and not the order of the slice, and the cause of an event that carries one MUST be `consumer` (3.3.3). |
| FR-INT-005 | An argument error MUST be a `*ValidationError` whose `Path` names the argument, matched by `errors.Is` with its sentinel; the checks of a call MUST run in the order of 3.2, the agent before the directive for `Direct`, and in the order of `spec-regions.md` 6.1.3 for `Relocate` (3.2, 3.2.2, 4.3.2). |
| FR-INT-006 | A call that needs a module that is off MUST return `ErrModuleOff` before it reads an entry, and a call that cannot have an effect for its key MUST return the sentinel of 5.1.3 and not be accepted and ignored (5.1.3). |
| FR-INT-007 | No call other than `AddAgent` MUST draw a random number, `AddAgent` MUST take the two creation draws at the call after its checks, in call order, and a refused call MUST draw none (3.5.2). |
| FR-INT-008 | A call MUST NOT be part of the configuration hash, and a run MUST be a function of the configuration, the seed and the sequence of calls: the same inputs MUST give a bit-identical run (3.5.1). |
| FR-INT-009 | No call other than `AddAgent` MUST create or destroy money or goods, and the identities of `spec-ledger.md` 9 MUST hold after every call (3.5.3). |
| FR-INT-010 | A call MUST accept a directive whose orders the agent cannot afford or whose discards it cannot cover, and the step MUST refuse them with the reason of the position, as it refuses any order (3.2.1). |
| FR-INT-011 | A call MUST be legal in the construction phase and its moves MUST go to the construction bucket there, and MUST be totalled in the tick the next step runs after it (3.1.1). |
| FR-INT-020 | `AddAgent` MUST return `ErrUnknownRegion`, `ErrOutsideNode`, `ErrUnknownRole`, `ErrUnknownStrategy`, `ErrOutOfBounds`, `ErrUnknownCommodity`, `ErrDuplicateID` or `ErrOverflow` for the cases of 4.1.1 and MUST create nothing when it returns one. |
| FR-INT-021 | `AddAgent` MUST create an agent of weight 1 with `External` true and the `Mortal` flag as asked, an account in the region, the stated money and goods by `initial` with the default lot key, beliefs by `spec-beliefs.md` 8.3, no job, an empty welfare record, `Hunger`, `Unfed` and `Privation` 0, `RoleSince` the current tick and the two creation draws, and MUST emit `AgentSpawned` with `External` true (4.1.2). |
| FR-INT-022 | An added agent MUST take the next agent ID and no other agent's ID, review bucket or key MUST change (4.1.3). |
| FR-INT-023 | An empty `Strategy` MUST mean `directed`; `default`, `directed` and every registered name MUST be accepted and any other name MUST be `ErrUnknownStrategy`; the world MUST hold a strategy directed if and only if the registered value's dynamic type is `Directed` (4.2.2). |
| FR-INT-024 | An external agent MUST be passed `Work` at P2.5 when free and `Trade` at P3.2 from the next step, MUST be refused every decision of the spawner unless mortal, and MUST NOT be reviewed or moved by the migration module (4.2.3, 4.6). |
| FR-INT-025 | `Direct` MUST return `ErrUnknownAgent`, `ErrAgentRemoved`, `ErrNotExternal` or `ErrNotDirected` for the agent, and for the directive `ErrUnknownRecipe`, `ErrNotOnRole`, `ErrUnknownCommodity`, `ErrNotTradable`, `ErrOutOfBounds` or `ErrDuplicateID` by the checks of 4.3.2, and MUST leave the mailbox as it was when it returns one. |
| FR-INT-026 | The mailbox MUST hold one directive: a second `Direct` before the step MUST replace the first, and a directive with nothing in it MUST empty the mailbox (4.3.1). |
| FR-INT-027 | A directive MUST apply to the next step only and MUST be dropped after it unless `Sticky`, and `Direct` MUST emit no event (4.3.1, 4.3.4). |
| FR-INT-028 | A directive MUST carry a job request only when the consumer sets one, so that a directive with intents and no job leaves `Work` to the fallback (4.3.3). |
| FR-INT-029 | `RemoveAgent` MUST return `ErrNotExternal` for a household of the configuration or the spawner, MUST apply the removal mechanics of `spec-agents.md` 3.6 with the cause `consumer` whether or not the agent is mortal, MUST NOT refuse for an obligation, MUST emit the events in the order of `spec-world.md` 6.11 item 8 and MUST NOT reuse the ID (4.4). |
| FR-INT-030 | A call that names the ID of a removed agent MUST return `ErrAgentRemoved`, which MUST match `ErrUnknownAgent`; the ID of one never issued MUST return `ErrUnknownAgent` and not `ErrAgentRemoved`, and the ID of a pseudo-agent MUST return `ErrUnknownAgent` from `Agent`, `Direct` and `RemoveAgent` and `ErrNotRelocatable` from `Relocate` and `RelocationRefusal` (7.1.1). |
| FR-INT-031 | `Relocate` MUST run the dry run with `skip` `on_journey`, return its first refusal and change nothing if there is one, and otherwise, atomically, drop the agent's journey, abandon its job with `region_changed`, drop its own-use flag, change its region by the guarded relocation with the cause `consumer`, reseed its beliefs with no draw and set `RoleSince` to the current tick (4.5.2). |
| FR-INT-032 | `Slots` MUST list exactly the slot values that differ from their defaults, in the canonical order of their owners, and a restored world MUST report the list the original did (5.3). |
| FR-INT-033 | `AgentState` MUST be a deep copy of plain data holding the groups of 4.7.2, MUST be a function of the state of the world alone and MUST NOT include `MaxExecutions`, a refusal reason or a stock after the next completions (4.7.2). |
| FR-INT-040 | `SetYieldShocks` MUST accept a multiplier in `[0, MaxYieldBP]` for a (region, recipe) of an ordinary region, MUST make the yield of a job started afterwards read it at P2.7 and MUST leave a job in progress its pending output, and MUST emit `YieldShocked` (5.1). |
| FR-INT-041 | `SetEdgeStatuses` MUST accept only configured edges and a factor in `[1, MaxEdgeFactorBP]`, MUST rebuild the graph and lag tables at the call, MUST NOT change the arrival tick of a shipment in flight and MUST emit `EdgeStatusChanged` in ascending `EdgeID` with the cause `consumer` (5.1; FR-REG-050). |
| FR-INT-042 | `SetTransitFactors` MUST accept a factor in `[0, MaxTransitFactorBP]` for a (from, to, mode) with from different from to, MUST affect new dispatches only and MUST emit `TransitFactorChanged` (5.1; FR-TRN-039). |
| FR-INT-043 | `SetFlowScales` MUST accept a scale in `[0, MaxFlowScale]` for a configured flow, MUST apply it from the next P1.7 and MUST emit `MoneyFlowScaled` (5.1; FR-MON-023). |
| FR-INT-044 | `SetBirthScales` and `SetDeathScales` MUST accept a scale in `[0, MaxBasisPoints]` for a region, MUST treat the region's configured value as the default and MUST emit `PopulationScaled` with which scale changed (5.1, 5.2.1; FR-POP-039). |
| FR-INT-045 | `SetCapacityScales` and `SetRegenScales` MUST accept a scale in `[0, MaxShockBP]` for a (region, resource); a capacity scale MUST recompute the effective capacity at the call and clamp a stock above it, counting the excess as lost; and each MUST emit `EndowmentAdjusted` with kind `shock` (5.1; FR-RES-029). |
| FR-INT-046 | `AdjustEndowments` MUST accept a signed delta of at most `MaxEndowment` in magnitude, MUST clamp the stock to `[0, capacity]`, MUST create no goods and MUST emit `EndowmentAdjusted` with kind `adjust` carrying the effective delta (5.1; FR-RES-030). |
| FR-INT-047 | `SetOutsideShocks` MUST accept a factor in `[1, MaxShockBP]` for a (node, commodity) that has a line, MUST apply it from the next step and MUST emit `OutsideShocked` (5.1; FR-EXT-027). |
| FR-INT-048 | A key set to its default MUST be cleared: the owner MUST hold the default and a checkpoint MUST be that of a world never shocked at the key (5.2.1). |
| FR-INT-049 | The world MUST keep, for each external agent of an `Informed` role with the information module on, the tick of its last view and the rumours given to it, MUST save them in a checkpoint, and `AgentState` MUST report them with the tick, or an empty list and the tick -1 when there is none (4.7.3, 13). |
| FR-INT-050 | A run with a recording observer and a capturing logger MUST give the same state, draws and snapshots as a run with neither, for a sequence of calls (observability-and-logging.md 8.1). |
| FR-INT-051 | Each call MUST emit exactly the events of 12.1 and no other, and none when no observer was supplied is constructed (12.1). |
| FR-INT-052 | A call MUST log nothing above `Debug` and MUST NOT log an error it returns (12.2). |
| FR-INT-053 | `Market` MUST return the published price and age, the volume and unmatched units, the best unmatched quotes with their presence flags and the window's length, mean, low and high of the last step, `ErrOutsideNode` for an outside node, and before the first step the seed price with age 0 (6.3). |
| FR-INT-054 | `Region` MUST return the region's row of the snapshot and no figure computed apart from it (6.5). |
| FR-INT-060 | Every query MUST take no draw, change no state, emit no event, return a deep copy of plain data and be legal in the construction phase; a query with an error result MUST return `ErrInStep` inside a step and `ErrWorldFailed` on a failed world (6.1). |
| FR-INT-061 | `Agents` MUST return the live agents whose account is in the region in ascending ID and no pseudo-agent (6.2). |
| FR-INT-062 | `Warnings` MUST return the warnings of the construction sorted by path then rule, and a restored world MUST return the same list (6.4). |
| FR-INT-070 | A world restored from a checkpoint taken after a sequence of calls MUST answer `Slots`, `Agent` and the mailbox as the original did and MUST continue exactly as an uninterrupted run with the same later calls (9.2, 13). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-INT-001 | A query of class 1 (6.1.3) MUST NOT allocate, checked with `testing.AllocsPerRun` (testing.md 8.2). |
| NFR-INT-002 | A full `Step` of the reference chain with one external agent MUST allocate no more than the baseline of the world's allocation test plus a constant for that agent's orders, and no call of this spec MUST add an allocation to a later step (12.5). |
| NFR-INT-003 | No call MUST start a goroutine, call `time.Now`, read the environment or keep package-level mutable state (architecture.md 7.1, 7.3, 12.2). |
| NFR-INT-004 | The work of a call MUST be bounded by the configuration: a setter by the number of keys it names, `AddAgent` by the tradable set and the stock, `Direct` by the commodities (5.2.6). |
| NFR-INT-005 | The argument checks of `Direct`, `AddAgent` and every setter MUST have fuzz tests that assert they never panic and that a refused call changes nothing (testing.md 7.4). |

## 19. Open questions

1. **A resolver for typed IDs.** A consumer resolves a string ID to a typed ID by
   position in `Config()` (3.4.2) and a region by `Graph.Lookup`. A small `Index`
   value (string to ID and back, for each kind) would remove a loop every consumer
   writes. OBS puts a names table in `WorldCreated` and the snapshot, which covers
   events. Next step: the first consumer example (architecture.md 1.3); add it
   only if the loop is written twice.
2. **Rumours kept for an external agent (4.7.3).** The one piece of state this spec
   adds. If INF, AGT or CHK refuse it, `AgentState.Rumours` is empty and says tick
   -1, and a player of an informed role is told nothing of other regions at the
   boundary (its strategy still sees them in its view). Next step: INF and CHK
   accept the edit, or a draw-free rumour read (a keyed error from `K(t, id)`) is
   specified by INF.
3. **The `directed` name and the presence of a job request (4.2.2, 4.3.3).**
   Both need an edit of `spec-agents.md`. Next step: AGT.
4. **A bundle of slots applied as one (3.3.4).** A scenario that sets a yield shock
   and closes a road wants both or neither. Next step: STY's first scenario that
   needs it; the shape would be one `Shocks` value with every slice of 5.3's
   `SlotState`, validated in full, applied in the order of 5.1.
5. **Turning migrants back.** A public cancel of every journey, or of those to a
   region (a quarantine), is not provided (4.5.3). Next step: a consumer's request;
   the module already has the internal cancel.
6. **An act-now operation (3.6).** A game with a real-time feel may want a trade
   that settles at the call. It would need a position of its own in the tick, its
   own admission and a place in the checkpoint's order of calls. Next step: a game
   that asks, with the rules of `spec-agents.md` 10.5.
7. **Goods into a region (3.6).** A relief shipment, a gift, a goods faucet. The
   supported path is an external agent that holds the goods, which is a named
   `initial`; a goods flow in MON's sense needs a ledger reason (N-021, N-107).
   Next step: FIS or a consumer that asks.
8. **A capacity multiplier of an outside line (N-116).** A partial embargo that is
   not a closed edge. Next step: when a consumer asks, as EXT open question 6 says.
9. **A household removed by the consumer (4.4.1).** `RemoveAgent` refuses a
   household of the configuration. A plague that kills named households is
   `SetDeathScales` or a consumer spawner; a call that removes a chosen household is
   not provided. Next step: when a consumer asks, with the cause `consumer` and the
   estate path of AGT 3.6 already in place.
10. **An external agent of weight above 1 (a faction, a guild).** Weight 1 only
    (`spec-scale.md` 4.7). Next step: when a consumer asks, with SCL.
11. **Who may see which agent (4.7.1).** `Agent(a)` answers for any live agent. A
    server that must keep a player from reading another player's state filters it
    itself. Next step: none; recorded so that a transport spec does not assume the
    library filters.
12. **Calls from a restored world before its first step (13.3).** A restored
    construction-phase world keeps its bucket (CHK's seam ask to WLD). Next step:
    WLD 5.5.3 and 12.1, CHK.

## Edits owed to other specs

One consolidated table: every file, section and change. The supervisor applies
them; the handoff is derived from this table. None alters a result that a test
could depend on (nothing has run).

| File | Section | Change |
|------|---------|--------|
| `spec-agents.md` | 10.3, FR-AGT-054 | `Directive.Job` has a presence: `Job *JobRequest` (recommended) or a `HasJob` flag, so that a directive with intents and no job leaves `Work` to the fallback (INT 4.3.3); the zero `JobRequest` is no longer read as recipe 0 |
| `spec-agents.md` | 6.6, 10.3, 10.6 | the name `directed` is built in beside `default`: `Directed{Fallback: DefaultStrategy{}}`; an empty strategy name of `AddAgent` means it; the world holds a strategy directed iff its dynamic type is `Directed`; the names of `AddAgent` and `Direct` are INT's (`NewAgent`, `Direct`); `ErrNotDirected` |
| `spec-agents.md` | 3.5.4 | an external agent added during a run takes its beliefs by `spec-beliefs.md` 8.3, which is 8.2 before the first step; "born as an initial agent" says so |
| `spec-agents.md` | 14.6, 15 | the per-agent query is `Agent(a)` returning `AgentState` (INT 4.7.2); the checkpoint list gains the rumours of the last view of an external informed agent (INT 13) |
| `spec-world.md` | 4.1, 4.3 | `directed` is a built-in name like `default` (a name given twice or `directed` is `ErrDuplicateID`) |
| `spec-world.md` | 8.2 | the row "Birth scale and death scale: by region, default 10 000" reads "default: the region's configured value, 10 000 with no row" (POP 6.6; INT 5.2.1); the public names of the calls are INT's (5.1) |
| `spec-world.md` | 3.7.1, 8.3, 13 (INT row) | the reads and calls of INT 6.2 and 4: `Agent`, `Agents`, `Market`, `Region`, `Warnings`, `Slots`; "Remove an agent (an external one the consumer asks for, or a mortal one)" reads "an external agent, mortal or not"; `Relocate` takes any agent |
| `spec-regions.md` | 7.3, FR-REG-050 | the exported `SetEdgeStatuses` has no `cause` parameter (INT 3.3.3, 5.1.2); the internal form keeps it for CAP, RSK and POL |
| `spec-regions.md` | 6.1.5, 6.6 | a forced move may name any agent that is not a pseudo-agent, not only an external one (INT 4.5.1); the journey's cancel is inside `Relocate`, not a public call |
| `spec-migration.md` | 8.4.1, 10.3, open question 16, FR-MIG-048 | the cancel is reached through `Relocate` (INT 4.5.2), with no public name of its own |
| `spec-money-supply.md` | 4.4.3, 10.4, FR-MON-023 | the public call is `SetFlowScales([]FlowScale)` of INT 5.1; `ErrUnknownFlow` and `ErrModuleOff` |
| `spec-external-sector.md` | 6.5, FR-EXT-027 | the public call is `SetOutsideShocks([]OutsideShock)`; the errors are `ErrModuleOff`, `ErrUnknownRegion`, `ErrUnknownLine`, `ErrUnknownCommodity`, `ErrOutOfBounds` |
| `spec-population.md` | 6.6, FR-POP-039 | the public calls are `SetBirthScales` and `SetDeathScales` of INT 5.1; the default of a scale is the configured value |
| `spec-resources.md` | 6.3, FR-RES-029, FR-RES-030 | the public calls are `SetCapacityScales`, `SetRegenScales` and `AdjustEndowments` |
| `spec-production.md` | 6.8 | the public call is `SetYieldShocks`; "its expiry is the integration spec's" is answered: there is no timer, the consumer clears (INT 5.2.2) |
| `spec-transport.md` | 10.3, FR-TRN-039 | the public call is `SetTransitFactors`; `from` differs from `to`; `TransitFactorChanged` |
| `spec-information.md` | 6.2, 7.1, 14 (checkpoint) | the world keeps, for an external agent of an informed role, the rumours of its last view and its tick (INT 4.7.3), saved |
| `spec-spawner.md` | 9.2 | the consumer's removal call is `RemoveAgent` (INT 4.4); `Mortal` is the flag of `NewAgent` |
| `spec-config.md` | 7.1, 7.4.1 | the sentinels of INT 11.2 are mapped at the root; `Warnings()` is also a method of `World` (INT 6.4) |
| `spec-world.md` | 8.3, 13 (INT row) | delete the row "Cancel a migration journey": the cancel is inside `Relocate` and there is no public cancel (INT 4.5.3); the list of 8.3 is add, remove, relocate, direct |
| `spec-world.md` | 6.11, 6.3 (P1.9) | the calls between ticks list gains `YieldShocked`, `TransitFactorChanged` and `PopulationScaled`; P1.9 reads the mailbox and takes a non-`Sticky` directive out of it into the tick's scratch (INT 4.2.3), so no position after P8.6 is needed |
| `spec-world.md` | 3.5.1, 3.6.1 | list `Endowment`, `InTransit`, `Away` and `RelocationRefusal` (and the derived-table readers `Calendar`, `ReferencePrice`, `Warnings`, `Graph`, `Travel`) beside `Tick`, `Config` and `ConfigHash` as readers with no error result that return their value (INT 6.1.2); not a change of the guard for methods with an error result |
| `spec-transport.md` | 5.9.2 | the consumer's removal of an external agent away on a trip is `RemoveAgent`, for any external agent, mortal or not (INT 4.4.1) |
| `spec-agents.md` | 10.1 | delete "or marked external in the configuration": no configuration field marks an agent external; an external agent is added by `AddAgent` (INT 4.1) |
| `spec-agents.md` | 10.6 | the consumer's removal call applies to any external agent, mortal or not; `Mortal` limits what the library decides (INT 4.4.1) |
| `spec-spawner.md` | 9.2 | as `spec-agents.md` 10.6: "the consumer's own removal call" removes an external agent whether or not it is mortal |
| `spec-config.md` | 3.3, 7.1.4 | `Role.Strategy` is `default`, `directed` or a registered name; `ErrUnknownMode` is a root sentinel, with the transport module alone (INT 5.1.3) |
| `spec-resources.md` | 14.1 | `EndowmentAdjusted` is emitted once per changed endowment, not once per call (INT 3.3.3; OBS 4.4.4) |
| `spec-external-sector.md` | 15.1, FR-EXT-037 | `OutsideShocked` is emitted once per changed (node, commodity), not once per call |
| `doc/rules/observability-and-logging.md` | 4.3 (rows `EndowmentAdjusted`, `OutsideShocked`) | "once per call" reads "once per changed key of a call" (with the fresh tag already owed) |
| `spec-migration.md` | 13.1, 14 | a journey cancelled by `Relocate` is counted in `Ended` of the next `MigrationReviewed` through the pending count (INT 4.5.2) |
| `spec-beliefs.md` | 8.3 | name the one helper that reseeds a newcomer, migrant, traveller or forced mover at arrival, and its home (INT 4.5.2, MIG 8.5, MER 12.2) |
| `spec-checkpoint.md` | 4.2 (tag 22), 13.2 | section 22 saves the rumours of the last view of an external agent of an informed role (INT 13); applied in the CHK draft, check only |
| `doc/rules/observability-and-logging.md` | 3.3.2 (key `cause`: widen to "cause of a removal, a weight change, a role change, a relocation, an edge status or a slot set by the consumer"), 4.3 | three events (`YieldShocked`, `TransitFactorChanged`, `PopulationScaled`: OBS's names); the value `consumer` of `cause`; a fresh tag |
| `doc/rules/architecture.md` | 3.3 | an entity ID in a call is a typed ID or, for a flow, its string (INT 3.4); a fresh tag (only if the rule's wording needs it) |
| `doc.go` (when code lands) | glossary | agent state, forced move, market state, slot state, query class, entry, key |
| `doc/notes.md` | the open notes of 17 | close N-022, N-079, N-091, N-116; update N-031, N-036, N-051, N-075, N-090, N-012, N-020, N-120, N-016, N-017, N-027, N-103; add the new notes of the handoff; the index; a fresh tag |
| `doc/specs/README.md` | index, area row 92 (INT), decisions, glossary | INT exists; the terms; a decisions row for "no act-now operation" and "a shock has no timer" |
| `doc/roadmap.md` | the INT row of tier 3 and the existing-specs table | `exists`; the open items of 19 |
| `doc/documentation_index.md` | rows | `spec-integration.md` |

## 20. Where this lives in the code

| Concern | Home |
|---------|------|
| `NewAgent`, `StockItem`, `AddAgent`, `Direct`, `RemoveAgent`, `Relocate` | root package, `integration.go` (the mechanism of each stays in its owner's file: `agent.go`, `regions.go`) |
| The slot setters and their entry types (`YieldShock`, `TransitFactor`, `FlowScale`, `RegionScale`, `EndowmentScale`, `EndowmentAdjustment`, `OutsideShock`), `SlotState`, `Slots` | root package, `integration.go`; each setter calls its owner's mechanism (`production.go`, `regions.go`, `transport.go`, `moneysupply.go`, `population.go`, `resources.go`, `external.go`) |
| `AgentState`, `Agent`, `Agents` | root package, `agent.go` |
| `MarketState`, `Market` | root package, `market.go` |
| `Region` and its row (an outside node returns its row, 6.5) | root package, `snapshot.go` (OBS's row) |
| `Warnings` | root package, `world.go` |
| The sentinels of 11.2 | root package, `errors.go` |
| The cause constant `consumer` (this spec's); `not_on_role` and `not_tradable` are PRO's and CLR's reasons, in the root package's `event.go` by their owners' rows | root package, `event.go`; `internal/logkeys` holds the key strings only |
| No new `internal/` package | the checks are root code over the owners' primitives (architecture.md 2.9) |
