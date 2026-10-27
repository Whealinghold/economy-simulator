# Testing Rules

This document is normative. It binds every contributor, human or agent, who
writes or changes code in this module. Its purpose is to make sure the library
has a solid testing pyramid and good coverage, that tests explain themselves,
and that tests can be traced back to the product requirements they verify.

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT** and **MAY** are
used as in RFC 2119. To deviate from a SHOULD, write the reason in the PR
description.

This document extends section 10 of [architecture.md](architecture.md) and
section 8 of [observability-and-logging.md](observability-and-logging.md). If
they conflict, `architecture.md` wins; for tests of observability behaviour,
observability-and-logging.md wins over this document. The losing document is
corrected in the same PR.

Rules that are project decisions rather than restatements of a cited source are
marked **[project]**. Section 13 lists the sources.

## 1. Principles

1.1. A change is not complete until it is tested. New behaviour, a changed
behaviour and a fixed bug each ship with tests in the same PR. **[project]**

1.2. Test behaviour through the exported API of the package under test, at the
lowest level of the pyramid that can detect the failure: the module's public
API for the root package, `config/` and `strategy/`, and the package's own
exported functions for a package under `internal/`. Do not test implementation
details that a refactor should be free to change.

1.3. A test is documentation. It states what is being verified, why it matters,
and which requirement it serves, so that a maintainer who has never seen the
code can decide whether a failure is a bug or an outdated expectation.
**[project]**

1.4. Coverage is evidence, not a goal. A line that is executed but never
asserted on is not tested. **[project]**

## 2. The testing pyramid

The pyramid prioritises many fast, cheap tests at the base and few slow,
expensive tests at the top. The three levels are unit, integration and
end-to-end, with cheaper levels preferred. The levels below are defined for this
library. **[project]**

| Level | What it exercises | Style | Share of tests |
|-------|-------------------|-------|----------------|
| **Unit** (base) | One type or function in isolation: clearing rules, belief updates, value types, recipe and config validation, strategies, event construction | Table-driven, deterministic, no I/O, parallel | The large majority |
| **Integration** (middle) | Several real components together through the public API: a `World` with agents and markets, config files loaded into a world, `Observer` and `*slog.Logger` wiring, `otelobserver` against the OpenTelemetry SDK's in-memory exporters | Real implementations, fixed seeds, `testdata/` fixtures | A moderate number |
| **End-to-end** (top) | A whole simulation as a consumer would use it: the reference supply chain run for many ticks and compared to a golden file; each module under `examples/` builds and runs | Few, slow, high-value | A handful |

2.1. Every behaviour **MUST** be verified at the lowest level that can detect
its failure. If a unit test would catch a bug, do not rely on an end-to-end test
for it. If a bug only appears when components interact, write an integration
test.

2.2. Do not invert the pyramid. If the end-to-end or integration tests
outnumber the unit tests for a package, add unit tests.

2.3. Every requirement (see section 4) **MUST** have at least one test at the
lowest level that can verify it. Every requirement that describes behaviour
visible to a consumer **MUST** additionally be exercised through the public API
at the integration level or above.

2.4. Cross-cutting techniques, used at whichever level fits:

- **Invariant tests**: properties that must hold for every seed (see 6).
- **Fuzz tests**: for parsers and validators (see 7).
- **Determinism tests**: the same seed gives identical results (architecture.md
  7.4).
- **Benchmarks**: for the order-matching and tick loops (section 8).

## 3. Unit tests

3.1. Tests with more than one case **MUST** be table-driven with `t.Run`
subtests. Each table entry is a complete case with its
inputs and expected results and a descriptive name. Write the test body once
and put the care into it, including good failure messages.

3.2. Subtest names are short lowercase phrases that describe the scenario
("bid below ask does not match"). A name is not a requirement ID (see 4.3).

3.3. Failure messages **MUST** include the function under test, the input, the
actual value and the expected value, in that order, and use the `got`/`want`
words:

```go
if got := clearing.Average(tt.bids, tt.asks); !slices.Equal(got, tt.want) {
	t.Errorf("Average(%v, %v) = %v, want %v", tt.bids, tt.asks, got, tt.want)
}
```

