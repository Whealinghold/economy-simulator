# Spec: Policy: policy actors, their levers and the interface that sets them

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-PLC-NNN` and
`NFR-PLC-NNN`, which tests cite (testing.md section 4). They are in section 18.

Layer: module, later stage ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)
decisions 2 and 4, [ADR 0003](../adr/0003-version-1-scope.md) decisions 2 and 3). It is
off unless `Config.Modules.Policy` is present, and the core does not depend on it. A
world with the module off is exactly the world without it (FR-PLC-001). It needs
`spec-polity.md` (3, 4, 5, 6.3: the jurisdiction, `Regions`, `Between`, the treasury,
hook H7), `spec-integration.md` (3, 5: the slots, the order of checks, the shape of a
setter), `spec-world.md` (6, 7, 8, 9: the tick and its positions, the stream, the rule
that a slot acts between ticks, the devices of a cadence, "no rate from an aggregate"),
`spec-observability.md` (5: the region state of a snapshot) and `spec-checkpoint.md`
(4, 7: the sections and `Stateful`). It reads the owners of the slots a lever can reach
(`spec-money-supply.md` 4.4.3, `spec-fiscal.md` 13, `spec-regions.md` 7,
`spec-transport.md` 10.3). It is read by the specs after it: the register's loops, the
scenarios of STY and the first consumer's player.

**Every number in this spec is a placeholder** except where a derivation is shown.
Nothing has been run against the library, bazaarBot has not been run and no acceptance
band exists (notes N-015, N-027). The spec sets no default macro constant: a module with
no actor does nothing, and every cadence, bound, target and gain is written by the pack.
The models of section 16 are small Docker Python models of the rules of this spec (a
controller on a plant with a delay, the integer rule with and without its dampers, and
the arithmetic of the examples); they are not claims about the library and no default is
claimed to be stable.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017). The
setting words of the examples (`crown`, `inland`, `coast`, `uplands`, `capital`,
`farm`, `harbour`, `hills`, `mill`, `loaf`, `grower`, `miller`, `head_tax`, `stipend`,
`gate`, `bread_stipend`, `*_council`) appear only in the **flagged examples** of section 14,
in the sections that refer back to it (15, 16, 20) and in sentences marked "for example"; they are data, never branched on (architecture.md 9.1; `AGENTS.md`,
setting-agnostic). The word **policy** has two senses here and the spec keeps them
apart: a **policy** is a value of the interface of section 4 (the code that decides),
and a **policy actor**, or **actor**, is a row of the configuration that a policy is
attached to. The word **rule** is the built-in reactive policy (8.3). The word
**lever** is new (3.4): it is the only thing an actor can move.

## 1. Purpose and scope

The engine measures and never sets (ADR 0001 decision 7, `spec-world.md` 9.4). Every
macro quantity that a setting needs to vary over time (a tax rate, a payout, a
release, a closed border) is already a named, bounded value that a consumer sets
between ticks: the flow scale of `spec-money-supply.md` 4.4.3, the edge status of
`spec-regions.md` 7, the transit factor of `spec-transport.md` 10.3 (the slot table of
`spec-integration.md` 5.1). What was left open is **who chooses it**: a fixed schedule,
a rule that reacts to a measure, or a player. ADR 0001 decision 4 put that choice in
one hook that reads measures and returns typed settings, and ADR 0003 decision 3 and
N-025 decided that it is its own interface and not `Strategy`. `spec-fiscal.md` 13 and
`spec-polity.md` 5.4.2 and 6.3 (H7) each say "PLC". This spec decides it. It covers:

- the **policy actor**: a row of configuration with an ID, one jurisdiction, a policy
  by name, a cadence and a list of levers; what it is not (section 3);
- the **lever**: a named, bounded and rate-limited handle on one slot key or on a
  group of them, which is both the mandate of an actor and the only thing it can move
  (3.4, 6);
- the **interface** `Policy`: one method, the view it receives, the settings it
  returns, what the world checks, registration, its default, its randomness and its
  state (section 4);
- what an actor **reads**: the state of the regions its jurisdiction covers at the
  boundary, its treasury, its own levers, and nothing else (section 5);
- what an actor **sets** and how each returned setting is checked and applied
  (section 6);
- **when** it acts: a position at the entry of the tick, a cadence by rotation, no
  draw in the built-ins (section 7);
- the four **built-in policies**: `hold` (the default), `schedule` (a fixed schedule),
  `rule` (a reactive rule on four gauges) and `directed` (a player) (section 8) and
  the call a player uses (section 9);
- parameters, validation, observability, checkpoint state, worked examples, the
  entries for the stability register, the models run, notes, requirements, open
  questions and where it lives in the code (sections 10 to 20).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The jurisdiction, the chain, the crossing, `Between`, the seat and the treasury account and agent | `spec-polity.md` (POL) |
| What a tax, a duty, a toll, a payout or a standing order is, its rate and its base | `spec-fiscal.md` (FIS); the flow scale is its lever |
| Creating or destroying money (an authority's mint, a tribute paid abroad) | `spec-money-supply.md` (MON): a grant or a levy whose target is a treasury, scaled by a lever |
| The edge status, the transit factor, the flow scale themselves: their arithmetic, bounds, events and saved state | REG 7, TRN 10.3, MON 4.4.3 and FIS 13 (the owners) |
| The public call that sets a slot, its errors and the sentinels | `spec-integration.md` (INT); this spec adds one call, `DirectPolicy` (9) |
| A charter, a monopoly, a price control | `spec-institutions.md` (ORG): its legal level is a slot and can be a lever (6.3) when ORG is on |
| Measures: an index, a share, a mean over a set of regions | `spec-indicators.md` (IND): preset asks in 17.2 |
| An authority that bids, offers or trades | `spec-fiscal.md` 9 (a treasury's standing orders, scaled by a lever) or `spec-agents.md` 10 (an external agent with `Directed`); not a policy (9.2) |
| The tick, the draws, the options | `spec-world.md` (WLD) |
| Whether any policy is stable | the register and the scenarios (15); STY and EXP own the bands and the sweep |

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | Does a policy actor reuse `Strategy`? | **No.** It has its own interface, `Policy`, with one method, `Decide`. It returns settings of levers; it does not bid, offer, produce or move | ADR 0003 decision 3, N-025: the signature and the returned type differ (`spec-agents.md` 6.2). One `Strategy` for every actor that bids, one `Policy` for the rest. **Closes N-025** |
| 2 | What is an actor? | A row of `Config.Modules.Policy`: an ID, a jurisdiction, a policy name, a cadence and a list of levers. It is **not an agent**: no account, no agent ID, no household count, no role, no upkeep, no belief, no draw of its own | POL 5.3.3: the treasury is a pseudo-agent so that others can name it; an actor needs no identity in the ledger, because it moves nothing there. An `ActorID` is its index in ascending ID order |
| 3 | What can an actor change? | **Only levers**: a named, bounded, rate-limited handle on a slot key. The lever list of the actor is its mandate. No transfer, no goods, no ledger call | ADR 0001 decision 4 lists three actions (a transfer, a rate, stock to a boundary). Each is a declared flow with a scale (MON 4, FIS 8); a policy moves the scale. The ledger therefore never sees a policy: it sees a flow whose scale changed, with the same checks as before |
| 4 | Which slots? | The ones an authority would hold: a flow scale (MON, FIS), an edge status (REG; a border, an embargo) and a transit factor (TRN), and, when their modules are on and INT lists the slot, a rate scale (CRD), a rent scale (LND) and a legal level (ORG). Not the shocks of the story (a yield shock, a regeneration or capacity scale, an endowment adjustment, an outside price shock, a birth or death scale) | A shock is what happens to a jurisdiction; a lever is what its authority does. The split is `spec-integration.md` 9.1's: a consumer's story keeps its own calls. The list is a closed table (6.3) and a new kind is a row there and a slot in INT |
| 5 | When does it act? | At **P0.3a**, the entry of the step after `TickStarted`, on a **cadence by rotation** (`Every` ticks, the bucket of the actor's index). Its settings act from the first position that reads the slot, exactly as if the consumer had made the call between the two steps | WLD 8.1: a slot is set between ticks and read at its position. P0.3a is before every position that reads a slot (P1.2 is the first), so the consumer's call and the actor's write are indistinguishable to the tick (7.1). A new position takes a letter after the one it follows, so no one renumbers (WLD 6.1.1) |
| 6 | What does it read? | The **boundary state** of the regions the jurisdiction covers (`RegionState`, `spec-observability.md` 5.3.5a), its treasury's balance, its own levers and the tick. Nothing else: no event, no other region, no live state of a step, no aggregate that the engine keeps (it keeps none) | A snapshot field is a function of the saved state (OBS 5.2.1), so the same inputs reproduce after a restore without an extra saved byte. An actor sees its jurisdiction; an outsider's prices arrive as a rumour for a household, not for an authority (open question 3) |
| 7 | Does the library read an indicator? | **No.** `spec-indicators.md` 3.1 and FR-IND-001 hold: nothing in the module imports `indicator`. A consumer's own `Policy` may import it and compute any preset from the view it was given. The built-in `rule` reads four **gauges**, each a field read or one weighted ratio over the view, with the definition of the IND preset it matches (8.3.2) | The brief says "IND measures". The import rule says the engine cannot call them. Both stand: the measure is IND's definition, the call is the consumer's, and the four gauges are the part of a measure that is arithmetic over a snapshot. A conflict logged for the supervisor (17.3) |
| 8 | Which policies ship? | Four, registered by name: `hold` (the default, returns nothing), `schedule` (a step function of the tick, data), `rule` (a bounded stepping controller on one gauge per lever, data) and `directed` (the player's mailbox) | ADR 0001 decision 3: "choose among the built-in rule shapes (fixed, proportional to a measure, threshold, scheduled)". The consumer's own is a fifth, registered with `WithPolicy` (4.5) |
| 9 | A player as an actor? | A `directed` actor and one call, `DirectPolicy`: a directive per actor, applied at that actor's next decision tick through the same checks as any policy. A consumer with no use for a mandate or a cadence calls INT's setters instead | A game's governing player should be unable to change a key outside its jurisdiction and to change one at any tick it likes; a mandate and a cadence are what the actor adds (9.1) |
| 10 | Who wins when two writers meet? | One **holder** per key: a key (or a key of an expanded group) belongs to at most one lever of one actor (`ErrLeverShared`). A consumer's call to a held key is legal and is a write like any: the last write wins, and the actor writes only keys it returns | One owner per fact; a loop with two controllers on one lever is the hazard of PLC-A3, and it is refused where it can be seen in the data |
| 11 | Randomness | The built-ins take **no draw**. A consumer's `Policy` receives the world's `*rand.Rand` (architecture.md 7.7), and its words are counted in call order as D17 of the table of `spec-world.md` 7.3 | One stream, positional (WLD 7.6); a rule that draws is a consumer's |
| 12 | State | The world keeps **none** for the built-ins except the mailbox of a `directed` actor (13). The value of a lever is the slot's own state, saved by its owner. A consumer's stateful `Policy` implements `Stateful` and is saved as a fourth kind of extension | Persistable by design (ADR 0002 decision 6, 7): a rule that steps the lever it reads needs no memory of its own |
| 13 | Is it a faucet or a sink, or does it change the ledger? | **Neither.** It adds no reason, no account and no flow. The conservation identities hold after every write | N-021: one place for a faucet or a sink. A scale is bounded by its owner and its flow's own limit |
| 14 | Does it add a feedback loop? | **Yes, when the policy reacts.** The `rule`, and any consumer policy that reads the view, is a controller with a gain and a delay (15, PLC-A1). The library guarantees conservation and determinism of every setting and **not** the stability of the loop it closes (N-131, STB-L121). `hold`, `schedule` and `directed` close none | `spec-stability.md` 9: a spec that adds a loop and no entry is not done. The register carries the condition and the model supports it |
| 15 | Can a policy change a cadence, a window or a gate? | **No.** No lever kind is a period, a probability or a window | WLD 9.4.3: a cadence is configuration, constant for a run; "no rate from an aggregate" binds the library's own rates, and a lever is not one of them |
| 16 | Does the module change a result by being on? | **No while no actor is due.** An actor that is due and returns nothing changes nothing and draws nothing (the built-ins) | The module is a pure addition at P0.3a (FR-PLC-001); a registered consumer policy that draws changes the stream by its words, in call order (D17) |

## 3. The module, the actor and the lever

### 3.1 The module

3.1.1. `Config.Modules.Policy` is a pointer; `nil` is off (`spec-config.md` 3.2). When
off, nothing in this spec runs: no position, no state in the world or in a checkpoint,
no event, no draw and no extra snapshot field. It needs `Modules.Polity` with at least
one jurisdiction (`ErrModuleDependency`): an actor is attached to one. A non-nil module
with no actor is legal and does nothing (the warning `no_actors`).

3.1.2. The module is plain data, each list sorted by ID, no maps:

```go
// PolicyConfig is the module. Its presence switches it on.
type PolicyConfig struct {
	Actors []PolicyActor // sorted by ID, at most MaxActors
}

// PolicyActor is a policy actor: one policy attached to one jurisdiction.
type PolicyActor struct {
	ID           string         // [a-z][a-z0-9_]*, at most 64 bytes, unique among actors
	Jurisdiction string         // a jurisdiction of Modules.Polity; at most one actor each
	Policy       string         // a registered name: "hold", "schedule", "rule", "directed", or a consumer's
	Every        int64          // ticks between decisions, 1 to MaxPolicyEvery; required
	ReadsRegions bool           // the view carries the RegionState of the covered regions
	Levers       []Lever        // sorted by ID, at most MaxLevers
	Schedule     []ScheduleStep // only with Policy "schedule"; sorted by (Lever, At)
	Cycle        int64          // only with Policy "schedule": 0 none, else the length of a repeating cycle in ticks
	Rules        []Reaction     // only with Policy "rule"; sorted by Lever
	Label        string         // presentation only
}

// Lever is a handle on one slot key, or on the group of keys that Against expands to.
type Lever struct {
	ID       string    // [a-z][a-z0-9_]*, unique within the actor
	Kind     LeverKind // "flow", "edge" or "transit" (6.3)
	Flow     string    // kind flow: a money flow or a fiscal flow ID
	From     string    // kind edge: a region ID; kind transit: a region ID, not an outside node, not equal to To
	To       string    // kind edge: a region ID; kind transit: a region ID, not an outside node
	Mode     string    // kind transit: a transport mode ID
	Against  string    // kind edge, instead of From and To: a jurisdiction ID (3.4.3)
	Min      int64     // the lowest value the actor may set
	Max      int64     // the highest value the actor may set
	MaxStep  int64     // the largest change of the value in one decision; 0 means no limit
	MayClose bool      // kind edge: the actor may set Closed
}

// ScheduleStep is one step of a fixed schedule (8.2).
type ScheduleStep struct {
	Lever  string // a lever of the actor
	At     int64  // the tick from which the value holds
	BP     int64  // the value the lever is set to
	Closed bool   // kind edge, with MayClose
}

// Reaction is one reactive rule on one lever (8.3).
type Reaction struct {
	Lever     string // a lever of the actor; at most one reaction each
	Gauge     Gauge  // "treasury", "unfed_share", "mean_ratio" or "price" (8.3.2)
	Commodity string // gauge price: the commodity ID; empty otherwise
	MaxAge    int64  // gauge price: the oldest price age that counts, in ticks; 0 otherwise
	Target    int64  // the reading at which the rule rests, in the gauge's unit
	Deadband  int64  // the half-width of the rest band, in the gauge's unit
	Span      int64  // an error of Span gauge units moves the lever by Move; at least 1
	Move      int64  // basis points of lever value per Span of error
	Raise     bool   // true: the lever rises when the reading is above Target; false: it falls
}
```

3.1.3. **Hash and file.** The module is hashed, when present (a module that is off writes nothing: `spec-config.md` 8.2.1),
the actors in ID order with every field except `Label`; the policy is hashed by its
**name** only, as a strategy is (`spec-agents.md` 6.6): the code behind a name is not
part of the hash and a consumer that changes it breaks bit-identity knowingly. The
file form is tables keyed by ID, so that overlay documents merge by key
(`spec-config.md` 6); a schedule step is keyed by lever and tick and a reaction by
lever (**an example, in the invented setting of 14**):

```toml
[modules.policy.actor.crown_council]
jurisdiction = "crown"
policy       = "schedule"
every        = 10

[modules.policy.actor.crown_council.lever.poll]
kind     = "flow"
flow     = "head_tax"
min      = 0
max      = 20000
max_step = 5000

