# 0009. The TOML library is pelletier/go-toml/v2, behind a guard pass

- Status: Accepted
- Date: 2026-10-12
- Supersedes: none

## Context

[`spec-config.md`](../specs/spec-config.md) 6.1 decided the file format (TOML) and
left the library open; note N-019 owes the record. Only `config/` may import a
third-party package (`doc/rules/architecture.md` 5.2), and a new dependency must be
justified by what it provides, why the standard library is insufficient, its
maintenance status and its licence (5.3).

The loader has three needs of a library and no more. It needs the syntax parsed into
a generic tree in which an integer and a float are told apart (`spec-config.md` 2.6:
a float literal is an error, not a value to round); it maps the tree to the root
structs by hand and checks unknown keys itself (6.1, open question 2); and it will be
fed packs that a third party wrote, so a hostile or merely malformed document must
cost a bounded amount of time and memory and must never crash the host process. A
stack overflow or an out-of-memory in Go is a fatal error that no `recover` catches,
so "never panics" is not enough: the question is what the parser does with 100 000
levels of nesting, with a million keys, and with a megabyte of garbage.

The standard library has no TOML parser. JSON (the standard library) was rejected as
the format in `spec-config.md` 6.1: no comments, and a number does not say whether it
is an integer.

## Decision

1. **The library is `github.com/pelletier/go-toml/v2`**, pinned at v2.4.3 in `go.mod`
   and `go.sum`, imported only by `config/`.
2. **It is used for two things only**: `toml.Unmarshal` into a `map[string]any`, which
   is the generic tree, and `unstable.Parser`, for the guard pass of decision 3. Its
   struct decoding, its `DisallowUnknownFields`, its marshaller and the text of its
   errors are not used.
3. **Before a document is decoded the loader runs a guard pass** and refuses, with
   `ErrTooLarge`, a document whose key has more than `MaxNestingDepth` segments, whose
   values nest deeper than that, or that has more than `MaxTableKeys` keys in a table;
   it checks the size and line limits on the bytes first. The limits are
   `MaxDocuments` 16, `MaxDocumentBytes` 4 MiB, `MaxMergedBytes` 16 MiB, `MaxLineBytes`
   256 KiB, `MaxNestingDepth` 16 and `MaxTableKeys` 10 000 (`spec-config.md` 4.1,
   6.3.1). The guard walks the expressions the library's parser yields one at a time
   and builds no tree.
4. **The loader writes its own error text**: it reads the line and column of a syntax
   error and nothing else of the library's error, and quotes and cuts every path
   segment and value it prints (`spec-config.md` 6.3.2).
5. **A change of version re-runs the experiments below**, whose generators are a page
   of Go each, before `go.mod` moves. The `unstable` package is documented as free to
   change; its use is one function of `config/`, and if its API moves the guard is
   replaced by a byte-level scan of the same limits without touching the rest.

## What was measured

Every experiment is a Go program in a scratch module, built with Go 1.27.2 and again
with Go 1.25.0, decoding into `map[string]any`. Each case ran in its own process under
a 6 GB address-space limit (12 GB for the breadth cases) and a 60 s limit, so that
a fatal error shows as a result and does not stop the run. The figures are from one
machine (Linux, amd64) and are orders of magnitude, not benchmarks.

**Facts.**

| | `pelletier/go-toml/v2` v2.4.3 | `BurntSushi/toml` v1.6.0 |
|---|---|---|
| Licence | MIT | MIT |
| First v2 / v1 release; last release | v2.0.0 2022-04-28; v2.4.3 on 2026-07-05, six releases since 2026-03-24 after an 11-month gap | v1.0.0 2022-01-12; v1.6.0 on 2025-12-18, about one release a year |
| `go` directive | 1.21.0 | 1.18 |
| Dependencies at run time | none | none |
| Builds and passes the experiments on Go 1.25.0 and 1.27.2 | yes | yes |
| `1`, `0x10`, `1_000`, `+5`, `-0` | `int64` | `int64` |
| `1.0`, `1e3`, `5.0e0`, `inf`, `nan` | `float64` | `float64` |
| `9223372036854775808`, `-9223372036854775809` | error | error |
| a date | `toml.LocalDate` (and the like) | `time.Time` |
| empty table `[modules.estates]` | present, `map[string]any{}` | present, `map[string]any{}` |
| duplicate key, table redefined, scalar then table | error | error |
| array with mixed types | accepted | accepted |

Both distinguish an integer from a float in the generic tree, and both keep an empty
table as a present, empty map, which is how a module is "on" (`spec-config.md` 6.2.1).
Both accept a mixed array, which the loader's own type check rejects where it matters.