3.4. Inside a test or subtest, use `t.Error` for checks that are independent of
each other, so one run reports every mismatch, and `t.Fatal` when the rest of
the test or subtest would be meaningless (failed setup, an unexpected error, a
nil result about to be dereferenced). In a table-driven test each case is a
`t.Run` subtest, so `t.Fatal` stops only that case. Do not loop over cases
without subtests and use `t.Error` plus `continue`. `t.Fatal` and `t.FailNow`
**MUST** be called only from the goroutine running the test.

3.5. Tests **SHOULD** call `t.Parallel()` at the top level and in subtests when
they share no state. Because the library has no package-level mutable state
(architecture.md 7.1), unit tests are normally safe to run in parallel.

3.6. Keep validation logic in the test function. Assertion libraries and
generated mocks **MUST NOT** be used, and do not write helpers that take
`*testing.T` just to compare values. Compare with `==`, `slices.Equal`,
`maps.Equal` or `reflect.DeepEqual` and write the message yourself
(architecture.md 10.3). A helper that does setup **MUST** call `t.Helper()` and may call
`t.Fatal` when its precondition fails; use `t.Cleanup` and `t.TempDir` for
resources.

3.7. Test the public API from an external `_test` package. Use an internal test
only when it needs unexported logic (architecture.md 10.2).

3.8. Cover the boundaries, not only the happy path: empty and single-element
inputs, zero and negative quantities, equal prices, exact ties, insufficient
inventory, an agent with no money, an unknown commodity, a recipe whose inputs
are unavailable, the largest and smallest valid values.

3.9. Test every error path. For each sentinel or typed error the library
returns (architecture.md 8.4), a test asserts it with `errors.Is` or
`errors.As`, not by comparing message text. Each configuration validation rule
(architecture.md 8.5) has a case whose configuration breaks only that rule and
asserts the specific error.

## 4. Traceability to requirements

Tests are written against the requirements, not only against the code.
Requirements are written in the specs under `doc/specs/`, one area per file.
Where a requirement has a test that verifies it, the test says so. **[project]**
for the whole section.

4.1. Each spec in `doc/specs/` has a Requirements section. A requirement is
identified by a stable ID of the form `FR-<AREA>-NNN` for functional and
`NFR-<AREA>-NNN` for non-functional requirements (for example `FR-CLR-012`).
`<AREA>` is a three-letter uppercase code assigned to a spec in the area table
in `doc/specs/README.md`, and a code is never reused. IDs **MUST** be stable and
unique across the module. A requirement that moves to another spec keeps its
ID. If a spec renumbers or removes a requirement, update the tests in the same
change.

4.2. A test that verifies a requirement **MUST** name it in a `Requirement:`
line at the end of its doc comment, with a comma-separated list when it covers
several:

```go
// TestClearing_MatchesHighestBidToLowestAsk verifies that the market sorts
// bids from highest to lowest and asks from lowest to highest before matching,
// so the most generous buyer is matched with the cheapest seller first.
//
// Requirement: FR-CLR-012, FR-CLR-013
func TestClearing_MatchesHighestBidToLowestAsk(t *testing.T) { /* ... */ }
```

For a table-driven test where only some cases serve a requirement, put a
`Requirement:` comment above those cases and keep the function comment for the
ones that apply to all.

4.3. Do not put requirement IDs in test or subtest names. Names are used with
`go test -run`, appear in CI output and change rarely; the requirement mapping
is metadata and lives in the comment. The line **MUST** use exactly the form
`// Requirement: <ID>[, <ID>...]` so it can be found with a plain text search.

4.4. When a test of public behaviour has no requirement to name, there are two
cases:
- the specs have a gap: raise it in the PR and add the requirement to the
  relevant spec before merge, or
- the test protects an invariant or edge the specs do not state: label it with
  `Guard:` (see 5.2) and state what it protects.

Tests of internal details that no requirement describes **MAY** omit the tag.

4.5. A requirement with no test is a defect in the change that introduces or
modifies it. The PR description **MUST** list the requirement IDs the change
affects and the tests that cover each one. **[project]**

