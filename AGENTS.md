# economy-sim: contributor and agent rules

A reusable Go library that simulates a market economy with configurable supply
chains, based on the bazaarBot approach. It is consumed by other modules (game
loops, gRPC and HTTP servers, CLIs). It is a library, not an application.

## Read the rules first

Before doing anything else on a task, read each of these files in full and
quote the `RULE-TAG` from the "Read receipt" at the end of each, verbatim, in
your first message:

- `doc/rules/architecture.md`
- `doc/rules/observability-and-logging.md`
- `doc/rules/testing.md`

Format: `Rule tags: architecture=<tag>, observability=<tag>, testing=<tag>`

The tags are only in the files. Do not guess or reuse one from memory. A wrong
or missing tag means you did not read the file. When you edit a rule file,
replace its tag with a fresh random value; new rule files go on this list.

## Before writing or changing a spec

Read `doc/notes.md` in full first. Its index lists open notes by spec area code
(the codes are in `doc/specs/README.md`). Before you write or change a spec, read
every open note tagged with that spec's code and every note tagged `ALL`, and for
each one either apply it, reject it with a reason, or leave it open with what you
learned. If you find something a future spec or a code change must take into
account, add a note in the same change. The file's own header gives the format
and the rules.

Quote the tag from its "Read receipt" at the end of the file, verbatim, in your
first message about that work, on its own line:

`Notes tag: <tag>`

This applies to any task that writes or changes a spec, adds, updates or closes a
note, or changes a requirement. It does not apply to other tasks. The tag is only
in the file: do not guess it or reuse one from memory, since a wrong or missing
tag means you did not read the file. When you edit `doc/notes.md`, replace its tag
with a fresh random value in the same change.

## Hard constraints

- **Pure library.** No server, transport, persistence layer or `main` package
  in the core module. Servers (gRPC, HTTP) are separate modules that import
  this one. The only exception is `examples/`, where each example is its own
  module.
