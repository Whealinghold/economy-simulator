# Plan: Stage 2, row 21: CLB, the calibration registry test, the pure rules and the dry run (`internal/calibration`, `testdata/calibration/`)

**Status: written (2026-10-22), not started.** Sequences the twenty-first plan of Stage 2 ([plan-2-0.md](plan-2-0.md)): the roadmap row "`internal/calibration` and `testdata/calibration/`" ([roadmap-v1.md](../roadmap-v1.md) Stage 2). A working document: the specs, the ADRs and `doc/rules/` decide, and where this plan disagrees with them they win.

Rule tags read (as the files end, 2026-10-22; they equal the contract's): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1. `architecture.md` 2, 3.5, 9 and 10, `testing.md` 3 to 7, 9 to 11 and `observability-and-logging.md` 2.3, 9 were read in full for what this row touches; the rest by section.

Implements [spec-calibration.md](../specs/spec-calibration.md) (CLB): FR-CLB-001 to 009, 010 to 014, 020 to 023, 025 to 027 (FR-CLB-024 is retired, D624), 030 to 041, 042, 043, 050 to 054, 060 to 063, 071, 100 to 109 and NFR-CLB-001 to 005, and FR-STY-092 of [spec-stylised-facts.md](../specs/spec-stylised-facts.md) (a screen sign that differs from the registry's is a finding, D634) (FR-CLB-004, 070, 072 to 074, 080, 081 and 090 are review items, 11.7; FR-CLB-090 only as a review item, the `-short` rule of D639; FR-CLB-091 is Phase 7). Reads [spec-experiments.md](../specs/spec-experiments.md) 3.3.4, 11.3, 12.4, 13.2 and [spec-stylised-facts.md](../specs/spec-stylised-facts.md) 3.2, 5.11, 7.3, 8.5, 12 (the facts, the band keys, the stability record's home). Decision IDs: **D620 to D639** (all used).

Notes read: the index of `doc/notes.md`, then N-015, N-124, N-134, N-135, N-158, N-162, N-169, N-199, N-207, N-208, N-214, N-220 (tag CLB) and N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217 (tag ALL). N-015: leave open (the procedure constants stay placeholders, D628). N-124: apply for CLB (the flag `d` goes, D624), leave open for EXP. N-134: apply (flags `in`/`tbl`, FR-CLB-009, 037; the dry run exercises `Prepare`), leave open for VIA. N-135: leave open (the `in_sample` rule is implemented, D633; the hold-out stays a SHOULD). N-158, N-169, N-199, N-207, N-208, N-162: leave open (they are inputs to a calibration, not to this code; N-162 now says the draws are aligned, so only the ID shift of a seat breaks a pair, and CLB 7.2.3 never relies on pairing). N-214: apply for CLB (Phase 1), leave open for EXP, STB and TEC. N-220: apply (Phase 1 lists the owed rows, D625). N-016: apply (the registry test is the second reader of the owner table). N-017: apply (the glossary terms of Phase 1). N-027: apply (no test asserts a number of the economy). N-103, N-126, N-128, N-163: not touched (no module, no event, no hosted field, no use of "treasury"). N-217: apply (this plan changes no root file, so `testdata/golden/off/` and the pinned hash cannot move).

## Scope

**Builds.**

- `internal/calibration/` (standard library and `internal/design` only): `grid.go`, `seeds.go`, `plan.go`, `choose.go`, `alias.go`, `process.go`, `record.go` (types and `Render`), `record_parse.go` (`Parse`), `record_check.go`, `overlay.go`, with unit tests, three fuzz targets and their corpora under `internal/calibration/testdata/fuzz/`.
- In the same package, `_test.go` files: the registry test of CLB 11.1 (`registry_test.go` and helpers), the record checks of 11.2 over files, the overlay purity test of 11.5.
- `internal/calibration/testdata/records/{valid,invalid}/` (the inputs of the record tests) and `testdata/calibration/`: `README.md` (the grammar and the layout) and the fixture of the dry run (`dryrun.shape.txt`, outside every `<pack>/` directory). No stage record: nothing has run (N-027).
- `experiment/calibration_dry_test.go` (package `experiment_test`): the dry run of 11.4, legs A and B, skipped with `-short`.
- The Phase 1 document edits (listed, applied by the supervisor except where a file is this plan's).

**Does not build.**

| Item | Owner |
| --- | --- |
| A public `calibration/` package, a public `verdict/` (CLB open question 1) | the supervisor with STY, after the first consumer runs its own calibration; architecture.md 2.8 |
| The loop that runs the stages C0 to C9 on `one`, the first record, any `Search` range with a number | the first reviewed run (CLB open question 2, N-015) |
| The harness: `Spec`, `Run`, `Check`, `Render`, `Compare`, the designs | plan 20 |
| The measures, the regime detectors, the stability record's writer | plans 18, 19 (the file name is STY's: `testdata/golden/stylised/<pack>.stability.txt`, CLB 8.4) |
| A calibration overlay under `testdata/chains/calibrated/` | none exists; the purity test is ready for the first one (REF 4.1) |
| Any change to a root file, `Config`, an event, a log key or a metric | none: CLB adds no API, no field, no default, no event (CLB 1, 13, 15) |

## Decisions

- **D620.** One new package, `internal/calibration`, created by this plan (architecture.md 2.1: added when needed; 2.6 and `AGENTS.md` already list it). Imports: the standard library (`crypto/sha256` for the digest of bytes, `sort`/`slices`, `strings`, `strconv`, `bytes`, `errors`, `fmt`), `internal/checked` for every overflow-prone product or ratio (architecture.md 3.5 wins over NFR-CLB-003) and `internal/design` for the overlay text. It imports none of the root, `config/`, `strategy/`, `indicator/`, `experiment/`, and not `internal/ordering`. Source: CLB 12.1, architecture.md 2.9, 3.5; E13 owes NFR-CLB-003 and 12.1 the `internal/checked` import.
- **D621.** No state, float, clock, random source, goroutine, logger or package-level variable (closed tables are `const` or functions). A failure is a returned `error`; a decision is a line of the record. Overflow of the geometric grid is a returned error from `checked.Mul` and `checked.MulDivHalfEven` (the half-to-even level `rdiv(b, 2^-k)`), not a wrap; no local multiply helper exists. Source: CLB 15.1, FR-CLB-013; architecture.md 3.5.
- **D622.** The registry test is the Go port of `doc/specs/models/clb/registry.py`, check by check, in the package `internal/calibration`: plain text parse of the tables, no dependency, no `World`, so it runs under `-short`. The script and its `registry.out` are the executable specification of the parse; Phase 2 compares the Go test's findings with the script's on the final spec. It reads, by relative path (testing.md 10.4): `../../doc/specs/{spec-calibration,README,spec-config,spec-indicators,spec-stylised-facts,spec-stability,spec-viability,spec-observability}.md` and `../../doc/rules/observability-and-logging.md`. Source: CLB 4.1, 11.1, NFR-CLB-001.
- **D623.** The closed sets of 4.2 and 4.4 (classes `K D S P`, units, flags, signs `+ - 0 ~ ?`, sources `S M E H`, the measure tokens of 4.4, stage names, edge kinds, record keywords) are constants of the check, not read from the spec (they are rule constants of the check, CLB 4.2). The flag set is `in chk out tbl z v1x off=<v>`: **`d` is not in it** and a fixture row that carries `d` must fail the flags check (the negative case pins the removal). The measure-token check also reads the `Result fields` column of IND 9.1, which exists (verified), and fails when a token of 4.4 is not in it; the constant list of 4.4 is the cross-check, so a rename in IND is found. Source: FR-CLB-104, FR-CLB-007, N-214.
- **D624.** The flag `d` is removed from CLB (N-214, ADR 0008: every draw is a stream keyed by (site, tick, entity), so a pair is aligned in the draws whatever the knob and diverges through state only, WLD 7.6, N-124 answer). Consequences, all in Phase 1: FR-CLB-024 is **retired** (the ID is not reused; 21 and the README row say so); 7.1.3, 7.2.3, 7.3 (C7a) and 11.4 leg B (a) lose their `d` clauses; 31 knob rows and 20 groups lose the token (`d` alone becomes `-`, `d off=1` becomes `off=1`); the prose of 4.2.1, 6.2.2 (the budget stack), 6.2.6 (MER, MIG), 6.2.11 (`Places`, SAV), 6.3 (S-SCL02), 7.6 and 22 question 9 is rewritten. The comparison of two points is a choice of the experiment (EXP), not of a row. Source: N-214, N-124, N-162, spec-world.md 7.6.
- **D625.** The owed rows of N-220 are four failing checks of `registry.out`: (1) 39 owner pairs with no key or group field (Config `Edges`, `RegionPrices`, `WarmStart`, `Yields`; `Item`, `JobSpec`, `Month`, `Output`, `Season`, `SeasonYield`, `SeedPrice`, `SeedWage`, `Tool`, `WantItem`, `WarmGroup`, `WarmPool`, `World.Scale`, `YieldProfile`); (2) 13 registry-only pairs under scope tokens the owner table does not spell (`Config section`, `Recipe output`, `Recipe tool`, `Region and recipe`, `Want item`, `Yield`); (3) the catalogue names `Days`, `InProgressBP` in no group; (4) the nested `Scale` is under no scope named for its table. Classification is a review decision (CLB 4.6), so Phase 1 **proposes** the table below and the supervisor confirms it; the exit check is that `registry.py` and the Go test both print no FAIL. Source: N-220, CLB 4.3, 4.6, 6.3.
- **D626.** `Known` is the data the record checks need from the registry and STY: `Knobs map[string]KnobInfo` (the registry ID is the key) with `KnobInfo{Class byte, Key string, Stage string, Owner string, V1x, Z bool, Off Off, Moves map[string]MoveSign}`, where `Stage` is `C1` to `C9` (orders stages, FR-CLB-035), `Owner` is the owner scope (splits a stage by owner), `Off` is the off value and its kind (`Has`, `Value`; the caller resolves `min`/`max`) for the FR-CLB-071 check, and `MoveSign{Sign byte; Source byte}` per Moves token (`+ - 0 ~ ?`, `S M E H`) for FR-CLB-060; `Facts map[string]byte` (STY ID to class `X O B R P`), `Detectors map[string]bool`. The package never reads a file; the registry test fills every field from the parse of the 4.1 row (its Class, Stage, Owner, flags, `off=` and Moves cells) and of STY 5.11. Source: CLB 11.2, 4.4, D622.
- **D627.** One map from a file path to a registry (scope, field) serves two checks: the `in`/`chk` check (VIA 11 paths, CLB 4.2.1, EXP 11.3.1: `world` to World, `role.<id>` to Role, `region.<id>` to Region, `region.<id>.group.<id>` to Group, `modules.<m>...` to the module scopes, as `registry.py` respells them) and the overlay purity test (11.5: a key of a `calibrated/` document must be a K or D row). It is a test helper (`scope_paths_test.go`), with the snake rule of spec-config.md 6.2 (acronyms `BP`, `ID`; the exception `in_progress`). Source: EXP 11.3.1, CLB 11.1, 11.5.
- **D628.** The grid API: `type Rule struct{ Kind Kind; Step int64; Set []int64 }` (`Geo`, `Lin`, `Set`), `type Bounds struct{ Lo, Hi int64 }`, `type Off struct{ Has bool; Value int64 }` (the caller resolves `off=min` and `off=max` against the owner's bound), `Build(r Rule, base int64, b Bounds, off Off, valid func(level int64) bool) (Grid, error)`. `Grid{ Levels []int64; Base int; Lower, Upper Side }`, `Side{ Present bool; Level int64; Edge Edge }`, `Edge` one of `None`, `Bound`, `Off`, `Rule`. Geo levels `b*2^k` and `rdiv(b, 2^-k)` half to even, floor 1, `b` always a level, the cut of 5.2 step 2 (base exempt), invalid levels skipped by `valid`, a `z` base of 0 returns `ErrNoGrid` (the caller records `unsearched at_off`). `Distance(g, i, j) int64` counts index steps. `Widen(g Grid, side, n)` adds two kept valid levels by the rule of 5.3 and never leaves `[Lo, Hi]` or reaches the off value. Placeholders (factor 2, two levels, two more) are named constants with the spec section in a comment. Source: CLB 5.2, 5.3, FR-CLB-010 to 014, 026, 108.
- **D629.** `seeds.go`: `Block(b int) (first, last uint64, err)` (`100b+1` to `100b+100`, `b` at least 1, error below 1), `ConfirmSeeds(tierL bool) (first, last uint64)` (1..50; 1..10), `Threshold(r int) int` (`r - r/10`, exact integer, not half to even: 25 gives 23), `InSTYRange(seed) bool` (1 to 60, for the checks). `plan.go`: `ScreenLevels(g Grid) []int64` (base and the two neighbours), `ExploreSize(k int) int` (`max(16, 10k)`), `Replicas(search bool, tierL bool) int` (10 search, 50 or 10 confirm), `ConfirmPlan(chosen, base)` (the chosen configuration and the base as control, same seeds, measures and windows). Source: CLB 7.1, 7.2, FR-CLB-020 to 026, 106.
- **D630.** `choose.go`: `Point{ Index int; Levels []int; ExactPass bool; BPass []int }` (all X, O and detector criteria on all seeds; B pass counts per criterion), `Choice{ Index int; Distance int64; Feasible []int; Pareto []int; None bool }`, `Choose(points []Point, base int, r int) Choice`. Feasible = `ExactPass` and every B count at least `Threshold(r)`; the base wins when feasible; else least grid distance, then fewest knobs changed, then lowest index; empty feasible set gives `None` and the Pareto set of B-count vectors among `ExactPass` points; no weighted sum exists in the package. `SpreadLevel(g Grid, x int64) int` maps a Spread value to the nearest level, tie to the **lower** (not half to even). Source: CLB 7.2.3, 7.5, FR-CLB-030 to 034, 013 (the two exceptions).
- **D631.** `alias.go`: `Aliased(ei, ej []int64, tolBP int64) (bool, error)`: dot products and squared norms in 128 bits through `checked.Acc` (`AddMul`) if it can compare two 128-bit sums, else through a comparison helper added to `internal/checked` with its tests (Phase 4 decides; no local `math/bits` copy), the test `10 000 (ei.ej)^2 >= (10 000 - tolBP)(ei.ei)(ej.ej)`, a zero vector never aliased, a reading above `2^26` in absolute value or more than 16 measures excluded (returned as an error naming the index, the caller marks it `OK` false). Source: CLB 10.2, FR-CLB-061.
- **D632.** `process.go`: pure checks over a sequence of stored `Record`s: `CheckSequence(rs []Record, k Known, maxFactors int) []Problem` enforcing the stage order C0 to C9 with the re-entry rule, a stage of more than `maxFactors` knobs split by owner, no fourth attempt before a `review`, no search while an X or O fact of the pack fails, the seven stop conditions of 7.4.1 (a `stop` needs them), the lineage rule (a new lineage names its predecessor as `parent` and carries no `set` or digest of the old), a block used once, no search seed in 1 to 60, no search seed in a confirm. `Problem{ Record int; Line int; Rule string; Detail string }`, sorted. `maxFactors` is a parameter so the package does not import `experiment`. Source: CLB 7.3.11, 7.4, 11.2, 11.3, FR-CLB-021, 027, 035, 036, 038, 053, 054.
- **D633.** The record. `Record` holds typed slices in the order of 8.2 (header, documents, experiments, factors, unsearched, measures, criteria, screens, screened_in, dormant, findings, feasible, chosen, pareto, sets, confirm, facts, detectors, stability, claims, unmet, decision, notes); `Render(Record) ([]byte, error)` is canonical: LF, decimal integers (no float, no sign for non-negative), 16 lower-case hex digits, `-` for none, **two spaces** between the groups of the `pack`, `commit` and `config` lines (as the example of 19.5 and the grammar of 8.2 write them) and one space elsewhere, the fields of every list in the order of its subject. `Parse([]byte) (Record, error)` is strict: it accepts exactly what `Render` produces, so `Render(Parse(b)) == b` for every accepted `b` (the fuzz property); an unknown keyword (the `claim` word set is `stable not_stable one_neighbour undetermined`, 8.2), a wrong field count, a float, a keyword out of order or a `note` with a newline is an error. `Digest(b []byte) string` is the first 8 bytes of the SHA-256 as 16 hex digits (documents and `Render` text, EXP 8.2.2). Source: CLB 8.1 to 8.3, FR-CLB-040 to 043, 100, NFR-CLB-002; EXP 8.2.2.
- **D634.** `record_check.go`: `CheckRecord(r Record, k Known) []Problem` implements 11.2 for one record: closed keywords (by the parser), `knob` tokens resolve (`unsearched v1x` names a `V1x` row, `at_off` a `Z` row), `fact` and `detector` IDs resolve, a `set` key is a K or D row, `feasible n of N` with `n <= N`, `chosen` below the points, `screened_in` and `dormant` disjoint, a `set` of a dormant knob only under `revert`, a screen whose sign differs from the registry's has a `finding`, `screened_in` at most the `measure` lines unless a `factor` is composite, a `stop` stage has every `fact` as `pass`, `report` or `na`, a `claim stable` has three `stability` lines all STABLE and no edge, `one_neighbour` two with one edge, `undetermined` for a default-only record exactly one line, point `default` (E14), a `holdout` line names seeds 51..60 and says `in_sample no` and every other says `in_sample yes`, no `set` moves a D knob to its `off=` value without a `note`. Source: CLB 11.2, 14, FR-CLB-039, 060, 062, 063, 071, 100.
- **D635.** `overlay.go`: `Overlay(h Header, sets []Set, class func(key string) byte) ([]byte, error)`. It refuses a key whose class is not K or D (FR-CLB-050), writes the header comment `# calibration <stage> attempt <n> of lineage <8 hex>: see <record path>` (FR-CLB-051, 19.6), and for each key calls the one function of `internal/design` that writes the document of a change (EXP 3.3.4), splits its two lines (header, key line), and groups the key lines under one header per table, tables and keys in byte order. The key line is byte for byte the design function's. The path of a key comes from the caller (`Set.Path`), because a key of a table keyed by an ID (`Role.ExpectedWorkTicks`) needs the instance (Risks 3). Source: CLB 9.1, 11.3, FR-CLB-050, 051, EXP 3.3.4.
- **D636.** The dry run is `experiment/calibration_dry_test.go`, package `experiment_test`, importing the root, `config`, `experiment`, `internal/calibration` and no `indicator` (CLB 11, architecture.md 2.7). It starts with `testing.Short()` skip and states its level. Its declared test constants (CLB 11.4, FR-REF-025, to be listed in the PR): search block 1 (seeds 101 to 103), confirm seeds 1 to 3, 3 replicas, 400 ticks, transient 100, windows of 100. Criteria are on `Record.Status` (`ok`) and on nothing economic. Leg A has the eight asserts of 11.4; leg B the stimuli (b) to (i); **(a) is retired with the flag `d`** (D624). Leg B (c) does not depend on the reference prices of the rebalanced pack: it layers a test-local overlay that sets the cheapest tradable commodity's reference price to 96, the number of 19.1, so `BeliefStep` 100 has a resolution floor above it and `Validate` rejects it; the test reads the rejection from `Check` and does not hard-code it. Leg B (b) uses `Prepare` with `economysim.SolveSteadyState` and sets `cfg.WarmStart` from `SteadyState.Table`; the "digest of the table" in the record is the first 8 bytes of the world's `ConfigHash` (which includes the table, `hash.go` `pairsWarmStart`), because there is no other canonical bytes of a Go-built table outside `config/` tests. Source: CLB 11.4, 12.1 item 4, FR-CLB-091.
- **D637.** `testdata/calibration/` holds `README.md` (the grammar of 8.2 by example, where records live, the rule that a stage record is data about a decision and never a golden file of the economy, CLB 8.4), `dryrun.shape.txt` and the directory convention `<pack>/<stage>-<n>.txt`. `shape.txt` is the dry run's record with every hex digest and every count that depends on the pack replaced by `<hex16>` and `<n>`, compared line by line after the same masking of the run's own record; it asserts the shape of the procedure (keywords, order, points, replicas, seeds, blocks, decisions) and no number of the economy. The shape file sits beside the pack directories so the `<pack>/*.txt` glob of `TestLibraryRecords_Checked` never reads it (it holds masks no `Parse` accepts). It is a golden-like file outside the three sets of testing.md 7.1.1, justified by 7.2 and 10.4: it is written only with `-update`, masked of every economic number, and moves when the procedure changes and not when the economy does. Source: CLB 12.1 item 3, 11.4.
- **D638.** Fuzz: `FuzzGrid` (any base, bounds, step, rule: levels ascending, unique, in bounds, containing the base, no panic, neighbours adjacent to the base), `FuzzChoose` (any result set: the chosen point is feasible, the Pareto set is non-dominated, the base wins when feasible, no panic), `FuzzParse` (never panics, `Render(Parse(b)) == b` when `Parse` succeeds, an unknown keyword errors). Corpora in `internal/calibration/testdata/fuzz/<Name>/`, each pinned input with a `Fix:` comment (testing.md 7.5). Seeds via `f.Add` include the rows of 11.3 and the record of Phase 5. Source: CLB 11.2, 11.3, NFR-CLB-004; testing.md 7.4, 7.5.
- **D639.** Tests are table-driven, parallel where they share nothing, no assertion library (testing.md 3), `Requirement:` lines as 4.2 asks. Every registry, grid, choice, alias, process, record and overlay test runs under `-short` (FR-CLB-090); only the dry run skips. The record bytes are identical on `amd64` and `arm64` by construction (integers and `math/bits`); the two-architecture run is the nightly of plan 24 and this plan only names its test (`TestRecord_RenderIsStable`). Source: CLB 11, NFR-CLB-002; testing.md 7.1.

## Phases

### Phase 1: documents (no code)

Goal: the specs say what the tests will assert. Files: those below (the supervisor applies shared ones; this plan lists, never edits).

| # | File, section | Edit |
| --- | --- | --- |
| E1 | `spec-calibration.md` 4.2 (Flags cell), 4.2.1 (row `d`, closing paragraph), 6.2.2 prose before K-AGT05, 6.2.6, 6.2.11 notes (K-ORG01, SAV), 6.3 (S-SCL02 prose), 7.1.3, 7.2.3, 7.3 (C7a row and the line "Eight knobs here have the flag `d`"), 7.6, 11.4 leg B (a), 11.7, 20 (N-124 row), 22 question 9 | remove the flag `d` (D624); strike the token in K-SCL01, K-SCL02, K-AGT05 to K-AGT07, K-SPN02, K-SPN05, K-SPN10, K-SPN11, K-POP04, K-POP05, K-POP07, K-MER03 to K-MER06, K-MER08, K-MIG01, K-MIG02, K-MIG05, K-FIS10, K-FIS13, K-ORG01, K-SAV01 to K-SAV08 and S-SCL02, S-INF01, S-EXT06, S-MIG03, S-MER01, S-STO01 to S-STO07, S-RSK01 to S-RSK03, S-ORG03, S-ORG06, S-SAV01 to S-SAV03; `d` alone becomes `-` |
| E2 | `spec-calibration.md` 21, 11.7 and `specs/README.md` (Requirement IDs row of `CLB`) | FR-CLB-024 retired (ID never reused), the README row says "FR-CLB-001 to 091 and 100 to 109 (024 retired)" |
| E3 | `spec-calibration.md` 6.3 and 6.2 | the owed rows of D625; the table below; scope tokens of the registry-only groups renamed to the owner table's |
| E4 | `spec-calibration.md` K-SCL02 row, 6.2.1 prose, 23 (`spec-scale.md` row), 12.2 | `MinAgents` has a validation bound now (`bounds.go` `min_agents` 1 to `MaxHouseholds`, FR-SCL-020): the grid is bounded above; the ask to SCL is met |
| E5 | `spec-calibration.md` 8.2 | (a) say that a record's separators are two spaces on the `pack`, `commit` and `config` lines and one elsewhere (the example's); (b) say what `fact <ID>` and `detector <R-id>` spell (the example writes `STY-P02` and `R5a`; 4.4 writes `P02`) and that the check strips the prefix; (c) say how a `factor` and a `set` name the instance of a table keyed by an ID (Risks 3) |
| E6 | `spec-calibration.md` 19.5 | the example fails its own checks (no `measure` line, so `screened_in` has more knobs than measures; 11.2): add the `measure` lines, move `unsearched at_off K-MON01` to follow the `factor` line (8.2 puts it there; the example puts it after `detector`), or label the example as a format example only in the text, not only in `models/clb/README.md` |
| E7 | `spec-calibration.md` 11.4 leg B (c) | "with the money unit of 19.1" becomes "with a test-local overlay that sets the cheapest tradable commodity's reference price to 96" (D636) |
| E8 | `spec-calibration.md` 11.1 | the `in`/`chk` row and the `out` row: no change of text, but state that the scope-path map is the test's (D627) |
| E9 | `doc.go` Glossary | knob, damper strength, structural choice, registry (of knobs), base value, grid, neighbour, grid edge, screened in, dormant, alias group, composite, search block, confirm seeds, feasible region, Pareto set, lineage, calibration record, calibration overlay, stage (of a calibration), attempt, off value, structure group, scope token (the list of CLB 23 Glossary row, minus terms the code does not use) |
| E10 | `models/clb/registry.py`, `registry.out`, `README.md` | after E1 to E4: re-run in Docker (read-only mount, `--rm`); `registry.out` has no FAIL; `choose.py` README line "does not model the `d` comparison" struck |
| E11 | `spec-stylised-facts.md` 10.1, 10.2 (N-214 stale text) | belongs to plan 2-19 Phase 1; this plan edits nothing there |
| E13 | `spec-calibration.md` 4.4, 11.1 (moves row), 12.1, FR-CLB-007, NFR-CLB-003, 23 | IND 9.1 now has a `Result fields` column: strike "IND publishes no field list: owed" and "does not read a field list"; the moves check reads it (D623); NFR-CLB-003 and 12.1 list `internal/checked` beside `internal/design` (D620) |
| E14 | `spec-calibration.md` 7.4.3, 8.2 (claim line), FR-CLB-039; `spec-stylised-facts.md` 7.3.3, 8.5 | confirm only that 7.4.3 and FR-CLB-039 accept the existing claim `undetermined` for a default-only record (one `stability` line, point `default`; plan 2-19 D599); no new claim word, 8.2 already has it |
| E12 | `doc/roadmap-v1.md` Stage 2 CLB row | status "planned" to "done" at the exit gate (supervisor) |

Met since CLB 23 was written, verified by grep on 2026-10-22 and so not owed: the README sentence and `Taste.Role`/`RegionTaste` rows; `spec-stability.md` 9.10 and the L142/D97 wording; `ProductionHalted` as D24's counter (OBS 4.4.6); the IND 9.1 `Result fields` column; STY 7.3.1's edge clause and 7.3.3's file name; the layout lines of `internal/calibration/` in `architecture.md` 2.6 and `AGENTS.md`; the CFG 9.2 reader sentence; the `testing.md` 6.7 sentence; the index rows and `models/` rows.

Proposed classification for D625 (to confirm; flags `in`, `chk`, `out` are read off VIA 11 by `registry.py`, never typed from memory):

| Owner pairs (scope: fields) | Group | Class | Stage | Notes |
| --- | --- | --- | --- | --- |
| Config: `Edges`, `RegionPrices`, `Yields` | one group per section | S | C1 | containers; their rows are the next lines |
| Item: `Commodity`, `Quantity`; Output: `Commodity`, `Quantity`, `CostShare`, `Discard`; Tool: `Commodity`, `WearPerUse`; WantItem: `Commodity`, `Weight` | one group per scope (keys C1, quantities C2) | S | C1 / C2 | the registry-only token `Recipe output` becomes `Output`, `Recipe tool` `Tool`, `Want item` `WantItem` |
| Month and Season: `ID`, `Days`; SeasonYield: `Season`, `BP`; YieldProfile: `All`, `Recipe`, `Region`, `Seasons` | calendar and yield groups | S | C1 / C2 | the registry-only `Yield` (`All`, `Recipe`, `Region`, `Seasons`) becomes scope `YieldProfile`; S-PRO05 (`Region and recipe`, field `YieldProfile`, RES 7) is rescoped to `Config` field `Yields` (the container, as in the first row) and its other fields move to the `YieldProfile` group; `SeasonYield` (`Season`, `BP`) is a separate group row. Supervisor confirms |
| JobSpec: `Recipe`, `InProgressBP` | warm group | S | C2 | `InProgressBP` is the CFG 3.7 name with no scope |
| Config: `WarmStart`; SeedPrice, SeedWage, WarmGroup, WarmPool: all fields | warm-start groups | S | C2 | the `out` flag only where FR-CLB-109 allows (`Prices`, `Wages`, `Groups`; not `Pools`) |
| World: `Scale` | S-SCL02 gains it, scope `World` | S | C3 | the nested `Scale` of CFG 3.7 sits under `Group` or `World` by the table it is in; X6 |

- Exit check: the spec reads true to the checks of Phase 2; `registry.py` prints no FAIL. No Go code before this.

### Phase 2: the registry test

Goal: CLB 11.1 as a Go test over the final spec text. Files: `internal/calibration/registry_test.go`, `registry_parse_test.go` (table cells, section 6 rows, owner table, CFG 3.7 catalogue), `registry_owner_test.go`, `registry_via_test.go`, `scope_paths_test.go` (D627), `imports_test.go`. Tests (each cites its IDs; each has a failing fixture built by editing the parsed text in memory):

- `TestRegistry_RowsAndIDs` (FR-CLB-001, 101): 12 and 7 cells, unique IDs, retired IDs not reused.
- `TestRegistry_CoverageOwnerToRegistry`, `TestRegistry_CoverageRegistryToOwner`, `TestRegistry_CoverageCatalogue` (FR-CLB-002): both directions, the `(also <Scope>` remark (`Role.Upkeep`), `Solve option` and `Presentation` exempt, CFG 3.7 nested fields.
- `TestRegistry_ClassOffAndFlags` (FR-CLB-003, 103, 104): one class of `K D S P`, `off=` on D rows only, flag closed set, a row with `d` fails.
- `TestRegistry_ZGeoAndOut` (FR-CLB-108, 109): `z` only where the owner default is 0, none or `the world's`; `geo` needs default at least 1 or `z`; `required` default has lower bound at least 1; `out` only on the five solver-written fields.
- `TestRegistry_NoRestatedNumber` (FR-CLB-005): no Default or Bound column; a number only in a `lin` step, `set:` level or `off=`.
- `TestRegistry_SearchAndGrid` (FR-CLB-006, 107): Search cell `U`, `E<n>` (anchor Status `anchor`), `I<nn>`, `-`; a K or D row not `-`; Grid `-` with a Stage needs an `A<nn>`.
- `TestRegistry_MovesTokens` (FR-CLB-007, 105): form of 4.4, fact or `S<nn>` in STY, measure in IND 9.1 fields or a band key or an OBS event (rules file 4.3), at most four tokens.
- `TestRegistry_References` (FR-CLB-008): `L<nn>`, `D<nn>` in `spec-stability.md`, `I<nn>`, `A<nn>` in 6.4 and 6.5, `D<nn>=<Counter>` names an OBS counter.
- `TestRegistry_OwnerSections` (FR-CLB-102): each Owner cell resolves in the owner spec and is a section of the README's owner column for the pair; a README row that parses no section fails.
- `TestRegistry_OffAfterCut` (FR-CLB-108): the grid of 5.2 built on the owner default (through `Build`, so Phase 3 must exist first for this one test: it is added at the end of Phase 3) and cut leaves no level equal to the off value except the base.
- `TestRegistry_SolverFlags` (FR-CLB-009): `in` and `chk` equal the first table of VIA 11 (a key and a container unchecked; `.*` names every field of its table and of nested scopes).
- `TestImports_CalibrationStdlibAndDesign` (NFR-CLB-001, 003): parses the package's files and fails on any import outside the standard library, `internal/checked` and `internal/design`; `TestNoFloatClockGoroutine` (NFR-CLB-003) scans for `float32`, `float64`, `time.`, `math/bits` (D621: overflow goes through `internal/checked`), `math/rand`, `crypto/rand`, `go ` statements and package-level `var` in non-test files.
- Exit: the test's findings equal `registry.out` on the same spec (counts of rows, groups, pairs); `go test -short -race ./internal/calibration/` passes.

### Phase 3: grid, seeds, plan

Files: `grid.go`, `seeds.go`, `plan.go` and their tests. Tests (arithmetic of each expectation in a comment, testing.md 5.5):

- `TestGrid_Levels` (FR-CLB-010, 011, 013, 014): the table of 19.1 (`CostFloor` 7 000 gives 1 750 to 28 000; `EntryShare` 50 gives 12, 25, 50, 100, 200 with the half-to-even 12; `MinWorkShare`, `lin1000`; a `set`), a base at a bound, a base of 1, a base of 0 (`lin` only), `ErrNoGrid` for a `z` base of 0.
- `TestGrid_Cut` (FR-CLB-108): `RestBand` base 500, `lin250`, off 0 loses 0 and has `edge:off`; `TargetSpread` base 0 keeps its base and has `edge:bound`.
- `TestGrid_Overflow` (FR-CLB-013): a level above `math.MaxInt64` is an error, not a wrapped level.
- `TestGrid_Neighbours` (FR-CLB-012): an invalid level is skipped; `BeliefStep` 200 with the floor rule has no lower neighbour and `edge:rule`; `ExitTicks` 3 with level 2 invalid.
- `TestGrid_Widen` (FR-CLB-026): two more kept levels, never past `[Lo, Hi]` or the off value.
- `TestSeeds_Blocks` (FR-CLB-020, 021): block 1 is 101 to 200, disjoint from 1 to 60, error for 0; `TestSeeds_Threshold` (FR-CLB-013, 106): `Threshold(25) == 23`, `Threshold(10) == 9`, `Threshold(50) == 45`.
- `TestPlan_Screen`, `TestPlan_Explore`, `TestPlan_Confirm` (FR-CLB-022, 023, 025): 1+2k points fewer at an edge, at least 10 replicas by default, `max(16, 10k)` (k=1: 16, k=2: 20), control present.
- `FuzzGrid` (D638). Then add `TestRegistry_OffAfterCut` to Phase 2's file.
- Exit: coverage of the three files at least 90%.

### Phase 4: choose, alias, process

Files: `choose.go`, `alias.go`, `process.go` and tests.

- `TestChoose_BaseWinsWhenFeasible` (FR-CLB-031), `TestChoose_LeastDistance` (FR-CLB-032: distance, then fewest knobs, then lowest index), `TestChoose_NoneAndPareto` (FR-CLB-033; the Pareto set among exact-passing points; the table of 19.3), `TestChoose_Pure` (FR-CLB-034: same input, same output), `TestChoose_SeedThreshold` (FR-CLB-106), `TestSpreadLevel_TieLower` (FR-CLB-013).
- `TestAlias_*` (FR-CLB-061): collinear, independent, one measure, a zero vector, the 128-bit comparison at the largest inputs (`2^26`, 16 measures), a reading above `2^26` excluded.
- `TestProcess_StageOrderAndReentry` (FR-CLB-035), `TestProcess_SplitByOwner` (FR-CLB-035), `TestProcess_ThreeAttempts` (FR-CLB-027), `TestProcess_OpenXFact` (FR-CLB-036), `TestProcess_SevenStopConditions` (FR-CLB-038: one case per condition failing), `TestProcess_BlocksAndSeeds` (FR-CLB-020, 021), `TestProcess_Lineage` (FR-CLB-053, 054).
- `FuzzChoose` (D638). Exit: coverage at least 90%.

### Phase 5: record and overlay

Files: `record.go`, `record_parse.go`, `record_check.go`, `overlay.go`, tests, `internal/calibration/testdata/records/{valid,invalid}/` (package-local inputs: a minimal complete valid record built by this phase, one invalid record per check of 11.2, and the example of 19.5 as a parse and render input only, since it fails its own checks, E6).

- `TestRecord_RenderParseRender` (FR-CLB-041, 043), `TestRecord_UnknownKeyword` (FR-CLB-043), `TestRecord_NoFloatNoMapOrder` (FR-CLB-041), `TestRecord_Digest` (FR-CLB-040: known SHA-256 vector, 16 hex), `TestRecord_StabilityAndClaim` (FR-CLB-039, 100), `TestRecord_RenderIsStable` (NFR-CLB-002: a fixed record against a literal), `TestRecord_Fixture19_5` (parse-and-render only, on a copy corrected for E5/E6: `unsearched` moved right after `factor`, the `measure` lines added; the uncorrected example is expected to fail `Parse`).
- `TestCheckRecord_*`, one per rule of D634 with its invalid record (FR-CLB-039, 060, 062, 063, 071, 100); `TestCheckRecord_ScreenSignFinding` (a screen whose sign differs from the registry's has a `finding`). Requirement: FR-STY-092 and `TestCheckSequence_*` (D632).
- `TestOverlay_KeysAndHeader` (FR-CLB-050, 051), `TestOverlay_RefusesSAndP` (FR-CLB-050), `TestOverlay_ByteForByte` (each key line equals the design function's output), `TestOverlay_GroupsByTable`.
- `FuzzParse` (D638). Exit: coverage at least 90%; `go vet`, `staticcheck` clean.

### Phase 6: purity test, library records, testdata

Files: `overlay_purity_test.go`, `records_files_test.go`, `testdata/calibration/README.md`, `testdata/calibration/dryrun.shape.txt` (written in Phase 7 with `-update`).

- `TestOverlays_Purity` (FR-CLB-050, 051): every `../../testdata/chains/calibrated/*.toml` (none today; the test passes on the empty set and has an in-memory good and bad document, so the check can fail); each key resolves through the scope-path map to a K or D row, and the first line is the header comment.
- `TestLibraryRecords_Checked` (FR-CLB-040 to 043, 053, 054): every `../../testdata/calibration/<pack>/*.txt` and every `../../testdata/golden/stylised/*.stability.txt` that exists is parsed and checked as a sequence per pack; an empty set passes.
- Exit: the README says no stage record is a golden file (CLB 8.4).

### Phase 7: the dry run

Needs plan 20 (`experiment`, `internal/design`). File: `experiment/calibration_dry_test.go`; `testdata/calibration/dryrun.shape.txt`.

- `TestCalibrationDry_LegA` (FR-CLB-012, 020, 021, 022 (the plan test owns the default of 10), 025, 031, 033, 034, 038, 040, 041, 091): the eight asserts of 11.4 in order: the plan is the base and two neighbours of `World.CostFloor` (`geo`) and `World.BeliefEvidenceShare` (`lin1000`), five points, `Check` resolves them with no tick; `choose` twice gives one index; criteria every point meets give the base and an empty overlay; criteria none meets give `chosen none` and a `decision` other than `stop`; seeds by step (screen on block 1, confirm on 1 to 3, no search seed in 1 to 60); two runs give the same record bytes; each step's `render` digest equals `Digest` of `Render` of the stored result; the control is the base on the same seeds, measures and windows.
- `TestCalibrationDry_LegB` (FR-CLB-023, 026, 027, 032, 035, 036, 037, 038, 050, 051, 052, 108): (b) `Role.ExpectedWorkTicks` as a `Grid` of three values with `Prepare` regenerating the table per point, the record names the table digest per point and the table is generated after every other document; (c) the `BeliefStep` lower level rejected by `Check` is skipped and recorded, the run goes on, `edge:rule`; (d) a third criteria set where the base fails and one neighbour passes gives that neighbour, the overlay is one key that `config.Parse` accepts and `Validate` passes, and the world built from base plus overlay has the `ConfigHash` named as `config chosen`; (e) an explore whose feasible points touch its end is widened by two levels on a new block; (f) to (i) are pure checks on stored results (`CheckSequence`) and run no tick.
- Declared test constants listed in the PR description (FR-REF-025): block 1 seeds 101 to 103, confirm seeds 1 to 3, three replicas, 400 ticks, transient 100, windows of 100.
- Exit: skipped with `-short`; runs in the full run; `shape.txt` recorded with `-update` and explained (testing.md 7.2).

### Phase 8: exit gate

`gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; `go mod tidy` no diff; `lizard -l go -C 10 -a 8 -L 100 -w` on the new files (the record parser and the checks are the files at risk: split by keyword family, not by length); coverage at least 90% for `internal/calibration`, read from run R as plan-2-3 D279 does and the root unchanged; mutation pass on `internal/calibration` with `gomu` only in Docker (`--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS=4`, `~/.claude/CLAUDE.md`; the package is pure, so the equivalent-mutant list is short); `registry.py` prints no FAIL; the dry run passes in the full run (`go test -race ./experiment/ -run CalibrationDry`); `testdata/golden/off/` and the pinned hash have not moved (no root file changed). Gate runs R and F exactly as plan-2-1 D229 states them; this plan states which it used.

## Catalogue appends

| File | Append | Position, id or tag | Spec line |
| --- | --- | --- | --- |
| `config.go`, `bounds.go`, `defaults.go`, `validate.go`, `hash.go` | none: CLB adds no `Config` field, default or bound | n/a | spec-calibration.md 13, 14 |
| `errors.go`, `event.go`, `option.go`, `tick.go`, `snapshot.go`, `ledger.go` | none | n/a | spec-calibration.md 15, 16 |
| checkpoint tags | none: a calibration contributes no state | n/a | spec-calibration.md 16 |
| log keys, metrics | none: `internal/calibration` has no logger | n/a | spec-calibration.md 15.1 |
| `registry_test.go` `notBuilt` (root) | none: CLB is no scope of the owner table | n/a | testing.md 6.7 |
| `doc.go` Glossary | E9 (supervisor applies) | before any code uses a term | architecture.md 3.2 |
| `testdata/chains/calibrated/` | not created | n/a | REF 4.1 |

No hash field id and no checkpoint tag: the plan adds no field and no state.

## Cross-plan contract

Consumes:
- `design.Overlay(path string, value int64) ([]byte, error)` (internal/design; EXP 3.3.4) <- plan-2-20
- Spec, Factor, Level, Change, Criterion, Measure (experiment/experiment.go) <- plan-2-20
- OneAtATime, Spread, Grid, SameSeeds, PerPointSeeds (experiment/experiment.go) <- plan-2-20
- Spec.Prepare (experiment/experiment.go) <- plan-2-20
- Check, Run, RunUnit (experiment/run.go) <- plan-2-20
- Result (experiment/result.go), Record.Status (experiment/record.go) <- plan-2-20
- Render (experiment/render.go) <- plan-2-20
- MaxFactors (experiment/bounds.go) <- plan-2-20
- MaxDocuments (bounds.go) <- Stage 1
- the default-only `testdata/golden/stylised/<pack>.stability.txt` (testdata/golden/stylised/) <- plan-2-19
- World.ConfigHash (world.go), NewWorld (world.go), SolveSteadyState, SteadyState.Table (viability.go), config.Parse (config/config.go): Stage 1 code
- testdata/chains/base.toml, one.toml, warm-one.toml as plan 8 leaves them (testdata/chains/) <- plan-2-8 (the dry run asserts no economy number, D636)

Provides:
- Render, Parse, Digest, CheckRecord, CheckSequence (internal/calibration/record*.go, process.go) -> plan-2-24
- the registry test and the purity test as part of `go test -short ./...` (internal/calibration) -> plan-2-24
- testdata/calibration/README.md and the directory convention `<pack>/<stage>-<n>.txt` (testdata/calibration/) -> plan-2-24
- TestCalibrationDry_LegA and TestCalibrationDry_LegB (experiment/calibration_dry_test.go) -> plan-2-24

(The facts, band keys and presets of STY and IND are read as spec text. Of plan 19's provides lines marked `-> plan-2-21`, this plan uses only the stability record file; `verdict.*`, the band files, `overlay/sink.toml`, the stylised flags and `<pack>.txt` are not consumed, and plan 19 already marks them "no consumer, available". It consumes nothing from plan 18 (`indicator.*`) and no `verdict.*` of plan 19: `internal/calibration` imports only `internal/checked` and `internal/design` (D620), so it cannot. The Needs entry for plan 18 is a build-order constraint only (plan-2-0 line 39), not something plan-2-0 is to drop.)

## Testing

Testing entry (testing.md 11): the requirement-to-test table is the one in Phases 2 to 7; the review items are FR-CLB-004 (the registry test fails a change that adds a `Config` field without its row, via owner-table coverage; a solve option is a review item), 070, 072, 073, 074, 080, 081, 090.

| Requirement | Tests |
| --- | --- |
| FR-CLB-001, 002, 003, 005, 006, 007, 008, 009, 101 to 105, 107 to 109 | Phase 2 `TestRegistry_*` |
| FR-CLB-010 to 014, 026 | Phase 3 `TestGrid_*` |
| FR-CLB-020 to 023, 025, 106 | Phase 3 `TestSeeds_*`, `TestPlan_*`; Phase 7 leg A |
| FR-CLB-024 | retired (D624) |
| FR-CLB-027, 035, 036, 038 | Phase 4 `TestProcess_*`; Phase 7 leg B (f) to (i) |
| FR-CLB-030 to 034 | Phase 4 `TestChoose_*`; Phase 7 leg A asserts 2 to 4, leg B (d) |
| FR-CLB-037, 052 | Phase 7 leg B (b) |
| FR-CLB-039, 100; FR-STY-092 | Phase 5 `TestRecord_StabilityAndClaim`, `TestCheckRecord_*` (FR-STY-092: `TestCheckRecord_ScreenSignFinding`, D634) |
| FR-CLB-040 to 043 | Phase 5 `TestRecord_*`, `FuzzParse`; Phase 7 leg A asserts 6, 7 |
| FR-CLB-050, 051 | Phase 5 `TestOverlay_*`; Phase 6 `TestOverlays_Purity`; Phase 7 leg B (d) |
| FR-CLB-053, 054, 060, 062, 063, 071 | Phase 4 `TestProcess_Lineage`; Phase 5 `TestCheckRecord_*` |
| FR-CLB-061 | Phase 4 `TestAlias_*` |
| FR-CLB-091 | Phase 7 (skip with `-short`) |
| CLB-S1 to CLB-S3 (S88 to S90, CLB 11) | S88 (the dry run) `TestCalibrationDry_LegA` and `_LegB`, on pack `one` as its fixture; S89 (coverage) `TestRegistry_CoverageOwnerToRegistry`, `_CoverageRegistryToOwner`, `_CoverageCatalogue`; S90 (the alias test) `TestAlias_*` |
| NFR-CLB-001 to 005 | Phase 2 import and source guards; Phase 5 `TestRecord_RenderIsStable`; the three fuzz targets; the coverage run |

Golden files: `testdata/golden/off/` unchanged (no root file is touched); no module-alone golden and no all-on golden (CLB is not a module); no fork-at-N test (CLB 16: no state). The only recorded file is `testdata/calibration/dryrun.shape.txt`, written with `-update` and masked of every economic number. Registry and catalogue rows: the CLB registry test is this plan's; the root registry test and the event catalogue test are unaffected. Fuzz targets: `FuzzGrid`, `FuzzChoose`, `FuzzParse` (D638). The race detector runs the package tests; the dry run is five points of three seeds of 400 ticks plus the confirm, about a minute without `-race` at the spec's placeholder (CLB 11.4), and Phase 7 measures it under `-race` (Risks 7).

## Observability

Observability entry (observability-and-logging.md 9): **None, because** the package is pure, has no `*slog.Logger`, emits no event, adds no `Warn`, no log key and no metric, and runs no loop of a tick (CLB 15, 15.1; observability-and-logging.md 2.3: a caller's needs go in the return value). A failure is an `error`, a decision a line of the record. It reads events only through `Known` (the registry test parses the event table of rules 4.3 as text to validate measure names).

## Risks and open items

1. **FR-CLB-004 has no mechanical gate for a solve option** (owner: CLB spec; review item as 11.7 says). A new `SolveOption` of `viability.go` needs a registry row (scope `Solve option`, exempt from coverage); nothing fails if it is forgotten. Not decided here.
2. **The worked record of 19.5 fails 11.2** (no `measure` lines for its `screened_in` knobs; `unsearched` after `detector` against 8.2's order; `fact STY-P02` and `detector R5a` spellings). E5 and E6 owe the spec edit; until then the fixture is parse-and-render only. Owner: CLB spec, supervisor.
3. **A `factor` or `set` for a table keyed by an ID has no instance** (`Role.ExpectedWorkTicks` is `role.<id>.expected_work_ticks` in the loader, EXP 3.2.1; the record writes the registry key only). Owner: CLB 8.2 with EXP 3.2.1. D635 takes the path from the caller (`Set.Path`) so the code is not blocked; the record grammar needs a field or a rule. The dry run uses the path in the factor of `Spec` and writes no `set` for it.
4. **`design.Overlay(path string, value int64) ([]byte, error)` is fixed by plan 20 D614**; D635 splits its two lines. If plan 20's output is not a header line and a key line, only Phase 5's wrapper changes.
5. **The 128-bit comparison of D631 may not fit `internal/checked`** (`Acc` exposes `AddMul` and `DivHalfEven`, no comparison). Phase 4 adds a helper to `internal/checked` with its tests rather than a local copy; NFR-CLB-003 and 12.1 are edited by E13.
6. **Plan 19 ships default-only stability records** (its D599) whose claim is `undetermined`, already in the closed set of 8.2. `CheckRecord` accepts that claim with exactly one `stability` line, point `default`, and rejects `one_neighbour` with fewer than two lines; `TestLibraryRecords_Checked` then passes plan 19's files. Owner: plan 19 for the files, this plan for the check.
11. **`MaxDocuments` (16) caps a screen below the 16-knob stage C4a** (plan 20 Risks 1: a base of 3 documents leaves 13 changes). The dry run uses two factors and is unaffected; `CheckSequence` takes `maxFactors` as a parameter and the stage split of D632 must also respect the document cap. Owner: EXP, CFG, plan 20. Note proposal below.
7. **Dry-run cost under `-race` is unmeasured.** Plan 1's race-aware runner is in the root package's tests and is not importable from `experiment_test`. If the race run exceeds ten minutes the dry run takes a smaller declared constant (fewer ticks) and says so in its description; the spec's constants are placeholders (CLB 11.4).
8. **Pack `one` collapses (N-312, N-314)**: the dry run asserts no economy number, so the collapse does not fail it; a run whose `Status` is not `ok` would. Plan 8 owns the rebalance; leg B (c) is independent of its prices (D636).
9. **Pairing is plan 20's.** Plan 20 (D602) has applied the `Pair` and fingerprint removal; CLB neither uses nor asserts `Pair` after D624, so no CLB test depends on it.
10. **The registry reads IND and STY as text; no code of plans 18 or 19 is consumed.** If those plans rename a result field or a band key, the spec table changes first and the registry test finds the drift. A cross-check from `indicator`'s own test package against the exported fields is possible and not done.

**Note proposals (text only; the supervisor is the only editor of `doc/notes.md`):**
- N-214 update: "(plan 2-21, Phase 1) CLB: the flag `d` is removed (4.2, 4.2.1, 7.1.3, 7.2.3, 7.3, 11.4 leg B (a), rows of 31 knobs and 20 groups) and FR-CLB-024 is retired. Open for STB and TEC."
- N-220 update: "(plan 2-21, Phase 1) the 39 owner pairs and 13 registry-only pairs are classified (proposal in the plan); `registry.py` and the Go test (`internal/calibration/registry_test.go`) print no FAIL after the edit. Close when the supervisor confirms the classes."
- N-124 update: "(plan 2-21) CLB side applied: no knob needs a distribution comparison for its draw count; the comparison remains the experiment's choice. Open for EXP."
- New note (areas CLB, REF): "The worked record of `spec-calibration.md` 19.5 fails the record checks of 11.2; fix the example or mark it. A `set` and a `factor` for a table keyed by an ID need an instance (EXP 3.2.1 path)."
- The 16-knob screen against `MaxDocuments` (Risks 11) is plan 20's note (its Proposed N-NEW, areas EXP, CFG, CLB); none is added here.
- Update to the note plan 19 proposes on the default-only stability record: "plan 21 checks the records with `TestLibraryRecords_Checked`; the neighbour claim waits for the grid." No new note here.
- N-015 update: "(plan 2-21) the procedure constants are named constants in `internal/calibration` with the spec section in a comment; they stay placeholders."