[modules.policy.actor.crown_council.schedule.poll.120]
bp = 15000
```

### 3.2 The actor

3.2.1. An actor is attached to **one jurisdiction**, governed or not. If it is
governed, the actor's view carries the balance of its treasury (`HasTreasury` true);
if it is a grouping, `HasTreasury` is false and a gauge that needs one is undefined
(`ErrNotGoverned` at validation for a `treasury` gauge, 11.1). **At most one actor per
jurisdiction** in version 1 (`ErrPolicyShape`, rule `one_actor_per_jurisdiction`): this
is POL H7's "one policy per governed jurisdiction", and a player who rules several
jurisdictions is several actors directed by one call (9.1).

3.2.2. An actor has no agent ID and appears in none of the agent counts, none of the
rotations of `spec-world.md` 6.1.4 and no tie-break of the library. Its identity is its
`ActorID`, the index of its `ID` in ascending bytewise order:

```go
// ActorID is the index of a policy actor in ascending order of its ID string. It
// is not stable across configurations; use the string ID to persist or exchange one.
type ActorID int

// LeverID is the index of a lever in the ascending order of the actor's lever IDs.
type LeverID int
```

3.2.3. An actor acts only inside the jurisdiction's reach (3.5): the structure of the
tree decides which keys a lever may name, and the pack's lever list says which of them
this actor holds. The tree is POL's and is fixed for a run (POL position 9), so the
reach is fixed too.

### 3.3 What an actor is not

3.3.1. It holds no aggregate (a revenue, a price level, a count of households): the
reading of a gauge exists for the length of one call (4.3). It places no order, takes
no relief, pays no upkeep and is never passed to a strategy. It has no account: the
treasury of its jurisdiction is POL's pseudo-agent and FIS's counterparty, and the
actor does not own it. It does not tax: a flow is FIS's, and the actor moves its scale.

### 3.4 The lever

3.4.1. A **lever** is the one thing an actor can move. It names a slot key (or a group)
and three bounds, all data:

| Field | Meaning | Rule |
|-------|---------|------|
| `Kind`, and `Flow`, `From`, `To`, `Mode` or `Against` | what the lever is attached to (6.3) | the key fields of the kind exactly (`ErrPolicyShape`, rule `lever_key`) |
| `Min`, `Max` | the values the actor may set | `Min <= Default <= Max`, both inside the slot's own bound (`ErrOutOfBounds`); the **default** is the value of an unset key (`spec-integration.md` 5.2.1) |
| `MaxStep` | the largest change per decision | 0 is no limit; otherwise at least 1 |
| `MayClose` | kind edge only: the actor may set `Closed` | an embargo is a mandate, so it is a flag of the lever |

3.4.2. **A lever has one value**, a number of basis points: the scale of a flow, the
factor of an edge's length, the factor of a transit. A lever of kind edge also has a
`Closed` flag. `Min`, `Max` and `MaxStep` act on the number only; `Closed` is a switch
that the actor may use only with `MayClose`, and the cadence is its only limiter.

3.4.3. **A group.** A lever of kind edge with `Against` names a jurisdiction `K` instead
of an edge: it holds every edge of `Between(g, J, K)` and of `Between(g, K, J)` for
the actor's jurisdiction `J`, computed once at construction from the polity and the
graph (POL 4.2.3, H13), in ascending `EdgeID`, and moves them as one. This is how a
pack writes "an embargo of one province against another" without listing edges. `K`
must not be `J`, an ancestor or a descendant of it (their regions are nested, no edge
crosses: `ErrPolicyShape`, rule `against_nested`). A group that expands to no edge is
allowed and draws the warning `lever_no_keys`.

3.4.4. **The value read.** The current value of a lever is the value of its slot key; of a
group, that of the first key in ascending order. The actor writes every key of the
group, so after one write they are equal; a consumer's call that touches one key of a
group makes them differ, and the view reports the first (the rule is fixed so that the
reading is a function of the state). The checks of 6.2 are made against **every** key: an
entry is `unchanged` only when every key of the group already has the value, and `Closed`
is a closing (check 3) unless every key is already closed.

### 3.5 Reach: which keys a lever may name

3.5.1. The reach is structural, checked at validation (`ErrLeverNotInReach`, with the
path of the lever and the key). `J` is the actor's jurisdiction, `Covers` and the
descendants are POL's (4.2):

| Kind | The key | In reach when |
|------|---------|---------------|
| flow | a fiscal flow ID | its authority is `J` or a descendant of `J` (a parent may set a child's rate) |
| flow | a MON flow ID | every region its target selects (every region if it names none) is covered by `J`; a target kind `treasury` (MON edit, `spec-fiscal.md` 4.2): the treasury is that of `J` or of a descendant |
| edge | `(From, To)`, a configured edge | `J` covers `From` or `To` |
| edge | `Against K` | by construction (3.4.3) |
| transit | `(From, To, Mode)`: a directed pair of regions (adjacent or not, as TRN 4.1 keys it) and a mode; `From` is not `To` and neither is an outside node (INT 5.1.3), so the owner's setter cannot refuse it | `J` covers `From` or `To` |

3.5.2. **One holder per key** (position 10). After the expansion of every `Against`,
the keys of all levers of all actors are pairwise distinct, within an actor
(`ErrDuplicateID`) and across actors (`ErrLeverShared`, with the two actors). A flow
whose scale an actor holds and a flow with the same ID are the same key: MON's and FIS's
IDs are one namespace (`spec-fiscal.md` 3.1.1).

3.5.3. **What reach is not.** It is not a law: it does not say that a rate "applies"
somewhere (a flow's own region and class do), and a jurisdiction's actor cannot set a
key it does not cover. A consumer's call is outside it (INT checks only the owner's
bounds), which is the point of a mandate being the actor's and not the world's.

## 4. The interface

### 4.1 `Policy`

```go
// Policy decides the settings of one policy actor, at the actor's decision tick.
// It returns settings of the actor's levers; it does not bid, offer, produce or
// move money. Decide is called between the previous step and the first position
// that reads a slot (P0.3a), on the world's goroutine, once per due actor, in
// ascending ActorID.
//
// Randomness: an implementation MUST take it from rng and MUST NOT create a
// source, use a global one or keep rng after Decide returns (architecture.md 7.7).
// State: an implementation that keeps state between calls MUST also implement
// Stateful, MUST NOT share an instance between worlds and MUST NOT read the clock,
// the environment, a map's order or an address (architecture.md 7.1, 7.4). The
// view is a deep copy the implementation may keep or change; the returned Settings
// is copied by the world.
type Policy interface {
	Decide(v PolicyView, rng *rand.Rand) Settings
}
```

4.1.1. **One method** (architecture.md 4.2). There is no `Revise`, no `Name` and no
per-event callback: a policy that wants memory keeps it (`Stateful`), and a policy that
wants an event keeps no observer (4.3.3).

4.1.2. **It returns no error.** An extension cannot fail a step (WLD 3.3.3): what it
returns is checked, and what fails a check is counted and dropped (6.2). A panic of a
consumer's `Decide` propagates and fails the world as any extension's does
(architecture.md 12.5; WLD 3.5.4).

### 4.2 The view

```go
// PolicyView is what Decide receives: plain data, a deep copy built for this call.
type PolicyView struct {
	Tick         int64          // the tick the step runs; Snapshot().Tick of the boundary
	Moment       Moment         // Calendar().At(Tick)
	Actor        ActorID
	Jurisdiction JurisdictionID
	Seat         RegionID       // NoRegion for a grouping
	HasTreasury  bool           // the jurisdiction is governed
	Treasury     Money          // the treasury's balance at the boundary; 0 without one
	Levers       []LeverView    // by LeverID
	Regions      []RegionState  // the regions the jurisdiction covers, ascending RegionID; empty unless ReadsRegions
}

// LeverView is one lever as the actor holds it now.
type LeverView struct {
	Lever    LeverID
	Kind     LeverKind
	BP       int64 // the current value (3.4.4)
	Closed   bool  // kind edge: whether the key is closed now
	Default  int64 // the value of an unset key
	Min, Max, MaxStep int64
	MayClose bool
}

// Settings is what Decide returns: values for some of the actor's levers. A
// lever with no entry keeps its value (it is held).
type Settings struct {
	Levers []LeverSetting
}