**Depth** (one document, n levels):

| Shape | n | `pelletier` | `BurntSushi` |
|-------|---|-------------|--------------|
| nested arrays `a = [[[...]]]` | 10 000 | ok, 13 ms | ok, 11 ms |
| | 100 000 | error: nested more than the maximum of 10 000 levels, 11 ms | ok, 70 ms, 16 MB |
| | 1 000 000 | the same error, 5 ms | ok, 0.6 s, 162 MB |
| nested inline tables `a = {a={a=...}}` | 10 000 | ok, 12 ms, 8 MB | **fatal: out of memory** (over 4.8 GB) |
| | 100 000 and more | the same error, 10 ms | fatal: out of memory |
| array of tables nested `[{a=[{...` | 10 000 and more | the same error | fatal: out of memory |
| dotted key `a.a.a. ... = 1` | 10 000 | ok, 3 ms, 5 MB | ok, 1.5 s, **3.8 GB** |
| | 100 000 | ok, 43 ms, 61 MB | fatal: out of memory |
| | 1 000 000 (a 2 MB document) | ok, 0.27 s, **606 MB** | fatal: out of memory |
| table header `[a.a.a. ...]` | 100 000 | ok, 41 ms, 61 MB | fatal: out of memory |
| | 1 000 000 | ok, 0.27 s, 606 MB | fatal: out of memory |
| array-of-tables header `[[a.a. ...]]` | 1 000 000 | ok, 0.32 s, 606 MB | fatal: out of memory |

`pelletier` bounds arrays and inline tables itself, at 10 000 levels, and returns an
error: it never recurses far enough to overflow the stack. It does **not** bound the
depth of a dotted key or a table header, and each level costs about 600 bytes, so a
16 MiB document of `a.` repeated would decode to about 5 GB. `BurntSushi` bounds
nothing, and its memory is quadratic in the depth of a key: 10 000 nested inline tables
or a dotted key 100 000 segments long takes the process down with a fatal error.
Neither bound is "never panics"; this is why the loader must enforce its own depth
bound before the document reaches the parser (decision 3).

**Breadth** (documents of about 16 MB unless stated):

| Shape | `pelletier` | `BurntSushi` |
|-------|-------------|--------------|
| one table of 10 000 / 20 000 / 40 000 / 80 000 keys | 0.1 / 0.3 / 1.2 / **5.5 s** | not run |
| one table of 1 000 000 keys | **over 100 s, stopped** | 1.4 s, 993 MB |
| 60 000 sibling tables | 3.9 s | not run |
| 800 000 sibling tables | **over 100 s, stopped** | 1.6 s, 1.2 GB |
| 1 300 000 `[[a]]` entries | 0.33 s, 551 MB | 1.2 s, 1.2 GB |
| one array of 8 000 000 integers | 0.7 s, 1 018 MB | 0.9 s, 817 MB |
| one 16 MB string, or one 16 MB key | 10 ms, 15 MB | 0.1 s, 64 to 94 MB |

`pelletier` is quadratic in the number of keys that are children of one table (about
1.1 ns times n squared, in every release tried: v2.0.0, v2.1.1, v2.2.4, v2.3.0, v2.4.0 and v2.4.3): its duplicate check is
a linear scan of the siblings. `BurntSushi` is linear. The memory of both is about 20
to 60 times the size of the document. This does not rank the libraries; it fixes what
the guard must bound: the keys of one table, at `MaxTableKeys` 10 000 (the number of
recipes a pack may have, `MaxRecipes`; the largest table a pack writes), which costs
about 0.1 s a table (ten tables of 10 000 keys, 0.8 MiB: 0.9 s; 160 tables, 12 MiB: 14 s,
458 MB), so that a document at the limits costs seconds and not a day.

**Malformed input.** Go's native fuzzer, 75 s per library, four workers, seeded with
every file of `testdata/chains/` and twenty hand-written malformed documents (a
truncated string, a bare `\xff`, a huge exponent, redefined tables, an unterminated
inline table, control characters): `pelletier` 2.6 million executions, `BurntSushi`
3.1 million, no panic, no crash. The last seconds of the `pelletier` run executed
nothing: the fuzzer had reached inputs in the quadratic region above and the run ended
at its time limit, with a pass.

**Error text.** For a 1 MiB value that is not valid, `pelletier`'s `Error()` is 55
bytes and `BurntSushi`'s is 1 MiB. For a duplicate key whose name is 1 MiB, both put
the whole name in the text (1 MiB and 2 MiB). `pelletier`'s `DecodeError.String()`,
which shows the document around the error, is up to 4 MiB. A library error is therefore
not safe to print or to wrap into a consumer's log (`spec-config.md` 6.3.2).

