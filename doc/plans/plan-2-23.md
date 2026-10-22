# Plan: Stage 2, row 23: denomination, the optional display package

**Status: written (2026-10-22), not audited. Contract: [plan-2-0.md](plan-2-0.md).**
Builds `denomination/`: the ladder, the formatter and the RPG preset of [spec-currency.md](../specs/spec-currency.md) 4 and 7. It is a leaf package outside the simulation. It is not a module in the sense of `architecture.md` 2.10: it has no `Config.Modules` pointer, no position in `tick.go`, no event, no hash field and no checkpoint section, and nothing in the core imports it.

Rule tags read (the files show the same tags as `plan-2-0.md`): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1.

Specs and requirement IDs implemented (all exist in `spec-currency.md` section 8): FR-CUR-002, FR-CUR-008, FR-CUR-009, FR-CUR-010, FR-CUR-011, FR-CUR-012, FR-CUR-013, FR-CUR-014, FR-CUR-015, FR-CUR-016, FR-CUR-017, FR-CUR-018, FR-CUR-019, NFR-CUR-001. Not this plan's, though the same spec (FR-CUR-001, 003 to 007 and 020 to 023, the core: minor unit, floor, basket), and where each is tested:

| ID | Where |
|---|---|
| 001, 003, 007, 021, 022 | Stage 1 tests |
| 004 | plan 6 (`TestFlow_PerYearUsesCalendar`, `TestFlow_PerTickIgnoresCalendar`) |
| 005, 006, 020, 023 | plan 8 (tests added in its pack-rebalance phase) |

Notes read (index; no open note is tagged CUR; the entries tagged ALL): N-016 (leave open: the plan adds no configuration field, so no registry row), N-017 (apply: names below are setting-neutral outside the preset), N-027 (leave open: no macro constant), N-103 (leave open: the supervisor is the only editor of `doc/notes.md`), N-126 (leave open: no event, no `Warn`), N-128 (leave open: no checkpoint section), N-163 (leave open: the word "treasury" is not used), N-217 (apply, standing: the pinned hash and `testdata/golden/off/` do not move, Phase 6 proves it); and from CODE N-243 (apply: Lizard on the new files). `doc/notes.md` was not read in full, so no Notes tag is quoted; this plan changes no spec.

Decision block: D660 to D679.

## Scope

In:

- `denomination/` (stdlib only, imports nothing of the module): `doc.go`, `ladder.go` (`Unit`, `Ladder`, the bounds, the sentinels, the validation), `format.go` (`Formatter`, `Option`, `WithPrecision`, `NewFormatter`, `Format`), `preset.go` (`RPG`, `Single`), and the `_test.go` files of Phases 2 to 4.
- The guards that make FR-CUR-002 true: `import_guard_test.go`, `determinism_guard_test.go` (`guardedFiles`), `glossary_test.go` (Phase 5). These are test code of the root package.
- Text for the supervisor (Phase 1): `architecture.md` 2.6, 2.7, 5.1; `AGENTS.md` Layout and the stdlib hard constraint; `testing.md` 9.1; `doc.go`; `spec-currency.md` 4.2, 7; `roadmap-v1.md` row for `denomination/`.

Out, with the owner:

- Parsing formatted text back to `Money` (`spec-currency.md` 6, non-goal). Exchange rates, physical coins, debasement (same).
- Any use of the formatter inside the library: events, logs, snapshots and configuration carry integers (FR-CUR-001, built in Stage 1). A consumer formats at its edge.
- `examples/<first-consumer>` using `RPG()` and `examples/<second-setting>` using `Single`: plan 24.
- `go.work` and a separate module: none needed, the package is stdlib only and lives in the core module (D661). `go.work` is plan 22's.
- A `Money` method such as `Money.Format(f)`: it would make the root import the package (FR-CUR-002).

## Decisions

