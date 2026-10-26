# 0007. A module is one root file that owns every one of its touch points

- Status: Accepted
- Date: 2026-10-12
- Supersedes: none

## Context

The specs fix a tick of 69 positions (`spec-world.md` 6), 41 of them belonging
to the 24 optional modules of `Config.Modules` (`spec-config.md` 3.2). A module
touches the world in more places than its positions: it fills a group of the
`AgentView` and checks a slice of the `TradePlan` (`spec-agents.md` 6.3, 6.4,
where about twelve view groups and eight plan slices are a module's), it adds a
step to the removal of an agent (`spec-agents.md` 3.6, steps 2b to 2g) and a
`region_bound` refusal to a relocation (`spec-regions.md` 6.5.2), it contributes
a section or hosted fields to the checkpoint (`spec-checkpoint.md` 4.2, 4.5),
rows to the snapshot (`spec-observability.md` 5), events, sentinels, bounds and
configuration validation rules. The removal cascade runs from two entry points,
P7.1 and the between-ticks calls of `spec-integration.md` 4.

Two rules already shape the answer. The root package is flat, with one file per
concept and no sub-package per module (`doc/rules/architecture.md` 2.6, 2.7,
2.9), and a module is "a plain struct built from configuration and called by
the scheduler in a fixed, documented order. There is no plugin registry"
([ADR 0001](0001-macro-economy-as-replaceable-layers.md) decision 2). What the
rules do not say is where, inside one package, the pieces of a module live, and
that is what decides whether `tick.go`, the view builder, `removeAgent` and the
checkpoint encoder become the four places every module edits.

The mature agent-based frameworks ended where the rules already are: an explicit
ordered phase list in one place (Mesa 3 removed its scheduler classes for an
inline `step()`; MASON and JAS-mine order same-tick events by an integer;
Caiani's AB-SFC benchmark is an 18-step list), a thin model object that owns
state while behaviour lives in functions, and observation outside the model.
Their post-mortems blame complexity, not a wrong pattern, for models that could
not be reused (EURACE, as Caiani et al. describe it; Richiardi 2015 on models
that "do not travel"). With fifty-one specs and no running code, that is the
risk this decision addresses.

## Decision

1. **One root file per module, named after the concept, and the file owns
   every touch point of the module.** `credit.go` holds the module's state
   struct, the validation of its configuration, its position methods, its
   view filler, its plan checker, its removal step, its relocation refusal, its
   snapshot rows and its checkpoint encoder and decoder. Outside the file the
   module appears only where a rule puts a catalogue: its pointer in `World`,
   its one-line calls in `tick.go`, its event types in `event.go`, its
   sentinels in `errors.go`, its configuration structs in `config.go` and its
   bounds in `bounds.go`. A reader who wants to know what the credit module does
   to the world reads one file. (The file name is the concept; a name that 2.4
   bans as a directory is still allowed as a file, as `infrastructure.go` is,
   note N-019.)

   *Note, 2026-10-26: narrowed by [ADR 0010](0010-a-module-has-named-hooks-outside-its-file.md) decision 1.*

2. **`tick.go` is a straight line of one-line position calls, and the position
   list is data a test can read.** Each of the positions of `spec-world.md` 6 is
   one call, a module's guarded by its pointer (`if w.credit != nil`), in the
   order of the spec. No position holds module logic. The list of positions in
   the code (identifier and owner, in order) is a value a test compares with
   the table of `spec-world.md` 6, the way the parameter-registry test reads
   `specs/README.md` (`doc/rules/testing.md` 6.7); this answers
   `spec-world.md` open question 8.

   *Note, 2026-10-26: narrowed by [ADR 0010](0010-a-module-has-named-hooks-outside-its-file.md) decision 2.*

3. **The fan-in functions are compositions of per-module steps, never module
   logic.** The world has one `fillAgentView`, one `checkTradePlan`, one
   `removeAgent`, one `relocationRefusal` and one `tradableItems`, each a fixed-order sequence of
   calls into the module files. `removeAgent` is the one entry point for a
   removal, used by P7.1 and by the between-ticks calls alike, so the event
   order of `spec-agents.md` 3.6 is written once. The same holds for the view
   and the plan: a module's group of the view is filled by the module's own
   function, in the order the spec fixes.

   *Note, 2026-10-26: narrowed by [ADR 0010](0010-a-module-has-named-hooks-outside-its-file.md) decision 3.*

4. **A module's per-agent state lives in the module's own slices, indexed by
   agent slot, not in fields of the agent type.** The checkpoint's hosting of
   per-agent fields in the agent record (`spec-checkpoint.md` 4.5 F3, note
   N-128) is an encoding layout that the module's encoder writes into and reads
   from; the in-memory owner is the module. Adding a module then widens no core
   type and touches no file but its own and the catalogues of decision 1.

5. **Arithmetic and matching go under `internal/<module>` over primitive
   structs, and the module file converts.** This is rule 2.9 restated so that
   it binds module files: the unit tests of a module's rules need no `World`.

6. **The tests of a module are the module-alone golden file and the
   non-interference of the off-only golden.** ADR 0001 tests "the zero
   configuration, each module alone, and each shipped pack". A change that adds
   a module must leave the off-only golden file unchanged; a module gets a
   golden of its own. A test-only hash of the state after each position of a
   step gives a golden failure its first divergent position instead of a diff
   of the whole rendering; `spec-world.md` 7.4.4 and `doc/rules/testing.md` 7.1
   gain it.

## Consequences

- The root package is large: one file per concept plus one per module, about
  forty files for version 1 and later. That is the cost of rules 2.6 and 2.7
  and is accepted; the discipline of decision 1 is what keeps it readable.
- `doc/rules/architecture.md` 2.6 gains the file-per-module rule and the
  position list; `doc/rules/testing.md` 7.1 and 10.4 gain the per-position
  hash and the off-only, module-alone and all-on golden files. Both are rules
  changes with a fresh read-receipt tag, made with the first code (note N-019).
- The slice comes first ([ADR 0003](0003-version-1-scope.md)): the 28 core
  positions, with `tick.go`, the fan-in functions and the checkpoint shape
  of this decision in place before any module file exists, so that every module
  is made to fit them rather than the other way round.
- `strategy/` (the merchant, `spec-merchants.md`) is a strategy and not a
  module file; the contract round and the contract state are the module file
  `contract.go` and the strategy stays in `strategy/`.

## Alternatives considered

- **A package per module.** Rejected: a module needs the root's types, and the
  root calls the module, so it is an import cycle (2.7); and 2.4 bans the names
  several modules would take.
- **A `Module` interface with a list of hooks (`Positions()`, `OnRemove()`) that
  the world iterates.** Rejected: it is the registry ADR 0001 rejected, it hides
  the order in a list built at runtime, and architecture.md 4.2 forbids an
  interface with one implementation. The explicit sequence in `tick.go` is the
  order's documentation.
- **Per-agent fields of each module on the agent type**, as the checkpoint's
  hosting might suggest. Rejected: every module would widen `Agent`, and the
  agent file would be the fifth place every module edits.
- **A watcher or event-driven dispatch** (Repast's `@Watch`, FLAME's message
  boards). Rejected: order becomes implicit, which is the hazard of
  `spec-world.md` 6.1.

## Open points

1. A conformance test suite for the five extension interfaces (`Strategy`,
   `Clearing`, `Spawner`, `Policy`, `Observer`), run by every built-in and by
   `examples/`, is the one idea of the research that rule 10.2 ("no test
   doubles exported solely for testing") does not obviously allow. Decide with
   the first consumer example.
2. Whether the per-position hash is a test-only helper or an `Observer` a
   consumer could use to compare two runs. Test-only until a consumer asks.

## Related

- `doc/rules/architecture.md` 2.6, 2.7, 2.9, 4.2, 10.2
- [ADR 0001](0001-macro-economy-as-replaceable-layers.md) decision 2,
  [ADR 0002](0002-persistence-by-checkpoint.md) decision 6,
  [ADR 0003](0003-version-1-scope.md), [ADR 0008](0008-random-draws-are-keyed-streams.md)
- `spec-world.md` 6, 22 (question 8), 23; `spec-agents.md` 3.6, 6;
  `spec-checkpoint.md` 4.2, 4.5; notes N-019, N-128