**The guard pass** (the `unstable.Parser`, counting only): a 16 MB single-line document
of `a.` is refused as too deep after 0.5 s, having allocated 0.9 GB because the parser
reads the whole key before it yields it (56 bytes a segment), which is why a line is
bounded at 256 KiB (about 14 MB of transient memory) before the pass; one million flat
keys pass in 31 ms and allocate nothing; arrays and inline tables nested 10 000 deep
are refused by the library's own limit in 4 ms; a key of 16 segments and a value
nested to a total depth of 16 are accepted, and one more is refused.

## Consequences

- One third-party dependency, confined to `config/`, with no dependency of its own:
  one line in `go.mod` and the checksums in `go.sum`. `go mod tidy` must produce no
  diff; `govulncheck` runs once on it before Stage 0 is closed because no CI does.
- A document that passes the guard decodes in memory of at most about 64 times its
  size and in time of at most about two seconds a mebibyte (1.2 measured at the table limit), and every recursion the loader
  writes over the tree is at most 16 deep, so none can overflow a stack. The values of
  the six limits are placeholders in the sense of note N-015: they come from the
  measurements above and not from a study of real packs, and a pack that needs more
  splits into documents (at most 16, 16 MiB together).
- The `MaxDocumentBytes` of `spec-config.md` 4.1 falls from 16 MiB to 4 MiB: at 16 MiB
  the worst document of the guard's limits is about 19 s of parsing and 4 MiB is about 5 s.
- The guard parses each document twice (once to count, once to decode). It costs a
  few milliseconds a megabyte on a legitimate pack, which is the price of not trusting
  the parser's depth.
- The loader carries its own copy of the key rules and the error rules, so a consumer
  that sees an error text sees the loader's, never the library's.
- An upgrade of the library is a reviewed change: the experiments are re-run and the
  table of this ADR is amended.

## Alternatives considered

- **`github.com/BurntSushi/toml`.** The same capability (integers and floats told
  apart, empty tables kept) and linear in the number of keys. Rejected because it has
  no bound at all on nesting and its memory is quadratic in depth: ten thousand nested
  inline tables is a fatal out of memory at 4.8 GB, and its parser has no public form
  that yields expressions, so the guard would be a hand-written lexer of TOML in the
  loader. That is more code to get wrong than the `unstable` parser of the chosen
  library. Its linear breadth is the one thing it does better, and `MaxTableKeys`
  removes that difference at the sizes a pack has.
- **A TOML parser written for this module.** Rejected: the grammar (strings of four
  kinds, dates, dotted and quoted keys, arrays of tables, inline tables) is where
  parsers go wrong, and a hand-written one would need the fuzzing the libraries have
  had for years.
- **JSON or YAML as the format.** Decided against in `spec-config.md` 6.1.
- **Go-only configuration.** Always supported (`spec-config.md` 2.2) and unaffected;
  it gives no data-pack file for a non-programmer.
- **Strict decoding by the library** (`DisallowUnknownFields`, struct decoding). Not
  used: it would tie the root structs to the library's field mapping and error text
  (`spec-config.md` 6.1, open question 2); the hand mapping gives the exceptions of
  6.2.2 and paths in the loader's own form.

## Open points

1. The quadratic breadth of `pelletier` is upstream. If a later release fixes it,
   `MaxTableKeys` can be relaxed to `MaxRecipes` times a small factor and the cost
   table re-measured; nothing else changes.
2. The `unstable` package is documented as unstable. The guard is the only user; the
   fallback is in decision 5.
3. The status stays Proposed until the first code that imports the library (Phase 5 of
   the plan of Stage 0), whose pull request carries this justification
   (architecture.md 5.3) and `FuzzLoad` with a nested seed and a flat 10 000-key seed.
   *Note, 2026-10-23.* Accepted: `config/guard.go` imports `unstable` and
   `config/fuzz_test.go` holds `FuzzLoad`; the condition of Open point 3 is met.
4. Whether a pack authoring tool should write documents already split under the
   limits is the data-pack deliverable's, not the loader's.

## Related

- [`doc/rules/architecture.md`](../rules/architecture.md) 5.2, 5.3
- [`doc/specs/spec-config.md`](../specs/spec-config.md) 4.1, 6.1, 6.3
- [ADR 0002](0002-persistence-by-checkpoint.md) decision 8 (the schema number)
- `doc/notes.md` N-019, N-015
- `doc/plans/plan-phase-0-foundation.md` Phase 1 item 4, Phase 5
