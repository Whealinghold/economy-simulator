# Plan: Stage 0 foundations

**Status: Stage 0 is implemented and tested, not yet committed, and awaits
review.** The exit gate of Phase 7 was run by hand on 2026-10-13 (no CI exists,
open item O3), on one architecture: `gofmt`, `go vet`, `staticcheck`,
`go test -short -race ./...`, the full run with `-coverpkg=./...`, `go mod tidy`
and `govulncheck` are clean; merged statement coverage is 99.9% for the root
package, 99.8% for `config/`, 100% for `internal/checked` and for
`internal/ordering`; the six fuzz targets ran five minutes each with no crasher
(N-019). The two statements
still uncovered carry a comment that says why (testing.md 9.5). The minor gaps
of the audit were closed on 2026-10-13 (guards, count-bound and sentinel tests,
loader and fuzz tests, doc and note edits; N-224 lists the guards); what
remains open is the owner's review and acceptance of ADR 0009 and ADR 0002,
CI (O3) and the Stage 1 deferrals of the rule ledger, and nothing is
committed yet.
Sequences Stage 0 of [roadmap-v1.md](../roadmap-v1.md)
into seven phases. It is built against `doc/rules/architecture.md` (the layout
of 2.6, the catalogue-file and append-only rules of 2.10, the `internal/`
boundary of 2.9), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR
or an answer the owner gave on 2026-10-12 (section "Decisions this plan rests
on"). Where a decision is still owed, Phase 1 makes it in the document that
owns it, before the code that depends on it.

This was written before any Go code existed. It is the first code change, so
the housekeeping of note N-019 fell on it. The text below is the plan as it was
sequenced; the status above is where it stands.

Spec codes: CUR currency, CFG config, WLD world, OBS observability, GDS goods,
PRO production, AGT agents, CAL calendar, CHK checkpoint, RES resources, WEL
welfare, BEL beliefs, SPN spawner, SCL scale, VIA viability, REG regions, CLR
clearing, REF reference chain. `FR-`/`NFR-` IDs are the requirement IDs of
those specs.

## Scope

**In scope.** The seven rows of roadmap Stage 0, cut to what the slice (one
region, the reference chain, no modules) needs:

- `go.mod` (`go 1.25`), `doc.go`, `value.go`
- `internal/checked`, `internal/ordering`, `internal/logkeys`
- `config.go`, `bounds.go`, `defaults.go`, `validate.go`, `hash.go`,
  `errors.go` for the slice's sections
- `config/` (the TOML loader)
- `event.go`, `observer.go`, `option.go`, `random.go`
- the spec, rule, ADR, note and roadmap edits those files depend on (Phase 1)

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| `WithStrategy`, `WithSpawner`, `WithClearing` and the name checks `ErrUnknownStrategy`/`Spawner`/`Clearing` | Roadmap Stage 1, with their interfaces (decision D2) |
| The five viability rules (CFG 7.6), `ViabilityError`, the solver-dependent warm-start checks, the warm-start check that a job's set of virtual start offsets is not empty (`ErrWarmStartJob`, CFG 7.4.1; it needs the calendar's season lookup), `internal/steady` | Roadmap Stage 1, `viability.go` row (decision D4) |
| `World`, `NewWorld`, `Step`, anything that draws, trades or produces | Roadmap Stage 1 |
| The observability tests that need a world (logs, non-interference, cardinality, counterpart, golden rendering, emission order, run lifecycle; the call-site and full-`Step` allocation limits) | Roadmap Stage 1 "Done when"; Phase 1 adds the missing lines |
| `WithReseed`, `WorldRestored`, `ErrOptionContext` behaviour | Roadmap Stage 2, `checkpoint.go` row |
| `Edges`, `RegionPrices`, `AgentRelocated`, `EdgeStatusChanged`, `YieldShocked` | Roadmap Stage 2 (REG, INT rows) |
| Every module struct, module field, module event, module log key and module validation rule | Each module's own PR, appended (architecture.md 2.10.5) |
| The denomination ladder and formatter (FR-CUR-008 to 019, NFR-CUR-001) | No stage yet. Phase 1 adds a roadmap row or records why not |
| The other spec rewrites N-214 lists (EXP 7, CLB 4.2.1, TEC 4.4, and the rest) | N-214 stays open for them |
| Glossary terms of Stage 1 files | Roadmap Stage 1; each file adds its terms before using them |
| `go.work`, `otelobserver/`, `examples/` | Roadmap Stage 2. `go.work` is added with the second module |
| CI configuration | Not in the roadmap. The commands of `AGENTS.md` are run by hand until it exists (open item O3) |

## Decisions this plan rests on

Decided by the owner on 2026-10-12, for this plan:

- **D1.** The owed spec decisions (WLD 7 keyed streams, the TOML library ADR,
  the module-table spelling) are Phase 1 of this plan, not prerequisites
  outside it.
- **D2.** The three registration options ship in Stage 1 with their
  interfaces. Stage 0 ships `WithLogger`, `WithObserver`, `WithAudit`,
  `WithTargetSpread` and `WithSeed`.
- **D3.** Slice fields only. A struct or event declares the fields the specs
  give the core, that is every field not owned by a module of
  `Config.Modules`; this includes every key the Stage 0 packs (`one`,
  `second`) set. Each module's PR appends its own fields, never inserts
  (architecture.md 2.10.5).
- **D4.** The viability rules and the solver-dependent warm-start checks ship
  with `internal/steady` in Stage 1.
- **D5.** An append that is off leaves no trace. A module that is off, and an
  appended field at its off value, contribute nothing to the configuration
  hash or to the golden rendering of an event. `testdata/golden/off/` then
  stays unchanged when a module is added (testing.md 7.1.1), by construction.
  The exact encoding and rendering rule is settled in the spec edit of
  Phase 1, not here.

