# Spec: Checkpoint: the complete state of a world as plain data, its canonical encoding, validated restore, extension state and the compatibility policy

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CHK-NNN` and
`NFR-CHK-NNN`, which tests cite (testing.md section 4). They are in section 19.

Layer: core ([ADR 0002](../adr/0002-persistence-by-checkpoint.md): a checkpoint is
not a candidate to defer). It needs `spec-world.md` (WLD: the tick boundary, the
draws, what the world contributes, 8.4 and 12) and reads the state section of every
spec that has one (section 4). It changes no rule of theirs; where one lists state
incompletely, twice or ambiguously, section 4.5 rules and section 20 lists the edit.

This spec adds **no `Config` field, no default and no bound**: a checkpoint is not
configuration, and the format's constants (the magic, the section tags, the version)
are rule constants beside their code (architecture.md 9.6). It adds one option
(`WithReseed`, restore only) and a small set of root names (section 3). The only
numbers in it are those constants and the placeholders of the cost model of section
10. Nothing has run in the library (notes N-015, N-027): the scripts of section 16
are stylised Docker Python models of the rules of section 5, not claims about the
library, and the facts taken from the Go standard library (`binary.Uvarint` accepts a
non-minimal encoding) were checked by running
Go 1.25 in Docker.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

A world runs for a long simulated time and a consumer must be able to stop it, keep
it, and resume it exactly, or branch many runs from one state. The library never
persists (ADR 0002, decision 1): it produces a value, the consumer stores it. This
spec is that value and the call that turns it back into a world. It covers:

- **what a checkpoint is** as a value and when one may be taken (section 3);
- **the inventory**: the state each spec contributes, what is derived and rebuilt,
  what is deliberately not saved, the order it is written in, and the specs that list
  state twice or not at all (section 4);
- **the format**: one canonical binary encoding, its primitives, its sections, its
  version and its digest (section 5);
- **restore**: the validation order, the errors, what is checked against the
  configuration and the registered extensions, the failed-world rule, partial and
  corrupt input (section 6);
- **extension state**: the optional interface a consumer's `Strategy`, `Spawner` or
  `Clearing` implements to be saved, and the amendment it needs to architecture.md 4.3
  and 7.7 (section 7);
- **bit-identity**: the requirement, the test, what is compared and what is not
  guaranteed (section 8);
- **compatibility**: the policy before and after 1.0, migration, and the growth of
  closed accounts (section 9);
- **cost**, in bytes and in counts (section 10), and **what it hands to** EXP, INT,
  OBS, STY and IND (section 11).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| What each piece of state means, and its value bounds | the spec that owns it (section 4 names each) |
| The tick boundary, the failed state, the draws and the options of `NewWorld` | `spec-world.md` (WLD) |
| The configuration hash: its encoding and what it covers | `spec-config.md` 8 (CFG); this spec stores it and compares it |
| The `Snapshot` (an observation summary, not a checkpoint) and the fields of the restore event | `spec-observability.md` (OBS) |
| The public names and bounds of the between-ticks calls (`AddAgent`, the shock slots) | `spec-integration.md` (INT); their state is saved by the spec that owns the slot |
| Where bytes are stored, compressed, encrypted or signed | the consumer (ADR 0002, decision 1) |
| Replay as a way to rebuild a run (the seed, the configuration and the calls) | the consumer; it is a debugging aid, not persistence (ADR 0002, decision 9) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | A checkpoint is an **opaque, immutable value** holding the canonical bytes. The library exports one type, one call to make it, one to rebuild a world and three methods to move the bytes; no struct of state is exported, so the compatibility surface is the byte format and its version | 3 |
| 2 | It is taken **between ticks, on a healthy world, and nowhere else**: a failed world and a world in a step refuse it. A world constructed and not stepped may be checkpointed | 3.2 |
| 3 | The encoding is **canonical**: equal states give equal bytes, and every input that a restore without `WithReseed` accepts re-encodes to the same bytes, so a decoder cannot be fooled by a second spelling of a state | 5 |
| 4 | Integers are minimal-form LEB128 varints (zigzag for signed); there is no map, no float, no pointer, no padding and no timestamp. The trailer is the SHA-256 of everything before it | 5.2, 5.4 |
| 5 | The state is written **by owning spec, in a fixed order**, in framed sections. A count that the configuration implies is not stored; a section of a module that is off is absent | 4, 5.3 |
| 6 | A restore is **all or nothing** and validated in a fixed order: frame, version, digest, configuration hash, extensions, sections, cross-references, the ledger's audit. It returns no world and emits nothing on error | 6.2 |
| 7 | The configuration is **not** in the checkpoint. The consumer passes it again and its hash must match; the registered extensions are checked by name | 6.2 |
| 8 | A restored world is a **new world** that is bit-identical in its future to the one that was checkpointed, given the same calls. Two restores of one checkpoint continue identically | 8 |
| 9 | `WithSeed` is refused on a restore. `WithReseed(salt)` is the restore-only option that gives a branch its own randomness, deterministically | 6.4, 6.8 |
| 10 | Extension state is explicit: an optional two-method interface, `Stateful`, for `Strategy`, `Spawner`, `Clearing` and `Policy`, saved per registered name with a version. An `Observer` is never saved | 7 |
| 11 | Closed accounts are **never compacted** in version 1; a closed slot is saved as two small numbers | 9.4 |
| 12 | Before 1.0 a checkpoint restores only into a library that writes its exact format version. From 1.0, a v1 library restores every v1 format and never a newer one | 9.1, 9.2 |
| 13 | A checkpoint carries no events and no configuration. A consumer that needs history keeps its own | 3.5 |
| 14 | A world checkpointed before its first `Step` is **still in construction** when restored, and keeps its construction bucket, so a later call between ticks behaves as it would have on the original | 4.4, 6.7 |

## 3. The checkpoint as a value

### 3.1 The surface

```go
// Checkpoint is the complete state of a world at a tick boundary, in one canonical
// binary encoding. It is immutable. The zero Checkpoint is empty and not valid.
type Checkpoint struct{ /* unexported: the encoded bytes */ }

// Checkpoint returns the checkpoint of the world. It changes no state and draws no
// random number. It returns ErrWorldFailed on a failed world, ErrInStep inside a step
// and, on a healthy world, ErrCheckpointExtension wrapping the error of an extension
// whose SaveState failed.
func (w *World) Checkpoint() (Checkpoint, error)

// RestoreWorld builds a new world from the configuration and a checkpoint taken from a
// world built from the same configuration (its hash must match).
func RestoreWorld(cfg Config, c Checkpoint, opts ...Option) (*World, error)

// MarshalBinary, AppendBinary and UnmarshalBinary implement encoding.BinaryMarshaler,
// encoding.BinaryAppender and encoding.BinaryUnmarshaler. UnmarshalBinary checks the frame
// (magic, version, digest, header) and nothing else; RestoreWorld checks the rest.
func (c Checkpoint) MarshalBinary() ([]byte, error)
func (c Checkpoint) AppendBinary(b []byte) ([]byte, error)
func (c *Checkpoint) UnmarshalBinary(data []byte) error

// Info reports what the frame says, before any configuration is chosen.
func (c Checkpoint) Info() CheckpointInfo