- **D660.** The package is not an `architecture.md` 2.10 module: no `Config.Modules` pointer, no `World` field, no `tick.go` position, no event, no snapshot row, no hash field id, no checkpoint tag, no option in `option.go`, no `internal/` package. It touches no catalogue file. Source: `spec-currency.md` 2.2 ("Nothing in the core imports the formatter ... removing it changes no simulation result") and 7; `architecture.md` 2.10 defines a module as a `Config.Modules` pointer.
- **D661.** The package is `denomination/` in the core module (no `go.mod` of its own), standard library only, importing no package of this module, not even the root. The spec's "name provisional" (7) is settled: `denomination`, the name the roadmap row, `AGENTS.md` and the spec already use. Rule 2.5 holds: one word, lowercase. Source: `spec-currency.md` 7; `architecture.md` 2.4, 2.5, 5.1; roadmap Stage 2 row.
- **D662.** `Format` takes an `int64`, not `economysim.Money`. A root import would make the package depend on the root (a cycle risk for the root's own external tests, and the spec says "standard library only"), and FR-CUR-012 and NFR-CUR-001 are stated over "any `int64` amount". A consumer writes `f.Format(int64(m))`; `Money` is `type Money int64` (`value.go`). The spec's "from `Money`" (4.2) is read as the `int64` count, and the wording is an owed edit (Phase 1). Source: `spec-currency.md` 4.2, 7; FR-CUR-012; `value.go`.
- **D663.** Types: `Unit{Name, Symbol string; Size int64}` (one denomination) and `type Ladder []Unit`, largest first. The element is `Unit`, not `Denomination`, because `denomination.Denomination` stutters (rule 2.5). `Unit` is read inside the package as "one rung of the ladder"; the glossary entry says so, so it is not confused with a unit of a commodity or the minor unit. Reversible before the first release (rule 6.6). Source: `spec-currency.md` 4 intro, 4.1; `architecture.md` 2.5.
- **D664.** Constructor: `NewFormatter(l Ladder, opts ...Option) (*Formatter, error)` (rule 6.1: the package has several types, so `NewX`, and `(*T, error)` because validation can fail). `Option` is `func(*options) error`, `options` unexported (rule 6.2), and the only option is `WithPrecision(p int) Option` (default 0). A precision outside `0..MaxPrecision` is the error of the option, returned by `NewFormatter`. Source: FR-CUR-010 ("at construction of the formatter"), 4.2 step 4; `architecture.md` 6.1, 6.2.
- **D665.** `NewFormatter` copies the ladder, so a caller that mutates its slice afterwards changes nothing, and `Format` reads only the formatter's own data: it is a pure function of ladder, precision and amount, and a `*Formatter` is safe for concurrent use. The doc comment says both. A nil `*Formatter` or a zero `Formatter` is not usable and the comment says so (rule 6.3). Source: FR-CUR-011; `architecture.md` 6.3, 7.1.
- **D666.** Sentinels, in this package (not in the root `errors.go`, which is the core's catalogue): `ErrNoUnits`, `ErrUnitName` (empty name or symbol), `ErrUnitDuplicate` (a name or a symbol used twice), `ErrUnitSize` (outside `1..MaxSize`), `ErrUnitOrder` (not strictly decreasing), `ErrPrecision`. Each is wrapped with the index of the offending unit, `fmt.Errorf("unit %d: %w", i, ErrUnitSize)`. The validation order is fixed: no units; then for each unit in order, name and symbol, size, duplicate against the earlier units, order against the previous unit; the first fault returned. Source: FR-CUR-008, 009, 010; `architecture.md` 8.2, 8.4; `testing.md` 3.9.
- **D667.** The bounds are exported constants of `ladder.go`: `MaxSize = 1_000_000_000_000` (10^12) and `MaxPrecision = 6`. They are the spec's bounds of a display helper, not tunables of the simulation (rule 9.5 does not apply: no result depends on them), so they are not in the root `bounds.go` and not in the parameter-owner table. Source: `spec-currency.md` 4.1, 4.2 step 4; `architecture.md` 9.5, 9.6.
- **D668.** Arithmetic is `uint64` throughout and needs no checked helper: the magnitude of a negative amount is `-uint64(amount)` (defined for unsigned types, exact for `math.MinInt64`); `count = r / size`, `r = r % size` cannot overflow; the only product is `r * 10^p` with `r < size <= 10^12` and `p <= 6`, so it is below `10^18 < 2^64`. The doc comment of the fraction step carries this proof. No floating point, no `math/big`, no `strconv.ParseFloat`. Source: `spec-currency.md` 4.2 steps 1, 4; FR-CUR-012, 015; `architecture.md` 3.5.
- **D669.** `Format` writes the symbols of the ladder (as the table of 4.3 shows). `Name` is validated and unique (FR-CUR-008) and is for the consumer's own use (`Ladder` is a plain slice it can read); the formatter does not write it. Source: `spec-currency.md` 4.1, 4.3.
- **D670.** Output form: parts joined by one space, each `<count> <symbol>`; the smallest unit with a precision is `<count>.<frac> <symbol>` with `frac` zero-padded to `p` digits; a leading `-` with no space. Count is `strconv.AppendUint` of the `uint64`. Source: `spec-currency.md` 4.2, table of 4.3.
- **D671.** Step 5 fallback (nothing printed by steps 3 and 4): the spec says "prints `0` of the smallest denomination". At precision `p > 0` this plan prints it in the same form as a zero amount (`0.00 cp` at `p = 2`), so the form of the smallest unit never depends on why it was printed. This is a reading, not a decision: the spec is silent on `p > 0` here, so the owner of CUR is asked to confirm in Phase 1, and the three table cases that pin it are marked in the test with the edit as their condition. The sign rule is as FR-CUR-016: written only if a printed count or `frac` is non-zero. Source: `spec-currency.md` 4.2 steps 4 and 5; FR-CUR-014, 016.
- **D672.** Presets are functions that return a fresh slice each call, never package variables (rule 7.1, no package-level mutable state): `RPG() Ladder` returns gold `gp` 10 000, silver `sp` 1 000, copper `cp` 100, largest first, with the names `Gold`, `Silver`, `Copper`; `Single(name, symbol string) Ladder` returns one unit of size 1. The setting vocabulary of `RPG` is the one exception to the `AGENTS.md` setting-agnostic bullet (Phase 1 row), whose answer is owner action (9) of plan 1 Phase 5; the `AGENTS.md` edit is applied only after it: a display-only preset that nothing in the root, `internal/`, `config/` or `strategy/` imports (FR-CUR-002); `Single` and the tests are neutral. If the owner refuses the exception, `RPG` moves to `examples/` with a `spec-currency.md` 7 edit and plan 24 owns it. Source: `spec-currency.md` 4.3, 7; FR-CUR-002, 017; `AGENTS.md` setting-agnostic constraint; `architecture.md` 7.1.
- **D673.** The package has no logger, observer, event, metric, random source or clock, writes nothing, and returns errors only at construction; `Format` has no error result (FR-CUR-010). Source: `observability-and-logging.md` 1.2, 2.3; `architecture.md` 7.3, 11.1.
- **D674.** FR-CUR-002 is enforced by the existing import guard, extended in `importProblems`: plan 23 adds `denomination` to the banned lists of `.` and `internal`, and adds a per-top-directory switch for `config` and `strategy` only if none exists when it is built; plan 18 and plan 20 add test rows and may add cases to the same function, and whichever is later appends: the root and `internal/` may not import `denomination`, and `config/` and `strategy/` get a banned list that holds it (today `importProblems` has a banned list for `.` and `internal` only); a file under `denomination/` may import the standard library and nothing of the module. `guardedFiles` gains `denomination` so that the determinism guard (no `init`, no `time.Now`, no print, no `panic` without "invariant") covers it. `indicator/` and `experiment/` are not forbidden: the spec names root, `internal/`, `config/`, `strategy/` only (open item O2). "A build and a run without it give identical results" follows: nothing links it, and Phase 6 checks it with `go list`. Source: FR-CUR-002; `architecture.md` 2.7, 2.9, 5.1.
- **D675.** Tests are in the external package `denomination_test`, through the exported API (`testing.md` 1.2, 3.7); no internal test file unless a branch cannot be reached otherwise. Coverage of the package is at least 90% (the table of `testing.md` 9.1 gains the row, Phase 1).
- **D676.** The fuzz target `FuzzFormat` builds a valid ladder from fuzzed sizes (sort descending, drop duplicates and out-of-range sizes, skip if empty), a precision `0..6` and any `int64`, and asserts: no panic, a non-empty result, a leading `-` only if some printed count or fraction is non-zero, and a round trip by parsing the output back: the sum of the printed `count * size` is at most the magnitude, and the magnitude minus that sum is less than the smallest size (less than `size / 10^p` at `p > 0`). Seeds by `f.Add`; a crash input is committed under `denomination/testdata/fuzz/FuzzFormat/` with a `Fix:` comment. Source: NFR-CUR-001; `testing.md` 7.4, 7.5.
- **D677.** Mutation: `gomu` on `denomination/` only inside Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4` and `--workers 4` (`~/.claude/CLAUDE.md`, `testing.md` 9.4). There is no new `internal/` package, so this is the nearest form of the contract's gate; surviving mutants are read, not zeroed. Source: plan-2-0 gates.
- **D678.** The new terms go to `doc.go` before code uses them (`architecture.md` 3.2): Denomination, Denomination ladder, Formatter, Precision, Denomination ladder preset, Unit (Phase 1 text). The bare entry "Ladder" (the want ladder) is plan 5's (its E9), which lands first; the type `Ladder` is named inside the "Denomination ladder" entry. `TestGlossary_DeclaredTypesNamed` is extended to the exported types of `denomination/ladder.go` and `format.go`, so a type without a word fails.
- **D679.** The plan depends on plan 1 only, for the gates (R and F of its D229, the Lizard rule of its D238). It can be built at any point after plan 1; it moves no golden, no hash and no fixture. Source: plan-2-0 table (Needs: 1).

## Phases

### Phase 1: edits owed before code (the supervisor applies; the plan edits none)

Goal: the specs and rules say what the code does, and the terms exist in `doc.go`.

| File and section | Edit |
| --- | --- |
| `doc/specs/spec-currency.md` 7 | Delete "(name provisional)" and the sentence "Until then the rules' layout does not list it" once the layout edit below lands. |
| `doc/specs/spec-currency.md` 4.2 intro, 4.1, FR-CUR-011, FR-CUR-012 | State that `Format` takes the `int64` count of minor units (D662; `Money` converts with `int64(m)`), that the formatter writes the symbols and not the names (D669), and that the ladder's slice is copied at construction (D665). |
| `doc/specs/spec-currency.md` 4.2 step 5, FR-CUR-016 | Say what the fallback prints at `p > 0` (D671: `0.00 cp`, the form of a zero amount) and that the sign is never written for a result whose printed counts and fraction are all zero. Owner: CUR. |
| `doc/specs/spec-currency.md` 4.3 | Add that a ladder is written largest first, so the preset's table (copper first) is the same ladder reversed (D672). |
| `doc/rules/architecture.md` 2.6 | Add under the root list, after `experiment/`: `denomination/   optional display of Money: the ladder, the formatter and the RPG preset [the words "and the RPG preset" only after owner action (9) of plan 1 Phase 5 answers yes; if no: "the ladder and the formatter"]; stdlib only; imported by consumers and examples, never by the root, internal/, config/ or strategy/`. |
| `doc/rules/architecture.md` 2.7 | Add `denomination` to the sentence on who imports whom: it imports nothing of the module; the root, `internal/`, `config/` and `strategy/` **MUST NOT** import it. |
| `doc/rules/architecture.md` 7.3 | Add `denomination/` to the packages in which `time.Now`, the global `math/rand` and `crypto/rand` **MUST NOT** be called (D674 extends the determinism guard to it). |
| `doc/spec-roadmap.md` item 4 ("When the first code is written, add `denomination/` to the layout") | Mark done by the 2.6 and `AGENTS.md` edits of this table. |
| `doc/rules/architecture.md` 5.1 | Add `denomination/` to the list that depends on the standard library only. |
| `doc/rules/architecture.md` 9.6 | Say that a validation bound of a package that validates its own input outside the root (`denomination.MaxSize`, `denomination.MaxPrecision`, and `experiment` for EXP 13.2) stays with that package; the root `bounds.go` holds the bounds of `Config` (D667). Same `architecture.md` edit and tag as the 2.6, 2.7, 5.1 and 7.3 rows. |
| `doc/rules/testing.md` 9.1 | Add `denomination/` to the 90% row of the coverage table. |
| `AGENTS.md` Layout, "Standard library only in the core" | Add `denomination/` to the layout in the same words as `architecture.md` 2.6 (including its owner-action (9) dependency on "and the RPG preset"), and to the packages that import the standard library only. |
| `AGENTS.md` Hard constraints, bullet "Setting-agnostic" | Waits for the answer to owner action (9) of plan 1 Phase 5 (`plan-2-1.md`) and is applied only after it; if the answer is no, `RPG` moves to `examples/` (D672) and this row is dropped. Add an exception: the `RPG` preset of `denomination/` (`spec-currency.md` 7, FR-CUR-017) is the one setting vocabulary allowed outside `examples/`, as a display-only preset that nothing in the root, `internal/`, `config/` or `strategy/` imports (FR-CUR-002) (D672). |
| `doc.go` glossary | Add the six entries below (the bare "Ladder" entry is plan 5's), in alphabetical place, and replace the end of "Minor unit", "the library has no currency, denomination or decimal place.", with "the core has no currency or decimal place; the optional denomination package names multiples of it." (the old sentence says there is no denomination, which stops being true). |
| `doc/roadmap-v1.md` Stage 2, `denomination/` row | Mark done with the commit, after Phase 6. |

Glossary text (D678): **Denomination**: a name and a symbol for a multiple of the minor unit; a display concern only, never in the core's types, events, snapshots, logs or configuration. **Formatter**: a pure function from an `int64` amount to text in the Units of a Ladder, built by `denomination.NewFormatter` with an Option such as `WithPrecision`. **Denomination ladder**: the type `Ladder`, an ordered list of Units, largest first, with strictly decreasing sizes. **Denomination ladder preset**: a ready-made `Ladder` (`RPG`, `Single`; written with `RPG,` only after owner action (9) answers yes, else `(Single)` and the RPG entry moves to plan 24's example). **Precision**: the number of decimal places, 0 to 6, with which the smallest Unit of a Ladder is written. **Unit**: one rung of a Ladder, with a name, a symbol and a size in minor units.

Editing `architecture.md` or `testing.md` replaces its read-receipt tag (the supervisor does it once for all plans that edit the file; plan 1 also edits 2.6 and `AGENTS.md`, so the three layout lines are merged, not duplicated). The code of Phase 2 does not start before the `architecture.md` 2.6 line exists (spec 7: "the PR that adds `denomination/` also adds it").

Exit check: `grep -n denomination doc/rules/architecture.md AGENTS.md doc.go` finds the new lines; `grep -n "provisional" doc/specs/spec-currency.md` is empty.

### Phase 2: the ladder and its validation (FR-CUR-008, 009, 010)

Files: `denomination/doc.go`, `denomination/ladder.go`, `denomination/ladder_test.go`.

`doc.go` is the package comment (name, what it is, that the core never imports it, the output form, the concurrency note). `ladder.go` holds `Unit`, `Ladder`, `MaxSize`, `MaxPrecision`, the six sentinels and `func (l Ladder) validate() error`, each exported identifier with a doc comment beginning with its name (6.5). Lizard: `validate` is split in `validateUnit` and `validateOrder` so each stays within CCN 10.

Tests (all `Requirement:` lines as shown; each case asserts with `errors.Is`, never message text):

- `TestNewFormatter_RejectsInvalidLadders`: nil ladder and empty ladder (`ErrNoUnits`); empty name, empty symbol (`ErrUnitName`); duplicate name, duplicate symbol with different names (`ErrUnitDuplicate`); size 0, -1 and `MaxSize+1` (`ErrUnitSize`); equal sizes and an increasing pair (`ErrUnitOrder`); one case per fault, each breaking only that rule (3.9). Requirement: FR-CUR-008, FR-CUR-009, FR-CUR-010.
- `TestNewFormatter_AcceptsBoundaryLadders`: size 1 alone, size `MaxSize` alone, `MaxSize` then 1, and a ladder whose sizes do not divide each other (100, 30, 7). Requirement: FR-CUR-008, FR-CUR-009.
- `TestNewFormatter_RejectsPrecisionOutsideRange`: -1 and 7 give `ErrPrecision`; 0 to 6 succeed; a valid ladder with a bad precision returns a nil formatter. Requirement: FR-CUR-010.
- `TestNewFormatter_ReportsTheFirstFaultWithItsIndex`: a ladder with two faults returns the first by D666, and the message contains the index (`errors.Is` plus `strings.Contains` on the index only: the one deliberate message-text assertion in this plan, because the index is part of the contract and a typed error is not worth a new exported type). Guard: callers read the index; a reordered check would change which sentinel they branch on.

Exit check: `go vet ./denomination/`, `go test -short -race ./denomination/`, `gofmt -l denomination` empty.

### Phase 3: the formatter (FR-CUR-011 to 016)

Files: `denomination/format.go`, `denomination/format_test.go`.

`format.go`: `Option`, `options`, `WithPrecision`, `Formatter{units Ladder; precision int}`, `NewFormatter` (validate, copy, apply options), and `Format`, split for Lizard into `magnitude(amount) (uint64, bool)`, `appendUnit(dst, count, sym)` and `appendSmallest(dst, count, r, size, p)`; `pow10(p)` is a loop, not a package table. The unit loop fills from the largest with `count = r / size`, `r = r % size` (D668), omits a zero count except for the smallest unit, then applies steps 4 and 5 and the sign rule.

Tests (table-driven, `t.Parallel()`, expected values carry their arithmetic, `testing.md` 5.5):

- `TestFormat_FillsFromLargestToSmallest`: RPG 12 345 = 1 gp (10 000), r 2 345; 2 sp (2 000), r 345; 3 cp (300), r 45 gives `1 gp 2 sp 3 cp`; 10 000 gives `1 gp`; 10 100 gives `1 gp 1 cp`; 10 000 000 gives `1000 gp` (the largest is unbounded). Requirement: FR-CUR-013, FR-CUR-014.
- `TestFormat_ZeroAmount`: 0 gives `0 cp` at precision 0 and `0.00 cp` at 2; a ladder of one unit gives `0 <symbol>`. Requirement: FR-CUR-014, FR-CUR-016.
- `TestFormat_Precision`: 1 460 at p = 0, 1, 2, 6 on RPG: r after cp is 60, so `frac = 60 * 10^p / 100` gives `1 sp 4 cp`, `4.6`, `4.60`, `4.600000` (the unit written `cp`); 5 at p = 2 gives `0.05 cp` (frac 5, zero-padded); 1 499 at p = 0 gives `1 sp 4 cp` (99 truncated); a ladder of one unit of size `MaxSize` with amount `MaxSize - 1` at p = 6 gives `0.999999 <symbol>` (r * 10^6 = 999 999 999 999 000 000 < 2^64, divided by 10^12 truncates to 999 999). Requirement: FR-CUR-015.
- `TestFormat_SignAndNeverNegativeZero`: -2 920 gives `-2 sp 9 cp`; -100 gives `-1 cp`; -1 and -99 at p = 0 give `0 cp`, not `-0 cp` (no printed count or fraction is non-zero); -1 at p = 2 gives `-0.01 cp` (frac 1 is non-zero); -5 at p = 1 on size 100 (frac 0) gives `0.0 cp` (D671, marked: pinned by the Phase 1 edit). Requirement: FR-CUR-012, FR-CUR-016.
- `TestFormat_MinAndMaxInt64`: RPG, `math.MinInt64` (magnitude 9 223 372 036 854 775 808 = 922 337 203 685 477 gp (9 223 372 036 854 770 000) + 5 808; 5 sp, r 808; 8 cp, r 8) gives `-922337203685477 gp 5 sp 8 cp` at p = 0 and `-922337203685477 gp 5 sp 8.08 cp` at p = 2; `math.MaxInt64` gives `922337203685477 gp 5 sp 8 cp` (remainder 5 807: 5 sp, r 807: 8 cp, r 7) and `8.07 cp` at p = 2. Requirement: FR-CUR-012, NFR-CUR-001.
- `TestFormat_IsPureAndIndependentOfTheCallersSlice`: the same formatter gives the same string twice; a ladder slice mutated after `NewFormatter` changes nothing; two formatters from two `RPG()` calls do not share data; the signature is `func(*Formatter, int64) string` (compile-time assertion, so formatting returns no error). Requirement: FR-CUR-010, FR-CUR-011.
- `TestFormat_ConcurrentUse`: 8 parallel subtests call one shared formatter on distinct amounts, expecting the same strings; run under `-race`. Guard: D665's claim of concurrent safety.

Exit check: `go test -short -race ./denomination/`; coverage of `format.go` reported by `go test -cover ./denomination/` at least 90%.

### Phase 4: presets, examples and the fuzz target (FR-CUR-017 to 019, NFR-CUR-001)

Files: `denomination/preset.go`, `denomination/preset_test.go`, `denomination/example_test.go`, `denomination/fuzz_test.go`.

- `TestRPG_FormatsTheSpecTable`: the five rows of `spec-currency.md` 4.3 (0, 100, 1 460, 12 345, -2 920) at precision 0 and at precision 2, expected text copied from the spec. Requirement: FR-CUR-017.
- `TestRPG_LadderIsLargestFirstWithTheSpecSizes`: sizes 10 000, 1 000, 100, symbols `gp`, `sp`, `cp`; it passes `NewFormatter`; two calls return distinct slices (mutating one leaves the other). Requirement: FR-CUR-017. Guard: D672, no package-level mutable state.
- `TestSingle_FormatsEveryAmountAsACount`: `Single("credit", "cr")` on 0, 1, -7, 123 456 789, `math.MaxInt64`, `math.MinInt64` gives `0 cr`, `1 cr`, `-7 cr`, the decimal count with `cr`, and the two extremes as their decimal text. Requirement: FR-CUR-018, FR-CUR-012.
- `TestFormat_IrregularLadders`: neutral names. Sizes 240, 12, 1 (symbols `L`, `M`, `S`): 500 = 2 * 240 (480), r 20; 1 * 12, r 8; 8 gives `2 L 1 M 8 S`; 239 gives `19 M 11 S` (239 = 19 * 12 + 11); 240 gives `1 L`. Sizes 100, 30, 7 (`A`, `B`, `C`; no size divides the larger one): 250 = 2 * 100, r 50; 1 * 30, r 20; 2 * 7, r 6 gives `2 A 1 B 2 C`, and at p = 2 `2 A 1 B 2.85 C` (6 * 100 / 7 = 85). Sizes 60 and 1 (`H`, `P`): 3 725 = 62 * 60 (3 720) + 5 gives `62 H 5 P`. Requirement: FR-CUR-019, FR-CUR-009.
- `ExampleFormatter_Format`: RPG at precisions 0 and 2 on 12 345 with `// Output:` (6.5). Examples are exempt from the doc-comment rule of `testing.md` 5.1.
- `FuzzFormat` (D676): seeds `f.Add` of 0, 1, -1, 1 460, `math.MinInt64`, `math.MaxInt64` with sizes (10 000, 1 000, 100), (1), (240, 12, 1), (`MaxSize`, 1) and precisions 0, 2, 6. Requirement: NFR-CUR-001, FR-CUR-012, FR-CUR-016.
- `TestFormat_NeverPanicsAtTheExtremes`: a table over (ladder, precision, amount) for the corners: `MaxSize` alone with `MinInt64` at p = 6, size 1 with p = 6 (frac is always 0), p = 6 with r = size - 1. Requirement: NFR-CUR-001. It is the always-on form of the fuzz properties, because `go test` runs seeds only.

Exit check: `go test -short -race ./denomination/`; `go test -run xxx -fuzz FuzzFormat -fuzztime 60s ./denomination/` is run once by hand and any crash input committed with its `Fix:` comment; coverage of the package at least 90%.

### Phase 5: the guards for FR-CUR-002 and the glossary

Files: `import_guard_test.go`, `determinism_guard_test.go`, `glossary_test.go` (root package tests).

- `importProblems`: append `denomination` to the banned lists; add a per-top-directory switch for `config` and `strategy` only if none exists when this is built (plan 18 and plan 20 may add cases to the same function; whichever is later appends). The module's own packages the importer may not import gain `denomination` for `.` and `internal`; add cases for top `config` and `strategy` (banned: `denomination`) and for top `denomination` (banned: every package of the module, the root included, and any path that is not the standard library; the parser import stays banned). Today `importProblems` has banned lists for `.` and `internal` only.
- `TestImportGuard_Detects` gains rows: root imports `denomination` (1); `internal/x` imports it (1); `config/c.go` imports it (1); `strategy/s.go` imports it (1); `denomination/d.go` imports the root (1), `internal/checked` (1), a third-party path (1) and `fmt` (0); `indicator/i.go` imports it (0, D674, open item O2). Requirement: FR-CUR-002. Guard: a checker that missed one of these would let the leaf package become a dependency of the core with every test green.
- `TestImports_StdlibAndLayers` needs no new code beyond the `guardedFiles` scope: it now walks `denomination/`. Its doc comment lists the package. Requirement: FR-CUR-002.
- `guardedFiles`: `"denomination"` joins the list of roots; `TestGuardedFiles_Scope` gains `denomination/d.go` in its tree and in `want`, and the doc comment names it. The existing function skips a root that does not exist, so the change is safe before Phase 2.
- `TestGlossary_DeclaredTypesNamed`: the file list gains `denomination/ladder.go` and `denomination/format.go` (the exported types `Unit`, `Ladder`, `Formatter`, `Option` must match `\bName\b` in `doc.go`; `Ladder` matches through the "Denomination ladder" entry, which names the type; `Formatter` and `Option` match only through the new entries, the Formatter entry carrying the word Option; `Unit` is matched vacuously today by "Unit of adjustment" and "Unit cost", so its check proves nothing). Requirement: none; Guard: architecture.md 3.2.
- A test `TestDenomination_NotLinkedByTheCore` is not written: a test cannot run `go list`, and the guard above already reads every import of the core's non-test files. The run-level half of FR-CUR-002 is the exit check of Phase 6.

Exit check: `go test -short -race -run 'TestImport|TestGuardedFiles|TestGlossary|TestBannedUses|TestNoBannedUses' .` passes; temporarily adding `import _ "github.com/Whealinghold/economy-sim/denomination"` to a root file makes `TestImports_StdlibAndLayers` fail (checked by hand, not committed; `testing.md` 9.3).

### Phase 6: exit gate

1. `gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go mod tidy` with no diff (`go.mod` and `go.sum` unchanged: the package is stdlib only).
2. Tests: `go test -short -race ./...` and, per plan 1's D229, gate R (`go test -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out -timeout 60m ./...`) and gate F; the plan states what each reported.
3. Coverage: `denomination/` at least 90% in `coverage.out`; the root's coverage does not fall (`testing.md` 9.2).
4. `lizard -l go -C 10 -a 8 -L 100 -w denomination/*.go` (non-test files) prints nothing. Markdownlint on this plan is clean.
5. Mutation: `gomu` on `denomination/` in Docker with the caps of D677; the survivors are listed in the PR with a reason each (equivalent mutants such as the order of the two unit checks that cannot both fail).
6. FR-CUR-002 at run level: `go list -f '{{.ImportPath}} {{join .Imports " "}}' . ./config ./strategy ./internal/... 2>/dev/null | grep denomination` prints nothing (`./strategy` is skipped while it does not exist). A build of the root after `denomination/` is moved aside succeeds.
7. Non-interference (N-217): `TestHashConfig_ReferencePinned` and every file of `testdata/golden/off/` are unchanged (`git diff --stat -- testdata` is empty). No golden file, fixture or hash is added by this plan.

## Catalogue appends

None, and the table says why (D660). Each row is checked by `git diff --stat` on the file at the exit.

| File | Append | Why none |
| --- | --- | --- |
| `config.go`, `defaults.go`, `bounds.go`, `validate.go`, `hash.go` | none | no configuration field; the bounds are the package's own (D667); no hash field id |
| `errors.go` | none | the sentinels are the package's (D666), not core errors |
| `event.go`, `snapshot.go`, `ledger.go` | none | no event, no snapshot row, no flow reason (D673) |
| `option.go` | none | `denomination.Option` is the package's own; no `With*` of the world |
| `tick.go` | none | no position (D660) |
| checkpoint tags, log keys, metrics | none | no module section; no logging; no instrument |
| test catalogues (`notBuilt`, position table, parameter registry, event catalogue) | none | nothing to delete from them; nothing is added to the specs' tables |

## Cross-plan contract

Consumes:

- Gate R and gate F of D229 (`doc/plans/plan-2-1.md`) <- plan-2-1
- The Lizard rule of D238 and the markdownlint rule (`.codacy.yml`, `.markdownlint.json`) <- plan-2-1
- `guardedFiles` (`determinism_guard_test.go`), `importProblems`, `TestImportGuard_Detects`, `TestImports_StdlibAndLayers` (`import_guard_test.go`) <- Stage 1 code
- `glossaryTerms`, `exportedTypes`, `TestGlossary_DeclaredTypesNamed` (`glossary_test.go`), `readFile` (`event_test.go`) <- Stage 1 code
- `TestHashConfig_ReferencePinned` (`hash_test.go`) and `testdata/golden/off/` <- Stage 1 code
- `spec-currency.md` 4, 7, FR-CUR-002, 008 to 019, NFR-CUR-001 <- spec

Provides:

- `denomination.Unit` (`denomination/ladder.go`) -> plan-2-24
- `denomination.Ladder` (`denomination/ladder.go`) -> plan-2-24
- `denomination.NewFormatter` (`denomination/format.go`) -> plan-2-24
- `denomination.WithPrecision` (`denomination/format.go`) -> plan-2-24
- `denomination.Formatter.Format` (`denomination/format.go`) -> plan-2-24
- `denomination.RPG` (`denomination/preset.go`) -> plan-2-24
- `denomination.Single` (`denomination/preset.go`) -> plan-2-24
- `denomination.MaxSize`, `denomination.MaxPrecision` and the sentinels `ErrNoUnits`, `ErrUnitName`, `ErrUnitDuplicate`, `ErrUnitSize`, `ErrUnitOrder`, `ErrPrecision` (`denomination/ladder.go`) -> plan-2-24
- The import guard for FR-CUR-002 (`import_guard_test.go`, `determinism_guard_test.go`) -> plan-2-24
- The glossary terms Denomination, Denomination ladder, Formatter, Precision, Denomination ladder preset, Unit (`doc.go`) -> plan-2-24

## Testing

Per `testing.md` 11. Requirements and the tests that cover them:

| Requirement | Tests |
| --- | --- |
| FR-CUR-002 | `TestImportGuard_Detects` (new rows), `TestImports_StdlibAndLayers` (scope extended), Phase 6 item 6 |
| FR-CUR-008 | `TestNewFormatter_RejectsInvalidLadders`, `TestNewFormatter_AcceptsBoundaryLadders` |
| FR-CUR-009 | the same two, `TestFormat_IrregularLadders` |
| FR-CUR-010 | `TestNewFormatter_RejectsInvalidLadders`, `TestNewFormatter_RejectsPrecisionOutsideRange`, `TestFormat_IsPureAndIndependentOfTheCallersSlice` (no error result) |
| FR-CUR-011 | `TestFormat_IsPureAndIndependentOfTheCallersSlice`, `TestFormat_ConcurrentUse` |
| FR-CUR-012 | `TestFormat_SignAndNeverNegativeZero`, `TestFormat_MinAndMaxInt64`, `TestSingle_FormatsEveryAmountAsACount` |
| FR-CUR-013 | `TestFormat_FillsFromLargestToSmallest` |
| FR-CUR-014 | `TestFormat_FillsFromLargestToSmallest`, `TestFormat_ZeroAmount` |
| FR-CUR-015 | `TestFormat_Precision` |
| FR-CUR-016 | `TestFormat_SignAndNeverNegativeZero`, `TestFormat_ZeroAmount`, `FuzzFormat` |
| FR-CUR-017 | `TestRPG_FormatsTheSpecTable`, `TestRPG_LadderIsLargestFirstWithTheSpecSizes` |
| FR-CUR-018 | `TestSingle_FormatsEveryAmountAsACount` |
| FR-CUR-019 | `TestFormat_IrregularLadders` |
| NFR-CUR-001 | `FuzzFormat`, `TestFormat_NeverPanicsAtTheExtremes`, `TestFormat_MinAndMaxInt64` |

- **Level.** Unit tests only, at the lowest level that can see each requirement. The package is a pure function, so there is no integration or end-to-end test of its own; the consumer example of plan 24 is the integration proof (`testing.md` 2.3, 7.3).
- **Golden files.** None: off unchanged (proved in Phase 6 item 7), no module-alone golden (not a module, D660), no all-on golden.
- **Fork-at-N, checkpoint, registry, catalogue, position table.** Not applicable: no state, no field, no event, no position (Catalogue appends table).
- **Fuzz.** `FuzzFormat` (D676); `Format` parses nothing, but NFR-CUR-001 asks for it and the constructor validates external input (`testing.md` 7.4).
- **Benchmarks.** None: not on the tick path (`testing.md` 8.1).
- **Boundaries and errors.** Size 1 and `MaxSize`, precision 0 and 6, amounts 0, 1, -1, `MinInt64`, `MaxInt64`, a non-dividing ladder, every sentinel on its own (3.8, 3.9).
- **Broken-version check (9.3).** Run once by hand before the PR: flip `>` to `>=` in the order check, drop the sign rule, change `10^p` to `10^(p-1)`, and see the named tests fail.

PR description `Testing` entry: the table above, plus "no golden moved; `denomination/` coverage X%; gomu survivors listed".

## Observability

None, because the package is a pure function with a constructor. Walking `observability-and-logging.md` 9:

1. New failure path: the constructor returns an error from the sentinel set; nothing is survived silently, so no `Warn` (D673).
2. New state transition: none, the package holds no state after construction. No event, no `Observer` use.
3. New decision point in a strategy, clearing rule or spawner: none.
4. New loop: the unit loop in `Format` is not in the tick path; it logs nothing and builds no attribute.
5. Changed event, metric or attribute: none. No log key is added, so the table of 3.3.2 is unchanged. No metric, so `economysim.*` is untouched.
6. New metric attribute: none.
7. `Error` or `Info` used: none; the package has no logger (rule 11.1: it writes to no logger it was not given, and it is given none).
8. Logging an error it returns: none.
9. Tests: section 8 does not apply. The non-interference of a world run is unaffected because nothing in the core links the package (FR-CUR-002).

## Risks and open items

- **O1 (owner: CUR spec).** Step 5 of 4.2 at `p > 0`: D671 reads it as `0.00 cp`. If the owner prefers a bare `0`, only `appendSmallest` and the marked cases change. Phase 1 carries the edit.
- **O2 (owner: architecture.md 2.7 and the IND and EXP plans, 18 and 20).** The spec forbids imports by root, `internal/`, `config/` and `strategy/` only. `indicator/` and `experiment/` could format text with the package, but a measure is a number and a report is the consumer's. Left allowed by the guard (a row pins it); if the owner forbids it, one row of `importProblems` changes.
- **O3 (owner: CUR spec, owner).** D662, `int64` and not `Money`. If the owner wants `Money` in the signature, the package must import the root, which changes `architecture.md` 2.7 and 5.1 and makes D674's leaf rule untrue; the plan recommends against.
- **O4 (owner).** `Unit` (D663) and the `Single` and `RPG` names are this plan's reading of rules 2.5 and 6.1; the spec names no identifier. Reversible until the first release.
- **O5 (owner: CUR spec).** `Name` is validated and unique but never written (D669). If a later consumer needs a long form (`1 Gold`), it is a new method, not a change of `Format`.
- **O6 (owner: CUR spec).** The spec does not say whether a name or symbol made of white space, or one containing a space, is valid. FR-CUR-008 says "non-empty" and the plan checks exactly that; a symbol with a space would still format.
- **O7 (owner).** Resolved by the `architecture.md` 9.6 row of Phase 1: D667 keeps this package's two bounds beside the code.
- **O8 (owner: plan 24).** The consumer example must show a world's `Money` formatted at the edge: `f.Format(int64(price))`. It does not depend on this plan beyond the Provides list.
- **O9.** The shared files (`architecture.md` 2.6, `AGENTS.md`, `testing.md` 9.1) are also edited by other plans; the supervisor merges the lines and replaces each read-receipt tag once.

Note proposals (text only; the supervisor edits `doc/notes.md`):

1. Tag CUR, CODE: "The denomination package formats an `int64`, not `Money` (D662 of plan 2-23), so the root never imports it; at `p > 0` the step 5 fallback prints the zero form with `p` places (D671). Confirm both in `spec-currency.md` 4.2 when the package lands, then remove."
2. Tag IND, EXP: "`indicator/` and `experiment/` are not forbidden to import `denomination/` (the spec names root, `internal/`, `config/` and `strategy/`); a measure stays a number and a report formats at the consumer. Decide when IND and EXP are written."
3. Tag ALL, CODE: "A leaf package that is not a module (`denomination/`) touches no catalogue file; its guards are the import guard and `guardedFiles`. A later optional package outside the simulation follows the same form."