Decided by documents already:

- TOML is the file format (CFG 6.1). The library is not chosen (N-019).
- Section structs live in `config.go` (architecture.md 2.6: "Config and every
  section struct"). Architecture.md wins over the spec rows that name other
  files.
- `WithSeed` lives in `option.go` (architecture.md 2.6, WLD 23), not
  `random.go` as the roadmap row says.
- Randomness is keyed streams (ADR 0008, architecture.md 7.3). The exact form
  is owed (Phase 1).
- `internal/checked` is the one shared 128-bit helper (N-019, 2026-10-13).
- The loader maps a generic TOML tree by hand and checks unknown keys itself
  (CFG 6.1); the hash is a hand-written encoding (CFG 8.2).

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** `Modules` is declared as a struct with no fields at Stage 0. Each
  module PR appends its pointer in the order of CFG 3.1. This is what "present
  and all nil" means under D3.
- **A2.** `internal/logkeys` holds only the keys the slice's log lines use
  (33), not the whole table of observability-and-logging.md 3.3.2. No line is
  logged in Stage 0; the package is built now because the roadmap row lists
  it and Stage 1's first log line needs it.
- **A3.** Every typed ID an event or the configuration needs is declared in
  `value.go` at Stage 0, even where a spec's code-home row names a Stage 1
  file. `Side` is declared in Phase 6 with its first user, `event.go`, in the
  home Phase 1 records, so Stage 1 does not move it.
- **A4.** `K(t, id)` follows the detailed form of RES 5.3.3 (add the increment
  once more before the mix), which FR-RES-019 cites. FR-WLD-091's shorter
  wording is corrected in Phase 1.
- **A5.** The signatures of `internal/checked` (names, signedness, which
  rounding variants) are the implementer's, within architecture.md 3.5. No
  spec fixes them. Callers need: multiply-divide with a 128-bit intermediate,
  quotient and remainder, an `ok` result, truncation toward zero by default.
- **A6.** The parameter-registry test carries an explicit list of the scopes
  not built yet, as the staged form Phase 1 writes into testing.md 6.7 and
  CFG 9.2 allows. A row whose scope is on the list is skipped. The test fails
  if a listed scope has a field in the code, so a stale entry cannot hide.
- **A7.** Options are tested from an internal test (`options` is unexported,
  testing.md 3.7 allows this for unexported logic) until `NewWorld` exists.
- **A8.** One PR per phase, in phase order, except Phase 2, which is two: the
  first may start while Phase 1 is in review; the second adds `Key` and the
  stream key after Phase 1 is merged. Section numbers of WLD 7 cited below
  are today's; Phase 1 may renumber them.
  Phases 3 to 6 do not start before Phase 1 is merged.

## Phase 1: settle the owed specs

**Why first:** `random.go`, `config.go`, `hash.go`, `config/`, `event.go` and
two Stage 0 exit tests depend on text that does not exist yet. This phase
changes specs, rules and notes, so it starts by reading `doc/notes.md` in full
and quoting its tag, and every edited rule file or notes file gets a fresh tag
(`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **WLD 7 rewritten for keyed streams** (N-214, ADR 0008 open point 1).
   Fix: the exact keying form; the type and width of `site`; how a signed
   tick and ID enter the key; the site constants as fixed values that do not
   depend on a row's position, so the slice's sites can be defined now and
   later sites appended; the `id` of each slice row (D1, D3, D5, D7, D8, D9,
   D10), including D8's two components against a one-`id` key; whether
   construction and a between-ticks call before the first step, both at tick
   0, are told apart by site alone; ADR 0008 open point 2 (the per-order key
   of D7); test vectors for the stream key and for `K(t, id)`. Open point 3
   concerns rows outside the slice and stays open. Reword FR-WLD-060, 062,
   065, 066 and 091 and the stale text of WLD 4.1, 4.4, 7.6, 12 and open
   questions 4 and 7. Regenerate the model scripts that hold the vectors
   (`doc/specs/models/wld/seed.py`, `wld/worked.py`, and `wld/stream.py`,
   which already models a keyed stream and is still labelled "not adopted")
   and archive their output, so there is one source of vectors.
2. **The reseed rule** `seed' = finalise(seed xor finalise(salt))` with
   vectors, replacing the old rule and vectors of CHK 6.8 and of
   `doc/specs/models/chk/worked.py`. Stage 0 builds the pure function only;
   the option is Stage 2.
3. **A stale marker** on the passages of WLD and CHK that this phase does
   not rewrite (WLD 13, 16.5, 17; the rest of CHK, FR-CHK-051): one line
   each, "stale under ADR 0008, see N-214". The rewrite of WLD 7 and CHK 6.8
   would otherwise leave those two specs contradicting themselves. The other
   specs N-214 lists are untouched; the note already tells their next editor
   what to apply.
4. **ADR 0009: the TOML library** (N-019). The justification architecture.md
   5.3 asks for: what it provides, why the standard library is not enough,
   maintenance status, licence. The spec needs only a generic tree with
   integer and float told apart. Check, and record in the ADR: it never
   panics on malformed input, it survives or bounds deeply nested input, and
   it supports `go 1.25`.
5. **The module-table spelling and the estates switch** (N-102, CFG open
   question 11). One rule for module tables in a file, applied to the specs
   and the fixtures together. Decide what `[module.estates]` maps to in Go:
   CFG 3.1 lists 24 `Modules` pointers and none is estates, yet CFG 6.2 says
   the table switches a module on. The answer also fixes whether `ReliefPaid`
   is a slice event. Checked by a text search: no spec or fixture is left
   with the old spelling.
6. **The amendments D2 to D5 need.**
   - D5: CFG 3.1, 8.1 and 8.2 (how an off module and an off-valued appended
     field encode to nothing; the presence byte per module goes), the golden
     rendering of OBS 9.5.2, FR-OBS-080 and REF 8.5 (the same for an appended
     event field), and a line in architecture.md 2.10.5 that an append must
     obey it. The rule also covers the core sections that arrive after the
     slice (`Edges`, `RegionPrices`, Stage 2): empty, they encode to nothing.
   - D3: FR-OBS-003 and 004 and OBS 3.3.4 in a staged form: the catalogue
     test checks the events that are built, in the order of OBS 4.1.
   - A6: architecture.md 9.7, testing.md 6.7, CFG 9.2 and FR-CFG-028 in a
     staged form: a row whose scope is not built yet is skipped by a named
     list. Architecture.md 9.7 states the rule the other three repeat, so it
     is amended first.
   - D2 and D4, in the roadmap: registration options and viability rules
     moved to Stage 1; `WithSeed` in the `option.go` row; "the reseed rule"
     as the function only; the spec list of the config row completed (BEL 11,
     SPN 11, WEL 4.2 and 12, SCL 3.1, CFG 5 and 9, VIA 8.1); "basis-point
     helpers" dropped from the `value.go` row, since no spec names one; exit
     tests added for events, options, `logkeys` and `doc.go`; the glossary row reworded
     to "the terms each stage uses, before it uses them"; the world-dependent
     observability tests added to Stage 1's "Done when". Record the OBS stage
     move in `vision.md` section 4 and `spec-roadmap.md` as the roadmap
     requires.
7. **Gaps a Stage 0 file runs into**, each settled where it lives:
   - the loader's limits on untrusted input (CFG 6.3): a nesting-depth bound,
     a bound on the merged size and on the number of documents, a bounded
     read in `LoadFS`, and whether loader errors are capped;
   - error text built from input (CFG 6.3, 7.1): a path segment taken from a
     document is quoted and cut to a stated length, so an error cannot carry
     megabytes or control characters;
   - whether `ErrTooMany` stops the rules that are super-linear in a count
     (CFG 7.1, 7.5);
   - label bound: FR-GDS-021 says 64 bytes, `MaxLabelBytes` is 256;
   - `JobSpec.InProgressBP` against the file key `in_progress`, and the other
     places the fixed key rule has exceptions (warm start nesting, yield
     `seasons`);
   - the home of the six loader sentinels (`ErrUnknownKey` and the rest), of
     `Side`, and of the `WarmStart` types (`config.go` by architecture.md
     2.6, `viability.go` by VIA and CFG 14), so Stage 1 does not move them;
   - `WarnRun` against `warnRun` (OBS 7.3, 20);
   - whether the default `Observer` of architecture.md 4.5 is exported;
   - the parameter-owner table rows the registry test cannot map one to one
     (`Scale`, `Upkeep`, `YieldProfile`), and the missing rows for calendar
     months and seasons and the warm-start item types;
   - `HashConfig` is a proposed name (CFG 8.5): adopt or rename;
   - a home for the denomination package, or a recorded reason for none.
8. **Notes.** N-102 closed. N-019 and N-214 updated with what was applied and
   what stays owed. A note for anything found that a later spec must take
   into account.

**Tests:** none; documents only. The regenerated model scripts are run and
their output committed, so the vectors in WLD 7 and CHK 6.8 can be checked
against them.

**Observability and rollback:** none, because no code changes. A wrong edit is
reverted like any document change; nothing depends on it until Phase 2.

**Acceptance:** each of the eight deliverables is merged; WLD 7 holds numeric
vectors for the stream key and for `K`; ADR 0009 is in `doc/adr/README.md`;
the roadmap's Stage 0 and Stage 1 rows match D2 and D4; CFG 8.2 states the
rule of D5 with a worked example of an all-off configuration, and OBS 9.5.2
states it for an appended event field with a worked rendering; ADR 0009 has
a row in `doc/documentation_index.md`; no open item in this plan still
blocks Phases 3 to 6.

## Phase 2: the module, the value types and the leaf helpers

**Why here:** everything later imports this phase. Only `Key` and the stream
key depend on Phase 1 (their form and vectors); they land last (A8).

- `go.mod` with `go 1.25` and no dependency.
- `doc.go`: the package name `economysim` (architecture.md 2.2), the
  overview, and the glossary seeded with the terms this phase uses. Each
  later phase adds its terms before using them (architecture.md 3.2, N-017).
- `value.go`: `Money`, `Price`, `Quantity` (`int64`); `CommodityID`,
  `RoleID`, `RecipeID`, `RegionID` (with `NoRegion`), `AgentID` (`int64`,
  with `NoAgent`), `NoCommodity` (A3). Arithmetic, comparison and formatting only (architecture.md 3.3). The
  roadmap's "basis-point helpers" are named in no spec: build one only when
  a Stage 0 caller needs it.
- `internal/checked`: the 128-bit product and ratio over `math/bits` (A5).
- `internal/ordering`: the finaliser, `Weyl32`, `Weyl2`, `Bucket`, `Key`
  (A4), then the stream key.
- `internal/logkeys`: key strings only, no import, no alias (A2).
- The layout sections of architecture.md 2.6 and `AGENTS.md` gain
  `doc/plans/` in this PR, the first code change (N-019), with a fresh tag.
- The roadmap status of each Stage 0 row moves in the PR that starts or
  finishes it, here and in every later phase (roadmap, "Updating this
  file").

**Tests:** unit only; every package here is pure.

- `value.go`: table tests of arithmetic and formatting at the boundaries
  (zero, negative, the bounds). FR-CUR-001, 003.
- `internal/checked`: table tests at the overflow edge, for each sign
  combination and each rounding variant, checked against `math/big` in the
  test only. A fuzz test that it never panics and agrees with `math/big`.
- `internal/ordering`: the vectors the specs hold: the four seed rows of WLD
  7.1.3 for the finaliser, `Weyl2` of 0, 1, 2 and 17, `Weyl32` and `Bucket`
  for IDs 10 to 15, and the Phase 1 vectors for `Key` and the stream key.
  `Bucket` in `[0, P)` for every `P` from 1 to 1 000 (FR-WLD-080), the
  evenness test of `Weyl32` (WLD 9.1.1), fuzz tests for the bucket and for
  the keyed tie (NFR-WLD-005). FR-WLD-090, 091, FR-RES-019.
- `internal/logkeys`: a `Guard:` test that every constant is a row of the
  table in observability-and-logging.md 3.3.2, read by a plain text parse
  (FR-OBS-073).

**Observability and rollback:** none, because nothing here logs, emits or
holds state. Rollback is reverting the PR; nothing outside the module
imports it.

**Acceptance:** `gofmt -l .` prints nothing; `go vet`, `staticcheck` and
`go test -race ./...` pass; every vector above passes; each `internal/`
package is at 90% statement coverage or more, read from
`go tool cover -func=coverage.out` and quoted per package in the PR.

## Phase 3: the configuration types, bounds, defaults and hash

**Why here:** validation, the loader, the events and the options all name
these types. It needs Phase 2's typed IDs and Phase 1's answers on the
estates switch, the encoding rule of D5, the label bound and the hash name.

- `config.go`: `Config` and the slice's section structs (`World`, `Calendar`,
  `Commodity`, `Recipe`, `YieldProfile`, `Role`, `Region`, `Group`,
  `WarmStart` and its item types), slice fields only (D3), and `Modules`
  (A1). Plain structs, no tags, no embedding (FR-CFG-001). Canonicalisation
  (CFG 3.4) and the assignment of typed IDs (CFG 3.3).
- `bounds.go`: the roughly 30 constants of CFG 4.1 the slice uses. Doc
  comments say they are provisional (N-015, N-061).
- `defaults.go`: `DefaultWorld`, `DefaultCalendar`, `DefaultCommodity`,
  `DefaultRecipe`, `DefaultRole`, `DefaultRegion`. No doc comment calls a
  default stable (N-027).
- `errors.go`: the sentinels the Stage 0 rules return and the type
  `ValidationError`, each added with its first user. `ViabilityError` and
  the sentinels of N-075 that nothing in Stage 0 returns wait for Stage 1.
- `hash.go`: the canonical encoding of CFG 8.2 as amended for D5, written by
  hand over `encoding/binary` and `crypto/sha256`, with its version byte. No
  reflection, no maps.
- One valid reference configuration built in Go, in a `_test.go` file of the
  root's external test package. Phase 4 reuses it as the base of the failing
  cases. The "file and Go agree" test of Phase 5 lives in the same package,
  which may import `config/`, so nothing is exported for tests
  (architecture.md 10.2).

**Tests:** unit.

- Canonical order: a configuration written in two orders gives equal typed
  IDs and an equal hash (FR-CFG-005).
- "Every result-affecting field moves the hash": walk the fields by
  reflection in the test, change each in turn, assert the hash moves unless
  the field is on the presentation-only allow-list, in which case assert it
  does not move; and assert the allow-list equals the fields whose doc comment says presentation only (FR-CFG-026, CFG
  8.3). This is a Stage 0 exit test. The other half of FR-CFG-026 (seed,
  logger and observers do not move the hash) holds here by signature, since
  the hash takes a `Config` only; its world half is Stage 1.
- Presence: `WarmStart` nil and empty hash differently.
- A `Guard:` test that pins the hash of the all-off reference configuration
  to a constant. A module PR that moves it has broken D5; this is the
  earliest signal, before any golden file exists.
- The parameter-registry test, checks 1 and 2 of CFG 9.2 (row has a field,
  field has a row), with the list of A6 (FR-CFG-028, 029; testing.md 6.7).

**Observability and rollback:** none, because the configuration code logs
nothing and emits nothing (CFG 10.2: warnings are logged by `NewWorld`,
Stage 1). Rollback is reverting the PR. The hash first reaches an output in
Stage 1 (the golden header and `WorldCreated`), so its encoding can still be
corrected until then; the version byte is there from the first commit so
that a later change is visible.

**Acceptance:** the two registry directions pass over the built scopes; the
hash test passes with an allow-list of exactly the six presentation fields of
the slice; the pinned all-off hash is committed; the root package is at 90%
or more.

## Phase 4: validation

**Why here:** it needs the types, bounds and sentinels of Phase 3. The loader
(Phase 5) is tested against it, and the Stage 0 exit test counts its rules.

- `validate.go`: the framework of CFG 7.1 (`Config.Validate`, every error
  joined with `errors.Join`, sorted by path then rule, capped at 100 with
  `ErrTooManyErrors`, pure and bounded) and `Config.Warnings` (7.2). Count
  bounds are checked first, and the rules that are super-linear in a count
  run only when it holds, as Phase 1 item 7 settles.
- The structural rules of CFG 7.3 for the slice (11 rows, 14 sentinels).
- The contributed rules of CFG 7.4.1 for the slice, less the three name
  checks (Stage 1, D2), the five viability rows and the solver-dependent
  warm-start checks and the empty-virtual-start-offsets check of
  `ErrWarmStartJob` (Stage 1, D4), and the run-time order-limit row. The
  derived overflow checks of CFG 4.2 that apply.
- The producibility fixpoint of CFG 7.5.
- The slice's warnings of CFG 7.4.2 that need no solver.

Before coding, list the rule rows in the PR description as built, deferred
to Stage 1 (with the reason) or module. That list is the checkable form of
the exit criterion.

**Tests:** unit, table-driven, the bulk of Stage 0's tests.

- One case per rule whose configuration breaks only that rule, asserted with
  `errors.As` on `*ValidationError` and `errors.Is` on the sentinel, with the
  path checked (testing.md 3.9). Each case starts from the reference
  configuration of Phase 3 and names the field it breaks.
- One case per warning.
- The framework: ordering, the cap, determinism across runs, and a
  configuration over a count bound returns `ErrTooMany` without running the
  super-linear rules. An oversized ID and an ID with control characters give
  an error whose path is quoted and cut (Phase 1 item 7).
- `FuzzValidate`: its inputs are primitives (which field, which value, how
  many entities) that perturb the reference configuration, because `f.Add`
  takes primitives and the loader is a later phase. It asserts: never
  panics, at most 101 errors, equal input gives equal output (NFR-CFG-002,
  003). The calendar and yield-profile validators are fuzzed through it
  (NFR-CAL-002, NFR-PRO-004); their lookups are Stage 1.
- The parameter-registry test, checks 3 and 5: each default passes
  validation, and each field is named by a failing case (the test reads the
  field names from the case table). Check 4 (spec default equals constructor
  default, a SHOULD) if the owner tables parse cleanly after Phase 1.

**Observability and rollback:** none, because a validation error is returned
and never logged (architecture.md 8.3), and warnings are values until
`NewWorld` logs them in Stage 1. Rollback is reverting the PR.

**Acceptance:** every rule row marked "built" has a failing case that fails
for that rule alone; `FuzzValidate` runs with its seed corpus; the registry
test passes all MUST checks; coverage holds at 90%.

## Phase 5: the loader

**Why here:** it needs ADR 0009, the module-table rule and the input limits
(Phase 1), the `Default` constructors (Phase 3) and `Validate` to prove a
loaded pack is the configuration the specs mean (Phase 4).

- `config/`: `Parse(docs ...[]byte)` and `LoadFS(fsys fs.FS, names
  ...string)` (CFG 6.5). It never opens a path.
- The limits before any decoding: each document's size (`MaxDocumentBytes`),
  and the document count, merged size, nesting depth and bounded read that
  Phase 1 item 7 fixes. Then the merge of documents in order (CFG 6.4), the
  `schema` check on the merged tree, and the hand-written mapping from the
  generic tree to the root structs, starting each entity from its `Default`
  constructor (CFG 5.4).
- Strict decoding done by `config/` itself: an unknown key at any depth, a
  float or string where an integer is needed, a missing required field, each
  with its path (CFG 6.3). Path segments taken from the document are quoted
  and cut as Phase 1 item 7 fixes.
- Module tables: every module is unbuilt at Stage 0, so a module table is
  rejected by the rule Phase 1 fixed. A module PR adds its table.
- The dependency is added to `go.mod` here, and the PR description carries
  the justification of ADR 0009 (architecture.md 5.3).

**Tests:**

- Unit: one case per rejection of CFG 6.3 and per merge rule of 6.4, with the
  sentinel and the path asserted; one case at each input limit (too large,
  too many documents, too deep, an oversized or control-character key in an
  error path). The key-mapping rule and its exceptions. `LoadFS` over
  `testing/fstest.MapFS`.
- `FuzzLoad`: never panics, an accepted document loads again to an equal
  `Config`, a rejected one returns an error (FR-CFG-024, NFR-CFG-002). Seeds:
  the layered documents of the pack `one` and of `second/`, and a deeply
  nested document. Corpora live in `config/testdata/fuzz/` (testing.md 7.5).
- Integration (skipped with `-short`), with `os.DirFS` over
  `testdata/chains`: the pack `one` (`base` + `one` + `warm-one`) and the
  pack `second` load, pass the Stage 0 `Validate`, and give a stable hash.
  "File and Go agree" (CFG 3.6), in the root's external test package: the
  pack `one` equals the reference configuration of Phase 3, field for field.
  The six `defects/` overlays load over the base layers REF names for each.
  `d1` to `d5` break viability rules (`cycle_not_productive`,
  `unproducible_in_realm`, `role_cannot_work`, `labour_short`, `storage_gap`),
  so their failing assertions are Stage 1's (D4). `d6_floor_full` gives the
  warning `cost_floor_at_full_cost`, which needs no solver and is built here:
  its test loads the overlay over `base` and `one`, asserts no error from
  `Validate` and that `Warnings` holds that rule (FR-REF-016).

**Observability and rollback:** none, because the loader returns errors and
logs nothing. Rollback is reverting the PR, which also removes the only
third-party dependency; no other package imports `config/`.

**Acceptance:** `FuzzLoad` runs with its seed corpus; both packs load and
validate; each input limit has a passing rejection case; `go mod tidy`
produces no diff; `config/` is at 90% or more.

## Phase 6: events, the observer, the options and the random stream

**Why here:** the events name the typed IDs of Phase 2 and their catalogue
depends on Phase 1's answers (the staged FR-OBS-003, the estates switch).
`WithTargetSpread` needs the bound and sentinel of Phase 3. `random.go` needs
the stream key, which needs the rewritten WLD 7.

- `event.go`: the sealed `Event`; `Side` (A3); the slice's events of OBS 4.1 (21, or 22
  with `ReliefPaid` if Phase 1 says so), slice fields only (D3); `Names`,
  `Amount`, `HaltRow`; `Reason`, `Cause`, `Kind` with the constants the slice
  can reach.
- `observer.go`: `Observer`, `ObserverFunc`, the default observer that
  discards with its compile-time assertion (architecture.md 4.5), the slice
  of observers with one delivery loop, the "observer supplied" flag, and the
  unexported run-warning constant. The doc comment states the calling
  contract (observability-and-logging.md 4.4, 7.2).
- `option.go`: `Option`, the unexported `options`, `WithLogger`,
  `WithObserver`, `WithAudit`, `WithTargetSpread`, `WithSeed`. A nil logger
  or observer means the default; `WithTargetSpread` records "given" apart
  from 0 and returns `ErrOutOfBounds` outside 0 to 9 999 (FR-WEL-026).
  `WithAudit` is a flag; its behaviour is Stage 1.
- `random.go`: the seed, the one `rand.PCG` and the one `*rand.Rand` built
  once, the site constants of the slice, the keyed reseed, the bounded draw
  of WLD 7.2.1, and the pure reseed function.

**Tests:** unit, plus the allocation limits as ordinary tests.

- Event shape, by reflection over the declared events: `Tick int64` first; no
  pointer, map, float, function, embedded type or `time` value; typed IDs for
  configured entities; fields in the order of OBS 4.1 for the slice subset
  (FR-OBS-001, 004, 005, 006). The catalogue test in the staged form of
  Phase 1: each declared event is a row of the tables in OBS 4.1 and
  observability-and-logging.md 4.3.
- Delivery: several observers receive each event in the order given; a nil
  observer is ignored; the flag is false with none (the option halves of
  FR-OBS-024, FR-WLD-003, FR-OBS-090).
- `testing.AllocsPerRun`: delivery with no observer allocates nothing, and a
  keyed reseed followed by a draw allocates nothing (testing.md 8.2). A
  benchmark beside each for the trend. These cover the delivery loop and the
  reseed only; the call sites of the tick loop and a full `Step`
  (observability-and-logging.md 8.2, NFR-OBS-001, NFR-WLD-001) are Stage 1.
- Options, from an internal test (A7): each option's effect on `options`, the
  bound and error of `WithTargetSpread`, "last `WithSeed` wins", default seed
  0.
- Random: the keyed vectors of the rewritten WLD 7 (a Stage 0 exit test); the
  draw stays in `[0, n)` and takes one word, with a fuzz test (FR-WLD-061,
  NFR-WLD-005); two streams of different sites, ticks or IDs do not depend on
  each other's draw count; the reseed vectors.
- A `Guard:` test, a text search over the non-test files of the root,
  `internal/` and `config/`, that none uses the global `math/rand`
  functions, `time.Now`, `time.Since`, `crypto/rand`, `slog.Default`,
  `fmt.Print*`, `log.Print*`, the builtins `print` and `println`,
  `os.Stdout` or `os.Stderr`, and that there is no `init`
  (architecture.md 7.1 to 7.3, 11.1, 11.2; NFR-WLD-003).

**Observability and rollback:** this phase is the observability surface, so
the checklist of observability-and-logging.md section 9 is answered in full
in the PR. No log line is emitted yet (nothing runs), no metric exists, and
the events are declared but not emitted. Rollback is reverting the PR; the
keying constants become part of the golden output only when Stage 1 records
a golden file, so until then they can still be corrected.

**Acceptance:** the keyed vectors pass; both allocation tests pass at zero;
the catalogue test passes in its staged form; coverage holds at 90%.

## Phase 7: the exit gate

**Why last:** it checks the stage as a whole and hands over to Stage 1.

- Run, on the whole module: `gofmt -l .`, `go vet ./...`, `staticcheck
  ./...`, `go test -short -race ./...`, the full run with
  `-covermode=atomic -coverpkg=./...`, `go mod tidy`. Read the per-scope
  coverage of testing.md 9.1 from `go tool cover -func=coverage.out` and
  record it.
- Run each fuzz target for a fixed time (five minutes each) and commit any
  crasher with its `Fix:` comment (testing.md 7.5). With no CI, this is the
  only fuzzing beyond the seed corpora.
- Run `govulncheck ./...` for the one dependency, since no CI does.
- Check the roadmap's "Done when" line by line, as corrected in Phase 1.
- The reviewer's traceability check of testing.md 4.6, by text search: every
  ID in a `Requirement:` line exists in a spec, and every requirement this
  plan lists as covered in full is cited. A requirement covered in part is
  listed in the PR as partial, with the stage that completes it.
- `doc.go` glossary holds every term the Stage 0 code uses.
- Confirm every Stage 0 roadmap row is **done** with its commit. N-019 and
  N-214 updated: what the first code settled, what stays owed.

Not gating, worth doing if time allows: a mutation pass over
`internal/checked` and `internal/ordering` (testing.md 9.4, a SHOULD), with
`gomu` inside Docker only, `--cpus` and `--workers` at no more than a quarter
of the host's processors and `--memory` equal to `--memory-swap`.

**Tests:** no new tests; this phase runs the suite and fills gaps the
traceability check and the fuzz runs find, at the unit level.

**Observability and rollback:** none, because no behaviour changes. A gap
fill is a test, reverted like any other.

**Acceptance:** every command above is clean; every line of the corrected
"Done when" is ticked with the test that proves it; Stage 1 can start.

## Rule ledger

The checkable list of Phase 4, as built (read from `validate.go`, CFG 7.3,
7.4.1, 7.5). Built means a code path returns the sentinel and a failing case
asserts it.

| Rule (CFG) | Status | Reason or owner |
|------------|--------|-----------------|
| 7.3: `ErrInvalidID`, `ErrDuplicateID`, `ErrUnknownCommodity`, `ErrUnknownRole`, `ErrUnknownRecipe`, `ErrUnknownRegion`, `ErrRecipeNoOutput`, `ErrOrphanCommodity`, `ErrOutOfBounds`, `ErrOverflow`, `ErrTooMany`, `ErrNoRegions` | built | |
| 7.3 / 7.5: `ErrUnproducible` (fixpoint, names one cycle) | built | |
| 7.3: `ErrModuleDependency` | module | declared in `errors.go`, no Stage 0 rule returns it; first user is a module PR (recipe with `Draws` while resources is off, `spec-production.md` 10.2) |
| 7.3: `ErrSelfEdge`, `ErrUnknownEdge` and the region-graph row | module | regions module not built (Stage 2) |
| 7.3: `ErrUnknownResource`, `ErrResourceShape`, `ErrCurve` | module | resources |
| 7.3: `ErrOutsideNodeInhabited`, `ErrCrossedQuotes`, `ErrNotAGate` | module | external sector |
| 7.4.1: resolution floor of the cheapest tradable reference price; lowest target ratio at least 10 000; `ErrBasketMismatch` | built | |
| 7.4.1: `TicksPerYear` and `ExpectedWorkTicks` bounds; calendar bounds, `ErrCalendarTick`, `ErrCalendarSeasons` | built | |
| 7.4.1: welfare window (`MaxWelfareWindow`, `WindowCycles`, `MinWorkShare`), traineeship weight one, `MinAgents`, welfare overflow | built | |
| 7.4.1: recipe shape (duration, quantities, wear, yield, trainee yield, item and tool counts, duplicate list entries, `ErrCostShares`, `ErrToolNotDurable`, `ErrEligibleHasTool`) | built | |
| 7.4.1: yields (`ErrUnknownSeason`, unknown region or recipe, duplicate profile), upkeep, `StartStock`, `Trades`, `Recipes` references and bounds | built | |
| 7.4.1: spawner and belief parameter bounds and orders (`ladder_order` included) | built | |
| 7.4.1: warm start (`ErrWarmStartPrice`, `ErrWarmStartJob` recipe on role list and share 0 to 10 000, wage times longest duration, duplicate wage, overflow, unknown reference), money at start | built | |
| 7.4.1: `ErrWarmStartJob` sub-rules "group shares exceed 10 000" and "empty set of virtual start offsets" | deferred to Stage 1 | needs the calendar's season lookup (`spec-viability.md` 8.5.2), `viability.go` (D4) |
| 7.4.1 / 7.6: `ErrNotViable`, the four viability rows (`unproducible_in_realm`, `role_cannot_work`, `cycle_not_productive`, `labour_short` and `storage_gap`) and `ViabilityError` | deferred to Stage 1 | needs the `internal/steady` solver (D4) |
| 7.4.1: `ErrUnknownStrategy`, `ErrUnknownSpawner`, `ErrUnknownClearing` | deferred to Stage 1 | `NewWorld` only (7.1.4), with the interfaces (D2) |
| 7.4.1: want rows (`ErrWantItems`, `ErrWantDurable`, `ErrUnknownWant`) | module | demand |
| 7.4.1: region-graph row and `MaxEdges` | module | regions |
| 7.4.1: order-limit row (clearing 3.2) | built | covered by `MaxPrice`, `MaxQuantity` and the other bounds themselves |

Count bounds checked in phase 1 (`ErrTooMany`, path in brackets; each has an
over-by-one case, and each non-huge one an at-the-bound case, in
`validate_counts_test.go`):

| Level | Bounds |
|-------|--------|
| Top level, constant work | `MaxCommodities` [commodities]; `MaxRecipes` [recipes]; `MaxRoles` [roles]; `MaxRegions` [regions]; `MaxGroups` [groups]; `MaxYields` [yields]; `MaxMonths` [calendar, months]; `MaxSeasons` [calendar, seasons]; `MaxUpkeepItems` [world, upkeep]; `MaxSeedPrices` [warm_start, prices]; `MaxRegions` [warm_start, wages]; `MaxGroups` [warm_start, groups]; `MaxRegions` [warm_start, pools] |
| Nested, walked only if every top-level count holds | recipe inputs, outputs (`MaxRecipeItems`), requires (`MaxRequires`); role recipes (`MaxRecipes`), upkeep (`MaxUpkeepItems`), trades and start stock (`MaxCommodities`); region extra upkeep (`MaxUpkeepItems`); yield seasons (`MaxSeasons`); warm group stock (`MaxCommodities`) and jobs (`MaxRecipes`) |
| Not checked at Stage 0 | `MaxDraws`, `MaxEdges`, `MaxRegionPrices`, `MaxWants*` (module or regions-slice specs not built) |

## Security review

The module is a library with no network, no file access of its own, no
credentials and no persistence. One trust boundary exists at Stage 0.

- **Configuration documents are untrusted input.** A consumer may load a pack
  written by a third party. Worst case is a crash or unbounded work in the
  host process. Mitigations:
  - size: each document against `MaxDocumentBytes` (16 MiB) before decoding;
    the document count and merged size bounded (Phase 1 item 7);
  - depth: a nesting bound, because a deep tree can overflow the stack in the
    parser, the recursive merge or the unknown-key walk, and a stack overflow
    cannot be recovered. ADR 0009 records how the chosen library behaves;
  - work: count bounds on every list, checked before the rules that are
    super-linear in them; the error cap of 100; `Validate` pure and bounded
    (NFR-CFG-003). The viability solver, the most expensive check, is not in
    this stage;
  - error text: a path segment from a document is quoted and cut, so an
    error cannot carry megabytes of attacker text or terminal control
    characters into a consumer's log;
  - no `panic` on input-derived data (architecture.md 8.1); `FuzzLoad` and
    `FuzzValidate` with a nested seed, a timed run in Phase 7 and committed
    crashers.
- **No path traversal.** `config/` takes bytes or an `fs.FS` and never opens
  a path; the consumer chooses the root.
- **Integer overflow.** Money and quantity products go through
  `internal/checked`; validation bounds are chosen so the rest fit. A wrap
  would be a silent wrong result, not memory corruption, and is a defect
  (architecture.md 3.5).
- **Supply chain.** One third-party dependency, confined to `config/`,
  chosen by ADR 0009 with its licence and maintenance status recorded, and
  pinned by `go.sum`.
- **Randomness is not for security.** The keyed stream is reproducible by
  design. The doc comment on `WithSeed` says so, so no consumer uses it for
  a secret.
- **Observers** run consumer code on the caller's goroutine. A panic
  propagates (architecture.md 12.5). No event carries a secret; the library
  is never handed one.
- **Not applicable:** authentication, authorisation, injection into a query
  or command, secrets handling, cross-tenant exposure. There is no endpoint,
  no query language, no secret and no tenant.

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer,
  and the code of this stage is the first. Semantic versioning (architecture.md 6.6) starts with the first
  tag, which is not in this stage.
- **Document contracts this plan touches.** Requirement IDs keep their
  numbers; Phase 1 rewords requirements and never renumbers one (testing.md
  4.1). The fixture files are respelled only by deliverable 5, with the specs
  in the same change. Rule files and `doc/notes.md` get fresh tags when
  edited.
- **Append-only stays true.** D3 appends and never inserts, as
  architecture.md 2.10.5 requires. D5 is what keeps an append from moving
  the hash or the golden rendering of a world with that module off; the
  pinned all-off hash of Phase 3 guards it from Stage 0 on.
- **What becomes hard to change, and when.** The keying rule and site
  constants and the hash encoding enter the golden files in Stage 1, the
  checkpoint in Stage 2, and dashboards at the first release. Until Stage 1
  records a golden file they may be corrected in a normal PR. The hash
  carries its version byte from the first commit.
- **A half-applied rewrite.** Phase 1 rewrites WLD 7 and CHK 6.8 while other
  passages still describe the positional stream. Architecture.md 7.3 wins
  over them, and deliverable 3 marks each one stale in place.
- **Rollout** is one PR per phase (A8). Each leaves `main` passing every
  command of `AGENTS.md`. A phase is rolled back by reverting its PR; later
  phases depend only on earlier ones.
- **Dependency failure.** The one dependency is a parser linked at build
  time; there is no run-time dependency to be slow or absent.
- **The Go toolchain.** The vectors of WLD 7 were computed on Go 1.25. A
  toolchain change that moves them is a behaviour change and is explained in
  its PR (architecture.md 7.4).

## Testing (summary, testing.md section 11)

The pyramid for this stage is almost all base: unit tests in every package,
six fuzz tests (`FuzzLoad`, `FuzzValidate`, the checked product, the bucket,
the keyed tie, the draw), a handful of integration tests in Phase 5 that load real packs, and no
end-to-end test (nothing runs yet). Requirements touched, by phase:

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-CUR-001, 003; FR-WLD-080, 090, 091; FR-RES-019 | NFR-WLD-005: bucket and keyed tie here, the draw in Phase 6. FR-OBS-073: the constants here, the keys of each record in Stage 1 |
| 3 | FR-CFG-001, 005, 025, 028 | FR-CFG-026: the world half (Stage 1). FR-CFG-006 to 008 and 029: the types and constructors here, their tests in Phases 4 and 5 |
| 4 | The requirement IDs of each rule row marked "built" in the Phase 4 PR (CFG 7.3 and 7.4.1, with the CUR, GDS, CAL, PRO, AGT, BEL, SPN, WEL and SCL IDs those rows cite, FR-WEL-025 among them); FR-CFG-006 to 008, 029, 031; FR-CFG-030 if check 4 is built; NFR-CFG-002, 003 | NFR-CAL-002, NFR-PRO-004: the validators here, the lookups in Stage 1. NFR-CFG-001: measured in the slice (Stage 1) |
| 5 | FR-CFG-019 to 024, 034 | FR-CFG-004: the run half (Stage 1) |
| 6 | FR-OBS-001, 006; FR-WLD-061; NFR-WLD-003; FR-WEL-026 | FR-OBS-003, 004, 005 (staged; values, names table and snapshot rows in Stage 1); FR-OBS-024, 090 and FR-WLD-003 (option halves; the world halves in Stage 1); FR-WLD-060 as reworded; NFR-OBS-001 and NFR-WLD-001 (delivery loop and reseed only; Stage 1) |

The Phase 4 PR fixes the row list, so it is where that phase's IDs are
enumerated; this table does not guess them. Deferred with their code:

- need a world or `NewWorld` (Stage 1): FR-CFG-002, 003, 018, 027, 032, 038,
  063, 081, 083; FR-WLD-001, 002, 020; FR-OBS-002, 007 onward; NFR-OBS-002,
  003;
- viability (D4, Stage 1): FR-CFG-033, 041 to 044; FR-VIA-004 to 008 and the
  other viability IDs;
- need a module: FR-CFG-016, 045, and the module ranges.

## Open items

- **O1.** Phase 1 deliverables 1, 4, 5 and 6 need the owner's decision in
  their own documents (the keying form and constants, the library, the
  module-table rule and estates switch, the encoding rule of D5). Phases 3
  to 6 wait for Phase 1; Phase 2 does not, except `Key` and the stream key.
- **O2.** ADR 0002 (persistence by checkpoint) is still "Proposed", and the
  hash rules rest on it. Not a blocker for Stage 0; it should be accepted
  before Stage 2.
- **O3.** No CI exists. testing.md 9.1 says CI enforces the coverage
  thresholds and architecture.md 7.4 asks for two architectures. Until CI is
  set up, the checks, the coverage reading and the timed fuzz runs are done
  by hand in each PR and in Phase 7, on one architecture. Setting up CI
  needs an owner and a roadmap row.
- **O4.** N-127 (`WithObserver` with event classes) is left open pending a
  measurement in the slice. The signature built in Phase 6 may change in
  Stage 1.