// LeverSetting asks that one lever take a value: the whole value, BP and Closed, as an
// INT setter takes a whole value (spec-integration.md 3.3.2). To change BP of a
// closed edge and keep it closed the entry carries Closed true (which is not a closing,
// 6.2.1 check 3).
type LeverSetting struct {
	Lever  LeverID
	BP     int64
	Closed bool // kind edge: closing a key needs MayClose; true on another kind is refused (no_close)
}
```

4.2.1. **What the view is built from.** `Tick`, `Moment` and `Levers` are derived;
`Treasury` is the balance of the ledger account at the boundary; `Regions` are the
region states of OBS 5.3.5a, **equal to the filter of a full snapshot by region**
(FR-OBS-044), because that is what a consumer's own indicator would have been given.
The world-level rows of a snapshot (`Boundary`, `Flows`, `Pending`) are not in the view:
an actor reads its jurisdiction, and a flow's own state is the lever's current value.

4.2.2. **Ordering and aliasing.** Every slice is in the canonical order of its key. The
view is built fresh for each call and shares no storage with the world or with another
actor's view.

### 4.3 What the contract forbids, and why

4.3.1. **No events.** The view has no event and the interface has no observer
callback. An event exists only when an observer was supplied (observability-and-logging.md
4.5), so a policy that read events would give a different result with and without an
observer, which is forbidden (architecture.md 11.3; observability-and-logging.md 1.4).
A consumer that wants a flow over a window computes it from **two views** (its own
`Stateful` memory of the previous reading) or from the cumulative totals the region
state carries; that is a consumer's model, and its loop is PLC-A1's.

4.3.2. **No `*World`.** The implementation receives plain data and never the world
(`spec-agents.md` 6.1.1): a call from inside a step is `ErrInStep` anyway
(`spec-world.md` 3.6), and an actor that could read live state would read a half-built
tick.

4.3.3. **No indicator through the world.** A consumer's implementation may import
`indicator/` and compute a preset from `v.Regions` (or from its own snapshot, if it
holds the world, which it must not do inside `Decide`). The library never calls it
(position 7). IND's rule that nothing in the world reads an indicator back (3.1.1 there)
stands because the caller is the consumer's code, in the consumer's own package.

### 4.4 The default and the compile-time assertion

4.4.1. The default is `HoldPolicy`, which returns no settings and keeps no state
(ADR 0001 decision 4: "the default does nothing"):

```go
var _ Policy = HoldPolicy{}
```

The other built-ins, `Schedule`, `Rule` and `Directed`, are values of the root package
built from an actor's row (8), each with the same assertion. They are registered under
their names by the world and are not options: `hold`, `schedule`, `rule` and `directed`
are reserved names (`ErrDuplicateID` for a consumer that registers one, as `default`
and `directed` are for a strategy, `spec-world.md` 4.3).

### 4.5 Registration

4.5.1. `WithPolicy(name string, p Policy) Option` registers a consumer's policy under a
name, additively, in the shape of `WithSpawner` (`spec-world.md` 4.1): a name given twice
or a reserved one is `ErrDuplicateID`, a nil policy `ErrNilExtension`. An actor names it
in `Policy`. A name that is neither built in nor registered is `ErrUnknownPolicy`, a
`ValidationError` with the path `modules.policy.actors[i].policy`, raised by `NewWorld`
after `Validate` succeeds (the name checks of `spec-world.md` 3.2.2).

4.5.2. The registration is **not** in the hash and the name is. A world built with a
different implementation under the same name differs silently, which is the same
hazard as a strategy (`spec-agents.md` 6.6, open question 4); the `Stateful` version
number is the only code identity the library compares (`spec-checkpoint.md` 7.3.3).

### 4.6 State: `Stateful`

4.6.1. A `Policy` that keeps state between calls MUST also implement `Stateful`
(`spec-checkpoint.md` 7). The extension kinds of the format gain a fourth value,
`3` policy (`spec-checkpoint.md` 4.2.4, 7.1.2). The built-ins keep no state of their
own: `directed` keeps its mailbox **in the world** (13), so that the checkpoint saves
it whether or not a consumer registered anything, and `hold`, `schedule` and `rule` have
none.

### 4.7 The amendment to architecture.md 4.3 and 7.7

4.7.1. A new extension interface requires an update to the table of architecture.md 4.3
and a default implementation and a compile-time assertion (4.5 there). The exact text
for the supervisor to apply and tag (an owed row, 17.2):

> **4.3 table**, a fifth row:
>
> | `Policy` | decides the settings of a policy actor's levers on its cadence; it returns settings and never bids; the default holds every lever |
>
> **4.3, Extension state**: "An implementation of `Strategy`, `Clearing`, `Spawner` or
> `Policy` that keeps state between calls MUST also implement the optional interface
> `Stateful`".
>
> **7.7**: "Methods on `Strategy`, `Clearing`, `Spawner` and `Policy` that need it receive
> the world's `*rand.Rand` as a parameter"; the contract paragraph on `Stateful` names
> `Policy` with the other three.

4.7.2. ADR 0001 decision 4 spoke of a "closed set of typed actions (transfer between
accounts, change a rate, add stock to a boundary)". This spec narrows it to one action,
**set a lever**, and says why in position 3: the other two are a flow's scale. The ADR's
open point 1 ("a fifth extension point or `Strategy`") is answered: a fifth.

## 5. What an actor reads

### 5.1 The list

5.1.1. A policy reads exactly this, in the view of 4.2, and nothing else. Each row says
where it comes from, so that none is an aggregate the engine keeps (ADR 0001 decision 7):

| Field | Source | Cost class (OBS 5.5) | Notes |
|-------|--------|----------------------|-------|
| `Tick`, `Moment` | the tick counter, `Calendar.At(t)` | D | derived, never state |
| `Actor`, `Jurisdiction`, `Seat`, `HasTreasury` | the configuration and the polity | D | structure |
| `Treasury` | the balance of the Treasury account at the boundary | K | POL 5.5.1 |
| `Levers` | the actor's row and the slots' current values | K | the slot's own state, saved by its owner |
| `Regions` | the region state of each covered region that is not an outside node (an outside node's row has `Outside` true and zero fields) | S, L, B by the row | `ReadsRegions` only; the filter of a full snapshot by region |

5.1.2. **What a region state holds**, for the policy's author (OBS 5.3.5a): the region
row (agents, households, `MoneyHouseholds`, `MoneyPool`, `MoneyContract` and, with POL,
`MoneyTreasury`, POL 10.5), the role
rows (`Unfed`, the mean welfare ratio, the observed wage), the market rows (the published
price, its age, the volume and unmatched units of the last clearing, the price window,
the stock held by agents and by the estate and, with FIS, by the treasury), the jobs,
the endowments, and the shipments and journeys that leave or reach the region. It holds
no agent record and no event.

### 5.2 The rule: a function of the saved state

5.2.1. Every field of the view is a function of the configuration, the tick and the
state that a checkpoint saves (OBS 5.2.1). A restored world therefore builds the same
view at the same tick, a policy needs nothing saved to resume, and the non-interference
test of observability-and-logging.md 8.1 holds: a run with a recording observer and a run
with none have the same views.

5.2.2. The view is the **boundary state**: what the previous step left at P8.6, plus the
calls the consumer made since (a flow scale it set, an agent it added). It is not the
state of the previous tick's events: a count of what happened in the last tick is an
event and not a snapshot field (OBS 5.2.2), and the view has none. An actor that wants "the
unfed households of the last tick" has the households unfed now (`RoleRow.Unfed`),
which is the last consumption's figure.

### 5.3 What an actor does not see

5.3.1. **Another jurisdiction's regions**, unless it covers them. A parent covers its
children's regions, and an actor reads all the regions it covers. A neighbour's price is not
in the view: that is information (INF), and a household learns it by rumour. An actor
that must react to a neighbour reads the market rows of the border region it covers and
the shipments that reach it.

5.3.2. **Another actor's levers or its settings**, the world's random state, an agent, an
order, a belief, or the registered extensions. The levers of an actor are its own.

5.3.3. **A prediction.** The view is the present. A policy that wants a forecast makes
it itself, with its own model, and its loop is PLC-A1's.

### 5.4 Cost

5.4.1. The cost of a decision is the cost of its view: `O(levers)` with `ReadsRegions`
false, and with it true the scans that the region states of the covered regions cost
(OBS 5.5: the S rows are `O(agents of the region)`, the L rows `O(lots)`, the B rows
`O(beliefs)`). The cost per tick is that divided by `Every`, so **the cadence is the
cost lever** (NFR-PLC-002). For the first consumer's size (five regions, about 5 000
agents, one actor over all of them at `Every` 30) it is one snapshot's scan in thirty
ticks, a counted figure and not a measurement (N-015).

5.4.2. A policy that reads no region (`hold`, `schedule` and `directed`, and a `rule`
whose only gauge is `treasury`) sets `ReadsRegions` to false and pays nothing for it. The
warning `reads_unused` names a configuration that sets it and uses neither a `rule` with
a region gauge nor a consumer's policy (11.2).

## 6. What an actor sets

### 6.1 The step

6.1.1. At its decision tick (7.2), the world, for the actor and in this order:

1. builds the view (5);
2. calls `Decide` (the built-ins are the world's own code and take no draw);
3. checks each entry of the returned `Settings` in the order returned (6.2) and
   computes the value each lever would take;
4. applies **all** the changes of the actor at once, through the internal form of each
   owner's setter, in the order of 6.4.2 (edge, transit, flow) and then the canonical key order of
   the owner (6.4);
5. emits the slot events of the application and then `PolicyDecided` (12.1).

6.1.2. **Atomic.** The checks of step 3 are complete before anything is applied, and no
check can fail afterwards: a lever's bounds lie inside its slot's own bound
(3.4.1) and every key check of INT 5.1.3 is made at validation (a configured edge, a region pair that is
distinct and not an outside node, a defined mode and flow, 11.1), so the owner's setter cannot refuse a
value that passed (a programmer error if it does: a panic with a message that says so, architecture.md 8.1, as `spec-world.md` 5.1.1).
An actor's application is therefore either complete or absent.

### 6.2 The checks

6.2.1. Each entry gets **one outcome**, from the first of these that holds, in this order.
`Current` is the lever's value when step 3 began (of a group, that of its first key, 3.4.4); the world reads it once and writes the result to every key of a group that differs from it.

| # | Check | Outcome | What happens |
|---|-------|---------|--------------|
| 1 | `Lever` is not in `[0, len(Levers))` | `not_held` | dropped |
| 2 | a lever already had an entry in this list | `duplicate` | dropped; the first entry stands |
| 3 | `Closed` is true, and it would close a key: the lever has no `MayClose` or is of another kind, and some key of the lever is open now (`Closed` false is never refused: opening is not an embargo; `Closed` true on a key that is already closed is no closing, so a `rule` or a consumer's own `Policy` can move the factor of an edge a consumer closed and keep it closed) | `no_close` | dropped whole (the entry's number is not applied either: a refused entry is refused whole) |
| 4 | otherwise the value is `BP` cut to `[Min, Max]`, then to `Current ± MaxStep` when `MaxStep` is above 0, then the lever takes it | `applied` if any key of the lever would change (its number or its `Closed` differs from the result), else `unchanged` | the cuts are flags: `Clamped` when the first cut moved the value, `Stepped` when the second did |

6.2.2. **The order of the two cuts** is fixed: bounds first, then the step against
`Current`. The result is inside `[Min, Max]` and within `MaxStep` of `Current` whenever
`Current` itself is inside the bounds. If `Current` is **outside** `[Min, Max]` (a
consumer's call put it there), the bounds cut wins and the step is taken from `Current`
toward the bound: the lever is brought back inside by at most `MaxStep` a decision. This
is a rule constant.

6.2.3. A refusal is **counted and never an error**, never a `Warn` (a refusal of one
entry is an event, observability-and-logging.md 3.2): the count is in `PolicyDecided`
(12.1) and the entry at `Debug` with the keys `actor`, `lever` and `reason`.

6.2.4. **What a check cannot see.** A policy that returns the value the lever already
has (a `schedule` at an idle tick, a `rule` in its rest band that does not return an
entry at all) gets `unchanged` or no entry, and no write: the owner's setter is not
called and its event is not emitted (INT 3.3.2: an entry equal to the value held is not
a change).

### 6.3 The kinds of lever

6.3.1. The table is closed and a new kind is a row here and the matching slot in INT
(`spec-integration.md` 5.1). A kind is offered only while its owner is on.

| Kind | Key | Owner and slot | Value and bound | Default | Reads at | Event of the owner |
|------|-----|----------------|-----------------|---------|----------|--------------------|
| `flow` | a flow ID | MON 4.4.3 and FIS 13: `SetFlowScales` | 0 to `MaxFlowScale` | 10 000 (0 pauses the flow) | P1.7, P1.7b, P3.2(c), P3.3, P3.4a, P4.1a | `MoneyFlowScaled` |
| `edge` | `(From, To)` or `Against` | REG 7: `SetEdgeStatuses` | `FactorBP` 1 to `MaxEdgeFactorBP`, and `Closed` | open at 10 000 | the graph at the call; P3.2, P3.3, P3.5, P7.2 | `EdgeStatusChanged` |
| `transit` | `(From, To, Mode)`, a region pair | TRN 10.3: `SetTransitFactors` | 0 to `MaxTransitFactorBP` | 10 000 (0 blocks new dispatches) | P3.2, P3.3 at dispatch | `TransitFactorChanged` |

6.3.2. **Reserved kinds**, which become offered with their module and the slot's
row in INT (their specs are written and their slots are in `spec-integration.md` 5.1; the kinds stay reserved, and
version 1 offers the three of 6.3.1): `rate` (a lender's posted-rate scale of a region, `spec-credit.md` 13.1,
`SetRateScales`), `rent` (a region's rent scale, `spec-land.md`, `SetRentScales`) and
`legal` (an ORG control's legal level, `spec-institutions.md` 7.1, `SetLegalLevels`). **A pack that names one is refused with `ErrOutOfBounds` (`.kind`) until its row is in 6.3.1**; they are reserved so that the names are not taken. Each
is a number of basis points with its owner's bound and default, a key that names a region or
a control, and the reach rule "the region is covered by `J`" (a control: its jurisdiction
is `J` or a descendant). A price control is the first consumer of this list: the cap of
a market is ORG's, and its **level** is a lever.

6.3.3. **Not offered, and why.** A yield shock, a capacity or regeneration scale, an
endowment adjustment, an outside price shock and a birth or death scale are shocks, not
rates of an authority (position 4): a game's story keeps those calls. A death scale in
particular would let a rule reduce a population, which is a decision of the
population module and a spawner (`spec-integration.md` 3.6). A cadence, a window and a
gate are never levers (position 15). The calls that are not a slot of a rate or a flow are not offered either: `SetRouteRiskScales` and `SetStoreRiskScales` (a hazard is a story shock, `spec-risk.md` 10.3), `SetRightSuspensions` (a suspension is a switch, not a basis-point number: a consumer's call), `AdjustTechnology`, `AdjustFacilities` and `TransferTitles` (adjustments and transfers, not slots, `spec-integration.md` 5.1 and 5.4). A policy actor writes slot keys only.

### 6.4 Writing a slot

6.4.1. The world writes each changed key through **the internal form of the owner's
setter**, the one the public call uses after its checks, so that the owner's state, its
derived tables (the graph is rebuilt **once for the whole application**) and its event
are exactly what the call would have produced. The `cause` of an event that carries one
is `policy` (a new constant of the closed set whose other member is `consumer`,
observability-and-logging.md 3.3.2); `MoneyFlowScaled` and `TransitFactorChanged` gain a
`Cause` field for this (additive, observability-and-logging.md 4.2).

6.4.2. **Order of application**: the lever kinds in PLC's own order, which is the
order of the rows of INT's slot table (`spec-integration.md` 5.1) that a lever can
reach: **`edge` first, then `transit`, then `flow`**; within a kind in ascending key order of the owner
(`EdgeID`, `(from, to, mode)`, flow ID), within a group in ascending key. The order does
not change a result (the keys are distinct), and it fixes the order of events.

6.4.3. **A consumer's own call** to a held key is legal: INT's setter does not know a
lever. Its value stands until the actor next returns an entry for the key. An actor that
returns no entry never overwrites it, so a consumer may hold a lever by hand while an
actor is idle, and a `rule` in its rest band leaves the consumer's value alone. This is
position 10 and the rule that makes a player and an actor mixable (9).

6.4.4. **The slot's own rules still hold**: a value that equals the owner's default
clears the key (`spec-integration.md` 5.2.1), there is no timer (5.2.2), a shock does not
cascade (5.2.3), a pause is a scale of 0, and a job started or a shipment dispatched
keeps what it was given (5.2.3). A policy cannot do what a consumer could not.

## 7. When an actor acts

### 7.1 The position

7.1.1. A new position **P0.3a** in the entry table of `spec-world.md` 6.2, after
`TickStarted` (P0.3):

| ID | What runs | Owner | Reads / writes | Draws | Events |
|----|-----------|-------|----------------|-------|--------|
| P0.3a | The policy actors, for each actor in ascending `ActorID` that is due (7.2): the view, `Decide`, the checks, the application (6.1) | PLC | reads the boundary state and the slots; writes the slot keys of the actor's levers | D17: a consumer's `Policy`'s own words, in call order; the built-ins take none | per actor: the owners' slot events (cause `policy`) in the order of 6.4.2, then `PolicyDecided` |

7.1.2. **Why here.** P0.3a comes after `TickStarted` so that its events are inside the
step's span and carry `Tick` `t`, and before P1.1, the first position, because every
position that reads a slot is at P1.2 or later (`spec-world.md` 8.2: the cost anchor
at P1.2 reads the yield shock, P1.7 the flow scale, the graph is rebuilt at the call).
So the tick sees the settings exactly as it would if a consumer had made the call
between the previous `Step` and this one (WLD 8.1.1 item 1). No reservation is open
(WLD 6.1.6), no job half started and no shipment half dispatched: the world is at a
boundary, which is what lets a slot that acts "at the call" (an edge status rebuilds
the graph) act here.

7.1.3. **WLD's rule that a slot is set between ticks** (8.1.1) is kept in its
sense: P0.3a is the entry of the step, before any position that reads a slot, so the
write is indistinguishable from a call made between the two steps. The public setters
still return `ErrInStep` to a caller inside a step; P0.3a uses their internal forms and is
the one place where the library itself is a writer of slots. WLD 8.2 says so (17.2).

7.1.4. **A failed step** has nothing to do with it: a step that fails at P4.1 failed
after its P0.3a, and the world is failed (WLD 3.5); the actors are not called again until a
restore.

### 7.2 The cadence: a rotation

7.2.1. An actor is **due** at tick `t` when

```
(t + bucket(a, Every)) mod Every = 0        with   bucket(a, P) = (Weyl2(a) * P) >> 32,
                                                   Weyl2(a)    = (a * 0x6A09E667) mod 2^32
```

where `a` is the actor's `ActorID`. This is the **rotation** device of
`spec-world.md` 9.1, with the **actor's index** as the key instead of an agent ID
(9.1.1 there: a rotation uses `bucket`, never the raw index or `Weyl32`). A cadence is
**data** (`Every`, in the hash) and constant for a run; no slot changes it
(position 15; WLD 9.4.3).

7.2.2. **Properties.** An actor with `a = 0` is due at every tick that is a multiple
of `Every`, tick 0 included; the next ones are due at an offset that is a fixed function
of the index, so indices 0, 1 and 2 are on three different ticks of ten at `Every` 10 (the worked
example of 14.1: buckets 0, 4 and 8, and 0, 12 and 24 at `Every` 30). **Two actors may
share a bucket**: indices 2 and 7 both give bucket 8 at `Every` 10, and indices 0 and 1
both give 0 at `Every` 2. WLD 9.1.1 claims no spread below about a thousand agents, and an
actor count is far below that (open question 9). The offset changes
when an actor is added or removed, as any ID-keyed device does (N-117): the actors are
renumbered in ID order, and the hash differs.

7.2.3. **A cadence that does not divide `TicksPerYear`** is legal and never rounded
(`spec-world.md` 9.3.1): an actor at `Every` 30 in a 365-tick year acts 12 or 13 times
a year and its phase against the seasons moves by `365 mod 30` each year. A pack that
wants an action at a season boundary uses a **schedule** with a `Cycle` (8.2), which is
data in ticks.

7.2.4. **Tick 0.** An actor due at tick 0 acts on the construction state: the first
step's view is the state `NewWorld` built (plus the consumer's calls), and its settings
take effect in tick 0. A `rule` reading `unfed_share` at tick 0 reads a world that has
not consumed yet, so every household is fed (`Unfed` 0) and the rule is at rest or below
target; a pack that wants a warm-up gives its rule a first decision later by `Every`
or by a schedule step that starts at the tick it wants.

### 7.3 The draws

7.3.1. The row of the table of draws (`spec-world.md` 7.3) is

| ID | When and where | Taken by | How many | In what order | Owner |
|----|----------------|----------|----------|---------------|-------|
| D17 | P0.3a, inside `Decide` | a consumer's `Policy`; **the built-ins take none** | its own | actors in ascending `ActorID`, call order | PLC 7.3 |

and the words of one tick (`spec-world.md` 7.3.1) gain a term, the consumer's `Policy`
draws, beginning the tick. An actor that is not due takes none. The count is a function
of the configuration, the tick and the policies registered (WLD 7.4.1): not of an
observer, a logger or a module that is on and idle.

7.3.2. **A draw that might be skipped is not taken anyway by the library**: the
built-ins draw nothing, so nothing is skipped. A consumer's policy that draws only when
a reading is above a threshold makes the count depend on the state, which is legal
(the world's own rule of WLD 7.4.3 is for the world's draws) and is the consumer's
hazard for a paired comparison (WLD-A5).

### 7.4 After a restore, and between ticks

7.4.1. A restored world is due at the same ticks (the rule is a function of `t`, `Every`
and the index), reads the same view (5.2.1), and finds each lever's value in its owner's
section; the only state of this module is the mailbox of a `directed` actor (13). The
consumer re-registers its policies by name before the restore, and a `Stateful` policy's
bytes are in the extension section (CHK 7.3).

7.4.2. The module adds **no slot** of its own and one call (`DirectPolicy`, 9.1). The
actors' cadence and levers are configuration and are not edited by a call.

## 8. The built-in policies

Four, registered by name, each a value of the root package built from the actor's row.

### 8.1 `hold`

8.1.1. Returns no settings, always: every lever keeps the value its slot holds. It is
the default of the module in the sense of architecture.md 4.5 (`HoldPolicy`), and the
policy of an actor a pack declares only to give a consumer a mandate to write by
`DirectPolicy` while the library does nothing of its own accord. (An actor whose policy
is `hold` and which has no lever draws the warning `actor_no_levers`.) It takes no draw,
keeps no state and builds no view that it would not use.

### 8.2 `schedule`

8.2.1. A **fixed schedule** is a step function of the tick, data in the actor's row.
`Schedule` lists steps `(Lever, At, BP, Closed)`. The value of lever `L` at tick `t` is

```
value(L, t) = the BP of the step of L with the largest At <= u,        u = t              if Cycle = 0
                                                                      u = t mod Cycle    if Cycle > 0 and t >= Cycle
                                                                      u = t              if Cycle > 0 and t < Cycle