4.6. A requirement **SHOULD** be traceable both ways. A repository check **SHOULD**
list the requirement IDs defined in the specs that appear in no `Requirement:`
line, and the IDs in `Requirement:` lines that are defined in no spec. Until
that check exists, reviewers do it by hand with a text search over
`doc/specs/` and the tests.

## 5. Commenting tests

Comments explain *why*, not *what*. The test body and its names show what
happens; the comment says what is being verified, what would go wrong without
the test, and anything that is not obvious from the code.

5.1. Every `Test`, `Fuzz` and `Benchmark` function **MUST** have a doc comment
that begins with the function name and states the behaviour under test in one
or two sentences. This is the test's contract (the same rule as for exported
identifiers, applied to tests). `Example` functions are exempt: their name ties
them to the identifier they document, and their `// Output:` comment states the
contract. **[project]**

5.2. Tests that exist for a reason beyond verifying a requirement **MUST** carry
a tag line in the doc comment, after the description and any `Requirement:`
line:

| Tag | Use it for | Content |
|-----|------------|---------|
| `Fix:` | A regression test that pins a bug that was found and fixed | The bug or issue ID, then what was wrong in one sentence |
| `Guard:` | A test that protects an invariant, contract or edge case that a future change could silently break, where no bug has occurred yet | What is protected, and what would go wrong if the test were removed |

```go
// TestBankruptcy_AgentWithOutstandingAskIsRemovedFromBook verifies that an
// agent that goes bankrupt in a tick has its unmatched asks withdrawn before
// the market clears.
//
// Fix: ECON-41. A bankrupt agent's ask stayed in the book and was matched in
// the same tick, creating a trade with a seller that no longer existed.
func TestBankruptcy_AgentWithOutstandingAskIsRemovedFromBook(t *testing.T) { /* ... */ }
```

```go
// TestTick_OrderOfEventsIsDeterministic verifies that events for one step are
// emitted in the same order on every run.
//
// Guard: observers and golden files depend on the order. If it depended on map
// iteration, runs would differ only occasionally and the failure would be
// hard to reproduce.
func TestTick_OrderOfEventsIsDeterministic(t *testing.T) { /* ... */ }
```

5.3. A `Fix:` test **MUST** be written before the fix and **MUST** fail without
it. Run it against the unfixed code, confirm it fails for the reason in the
comment, then apply the fix and confirm it passes. State in the PR description
that you did. A fix test that passes on the unfixed code does not guard
anything.

5.4. A `Fix:` or `Guard:` comment **MUST** stay accurate. If the code or the
reason changes, update the comment. A comment that describes a different case
than the test covers is worse than none.

5.5. Comment unusual test data. A literal that came from a calculation (an
expected price, a quantity after several production steps) **MUST** be
accompanied by the arithmetic, so a reader can tell whether the expectation is
right:

```go
// Two bids (10, 8) against two asks (6, 9): the first pair clears at the
// average, (10+6)/2 = 8; the second bid is below the second ask and does
// not match.
```

5.6. Comment the setup when the scenario is not obvious: why this seed, why this
number of agents, why this recipe chain. Prefer a short comment on the fixture
over a long explanation in each test.

5.7. Do not comment what the code already says (`// loop over cases`), do not
leave commented-out tests, and do not use a comment where a better name would
do. Skipped tests (`t.Skip`) **MUST** state the reason and an issue ID, except
the `testing.Short()` skip of 10.5, whose reason is the test's level.

## 6. Integration tests, invariants and determinism

6.1. Integration tests use real components, not fakes, wherever the real thing
is practical. A fake is acceptable only at an extension point from
architecture.md 4.3, is hand-written, and lives in a `_test.go` file (3.6).

6.2. Every random scenario uses a fixed seed. A test that runs over many seeds
**MUST** report the failing seed in the message, so a failure is reproducible by
one `WithSeed` call.

