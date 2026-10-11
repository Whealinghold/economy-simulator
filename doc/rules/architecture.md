# Architecture Rules

This document is normative. It binds every contributor, human or agent, who
writes code in this module.

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT** and **MAY** are
used as in RFC 2119. A rule marked MUST has no exceptions unless this document
says so. To deviate from a SHOULD, write the reason in the PR description.

If a rule here conflicts with a general style guide, this document wins. Among
the rule files, this document wins over
[observability-and-logging.md](observability-and-logging.md) and
[testing.md](testing.md); for tests of observability behaviour,
observability-and-logging.md wins over testing.md. The losing document is
corrected in the same PR. Where this document is silent, follow, in order:

1. [Effective Go](https://go.dev/doc/effective_go)
2. [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
3. [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions)
4. [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

Rules that are deliberately stricter than those sources, or specific to this
project, are marked **[project]**. Everything else restates the cited guidance.
See section 15.

## 1. Purpose and shape

`economy-sim` is a reusable Go library that simulates a market economy with
configurable supply chains, in the style of bazaarBot. It is consumed by other
modules (game loops, gRPC servers, HTTP servers, CLIs). It is not an application.

1.1. The module **MUST** remain a pure library. It **MUST NOT** contain a server,
a transport, a persistence layer or a `main` package, except under `examples/`
(see 2.6). **[project]**

1.2. The module **MUST NOT** import transport, RPC, HTTP-framework, database,
telemetry (OpenTelemetry) or serialization-format dependencies, with the single
exception allowed by 5.2. **[project]**

1.3. The public API **MUST** be sufficient to build a complete server without
importing anything under `internal/`. Verify this with a consumer example.
**[project]**

## 2. Module and package layout

2.1. Use the simplest layout that works. Add a package only when it has a clear
single purpose and at least one reason to exist independently.

2.2. The root package is the public API and holds the types most consumers need:
`World`, `Market`, `Agent`, `Commodity`, `Recipe`, `Role`, `Order`, `Snapshot`
and the `Option` type. The go.dev module-layout guide shows this shape: the
importable package at the module root, supporting packages under `internal/`.

The root package is named `economysim`. The last element of the import path,
`economy-sim`, is not a valid identifier, so the package name and the import
path differ. `doc.go` **MUST** state the package name, and examples and
documentation import it with an explicit name (`economysim "<module path>"`) so
the name is visible where it is imported. **[project]**

2.3. Packages **MUST** be named for what they provide, not for their
architectural layer.

2.4. The following are forbidden as package names and directory names:

- from the Go guidance: `util`, `utility`, `common`, `helper`, `misc`, `api`,
  `types`, `interfaces`, `model`, `testhelper`
- **[project]**: `utils`, `helpers`, `base`, `shared`, `models`, `pkg`, `ports`,
  `adapters`, `domain`, `application`, `infrastructure`, `service`, `services`,
  `repository`, `usecase`

2.5. Package names **MUST** be short, lowercase, a single word, without
underscores or mixedCaps. Exported names **MUST NOT** repeat the package name as
a prefix (`market.MarketBook` is wrong, `market.Book` is right). A package that
exports a single primary type uses `New` as its constructor, not `NewX`
(`widget.New`, not `widget.NewWidget`). See 6.1 for packages with several
types.

2.6. Layout: **[project]**

```
economy-sim/
  go.mod
  go.work                local development only (core, otelobserver, examples)
  doc.go                 package economysim: overview and the glossary (3.2)
  *.go                   public API, one file per concept: world.go, market.go,
                         agent.go, order.go, value.go, snapshot.go, strategy.go,
                         clearing.go, spawner.go, observer.go, event.go, errors.go,
                         checkpoint.go, option.go, random.go, tick.go,
                         integration.go
  config/                file format -> root Config structs; the only package
                         that may import a third-party parser
  strategy/              additional strategies (the default implementation of
                         each extension interface lives in the root, see 4.5)
  indicator/             derived measures over a snapshot and events; imports
                         the root, and nothing in the module imports it
  experiment/            sweeps, replicas and reductions of runs; imports the
                         root and config/, and nothing in the module imports it
  internal/              everything not meant to be public
    clearing/            order-matching algorithm over primitive structs
    ledger/              goods and money accounting
    beliefs/             price-belief updates
    production/          recipe execution
    ordering/            deterministic sorted iteration helpers, the SplitMix64
                         finaliser, and the Weyl, bucket and key helpers
    codec/               canonical encoding primitives of a checkpoint
    measure/             checked arithmetic of indicator/ over primitive types
    verdict/             detectors, seed rules and the band-setting rule of the
                         stylised facts
    design/              experiment design: grids, permutations, seeds, overlay
                         text
    reduce/              checked reductions of experiment/ over primitive types
    logkeys/             slog attribute key strings; it imports nothing and
                         holds no type of the root
  otelobserver/          separate module (own go.mod): Observer backed by OpenTelemetry
  examples/<name>/       each example is its own module with its own go.mod
  testdata/              fixtures shared across packages: chains/ (config
                         fixtures, including the reference chain), golden/,
                         checkpoints/ (one fixture per checkpoint format version)
  doc/specs/             design specs; their Requirements sections carry the
                         FR-/NFR- IDs that tests cite
  doc/rules/             this directory
```

The `internal/` packages, `strategy/`, `indicator/` and `experiment/` are
created when first needed, not up front (2.1). The names above are the intended
homes, not a checklist.
`strategy/` exists once there is a second strategy; the default stays in the
root package, which cannot import `strategy/` (2.7).

Fuzz corpora are not in the root `testdata/`. `go test` reads and writes them
in `testdata/fuzz/` of the package that owns the fuzz test (for example
`config/testdata/fuzz/`).

`otelobserver/` is the only sanctioned in-repo module besides examples. It
depends on the OpenTelemetry API packages and on the root module, never the
reverse. Use a `go.work` file for local development, and tag its releases with
the sub-module prefix (`otelobserver/v0.1.0`).

2.7. Import cycles are forbidden. `config`, `strategy`, `indicator` and
`experiment` MAY import the root package, and `experiment` MAY also import
`config` (to load a configuration); the root package **MUST NOT** import any of
them. Nothing in the module imports `indicator` or `experiment` (its own tests,
`otelobserver` and `examples/` aside), and `experiment` imports neither
`indicator` nor `internal/measure`.

2.8. Code that is not part of the intended API **MUST** live under `internal/`.
Keep as much as possible there: it lets the module refactor freely without
breaking consumers. Every exported identifier outside `internal/` is a
compatibility promise, so export as little as possible.

2.9. Packages under `internal/` **MUST NOT** import the root package, `config/`,
`strategy/`, `indicator/` or `experiment/`. Domain types (`Commodity`, `Recipe`,
`Role`, `Order`, `Price` and so on) are owned by the root package. An `internal/`
package defines the small structs it needs (for example `clearing.Offer`) using
primitive fields, and the root converts to and from them. This keeps the
dependency direction `config`, `strategy`, `indicator` and `experiment` → root →
`internal`, avoids import cycles (2.7), and makes each internal package testable
on its own. **[project]**

- Convert only at the call from the root into `internal/`, inline in the calling
  code. `indicator` converts at its own call into `internal/measure`, and
  `experiment` into `internal/design` and `internal/reduce`, in the same way.
  Do not create a package, file or type whose job is converting. This is not the
  mapper layer forbidden by 3.1.
- An `internal/` package does not know what a commodity, role or agent is. It
  receives opaque numbers and references, consistent with 9.1.
- When the root needs the original object back, the internal struct carries a
  `Ref` (normally the index into the caller's slice) and the result returns it.
  The internal package never copies or returns the caller's rich types.
- Conversion allocates. In hot paths (10.6) benchmark it, and if it matters,
  reuse scratch slices held by the `World` rather than merging the types.

```go
// internal/clearing: stdlib only, no root types.
type Offer struct {
	Ref   int // index into the caller's slice, returned in Match
	Price int64
	Qty   int64
}

type Match struct {
	BidRef, AskRef int
	Price, Qty     int64
}

func Average(bids, asks []Offer) []Match { /* ... */ }

// Root package: convert in, call, convert out.
func (AverageClearing) Clear(bids, asks []Order) []Trade {
	b := make([]clearing.Offer, len(bids))
	for i, o := range bids {
		b[i] = clearing.Offer{Ref: i, Price: int64(o.Price), Qty: int64(o.Quantity)}
	}
	a := make([]clearing.Offer, len(asks))
	for i, o := range asks {
		a[i] = clearing.Offer{Ref: i, Price: int64(o.Price), Qty: int64(o.Quantity)}
	}

	matches := clearing.Average(b, a)

	trades := make([]Trade, len(matches))
	for i, m := range matches {
		trades[i] = Trade{
			Commodity: bids[m.BidRef].Commodity,
			Buyer:     bids[m.BidRef].Agent,
			Seller:    asks[m.AskRef].Agent,
			Price:     Price(m.Price),
			Quantity:  Quantity(m.Qty),
		}
	}
	return trades
}
```

## 3. Architectural style

The module is separated from transports by the module boundary itself. No
additional layering is needed or allowed.

3.1. Contributors **MUST NOT** introduce hexagonal, clean or onion layering, or
DDD tactical constructs such as aggregates, repositories, services, use cases,
factories-of-factories or mapper layers. The module has no persistence and no
transactional workflows, so these add indirection with no benefit. The inline
conversion at the `internal/` boundary described in 2.9 is not a mapper layer.
**[project]**

3.2. Use a single ubiquitous vocabulary in code, config and docs:
`Commodity`, `Recipe`, `Role`, `Agent`, `Market`, `Order`, `Trade`, `World`,
`Tick`, `Run`, `Money`, `Price`, `Quantity`, `Strategy`, `Clearing`, `Spawner`,
`Snapshot`, `Event`, `Observer`.
A new domain term **MUST** be added to the glossary in `doc.go` before use.
**[project]**

3.3. Domain quantities **MUST** use small named value types (`Money`, `Price`,
`Quantity`, `CommodityID`, `RoleID`) so that mixing units is a compile error.
Every other configured entity has a typed ID of its own kind in the same way
(`RegionID`, `RecipeID`, `ModeID`, `ResourceID`, `AgentID`); a money flow, which
has no typed ID, is named by its string ID (also its ledger reason).
These types **MUST** be immutable and **MUST NOT** carry behaviour beyond
arithmetic, comparison and formatting. **[project]**

3.4. Domain events are delivered through the `Observer` interface (see 4.3),
not through an event bus, channels or global registries. **[project]**

3.5. Money, prices and quantities are fixed-point integers. **[project]**

- `Money` is an `int64` count of the smallest currency unit (a minor unit). The
  library has no notion of decimal places or of a currency; how many minor
  units make one displayed unit is the consumer's concern.
- `Price` is money per unit of a commodity, an `int64` in minor units.
  `Quantity` is an `int64` count of whole units of a commodity.
- Ratios, rates and factors in configuration (belief-update step sizes,
  margins, faucet and sink rates) are integers in basis points (1/10 000) and
  are applied with integer arithmetic: `v * bp / 10_000`.
- `float32`, `float64` and the floating-point functions of `math` **MUST NOT**
  be used to compute anything that affects simulation state, events, snapshots
  or results. Floating point is not associative, and the Go specification lets
  an implementation fuse `x*y + z` into a single operation, which arm64 and
  other architectures do, so float results can differ by platform and break
  7.4. Consumers convert to floating point for display and metrics, outside the
  core.
- Go's integer division truncates toward zero. A function whose result is
  rounded (for example the midpoint of a bid and an ask) states its rounding
  rule in its doc comment. Truncation toward zero is the default; changing a
  rounding rule is a behaviour change (6.6).
- Configuration validation (8.5) bounds prices, quantities, initial balances
  and basis-point factors with documented constants, chosen so that the
  products the simulation computes fit in `int64`. Arithmetic that can still
  overflow (price × quantity, accumulated balances, value × basis points)
  goes through checked helpers in `internal/` (using `math/bits` or an
  explicit range check) that report overflow to the caller. The caller
  handles it as an operation that cannot be applied, for example an order
  rejected with a `reason`. Silent wrap-around is a defect.
- One exception, for the exact solver in `internal/steady`
  (`doc/specs/spec-viability.md` sections 6 and 7): it **MAY** compute with
  unbounded integers (`math/big`, `big.Int`). Those cannot overflow, so the
  checked helpers of the previous bullet are not required for arithmetic
  inside the solver. Every other rule of this section still applies to it:
  `big.Float` and the other floating-point types stay banned, the rounding
  rule is stated, and the result is deterministic. A result leaves the solver
  as fixed point only at its boundary, once, by the rounding rule stated
  for that field in `spec-viability.md` (prices and wages are rounded half to even, as in clearing, a surplus down and a labour share up), and each
  conversion to `int64` is checked against `int64` and the field's bound
  (`spec-viability.md` 6.4) and reports overflow to the caller like any other
  operation that cannot be applied. No `math/big` type appears in the
  signature of anything exported from the root package, `config/` or
  `strategy/`, nor in an event or a snapshot: `internal/steady` takes and
  returns primitive types (2.9), and the big values stay inside it.

## 4. Interfaces and extension points

4.1. Interfaces **MUST** be defined in the package that consumes them, not in
the package that implements them.

4.2. Interfaces **MUST** be small (one to three methods) and **MUST NOT** be
created until a real need exists: a second implementation or a test seam.

4.3. The only sanctioned extension interfaces are: **[project]**

| Interface  | Purpose                                                  |
|------------|----------------------------------------------------------|
| `Strategy` | decides an agent's work (`Work`), its bids and asks (`Trade`) and its belief updates (`Revise`); the world checks what each returns |
| `Clearing` | the rule by which a market matches orders and sets price |
| `Spawner`  | decides replacement and entry of agents                  |
| `Observer` | receives events (trade, bankruptcy, tick) for metrics and tracing; the default discards events |

Adding a new extension interface requires an update to this table.

`Event` (4.8) is also an interface, but it is a sealed set of types the library
defines, not an extension point: consumers cannot implement it, and it is not
subject to 4.5.

**Extension state.** An implementation of `Strategy`, `Clearing` or `Spawner`
that keeps state between calls **MUST** also implement the optional interface
`Stateful`, so that a checkpoint can save it and a restore can load it (ADR
0002, decision 7): **[project]**

```go
type Stateful interface {
	SaveState() (version uint32, state []byte, err error)
	LoadState(version uint32, state []byte) error
}
```

`Stateful` is not an extension point and is not subject to 4.5: no code requires
it, an implementation that keeps no state does not implement it, and it has no
default. `Observer` is never saved: an observer must not influence results
(observability-and-logging.md 1.4), so it has no simulation state.

4.4. Functions **MUST** accept interfaces and return concrete types. A function
**MUST NOT** return an interface unless hiding unexported state across a
package boundary requires it, and the reason **MUST** be commented.

4.5. Each extension interface **MUST** ship with a default implementation in
the module and a compile-time assertion:

```go
var _ Clearing = AverageClearing{}
```

Use `(*T)(nil)` instead when the methods have pointer receivers.

4.6. Do not use `any`/`interface{}` in public signatures. Use generics only
where they remove duplication without obscuring intent (typed IDs, small
containers). Generics **MUST NOT** replace the extension interfaces above.
**[project]**

4.7. Public structs **MUST NOT** embed other types. Embedding leaks
implementation details and inhibits evolution of the type; use a named field
and delegate methods instead.

4.8. `Observer` has exactly one method, and events form a sealed set:
**[project]**

```go
type Observer interface {
	Observe(Event)
}

// Event is implemented only by the event types defined in this package.
type Event interface {
	isEvent()
}

// ObserverFunc adapts a function to the Observer interface.
type ObserverFunc func(Event)

func (f ObserverFunc) Observe(e Event) { f(e) }
```

Observers type-switch on the concrete event type (`TradeExecuted`,
`AgentBankrupted`, ...) and **MUST** ignore event types they do not recognise,
so adding an event is not a breaking change. Do not add one method per event: a
wide interface breaks 4.2 and makes every new event a breaking change for every
implementer. Events are passed by value. The detailed event rules are in
[observability-and-logging.md](observability-and-logging.md) section 4.

## 5. Dependencies

5.1. The root package, `indicator/`, `experiment/` and `internal/` **MUST**
depend on the standard library only (and, within the module, on the packages 2.7
allows). **[project]**

5.2. Only `config/` MAY import a third-party package, and only for file-format
parsing. Everything it produces **MUST** be plain structs defined in the root
package, so that callers can build a configuration in Go without any file
format. `experiment/` reaches a parser only through `config/`. **[project]**

5.3. A new third-party dependency **MUST** be justified in the PR description
(what it provides, why the standard library is insufficient, its maintenance
status and licence). **[project]**

5.4. `go.mod` declares `go 1.25` as the minimum. Raising it is a breaking
change and **MUST** be called out in the changelog. **[project]**

## 6. Public API design

6.1. A constructor is named `New` when the package exports a single primary
type that the package name already identifies, and `NewX` when the package has
several types (Effective Go; `bufio.NewReader`, `http.NewRequest`). The root
package has several types, so the world's constructor is `NewWorld`.
Constructors **MUST** return `(*T, error)` when validation can fail.

6.2. Optional configuration **MUST** use functional options: **[project]**

```go
type Option func(*options) error

func WithSeed(seed uint64) Option {
	return func(o *options) error { o.seed = seed; return nil }
}

func NewWorld(cfg Config, opts ...Option) (*World, error) { /* ... */ }
```

The core options include `WithSeed`, `WithLogger(*slog.Logger)`,
`WithObserver(Observer)`, `WithAudit()`, `WithTargetSpread(bp)` and the
registration options `WithStrategy(name, s)`, `WithSpawner(name, s)` and
`WithClearing(name, c)` (`spec-world.md` 4.1), and, for a restore only,
`WithReseed(salt)` (`spec-checkpoint.md` 6.8). Without `WithSeed` the seed is 0,
so a world built with no options is still reproducible (7.4); the last
`WithSeed` given wins. `WithSeed` on a restore and `WithReseed` on `NewWorld`
are `ErrOptionContext`, and `RestoreWorld` takes the same `Option` type. The
logger and observer are optional and default to no-ops (see 11.2). Passing a nil
logger or observer **MUST** be treated as "use the no-op default", not as an
error. `WithObserver` is additive: it MAY be passed more than once, and every
event is delivered to each observer in the order the options were given. The
registration options are additive too: a name given twice, or a nil extension,
is an error.

Options **MUST** validate their input and return an error rather than panic.
The `options` struct **MUST** be unexported.

6.3. Types **SHOULD** be usable at their zero value where that is sensible, and
**MUST** document when they are not.

6.4. Methods and functions **MUST NOT** use a `Get` or `get` prefix (a getter
for field `owner` is `Owner`, not `GetOwner`), unless the underlying concept
is itself a "get". Prefer exported fields for plain data.

6.5. Every exported identifier **MUST** have a doc comment beginning with its
name. The root package **MUST** have a `doc.go` with an overview and the
glossary from 3.2. Each public entry point **SHOULD** have a runnable `Example`
test.

6.6. This module follows semantic versioning. A change to an exported
identifier's signature or documented behaviour is breaking and **MUST NOT** be
released without a major version bump. From v2 onward the module path **MUST**
gain the major version suffix (`.../economy-sim/v2`), which forces every
consumer to update its imports, so avoid breaking changes and keep the exported
surface small.

## 7. State, determinism and randomness

A simulation library is only useful if its runs can be reproduced. **[project]**
for the whole section; 7.1 and 7.2 are stricter than the Uber guide's "avoid".

7.1. Package-level mutable state is forbidden. Package-level variables **MUST**
be effectively constant (sentinel errors, compile-time assertions, lookup
tables that are never mutated). Inject dependencies instead.

7.2. `init()` functions **MUST NOT** be used.

7.3. Code in the root package, `indicator/`, `experiment/` and `internal/`
**MUST NOT** call `time.Now`, `time.Since`, the global `math/rand` or
`math/rand/v2` functions, or `crypto/rand`. Time is simulated ticks. Randomness
comes from one `*rand.Rand` (`math/rand/v2`) per world, built on a `rand.PCG`
source seeded from `WithSeed`. The choice of source and the way the seed maps
onto it are part of the golden output; changing either is a behaviour change
(6.6).

- The seeding rule: the state of the `rand.PCG` is the first two outputs of the
  SplitMix64 generator started from the seed (`spec-world.md` 7.1.2).
- The world's draw for a bound `n` of at least 1 is the high word of
  `Uint64() * n` (`math/bits.Mul64`): one source word per draw, no loop and no
  rejection (`spec-world.md` 7.2.1). The methods of `rand.Rand` that an extension
  may call can take more than one word per call (`spec-world.md` 7.2.2).
- On a restore the source is installed from the saved 20-byte state of the
  `rand.PCG` and not seeded; the one option that changes it is `WithReseed`,
  whose rule is part of the golden output as the seeding rule is.

7.4. For a given seed, configuration and sequence of calls, the output
**MUST** be bit-identical across runs and platforms. This is enforced by a
golden-file test (see 10.4) and depends on 3.5, 7.5 and 7.6. CI **SHOULD** run
the determinism and golden-file tests on at least two architectures (for
example amd64 and arm64). A Go toolchain upgrade that changes the golden output
is a behaviour change like any other and is explained in its PR.

7.5. Iteration over a map **MUST NOT** influence simulation results. Where
order matters, iterate over a sorted slice of keys or use an ordered structure.

7.6. A sort that can influence results **MUST** give the same order for equal
elements on every run and Go version. `sort.Slice`, `sort.Sort` and
`slices.SortFunc` are not stable, and their order for equal elements changed
when Go 1.19 switched to pattern-defeating quicksort. Either make the
comparator a total order that ends in a unique tie-break (an ID or an index),
or use `slices.SortStableFunc` on input whose order is itself deterministic.
This matters most for clearing, where equal prices are common. **[project]**

7.7. Extension implementations (4.3) get their randomness from the world.
Methods on `Strategy`, `Clearing` and `Spawner` that need it receive the
world's `*rand.Rand` as a parameter. Implementations **MUST NOT** create their
own source, use a global one, or keep the one they were given after the call
returns. The interfaces' doc comments **MUST** state this contract, because a
consumer's implementation that breaks it breaks 7.4. **[project]**

An implementation that is `Stateful` (4.3) **MUST** keep in what `SaveState`
writes every piece of state that can influence a result (no state in closures,
package variables or goroutines, 7.1 and 12.2). `SaveState` **MUST NOT** change
the extension and **MUST** give equal bytes for equal states. `LoadState`
**MUST** replace the whole state, **MUST** validate its input and return an error
for a version or bytes it cannot read, and **MUST NOT** keep the slice it is
given. Neither method receives or may create a random source. An instance that
holds state **MUST NOT** be shared between worlds. The doc comments of
`Strategy`, `Spawner`, `Clearing` and `Stateful` **MUST** state this contract,
because an implementation that breaks it breaks 7.4 after a restore and the
library cannot detect it. **[project]**

## 8. Errors

8.1. Errors **MUST** be returned, not panicked. `panic` is allowed only for
violated programmer invariants that cannot be caused by input or
configuration, and the message **MUST** say so. `os.Exit` and `log.Fatal*`
**MUST NOT** be called anywhere in the module.

8.2. Wrap with `%w` when callers need to match the underlying cause with
`errors.Is` or `errors.As`; use `%v` to deliberately hide it. Add context:
`fmt.Errorf("load recipe %q: %w", id, err)`. Error strings **MUST NOT** be
capitalised (unless they begin with a proper noun, acronym or exported name)
or end with punctuation, and **SHOULD NOT** start with "failed to".

8.3. An error **MUST** be handled exactly once: returned, logged, or handled,
never more than one. The library does not log an error it returns; the caller
decides whether to (observability-and-logging.md 2.2). Errors **MUST NOT** be
discarded with `_`.

8.4. Conditions a caller may branch on **MUST** be exposed as sentinel errors
(`var ErrUnknownCommodity = errors.New(...)`) or as typed errors usable with
`errors.As`.

8.5. Configuration **MUST** be validated in full at load or construction time.
A running simulation **MUST NOT** fail because of a configuration mistake. The
validator **MUST** reject unknown commodity references, duplicate IDs,
recipes with no output, goods that no role can produce or consume, supply
chains with a cycle that has no raw-material source, and numeric values outside
the bounds of 3.5. **[project]**

## 9. Configuration as data

**[project]** for the whole section.

9.1. Commodities, recipes and roles **MUST** be data. Code **MUST NOT** branch
on a specific commodity, recipe or role identifier. A `switch` or `if` on such
an identifier is a defect.

9.2. A supply chain of any length **MUST** emerge from recipes alone: one
recipe's outputs are another's inputs, and agents obtain inputs through the
market. There is no chain-specific code.

9.3. The in-memory configuration types live in the root package and are plain
structs. `config/` only translates a file format into those structs.

9.4. Adding a new commodity or chain **MUST NOT** require changing Go code.
A test that does so with a new configuration file only is required.

9.5. A number that affects simulation results and that a user could reasonably
vary (a scale, a floor, a step size, a ratio, a rate) is a field of the
configuration types in the root package. It has a default, an owner spec, a
documented validation bound (3.5, 8.5) and a doc comment. It **MUST NOT** be a
literal in logic. Rule 9.1 forbids branching on an identifier; this rule forbids
burying a tunable in an algorithm, where it would change results without
changing the configuration hash of a checkpoint
([ADR 0002](../adr/0002-persistence-by-checkpoint.md)).

9.6. There are three kinds of constant, and each has one home:

| Kind | Examples | Home |
|------|----------|------|
| Tunable (9.5) | `Scale`, `MinAgents`, belief step sizes | the configuration types, with defaults defined in one place |
| Validation bound | maximum price, quantity and weight | one file, `bounds.go`, in the root package |
| Rule constant | the rounding rule, rejection `reason` strings, log keys | next to the code that owns it |

A rule constant is part of the definition of a behaviour, not a setting; do not
move it into configuration to look flexible. Packages under `internal/` receive
tunables and bounds as parameters and never import the root package to read
them (2.9). There is no package for constants (2.4).

9.7. Every configuration field that 9.5 covers **MUST** appear in the
parameter-owner table of `doc/specs/README.md`, and every row of that table
**MUST** have a field, a default and a validation test. A test checks both
directions (testing.md 6.7). This rule takes effect when `spec-config.md` fixes
the shape of the configuration; until then, a spec that introduces a parameter
adds its row to the table.

9.8. A default is defined once. Changing it changes results, so the PR
**MUST** say which behaviour changes and, if the golden file changes, follow
testing.md 7.2.

## 10. Testing

[testing.md](testing.md) specifies how tests are written. This section keeps
only the rules the architecture depends on and points to the details.

10.1. Tests with more than one case **MUST** be table-driven with `t.Run`
subtests. Test names describe behaviour. When to use `t.Error` and `t.Fatal`,
and the rules for test helpers, are in testing.md section 3.

10.2. Public behaviour **SHOULD** be tested from an external `_test` package
through the public API. Use internal tests only for unexported logic that
needs it. Production code **MUST NOT** export back doors or test doubles
solely for testing.

10.3. Prefer the real implementation over a double. Where a seam is needed
(an extension interface from 4.3), use a small hand-written fake defined in a
`_test.go` file. Generated mocks and assertion frameworks **MUST NOT** be used.
Comparisons use the standard library (`reflect.DeepEqual`, `slices.Equal`,
`maps.Equal`). **[project]**

10.4. The following are required; testing.md specifies each: **[project]**
- **Invariant tests** (testing.md 6.3): conservation of goods and money,
  non-negative inventories and balances.
- **Determinism and golden-file tests** (testing.md 6.3, 7.1, 7.2): they
  enforce 7.4.
- **Configuration-only chain test** (testing.md 6.4): it enforces 9.4.
- **Config tests** (testing.md 3.9): each validation rule in 8.5 has a
  failing-case test.
- **Parameter registry test** (testing.md 6.7): it enforces 9.7, once
  `spec-config.md` fixes the configuration shape.

10.5. Tests **MUST** pass with `go test -race ./...`. Tests **MUST NOT**
sleep, depend on wall-clock time, or touch the network. **[project]**

10.6. Performance-sensitive code (order matching, tick loop) **SHOULD** have
benchmarks, and changes to it **SHOULD** include benchmark results
(testing.md section 8).

10.7. The testing pyramid, test comments (`Requirement:`, `Fix:` and `Guard:`
tags), traceability to the requirements in the specs and the coverage thresholds are
specified in [testing.md](testing.md). Contributors **MUST** follow both
documents.

## 11. Observability

11.1. The library **MUST NOT** write to stdout, stderr or any logger it was not
given. It logs only through an injected `*slog.Logger`. Do not define a custom
logger interface; `log/slog` is the standard abstraction and `slog.Handler` is
the adaptation point for clients.

11.2. Logging and metrics are optional. When the client passes nothing, both
are no-ops:
- the default logger is `slog.New(slog.DiscardHandler)`
- the default observer discards every event

The library **MUST NOT** fall back to `slog.Default()`, and **MUST NOT** call
any global registry (including OpenTelemetry's `otel.GetTracerProvider` and
`otel.GetMeterProvider`).

11.3. Logging and observation **MUST NOT** influence simulation results. Code in
hot paths **SHOULD** check `logger.Enabled(ctx, level)` before building
attributes. **[project]**

11.4. Telemetry is delivered through the `Observer` interface (see 4.3) plus
`Snapshot`. The core module **MUST NOT** import `go.opentelemetry.io/*`.
OpenTelemetry support lives in a separate module (`otelobserver`, see 2.6) that
implements `Observer` and accepts a `TracerProvider` and `MeterProvider`
supplied by the client. A client who wants OTel imports that module and passes
it with `WithObserver`; a client who does not pays no dependency cost.
**[project]**

11.5. The detailed rules for log levels, events, metrics, traces and the
observability checklist for every change are in
[observability-and-logging.md](observability-and-logging.md). Contributors
**MUST** follow both documents.

## 12. Concurrency and consumers

12.1. `World` and everything it owns are single-threaded and **NOT** safe for
concurrent use. This **MUST** be stated in the type's doc comment. Callers
that need concurrency serialise access themselves. **[project]**

12.2. The library **MUST NOT** start goroutines. If parallelism is ever added
it is opt-in, owned by an object with an explicit `Start`/`Close` lifecycle,
documented here first, and every goroutine **MUST** have a predictable stop
time or a way to be signalled to stop. **[project]** for the ban; the lifecycle
requirement is from Code Review Comments and the Uber guide.

12.3. Any call that may run for a long time or block **MUST** take
`context.Context` as its first parameter and honour cancellation. A
`context.Context` **MUST NOT** be stored in a struct field. `Step()` and other
short, CPU-bound calls do not need one. A multi-tick convenience such as
`Run(ctx, ticks)` returns `(completed int64, err error)`, **MUST** check `ctx`
before every tick, including the first, and never inside a tick. On cancellation
it returns the wrapped `ctx.Err()` and leaves the world at a tick boundary,
never half-way through a step. A failed step is not a cancellation: `Run`
returns the step's error with the ticks completed, and the world is then failed:
every later call that has an error result returns `ErrWorldFailed`
(`spec-world.md` 3.5). `Run` emits `RunStarted` before its first tick and
`RunEnded` after its last completed tick, including when it is cancelled or a
step failed (observability-and-logging.md 4.3).

12.4. `Snapshot` and every value returned across the API boundary **MUST** be a
deep copy containing plain data (no pointers into live state, no function
fields), so a consumer can marshal it on another goroutine without races.
**[project]**

12.5. Observers are called synchronously on the stepping goroutine, in the
order they were registered, **MUST NOT** call back into the `World`, and
**SHOULD** return quickly. The library does not recover a panicking observer:
the panic propagates to the caller of `Step`. Events are emitted after the state
change they describe is complete, in a deterministic order. A failed step is the
one exception: the world is marked failed before `TickEnded` is emitted, and the
event describes the end of the step (`spec-world.md` 3.5.6). **[project]**

12.6. Transports (gRPC, HTTP, CLI) are separate modules that import this one.
They own serialisation, authentication, concurrency control and lifecycle.
Types in this module **MUST NOT** carry struct tags for any transport.
**[project]**

## 13. Anti-patterns checklist

Reject a change that contains any of the following:

- a forbidden package name (2.4) or a layered directory structure (3.1)
- an interface with no second implementation and no test need (4.2)
- a function returning an interface (4.4)
- an embedded type in a public struct (4.7)
- `init()`, package-level mutable state, global `rand` or `time.Now` (7)
- floating point in anything that affects results, or unchecked arithmetic
  that can overflow (3.5; the unbounded integers of `internal/steady` cannot)
- an unstable sort, or a comparator without a unique tie-break, where the
  order affects results (7.6)
- an extension implementation with its own random source (7.7)
- a `Get`-prefixed accessor (6.4)
- naked returns in functions longer than a few lines
- ignored errors (`_ = f()`), or logging and returning the same error (8.3)
- `panic`, `os.Exit` or `log.Fatal` on input-derived data (8.1)
- a branch on a commodity, recipe or role identifier (9.1)
- a result-affecting numeric literal in logic that is neither a documented rule
  constant nor a configuration field (9.5, 9.6)
- a package for constants, or an `internal/` package importing the root to read
  a tunable (9.6, 2.9)
- a goroutine without an owner and a stop signal (12.2)
- a `context.Context` stored in a struct (12.3)
- a dependency not justified under 5.3
- an import of `go.opentelemetry.io/*` in the core module (11.4)
- use of `slog.Default()` or a global OTel provider (11.2)
- a custom logger interface instead of `*slog.Logger` (11.1)
- logging or observation that changes simulation results (11.3)

## 14. Tooling

14.1. `gofmt`, `go vet` and `staticcheck` run clean in CI.
`go test -race ./...` passes. `go mod tidy` produces no diff. The exact
commands are in [AGENTS.md](../../AGENTS.md) and testing.md 9.1. **[project]**

14.2. A change to architecture **MUST** update this document in the same PR.
**[project]**

## 15. Provenance

Rules without a **[project]** marker restate the following sources, checked
directly against the primary pages:

| Rule(s)                        | Source                                                      |
|--------------------------------|-------------------------------------------------------------|
| 2.2, 2.8                       | go.dev "Organizing a Go module": root package, `internal/`  |
| 2.2 (explicit import name)     | Google decisions (import renaming when the name differs from the path) |
| 2.4 (first list), 2.5          | Code Review Comments (package names); Google decisions      |
| 2.5 (`widget.New`), 6.1        | Effective Go (names, `ring.New`); Google decisions (package name repetition) |
| 3.5 (floating point)           | The Go Programming Language Specification (arithmetic operators: fused floating-point operations; integer division truncates) |
| 7.6                            | `sort` and `slices` package documentation (sorts not guaranteed stable); Go 1.19 release notes (pattern-defeating quicksort) |
| 4.1, 4.2, 4.4                  | Code Review Comments (Interfaces); Google decisions         |
| 4.5, 4.7                       | Uber guide (verify interface compliance; avoid embedding)   |
| 6.4                            | Effective Go (getters); Google decisions                    |
| 6.5                            | Code Review Comments (doc comments)                         |
| 6.6                            | go.dev "Module version numbering"                           |
| 8.1                            | Code Review Comments (Don't Panic); Uber guide             |
| 8.2, 8.3                       | Uber guide (error wrapping, handle once); Code Review Comments (error strings, ignored errors); Google decisions |
| 10.1, 10.2                     | Google decisions (table-driven tests, test doubles)         |
| 12.2 (lifecycle), 12.3         | Code Review Comments (goroutine lifetimes, contexts); Google decisions; Uber guide |
| 11.1, 11.2                     | `log/slog` package documentation (`DiscardHandler`, `Enabled`); OpenTelemetry library guidelines (libraries use the API only; no-op without an SDK) |

Rules 4.8, 11.3–11.5 and 12.5, the extension-state paragraph of 4.3 and the
last paragraph of 7.7 are project decisions: the sources do not prescribe an
observer shape, a sealed event set, delivery ordering, or how an extension saves
its state. The
fixed-point representation in 3.5 and the tie-break rule in 7.6 are project
decisions built on the language and library facts cited above.

The go.dev layout page does not mention a `pkg/` directory. The ban in 2.4 is a
project decision, not a quote from that page.

## Read receipt

If you are an agent, you quote the tag below in your first message, as
required by [AGENTS.md](../../AGENTS.md). Reaching it means you have read this
file in full. Do not guess it or reuse one from memory.

When you edit this file, replace the tag with a fresh random value in the same
change (for example `od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`). A stale
tag in an agent's first message means it read an older version.

RULE-TAG: 42eeab37487ade5f18253f16