type CheckpointInfo struct {
	Format     uint32   // the format version of the bytes
	ConfigHash [32]byte // the hash of the effective configuration that the world had
	Tick       int64    // the tick the next Step would have run
	Seed       uint64   // the effective seed: the seed after a reseed, read back by a restore (5.4.1, 6.8.3)
	Bytes      int      // the length of the encoding
}
```

Names are provisional until the code (the same caveat as `spec-world.md` 3.7.2); the
rules fix what they fix: standard-library interfaces (ADR 0002, decision 5), no
struct tags (architecture.md 12.6), `New`-style constructor naming (architecture.md
6.1: the root package has several types, so the constructor is `RestoreWorld`, as
`NewWorld`).

3.1.1. **Why an opaque value and not a struct of plain data.** ADR 0002 decision 3
says a checkpoint is plain data and decision 5 gives one encoding. A struct of
exported state would make every field a compatibility promise (architecture.md 2.8)
and would need a second walk of the world (build the struct, then encode it). The
value holds the bytes the encoder wrote directly from the world, so *equal states
give equal bytes* is a property of the value itself, a consumer cannot build an
inconsistent one, and the plain data is the encoding. A consumer that wants JSON,
protobuf or a database row converts the bytes with a decoder of its own or stores
them whole; there are no struct tags in the core.

3.1.2. **Why `Info`.** A consumer holding several packs chooses the configuration to
restore with by the hash, and wants the tick and the format before it builds a world.
`Info` reads only the frame, so it is available after `UnmarshalBinary`, which has
verified the digest. It has no error result and no section is decoded for it. A checkpoint holds
no state of the random source (ADR 0008): the seed in the header is the one random datum,
so `Info` reports the format, the configuration hash, the tick, the effective seed and
the length, and there is no accessor for a stream.

### 3.2 When

3.2.1. **Between ticks, on a healthy world** (`spec-world.md` 8.4): no reservation is
open, every counterparty account is empty, no shipment is half dispatched and no job is
half started. A world that has been constructed and not stepped may be checkpointed: it
is at the tick 0 boundary and may still be inside its construction phase (4.4, 6.7).

3.2.2. **Refusals.** `Checkpoint` on a failed world returns `ErrWorldFailed` wrapping
the failure (`spec-world.md` 3.5.1), and on a world whose step is running returns
`ErrInStep` (3.6). No checkpoint of a half-finished tick exists, so none can be saved
or resumed (ADR 0002 open point 4, answered: between ticks only).

3.2.3. **The recovery path of a failed world is a restore** of a checkpoint taken at an
earlier boundary (`spec-world.md` 3.5.2). The library keeps no ring of recent
checkpoints; a consumer that wants one keeps its own (a checkpoint of a world of the
first consumer's size is about 2 MB, section 10).

### 3.3 Immutability, copies and threads

3.3.1. A `Checkpoint` holds its bytes and never changes them. `MarshalBinary` returns a
copy; `AppendBinary` appends a copy; `UnmarshalBinary` copies its argument, so the
caller may reuse the slice. `Info` returns values. A `Checkpoint` is therefore safe to
read from another goroutine than the world's (architecture.md 12.4), and one value may
be passed to `RestoreWorld` any number of times, from any number of goroutines, each
building its own world.

3.3.2. `RestoreWorld` reads the value and never writes it.

### 3.4 What the consumer does with it

The consumer stores the bytes where it likes (a file, a row, an object store), may
compress them (they are varints and compress well, 10.3), and hands them back through
`UnmarshalBinary`. It should store the configuration or its identity beside them: the
hash in `Info` says which. The library offers no file name, no `io.Writer` variant and
no store (architecture.md 1.1): `AppendBinary` into the consumer's buffer is the one
step.

### 3.5 What is not in a checkpoint

3.5.1. **The configuration**, the options, the observers and the logger (ADR 0002,
decision 3). The configuration is the pack the consumer already has; carrying it would
make a checkpoint depend on a file format and would allow a restore to "succeed" by
adopting a configuration the consumer did not choose.

3.5.2. **The registrations of extensions** (the code). They are checked by name
against the hash and against the extension blobs (6.2, 7.3).

3.5.3. **Events** (ADR 0002 open point 5, answered: no). Events are a stream already
delivered; a copy in the checkpoint would tie its size to the event volume of the
recent past, would make a restored observer count the same trade twice, and would
carry nothing the state does not. A restored consumer rebuilds its derived state from
`WorldRestored` and a `Snapshot` taken at the boundary (6.7, 11).

3.5.4. **Scratch**: the views, the plans, the per-tick counters and every other value
that `spec-world.md` 6 clears or rebuilds (4.3).

## 4. The inventory

### 4.1 How to read it

4.1.1. A **section** is the state one spec contributes, in one framed block (5.3).
Sections are written, and restored, in the order of the table of 4.2, which is also an
order of dependency: a section refers only to things an earlier section defined (a
ledger account, an agent, a region), so a restore needs no forward reference **except one**: the owner reference of an Agent account in section 2 is an agent ID of section 3, and is checked at step 7 of 6.2.

4.1.2. **Saved** means: it affects a later result or a later event, and it cannot be
recomputed from the saved state and the configuration. **Diagnostic** state affects no
result but affects a log line (a run of ticks that decides when a `Warn` repeats); it is
saved so that a resumed run logs as the original did (FR-CHK-043), as the specs that own
it already say. A value that a position reads and that is not a function of the saved
state and the configuration **is state**, however it is cached: a latch is state
(4.5, F5).

4.1.3. **A count the configuration implies is not stored.** The number of beliefs of an
agent is the size of its role's tradable set, the number of residues is a function of
its role's recipes, the number of flows is the number of flows of the configuration.
These counts are functions of the configuration (whose hash has been checked, 6.2) and
of earlier saved values (the agent's role). A count that depends on the run (lots, jobs,
shipments, closed accounts) is stored. 5.5 states the rule.

4.1.4. **Rows of diagnostic runs (except `BoundRun`, which is dense in each market row, 4.2.5) and of non-default scales are sparse**: only the rows
that differ from the default are written, in ascending key order, and a row equal to the
default is **corrupt** (the one-spelling rule of 5.2, R10).

### 4.2 The sections

| Tag | Section | Owner (and hosted fields) | State, in the order written | Present when |
|-----|---------|---------------------------|-----------------------------|--------------|
| 1 | World | WLD 12 | the next agent ID and whether the world is still in its construction phase. The tick and the effective seed are in the header (5.4); the construction bucket and the pending totals that calls between ticks added are the ledger's, in tag 2 | always |
| 2 | Ledger | LED 11, 7.4, 3; hosts the claims (CRD 17.1, INS 16.1) | every account that is not a boundary, in ID order: kind, closed flag and, if open, its region, owner reference, money, lots (commodity, quantity, `Born` signed, durability) in canonical order; the balance of each boundary account (a count the configuration implies); the **claims**: the next claim ID and the count of open claims, both always written (the count is 0 with neither the credit nor the insurance module on); then, when the ledger can hold a claim (`MaxClaims` above 0: the credit or the insurance module is on), the seven cumulative totals of `spec-credit.md` 3.3.2 (`Lent`, `Awarded`, `Repaid`, `InterestPaid`, `InterestFixed`, `LostPrincipal`, `LostInterest`) and the open claims in ascending ID, each as the ledger numbers its accounts: ID, creditor and debtor (account IDs), `Principal`, `Interest`, `PrincipalOwed`, `InterestOwed`, `RateBP`, `Formed`, `Due`, `Overdue` (signed, -1 for none) and `Origin` (`0` loan, `1` payout); the **pending totals** that calls between ticks added since the last P8.6: the flow totals as sparse rows (region, reason code, thing, total), a reason written as its pinned code and a thing as `0` for money or `1 +` the commodity index (4.2.7), the wear units as sparse rows (commodity, units), and the **pending relocation totals**, always written: a count and then the rows (`from`, `to`, thing, total) sorted by (`from`, `to`, thing), money first, the count being 0 until a relocation is recorded (`spec-regions.md` 6.4); and, when the world is in construction, the **construction bucket**, as flow rows and wear rows in the same layout (the pending totals are then empty: what a call moves in construction is in the bucket) | always |
| 3 | Agents | AGT 15; hosts `RoleSince` (SPN 14), `DeathCarry` (POP 3.8), `OwnUse` and `OwnUseJobs` (SUB 5.3), `Journey` (MIG 8.2), the stores (STO 15.1) and the membership (ORG 12.1) | the table of strategy names used by agents (sorted); for each agent in ascending ID: ID, account, role, weight, `External` and `Mortal`, strategy name index, `Unfed`, `Hunger`, `Privation`, `RoleSince`, `DeathCarry`, `OwnUse`, `OwnUseJobs`, the journey (destination, arrival tick) if any, the directive if any (whether it carries a job request, since `Directive.Job` is a pointer, then the job request if it does, the intents, discards, holds, `Sticky`); then, when `Modules.Storage` is on, the stores (a count, then (class index, count) pairs in ascending class index, a count of 0 being the common case); then, when `Modules.Institutions` is on, the membership as `BodyID + 1` and `RegionID + 1` (0 and 0 for none; one of the two zero is corrupt); then the diagnostic run of consecutive unfed ticks per (region, role), sparse | always |
| 4 | Markets | CLR 5.4, 5.5; BEL 7, 13; the report of INT 6.3 | for each (region, commodity) of a region that is not an outside node, region then commodity: the **market report of the last clearing**: the published price, the price age, the volume, the unmatched bid units and ask units, the best unmatched bid (present flag, price), the best unmatched ask (present flag, price); the price window (count, then the prices **oldest first**); and the diagnostic run `BoundRun` | always |
| 5 | Beliefs | BEL 13 | for each agent in ascending ID and each commodity of its role's tradable set in `CommodityID` order: `Centre`, `Width` | always |
| 6 | Production | PRO 15 | for each agent in ascending ID: whether it has a job and, if so, `Recipe`, `Remaining`, `Executions`, `Principal` (agent ID), `Yield` (basis points) and the pending quantity of each output in output order; its residues, one per (recipe of its role or of its effective own-use list, output) in canonical order; then the yield shocks that are not 10 000 as (region, recipe, factor), sparse | always |
| 7 | Welfare | WEL 13 | for each agent in ascending ID: the records held, then each record **oldest first** (`income_t`, `cost_t`, relief, grant, own-use receipt, own-use household-ticks, working flag, working household-ticks), then the target ratio drawn at creation; then the diagnostic runs per (region, role) of the two `Warn`s of WEL 11.3 (the work floor run and the clamp run), sparse | always |
| 8 | Estates | EST 2.7.5 | for each region in ascending `RegionID`: the current ask price of each lot the estate **holds** that has a row, by lot key (commodity, born, durability) in canonical order (held lots only; a held lot with no row is priced at the first price when next listed, and the `asked` mark is scratch and not saved) | always |
| 9 | Spawner | SPN 14 | the renewal carry per (region, role), sparse; the per (region, role) bit that the role had at least one agent (`Extinct`'s `everHad`, FR-SPN-042), sparse (only the set bits); the run of ticks with unfunded entrants per region, the run of empty ticks per (region, role) and the run of ticks of "group below floor" per (region, role), sparse | always |
| 10 | Skills | SKL 9 | the trainees (agent, trainer, target role, ticks served, application tick) and the trainers (agent, teaching accumulator, trainees) in ascending agent ID; the queue of applicants for each (region, role) that has one, in queue order, each entry (agent, application tick, ticks served) (sparse: an empty queue is not written); the `Warn` run, sparse. Written as four counts, in this order: the trainees, the trainers, the (region, role) rows of the queue and the rows of the `Warn` run, each followed by its rows; four counts of 0 when no role has `TrainingTicks` above 0 or nothing is held (4.2.8) | always |
| 11 | Regions | REG 11 | a count, then the edges whose status is not the default, in ascending (`From`, `To`): `Closed`, `FactorBP`; one count of 0 when no status differs (4.2.8) | always |
| 12 | Demand | DEM 12 | for each agent whose role has a ladder, in ascending ID: the carry of each taste row in ladder order; the diagnostic run per (region, role, want), sparse | `Modules.Demand` |
| 13 | MoneySupply | MON 14 | per flow in ID order: `Total`, `Scale`, `Ended`; for each flow with a household target and a carry shape, the (account, carry) pairs in ascending account ID, open accounts only; for each flow with a stores target and a carry shape, the same pairs, open accounts only (a third case, `spec-storage.md` 15.1); for each flow with a pool target and a carry shape, one carry per region | `Modules.MoneySupply` |
| 14 | Population | POP 11 | the birth carry per (region, role), sparse; the effective `BirthScale` and `DeathScale` per region when a shock set them, sparse; the three diagnostic runs (births unfunded, deaths held, and "region extinct" per region), sparse | `Modules.Population` |
| 15 | Resources | RES 15 | for each endowment with a configured capacity above 0, in (`RegionID`, `ResourceID`) order: `Stock`, `Carry`, `ExhaustedRun`; the shock scales that are not 10 000 as (region, resource, capacity, regeneration), sparse | `Modules.Resources` |
| 16 | Information | INF 14 | for each source (region, commodity) in ascending order: the entries held (a count), each oldest first as `Price`, `Mean`, `Age` | `Modules.Information` |
| 17 | Transport | TRN 14 | the next shipment ID; the shipments in flight in ID order, every field of `spec-transport.md` 5.2 (modes and regions by index, lines in commodity order; with `Modules.Risk`, `LossBP`, unsigned, 0 to 10 000, which the shipment record of TRN 5.2 carries); the transit factors that are not 10 000 as (from, to, mode, factor), sparse | `Modules.Transport` |
| 18 | External | EXT 16 | for each line in (node, commodity) order: the drift when it walks, the shock; for each present side, **buy side then sell side** (a side is present when its `Capacity` is above 0, `spec-external-sector.md` 3.3): the capacity carry, the cumulative units, the cumulative list value, the ended flag | `Modules.External` |
| 19 | Subsistence | SUB 15 | the two diagnostic runs per (region, role), sparse. The per-agent flag and counter are in section 3 | `Modules.Subsistence` |
| 20 | Migration | MIG 14 | the two diagnostic runs per region and the count of journeys ended by a removal per region, sparse. The journeys are in section 3 | `Modules.Migration` |
| 21 | Merchants | MER 16 | the next contract ID; the contracts in flight in ID order (ID, payer, deliverer, shipment, commodity, quantity, price, Escrow account, formation tick, destination region) | `Modules.Merchants` |
| 22 | Integration | INT 13, 4.7.3 | for each external agent of an `Informed` role that has had a view, in ascending agent ID: the tick of its last view and the rumours given to it (region, commodity, price, mean, lag, source age), in the order of the view, sparse | `Modules.Information` |
| 23 | Settlements | SET 11 | for each region that is not an outside node, in ascending `RegionID`: `Class` (an index into the sorted classes) and `Since` (the tick of the last change, 0 at construction). Dense; no diagnostic run, no hosted field | `Modules.Settlements` |
| 24 | Employment | EMP 16 | the next hire ID; the hires in flight in ascending hire ID: ID, employer and worker (agent IDs), the wage per household-tick, the Escrow account, the formation tick, the region (by index); then the two diagnostic runs of the `Warn` lines of `spec-employment.md` 15.2.2 per (region, role), sparse. The recipe, executions and ticks left of a hire are the worker's job in section 6 | `Modules.Employment` |
| 25 | Tenure | LND 14 | for each endowment with at least one title, in (`RegionID`, `ResourceID`) order: the number of titles and then each as (holder agent ID, `ShareBP`) in ascending holder ID; for each endowment with a posted rent: the posted rent; the rent scales that are not 10 000 as (region, resource, `BP`), sparse; the two diagnostic runs `NoTitleRun` and `RentFundsRun`, sparse | `Modules.Tenure` |
| 26 | Credit | CRD 17.1 | the rate scales that are not 10 000 as (region, `BP`) in ascending region, sparse; the diagnostic run of the "claims overdue" `Warn` per region, sparse; the run of "claim cap reached", one value. The claims are in section 2 | `Modules.Credit` |
| 27 | Facilities | CAP 15.1 | the next facility ID; the count; for each facility in ascending ID: `ID`, class index (the position in the configuration's sorted classes), `Region`, `To` (route classes only), `Owner` (an agent ID or the estate of `Region`), `Condition`, `Built`, `Installed` (a store class only) | `Modules.Infrastructure` |
| 28 | Risk | RSK 15.1 | the route scales that are not 10 000 as (`From`, `To`, mode, `BP`) in ascending key, then the store scales that are not 10 000 as (region, class, `BP`) in ascending key; both sparse. `LossBP` of a shipment is in section 17 | `Modules.Risk` |
| 29 | Insurance | INS 16.1 | the next policy ID; the open **route** policies in ascending ID: ID, insured and underwriter (agent IDs), shipment ID, `Sum`, `MaxPayout`, `CoverBP`, `Premium`, `Formed`; the six totals of `spec-insurance.md` 9.3. A store policy is never saved: it is formed at P3.3c and ended at P8.1b of one tick. The payout claims, `Origin` and `Awarded` are in section 2 | `Modules.Insurance` |
| 30 | Fiscal | FIS 17 | for each flow in the order of the six lists (taxes, duties, tolls, payouts, purchases, releases, each by ID): for a tax, a duty or a payout with a rate shape the (account ID, carry) pairs in ascending account ID for every open account on which the flow has acted, zeros included; for a purchase or a release one carry, always written; then the scales that are not 10 000 as (flow position, scale), sparse. The treasury's money and goods are the ledger's | `Modules.Fiscal` |
| 31 | Institutions | ORG 12.2 | the levels that differ from the configured ones as (`ControlID`, `CapBP`, `FloorBP`) in ascending `ControlID`; the suspended rights as `RightID` in ascending order; the run of consecutive ticks with waiting households per (region, body) that has one, sparse. Zero rows when nothing differs. The membership is in section 3 | `Modules.Institutions` |
| 32 | Policy | PLC 13.1 | for each `directed` actor with a non-empty mailbox, in ascending `ActorID`: the actor, the count of settings, then each setting in ascending `LeverID` as the lever, the value and the closed flag. Sparse | `Modules.Policy` |
| 33 | Technology | TEC 12.1 | for each (region, field) cell in ascending (`RegionID`, `TechFieldID`): `Experience`, `Advance`, `CatchUp` (the count is `R * F` from the configuration and is not written); for each field in ascending order: `TrendCarry`; then `Key` (unsigned, 64 bits), **present only when some field has `DiscoveryBP` above 0** | `Modules.Technology` |
| 34 | Extensions | CHK 7 | for each registered `Stateful` extension in (kind, name) order: kind, name, version, state bytes | always (a count; the built-in **strategy** `directed`, which `spec-integration.md` 4.2.2 builds in as `Directed` over the default, has an entry in every world, empty unless its fallback is `Stateful`; the built-in policy `directed` has none, 7.5) |

4.2.1. **Order.** The tag order is the write order and the restore order. Sections 12
to 33 are in the order of the dependencies between modules (a shipment of 17 before the
contracts of 21 that name it, the claims of 2 and the Escrow accounts of 2 before the hires
of 24, the endowments of 15 before the titles of 25, the stores of 3 and the shipments of 17
before the facilities of 27 and the policies of 29). Tag 22 was assigned before the later-stage
modules and a tag is never renumbered (5.3.3), so Integration stays 22, the sections 23 to 33
follow it in dependency order, and Extensions, which loads after every other section
(FR-CHK-029), is 34. A module with no section is listed in 4.2.6. A section of a module that is off is **absent**: it has
no tag in the frame, the decoder derives the expected list of tags from the
configuration, and an unexpected tag or a missing one is corrupt (5.3). The sections
that depend on no module (1 to 11 and 34) are always present, with zero rows when there
is nothing to save, so that the shape of a checkpoint does not depend on a run's
history.

4.2.2. **Pseudo-agents are not agents.** The estate pseudo-agents, the counterparties
of the outside and the treasury pseudo-agents of the governed jurisdictions have no row in
section 3. Their IDs and accounts are a function of the
configuration (`spec-world.md` 5.2, 5.3): the ledger saves their accounts like any
other, and a restore rebuilds the pseudo-agent of region `r` and of each counterparty
from the configuration and the account list. The **owner reference** of an account is the
agent ID for an Agent account, the `RegionID` for an Estate account (equal to its
pseudo-agent's ID), the index of the counterparty in the configuration's order (node,
gate, buy before sell) for an Outside account, the `JurisdictionID` for a Treasury account (a
restore rebuilds the treasury pseudo-agent of each governed jurisdiction from the
configuration and the account list, as it does an estate), and absent for an Escrow or a Transit
account, whose region is saved and whose user is named by the shipment, the contract or the hire
that holds it (an account that no record names is free, 4.3).

4.2.3. **Agent references.** Every other section names an agent by its **ID**. The
decoder checks that the ID is that of a live agent of section 3, **except** for the
references that name an agent that may since have been removed: a shipment's `Shipper`,
`Traveller` and `Consignee` (a shipment outlives all three, `spec-transport.md` 5.9, 5.2),
and a contract's payer (`spec-merchants.md` 9.7); for those the ID is only required to be
less than the next agent ID (an ID is never reused). Two references may also name the **estate
pseudo-agent** of a region, whose ID is the `RegionID` and which has no row in section 3: a
facility's `Owner` (`spec-infrastructure.md` 15.3) and a title's holder
(`spec-land.md` 14.2); each is checked against the region instead (the facility's region,
the endowment's region). A reference to a region, a role, a recipe, a commodity, a
mode or a resource is an index into the configuration's canonical order, checked
against its count.

4.2.4. **Enumerations of the format.** An account kind is `0` Agent, `1` Estate, `2`
Escrow, `3` Transit, `4` Outside, `5` Treasury, in the order of `spec-ledger.md` 3.1 (a boundary has no
entry in the account list: its balance is written in the boundary list); an extension
kind is `0` strategy, `1` spawner, `2` clearing, `3` policy; the origin of a claim is `0` loan, `1` payout; an agent flag word has bit `1` External
and bit `2` Mortal; the ended state of a money flow is `0` not ended, `1` limit, `2` cap
(`spec-money-supply.md` 14). A value outside an enumeration is corrupt (R6).

4.2.5. **Dense or sparse, signed or not.** The layout of a row is this spec's choice; the
owners say "per (region, role)" and not how many rows are written. **Dense** (every
element of the implied or stored count, no omission): the accounts, the lots, the agents,
the beliefs, the markets (including `BoundRun` in each market row), the welfare records of
an agent, the ring entries, the contracts, the shipments, the flow rows, the endowments and
the lines, the hires, the claims, the facilities, the policies, the settlement rows (23) and
the technology cells (33). The (account, carry) pairs of a fiscal flow (30) are dense over the
accounts the flow has acted on, zeros included, which is not the sparse rule of "a row at its
default is corrupt": an account with no pair has the flow's initial value and a pair at 0 is
not that value (`spec-fiscal.md` 17). **Sparse** (only rows that differ from the default, ascending by key; a row at
its default is corrupt, R10): every diagnostic run (tags 3, 7, 9, 10, 12, 14, 19, 20, 24, 25, 26, 31), the
renewal carry and birth carry (default 0), the had-an-agent bit (default unset), the count of journeys ended by a removal
(default 0), the yield shocks (default 10 000), the transit factors (default 10 000), the
resource shock scales (default 10 000), the rent scales (25), the rate scales (26), the route
and store risk scales (28), the scales of the fiscal flows (30), the mailboxes of tag 32, the
pending totals (default 0), the queues of tag 10, the legal levels and suspended rights of tag
31 (a level equal to the configured one is corrupt), the effective `BirthScale` and `DeathScale` (default: **the region's configured
value**, which is 10 000 with no row, `spec-integration.md` 5.1) and the rumours of tag 22.
**Signed (I):** `RoleSince`, `Born`, money and boundary balances, the welfare `income_t`,
`cost_t`, relief, grant and own-use receipt, a trainee's application tick, the drift,
the tick of a last view and a claim's `Overdue` (-1 for none). Every other
integer is unsigned (U). A value of the wrong sign for its owner's range is corrupt (R13).

4.2.6. **Modules with no section of their own.** POL adds none: its configuration is in the hash,
its treasury is an account of section 2 (kind `5`) and a pseudo-agent rebuilt from the
configuration and the account list (4.2.2), and nothing of it is state. STO has none: its stores
are hosted in section 3 and its flows' carries are a third case of section 13 (N-128). SAV has
none: the target, the goal, the test and the request are functions of the beliefs, the welfare
ring, the facilities, the money, the configuration and the agent ID
(`spec-saving-and-investment.md` 13). ORG hosts the membership in section 3 beside its own
section 31. A field hosted in section 3 is present only with its module (4.2.1).

4.2.7. **A reason is saved by a pinned code, a thing by `0` or `1 + c`.** The ledger numbers
its reasons itself and the numbers change with the modules that are on (`spec-ledger.md`
7.2.2, N-225), so a checkpoint never holds one. The code of a reason of the table of
`spec-ledger.md` 7.2 is its position in that table, counted from 0, as the table stands:
`trade` 0, `initial` 1, `recipe_input` 2, `recipe_output` 3, `disposal` 4, `consumption` 5,
`want_consumption` 6, `spoilage` 7, `wear` 8, `estate_money` 9, `estate_goods` 10,
`start_money` 11, `relief` 12, the six `outside_*` 13 to 18 in the order of the table,
`escrow_in` 19, `escrow_out` 20, `escrow_refund` 21, `wage_escrow` 22, `wage_pay` 23,
`wage_refund` 24, `loan_out` 25, `loan_repay` 26, `loan_interest` 27, `rent` 28,
`title_sale` 29, `premium` 30, `payout` 31, `transit_out` 32, `transit_in` 33,
`haul_use` 34, `transit_use` 35, `loss` 36, `split` 37, `merge` 38, `births` 39,
`starvation` 40 and `deaths` 41. The table has 42 named rows (`pinnedRows`), including the
rows no module has built yet. The codes are part of format 1: a row added to the table
later takes the next number, and a code is never moved or reused, as a tag is not (5.3.3).
The code of a configured money flow or fiscal flow is 42 plus its position in the ID order
of the flows the configuration declares (the two kinds share one ID namespace,
`spec-money-supply.md` 4.5.2), so adding a reason to the library never changes saved
bytes. A code at or above 42 plus the number of flows, or the code of a reason whose module
is off, is corrupt (R6). A **thing** is `0` for money and `1 + c` for the commodity index
`c`, checked against the commodity count (R6).

4.2.8. **A section of a module that is not built yet is a zero-row skeleton.** Tags 10
and 11 are always present (4.2.1) and their owners (SKL, REG) are built after the
checkpoint. Until then each is written as its counts at 0: four counts for tag 10 and one
for tag 11. The owner replaces the writer and the reader and keeps the skeleton as a
**prefix** of the layout it adds, so a world with no row has the same bytes before and
after the owner is built and the digest of a checkpoint does not move.

### 4.3 Derived and rebuilt on restore

Everything below is a function of the saved state and the configuration and is
**rebuilt**, never saved. A test (FR-CHK-045) lists, from the code, every field of a
live type that is neither saved nor named here, and fails on an unnamed one, as the
parameter registry test does for configuration fields.

| Spec | Derived (rebuilt at restore, or recomputed each tick) |
|------|--------------------------------------------------------|
| CAL | the moment of a tick, `TicksPerYear`, the day-of-year table (`spec-calendar.md` 4 and 12) |
| REG | `RegionID` and `EdgeID`, the distance, next-hop and component tables, the adjacency lists, the resolved override arrays, the lists of the agents of a region |
| LED | the earliest-expiry hint of each account (recomputed from the lots), the free lists of the Transit and Escrow accounts (an account no shipment or contract names), the derived account index of each pseudo-agent |
| AGT | the effective upkeep, the tradable sets, targets, budgets, views, plans, the planned recipe and the work share |
| BEL | `Floor`, `Ceiling` and the effective range, the stock ratio, the window's mean, low and high |
| PRO | the yield table, the cost anchors of 9.3 (recomputed each tick from the published prices and the regional wage), the season, the rotation offset |
| WEL | the running sums of a ring (recomputed from the records), the effective window, the ratio and the work share, **the regional observed wage and the starting wage** (4.5, F5) |
| SPN | the signals `E` and `V`, the budget, the view and the plan |
| POP | the exposed households, `PopBirths` and `PopDeaths`, `r*` and the gain |
| DEM | the price ratios, the schedule values, the shares, the budget, the planned want units |
| MON | the due of a tick, the per-tick aggregates, `Applied` of the last tick, the carry of a closed account |
| TRN | the away flags, the arrival schedule, the travel times, the routes, `Inbound` and `Outbound` |
| RES | the effective capacity and scale, the abundance snapshot, the grants, the ration key, the dense index arrays |
| INF | the lag table, the sources, the candidates and the rumours of a view (scratch), the ring's write position. **Not derived:** the rumours of the last view of an external agent, which INT keeps and section 22 saves |
| EXT | the mid and the quotes, the gates, the counterparty list |
| SUB | the effective own-use lists, the covered sets, `Cover`, `Cost`, `Afford`, `Secure`, the ramp |
| MIG | the households by region, the households on a journey to a region, the caps of a tick, the candidates, the estimates and the rankings |
| MER | the fee share and the spread (a function of the agent ID), the trip, the lot, the ceilings, the plan notes |
| STO | the room of an account, its load, its pledge (from the jobs of section 6 and the shipments of section 17), the reserved room (0 at a boundary), the stored load, the table of sizes and the free room (`spec-storage.md` 15.2) |
| EMP | the free list of Escrow accounts (an account no hire, shipment or contract names), the committed executions, the reservation and the bid, the positions, `Employed` and `Employing` (functions of the hires and the jobs) |
| SET | the effective thresholds per region, the flag and limit tables, the household counter and the headroom (`spec-settlements.md` 11) |
| LND | the entitlements, the tiers, the supply and the attribution of a tick, the rent in force and its key, the commons share (10 000 less the titles) and the scratch of the reservation (`spec-land.md` 14.1.1) |
| POL | the `Polity` tables (parent, depth, chains, children, covered lists), the resolved arrays of readers, the treasury pseudo-agent IDs and their account index |
| CRD | the due index, the overdue list, the by-account indexes, `Debt`, `Owed`, `Overdue`, `Lent`, `Receivable` and `CreditRoom` of an agent, the open count (`spec-credit.md` 17.3) |
| CAP | the effect tables, `eff` and `cap`, the per-owner counts, the per-tick counters, `used(principal, class)` and `served(f)`, the order of use wear (`spec-infrastructure.md` 15.2) |
| RSK | the common outcomes of P3.1a and P8.1a, the table of expected losses, the dense arrays of rows and the effective probabilities (`spec-risk.md` 15.2) |
| INS | the by-shipment index, the by-agent indexes, `Open` and `Capacity` of an underwriter, the open count and the expected-loss table (`spec-insurance.md` 16.3) |
| FIS | the dues, the lists of trades of the tick, the cached tolls, the per-agent tax and payout sums and the carry of a closed account (`spec-fiscal.md` 17) |
| ORG | the resolved rights, the legal ranges in minor units, the scope regions of a body, the headroom and the scratch counts of the tick (`spec-institutions.md` 12.4) |
| SAV | nothing: the target, the goal, the test and the request are functions of state that other sections hold (4.2.6) |
| TEC | the gain of a cell, the contact weights and the per-tick counters (`spec-technology.md` 12.1) |
| PLC | the due ticks and buckets, the expanded key lists, the views and the gauges (`spec-policy.md` 13.3) |
| VIA | nothing: the warm start is configuration and its effect is ordinary state after tick 0 (`spec-viability.md` 18) |
| WLD | the rotation positions and counts, the failed flag (a failed world is not checkpointed), the in-step guard, the module scratch cleared at P8.6 |
| CFG | the effective configuration, rebuilt from the `Config` the consumer passes and checked by its hash |

### 4.4 Deliberately not saved

| What | Why | Where it is settled |
|------|-----|---------------------|
| Reservations and earmarks | none exists between ticks, so `Reserved` is 0 in every saved lot | `spec-ledger.md` 6.5, 11 |
| The flow totals of finished ticks | they are cleared at P8.6; `BoundaryFlow` already delivered them | `spec-world.md` 6.10, 8.1.3 |
| **The pending totals are saved**, the one exception | a call between ticks (an external agent's `initial`, a relocation) is totalled in the tick the next step runs, so at a boundary the totals are state | `spec-world.md` 8.1.3, 12.1, open question 16; section 4.2, tag 2 |
| The construction bucket, **once the world has stepped** | a world that has stepped has its total in its boundary balances, and the audit's baseline is the balances | `spec-ledger.md` 7.4.1, 11 |
| **The construction bucket while the world is in construction**, the other exception | a world checkpointed before its first `Step` is restored still in construction, with its bucket, so a later call before the first step is totalled as it would have been on the original (6.7, WLD edit in 20) | this spec |
| Pending output of jobs | it is saved, with the job, by PRO; it is in no account | `spec-production.md` 15 |
| The configuration, the options, the observers, the logger | 3.5 | ADR 0002, decision 3 |
| Events | 3.5.3 | ADR 0002 open point 5 |
| Failed flag | a failed world is refused, so no checkpoint carries one | `spec-world.md` 3.5, 12.1 |

### 4.5 What the specs list twice, or not at all

Each row is a finding of reading the specs against each other. The ruling is the rule
of this spec; the edit it needs is in section 20.

| # | Finding | Specs | Ruling |
|---|---------|-------|--------|
| F1 | **State read by a position and listed nowhere.** `spec-clearing.md` 7.1 says the market carries only the published price, the price age and the price window. `Trade` is given, per commodity, the market report of the last clearing: the price, the age and the **best unmatched bid and ask with flags** (`spec-agents.md` 6.4.1, `spec-clearing.md` 5.5), read at P3.2 of the next tick. They are boundary state. So are the volume and the unmatched units of the last clearing, which INT's query `Market(r, c)` and the snapshot report (`spec-integration.md` 6.3, `spec-observability.md` 5.3.5) and which a restored world must answer as the original did | CLR 7.1, 5.5; AGT 6.4.1; INT 6.3; OBS 5.3.5 | Saved in section 4 with the price and age: the whole report of the last clearing. CLR gains a state section (20) |
| F2 | The price window is saved with "the position of the next write" (`spec-beliefs.md` 13): a physical detail, so two equal windows could differ in bytes | BEL 13 | The ring is saved **oldest first** with a count and no position; the position is derived. INF 14 and WEL 13 already say so |
| F3 | **Per-agent fields listed by two specs.** `Privation` (AGT 15, SPN 14), `DeathCarry` (AGT 15, POP 11), `OwnUse` and `OwnUseJobs` (AGT 15, SUB 15) and the journey (AGT 15, MIG 14) are each "per agent" in two sections | AGT, SPN, POP, SUB, MIG | Each is saved **once**, in section 3, because the agent hosts it. The module's section lists its own state only (the runs, the carries per (region, role)). `RoleSince` is SPN's and is hosted in the same agent record |
| F4 | `spec-agents.md` 3.2 and open question 9 say the welfare ring "has no checkpoint section yet" | AGT | Stale: WEL 13 lists it. Section 7 of this spec is that list |
| F5 | **A latch would be state.** `spec-production.md` 9.3 and `spec-viability.md` 8.2.5 say the starting wage is "used until the first working household-tick is observed". Read as a latch it is a per-region bit that is neither in a ring nor in a section | PRO 9.3, VIA 8.2.5, WEL 4.3 | **Ruled as a pure function of the rings:** the starting wage is used whenever the pooled figure is unknown (no working household-tick in any window of the region, or a non-positive pooled figure). Nothing is saved. If the intended meaning is a latch, WEL or PRO adds one bit per region to its section and this table changes by one row |
| F6 | Closed accounts and "compaction at a checkpoint" | LED 3.3, open question 6 | Never compacted in version 1 (9.4) |
| F7 | A trainee's trainer and a trainer's trainees are both listed (`spec-skill-barriers.md` 9) | SKL | Both are saved and cross-checked at restore (6.2 step 7): an edge in one and not the other is corrupt |
| F8 | The directive and the strategy name of an **external agent** are not in the configuration, so the hash does not cover them | AGT 10, 6.6; INT | The name table of section 3 carries them, and every name must be registered at restore (6.2 step 4) |
| F9 | The pending totals of calls between ticks (WLD open question 16) | WLD 8.1.3, 12.1; LED 7.4, 11 | Saved (4.4), and the audit's baseline after a restore is the restored balance less the pending totals (6.2 step 10) |
| F10 | `spec-world.md` 5.5.3 and `spec-ledger.md` 11: "a restored world is never in construction; the bucket is not saved" | WLD 5.5.3, 12.2; LED 7.4.1, 11 | True for a checkpoint taken after the first `Step`. For one taken before it, the world is restored in construction with its bucket (4.4). Edit owed to both |
| F11 | `spec-world.md` 12.3 recommends that a restore reject `WithSeed` | WLD 12.3 | Adopted (6.4), with `WithReseed` as the restore-only way to a different continuation (6.8) |

## 5. The format

### 5.1 The frame

One checkpoint is one byte string:

```
magic         4 bytes   45 53 43 4B  ("ESCK")
format        U         the format version (this spec defines format 1)
hash version  U         the version of the configuration encoding that produced the hash (spec-config.md 8.2)
config hash   32 bytes  SHA-256 of the effective configuration
tick          U         the tick the next Step would run (Tick())
seed          U         the effective seed: the seed after a reseed, read back by a restore (6.8.3)
sections      ...       for each present section in tag order: tag U, length U, body
trailer       32 bytes  SHA-256 of every byte before it
```

`U` and the other primitives are those of 5.2. There is nothing before the magic and
nothing after the trailer.

### 5.2 Primitives and the canonical rules

Every number in a checkpoint is a **varint**: byte order does not exist, so the
encoding is the same on every platform (architecture.md 7.4). The rules are numbered
because the decoder, the fuzz test and the golden digest all depend on them.

| Rule | Primitive or rule |
|------|-------------------|
| R1 | **U**: an unsigned integer as LEB128, least significant seven bits first, the high bit of a byte set when another byte follows. At most 10 bytes, the tenth at most 1. **Minimal**: the last byte of a value of more than one byte is not 0. `U(0)` is `00`, `U(128)` is `80 01`, `U(300)` is `ac 02` |
| R2 | **I**: a signed integer as zigzag, `(v << 1) xor (v >> 63)`, then U. `I(0)` is `00`, `I(-1)` is `01`, `I(1)` is `02`, `I(10000)` is `a0 9c 01`; the whole `int64` range is representable |
| R3 | **B**: one byte, `00` or `01`; any other value is corrupt |
| R4 | **Raw**: a byte array whose length the format fixes (the 32 bytes of a hash) |
| R5 | **Bytes**: U length, then that many bytes; the length is at most the bytes remaining. Used for the names and the state blobs of extensions |
| R6 | **Index or enum**: U, checked against its bound (the count of the thing it indexes, or the values the format defines); anything else is corrupt |
| R7 | **Optional**: B, then the body if it is 1 |
| R8 | **Sequence**: a count U (when 4.1.3 does not imply it), then the elements. The count is at most the bytes remaining (every element takes at least one byte), so no count can ask for an allocation larger than the input, and at most the bound of its type. A count the configuration implies is checked the same way before its slice is allocated: at least one byte per element must remain |
| R9 | **Order**: each sequence has the order 4.2 states. A sequence keyed by an ID is **strictly ascending**: a duplicate or a descent is corrupt. A positional sequence (the beliefs of an agent in `CommodityID` order) has the length the configuration implies |
| R10 | **One spelling**: a sparse row (4.1.4) equal to its default is corrupt; a count that 4.1.3 implies is never written; a field of a module that is off is not written (the module's section is absent), **except** the fields hosted in the agent record and in a welfare record, which are written always. Of these, `OwnUse`, `OwnUseJobs` and the welfare own-use receipt and own-use household-tick fields must be 0 when `Modules.Subsistence` is off, and the journey must be absent when `Modules.Migration` is off (a non-zero value then is corrupt); `DeathCarry` (`offset(ID)` at creation, `spec-population.md` FR-POP-010) and `RoleSince` (the creation tick, `spec-spawner.md` 7.4) are **always** written with their owners' values whatever modules are on, and `Privation`, `Unfed` and `Hunger` are core fields with no module condition |
| R11 | **No other bytes**: no padding, no alignment, no reserved field, no trailing byte inside a section or after the trailer |
| R12 | **No maps, floats, pointers, functions, timestamps, host or build identity** (architecture.md 3.5, 12.4; observability-and-logging.md 4.1) |
| R13 | **Value bounds**: every value is within the range its owning spec gives (a non-boundary balance in `[0, MaxMoney]`, a quantity in `[1, MaxQuantity]`, a basis-point factor within its bound, a tick that is at least 0 where the spec says so). A value outside is corrupt |

5.2.1. **Why the minimal-form rule is stated.** `binary.Uvarint` of the standard
library accepts a non-minimal encoding (`80 00` decodes to 0 with two bytes read, `81
80 00` to 1; checked with Go 1.25) and reports only overflow. A decoder built on it
alone would accept two spellings of one state, and "accepted input re-encodes to the
same bytes" (FR-CHK-005) would fail. The decoder reads a varint with the check of R1.

5.2.2. **Why varints.** Fixed 8-byte integers were the other candidate (the
configuration hash uses them, `spec-config.md` 8.2). The model of section 16 gives, for
the first consumer's world, 2.1 MB as varints against 13.6 MB as fixed integers (6.4
times), because most saved integers are small (a count, an index, a quantity, a flag,
a welfare record that is zero). Fixed width makes the decoder a little simpler and
needs no minimality check; a checkpoint 6 times larger is paid on every save, copy and
restore, so the varint is chosen, and its one extra rule (R1) is a line of code and
one fuzz property.

### 5.3 Sections

5.3.1. A section is `tag U`, `length U`, `body`. The **expected list of tags** is a
function of the configuration: the tags of 4.2 in ascending order, those of the
modules that are on, and always tags 1 to 11 and 34. The decoder reads the next tag
and requires it to be the next expected one; reads the length and requires it to be at
most the bytes remaining; decodes the body from exactly those bytes and requires that
every byte is consumed. A section that ends early or runs over is corrupt, and the
error names the tag (6.3).

5.3.2. **The length is a check, not a skip.** A restore never skips a section: format 1
has no optional section and no unknown tag. The length lets a decoder fail at the
section that is wrong and lets a human find it. A future format that wants forward
compatibility changes the format version (9.1). Section 22 is therefore mandatory when
`Modules.Information` is on: it is saved so that `Agent` answers as the original did
(FR-CHK-052), as INT 13.2 says.

5.3.3. A tag is assigned once and never reused. A new section takes the next number and,
once a format has been released, a new format version (9.3); a section that is dropped leaves
its number unused.

### 5.4 The header and the trailer

5.4.1. **The header** is what a consumer needs before it has chosen a configuration
(`Info`, 3.1): the format, the version of the configuration encoding, the hash, the
tick and the seed. The seed is the **effective seed** (`spec-world.md` 12.3, ADR 0008): it
is the one random datum of the world, a restore reads it back and every stream is a
function of it, and `WithSeed` is refused (6.4) because the header already holds the
seed. A reseed (6.8) rewrites it. A restored world reports its effective seed, as the
snapshot does (`spec-world.md` 16.5).

5.4.2. **The trailer** is the SHA-256 of every byte before it. It detects truncation,
bit flips and a partial write with certainty for practical purposes, and it is the
digest the golden file of `spec-reference-chain.md` 8.5 item 6 compares. It is **not an
authenticator**: anyone can recompute it, so a deliberately altered checkpoint carries a
valid one. What stops an altered state from damaging a world is the validation of 6.2
(structure, references, the ledger's identities); that cannot make an implausible
economy plausible. A consumer that needs authenticity signs the bytes itself.

5.4.3. **The tick is in the header**, not in section 1, so that `Info` needs no section.
`FR-WLD-100` lists the tick among the world's contributions; where it is written is
this spec's choice.

### 5.5 Counts, order and references

5.5.1. **Counts that the configuration implies** (not stored): the boundary accounts
(5.2 of `spec-world.md`), the markets (every region that is not an outside node times
every commodity), the beliefs of an agent (its role's tradable set), the residues of an
agent (its role's recipes and their outputs), the pending quantities of a job (its
recipe's outputs), the carries of an agent (its role's taste rows), the flows of the
money-supply module, the endowments of the resources module, the sources of the
information module, the lines and sides of the external module. (The strategy names of agents are **not** implied: an external agent's name is not in the configuration; they are the table of section 3.)

5.5.2. **Counts that the run implies** (stored): the non-boundary accounts and the lots
of each, the agents, the welfare records held by an agent (at most its effective
window), the shipments and contracts in flight, the queue of a role, the entries of a
price ring, the rows of a sparse list, the extension entries.

5.5.3. **References** are by ID for agents and accounts (which are never reused) and by
index into the canonical order of the configuration for everything else
(`spec-config.md` 3.3, 3.4). A reference is checked when it is read where the thing it
names already exists, and in the cross-check of 6.2 step 7 where it does not (a
shipment names a Transit account that section 2 has already defined, so it is checked
at read time; a trainee names a trainer that section 3 defined, likewise).

### 5.6 What the encoding is not

It is not self-describing: a decoder needs the configuration to know the counts. It is
not forward compatible: a library never reads a format newer than its own. It is not
streamed: the digest is over the whole, so a restore needs all the bytes. These are the
costs of the canonical form and of strictness (ADR 0002, decision 8), and each is
accepted.

### 5.7 Alternatives considered

| Alternative | Why not |
|-------------|---------|
| `encoding/gob` | not canonical (type descriptors, map order), reflective, and a stable form across versions is not promised (ADR 0002, alternatives) |
| JSON with tags | tags are banned in the core (architecture.md 12.6); floats and map order; large |
| Protocol buffers | a third-party dependency (architecture.md 5.1) and a code generator |
| Fixed-width integers | simpler, 6.4 times larger (5.2.2) |
| A tag-length-value body for every field | would allow unknown fields to be skipped, which format 1 refuses (5.3.2), at a byte or two per field |
| Compression inside the format | a second algorithm to keep stable; the consumer compresses the bytes if it wants (10.3) |
| One section per agent (agent-major) | each spec would need a central writer to call its piece per agent, and a section could not be owned; the spec-major order matches "every spec lists the state it contributes" |

## 6. Restore

### 6.1 The call

```go
func RestoreWorld(cfg Config, c Checkpoint, opts ...Option) (*World, error)
```

6.1.1. A restore is **a constructor**, like `NewWorld` (`spec-world.md` 3.2): it applies
the options in order, validates the configuration in full, and returns a world or an
error and no world; it never returns a partly built one (FR-CHK-017). The world it
returns is **new**: it shares nothing with the world that was checkpointed, which may
still be running.

6.1.2. A restored world is **not in construction** when the checkpoint was taken after
the first `Step`; it **is** in construction, with its bucket, when it was taken before
(4.4, 6.7). Either way it emits no `WorldCreated` and no `AgentSpawned`: its event is
`WorldRestored` (6.7).

### 6.2 The order of validation

Cheap checks come first and nothing is allocated in proportion to the input before the
frame has been verified (6.6). **Step 1 is done by `UnmarshalBinary`**, so a
`Checkpoint` value always has a verified frame (a value made by `World.Checkpoint` is
trusted); `RestoreWorld` begins at step 2.

| # | Step | On failure |
|---|------|-----------|
| 1 | **Frame** (`UnmarshalBinary`): length at least the header and trailer; the magic; the **format** is one this library reads (`ErrCheckpointVersion` otherwise, checked before the digest so that a newer format is not reported as damage); the trailer equals the SHA-256 of the bytes before it; the header fields are in range (a hash version this library does not know is a header range, `ErrCheckpointCorrupt`, and not a version error: a future encoder of the hash is a change of the format, 9.3) | `ErrCheckpointVersion`, `ErrCheckpointCorrupt` |
| 2 | **Options**: applied in the order given, each validated; `WithSeed` is `ErrOptionContext`; a nil extension is `ErrNilExtension`; a name registered twice is `ErrDuplicateID` | those |
| 3 | **Configuration**: deep copy, `WithTargetSpread` applied, canonical order, `Validate` in full, the name checks of `spec-world.md` 3.2.2; warnings collected. Then the effective configuration is hashed **with the encoding version in the header** and compared with the header's hash | the validation errors; `ErrCheckpointConfig` |
| 4 | **Extensions**: the set of registered extensions that implement `Stateful` equals the set of entries of section 34 by (kind, name); each strategy name in the table of section 3 is registered | `ErrCheckpointExtension` (joined with `ErrUnknownStrategy` for an unregistered strategy) |
| 5 | **Derive** what is not state from the configuration: the calendar, the region graph tables, the lag table, the module tables, the weights of the groups (`spec-world.md` 5.1 step 5) | none: a validated configuration cannot fail here (programmer error, panics) |
| 6 | **Sections**: in the order of the expected tag list, each decoded from exactly its bytes into new structures, with R1 to R13 checked as it is read and every reference to an earlier section checked | `ErrCheckpointCorrupt` naming the tag and the offset |
| 7 | **Cross-checks** that need two sections: every open Agent account is the account of exactly one agent and of the agent that names it; every (region, role) that has a live agent has its had-an-agent bit of section 9 set; every estate listing is a lot the estate holds; every Transit account of a shipment and Escrow account of a contract exists and holds what the record says; every trainee has its trainer and the converse; every money-flow carry is for an open account; every journey names a live agent and every other reference to an agent is live or one of the removable references of 4.2.3 (below the next agent ID); every Escrow account of a hire exists and holds what the record says; every claim's creditor and debtor are open accounts of the kinds of `spec-credit.md` 3.6; every open policy names a shipment in flight whose shipper is its insured; the account list holds exactly one Treasury account for each governed jurisdiction, in ascending `JurisdictionID` and in the seat's region (`spec-polity.md`, FR-POL-030), and a facility's owner is a live agent or the estate of its region | `ErrCheckpointCorrupt` |
| 8 | **Install**: no random state is installed and no word is taken (the seed is that of the header, or the reseeded one at step 9): the next agent ID, the construction flag, the tick; then the derived tables that need the state (the expiry hints, the tradable sets of the agents' roles, the Transit and Escrow free lists) | `ErrCheckpointCorrupt` |
| 9 | **Reseed**, if `WithReseed` was given (6.8) | none |
| 10 | **Audit**: the ledger's audit and every check of a module that is on, as `World.Audit` runs them, with the baseline of the boundary identity taken as the restored balance **less the pending totals** and, for a world restored in construction, with the saved bucket as the identity's construction term (`spec-ledger.md` 9.1). The audit runs whether or not `WithAudit` was given; `WithAudit` governs `NewWorld` and every `Step`, and this restore audit is a ruling of this spec that `spec-world.md` 3.3.3 and 12.2 are edited to state (20) | `ErrCheckpointCorrupt` wrapping `ErrAuditFailed` |
| 11 | **Extension state**: `LoadState` of each `Stateful` extension in (kind, name) order with the version and the bytes of its entry | `ErrCheckpointExtension` wrapping the extension's error |
| 12 | **Report**: the configuration's warnings at `Warn` (once each, as at construction), `Info` "world restored", `WorldRestored` | none |

6.2.1. **Why the extensions' `LoadState` is last.** Steps 2 to 10 touch nothing the
consumer owns. `LoadState` replaces the state of an instance the consumer registered,
so it runs only when everything else has succeeded; a failure at step 11 is the one
failure that leaves an instance modified, and 7.3 says what the consumer must then
assume.

6.2.2. **Why the ledger's audit always runs.** It costs time proportional to accounts
and lots, once, and it is the only check that the bytes describe a state that
conserves money and goods. A restore that skipped it would let a checkpoint that was
altered (and re-digested) break conservation silently. `WithAudit` does not change
this; it governs every `Step`, not the restore.

6.2.3. **What is not validated.** That the state is one a run could have produced: a
price that is possible but never reached, a welfare record no tick wrote. The format
checks structure, bounds, references and conservation. A consumer that restores bytes
it did not write has the trust problem of any save file; the library's guarantee is
memory safety and the invariants of the ledger.

6.2.4. **The checks of the later modules** are those their owners list, run in step 6 as the
section is decoded and in step 7 where they need two sections, each a failure with
`ErrCheckpointCorrupt` naming the tag and the entry: STO `spec-storage.md` 15.3 (the pledge is
not tested), SET 11, EMP 16, LND 14.2, CRD 17.5, CAP 15.3, RSK 15.3, INS 16.5, FIS 17, ORG 12.3, PLC
13.2 and TEC 12.2. A saved scale equal to its default, a pair or a row out of order and a count
that the configuration does not imply are corrupt like any other (4.1.4, R10).
### 6.3 Errors

Sentinels at the root (architecture.md 8.4), matched with `errors.Is`, each with a
failing-case test (testing.md 3.9). None is logged by the library (architecture.md
8.3). The wrapping error names what failed (a tag, an offset, a name, the two versions
or hashes) and is built with `%w`.

| Sentinel | When | Step |
|----------|------|------|
| `ErrCheckpointVersion` | the format version is not one this library reads. It carries the version read and the supported range | 1 |
| `ErrCheckpointCorrupt` | the bytes are not a checkpoint: too short, wrong magic, wrong digest, a non-minimal or overflowing varint, a count or length larger than the input, a bad boolean or enum, an unsorted or duplicated key, a value outside its bound, a section that ends early or runs over, an unexpected tag, a dangling reference, a failed cross-check, a failed audit | 1, 6, 7, 8, 10 |
| `ErrCheckpointConfig` | the configuration's hash is not the checkpoint's. It carries both hashes and the encoding version | 3 |
| `ErrCheckpointExtension` | an extension does not match: a saved name that is not registered or not `Stateful`, a registered `Stateful` extension with no saved entry, a kind that differs, a version the extension refuses, a `LoadState` error | 4, 11; and `Checkpoint` on a healthy world when a `SaveState` returned an error, which it wraps (3.2.2) |
| `ErrOptionContext` | an option that is not valid in this call: `WithSeed` on a restore, `WithReseed` on `NewWorld` | 2 |
| `ErrWorldFailed`, `ErrInStep` | `Checkpoint` on a failed world or inside a step (`spec-world.md` 15.1) | |

6.3.1. **A `ValidationError` from step 3** is the same error `NewWorld` would return
for the same `Config`; a consumer that passes an invalid configuration to a restore
sees what it would see at construction, before any comparison with the checkpoint.

### 6.4 Options on a restore

| Option | On a restore |
|--------|--------------|
| `WithSeed` | **refused**, `ErrOptionContext`: the header holds the seed, so a seed given to a restore would be accepted and ignored, which is a trap (`spec-world.md` 12.3) |
| `WithReseed(salt uint64)` | **restore only** (6.8); on `NewWorld` it is `ErrOptionContext` |
| `WithLogger`, `WithObserver`, `WithAudit` | as in `NewWorld` |
| `WithTargetSpread` | accepted, and part of the effective configuration, so the value must be the one the original had or the hash differs (`ErrCheckpointConfig`) |
| `WithStrategy`, `WithSpawner`, `WithClearing` | as in `NewWorld`: the names the configuration and the checkpoint use must be registered, and a `Stateful` extension's entry is matched by name (6.2 step 4) |

### 6.5 The failed-world rule

6.5.1. A world that failed a step is **restored, not continued** (`spec-world.md`
3.5). The consumer's path is `Checkpoint` taken at a healthy boundary before, then
`RestoreWorld` with the same configuration and, if the failure was a faulty extension,
another extension. A checkpoint **never encodes** a failed world (`Checkpoint` refuses
one), so no checkpoint can resurrect a failure.

6.5.2. A restore that itself fails returns no world; the consumer has the same
checkpoint and may try again with another configuration or extension.

### 6.6 Partial and corrupt input

6.6.1. **Memory safety.** A restore of any byte string never panics and never allocates
more than a constant times the length of the input plus the size the configuration
implies. R8 (a count is at most the bytes remaining) and R5 (a length is at most the
bytes remaining) are what make that true; the beliefs of an agent, the residues and the
other implied counts are bounded by the configuration, which the consumer controls and
validated already. It is tested by a fuzz test of `UnmarshalBinary` and of
`RestoreWorld` over a small configuration (FR-CHK-016, NFR-CHK-003).

6.6.2. **Truncation and extension.** A checkpoint cut short fails the digest or the
length checks; one with bytes added moves the trailer, so the SHA-256 of what
precedes the last 32 bytes is not those 32 bytes. Neither reaches a section.

6.6.3. **The canonical property.** If `RestoreWorld` accepts a byte string, then
`Checkpoint()` of the world it returns, taken at once, is **byte-equal** to it
(FR-CHK-005), **when it was restored without `WithReseed`** (a reseed rewrites the seed of the header, so the next checkpoint differs by design,
6.8.3). That is the
property of one spelling (R10) and is the fuzz test's oracle: every accepted input is a
fixed point of restore-then-save.

### 6.7 What the restored world is

6.7.1. It is at tick `Info().Tick`, its seed is that of the header (and it has no stream position: every stream is a function of the seed), its next
agent ID is the saved one, and `Err()` is nil, `Config()` is the effective configuration it was built from and `ConfigHash()` is the header's hash (FR-CHK-082). Every agent, account, job, belief,
record, shipment and contract of the original is there with the same ID. The observers
and the logger are the ones given to the restore, and have seen nothing of the world
before it.

6.7.2. **Construction.** If the flag of section 1 is set (the checkpoint was taken
before the first `Step`), the world is in its construction phase: its bucket is the
saved one, a call between now and the first `Step` takes the ordinary defaults of
`spec-ledger.md` 5.2.3 and is totalled in the bucket, exactly as on the original. If it
is not set, the world is not in construction and the boundary balances hold the total
(`spec-world.md` 5.5.3). OBS emits the bucket as a `BoundaryFlow` with `Tick` -1 **at the
first step** (`spec-observability.md` 4.4.2, N-072), so a world restored in construction
emits it at its first step, as the original would have, and a world restored after its
first step emits none (the original did, if anyone listened).

6.7.3. **Events.** `WorldRestored` once, as the **first** event, after the world is
valid and before any other, to the observers given to the restore, with `Tick` the
restored tick; **OBS fixes its fields** (`spec-observability.md` 4.4.5: the hash, `Format`, whether it was reseeded and the `Salt`, the
names table and the **snapshot at the restored boundary**, built
only when an observer was supplied). It is not a replay: no `AgentSpawned` is emitted for
the agents that exist, so an adapter that keeps counts from deltas starts from the
levels the event carries (11). A restore that fails emits nothing at all.

6.7.4. **Logs.** The configuration's warnings at `Warn` with `rule` and `path`, once
each, as at construction; then `Info` "world restored" with `tick`, `seed`,
`config_hash`, `agents`, `households`, `format` and `bytes`. Nothing at `Info` or above is
emitted by `Checkpoint`.

### 6.8 Reseeding a branch

6.8.1. A consumer that restores one checkpoint many times (a branching experiment, 11)
gets identical continuations unless something differs: the same randomness. For a
different continuation it passes `WithReseed(salt)`.

6.8.2. **The rule** (a rule constant, part of the golden output once used): with the seed
of the header, which is the effective seed of the world that was checkpointed
(`spec-world.md` 7.1.1),

```
seed' = finalise(seed xor finalise(salt))
```

with `finalise` the SplitMix64 finaliser of `spec-world.md` 7.1.2 (it holds no increment: the
increment `0x9E3779B97F4A7C15` is in the keys of 7.1.3 and not in this rule). `seed'` is the
effective seed of the restored world from its first draw: every stream is a function of the
seed (`spec-world.md` 7), so the restore **installs no random state and takes no word**, and
a module that keys anything from the seed (the technology key of `spec-technology.md` 4.4.1
and 4.4.6) derives it from the effective seed and re-derives nothing of its own at the
reseed (N-202, answered by ADR 0008). `finalise` and `xor` with a constant are bijections of
`uint64`, so for one seed distinct salts give distinct `seed'`, and so distinct streams. A
salt of 0 is a reseed like any other (the option's absence is "no reseed"): it gives
`finalise(seed)`. One case is the identity: `finalise(0)` is 0, so **the seed 0 with the
salt 0 gives `seed'` 0**, the streams of no reseed at all, and `WorldRestored` still
reports it as reseeded with the salt 0 (the only such pair known: 0 is the one fixed point of
`finalise` known). A harness that wants every branch to differ from the unreseeded
continuation gives a salt other than 0, or a seed other than 0. The vectors, sixteen of them
and each with the first word of a keyed stream under `seed'`, are in 17.5. **Stage 0 of the
roadmap builds the pure function only** (`reseed(seed, salt)` in `random.go`, over the
finaliser of `internal/ordering`); the option, the restore step 9 and `WorldRestored` are
Stage 2 (`checkpoint.go`).

6.8.3. A reseed is part of **the sequence of calls** of architecture.md 7.4 (like an
external agent's `AddAgent`), is not hashed, and is reported in `WorldRestored`
(`Reseeded`, `Salt`). It is the **one option** that is both restore-only and a call in
the sequence: it changes what the next checkpoint saves (the seed of the header becomes
`seed'`), which no other option does (`spec-world.md` 4.4 and FR-WLD-020, FR-WLD-021 are
edited to say so, 20). **The header of the reseeded world's own checkpoints carries
`seed'`, not the original seed**: the seed is the one random datum and a restore reads it
back (5.4.1). A second reseed composes (`seed''` is the rule applied to `seed'` and the second
salt), and a checkpoint of a reseeded world restores, with no `WithReseed`, to the same
streams.

## 7. Extension state

### 7.1 The interface

ADR 0002 decision 7 asks for "an optional pair of methods that save and load [an
extension's] state as opaque bytes" and leaves the exact interface open (open point 1).
It is:

```go
// Stateful is implemented by a Strategy, Spawner, Clearing or Policy that keeps state between
// calls, so that a checkpoint can save it and a restore can load it. It is optional: an
// implementation that keeps no state does not implement it.
type Stateful interface {
	// SaveState returns the whole state of the extension and the version of its encoding.
	SaveState() (version uint32, state []byte, err error)
	// LoadState replaces the whole state of the extension with the one SaveState wrote
	// under the same version. It returns an error for a version or bytes it cannot read.
	LoadState(version uint32, state []byte) error
}
```

7.1.1. **It is not an extension point.** The table of architecture.md 4.3 (five rows with
`Policy`) is unchanged by it: no code requires `Stateful`, and the world looks for it by a type assertion
at construction and at `Checkpoint`. It is defined in the root package, where it is
consumed (architecture.md 4.1), and has two methods (4.2). It has no default
implementation and no compile-time assertion (4.5 is for extension interfaces).

7.1.2. **Which kinds.** `Strategy` (registered by name, `spec-agents.md` 6.6), `Spawner`
(`spec-spawner.md` 3.1.3), `Clearing` (`spec-world.md` 4.2) and `Policy`
(`spec-policy.md` 4.5). The four registration tables are the four kinds of an entry; the kind is a
value of the format (`0` strategy, `1` spawner, `2` clearing, `3` policy).

7.1.3. **Not `Observer`.** An observer must not influence a result
(observability-and-logging.md 1.4, architecture.md 11.3), so it has no state that a
resumed run depends on; its own aggregates are the consumer's. After a restore it has
seen nothing and rebuilds from `WorldRestored` and its snapshot (6.7.3). Saving an
observer's state would also make the library depend on code that is optional and
no-op by default.

### 7.2 The contract

The doc comments of `Strategy`, `Spawner`, `Clearing`, `Policy` and `Stateful` state these, as
architecture.md 7.7 requires for the random source, because an implementation that
breaks one of them breaks bit-identity after a restore (8) and the library cannot see it
(7.3).

1. **When.** `SaveState` is called by `World.Checkpoint` and `LoadState` by
   `RestoreWorld`, between ticks; never inside a step, never by the world otherwise.
   Each registered `Stateful` extension is called once per checkpoint or restore, in the
   order of (kind, name).
2. **`SaveState` changes nothing.** It does not change the extension, and equal states
   give equal bytes: no map order, no pointer or address, no time, no host data. The
   world copies the returned slice before it returns, so the extension may reuse its
   buffer.
3. **`LoadState` replaces the whole state**, as if the instance were freshly built and
   then loaded. It **validates** its input (the bytes passed digest, and may still have
   been altered, or written by another version), returns an error rather than panic, and
   does not keep the slice it is given, which aliases the checkpoint's bytes.
4. **All state that can influence a result is in the bytes**, or is a function of them
   and of the arguments of the calls. No state in closures, captured variables,
   package-level variables, goroutines or files (architecture.md 7.1, 12.2).
5. **No randomness.** Neither method receives the world's `*rand.Rand` and neither may
   create a source; the extension's draws in `Work`, `Trade`, `Decide` and `Clear` come
   from the world as before (architecture.md 7.7), and its position in the stream is the
   world's state.
6. **One instance per world.** An instance that holds state is not shared between two
   worlds: `LoadState` into an instance another world is using changes that world. A
   consumer that runs many worlds (EXP, 11) registers a new instance for each.
7. **One registration of an instance.** An instance registered under two names, or
   reachable twice (as a `Directed` fallback and as a registration), is called once per
   entry and must give the same state each time.

### 7.3 What the world saves, and what it can and cannot detect

7.3.1. **The entry.** For every registered extension that implements `Stateful`, used
or not, section 34 holds one entry: kind, name, the version and the bytes `SaveState`
returned (possibly none). Entries are in kind order, names ascending as byte strings.
An extension that does not implement `Stateful` has no entry.

7.3.2. **What the world detects**, at restore step 4 and step 11 (6.2), as
`ErrCheckpointExtension`:

| Case | Detected |
|------|----------|
| the checkpoint has an entry for a name that is not registered | yes |
| the name is registered, but as another kind or without `Stateful` | yes |
| a registered `Stateful` extension has no entry | yes: the checkpoint was taken with another set of registrations, and the instance would be left in its construction state, which is silent divergence |
| the version is one the extension refuses, or the bytes do not decode | yes, from `LoadState`'s error |
| an agent names a strategy (an external agent's, which the hash does not cover) that is not registered | yes, at step 4 |

7.3.3. **What it cannot detect.** (a) An extension that **has state and does not
implement `Stateful`**: the world cannot see a closure or a field. ADR 0002 open point 1
asked; the answer is no, and the guard is a test the consumer owns: checkpoint, restore,
step both and compare the checkpoints byte for byte (8.2), which the library's own test
uses with a deliberately stateful extension and a deliberately hidden one to show the
difference (FR-CHK-036). (b) A different **implementation under the same name** with the
same version: the code is not hashed and cannot be (`spec-agents.md` 6.6, open question
4). The version number of `Stateful` is the only code identity the library compares, and
it is offered for that: an extension with **no state** may still implement `Stateful`,
return `(v, nil, nil)` and refuse any `v` it does not know, to get the check for the
price of two methods. AGT open question 4 ("a version string would let the hash cover
it") is answered this way and not by a new optional method.

7.3.4. **After a failed restore** at step 11 (an extension's `LoadState` refused), the
instances that were already loaded are changed and the instance that refused may be
half loaded. A consumer **discards** the registered `Stateful` instances of a failed
restore and builds new ones. Steps 2 to 10 never touch them (6.2.1).

### 7.4 Versioning

7.4.1. The version is the **extension's**: a number it chooses, incremented when the
shape of its bytes changes. The world does not interpret it, compares it with nothing
and does not migrate it; it passes it to `LoadState`, which may read older versions
(migration is the extension's, as the library's is its own, 9) and refuses newer or
unknown ones. The format version of this spec (5.1) is separate: a change of the frame
never changes an extension's version and the converse.

7.4.2. **Why a number in the interface, and not inside the bytes.** The world can then
report a refusal uniformly (`ErrCheckpointExtension` with the name, the kind and both
versions) and a tool can list the extensions of a checkpoint and their versions without
running them.

### 7.5 The library's own extensions

`DefaultStrategy`, `DefaultSpawner` and `AverageClearing` have **no state** (the
default strategy and spawner say so in `spec-agents.md` 15 and `spec-spawner.md` 14;
`spec-clearing.md` 7.1 for the clearing) and do not implement `Stateful`.
`strategy.Merchant` has none (`spec-merchants.md` 3.5). The built-in policies `hold`, `schedule` and
`rule` keep no state (`spec-policy.md` 4); the built-in **policy** `directed` keeps its mailbox in the world,
in section 32, and is not `Stateful`. The built-in **strategy** `directed` is the library's `Directed` over
`DefaultStrategy`: it is `Stateful` (below), so every world has its entry, of kind 0 (strategy), name
`directed`, version 0 and no bytes while the fallback keeps none (FR-CHK-037); the two names are
different things in different registries. `Directed`
(`spec-agents.md` 10.3) has no state of its own (the mailbox is in the agent) and
**delegates**: its `SaveState` returns the fallback's when the fallback implements
`Stateful` and `(0, nil, nil)` otherwise, and its `LoadState` hands the bytes to the
fallback or, if the fallback is not `Stateful`, requires them to be empty. So
`Directed{Fallback: x}` registered by a consumer is saved exactly when `x` is.

### 7.6 The amendment to architecture.md 4.3 and 7.7

ADR 0002 consequences ask for the rules to say this "in the same pull request as the
code, with a fresh read-receipt tag". The exact text, for the supervisor to apply and
tag when the first code lands (notes N-019):

> **4.3**, after the paragraph that ends "is not subject to 4.5.":
>
> **Extension state.** An implementation of `Strategy`, `Clearing`, `Spawner` or `Policy` that
> keeps state between calls MUST also implement the optional interface `Stateful`, so
> that a checkpoint can save it and a restore can load it (ADR 0002, decision 7):
>
> ```go
> type Stateful interface {
> 	SaveState() (version uint32, state []byte, err error)
> 	LoadState(version uint32, state []byte) error
> }
> ```
>
> `Stateful` is not an extension point and is not subject to 4.5: no code requires it,
> an implementation that keeps no state does not implement it, and it has no default.
> `Observer` is never saved: an observer must not influence results
> (observability-and-logging.md 1.4), so it has no simulation state.
>
> **7.7**, at the end:
>
> An implementation that is `Stateful` MUST keep in what `SaveState` writes every piece
> of state that can influence a result (no state in closures, package variables or
> goroutines, 7.1 and 12.2). `SaveState` MUST NOT change the extension and MUST give
> equal bytes for equal states. `LoadState` MUST replace the whole state, MUST validate
> its input and return an error for a version or bytes it cannot read, and MUST NOT keep
> the slice it is given. Neither method receives or may create a random source. An
> instance that holds state MUST NOT be shared between worlds. The doc comments of
> `Strategy`, `Spawner`, `Clearing`, `Policy` and `Stateful` MUST state this contract, because an
> implementation that breaks it breaks 7.4 after a restore and the library cannot detect
> it.

The options list of architecture.md 6.2 gains `WithReseed(salt uint64)` (restore only)
beside `WithSeed`, and the glossary of architecture.md 3.2 the terms of section 20.

## 8. Bit-identity

### 8.1 The requirement

8.1.1. **Statement.** Let `W` be a healthy world at a tick boundary `t`, built from a
configuration and a sequence of calls, and let `c = W.Checkpoint()`. Let `W'` be
`RestoreWorld(cfg, c, opts)` with the same configuration, the same options that affect
results (`WithTargetSpread`), the same registered extensions and no `WithReseed`. Then
for every sequence of later calls `S` (steps, runs, between-ticks calls) applied to both:

1. the events of `W'` equal the events of `W` after the boundary, in the same order,
   field for field;
2. every `Snapshot` and every query of `W'` equals that of `W` at the same ticks;
3. the checkpoints of `W'` and of `W` are **byte-equal** at every boundary after `t`;
4. the seed is the same, and so is every stream, so every later draw is the same;
5. the errors the two return are the same, and one fails at the same step as the other.

8.1.2. **Why byte-equal checkpoints are the test.** The checkpoint contains all state
(4), so equality of checkpoints at every boundary is equality of the worlds, and it
needs no knowledge of what is inside: the test is one comparison of two byte strings.
Equal events, snapshots and queries are the observable half; they catch a derived value
that was rebuilt wrongly (a rebuilt table that differs from the original) where the
state matched.

### 8.2 The test

8.2.1. **The test (ADR 0002, required tests (a) to (c))**, for the reference chain and a
range of seeds (testing.md 6.2: a failing seed is printed):

```
run N ticks on W; c := W.Checkpoint()
W' := RestoreWorld(cfg, c, ...)
record the events of W' and of W for M more ticks, compare them
at each of the M boundaries compare W.Checkpoint() and W'.Checkpoint() bytes
```

with `N` and `M` varied: `N = 0` (before the first step: construction, 6.7.2), a `N`
inside a job and a season, a `N` after an external agent was added, a shock set and a
shipment launched. Required legs: (b) a checkpoint of the same state twice gives the
same bytes; (c) a restore with another configuration, another format version or a
removed extension fails with the right sentinel.

8.2.2. **Splits.** The run is also split into `Run(n)` calls and a restore in each gap
(`FR-WLD-063`); and a restore-of-a-restore (a restored world is checkpointed and
restored again) is equal to the original.

8.2.3. **The scenarios that wait for this spec**: X5 of `spec-reference-chain.md` 7.4
(200 ticks, save, restore, 200 more against 400) on `one` and `five`, and the
checkpoint legs of S60 (a household in the own-use mode) and S72 (merchants and
consignees, a checkpoint at 80) of the register, and the checkpoint leg of SET-S5 of
`spec-settlements.md` 14.2: a checkpoint taken during a held class change, restored and run
on, with the class, `Since` and the tick at which the hold ends those of the uninterrupted
world (the register number is assigned with the others, `spec-stability.md`).

### 8.3 What the golden rendering compares

`spec-reference-chain.md` 8.5 item 6 compares the **digest of the checkpoint** at tick
`T`: the 32 bytes of the trailer (5.4.2), as hexadecimal. It is the one value that
changes when **any** saved state changes, so it is the cheapest golden check that the
whole state is deterministic, and it is checked on two architectures (architecture.md
7.4). A change in it follows testing.md 7.2: the PR says what changed. Because the
digest includes the format version, a format change moves every golden digest at once
and says so in its PR (9.3).

### 8.4 The random source

A checkpoint saves the seed and no stream state (ADR 0008): every draw is a function of
the seed, a site, the tick and an entity (`spec-world.md` 7.1), so there is no position in
a stream to save. Everything that is "a position in a sequence" is saved by the module
that owns it and is ordinary state: the carried remainders of money flows, population,
demand, regeneration and the outside's capacity, the death carry of an agent, the residues
of a recipe, the privation count, the review and cadence positions (a function of the tick
and the ID, so derived).

8.4.2. A restore installs nothing and takes no word, so after `RestoreWorld` the next
word of a stream is the next word of the original (FR-CHK-051 holds trivially). A reseed
(6.8) changes the seed by the rule there and takes no word.

### 8.5 What is not guaranteed

| Not guaranteed | Because |
|----------------|---------|
| The state of an observer, of a logger, of anything the consumer holds | they are not simulation state (7.1.3) |
| The log lines before the boundary, and the `Info` "world created" of the original | a restore starts a log of its own; "world restored" replaces the creation line, and the warnings of the configuration are logged again, once each |
| A restore into a library of **another behaviour**, even with the same format | a rule that changed changes the continuation; the format version does not say it (9.2). Bit-identity is between two worlds of one library version |
| The code behind a registered name | not hashed (7.3.3) |
| Equal wall-clock or memory | not a state |
| That a hidden-state extension survives | 7.3.3 |
| Identity between a world and a **reseeded** restore | by design (6.8) |

## 9. Compatibility, migration and compaction

### 9.1 Before 1.0

9.1.1. **Strict.** A checkpoint restores only into a library that writes **its exact
format version**; any other version is `ErrCheckpointVersion`, reported with the version
read and the version supported (ADR 0002, decision 8). The format version is bumped
whenever anything in 9.3 changes, as often as development needs, and no migration is
offered. Before 1.0 a checkpoint is **not an archive**: a consumer that must keep a
world across a release pins the library version, or keeps the seed, the configuration
and its call log and replays (ADR 0002, decision 9; slow and only sound if the behaviour
did not change in between).

9.1.2. **Why strict.** Every field of every module would otherwise have to be
versioned by the specs that own them while those specs are still changing weekly (every
spec in this directory is a draft with placeholder numbers, N-027). A migration written
now would be rewritten with each of them.

### 9.2 From 1.0

9.2.1. **The policy** (ADR 0002 open point 3, decided here for the v1 series):

1. The library **writes only its current format** and **reads every format that any
   v1.x release wrote**. A v1.N library restores a checkpoint of a v1.M library for
   `M <= N`; it never restores a newer one (`ErrCheckpointVersion`).
2. The format version is a single increasing integer within a major version. A change
   of the shape of the state **adds** a format version and a decode path for the old
   one that fills a field the old format lacked with the default the new spec defines
   (it cannot invent state). A change that cannot be expressed that way, because the
   meaning of saved state changed, is a **breaking change** and needs a major version
   (architecture.md 6.6); the module path then gains the suffix and a v2 library may
   read v1 formats or not, as that release says.
3. Each format version ever released has a **fixture**: a checkpoint of the reference
   chain at a stated tick, written by the release that introduced it and committed under
   `testdata/checkpoints/` with the digest of the world it continues to, checked by a
   restore-and-continue test (FR-CHK-062). A format may be dropped only in a major
   version.
4. The **configuration hash version** in the header (5.1) is kept the same way: a v1
   library retains the encoder of every version of the configuration encoding it ever
   wrote (`spec-config.md` 8.2), so an unchanged pack continues to match a checkpoint
   across a change of that encoding. A pack that changed is a different hash, as ever.

9.2.2. **Behaviour is not format.** A checkpoint restored by a later release continues
under the later release's rules. That is not an error and is not detectable: the
checkpoint holds state, not rules. Bit-identity (8.1) is between two worlds of one
library version. A consumer that needs the old behaviour pins the release.

### 9.3 What changes the format version

| Change | Version |
|--------|---------|
| a field added, removed, reordered or given a different range or meaning in any section | yes |
| the order of a collection, or the key it is sorted by | yes |
| a primitive or a rule of 5.2, the frame, the header or the trailer | yes |
| a piece of state becomes derived, or a derived value becomes state | yes |
| a section added, dropped, renumbered or changing the condition under which it is present | yes |
| the seed, the key of a stream, the site constants or the reseed rule | yes (the streams would mean something else; the row is reworded under ADR 0008, see N-214) |
| the encoding of the configuration hash | **no**: the header's hash version covers it (9.2.1 item 4) |
| an extension's own bytes | **no**: its own version (7.4) |
| a rule of behaviour that leaves the shape of state alone | no (and a golden change, testing.md 7.2) |

Nothing has been released, so by 9.1.1 the sections 23 to 33 and the fields they host (the
stores, the membership, `LossBP`, the claims and their `Origin`, kind `5`, extension kind `3`) are
written into **format 1**, and the format version is not bumped before the first release that
has a later-stage module. The module specs that say "the format version changes with the first
release that has the module" are met by this sentence and by the fixture of 9.2.1 item 3, which is
one checkpoint of a world with every module on (`testdata/checkpoints/`).
### 9.4 Closed accounts: compaction at save, or never

9.4.1. **Decision: never, in version 1.** Account IDs are never reused
(`spec-ledger.md` 3.2) and an agent's removal closes its account; a closed account
holds nothing. The ledger's open question 6 and N-008 asked whether a checkpoint should
**compact** them. It does not: a closed slot is saved as the pair *(kind, closed)*,
**two bytes**, and a weight change closes nothing.

9.4.2. **The growth is small.** One slot per removal (`spec-spawner.md` 14: reductions,
entries into a cohort and role changes close none). The model of section 16 (case C):
10 000 deaths in a world of 5 000 agents add about 20 KB to a checkpoint of about 2 MB,
1%; a full-detail realm of 200 000 agents that lost a million households would add 2 MB
to 86 MB. A weight-1 population module in a closed region is the worst case the specs
produce (`spec-population.md` 11), and it is bounded by the deaths of the run.

9.4.3. **Why compaction would be a design change.** Renumbering accounts would have to
rewrite every reference to one (an agent's account, a flow's carries keyed by account
ID, a shipment's Transit account, a contract's Escrow account), and the restored world
would then have account IDs that differ from the original's: any later position or
query that orders or exposes an account ID (the ageing order, a carry list, an audit
message) would differ from the uninterrupted run, which is exactly what bit-identity
forbids (8.1). It is possible only with an order-preserving renumbering and an
indirection kept in the live ledger, a cost on every run to save bytes on some saves.

9.4.4. **The trigger to revisit.** A world whose closed slots are more than a quarter
of its checkpoint bytes, or whose restore time is dominated by them (NFR-CHK-004
reports the share). Until a consumer shows one, the answer stands (register entry
CHK-A3).

### 9.5 A changed configuration

No migration between configurations exists: the hash differs and the restore is
`ErrCheckpointConfig`. A pack that gained a commodity is a different world with
different dense indices, IDs and counts (`spec-world.md` 5.3.2). A consumer that
upgrades a pack starts a new run from its seed (with a warm start, VIA) or keeps the
old pack beside the old checkpoint. A "soft" restore that adopts what matches and
defaults the rest was considered and rejected: it would make bit-identity depend on
which fields happened to match.

## 10. Cost

### 10.1 What is counted

The library measures no wall-clock time (architecture.md 7.3, `spec-scale.md` 6.1), so
cost is stated as **bytes** and **counts** and, with placeholders (N-015), as a multiple
of the tick budget of `spec-scale.md` 6.3. A checkpoint is not part of a tick and is
outside the budget of NFR-SCL-002; it is a call the consumer makes.

### 10.2 Bytes (the model of section 16, `models/chk/size.py`)

All numbers are of a stylised inventory with invented value magnitudes (the ratio and
the shape matter, not the digits). Weights of 1; `|T_a|` 7.7 (the five-region pack of
`spec-reference-chain.md`) or 8.

| Case | Agents | Welfare window | Modules | Varints | Fixed 8-byte |
|------|-------:|---------------:|---------|--------:|-------------:|
| A. first consumer | 5 000 | 30 | none | **2.1 MB** | 13.6 MB |
| B. long-job window | 5 000 | 120 | demand, information, population | 6.6 MB | 42.6 MB |
| C. ten thousand deaths | 5 000 | 30 | the same | 2.2 MB | 14.0 MB |
| D. full-detail realm (`spec-scale.md` 6.3) | 200 000 | 30 | the same | 86 MB | 546 MB |

Of case A the welfare rings are 1.48 MB (70%), the ledger's lots and balances 0.30 MB
(14%), the beliefs 0.16 MB (7.5%), production 0.09 MB (4%), the agents
0.08 MB (4%), and the markets and estates together under 0.01 MB. **The welfare ring dominates** because it is
eight integers a tick for `T` ticks for every agent: the size is `O(agents * T)`, not
a function of weights or households (NFR-SCL-004), and `T` is the effective window
(`spec-welfare-and-wages.md` 2.3.3, at least twice the longest job).

10.2.1. **The later-stage sections** add little to case A, and none grows with the welfare window.
The two fields hosted in section 3 are per agent: the stores (one byte, a count of 0, with
`Modules.Storage`) and the membership (two bytes, 0 and 0, with `Modules.Institutions`), so
`5 000 * (1 + 2)` is 15 KB, about 0.7% of the 2.1 MB; `LossBP` is up to two bytes a shipment in
flight. The sections themselves are a count of rows of the world, not of its agents: the
hires (about eight varints each), the claims (about twelve), the policies (about nine), the
facilities (about eight), the titles (two per holder) and the diagnostic runs and scales,
which are sparse; technology is `R * F` cells of three varints and the settlement section
is two varints per region. At 100 rows of each kind the module sections together are
under 10 KB. The figures are counts of the layout of 4.2, not a run of `size.py`, which
was not extended for them (16).
### 10.3 Time

Counted as integers written (about 1.7 million in case A, 68 million in case D) and
bytes hashed. With the placeholder costs of 5 ns an integer to encode, 8 ns to decode
and validate, 2 ns a byte for SHA-256 and a scan of 30 ns a lot for the audit, the
first consumer's checkpoint is about 13 ms and its restore, with the derived tables and
the audit, about 25 ms: **a half to a quarter of a tick** of the 53 ms of
`spec-scale.md` 6.3. The full-detail realm is about 0.5 s and 1 s against a 2 s tick.
Both are `O(integers)`, linear in agents and lots, nothing quadratic, and none of it is
measured (N-015). A general-purpose compressor applied by the consumer will shrink the
bytes further (most records are small and repetitive); the library does not compress.

### 10.4 Memory

A checkpoint is one copy of the state in encoded form: the world's working set is
untouched, and the extra memory during `Checkpoint` is the size of the encoding (about
2 MB for case A). `RestoreWorld` holds the input and the new world at once; the original
may still be alive, so a branching consumer holds the checkpoint once and as many worlds
as it has restored (11).

### 10.5 The scale-limit case

The configuration bounds (`spec-config.md` 4.1) allow 10^6 weight per agent and as many
agents as memory holds. The bytes do not grow with weights; they grow with agents,
lots and `T`. The levers for a world too large to checkpoint comfortably are the scale
of the groups (`spec-scale.md` 3), the window (`WelfareWindow`) and the length of the
longest job, all of them the pack's, none of them the format's.

## 11. What CHK hands to EXP, INT, OBS, STY and IND

Stated as guarantees and asks on the surface of this spec.

| To | Guarantee or request |
|----|----------------------|
| **EXP** | (1) `RestoreWorld` builds a **new** world that shares nothing with the checkpoint's source, so a harness may restore many worlds from one `Checkpoint` and step them on separate goroutines (`spec-world.md` 3.1.1). (2) **Branching** is checkpoint once, restore many: the cost of a branch is a restore (10.3), cheaper than a construction with a warm-start solve (`spec-viability.md`). (3) Two restores of one checkpoint continue **identically** given the same calls, so a paired comparison (same randomness, one parameter or shock differing; two restores of one checkpoint share the configuration, so a differing parameter is not a case between two restores: it would be `ErrCheckpointConfig`, 6.2.1, and the difference is a shock or a call) is aligned in the draws by construction (`spec-world.md` 7.6): every draw is a function of its key, so the runs diverge through state only and the stream fingerprint of EXP 7.3 reads nothing. (4) For **independent** continuations from one state the harness passes `WithReseed(salt)` with a different salt per branch: deterministic, so a run is reproducible from (checkpoint, salt, calls). (5) A **cloned** world is `RestoreWorld(cfg, w.Checkpoint())`; there is no in-memory clone (open question 2 of 20). (6) A stateful extension is **one instance per world** (7.2 item 6): the harness registers a new instance for each branch; a shared one corrupts the others. (7) The key of a result is (`Info().ConfigHash`, the digest of the trailer, the salt) |
| **INT** | (1) The names are `World.Checkpoint`, `Checkpoint` (with `MarshalBinary`, `AppendBinary`, `UnmarshalBinary`, `Info`) and `RestoreWorld`; INT lists them in its surface and adds no second path. (2) `Checkpoint` is a **read** (WLD 3.7.1): it changes nothing, draws nothing and is refused with `ErrInStep` inside a step. (3) The **sequence of calls** is not saved (INT 9.2): a restored world has no history of calls, and a consumer that wants a replay log keeps one. (4) What a restored world answers (`Slots`, `Agent`, `Direct`, `Market`, `Region`) equals the original's at the same boundary (FR-CHK-052); INT's one piece of state, the rumours of the last view of an external agent, is section 22. (5) A restore **before the first step** keeps the construction phase (6.7.2), so INT 3.1.1 holds for a restored world. (6) An external agent's strategy name is checked at restore (6.2 step 4) |
| **OBS** | (1) A restore emits `WorldRestored` as its first event with the snapshot at the restored boundary (OBS 4.4.5), and **no** `AgentSpawned`, `AgentBankrupted` or `BoundaryFlow` for the past. (2) A snapshot is a function of the checkpoint and the configuration (OBS 5.2.1), which this inventory makes testable: every field of a snapshot is derivable from the sections of 4.2 (FR-CHK-053). (3) `Checkpoint` emits no event (OBS 4.4.7: a read emits nothing), so reading cannot change the stream; a `Debug` line "checkpoint taken" with `tick` and `bytes` is the only trace. (4) Two log keys are new: `format` and `bytes` (14). (5) The adapter's instruments fed by deltas start from the levels of the event, not from zero (OBS 6.4) |
| **STY, IND** | (1) A run split by a restore is **the same run**: the indicators and the stylised-fact tests may take a checkpoint anywhere without changing a measure (FR-CHK-050). (2) Events are not repeated across a restore, so an indicator that accumulates keeps its tally as the consumer's plain value (`spec-indicators.md` 10.5): it survives a restore as it is, and a consumer that lost it opens a new one at `WorldRestored` (IND 10.5.2). (3) The digest of the trailer is the golden check of a state (8.3) |
| **WLD, CFG, LED** | the edits of section 20, in particular: the construction flag and bucket, the audit baseline, the hash version in the header, `ErrOptionContext` and `WithReseed` in the options table |

## 12. Parameters

This spec adds **no `Config` field, no default and no bound**, and so no row to the
parameter-owner table: a checkpoint is not configuration, and what a restore compares
is the hash of the configuration that already exists. The parameter registry test
(testing.md 6.7) needs nothing from it. The options of section 6.4 are not
configuration (`spec-config.md` 9.2).

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the magic
`45 53 43 4B`; the format version (1); the section tags of 4.2 and their order; the
primitives and rules R1 to R13 (5.2); the kinds of an extension entry (7.1.2); the
reseed rule `finalise(seed xor finalise(salt))` (6.8.2; the old constant `0x243F6A8885A308D3` is gone, ADR 0008); the order of validation (6.2);
the one-spelling rules.

**Not parameters**: the tick, the seed, the digest, the construction flag.

## 13. Validation

13.1. **Errors this spec adds** are the five sentinels of 6.3 (`ErrCheckpointVersion`,
`ErrCheckpointCorrupt`, `ErrCheckpointConfig`, `ErrCheckpointExtension`,
`ErrOptionContext`), mapped at the root as N-075 asks (they are root errors, not
`internal/ledger` ones). Each has a failing-case test (testing.md 3.9) and none is
matched by message text.

13.2. **Configuration validation is unchanged.** A restore runs `Validate` and the
three name checks exactly as `NewWorld` does (6.2 step 3); this spec adds no rule to
`spec-config.md` 7.

13.3. **What is validated on input** is in 6.2 and 5.2 (R1 to R13), and is the subject of
the fuzz test (testing.md 7.4): the decoder is the one function of the library that
reads bytes it did not write.

## 14. Observability

14.1. **Events.** One: `WorldRestored`, emitted once by a successful `RestoreWorld` as
the first event to the observers given to the restore (6.7.3). Its fields are OBS's
(`spec-observability.md` 4.4.5: `Tick`, `ConfigHash`, `Format`, `Reseeded`, `Salt`, the names
table and the snapshot at the restored boundary); the seed (the original, for
information) and the agent and household totals are in the snapshot's header once. It
is emitted only when an observer was supplied, and the snapshot is built only then
(observability-and-logging.md 4.5). `Checkpoint` emits **no** event: it is a read and a
read changes nothing and emits nothing (OBS 4.4.7). A failed restore emits nothing. The
event is a state transition (a world comes to exist from a checkpoint), so it has one;
it is not per agent, order or trade, and its volume is one per call.

14.2. **Logs**, through the injected logger, constant lowercase messages and the keys of
observability-and-logging.md 3.3.2:

| Level | Message | Attributes | When |
|-------|---------|------------|------|
| `Warn` | the configuration's warnings | `rule`, `path` | once each at a restore, as at construction (`spec-config.md` 7.2) |
| `Info` | "world restored" | `tick`, `seed`, `config_hash`, `agents`, `households`, `format`, `bytes` | once, at the end of a restore |
| `Debug` | "checkpoint taken" | `tick`, `bytes` | once per `Checkpoint`, guarded by `Enabled` |

Two **new keys** (to add to table 3.3.2 in the change that introduces them, as the rule
says): `format` (the format version, logged as `slog.Int64(int64(f))`) and `bytes` (a count of bytes, `int64`).
The errors of a restore are returned and not logged (architecture.md 8.3), and nothing
at `Info` or above is emitted inside a step. A checkpoint of a failed world is an error
return and no line.

14.3. **Metrics.** None are new. The size of a checkpoint and the time of a restore are
the consumer's to measure around the call (the library measures no time, 10.1), and
`bytes` is on the `Debug` line.

14.4. **Snapshot.** Nothing is added: a snapshot is a function of the saved state
(OBS 5.2.1), and a checkpoint is not part of it (OBS 5.7).

14.5. **The Observability entry** (observability-and-logging.md 9): new failure paths
are returned as five sentinels, none logged; one new state transition, a restore, has
one event and one `Info` line; no new loop; no new metric attribute; two new log keys;
the no-op path allocates nothing for it (the event and the snapshot are built only when
an observer was supplied) and `Checkpoint` is outside the tick loop (NFR-CHK-001).

## 15. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. This spec adds **no
feedback loop**: a checkpoint reads state and writes bytes, and a restore installs state
a run already had (FR-STB-003: a rule that reads a price, a stock, an income, a count or
a share and changes one needs an entry; neither call does). It adds three hazards, two
dampers and two scenarios.

**CHK-A1 A hidden-state extension survives no restore.** *Kind:* hazard. *Loop:* none:
an extension (`Strategy`, `Spawner`, `Clearing`) that keeps state outside what it
saves (a field not in `SaveState`, a closure, a package variable) makes a resumed run
diverge from the original with no error, because the library cannot see it (7.3.3).
*Owner:* CHK 7. *Damper:* CHK-D1: the `Stateful` pair with a version, the entry per
registered name, the detection of a missing or surplus entry, and the consumer's
round-trip test of 8.2 with the library's own example. *Undamped:* an extension with
hidden state, by construction. *Condition:* the consumer runs the round-trip test on
any extension that keeps state. *Test:* CHK-S1 with a deliberately hidden extension.
*Status:* hand.

**CHK-A2 Branches that share their randomness are not independent.** *Kind:* hazard
(and a feature for paired comparison). *Loop:* none: two worlds restored from one
checkpoint and given the same calls give **identical** runs; a statistic over such
branches has one sample, not many. *Owner:* CHK 6.8, 11 (EXP). *Damper:* CHK-D2:
`WithReseed(salt)`, one salt per branch, deterministic. *Undamped:* a harness that
forgets it. *Condition:* a Monte Carlo over branches gives each its own salt; a paired
comparison keeps the same one and changes only a parameter that moves no draw count
(WLD-A5): a paired comparison is aligned in the draws by construction (`spec-world.md` 7.6). *Test:* CHK-S2. *Status:* model (a bijection of the salt: the 10^6 first words
of the Go check are distinct).

**CHK-A3 Closed accounts accumulate.** *Kind:* hazard. *Loop:* none: every removal
leaves a two-byte slot that is saved for the rest of the run (9.4); a long run with the
population module in a closed region grows the checkpoint by the number of its deaths.
*Owner:* CHK 9.4 with LED 3.3. *Damper:* none by design: compaction was rejected as a
design change. *Undamped:* by design; the growth is about 1% of a first-consumer
checkpoint per ten thousand deaths. *Condition:* revisit when closed slots exceed a
quarter of the checkpoint bytes (9.4.4). *Test:* NFR-CHK-004 reports the share. *Status:*
hand.

**Dampers.** CHK-D1 *The Stateful pair and its detection* (7). CHK-D2 *Reseed by salt*
(6.8). Neither is a runtime controller (`spec-stability.md` 2.6).

**Scenarios.** CHK-S1 *Round trip over seeds and modules*: the reference chain, `one`
and `five`, with each module in turn on, a range of seeds, a restore at several ticks
including 0, byte-equal checkpoints and equal events for 100 ticks; and a deliberately
hidden-state extension that **fails** the same test (the pass is the detection of the
failure). CHK-S2 *Branching*: from one checkpoint, 100 restores with 100 salts give 100
distinct runs of the same length and 100 restores with one salt give one; the pass is
the count of distinct final digests.

## 16. What was run

All in Docker (`python:3.12-slim` and `golang:1.25`, `PYTHONDONTWRITEBYTECODE=1`),
standard library only, in `doc/specs/models/chk/`. They are models of the rules of this
spec, not of the library, and they show no default to be stable (N-027).

| Script | What it does | Result quoted |
|--------|--------------|---------------|
| `size.py` | the integers each section writes for the first consumer's world and larger, and the bytes as minimal varints and as 8-byte integers, with invented value magnitudes | 10.2: 2.1 MB against 13.6 MB; the welfare ring is 70% |
| `worked.py` | the primitives of 5.2 with the rejected spellings, the world section of the construction of `spec-world.md` 19.1 (the next agent ID and the construction flag), a toy frame and its digest, an account, and the reseed vectors of 6.8 (the rule of ADR 0008), each asserted against the values a Go program printed | 17 |
| (Go, not archived: `go run` of a standard-library file in `golang:1.25`) | `binary.Uvarint` of `80 00` and `81 80 00` (accepted, two and three bytes read) and of an overflowing ten bytes (rejected, `n` negative); the reseed vectors; a million salts give a million distinct first words, and the first formula (the seeding constant) gave the state `(0, 0)` for salt 0 on seed 0, which is why the constant is the digits of pi | 5.2.1, 6.8, 17 |

What they do not show: that the checkpoint of a real world has the sizes of the model
(its value magnitudes are invented, and the inventory is a hand count from the specs'
lists), that the times of 10.3 are right (a count and placeholders), or that a restore of
a world with every module on continues exactly (nothing has run in the library).

One **slip**: the host `python3` was called twice by mistake (once with no script, once
with `--version`), against the rule of `.supervisor/writer-common.md` 2; nothing else ran
on the host and every model ran in Docker.

## 17. Worked examples

All settings are invented **examples**; every number is a placeholder (N-027). The bytes
were produced by `models/chk/worked.py` and its output is `worked.out`.

### 17.1 The primitives

| Value | Bytes |
|-------|-------|
| `U(0)`, `U(1)`, `U(127)` | `00`, `01`, `7f` |
| `U(128)`, `U(300)`, `U(16384)` | `80 01`, `ac 02`, `80 80 01` |
| `U(2^64 - 1)` | `ff ff ff ff ff ff ff ff ff 01` |
| `I(0)`, `I(-1)`, `I(1)`, `I(-64)`, `I(64)` | `00`, `01`, `02`, `7f`, `80 01` |
| `I(10000)` | `a0 9c 01` |
| `I(-2^63)` | `ff ff ff ff ff ff ff ff ff 01` |

Rejected as corrupt (R1): `80 00` (zero in two bytes), `81 80 00` (one in three),
`ff ff ff ff ff ff ff ff ff 02` (overflows 64 bits), `80` (truncated).

### 17.2 The world section of an example construction

`spec-world.md` 19.1 (which states no seed; this example takes seed 0): three regions and seven household agents (IDs 4 to 10),
so the next agent ID is 11. The world has not stepped, so the construction flag is set. The
seed is in the header (5.4) and no random state is saved (ADR 0008). Section 1:

```
tag 01 | length 02 | next agent ID 0b | construction 01
01 02 0b 01
```

The length is 2 bytes (one `U`, one `B`).

### 17.3 A toy frame and its digest

A frame with the header and **only** section 1 (a real checkpoint has every section of
4.2; this is a framing test vector, not a valid world): magic `ESCK`, format 1, hash
version 1, a configuration hash of the bytes `00` to `1f`, tick 0, seed 0.

```
45 53 43 4b | 01 | 01 | 00 01 02 03 ... 1d 1e 1f | 00 | 00 | 01 02 0b 01
```

It is 44 bytes before the trailer and 76 with it; the SHA-256 of the 44 bytes, which is
the trailer, is
`353aa04e43ddfb9955c443efbba401f79839b059efda0200bbf2b8465ec3cf45`. Changing any bit
changes the digest; appending a byte or cutting one fails the length and the digest
(6.6.2).

### 17.4 An account

An open Agent account in region 1, owned by agent 4, holding 10 000 minor units and one
lot of 5 units of commodity 3 that does not spoil or wear (kind 0, closed 0):

```
00 | 00 | 01 | 04 | a0 9c 01 | 01 | 03 05 00 00
kind closed region owner money(I) lots commodity qty born(I) durability
```

12 bytes. A **closed** account is `00 01`: its kind and the closed flag, nothing else
(9.4). Two accounts of equal state have equal bytes whatever the order of operations
that made them (`spec-ledger.md` 5.1.3).

### 17.5 A reseed

The rule of 6.8.2 on four seeds and four salts. The last column is the first word of the
stream `(siteWork, 0, 0)` (`spec-world.md` 7.1.4) under `seed'`, which ties the rule to a
keyed draw. Produced by `models/chk/worked.py` from `models/wld/seed.py` and printed again by
a Go program (`math/rand/v2`, Go 1.25 and Go 1.27): every line equal.

| Seed | Salt | `seed'` | First word of `(siteWork, 0, 0)` |
|-----:|-----:|---------|----------------------------------|
| 0 | 0 | `0000000000000000` | `264d986499a1a1fa` |
| 0 | 1 | `7ab40e090f363a7d` | `9fea3673fd1e605f` |
| 0 | 2 | `52ead7e36ea7fea8` | `73a2087f8eae98d5` |
| 0 | 42 | `97ea87f7e45c00a5` | `1b4989fe2e01283b` |
| 1 | 0 | `5692161d100b05e5` | `3a90ed51beb4e41d` |
| 1 | 1 | `83ec686c1600460a` | `a5d0a8b51be8378f` |
| 1 | 2 | `3ec2d42f3a45cc6e` | `47f38e4d9854e8fd` |
| 1 | 42 | `901883d2427428f4` | `00c073c4d5d6c31a` |
| 42 | 0 | `a759ea27d4727622` | `e3082397f3bc8ba6` |
| 42 | 1 | `c2a6eebdf3976ad0` | `cd5e38f87f0ec634` |
| 42 | 2 | `7bfa87c92aa0cff0` | `5b73a5b88ec53d5a` |
| 42 | 42 | `36cdb0593534a37b` | `472c27560611be0f` |
| 2^64 - 1 | 0 | `b4d055fcf2cbbd7b` | `9cbc3a9d395aed3d` |
| 2^64 - 1 | 1 | `f3759af4852e5904` | `e6b61a2920e28083` |
| 2^64 - 1 | 2 | `ab47721b00f39ddb` | `649e9d647ec06743` |
| 2^64 - 1 | 42 | `eab06b8e3b2074d7` | `71829937eb038fa1` |

The first row is the case of 6.8.2 in which a reseed is the identity (seed 0, salt 0, and
`finalise(0)` is 0): its first word is that of the unreseeded seed 0, the row k1 of
`spec-world.md` 7.1.5 (b).

### 17.6 What a corrupt input does

| Input | Result |
|-------|--------|
| the frame with the last byte of the digest flipped | `ErrCheckpointCorrupt` from `UnmarshalBinary` |
| a frame with format 2 and a wrong digest | `ErrCheckpointVersion` (the version is checked first) |
| a valid frame of a world built from another pack | `ErrCheckpointConfig` from `RestoreWorld`, with both hashes |
| a section 3 whose agent IDs descend | `ErrCheckpointCorrupt`, tag 3 and the offset |
| an estate listing for a lot the estate does not hold | `ErrCheckpointCorrupt` from the cross-check (6.2 step 7) |
| a registered `Stateful` strategy `"market-maker"` and no entry for it | `ErrCheckpointExtension` (7.3.2) |
| `WithSeed(7)` | `ErrOptionContext`, before the configuration is looked at |
| a conserving world altered to give an agent 1 unit of money from nowhere, then re-digested | `ErrCheckpointCorrupt` wrapping `ErrAuditFailed` (6.2.2) |

## 18. Notes settled

For `AGENTS.md`: each open note tagged `CHK` or `ALL`, and the others that name the
checkpoint, is applied, rejected or left open here, with the text to append in the
handoff.

| Note | Decision | Where |
|------|----------|-------|
| N-020 What each spec contributes to a checkpoint | **Applied; close for CHK.** The inventory of 4.2 is the one table; the findings F1 to F11 of 4.5 are the gaps and duplicates the reading found, each with a ruling and an edit. The four things N-020 left open are answered: compaction (9.4: never), the extension-state interface (7), the compatibility policy (9), and the hash (5.1, 6.2). Close it when the edits of section 20 are made | 4, 7, 9 |
| N-051 What the Strategy contract needs from other specs | **Applied for CHK:** the strategy **names** are hashed and the code is not; the name table of an agent record covers the names of external agents, which the hash does not; the directive is state (AGT 15). The code identity is the extension's version (7.3.3). **Leave open** for INT and WLD as before | 7, 4.5 F8 |
| N-008 Weight changes, cohort failure and the estate (closed) | the closed-accounts growth question it passed to CHK is answered: never compacted in version 1 | 9.4 |
| N-019 Housekeeping for the first code change | **Leave open**, with this added: the amendment text of 7.6 for architecture.md 4.3 and 7.7 with a fresh tag; `checkpoint.go` (`Checkpoint`, `RestoreWorld`, `Stateful`, `WithReseed`) and `internal/codec` (21); the glossary terms of 20; the log keys `format` and `bytes`; the committed fixtures per format version (9.2.1) | 7.6, 21 |
| N-075 Mapping the ledger's sentinels | **Leave open**, with this added: `ErrCheckpointVersion`, `ErrCheckpointCorrupt`, `ErrCheckpointConfig`, `ErrCheckpointExtension` and `ErrOptionContext` are root sentinels (13.1) | 13 |
| N-016 Keep the parameter registry true | **Applied:** CHK adds no field, default or bound, so no row (12) | 12 |
| N-017 Words and names | **Applied:** the terms are in the list of 20; the examples are flagged; "consumer" means the library consumer | 20 |
| N-027 No default macro constant is stable | **Applied:** this spec sets no default; the cost model's numbers are placeholders | preamble, 10, 12 |
| N-103 Running several spec writers in parallel | **Applied:** no shared file was edited; proposals are in the handoff with provisional IDs (`CHK-A1` to `A3`, `CHK-D1`, `D2`, `CHK-S1`, `S2`); the models ran in Docker with no bytecode. **Two slips** recorded in 16 | 15, 16 |
| N-009 What the world and tick phases must decide | WLD's; **applied here** for the point that touches CHK: the construction phase and the pending totals at a checkpoint (4.4, 6.7.2) | 4.4 |
| N-056 / N-074 (closed) | the construction phase and its bucket: a world restored from a checkpoint taken before the first step stays in construction (6.7.2); edits to WLD 5.5.3 and 12.1 and LED 7.4.1 and 11 | 6.7, 20 |
| N-072 The construction bucket as an event | **Leave open for OBS** (decided there: `BoundaryFlow` with `Tick` -1 on the first step); this spec adds that a world restored in construction emits it at its first step like the original, and one restored after it emits none (6.7.2) | 6.7 |
| N-012 Events, metrics and log keys to define | **Leave open**, with this added: `WorldRestored` (OBS fixes the fields), the log keys `format` and `bytes`, no metric (14) | 14 |
| N-091 The world slots INT must expose | **Leave open for INT**, with this added: every slot's state is saved by its owner and listed in 4.2 (REG 11, MON 14, POP 11, TRN 14, RES 15, EXT 16, PRO 15) | 4.2 |
| N-045, N-065 Dampers that act silently | **Leave open**, with this added: the diagnostic runs that decide when a `Warn` repeats are saved so a resumed run logs as the original (4.1.2, FR-CHK-043) | 4.1.2 |
| N-111 A view takes draws | **Leave open**, with this added: a restore takes no draw (8.4.2); the stream resumes from the saved state | 8.4 |
| N-084, N-092 (overlays and scenarios) | **Leave open for REF/STY**, with this added: X5, S60 and S72 have their checkpoint legs defined (8.2.3) | 8.2.3 |

New notes are in the handoff.

## 19. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. This spec has
no default or bound of its own.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CHK-001 | `World.Checkpoint` MUST return a `Checkpoint` holding the canonical encoding of the complete state of the world at the boundary, and MUST change no state, draw no random number and emit no event (3.1, 3.2). |
| FR-CHK-002 | `Checkpoint` MUST return `ErrWorldFailed` on a failed world and `ErrInStep` inside a step, MUST return `ErrCheckpointExtension` wrapping the error of an extension whose `SaveState` failed, and MUST succeed otherwise on a healthy world between ticks, including before the first `Step` (3.2). |
| FR-CHK-003 | `MarshalBinary` and `AppendBinary` MUST return a copy, `UnmarshalBinary` MUST copy its argument, and a `Checkpoint` MUST be immutable, readable from other goroutines and usable for any number of restores (3.3). |
| FR-CHK-004 | Equal states MUST give equal bytes: two checkpoints of one world at one boundary, and checkpoints of two worlds built from the same configuration, seed and sequence of calls, MUST be byte-equal. |
| FR-CHK-005 | If `RestoreWorld` without `WithReseed` accepts a `Checkpoint`, a `Checkpoint()` of the world it returns, taken at once, MUST be byte-equal to it (6.6.3). |
| FR-CHK-006 | `UnmarshalBinary` MUST check the magic, then the format (`ErrCheckpointVersion` before the digest is looked at), then the digest and the header ranges (`ErrCheckpointCorrupt`), and `Info` MUST report the format, the configuration hash, the tick, the effective seed and the length (5.1, 6.2 step 1), with no error result; a checkpoint MUST hold no state of the random source (3.1.2). |
| FR-CHK-007 | A checkpoint MUST NOT contain the configuration, the options, an observer, the logger, the code of an extension, or an event (3.5). |
| FR-CHK-008 | The sections MUST be written in the tag order of 4.2, the set present MUST be a function of the configuration, an unexpected or missing tag MUST be corrupt, and every section MUST be consumed exactly (4.2.1, 5.3). |
| FR-CHK-009 | The frame MUST be the magic, the format, the hash version, the configuration hash, the tick, the seed, the sections and a trailer of the SHA-256 of every byte before it, and nothing else (5.1, 5.4). |
| FR-CHK-010 | An unsigned integer MUST be written as a minimal LEB128 varint of at most 10 bytes, and the decoder MUST reject a non-minimal, an overflowing and a truncated one (R1). |
| FR-CHK-011 | A signed integer MUST be written as the zigzag of itself and every `int64` MUST round-trip (R2). |
| FR-CHK-012 | A boolean MUST be one byte of 0 or 1, and an enumeration or an index MUST be checked against its bound, anything else being corrupt (R3, R6). |
| FR-CHK-013 | A count or length read from the input MUST be at most the bytes remaining and at most the bound of its type, and a count that the configuration implies MUST NOT be written (R5, R8, 4.1.3). |
| FR-CHK-014 | A sequence keyed by an ID MUST be strictly ascending, a sparse row equal to its default MUST be corrupt, and no padding, reserved field or trailing byte MUST exist (R9 to R11). |
| FR-CHK-015 | The encoding MUST contain no map, float, pointer, function, timestamp or host identity and MUST be byte-identical on every platform, checked on at least two architectures (R12, NFR-WLD-004). |
| FR-CHK-016 | `UnmarshalBinary` and `RestoreWorld` MUST never panic on any input and MUST NOT allocate more than a constant times the length of the input plus what the configuration implies (6.6.1). |
| FR-CHK-017 | `RestoreWorld` MUST return `(nil, err)` and emit no event on any failure, and MUST NOT return a partly built world (6.1.1). |
| FR-CHK-018 | `RestoreWorld` MUST validate in the order of 6.2: options, configuration and hash, extensions, sections, cross-checks, install, audit, extension state, report. |
| FR-CHK-019 | The hash MUST be compared using the encoding version in the header, a mismatch MUST be `ErrCheckpointConfig` carrying both hashes, and an invalid configuration MUST be the `ValidationError` of `NewWorld` (6.2 step 3, 6.3.1). |
| FR-CHK-020 | `WithSeed` on a restore and `WithReseed` on `NewWorld` MUST be `ErrOptionContext`, `WithTargetSpread` MUST be accepted and hashed, and the other options MUST behave as in `NewWorld` (6.4). |
| FR-CHK-021 | A saved extension name that is not registered, a registered `Stateful` extension with no entry, an entry whose kind or `Stateful` status differs, and a `LoadState` error MUST each be `ErrCheckpointExtension` (7.3.2). |
| FR-CHK-022 | Every strategy name of the table of section 3, including that of an external agent, MUST be registered at restore, or `ErrCheckpointExtension` joined with `ErrUnknownStrategy` MUST be returned (6.2 step 4). |
| FR-CHK-023 | Restore MUST cross-check that each open Agent account is the account of exactly one agent, each estate listing is a lot the estate holds (rows in ascending key order without repeats, each price from 1 to `MaxPrice`), each shipment's Transit account and each contract's Escrow account exists and holds what its record says, each trainee has its trainer and the converse, each flow carry is for an open account, each journey names a live agent, and each reference to an agent is to a live one except the shipper, traveller and consignee of a shipment and the payer of a contract, which need only be below the next agent ID (4.2.3, 6.2 step 7). |
| FR-CHK-024 | Restore MUST run the ledger's audit and every check of a module that is on, with the boundary baseline taken as the restored balance less the pending totals (and, for a world restored in construction, the saved bucket as the construction term), whether or not `WithAudit` was given, and a failure MUST be `ErrCheckpointCorrupt` wrapping `ErrAuditFailed` (6.2.2). |
| FR-CHK-025 | A restored world MUST be at the saved tick with the saved seed and next agent ID, MUST report `Err()` nil, and MUST share no state with the world that was checkpointed (6.7.1). |
| FR-CHK-082 | A restored world's `Config()` MUST be the effective configuration of the restore and its `ConfigHash()` MUST equal `Info().ConfigHash` (6.7.1). |
| FR-CHK-026 | A world checkpointed before its first `Step` MUST be restored in its construction phase with its bucket, and a world checkpointed after it MUST NOT be, with the bucket neither saved nor emitted (4.4, 6.7.2). |
| FR-CHK-027 | A successful restore MUST emit `WorldRestored` as its first event, and MUST emit no `AgentSpawned`, `AgentBankrupted` or `BoundaryFlow` for the past (6.7.3). |
| FR-CHK-028 | A restore MUST log the configuration's warnings once each at `Warn` and one `Info` "world restored" with the keys of 14.2, and `Checkpoint` MUST log nothing at `Info` or above (6.7.4). |
| FR-CHK-029 | `LoadState` of the extensions MUST be called only after every other step has succeeded, once per extension, in (kind, name) order (6.2.1). |
| FR-CHK-030 | `Stateful` MUST be optional, MUST have the two methods of 7.1, and MUST apply to `Strategy`, `Spawner`, `Clearing` and `Policy` and never to `Observer` (7.1). |
| FR-CHK-031 | Section 34 MUST hold, for every registered `Stateful` extension used or not, one entry (kind, name, version, bytes) in (kind, name) order, and none for one that is not `Stateful` (7.3.1). |
| FR-CHK-032 | `SaveState` MUST be called once per extension per `Checkpoint` and `LoadState` once per restore, only between ticks, never inside a step (7.2 item 1); an error of `SaveState` fails the `Checkpoint` with `ErrCheckpointExtension` wrapping it (3.2.2). |
| FR-CHK-033 | The world MUST copy the bytes `SaveState` returns before `Checkpoint` returns, and MUST pass `LoadState` a slice valid for the call only (7.2 items 2 and 3). |
| FR-CHK-034 | The world MUST pass the version it read to `LoadState` and compare it with nothing itself (7.4.1). |
| FR-CHK-035 | `Directed` MUST implement `Stateful` by delegating to its fallback when that is `Stateful` and by saving nothing otherwise (7.5). |
| FR-CHK-036 | A test MUST show that an extension that implements `Stateful` and saves its state continues identically after a restore, and that one that keeps the same state hidden does not and is not detected by the library, so that the consumer's round-trip test is the guard (7.3.3). |
| FR-CHK-037 | The library's own extensions (`DefaultStrategy`, `DefaultSpawner`, `AverageClearing`, `strategy.Merchant`) MUST keep no state and have no entry (7.5); `Directed` has an entry, possibly empty, wherever it is registered, and the built-in `directed` of INT is one. |
| FR-CHK-038 | A `Checkpoint` of a world with a `Stateful` extension taken twice at one boundary MUST give equal bytes (ADR 0002 test (b), 8.2.1). |
| FR-CHK-040 | Each section MUST contain exactly the state of 4.2 in the order given there, and no spec's saved state MUST be omitted or written twice (4.2, 4.5). |
| FR-CHK-041 | A per-agent field hosted in the agent (`Privation`, `RoleSince`, `DeathCarry`, `OwnUse`, `OwnUseJobs`, the journey, the directive, the stores and the membership) MUST be saved once, in section 3, each only with its module, and its module's section MUST list its own state only (4.5 F3). |
| FR-CHK-042 | Section 4 MUST hold, for every market, the whole report of the last clearing (price, age, volume, unmatched units, best unmatched bid and ask with their flags) and the window oldest first with no position (4.5 F1, F2). |
| FR-CHK-043 | The diagnostic runs that decide when a `Warn` repeats MUST be saved, as sparse rows, so that a resumed run logs as the original did (4.1.2). |
| FR-CHK-044 | The pending totals that calls between ticks added since the last P8.6, and the relocation totals, MUST be saved, and the construction bucket MUST be saved if and only if the world is in its construction phase (4.4). |
| FR-CHK-045 | A test MUST list every field of the live state types that is neither saved nor named in 4.3 as derived or scratch, and MUST fail on one that is unnamed. |
| FR-CHK-046 | A closed account MUST be saved as its kind and a closed flag, account IDs MUST be preserved, and no compaction MUST be done (9.4). |
| FR-CHK-047 | No reservation or earmark MUST be in a checkpoint, and every saved lot MUST have `Reserved` 0 (4.4). |
| FR-CHK-048 | The estate pseudo-agents, the counterparties and the treasury pseudo-agents MUST have no row in section 3 and MUST be rebuilt from the configuration and the account list (4.2.2). |
| FR-CHK-049 | The region of an open account, the owner reference of an account and the agent's reference to its account MUST agree, and the first accounts MUST be those of 5.2 of `spec-world.md` in the order given there. |
| FR-CHK-050 | A restored world MUST continue exactly as the uninterrupted one: equal events, snapshots, queries, errors and byte-equal checkpoints at every later boundary, for a range of seeds and for the checkpoint ticks of 8.2.1 (8.1.1). |
| FR-CHK-051 | A restore MUST take no random word: the next word of the restored world MUST be the next word of the original (8.4.2); the next word of a stream is a function of its key, so the requirement holds trivially. |
| FR-CHK-052 | A restored world MUST answer every read between ticks (`Slots`, `Agent`, `Market`, `Region`, `Tick`, `Audit`, `Snapshot`) as the original did at that boundary (11). |
| FR-CHK-053 | Every field of a `Snapshot` MUST be a function of the saved state and the configuration, so that a snapshot of a restored world equals that of the original at the same tick (OBS 5.2.1, 11). |
| FR-CHK-054 | A run split into several `Run` calls, with a restore in each gap, MUST equal an uninterrupted run (8.2.2). |
| FR-CHK-055 | A checkpoint of a restored world MUST restore again to a world equal to the original (8.2.2). |
| FR-CHK-056 | The digest of the checkpoint of the reference chain at a stated tick MUST be equal on two architectures (8.3). |
| FR-CHK-057 | The checkpoint legs of X5, S60 and S72 MUST pass (8.2.3). |
| FR-CHK-060 | Before 1.0 a checkpoint of another format version MUST be `ErrCheckpointVersion`, reporting the version read and the version supported (9.1). |
| FR-CHK-061 | From 1.0 a library MUST read every format version any v1 release wrote and MUST NOT read a newer one (9.2.1). |
| FR-CHK-062 | Each released format version MUST have a committed fixture checkpoint that is restored and continued in a test (9.2.1 item 3). |
| FR-CHK-063 | The header MUST record the version of the configuration encoding, and from 1.0 the library MUST retain the encoder of every version it wrote (9.2.1 item 4). |
| FR-CHK-064 | A change of the digest of the reference chain's checkpoint MUST be accompanied by the explanation of testing.md 7.2 and, if the shape of state changed, by a format version (9.3). |
| FR-CHK-070 | The reseed MUST be `seed' = finalise(seed xor finalise(salt))` on the seed of the header, with the vectors of 17.5; a salt of 0 MUST be a reseed (it is the identity only for the seed 0); and the restored world MUST install no random state and take no word (6.8.2). |
| FR-CHK-071 | A reseed MUST be reported by `WorldRestored` (`Reseeded`, `Salt`), MUST make `seed'` the seed of the world and of the header of its later checkpoints, and MUST take no random word (6.8.3). |
| FR-CHK-072 | Two restores of one checkpoint with one salt MUST continue identically, and with two salts MUST differ (CHK-S2). |
| FR-CHK-080 | `WorldRestored` MUST be emitted, and its snapshot built, only when an observer was supplied, and a failed restore MUST emit nothing (6.7.3, 14.1). |
| FR-CHK-081 | `Checkpoint` MUST emit no event and MUST log only a `Debug` line guarded by `Enabled` (14.1, 14.2). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-CHK-001 | With the default logger and observer, a restore MUST construct no event and no snapshot, and neither `Checkpoint` nor `RestoreWorld` MUST add to the allocation baseline of a `Step` (observability-and-logging.md 4.5, 8.2). |
| NFR-CHK-002 | The size of a checkpoint MUST depend on the number of agents, their lots and the effective windows, and MUST NOT depend on the weights (NFR-SCL-004); the first consumer's world SHOULD be under 5 MB (a placeholder: the model gives 2.1 MB, 10.2). |
| NFR-CHK-003 | Fuzz tests (testing.md 7.4) of `UnmarshalBinary`, of `RestoreWorld` over a small configuration, and of each section's decoder MUST assert that nothing panics, that the allocation is bounded, and that every input accepted by a restore without `WithReseed` is a fixed point of restore-then-save, and every crash input MUST be committed with a `Fix:` comment (testing.md 7.5). |
| NFR-CHK-004 | A benchmark with `b.ReportAllocs()` of `Checkpoint` and `RestoreWorld` over the reference chain MUST report the bytes, the allocations and the share of closed-account slots (testing.md 8.1; 9.4.4). |
| NFR-CHK-005 | The encoder, the decoder and the digest MUST use the standard library only (`encoding/binary`, `crypto/sha256`), MUST NOT call `time.Now`, start a goroutine or keep package-level mutable state, and MUST NOT use the random source (architecture.md 5.1, 7.1, 7.3, 12.2). |

## 20. Open questions

1. **The shape of `Stateful`.** `SaveState() (version, state, err)` allocates a slice
   per extension per checkpoint; an `AppendState(dst)` form would not. The cost is
   one allocation per stateful extension per save, outside every tick; next step: the
   first code, with a benchmark.
2. **An in-memory clone.** `RestoreWorld(cfg, w.Checkpoint())` encodes and decodes
   (10.3). A direct copy would be faster for a harness that branches thousands of
   times and would be a second code path to keep equal to the first. Next step: EXP's
   first branching experiment, with the measure.
3. **A reseed between ticks.** `WithReseed` exists only at a restore (6.8). A consumer
   that wants a different future from a live world checkpoints and restores with a salt.
   If that proves awkward, INT adds a call that reseeds a live world between ticks;
   it would be a call in the sequence of calls like `AddAgent`. Next step: INT.
4. **A starting wage that latches** (F5). The ruling here is a pure function of the
   rings; WEL, PRO or VIA may mean a latch. Next step: those specs say which; this spec
   changes by one row (4.5) and one bit per region in the Welfare section.
5. **The fixture for each format version** (9.2.1 item 3) is a checkpoint of a world
   of about 2 MB; fixtures for many versions grow the repository. Recommendation: use
   the single-region pack `one` (a few hundred agents) for fixtures and keep `five` for
   the digests of the golden file. Next step: with the first release.
6. **Per-section sizes in `Info`.** A tool that says which section is big would help to
   see the welfare ring (10.2). It would be a field `Sections []SectionInfo` of
   (tag, length). Cheap; left to the code.
7. **Authenticity** is the consumer's (5.4.2). A consumer that restores bytes from a
   player's save file has the trust problem of any save file; whether the library
   should offer an optional keyed digest (an HMAC) was not decided and is not needed
   for a server's own checkpoints.
8. **The claims list** was empty in the first draft of format 1 (`spec-ledger.md` 10). **Closed:**
   Credit (`spec-credit.md`) fills it (4.2, tag 2), and with nothing released it is part of
   format 1 (9.3), not a new version.
9. **Closed slots** (9.4.4): the trigger is a quarter of the bytes; no consumer has
   shown one.
10. **Streaming.** The digest covers the whole, so a restore needs all the bytes in
    memory. A world of 86 MB (10.2, case D) is the largest the specs describe; if a
    consumer needs a stream, a block-wise digest would be a format version.

### Edits owed to other specs

One consolidated table: every file, section and change. None alters a result that a
test could depend on (nothing has run). The supervisor applies them; the handoff is
derived from this table.

| File | Section | Change |
|------|---------|--------|
| `spec-world.md` | 5.5.3, 12.2, 8.4.1, 13 (CHK row), FR-WLD-100, FR-WLD-101 | applied (ADR 0008; plan 2-1 Phase 1): a world restored from a checkpoint taken **before its first `Step`** stays in construction with its bucket (4.4, 6.7.2); one taken after is not in construction; 12 table: the tick and the effective seed are in the header, not in section 1; FR-WLD-101 reads "a world restored from a checkpoint taken after its first `Step` MUST NOT be in construction, and one taken before it MUST be"; FR-WLD-100 reads the effective seed, the next agent ID and the construction flag, with the tick and the seed written in the header, the bucket and the pending totals being the ledger's (4.2 tag 2) |
| `spec-world.md` | 4.1, 4.3, 15.1 | `WithReseed(salt)` in the options table (restore only, not hashed, in the sequence of calls) and `ErrOptionContext` (`WithSeed` on a restore, `WithReseed` on `NewWorld`) |
| `spec-world.md` | 12.3, open question 16 | 12.3 adopted: `WithSeed` is refused on a restore, the header's seed is the effective seed and is read back; question 16 answered: the pending totals are saved (4.4) |
| `spec-world.md` | 4.4, FR-WLD-020, FR-WLD-021 | 4.4 "An option never changes what is saved in a checkpoint" gains "except `WithReseed`, the one restore-only option, which is a call in the sequence of calls like `AddAgent` and sets the effective seed by the rule of `spec-checkpoint.md` 6.8.2"; FR-WLD-020 "`WithSeed`, `WithLogger`, `WithObserver` and `WithAudit` MUST NOT change the configuration hash" gains "and neither MUST `WithReseed`"; FR-WLD-021 "every option that can change a result MUST be a datum ... or a registered name" gains "or, for `WithReseed`, a call of the sequence of calls (8.1.1)" |
| `spec-world.md` | 3.3.3, 12.2 | 3.3.3 `ErrAuditFailed` row: add "and a restore always audits once; its failure is `ErrCheckpointCorrupt` wrapping `ErrAuditFailed`"; 12.2: "the restore runs the audit whether or not `WithAudit` was given (`spec-checkpoint.md` 6.2.2)" |
| `doc/rules/architecture.md` | 7.3 | applied by ADR 0008: a checkpoint saves no random state, a restore installs nothing, and `WithReseed` replaces the seed (7.3, last bullet) |
| `spec-regions.md` | 11, FR-REG-042, FR-REG-052 | 11 "Not saved (derived): ... the relocation totals (6.4.1, like the flow totals)" becomes "the relocation totals of finished ticks; the relocation totals that a call between ticks added since the last tick's end are pending totals, saved with the ledger's (`spec-checkpoint.md` 4.2 tag 2)"; FR-REG-042 and FR-REG-052: "and nothing else of this spec" becomes "and nothing else of this spec's own state (the pending relocation totals are the ledger's section)" |
| `spec-ledger.md` | FR-LED-029, FR-LED-054, 5.4.2 or 7.4 (wear units) | FR-LED-029 "MUST NOT contain reservations, flow totals, the construction bucket or expiry hints" becomes "... flow totals of finished ticks, the construction bucket once the world has stepped or expiry hints, and MUST contain the pending totals and, in construction, the bucket"; FR-LED-054 the same for "flow totals are not saved"; 7.4 names the wear units, the flow totals and the relocation totals together as the "pending totals" that a call between ticks adds to |
| `spec-integration.md` | 13, 13.2 | 13: the rumours are saved in section 22 of the checkpoint (`spec-checkpoint.md` 4.2), in ascending agent ID, rows only for an external agent that has had a view; 13.2: "A world restored without them ... runs identically" becomes "The rumours are mandatory in a checkpoint: a restore without them would answer `Agent` differently (FR-CHK-052); they do not affect a result" |
| `doc/adr/0002-persistence-by-checkpoint.md` | decision 3 | "A checkpoint is plain data" gains "held as one canonical byte string; the plain data is the encoding, and no struct of state is exported (`spec-checkpoint.md` 3.1.1)" |
| `spec-ledger.md` | 3.3, open question 6 | closed accounts are never compacted in version 1 and are saved as a kind and a closed flag (9.4); the trigger of 9.4.4 |
| `spec-ledger.md` | 7.4.1, 11 | the bucket is saved while the world is in its construction phase and not otherwise; the pending flow, wear and relocation totals are saved (4.4) |
| `spec-ledger.md` | 9.1 | after a restore the baseline of the boundary identity is the restored balance **less the pending totals** (6.2 step 10) |
| `spec-clearing.md` | 7.1, and a new section "State contributed to a checkpoint" | the market carries the published price, the age, the volume, the unmatched units and the best unmatched bid and ask of the last clearing, besides the window (4.5 F1); requirement that it is saved |
| `spec-beliefs.md` | 13 | the price window is saved oldest first with a count and no write position (F2); the market record is shared with CLR |
| `spec-agents.md` | 15, 3.2, open questions 4 and 9 | the agent record of section 3 of this spec is the list: `Privation`, `RoleSince`, `DeathCarry`, `OwnUse`, `OwnUseJobs`, the journey and the directive are saved once there (F3); the welfare ring is WEL's (F4); open question 4 answered by the version of `Stateful` (7.3.3); the strategy name table covers external agents (F8) |
| `spec-agents.md` | 10.3 | `Directed` implements `Stateful` by delegation (7.5) |
| `spec-spawner.md` | 14 | `Privation` is saved in the agent record (F3); the section lists the renewal carry and the runs |
| `spec-population.md` | 11 | `DeathCarry` is saved in the agent record (F3) |
| `spec-subsistence.md` | 15 | `OwnUse` and `OwnUseJobs` are saved in the agent record (F3) |
| `spec-migration.md` | 14 | the journey is saved in the agent record (F3) |
| `spec-welfare-and-wages.md` | 13, 4.3 | the observed wage and the starting wage are functions of the rings (F5); no latch |
| `spec-production.md` | 9.3, 15 | "used until the first working tick is observed" reads "used whenever the pooled figure is unknown" (F5); the count of pending quantities is implied by the recipe |
| `spec-viability.md` | 8.2.5 | the same sentence (F5) |
| `spec-config.md` | 8.1, 8.2 | the version of the configuration encoding is recorded in the checkpoint header; a v1 library keeps the encoder of every version it wrote (9.2.1 item 4); a restore compares the hash using the header's version |
| `spec-integration.md` | 13, 6.2, 9.2 | check only: `Checkpoint` is a value (3.1), the section is tag 22 (4.2), `RestoreWorld` is the name |
| `spec-observability.md` | 4.4.5, 5.2.4 | check only: `WorldRestored` first, after validation, nothing on failure (6.7.3); its fields are OBS's (`Format`, `Salt`; the seed and totals in the snapshot header); the bucket of a world restored in construction is emitted at its first step, as the original would have (6.7.2) |
| `spec-reference-chain.md` | 7.4 (X5), 8.5 item 6, 12 | X5 compares the checkpoints byte for byte (8.2.1); the digest is the trailer (8.3); section 12 "None" gains "checkpoint legs of X5, S60, S72 are 8.2.3" |
| `spec-stability.md` | 4, 5, 8, 13 | merge CHK-A1 to A3, CHK-D1, D2, CHK-S1, S2; the CHK hook row goes; 13 "None" stays |
| `spec-scale.md` | 6.4 | the sizes of 10.2 and that the welfare ring dominates |
| each spec with a checkpoint section | the section | optional: "the layout and order are those of `spec-checkpoint.md` 4.2" |
| `doc/rules/architecture.md` | 4.3, 7.7, 6.2, 3.2 | the text of 7.6 and `WithReseed`; the glossary terms below; a fresh read-receipt tag |
| `doc/rules/observability-and-logging.md` | 3.3.2, 4.3 | the keys `format` and `bytes`; the `WorldRestored` row (when emitted; fields OBS's); a fresh read-receipt tag |
| `doc/adr/0002-persistence-by-checkpoint.md` | decision 5, open points 1 to 5, status | the value implements `BinaryMarshaler`, `BinaryAppender` and `BinaryUnmarshaler`; open point 1: `Stateful` (7); 2: hand-written varints (5.2.2); 3: 9; 4: between ticks only; 5: no events (3.5.3); status to the human |
| `specs/README.md` | index, reading order, area row `CHK`, decisions, glossary | CHK exists; the terms below; the decisions: checkpoint is an opaque value, varints, reseed, never compact |
| `doc/spec-roadmap.md` | the CHK row (line 184), tier tables, existing-specs table | `exists` |
| `doc/documentation_index.md` | rows | `spec-checkpoint.md` and `models/chk/` |
| `doc/specs/models/README.md` | contents | a row for `chk/` |
| `doc/notes.md` | the open notes of 18 | close N-020; update N-051, N-019, N-075, N-016, N-017, N-027, N-103, N-012, N-072; add the new notes of the handoff; the index; a fresh tag |
| `AGENTS.md` | layout | `checkpoint.go` and `internal/codec` when first needed (N-019) |
| the module specs of the later stage | the sentence "its tag is the supervisor's" or "assigned at the integration" | the tags are 23 Settlements, 24 Employment, 25 Tenure, 26 Credit, 27 Facilities, 28 Risk, 29 Insurance, 30 Fiscal, 31 Institutions, 32 Policy, 33 Technology and 34 Extensions (4.2); Integration stays 22; and no format bump (9.3) |

**Glossary terms** (to add to `doc.go` and the README list): checkpoint, restore,
format version, section, frame, header, trailer, digest (of a checkpoint), canonical
encoding, one spelling, sparse row, pending totals, construction flag, extension state,
`Stateful`, entry (of an extension), reseed, salt, branch (of a checkpoint), fixture
(of a format).

## 21. Where this lives in the code

| Concern | Home |
|---------|------|
| `Checkpoint`, `CheckpointInfo`, `World.Checkpoint`, `RestoreWorld`, the frame, the header, the section order and tags, the validation order, the cross-checks | root package, `checkpoint.go` |
| `Stateful` | root package, `checkpoint.go` (or `strategy.go` beside the extension interfaces; one file, no new package) |
| `WithReseed` and `ErrOptionContext` beside `WithSeed` | root package, `option.go`, `errors.go` |
| The sentinels `ErrCheckpointVersion`, `ErrCheckpointCorrupt`, `ErrCheckpointConfig`, `ErrCheckpointExtension` | root package, `errors.go` |
| `WorldRestored` | root package, `event.go` |
| The primitives U, I, B, raw and bytes, with the checks of R1 to R13 | `internal/codec` (stdlib only: `encoding/binary`; it does not know what a commodity or an agent is, architecture.md 2.9) |
| The save and load of each module's state | beside the module: a method pair in the root file of its concept (`world.go`, `agent.go`, `market.go`, `resources.go`, ...), and in `internal/ledger` for the ledger, each over `internal/codec` |
| The digest | root package, `crypto/sha256` |
| The reseed rule (`seed' = finalise(seed xor finalise(salt))`, a pure function) | root package, `random.go`; the finaliser is the one helper of `internal/ordering` (`spec-world.md` 23); built in Stage 0, the option in Stage 2 |
| Fixtures per format version | `testdata/checkpoints/` |
| The models | `doc/specs/models/chk/` |