- **Standard library only in the core.** The root package, `indicator/`,
  `experiment/` and `internal/` import the standard library only (and the
  module's own packages that `doc/rules/architecture.md` 2.7 allows).
  `config/` may import a file-format parser; `experiment/` reaches it only
  through `config/`. No OpenTelemetry, RPC, HTTP-framework or database
  dependencies in this module; OpenTelemetry support is the separate
  `otelobserver` module.
- **Supply chains are data.** Commodities, recipes and roles come from
  configuration. No code may branch on a specific commodity, recipe or role
  identifier.
- **Setting-agnostic.** The library is for any setting, not for a fantasy RPG
  (the first consumer). Names in the public API, configuration, events, logs and
  docs must be generic: no coins, metals, titles, races or other setting
  vocabulary. Prefer neutral words (`Household`, `Settlement`,
  `Carrier`, not `Peasant`, `Village`, `Caravan`) and make anything a setting
  might vary a configuration value or an extension point, not a constant.
  Setting-specific content (names, currencies, denominations, role and region
  lists) belongs in `examples/` or in configuration files, never in the core.
  When a design choice is a trade-off between generic and convenient for the
  RPG, choose generic.
- **Deterministic.** No `time.Now`, no global or `crypto` randomness, no
  package-level mutable state, no `init()`. Randomness comes from an injected,
  seeded source, and extensions receive it from the world. Sorts that affect
  results break ties deterministically.
- **Fixed-point money.** Money, prices and quantities are `int64` fixed-point
  values (money in minor units). No floating point in anything that affects
  results; overflow is checked, never wrapped (`doc/rules/architecture.md`
  rule 3.5).
- **No layered architecture.** No `ports/`, `adapters/`, `domain/`,
  `service/` or `repository/` packages. Interfaces appear only at the
  extension points listed in `doc/rules/architecture.md` section 4, apart from
  the sealed `Event` and the optional `Stateful` (4.3), which are not extension
  points.
- **Observability is optional and a no-op by default.** Log through an
  injected `*slog.Logger` and report through the `Observer` interface; never
  use `slog.Default()` or global providers.

Everything else (layout, API design, errors, concurrency, logging levels,
events, metrics, the testing pyramid, traceability to requirements) is in
the rule files.

## Reporting a change

Every PR description and every agent's final summary has:

- a `Testing` entry (`doc/rules/testing.md` section 11), including the
  requirement IDs the change touches and the tests that cover each; and
- an `Observability` entry (`doc/rules/observability-and-logging.md`
  section 9). "None, because ..." is a valid answer; a missing entry is not.

## Layout

```
economy-sim/
├── go.mod
├── go.work                 # local development only: core, otelobserver, examples
├── doc.go                  # package economysim: docs and the glossary
├── *.go                    # public API, one file per concept: world.go,
│                           #   market.go, agent.go, order.go, value.go,
│                           #   snapshot.go, strategy.go, clearing.go,
│                           #   spawner.go, observer.go, event.go, errors.go,
│                           #   checkpoint.go, option.go, random.go, tick.go,
│                           #   integration.go
├── config/                 # file format -> root Config structs; only package
│                           #   that may import a third-party parser
├── strategy/               # additional strategies (add when a second strategy
│                           #   exists; defaults live in the root package)
├── indicator/              # derived measures over a snapshot and events;
│                           #   imports the root, imported by nothing
├── experiment/             # sweeps, replicas and reductions of runs; imports the
│                           #   root and config/, imported by nothing
├── internal/               # stdlib only; never imports the root, config/,
│   │                       #   strategy/, indicator/ or experiment/
│   ├── clearing/           # order-matching algorithm over primitive structs
│   ├── ledger/             # goods and money accounting
│   ├── beliefs/            # price-belief updates
│   ├── production/         # recipe execution
│   ├── ordering/           # deterministic sorted iteration helpers, the
│   │                       #   finaliser, and the Weyl, bucket and key helpers
│   ├── codec/              # canonical encoding primitives of a checkpoint
│   ├── measure/            # checked arithmetic of indicator/
│   ├── verdict/            # detectors, seed rules and the band-setting rule
│   ├── design/             # experiment design: grids, permutations, seeds
│   ├── reduce/             # checked reductions of experiment/
│   └── logkeys/            # slog attribute key strings; imports nothing
├── otelobserver/           # separate module: Observer backed by OpenTelemetry
├── examples/<name>/        # each example is its own module
├── testdata/               # shared fixtures: chains/ (config), golden/,
│                           #   checkpoints/; fuzz corpora live in each
│                           #   package's testdata/fuzz/
├── doc/specs/              # design specs, one area per file (README.md indexes
│                           #   them and assigns area codes; evidence.md holds
│                           #   the sources); each spec's Requirements section
│                           #   carries the FR-/NFR- IDs cited by tests
├── doc/documentation_index.md  # every file under doc/: summary and when to read it
├── doc/overview.md        # the approach in one page: pictures, principles, status
├── doc/vision.md          # draft: first consumer, goals, version 1 scope
├── doc/roadmap.md         # the specs still to write, in order, with area codes
│                          #   and the tier 1 dependency picture
├── doc/notes.md           # notes for future work, indexed by spec area code;
│                          #   read before writing or changing a spec
├── doc/adr/              # architecture decision records, numbered NNNN-title.md
└── doc/rules/              # normative rules (architecture, observability, testing)
```

Domain types live in the root package. `internal/` packages work on their own
small structs (for example `clearing.Offer`) and the root (or `indicator/` or
`experiment/`, for its own `internal/` package) converts to and from them inline
at the call boundary (`doc/rules/architecture.md` rule 2.9). Packages under
`internal/`, `indicator/` and `experiment/` are added when needed, not up front.

This is the target layout from `doc/rules/architecture.md` section 2.6; the
repository is new, so most of it does not exist yet. Update this section when
the structure changes.

## Commands

Run the module's checks with the standard Go toolchain:

```sh
gofmt -l .                     # must print nothing
go vet ./...
staticcheck ./...
go test -short -race ./...     # unit tests only, fails fast
go test -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...
go mod tidy                    # must produce no diff
```

Coverage thresholds and the use of mutation testing (`gomu` runs only inside
Docker with a CPU cap) are in `doc/rules/testing.md` section 9.
