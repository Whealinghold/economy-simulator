# Spec: World: the tick and its positions, the keyed random streams, options, construction, the seam between ticks and decisions at different rates

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-WLD-NNN` and
`NFR-WLD-NNN`, which tests cite (testing.md section 4). They are in section 21.

Layer: core. It is the spec every other spec's "phase N" assumes. It needs the
specs of tiers 1 and 2, reads each of them, and takes from each the position it
asked for in the tick (N-009); it changes no rule of theirs. Where two of them
claimed one position, or said different things, section 6.12 rules and names the
spec whose text wins, and section 22 lists what must still be edited elsewhere.

This spec adds one `Config` field, the name `World.Clearing` (4.2; its owner is `spec-clearing.md` 8, as `World.Spawner` is the spawner's), and no other field or default. The only numbers in it are
the constants of the SplitMix64 finaliser (published constants, rule constants,
7.1.2), the site constants of the keyed streams (rule constants, 7.1.4) and the example numbers of section 19, which are labelled. Nothing has run in
the library and no acceptance band exists (notes N-015, N-027); the scripts of
section 18 are stylised Docker Python models of the rules in section 8 to 11, not
claims about the library, and the one fact taken from the Go standard library
(the PCG-DXSM, `PCG.Seed` and the vectors of 7.1.5) was checked by running Go 1.25 and Go 1.27 in Docker.
The settings of the worked examples (19) are invented **examples**: the words
in them (`grain`, `loaf`, `baker`, `grower`, `alpha`) are flagged as such and are
data, never branched on (`AGENTS.md`, N-017).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

Every spec of the economy says "in phase 3" or "after the estates' asks" and
leaves the question of where that is to this one. The world is the one place that
decides, for a tick, what runs, in what order, who owns each position, what it
reads and writes, which random numbers it takes and which events it emits, and it
is the one place that says what a run is made of before the first tick and
between two ticks. It covers:

- the **world's calls**: `NewWorld`, `Step`, `Run`, the errors a step surfaces and
  the state of a world after one (section 3);
- the **options**, which of them change results, and what the hash covers
  (section 4);
- **construction**: the order of the ledger's accounts, the agent ID space, the
  construction phase, the warm start, the creation draws, the events (section 5);
- **the tick**: eight phases, each position in each phase with its owner, its
  reads, its writes, its draws and its events, and the order of the events of a
  step (section 6);
- **the keyed random streams**: the key of a stream and the site constants, every
  draw of a run in one table with its stream, the rule that the count of draws is a
  function of the configuration and of state (section 7);
- **between ticks**: the rule, the list of slots, the position of a checkpoint
  (section 8);
- **decisions at different rates**: every cadence of the written specs in one
  table and the rules that make a rate data (section 9);
- **fairness**: where an ID or an order gives an advantage and what neutralises it
  (section 10), **cost** (11), **checkpoint state** (12) and **what the world
  hands to the specs that come after it** (13).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The content of any step: what a recipe does, what an order is, how a price moves | the spec of the step (section 6 names it in each row) |
| The encoding of a checkpoint, the restore and its errors | `spec-checkpoint.md` (CHK) |
| The Snapshot's fields, the events' fields, metric and log names | `spec-observability.md` (OBS); the events of the rules are already fixed by `rules/observability-and-logging.md` 4.3 |
| The public names and the bounds of every call a consumer makes between ticks (`AddAgent`, the shock slots) | `spec-integration.md` (INT); this spec names the mechanism and its position only |
| The measures a consumer reads (indicators), the bands, a sweep | IND, STY, EXP |
| Parallel stepping | not in version 1 (architecture.md 12.2, 11) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | One authoritative tick of eight phases. Every position in it has a stable ID (`P3.2`, the second position of phase 3, is stable; new positions take a letter suffix) and one owner. A module that is off skips its positions and changes none of the others | 6 |
| 2 | The order of a phase is the order of its table. Where two specs claimed one slot, the ruling of 6.12 says which wins; the wordings and positions in the written specs that disagree are ruled one by one (twenty-eight entries) and the edit each needs is listed in 22 | 6.11 |
| 3 | A `Step` that returns an error leaves the world **failed**: every later call that would change or read live state returns `ErrWorldFailed`, and the consumer restores the checkpoint it took at an earlier boundary. No half-finished tick is ever observable, saved or resumed | 3.5 |
| 4 | A call made from inside a step (an observer, a strategy, a spawner, a clearing) returns `ErrInStep` and changes nothing | 3.6 |
| 5 | One seed, one `*rand.Rand` on one `rand.PCG` per world, and no stream position. At every draw site the PCG is reseeded from a key of (seed, site, tick, id) by SplitMix64 finalisations; no draw depends on how many another site took. Every bounded draw the world takes is one `Uint64` of its stream and the high word of its product with the bound: one source word, no rejection | 7.1, 7.2 |
| 6 | The count of draws of each stream is a function of the configuration and of the state of the world, and never of an observer, a logger or a module that is on and inactive; a change in one stream moves no other | 7.4, 7.6 |
| 7 | Every slot, shock and external call acts between ticks, and a between-ticks call that draws takes its draws at the call, in the order of the calls | 8 |
| 8 | A rate is data and is applied by one of four devices: a rotation by the bucket of the agent ID, a carried remainder over `TicksPerYear`, a window or a gate. A cadence that does not divide `TicksPerYear` is legal and is never rounded. No rate is derived from an aggregate | 9 |
| 9 | Ties go to a random key or a rotation, never to the lower ID, in new specs; the places in the written specs that still favour a low ID or an early role are registered as hazards with a named remedy | 10 |
| 10 | `Work` is always called for every free agent before any job starts (two passes), whatever modules are on | 6.4.1 |
| 11 | The privation count is set once, in phase 6, with the hunger count | 6.8, 6.12 |
| 12 | The world contributes to a checkpoint the next agent ID and whether it is still in its construction phase (the tick counter and the seed are written in the checkpoint's header, `spec-checkpoint.md` 5.4); no random state is saved; everything else of the world is derived | 12 |

## 3. The world's calls

### 3.1 The type and its lifetime

3.1.1. `World` is a struct from the root package that owns all simulation state
(ADR 0002, decision 6): the configuration it was built from, the ledger, the
agents, every module's state, the random source and the observers. It is **not
safe for concurrent use**, and its doc comment says so (architecture.md 12.1). Two
worlds share nothing: there is no package-level state (architecture.md 7.1), so a
consumer may step two worlds on two goroutines, and a sweep (EXP) may build many.

3.1.2. The zero `World` is not usable. `NewWorld` is its only constructor; a
restored world (CHK) is built by the checkpoint's restore function, not by
`NewWorld`.

3.1.3. The library starts no goroutine (architecture.md 12.2). Every call runs on
the caller's goroutine, and so do the observers (12.5).

### 3.2 NewWorld

```go
func NewWorld(cfg Config, opts ...Option) (*World, error)
```

3.2.1. `NewWorld` applies the options in the order given (section 4), validates
the configuration in full (`spec-config.md` 7), builds the world as section 5
says, and returns either a world at tick 0 or an error and no world. It never
returns a partly built world, and an error leaves nothing behind (the library has
no global to dirty).

3.2.2. The errors are the validation errors of `spec-config.md` 7.1 (joined, each
with a path and a rule), the errors of an option (4.3), `ErrAuditFailed` (5.1
step 9, with `WithAudit`; with `ErrWorldFailed` and `ErrInStep` it is a root sentinel whose first users are construction, P8.4 and the guards of P0.1), and the four name checks that need the options and so
live in `NewWorld`: a `Role.Strategy` name that is neither `default`, `directed` nor
registered (`ErrUnknownStrategy`, `spec-config.md` 7.1.4), a `World.Spawner` name that is not
registered (`ErrUnknownSpawner`, `spec-spawner.md` 3.1.3), a `World.Clearing`
name that is not registered or is empty (`ErrUnknownClearing`, 4.2) and, with the policy module
on, a `Policy` name of an actor that is neither built in nor registered
(`ErrUnknownPolicy`, `spec-policy.md` 4.5.1). The name
checks run **after `Validate` succeeds**, and each returns a `ValidationError` with
the path of the field and the rule, like any validation error (`spec-config.md`
7.1.1). Nothing else in a running simulation fails because of the configuration
(architecture.md 8.5).

### 3.3 Step

```go
func (w *World) Step() error
```

3.3.1. `Step` runs the tick `t = w.Tick()` through the eight phases of section 6
and, on success, advances the tick counter to `t + 1`. It takes no context: it is
short and CPU-bound (architecture.md 12.3).

3.3.2. A successful `Step` returns nil and leaves the world at a tick boundary
(8.4): no reservation open, every counterparty account empty, no job half
started, no shipment half dispatched.

3.3.3. The errors a step surfaces, all sentinels matched with `errors.Is`
(architecture.md 8.4), none logged by the library (architecture.md 8.3):

| Error | When | Position | Source |
|-------|------|----------|--------|
| `ErrInvalidTrade` | a `Clearing` returned a list of trades that breaks the checks of `spec-clearing.md` 8.2: a buyer or seller with no order in the market, a quantity of 0 or above the order's, a price outside the two limits, the same agent on both sides | P4.1 | `spec-clearing.md` 8.2, open question 5 |
| `ErrAuditFailed` | `WithAudit` is on and an identity of `spec-ledger.md` 9.1 or a module's check (12.4 there) failed. (A restore always audits once; its failure is `ErrCheckpointCorrupt` wrapping `ErrAuditFailed`, `spec-checkpoint.md` 6.2.2, and is not an error of `Step`) | P8.4 | `spec-ledger.md` 12.4 |
| `ErrWorldFailed` | an earlier step failed (3.5) | entry | this spec |
| `ErrInStep` | `Step` was called from inside a step (3.6) | entry | this spec |

The wrapping error names the tick and, for `ErrInvalidTrade`, the region and
commodity of the market; it is built with `%w` (architecture.md 8.2). **No other
condition is an error of `Step`.** A refused order, a refused shipment, a refused
offer, a refused decision, a start that finds no input are counted and reported as
events and refusals, as their specs say; an overflow of checked arithmetic is an
operation that cannot be applied and is handled where it occurs
(architecture.md 3.5); a violated internal invariant found with `WithAudit` off is
a programmer error and panics with a message that says so (architecture.md 8.1,
`spec-ledger.md` 12.2).

### 3.4 Run

```go
func (w *World) Run(ctx context.Context, ticks int64) (completed int64, err error)
```

3.4.1. `Run` is the multi-tick convenience of architecture.md 12.3. It emits
`RunStarted` (with `ticks`), then repeats: check `ctx.Err()`, then `Step`. It
checks the context **before every tick, including the first**, and never inside a
tick, so cancellation leaves the world at a tick boundary and usable. It emits
`RunEnded` before returning, with the ticks completed, whether it was cancelled and
whether a step failed (`Failed`; the two flags are never both true,
`spec-observability.md` 4.4.3.2).

3.4.2. Returns:

| Outcome | `completed` | `err` | `RunEnded` | World after |
|---------|-------------|-------|------------|-------------|
| All `ticks` steps ran | `ticks` | nil | completed `ticks`, not cancelled, `Failed` false | usable, at tick `t0 + ticks` |
| The context was done before tick `k` | `k` | `ctx.Err()` wrapped with `%w` and the number completed | completed `k`, cancelled, `Failed` false | usable, at tick `t0 + k` |
| Step `k + 1` failed | `k` | the step's error, wrapped | completed `k`, not cancelled, `Failed` true | failed (3.5) |
| `ticks` is 0 | 0 | nil | completed 0, not cancelled, `Failed` false | unchanged |
| `ticks` is negative | 0 | `ErrOutOfBounds` naming `ticks`, and **no** `RunStarted` | none | unchanged |
| The world is failed or in a step | 0 | `ErrWorldFailed` or `ErrInStep`, and no `RunStarted` | none | unchanged |

3.4.3. `Run` does not retry a failed step and does not log: the consumer decides
(architecture.md 8.3). A consumer that wants a snapshot every `n` ticks calls
`Run(ctx, n)` and `Snapshot()` in a loop; the world has no timer.

### 3.5 After an error

3.5.1. **Decision.** A `Step` that returns an error leaves the world **failed**.
Every later call that would change live state or read it and that has an error
result (`Step`, `Run`, every slot and external call of section 8, `Snapshot`,
`Audit`, `Checkpoint`, and the queries `Market`, `Region`, `Agents`, `Agent` and
`Slots` of `spec-integration.md` 6.2) returns `ErrWorldFailed` wrapping the error
that failed the world. The readers that have **no error result** still work, in two
groups (`spec-integration.md` 6.1.2):

- **Values the failure cannot have made inconsistent.** `Err() error` (the failure,
  nil on a healthy world), `Tick() int64` (the tick the failed step was running),
  the accessors of the immutable configuration (`ConfigHash`, `Config`) and the
  tables derived from the configuration and the edge statuses, which a step does
  not change (`Calendar`, `ReferencePrice`, `Warnings`, `Graph`, `Travel`, `Polity`, `TreasuryAgent`).
- **Readers of live state.** `Endowment`, `InTransit` and `Away` return the state
  that the failed step left. It is **not** a boundary state and has no guarantee of
  consistency: a consumer reads `Err()` first. The rationale of the first group
  does not cover them.

`RelocationRefusal` is listed apart: its `error` result is the refusal it reports
and not an error of the call, so it is not refused with `ErrWorldFailed`; on a
failed world it reads live state like the second group.

3.5.2. **Why.** A step that fails has already changed state in earlier positions:
jobs started, orders admitted with reservations open, shipments dispatched,
contracts formed, earlier markets settled. Nothing in version 1 can put that back:
a rollback needs a journal of every write of every tick, which costs time and
memory proportional to the writes of a tick on every tick, to protect against two
failures that are not the normal case. One is a consumer's faulty `Clearing`, which
the world already stops from corrupting the ledger (`spec-clearing.md` 8.2). The
other is a library defect that `WithAudit` found. Neither is a state to continue
from. The way back is the checkpoint the consumer took at an earlier boundary
(ADR 0002): it is exact, and it lets the consumer choose another extension.

3.5.3. **What was rejected.** *Complete the tick and then return the error*: the
step would return an error and also advance the tick, which breaks the convention
that an error means the call did not happen, and a consumer that retried would run
the tick twice. *Make a failed step a no-op by restoring an internal copy*: a copy
of the world on every tick is the cost that 3.5.2 refuses. *Leave the world as it
is and let the consumer inspect it*: a snapshot of a half-finished tick has open
reservations and a half-settled market, and a consumer that saved it (CHK refuses,
8.4) or analysed it would be reading a state no run produces.

3.5.4. **A panic.** The library does not recover a panicking observer, strategy,
spawner, clearing or policy (architecture.md 12.5): the panic propagates to the caller of
`Step`. The world marks itself failed with a deferred function that does not call
`recover`, so if the consumer recovers the panic and calls the world again it gets
`ErrWorldFailed`, not a world in an unknown state. The world is unusable until
restored. **A step that ends in a panic emits nothing**, `TickEnded` included: the
deferred function does not call observers during a panic, and the process owner has
the panic.

3.5.5. **A cancelled `Run`** is not a failure (3.4.2): the context is checked only
at a boundary.

3.5.6. **What an observer sees of a failed step.** **Every step that emitted
`TickStarted` emits `TickEnded`, a failed step included**, and `TickEnded` carries
`Failed`, true when the step returned an error, set at the point of failure. A step
that fails at P4.1 emits `TickEnded` with `Failed` there, after the events before the
failure and before it returns the error; a step that fails its audit (P8.4) emits it
at P8.5 with `Failed` true. A guard that fails at P0.1 emits nothing: no
`TickStarted` was emitted. So a bare `Step` is closed too, and an adapter's span
ends at `TickEnded` whether or not a `Run` surrounds it. `TickEnded` of a failed
step describes the end of the step, not a completed tick (architecture.md 12.5 and
observability-and-logging.md 4.4 read alike: it is emitted after the state change
it describes, which here is the failure, and the world is failed **before** the
event is emitted, so no observer can see a half-updated world). `Tick()`
is unchanged. `RunEnded.Failed` stays, for the run (13).

### 3.6 Calls from inside a step

3.6.1. Observers are called synchronously and **must not** call back into the
world (architecture.md 12.5); the same holds for `Strategy`, `Spawner` and
`Clearing`, which receive plain views and never a `*World` (`spec-agents.md`
6.1.1). The rule is enforced, because it is one flag: while a step runs, every
method of `World` that has an error result returns `ErrInStep` and changes nothing,
reads included, except `RelocationRefusal`, whose `error` is the refusal it
reports (3.5.1). The readers that have no error result return their value, in the
two groups of 3.5.1. `Err`, `Tick`, `Config`, `ConfigHash` and the derived tables
(`Calendar`, `ReferencePrice`, `Warnings`, `Graph`, `Travel`, `Polity`, `TreasuryAgent`) return a value that a
step does not change mid-call (`Tick` advances at P8.6, after the last event). The
readers of live state (`Endowment`, `InTransit`, `Away`) and `RelocationRefusal`
return the state as the running step has left it, which is not a boundary value and
which an observer sees complete up to the event it was given (events follow the
transition they describe). No conforming code calls them from inside a step
(architecture.md 12.5): the clause says what the value would be. An observer that
wants a snapshot keeps the events it was given, or the consumer asks
for the snapshot after `Step` returns.

3.6.2. The flag is set at the entry of `Step` and cleared at its exit by a deferred
function. It is not state and is not saved.

3.6.3. **A policy actor is not a call from inside a step.** A policy actor's decision at
P0.3a (`spec-policy.md` 7.1) writes the slots of kinds `flow`, `edge` and `transit` through the
owners' internal setters, with the cause `policy`; the public setters keep returning
`ErrInStep` (8.2.2). The actor's `Decide` receives a plain view and never a `*World`, as a
`Strategy` does (3.6.1).
### 3.7 Reads and the other calls

3.7.1. Between steps, on a healthy world, a consumer may call any read: `Tick`,
`Err`, `Config`, `ConfigHash`, `Snapshot` (OBS), `Audit` (`spec-ledger.md` 12.4), `Checkpoint` (CHK), `Graph`
(`spec-regions.md` 4.3), `Calendar`, `ReferencePrice` and `Warnings`, the transport
reads `InTransit`, `Travel` and `Away` (`spec-transport.md` 9.2), `Endowment`
(`spec-resources.md` 5.1), the queries `Agent`, `Agents`, `Market`, `Region` and
`Slots` (`spec-integration.md` 6.2) the guard query `RelocationRefusal`
(`spec-regions.md` 6.1.6) and, with the polity module on, `Polity` and `TreasuryAgent`
(`spec-polity.md` 4.2, 5.5.1). Each returns a deep copy of plain data (architecture.md
12.4), changes no state and draws no random number. The readers of the first slice are `Tick`, `Err`, `Config`, `ConfigHash`, `Snapshot` and `Audit`; `Warnings`, `Calendar`, `ReferencePrice`, `Graph`, `Travel`, `Polity`, `TreasuryAgent` and the queries of `spec-integration.md` 6.2 arrive with their owners, and FR-WLD-110 is tested for the readers that exist.

3.7.2. `Tick() int64` is the number of steps completed since tick 0: the tick the
next `Step` will run. It is the clock of the whole library (architecture.md 7.3). The names `Err`, `Tick`, `ConfigHash` and `Config` are confirmed by the code; `NewWorld`, `Step` and `Run` are fixed by the rules.

## 4. Options

```go
type Option func(*options) error
```

4.1. **The table.** `options` is unexported (architecture.md 6.2). Every option
validates its argument and returns an error rather than panic.

| Option | Changes results? | In the effective configuration and the hash? | Rules |
|--------|------------------|-----------------------------------------------|-------|
| `WithSeed(seed uint64)` | yes: it fixes every stream (7.1) | **no**: the checkpoint carries the seed in its header instead (`spec-config.md` 8.1; `spec-checkpoint.md` 5.4). It is in `WorldCreated` and in the `Info` line of construction | default 0; the last one given wins; refused on a restore (`ErrOptionContext`, 12.3) |
| `WithLogger(*slog.Logger)` | no | no | nil means the discard logger (architecture.md 6.2) |
| `WithObserver(Observer)` | no | no | additive: each is called for every event in the order the options were given; nil is ignored |
| `WithAudit()` | no, on a world that holds its invariants: it changes only what happens when one fails (an error where there would be a panic) | no (`spec-ledger.md` 12.4) | |
| `WithTargetSpread(bp int64)` | **yes**: it overrides the `TargetSpread` of every role (`spec-welfare-and-wages.md` 5.2) | **yes**: it is part of the effective configuration, so it is applied before the hash is taken and before validation (`spec-config.md` 8.1 lists it among the options that change results) | bound as the field's; the last one given wins; the creation draw is taken whatever the value (5.7) |
| `WithStrategy(name string, s Strategy)` | yes (what a role that names it does) | the **name** only, through `Role.Strategy`; the code behind a name is not hashed (`spec-agents.md` 6.6) | additive; the names `default` and `directed` (the library's own, `spec-agents.md` 6.6) and a name given twice are `ErrDuplicateID`; a nil `Strategy` is `ErrNilExtension` |
| `WithSpawner(name string, s Spawner)` | yes | the name only, through `World.Spawner` (`spec-spawner.md` 3.1.3) | registers under a name; `World.Spawner` selects it; same duplicate and nil rules |
| `WithClearing(name string, c Clearing)` | yes | the name only, through `World.Clearing` (4.2) | same shape; the name `average` is `AverageClearing` and is built in; an empty configured `World.Clearing` is `ErrUnknownClearing` in `NewWorld` (3.2.2) |
| `WithPolicy(name string, p Policy)` | yes (what an actor that names it decides) | the name only, through `Actor.Policy` (`spec-policy.md` 4.5): the code behind a name is not hashed | additive; the names `hold`, `schedule`, `rule` and `directed` (the library's own) and a name given twice are `ErrDuplicateID`; a nil `Policy` is `ErrNilExtension`; an actor's name that is neither built in nor registered is `ErrUnknownPolicy` in `NewWorld` (3.2.2) |
| `WithReseed(salt uint64)` | yes: it replaces the seed at a restore by the rule of `spec-checkpoint.md` 6.8.2 | **no**: it is not configuration and not a registered name; it is a call of the **sequence of calls** (8.1.1), reported in `WorldRestored` | **restore only**: on `NewWorld` it is `ErrOptionContext`; a salt of 0 is a reseed like any other (`spec-checkpoint.md` 6.8) |

4.1.1. **One rule for the four registrations.** `WithStrategy`, `WithSpawner`,
`WithClearing` and `WithPolicy` validate the name the same way, once, and the
rows above say only what is particular to each: a name satisfies the ID rule of
`spec-config.md` 3.3 (`validID`: a lowercase letter, then lowercase letters,
digits and `_`, at most `MaxIDLength` bytes), else `ErrInvalidID`, so an empty name given to an option is
`ErrInvalidID`; a name given twice, or a name the library builds in, is
`ErrDuplicateID`; a nil extension is `ErrNilExtension`. The three rules are
checked in that order and all fail the option, so `NewWorld` fails before any
state is built. An empty or unregistered name in the *configuration* is another
thing: it is the `ErrUnknown*` of the field, a `*ValidationError` with its path,
raised by `NewWorld` (3.2.2; `spec-config.md` 7.1.4).

4.2. **A gap this spec fills.** `Clearing` is one of the four extension points
(architecture.md 4.3), `spec-clearing.md` 8 defines its shape, and no written spec
says how a consumer's implementation is chosen or how the choice enters the hash.
It is decided here in the shape of the spawner's: a name in the configuration
(`World.Clearing`, whose default `average` comes from the defaults of
`spec-config.md` 5 like every field, so that a `Config` that leaves it empty fails:
`spec-config.md` 5.2, a zero is not a default), a registration option, and a check
in `NewWorld`. The owner of the field is `spec-clearing.md` 8, as `Spawner` is the
spawner's (`spec-spawner.md` 3.1.3). A consumer's `Clearing` draws from the
world's `Rand` as it likes (7.2.2); D7 is the default's count. Without the name a world with a custom clearing would hash like one
with the default and a checkpoint would restore into the wrong rule silently. The
edits are listed in 22.

4.3. **Errors of options.** `ErrInvalidID` (a name that is not an ID, empty included, 4.1.1), `ErrDuplicateID` (a name registered twice, or one of the library's own: `default` or `directed` of a strategy, `average` of a clearing, `hold`, `schedule`, `rule` or `directed` of a policy), `ErrNilExtension` (a nil `Strategy`, `Spawner`, `Clearing` or `Policy`),
`ErrOutOfBounds` (an argument outside its bound), `ErrOptionContext` (`WithSeed`
on a restore, `WithReseed` on `NewWorld`; `spec-checkpoint.md` 6.4). An option that
fails fails `NewWorld` (or the restore) with its error before any world state is
built.

4.4. **The rule for a new option.** A new option is added to this table in the
same change. If it can change a result it is in the effective configuration (a
datum), is a registered name (code) or, for `WithReseed` alone, is a call of the
sequence of calls (8.1.1); if it cannot change a result, the spec says so and a
non-interference test proves it (observability-and-logging.md 8.1). An option
never changes what is saved in a checkpoint, except `WithReseed`, the one
restore-only option, which is a call in the sequence of calls like `AddAgent` and
replaces the seed by the rule of `spec-checkpoint.md` 6.8.2.

4.5. **What is not a world option.** `WithFreeTrade` is an option of `Validate`
and of the steady-state solver (`spec-viability.md` 9), not of `NewWorld`.
`strategy.Options()` returns a slice of `Option` that registers the standard names
(`spec-merchants.md` 3.1); a consumer passes it with `strategy.Options()...`.

## 5. Construction

### 5.1 What `NewWorld` does, in order

The order is part of the golden output (architecture.md 7.4): nothing in it takes
a random number until step 8. The words of step 8 come from keyed streams (7.1.3):
each agent's two creation words are a function of the seed and of its ID, so they do
not depend on the order of the agents or of the steps before them.

| # | Step | Owner |
|---|------|-------|
| 1 | Apply the options in the order given; each validates its argument (4) | this spec |
| 2 | Deep-copy the `Config`, apply `WithTargetSpread`, sort every slice into canonical order (groups by region then role) | `spec-config.md` 3.4, 3.5 |
| 3 | `Validate`: every rule, every error, sorted; if it succeeds, the four name checks of 3.2.2. Collect the warnings | `spec-config.md` 7 |
| 4 | Compute the configuration hash (SHA-256 of the canonical encoding of the effective configuration) | `spec-config.md` 8.2 |
| 5 | Derive what is not state: the calendar and `TicksPerYear`, the region graph tables and the lag table, the module tables, the polity tables (the jurisdictions, the governed set and the treasury of each), the weights of each group | CAL, REG, INF, SCL, POL |
| 6 | Fix the seed, and build the one PCG and the one `Rand` (7.1.1); no word is taken | this spec |
| 7 | Open the ledger in its construction phase (5.5) and create the accounts in the order of 5.2, the Treasury accounts of the governed jurisdictions among them | LED, POL |
| 8 | Create the treasury pseudo-agents with their `StartMoney` and then the household agents, then the sub-steps 8a to 8c of 5.1.2: the titles (8a), the warm start (5.6), the creation draws (5.7), the initial facilities (8b), the settlement class of each region, the technology key (8c) | POL, AGT, LND, VIA, WEL, PRO, CAP, SET, TEC |
| 9 | With `WithAudit`: run the audit once, and fail with `ErrAuditFailed` if it fails (a warm start that breaks conservation is found here, not at tick 0) | LED 12.4 |
| 10 | Log the warnings and the `Info` line, then emit `WorldCreated` and the `AgentSpawned` events (5.8) | OBS |

5.1.1. A configuration that passes `Validate` and the name checks cannot fail in
steps 5 to 8: the validator is the only place a mistake is found (architecture.md
8.5). A failure there is a programmer error and panics with a message that says
so (architecture.md 8.1).

5.1.2. **Step 8, in order (ruling C26 of 6.12).** Step 8 is one step with sub-steps, and the
letters name a sub-step the way a letter names a position (6.1.1); nothing takes a random
number before the creation draws of item 4, and the draws of items 1 to 4 are the same
whatever the modules that are on (5.7.2):

1. the treasury pseudo-agents, one per governed jurisdiction in ascending `JurisdictionID`,
   each with the `StartMoney` of its jurisdiction by `initial` (`spec-polity.md` 5.4.1), then the
   household agents of the groups (5.3, 5.4); no draw;
2. **8a**, with `Modules.Tenure`: the titles of the configuration (`spec-land.md` 6.1), after
   the agents exist and before the warm start; no draw;
3. the warm start (5.6); no draw;
4. the creation draws (5.7): draw D1, two for each household agent;
5. **8b**, with `Modules.Infrastructure`: the initial facilities (`spec-infrastructure.md`
   3.4.2), after the warm start and the creation draws, so that a facility's endowment and store
   see the stocks the warm start made; no draw;
6. with `Modules.Settlements`: the settlement class of each region from the household count
   (`spec-settlements.md` 4.5), no draw;
7. **8c**, with `Modules.Technology` and some field with `DiscoveryBP` above 0: the technology
   key, **derived from the effective seed and taking no word** (draw D16 is retired, 7.3;
   `spec-technology.md` 4.4.1, owed under N-214); the step keeps its place after the creation
   draws, so that the order of construction is unchanged.

The audit of step 9 follows, and the events of step 10 come in the order of the sub-steps
(5.8.2).

### 5.2 The ledger's accounts

Account IDs are assigned in creation order and never reused (`spec-ledger.md`
3.2), so this order is the order of every iteration over accounts and is part of
the hash of a checkpoint. At construction the accounts are created in this order:

| Order | Accounts | Within the group | Source |
|-------|----------|------------------|--------|
| 1 | The Boundary accounts of the built-in reasons that are faucets or sinks: `initial` (money), then for each commodity in ascending commodity index the seven goods reasons `initial`, `recipe_input`, `recipe_output`, `disposal`, `consumption`, `spoilage`, `wear`; `1 + 7 * C` accounts for `C` commodities, whatever modules are on | the order of `spec-ledger.md` 3.4.2, which is not the registry order of 7.2 (the order of `BoundaryFlow` rows is reason-major, OBS 4.4.1.2) | LED 3.4.1 to 3.4.3, FR-LED-052 |
| 2 | The Boundary accounts of the reasons of the declared money flows | ascending flow ID (a flow's reason is its ID) | LED FR-LED-052; MON 4.5 |
| 3 | The Boundary accounts of the six `outside_*` reasons, with the module on: the money reasons first, then the goods reasons for each commodity that has a line side, in commodity order | | EXT 4.2.4 |
| 3a | The Boundary accounts of the other faucet and sink reasons of the modules that are on (`want_consumption`, `haul_use`, `transit_use`, `loss`); a module that is off has none | the order of the `Modules` pointers, and within a module as row 1 (money reasons first, then goods by commodity) | LED 3.4.2, 7.2.2 |
| 4 | The estate account of each region | ascending `RegionID` | REG 3.6.1 |
| 5 | The Outside account of each counterparty | ascending agent ID of the counterparty (5.3) | EXT 4.2.3 |
| 5a | The Treasury account of each governed jurisdiction, with the polity module on | ascending `JurisdictionID` | POL 5.2.1 |
| 6 | The Agent account of each household agent | ascending agent ID | AGT 3.5.1 |

After construction the ledger opens accounts as the run needs them: a Transit
account for a shipment (the open one with the lowest ID that no shipment in flight
uses, or a new one), an Escrow account for a contract (the same rule), an Agent
account for each agent created later. Each is the next account ID. The accounts of
orders 1 to 5a are never closed or relocated.

### 5.3 The agent ID space

| IDs | What | Order | Source |
|-----|------|-------|--------|
| `0 .. n-1` | the estate pseudo-agents, one per region **including** an outside node: the ID equals the `RegionID` | ascending `RegionID` | REG 3.6.1; EST 2.7.6 |
| `n .. n+m-1` | the counterparties of the outside, with the external module on | ascending (node `RegionID`, gate `RegionID`, side with buy before sell) | EXT 4.2.2 |
| `n+m .. n+m+g-1` | the treasury pseudo-agents, one per governed jurisdiction, with the polity module on | ascending `JurisdictionID` | POL 5.2.2 |
| `n+m+g ..` | the household agents of the groups | the groups in canonical order (region, then role), within a group ascending index; the first `H mod a` agents of a group take the extra household | SCL 3.2; CFG 3.4 |
| after construction | every later agent: spawner entrants and replacements, births, external agents | the next ID at creation, in creation order | AGT 3.5.3 |

5.3.1. An ID is never reused, so a removed agent's ID stays unused for the run. The
pseudo-agents are not agents or households (`spec-estates.md` 2.7.4) and are never
passed to a strategy.

5.3.2. **Consequence (N-117).** Adding a region, an outside node, a gate or a governed jurisdiction shifts
every household's ID, so every tie-break that reads an ID, every rotation phase and
every cadence offset (9) changes with it, as it does in any change of the
configuration. That is not a defect: the hash differs, so no checkpoint of the one
restores into the other. A consumer that needs identities that survive a change of
the configuration uses the string IDs of regions, roles and commodities, never an
agent ID.

### 5.4 The agents at construction

5.4.1. A group of `H` households with scale `S` and floor `K` gives `ceil(H / S)`
agents (raised to `min(K, H)`), with weights that differ by at most 1 and sum to
`H` (`spec-scale.md` 3.2). No random number is drawn.

5.4.2. Each household agent is born as `spec-agents.md` 3.5.1 says: an account in
its region, money `w * StartMoney` by `initial`, goods `w * quantity` of each
`StartStock` item by `initial` (**unless its group has an entry in the warm
start, which replaces both defaults: 5.6.1 item 3, so money and goods are created
once**), beliefs at the seed price with width
`BeliefWidthInit`, no job, an empty welfare record padded by `DefaultRatio`, hunger
and unfed 0, a death carry `offset(ID)` of `spec-population.md` 3.8.3 (a function
of the ID, no draw), its role's strategy. Its target ratio and its residues come
from the creation draws (5.7), after the warm start.

5.4.3. **External agents** are not created by construction. A consumer adds them
with INT's call, between ticks (8.3), and each takes the next ID and its creation
draws at the call.

### 5.5 The construction phase

5.5.1. The ledger's **construction phase** (`spec-ledger.md` 5.2.4) begins at the
first line of `NewWorld`, or at the restore of a checkpoint taken before the first
`Step` (5.5.3), and ends when the first `Step` begins (N-074). In it the
current tick is 0, every move is totalled in the construction bucket and not in
the flow totals of tick 0 (`spec-ledger.md` 7.4.1), and the world may state a lot
key (a negative `Born`, a worn durable) for the warm start. The public API offers no
way to state a key, so only the world's own warm start uses the privilege.

5.5.2. The consumer's calls between `NewWorld` (or a restore into the phase, 5.5.3)
and the first `Step` (INT's `AddAgent` and the slots of section 8) are inside the
phase: their moves go to the bucket and take the ordinary defaults of
`spec-ledger.md` 5.2.3. `Audit()` works inside it and checks the bucket
(`spec-ledger.md` 12.4). The bucket's `BoundaryFlow` is emitted at the entry of the
first step, a world whose first step this is and which holds a bucket (P0.2a), so
that those calls are in it.

5.5.3. A world restored from a checkpoint taken **after** its first `Step` is not in
construction: its boundary balances already hold the total and the bucket is not
saved. One taken **before** its first `Step` keeps its construction phase and its
bucket: the flag and the bucket are saved while the world is in construction
(`spec-checkpoint.md` 4.4, 6.7.2; `spec-ledger.md` 7.4.1, 11), a call between the
restore and the first `Step` is totalled in the bucket as it would have been on the
original, and the bucket's `BoundaryFlow` is emitted at that first step (P0.2a).

### 5.6 The warm start and the jobs created at construction (N-056 items 4 and 5)

5.6.1. The warm start (`spec-viability.md` 8) is applied after the agents exist and
**before** the creation draws, in this order, and takes **no** random number
(FR-VIA-040):

1. `WarmStart.Prices` seed the beliefs and the published price of each market
   (`spec-beliefs.md` 8.1, 8.4, `spec-clearing.md` 5.4);
2. `WarmStart.Wages` give each region its starting wage for the cost anchor, used
   only until the first working household-tick is observed
   (`spec-viability.md` 8.2.5);
3. `WarmStart.Groups`: a group with an entry is created with the entry's money
   and stock **in place of** `StartMoney` and `StartStock` (one `initial`
   creation, `spec-agents.md` 19, `spec-estates.md` 2.3, FR-VIA-032), the stock
   with stated lot keys staggered at fixed phases (`spec-viability.md` 8.4.3,
   `spec-ledger.md` 5.2.4); a group with no entry takes the defaults;
4. the jobs in progress, after item 3 (`spec-viability.md` 8.1 fixes no order;
   this one is fixed here because the oldest-first moves of item 4 would otherwise
   draw on staggered lots): the agents chosen by the selection rule of
   `spec-viability.md` 8.5.2 get `Remaining = d - o`, their inputs are granted by
   `initial` as separate lots at the default key and taken at once by
   `recipe_input` in one construction `Transform` (8.5.3; the `Transform` consumes oldest first, so the stagger vector is a property of the granted stock before the job's inputs are taken: the stock after the warm start is the stagger lots plus the granted input lots, fresh with `Born` 0, minus the oldest-first consumption of the input quantity, and its total equals the stagger total, granted equals consumed, FR-VIA-036), their tools are worn by
   `WearItems`, no endowment is drawn at construction (the virtual start precedes
   the run: a recommendation to VIA, 22), and the pending output is fixed from the virtual start with the
   effective factor of the season of that tick (shock at its default) and a
   starting residue of 0, and, with the storage module on, **pledged**: the load of the
   pending output is counted in the agent's load from the first tick, as it is for a job
   started by `Work` (`spec-storage.md` 4.3; the check `room_below_warm_job` is made here,
   13.1 there);
5. `WarmStart.Pools` create each region's pool by `initial` into its estate account
   (`spec-viability.md` 8.3.3).

5.6.2. **The residue of such a job (ruling C4 of 6.12).** The creation draw of
5.7 comes after, and it sets every residue of the agent, so the pending output of a
job created here was fixed with a residue of 0 and is not recomputed. The rule is
one sentence and has no other effect: a warm-start job's output is the floor of
`k * q * E / 10 000`, and the next job of the agent starts from the drawn residue. A new agent starts
with `Privation` and `Hunger` 0 (C2).

5.6.3. With no warm start there is no job and no stagger: every agent is free at
tick 0, so the agents of a role start their first jobs together and complete
together (STB-L22, WLD-A3). That is registered and left to the consumer's choice of
pack (17).

### 5.7 The creation draws

5.7.1. After the warm start, for every household agent in ascending agent ID, the
world takes **two** draws, in this order, whatever the configuration:

| # | Draw | Bound | Owner |
|---|------|-------|-------|
| 1 | the target ratio | uniform in `[lo, hi]` with `lo = TargetRatio * (10 000 - spread) / 10 000` and `hi = TargetRatio * (10 000 + spread) / 10 000`, as `lo + draw(hi - lo + 1)`; taken even when the spread is 0 | `spec-welfare-and-wages.md` 5.1, FR-WEL-024 |
| 2 | the residue | in `[0, 10 000)`, copied to every residue of the agent | `spec-production.md` 6.6.2, FR-PRO-026 |

5.7.2. The list belongs to this spec from here on (AGT 3.5.2 left it to WLD, and
adopted it here without change). **A spec that adds a creation draw appends it to
the table with a new row and says what the draw is for; it never makes the draw
conditional on a module or an option**, so that the words of an agent's creation
stream do not depend on which modules are on (WEL 5.1, NFR-POP-004, FR-INF-047; under
keyed streams a conditional word would move only the later words of that agent's own
stream, and the rule keeps even that out). There is no module-level word: the technology
key of 8c takes none (7.3). The pseudo-agents take none. A later agent (a replacement, a birth, an
external agent) takes the same two, in this order, at its creation
(`spec-agents.md` 3.5.3).

5.7.3. The construction therefore takes `2 * H_agents` words and nothing else, each
pair from the stream `(siteConstruction, 0, agent ID)` of its agent (7.1.4). No word has a
position in a sequence, so the first draw of the first tick is not "the next word of the
construction", and a pair of runs that differs in whether a discovery field exists is
aligned from the first tick (the pair of `spec-technology.md` 6.3.2 is no longer
misaligned: that text is stale under ADR 0008, see N-214).

### 5.8 Events and logs at construction

5.8.1. Logs, through the injected logger and bounded by the size of the
configuration (observability-and-logging.md 3.2.2): each warning once at `Warn`
with `rule` and `path`, then one `Info` line "world created" with `seed`,
`config_hash`, `commodities`, `recipes`, `roles`, `regions`, `agents`,
`households`, `modules`, `warm_start` and, for the modules that are on, their counts
(`prices`, `wages`, `groups`, `jobs`, `edges`, `components`, `nodes`,
`counterparties`, `modes`, `merchants`, and, of the later modules, `store_classes`,
`settlement_classes`, `facility_classes`, `facilities`, `jurisdictions`, `governed`,
`fiscal_flows`, `bodies`, `rights`, `controls`, `rows`, `fields`, `actors`, `levers` and
`underwriters`), all keys of the rules' table 3.3.2; `modules` lists the lower-case name of
each module that is on, `polity` and `policy` among them.

5.8.2. Events, with `Tick` 0, only when an observer was supplied
(observability-and-logging.md 4.5): `WorldCreated` (the seed, the whole hash,
whether a warm start was present, the names table of the IDs,
`spec-observability.md` 3.2.4), then one `AgentSpawned` for each household
agent in ascending agent ID, then, with the infrastructure module on, one `FacilityChanged`
(`Kind` `added`, `Cause` `initial`) for each initial facility in ascending facility ID
(8b), then, with the settlements module on, one `SettlementClassChanged` (`Cause`
`founded`) for each region in ascending `RegionID` (ruling C26 of 6.12). The estates, the
counterparties and the treasury pseudo-agents emit none; a treasury's money is the
`StartMoney` of its jurisdiction in the `Jurisdictions` rows of `WorldCreated`
(`spec-observability.md` 3.2.4). The
construction bucket is emitted as a `BoundaryFlow` with `Tick` -1, not here but at
the entry of the first step (P0.2a, `spec-observability.md` 4.4.2; N-072 answered).

## 6. The tick

This section is the one authoritative tick. It consolidates the phase list of
`spec-agents.md` 4 and every update of N-009 (waves 1 to 4) and fixes what they
left open. Each row has an **owner** (the spec that defines the content), what it
reads and writes, its draws (section 7 holds the table of all of them) and its
events.

### 6.1 The rules of the table

6.1.1. **Names.** A tick has an entry (6.2) and eight phases. A position is named
`Pp.q` (the second position of phase 3 is `P3.2`). The names are stable and are what
tests and other specs cite. A new position takes a letter after the position it
follows (`P3.2a`) and is added only by editing this section (6.1.10). Nothing is
renumbered.

6.1.2. **Order.** The positions of a phase run in the order of their table, each to
its end before the next begins. A loop "for each X" visits X in the order the row
states, and that order is always total (architecture.md 7.5, 7.6): regions in
ascending `RegionID` (`spec-regions.md` 3.3.2), agents in ascending agent ID,
commodities in ascending `CommodityID`, recipes and roles in ID order, flows by ID,
shipments and contracts by ID. No position iterates a map.

6.1.3. **Skips.** An agent that is **away** (a traveller on a self-carried trip,
`spec-transport.md` 5.5.2) or **on a journey** (a migrant, `spec-migration.md`
8.3) is passed to neither `Work` nor `Trade` and places no order; it is still
consumed, still reviewed by the spawner and still subject to removal. A trainer
that is **teaching** this tick (P2.2) is not passed to `Work`. A pseudo-agent has no
strategy and is never passed to one.

6.1.4. **The rotation.** The starts of a tick, and the calls of `Work`, run region
by region in ascending `RegionID`; within a region in ascending agent ID starting
at the position `t mod n_r` of the region's agent list, wrapping, where `n_r` is
the number of agents (not pseudo-agents) whose account is in the region, taken
once at the start of phase 2 (`spec-production.md` 5.5.1, N-036). The list
includes an agent that is away, which is skipped when its turn comes. It is the
one rotation of the library, and it decides **who goes first**; a scarce stock is
rationed by P2.6, not by the rotation.

6.1.5. **Modules.** A module that is off has none of its positions, takes no draw,
emits no event and leaves every result as it would be without it (the first
requirement of each module spec). A module that is on and has nothing to do (no
informed role, no walking line, no admitted offer) takes no draw either
(FR-INF-047, FR-EXT-021, FR-MER-033).

6.1.6. **Reservations.** Reservations open only at positions P3.2, P3.4, P3.4a and P3.5 (an
order is admitted and its cash or stock earmarked; P3.4a is the treasuries' standing orders) and are all settled or released
by the end of P4.1; none is open from P4.2 to the next tick's P3.2
(`spec-ledger.md` 6.5 states the boundary; the window is this spec's). Every position that needs none open (a relocation, a
removal, a weight change, relief, a money flow) is therefore before P3.2 or after
P4.1. A contract's money is escrow, not a reservation (`spec-ledger.md` 6.6), and so is a hire's wage (`spec-employment.md` 5.5). P3.3b opens no reservation and moves only money that is not reserved at that point (P3.3's escrow is not a reservation); P1.7a, P1.7b, P3.3c and P4.1a open none.

6.1.7. **Events** are emitted after the transition they describe is complete
(architecture.md 12.5, observability-and-logging.md 4.4), and only when an observer
was supplied (4.5 there).

6.1.8. **The one-tick delay.** Money earned in P4.1 can be spent no earlier than
P3.2 of the next tick, goods bought in P3.2 arrive in P4.1 and can be eaten in P6.1 of
the same tick, a price formed in P4.1 reaches a decision at P3.2 of the next tick,
and a belief revised in P5.4 is used at P3.2 of the next tick (the cash half is
`spec-clearing.md` 3.3.1, the goods half `spec-agents.md` 4.1; the rest follows from
the order of the phases). The phase order is therefore a delay of one tick in
every loop through prices, income and beliefs. It is registered once (WLD-A1) and
no loop's register entry needs to count it again.

6.1.9. **A step is a function** of the world's state, the configuration, the
registered extensions and the random source (section 7). Nothing in it reads the
clock, the environment, a map's order, the address of an object, an observer or a
logger.

6.1.10. **Claiming a position.** A spec that needs a position states, in its own
text and its handoff, the phase and the position it follows or precedes, its
reads, writes and draws. The change enters this table in the same change as the
spec that asks for it. If two specs ask for one slot, this section rules and the
loser's text is edited (6.12).

6.1.11. **Outside nodes.** "For each region" in P1.6, P3.3b, P3.4, P7.1, P7.2a and the rest means
every region that is not an outside node, except where a row says estates: an
outside node has an estate that holds nothing (`spec-external-sector.md` 3.1.1), no
agents, no pool and no market. P7.1 is not run for it and no spawner is called for
it (FR-SPN-002 is edited), so D9 takes no word for it.

6.1.12. **The position test.** The table of this section is the text the position test
parses (`testing.md` 6.8). The test compares, for each position, its identifier, its owner (the
first area code of the Owner cell; "this spec" reads as WLD) and its order. Each
position of the list in `tick.go` has a `where`: `line` (a call in the straight line of the
step) or `bracket` (run inside `beginStep` or `endStep`: P0.1, P0.2a, P0.3, P8.5 and P8.6).
P0.2 ("nothing is cleared at entry") is an empty position method, `p02()`, a `line` entry
with a one-line comment; it is called in the straight line, after `beginStep` has run P0.1,
P0.2a and P0.3, and takes no state, so the difference is unobservable.

6.1.13. **The calls that earlier specs assumed the world makes.** The owner of each
count calls it at its position. The world makes these calls, and no other position makes
them: the welfare feeds (`markWorking` and `addWear` with the facts of P2.1 and P2.7, once per
agent per tick; `markTraded` after each market clears at P4.1, for the buyer and the seller of
each settled `Trade`; `addReceipt` and `addInputSpend` from the settled trades; `record`, the push
and the reset of the scratch at P6.3, and the clear of the whole scratch at P8.6); the cost anchor
of each market, set on the book before its clearing at P4.1 from the published price left by the
previous P4.1 (P1.2); the flush of the `refusalTally` at P2.8a; and the advance of the bounded
`Warn` runs for every cell every tick.

### 6.2 Entry

| ID | What runs | Owner | Notes |
|----|-----------|-------|-------|
| P0.1 | Guards: `ErrInStep` if a step is running, `ErrWorldFailed` if the world failed | this spec | 3.5, 3.6 |
| P0.2 | Nothing is cleared at entry. The flow totals and the per-tick counters of every module are cleared at P8.6, so that a move made by a call between ticks is totalled in the tick the next step runs (8.1.3) | LED 7.4; each module | derived, not saved |
| P0.2a | At the entry of the **first** step of a world that is in its construction phase and holds a bucket (built by `NewWorld`, or restored from a checkpoint taken before its first step, 5.5.3), before `TickStarted`: the construction bucket as `BoundaryFlow` events with `Tick` -1, once, in the order of `spec-observability.md` 4.4.1.2 over the bucket's rows. A world that never steps emits none, and a world restored after its first step has no bucket | OBS 4.4.2; LED 7.4.1 | no draw; the bucket is read, not cleared; the events of calls between ticks were emitted at their calls and come before it (6.11) |
| P0.3 | `TickStarted` | OBS | the first event of the step, after the bucket's `BoundaryFlow` events (P0.2a) in the first step of a world in construction |
| P0.3a | The policy actors, with the policy module on and an actor due (`spec-policy.md` 7.2), in ascending `ActorID`: the view, `Decide`, the checks, and the application through the owners' internal setters of the slots of kinds `flow`, `edge` and `transit`, cause `policy` (8.2.2) | PLC 7.1 | reads the boundary state and the slots; writes the slot keys of the actor's levers; draws: a consumer's `Policy` takes its own words in call order (D17, 7.3), the built-ins take none; events: after `TickStarted`, per due actor, the owners' slot events (cause `policy`) and then `PolicyDecided`, before the first event of phase 1 (6.11 item 1) |

### 6.3 Phase 1: start

Nothing in phase 1 reads an order or a price formed this tick. Everything in it is
the consequence of what the previous tick left or of what a slot set between ticks.

| ID | What runs, in this order | Owner | Reads / writes | Draws | Events |
|----|--------------------------|-------|----------------|-------|--------|
| P1.1 | The moment: `Calendar.At(t)`, once. It is derived and adds nothing to a checkpoint | CAL 6.5; N-031 | reads `t`; scratch | none | none |
| P1.2 | The cost anchors: per (region, commodity) the unit cost of what a role there can make, from the previous tick's observed wage per region and the published price left by P4.1 of the previous tick (the seed price at tick 0, not the price window) | PRO 9.3, FR-PRO-037, FR-PRO-047, 048 | reads wage record, published prices, yields (shock, abundance); writes the anchor table, which is derived | none | none |
| P1.3 | The arrivals of transport, in ascending shipment ID: a shipment whose loss share (`LossBP`, recorded at its dispatch by draw D14) is above 0 first loses that share (step 0 of `spec-risk.md` 5.1, before the traveller's region change and the delivery); then a traveller's account changes region by `Relocate` (cause `transport`), after the guard query run **ignoring only the shipment that arrives and the contract formed on it** (by instance, ruling C16; every other shipment, contract or obligation of the agent still refuses), and its beliefs are reseeded (no draw); a hired cargo is delivered to its consignee, bounded by the room left of a limited account, the part that does not fit diverted to the estate of the destination or disposed (STO 5.3.4); the Transit account joins the free list | TRN 5.7, 5.8; MER 12.2 | reads shipments with `Arrives = t`; writes lots, the account's region, beliefs, the away flag | none | per shipment: `AgentRelocated` (a traveller), then `ShipmentArrived` (with the loss, `Lost` and `AccountLost`) |
| P1.3a | The settlement of the route covers of the shipments that arrived in P1.3, ascending cover ID, one pass per underwriter: the loss value, the due, the shield and the allocation; what the underwriter cannot pay becomes a payout claim | INS 8 | reads the lost units of the arrival (scratch of P1.3), covers, balances, the published prices; writes balances, covers, claims | none | per cover `CoverEnded` in ascending cover ID, then `CoverSettled` per region |
| P1.4 | The settlement of delivery contracts whose shipment arrived in P1.3, ascending contract ID: `escrow_out` for the units that reached, `escrow_refund` for the rest | MER 9.7 | contracts, Escrow accounts; writes balances; the contract ends | none | `ContractSettled` |
| P1.5 | The arrivals of migration journeys, ascending agent ID: the guard is run again with `skip = on_journey`, the account changes region by `Relocate` (cause `migration`), beliefs are reseeded (no draw), `RoleSince` is set; a refusal holds the arrival | MIG 8.4 | journeys; writes region, beliefs, `RoleSince` | none | `AgentRelocated` (cause `migration`) |
| P1.6 | Relief, per region ascending: the release is `pool * PoolRelease / 10 000`, shared by need to the agents whose ratio is below `ReliefBelow` | EST 2.6 | pool, welfare records up to the previous tick, effective upkeep; writes balances (`relief`) | none | `ReliefPaid` per region that paid |
| P1.7 | The money flows: grants in ascending flow ID, then levies in ascending flow ID; each over regions ascending, then accounts ascending | MON 5.1, 5.2 | flow state, balances, weights; writes balances, carries, totals, `Ended` | none | after the step, per flow in ascending flow ID: `MoneyFlowLimited`, then `MoneyFlowEnded` of the same flow |
| P1.7a | The settlement of the claims, in ascending claim ID: interest and principal due, the pot and its pro rata share, payments, arrears, ends, defaults and write-offs; the payout claims of the covers are settled in the same pass | CRD 8 | reads claims, balances (after the tick's money flows), the shield; writes balances, claims, the totals of the module | none (ties by the stateless key, remainders carried) | `LoanEnded` per ended claim in ascending claim ID, then `CreditSettled` per region |
| P1.7b | The fiscal flows: the taxes on households and pools by flow ID, the remittances by flow ID, then the payouts per authority; a tax a payer cannot meet is forgiven for the shortfall (FIS 5.3) | FIS 5.3, 8.4 | reads the fiscal flows, balances after P1.7a, the treasuries' accounts; writes balances, carries, totals | none (a cut is pro rata by the keyed tie of 10.1.1) | `FiscalFlowApplied` per flow and region |
| P1.8 | The outside's walk: for each line with `WalkStepBP` above 0, ascending (node `RegionID`, `CommodityID`), the drift moves | EXT 5.1, 6.4 | line state; writes the drift | one per walking line, in that order | none |
| P1.9 | The directives: the mailbox of each `Directed` external agent is read, and a non-`Sticky` directive is taken out of it into the tick's scratch (INT 4.2.3) | AGT 10; INT 4.2.3 | mailbox; writes the directive of the tick | none | none |
| P1.9a | The membership round, with the institutions module on: for each body in ascending `BodyID`, the lapse pass over every household of the body and then the admission per region | ORG 4 | reads households, the bodies' tables, the places; writes membership, the waiting counts | none | none (counted at P7.2b) |

Starting money is not paid in phase 1: it moves at the creation of an agent
(phase 7, or an external agent's call between ticks). Ruling C1 of 6.12.

### 6.4 Phase 2: production

| ID | What runs, in this order | Owner | Reads / writes | Draws | Events |
|----|--------------------------|-------|----------------|-------|--------|
| P2.1 | Continuations and completions, in the rotation order (6.1.4): `Remaining` falls by 1; at 0 the job completes: one `Transform` creates the pending outputs that fit from `recipe_output`, and the ledger's create-and-dispose operation disposes of the part above `MaxQuantity` and of each discard output, the credit bounded by the room left (STO 5.3.4); a hired job's wage released to the worker (EMP 6.2); a build or repair completes (CAP 4.3); the executions of a job of a technology field are added to the counter of its cell (scratch, TEC 3.4) | PRO 5.2, 5.3 | jobs; writes lots, clears the job | none | `RecipeExecuted` |
| P2.2 | Teaching: each trainer's accumulator advances by the sum of its trainees' loads, trainers in ascending ID; a trainer whose accumulator reaches 10 000 subtracts it and is teaching this tick | SKL 4.2 (position: C5) | trainees' `TeachingLoad`; writes the accumulator, the flag | none | none |
| P2.3 | The snapshot of every endowment's stock and abundance, ascending (`RegionID`, `ResourceID`), fixed for the tick | RES 5.2 | stocks; scratch | none | none |
| P2.4 | The own-use flag step for every eligible agent, ascending agent ID: a pure function of the household's own stock, money, `Hunger` and own-market price | SUB 5.4 | agent state, the snapshot; writes `OwnUse` and its runs | none | none (counted in 2.8) |
| P2.5 | `Work` for every free agent (no job in progress, and none continued or completed in P2.1; not away, not on a journey, not teaching), in the rotation order: the strategy returns a `JobRequest`; the world derives `k0`, the executions it could start ignoring draws, and refuses what it must: `k0` includes the facility limits (CAP 4.2.2) and the room bound `k_room` (STO 5.3.1), so that P2.6 shares a request the room can hold | AGT 6.2, 7.1; PRO 4.2; RES 5.2 | the view built from the state after 2.1 and the snapshot; writes the requests (scratch) | the default takes none; a consumer's own are counted in call order | none (a refusal is a `Debug` line) |
| P2.5a | The labour round, with the employment module on: the admission of the offers and vacancies posted at P2.5, their order, the matching and the formation of the hires; a hired household's job starts at P2.7 of the tick | EMP 5.1.1 | reads the `Offer` and `Hire` requests of P2.5, stock, tools, cash, the P2.3 snapshot; writes hires, Escrow balances, the hired requests, the withdrawn own requests | none (ties by the keyed `K(t, id)`, 10.1.1) | `HireFormed` in formation order |
| P2.6 | The ration, for each endowment some request draws on, ascending (`RegionID`, `ResourceID`): requests collected in agent ID order, shared proportionally, the leftovers by largest fractional part, then the stateless hash key of (tick, agent), then agent ID | RES 5.3 | requests, snapshot and, for the tiers of tenure (own, commons, let: LND 4.3, each by RES 5.3.2), the principals' balances for the clip of the rent (no reservation is open in phase 2, 6.1.6); writes the grants (scratch) | none | none |
| P2.7 | The starts, in the rotation order: `k = min(k0, grants, live inputs, bounded again by the facility limits 1 and 3 against the running count of the tick (CAP 4.2.2), by the room at that moment (STO 5.3.2) and by the largest `k` whose rent the principal can pay (LND 5.3, 5.4)`; one `Transform` (`recipe_input`), tools worn, endowment draws taken whole or not at all, the job created with its pending output (yield read now); a job of duration 1 then completes as in 2.1; the use wear of a facility a recipe `Needs` is applied once `k` is final (CAP 5.1.3) | PRO 5.2, 5.3, 6.4; RES 5.4 | requests, grants, ledger; writes jobs, lots, endowment stock | none | `RecipeExecuted` for a one-tick job, in its start tick |
| P2.8 | The own-use counts, ascending `RegionID`, then role | SUB 14.1 | the counts of P2.1 (units output), P2.4 (entered, left), P2.5 (blocked, retried) and P2.7 (started, and blocked: a household in the mode whose own-use requests the ration cut entirely and which started nothing, SUB 5.5 item 6) | none | `OwnUseChanged` |
| P2.8a | The production halt's count, ascending `RegionID`, then role: per (region, role) the free households passed to `Work` at P2.5, those declined for the halt, those that rested, those with no recipe worth running, and per recipe the households for which it was skipped; only where some household was declined, rested or had no recipe; then the job requests of the tick that were refused with a count of 0 and the households whose work report named a blocked recipe (`spec-agents.md` 7.1.4), per (region, recipe), ascending `RegionID` then `RecipeID`, only where some was refused | OBS 4.4.6, 4.4.6a; AGT 7.1.5, 14.3; PRO 14.1 | the halt report that `Work` returned at P2.5 and the `refusalTally` of the starts (scratch) | none | `ProductionHalted`, after `OwnUseChanged`, then `RecipeRefused` (the refused job requests of the tick per region and recipe, ascending `RegionID` then `RecipeID`; `spec-observability.md` 4.4.6a) |
| P2.8b | The labour-round aggregate, with the employment module on, ascending `RegionID`, where any offer, vacancy or hire was counted | EMP 6.3.3 | the counters of P2.5a and of the wage releases of P2.1 and P2.7 (scratch) | none | `LabourRound` per region, after `ProductionHalted` |

6.4.1. **`Work` is always two-pass (ruling C9).** All the calls of `Work` of P2.5
come before any start of P2.7, with the module of resources on or off. Interleaving
a call and a start would let one agent's start change what the next agent's view
shows (a trainee takes its inputs from its trainer's account, an own-use job and a
role job may draw on one endowment), and would make the stream a consumer's
strategy sees depend on a module switch.

### 6.5 Phase 3: trade decision

| ID | What runs, in this order | Owner | Reads / writes | Draws | Events |
|----|--------------------------|-------|----------------|-------|--------|
| P3.1 | `Inbound` and `Outbound` are fixed for every agent: derived from the shipments and contracts, so that they do not depend on the order of agents | TRN 5.3.3, 9.1; MER 4.1 | shipments, contracts; scratch | none | none |
| P3.1a | The common strikes of the routes, with `Modules.Risk`: for each route row with `Common` and a probability above 0, ascending (`From`, `To`, `Mode`), one word; the outcome is kept for the tick | RSK 4.2 | the route rows and their risk scales; writes the strike outcome (scratch) | D13: one per such row in that order | none (a risk scale is set at a call: `RouteRiskScaled`) |
| P3.2 | For each agent in ascending ID (6.1.3): (a) if its role is `Informed` and the information module is on, build the rumours of its view; (b) `Trade` appends to the plan; (c) execute the plan: discards, then shipments in plan order, then orders in plan order, each admitted with its reservation or rejected; the toll of a shipment is charged at its dispatch (step 3a of `spec-transport.md` 5.5.1, FIS 10) | AGT 4, 6.5; INF 5.5.2, 7.1; TRN 5.3.2; CLR 3 | agent state, market reports, rumours, `Inbound`; writes the ledger (discards, Transit), reservations, the order list of each market, shipments | (a) one per candidate, offered or not, in ascending source `RegionID`, then `CommodityID`, when `RumourErrorBP` is above 0; (b) the default takes one per intent it considers, in canonical commodity order; a consumer's own in call order; (c) with `Modules.Risk`, D14 at the record of each dispatched shipment, one per unit row on its path with a loss probability above 0, inside the agent's turn after its own words (RSK 4.1) | per agent: `ShipmentDispatched` or `ShipmentRejected` in plan order, then `OrderPlaced` or `OrderRejected` in plan order |
| P3.3 | The contract round, with `Modules.Merchants` present: admission of every offer and take of the plans in one pass in agent ID and plan order; then the keys; then the lanes in ascending (destination `RegionID`, `CommodityID`): matching, formation (checks, dispatch, escrow) | MER 9.3 to 9.5 | the offers and takes, stock, cash; writes shipments, Escrow, contracts | one per admitted offer in admission order, then one per admitted take in admission order, all before the first lane; then, with `Modules.Risk`, D14 at the record of each shipment a formation dispatched (RSK 4.1) | per formed contract in formation order: `ShipmentDispatched`, then `ContractFormed`; the `ShipmentRejected` of a formation that fails at dispatch; then `ContractRound` per destination region ascending |
| P3.3a | `MerchantsPlanned` per region ascending, whenever an agent with a merchant block was given a view there, **also with the module absent** | MER 15.1.1 | the plan notes of P3.2 | none | `MerchantsPlanned` |
| P3.3b | The credit round, with the credit module on: the admission of the requests and offers of the plans in one pass, the order, the matching and the formation of the claims; it uses only money that is available (not reserved) after P3.2 | CRD 5, 6 | the `Borrow` and `Lend` requests of P3.2, balances; writes balances, claims | none | `LoanFormed` in formation order, then `CreditRound` per region |
| P3.3c | The cover round, with the insurance module on: the admission, the order, the matching and the formation of the store and voyage covers | INS 5, 7 | the requests and offers of the plans, balances less reservations, the lost-unit estimates, the shipments of the tick, the published prices; writes balances, covers | none | `CoverFormed` in formation order, then `CoverRound` per region, after `CreditRound` |
| P3.4 | The estates' asks, per region ascending: one ask for each lot held, named, at the lot's current price | EST 2.7 | estate lots and prices; writes reservations, order lists | none | `OrderPlaced`, and `OrderRejected` for an ask the clearing refuses |
| P3.4a | The treasuries' standing orders, with the fiscal module on, for each authority in ascending jurisdiction order: the purchases and the releases of its goods, each order admitted with its reservation or rejected | FIS 9 | the treasuries' balances and stocks, the published prices; writes reservations, order lists | none (a treasury's order has no outcome and takes no word, ruling C21) | `OrderPlaced` or `OrderRejected`, then `TreasuryOrdered` per authority |
| P3.5 | The counterparties of the outside, ascending agent ID: for each line of its node in commodity order whose side is present: the wanted units, whether the edge is open, the quote, the offered units, the top-up from the boundary, the order | EXT 5.2 | line state, moment, edge status; writes the Outside account, the order lists, carries | none | `OrderPlaced`, `OrderRejected` |
| P3.6 | The aggregate events of the phase: `RumoursServed` per reader region ascending, then `WantShortfall` per (region, role, want) ascending | INF 10.1; DEM 11 | the counts of P3.2 | none | those events |
| P3.6a | The room aggregate, with the storage module on, per region ascending then role, where any count is above zero | STO 14.1 | the counts of P1.3, P2.1, P2.5, P2.7, P3.2 and P7.1 (scratch) | none | `RoomLimited` |
| P3.6b | The saving aggregate, with a role that has the saving block: per region ascending then role, where a triggering count is above zero | SAV 9, 14 | the counts of the saving steps inside P2.5, P3.2(b) and P8.6 (scratch) | none | `SavingPlanned` |
| P3.6c | The toll aggregate, with the fiscal module on: the tolls charged at dispatch in P3.2(c) and P3.3, per flow and region, in the order of the flows | FIS 10.1 | the toll counts of the tick (scratch) | none | `FiscalFlowApplied` (kind `toll`) |

6.5.1. In a market's input order the orders come in this sequence: the agents'
orders in ascending agent ID and plan order, the estates' asks, the treasuries'
orders, the counterparties' orders. The tie-break keys of P4.1 are drawn in that order, so the order itself
gives no agent a claim (10).

### 6.6 Phase 4: clearing

| ID | What runs, in this order | Owner | Reads / writes | Draws | Events |
|----|--------------------------|-------|----------------|-------|--------|
| P4.1 | For each market in ascending `RegionID`, then `CommodityID` (there is a market for every region that is not an outside node and every commodity): `Clear(bids, asks, rng)`; validate the whole list; settle the trades in order; release the unmatched reservations; publish the price and its age | CLR 4 to 8, 5.4 | the admitted orders of the market in input order; writes balances, lots, the published price and age | one per order, bids first then asks, each in input order, before sorting | `TradeExecuted` in trade order, then `MarketCleared`, also with no trade |
| P4.1a | The duties on the trades of each market just cleared, with the fiscal module on, in the order of the flows, after the clearing of all markets and before the sweep | FIS 6 | the trades of P4.1, the fiscal flows; writes balances, carries, totals; no reservation is open | none | `FiscalFlowApplied` (kind `duty`) per flow and region, after the `MarketCleared` events |
| P4.2 | The sweep, for each counterparty in ascending agent ID: its account is emptied to the boundary, money then goods in commodity order; the cumulative units and list value and the ended flags are updated | EXT 5.3 | the Outside accounts; writes boundary balances, side state | none | `OutsideTraded` per (node, gate, commodity, side) with any unit wanted or cut, then `OutsideEnded` |

6.6.1. An invalid list at P4.1 settles none of that market's trades, releases its
reservations, and fails the step with `ErrInvalidTrade` at once (3.3.3); the
markets after it are not cleared. The step emits `TickEnded` with `Failed` and
returns the error; the world is failed (3.5).

### 6.7 Phase 5: outcomes

| ID | What runs, in this order | Owner | Reads / writes | Draws | Events |
|----|--------------------------|-------|----------------|-------|--------|
| P5.1 | The price window of every market **in which units traded this tick** is pushed with the market's published price, ascending (`RegionID`, `CommodityID`); a market that did not trade is not pushed | CLR FR-CLR-032; BEL 7.4 | published price; writes the window | none | none |
| P5.2 | The information record: one entry for every source (region, commodity) with a market, ascending, after the windows | INF 4.3, 7.1 | published price and age; writes the ring | none | none |
| P5.3 | The estate listing step: each estate lot that had an ask this tick and was not wholly sold has its price reduced by `EstateMarkdown`, truncated, never below the resolution floor | EST 2.7.3 (position: C6) | estate lots; writes prices | none | none |
| P5.4 | The outcomes, for each admitted order **of a household agent** in ascending `RegionID`, `CommodityID` and admission order: the world takes one `uint64`, builds the outcome with the market report, calls `Revise`, and applies the returned belief inside its bounds | CLR 7.3; BEL 5.1; AGT 6.2 | the outcome, the report, the stock after the trades less this tick's upkeep reserve, `Target`, `UnitCost`; writes the belief | one per admitted order (the world's `Draw`), then a consumer's own in call order; **an estate's ask, a treasury's order and a counterparty's order has no outcome and takes no word** (ruling C21) | `BeliefsUpdated` after the last outcome of each market that had one |

### 6.8 Phase 6: consumption

| ID | What runs, in this order | Owner | Reads / writes | Draws | Events |
|----|--------------------------|-------|----------------|-------|--------|
| P6.1 | Upkeep, for each agent in ascending ID and each upkeep commodity in canonical order: `consumed = min(held, U)` to `consumption`, oldest lots first; a trainee's upkeep is taken from its trainer's account at the trainee's own position; then `Fed`, `Unfed`, `Hunger` and **`Privation`** are set (ruling C2) | AGT 5.3, 5.4, 5.4.1; SKL 4.2 | holdings; writes lots, `Unfed`, `Hunger`, `Privation` | none | none |
| P6.2 | Wants, with the demand module on, for each agent ascending and each want commodity: `consumed = min(Want, max(0, held - U))` to `want_consumption` | DEM 4.7 | holdings, the planned units of P3.2; writes lots | none | none |
| P6.3 | The welfare record of the tick is pushed for each agent ascending: income (spending on `Input` and tool wear netted), the effective upkeep as cost, the working household-ticks, the own-use ticks and units, the in-kind receipts | WEL 2, 4.3; SUB 7 | the tick's flows and a per-agent scratch of the tick (working ticks: set at P2.1 for a job continued or completed and at P2.7 for a start, because the job is cleared at completion and a one-tick job is never seen at P6.3; own-use ticks at P2.7); writes the ring | none | none |
| P6.4 | The shortfall, ascending (region, role), only where a household went unfed | AGT 14.1 | counts of P6.1 | none | `UpkeepShortfall` |

6.8.1. **A flow after P6.3 amends the record of its tick.** A flow of the insurance result `N_t`
that moves after P6.3 (at P7.1, the principal that a removal's waterfall pays; at P8.1b, a store
payout) is added to the record of its tick by amending the last record, which adds no state; an
agent with no record of tick `t` (one created at P7.1) is not amended (`spec-insurance.md` 11.1.3,
`spec-welfare-and-wages.md` 2.1).

### 6.9 Phase 7: entry and exit

No reservation is open (6.1.6). Removal, weight change and role change
happen only here and between ticks. A **relocation** also happens at the arrivals
of phase 1 (P1.3, P1.5), which are guarded, and between ticks; `spec-regions.md`
6.1.2 and 17.8 still recommend phase 7 for it and are edited (22).

| ID | What runs, in this order | Owner | Reads / writes | Draws | Events |
|----|--------------------------|-------|----------------|-------|--------|
| P7.1 | For each region in ascending `RegionID`, the entry step, in this order: (a) traineeships: completions (a role change, cause `traineeship`), then places filled from the queue in queue order, then the ticks served advance (position: C5); (b) the population module computes `PopDeaths` per agent in ID order and `PopBirths` per role in role order, the carries advancing; (c) the view is built (its `Budget` is advisory, a ceiling at view time) and `Decide` is called; (d) the plan is executed in plan order: rung-3 reductions and removals; the module's deaths in agent ID order; **then the pool budget `PoolMoney * EntryPoolShare / 10 000` is taken once**, so that the estates of the dead fund the same tick's births (ruling C15); rung-1 and review switches and applies; renewal entries (none while the population module is on); a removal and a role change also end what the agent holds, in the removal steps of `spec-agents.md` 3.6 (2b titles, 2c hires, 2d and 2e claims, 2f covers, 2g facilities), and a role change into a role with no base room ends its stores (STO 3.6.1a); the module's births | SPN 10.2, 7.3; POP 4.1, 4.8; SKL; AGT 3.6, 3.7, 3.8 | agents, pool, `Privation`, welfare records; writes weights, roles, accounts, jobs, estates, pool, queues, carries | (c) a consumer spawner's own; (d) two creation draws (5.7) as each new agent is created, in plan order | in step (a), in queue order, each completion's `RecipeAbandoned` (if the trainee had a job, reason `role_changed`), then `TraineeshipCompleted`, then the `RoleChanged` it causes, and then `TraineeshipStarted` for each place filled, before the events of steps (b) to (d); then each executed operation's own events, in plan order; a removal's, in the order of 6.11 item 8; then `EntryLimited`; then `PopulationChanged` per role ascending |
| P7.2 | After the entry step of **every** region: one migration pass for the world: the counts are taken once; each agent that is eligible, settled and pushed is evaluated in ascending agent ID with its rumours, **as of tick `t + 1`** (ruling C17); admission in order of earned ratio then agent ID against the guard, the caps and the provisioning; the departures in admission order | MIG 10.2 | counts, journeys, rumours; writes journeys, job abandons, the refusal `on_journey` | one per source pair for each pushed household that is evaluated (a pushed household whose role has no recipe ends in `no_estimate` before its candidates are formed, MIG 6.3.4, and takes none), in ascending agent ID, then (destination, commodity), when `RumourErrorBP` is above 0 | per departure in admission order the `RecipeAbandoned` of its job (if any) and then, after the last departure, `MigrationStarted` in admission order; then `MigrationReviewed` per region ascending |
| P7.2a | The settlement review, with the settlements module on, per region ascending, an outside node skipped: the class by its thresholds, with the dwell | SET 4, 5.1 | the weights of the region's agents, the saved class and `Since`, the floor, the effective thresholds; writes class and `Since` | none | `SettlementClassChanged` or `SettlementHeld` per region |
| P7.2b | The aggregates of the institutions module, per region ascending, then body, right or control | ORG 7.3.1 | the counts of P1.9a, P2.5, P3.2 and P7.1 (scratch) | none | `MembershipChanged`, `RightRefused`, `PriceControlled` |

6.9.0. **Who executes the plan.** The world, not the spawner, executes the spawner's plan at P7.1(d): the
executor composes the primitives of the ledger, the estates, the production and the welfare
records (in `phase_exit.go`), and `removeAgent` is the one removal path (architecture.md 2.10.4; in `fanin.go`). The removal composes the steps of `spec-agents.md` 3.6 in order and its events
in the order of 6.11 item 8. `GoodsListed` of `EstateOpened` counts lots: one per distinct
lot key (commodity, born, durability) of the holdings that moved to the estate, whatever the unit
count; units travel separately in `WeightChanged.Units` (`spec-estates.md` 2.1).

6.9.1. **Per region, interleaved (ruling C3).** Region `r + 1`'s computation,
view and `Decide` come after region `r`'s execution and its creation draws. A
consumer spawner's draws in `Decide` come from the stream `(siteSpawner, t, region)` and each
new agent's creation words from its own `(siteCreation, t, agent)` (7.3), so the interleaving
moves no word of any stream. The regions share no pool, so the interleaving changes no result
of the default.

### 6.10 Phase 8: end

| ID | What runs, in this order | Owner | Reads / writes | Draws | Events |
|----|--------------------------|-------|----------------|-------|--------|
| P8.1 | Ageing and spoilage: every account in ascending ID, every lot whose `Born + ShelfLife - 1 <= t` moves whole to `spoilage`; the earliest-expiry hint makes the cost proportional to what expires | LED 5.3 | lots; writes lots, boundary balances | none | none |
| P8.1a | The strikes on stores, with `Modules.Risk`: the common store rows, then the stores of each limited agent with a stored load above 0; all are decided, then applied; the goods lost go to `loss` | RSK 6.2 | store rows, lots; writes lots, boundary balances | D15: one per common store row, then one per limited agent's store row, in that order | `StoreLoss` per (agent, store class), ascending agent ID then class |
| P8.1b | The settlement of the store covers of the tick, ascending cover ID | INS 8 | the amounts of `StoreLoss` (scratch), covers, balances; writes balances, covers, claims | none | `CoverEnded`, then `CoverSettled` per region |
| P8.2 | The flow totals of the tick are final | LED 7.4, 12.1 | totals | none | `BoundaryFlow` for each non-zero (region, reason, money or commodity) of the reasons of kind faucet or sink only (the internal reasons are told by their own events), in the order of `spec-observability.md` 4.4.1.2 |
| P8.3 | Regeneration of every endowment, ascending (`RegionID`, `ResourceID`), with the season of the tick; the carry advances | RES 4.1 | the stock the draws left; writes stock, carry, run | none | `EndowmentChanged` in that order |
| P8.3a | The rent step of every endowment with a title, in (`RegionID`, `ResourceID`) order, then the title step: the posted rent moves by a bounded step on the counters of the tick; the internal reasons `rent` and `title_sale` | LND 5.5 | the counters of the tick, titles, balances; writes balances, titles, the posted rent | none | `RentPaid` per endowment, after `EndowmentChanged` |
| P8.3b | The facility step: decay, ruin (a forced `removeStore`), the install of store facilities, the rebuild of the effect tables with the ground clamp, `FacilityLimited` | CAP 6.3 | the facility list and the stocks; writes conditions, the list, `Installed`, the tables, stores, endowment stocks; creates no goods (FR-CAP-042) | none | `StoreChanged` (cause `facility`), `FacilityChanged`, `EndowmentAdjusted` (kind `facility`), then `FacilityLimited` |
| P8.3c | The technology step, with the module on: the fold of the execution counters, the trend, the discovery, the snapshot of the gains, the diffusion and the clamp | TEC 4.6 | the counters, the saved cells, the graph tables, the key; writes the cells and the derived gains | none (the keyed hash of D16 takes no word) | `TechnologyAdvanced` per cell in ascending (`RegionID`, field), after the events of P8.3b |
| P8.4 | With `WithAudit`: the ledger's audit and the checks of the modules that are on | LED 12.4 | everything; writes nothing | none | none (the error `ErrAuditFailed`; it sets `Failed` of P8.5) |
| P8.5 | `TickEnded`, with `Failed` true if the audit failed (and at P4.1 on that failure, 3.5.6) | OBS | | none | `TickEnded` |
| P8.6 | The tick counter advances: `Tick()` is `t + 1`; then the flow totals and the per-tick counters of the modules are cleared (so a move made by a call between ticks is totalled in the next tick, 8.1.3) | this spec | | none | none |

6.10.1. **P8.1a to P8.3c and the totals of P8.2.** P8.1b and P8.3a move money between accounts under
internal reasons (a payout, `rent`, `title_sale`) and P8.3b and P8.3c create no goods and no
money across a boundary reason (FR-CAP-042), so the flow totals that P8.2 declares final stay
final; the only boundary reason that moves before P8.2 is P8.1a's `loss`.

### 6.11 The order of the events of a step

The events of one `Step`, in the order a recording observer sees them. An event
type appears where its position is; within a position the order is the one in the
row.

1. In the first step of a world in its construction phase that holds a bucket
   (5.5.3): the construction bucket's `BoundaryFlow` events with `Tick` -1 (P0.2a),
   then `TickStarted`. In every other step `TickStarted` is first. Then, with the policy
   module on, for each due actor in ascending `ActorID` (P0.3a): the slot events of the
   application of its entries (cause `policy`: `MoneyFlowScaled`, `EdgeStatusChanged`,
   `TransitFactorChanged`), then its `PolicyDecided`, before the first event of phase 1.
2. Phase 1: per shipment `AgentRelocated` (a traveller) then `ShipmentArrived`; the
   `CoverEnded` events of the route covers of those shipments in cover ID order, then
   `CoverSettled` per region (P1.3a); the `ContractSettled` events in contract ID order;
   the migration arrivals (`AgentRelocated`, cause `migration`) in agent ID order;
   `ReliefPaid` per region; per money flow in flow ID order `MoneyFlowLimited`, then
   `MoneyFlowEnded` of the same flow; the `LoanEnded` events in claim ID order, then
   `CreditSettled` per region (P1.7a); `FiscalFlowApplied` per flow and region (P1.7b).
3. Phase 2: for each completion of P2.1 in rotation order `RecipeExecuted`, then the
   `HireEnded` (`completed`) of its hire, then the `FacilityChanged` (`built`,
   `repaired`) of a build or repair it completes; then the `HireFormed` events of the
   labour round in formation order (P2.5a); then the one-tick jobs of P2.7 in start
   order, each in the same way; `OwnUseChanged`; then `ProductionHalted` and `RecipeRefused` (P2.8a); then
   `LabourRound` per region (P2.8b).
4. Phase 3: per agent in ID order its `ShipmentDispatched` or `ShipmentRejected`,
   then its `OrderPlaced` or `OrderRejected`; then per contract in formation order
   `ShipmentDispatched` and `ContractFormed` (a formation that fails at dispatch
   gives its `ShipmentRejected`); `ContractRound` per region; `MerchantsPlanned` per
   region (P3.3a); the `LoanFormed` events in formation order and `CreditRound` per
   region (P3.3b); the `CoverFormed` events in formation order and `CoverRound` per
   region (P3.3c); the estates' `OrderPlaced` (and `OrderRejected` for an ask the
   clearing refuses); the treasuries' `OrderPlaced` and `OrderRejected` and then
   `TreasuryOrdered` per authority (P3.4a); the counterparties' `OrderPlaced` and
   `OrderRejected`; `RumoursServed`; `WantShortfall`; `RoomLimited` (P3.6a);
   `SavingPlanned` (P3.6b); the toll `FiscalFlowApplied` events (P3.6c).
5. Phase 4: per market `TradeExecuted` then `MarketCleared`; the duty
   `FiscalFlowApplied` events (P4.1a); `OutsideTraded`, `OutsideEnded`.
6. Phase 5: `BeliefsUpdated` per market.
7. Phase 6: `UpkeepShortfall`.
8. Phase 7: per region, **first** the traineeship events of the entry step's
   part (a), in queue order: each completion as the `RecipeAbandoned` of its job (if
   any, reason `role_changed`), then `TraineeshipCompleted`, then the `RoleChanged`
   (cause `traineeship`) it causes, and then `TraineeshipStarted` for each place
   filled; then the executed operations' events of parts (b) to (d), then
   `EntryLimited`, then `PopulationChanged`; after the last region, per departure in
   admission order the `RecipeAbandoned` of its job, then `MigrationStarted` in
   admission order, then `MigrationReviewed`; then, per region in ascending `RegionID`,
   `SettlementClassChanged` or `SettlementHeld` (P7.2a); then `MembershipChanged`,
   `RightRefused` and `PriceControlled` in (`RegionID`, body, right or control) order
   (P7.2b).

   **One removal**, from P7.1(d) or a call between ticks, emits in this order:
   `RecipeAbandoned` for its job (reason `agent_removed`), `TraineeshipEnded` if it
   was in a traineeship, the `ContractSettled` of each contract of which it was the
   deliverer (`deliverer_removed`), then the events of the removal steps of
   `spec-agents.md` 3.6, in step order and all before `AgentBankrupted` (ruling C28): the
   `TitleTransferred` of each title moved, in ascending (region, resource) order (2b);
   the `HireEnded` of each hire, the worker's side first and then, in ascending hire ID,
   the employer's, each with the `RecipeAbandoned` of the worker's job it ends (2c);
   the `LoanEnded` of each claim ended by the waterfall, in ascending claim ID (2d), and the
   `LoanPassed` of each claim passed to the estate (2e); the `CoverEnded` of each cover
   ended, in ascending cover ID (2f); and the `FacilityChanged` of each facility of the agent, in ascending facility ID
   (2g): `transferred` (cause `owner_removed`) for a site, route or ground facility,
   which passes to the estate of its region, and `ruined` (cause `owner_removed`) for a
   store facility, its `StoreChanged` first (`spec-infrastructure.md` 7.1.1, AGT 3.6 step 2g); and last `AgentBankrupted`, then `EstateOpened`.
   The removal of a **trainer** first ends its trainees' traineeships, and those
   events come first: trainee by trainee in ascending agent ID, each trainee's
   `RecipeAbandoned` (reason `principal_left`) and then its `TraineeshipEnded`
   (`trainer_left`), and then the trainer's own list above.
   `AgentBankrupted` is **the only event that removes the agent**; the others
   accompany it (ruling C11). A role change emits `RecipeAbandoned` (`role_changed`)
   and then `RoleChanged`, and a role change that completes a traineeship emits
   `RecipeAbandoned`, `TraineeshipCompleted` and then `RoleChanged`; a role change ends the
   agent's hires first (for an employer, each hire in ascending hire ID with the
   `RecipeAbandoned` of the worker's job and the `HireEnded` of cause `employer_role`; for
   a worker its `HireEnded` of cause `worker_role`), before its `RoleChanged`
   (`spec-employment.md` 6.5.2), and after the `RoleChanged` the `StoreChanged`
   (`removed`, cause `role_changed`) and the `FacilityChanged` (`ruined`, cause
   `role_changed`) in ascending facility ID; a weight reduction emits `WeightChanged` and no abandon
   (the job's `Executions` is scaled, `spec-production.md` 5.4). A journey dropped by
   a removal has no event of its own: it is counted in `MigrationReviewed`.
9. Phase 8: `StoreLoss` per (agent, store class) in ascending agent ID then class order
   (P8.1a); the `CoverEnded` events of the store covers settled in P8.1b in ascending
   cover ID, then `CoverSettled` per region; `BoundaryFlow`; `EndowmentChanged`;
   `RentPaid` per endowment (P8.3a); the `StoreChanged`, `FacilityChanged` and
   `EndowmentAdjusted` (kind `facility`) events of the facility step and then
   `FacilityLimited` per region and class (P8.3b); `TechnologyAdvanced` per (`RegionID`,
   field) (P8.3c); `TickEnded` (with `Failed` true if the audit failed). A step that fails at P4.1 ends with its `TickEnded` of `Failed`
   true after the events before the failure (3.5.6).

Events of a call between ticks (`EdgeStatusChanged`, `MoneyFlowScaled`,
`EndowmentAdjusted` (also kind `facility`), `OutsideShocked`, `YieldShocked`,
`TransitFactorChanged`, `PopulationScaled`, `RentScaled`, `CreditRateScaled`,
`RouteRiskScaled`, `StoreRiskScaled`, `LegalLevelSet`, `RightSuspensionSet`,
`TechnologyAdjusted`, `StoreChanged`, `FacilityChanged`, `TitleTransferred`, the
`LoanEnded` of a forgiven claim, and the `AgentSpawned`, `AgentRelocated` or the events
of a removal (item 8) for an external agent) are emitted at the call, carry the `Tick`
the next step will run, and come before that step's `TickStarted`. The one exception is
the write of a policy actor (8.2.2): its slot events come at P0.3a, after `TickStarted`
(item 1). The construction
bucket's `BoundaryFlow` events (item 1) are emitted at the entry of the first step,
so they come after the events of every call made between `NewWorld` (or a restore
into the phase) and that step, and before its `TickStarted`.

### 6.12 Positions that two specs claimed, and wordings that disagree (the rulings)

Where a ruling says a spec's text wins, the other text is edited (22). None of
these changes a result that any spec's tests could depend on, because nothing has
run: they fix a sentence, a position or an order that two readings of the
written specs would have implemented differently.

| # | Matter | The texts | Ruling | Text that wins | Edit owed |
|---|--------|-----------|--------|----------------|-----------|
| C1 | Starting money in phase 1 | AGT 4 phase 1 and 4.0 say "relief and starting money are paid"; EST 2.3 and SPN 4.3 fund an entrant at its creation, in phase 7 | Phase 1 pays relief only. Starting money moves at creation | EST 2.3, SPN 4.3 | AGT 4 (row 1, 4.0) |
| C2 | Who sets `Privation` and when | AGT 5.4.1: phase 6 with `Hunger`; SPN 5.1 and 10.2: "updated by the world before `Decide`", per region in phase 7 | Once, at P6.1; the spawner and POP read it. A new agent starts at 0 | AGT 5.4.1 | SPN 5.1, 10.2 wording; POP 4.1 |
| C3 | Creation draws against `Decide` | N-051 (SPN update): "the creation draws after all `Decide` calls"; SPN 10.2: per region, view, `Decide`, execute | Per region (6.9.1) | SPN 10.2 | N-051 wording; SPN 3.1.2's last sentence |
| C4 | The residue of a job created at construction | PRO 6.6.2: the draw is made "when an agent is created"; VIA 8.6 item 5: the draws come after the warm start | After; the job's pending output uses residue 0 (5.6.2) | VIA 8.6 | PRO 6.6.2 (one sentence); VIA 8.5.2 (one sentence) |
| C5 | The positions of the traineeship | SKL 4.2, 4.4 and FR-SKL-015 give the rules and no tick position for the accumulator, the places, the ticks served or the completion | P2.2 and P7.1(a) | this spec | SKL 4.2 and 4.4 cite them; SKL fixes the arithmetic of "ticks served" |
| C6 | The estate's markdown | EST 2.7.3, FR-EST-009: "each tick a lot stays unsold", no position | P5.3 | this spec | EST 2.7.3 cites it |
| C7 | The registration of `Clearing` and its place in the hash | CLR 8 shows the interface and says nothing of registration; CFG hashes strategy and spawner names only | 4.2: a name, an option, a check | this spec | CFG 3.7, 8.1; CLR 8; the parameter-owner table |
| C8 | Which random calls | INF 5.5.1 `Int64N` (FR-INF-014); EXT 6.4.1 `Uint64N`; WEL 5.1 and PRO 6.6.2 (FR-PRO-026) "one draw"; BEL 5.2 `Uint64` and `bits.Mul64`; CLR 4.2 and MER 9.4.2 raw `uint64` keys | One source word per draw, the high word of the product (7.2). The keys are raw words and need no edit | BEL 5.2 | INF 5.5.1, FR-INF-014; EXT 6.4.1; WEL 5.1; PRO 6.6.2, FR-PRO-026 (wording) |
| C9 | `Work` interleaved with starts | RES 5.2 describes the two passes with the module on; PRO 5.2 item 2 says nothing of a pass without it | Always two-pass (6.4.1): this spec's own rule | this spec | PRO 5.2 and 5.5.1 wording; RES 5.2 cites it |
| C10 | Where a trainee's upkeep is eaten | AGT 5.7, SKL 4.2: "in the consumption phase, from the trainer's account", no position | At the trainee's own position of P6.1 | this spec | AGT 5.7 cites it |
| C11 | `AgentBankrupted` "the only event of a full removal" | AGT 3.6 step 6, 14.1, FR-AGT-008, FR-AGT-077; SPN 5.3.1, 13.1.1, FR-SPN-032; EST 2.1 and 2.5 (`EstateOpened` "accompanies"); the rules' `EstateOpened`, `ContractSettled`, `RecipeAbandoned`, `TraineeshipEnded` | `AgentBankrupted` is the only event that **removes** the agent; the others accompany it, in the order of 6.11 item 8 | this spec | the five places say "the only event that removes the agent" (22) |
| C12 | `WantShortfall` "after the trade phase", `RumoursServed` "before phase 4" | DEM 11; rules 4.3 | P3.6, both at the end of phase 3, in that order | this spec | none |
| C13 | Whether the audit precedes `TickEnded` | LED 12.4 "after the tick's events"; RES 4.1.1 "then the optional audit" | the audit (P8.4) precedes `TickEnded` (P8.5), so that its failure is carried by `TickEnded.Failed` and every step that started is closed (3.5.6) | RES 4.1.1; LED 12.4 reworded | LED 12.4: "before `TickEnded`" |
| C14 | The offset of a rotation | SPN 6.2 and MIG 5.1: `(tick + AgentID) mod P`; MIG FR-MIG-017: the margin offset is `Weyl32(id)` | `(t + bucket(id, P)) mod P` with `bucket(id, P) = (Weyl2(id) * P) >> 32` and `Weyl2(id) = (id * 0x6A09E667) mod 2^32`, a second Kronecker sequence that does not share a value with `Weyl32`: two spreads that act on one decision must not share a value (9.1.1) | this spec | SPN 6.2, FR-SPN-018, 15.2; MIG 3.3(5), 5.1, 11.3, MIG-D1, FR-MIG-005; STB D33, D72; MON 6.2, FR-MON-013 and DEM 4.4 (they use `ID mod P` or their own start: not rotations, noted) |
| C15 | The entry budget against the deaths | SPN 3.2, 7.3, FR-SPN-026 and EST 2.8.1: taken "once at the start of the entry step"; POP 4.8, FR-POP-019, SPN 8.2, 10.2, AGT 4: "the deaths go before the budget is taken" | The view and `Decide` precede execution; the budget is taken in P7.1(d) after the module's deaths and before the switches; the view's `Budget` is advisory | POP 4.8, SPN 10.2 | SPN 3.2, 7.3, FR-SPN-026; EST 2.8.1 |
| C16 | Transport's relocation and the guard | REG FR-REG-060: a module that relocates MUST first ask the query with its own refusal in `skip`; TRN 5.5.3, 5.7 call `Relocate` directly; MER registers `open_contract` for a deliverer who arrives at P1.3 while its contract settles at P1.4 | Transport relocates at P1.3 after the guard query run ignoring only the shipment that arrives and the contract formed on it, by instance; every other shipment, contract or obligation of the agent still refuses | this spec | REG 6.1.6, 6.5.2, 6.6, FR-REG-059, FR-REG-060; MIG 7.2.1; TRN 5.5.3, 5.7, FR-TRN-022 |
| C17 | The price record read in phase 7 | INF 4.3.2, 5.4.1, 7.1, FR-INF-018: the record is sized so that a phase-3 rumour of tick `t + 1` reads vintage `t + 1 - D`, and "is not read in phase 5 or later of the tick it is written in"; MIG 6.2, 6.3.2 read it at P7.2 "the tick just ended" | A phase-7 reader reads as of tick `t + 1`: vintage `(t + 1) - Lag`, `NotYet` against `t + 1`; the ring depth is unchanged and the oldest vintage is not overwritten under it. A candidate's effective age at P7.2 is one tick less than at phase 3; INF restates its lag floor for a phase-7 caller | this spec | INF 4.3.2, 5.4.1, 7.1, FR-INF-018; MIG 6.2, 6.3.2 |
| C18 | When the flow totals are cleared | LED 7.4 "cleared at the start of each tick"; P0.2 of the first draft | At P8.6, after the events and the audit, so a move made between ticks is totalled in the tick the next step runs (8.1.3) | this spec | LED 7.4, 9.1; REG 6.4.1 |
| C19 | The price window | BEL 7.4, FR-CLR-032: pushed only when units traded; the first draft pushed every market | P5.1 pushes only a market that traded | BEL 7.4 | none |
| C20 | What the cost anchor reads | PRO 9.3, FR-PRO-037, AGT 4: the prices published by the previous clearing; the first draft: the price windows | The published price left by P4.1 of the previous tick (the seed at tick 0) | PRO 9.3 | none |
| C21 | Outcomes of pseudo-agents' orders | CLR 7.3, FR-CLR-025, FR-AGT-023, FR-BEL-008: the outcome goes to the agent's `Revise`; an estate and a counterparty have no strategy or belief | A pseudo-agent's order has no outcome and takes no D8 word | this spec | CLR 7.3, FR-CLR-025; FR-AGT-023; FR-BEL-008 repeat it |
| C22 | Markets in any order | CLR 3.3.1, 7.2, FR-CLR-023, SCL 6.6: clearing in any order gives the same trades | Independent **given the keys**; each market's keys come from its own stream `(siteClearing, t, region, commodity)` (ADR 0008, 7.3), so the fixed order of the markets is no longer part of the draw sequence and a market's keys do not depend on the markets cleared before it | this spec | CLR 3.3.1, 7.2, FR-CLR-023; SCL 6.6 |
| C23 | Warm money and stock | AGT 19, EST 2.3, FR-VIA-032: the group's entry replaces the defaults; the first draft created both | A group with a warm entry is created with its entry in place of the defaults, one `initial` (5.4.2, 5.6.1 item 3) | AGT 19, VIA | none |
| C24 | Two specs claimed P1.7a | CRD 8.1.1 (the settlement of the claims, "the new position P1.7a") and FIS 5.3, 6, 8.4, 10.1, 12 (taxes, remittances and payouts, "P1.7a (new)"); INS, SAV, RSK, CAP and PLC cite CRD's name | By 6.1.1 the letter goes with the order, not the author: the position that runs first after P1.7 is **P1.7a CRD**, the second **P1.7b FIS**. Debt service therefore runs before taxes, and a debtor short of cash pays its claim and the tax is forgiven for the shortfall (FIS 5.3) | this spec | FIS: P1.7a becomes P1.7b everywhere (about twenty mentions); CRD, INS, SAV, RSK, CAP and PLC keep P1.7a |
| C25 | Draw IDs of the later modules | RSK 11.2 takes D13, D14 and D15; PLC 7.1.1 takes "D13"; TEC 6.3.1 takes "D16", which is also the label of the register's damper D16 (`spec-stability.md` 5: another namespace); EMP's first "D13" is withdrawn | **D13, D14, D15 RSK; D16 TEC; D17 PLC**, in the order of a run in 7.3; PLC's built-ins take none and D17 is a consumer's own words. Prose that could mean the register's damper says "draw D16" | this spec | PLC: "D13" becomes "D17"; TEC and RSK keep theirs; any text that writes D13 to D17 for a draw says "draw" |
| C26 | Step 8 of construction | LND 6.1.1: "step 8a", titles before the warm start; TEC 6.3.1: "step 8a", the key after the creation draws; CAP 3.4.3: the initial facilities after the warm start; SET 4.5: the classes after the agents; POL 5.4.1: the treasury pseudo-agents with their money before the households | Step 8 is one step with an ordered list of sub-steps (5.1.2): the pseudo-agents and then the agents, **8a LND** the titles, the warm start, the creation draws, **8b CAP** the initial facilities, the settlement classes (no letter), **8c TEC** the key; the events of step 10 follow the same order (5.8.2) | this spec | TEC: "8a" becomes "8c" (6.3.1 table, FR-TEC-011, 6.3.4, 17.2); CAP 3.4.3 names 8b; LND keeps 8a |
| C27 | The tail of P3.6 | STO 14.1 takes P3.6a and says both "new" and "no new position"; SAV 9, 14 and FIS 10.1 ask for "P3.6, after `WantShortfall`", and SAV wants `RoomLimited` before `SavingPlanned` | **P3.6a STO** (`RoomLimited`), **P3.6b SAV** (`SavingPlanned`), **P3.6c FIS** (the toll `FiscalFlowApplied`), after `WantShortfall`: stable names rather than an ordered list inside P3.6 | this spec | SAV 9, 14 and FR-SAV-027 ("no new position"): P3.6b; FIS 10.1: P3.6c; STO 11.1 drops "no new position" |
| C28 | The removal steps of `spec-agents.md` 3.6 | EMP 6.5.2 and LND 6.4.1 both ask "2b"; CRD 9 takes 2d and 2e; INS 8.6 takes 2f; CAP 6.3 and 14.1 place `FacilityChanged` after `EstateOpened` | **2b LND** (titles to the pool), **2c EMP** (the hires end, the worker's side first, the remainder of the escrow returned to the employer's account before the waterfall), **2d and 2e CRD**, **2f INS**, **2g CAP**; every event of a removal precedes `AgentBankrupted` and `EstateOpened` (6.11 item 8) | this spec; `spec-agents.md` 3.6 carries the steps | AGT 3.6 (the steps); EMP 6.5.2 and EMP-22; CAP 6.3, 14.1 (before `EstateOpened`); INS 8.6.3 (2f ends a store cover of the tick unpaid, the premium kept) |

The rulings of the wave 3 and 4 integrations stand and are the order of the tables
above: settlement of contracts before the arrivals of migration journeys, because a
party to an open contract cannot be on a journey (`spec-migration.md` 7.2); the
walk after the money flows and before the directive; the contract round after the
last plan and before the estates' asks and the counterparties' orders; the
migration pass after the entry step of every region.

## 7. The keyed random streams

Randomness is a function of `(seed, site, tick, id)` and of nothing else
([ADR 0008](../adr/0008-random-draws-are-keyed-streams.md), architecture.md 7.3, which
states the rule and wins over any text of this spec that disagrees). A world holds one
seed and no stream position. At the start of each **draw site** (a position, or a call
into an extension, that takes words) it reseeds its one PCG from a key; the words that
site takes are the words of that stream, in order. No draw depends on how many words
another site, another entity or another tick took. Open points 1 and 2 of the ADR are
answered here (7.1.3, 7.1.4, 7.3); point 3 (a site with no natural entity) stays open
for the rows outside the slice (7.3).

### 7.1 One seed, one source, and the key of a stream

7.1.1. A world has one **seed**, a `uint64`: the argument of `WithSeed`, or, after a
restore with `WithReseed`, the seed that CHK 6.8.2 derives (the **effective seed**). It
has one `*rand.PCG` (`math/rand/v2`) and one `*rand.Rand` built on it with `rand.New`,
created once at construction (architecture.md 7.3). The `Rand` is what extensions
receive (architecture.md 7.7). Between two site calls the PCG holds the state the last
stream left, and nothing reads it: **the position in a stream is not world state**, and a
checkpoint saves the seed and no PCG state (12). No other source exists in the library.

7.1.2. **The finaliser** (a rule constant, part of the golden output; changing it is a
behaviour change, architecture.md 6.6). It is the finaliser of the SplitMix64 generator,
without the increment, and the one helper of `internal/ordering` that the ration's key
(`spec-resources.md` 5.3.3) shares:

```
γ = 0x9E3779B97F4A7C15                                  the increment
finalise(z):  z = (z xor (z >> 30)) * 0xBF58476D1CE4E5B9      (mod 2^64)
              z = (z xor (z >> 27)) * 0x94D049BB133111EB      (mod 2^64)
              return z xor (z >> 31)
```

All arithmetic of this section is modulo 2^64 on `uint64`. In Go, `2 * γ` written as a
constant expression overflows `uint64` and does not compile: write `s + γ + γ`, or hold
γ in a variable.

7.1.3. **The key of a stream** (a rule constant, part of the golden output). A stream is
named by the seed and by `(site, tick, id)`, with a fourth word `sub` for a site whose
row says it needs one (7.3):

```
s = seed
for w in (site, tick, id):                  (a fourth word, sub, when the row has one)
    s = finalise(s + w + γ)
hi = finalise(s + γ)
lo = finalise(s + γ + γ)
PCG.Seed(hi, lo)                            the first word taken is the first word of the stream
```

Types and widths:

| Word | Type | Source |
|------|------|--------|
| `seed` | `uint64` | the effective seed (7.1.1) |
| `site` | `uint64` | a constant of 7.1.4 |
| `tick` | `int64` | the tick of the step the draw belongs to (7.3.2) |
| `id` | `int64` | the ID the row names: an `AgentID`, a `RegionID`, a `CommodityID`, widened to `int64` |
| `sub` | `int64` | the second ID of a row that names a pair (a commodity, with a region or an agent) |

A signed value enters as its **two's-complement** `uint64` (in Go, `uint64(x)` of the
`int64`), so `-1` is `0xFFFFFFFFFFFFFFFF`; there is no sign test and no error, and a
negative tick or ID is never a reason to refuse a draw. The arity of a site (three words
or four) is fixed by its row and never varies. An empty fold (`s = seed`, no word) is
the seeding rule that this section replaces; no site draws from it, and its four rows in
7.1.5 pin the finaliser and the generator.

The cost of a site call is one `Seed` and `3 + 2` finaliser evaluations (`4 + 2` with a
`sub`); it allocates nothing, because the PCG and the `Rand` are built once and
`PCG.Seed` writes the two words in place. A `rand.Rand` holds no state beyond its source
(checked on Go 1.25 and Go 1.27 over a battery of ten methods, after a reseed in the middle of a stream), so a
reseed of the PCG resets the whole stream.

7.1.4. **The site constants.** A site constant is a literal 64-bit value, fixed here and
in `random.go`. It does not depend on the position of its row in 7.3, on the draw ID, on
the number of sites or on the order of the modules: a later row **appends** its own
name and value, and no constant ever changes (a change is a behaviour change, 6.6). The
sites of the slice are:

| Name | Row | Value | Arity: the words after `site` |
|------|-----|-------|-------------------------------|
| `siteConstruction` | D1 | `0xC103123756D13F20` | `tick` (always 0), `id` = the agent |
| `siteWork` | D3 | `0xDE594838D3779A37` | `tick`, `id` = the agent |
| `siteTrade` | D5 | `0xFC14CD10E8A8EF43` | `tick`, `id` = the agent |
| `siteClearing` | D7 | `0xD364E6AE73E1313F` | `tick`, `id` = the region, `sub` = the commodity |
| `siteOutcome` | D8 | `0xD9CC0A517EB0D9F5` | `tick`, `id` = the agent, `sub` = the commodity |
| `siteSpawner` | D9 | `0xBE0B1D169200E9FF` | `tick`, `id` = the region |
| `siteCreation` | D10 | `0x9B8FFA303CB545DC` | `tick`, `id` = the new agent |

The rows D2, D4, D6, D11, D12, D13, D14 and D15 (and D17, a consumer's policy) have **no
constant yet**: each is appended by the change that builds its module, with its name,
its value and the `id` the owner chooses (7.3, which lists the proposals of N-214); no
code calls into an extension, or takes a word, at a row that has no constant (the
modules of those rows are off in the slice). D16 takes no word (7.3).

Why the values are what they are. The first step of the fold adds the site to the seed,
so `(seed a, site X)` and `(seed b, site Y)` name one stream if and only if
`a - b = Y - X`. Seeds are small integers chosen by people, so the sites must not be
small or near each other: no two of them may differ by less than 2^32 (a `Guard:`
test, testing.md 5.2). Each value is the first eight bytes, big-endian, of SHA-256 of
`economy-sim/site/` and the name (`models/wld/seed.py` asserts it). That is a record of
where the literal came from, which anyone can check, and not a rule: the literal is
the rule.

7.1.5. **Test vectors**, produced by `models/wld/seed.py` and printed again by a
separate Go program that uses `math/rand/v2` only, on Go 1.25.14 and Go 1.27.2; the two
are equal in every line. The PCG is a specified generator (the PCG-DXSM that its documentation names), so
the words are a function of the state and the vectors above are the check; the reduction
methods of `rand.Rand` are not part of the generator, which is why the world's own draw
is defined from `Uint64()` alone (7.2). All values are hexadecimal unless marked.

(a) **A bare seed** (the empty fold; the rows that were checked against Go 1.25 before
ADR 0008 and that the finaliser test of the first code keeps):

| Seed | `hi` | `lo` | First word | Second word |
|-----:|------|------|------------|-------------|
| 0 | `e220a8397b1dcdaf` | `6e789e6aa1b965f4` | `b18c97843f57bb91` | `d7046f21f79eba31` |
| 1 | `910a2dec89025cc1` | `beeb8da1658eec67` | `af9d32bf75779748` | `b7672df032bf0473` |
| 2 | `975835de1c9756ce` | `bfc846100bfc1e42` | `a28db69c539314fb` | `fe57a921339b50b9` |
| 42 | `bdd732262feb6e95` | `28efe333b266f103` | `61c88529c9612c1b` | `2608d8a3075aa168` |

(b) **A stream key.** `s` is the value after the fold. The last two columns are the
world's draw (7.2.1) of bound 1 on the first word and of bound 100 on the second: the
first is 0 and **still consumes its word**, so the second draw sees the second word.

| Row | Seed | Site | `tick` | `id` | `sub` | `s` | `hi` | `lo` | First word | Second word | `draw(1)` | `draw(100)` |
|-----|-----:|------|-------:|-----:|------:|-----|------|------|------------|-------------|----------:|------------:|
| k1 | 0 | `siteWork` | 0 | 0 | | `f26e83000fb3a188` | `79ae1abc6fb3c555` | `32a6affd0dbf0521` | `264d986499a1a1fa` | `0fb3474a976f0acf` | 0 | 6 |
| k2 | 1 | `siteTrade` | 5 | 17 | | `0e4289002a489b3d` | `fba1decca480b034` | `59c9e7d8138ce36b` | `e87e37a8f2ed3a12` | `46c2cfef3d61d69c` | 0 | 27 |
| k3 | 42 | `siteClearing` | 100 | 2 | 3 | `90d85e42a2621beb` | `92cfbdaff0085569` | `df1d7be0b72c91fe` | `366090c9bf387982` | `fe2c271dd54ed665` | 0 | 99 |
| k4 | 42 | `siteOutcome` | 100 | 17 | 3 | `bb905db0b8a8bb2e` | `4c1a052c0514495c` | `1ebc9cbf8410ab07` | `30967dc8ee4ec36e` | `d5b237e50696f3bf` | 0 | 83 |
| k5 | 2^64 - 1 | `siteConstruction` | 0 | 4 | | `a567852979e47fd7` | `7e12e8d9e610f1f3` | `11e76e20e84424df` | `64d9772cd475bf00` | `4dd6b1aa70460b76` | 0 | 30 |
| k6 | 7 | `siteCreation` | -1 | 12 | | `8a6566c1ca36e2f3` | `290ef3893fee12ef` | `aa22a25ddaace732` | `1b6b4a321827daf5` | `240b83bc1e00988b` | 0 | 14 |
| k7 | 7 | `siteSpawner` | 3 | -1 | | `ef7edb14a4433105` | `11f3bdf788e5ffb9` | `4a115493bf1bf035` | `eeeaa6fd69c7b339` | `3355b2e3c0d174dd` | 0 | 20 |
| k8 | 7 | `siteSpawner` | -1 | -1 | | `b4a503670dc250f1` | `cc1b665a6f86d8ba` | `6901bc968a06bc0f` | `f5455bb2a283bc00` | `85eba190e6f38b27` | 0 | 52 |
| k9 | 2^63 | `siteTrade` | 2^40 | 2^40 + 1 | | `f01e597ee04203eb` | `87e7225b79ecaea7` | `ca410d7c919c53eb` | `3c277642ea207186` | `510c6f3a642bcd9d` | 0 | 31 |

Rows k6 to k8 pin the signed conversion (a negative tick or ID never occurs in a run,
`NoRegion` and `NoAgent` aside, and the conversion is pinned all the same); k9 pins
values above 2^32 in both words.

(c) **`K(t, id)`**, the stateless key of 7.1.7:

| `t` | `id` | `K(t, id)` |
|----:|-----:|------------|
| 0 | 0 | `e220a8397b1dcdaf` |
| 1 | 0 | `6e789e6aa1b965f4` |
| 0 | 1 | `910a2dec89025cc1` |
| 5 | 17 | `14db9121e4c732ea` |
| 100 | 3 | `a854cdf21653e5e5` |
| 2^40 | 123456789 | `2cd5718615e1be22` |
| -1 | -1 | `b4d055fcf2cbbd7b` |
| -1 | 0 | `0000000000000000` |

`K(0, 0)` and `K(1, 0)` equal the `hi` and `lo` of the bare seed 0 because all three are
`finalise(γ)` and `finalise(2γ)`: a test that compares a stream state with a key must
not take them for one rule. `K(-1, 0)` is 0 because `-γ + γ` is 0 and `finalise(0)` is 0;
a key of 0 is a legal key.

(d) **The reseed** `seed' = finalise(seed xor finalise(salt))` and its sixteen vectors are
in `spec-checkpoint.md` 6.8.2 and 17.5, from the same file.

7.1.6. **What the chain claims, and what it does not.** Three finaliser evaluations
(four with a `sub`) separate the seed from the state, so a one-bit change in the seed, a
site, a tick or an ID changes about half the bits of the first outputs: the model of
`seed.py` measures 32.04, 32.03 and 32.03 differing bits of 64 on average over 2 000
neighbouring IDs, ticks and seeds (the bare mix gives 31.99 and the plain
`NewPCG(seed, 0)` 31.94, so the mix is chosen for the guarantee and for the cost of one
function, not because the plain rule fails). The independence of two streams rests on the
mixer, which is the assumption the stateless keys already make (`K`, the ration's key).
A stream is a function of the 64-bit value `s`, so two keys collide, and then draw the
same words, with probability about 2^-64 for one pair; over a run with `n` keys the chance
of some pair is about `n^2 / 2^65`, so about 0.7% for `5 * 10^8` keys (5 000 agents, ten
sites, 10 000 ticks), and the consequence is two draws of `n` that are equal, which
changes no statistic and is deterministic. The mixer is not a cryptographic one and no
claim is made that a stream cannot be predicted (the stream is not for secrets,
`WithSeed` says so).

7.1.7. **The stateless key `K(t, id)`** (a rule constant, part of the golden output),
the key of RES 5.3.3 and the tie-break of 10.1.1, in the form that FR-RES-019 cites:

```
x = t * γ + id            (mod 2^64, with t and id as in 7.1.3)
z = x + γ
K(t, id) = finalise(z)
```

that is, `K(t, id) = finalise(t * γ + id + γ)`: the extra increment before the mix is
part of the rule (FR-WLD-091 said `finalise(t * γ + id)` and omitted it, and RES 5.3.3
and the vectors of 7.1.5 (c) have it). It takes no word, keeps no state and is not a
stream. A tie broken by it is a total order that ends in the ID (10.1.1).

### 7.2 What a draw is

7.2.1. **The world's draw.** For a bound `n` of at least 1, from the stream in use:

```
draw(n) = high word of ( Uint64() * n )              a value in [0, n)
```

with `math/bits.Mul64`. It consumes exactly **one source word** of that stream, has no
loop and no rejection, and its bias is at most `n / 2^64`, which is below any quantity
the library handles. It is the reduction of `spec-beliefs.md` 5.2 and 6, adopted for
every draw the world takes (ruling C8 of 6.12). A bound of 1 still consumes a word
(row k1 of 7.1.5), so a stream does not depend on a bound that happens to be 1 (BEL 6).
A range `[lo, hi]` is `lo + draw(hi - lo + 1)`. The raw `Uint64()` is used where a key is
wanted (the tie-break key of a market, a contract key, the world's outcome draw).

7.2.2. **An extension's draws.** A `Strategy`, `Spawner`, `Clearing` or `Policy`
receives the world's `*rand.Rand`, reseeded for the call it is in, and may call any
method of it. Such a method may consume more than one word: in the check of 7.1, `Int64N` with a
bound below 2^32 took one word in each of 200 000 calls, and with a bound near 2^62
about 1.33. The number is a deterministic function of the stream, so it never breaks
reproduction, and it moves only **the later draws of the same stream** (a strategy that
takes more words from its agent's `Trade` stream moves that agent's later words in that
call and nothing else). The extension may not keep the `Rand` after the call returns
(architecture.md 7.7).

7.2.3. **The unit of this spec** is therefore the **word**: a row of 7.3 says how many
words the world takes from its stream at that position, and a consumer's own calls are
added to the same stream in the order they are made.

### 7.3 The draws of a run

Every draw the library takes, in the order of a run. IDs are stable; a spec that adds a
draw appends a row with a new ID, its count and its stream (7.1.4). The last column
names the stream: the site, then the `id` (and the `sub`) with the `tick` of 7.3.2.
`none yet` means that the row has no site constant (7.1.4); the `id` after it is the
proposal of N-214 that the owner confirms when it appends the constant, and ADR 0008's
open point 3 (a site with no natural entity: D2, D13) stays open for it.

| ID | When and where | Taken by | How many | In what order | Stream | Owner |
|----|----------------|----------|----------|---------------|--------|-------|
| D1 | construction, after the warm start (5.7) | the world, per household agent | 2: the target ratio, then the residue | ascending agent ID | `siteConstruction`, tick 0, the agent | WEL 5.1; PRO 6.6.2 |
| D16 | construction, step 8c (5.1.2), only when the technology module has a field with `DiscoveryBP` above 0 | **no word**: TEC derives its key from the effective seed (`spec-technology.md` 4.4.1, owed under N-214) | 0 | | none | TEC 4.4.1 |
| D17 | P0.3a, inside a policy actor's `Decide` | a consumer's `Policy`; the built-in policies take none | its own | due actors in ascending `ActorID`, call order | none yet: the actor | PLC 7.1.1 |
| D2 | P1.8 | the world, per line with `WalkStepBP` above 0 | 1 | ascending (node `RegionID`, `CommodityID`) | none yet: the line (open point 3) | EXT 5.1, 6.4 |
| D3 | P2.5 | a consumer's `Work`; the default takes none | its own | rotation order, in call order | `siteWork`, the tick, the agent | AGT 7.1 |
| D13 | P3.1a, with `Modules.Risk` | the world, per route row with `Common` and a loss probability above 0 | 1 | ascending (`From`, `To`, `Mode`) | none yet: the route row (open point 3) | RSK 4.2 |
| D4 | P3.2(a), just before that agent's `Trade` | the world, per candidate of an informed agent's view, offered or not, when `RumourErrorBP` is above 0 | 1 | informed agents in ascending ID; within a view ascending source `RegionID`, then `CommodityID` | none yet: the agent | INF 5.5.2 |
| D5 | P3.2(b), in `Trade` | the strategy: the default one per intent it considers; a consumer's own | per intent | agents in ascending ID; intents in canonical commodity order | `siteTrade`, the tick, the agent | AGT 7.5.3 |
| D14 | P3.2(c) inside each agent's turn, after that agent's own words, and P3.3 after D6's words, with `Modules.Risk`: at the record of each dispatched shipment | the world, per unit row on the shipment's path with a loss probability above 0 | 1 | the shipments in the order they are dispatched, the rows of a path in path order | none yet: the shipment | RSK 4.1 |
| D6 | P3.3, after admission and before the first lane | the world, per admitted offer in admission order, **then** per admitted take in admission order (admission itself is one pass over offers and takes in agent ID and plan order, MER 9.3; within each kind that pass is the admission order) | 1 | admission order | none yet: the offer or take | MER 9.4.2 |
| D7 | P4.1, inside each `Clear` | the **default clearing**, per admitted order of the market (the key), bids then asks, an estate's ask, a treasury's order and a counterparty's order included; a consumer's `Clearing` draws as it likes from the `Rand` it receives (7.2.2) | 1 | markets in ascending (`RegionID`, `CommodityID`); orders in input order (6.5.1) | `siteClearing`, the tick, the region, the commodity | CLR 4.2 |
| D8 | P5.4 | the world, per admitted order **of a household agent** (the outcome's `Draw`), then the strategy's own in `Revise`; an estate's ask, a treasury's order and a counterparty's order has no outcome and takes no word (C21) | 1 | ascending (`RegionID`, `CommodityID`), admission order | `siteOutcome`, the tick, the agent, the commodity: **one stream per order** | BEL 4.5, 5.1.1 |
| D9 | P7.1(c) | a consumer's `Spawner.Decide`; the default takes none | its own | regions ascending, call order | `siteSpawner`, the tick, the region | SPN 3.1.2, 10.2 |
| D10 | P7.1(d), at each creation | the world, per new agent: the target ratio then the residue (D1's two) | 2 | plan order, regions ascending | `siteCreation`, the tick, the new agent | AGT 3.5.3 |
| D11 | P7.2 | the world, per source pair of each **pushed** household that is evaluated, when `RumourErrorBP` is above 0, offered or not (INF 6.1.3 and FR-INF-051 say "reviewed", a larger set of MIG 13.1: the set is the pushed households, FR-MIG-013) | 1 | ascending agent ID; within an agent ascending (destination, commodity) | none yet: the agent | MIG 6.2.1, 10.2.1; INF 6.1.3 |
| D15 | P8.1a, with `Modules.Risk` | the world, per common store row with a probability above 0, then per store of a limited agent with a stored load above 0 | 1 | common rows in table order, then agents ascending and their stores by class | none yet: the row, or the agent and the store class | RSK 6.2 |
| D12 | between ticks, at the call | the world, per agent that INT adds: D10's two creation words | 2 | in the order of the calls | none yet: the new agent, with the tick of the next step | AGT 10 |

The rows are in the order of a run, not of the ID. The IDs D13 to D17 are the draw IDs
of ruling C25 (6.12). D16 is not the damper D16 of the register (`spec-stability.md`
5): wherever the two could be meant a text says "draw D16". The draw D17 and D3, D5 and
D9 of a consumer's extension are in call order and are not terms of 7.3.1.

7.3.1. **The words of one tick**, for a world with no consumer extension, is the sum of
the words of the streams of the tick:

```
walk lines                                        D2
+ common route rows with P > 0 (Risk)          D13
+ candidates of the informed views (E > 0)        D4
+ intents the default considers                   D5
+ unit rows on the paths of the shipments dispatched (Risk)   D14
+ admitted offers + admitted takes                D6
+ admitted orders (default clearing)               D7
+ admitted orders of household agents              D8
+ 2 * agents created in phase 7                   D10
+ source pairs of the evaluated pushed households (E > 0)   D11
+ common store rows with P > 0 + stores drawn (Risk)  D15
```

with `E` the information error. Each term is a function of the configuration and of the
state at the start of the tick (7.4): the number of walking lines, of informed agents
and the tradable pairs they can see, of intents the strategy considers, of orders the
agents' budgets admit, of agents the ladder and the module of population create, of
households that the migration rule pushes. Under ADR 0008 the terms are counts of
**separate streams** and no term moves a word of another: a number that changes a term
changes the words of its own stream and of nothing else. The risk module's three terms
are present only with `Modules.Risk`, and D17 (a consumer's own) is no term of a tick.

7.3.2. **The tick of a key, and construction against a call between ticks.** The `tick`
word is the tick of the step that takes the draw, the value `Tick()` has during that
step, whatever the position (P7.1 of tick `t` keys `t`, though P7.2 reads as of
`t + 1`). At construction it is 0. A call between ticks that draws (D12) uses `Tick()`,
which is the tick the next step runs, and so is **0 for a call made before the first
step, the same tick as construction**. The two are told apart **by the site alone**:
construction draws at `siteConstruction` and a call at the site of its own row (D12,
appended with its constant), and a draw about an agent is keyed by an ID that no other agent has (IDs are never
reused, 5.3.1). A row whose key could be taken twice in one run for one `(site, tick, id)` is a
defect of the row, and its owner adds a component (a `sub`) when it appends the site.

7.3.3. **A site is called once per key.** D1 once per household agent at construction,
D3 once per free agent per tick, D5 once per agent per tick, D7 once per market with an
admitted order per tick, D8 once per admitted order of a household agent, D9 once per
region per tick, D10 once per created agent. At each call the world reseeds before it
calls the extension or takes its first word, so that what a consumer's code draws in the
call is a function of the key and of its own earlier draws in the same call.
A call to the default strategy at a site where the default takes no word needs no
reseed: the reseed has no observable effect, so the world may skip it.

### 7.4 The count is a function of configuration and state

7.4.1. **Rule.** The number of draws in a tick, and the order of every one of them
within its stream, depend on the configuration, the state of the world, the registered
extensions and the calls made between ticks. They do **not** depend on: an observer, a
logger, `WithAudit`, whether an event was constructed, a snapshot or any read, the
configuration hash, the number of `Run` calls the ticks were split into, or whether the
world was checkpointed and restored in between. And they do not depend on the number of
words that **another** site, entity or tick took (7.6).

7.4.2. **Modules.** A module that is on and does nothing takes no word, because it calls
no site (FR-INF-047, FR-RES-045, FR-POP-036, FR-MON-041, FR-SUB-035, FR-EXT-021,
FR-EMP-061, FR-ORG-017, FR-POL-020, FR-CAP-002, FR-STO-037). Under keyed streams this is
no longer what keeps two runs aligned, since a module's words are in streams of its
own: it is kept because a site that is not called takes no `Seed` and no word. A module
that adds a **creation** word adds it to D1's table for every agent whatever its switches
(5.7.2), so that the words of an agent's creation stream do not depend on the modules
that are on. There is no module-level word: D16 takes none (7.3).

7.4.3. **A draw that might be skipped is taken anyway** inside one stream, when
skipping would make a later word of that stream depend on an outcome: the default
strategy draws for every intent it considers even when affordability then removes the
order (`spec-agents.md` 7.5.3), so that the later intents of the same agent keep their
words; a pushed household draws for every candidate whether or not a rumour is offered.
It is an owner's choice inside a stream and no longer a rule that keeps two runs aligned
(the belief draw per outcome is a stream of its own, D8, and needs it for nothing). The
storage, saving and institutions modules change the **count** of D5 to D8 by what they
refuse, cut or bar (a bid cut by the room, a project that changes a budget, a place a
household is refused), because those counts are functions of state (7.4.1); none of them
adds a word, and a count that falls moves only the words of its own stream.

7.4.4. **How it is tested.** A run with a recording observer and a capturing logger and a
run with neither produce the same bytes of `Snapshot` at every tick and the same
effective seed (observability-and-logging.md 8.1); a run split into `Run(n)` calls, and
a run checkpointed and restored at a tick boundary, equal an uninterrupted run
(FR-WLD-063). A world that draws is also tested for **alignment**: two runs of one seed
that differ in a parameter, a shock or a module draw the same word at every
`(site, tick, id[, sub])` they share (FR-WLD-069).

### 7.5 What does not draw

The ledger and ageing (LED 5.3), the ration of a scarce endowment (a stateless
hash key of tick and agent, RES 5.3.3), the carries of population, demand, money
flows, regeneration and the outside's capacity, the per-agent spreads of the
migration margin, the merchant's fee share and the death carry (the Weyl offset of
the agent ID, POP 3.8.3), the stagger of a warm start (VIA 8.4.3), the rotation of
starts, the review rotation, the subsistence flag, the settlement of contracts,
the sweep, regeneration, the cost anchors and the welfare record, and, of the later modules,
the credit round's and the cover round's order (the stateless key) and the settlement's
remainders (CRD 6, 8; INS 7.3, 8), the tiers of tenure, the apportionment and the rent step
(LND 8.2), the labour round (ties by the keyed `K(t, id)`, 10.1.1; EMP 5.3), the membership
round (ORG 4), the fiscal flows' pro rata cut (a keyed tie, FIS 10.2.5), the technology
step's keyed discovery hit (TEC 4.6.4, whose key is derived from the effective seed and
takes no word), the settlement review, the facility step, the polity
and the built-in policies.

### 7.6 The positional hazard (WLD-A5), answered by ADR 0008

7.6.1. Before ADR 0008 one stream served the whole world, and **every draw's value
depended on how many were taken before it**: one extra draw early in a run changed every
later draw (the model of `stream.py`: 98.00% of the later draws of a toy world differ). That
hazard is closed between sites, entities and ticks: with a stream per `(site, tick,
id)` one extra word moves only the later words of its own stream (the same model: 1 of 10 000,
0.01%), and **two runs of one seed that differ in a parameter, a shock or a module are
aligned by construction**: they draw the same value at every key they share and diverge
through state only. A paired comparison is therefore exact in the draws for every knob,
and the variance reduction that the harness measures (`spec-experiments.md` 7.4) is a
number of the state's divergence alone. The estimate of a difference of means from
seed-matched runs is unbiased, aligned or not.

7.6.2. What remains is narrower. **Inside one stream the draws are still positional**: an
intent added to an agent's `Trade` moves that agent's later intents' words in that tick; an
admitted order added to a market moves the later keys of that market in that tick (D7). A
change of an ID (adding a region, an outside node, a gate or a governed jurisdiction
shifts every household's ID, 5.3.2) keys every per-agent stream by another ID, so a
paired comparison across such a change is not aligned, and is made over seeds
(N-162). And a draw that changes a state changes every later decision that reads it:
that divergence is the effect being measured and not a defect.

7.6.3. A stream per site call also means that markets no longer draw from one sequence in
market order (ruling C22): P4.1 could clear its markets concurrently once architecture.md
12.2 has an owner and a lifecycle for the goroutines (open question 7).

## 8. Between ticks

### 8.1 The rule

8.1.1. Every change to the world that does not come from a step acts **between
ticks**: a shock, a slot, an adjustment, an external agent, a relocation, a
directive, a checkpoint. It never acts inside `Step`; the guard of 3.6 refuses it.
The one other writer of a slot is a policy actor at P0.3a (8.2.2), which uses the owners'
internal setters, so that the public call stays a call between ticks only.
Consequences, which hold for every call of 8.2 and 8.3:

1. it takes effect at the first position of the next step that reads it
   (8.2 says which);
2. it is atomic: it validates in full and then applies, or returns an error and
   changes nothing (`spec-regions.md` 7.3.1, `spec-ledger.md` 2.6);
3. it emits its own event at the call, carrying the tick the next step will run,
   before that step's `TickStarted` (6.11); a policy actor's write at P0.3a (8.2.2)
   emits its events after `TickStarted`;
4. its state is saved in a checkpoint by the spec that owns it (12);
5. it is not configuration and is not hashed: a run with shocks and the same
   configuration has the same hash, and the shocks belong to the **sequence of
   calls** that architecture.md 7.4 names beside the seed and the configuration.
   `WithReseed` of a restore (4.1) is also a call of that sequence, and acts at the
   restore and not between ticks;
6. a call that draws takes its draws at the call, from the stream of its row (D12, 7.3),
   keyed by the tick the next step runs.

8.1.2. A call on a failed world returns `ErrWorldFailed` (3.5); during a step,
`ErrInStep` (3.6).

8.1.3. **A move made by a call between ticks is totalled in the next tick.** The
flow totals are cleared at P8.6, after the events and the audit of a tick, and not
at the start of the next (ruling C18). So money or goods that a call creates or
moves between ticks (an external agent's `initial`, a relocation's totals) fall in
the totals of the tick the next step runs, are reported by that tick's
`BoundaryFlow`, and keep `spec-ledger.md` 9.1 (boundary balance = construction
bucket + the totals of every tick since) true when `WithAudit` runs. Before the
first step the same moves go to the construction bucket (5.5.2). A **window of
flows** that a consumer or an indicator reads between two snapshots therefore
begins after the calls it includes: a window opened after such a call holds the
move in its opening snapshot and would count it again in the first tick's flow,
and a call made after the last step is in no flow of the window. The snapshot's
`Pending` (what the next tick's `BoundaryFlow` will include besides its own) is
the correction, subtracted at the opening and added at the closing
(`spec-indicators.md` 7.1.5; `spec-observability.md` 5.3.3).

### 8.2 The slots

A **slot** is a piece of world state that is set from outside, between ticks,
over a configured value, and read by a position of the tick. The mechanism and the
arithmetic belong to the spec that owns it; the public call, its error values and
its bounds are INT's (N-091). This table does not name the calls.

| Slot | Mechanism owner | Read in the tick | Event at the call | Saved by |
|------|-----------------|------------------|-------------------|----------|
| Yield shock: a multiplier by (region, recipe), 0 to `MaxYieldBP`, default 10 000 | PRO 6.8 | P1.2 (the cost anchor), P2.5 (`k0`), P2.7 (the yield read at the start of a job; a running job keeps its pending output) | `YieldShocked`, one per changed key (`spec-observability.md` 4.4.4) | PRO 15: sorted by region, recipe |
| Edge status: closed, or a length factor, by edge | REG 7 | the graph tables and the lag table are rebuilt **at the call** (REG 4.4.4, INF 7.2); read at P3.2 and P3.3 (route, travel time), P3.5 (a closed edge cuts the outside) and P7.2 (a journey's distance); shipments in flight keep their arrival tick | `EdgeStatusChanged`, in `EdgeID` order, after the tables are in place | REG 11: the non-default edges |
| Transit factor: by (from, to, mode), 0 to `MaxTransitFactorBP`, default 10 000 | TRN 10.3 | P3.2 and P3.3, at dispatch; new dispatches only | `TransitFactorChanged`, one per changed key | TRN 14: the non-default factors |
| Flow scale: by money flow or fiscal flow, 0 to `MaxFlowScale`, default 10 000 | MON 10.4; FIS 13.1 | P1.7 (money flows), P1.7b (taxes, remittances, payouts), P3.2(c) and P3.3 (tolls), P3.4a (the treasuries' orders), P4.1a (duties) | `MoneyFlowScaled` (cause `consumer`, or `policy` at P0.3a) | MON 14; FIS 14 |
| Rate scale: by region, 0 to `MaxShockBP`, default 10 000 | CRD 13.1 | P3.3b (the effective rate of a new loan) | `CreditRateScaled`, one per changed key | CRD 17 |
| Rent scale: by (region, resource), 0 to `MaxShockBP`, default 10 000 | LND 5.2 | P1.2, P2.3, P2.6, P2.7 | `RentScaled`, one per changed key | LND 14 |
| Route risk scale: by (`From`, `To`, `Mode`), 0 to `MaxShockBP`, default 10 000 | RSK 9 | P3.1a and the record of a dispatch (P3.2(c), P3.3) | `RouteRiskScaled`, one per changed key | RSK 15 |
| Store risk scale: by (region, store class), 0 to `MaxShockBP`, default 10 000 | RSK 9 | P8.1a | `StoreRiskScaled`, one per changed key | RSK 15 |
| Legal level: by control, a cap and a floor, 0 to `MaxBasisPoints` (a rate limit: 0 to `MaxCreditRateBP`) | ORG 6, 7.2 | the admission of P3.2, P3.3, P3.3b, P3.4 and P3.5; **the resolved ranges are rebuilt at the call** | `LegalLevelSet`, one per changed key in ascending `ControlID` | ORG 12 (not hashed) |
| Right suspension: by right, a bool, default false | ORG 5.4.3, 7.2 | P2.5 (and the formation of a hire), P3.2(c), P7.1(a); **the resolved rights are rebuilt at the call** | `RightSuspensionSet`, one per changed key in ascending `RightID` | ORG 12 (not hashed) |
| Birth scale and death scale: by region, default the region's configured value (10 000 with no row) | POP 3.6, 6.6 | P7.1(b) | `PopulationScaled`, one per changed key, naming which scale | POP 11 |
| Endowment capacity scale and regeneration scale: by (region, resource), 0 to `MaxShockBP` | RES 6.3.1 | the capacity scale **at the call** (the effective capacity is recomputed and a stock above it is clamped, the excess lost); the regeneration scale at P8.3 | `EndowmentAdjusted` (kind `shock`) | RES 15 |
| Endowment adjustment: a signed one-off delta to a stock | RES 6.3.2 | at the call (the stock changes now) | `EndowmentAdjusted` (kind `adjust`) | RES 15: the stock |
| Outside price shock: a factor by (node, commodity), 1 to `MaxShockBP`, default 10 000 | EXT 6.5 | P1.8 and the quotes of P3.5 | `OutsideShocked` | EXT 16 (not hashed) |
| Directive: the mailbox of an external agent, applied to the next tick, discarded unless `Sticky` | AGT 10 | P1.9 | none | AGT 15: the directive is state |
| Policy directive: the mailbox of a `directed` policy actor, a list of lever settings written by `DirectPolicy`, read at the actor's decision and cleared after it (the Directive row is its model) | PLC 8.4 | P0.3a, when the actor is due | none at the call; the slot events of its application at P0.3a (cause `policy`) | PLC 13: the mailbox is state |
| Technology advance: a signed delta of the advance of a (region, field) cell, clamped to `[0, CapBP]` | TEC 7.1 | at the call (the gain is re-derived now; modelled on Endowment adjustment) | `TechnologyAdjusted`, one per changed key | TEC 12 (not hashed, not listed by `Slots()`) |

8.2.1. **A new slot** gets a row here in the change that introduces it, with its
read position, and says whether it acts at the call or at a position. No slot
edits a configured value (`spec-regions.md` 5.3.6, `spec-resources.md` 6.3). The
cases of "acts at the call" (the graph tables, the endowment's capacity, the effect tables
of the facilities, CAP 11, and the legal and right tables of the institutions module, ORG 7.2)
exist because a derived table must be consistent before the next step, and the call is
the only place that can rebuild it without an observer seeing a half-built world.

8.2.2. **A policy actor is the second writer of three kinds of slot (ruling of F8; PLC 7.1.3).**
At P0.3a a due policy actor writes the slots of kinds `flow`, `edge` and `transit` (the flow
scale, the edge status and the transit factor) through the owners' internal setters, with
the cause `policy` on their events; no other slot has a second writer. The write is
indistinguishable in effect from a call made between the previous `Step` and this one: P0.3a
is the entry of the step, before any position that reads a slot, and no reservation is open
(6.1.6). The last write wins, a consumer's call before the step and an actor's at P0.3a
in that order. The public setters keep returning `ErrInStep` (3.6.3), and a call between
ticks never takes effect through an actor. An actor writes no `SetRightSuspensions`,
`AdjustTechnology`, `AdjustFacilities` or `TransferTitles` (`spec-policy.md` 6.3.3).

### 8.3 Calls that act on agents

| Call (public name: INT) | What it does | Mechanism | Events |
|-------------------------|--------------|-----------|--------|
| Add an external agent | creates an agent of weight 1 with the next ID: an account, its money and goods by `initial`, beliefs, and D12's two creation draws | AGT 3.5.4, 10 | `AgentSpawned` |
| Remove an agent (an external agent, mortal or not) | the removal mechanics of `spec-agents.md` 3.6: job abandoned, trainee relationship ended, journey dropped, contracts of a deliverer settled, money and goods to the estate | AGT 3.6; SPN 9 | in the order of 6.11 item 8 |
| Relocate an agent (the guarded path; any agent that is not a pseudo-agent, `spec-integration.md` 4.5.1) | the account's region changes if no reservation is open and no `region_bound` refusal applies; a journey the agent is on is cancelled inside the call, with the move | REG 6; MIG 8.4.1 | `AgentRelocated` |
| Direct an agent | writes the mailbox | AGT 10 | none |
| Add a store to an agent | gives the agent one more store of a class; refused for a class the region's settlement class lacks (`ErrStoreLocked`), an agent with no base room (`ErrNotLimited`) and an agent away or on a journey (`ErrAgentAway`) | STO 3.6.2 | `StoreChanged` (`added`, cause `consumer`) |
| Remove a store from an agent | takes one store of a class; refused when the load plus the pledge would no longer fit (`ErrNoRoom`) or a store facility is installed in it (`ErrStoreInUse`) | STO 3.6.2 | `StoreChanged` (`removed`, cause `consumer`) |
| Add facilities | creates facilities of a class in a region, for an owner that is a household agent, an external agent or the estate of the region, at a given or the full condition | CAP 11.1 | `FacilityChanged` (`added`), `EndowmentAdjusted` (kind `facility`) |
| Adjust facilities | changes the condition of one facility by a signed delta, clamped to `[0, Durability]`; a facility taken to 0 is ruined, and a store facility takes its store with it | CAP 11.2 | `FacilityChanged`, `EndowmentAdjusted` (kind `facility`), `StoreChanged` (cause `facility`) |
| Transfer facilities | gives a facility to another owner (a live household or external agent, or the estate of its region); moves no money | CAP 11.3 | `FacilityChanged` (`transferred`) |
| Transfer titles | moves a share of an endowment's title between holders, or between the commons and a holder, at an optional price (the internal reason `title_sale`) | LND 6.3 | `TitleTransferred` (cause `consumer`) |
| Forgive claims | ends every open claim whose debtor's region and role match a filter, and moves no money | CRD 13.2 | `LoanEnded` (cause `forgiven`) |

8.3.1. A call that creates or removes an agent between ticks changes the agent ID
space (5.3): the next ID, the rotation lists and the review rotations of every
later step follow from it, which is why the call is part of the sequence of calls
that fixes a run.

### 8.4 Where the checkpoint is taken

8.4.1. **Between ticks, on a healthy world, and nowhere else** (ADR 0002 open
point 4, answered here and in `spec-checkpoint.md` 3.2; `spec-ledger.md` 6.5). At that boundary: no reservation is open, every
counterparty account is empty (EXT 5.3.3), no shipment is half dispatched and no
job is half started, the flow totals and every per-tick scratch of 0.2 are not
state, `Tick()` is the tick the next step runs. The construction bucket is saved
**only while the world is in its construction phase** (5.5.3; `spec-checkpoint.md`
4.4): a world that was never restored keeps it, readable, for the run; a world
restored from a checkpoint taken after its first step has none, and one taken
before it has the saved bucket. The one exception to "the flow totals are not
state" is the totals that calls between ticks added since the last P8.6 (8.1.3):
they belong to the next tick, so a checkpoint taken after such a call must save
them (12).

8.4.2. A checkpoint of a failed world, or of a call made from inside a step, is
refused (3.5, 3.6). A world that has been constructed and not stepped may be
checkpointed: it is at the tick 0 boundary, still in its construction phase, and
the checkpoint keeps the flag and the bucket (5.5.3).

8.4.3. A restored world continues as an uninterrupted one: the same draws, events
and results (FR-WLD-063). What the restore needs from the world is 12.

## 9. Decisions at different rates

The roadmap's stability duty of this spec (D25 of the register): trade runs every
tick, and price, hiring, occupation and migration decisions run at slower rates
that are **data**. This section collects every cadence of the written specs in one
table and states the rules that make them one mechanism.

### 9.1 The devices

A rate is applied by one of four devices. A spec that adds a rate uses one of them
and says which; it does not invent a fifth without a row here.

| Device | What it is | Draw? | Where it appears |
|--------|-----------|-------|------------------|
| **Rotation** | An agent acts on the ticks where `(tick + bucket(id, P)) mod P = 0`, with `P` a period in ticks and `bucket(id, P) = (Weyl2(id) * P) >> 32` (9.1.1). The load is `agents / P` a tick, spread to within a few agents whatever the layout of the IDs for a group of about a thousand agents or more; a small group is looser (9.1.1) | no | SPN 6.2, MIG 5.1, PLC 7.2 |
| **Carried remainder** | A quantity stated per period or per year is applied each tick as `floor((carry + amount) / D)` with the remainder kept, `D` containing the period or `TicksPerYear`. Exact over any run, no lump, any divisor | no | MON 6.1, POP 3.8, DEM 4.4, RES 4.3, EXT 7.1 |
| **Window** | A memory of the last `N` ticks (a ring) that a decision reads. It is not a cadence: the decision that reads it still runs when it is asked | no | BEL 7, WEL 2.3, INF 4.3 |
| **Gate** | A probability per outcome: the decision runs for the outcome with probability `p`. Asynchronous adjustment at a random rate | one word per outcome | BEL 5.2 |

9.1.1. **The Weyl value and the bucket (ruling C14).** `Weyl32(id) = (id * 2 654 435 769) mod 2^32` (POP 3.8.3) is the fixed per-agent spread of the library for **carries and margins**: the death carry, the migration margin, the merchant's fee share and the saver's hurdle spread use it, and one helper in `internal/ordering` (N-019, wave 4) carries one test of its evenness. It draws nothing and adds no creation draw. A rotation does **not** use it. A rotation uses `bucket(id, P) = (Weyl2(id) * P) >> 32`, a value in `[0, P)`, where `Weyl2(id) = (id * 0x6A09E667) mod 2^32` is a **second Kronecker sequence** with its own multiplier (the fractional part of the square root of 2, the first constant of SHA-256). The rule behind it: **two spreads that act on one decision must not share a value.** MIG's margin `MoveGain + Weyl32(id) * MoveWidth / 2^32` and a rotation bucket taken from the same `Weyl32` are monotone in one number, so the households that meet the margin review on the same few ticks of the period: in the model of 18 (`cadence.py`; 1 000 consecutive IDs, a period of 30, the 400 households that meet a margin of 3 000) the Weyl32 bucket puts them on 12 ticks of 30, with 18 empty and up to 34 on one, against 11 to 15 on every tick with the rule bucket and 12 to 15 with raw IDs. The model also shows why the raw ID is not used: with 1 000 agents and a period of 30, raw IDs spaced by 10 (a role created in a regular interleave with two others) put 333, 333 and 334 reviews on three ticks of 30 and none on the other 27. The rule bucket stays at 30 to 37 on a tick against a mean of 33.3 (98 to 103 against 100 at a period of 10) for every stride of 1 to 31 with IDs from 100; over the starting IDs 0, 100 and 12345 it is 24 to 39 at a period of 30 and 94 to 113 at 10 for 1 000 agents (bounds 40 and 120), and 152 to 182 and 485 to 511 for 5 000. That is a property of a large group: with 200 agents and a period of 30 it reaches 0 to 16 against a mean of 6.7 and a bound of 8 at stride 13 (the Weyl32 bucket is as bad at stride 29), so **no bound is claimed below about a thousand agents**. `Weyl2` is `Weyl32 * c mod 2^32` with `c` fixed, so it is a function of `Weyl32`: the rule "must not share a value" is met **operationally**, not by independence (Spearman rank correlation 0.008 over 1 000 consecutive IDs, and the margin households fall on every tick), and the model is the check. An independent hash of the ID was considered and not adopted: it is binomial, 17 to 55 a tick at the same size (`cadence.out`). The written specs say `(tick + AgentID) mod P`; the edit is one expression (22).

### 9.2 The table

`TicksPerYear` is derived from the calendar (`spec-calendar.md` 3.2). All defaults
are the owners' placeholders (N-015, N-027); none is repeated as a claim here.

| Cadence | Parameter | Owner | Default | Position | How the tick counter selects it |
|---------|-----------|-------|---------|----------|---------------------------------|
| Trade, clearing, belief revision | none: every tick | AGT, CLR, BEL | | P3.2, P4.1, P5.4 | every tick |
| A job | `Duration` of the recipe, ticks | PRO 5 | required | P2.1, P2.7 | a job of `d` started in `s` completes in `s + d - 1`; a free agent decides at every tick it is free |
| Belief adjustment | `BeliefAdjustProb`, basis points | BEL 5.2 | 5 000 | P5.4 | gate: one word per admitted order |
| Price window | `PriceWindow`, ticks of trading | BEL 7 | 15 | P5.1 | ring of the last `N` ticks in which units traded |
| Fresh price | `BeliefMaxAge`, ticks | BEL 5.4 | 3 | P5.4 | compared with the price age of the report |
| Welfare window | `WelfareWindow`, and `WindowCycles` with the effective `T = max(WelfareWindow, WindowCycles * longest Duration)` | WEL 2.3 | 30; 2 | P6.3 (push), read at P1.6, P2.5 and P7.1, P7.2 | ring of the last `T` records |
| Occupation review | `ReviewTicks` | SPN 6.2 | 10 | P7.1(c) | rotation: `(t + bucket(id, ReviewTicks)) mod ReviewTicks = 0` |
| Role cooldown | `RoleCooldown`, ticks | SPN 6.2 | 30 | P7.1(c) | `t - RoleSince >= RoleCooldown` |
| Migration review | `MoveReviewTicks` | MIG 5.1 | 30 | P7.2 | rotation: `(t + bucket(id, MoveReviewTicks)) mod MoveReviewTicks = 0` |
| Settling | `MoveCooldown`, ticks | MIG 5.2 | 60 | P7.2 | `t - RoleSince >=` the longer of it and the agent's window |
| A want | `Period`, ticks, per want | DEM 4.4 | 1 | P3.2, P6.2 | carried remainder per household and want, started at `(index * 7 919 + position * 2 777) mod 10 000` |
| A money flow | `Per` (`tick` or `year`), `Start`, `Until` | MON 4.2, 4.4, 6.1 | per flow | P1.7 | carried remainder over `P * 10^4`; `Start` and `Until` compared with `t` |
| Births and deaths | annual rates by schedule | POP 3.8 | by schedule | P7.1(b) | carried remainder over `10 000 * TicksPerYear`; a death carry starts at the Weyl offset |
| Regeneration | `RatePerYear` or `RateBP` per year, seasonal profile | RES 4.3, 4.5 | per resource | P8.3 | carried remainder per endowment; the season is `Calendar.At(t).Season` |
| The outside's capacity | `Capacity` per tick, window by season | EXT 7.1, 7.2 | per side | P3.5 | carried remainder per side; the window is a table by season |
| The outside's walk | `WalkStepBP`, `WalkRevertBP` | EXT 6.4 | per line | P1.8 | one word per walking line, every tick |
| News | `InfoSpeed`, `RumourMaxAge` | INF 4, 5 | speed required; 60 | P3.2(a), P5.2, P7.2 | the lag `ceil(distance / speed)` ticks; a ring of records |
| A shipment | `Speed` | TRN 4 | required | P1.3 | `Arrives = dispatched + k`, delivered at the first position of that tick |
| A migration journey | `MoveSpeed` | MIG 6.4, 8.2 | required | P1.5 | `Arrives = tick + 1 + k` for a departure at P7.2 of `tick` (FR-MIG-026), arriving at the first position of that tick |
| Relief and listing | `PoolRelease`, `EstateMarkdown`, basis points a tick | EST 2.6, 2.7.3 | 100; 500 | P1.6, P5.3 | every tick, no cadence |
| The own-use mode | `EnterTicks`, `ExitTicks`, `RetryJobs` | SUB 5.3, 5.5 | 2; 3; 10 | P2.4 | thresholds on the household's own state, flag stepped every tick |
| A traineeship | `TrainingTicks`, `TeachingLoad` | SKL 4.1, 4.2 | per role | P2.2, P7.1(a) | a per-trainer accumulator and the ticks served |
| Ageing | `ShelfLife`, `Durability` | GDS 4, LED 5.3 | per commodity | P8.1 | `Born + ShelfLife - 1 <= t` |
| Seasons and the year | the calendar | CAL | Earth | P1.1 | `Calendar.At(t)`, derived |
| The contract round | every tick; `MaxSourceAge` | MER 9.4, 5.2 | 2 | P3.3 | every tick; the bound compares with `SourceAge` |
| The labour round | none: every tick | EMP 5.1 | | P2.5a | every tick, no cadence |
| The credit round and the settlement of claims | none: every tick | CRD 12.3 | | P3.3b, P1.7a | every tick, no rate, no gate |
| The membership round | none: every tick | ORG 4 | | P1.9a | every tick, no cadence |
| The posted rent | `RentStepBP`, `RentFloorBP`, `RentCeilingBP` | LND 5.5 | 0; 0; 20 000 | P8.3a | every tick; a bounded step on the counters of that tick |
| A settlement class | `Dwell`, `BandBP` | SET 3.4 | 30; 1 000 | P7.2a | a change applies when `t - Since >= Dwell` |
| A policy actor's decision | `Every`, ticks | PLC 7.2 | required | P0.3a | rotation: `(t + bucket(a, Every)) mod Every = 0`, `a` the `ActorID` |

### 9.3 A cadence that does not divide `TicksPerYear`

9.3.1. **It is legal and is never rounded.** A cadence of `P` ticks is applied by
rotation, which refers to `t` and `P` only, so it needs no relation to the year.
With `TicksPerYear` 365 and `P` 30, an agent acts 12 or 13 times a year, and the
tick of its first action in a year moves by `TicksPerYear mod P` (5) each year. A
cadence that does divide the year would put every agent's actions on the same
days of the year, year after year, so that any seasonal forcing (STB-L23 is the
register's entry for it) would meet every review at one phase of its cycle; one
that does not divide the year moves that phase by `TicksPerYear mod P` a year.
This is a property of the device, not a claim of the register and not a choice of
this spec.

9.3.2. **A per-year quantity** (a flow `Per` year, the annual rates of POP, `RatePerYear`
of RES) is applied by the carried remainder, whose denominator holds
`TicksPerYear`: any `TicksPerYear` is exact, and over a year the total is the
annual amount to within one unit of carry. A quantity per year is never converted
to an integer number of ticks.

9.3.3. **Calendar units are not cadence units.** A cadence is in ticks. A pack that
wants a monthly review states the number of ticks of its month in its own calendar
(`spec-calendar.md` 6.1: durations are in ticks and are not rescaled by the
calendar). No cadence is aligned to a season boundary by the engine; seasons act
through the tables keyed by `Calendar.At(t).Season`.

9.3.4. **Two cadences with a common factor** (a review of 30 ticks and a window of
30) are applied with buckets of their own period, so they are not the same function
of the ID: in the model, the 30-tick review tick of an agent is also a 10-tick review
tick for 13.3% of agents (4 in 30; raw IDs would give 100%). Where they do coincide
an agent's occupation review and migration review fall on one tick, serially, with
the settling rule between them. That, and a group smaller than its period (where the
spread is over fewer ticks than the period), is registered as WLD-A4.

### 9.4 No rate from an aggregate

9.4.1. **Rule.** No cadence, period, window length, gate probability or share of
the library is a function of a **measured aggregate** of the world: a price level,
a money supply, a count of households or agents, an unemployment or welfare
average, a flow total. A rate comes from the configuration, from a slot an
outsider set between ticks (8.2), or from a derivation over configuration alone,
such as the effective welfare window `max(WelfareWindow, WindowCycles * longest
Duration)` or the lag of a rumour from the distance table. "Macro is measured,
never set" (roadmap; ADR 0001 decision 7; `spec-stability.md` 5.1.3).

   **Three exceptions, each owned (ADR 0006, which narrows ADR 0001 decision 7).** A derived
   structural rank, a step function of the household count of one region with a band and a
   dwell (the settlement class of `spec-settlements.md`), may gate an entry, a right or a
   number of places; the household count of one region may gate a permission, a place or a
   structure and is read against a body's `Places` by the institutions module
   (`spec-institutions.md` 3.4.1); and a quantity local to the entity a price or rent
   belongs to (the units unserved or idle at one endowment, a title's share: `spec-land.md`)
   is not an aggregate. None of them reads an aggregate into a rate, price, wage, share,
   probability, period or window, none is read by an agent's view, and a new reader is a new
   ADR.

9.4.2. **What is allowed.** A decision reads what a household can know: its own
state, what its market publishes (a price, the price window, the observed wage of
its region and role), the structure of the world and rumoured prices. Reading a
published price to decide is a decision; changing **how often** anyone decides
because of an aggregate is not. A controller that wants a rate to follow a measure
is a policy actor and plugs in through a slot (ADR 0001 decision 4, MON 10.4); the
library contains one only as the policy actors of `spec-policy.md`, which set a slot at P0.3a
(8.2.2) and never a cadence.

9.4.3. **A cadence is configuration.** It is in the hash and constant for a run.
No slot of 8.2 changes a cadence. A blackout that slows the news would be a slot
(INF open question 6) and would change a speed, not a period.

## 10. Fairness and order effects

### 10.1 What neutralises an order

| Device | Where | What it does |
|--------|-------|--------------|
| The rotation of starts | P2.1, P2.5, P2.7 (6.1.4) | the agent that goes first moves one place a tick, so no ID has the first claim on a shared input |
| The tie-break key of a market | P4.1 | every order draws a key, so the position in the list gives no priority at an equal price |
| The ration of a scarce endowment | P2.6 | proportional by weight; leftovers by the largest fractional part, then a stateless key of (tick, agent) that changes every tick |
| The contract key | P3.3 | equal limits are ordered by a drawn key |
| Relief and the money flows' dues | P1.6, P1.7 | each recipient's amount depends on its own state and on totals, never on its place in a list |
| Proportional deaths and weights | P7.1 | per-agent carries with a Weyl start, so no agent is first |

10.1.1. **The rule for new ties.** A tie between candidates of equal merit is
broken by the stateless key `K(t, id)` of RES 5.3.3 (the SplitMix64 finaliser of
`t * γ + id + γ`, γ = `0x9E3779B97F4A7C15`, in modular `uint64` arithmetic, by design: the
wrap is the hash, not an overflow, architecture.md 3.5; the exact form and the vectors are
7.1.7 and 7.1.5 (c)), smaller first, then by ID. It takes no draw, needs
no state, and changes every tick, so a lower ID is never systematically first. In the model of 18 (`rotation.py`: 100 tied candidates, a cap of 5, 1 000 ticks, 50 admissions each in expectation) the lower ID wins all 5 000 admissions for the first five IDs and none for the rest, the key gives each candidate 36 to 73, and a rotation of the starting index gives exactly 50. A
tie broken by "the lower ID" in a new spec needs a reason written in it.

### 10.2 What remains

The remaining places where an ID or an order gives a systematic advantage, none of
which this spec changes (they belong to the owners; each is a hazard with a
remedy named, and an edit is **proposed**, not owed):

| # | Place | The order | Who is systematically first | Remedy proposed |
|---|-------|-----------|-----------------------------|-----------------|
| O1 | Entry budget and births, SPN 7.3, POP 4.8 | roles in role order draw on the pool budget | the roles with earlier IDs, when the budget binds | serve the roles from a starting role that rotates with the tick, or share the budget in proportion to the asks, as the ration does |
| O2 | Role-change and entry caps, SPN 6.3; admission of movers, MIG 7.1 | by need, then by agent ID | the lower ID among equal need | break the tie by `K(t, id)` (10.1.1) |
| O3 | Ladder candidates of equal privation and gap, SPN 5, 6.3 | then by ID | the lower ID | as O2 |
| O4 | A flow's total limit, MON 5.4 | recipients in (region, account) order; the amount is cut when the total would pass the limit, then the flow ends | the earlier regions and accounts of the tick the limit binds, the later get nothing | cut pro rata, or rotate the starting region; this is the only place where a limit is first-come |
| O5 | The contract round's lanes, MER 9.4, and `MaxOpen` | ascending (destination, commodity) | a payer's funds go to the earlier lanes (`payer_funds`), and contracts formed up to `MaxOpen` go to the earlier lanes | rotate the starting lane by `t mod lanes` |
| O6 | A trainee's upkeep out of a trainer's account (P6.1) | at the trainee's ID position | a trainee whose ID comes first eats before the trainer's own use if the stock is short | rare (the trainer's targets include its trainees' upkeep, `spec-agents.md` 5.7); accept and count |
| O7 | The ration's leftovers, RES 5.3.4 | the fractional part, the same order every tick for a fixed mix of request sizes | agents asking for many executions | a carried remainder per agent (RES open question 4) |
| O8 | The input order of a market | agents by ID, estates, treasuries, counterparties | none: the keys of P4.1 make it irrelevant | none |
| O9 | A role ceiling that binds, SET 6.4.5 | the entrants in the order of the spawner's plan: need, then agent ID | the lower ID among equal need | break the tie by `K(t, id)` (10.1.1), the remedy of O2 and O3; the counter is `RoleCeiling` |
| O10 | The places of a body, ORG 4.3 and 4.5 | by seniority, then the stateless key of (tick, agent), then agent ID; the headroom is shared with the class limit and the entry budget among the first-come places above | the household whose role has been held longest, when the places bind | none beyond the keyed tie ORG already takes (10.1.1); the seniority order is the rule, and a household that waits is counted in `MembershipChanged` |

10.2.1. O1 to O5, O9 and O10 are registered as one hazard, WLD-A2 (17). They are
**biases, not loops**: they pick who is served when a cap binds, and they do not
feed back, unless an agent's chance to be served changes its later welfare, in
which case the lower ID or the earlier region is selected for survival, and a
statistic that groups by region or role would show it as a difference between
regions that the data did not put there. IND and STY must not read a regional
difference as real before checking whether a cap or a limit was binding there
(the counters exist: `EntryLimited`, `MoneyFlowLimited`, `ContractRound`). IND
honours this with the warning `Binding` on every measure that compares regions or
roles (`spec-indicators.md` 8.7), and its binding counters also count the caps of
`PopulationChanged` and `MigrationReviewed`.

## 11. Cost

11.1. **The shape of a tick** in terms of `A` agents (not households), `R`
regions, `C` commodities, `M` markets (at most `R * C`), `T_a` the tradable set of
an agent, `O` the admitted orders, `L` lots, `S` shipments in flight, `K`
contracts open, `E` endowments, `X` counterparties, `I` informed agents and `V` the
candidates of a view (at most `MaxRumoursPerView`). Costs are counts, not timings
(N-015), and the library measures no wall-clock time (architecture.md 7.3).

| Position | Cost shape | Dominant term of |
|----------|-----------|------------------|
| P0.2a | `O(rows of the bucket)`, once in a run, at most `R * reasons * C` | regions, reasons, commodities |
| P0.3a | `O(due actors * (levers + readings))`, plus a consumer's own `Decide` | policy actors |
| P1.3a, P8.1b | `O(arrived shipments with a cover + underwriters paid)`; `O(store owners struck + underwriters paid)` | the arrivals and the strikes, not the owners |
| P1.2 | `O(R * recipes)` | regions and recipes |
| P1.3, P1.4, P1.5 | proportional to the arrivals, settlements and journeys of the tick | the pipeline |
| P1.6 | `O(A)` for the agents of regions whose pool is positive | agents |
| P1.7 | `O(recipients of each flow)`, at most `A * flows` | flows |
| P1.7a | `O(debtors visited + their claims)`: the visited are those with a claim due or overdue this tick | claims falling due and overdue, not open claims |
| P1.7b | `O(payers and recipients of each fiscal flow)`, at most `A * fiscal flows` | fiscal flows |
| P1.8, P1.9 | `O(lines)`, `O(external agents)` | the outside |
| P1.9a | `O(households of the roles of the bodies)` for the lapse pass, `O(R * bodies)` for the admission | households of a body's roles |
| P2.1, P2.4, P2.5, P2.7 | `O(A)`; `Work` is `O(recipes of the role)` per agent | agents |
| P2.3, P2.6 | `O(E)`; `O(requests log requests)` only for an endowment that is short | endowments |
| P2.5a | `O(offers + vacancies)` per region and role pair for the sort and the matching, `O(hires formed)` | agents with an `Offer` or a `Hire` |
| P2.8, P2.8a | `O(regions * roles)`, from counts the earlier positions kept | regions and roles |
| P2.8b | `O(regions * roles)`, from the counts of the labour round | regions and roles |
| P3.1a | `O(common route rows with P > 0)` | route rows |
| P3.2 | `O(A * (c_0 + T_a * c_t))`, plus `O(I * V)` for the rumours | the cost model of `spec-scale.md` 6.1 |
| P3.3 | `O(offers log offers + takes)` | contracts |
| P3.3b, P3.3c | `O(Q + F)` per lane for the sort of requests and offers by rate and key, `O(fills)` for the matching; `O(R + O)` per lane and `O(candidates * scanned offers)` for the covers | agents with a credit or a cover block |
| P3.4, P3.5 | `O(estate lots)`, `O(X * lines)` | the estates, the outside |
| P3.4a | `O(standing orders of each authority)`, one order per lot and side | treasuries |
| P3.6a, P3.6b, P3.6c | `O(regions * roles)` from counts the earlier positions kept; `O(fiscal flows * regions)` | regions and roles |
| P4.1 | `O(sum over markets of n_m log n_m)` | orders |
| P4.1a | `O(trades of the tick * duties)` | trades |
| P4.2 | `O(X * lines)` | the outside |
| P5.1, P5.2 | `O(M)` | markets |
| P5.4 | `O(O)` | orders |
| P6.1, P6.2, P6.3 | `O(A * items)` | agents |
| P7.1 | `O(agents of the region)` for the view; the plan is `O(changes)` | agents |
| P7.2 | `O(A / MoveReviewTicks * V)` a tick, amortised | agents, candidates |
| P7.2a, P7.2b | `O(R)` with a counter per region and `O(A)` once with a scan; `O(R * (bodies + rights + controls))` | regions, bodies |
| P8.1 | `O(accounts with a lot due + lots due)`, by the earliest-expiry hint | what expires |
| P8.1a | `O(common store rows + stores of the limited owners)` | store rows |
| P8.2, P8.3 | `O(R * reasons * C)`, `O(E)` | regions, resources |
| P8.3a, P8.3b, P8.3c | `O(E)` and its tenants; `O(facilities)`; `O(regions * fields)` | endowments, facilities, fields |

11.2. **Against the budget of `spec-scale.md` 6.** The terms that grow with the
number of agents are P2 (`O(A)`), P3.2 (`O(A * (c_0 + T_a * c_t))`), P6 and P7.1;
the term that is `n log n` is P4.1; every other position is bounded by regions,
commodities, endowments, nodes or the pipeline. **Nothing in a tick is quadratic
in the number of agents**, and nothing in it depends on the number of households beyond the
weights that a tax, a lapse pass or a count of a class reads.
The model of `spec-scale.md` 6.1 (`c_0 + |T_a| c_t` per agent, and the clearing)
therefore stands as the per-tick budget: this spec adds no term to it that is
proportional to agents beyond those of the cost model (P1.6, P1.7, P1.7b, P1.9a and P7.2 are
`O(A)` shapes of their own: relief reads every agent of a region with a pool, a
flow and a tax every recipient or payer, the membership lapse pass every household of a body's
roles, the migration pass the agents on their review tick), and the
only added work is `O(R * recipes)` (the cost anchors), `O(lines)` and the position
scratch.

11.3. **No allocation with the no-op observer.** The positions allocate nothing
beyond scratch held by the world and reused across ticks, and no event is
constructed when no observer was supplied (observability-and-logging.md 4.5, 8.2);
an allocation test over a full `Step` of the reference chain enforces a baseline
constant (NFR-WLD-001).

## 12. What the world contributes to a checkpoint

CHK owns the encoding, the version, the restore and its errors. This is what the
world adds, as plain data in a fixed order (ADR 0002 decisions 3 and 5):

| State | Why it cannot be derived |
|-------|--------------------------|
| The tick counter, `int64` | it is the clock; it is written in the checkpoint's header, so that `Info` needs no section (`spec-checkpoint.md` 5.4.3) |
| The seed, `uint64`: the effective seed (7.1.1) | it is the one random datum of the world and every stream is a function of it; it is written in the checkpoint's header (`spec-checkpoint.md` 5.4), so that no PCG state, and no position in any stream, is saved (ADR 0008; the section of CHK 4.2 that still lists a PCG state is stale under ADR 0008, see N-214) |
| The next agent ID, `int64` | an agent's ID is never reused; the highest ID of the live agents is not enough if the highest was removed, and the closed accounts that would imply it may one day be compacted (`spec-ledger.md` open question 6) |
| Whether the world is still in its construction phase (a flag) | a world checkpointed before its first `Step` is restored in construction, with its bucket, and a later call is totalled in it as on the original (5.5.3; `spec-checkpoint.md` 4.4, 6.7.2) |

The configuration hash is stored by CHK and is not world state. The world's other
contributions are those of the specs that own them (section 8.2 lists the slots).

12.1. **Derived on restore, never saved:** the moment (`Calendar.At`) and
`TicksPerYear`, the rotation positions and counts, the graph and lag tables, the
cost anchors, the module scratch of 0.2, the flow totals, the earliest-expiry hints
and the free lists of Transit and Escrow accounts (their owners'), and the room, load and
pledge of a limited account (STO 4: derived from the stores, the lots and the jobs, with no new
saved world state), the effect tables of the facilities and the resolved legal and right
tables, which the call that changed them rebuilt (8.2.1). Not saved
either: the failed flag (a failed world is not checkpointed), the in-step guard,
the observers, the logger and the registrations of extensions. The registrations
are checked by **name** against the hash (a strategy, a spawner, a clearing and a policy); the
**state of an extension** that implements the optional save and load pair is saved by CHK as
ADR 0002 decision 7 says. The flow totals are derived (cleared at P8.6) with one exception: the totals
that calls between ticks added since the last P8.6 (the **pending totals**) are
state at a boundary and are saved (8.1.3, open question 16: answered, CHK 4.4). The
construction bucket is saved only while the world is in its construction phase
(5.5.3, 8.4.1).

12.2. **What a restore needs from the world** is a way to build the derived tables
and install the saved state without running steps 7 and 8 of 5.1 (the audit of
step 9 and the events of step 10 are replaced by the restore's own, below): the
validated configuration and options (steps 1 to 6 of 5.1, with the seed of the header as the seed, or the reseeded one, 12.3), then the ledger, the
agents and every module's state from the checkpoint, then the tick counter, the
next agent ID and the construction flag; **no random state is installed and no word is
taken**. A world restored from a
checkpoint taken after its first `Step` is not in construction; one taken before it
is, and keeps its bucket (5.5.3). It emits no `WorldCreated`; its event is
`WorldRestored`, the first event to the observers given to the restore
(`spec-checkpoint.md` 6.7.3, `spec-observability.md` 4.4.5). The restore runs the
audit whether or not `WithAudit` was given (`spec-checkpoint.md` 6.2.2).

12.3. **The seed.** `WithSeed` is not hashed, and the checkpoint carries the seed in its
header, so a seed given to a restore has no use. CHK adopts the recommendation of this
spec (`spec-checkpoint.md` 6.4): `WithSeed` is **refused** on a restore (`ErrOptionContext`;
an option that is accepted and ignored is a trap). The seed of a restored world is the
seed of the header, and it is read back as the seed: the header's seed is the effective
seed, not information only (`spec-checkpoint.md` 5.4 still says "informational only", stale
under ADR 0008, see N-214). `WithReseed(salt)` is the one way to give a restored world its
own streams: it replaces the effective seed by `finalise(seed xor finalise(salt))`
(`spec-checkpoint.md` 6.8; 4.1).

## 13. What the world hands to the specs that come after it

Stated as requirements on the world, not as their content.

| To | The world guarantees |
|----|----------------------|
| **OBS** | every event carries `Tick`; the events of a step come in the order of 6.11 and **every** step that emitted `TickStarted` is closed by `TickEnded`, a failed one with `Failed` true, which describes the end of the step and not a completed tick (3.5.6); events are constructed only when an observer was supplied; a snapshot is available only at a boundary on a healthy world (3.7, 8.4); construction events carry `Tick` 0 and come in the order of 5.8 (`AgentSpawned`, then the initial `FacilityChanged`, then the founding `SettlementClassChanged`); a between-ticks call emits its event at the call with the tick of the next step, and a policy actor's write emits its slot events at P0.3a after `TickStarted`, then `PolicyDecided` (6.11 item 1). OBS **adds** what this spec asked for: a `Failed` field on `TickEnded` (3.5.6) and on `RunEnded` (3.4.2), the events `YieldShocked`, `TransitFactorChanged` and `PopulationScaled` for the slots that had none (8.2), the names table on `WorldCreated` (5.8.2), and the construction bucket as a `BoundaryFlow` with `Tick` -1 at the entry of the first step (P0.2a, N-072) |
| **CHK** | the contribution of 12 (the PCG state, the next agent ID and the construction flag with the bucket while in construction; the tick in the header); the boundary of 8.4; a failed world is refused; the register of extension names is in the hash; `WithSeed` is refused by a restore and `WithReseed` is the restore-only option (4.1, 12.3); `Tick()`, `ConfigHash` and `Config` are readable |
| **INT** | every call acts between ticks (8.1), is atomic, emits its event at the call, takes its draws at the call, and is refused with `ErrInStep` or `ErrWorldFailed`; the slots of 8.2 and the calls of 8.3 (add, remove, relocate, direct; add and remove a store; add, adjust and transfer facilities; transfer titles; forgive claims; the cancel of a journey is inside `Relocate`) are the whole list of mechanisms; INT names them and fixes their bounds and errors; a slot never edits a configured value; no "act now" exists inside a step, and the one writer inside a step is a policy actor at P0.3a through the owners' internal setters (8.2.2), which no public call reaches |
| **IND** | indicators are a function of the events and the snapshot at a boundary; the flow totals of a finished tick reach IND through `BoundaryFlow` only; the snapshot holds the pending totals of 8.1.3 and no more; counts of agents and households are separate (the weights); a tick is the finest resolution: no value exists between `TickStarted` and `TickEnded`; the counters that reveal a binding cap (O1 to O5, O9 and O10) are events |
| **STY** | a run is a function of (configuration, seed, sequence of calls); `Run(ctx, n)` runs `n` ticks and leaves a snapshot-able world; two runs with the same inputs are bit-identical across platforms (architecture.md 7.4); a golden test is enforced; a run with an observer equals one without |
| **EXP** | worlds share nothing, so a harness may build many and step them on separate goroutines (3.1.1); `Config()` returns the effective configuration and `ConfigHash` keys a result; a paired comparison is exact, and a measured variance reduction, only while the draw counts are equal (7.6.1); the cost shape of 11 budgets a sweep; a failed world is detectable by `Err()` and is discarded |

Stale under ADR 0008, see N-214: the **CHK** row (12 holds no PCG state and CHK saves the seed alone) and the **EXP** row (a paired comparison is exact in the draws by construction, 7.6; the pairing gain is measured on the state's divergence alone). The rows are not rewritten here: their owners edit them.

## 14. Parameters

This spec adds **one `Config` field**, the name `World.Clearing` (4.2, default
`average`, owned by `spec-clearing.md` 8 as `World.Spawner` is the spawner's), which
makes a registered `Clearing` part of the hash, and no other field, default or
row. The world has no tunable of its own: every cadence and every bound it applies
belongs to the spec in the table of 9.2, and the options of section 4 are not
configuration. A test of the parameter registry (testing.md 6.7) needs the one row
of 22 and the field.

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the
order of the phases and the positions of section 6; the key of a stream, the
constants of the SplitMix64 finaliser and the site constants (7.1.2 to 7.1.4), `K(t, id)`
(7.1.7); the world's draw (7.2.1) and the list of draws with their streams (7.3); the rotation of starts (6.1.4); the Weyl multiplier
`2 654 435 769` and the bucket (9.1.1); the keyed tie-break (10.1.1); the failed
state (3.5) and the guard (3.6); the order of accounts (5.2) and of agent IDs
(5.3); the sub-steps 8a to 8c of construction (5.1.2); the order of events (6.11).

**Not parameters:** the tick counter, the moment, `TicksPerYear` (derived), the
rotation positions, the failed flag, the effective configuration.

## 15. Validation

15.1. **Errors this spec adds** (sentinels at the root, mapped as N-075 asks; each
has a failing-case test, testing.md 3.9):

| Sentinel | When | Where |
|----------|------|-------|
| `ErrUnknownClearing` | `World.Clearing` names a clearing that was not registered | `NewWorld` (3.2.2) |
| `ErrUnknownPolicy` | an actor's `Policy` names a policy that is neither built in nor registered (`spec-policy.md` 4.5.1) | `NewWorld` (3.2.2) |
| `ErrNilExtension` | `WithStrategy`, `WithSpawner`, `WithClearing` or `WithPolicy` was given a nil implementation | the option (4.3) |
| `ErrWorldFailed` | any call that has an error result on a failed world, other than `RelocationRefusal` (the readers with no error result work, 3.5.1) | 3.5 |
| `ErrInStep` | any call on a world whose step is running | 3.6 |
| `ErrOptionContext` | `WithSeed` given to a restore, or `WithReseed` given to `NewWorld` (a sentinel of CHK, used by the options of 4.1) | the option (4.3) |

`ErrInvalidTrade` and `ErrAuditFailed` are the sentinels of `spec-clearing.md` 8.2
and `spec-ledger.md` 12.4; this spec fixes where `Step` returns them (3.3.3).
`ErrDuplicateID` and `ErrOutOfBounds` are the existing sentinels of
`spec-config.md` 7.

15.2. **Warning proposed to CFG:** `cold_start_lockstep`, informational: a role
with a recipe of `Duration` above 1 and more than one agent in a group, with no
warm start configured, **or with a warm start that gives the role no job row**
(the reference pack `one`'s growers). It says that the first jobs of the group start and complete
together (STB-L22, WLD-A3). Its threshold is a count, not a number to be calibrated.
It is raised once at construction at `Warn` with `rule` and `path`, like every
warning (`spec-config.md` 7.2).

15.3. **What is not validated here:** a cadence that does not divide `TicksPerYear`
is legal (9.3); two cadences with a common factor are legal (9.3.4); a seed of any
value is legal.

## 16. Observability

16.1. **Events owned by this spec**, with the fields the rules already fix
(observability-and-logging.md 4.3):

| Event | Emitted | Fields |
|-------|---------|--------|
| `WorldCreated` | at the end of construction, before the initial `AgentSpawned` events (5.8) | `Tick` 0, `Seed`, the whole configuration hash, whether a warm start was present, the names table of the IDs (`spec-observability.md` 3.2.4) |
| `RunStarted` | by `Run` before its first tick (3.4) | `Tick`, `Ticks` requested |
| `RunEnded` | by `Run` before it returns, whenever `RunStarted` was emitted | `Tick`, ticks completed, whether it was cancelled, `Failed` (true when the run ended on a failed step) |
| `TickStarted`, `TickEnded` | at P0.3 and P8.5 (and at the failure of P4.1) | `Tick`; `TickEnded` also `Failed` |

`Failed` on `RunEnded` and on `TickEnded` are fields that OBS added at this
spec's request (`spec-observability.md` 4.4.3; observability-and-logging.md 4.2).
`TickEnded` carries `Tick` and `Failed` (R2, 3.5.6) and nothing else: it is not a
summary, and a count that an observer wants is derived from the events of the step
(`spec-observability.md` 4.4.3.1).

16.2. **Events of the slots.** The yield shock, the transit factor and the birth
and death scales have an event each, `YieldShocked`, `TransitFactorChanged` and
`PopulationScaled`, one per changed key of a call (`spec-observability.md` 4.4.4);
the other slots have one (`EdgeStatusChanged`, `MoneyFlowScaled`,
`EndowmentAdjusted`, `OutsideShocked`, and, of the later modules, `RentScaled`,
`CreditRateScaled`, `RouteRiskScaled`, `StoreRiskScaled`, `LegalLevelSet`,
`RightSuspensionSet` and `TechnologyAdjusted`); N-091 says each slot emits its own event
and none is configuration. The events of a policy actor's write are the owners' slot events
with the cause `policy`, at P0.3a (8.2.2).

16.3. **Logs**, through the injected logger, constant lowercase messages, the keys
of the rules' table 3.3.2 (this spec adds none; the keys that the later modules added to
the "world created" line are in 5.8.1):

| Level | Message | Attributes | When |
|-------|---------|------------|------|
| `Info` | "world created" | the keys of 5.8.1 | once, at the end of construction |
| `Info` | "run started" | `ticks` | `Run`, once |
| `Info` | "run ended" | `ticks` (completed), `reason` (`completed`, `cancelled` or `failed`) | `Run`, once; the error is returned, not logged |
| `Warn` | the configuration's warnings | `rule`, `path` | once each at construction (`spec-config.md` 7.2) |
| `Debug` | "tick ended" | `tick`, `agents`, `households` | once per step, guarded by `Enabled`; the two counts are read from the world's counters, not computed |

No log line at `Info` or above is emitted inside a step (observability-and-logging.md
3.2.2), and a returned error is never also logged (architecture.md 8.3).

16.4. **Metrics.** None are new. `economysim.world.step.duration` is measured by
the adapter from `TickStarted` to `TickEnded` (observability-and-logging.md 5.3.1);
the library measures no time. A span per `Run` and per `Step` follows from the
events (6.1 there).

16.5. **Snapshot.** The tick counter and the seed are in the snapshot. A failed
world refuses a snapshot (3.5.1). The other fields are OBS's.

The seed of the snapshot is the effective seed, which a reseed replaces (12.3, ADR 0008, CHK 6.8); it is the seed read back, not information only. The event `WorldRestored` is not in the first slice.

16.6. **Cost with no observer.** With the default logger and observer no event of
this spec is constructed and no log call allocates (4.5 there); the guard of 3.6
is one boolean test.

16.7. **The Observability entry** (observability-and-logging.md 9): see the
handoff. In short: new failure paths are returned (`ErrInvalidTrade`,
`ErrAuditFailed`, `ErrWorldFailed`, `ErrInStep`); no new state transition needs an
event beyond those already listed, and three slots gained one (`YieldShocked`,
`TransitFactorChanged`, `PopulationScaled`; the birth and the death scale share
one); the
`Run` loop has `Info` lines only; nothing in a tick is logged above `Debug`;
no new metric attribute. The events of the later modules' positions (6.3 to 6.10) are owned
by their specs and listed in the rules' table 4.3; this spec fixes only where each comes in
the order of a step (6.11).

## 17. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. This spec adds
two amplifiers, one bias and two hazards, and no new loop; it adds no
feedback loop of its own (it has no rule that reads a measure and changes a rate,
9.4). The checklist of `spec-stability.md` 9 was followed: 9.1 each rule of the
tick was read for the chain it creates (the only chains are the one-tick delay,
the order effects and the stream); 9.6 the delays were checked against the
neighbours (the review periods against the windows and the season, 9.3.4).

**WLD-A1 The phase order is a one-tick delay in every loop.** *Kind:* amplifier.
*Loop:* none of its own. *Delay:* one tick: a price formed at P4.1 reaches a
decision at P3.2 of the next tick; money earned at P4.1 is spendable at P3.2 of the
next tick; a belief revised at P5.4 is used at P3.2 of the next tick; an outcome
is seen at P5.4 and not before. *Owner:* WLD 6.1.8 (CLR 3.3.1, AGT 4.1).
*Damper:* none by design: the delay cannot be shorter without letting an order
spend what it has not yet earned. *Undamped:* by design. *Condition:* the gain
bounds of the loops through prices, income and beliefs (STB-L01 to L21, L84, L85,
L93) are written for a delay of at least one tick and must not assume zero.
*Test:* none of its own; every price-loop scenario runs on it. *Status:* hand.

**WLD-A2 A cap is first-come.** *Kind:* bias. *Loop:* none: it chooses who is
served when a cap or a budget binds (the places O1 to O5, O9 and O10 of 10.2), and does not
feed back unless being served changes an agent's later welfare, in which case it
selects for survival by ID, role or region. *Owner:* WLD 10.2 with SPN 6.3, 7.3,
MIG 7.1, MON 5.4, MER 9.4, SET 6.4.5, ORG 4.3. *Damper:* the keyed tie of 10.1.1 and a rotation of the
starting role, region or lane (WLD-D1), **proposed, not adopted**. *Undamped:* the
seven places as written. *Condition:* a cap that binds a fraction of the ticks and
a difference between roles or regions that is not read as real before its counter
(`EntryLimited`, `MoneyFlowLimited`, `ContractRound`) is checked. *Test:* WLD-S1
(the unit model `rotation.py`). *Status:* model of the ordering alone.

**WLD-A3 A cold start is a lockstep.** *Kind:* amplifier (the remainder of STB-L22
and AGT-A8). *Loop:* agents free at tick 0 start their jobs together and complete
together, so one tick's supply and demand for inputs is a pulse repeated every
`Duration` ticks. *Delay:* `Duration`. *Owner:* PRO 16 and VIA 8.5.4; the position
is the world's (5.6.3). *Damper:* D22, the stagger of a warm start, which exists
only with a warm start. *Undamped:* a world with none. *Condition:* a warm start for
any chain with a recipe of `Duration` above 1, or the warning `cold_start_lockstep`
is read as a request for one. *Test:* S18 (VIA) with a cold-start leg, as WLD-S3.
*Status:* hand. Merged into L22.

**WLD-A4 Cadences that coincide, and a group smaller than its period.** *Kind:*
hazard. *Loop:* none. Two rotations whose periods have a common factor (30 and 10) put an
agent's occupation review and migration review on the same tick for 13.3% of agents
with the rule bucket (4 in 30; always with raw IDs), and a
group of `n` agents with a period `P > n` acts on at most `n` of the `P` ticks. The
first is harmless because the settling rule (`spec-migration.md` 5.2) holds an agent
that changed role this tick; the second means a small group acts in bursts.
*Owner:* WLD 9.3.4 with SPN 6.2, MIG 5.1. *Damper:* the bucket of 9.1.1,
which spreads the load whatever the layout of the IDs for a group of about a
thousand agents or more. *Undamped:* the burst of a
small group, which is a property of the period. *Condition:* a role of fewer
agents than its review period is read as bursty. *Test:* WLD-S2 (`cadence.py`).
*Status:* model of the load alone.

**WLD-A5 The stream is positional.** *Kind:* hazard. *Loop:* none: one
extra draw early in a run changes every later draw of the run (7.6). Two runs that
differ in a datum that changes a draw count are alike in distribution and are no
longer draw for draw. *Owner:* WLD 7.6, architecture.md 7.3. *Damper:* WLD-D2: draw anyway
(5.7.2, 7.4.3). *Undamped:* any change in the number of intents considered, orders
admitted or agents created, which is what a different parameter does. *Condition:*
a paired comparison between two runs is exact, and a measured variance
reduction, only for a change that moves no draw count (while aligned); the
estimate of a difference of means from seed-matched runs is unbiased either way,
and the harness measures the gain (`spec-experiments.md` 7). *Test:* WLD-S4
(`stream.py`: 98.0% of the later draws differ with one stream, 2.0% with a stream
per tick). *Status:* model.

Stale under ADR 0008, see N-214: WLD-A5 is closed between streams (7.6): one extra word moves the later words of its own stream only (`stream.py`: 0.01% of the later draws, against 98.0% for one positional stream). The register entry L120, WLD-D2 (D88: "draw anyway" is an owner's choice inside one stream, 7.4.3) and WLD-S4 are rewritten with STB, which owns them. A hazard remains for a change of an ID (7.6.2).

**Dampers.** WLD-D1 *Ties and caps by key or rotation* (10.1.1): stated as the
rule for new specs and proposed for O1 to O5, O9 and O10. D25 *Decision rates*, which the
register lists as proposed with WLD as owner, is **specified** here (section 9);
status moves from proposed to specified. WLD-D2 *Draw anyway* (5.7.2, 7.4.3), an
existing practice (WEL 5.1, AGT 7.5.3, MIG 6.2.1) made a rule.

**Scenarios.** WLD-S1 *Fairness of a binding cap*: 100 tied candidates, a cap of 5,
1 000 ticks; pass when no candidate is admitted fewer than half or more than twice
its expected share, at each of the seven first-come places O1 to O5, O9 and O10 of 10.2 in turn (the cap standing for that place's budget, limit or places). WLD-S2 *Load of a cadence*: 1 000 agents at periods 10 and 30
with IDs spaced by every stride of 1 to 31; pass when no tick has more than
`ceil(1.2 * n / P)` reviews, for `n` of 1 000 and 5 000 (the rule bucket reaches 39
against a bound of 40 for `n` 1 000 and `P` 30 over three starting IDs, 113 against 120
for `P` 10, and 182 against 200 for `n` 5 000); no bound is claimed below about a
thousand agents (9.1.1). WLD-S3 *Cold and warm start*: a chain with a recipe of
`Duration` 10 run with and without a warm start; the completions per tick in the
first 40 ticks (with one: spread; without: a pulse). WLD-S4 *The stream after a
change*: two runs that differ by one extra draw in the first tick; report the share
of later draws that differ.

## 18. What was run

All in Docker (`python:3.12-slim`, `PYTHONDONTWRITEBYTECODE=1`), Python standard
library only, in `doc/specs/models/wld/`. They are models of the rules of this
spec, not of the library, and they show no default to be stable (N-027). Two
checks used Go in Docker (`golang:1.25`, and the installed Go 1.27; `go run` of stdlib-only
files that are not archived): the PCG vectors of 7.1.5 and the reseed of CHK 6.8 against
`seed.py` (every line equal), and that a `rand.Rand` holds no state beyond its source;
earlier, on Go 1.25, the 20-byte `MarshalBinary` and the words consumed by `Int64N`
(7.2.2).

| Script | What it does | Result quoted |
|--------|--------------|---------------|
| `seed.py` | the finaliser, the key of a stream, the site constants, `K(t, id)`, the reseed rule and a PCG-DXSM; the vectors of 7.1.5 and of CHK 6.8 (one source for both), asserted against the four bare seeds Go 1.25 printed, and every line equal to a Go program's; the bits that differ between the first outputs of neighbouring seeds, ticks and IDs | 7.1.5; 31.99 bits (bare mix), 31.94 (plain), 32.04, 32.03 and 32.03 (keyed, by ID, tick and seed) of 64 |
| `rotation.py` | a binding cap with tied candidates under the lower ID, the keyed tie and a rotation | 7.1; 10.1.1; WLD-S1 |
| `cadence.py` | the reviews per tick of a rotation by raw ID, by `Weyl32 mod P`, by the Weyl32 bucket (correlated with MIG's margin), by the hash bucket (binomial, not adopted) and by the rule bucket of 9.1.1 (a second multiplier, `0x6A09E667`), for strides 1 to 31 and periods 10 and 30; the 400 households of a migration margin and the ticks they review on under each; the share of agents whose 30-tick review is also a 10-tick review; a group smaller than its period; the drift of a review against a year | 9.1.1, 9.3, WLD-S2 |
| `stream.py` | the share of later draws that differ after one extra draw, with one positional stream and with a stream per tick (both rejected) and with a stream per (site, tick, agent) (ADR 0008) | 7.6, WLD-A5, WLD-S4: 98.00%, 2.00% and 0.01% |
| `worked.py` | the arithmetic of the examples of 19: the buckets, the creation words of example A and the streams of the tick of example B | 19 |

What they do not show: that the phase order is stable for any chain (nothing has
run in the library), that the bucket is better than the raw ID for the layouts a
real run produces (the stride cases are constructed), or that any cadence of 9.2
is right.

## 19. Worked examples

All settings are invented **examples**: the words in them (`grain`, `loaf`,
`baker`, `grower`, `alpha`) are flagged as example data, never branched on
(N-017), and every number is a placeholder for illustration (N-027).

### 19.1 Example A: the order of construction

Three regions sorted by ID: `alpha` (RegionID 0), `beta` (1) and `edge` (2),
where `edge` is marked an outside node with a gate at `alpha` and one line with a
sell side for `grain`. Commodities `grain` (0) and `loaf` (1). One money flow
`stipend`. Groups: `alpha`/`baker` 2 households, `alpha`/`grower` 3,
`beta`/`grower` 2, all at scale 1, so one agent each.

| Thing | IDs |
|-------|-----|
| Estate pseudo-agents | `alpha` 0, `beta` 1, `edge` 2 (an unused estate, `spec-external-sector.md` 3.1.1) |
| Counterparties | (node `edge`, gate `alpha`, sell) 3 |
| Households | `alpha`/`baker` 4, 5; `alpha`/`grower` 6, 7, 8; `beta`/`grower` 9, 10 |

The ledger's accounts, in creation order, with `B` the number of built-in
boundary accounts: `0 .. B-1` built-in boundaries; `B` the boundary of `stipend`;
`B+1 .. B+3` the money reasons of the outside and `B+4 .. B+6` its goods reasons for
`grain`; then the estate accounts of `alpha`, `beta`, `edge` (`B+7 .. B+9`); the
Outside account of counterparty 3 (`B+10`); the agent accounts of agents 4 to 10
(`B+11 .. B+17`).

The creation draws are taken after the warm start, one stream for each agent, `(siteConstruction, 0, id)`
(7.1.4): its word 0 is the target ratio and its word 1 the residue. Fourteen words are the whole
construction, in seven streams, with the technology module on or off (5.7.3), and the first draw of
tick 0 belongs to a stream of its own. For seed 0 (`worked.py`), the words of each agent and its
residue `draw(10 000)` of word 1:

| Agent | Word 0 (target) | Word 1 (residue) | Residue |
|------:|-----------------|------------------|--------:|
| 4 | `f4aa09314fe44644` | `ef7ea66b5a29b979` | 9355 |
| 5 | `9c1eb28b41d67934` | `c1a4ddc74bcb96e2` | 7564 |
| 6 | `0f06ca331ae3bce8` | `dc3ad5fd46bb3d42` | 8602 |
| 7 | `0d2a6d25ba31d1f9` | `a5c78a135e49c31b` | 6475 |
| 8 | `ee4a4ecf40900610` | `8cad7e44acddda54` | 5495 |
| 9 | `9957ed1d2127e557` | `8dafe94e04a12c05` | 5534 |
| 10 | `f9b97445db9bf2f2` | `5373bd0ce51ecc45` | 3259 |

The values do not depend on the order in which the agents are created. The events:
`WorldCreated`, then `AgentSpawned` for agents 4 to 10, each with `Tick` 0.

### 19.2 Example B: the streams of one tick

One region `r0`, commodities `grain` (0) and `loaf` (1), roles `grower` (agents 1 and 2)
and `baker` (agent 3), estate agent 0, no module, default strategies. In one tick
the three agents consider 2, 2 and 3 intents; one of the seven is dropped by
affordability, so six orders are admitted: the market `(r0, grain)` has 1 bid (agent 3)
and 2 asks (agents 1 and 2) and `(r0, loaf)` has 2 bids (agents 1 and 2) and 1 ask
(agent 3). Each draw site is a stream of its own, and the words are counted from 0 in
each. For seed 42 at tick 7 (`worked.py`):

| Position | Stream | Words | Draw | First word |
|----------|--------|-------|------|------------|
| P3.2, agent 1 `Trade` | `(siteTrade, 7, 1)` | 0, 1 | D5 | `e00b223f9aeed1d9` |
| P3.2, agent 2 `Trade` | `(siteTrade, 7, 2)` | 0, 1 | D5 | `2f20f13dc103e16b` |
| P3.2, agent 3 `Trade` | `(siteTrade, 7, 3)` | 0, 1, 2 | D5 (three intents, the dropped one included) | `7e13c409cdff5e06` |
| P4.1, market `(r0, grain)` | `(siteClearing, 7, 0, 0)` | 0, 1, 2 | D7: the bid, then the two asks | `a0043220bd826c9b` |
| P4.1, market `(r0, loaf)` | `(siteClearing, 7, 0, 1)` | 0, 1, 2 | D7: the two bids, then the ask | `9f4b829a3880ffc9` |
| P5.4, the outcome of agent 1 in grain | `(siteOutcome, 7, 1, 0)` | 0 | D8 | `500ff86076a8245a` |
| P5.4, the outcome of agent 2 in grain | `(siteOutcome, 7, 2, 0)` | 0 | D8 | `ca5031df09cef670` |
| P5.4, the outcome of agent 3 in grain | `(siteOutcome, 7, 3, 0)` | 0 | D8 | `eb2a44a7538a0ffe` |
| P5.4, the outcome of agent 1 in loaf | `(siteOutcome, 7, 1, 1)` | 0 | D8 | `71975eae0678752f` |
| P5.4, the outcome of agent 2 in loaf | `(siteOutcome, 7, 2, 1)` | 0 | D8 | `3610982609122f2d` |
| P5.4, the outcome of agent 3 in loaf | `(siteOutcome, 7, 3, 1)` | 0 | D8 | `7a47a487b36187e1` |

Nineteen words in eleven streams. An observer, a logger or `WithAudit` adds none. If agent
3's role is informed with `RumourErrorBP` above 0 and its view has four candidates, four more
words (D4) are taken, from a stream of that agent's own (its site is appended with the
information module, 7.1.4), and the tick takes 23 words: the count is a function of the
configuration. **No word of any other stream moves**, agent 3's `Trade` included, which still
starts at its own word 0; before ADR 0008 the same four words moved every later word of the
tick (agent 3's `Trade` started at word 8).

### 19.3 Example C: a rotation and a review

A region of four agents (IDs 10 to 13): the agent that goes first in P2.1, P2.5 and
P2.7 is 10, 11, 12, 13, 10, ... at ticks 0, 1, 2, 3, 4 (`t mod 4`). A review with
period 3 of the agents 10 to 15, with `bucket(id, 3)` 0, 1, 2, 1, 2, 0 (`worked.py`):
agents 10 and 15 act at ticks 0, 3, 6; agents 12 and 14 at 1, 4, 7; agents 11 and 13
at 2, 5, 8; so 2 reviews a tick (raw IDs would give the same for consecutive IDs;
the bucket is not exact in general, and at a thousand agents and a period
of 30 it is 30 to 37 a tick against a mean of 33.3 for any stride (`cadence.py`).

## 20. Notes settled

For `AGENTS.md`: each open note tagged `WLD` or `ALL`, and the others that name
the world, is applied, rejected or left open here, with the text to append in the
handoff.

| Note | Decision | Where |
|------|----------|-------|
| N-009 What the world and tick phases must decide | **Applied; close for WLD.** One tick of eight phases with every position (6); every update of waves 1 to 4 consolidated; the wordings that disagree are ruled (6.12); `Step`'s errors (3.3.3); the checkpoint position (8.4); the options in the hash (4); the rates (9) | 3, 4, 6, 8, 9 |
| N-031 What the calendar hands to other specs | **Applied for WLD:** the moment is computed once at P1.1, derived, adds nothing to a checkpoint. **Leave open** for OBS (a season event, the moment in the snapshot), INT and GDS as before | 6.3 |
| N-036 Production's draws, phases and shocks in the world | **Applied for WLD:** production runs before orders; continuations and completions before starts; the rotation by `t mod n_r` with the count taken at the start of the phase; the creation draws in the order target then residue; the yield-shock slot is read at P2.7 and P1.2. **Leave open** for INT (the public call) | 5.7, 6.1.4, 6.4, 8.2 |
| N-046 Register entries owed by unwritten specs | **Leave open** for the other specs, with this added: WLD has its entries (17); D25 is specified (section 9); the WLD row of `spec-stability.md` 4.8 can be removed | 17 |
| N-051 What the Strategy contract needs from other specs | **Applied for WLD:** the phase order of `spec-agents.md` 4 stands; `Work` between completions and starts and always two-pass (6.4.1); `Trade` after production; `Revise` per outcome with the world's `Draw` (P5.4); the spawner after consumption with the draws per region (6.9.1). **Leave open** for INT and CHK | 6 |
| N-056 What the ledger and world need for a warm start | **Applied; close.** Items 4 and 5: a job created at construction with `Remaining < Duration` and its pending output (5.6.1 step 4, residue 0), and the warm start before the creation draws (5.6, 5.7) | 5.6 |
| N-074 The construction phase ends at the first Step | **Applied; close.** It begins at the first line of `NewWorld` and ends when the first `Step` begins; the consumer's calls in between are inside it; the lot key is the warm start's privilege (5.5) | 5.5 |
| N-082 Relief and entry share the pool in time | **Applied:** phase 1 relief at P1.6, phase 7 entry at P7.1; the order holds (6.3, 6.9); money flows with a pool target run at P1.7 after relief. **Leave open** for the notes' other owners (N-096, N-119) | 6.3 |
| N-111 A view takes draws | **Applied for WLD:** the view of an informed agent is built immediately before its `Trade`, its draws in ascending source `RegionID` then `CommodityID`; the migration pass takes its own at P7.2; the two cannot interleave; the count is a function of the configuration and the informed agents present (D4, D11, 7.3) | 7.3 |
| N-117 The agent ID space and the account order with an outside | **Applied for WLD:** estates `0..n-1`, counterparties next, households after; accounts: built-in boundaries, flows' reasons, the outside's six, estates, Outside accounts, agents (5.2, 5.3) | 5.2, 5.3 |
| N-016 Keep the parameter registry true | **Applied:** WLD adds no field; one row is proposed for the name `World.Clearing` (4.2, 14) | 14 |
| N-017 Words and names | **Applied:** twenty terms are defined here and go to the glossary (the list is in the handoff): tick boundary, phase, position, entry, slot, between ticks, cadence, rotation, bucket, Weyl value, keyed tie, failed world, word, the world's draw, creation draw, construction phase, gate, carried remainder, window, outside node as a region of a loop; the examples are flagged setting words | 19 |
| N-027 No default macro constant is stable | **Applied:** this spec sets no default; the numbers of the examples are placeholders; the models show no default stable | preamble, 14, 18 |
| N-103 Running several spec writers in parallel | **Applied:** no shared file was edited; the proposals are in the handoff with provisional IDs (`WLD-A1` to `A5`, `WLD-D1`, `D2`, `WLD-S1` to `S4`); the models ran in Docker with no bytecode. **One slip:** the host `python3` was called once with `--version` early on; nothing else ran on the host | 17, 18 |
| N-012 Events, metrics and log keys to define | **Leave open**, with this added: OBS accepted `Failed` on `RunEnded` and `TickEnded` and the events `YieldShocked`, `TransitFactorChanged` and `PopulationScaled` for the slots without one (16) | 16 |
| N-020 What each spec contributes to a checkpoint | **Leave open**, with this added: the effective seed (in the header, no PCG state: ADR 0008), the next agent ID and the construction flag; the tick is written in the header (12) | 12 |
| N-045 Dampers that act silently | **Leave open**, with this added: the binding caps are counted by their modules' events and WLD-A2 asks the readers to check them (10.2.1) | 10.2 |
| N-075 Mapping the ledger's sentinels | **Leave open**, with this added: `ErrUnknownClearing`, `ErrNilExtension`, `ErrWorldFailed`, `ErrInStep` and `ErrOptionContext` need a root mapping (15.1) | 15 |
| N-091 The world slots INT must expose | **Leave open for INT**, with this added: the slots as one table with the position each is read at (8.2) and the calls that act on agents (8.3) | 8 |
| N-019 Housekeeping for the first code change | **Leave open**, with this added: `random.go`, `tick.go` and `option.go` and the `internal/ordering` helpers (23); the options and terms for the glossary | 23 |
| N-022 The player and other external agents | **Leave open**, with this added: an external agent is created at the call with the next ID and the two creation draws (8.3, D12) | 8.3 |
| N-079 External agents are mortal only when the consumer says so | **Leave open** (INT); the removal call is between ticks (8.3) | 8.3 |
| N-214 Keyed random streams (ADR 0008): the edits owed | **Applied for WLD 7** (the key, the site constants of the slice D1, D3, D5, D7, D8, D9 and D10, the vectors, `K(t, id)`, the draw table with its streams, 7.4 and 7.6 narrowed) and for 4.1, 4.4, 5.1, 5.7, 6.9.1, 6.12 (C22), 8.1.1, 12, 14, 19 and 22 (questions 4 and 7). **Leave open**: the sites of the rows outside the slice and their `id` (open point 3), and the passages marked "stale under ADR 0008" in 13, 16.5 and 17 | 7, 12, 22 |

Proposed new notes are in the handoff.

## 21. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. This spec
has no default or bound of its own.

### Functional

| ID | Requirement |
|----|-------------|
| FR-WLD-001 | `NewWorld` MUST apply its options in the order given, MUST return `(nil, err)` and no world when an option or the validation fails, and MUST NOT return a partly built world (3.2.1). |
| FR-WLD-002 | `NewWorld` MUST reject a `Role.Strategy`, `World.Spawner`, `World.Clearing` or `Actor.Policy` name that is not registered with `ErrUnknownStrategy`, `ErrUnknownSpawner`, `ErrUnknownClearing` or `ErrUnknownPolicy`, a name registered twice or a built-in name (`default` and `directed` of a strategy; `average` of a clearing; `hold`, `schedule`, `rule` and `directed` of a policy) with `ErrDuplicateID`, a name given to a `With...` option that is not an ID (empty included) with `ErrInvalidID`, and a nil `Strategy`, `Spawner`, `Clearing` or `Policy` with `ErrNilExtension` (3.2.2, 4.1, 4.1.1, 4.3). |
| FR-WLD-003 | A nil logger or observer MUST mean the discard default and not an error; `WithObserver` MUST be additive and deliver each event to every observer in the order the options were given (4.1). |
| FR-WLD-004 | `Step` MUST run the positions of section 6 in the order of their tables and MUST advance `Tick()` by one if and only if every position ran and the audit, when on, passed (3.3.1, 6, P8.6). |
| FR-WLD-005 | `Step` MUST return an error wrapping `ErrInvalidTrade` when a clearing returns an invalid list and `ErrAuditFailed` when the audit fails, MUST name the tick, and MUST NOT return an error for any other condition of a running simulation (3.3.3). |
| FR-WLD-006 | After `Step` returns an error the world MUST be failed: every later call that has an error result, other than `RelocationRefusal` (whose error is the refusal it reports), MUST return `ErrWorldFailed` wrapping the cause and change nothing; a reader with no error result MUST return its value (3.5.1); and a panic that leaves a step MUST mark the world failed without the library calling `recover` (3.5). |
| FR-WLD-007 | A call of any method of `World` that has an error result, made while a step is running, MUST return `ErrInStep` and change nothing, except `RelocationRefusal`, whose error is the refusal it reports; a reader with no error result MUST return its value (the immutable group of 3.5.1: `Tick`, `Config`, `ConfigHash` and the derived tables; the readers of live state return the state the step has left), and `Err` MUST return the failure (nil on a healthy world) (3.6). |
| FR-WLD-008 | `Run` MUST emit `RunStarted` before its first tick, check `ctx.Err()` before every tick including the first and never inside one, and emit `RunEnded` with the ticks completed, whether it was cancelled and whether a step failed (`Failed`) before it returns whenever it emitted `RunStarted` (3.4.1). |
| FR-WLD-009 | `Run` MUST return the ticks completed and, on cancellation, an error that wraps `ctx.Err()`; a negative `ticks` MUST return `ErrOutOfBounds` with no event; a cancelled run MUST leave the world at a tick boundary and usable (3.4.2, 3.5.5). |
| FR-WLD-010 | Every read of a healthy world between ticks MUST return a deep copy of plain data and MUST change no state and draw no random number (3.7.1). |
| FR-WLD-011 | Two worlds MUST share no state, so that stepping them on two goroutines is free of races (3.1.1). |
| FR-WLD-012 | Every step that emitted `TickStarted` MUST emit `TickEnded`, a failed one included, with `Failed` true when the step returned an error, set at the point of failure and after the world is marked failed; a guard that fails at P0.1 MUST emit nothing; a step that ends in a panic MUST emit nothing; `Tick()` MUST be unchanged by a failed step (3.5.6). |
| FR-WLD-020 | `WithSeed`, `WithLogger`, `WithObserver`, `WithAudit` and `WithReseed` MUST NOT change the configuration hash, and `WithTargetSpread` MUST change it (4.1). |
| FR-WLD-021 | Every option that can change a result MUST be a datum in the effective configuration or a registered name that a role, the spawner, the clearing or a policy actor selects, or, for `WithReseed`, a call of the sequence of calls (4.4, 8.1.1). |
| FR-WLD-030 | At construction the ledger's accounts MUST be created in the order of 5.2 (the Treasury accounts at 5a, after the Outside accounts and before the agent accounts) and an account ID MUST never be reused. |
| FR-WLD-031 | The agent IDs MUST be allocated as 5.3 says: the estates `0 .. n-1` with the ID equal to the `RegionID`, then the counterparties, then the treasury pseudo-agents in ascending `JurisdictionID`, then the households in (region, role, index) order, and a later agent the next ID. |
| FR-WLD-032 | The agents and weights of a group MUST be derived without a draw, and the pseudo-agents MUST NOT be passed to a strategy or counted as agents or households (5.4.1, 5.3.1). |
| FR-WLD-033 | The warm start MUST be applied before the creation draws, MUST take no draw, and a job created at construction MUST have its pending output fixed with a residue of 0 (5.6). |
| FR-WLD-034 | For every household agent, in ascending ID, the world MUST take two draws at its creation, the target ratio and then the residue, whatever the configuration, and a later agent MUST take the same two at its creation (5.7). |
| FR-WLD-035 | The construction phase MUST end when the first `Step` begins, the consumer's calls before it MUST be inside it, and only the warm start MUST be able to state a lot key (5.5). |
| FR-WLD-036 | With `WithAudit` `NewWorld` MUST run the audit once after construction and fail with `ErrAuditFailed` if it fails (5.1). |
| FR-WLD-037 | Construction MUST emit `WorldCreated` (with the names table of the IDs) and then one `AgentSpawned` for each household agent in ascending ID, then each initial `FacilityChanged` and then each founding `SettlementClassChanged`, with `Tick` 0, only when an observer was supplied, and the logs of 5.8.1 (5.8); the construction bucket MUST NOT be emitted at construction but at the entry of the first step (P0.2a). |
| FR-WLD-038 | A group with a warm-start entry MUST be created with the entry's money and stock in place of `StartMoney` and `StartStock`, in one `initial` creation (5.4.2). |
| FR-WLD-040 | The positions of a tick MUST be the positions of section 6 with the IDs given there, a new position MUST take a letter after the position it follows, and no position MUST be renumbered (6.1.1). |
| FR-WLD-041 | Every loop of a position MUST visit its items in the total order that the row states and MUST NOT iterate a map (6.1.2, architecture.md 7.5, 7.6). |
| FR-WLD-042 | An away agent and an agent on a journey MUST be passed to neither `Work` nor `Trade`, a teaching trainer MUST NOT be passed to `Work`, and a pseudo-agent MUST NOT be passed to a strategy (6.1.3). |
| FR-WLD-043 | The starts and the calls of `Work` MUST run region by region and, within a region, in ascending ID from the position `t mod n_r` of the agents whose account is there, with `n_r` taken once at the start of phase 2 (6.1.4). |
| FR-WLD-044 | A module that is off, or on with nothing to do, MUST have no position that takes a draw or emits an event, and MUST leave every result as a world without it (6.1.5). |
| FR-WLD-045 | A reservation MUST be opened only at P3.2, P3.4, P3.4a and P3.5 and every reservation MUST be settled or released by the end of P4.1 (6.1.6). |
| FR-WLD-046 | All calls of `Work` of P2.5 MUST precede every start of P2.7, whatever modules are on (6.4.1). |
| FR-WLD-047 | In phase 3 the agents' plans MUST be executed in ascending agent ID, then the contract round, the credit round and the cover round, then the estates' asks, then the treasuries' orders, then the counterparties' orders, then the aggregate events, and a market's input order MUST be the agents' orders, the estates', the treasuries' and the counterparties' (6.5, 6.5.1). |
| FR-WLD-048 | At P4.1 the markets MUST clear in ascending (`RegionID`, `CommodityID`); an invalid list MUST settle nothing of its market, release its reservations and fail the step before any later market clears (6.6.1). |
| FR-WLD-049 | At P5.4 the world MUST take one `uint64` for each admitted order of a household agent, in ascending (`RegionID`, `CommodityID`) and admission order, as the first word of the stream `(siteOutcome, t, agent, commodity)` of that order, and pass it as the outcome's `Draw`; an estate's ask, a treasury's order and a counterparty's order MUST have no outcome and take no word (6.7, 7.3). |
| FR-WLD-050 | `Privation` MUST be set once per tick, at P6.1 with `Hunger`, and the spawner and the population module MUST read it (6.8, ruling C2). |
| FR-WLD-051 | In phase 7 each region's computation, view, `Decide` and execution MUST run in ascending `RegionID` with the creation draws of that region before the next region's `Decide`, and the migration pass MUST run once after every region (6.9). |
| FR-WLD-052 | The tick counter MUST advance after `TickEnded`; the audit MUST precede `TickEnded` and MUST NOT change any state (6.10). |
| FR-WLD-053 | The events of a step MUST come in the order of 6.11, and the events of a call between ticks MUST come at the call with the `Tick` of the next step, before its `TickStarted`. |
| FR-WLD-054 | An event MUST be emitted after the transition it describes is complete, and MUST NOT be constructed when no observer was supplied (6.1.7). |
| FR-WLD-055 | Transport MUST relocate a traveller at P1.3 after the guard query run ignoring only the arriving shipment and the contract formed on it, so that any other shipment, contract or obligation of the agent still refuses (ruling C16). |
| FR-WLD-056 | The entry budget MUST be taken in P7.1(d) after the module's deaths and before the switches, and the view's `Budget` MUST be advisory (ruling C15). |
| FR-WLD-057 | A reader of the information record at P7.2 MUST read as of tick `t + 1` (ruling C17). |
| FR-WLD-058 | P5.1 MUST push the price window only of a market in which units traded this tick, and P1.2 MUST read the published price left by P4.1 of the previous tick, the seed price at tick 0 (rulings C19, C20). |
| FR-WLD-059 | The events of one removal MUST come in the order of 6.11 item 8, with `AgentBankrupted` as the only event that removes the agent (ruling C11). |
| FR-WLD-060 | The world MUST hold one seed, and one `*rand.PCG` and one `*rand.Rand` on it created once, and MUST reseed the PCG at the start of each draw site by `Seed(hi, lo)` of the key of 7.1.3: `s = seed`, `s = finalise(s + w + γ)` (γ = `0x9E3779B97F4A7C15`) for the site, the tick, the ID and, when its row has one, the `sub`, then `hi = finalise(s + γ)` and `lo = finalise(s + γ + γ)`; the bare seeds 0, 1, 2 and 42 and the keys k1 to k9 MUST give the states and the first two words of 7.1.5 (a) and (b). |
| FR-WLD-061 | The world's bounded draw MUST be the high word of `Uint64()` times the bound, consuming one source word, and a range `[lo, hi]` MUST be `lo + draw(hi - lo + 1)` (7.2.1). |
| FR-WLD-062 | The world MUST take no random number other than those of the rows of 7.3, each from the stream of its row, in the order there within that stream. |
| FR-WLD-063 | A run split into several `Run` calls, and a run checkpointed and restored at a tick boundary, MUST give the same draws, events, snapshots and final state as an uninterrupted run (7.4.1, 8.4.3). |
| FR-WLD-064 | The number and order of the draws of a tick MUST NOT depend on an observer, a logger, `WithAudit`, a snapshot or other read, or the hash (7.4.1). |
| FR-WLD-065 | A module that is on and inactive MUST take no word and MUST NOT reseed a stream, and the streams of every other site MUST equal those of the module off (7.4.2). |
| FR-WLD-066 | A creation draw MUST be a row of the table of 5.7.1, MUST be a word of its agent's creation stream, and MUST NOT be conditional on a module or an option (5.7.2). |
| FR-WLD-067 | The site of each row that takes a word MUST be a literal 64-bit constant of 7.1.4 that does not depend on the position of its row in 7.3, no two sites MUST differ by less than 2^32, and a constant MUST NOT change once it has a golden output (7.1.4). |
| FR-WLD-068 | A signed tick or ID MUST enter the key as the two's-complement `uint64` of its `int64` value with no sign test, a site MUST have the arity its row gives (three words or four) and keep it, and the rows k6 to k8 of 7.1.5 (b) MUST hold (7.1.3). |
| FR-WLD-069 | Two runs of one seed that differ in a parameter, a shock or a module MUST draw the same word at every `(site, tick, id[, sub])` they share, and a change in the number of words taken from one stream MUST move no word of another (7.4.4, 7.6). |
| FR-WLD-070 | A call that acts between ticks MUST validate in full and then apply or change nothing, MUST emit its event at the call, MUST NOT be part of the configuration hash and MUST be saved by the spec that owns the state (8.1). |
| FR-WLD-071 | A call that draws MUST take its draws at the call, from the stream keyed by its row's site, by the tick the next step runs and by its ID, and a consumer's own draws in that call MUST follow in the order they are made (8.1.1, 7.3.2). |
| FR-WLD-072 | Each slot of 8.2 MUST act at the position or at the call that 8.2 gives, and a slot MUST NOT edit a configured value (8.2.1). |
| FR-WLD-073 | A checkpoint MUST be refused on a failed world and inside a step, and MUST be possible on a healthy world between ticks, including before the first step (8.4). |
| FR-WLD-074 | The flow totals MUST be cleared at P8.6 and not at the entry of a step, so that a move made by a call between ticks is totalled in the tick the next step runs (8.1.3, ruling C18). |
| FR-WLD-075 | A policy actor MUST write only the slots of kinds `flow`, `edge` and `transit`, at P0.3a, through the owners' internal setters and with the cause `policy`; the public setters MUST still return `ErrInStep` inside a step; and the slot events of the write MUST come after `TickStarted`, then `PolicyDecided` (8.2.2, 6.11 item 1). |
| FR-WLD-076 | Step 8 of construction MUST run its sub-steps in the order of 5.1.2: the treasury pseudo-agents, the household agents, 8a the titles, the warm start, the creation draws, 8b the initial facilities, the settlement classes, 8c the technology key; no construction word MUST be taken that is not an agent's creation word, and the technology key MUST take none (5.7.2, 5.7.3). |
| FR-WLD-077 | The positions of the later modules MUST be those of 6.2 to 6.10 (P0.3a, P1.3a, P1.7a, P1.7b, P1.9a, P2.5a, P2.8b, P3.1a, P3.3b, P3.3c, P3.4a, P3.6a, P3.6b, P3.6c, P4.1a, P7.2a, P7.2b, P8.1a, P8.1b, P8.3a, P8.3b, P8.3c), the draws those of 7.3 (D13 to D17), and the removal steps those of `spec-agents.md` 3.6 (2b to 2g), with every event of a removal before `AgentBankrupted` (6.11 item 8, rulings C24 to C28). |
| FR-WLD-080 | A rotation cadence MUST select an agent on the ticks where `(t + bucket(id, P)) mod P = 0`, with `bucket(id, P) = (Weyl2(id) * P) >> 32` and `Weyl2(id) = (id * 0x6A09E667) mod 2^32`, in `[0, P)` for every ID and every `P` from 1 to 1 000, and the bucket MUST NOT use `Weyl32` (9.1.1). |
| FR-WLD-081 | A quantity stated per year MUST be applied by a carried remainder whose denominator contains `TicksPerYear`, and MUST NOT be converted to an integer number of ticks (9.3.2). |
| FR-WLD-082 | A cadence that does not divide `TicksPerYear` MUST be accepted and applied unchanged (9.3.1). |
| FR-WLD-083 | No cadence, window, gate probability or share MUST be a function of a measured aggregate of the world, and a cadence MUST be a datum of the configuration, in the hash and constant for a run (9.4). |
| FR-WLD-090 | A tie between candidates of equal merit in a rule that a spec adds MUST be broken by `K(t, id)` and then the ID, unless the spec states a reason (10.1.1). |
| FR-WLD-091 | `K(t, id)` MUST be `finalise(t * γ + id + γ)` modulo 2^64 with γ = `0x9E3779B97F4A7C15`, that is the SplitMix64 finaliser applied to `x + γ` with `x = t * γ + id` as in RES 5.3.3, with `t` and `id` as the two's-complement `uint64` of their `int64` values, and MUST take no draw and keep no state; the vectors of 7.1.5 (c) MUST hold (7.1.7, 10.1.1). |
| FR-WLD-100 | The world's contribution to a checkpoint MUST be the effective seed, the next agent ID and the construction flag, in a fixed order (the tick counter and the seed are written in the checkpoint's header), plus the flow totals that calls between ticks added since the last P8.6 when there are any; no random state and nothing derived MUST be saved (12). |
| FR-WLD-101 | A restore from those and the other specs' state MUST continue exactly as an uninterrupted run; a world restored from a checkpoint taken after its first `Step` MUST NOT be in construction, and one taken before it MUST be, with its bucket (12.2, 5.5.3). |
| FR-WLD-110 | On a failed world `Err`, `Tick`, `ConfigHash` and `Config` MUST remain callable and return the failure, the tick the failed step was running, the hash and a deep copy of the effective configuration, the derived tables (`Calendar`, `ReferencePrice`, `Warnings`, `Graph`, `Travel`, `Polity`, `TreasuryAgent`) MUST return their value, and the readers of live state (`Endowment`, `InTransit`, `Away`) and `RelocationRefusal` MUST return the state the failed step left, with no guarantee of consistency; `Tick`, `ConfigHash` and `Config` MUST return their value on a world in a step (3.5.1, 3.6). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-WLD-001 | With the default logger and observer, a full `Step` of the reference chain MUST allocate no more than a baseline constant held in the test, and no event of this spec MUST be constructed (observability-and-logging.md 4.5, 8.2). |
| NFR-WLD-002 | No position of a tick SHOULD cost more than the shape of 11.1; in particular nothing in a tick SHOULD be quadratic in the number of agents or depend on the number of households. A benchmark with `b.ReportAllocs()` per phase reports the trend (testing.md 8.1). |
| NFR-WLD-003 | The world MUST NOT call `time.Now`, start a goroutine, read the environment or keep package-level mutable state (architecture.md 7.1, 7.3, 12.2). |
| NFR-WLD-004 | For a given seed, configuration and sequence of calls the whole output MUST be bit-identical across runs and platforms, checked by a golden test on at least two architectures (architecture.md 7.4, testing.md 7.1). |
| NFR-WLD-005 | The bucket, the world's draw and the keyed tie MUST have fuzz tests asserting that they never panic and stay within their range for every input (testing.md 7.4). |

## 22. Open questions

1. **Is one word per draw acceptable to the specs that wrote a method name?** INF
   5.5.1 writes `rng.Int64N(2E + 1) - E` and EXT 6.4.1 `rand.Uint64N`; this spec
   defines the world's draw as one `Uint64` and the high word of a product (7.2.1),
   which is BEL 5.2's reduction, so that the count of words is the count of draws
   and no Go release can change the golden output through a reduction method.
   Their authors may have wanted the standard methods for their statistical
   properties; the bias of the multiply-high is at most `n / 2^64`. Next step: INF
   and EXT accept the edit (C8), or this spec's unit becomes "call".
2. **A cold-start stagger in the world.** WLD-A3 is registered and left to the
   consumer's choice of a warm start. A rule that places the first start of an agent
   `i` of `n` at tick `floor(i * d / n)` would remove the pulse with no draw, at the
   cost of idling agents for up to `d - 1` ticks and of a branch in `Work`'s refusals.
   Next step: the first scenario (S18, WLD-S3) that shows a pulse in a pack with no
   warm start.
3. **Recovery of a failed step.** The world is failed (3.5). A journal of the
   step's writes would allow a rollback to the boundary; its cost is time and memory
   on every tick. Next step: when a consumer asks, with the measure of the cost.
4. **Per-purpose streams.** **Answered (ADR 0008, architecture.md 7.3, 7):** the world
   holds one seed and reseeds its one PCG at each draw site from `(seed, site, tick, id)`,
   so a count change moves only the later words of its own stream (WLD-A5: 98% of the
   later draws before, 0.01% after, `stream.py`), and the checkpoint holds the seed and
   no stream state (12). What stays open is narrower: ADR 0008 open point 3 (whether a
   site with no natural entity, D2 and D13, is keyed by its row index or by the row's IDs)
   for the rows outside the slice, and the cost of a site call against the budget of
   `spec-scale.md` 6, which the first code measures (NFR-WLD-001 covers the reseed).
5. **`RunEnded` and a failed step.** **Closed (OBS):** `RunEnded.Failed` is
   accepted as an added field, true only for a failed step and never together with
   `Cancelled` (`spec-observability.md` 4.4.3, 16.1 here).
6. **A sub-tick or a per-region tick rate.** Every position is per tick for the
   whole world; a region that ticks more often than another is not provided, and a
   tick that is several days is a calendar choice (`spec-calendar.md` 4). Next step:
   the first consumer's tick rate (N-030).
7. **Parallel stepping.** Markets are independent within a tick (`spec-clearing.md`
   7.2), and each draws its tie-break keys from its own stream
   `(siteClearing, t, region, commodity)` (7.3, ruling C22), so P4.1 may clear them
   concurrently: the stream is no longer a precondition (question 4 is answered).
   Parallelism stays forbidden by architecture.md 12.2 until an owner and a lifecycle
   exist for the goroutines.
8. **A named position as data.** The positions of section 6 are in prose and in
   tables; a test could read them from this file the way the parameter registry
   does (testing.md 6.7) and compare them with the code's list. Next step: with the
   first code.
9. **`TickEnded` as a summary.** An indicator may want the counts of a tick
   (orders, trades, agents) on the closing event. **Closed (OBS, IND):**
   `TickEnded` is not a summary (`spec-observability.md` 4.4.3.1); the counts of a
   tick are derived from the events of the step, which every observer receives, and
   the standing counts are in the snapshot.
10. **The first position of the first step.** At tick 0 the first step reads a
    record that is empty (INF 7.6), a welfare record padded by `DefaultRatio`, and a
    cost anchor of 0 when there is no wage. Whether the run should begin with a
    settling period that is not observed is STY's (`spec-scale.md` 7.2 says the
    consistency check runs "with no settling period"). For a pack with a warm
    start the first tick reads a populated record and a seeded price (5.6).
11. **Closed (decided by the supervisor): the review bucket is a second Kronecker
    sequence.** `bucket(id, P) = (Weyl2(id) * P) >> 32` with `Weyl2(id) = (id *
    0x6A09E667) mod 2^32` (9.1.1, C14). It keeps the even load (30 to 37 reviews a
    tick against a mean of 33.3 at 1 000 agents and a period of 30; the independent
    hash gave 17 to 55) and is not correlated with `Weyl32` (Spearman 0.008; 11 to 15 of the 400
    households that meet MIG's margin per tick, no empty tick). WLD-S2's bound is
    `ceil(1.2 * n / P)` again.
12. **`Work` as if not in the mode (SUB 5.5 item 6, FR-SUB-023).** Under two passes
    (C9) `Work` runs at P2.5 before the ration of P2.6, so it cannot know that the
    ration cut every request. Recommendation: the household that the ration cut
    entirely is flagged `Blocked` at P2.7 and its next `Work` is the next tick's.
    **Answered (SUB 5.5 item 6, FR-SUB-023):** a household whose own-use requests the
    ration cut entirely and which started nothing is counted in `Blocked` at P2.7, the
    flag stays, and its next `Work` is the next tick's; P2.8 reports the count.
13. **The carrier of planned want units.** The demand module plans want units at P3.2
    and consumes them at P6.2, and `Intent` and `TradePlan` have no field for them
    (DEM 4.7, AGT 6). Recommendation: `Intent.WantUnits`, saved nowhere (scratch of
    the tick). Owners: AGT and DEM, routed.
14. **`Role.InKind` has no mover and no position (WEL 6.1).** Not in version 1: income
    paid as upkeep goods would be a goods faucet and needs a ledger reason (N-021,
    testing.md 6.3) and a position. Owners: WEL and MON, routed.
15. **An endowment drawn at construction by a warm job.** Unstated (VIA 8.5.3).
    Recommendation: no draw, since the virtual start precedes the run and the stock at
    tick 0 is the stock after it. Owner: VIA, routed.
16. **The totals of a call between ticks at a checkpoint.** A move that a call made
    since the last P8.6 has a total that belongs to the next tick (8.1.3); a
    checkpoint taken then must save it or the restored audit fails (12). **Closed
    (CHK, LED):** the pending totals are saved (`spec-checkpoint.md` 4.4;
    `spec-ledger.md` 7.4, 11); the restore's baseline is the restored balance less
    them (`spec-checkpoint.md` 6.2 step 10).

### Edits owed to other specs

One consolidated table: every file, section and change. None alters a result that a
test could depend on (nothing has run), except where the row says a rule is
proposed and not owed. The supervisor applies them; the handoff is derived from this
table. A letter in brackets is the ruling of 6.12.

| File | Section | Change |
|------|---------|--------|
| `spec-agents.md` | 4, 4.0 | Phase 1 row: delete "and starting money" (C1); say the order of positions is `spec-world.md` 6 and supersedes this table |
| `spec-agents.md` | 3.5.1, 3.5.2 | A new agent starts with `Privation` and `Hunger` 0 (C2); the list of creation draws is `spec-world.md` 5.7.1 ("WLD may adopt it" becomes "WLD adopts it") |
| `spec-agents.md` | 3.6 step 6, 14.1, FR-AGT-008, FR-AGT-077 | `AgentBankrupted` is "the only event that removes the agent; the others accompany it", with the order of `spec-world.md` 6.11 item 8 (C11) |
| `spec-agents.md` | 5.7 | A trainee's upkeep is taken at the trainee's own position of P6.1 (C10) |
| `spec-agents.md` | FR-AGT-023 | `Revise` is called for the orders of household agents; an estate's ask and a counterparty's order has no outcome (C21) |
| `spec-beliefs.md` | 5.1.2, FR-BEL-008 | the same (C21); 7.4 and FR-CLR-032 agree with P5.1 (C19) |
| `spec-clearing.md` | 3.3.1, 7.2, FR-CLR-023 | "independent given the keys; the fixed order of the markets is part of the draw sequence (D7)" (C22) |
| `spec-clearing.md` | 7.3, FR-CLR-025 | the outcome of an order of a household agent only (C21) |
| `spec-clearing.md` | 8, open question 5 | owner of `World.Clearing` and its registration by name (C7, as `Spawner` in `spec-spawner.md` 3.1.3); question 5 answered: `Step` returns `ErrInvalidTrade` at P4.1 and the world is failed |
| `spec-clearing.md` | 8.3 | `Average` takes the keys (one `uint64` per order) and not `*rand.Rand`: the root takes the words (`spec-world.md` 23) |
| `spec-config.md` | 3.7, 5.2 | the field `World.Clearing`; its default comes from the defaults, an empty name fails (a zero is not a default) |
| `spec-config.md` | 7.1.2 to 7.1.4 | the name checks (strategy, spawner, clearing) run after `Validate` succeeds and return a `ValidationError` with path and rule; "the one config rule in `NewWorld`" becomes three |
| `spec-config.md` | 7.3 (N-075 line), 7.4.1, 7.4.2 | the sentinels `ErrUnknownClearing`, `ErrNilExtension`, `ErrWorldFailed`, `ErrInStep`; the rule for the field; the warning `cold_start_lockstep` (15.2) |
| `spec-config.md` | 8.1, 9.2, 10.1, 11, 13 (question 8) | the clearing name is hashed; the options list of 9.2 follows `spec-world.md` 4.1; 8.1 says that a stand-alone `Validate` does not see `WithTargetSpread`; FRs for the field and the warning |
| `spec-estates.md` | 2.1, 2.5 | the removal events accompany `AgentBankrupted` (C11) |
| `spec-estates.md` | 2.3 (last sentence) | starting money moves at creation, phase 7 (C1) |
| `spec-estates.md` | 2.7.3, FR-EST-009 | the markdown is applied at P5.3 to a lot that had an ask and was not wholly sold ("stays unsold" becomes that) (C6) |
| `spec-estates.md` | 2.8.1 | the budget is taken after the module's deaths, not at the start of the entry step (C15) |
| `spec-external-sector.md` | 6.4.1 | the walk's draw is the world's draw (C8) |
| `spec-information.md` | 4.3.2, 5.4.1, 7.1, FR-INF-018 | a phase-7 reader reads as of tick `t + 1`; the lag floor is restated for it; 7.1 no longer says the record is not read in phase 5 or later of the tick it is written in without the phase-7 exception (C17) |
| `spec-information.md` | 5.5.1, FR-INF-014 | the error is `lo + draw(2E + 1)` of `spec-world.md` 7.2.1 (C8) |
| `spec-information.md` | 6.1.3, FR-INF-051 | "reviewed" becomes "pushed" (D11) |
| `spec-information.md` | 6.3, 7.1, FR-INF-022 | a view is not built for an agent that is away **or on a migration journey** (TRN 10.5) |
| `spec-ledger.md` | 5.2.4 | the construction phase begins at the first line of `NewWorld` (5.5) |
| `spec-ledger.md` | 7.4, 7.4.1, 9.1, FR-LED-045, FR-LED-054 | the flow totals are cleared at the end of the tick, after the audit, not at its start; "read by the observability and indicator specs" becomes "reach them through `BoundaryFlow`"; a move between ticks falls in the tick the next step runs (C18); the bucket is kept for the run by a world that was never restored (8.4.1); the totals that calls between ticks added are saved (12) |
| `spec-ledger.md` | 11, 12.4 | the pending totals of 12 are state; the audit precedes `TickEnded` (C13); `NewWorld` runs it once with `WithAudit` (5.1 step 9) |
| `spec-merchants.md` | 12.5 | phase 1 pays relief only (C1); `MerchantsPlanned` is P3.3a and is emitted with the module absent |
| `spec-merchants.md` | 9.8.1 | `open_contract` stays registered for the payer and the deliverer and exposes the **contract IDs** as its instance keys; C16 ignores only the contract formed on the arriving shipment |
| `spec-migration.md` | 3.3(5), 5.1, 11.3, MIG-D1, FR-MIG-005 | the review rotation is the bucket; delete "consecutive IDs review on consecutive ticks" (C14) |
| `spec-migration.md` | 5.1, 6.2 case 3, 6.3.2 | 5.1: the sentence "so it sees the tick just ended" says the pass reads as of tick `t + 1`; 6.2 case 3: `NotYet` against `t + 1`; 6.3.2: the home prices are those of tick `t` (the window already holds the tick's push) (C17) |
| `spec-migration.md` | 7.1 | proposed, not owed: break equal earned ratios by `K(t, id)` (O2) |
| `spec-migration.md` | 8.2 step 4, 8.4, 10.1 | the `RecipeAbandoned` of a departure is emitted per departure in admission order before the `MigrationStarted` events (10.2 step 5); phase 1 pays relief only |
| `spec-money-supply.md` | 5.1.1, 5.1.2, 7.5, FR-MON-019 | "relief and starting money" becomes "relief" (C1); a grant to a pool at P1.7 feeds the **same tick's** entry (P7.1) and the next tick's relief |
| `spec-money-supply.md` | 5.4, 6.2, FR-MON-013 | proposed, not owed: a flow's limit cut pro rata (O4); the carry's start is not a rotation and may keep its own |
| `spec-demand.md` | 4.4, FR-DEM-013 | `AgentIndex` is the 0-based position in the region's agent list, away agents included (6.1.4) |
| `spec-population.md` | 4.1, FR-POP-002 | "after the world has updated `Privation`" becomes "which P6.1 set" (C2) |
| `spec-production.md` | 5.2, 5.5.1 | `Work` is two-pass whatever modules are on (C9) |
| `spec-production.md` | 5.2.1 | working ticks are read from a per-agent scratch set at P2.1 and P2.7, because the job is cleared at completion (P6.3) |
| `spec-production.md` | 6.6.2, FR-PRO-026 | the starting residue is drawn after the warm start; a virtual job's pending output uses residue 0 (C4); the draw is the world's (C8) |
| `spec-regions.md` | 6.1.2, 17.8 | arrivals relocate at P1.3 and P1.5 as well as in phase 7 |
| `spec-regions.md` | 6.1.6, 6.5.2, 6.6, FR-REG-059, FR-REG-060 | each registered obligation also reports the **instance keys** it holds for an agent, and `skip` of the query takes keys as well as names (a fixed-size key, so 6.1.6's "allocates nothing" holds); a module ignores "its own obligation" by key, so every other obligation still holds the move; transport relocates by it (C16); MIG's own call (`skip` = `on_journey`) stays by name, because an agent has at most one journey |
| `spec-migration.md` | 7.2.1 | the signature of `RelocationRefusal` takes keys beside names, with MIG's call by name unchanged |
| `spec-regions.md` | 6.4.1 | the relocation totals of a call between ticks belong to the tick the next step runs (C18) |
| `spec-scale.md` | 6.6 | markets are independent given the keys (C22) |
| `spec-skill-barriers.md` | 4.2, 4.4, FR-SKL-015 | the accumulator is stepped at P2.2, completion, placement and ticks served at P7.1(a); the arithmetic of "ticks served" is SKL's; a teaching tick only bars a start (C5) |
| `spec-spawner.md` | 3.1.2, 5.1, 10.2 | `Privation` is set at P6.1 and read here (C2); creation draws follow the plan of the region (C3) |
| `spec-spawner.md` | 3.2, 7.3, FR-SPN-026 | `Budget` in the view is advisory; the budget is taken after the module's deaths and before the switches (C15) |
| `spec-spawner.md` | 6.2, FR-SPN-018, 15.2 | the review rotation is the bucket (C14) |
| `spec-spawner.md` | FR-SPN-002, FR-SPN-012 | the spawner is not called for an outside node (6.1.11); FR-SPN-012 reads `Privation` as set at P6.1 (C2) |
| `spec-spawner.md` | 5.3.1, 13.1.1, FR-SPN-032 | "the only event that removes the agent" (C11) |
| `spec-spawner.md` | 6.3, 7.3 | proposed, not owed: ties by `K(t, id)`; the roles served from a rotating start or by share (O1, O3) |
| `spec-stability.md` | 4.8, D25, D33, D72, 10 (L22 row) | merge WLD-A1 to A5 (kinds: amplifier, bias, hazard); D25 specified; D33 and D72 cite the bucket; the WLD hook row goes; L22 cites WLD-A3 and S18's cold leg (WLD-S3) |
| `spec-stability.md` | S63, MIG-S3 | "the same seeds" is a paired design that is exact only while the draw counts are equal (WLD-A5, 7.6.1); where a draw count moves the estimate of a difference of means is still unbiased and the harness measures the gain (`spec-experiments.md` 7) | stale under ADR 0008, see N-214: a paired design is exact in the draws by construction (7.6.1) |
| `spec-subsistence.md` | 5.5 item 6, FR-SUB-023 | "`Work` acts as if not in the mode when the ration cut every request" cannot hold under two passes (`Work` is P2.5, the ration P2.6): applied, the household the ration cut entirely is `Blocked` at P2.7 and its next `Work` is the next tick's (open question 12) |
| `spec-subsistence.md` | 7 | own-use ticks and units enter the welfare record from the per-agent scratch (P6.3) |
| `spec-transport.md` | 5.3.2, 5.5.3, 5.7, FR-TRN-022, FR-TRN-024 | relief only in phase 1 (C1); `open_shipment` exposes the **shipment IDs** the agent is shipper or consignee of and, for an away traveller, its own trip; the arrival runs the guard ignoring its own shipment and the contract formed on it (C16) |
| `spec-transport.md` | 5.5.2 | "places no order" while away begins after the dispatch, which is in P3.2(c) |
| `spec-viability.md` | 8.1, 8.5.2, 8.5.3, 8.6 | the order of items 3 and 4 of `spec-world.md` 5.6.1; the pending output of a job created at construction uses residue 0 (C4); item 5 cites 5.6 and 5.7; no endowment is drawn at construction (recommendation) |
| `spec-welfare-and-wages.md` | 5.1 | "one draw" is the world's draw (C8) |
| `spec-welfare-and-wages.md` | 6.1 | `Role.InKind` has no mover and no position: routed (open question 14) |
| `spec-reference-chain.md` | 2 item 5, S18, X8(b)(d), FR-REF-005 | a cold pack expects the warning `cold_start_lockstep`; S18 gains the cold leg (WLD-S3) |
| `spec-fiscal.md` | 5.3, 6, 8.4, 10.1, 12 and its handoff | P1.7a becomes **P1.7b** (C24, about twenty mentions); the toll aggregate is **P3.6c** (C27); the toll step is step 3a of `spec-transport.md` 5.5.1 |
| `spec-credit.md`, `spec-insurance.md`, `spec-saving-and-investment.md`, `spec-risk.md`, `spec-infrastructure.md`, `spec-policy.md` | where each cites P1.7a | they cite CRD's P1.7a and keep it (C24); INS "P1.7a of `t + PayoutTicks`" is CRD's |
| `spec-technology.md` | 6.3.1 (table), 6.3.3, 6.3.4, FR-TEC-011, 17.2 | step **8c** and not 8a (C26); the draw is D16 and is written "draw D16" where the register's damper D16 could be meant (C25) |
| `spec-policy.md` | 7.1.1, FR-PLC (every mention of "D13") | the consumer's own words are **D17** (C25); the built-ins take none |
| `spec-risk.md` | 4.2, 6.2, 11.2, 16 (every "CAP's P8.3a") | the facility step is **P8.3b**; the rent step is P8.3a (LND keeps it) |
| `spec-infrastructure.md` | 3.4.3, 6.3, 14.1, FR-CAP | the initial facilities are step **8b** of construction (C26); `FacilityChanged` of a removed owner's facilities is removal step **2g** and comes **before** `AgentBankrupted` and `EstateOpened` (C28) |
| `spec-employment.md` | 6.5.2, FR-EMP-038, the removal rows | the hires end at removal step **2c**, after LND's 2b; the remainder of the escrow returns to the employer's account before CRD's waterfall (C28) |
| `spec-land.md` | 6.4.1, 18.2 | the titles go to the pool at removal step **2b**; the construction step 8a is kept (C26, C28) |
| `spec-credit.md` | 9 | the waterfall and the passing of claims are removal steps **2d** and **2e** (C28) |
| `spec-insurance.md` | 8.6, 8.6.3, 12.1.1 and every `PolicyFormed`, `PolicyEnded` | removal step **2f**; a store cover of the tick ends unpaid at 2f with the premium kept; the events are `CoverFormed` and `CoverEnded` (the name `Policy` is the policy actor's, architecture.md 4.3) |
| `spec-saving-and-investment.md` | 9, 14, FR-SAV-027 | the aggregate is **P3.6b**, after `RoomLimited` (C27) |
| `spec-storage.md` | 11.1, 14.1 | `RoomLimited` is **P3.6a**; drop "no new position" (C27) |
| `spec-agents.md` | 3.6 | the removal steps 2b LND, 2c EMP, 2d and 2e CRD, 2f INS, 2g CAP, all before `AgentBankrupted` (C28); 7.1.10 to 7.8 are the later modules' default rules |
| `spec-money-supply.md` | 4.4.3, 10.4, FR-MON-023 | "the only writer of `Scale` is a call from outside" gains: "or a policy actor at P0.3a, through the internal setter (`spec-world.md` 8.2.2)" |
| `specs/README.md` | index, reading order, area row 78 (WLD), parameter owners (a row for `World.Clearing`; line 147 `WithTargetSpread` "hashed"), decisions (row 480 and the decisions of 3.5, 7.1, 7.2), glossary | WLD exists; the 20 terms of N-017; the rows |
| `doc/specs/models/README.md` | contents | a row for `wld/` |
| `doc/rules/architecture.md` | 3.2, 6.2, 7.3, 12.3 | the glossary terms; the option list (4.1); the seeding rule and the world's draw (7.1, 7.2); `Run` returns two values and the state of a world after an error (3.4, 3.5); a fresh read-receipt tag |
| `doc/rules/observability-and-logging.md` | 3.3.2, 4.3, 5.3.1, 6.1 | the value `reason` of the run-ended line; `RunEnded.Failed`; the three events for the slots (`YieldShocked`, `TransitFactorChanged`, `PopulationScaled`); `TickEnded.Failed`: every started step is closed (a panic emits nothing), a failed one with `Failed` true, and 6.1 closes the span at `TickEnded`; 4.4 and architecture.md 12.5 "after the transition it describes" read "the end of the step" for a failed step, the world being failed before the event (3.5.6); a fresh tag |
| `doc/notes.md` | the open notes of 20 | close N-009, N-056, N-074, N-117; update N-031, N-036, N-046, N-051, N-082, N-111, N-016, N-017, N-027, N-103, N-012, N-020, N-045, N-075, N-091, N-019, N-022, N-079, N-084, N-092; add the new notes of the handoff; the index; a fresh tag |
| `doc/overview.md` | 143 and its `Run` sample | `Run(ctx, ticks)` returns two values (3.4) |
| `doc/adr/0002-persistence-by-checkpoint.md` | open point 4, decision 7 | open point 4 answered: between ticks only (8.4); an extension's own state is saved as decision 7 says, and its registration is not (12.1) |
| `doc/adr/0005-delivery-contracts-are-matched-by-a-first-fit-round.md` | the position sentence | the position is P3.3 (6.5) |
| `doc/spec-roadmap.md` | the WLD row (line 181), the tier tables and the existing-specs table | `exists` |
| `doc/documentation_index.md` | rows | `spec-world.md` and `models/wld/` |

## 23. Where this lives in the code

| Concern | Home |
|---------|------|
| `World`, `NewWorld`, `Step`, `Run`, `Err`, `Tick`, `Config`, `ConfigHash`, the failed state and the in-step guard | root package, `world.go` |
| The eight phases and their positions, each a method that calls into the module files | root package: `tick.go` holds the list and the straight line; the methods are in `phase_start.go` (P0, P1), `phase_production.go` (P2), `phase_trade.go` (P3, P4), `phase_outcomes.go` (P5, P6) and `phase_exit.go` (P7, P8) |
| `world.go` | the type, `NewWorld` steps 1 to 4, the readers, `Step`, `Run` and the guards; `construct.go` holds steps 5 to 10 and `construct_warm.go` the application of the warm start |
| The fan-ins (architecture.md 2.10.4), the executor of the spawner's plan and `removeAgent` | root package, `fanin.go`; `market.go` holds the market records and `snapshot.go` the snapshot |
| `Option`, `options`, `WithSeed`, `WithLogger`, `WithObserver`, `WithAudit`, `WithTargetSpread`, `WithStrategy`, `WithSpawner`, `WithClearing`, `WithPolicy` | root package, `option.go` |
| The seed, the one PCG and `Rand`, the site constants and the stream key (reseed at a site), the world's draw and the pure reseed function | root package, `random.go` (a small struct holding the seed, the PCG and the Rand; the key arithmetic is `internal/ordering`, which receives and returns plain `uint64`, architecture.md 2.9) |
| The sentinels `ErrUnknownClearing`, `ErrUnknownPolicy`, `ErrNilExtension`, `ErrWorldFailed`, `ErrInStep`, `ErrAuditFailed` | root package, `errors.go` |
| `WorldCreated`, `RunStarted`, `RunEnded`, `TickStarted`, `TickEnded` | root package, `event.go` |
| `Weyl32`, `Weyl2`, `Bucket(id, P)`, the finaliser, `Key(t, id)` (shared with the ration of RES) and the stream key of 7.1.3 | `internal/ordering` |
| The construction order, the warm start application, the creation draws | root package, `world.go` calling `internal/ledger` and the modules |
| The models | `doc/specs/models/wld/` |
