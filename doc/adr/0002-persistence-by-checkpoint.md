# 0002. State is persisted by the consumer, from a checkpoint the library produces

- Status: Proposed
- Date: 2026-10-09
- Supersedes: none

## Context

The library is a pure library: no server, transport or persistence layer
(`AGENTS.md`, `doc/rules/architecture.md` 1.1). A game or a server that embeds
it still needs to save a world and resume it, and the economy runs for long
simulated times, so losing it on a restart is not acceptable.

Two things already exist or are required. A `Snapshot` is a deep-copied,
read-only summary for observation (architecture.md 12.4); it is not meant to
resume a run. And a run is deterministic: the same seed, configuration and
sequence of calls give a bit-identical result (architecture.md 7.4). This
decision says how a run is resumed, and `spec-checkpoint.md` works it out.

Persistence is expensive to retrofit. If state is hidden in closures, package
variables or an extension's private fields, no checkpoint can capture it.

## Decision

1. **The library never persists.** It does not write files, open connections or
   choose a store. The consumer stores whatever the library gives it, as it
   already does for logs and metrics.

2. **Snapshot and checkpoint are different things.** A snapshot is a summary for
   observation. A **checkpoint** is the complete state needed to resume a run
   exactly.

3. **A checkpoint is plain data.** It is held as one canonical byte string: the
   plain data is the encoding, and no struct of state is exported
   (`spec-checkpoint.md` 3.1.1). It is a deep copy with no pointers, functions
   or maps in the encoded form. It contains the tick, the state of the random
   source, the ledger (accounts, balance sheets, obligations), every agent
   (inventories, money, beliefs, welfare history, tool wear, trainee queues),
   regional pools and market histories, everything in flight (goods in transit,
   carriage contracts, undelivered rumours), the state of each enabled module,
   and a format version and a hash of the configuration. It excludes the
   configuration itself, observers and the logger.

4. **Restore is validated.** A restore function takes the configuration, the
   checkpoint and options, and returns an error (a sentinel the caller can match
   with `errors.Is`) if the format version is unsupported or the configuration
   hash does not match. A restored world continues exactly as if it had never
   stopped.

5. **One canonical encoding.** The library offers a single deterministic binary
   encoding of a checkpoint (`encoding.BinaryMarshaler`, `BinaryAppender` and
   `BinaryUnmarshaler`, standard library only: `MarshalBinary`, `AppendBinary`
   and `UnmarshalBinary`, `spec-checkpoint.md` 3.1). Equal states produce equal
   bytes: collections are written in sorted order, never in map order. Consumers
   that want JSON, protobuf or a database row convert the plain data themselves.
   There are no struct tags in the core (architecture.md 12.6).

6. **Persistable by design.** All simulation state lives in plain structs owned
   by the world. No state is held in closures, package variables or goroutines.
   This is checked in review and by the checkpoint tests, even before the
   checkpoint feature ships.

7. **Extension state is explicit.** The default `Strategy`, `Clearing`,
   `Spawner` and `Policy` implementations keep no state of their own or keep it
   in the world. A consumer implementation that keeps state must either be
   stateless or implement an optional pair of methods that save and load its
   state as opaque bytes, which the checkpoint stores. The consumer is
   responsible for that state being deterministic. The interface is the optional
   `Stateful` pair (`spec-checkpoint.md` 7.1). The registration of an extension
   (the code) is not saved: it is checked by name against the configuration hash
   and against the saved entries (`spec-world.md` 12.1, `spec-checkpoint.md`
   3.5.2, 6.2).

8. **Versioning is strict at first.** A checkpoint restores only into a library
   version that supports its format version. The format version changes whenever
   the shape of the state changes. The library does not migrate checkpoints
   between versions before 1.0. From 1.0 a library reads every format that a
   v1.x release wrote and writes only its current one (`spec-checkpoint.md`
   9.2).