6.3. The following invariants **MUST** be asserted over the reference chain and
over a range of seeds, after every tick: **[project]**
- goods are conserved except by recipes (inputs consumed, outputs produced);
- money is conserved except by explicit faucets and sinks;
- no inventory or balance is negative;
- no trade has a buyer or seller that does not exist;
- the clearing price lies between the matched bid and ask;
- identical seed and configuration give identical results.

Money and quantities are integers (architecture.md 3.5), so conservation is
asserted with exact equality, never with a tolerance.

6.4. The integration level **MUST** include a test that builds a world from a
new configuration file alone, with no Go code specific to it, and runs it to a
stable state. This enforces architecture.md 9.4.

6.5. Observability is tested at this level too, per observability-and-logging.md
section 8: recorded events, captured log records, no-op defaults,
non-interference, composition and emission order.

6.6. `otelobserver` is tested in its own module against the OpenTelemetry SDK's
in-memory exporters. The core module's tests **MUST NOT** import OpenTelemetry.

6.7. A parameter registry test enforces architecture.md 9.7, in both
directions: every configuration field that architecture.md 9.5 covers is a row
of the parameter-owner table in `doc/specs/README.md`, and every row has a
field, a default that passes validation, and a failing-case validation test
(3.9). The test reads the table with a plain text parse, so it needs no
extra dependency. It is written when `spec-config.md` fixes the configuration
shape; until then reviewers check the table by hand. While the modules are
being built it runs in the staged form of architecture.md 9.7: one explicit
list, `notBuilt`, names the scopes (a Go type name) and the fields of a built
scope (`Scope.Field`) that the code does not have yet, and the rows it names
are skipped by the checks that need a field (a row has a field, the default
passes validation, a failing case exists). The test fails when a listed entry
exists in the code and when a row is neither built nor listed. The direction
"every field has a row" is never staged. The change that builds a scope or a
field deletes its entries. The catalogue test of events
(`spec-observability.md`, FR-OBS-003) is staged the same way. The registry of
knobs of
`doc/specs/spec-calibration.md` section 6 has a second test of the same kind
(that spec's 11.1): it reads the same table and the classification of the
registry by the same plain text parse, and needs no extra dependency.
**[project]**

6.8. A position table test enforces architecture.md 2.10.3: the list of
positions in `tick.go` (identifier, owner, in order) equals the table of
`spec-world.md` 6, read by a plain text parse as 6.7 reads the parameter-owner
table. A position in the spec with no call, a call with no row, or a different
order fails it. **[project]**

It runs in the staged form of 6.7 (architecture.md 2.10.3): `tick.go` holds the
list of every position of `spec-world.md` 6 that has a call, and one explicit
list, `notBuiltPositions`, names the positions that have no call yet. The test
compares identifier, owner and order with the table by a plain text parse. It
fails when a listed position has a call (a stale entry) and when a spec
position is neither called nor listed (a typo). A module's change deletes its
entries and appends its guarded call.

- **Owner.** The test compares the first area code of the Owner cell; the cell
  "this spec" (P0.1, P8.6) reads as WLD.
- **P0.2.** "Nothing is cleared at entry" is an empty position method `p02()`
  called in the straight line of `tick.go`, a `line` entry; the parse does not
  special-case it and it has no permanent `notBuiltPositions` entry.
- **`where`.** Each list entry has a third field: `line` (a call in the
  straight line) or `bracket` (run inside `beginStep` or `endStep`: P0.1,
  P0.2a, P0.3, P8.5, P8.6). A `bracket` entry is compared for identifier, owner
  and order like any other but has no call to find in the straight line.
- **The straight line.** The test reads the straight-line function of
  `tick.go` with `go/parser` and checks that it calls the `line` entries in the
  listed order and nothing else, so a call with no row fails too.

## 7. End-to-end, golden-file and fuzz tests

7.1. A golden-file test runs the reference supply chain for a fixed number of
ticks and compares a canonical rendering of the result (prices, volumes,
inventories, event counts) to a file in `testdata/golden/`. It exists to detect
unintended behaviour change. The golden-file and determinism tests **SHOULD**
run in CI on at least two architectures (architecture.md 7.4). **[project]**

7.1.1. There are three sets of golden files, under `testdata/golden/`
(architecture.md 2.10.6): **[project]**
- `off/`: the reference chain with every module off. A change that adds a
  module **MUST** leave these files unchanged; a diff here is a non-interference
  failure, not a golden to regenerate. A field a module appends to an event or
  a row is omitted from the rendering at its off value
  (`spec-observability.md` 9.5.2), so an unchanged file is what a correct
  append produces.
- `<module>/`: one directory per module, the reference chain with that module
  alone on (its overlay from `testdata/chains/overlay/`). A module ships with
  its golden in the same PR.
  One exception: a module whose pack carries its own census (the warm start is solved
  again for the module, as the pack `one-demand` does for the wants) has that pack as its
  module-alone golden, not the reference chain with an overlay. The proof that `off/` did
  not move is the same.
- `all/`: every module on. The one set that may move when a module is added,
  with the explanation 7.2 requires.

7.1.2. A test-only per-position state hash: a hash of the ledger, the agents
and every module's state after each position of a step, recorded for one
scripted run of each golden set. When a golden file fails, the hash test
reports the first tick and position at which the state diverged, so the
failure names a position instead of a rendering. The hash is a test helper in
the root package's `_test` files, not an exported API (architecture.md 10.2).
It is computed by the test file replaying the step through the unexported
position methods in the order of the position table, so the hash adds no hook
to production code, and a guard test compares the replay's final state hash and
golden bytes with a plain `Run`. **[project]**

7.2. Golden files are updated only by an explicit flag (`-update`) and
**MUST NOT** be regenerated to make a failing test pass without understanding
why it failed. A PR that changes a golden file **MUST** explain, in the PR
description, what behaviour changed and which requirement or fix it serves.
This includes a change caused by a Go toolchain upgrade.

7.3. Each module under `examples/` **MUST** build and run in CI, with a test or
`Example` test that checks its output. They are the proof that the public API
is sufficient for a consumer (architecture.md 1.3).

7.4. Fuzz tests (`FuzzXxx`, taking `*testing.F`) are **required** for every
function that parses or validates external input, notably the configuration
loader and its validators. Provide seed inputs with `f.Add`, and assert
properties (never panics, valid input round-trips, invalid input returns an
error), not exact outputs.

7.5. `go test` writes a failing fuzz input to `testdata/fuzz/<FuzzName>/` in
the directory of the package that owns the fuzz test (for example
`config/testdata/fuzz/FuzzLoad/`), not in the module root, and runs it as part
of an ordinary `go test`. These files **MUST** be committed. They are
permanent regression tests and **MUST** have a `Fix:` comment in the fuzz
function explaining the case they pin.

## 8. Benchmarks

8.1. The order-matching and tick loops **SHOULD** have benchmarks that call
`b.ReportAllocs()`. A change to those paths includes before and after results.
Benchmarks use `for b.Loop() { ... }`, which keeps setup out of the timing and
stops the compiler from optimising the body away.

8.2. Benchmarks report; they never fail CI. A limit that **MUST** hold, such as
the allocation limit of the no-op observability path (architecture.md 11.2,
observability-and-logging.md 8.2), is enforced by an ordinary test using
`testing.AllocsPerRun`, with a benchmark alongside it for the trend.

## 9. Coverage

9.1. CI **MUST** run two commands and fail the build when either fails or when
coverage falls below the thresholds. **[project]**

- **R**, the race run: `go test -race -timeout 60m ./...`. Under `-race` the
  scenario and determinism tests run a reduced seed set (`raceSeeds`, one).
- **F**, the full run with coverage, without `-race`:
  `go test -covermode=atomic -coverpkg=./... -coverprofile=coverage.out -timeout 120m ./...`.
  All seeds and all cases run, and the thresholds below are read from its profile.
  Its timeout is 120m, not 60m: the root package needs about 60 minutes under atomic
  coverage, so 60m left no margin (owner ruling, stage 2-6).

Coverage is not taken under `-race`: with atomic coverage of `-coverpkg=./...`
the race run took 48 to 58 minutes, against 164 seconds without coverage and 15
minutes for F. A quick run before a push is
`go test -timeout 60m -run 'TestScenario|TestDeterminism|TestGolden|TestInvariants|TestCheckpoint|TestFork|TestConfigOnly|TestFixture|TestPacks|TestCensus|TestExpected|TestStylised|TestCost|TestStable' ./...`
(about half a minute, all seeds, no coverage). A root-package test that skips under
`raceEnabled` **MUST** have a name that starts with one of the prefixes of that
pattern (`gateFPrefixes` in `scenario_helpers_test.go`), so the quick run selects
it; `TestRaceSkips_HaveAnFPrefix` enforces it. `-update` is refused under `-race`.

`-coverpkg=./...` counts a statement as covered when any test in the module
executes it. Without it, `go test` counts only the package under test, so code
in `internal/` that is exercised through the root package's tests is reported
as uncovered. A CI script reads `coverage.out` (`go tool cover -func`) and
checks each scope against the table.

| Scope | Minimum statement coverage |
|-------|----------------------------|
| Root package | 90% |
| Each package under `internal/` | 90% |
| `config/`, `strategy/`, `indicator/` and `experiment/` | 90% |

Because `-coverpkg` credits integration tests too, coverage does not show that
an `internal/` package has its own unit tests; 2.1 and 2.2 still apply.
`examples/` and `otelobserver/` are separate modules with their own threshold
(80% for `otelobserver`). Generated code and `main` packages are excluded.

9.2. Coverage **MUST NOT** decrease in a PR. A PR that lowers coverage needs a
written reason.

9.3. New or changed code **SHOULD** be covered by tests that fail if the code is
wrong. Run the tests against a deliberately broken version of the code (flip a
comparison, delete a line) to check that they notice.

9.4. For the core packages, mutation testing **SHOULD** be run periodically to
find tests that execute code without asserting on it. Mutation testing is
CPU-heavy. When using `gomu`, always run it inside Docker with its CPU use capped
to at most a quarter of the host's processors (`--cpus` and `--workers`); never
run it directly on the host.

9.5. A function that is excluded from coverage or left untested **MUST** have a
comment saying why.

## 10. Reliability of the test suite

10.1. Tests **MUST** be deterministic, independent and order-insensitive. A test
that passes only when run after another is broken.

10.2. Tests **MUST NOT** sleep, depend on wall-clock time, use the network, read
outside their own module, or write outside `t.TempDir()` (architecture.md 10.5).

10.3. A flaky test is a bug. Fix it or delete it; do not retry it. Never mark it
skipped without an issue ID (5.7).

10.4. Test fixtures live in `testdata/` next to the package that uses them.
Fixtures shared by several packages live in the module root's `testdata/`
(architecture.md 2.6). The reference supply chain is a configuration file in
`testdata/chains/` and is the standard fixture for integration and end-to-end
tests; tests in other packages read it by relative path
(`../testdata/chains/...`), which stays inside the module (10.2). Golden files
are laid out as 7.1.1 says: `testdata/golden/off/`, one directory per module,
and `testdata/golden/all/`.

10.5. All tests pass with `-race`. Integration and end-to-end tests begin with
`if testing.Short() { t.Skip("integration test, skipped with -short") }` (or
"end-to-end test"). CI runs `go test -short -race ./...` first, so the unit
tests fail fast, and then the full suite with coverage (9.1). Under `-race` the
scenario and determinism tests run a reduced seed set; the full set runs in F,
without `-race` (9.1).

## 11. Considerations when writing code

Before opening a PR, answer these in the PR description under a "Testing"
heading. **[project]**

1. **Which requirements does this change touch?** List the IDs and the tests
   that cover each (4.5).
2. **What is the lowest level that can catch a failure of this change?** Is
   there a test at that level (2.1)?
3. **Which boundaries and error paths are new?** Are they tested (3.8, 3.9)?
4. **Is this a bug fix?** Does a `Fix:` test exist, and did it fail before the
   fix (5.3)?
5. **Is there an invariant or edge case a future change could break?** Add a
   `Guard:` test (5.2).
6. **Did the golden file change?** Why, and for which requirement (7.2)?
7. **Does the change add parsing or validation?** Is there a fuzz test (7.4)?
8. **Did coverage drop?** Why (9.2)?
9. **Did you run the tests against a broken version** to check they can fail
   (9.3)?

## 12. Anti-patterns checklist

Reject a change that contains any of the following:

- new behaviour or a bug fix with no test
- a `Fix:` test that was never seen to fail without the fix (5.3)
- an end-to-end test for something a unit test would catch (2.1)
- a test without a doc comment, or with a comment that restates the code (5.1, 5.7)
- a requirement ID in a test name instead of a `Requirement:` line (4.3)
- an expected value with no explanation of where it came from (5.5)
- assertion libraries, generated mocks, or a helper that takes `*testing.T` to
  compare values (3.6)
- `t.Error` plus `continue` in a loop over cases without subtests (3.4)
- comparing errors by message text (3.9)
- a tolerance when comparing money or quantities (6.3)
- a benchmark as the only enforcement of a limit that must hold (8.2)
- `time.Sleep`, wall-clock time or the network in a test (10.2)
- a random scenario with no fixed seed, or a failure that does not print the
  seed (6.2)
- a configuration field with no row in the parameter-owner table, or a row
  with no field, default or validation test (6.7)
- a golden file regenerated without explanation (7.2), or an `off/` golden
  file regenerated to absorb a module's change (7.1.1)
- a module without its module-alone golden file (7.1.1)
- a fuzz crash input not committed (7.5)
- an unexplained `t.Skip` (5.7)
- coverage that falls with no stated reason (9.2)
- running `gomu` directly on the host (9.4)

## 13. Provenance

| Rule(s) | Source |
|---------|--------|
| 2 (pyramid shape and the priority of unit tests) | The testing pyramid as described in widely published guides (secondary sources); the unit/integration/end-to-end levels and their share are standard, and the definitions for this library are project decisions |
| 3.1, 3.2, 3.4, 3.5 | Go wiki, [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests): table entries with inputs and expected results, `t.Run` subtests, `t.Errorf` continuing, `t.Parallel` |
| 3.3, 3.4, 3.6, 3.7 | Google Go Style Guide, [best practices: testing](https://google.github.io/styleguide/go/best-practices): failure messages with function, input, got and want; `t.Error` versus `t.Fatal`; `t.Fatal` only from the test goroutine; `t.Helper` and `t.Cleanup`; assertion helpers are not idiomatic |
| 7.4, 7.5 | Go tutorial, [Getting started with fuzzing](https://go.dev/doc/tutorial/fuzz): `FuzzXxx`, `f.Add`, `f.Fuzz`, failing inputs written to the package's `testdata/fuzz` and re-run by `go test` |
| 8.1, 8.2, 10.5 | [`testing` package documentation](https://pkg.go.dev/testing): `B.Loop`, `B.ReportAllocs`, `AllocsPerRun`, `Short` |
| 9.1 | `go help testflag` (`-coverpkg`, `-covermode`) and `go tool cover` |
| 4 (requirements in tests) | General requirements-traceability practice (linking requirement IDs to test cases, as annotations or documentation); the `Requirement:` line format is a project decision |
| 5.2, 5.3 (`Fix:`, `Guard:`) | Regression-test practice (name or tag the test after the bug, and write the test first so it fails without the fix); the `Fix:` and `Guard:` tags are project conventions |

I could not find a published, widely adopted "fix/guard" test-commenting
convention. The `Fix:` and `Guard:` tag definitions in 5.2 are this project's own,
built on the common regression-test practice above. The coverage thresholds in
section 9, the invariant list in 6.3 and the share of tests per level are project
decisions and should be tuned once real measurements exist.

## Read receipt

If you are an agent, you quote the tag below in your first message, as
required by [AGENTS.md](../../AGENTS.md). Reaching it means you have read this
file in full. Do not guess it or reuse one from memory.

When you edit this file, replace the tag with a fresh random value in the same
change (for example `od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`). A stale
tag in an agent's first message means it read an older version.

RULE-TAG: cb434ac287a78ec8e68eee2f
