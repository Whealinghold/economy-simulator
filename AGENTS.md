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
├── go.mod  go.work  doc.go  bounds.go  defaults.go
├── config.go validate.go hash.go errors.go event.go option.go tick.go
│                           # the CATALOGUE files: the only files outside its own
│                           #   file in which a module appears; append-only in
│                           #   module order (architecture.md 2.10.2, 2.10.5);
│                           #   tick.go is one line per position of spec-world.md 6
├── value.go random.go world.go calendar.go commodity.go recipe.go role.go
│   ledger.go market.go order.go clearing.go agent.go strategy.go spawner.go
│   beliefs.go production.go estates.go welfare.go skills.go observer.go
│   snapshot.go regions.go scale.go checkpoint.go integration.go moneyflow.go
│   viability.go           # CORE concept files, one per concept; the five fan-ins
│                           #   and the two agent-lifecycle fan-ins
│                           #   (architecture.md 2.10.4) are in fanin.go
├── strategy_default.go strategy_work.go strategy_view.go (AGT)
│   spawner_report.go (SPN)  production_report.go (PRO)
│   viability_check.go viability_table.go viability_report.go (VIA)
│   construct.go construct_warm.go fanin.go phase_start.go phase_production.go
│   phase_trade.go phase_outcomes.go phase_exit.go (WLD)
│   checkpoint_save.go checkpoint_restore.go checkpoint_sections.go (CHK)
│   regions_relocate.go regions_status.go (REG)
│   demand_plan.go demand_state.go (demand (DEM); only if Lizard forces the
│   split of demand.go)
│                           # CONTINUATION files: a concept past the Lizard limits
│                           #   of N-243 is split; each maps to the area code of
│                           #   its concept, and its _test.go carries that area's
│                           #   Requirement: lines (architecture.md 2.6)
├── demand.go resources.go subsistence.go population.go transport.go
│   information.go migration.go contract.go external.go settlements.go
│   storage.go employment.go tenure.go polity.go credit.go risk.go
│   infrastructure.go insurance.go fiscal.go institutions.go policy.go
│   saving.go technology.go # MODULE files: one per optional module, owning every
│                           #   touch point of the module (architecture.md 2.10.1)
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
│   ├── checked/ logkeys/ ordering/ codec/        # leaf helpers; import nothing of
│   │                       #   the root (ordering holds the stream key, ADR 0008)
│   ├── ledger/ clearing/ beliefs/ production/ welfare/ household/ occupation/
│   │   steady/ graph/ curve/
│   │                       # core rules over primitive structs (household: the
│   │                       #   agent's consumption, targets, tiers, spend cap and
│   │                       #   work rules; occupation: the spawner's signals, caps,
│   │                       #   ranking, renewal and placement; steady: the exact
│   │                       #   solver, math/big allowed)
│   ├── moneyflow/ population/ transit/ endowment/ rumour/ outside/ ownuse/
│   │   migration/ merchant/ room/ employment/ settlement/ tenure/ polity/
│   │   credit/ facility/ risk/ fiscal/ institution/ saving/ technology/
│   │   policy/ insure/     # one package per module with arithmetic or matching
│   └── measure/ verdict/ design/ reduce/ calibration/   # the platform
├── otelobserver/           # separate module: Observer backed by OpenTelemetry
├── examples/<name>/        # each example is its own module: the first consumer's
│                           #   pack and a second, unrelated setting
├── testdata/               # shared fixtures: chains/ (config), golden/ (off/, one
│                           #   directory per module, all/: testing.md 7.1.1),
│                           #   checkpoints/ (one fixture of pack one and one
│                           #   all-modules-on fixture per format version), calibration/ (records per pack); fuzz
│                           #   corpora live in each package's testdata/fuzz/
├── doc/specs/              # design specs, one area per file (README.md indexes
│                           #   them and assigns area codes; evidence.md holds
│                           #   the sources); each spec's Requirements section
│                           #   carries the FR-/NFR- IDs cited by tests
├── doc/documentation_index.md  # every file under doc/: summary and when to read it
├── doc/overview.md        # the approach in one page: pictures, principles, status
├── doc/vision.md          # accepted: first consumer, goals, version 1 scope
├── doc/spec-roadmap.md    # the specs, in order, with area codes and dependencies
├── doc/roadmap-v1.md      # the delivery order of the code: stages, files, exit tests
├── doc/notes.md           # notes for future work, indexed by spec area code;
│                          #   read before writing or changing a spec
├── doc/plans/              # implementation plans, one or more per stage of roadmap-v1.md;
│                          #   working documents, deleted after the stage's sign-off (the
│                          #   commit is recorded in the roadmap); a plan decides nothing
├── doc/adr/              # architecture decision records, numbered NNNN-title.md
└── doc/rules/              # normative rules (architecture, observability, testing)
```

Domain types live in the root package. `internal/` packages work on their own
small structs (for example `clearing.Offer`) and the root (or `indicator/` or
`experiment/`, for its own `internal/` package) converts to and from them inline
at the call boundary (`doc/rules/architecture.md` rule 2.9). A file or package
is added when needed, not up front, in the delivery order of `doc/roadmap-v1.md`;
no module file before the slice's golden file runs (architecture.md 2.10.6).

This is the target layout from `doc/rules/architecture.md` section 2.6, which
is normative and wins over this summary; the module-file rule is its 2.10.
Update this section when the structure changes, in the same change as
`architecture.md` 2.6 and with the same files in both lists.

## Commands

Run the module's checks with the standard Go toolchain:

```sh
gofmt -l .                     # must print nothing
go vet ./...
staticcheck ./...
go test -short -race ./...     # unit tests only, fails fast
go test -race -timeout 60m ./...   # gate R: reduced seeds under -race
go test -covermode=atomic -coverpkg=./... -coverprofile=coverage.out -timeout 120m ./...   # gate F (the root package needs about 60 min under coverage)
go mod tidy                    # must produce no diff
```

Coverage thresholds and the use of mutation testing (`gomu` runs only inside
Docker with a CPU cap) are in `doc/rules/testing.md` section 9.