9. **Replay is a debugging aid, not persistence.** Because runs are
   deterministic, a log of the seed, configuration and the consumer's calls can
   reproduce a run, and tests use this. It is not a save mechanism: restoring by
   replay is slow for a long world and breaks when behaviour changes between
   library versions.

## Consequences

- `spec-checkpoint.md` is core, not a candidate to defer. The ledger and every
  module spec must say what state they contribute to a checkpoint.
- The architecture rules needed amending: the public API list and the extension
  rules (architecture.md 4.3 and 7.7) must mention checkpoint, restore and
  optional extension state. The extension-state part is done (`Stateful`,
  architecture.md 4.3 and 7.7, from `spec-checkpoint.md` 7.6); the rest of the
  public API list follows with the code, in the same pull request, with a fresh
  read-receipt tag.
- Observability needs an event for a restored world, and a lifecycle log line
  at `Info` (`doc/rules/observability-and-logging.md` 4.3, 3.2).
- A decoder reads external bytes, so it needs a fuzz test
  (`doc/rules/testing.md` 7.4).
- Required tests: (a) run N ticks, checkpoint, restore and run M more, and the
  result is bit-identical to an uninterrupted N+M run, over a range of seeds;
  (b) checkpointing the same state twice gives identical bytes; (c) a restore
  with a mismatched configuration or version fails with the right error.
- The cost is a second representation of every piece of state, and a format
  that must be kept in step with the model. That is accepted.

## Alternatives considered

- **A persistence interface in the library** (a `Store` the world saves to).
  Rejected: it is a persistence layer, which the architecture forbids, and it
  pushes storage choices into the core.
- **Replay only** (store the seed and the log). Rejected as the main
  mechanism for the reasons in decision 9.
- **JSON with struct tags, or `encoding/gob`.** Rejected: tags are banned in
  the core, and neither gives a canonical byte form that stays stable across
  versions.
- **Leave it to the consumer entirely.** Rejected: the consumer cannot reach
  private state, and a checkpoint is only correct if the library produces it.

## Open points

All five are answered (2026-10-11), by `spec-checkpoint.md` and `spec-world.md`:

1. The optional extension-state interface: names, signatures, and whether a
   world can detect an extension that has state but does not implement it.
   **Answered:** the optional `Stateful` pair `SaveState() (version uint32,
   state []byte, err error)` and `LoadState(version, state)`
   (`spec-checkpoint.md` 7.1); the world detects a missing or surplus entry by
   name (7.3, 6.2).
2. Whether the canonical encoding is hand-written with `encoding/binary` or
   built on another standard library facility, judged by size and speed on a
   large world. **Answered:** hand-written varints, chosen by size: 2.1 MB
   against 13.6 MB as fixed integers for the first consumer's world
   (`spec-checkpoint.md` 5.2.2, 10.2).
3. The compatibility policy after 1.0. **Answered** for the v1 series: a library
   reads every v1.x format and writes only the current one
   (`spec-checkpoint.md` 9.2.1).
4. Whether a checkpoint can be taken between ticks only, or also in the middle
   of a step. The default is between ticks only (architecture.md 12.3).
   **Answered:** between ticks only, on a healthy world (`spec-checkpoint.md`
   3.2, `spec-world.md` 8.4).
5. Whether the checkpoint includes a copy of recent events, so a restored
   consumer can rebuild its derived state. **Answered:** no events; a restore
   emits `WorldRestored` with the snapshot at the restored boundary
   (`spec-checkpoint.md` 3.5.3, 6.7.3).

## Related

- `AGENTS.md`, "Pure library" and "Deterministic"
- `doc/rules/architecture.md` 7.4, 12.3, 12.4, 12.6
- `doc/adr/0001-macro-economy-as-replaceable-layers.md`
- `doc/roadmap.md`, [spec-checkpoint.md](../specs/spec-checkpoint.md) (CHK) and
  [spec-world.md](../specs/spec-world.md) (WLD)