```

and no value (the lever is held) when no step of `L` has `At <= u`, **except** that with a
`Cycle` and `t >= Cycle` a lever with no step at or before `u` takes the value of its
**last** step (the value carried across the wrap). So with a `Cycle` the schedule
**repeats**: in tick `Cycle + k` it reads the step that held at `k`, and in the first cycle
the value before a lever's first step is "held". Every `At` is below `Cycle` when there is
one (`schedule_value`).

8.2.2. **At each decision the schedule returns the value of every lever that has one.**
It is stateless and idempotent: it returns the same value on every decision until the
next step, which the checks of 6.2 turn into `unchanged`, so no event and no write. It
returns an entry at **every** decision, so it re-asserts its value over a consumer's call
to the same key at the next one (a `rule` in its band and a `directed` actor with an empty
mailbox return none, 6.4.3).
Because it applies at the **first decision tick at or after** a step's `At`, a step takes
effect up to `Every - 1` ticks late (a step at 120 is applied at tick 126 for an actor
with `Every` 10 and bucket 4, 14.1); a pack that needs a step at an exact tick sets
`Every` 1.

8.2.3. **A schedule is checked at validation, against the lever's own limits**, so that
no step of the pack's plan is refused by the lever's own bounds or step limit: every `BP` lies in `[Min, Max]`; a step with
`Closed` is only on a lever with `MayClose`; and, when `MaxStep` is above 0, two
consecutive values of one lever differ by at most `MaxStep`, the first from the lever's
**default** and, with a `Cycle`, the first again from the last (the wrap) (`ErrPolicyShape`, rules
`schedule_value` and `schedule_step`). A pack that wants a jump larger than a step writes
several steps `Every` ticks apart: a ramp is data. Two steps of one lever **closer than
`Every`** (with a `Cycle`, the last and the first plus `Cycle` as well) reach one decision as
one change that may exceed `MaxStep`; the step cut then spreads it over later decisions,
counted in `Stepped` and not silent, and the pack gets the warning `schedule_dense` (11.2)
when `MaxStep` is above 0.

8.2.4. **What a schedule is not**: it does not read the state (it is open loop, PLC-A5),
and it is not a timer on a shock (`spec-integration.md` 3.6: a timer is a rule in the
tick). It is a pack's plan for a rate that follows the calendar or a policy of a regime,
and it is the form a player's fixed program takes when the consumer would rather write
it in the pack than direct it.

### 8.3 `rule`

8.3.1. **The reactive rule** is a bounded stepping controller, one reaction per lever:
every decision it reads one gauge, compares it with a target and moves the lever by a
step in proportion to the error, inside a rest band and the lever's limits. For each
reaction in ascending lever order, with `m` the reading, `T` = `Target`, `D` =
`Deadband`, `S` = `Span`, `M` = `Move` and `v` the lever's current value:

1. if the gauge is **undefined** (8.3.2): no entry, counted `NoReading`;
2. `e = m - T`. If `|e| <= D`: no entry, counted `InBand`;
3. `raw = trunc(e * M / S)` in 128 bits, truncated toward zero (architecture.md 3.5), and
   `d = raw` when `Raise` is true, `-raw` when it is false;
4. the entry is `(Lever, v + d)` with the lever's `Closed` as the view has it (the rule neither
   closes nor opens an edge; a `Closed` true that the key already has is no closing, so the
   rule moves the factor of an edge a consumer closed and leaves it closed, 6.2.1 check 3). A sum outside `int64` is cut to the bound of `int64` in its
   sign, which the checks of 6.2 then cut to `[Min, Max]`, so no wrap occurs.

The checks of 6.2 then apply as to any policy: the bounds, then `MaxStep`.

8.3.2. **The gauges.** A gauge is a pure function of the view, with a unit and a rule
for "undefined". `rdiv` is the half-to-even division of IND 5.2.2. The set is closed
(a new gauge is a row here, and its IND preset is an ask, 17.2):

| Gauge | Reading | Unit | Undefined when | The measure it matches |
|-------|---------|------|----------------|------------------------|
| `treasury` | `Treasury` of the view | minor units | the jurisdiction is not governed | the treasury term of IND `Sectors` (an ask) |
| `unfed_share` | `rdiv(10 000 * U, H)`, with `U` the sum of `RoleRow.Unfed` and `H` the sum of `RegionRow.Households` over the covered regions that are not outside nodes | bp of households | `H = 0` | IND preset 12 `Unfed`, boundary share, over a set of regions |
| `mean_ratio` | `rdiv(sum(MeanRatioBP * RatioHouseholds), sum(RatioHouseholds))` over the role rows of the covered regions with `HasMeanRatio` | bp (10 000 is a ratio of 1) | no role row has a ratio | IND preset 10 `WelfareDistribution.Mean` over a set; equal to it up to the rounding of the role means (at most 1 bp, FR-WEL-036) |
| `price` (a commodity, `MaxAge`) | `rdiv(sum(H_r * P_r), sum(H_r))` over the covered regions `r` with a market row of the commodity whose `State.Age` is at most `MaxAge`, with `H_r` the households of `r` and `P_r` the published `State.Price` | minor units | no such region, or `sum(H_r) = 0` | a household-weighted regional price, an IND ask (`PriceLevelOf`) |

8.3.3. **Why these four.** Each is a field read or one weighted ratio over fields
that OBS already lists, so the engine computes no index it would have to define (the
hierarchy of the presets is IND's). They are the measures an authority reads in any setting: its purse, the share of its people who went hungry, how well
off they are and what its staple costs. A gauge that needs an index number, a flow over
a window, a distribution or a forecast is a consumer's `Policy` calling `indicator/`
(4.3.3).

8.3.4. **Rounding and arithmetic** (rule constants): a ratio of gauges is `rdiv` (half to
even); the step is truncated toward zero; the product is in 128 bits with `math/bits` inside `internal/policy`,
which imports neither `internal/measure` nor `internal/reduce` (FR-IND-001, architecture.md
2.6) and carries its own product until N-019's shared helper is decided; no float. The gauges are computed in the order of the view
(ascending `RegionID`, `RoleID`, `CommodityID`), so the sum is exact and independent of
any order.

8.3.5. **An integrator, and what it lacks.** The rule adds a share of the error to the
lever each decision, so the lever **integrates** the error: its rest value is the
value at which the reading sits in the band, whatever that value is. It has no
derivative term, no memory of its own (the lever's value is its memory, saved by the
owner), no gain schedule and no prediction. A **proportional** rule (the lever is a
function of the reading) is not offered: it would need the value the lever "should"
have for each reading, which is the plant's model, and a consumer's `Policy` can say it.
Its loop is the integral controller of PLC-A1: stable only if the gain (`Move / Span`
times the plant's slope) is below `2 sin(pi / (2 (2d + 1)))` for a delay of `d`
decisions (15, model `delay.py`).

8.3.6. **What damps it**, all data: the **deadband** (a reading inside `Target ± D` is
a rest, which also stops a rule from hunting on noise), the **step limit** (`MaxStep`),
the **lever's bounds** (`Min`, `Max`), the **cadence** (`Every`: the plant gets time to
answer), and, for `price`, **`MaxAge`** (a stale price is not a reading). None is a
default: a pack that gives a `rule` no `MaxStep` and no deadband has chosen an undamped
controller (the warning `rule_undamped`, 11.2).

### 8.4 `directed`

8.4.1. The player's policy. The world keeps for each `directed` actor a **mailbox**: a
list of `LeverSetting` sorted by lever, set by `DirectPolicy` (9.1). At the actor's
decision tick the policy returns the mailbox's content, the checks of 6.2 apply, and the
mailbox is **cleared**, whatever the outcomes: a directive acts once. A lever whose
value should stay is not in the mailbox (a slot holds its value, `spec-integration.md`
5.2.2). With an empty mailbox the policy returns nothing: no event, no write.

8.4.2. **Not sticky.** Unlike an agent's directive (`spec-agents.md` 10.2), an actor's
directive has no `Sticky` flag: a slot is its own memory, so "the same again next time"
is "nothing". A mailbox is replaced by a later call before the decision, never merged.

8.4.3. The mailbox is state in the world (13) and not in the policy, so that a checkpoint
saves it whether or not the consumer registered anything, and `hold`, `schedule` and
`rule` need none.

## 9. A player, and an authority that trades

### 9.1 `DirectPolicy`

9.1.1. The call a player (or a bot, or a script) uses to rule:

```go
// PolicyDirective is the whole content of one actor's mailbox.
type PolicyDirective struct {
	Actor  ActorID
	Levers []LeverSetting // sorted by Lever or not: the world sorts; empty clears the mailbox
}

// DirectPolicy replaces the mailbox of each named actor. It acts between ticks, is
// atomic, draws nothing and emits no event; the directive is applied at the actor's
// next decision tick, through the checks of 6.2.
func (w *World) DirectPolicy(d []PolicyDirective) error
```

9.1.2. **The shape and the checks** follow `spec-integration.md` 3: a slice of entries,
one per actor, atomic, an empty or nil slice a no-op. In the order of INT 3.2: a step
running (`ErrInStep`), a failed world (`ErrWorldFailed`), the module off
(`ErrModuleOff` naming `policy`), then each entry in order: the actor
(`ErrUnknownActor`, new), a repeat of the actor in one call (`ErrDuplicateID`), the
actor's policy is not `directed` (`ErrNotDirected`, INT's sentinel for "this target
takes no directive"), then each setting: the lever (`ErrUnknownLever`, new), a repeat of a
lever in the entry (`ErrDuplicateID`), the value in `[Min, Max]` and `Closed` only
with `MayClose` (`ErrOutOfBounds`, path `.bp` or `.closed`); the `Closed` check is static, so a directive that keeps a consumer-closed edge closed without `MayClose` is refused at the call (a `rule` is not, 6.2.1 check 3). The step limit is **not** a
check of the call: it is taken against the value the lever has at the decision, which a
call in between may change. A directive that the step limit cuts shows in
`PolicyDecided.Stepped`, and in `Slots()` after the decision.

9.1.3. **Why a cadence and a mandate.** Without an actor a game's governing player can call
`SetFlowScales` whenever it likes on any flow. With one, its calls are bounded by the
levers it holds (a mandate), act on a rhythm (a body that sits every 30 ticks) and
are rate-limited (`MaxStep`), and each is an event with its actor. A game that wants none
of that calls INT's setters and registers no actor (9.3). The two mix: 6.4.3.

9.1.4. **The latency** is the cadence: a directive left at tick 95 for an actor due at
ticks ≡ 0 mod 10 is applied at P0.3a of tick 100. A consumer that wants the effect at the
next step gives the actor `Every` 1. The checkpoint and the restore keep the directive
(13).

### 9.2 An authority that also trades

9.2.1. ADR 0003 decision 3 says "a ruler that also trades uses `Strategy` for its orders
and the policy interface for its settings". Version 1 settles it as two objects, each
with the mechanism it already has:

- the **treasury** trades only by FIS's standing orders (`spec-fiscal.md` 9), data whose
  quantity is scaled by a flow lever; the treasury pseudo-agent has no strategy
  (`spec-polity.md` 5.3.3);
- a **person** who governs and also trades (a player) is an external agent
  with the `Directed` strategy (`spec-agents.md` 10), with its own money, and a `directed`
  actor of its jurisdiction. Its orders are directives of INT's `Direct` and its rates
  are directives of `DirectPolicy`; the two are one human and two calls.

This **narrows** the second sentence of decision 3: no authority object of the library
holds both a `Strategy` and a `Policy`, and the treasury's trade is data (FIS 9). The
ADR note of 17.2 says so (open question 5 is the way back).

9.2.2. A **treasury that bids under a strategy** (a pseudo-agent that holds one) is not
offered: it needs the admission rule of a pseudo-agent that both buys and sells
(`spec-clearing.md` 3.1.2) and a place for the strategy's state (open question 5).

### 9.3 Which tool for which need

| Need | Use | Why |
|------|-----|-----|
| A story shock (a failed yield, a flood, a plague) | INT's setter, from the consumer's script | a shock has no mandate and no cadence; it is what happens to the jurisdiction |
| A rate that follows a plan | `schedule`, or INT's setter at the tick | the plan is data, in the hash, and reproducible by a sweep |
| A rule that reads the jurisdiction | `rule` (four gauges), or a consumer's `Policy` | the loop is registered and bounded by the levers' limits |
| A rule that needs an index, a flow over a window or a forecast | a consumer's `Policy` (with `Stateful` for memory) calling `indicator/` | the engine does not call indicators (position 7) |
| A governing player | `directed` and `DirectPolicy`, or INT's setters when no mandate is wanted | a mandate and a cadence are what the actor adds |
| A controller that is none of the above, or runs on a clock the engine does not have | the consumer's own loop between `Step` calls (STB-L121, INT-A1) | outside the engine; its stability is the consumer's |

## 10. Parameters

All are fields of `Config` with a bound, a doc line and an owner (architecture.md 9.5 to
9.7), under the module entry `Modules.Policy` (a pointer; nil is off, `spec-config.md`
3.2). **The module has no tunable with a default of its own**: a mandate, a cadence, a
target and a gain are choices of a setting, so "required" below means that the field
has no default and a missing one is `ErrPolicyShape` or `ErrOutOfBounds`, and the
default of the section as a whole is "no actors", which is the world without the module.
The bounds are in `bounds.go` (architecture.md 9.6).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Policy` | the module; present means on | nil (off) | |
| Policy | `Actors` | the policy actors | none | at most `MaxActors` |
| Actor | `ID` | the identity | required | `[a-z][a-z0-9_]*`, at most 64 bytes, unique |
| Actor | `Jurisdiction` | the jurisdiction the actor is attached to | required | a defined jurisdiction, at most one actor each |
| Actor | `Policy` | the policy by name | required | a built-in or a registered name |
| Actor | `Every` | ticks between decisions | required | 1 to `MaxPolicyEvery` |
| Actor | `ReadsRegions` | the view carries the covered regions | false | |
| Actor | `Levers` | the levers | none | at most `MaxLevers` |
| Actor | `Schedule` | the steps of a `schedule` | none | at most `MaxScheduleSteps`; only with `schedule` |
| Actor | `Cycle` | the length of a repeating schedule, in ticks | 0 (no cycle) | 0, or 1 to `MaxPolicyCycle`; only with `schedule` |
| Actor | `Rules` | the reactions of a `rule` | none | one per lever; only with `rule` |
| Lever | `ID` | the identity within the actor | required | `[a-z][a-z0-9_]*`, at most 64 bytes, unique |
| Lever | `Kind` | `flow`, `edge` or `transit` | required | the closed set of 6.3.1 (a reserved kind of 6.3.2 is `ErrOutOfBounds`) |
| Lever | `Flow`, `From`, `To`, `Mode`, `Against` | the key | by kind | defined, in reach (3.5) |
| Lever | `Min`, `Max` | the values the actor may set | required | inside the slot's bound; `Min <= Default <= Max` |
| Lever | `MaxStep` | the largest change per decision | 0 (no limit) | 0, or 1 to the slot's bound |
| Lever | `MayClose` | the actor may close an edge | false | kind `edge` only |
| Step | `Lever`, `At`, `BP`, `Closed` | one step of a schedule | required | `At` 0 to `MaxScheduleTick` (below `Cycle` when there is one); `BP` in `[Min, Max]` |
| Reaction | `Lever` | the lever it moves | required | one per lever |
| Reaction | `Gauge` | the reading | required | `treasury`, `unfed_share`, `mean_ratio` or `price` |
| Reaction | `Commodity` | the commodity of a `price` gauge | required for `price` | a defined commodity; empty otherwise |
| Reaction | `MaxAge` | the oldest price age that counts | required for `price` | 0 to `MaxPolicyEvery`; 0 otherwise |
| Reaction | `Target` | the reading at which the rule rests | required | 0 to `MaxMoney` |
| Reaction | `Deadband` | the half-width of the rest band | 0 | 0 to `MaxMoney` |
| Reaction | `Span` | the error that moves the lever by `Move` | required | 1 to `MaxMoney` |
| Reaction | `Move` | basis points of lever value per `Span` | required | 0 to `MaxMoveBP` |
| Reaction | `Raise` | the lever rises when the reading is above `Target` | false | |

Presentation-only, so exempt from the registry test and not hashed
(`spec-config.md` 8.3): `PolicyActor.Label`, empty by default and at most `MaxLabelBytes`
bytes.

**Bounds** (placeholders, N-015): `MaxActors` 64 (at most one per jurisdiction, and
`MaxJurisdictions` is 1 000, so more is pointless in a pack that has few authorities),
`MaxLevers` 32 per actor, `MaxScheduleSteps` 256 per actor, `MaxPolicyEvery` 3 650
(ten years of a 365-tick year), `MaxPolicyCycle` 3 650, `MaxScheduleTick` 1 000 000 000 (the largest `At`), `MaxLeverKeys` 256 (the keys a
single `Against` expands to) and `MaxMoveBP` 100 000. `MaxMoney`, `MaxFlowScale`,
`MaxEdgeFactorBP`, `MaxTransitFactorBP` and `MaxLabelBytes` are the owners' and the
world's (`spec-config.md` 4.1).

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the position
P0.3a and the order of actors and of levers within an application (6.4.2, 7.1); the
rotation formula and `Weyl2` (7.2); the order and the outcomes of the checks (6.2); the
order of the two cuts (6.2.2); the rule `d = trunc(e * Move / Span)` and its 128-bit
product (8.3.4); the four gauges and their rounding (8.3.2); the reserved names `hold`,
`schedule`, `rule` and `directed` (4.4.1); the cause `policy`; the clearing of a
mailbox after one decision (8.4).

**Not parameters:** the `ActorID` and `LeverID` (derived), the due ticks and buckets
(derived), the expanded key list of an `Against` lever (derived from the polity and the
graph), the value of a lever (the slot's state), the mailbox (state, 13).

The placeholders, so that none is mistaken for a result: every bound above, the operation
counts of 5.4 and the numbers of the examples of section 14.

## 11. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). Errors name the path,
for example `modules.policy.actors[1].levers[0].min`. The framework of `spec-config.md` 7
runs them in the order below, so the first error is the same for the same input.

### 11.1 Errors

| Order | Sentinel (rule) | When |
|------:|-----------------|------|
| 1 | `ErrModuleDependency` | `Modules.Policy` with `Modules.Polity` nil or with no jurisdiction; a `flow` lever with neither money supply nor fiscal on; a `transit` lever with transport off |
| 2 | `ErrTooMany` | more than `MaxActors` actors, `MaxLevers` levers, `MaxScheduleSteps` steps, or an `Against` that expands to more than `MaxLeverKeys` |
| 3 | `ErrDuplicateID` | an actor ID, a lever ID within an actor, a step `(Lever, At)` or a reaction `Lever` repeated; two levers of one actor naming one key |
| 4 | `ErrUnknownJurisdiction` | `Jurisdiction` or `Against` undefined |
| 5 | `ErrPolicyShape`, rule `one_actor_per_jurisdiction` | two actors on one jurisdiction |
| 6 | `ErrOutOfBounds` | a `Kind` that is not offered (6.3.1; a reserved kind of 6.3.2 is refused, path `.kind`, until its slot is integrated in INT and its row is in 6.3.1); `Every`, `Cycle`, `Move`, `Span`, `Target`, `Deadband`, `MaxAge`, `Min`, `Max`, `MaxStep` or a step's `BP` outside its bound; `Min > Max`; the default outside `[Min, Max]` |
| 7 | `ErrPolicyShape`, rule `policy_data` | `Schedule` or `Cycle` with a policy other than `schedule`; `Rules` with a policy other than `rule` |
| 8 | `ErrPolicyShape`, rule `lever_key` | the key fields of a kind wrong (a `Flow` on an edge, `From` without `To`, `Against` with `From`, `Mode` on a flow); `MayClose` on a kind other than `edge` |
| 9 | `ErrUnknownFlow`, `ErrUnknownEdge`, `ErrUnknownRegion`, `ErrUnknownMode` | a key that is not defined (an edge lever's pair that is not a configured edge, a region of a transit lever that does not exist, a mode that does not exist) |
| 9a | `ErrPolicyShape`, rule `transit_pair` | a `transit` lever with `From` equal to `To`, or with `From` or `To` an outside node of the external module (INT 5.1.3: `ErrSameRegion`, `ErrOutsideNode`); an `edge` lever may name a gate edge to an outside node (REG 7 refuses none) |
| 10 | `ErrLeverNotInReach` | a key outside the reach of 3.5 |
| 11 | `ErrPolicyShape`, rule `against_nested` | `Against` names the actor's own jurisdiction, an ancestor or a descendant |
| 12 | `ErrPolicyShape`, rules `schedule_lever`, `schedule_value`, `schedule_step` | a step names no lever of the actor; a value outside the lever's bounds or `Closed` without `MayClose`; two consecutive values further apart than `MaxStep` (8.2.3) |
| 13 | `ErrPolicyShape`, rules `rule_lever`, `rule_gauge`, `rule_reads` | a reaction names no lever; two reactions on one lever; a `price` gauge without `Commodity` or `MaxAge`, or another gauge with them; a region gauge with `ReadsRegions` false |
| 14 | `ErrNotGoverned` | a `treasury` gauge on a jurisdiction with no seat |
| 15 | `ErrLeverShared` | a key held by levers of two actors, after the expansion of every `Against` |
| 16 | `ErrUnknownPolicy` (in `NewWorld`) | a `Policy` name that is neither built in nor registered, after `Validate` succeeds |

`ErrModuleDependency`, `ErrTooMany`, `ErrDuplicateID`, `ErrUnknownJurisdiction`,
`ErrUnknownFlow`, `ErrUnknownEdge`, `ErrUnknownRegion`, `ErrUnknownMode`,
`ErrOutOfBounds` and `ErrNotGoverned` exist in other specs; this spec adds
`ErrPolicyShape`, `ErrLeverNotInReach`, `ErrLeverShared`, `ErrUnknownPolicy`,
`ErrUnknownActor` and `ErrUnknownLever`, mapped at the root as N-075 asks.

### 11.2 Warnings

They do not fail validation; each names its rule and path (`spec-config.md` 7.4).

| Rule | When |
|------|------|
| `no_actors` | the module is present with no actor |
| `actor_no_levers` | an actor has no lever (a `hold` actor that is only a name, or a policy with nothing to set) |
| `lever_fixed` | `Min = Max`: the lever cannot move |
| `lever_no_keys` | an `Against` lever expands to no edge |
| `reads_unused` | `ReadsRegions` true with a policy that reads no region (`hold`, `schedule`, `directed`, or a `rule` with only a `treasury` gauge) |
| `rule_undamped` | a reaction whose lever has `MaxStep` 0 and whose `Deadband` is 0 (8.3.6) |
| `rule_inert` | a reaction with `Move` 0 |
| `cadence_long` | `Every` above `TicksPerYear`: an actor that acts less than yearly |
| `schedule_dense` | with `MaxStep` above 0, two steps of one lever closer than `Every` ticks (with a `Cycle`, the last and the first plus `Cycle` as well): the plan reaches one decision as one change and is spread by the step limit (8.2.3) |

## 12. Observability

### 12.1 Events

12.1.1. **One new event**, class A (an aggregate, bounded by the actors of the
configuration; none is per agent, order or trade), position P0.3a, gate "policy"
(observability-and-logging.md 4.3):

| Event | Emitted when | Fields (besides `Tick`) |
|-------|--------------|-------------------------|
| `PolicyDecided` | once per due actor per tick in which any count below is non-zero, after the slot events of its application; never when the actor returned nothing and held nothing for a reason | `Actor`, `Jurisdiction`, `Entries` (entries returned), `Applied`, `Unchanged`, `NotHeld`, `Duplicate`, `NoClose` (refusals), `Clamped`, `Stepped` (cuts, as flags on entries; an entry may be both), `InBand`, `NoReading` (the `rule`'s own holds) |

`Entries = Applied + Unchanged + NotHeld + Duplicate + NoClose` (a requirement,
FR-PLC-082). A policy that returns nothing and, for a `rule`, holds no reaction for a
reason (no reading, in band), emits nothing: `hold` never emits. No field is a map and
every value is an `int64` or an ID (observability-and-logging.md 4.1).

12.1.2. **Existing events that change** (additive, observability-and-logging.md 4.2):
`MoneyFlowScaled` and `TransitFactorChanged` gain `Cause` (`consumer` or `policy`);
`EdgeStatusChanged.Cause` gains the member `policy`. A call of the consumer's
carries `consumer` as before. Their gates do not change.

12.1.3. **Order.** At P0.3a, per actor in ascending `ActorID`: the slot events of the
application in the order of 6.4.2, then its `PolicyDecided`. They come after
`TickStarted` and before the first event of phase 1 (item 1 of `spec-world.md` 6.11 gains
a sub-item). An observer that wants to know which actor caused a slot event reads the next
`PolicyDecided`; one that wants only whether it was a policy reads the cause.

### 12.2 Logs

12.2.1. At `Debug`, guarded by `Enabled`, with constant messages: "policy decided" (the
keys `actor`, `entries`, `changed`) and, per refused entry, "policy entry refused" (the
keys `actor`, `lever` and `reason`, one of `not_held`, `duplicate`, `no_close`). A
reaction's reading and the move it made are at `Debug` with "rule moved lever" (the keys
`actor`, `lever`, `reading` and `from`, `to`). **No `Info`, `Warn` or `Error` inside the
tick** (observability-and-logging.md 3.2.2).

12.2.2. **New keys** for the table of observability-and-logging.md 3.3.2: `actor` (the
actor's ID string), `lever` (the lever's ID string), `reading` (a gauge reading, `int64`,
in its unit), `entries` and `changed` (counts of entries returned and of levers whose value changed), `actors` and `levers` (counts, in the construction summary). `policy` joins
the `modules` list. `reason` and `cause` are reused (`cause` gains the value
`policy`); `from` and `to` are the existing keys for a value before and after.

12.2.3. At construction the warnings of 11.2 are logged once each at `Warn` with `rule`
and `path`, as every construction warning (`spec-world.md` 5.8.1), and the "world
created" line gains `actors` and `levers`.

### 12.3 Snapshot

12.3.1. `Snapshot.Actors []ActorRow`, one per actor in ascending `ActorID`:
`Actor`, `Jurisdiction` and `Pending` (the number of settings in the mailbox of a
`directed` actor, 0 for another policy). Cost K. The levers' values are not in it: they are
INT's `Slots()` (`spec-integration.md` 5.3) and the rows `Flows`, `Edges` and
`TransitFactors` of the snapshot, as before. The part of the snapshot's header that
names the modules (`Modules`) gains `policy`. An actor's view is built from the same
rows, so a consumer can reproduce what a policy saw.

### 12.4 Metrics

12.4.1. Four instruments, all derived from `PolicyDecided` (observability-and-logging.md
5.3, 5.3.1; names with units and attributes, the cardinality of 5.4.1 bounded by the
configuration):

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.policy.decisions` | Counter | `{decision}` | `economysim.actor` | `PolicyDecided` (one per event) |
| `economysim.policy.entries` | Counter | `{entry}` | `economysim.actor`, `economysim.entry_outcome` | `PolicyDecided` (`Applied`, `Unchanged`, `NotHeld`, `Duplicate`, `NoClose`) |
| `economysim.policy.cut` | Counter | `{entry}` | `economysim.actor`, `economysim.cut` | `PolicyDecided` (`Clamped`, `Stepped`) |
| `economysim.policy.held` | Counter | `{reaction}` | `economysim.actor`, `economysim.hold` | `PolicyDecided` (`InBand`, `NoReading`) |

12.4.2. **New attributes** (closed sets, so bounded): `economysim.actor` (the actors of
the configuration), `economysim.entry_outcome` (`applied`, `unchanged`, `not_held`,
`duplicate`, `no_close`), `economysim.cut` (`clamped`, `stepped`) and `economysim.hold`
(`in_band`, `no_reading`). For the first consumer's size (one actor) the instruments are
1, 5, 2 and 2 series; an actor adds that many.

### 12.5 What an observer can rebuild

12.5.1. **The settings of every lever** from the slot events and their cause (every
change of a slot is an event of its owner, now with a cause). An observer that wants the
number of times a rule moved a flow counts `MoneyFlowScaled` with the cause `policy`. A
consumer's own call has the cause `consumer`. The gap that the rebuild of OBS 4.3.3 has
to close is none: the module moves no money and no goods, so no sector balance changes.

### 12.6 The Observability entry (observability-and-logging.md 9)

1. **New failure path or rejected input?** A refused entry is not an error: it is counted
   in `PolicyDecided` and at `Debug`. A bad configuration is a validation error at
   construction (11.1); `DirectPolicy` returns its errors (9.1.2).
2. **New state transition?** A lever's value changes: its owner's event, with the cause.
   A decision has an event, `PolicyDecided`, with no map and fixed-point values.
3. **New decision point in a policy?** Yes, the `rule`: a `Debug` line with the reading,
   the lever and the move.
4. **New loop?** At most one `Debug` line per entry in a loop over the actors, guarded by
   `Enabled`; no `Warn` inside the tick; event construction is gated on an observer
   (observability-and-logging.md 4.5) and the view is built only when a policy needs it.
5. **Changed event, metric name, unit or attribute?** Two events gain `Cause` and one
   gains a member; additive, not breaking (4.2).
6. **New metric attribute?** Bounded by the configuration (12.4.2).
7. **`Error` or `Info` used?** `Info` only in the construction summary.
8. **Logging an error also returned?** None.
9. **Tests?** The recording observer asserts the order of 12.1.3 (FR-PLC-083), a capturing
   handler asserts that nothing above `Debug` is logged in a step (FR-PLC-084), and the
   non-interference test compares a run with and without an observer (FR-PLC-033,
   NFR-PLC-005).

## 13. State contributed to a checkpoint

13.1. **One section**, present with `Modules.Policy` and absent otherwise
(`spec-checkpoint.md` 4.2.1; its tag is 32, placed after the sections of
the modules whose slots a lever can reach):

| Section | Owner | State, in the order written | Present when |
|---------|-------|-----------------------------|--------------|
| Policy | PLC 13 | for each `directed` actor with a non-empty mailbox, in ascending `ActorID`: the actor, the count of settings, then each setting in ascending `LeverID` as the lever and the value and the closed flag | `Modules.Policy` |

13.2. **Sparse**: an actor with an empty mailbox has no row (a row with no setting is
corrupt, `spec-checkpoint.md` 4.2.5 R10). The decoder checks that the actor exists and
has the policy `directed`, that each lever is in range and ascending without repeats,
and that each value is inside the lever's `[Min, Max]` (`ErrCheckpointCorrupt`).

13.3. **Not saved**: the due ticks and buckets (derived), the expanded key lists
(derived), the views and the gauges (a function of the saved state, 5.2.1), the mailbox
of an actor whose policy is not `directed` (none exists), and the value of a lever: it is
the slot's own state in the section of its owner (MON, FIS, REG, TRN), which a restored
world finds there. A consumer's `Stateful` policy is in the extension section as a
fourth kind (4.6).

13.4. **Hosted per-agent fields**: none (N-128). An actor is not an agent and the module
adds nothing to an agent record.

13.5. **Size**: the mailboxes are at most `MaxLevers` settings for each of at most
`MaxActors` actors, each a lever, a value (a 3-byte varint at 100 000) and a
flag, about 5 bytes: about 10 KB at the bounds (64 x 32 settings), 160 bytes for one actor, and
zero for a world with no directive pending.

## 14. Worked examples

The setting is the invented kingdom of `spec-polity.md` 12.1 (regions `capital`, `farm`,
`harbour`, `hills`, `mill`; jurisdictions `coast`, `crown`, `inland`, `uplands`; `crown`
and `inland` governed) with the fiscal flows of an invented pack (`head_tax`, a tax of
authority `crown`, and `bread_stipend`, a payout of authority `inland`, `spec-fiscal.md`
18). All words are data and flagged examples; all numbers are placeholders. The
arithmetic is `models/plc/worked.py`.

```toml
[modules.policy.actor.coast_council]          # ActorID 0: a grouping, no treasury
jurisdiction = "coast"
policy       = "directed"
every        = 10
[modules.policy.actor.coast_council.lever.gate]
kind = "edge"
against = "inland"
min = 10000
max = 40000
may_close = true

[modules.policy.actor.crown_council]          # ActorID 1
jurisdiction = "crown"
policy       = "schedule"
every        = 10
[modules.policy.actor.crown_council.lever.poll]
kind = "flow"
flow = "head_tax"
min = 0
max = 20000
max_step = 5000
[modules.policy.actor.crown_council.schedule.poll.0]
bp = 10000
[modules.policy.actor.crown_council.schedule.poll.120]
bp = 15000
[modules.policy.actor.crown_council.schedule.poll.240]
bp = 10000

[modules.policy.actor.inland_council]         # ActorID 2
jurisdiction  = "inland"
policy        = "rule"
every         = 30
reads_regions = true
[modules.policy.actor.inland_council.lever.stipend]
kind = "flow"
flow = "bread_stipend"
min = 0
max = 20000
max_step = 1000
[modules.policy.actor.inland_council.rule.stipend]
gauge    = "unfed_share"
target   = 500
deadband = 100
span     = 1000
move     = 2000
raise    = true
```

### 14.1 Example A: cadences and a schedule

The actors in ascending ID are `coast_council` (0), `crown_council` (1) and
`inland_council` (2). `Weyl2(a) = a * 0x6A09E667 mod 2^32` is 0, 1 779 033 703 and
3 558 067 406. The buckets and the due ticks:

| Actor | `Every` | `bucket` | Due when `(t + bucket) mod Every = 0` | First due ticks |
|-------|--------:|---------:|----------------------------------------|-----------------|
| `coast_council` | 10 | `0` | `t` a multiple of 10 | 0, 10, 20, ... |
| `crown_council` | 10 | `(1 779 033 703 * 10) >> 32 = 4` | `t mod 10 = 6` | 6, 16, 26, ... |
| `inland_council` | 30 | `(3 558 067 406 * 30) >> 32 = 24` | `t mod 30 = 6` | 6, 36, 66, 96, 126 |

`coast_council` is due on ticks that are multiples of 10 and `crown_council` on ticks
that are 6 mod 10. `inland_council`, at `Every` 30, is also on ticks that are 6 mod 10, so
the two meet at 6, 36, 66, ... (the rotation spreads actors of one cadence, 7.2.2; it does not
exclude a meeting), and at those ticks they are called in ascending `ActorID`:
`crown_council` first.

`crown_council` returns the value of its schedule at each due tick: 10 000 until the
step at 120. At tick 6 it returns 10 000, which equals the lever's default, so the entry
is `unchanged` and nothing is written (6.2.4). The step at 120 is applied at the **first
due tick at or after it**, 126 (8.2.2): the entry is `(poll, 15 000)`, the current value
is 10 000, the difference 5 000 equals `MaxStep` (validation required it, 8.2.3), the
outcome is `applied`, and P0.3a of tick 126 emits `MoneyFlowScaled {head_tax, from 10 000,
to 15 000, cause policy}` and `PolicyDecided {Actor 1, Entries 1, Applied 1}`. The step at
240 is applied at 246. Over ticks 0 to 259 the lever is written twice and the actor is
called 26 times.

### 14.2 Example B: the rule and its gauges

`inland` covers `farm` (400 households), `hills` (150) and `mill` (450): `H = 1 000`.
`inland_council` is due at tick 6 and then every 30 ticks. The rule: target 500 bp (5%
unfed), deadband 100, span 1 000, move 2 000, `Raise`: **an error of 1 000 bp of unfed
share (ten percentage points) moves the stipend by 2 000 bp of its configured amount**.
The lever starts at its default 10 000, bounds 0 to 20 000, `MaxStep` 1 000.

| Decision tick | Unfed households | Reading (bp) | Error | `d` (8.3.1) | Outcome | Lever |
|--------------:|-----------------:|-------------:|------:|------------:|---------|------:|
| 6 | 90 | `rdiv(10 000 * 90, 1 000) = 900` | +400 | `trunc(400 * 2 000 / 1 000) = +800` | applied | 10 000 to 10 800 |
| 36 | 70 | 700 | +200 | +400 | applied | 10 800 to 11 200 |
| 66 | 52 | 520 | +20 | `|20| <= 100`: rest | no entry, `InBand` 1 | 11 200 |
| 96 | 160 | 1 600 | +1 100 | +2 200, cut to `MaxStep` | applied, `Stepped` | 11 200 to 12 200 |
| 126 | 30 | 300 | -200 | -400 | applied | 12 200 to 11 800 |

At tick 66 `PolicyDecided` is emitted with `InBand` 1 and nothing else (the actor was due
and held a reaction for a reason, 12.1.1). The stipend rises while the unfed share is
above 5% and falls when it is below, and rests inside 4% to 6%: an **integrator** with a
rest band (8.3.5).

**The other gauges** on made-up rows. `mean_ratio`: role rows `(farm, grower, 12 000,
300 households)`, `(hills, grower, 8 000, 100)`, `(mill, miller, 15 000, 200)` give
`rdiv(12 000 * 300 + 8 000 * 100 + 15 000 * 200, 600) = rdiv(7 400 000, 600)` = 12 333
(12 333.33, half to even). `price` of `loaf` with `MaxAge` 3: `farm` 24 (age 1, 400
households), `hills` 30 (age 5: **not counted**, older than `MaxAge`), `mill` 20 (age
2, 450): `rdiv(400 * 24 + 450 * 20, 850) = rdiv(18 600, 850)` = 22 (21.88).

### 14.3 Example C: the checks of a returned list

A consumer's `Policy` for `inland_council` returns, with `stipend` now at 11 000, bounds
0 to 20 000 and `MaxStep` 1 000: `[(stipend, 25 000), (stipend, 9 000), (lever 7, 1)]`.

| Entry | Check | Outcome |
|-------|-------|---------|
| `(stipend, 25 000)` | cut to `Max`: 20 000 (`Clamped`); then to `11 000 + 1 000`: 12 000 (`Stepped`) | `applied`, to 12 000 |
| `(stipend, 9 000)` | the lever already has an entry in this list | `duplicate` (the first stands) |
| `(lever 7, 1)` | no lever 7 | `not_held` |

`PolicyDecided {Entries 3, Applied 1, NotHeld 1, Duplicate 1, Clamped 1, Stepped 1}`;
`Entries = Applied + Unchanged + NotHeld + Duplicate + NoClose` holds (1 + 0 + 1 + 1 +
0 = 3).

### 14.4 Example D: an embargo as a directed lever

`coast_council` holds `gate`, an `Against inland` lever. `Between(g, coast, inland)` is
edge 4 (`harbour>mill`) and `Between(g, inland, coast)` is edge 8 (`mill>harbour`)
(`spec-polity.md` 12.1), so the lever holds `{4, 8}` and no other actor may hold them
(`ErrLeverShared`). At tick 95 the player calls `DirectPolicy([{Actor 0, Levers:
[{Lever 0, BP 10 000, Closed true}]}])`: checked (the lever exists, 10 000 is in
`[10 000, 40 000]`, `MayClose`), stored; `Snapshot().Actors[0].Pending` is 1. The actor
is due at ticks 100, 110, ...: a checkpoint at tick 97 saves the mailbox (13), and a
restore finds it. At P0.3a of tick 100: the lever's current value is `(10 000, open)`, the
entry sets `Closed`, the outcome is `applied`, the owner's setter is called once for
both keys (the graph is rebuilt once), and the events are `EdgeStatusChanged {4, cause
policy}`, `EdgeStatusChanged {8, cause policy}` in `EdgeID` order and then
`PolicyDecided {Actor 0, Entries 1, Applied 1}`. The mailbox is cleared. A shipment
already on edge 4 keeps its arrival tick (REG 7). The player reopens the border by
another directive, due at 110 at the earliest.

## 15. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9; fields as its section 3. The
register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn` numbers (N-103); entries
here are `PLC-An`, dampers `PLC-Dn` and scenarios `PLC-Sn`, and say what they merge
with. **Status is `model` where section 16 supports it and `hand` otherwise; nothing
here claims stability** (N-027). The checklist's 9.1 question has one answer: the module
reads the boundary state of the regions its jurisdiction covers, its treasury and its
own levers, and changes the value of a slot that an owner's rule reads at its position;
it reads no event, an agent or the engine's own aggregate, and a built-in that does not
read the jurisdiction (`hold`, `schedule`, `directed`) closes no loop. **A policy that reads
the jurisdiction closes one**, and the library guarantees conservation and determinism of the
setting, not the stability of the loop (N-131; STB-L121).

**Dampers this spec adds or specifies:**

| ID | Damper | Acts on | Home | Status |
|----|--------|---------|------|--------|
| PLC-D1 | The lever's bounds `Min` and `Max` (a mandate): an actor cannot set a value outside them whatever its policy returns | PLC-A1, A3, A4 (the amplitude of any cycle) | PLC 3.4, 6.2 | specified; counted by `Clamped` |
| PLC-D2 | The step limit `MaxStep`: the amplitude of a limit cycle is proportional to it (model) | PLC-A1, A3, A6 | PLC 6.2 | specified; counted by `Stepped` |
| PLC-D3 | The deadband of a rule: a reading inside the band is a rest, so noise does not make the lever hunt | PLC-A1, A2 | PLC 8.3 | specified; counted by `InBand` |
| PLC-D4 | The cadence `Every`, by rotation: the plant is given time to answer, and actors of one cadence are spread over its period (two may share a tick) | PLC-A1, A3, A6 | PLC 7.2 | specified |
| PLC-D5 | The stale-reading guard `MaxAge` of the `price` gauge: a price older than the bound is not a reading, the rule holds | PLC-A2 | PLC 8.3.2 | specified; counted by `NoReading` |
| PLC-D6 | One holder per key and refusals counted: no two writers on a lever, and a setting outside the mandate is dropped and counted | PLC-A3 (two writers on one key) | PLC 3.5.2, 6.2 | specified; counted by `NotHeld`, `Duplicate`, `NoClose` |

**PLC-A1. A rule that reads the jurisdiction is a controller (neighbours STB-L121, MON-A1, FIS
neighbours L84 and L85).** *Kind:* loop. *Loop:* a reading (+/-) the rule's move (+/-)
the lever (a scale, a factor) (+/-) the plant (money, prices, hunger, trade) (+/-) the
reading. The sign is set by the pack: a rule that **raises a payout when the unfed
share is above target** is balancing if the payout reduces hunger, and a rule that
**raises a duty when the treasury is below target** is balancing through revenue only
below the peak of the revenue curve (FIS-A2) and reinforcing above it. *Delay:* the
plant's response time `L` (the slot's own: a flow scale at P1.7 acts in the tick; a
transit factor only on new dispatches; a price age adds its own) plus the sampling of
`Every`, in decisions `d = max(0, ceil(L / Every) - 1)` (0 for a plant that answers inside the tick). *Owner:* PLC 8.3, the consumer's policy.
*Damper:* PLC-D1 to D4. *Undamped:* a rule with no deadband and no step limit, or a
gain above the bound. *Condition:* for an integrating rule with a loop gain
`K = (Move / Span) * |plant slope|` per decision, the loop is stable iff `K < 2
sin(pi / (2 (2d + 1)))` (`delay.py` matches it to 0.0005 for `d` 0 to 8: 2, 1, 0.618,
0.445, 0.347, 0.285, 0.241, 0.209, 0.185); the bound per tick is nearly independent of
the cadence and is highest when `Every` equals the delay (`delay.py`, 16). *Test:*
PLC-S2. *Status:* model, on a linear plant with a pure delay; no run of the library.

**PLC-A2. Windup on a stale reading (neighbour N-045, the price age of CLR 5.4).**
*Kind:* hazard. *Loop:* a price that is carried forward from the last trade, above the
target, keeps the rule stepping each decision: the lever integrates a number that has
stopped changing and goes to its bound (`cycle.py`, second part: 100 decisions give a
lever at 20 000 from 10 000 with the step limit 100 and no other guard). *Delay:* none; it
is an open-loop drift. *Owner:* PLC 8.3.2. *Damper:* PLC-D5, PLC-D1, PLC-D3.
*Undamped:* a `price` gauge with a large `MaxAge`. *Condition:* `MaxAge` below the
longest time a market is expected to go without a trade; with `MaxAge` 3 the lever made
3 changes and stopped (`cycle.py`). *Test:* PLC-S2. *Status:* model.

**PLC-A3. Two actors, one economy (neighbours FIS-A7, L84, L85, MER-A1).** *Kind:* loop,
a strategic interaction. *Loop:* one actor's lever moves a trade, a price or a
treasury that another actor's gauge reads, and each answers: a duty lowered to draw
trade, then the neighbour's, to the lower bound of both; or a closed border answered by
a closed border. *Delay:* the sum of both. *Owner:* PLC 3.5.2, the pack. *Damper:*
PLC-D1 (the race ends at `Min`), PLC-D2, PLC-D4; the one-holder rule prevents only two
actors on one key. *Undamped:* by design: whether two authorities should compete is the
setting's. *Condition:* none; the amplitude of any cycle is bounded by the bounds.
*Test:* PLC-S3 (a report). *Status:* hand.

**PLC-A4. A bound hides a runaway (neighbour N-045).** *Kind:* hazard. *Loop:* a lever
pinned at `Min` or `Max` while the reading stays far from target: the rule's step is cut
to nothing each decision, the plant is not being acted on, and the lever's value looks
stable. *Delay:* none. *Owner:* PLC 6.2. *Damper:* counted, not removed: `Clamped` in
`PolicyDecided` and `economysim.policy.cut` say it every decision (a silent clamp is a
defect, N-045). *Undamped:* a mandate too narrow for the shock. *Condition:* a
consumer reads `Clamped` over a window with the unfed share or the gauge in view.
*Test:* PLC-S2. *Status:* hand.

**PLC-A5. A schedule is open loop (neighbour STB-L23, the seasonal forcing).** *Kind:*
bias. *Loop:* none: a plan raises a rate on its dates whatever the state, so a duty
raised on a seasonal date during a failed yield is pro-cyclical. *Owner:* PLC 8.2.
*Delay:* none (open loop). *Damper:* none, by design; the consumer's alternative is a `rule`
or a directive. *Undamped:* the whole plan: nothing in the design checks a schedule
against the state. *Condition:* a pack with a schedule and a shock script reads the scenario S-run before it
reads a policy effect. *Test:* PLC-S1. *Status:* hand.

**PLC-A6. A directive waits for its cadence (neighbour STB-L121).** *Kind:* hazard. *Loop:*
a human reads the jurisdiction, directs a lever, and the effect arrives after the actor's next
decision tick and the plant's lag: the human closes a loop of delay `Every / 2 + L` on
average. A slow-sitting actor with a fast reading is a controller with a long delay and a high
gain. *Delay:* `Every / 2 + L` ticks. *Owner:* PLC 9.1.4. *Damper:* PLC-D1, D2, D4.
*Undamped:* the player's own gain, which no mechanism limits. *Condition:* `Every` no larger than
the plant's lag when the player is expected to track a measure. *Test:* PLC-S4.
*Status:* hand.

**PLC-A7. A consumer's policy that draws is not a paired neighbour (WLD-A5, L120).**
*Kind:* bias. *Loop:* none: a consumer's `Decide` that takes words from the world's
stream shifts every later draw, so "actor on, actor off" is not an aligned pair
(`spec-world.md` 7.6.1). The built-ins take none, so with them a comparison is exact
until a lever changes a count. *Delay:* none. *Owner:* PLC 7.3. *Damper:* none by design;
compare over seeds. *Undamped:* the shift of the stream, whole. *Condition:* a paired
comparison of two worlds is exact only while every registered policy takes the same count of
words in both, or none (the built-ins); otherwise compare over seeds. *Test:* PLC-S5. *Status:* hand.

**Scenarios** (for `spec-stylised-facts.md` and `spec-experiments.md`, which own the
bands; the first, fourth and fifth are harness tests of the library's own procedure, in
the class of S77 and S78, outside the band rule of FR-STY-098). **PLC-S1** *A schedule is
exact*: the kingdom with `crown_council` of 14.1 over seeds 1 to 10, 260 ticks; pass when
`MoneyFlowScaled` with the cause `policy` is emitted at ticks 126 and 246 and no other,
and the flow's scale is 15 000 over 126 to 245. **PLC-S2** *A rule recovers a shock*:
`five` with a fiscal-governed overlay, INT-S1's yield shock, an actor of 14.2 on the
unfed share against the same world with the actor at `hold`; pass when conservation
holds, the lever stays in its bounds and the unfed share returns inside the band STY
sets; also run the same actor with `Every` 1 and a gain above the model's bound, which
is **reported, not asserted** (it may ring: PLC-A1). **PLC-S3** *Two rules face to face*:
two jurisdictions each with a duty rule on the other's border trade; report the lever
paths (PLC-A3). **PLC-S4** *A directive is applied at the cadence*: `DirectPolicy` at
tick 95, due at 100, a checkpoint at 97 and a restore; the restored world applies it
at 100 and its checkpoint at 101 equals the original's byte for byte. **PLC-S5**
*Neutrality*: a world with every actor on `hold` equals the world without the module
(FR-PLC-001), and a `rule` that never leaves its band equals `hold`.

## 16. What was run

`models/plc/` (Docker, standard library only, `PYTHONDONTWRITEBYTECODE=1`, a read-only
mount; commands in its README). They are prototypes of the rule, not of the library, and
no default is claimed stable:

- `delay.py`: the integral rule `u[k+1] = u[k] - K (y[k] - r)` on a plant `y[k] = u[k-d]`
  (a pure delay of `d` decisions), by simulation with bisection on `K`, against the closed
  form `2 sin(pi / (2 (2d + 1)))`. The two agree to 0.0005 for `d` 0 to 8 (0 to 8: 2.0000,
  1.0000, 0.6180, 0.4450, 0.3473, 0.2846, 0.2411, 0.2091, 0.1845). For a plant lag of 30
  ticks the largest gain **per tick** is 0.053 (a decision each tick), 0.057 (each 5),
  0.062 (each 10) and 0.067 (each 30, when the cadence equals the lag), and 0.033 when
  the actor decides each 60; for a lag of 60 it is 0.026, 0.027, 0.029, 0.033 (each 30)
  and 0.033 (each 60). The cadence is not a free lever on gain: deciding faster than the
  plant answers buys no gain and costs a view.
- `cycle.py`: the integer rule of 8.3 on a plant with slope -0.4 and a delay of 0 or 2
  decisions after a +600 shock at tick 20 (equilibrium lever 11 500), 600 decisions.
  At `d = 0`, `K` 0.4 settles at 11 498 in 14 changes; `K` 1.9 (below 2) rings slowly (580
  changes, ending 11 489 to 11 508); `K` 2.4 (above 2) swings between 0 and 27 600 with
  287 decisions on the lower bound. At `d = 2` (bound 0.618), `K` 0.4 settles in 54
  changes; `K` 0.8 with no damper swings 0 to 29 440 with 132 decisions at a bound; with
  `MaxStep` 200 it holds a limit cycle of 11 120 to 11 880, with `MaxStep` 50 of 11 410 to
  11 590 (the amplitude is proportional to the step limit), and with `MaxStep` 50 and a
  deadband of 40 it **settles at 11 500 in 30 changes**; bounds of 8 000 to 12 000 hold
  the cycle at 10 720 to 12 000 with 137 decisions on the bound. With noise of 30 and
  `K` 0.4, a deadband of 40 gives 22 changes in 600 decisions and none without it gives
  587. The second part is PLC-A2's: a reading stuck at 1 500 against a target of 1 000.
- `worked.py`: the arithmetic of 14: the buckets and due ticks, the schedule's
  applications, the five decisions of the rule, the two weighted gauges with their
  rounding, the expansion of `Against` in the kingdom (`Between(coast, inland)` = [4],
  `Between(inland, coast)` = [8]) and the checks of a returned list.

**What they do not show.** They check that the rule has the properties claimed and give
the numbers of the examples. The plant is linear and has a pure delay; the library has
none of it, no beliefs and no entry, so PLC-A3 to A7 are `hand`. They are not evidence that
any target, move, span, bound or cadence is stable (N-027).

## 17. Notes settled, and what this spec needs from the others

### 17.1 Notes

| Note | What this spec does |
|------|---------------------|
| N-016 (the parameter registry) | **Applied.** Section 10 gives every field a row (the handoff has them); no unnamed parameter; the registry test is still to be written |
| N-017 (words and names) | **Applied.** Terms fixed: policy (the interface), policy actor (actor), lever, mandate, reach, holder (of a key), decision (of an actor), due, cadence (`Every`), mailbox, directive (of an actor), gauge, reading, target, deadband, span, move, step limit, rest, reaction, schedule (step, cycle), `ActorID`, `LeverID`. `directive` also means AGT's; the actor's is qualified "of an actor". Setting words only in the flagged examples |
| N-021 (anything that creates or destroys money or goods) | **Applied: none.** No faucet, no sink, no reason, no account (position 13) |
| N-025 (does a policy actor reuse `Strategy`?) | **Applied and closed.** No: its own interface `Policy` with one method; the table of architecture.md 4.3 gains a row (4.7). An authority that also trades is two objects (9.2) |
| N-027 (no default is stable) | **Applied.** No default; the models show properties of the rule |
| N-045 (silent dampers) | **Applied.** Every cut and refusal is counted in `PolicyDecided` (12.1) |
| N-046 (register entries owed) | **Applied for PLC.** PLC-A1 to A7, dampers PLC-D1 to D6 and scenarios PLC-S1 to S5 (15); remove PLC from the areas. PLC leaves the areas; N-046 stays open until the entries of EMP, SAV, CRD, STO, RSK, FIS and LND are merged |
| N-090 (a module that keeps an account in a region) | **Applied: none.** An actor keeps no account and no agent anywhere |
| N-091 (the slots INT must expose) | **Applied.** No slot of its own; one call, `DirectPolicy` (9.1), and an edit to the framing of INT 5 that a policy is a second writer (17.2) |
| N-103 (running writers in parallel) | **Followed.** Only this spec, `models/plc/` and `.supervisor/plc.*` were created; every model ran in Docker with no bytecode; the host `python3` was not run |
| N-104 (region-keyed tables) | **Applied: none.** A lever names a key; a region-keyed rate is reserved with CRD's slot (6.3.2) |
| N-107 (new faucets and sinks start from MON) | **Read.** The module starts from MON's seam (10.4): the flow scale; it adds no flow |
| N-117 (the agent ID space) | **Applied: none.** An actor has no agent ID and shifts no ID; the actors are numbered among themselves (3.2.2) |
| N-122 (first-come caps) | **Read.** The order of actors and levers is ascending ID; no cap of this module is first come, and the one key conflict is refused in the data (3.5.2) |
| N-124 (a paired comparison is exact while aligned) | **Applied.** The built-ins take no draw (7.3); a consumer's that draws is PLC-A7 |
| N-126 (a new event or `Warn` adds a row) | **Applied.** One event and no runtime `Warn`; the rows are in the handoff |
| N-128 (a hosted per-agent field is saved once) | **Applied: none** (13.4) |
| N-130 (what IND cannot measure) | **Read.** Gauge `mean_ratio` reads the role rows, not the households, so it differs from the preset's mean by the rounding of the role means (8.3.2) |
| N-131 (not in version 1) | **Respected and extended.** Not offered: a timer, a condition ("if X then at tick"), a bundle of levers applied as one beyond the actor's own atomic application, a policy that reads another jurisdiction's regions, a treasury that bids under a strategy, an actor that changes a cadence, a proportional rule (19) |

### 17.2 What this spec needs from the other specs

Each row is an owed edit; the consolidated table with exact text is in the handoff
(Edits owed to other specs).

| File | Section | The change |
|------|---------|------------|
| `rules/architecture.md` | 4.3 (table, Extension state), 7.7, 2.6 | the fifth row of 4.7; the three words; `policy.go` and `internal/policy/` (fresh tag, by the supervisor) |
| `spec-world.md` | 6.2, 6.11, 7.3, 8.2, 9.2, 9.4.2, 13 | the position P0.3a; the event order item; the row D17 and the tick's words; a note under the slot table that a policy actor is a second writer at P0.3a; the row of the cadence table; the sentence "the world contains none"; the row for PLC in 13 |
| `spec-world.md` | 4.1, 3.2.2, 8.2 / 8.3 | the option `WithPolicy`; the name check `ErrUnknownPolicy`; a row in the slot table 8.2 or a line in 8.3 for the actor's mailbox (WLD 8.2.1: a new slot gets a row in the change that introduces it) |
| `spec-integration.md` | 3.3.3, 12.1 | "the cause of an event that carries one is `consumer`" and "events at the call, before `TickStarted`" are true of a consumer's call only: an actor's write is at P0.3a, cause `policy` |
| `spec-integration.md` | 5 (framing), 5.1.3, 7.1, 11 | "a slot is set by a consumer, or by a policy actor at P0.3a (`spec-policy.md`)"; the new call `DirectPolicy` and its errors in a subsection; the sentinels `ErrUnknownActor` and `ErrUnknownLever`; `ErrNotDirected` also names an actor; a row in the list of calls |
| `spec-observability.md` | 3.2.4, 4.1, 4.4, 5.3, 6.2, 6.5, 7.2, 7.3, 7.5, 7.6 | the names table gains `Actors`; one event; `Cause` on two events and a member, and the Position column of 4.1 ("between steps", "call") for `MoneyFlowScaled`, `EdgeStatusChanged` and `TransitFactorChanged`; the order item; `Snapshot.Actors`; four instruments; the new attributes; the construction warnings; 7.2: the "world created" keys `actors` and `levers`; 7.5 and 7.6: the new keys against the rules |
| `rules/observability-and-logging.md` | 3.3.2, 4.3, 5.3, 5.3.1, 5.4.1 | the keys `actor`, `lever`, `reading`, `entries`, `changed`, `actors`, `levers`; the event and the changed ones, and the "Emitted when" cells of `MoneyFlowScaled`, `EdgeStatusChanged` and `TransitFactorChanged` ("between steps", "at the call"), which gain "or at P0.3a for an actor's write"; the four instruments and their feeders; the attributes (fresh tag) |
| `spec-checkpoint.md` | 4.2, 4.2.4, 4.3, 6.2, 7.1.2, 7.2, 7.5 | the section `Policy`; the extension kind `3` policy; the derived-and-not-saved list; the restore checks of 13.2; `Stateful`'s doc comment names `Policy`; 7.5: the built-ins keep no state |
| `spec-config.md` | 3.2, 3.7, 4.1, 7.3 to 7.6, 8, 9.3 | the module row, the fields, the bounds, the sentinels and warnings, the hash |
| `spec-polity.md` | 5.4.2, 6.3 (H7, H10, H13), 17 | the PLC row of 5.4.2 and H7 marked used; "H13: INT's consumer embargo" gains "or a lever of kind edge with `Against`" |
| `spec-fiscal.md` | 13.3, 13.4 | "PLC" names `spec-policy.md` 6.3 (a lever of kind `flow`); FIS-A2 and A4 are neighbours of PLC-A1 |
| `spec-money-supply.md` | 10.4 | "a future PLC" becomes `spec-policy.md`; the lever's reach for a target kind `treasury` |
| `spec-regions.md` | 7.3 | a second writer of an edge status: the internal form is called by PLC at P0.3a with the cause `policy` |
| `spec-transport.md` | 10.3 | the same for the transit factor, the cause `policy` and the event's `Cause` |
| `spec-agents.md` | 10.2, 6.6 | one sentence: a policy actor's directive is `spec-policy.md` 8.4; `Directed` strategies and `directed` policies are different objects |
| `spec-indicators.md` | 3.1.3, 9 | 3.1.3 points to PLC; the asks below |
| `spec-stability.md` | 4, 4.8, 8, 10 | merge PLC-A1 to A7, D1 to D6, S1 to S5 with numbers; drop the PLC hook rows of 4.8 and 10; FR-STB counts |
| `spec-calibration.md` | 6 | one registry row per tunable of 10, classified (handoff) |
| `spec-reference-chain.md` | 9, 10 | an overlay `policy.toml` for `five` and the scenarios PLC-S1 to S5 |
| `spec-stylised-facts.md` | 5, 8 | the class of PLC-S1, S4, S5; PLC-S2's pass is a band to set |
| `spec-institutions.md`, `spec-credit.md`, `spec-land.md` | the slot's section | when integrated, the slot is a reserved lever kind (6.3.2): one sentence each |
| `README.md`, `spec-roadmap.md`, `documentation_index.md`, `notes.md` | index, code and parameter-owner tables, glossary; PLC row; rows for `spec-policy.md` and `models/plc/`; the notes of 17.1 and the new notes (fresh `NOTES-TAG`) | the handoff's rows |
| `AGENTS.md` | layout | `policy.go` and `internal/policy/` |
| `doc/adr/0001-...`, `0003-...` | open point 1; decision 3 | a note that open point 1 is answered: a fifth extension point (4.7.2), the typed actions of ADR 0001 decision 4 narrow to one, and the second sentence of ADR 0003 decision 3 ("a ruler that also trades uses `Strategy` for its orders and the policy interface for its settings") is narrowed to two objects (9.2) |

**IND preset asks** (not adopted by this spec): the **selection of a region set** for
presets 3, 4, 10 and 12 (a jurisdiction's `Regions(j)` as a first-class set, so that a
consumer's policy and the gauges of 8.3.2 are the same function); `PriceLevelOf(commodity,
set)`, the household-weighted published price; and the **treasury term** of `Sectors`
(an ask FIS already has). A reading of a *flow* over a window by a consumer's policy
(revenue, trade) is IND's `Revenue` and `TradeBalance` over two snapshots; none needs an
engine aggregate.

### 17.3 Conflicts found between specs

1. **IND position 2 against the brief's "reads IND measures".** `spec-indicators.md` 3.1.1
   and FR-IND-001 forbid any package of the module from importing `indicator`; a built-in
   rule cannot call a preset. Resolved here by four gauges that are field reads, with the
   definition of the matching preset, and the consumer's policy as the place a preset is
   called (position 7, 8.3.3). The alternative, a `policy/` package that imports `indicator/`,
   needs `architecture.md` 2.7 and FR-IND-001 amended; it is open question 1.
2. **WLD 8.1.1 (a slot acts between ticks, never inside `Step`) against a policy that
   writes inside one.** Resolved by P0.3a at the entry, before any position that reads a
   slot, and by a sentence in WLD 8.2 (7.1.3).
3. **ADR 0001 decision 4 lists three actions; this spec has one.** Resolved in 4.7.2 and
   position 3: a transfer and stock to a boundary are a declared flow's scale.
4. **INT 3.6 / N-131 "FIS and PLC are where a rule that sets taxes or rates belongs"** is
   met by 6.3 (a lever of kind `flow`); INT's wording "outside the engine" is true of a
   consumer's own controller and not of this module's.
5. **POL 5.4.2 and H7 say "one policy per governed jurisdiction".** This spec also
   accepts an actor on a grouping (no seat): an actor has no treasury then, and a
   regional rate or an embargo is a mandate over a set of regions, for which a grouping is
   the natural holder (3.2.1). POL's text is edited (17.2).

## 18. Requirements

IDs are stable (testing.md 4.1). A requirement is a single testable statement. The
numbers run in groups (001 to 013 the module and its validation, 020 to 025 the
interface, 030 to 035 reads, 040 to 047 settings, 050 to 055 position, cadence and draws,
060 to 064 the built-ins, 070 to 074 the directed policy and its call, 080 to 088
observability, 090 to 093 state and determinism), and a gap is not a removed requirement.

### Functional

| ID | Requirement |
|----|-------------|
| FR-PLC-001 | With `Modules.Policy` nil, with the module on and every actor's policy `hold`, and with the module on and no actor due in a tick, no result, event, snapshot field or golden line of any world MAY differ from a world without the module, except the configuration hash, the names table, `Snapshot.Modules` and `Snapshot.Actors`, the construction summary and the module's warnings (1, 3.1). |
| FR-PLC-002 | `Modules.Policy` MUST be rejected with `ErrModuleDependency` while `Modules.Polity` is nil or has no jurisdiction; a `flow` lever with neither the money supply nor the fiscal module on, a `transit` lever with transport off and MUST be rejected the same way (3.1.1, 11.1). |
| FR-PLC-003 | Actors MUST be sorted by ID bytewise before a world is built or a hash taken, the `ActorID` MUST be the index in that order and the `LeverID` the index in the actor's lever order; the hash MUST cover the module when present (and nothing when it is off, `spec-config.md` 8.2.1), every field of every actor except `Label`, and the policy by name only (3.1.3, 3.2.2). |
| FR-PLC-004 | At most one actor MUST attach to one jurisdiction (`ErrPolicyShape`, `one_actor_per_jurisdiction`); an actor ID, a lever ID within an actor, a step `(Lever, At)` or a reaction `Lever` repeated MUST be `ErrDuplicateID`; more than `MaxActors`, `MaxLevers`, `MaxScheduleSteps` or `MaxLeverKeys` MUST be `ErrTooMany` (3.2.1, 11.1). |
| FR-PLC-005 | `Schedule` or `Cycle` with a policy other than `schedule`, and `Rules` with a policy other than `rule`, MUST be `ErrPolicyShape` (`policy_data`) (11.1). |
| FR-PLC-006 | A lever MUST have exactly the key fields of its kind and `MayClose` only on kind `edge` (`ErrPolicyShape`, `lever_key`); a key that is not defined MUST be the unknown-entity sentinel of its kind; a `transit` lever MUST be a directed pair of two distinct regions that are not outside nodes, adjacent or not (`ErrPolicyShape`, `transit_pair`); and a kind that is not in 6.3.1, a reserved kind included, MUST be `ErrOutOfBounds` (11.1). |
| FR-PLC-007 | A key outside the reach of 3.5.1 MUST be `ErrLeverNotInReach` with the path of the lever and the key. |
| FR-PLC-008 | An `Against` lever MUST hold exactly the edges of `Between(g, J, K)` and `Between(g, K, J)` in ascending `EdgeID`, MUST be rejected when `K` is `J`, an ancestor or a descendant (`against_nested`), and MUST be moved as one (3.4.3). |
| FR-PLC-009 | After the expansion of every `Against`, a key held by two levers of two actors MUST be `ErrLeverShared` (3.5.2). |
| FR-PLC-010 | A lever MUST satisfy `Min <= Default <= Max` with both inside the slot's own bound and `MaxStep` 0 or at least 1, else `ErrOutOfBounds` (3.4.1). |
| FR-PLC-011 | A schedule MUST be rejected when a step names no lever of the actor, a `BP` is outside the lever's bounds, `Closed` is set without `MayClose`, or, with `MaxStep` above 0, two consecutive values of one lever (the first from the default, and with a `Cycle` the first again from the last) differ by more than `MaxStep`, or, with a `Cycle`, an `At` is not below it (`schedule_lever`, `schedule_value`, `schedule_step`) (8.2.3). |
| FR-PLC-012 | A reaction MUST be rejected when it names no lever of the actor, a second reaction names the same lever, a `price` gauge lacks `Commodity` or `MaxAge` or another gauge has them, a region gauge has `ReadsRegions` false (`rule_lever`, `rule_gauge`, `rule_reads`), or a `treasury` gauge is on an ungoverned jurisdiction (`ErrNotGoverned`) (11.1). |
| FR-PLC-013 | The warnings of 11.2 MUST NOT fail validation and each MUST name its rule and path; `schedule_dense` MUST be raised for two steps of one lever closer than `Every` with `MaxStep` above 0. |
| FR-PLC-020 | `Policy` MUST have exactly the method `Decide(PolicyView, *rand.Rand) Settings`, the default `HoldPolicy` MUST return no settings, and every built-in MUST have a compile-time assertion (4.1, 4.4). |
| FR-PLC-021 | `WithPolicy` MUST be additive; a name given twice or one of `hold`, `schedule`, `rule`, `directed` MUST be `ErrDuplicateID`, a nil policy `ErrNilExtension`, and an actor's name that is neither built in nor registered `ErrUnknownPolicy`, after `Validate` succeeds (4.5). |
| FR-PLC-022 | `Decide` MUST receive the world's `*rand.Rand`; the doc comment of `Policy` MUST state the contract of architecture.md 7.7; the library MUST create no other source for a policy (4.1). |
| FR-PLC-023 | A registered policy that implements `Stateful` MUST be saved and restored as an extension of kind `3` by name; a built-in MUST implement none (4.6). |
| FR-PLC-024 | `Decide` MUST return no error; a refused entry MUST be counted and never fail a step; a panic in `Decide` MUST fail the world as any extension's does (4.1.2, 6.2.3). |
| FR-PLC-025 | A `PolicyView` MUST be a deep copy built fresh for each call, MUST contain no event, no pointer and no function, and its slices MUST be in the canonical order of their keys (4.2, 4.3.1). |
| FR-PLC-030 | `PolicyView.Regions` MUST be empty when `ReadsRegions` is false, and otherwise MUST hold the region state of each covered region that is not an outside node, ascending, equal to the filter of a full snapshot by region (5.1). |
| FR-PLC-031 | `HasTreasury` MUST be true exactly when the jurisdiction is governed, and `Treasury` the balance of its account at the boundary (3.2.1, 5.1). |
| FR-PLC-032 | Every field of a view MUST be a function of the configuration, the tick and the state a checkpoint saves: a restored world MUST build the same view at the same tick (5.2.1). |
| FR-PLC-033 | A run with a recording observer and a capturing logger and a run with neither MUST give the same views, settings, checkpoint bytes and final stream state (5.2.1, 4.3.1). |
| FR-PLC-034 | Each gauge MUST be the value of 8.3.2, rounded half to even where it is a ratio, and MUST be undefined where that table says so, in which case the rule holds and counts `NoReading`. |
| FR-PLC-035 | No package of the module MAY import `indicator`, and no exported identifier of this module MAY take a `*World` (4.3.2, 4.3.3). |
| FR-PLC-040 | Each returned entry MUST get exactly one outcome from the checks of 6.2.1 in their order, and `Entries` MUST equal `Applied + Unchanged + NotHeld + Duplicate + NoClose`. |
| FR-PLC-041 | A value MUST be cut to `[Min, Max]` and then to `Current ± MaxStep` when `MaxStep` is above 0, in that order; when `Current` is outside `[Min, Max]` the lever MUST move toward the bound by at most `MaxStep` (6.2.2). |
| FR-PLC-042 | An entry with `Closed` true that would close a key (the lever has no `MayClose` or is of another kind, and some key of it is open) MUST be refused whole (`no_close`): neither its number nor its flag is applied; `Closed` true on a lever whose every key is already closed MUST NOT be refused (6.2.1). |
| FR-PLC-043 | An actor's changes MUST be applied together through the owners' internal setters, in the order of 6.4.2, with the graph rebuilt once, each owner's event with the cause `policy` and then `PolicyDecided`; an entry equal to the value held by every key of its lever MUST write nothing and emit no owner's event, and a group MUST be written at every key that differs (6.2.4, 6.4). |
| FR-PLC-044 | A lever set to its default MUST clear its key, and `Slots()` MUST not list it (6.4.4). |
| FR-PLC-045 | A consumer's call to a held key MUST be legal and MUST stand until the actor returns an entry for it; an actor MUST write only keys it returns (6.4.3). |
| FR-PLC-046 | A lever MUST NOT be of a kind that is a period, a probability or a window, and the kinds offered MUST be exactly those of 6.3.1 (position 15, 6.3). |
| FR-PLC-047 | After every application the conservation identities of `spec-ledger.md` 9 MUST hold and the audit MUST pass; no application MAY create or destroy money or goods (position 13). |
| FR-PLC-050 | The actors MUST be called at P0.3a, after `TickStarted` and before P1.1, in ascending `ActorID`, and an actor that is not due MUST not be called (7.1). |
| FR-PLC-051 | An actor MUST be due at tick `t` exactly when `(t + bucket(a, Every)) mod Every = 0` with `bucket(a, P) = (Weyl2(a) * P) >> 32` and `Weyl2(a) = (a * 0x6A09E667) mod 2^32`, `a` its `ActorID` (7.2.1). |
| FR-PLC-052 | An actor due at tick 0 MUST act on the construction state and its settings MUST take effect in tick 0 (7.2.4). |
| FR-PLC-053 | The built-in policies MUST take no random number; a consumer's policy's words MUST be counted in D17, actors ascending, in call order; the count in a tick MUST be a function of the configuration, the state and the registered policies, and MUST NOT depend on an observer, a logger or an idle module (7.3). |
| FR-PLC-054 | No setting of an actor MAY change a cadence, a window length, a gate probability or an `Every` (position 15). |
| FR-PLC-055 | A run split into `Run(n)` calls, and a run checkpointed and restored at a boundary, MUST equal an uninterrupted run (7.4.1). |
| FR-PLC-060 | `hold` MUST return no settings at any tick (8.1). |
| FR-PLC-061 | `schedule` MUST return, at each decision, the value of 8.2.1 for every lever that has one, MUST return none for a lever with no step at or before `u`, and with a `Cycle` MUST repeat the plan and carry the last step across the wrap after the first cycle (8.2.1). |
| FR-PLC-062 | `rule` MUST, for each reaction in ascending lever order, hold with `NoReading` when the gauge is undefined, hold with `InBand` when `|m - Target| <= Deadband`, and otherwise return `v + d` with `d = trunc(e * Move / Span)` in 128 bits (a product of the largest `|e|` and `MaxMoveBP` MUST NOT wrap), negated when `Raise` is false (8.3.1, 8.3.4). |
| FR-PLC-063 | `rule` on a lever of kind `edge` MUST leave `Closed` unchanged, and on an edge a consumer closed, with no `MayClose`, MUST move the factor and keep it closed (8.3.1, 6.2.1). |
| FR-PLC-064 | A `rule` with `MaxStep` 0 and `Deadband` 0 MUST be accepted with the warning `rule_undamped` and MUST NOT be altered (8.3.6). |
| FR-PLC-070 | A `directed` policy MUST return the actor's mailbox and the world MUST clear it after the decision whatever the outcomes; with an empty mailbox it MUST return nothing (8.4.1). |
| FR-PLC-071 | `DirectPolicy` MUST make its checks in the order of 9.1.2, atomic, and MUST draw no random number and emit no event; an empty slice MUST be a no-op; an entry MUST replace the actor's mailbox whole (9.1). |
| FR-PLC-072 | `DirectPolicy` MUST check the bounds `[Min, Max]` and `MayClose` at the call and MUST NOT check `MaxStep`, which acts at the decision (9.1.2). |
| FR-PLC-073 | A mailbox MUST be saved and restored: a directive left before a checkpoint MUST be applied at the same decision tick of the restored world with the same result (13). |
| FR-PLC-074 | `Snapshot.Actors` MUST list every actor ascending with `Pending` the number of settings in a `directed` actor's mailbox and 0 otherwise (12.3). |
| FR-PLC-080 | `PolicyDecided` MUST be emitted once per due actor per tick in which any of its counts is non-zero, after the slot events of the application, with the fields of 12.1.1, and MUST NOT otherwise. |
| FR-PLC-081 | An actor with policy `hold` MUST never emit `PolicyDecided`, and a due actor that returned nothing and held no reaction for a reason MUST emit nothing (12.1.1). |
| FR-PLC-082 | In every `PolicyDecided`, `Entries` MUST equal the sum of the five outcomes of 6.2.1 (12.1.1). |
| FR-PLC-083 | The events of P0.3a MUST come after `TickStarted` and before the first event of phase 1, per actor in ascending `ActorID`: the slot events of 6.4.2 and then `PolicyDecided` (12.1.3). |
| FR-PLC-084 | A step MUST NOT log at `Info`, `Warn` or `Error` for this module; its `Debug` lines MUST be guarded by `Enabled`, have constant messages and use the keys `actor`, `lever`, `reason`, `reading`, `from`, `to`, `entries`, `changed` (12.2). |
| FR-PLC-085 | `MoneyFlowScaled` and `TransitFactorChanged` MUST carry `Cause`, and `EdgeStatusChanged.Cause` MUST have the member `policy`; a consumer's call MUST carry `consumer` (12.1.2). |
| FR-PLC-086 | The four instruments of 12.4 MUST be derived from `PolicyDecided` alone and their attributes MUST take only the values of the configuration and of their closed sets (12.4). |
| FR-PLC-090 | The section of 13.1 MUST be present with the module and absent without, written in the order given, sparse; a restore MUST reject as corrupt an actor that does not exist or is not `directed`, a lever out of range, repeated or descending, a value outside `[Min, Max]`, or a row with no setting (13.2). |
| FR-PLC-091 | `internal/policy` MUST declare no floating-point type or constant, no mutable package-level variable and no `go` statement, and MUST import none of `time`, `math/rand`, `math/rand/v2`, `crypto/rand`, `os` and `reflect` (architecture.md 3.5, 7.1, 12.2); an AST and import test of the package checks it. |
| FR-PLC-092 | A policy's levers MUST NOT be saved by this module: a lever's value MUST be the slot's own state in its owner's section (13.3). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-PLC-001 | With the default logger and observer and no actor due, a step of the reference chain MUST allocate nothing for the module; a due `hold`, `schedule` or `directed` actor MUST allocate nothing beyond scratch held by the world and reused across ticks (observability-and-logging.md 8.2). |
| NFR-PLC-002 | The cost of a tick SHOULD be proportional to the number of due actors times the cost of their views divided over the cadence, `O(levers)` with `ReadsRegions` false, and SHOULD have a benchmark with `b.ReportAllocs()`. |
| NFR-PLC-003 | Validation of the module SHOULD have a fuzz test with random actor tables (never panics; a valid table gives the same effective configuration and hash built in Go or read from a file; an invalid one returns a sentinel). |
| NFR-PLC-004 | Every output MUST be bit-identical across runs and platforms: no iteration over a map and no sort without a total order influences a result. |
| NFR-PLC-005 | Observation MUST NOT change a result: the same seed and configuration give identical results with and without a logger and observer. |

## 19. Open questions

1. **A `policy/` package that reads IND measures.** The brief asks for "IND measures";
   the import rule forbids the engine to call them (position 7). A package `policy/`
   that imports the root and `indicator/` would let built-in rules read any preset,
   at the price of amending `architecture.md` 2.7 and FR-IND-001. *Next step:* the
   first consumer that needs a gauge the four do not give, with the measure it needs.
2. **Richer built-in rules.** A proportional rule, a rule with a derivative, a rule that
   acts on a condition ("if the reading is above X for n decisions, close the border")
   needs state of its own (a count of decisions), which is saved by this module. Not
   offered. *Next step:* the first run that shows a stepping rule too slow or too
   noisy; a consumer's `Policy` is the form until then.
3. **What an authority sees beyond its jurisdiction.** An actor reads the regions it covers
   (5.3.1). A neighbour's price or a trade partner's treasury reaches it as nothing.
   A `Sees` list of jurisdictions, or INF's rumour for an authority, would change the
   information structure of a jurisdiction. *Next step:* MIG and INF, if a scenario needs it.
4. **More than one actor per jurisdiction, and one actor for several.** Two bodies of one jurisdiction with disjoint levers; a player over a federation.
   One-per-jurisdiction and several directives in one call cover the cases of version 1.
   *Next step:* the first pack that wants two policies on one jurisdiction.
5. **A treasury that bids under a strategy.** 9.2.2. *Next step:* FIS and STO, if the
   standing orders of FIS are too coarse (a release rule with a state).
6. **A transfer of a region** (POL 7.4). An `Against` lever's key list is derived at
   construction; a transfer changes `Between`. *Next step:* when POL offers it, the
   list is rebuilt at the call, and the levers of the two jurisdictions are
   re-validated; a lever that leaves its reach is the open part.
7. **A query of the actors.** A derived-table reader `Actors()` (ID, jurisdiction,
   policy, levers with their bounds and values, the last decision tick) would save a
   consumer a read of `Config()` and `Slots()`. *Next step:* INT, with the first
   consumer's player UI.
8. **More gauges.** Stock cover of a commodity, an outside trade balance, the price of
   the cheapest region. Each is a row of 8.3.2 and an IND ask. *Next step:* the first
   rule that needs one.
9. **Staggering of equal cadences.** The rotation spreads actors of one cadence by
   their index, which is lumpy for a handful (the buckets 0, 4 and 8 at 10, and an
   exact meeting of cadences with a common factor, 14.1). Acceptable for a few actors; a
   pack with a dozen at one cadence is not a measured case. *Next step:* STY, if a
   regime shows synchronised authorities.
10. **A lever of a MON flow with no authority.** 3.5.1 requires every region the flow
    selects to be covered. A world-wide payout with no jurisdiction above the regions
    is out of reach of every actor. *Next step:* a pack that wants a "world" authority
    writes a root jurisdiction over every region.

## 20. Where this lives in the code

Nothing is written; this is the proposed layout for the first code, checked against
architecture.md 2.6 and `AGENTS.md`.

- `policy.go` (root): the types `PolicyConfig`, `PolicyActor`, `Lever`, `LeverKind`,
  `ScheduleStep`, `Reaction`, `Gauge`, `ActorID`, `LeverID`, `PolicyView`, `LeverView`,
  `Settings`, `LeverSetting` and `PolicyDirective`; the interface `Policy`, `HoldPolicy` and the
  built-ins `Schedule`, `Rule` and `Directed` with their assertions; the position P0.3a; the
  event `PolicyDecided` (in `event.go`); the sentinels `ErrPolicyShape`, `ErrLeverNotInReach`,
  `ErrLeverShared`, `ErrUnknownPolicy`, `ErrUnknownActor` and `ErrUnknownLever` (in
  `errors.go`); `WithPolicy` in `option.go`; `DirectPolicy` in `integration.go`. One file per
  concept.
- `internal/policy/` (standard library only, primitive slices, no root type, architecture.md
  2.9): the schedule's value function, the rule's step with the 128-bit product and the
  truncation, the four gauges over primitive rows, the checks and the two cuts, and the due
  test using the `Weyl2` bucket of `internal/ordering`. The root converts at the call, inline.
- `config/`: the tables `[modules.policy.actor.<id>]`, `.lever.<id>`, `.schedule.<lever>.<tick>`
  and `.rule.<lever>` (the form of `[modules.polity.jurisdiction.<id>]`) and the rules of 11.
- `testdata/chains/overlay/policy.toml` and `policy-directed.toml` (a proposal for REF).
- Tests: table-driven unit tests in `internal/policy` with the cases of the models (the
  schedule's wrap, the rule's rest band and cuts, the gauges' rounding and undefined cases,
  the due test over many `Every`); an external `_test` of the neutrality of FR-PLC-001 over
  seeds; the identity of FR-PLC-040; the application test of 6.1 against a recording observer
  (order, cause, graph rebuilt once); a failing-case test for each sentinel and warning of
  11; `DirectPolicy`'s order of checks; a checkpoint round trip with a directive pending and
  with a stateful consumer's policy; a fuzz test on the module's tables; an AST and import test of `internal/policy` (FR-PLC-091); a test of the 128-bit product at the largest `|e|` and `MaxMoveBP`; an `AllocsPerRun` test
  for NFR-PLC-001; and an `Example` of a player governing the example jurisdictions of 14.
- Rule files, at the first code (owed rows, with a fresh tag each): architecture.md 2.6
  (`policy.go`, `internal/policy/`), 4.3 and 7.7 (4.7); observability-and-logging.md 3.3.2,
  4.3, 5.3 and 5.4.1. The glossary words go to `doc.go` (N-017).
