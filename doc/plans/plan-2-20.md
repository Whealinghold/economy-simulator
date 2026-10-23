# Plan: Stage 2, row 20: EXP, `experiment/`, `internal/design`, `internal/reduce`

**Status: plan written (2026-10-22), not audited. Contract: [plan-2-0.md](plan-2-0.md).** Builds the sweep harness: sweeps, replicas, branches from a checkpoint, integer reductions, a comparison, a sensitivity measure and the canonical rendering. It is the "run again" of the tuning loop; it changes no world, no configuration field and no hash.

Rule tags read (the files show the tags of the contract, no difference): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1.
Notes read (index, then the entries tagged EXP or ALL with their updates): EXP N-015, N-124, N-127, N-129, N-134, N-162, N-179, N-203, N-206, N-207, N-211, N-214, N-220; ALL N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217. The disposition of each is in "Risks and open items".
Specs implemented: [spec-experiments.md](../specs/spec-experiments.md) (EXP) sections 3 to 14 and 21: FR-EXP-001 to 005, 007 to 024, 030 to 034, 040 to 047, 050 to 053, 060, 061, 063 to 065, 070 to 072 NFR-EXP-001 to 006 and FR-STY-090 (the harness half; plan 19 provides the verdict). **FR-EXP-062 is retired** (D602; there is no FR-EXP-006). Also read: spec-checkpoint.md 3, 5.4, 6 (as plan 2 and plan 1 leave it), spec-indicators.md 3.1, 5.2, 5.5, 10.5 and FR-IND-001, spec-world.md 7.6, 12, ADR 0002, ADR 0008, spec-config.md 6.3 and 7.1.
Decision block: **D600 to D619**.

## Scope

**Builds:**

| Item | Home |
| --- | --- |
| `Spec`, `Factor`, `Level`, `Change`, `Step`, `Measure`, `Sample`, `Criterion`, `Branch`, `Variant`, `Keep`, the closed sets, `Validate`, the bounds of EXP 13.2, the sentinels | `experiment/experiment.go`, `bounds.go`, `errors.go`, `validate.go` (D600) |
| `Key`, `Unit`, `Plan`, `RunUnit`, `ExecuteUnit`, `Run`, `RunOption`, `WithRecordFunc`, `WithPrior`, `Check`, `PointCheck`, `Warning` | `experiment/plan.go`, `resolve.go`, `run.go`, `segment.go`, `classify.go` |
| The branch procedure | `experiment/branch.go` |
| `Record`, `Reading`, `Status`, `Result`, `PointResult`, `VariantResult`, `Assemble` | `experiment/record.go`, `result.go` |
| `Reduction`, `Group`, `Compare`, `Comparison`, `MainEffects`, `Elasticity` | `experiment/reduce.go`, `compare.go`, `sensitivity.go` |
| `Render` | `experiment/render.go` |
| Designs, the Feistel permutation, seeds, the overlay text, the path syntax, checked counts | `internal/design` |
| 128-bit sums, half-to-even division, the 128-bit nearest integer root, the nearest-rank quantile, the exact sign tail | `internal/reduce` |
| Tests that need `indicator` (the equivalence test of `internal/reduce` against `internal/measure`, which builds its own run inside the test; one `Tally` measure through the harness) | `indicator/reduce_equivalence_test.go`, `indicator/experiment_measure_test.go` (external test package; D603) |
| `testdata/golden/experiment/one.txt` | new directory (D615) |

**Leaves out, with the owner:** any measure, band or preset (IND, plan 18; STY, plan 19, except the sweep test of FR-STY-090); the registry of knobs and the calibration loop (CLB, plan 21); a path setter over a Go-built `Config` (EXP open question 1; `ExecuteUnit` is the seam); a parallel runner or pool (architecture.md 12.2; the unit is the seam, EXP 5.3); `stale_warm_start` and `no_effect` (D616); the stream fingerprint, `Spec.Pair`, `Record.Stream`, `Comparison.AlignedPairs` and `Horizon` (D602); a file or log output of any kind (EXP 3.7, 15); the examples (plan 24); the two-architecture CI (plan 1 D236, plan 24).

## Decisions

Each decision has its source. Where a spec is silent or stale the item is an owed edit in Phase 1 and a line in "Risks".

- **D600. Files.** The package is `experiment` at `<module>/experiment` (module `github.com/Whealinghold/economy-sim`, root package `economysim`). EXP 23 names `experiment.go`, `run.go`, `branch.go`, `result.go`, `reduce.go`, `render.go`; the Lizard limits (`-C 10 -a 8 -L 100`, N-243) need more files, so the concept files above are continuation files of those and each maps to area EXP; its `_test.go` carries the `Requirement:` lines (architecture.md 2.6). EXP 23 is amended (Phase 1). Source: EXP 23; architecture.md 2.5, 2.6; N-243.
- **D601. Names fixed here** (EXP 3.1: "provisional until the code"). Closed sets: `type Design int` (`Grid`, `OneAtATime`, `Spread`), `type Seeding int` (`SameSeeds`, `PerPointSeeds`), `type Op int` (`AtMost`, `AtLeast`, `Between`), `type Status string` with the codes of EXP 13.4 as the values and constants `StatusOK`, `StatusUnresolved`, `StatusInvalidConfig`, `StatusWorld`, `StatusScript`, `StatusInvalidTrade`, `StatusAudit`, `StatusCheckpoint`, `StatusStep`, `StatusMissing`. `type RunOption func(*runConfig)`. `Keep struct{ Series, Snapshot bool }`. `MainEffects(r Result, measure string, factor int, bins int) ([]Effect, error)` and `Elasticity(r Result, measure string, factor, level int) (Sensitivity, error)` (EXP 8.5.1 "names provisional"). `Quantiles` is an unexported array of the closed list 1 000, 2 500, 5 000, 7 500, 9 000 and the exported `func Quantile(i int) int` reads it (architecture.md 7.1: a table never mutated). No exported name begins with `Experiment` (EXP 23). Source: EXP 3.1, 8.5, 13.4, 23.
- **D602. The stream fingerprint is removed (N-214).** ADR 0008 decision 6 and `spec-world.md` 7.6: runs of one seed are aligned by construction and diverge through state only, and a checkpoint holds no stream state (plan 2 D241 builds no `Stream`). So `Spec.Pair`, `Record.Stream`, the `stream` line of `Render`, `Comparison.AlignedPairs` and `Horizon`, the `Pair` row of the `Validate` table and FR-EXP-062 are removed from the spec text (Phase 1) and not built. **The pairing gain stays**: `Compare` always computes `PairingGain`, `SdIndep`, `SdDiff`, `Identical` from the readings (EXP 7.4.2, which already says so), "`Pair` keeps measuring the gain" (ADR 0008) read as that. FR-EXP-062's ID is retired, not reused (testing.md 4.1). `Result.Format` stays 1: nothing released has rendered a stream line and no golden holds one. Source: N-214, N-124, ADR 0008 6, WLD 7.6, plan-2-1 D224.
- **D603. Imports and where tests live.** `experiment/` imports the standard library, `economysim`, `config`, `internal/design`, `internal/reduce` and `internal/checked` (for `Mul` in the size bounds) and nothing else; `internal/design` adds `internal/ordering` (`Finalise`). Neither `experiment/` nor its tests import `indicator` or `internal/measure` (architecture.md 2.7, FR-IND-001: only `indicator`, its tests, `otelobserver` and `examples/` may import `indicator`). The tests that need both live in the external test package of `indicator/` (plan 18 D576). `import_guard_test.go` `importProblems` has cases for `.` and `internal` only (read), so the harness's rule is added: a file under `experiment/` may not import `indicator` or `strategy`, and no file outside `indicator/` imports `internal/measure` (if plan 18 already adds the second, only the first is added). `determinism_guard_test.go` already lists `experiment/` in `guardedFiles`. Source: architecture.md 2.7, 2.9; EXP 3.4.0, X1; FR-IND-001.
- **D604. Resolving a point.** `Parse(base..., overlays...)` per unit (no cache: EXP 5.3.2). The overlays of a point are the documents of its levels in factor order, each level's `Docs` first, then one document per `Change` (EXP 3.3.2, 3.3.4, FR-EXP-003; bytes by `design.Overlay`). `config.Parse` refuses more than `economysim.MaxDocuments` = 16 documents (read, `bounds.go:123`), so a point with `len(Base) + sum over factors of (len(Docs)+len(Changes))` above 16 would be `unresolved` for a reason the loader's message shows only at run time. `Validate` therefore returns `ErrOversize` when the largest such count over the levels of each factor exceeds `MaxDocuments` (the largest level of each factor, summed). EXP 13.2 gives `MaxFactors` 16 and CLB's screen puts one factor per knob, so the cap binds first; EXP 3.6 and 13.2 define `ErrOversize` for points, runs and cells only, so this document cap is a rule owed to those sections (Phase 1 edit 2, Risks 1), cited to FR-EXP-004 for the sentinel and not to FR-EXP-064 until the spec says so; this is a finding (Risks 1), not a decision to coalesce documents, which FR-EXP-003 forbids. Then `Prepare`, then `Config.Validate`. Source: EXP 3.3, 5.1.3; CFG 6.3.1.
- **D605. The run of one key** is EXP 5.1.3 as written, with these choices the text leaves open. The boundaries are the sorted union of `Transient`, every window end, every step tick and `Ticks`. `w.Run(ctx, n)` is called for each non-empty segment. At a boundary the order is close, steps, open (FR-EXP-019). A cancelled run: after `w.Run` returns an error, `w.Err() != nil` means the world failed (classify, D606); otherwise `ctx.Err() != nil` means cancelled and the run is discarded (EXP 5.4.2); `Run` and `RunUnit` check `ctx` before each unit. `Run` starts no goroutine and holds one world at a time. Source: EXP 5.1.3, 5.2, 5.4; architecture.md 12.2, 12.3.
- **D606. Failure classification** is one function `classify(stage, err) (Status, tick)` in `classify.go`, a table of the rows of EXP 9.1, so two code paths cannot give one error two codes: at `NewWorld`, `errors.Is(err, ErrAuditFailed)` is `audit` (tick 0), then `errors.As(*ValidationError)` is `invalid_config`, else `world`; at `RestoreWorld` every error is `world`; in a step, `ErrInvalidTrade` is `invalid_trade`, `ErrAuditFailed` is `audit`, anything else `step`; `Step.Do` errors are `script`; a `Checkpoint` error on a healthy world is `checkpoint`. The error text is `Detail`; a failed run has `Final` all undefined, `Pass` all false, `Digest` zero, and keeps `Hash` once a world was built. A panic is not recovered (EXP 9.1.2). Source: EXP 9.1 to 9.4.
- **D607. Readings.** A `Measure.Fn` result with `|value| > MaxReading` (2^40) is undefined with an `ErrRange` detail (FR-EXP-047); a returned error is undefined with the first error in `Detail`, the run goes on (FR-EXP-053). With `Every` 0 there is one window, the final reading is the series, and the record keeps it once: `Series` and `Ticks` are nil (EXP 3.4.3 "kept once"; clarified in Phase 1). With `Every` above 0 and `Keep.Series`, `Series[w]` holds the window readings and `Ticks[w]` its end tick; a failed run keeps the windows that closed before the failure (EXP 9.2). The span (EXP 3.4.3) is a second pair of calls: at `Transient` the harness calls `Begin` with `span` false for the first window and with `span` true for the span; at each window close `Fn` with `Span` false; at `Ticks` `Fn` with `Span` true, which is the final reading. With `Every` 0 the window and the span coincide and `Begin` and `Fn` are called once, with `Span` true (EXP 3.4.3 "kept once"). The span stays open across the branch point: the burn-in's span accumulator is what `Fork` copies (D609), so a branch continues it. Source: EXP 3.4, 8.1, 9.2.
- **D608. The digest** is the last 32 bytes of `Checkpoint().MarshalBinary()` taken at `Ticks` on a healthy world (EXP 5.1.3 step 4, CHK 5.4). The harness never checkpoints or snapshots a failed world (FR-EXP-051). Plan 2 gives `Info()` without the digest, so this copies the encoding once per run (about 2 MB at the first consumer's size, CHK 10.2): a `Digest()` accessor is an open item (Risks 8), not a decision. Source: EXP 5.1.3, 9.4; plan-2-2 D241.
- **D609. Branching.** For a unit: build the burn-in (`Options` with the key of variant 0, continuation 0), run to `At` through the boundaries below `At`, close the window ending at `At`, apply the steps below `At`, take the checkpoint, drop the burn-in world. For each variant `v` and continuation `c` in order: `RestoreWorld(cfg, cp, Options(key)..., observers...)` plus `WithReseed(Salt0+c)` only when `Continuations >= 2` (FR-EXP-032; never `WithSeed`: it is `ErrOptionContext`), call each measure's `Fork(burnInKey, branchKey)`, apply the steps at or above `At` in the order script, levels, variant (a step at `At` runs right after the restore), open the window that starts at `At` after those steps, and run `[At, Ticks)`. Branch (0,0) has the burn-in's key, so `Fork(k, k)` is called with equal keys and means "keep"; this is stated in EXP 6.1.1 by Phase 1 (Risks 2). A burn-in failure is recorded for every key of the unit with its status and tick and no branch runs (FR-EXP-033). Worlds are held one at a time; the records of a branch carry the burn-in windows as the first part of the series. Source: EXP 6, 5.5.2; CHK 6.2, 6.8; FR-EXP-030 to 034.
- **D610. Options.** `Options(key)` is called once for each world built (the burn-in; each restore), with the key of that world; the harness appends the observers of the measures that have one, in measure order, after the consumer's options, and `WithSeed(seed)` last on a build and not at all on a restore; a nil `Options` gives no options; with no `Observer` on any measure no observer is registered (FR-EXP-017). An `Options` that returns `WithSeed` or `WithReseed` gives `world` for a branched unit and is overridden for an unbranched one (FR-EXP-018, 024). One instance per world of every stateful extension is the consumer's duty (N-129); the harness cannot see a shared one. Source: EXP 3.4.4, 5.5; N-129.
- **D611. Assemble.** `Assemble(e, recs)` runs `Plan`, sorts the records by key, returns `ErrUnknownRun` for a key outside the plan and `ErrDuplicateRun` for a key twice with different contents (`reflect.DeepEqual` is not used: a field-wise `equalRecord`, because `Record` holds `Snapshot` slices), accepts an identical duplicate, and fills an absent key with a `missing` record. `PointResult.Hash` is the first non-zero hash among the point's records in key order; `SameAs` is the first earlier point with an equal non-zero hash, else -1, and an equal hash adds the warning `duplicate_point` (FR-EXP-063, 071); `PointResult.Status` is the status of the first record in key order whose status is `unresolved` or `invalid_config`, else `ok` (EXP 8.2.1 says "the point's resolution"; the derivation is clarified in Phase 1); `Levels` and `Values` come from `internal/design` and the spec alone. Source: EXP 5.3.4, 8.2.1, 14.4.
- **D612. Reductions, `Compare`, sensitivity.** `Reduction{Planned, Used, Failed, Undefined, Missing int; Min, Max, Mean, Std Reading; Q [5]Reading}` and `PassShare{Passes, Planned, ShareBP int}`; the arithmetic is only calls into `internal/reduce`, the group is the records of one (point, variant) over replicas and continuations (EXP 8.3). `Compare(r, a, b Group, measure)` matches by (replica, continuation), gives the fields of EXP 8.4.2 less `AlignedPairs` and `Horizon` (D602), is unpaired (`Pairs` 0) for two points of a `PerPointSeeds` experiment, and returns `OK` false beyond 62 pairs for the sign tail. `MainEffects` and `Elasticity` are EXP 8.5. A reading above `MaxReading` never enters a sum (D607). Source: EXP 8.3 to 8.5.
- **D613. `internal/reduce`.** Stdlib and `math/bits` only, no float, no `math`. `Int128` signed two's complement with checked `Add`, `Sub`, `Mul` (an overflow returns `ok` false), `Cmp`; `RDiv(n, d Int128) (Int128, bool)` half to even; `IsqrtNearest(n Int128) uint64` (the floor root `f`, plus 1 when `n >= f*f + f + 1`); `Quantile(sorted []int64, bp int) int64` (rank `max(1, ceil(bp*n/10000))`); `Moments` (count, sum, sum of squares in 128 bits, `Std` as in EXP 8.3); `SignTailPPM(wins, losses int) (int64, bool)` exact for `n <= 62`; the 3.2.3 and 4.4.3 products in 128 bits. It does not import `internal/measure`. Source: EXP 8.3, 8.3.2, 23; IND 5.2.2, 5.5.2.
- **D614. `internal/design`.** Primitive ints and strings only: `GridCount`, `GridCoords(index, levels, dst)`, `OATCount`, `OATCoords`, `SpreadCell(salt uint64, axis, n, i int) int` (EXP 4.4.2: `S = salt ^ 0x243F6A8885A308D3`, gamma `0x9E3779B97F4A7C15`, `k = fin(fin(S + (axis+1)*G) + n)`, four Feistel rounds over `2w` bits, cycle walking), `SpreadLevel`, `SpreadValue` (128-bit, floor), `RangeValue(k, steps, min, max)` (half to even), `Seed(seed0 uint64, perPoint bool, p, r, replicas int) uint64`, `ValidPath`, `Overlay(path string, value int64) ([]byte, error)`, `CheckedCount(factors ...int) (int, bool)`. Constants are rule constants beside their code (architecture.md 9.6); the design constant `0x243F6A8885A308D3` is not a reseed constant (EXP 4.4.2). The oracle of the tests is the output of `doc/specs/models/exp/lhs.out` and `worked.out`, copied into tables with a comment naming the file. Source: EXP 3.2, 4, 23.
- **D615. `Render` and its golden.** `Render(r Result) string` is EXP 8.2.2 less the `stream` line (D602). The golden is `testdata/golden/experiment/one.txt` (EXP 11.1 says `testdata/golden/experiment-one.txt`; the directory is allowed by testing.md 7.1.1 through plan 18 Phase 1; edit owed to EXP 11.1). It is written only with `-update`, flag defined in `experiment/golden_test.go`, refused under `-race` (plan 1 D228). It sweeps pack `one` (`packOne` layers, after plan 8 rebalances it): factor `world.belief_step` at 100, 200, 400, three seeds, 120 ticks, window 30, one branch of two variants at tick 60; snapshot-only measures (`households`, `money`, `agents`) and one recording-observer measure that counts events. The spec's 400 ticks and ten seeds (EXP 11.1) are the scenarios' size, not the harness's: this golden proves the bytes, not the economy (N-027). Source: EXP 8.2.2, 11.1; testing.md 7.1, 7.2.
- **D616. `Check`.** `Check(ctx, e) ([]PointCheck, error)` resolves every point (parse, overlays, `Prepare`, `Config.Validate`), builds no world, runs no tick, calls no `Options`, computes no hash (EXP 14.3), and reports `ranged_narrower_than_design` for a spread over a range with fewer values than `N` (FR-EXP-071). **Not built:** `stale_warm_start` (EXP 14.4 says "not specified here"; N-134 says VIA 11 now publishes the path list, so the rule can be specified; owed edit, Risks 5) and `no_effect` (needs a pure hash function of a `Config`, CFG X7). Source: EXP 14.3, 14.4.
- **D617. Bounds and sentinels.** The bounds of EXP 13.2 are `const`s in `experiment/bounds.go` with doc lines saying they are placeholders (N-015) (the home owed to architecture.md 9.6 by Phase 1 item 5a, not the root `bounds.go`); the sentinels of EXP 14.1 are in `experiment/errors.go`; `Validate` returns `*economysim.ValidationError` values (`Path` is `[]string` as in errors.go, one segment per step, e.g. `{"Factors[2]","Levels[0]","Changes[1]","Path"}`; `Rule` the sentinel) sorted by path then rule, at most `MaxErrors` 100, joined with `errors.Join` and followed by `economysim.ErrTooManyErrors` (spec-config 7.1.3); the root's sort and cut helpers are unexported, so `experiment/validate.go` carries its own copy and `TestValidate_AtMost100Errors` also checks order and cut against 7.1.3; `ErrDuplicateID` is the root's. All products are `internal/checked.Mul` (`ErrOversize` on overflow). Source: EXP 3.6, 13.2, 14.1.
- **D618. The registry link (FR-EXP-072).** `experiment/registry_test.go` reads `doc/specs/README.md` by relative path, resolves each path of the golden `Spec` through the scope table of EXP 11.3.1 and the snake form of the row's parameter name, and fails for a path with no row. The shipped experiments of STY and CLB are plans 19 and 21's; Go test code cannot be imported across packages, so each repeats the call or the helper moves (Risks 6). Source: EXP 11.3; testing.md 6.7.
- **D619. Gates and sizes.** Integration tests use at most 40 ticks and 3 seeds (the golden: D615) so the package finishes under `-race`; all world-building tests begin with the `testing.Short()` skip (EXP 10.6). Mutation testing runs on `internal/design` and `internal/reduce` in Docker, `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`, one file at a time (`~/.claude/CLAUDE.md`). The golden and the digests are compared on amd64 and arm64 by hand until plan 1's CI decision lands (plan-2-1 D236, D237). Source: plan-2-0 gates; testing.md 9.

## Phases

### Phase 1: the documents (no code)

Goal: every statement the code relies on is true. Files: none by this plan; the supervisor applies these (plan-2-0). `AGENTS.md` "Before writing or changing a spec": the supervisor quotes the notes tag when it edits EXP.

**Edits owed.**
1. `spec-experiments.md` 7.2 to 7.5, 2 (position 8), 3.1 (`Pair`, the field comment), 3.6 (`Pair` clause of the first row), 5.1.3 (step 3a "with `Pair` take the fingerprint"), 5.6, 6.3 (first row "same words from the same position"), 8.1.1 (`Stream`), 8.2.2 (the `stream` line), 8.4.2 (`AlignedPairs`, `Horizon`), 10.2, 11.1 and 11.2 (the fingerprint row), 12.2 ("`Pair` on when CHK offers the read"), 14, 15.2 and FR-EXP-065 (`Pair` clause), 19.6, 22 question 4, X3 and the owed row for `spec-checkpoint.md`: rewrite to D602. 7.2.1 and 7.2.2 state "aligned by construction" (WLD 7.6.1) and keep the estimate/gain text of 7.4.1 to 7.4.2; 7.2.3's "Known to break alignment" list (a binding right, TEC D16) is stale under ADR 0008 (N-162: a right or a clamp is aligned in the draws; what still breaks a pair is an ID shift, WLD 7.6.2); 7.4.3 and `models/exp/pairing.py` describe a toy on one positional stream and are regenerated or labelled; FR-EXP-062 is deleted and its ID noted as retired in 21.
2. `spec-experiments.md` 3.4.3 and 8.1.1: with `Every` 0 `Series` and `Ticks` are nil (D607). 6.1.1: `Fork(k, k)` is called for branch (0,0) and means keep (D609). 8.2.1: the derivation of `PointResult.Status` (D611). 23: the continuation files (D600). 11.1: the golden path (D615). 13.2: see Risks 1.
3. `spec-experiments.md` 14.4, 3.3.3 and 22 question 7: apply N-134, specify `stale_warm_start` from the table of VIA 11 (owner: the EXP spec; D616 builds nothing until then).
4. `spec-indicators.md` 4.2.5, 10.5.1, FR-IND-055 (X6) and 11.2, `spec-config.md` 6.2, FR-CFG-021, `spec-viability.md` 11 (X7): as EXP's owed table lists; the first is needed before Phase 7's Tally test (plan 18 owns it).
5. `rules/testing.md` 7.1.1 (the golden directories): covered by plan 18 Phase 1 (edit 8, a second edit to 7.1.1 after plan 5 E10, with its own fresh tag; it allows `golden/experiment/`). `rules/architecture.md` 2.6 and `AGENTS.md` layout already list `experiment/`, `internal/design`, `internal/reduce` (read), so the X4 and X5 rows owed by EXP are done except `testing.md` 9.1, which lists `experiment/` at 90% (read) and needs nothing.
5a. `rules/architecture.md` 9.6: the bounds of EXP 13.2 stay in `experiment/bounds.go` (D617); the wording is plan 23's 9.6 row, one fresh tag. Nothing for this plan to edit.
6. `doc.go` glossary, before any code uses them (architecture.md 3.2): experiment, factor, level, change, design (grid, one at a time, spread), point, replica, run key, unit (of work), plan (of an experiment), record (of a run), result, transient, final reading, series, measure, criterion, pass share, variant (of calls), continuation, pairing gain, effect range, elasticity. Not added: aligned, stream fingerprint (D602). `branch`, `salt` and `reseed` are plan 2's. The `window` entry is the existing one, extended with EXP's sense (OBS, IND already have senses); it is not a new term (specs/README.md glossary). The existing `Reduction` entry (the fall of an agent's weight, spawner rung 3) is extended, not duplicated, with EXP's sense (a summary of one reading over the runs of a group: planned, used, failed, min, max, mean, quantiles); `TestGlossary_Vocabulary` fails on a duplicate term.
7. `doc/roadmap-v1.md` row `experiment/`: done at the exit gate (supervisor).

Exit check: edits 1, 2 and 6 are applied (the others are rulings or later); `TestGlossary_Vocabulary` still passes.

### Phase 2: `internal/reduce`

Goal: D613. Files: `internal/reduce/{int128,round,moments,quantile,signtail}.go` and their tests. Tests (unit, all run under `-short`):
- `TestRDiv_HalfEven`: `rdiv(-5,2) = -2`, `rdiv(-3,2) = -2`, `rdiv(25,10) = 2`, `rdiv(35,10) = 4`, 128-bit operands, a zero divisor returns `ok` false; 20 000 cases against `math/big` in the test (the test may import it: the ban is on non-test files). Requirement: FR-EXP-040.
- `TestIsqrtNearest`: `s*s`, `s*s+s`, `s*s+s+1` for s up to 2^63 - 1, `2^126`; against `big.Int.Sqrt`. Requirement: FR-EXP-040, FR-EXP-045.
- `TestQuantile_NearestRank`: the ten readings of `reduce.out` (10 100 to 12 100 give 10 100, 10 300, 10 500, 10 900, 11 800 at the five quantiles), two readings (rank 1 up to 5 000, rank 2 above), one reading, unsorted input is the caller's duty. Requirement: FR-EXP-042.
- `TestMoments_MeanAndStd`: mean 10 760 and std 674 for the ten readings; mean 10 650 and std 354 for 10 400, 10 900 (EXP 19.9); `Used` below 2 has no std; the exception `Used*SS - S^2` is exact at `Used` 10^6 and readings 2^40. Requirement: FR-EXP-040, FR-EXP-042, FR-EXP-047.
- `TestSignTail_ExactAgainstFractions`: every `n` from 1 to 62 and every `m`, against `big.Rat`; `n` 63 returns `ok` false; the worked 6 wins, 1 loss gives 125 000. Requirement: FR-EXP-044.
- `TestInt128_Checked`: `Mul` overflow at 2^127, `Add` carry across the words, `Cmp`, negation of the minimum. Requirement: FR-EXP-040. Guard: a wrapped sum looks like a small one.
- `FuzzRDiv`, `FuzzIsqrtNearest`: no panic, `|n - q*d| <= |d|/2`, `r*r` nearest. Requirement: NFR-EXP-005.
Exit check: coverage of `internal/reduce` at 90% or more; `lizard` clean; `go vet`.

### Phase 3: `internal/design`

Goal: D614. Files: `internal/design/{grid,spread,seed,overlay,count}.go` and tests. Tests (unit, `-short`):
- `TestGrid_IndexAndInverse`: 12 points of EXP 19.1 in `worked.out` order, `(8 000, 400)` is index 5, the inverse for every index, last factor fastest, a factor-free design has one point. Requirement: FR-EXP-005.
- `TestOneAtATime_Order`: the 6 points of 19.2, count `1 + sum(L-1)`. Requirement: FR-EXP-005.
- `TestSpread_IsAPermutation`: every `N` from 1 to 300 and 511, 512, 513, 1 000, 4 096, 5 000, for five axes and three salts; each cell used exactly once. Requirement: FR-EXP-007.
- `TestSpread_WorkedPoints`: the eight points of 19.3 and `worked.out` section C (cells and values), salt 0. Requirement: FR-EXP-008.
- `TestSpread_ValueRules`: a range narrower than `N` repeats values; an explicit factor uses `floor((2*cell+1)*L/(2*N))`; `Max - Min + 1` near 2^64 does not wrap. Requirement: FR-EXP-008.
- `TestSpread_Stateless`: the point computed alone equals the point in the full design; another `DesignSalt` gives another batch; no call reads any global (the package has no variable). Requirement: FR-EXP-009.
- `TestRangeValue_Grid`: end points exact, half to even at ties. Requirement: FR-EXP-012.
- `TestSeed_SameAndPerPoint`: the seeds of 19.5; modulo 2^64 at `Seed0` `1<<64 - 1`. Requirement: FR-EXP-010.
- `TestOverlay_Bytes`: `[world]\ncost_floor = 8000\n`, a path of 3 segments, a negative value, zero, `math.MinInt64`; the bytes are fixed. Requirement: FR-EXP-003.
- `TestValidPath_Syntax`: 2 to 8 segments of `[a-z][a-z0-9_]*`, 64 bytes each; empty segment, uppercase, a leading digit, nine segments. Requirement: FR-EXP-004.
- `TestCheckedCount_Overflow`: products that wrap return `ok` false. Requirement: FR-EXP-064.
- `FuzzOverlay` (any path and value: no panic, a path that passes `ValidPath` gives bytes `config.Parse` accepts or rejects but never both outcomes, tested in the experiment package's copy, Phase 5), `FuzzPermutation` (in range; a bijection for `N` up to 200). Requirement: NFR-EXP-005. Corpora in `internal/design/testdata/fuzz/`.
Exit check: coverage at 90% or more; the table values equal the files of `doc/specs/models/exp/` (read, never run on the host: Python runs in Docker only).

### Phase 4: the types, `Validate`, `Plan`

Goal: D600, D601, D603, D604 (count rule), D617. Files: `experiment/{experiment,bounds,errors,validate,plan}.go`; `import_guard_test.go` (the case of D603). Tests:
- `TestValidate_EachRule` (one failing `Spec` for each row of EXP 3.6; `errors.Is` the sentinel; the Go path in the message). Requirement: FR-EXP-004.
- `TestValidate_AtMost100Errors` (150 faults give 100 and one `ErrTooManyErrors` that says 50). Requirement: FR-EXP-004.
- `TestValidate_ZeroValueInvalid`, `TestValidate_DocumentCount` (D604, the 16 cap; the exact count at the boundary; the rule is owed to the spec). Requirement: FR-EXP-004.
- `TestValidate_RunsNothing` (an `Options` and a `Prepare` that fail the test if called; no document parsed). Requirement: FR-EXP-004.
- `TestPlan_PointMajorOrder`, `TestPlan_BranchKeysByVariantThenContinuation`, `TestPlan_FactorFreeOneUnit`. Requirement: FR-EXP-005.
- `TestPlan_OversizeBeforeAnything` (a design whose product overflows `int`; `MaxRuns`, `MaxCells` with the series). Requirement: FR-EXP-064.
- `TestPackage_PurityAST` (go/parser over `experiment/*.go` non-test and `internal/design`, `internal/reduce`: no `go` statement, no `float32`/`float64`, no float literal, no `time.Now`, no `math/rand`, no `os` write, no package-level `var` except sentinels and the quantile table). Requirement: NFR-EXP-002. Guard: the existing guards cover imports and `time.Now` but not goroutines or floats.
- `TestImportGuard_Experiment` (extends `TestImportGuard_Detects`: `experiment/x.go` importing `indicator` or `strategy` is a problem; `config` and `internal/design` are not). Requirement: NFR-EXP-002. Guard: architecture.md 2.7.
Exit check: `go vet`, `staticcheck`, coverage of what exists, `TestImports_StdlibAndLayers` and `TestNoBannedUses_InNonTestFiles` pass on the new files.

### Phase 5: resolving and running one key (no branch)

Goal: D604 to D608, D610, D616. Files: `experiment/{resolve,run,segment,classify,record}.go`. Packs are read from `../testdata/chains` with `os.ReadFile` of `base.toml`, `one.toml`, `warm-one.toml` (the layers `packOne`, `support_test.go:34`; the root's helper is in another package). Tests (`experiment_test`, integration, `testing.Short()` skip, at most 40 ticks and 3 seeds; each names the seed and point in its message, testing.md 6.2):
- `TestRunUnit_PureFunction` (twice: equal records, equal `Render`; no clock). Requirement: FR-EXP-001, NFR-EXP-001.
- `TestResolve_LoaderRejections` (`world.cost_flor` is `unresolved`, a float and a scalar over a table too; `belief_step` 9 999 is `invalid_config`; the other points run). Requirement: FR-EXP-002, FR-EXP-050.
- `TestResolve_PrepareAfterOverlaysBeforeValidate` (a recording `Prepare` sees the point's value; its error is `invalid_config`). Requirement: FR-EXP-013.
- `TestRun_StepsMergedByTick` (script, level, then variant at equal ticks; a recording `Do` sees the tick; never inside a step). Requirement: FR-EXP-014.
- `TestRun_StepErrorIsScript` (the earlier calls stay applied; the world is healthy; the digest is zero; later keys run). Requirement: FR-EXP-014, FR-EXP-050.
- `TestRun_WindowsAndFinalReading` (`Every` 0, 30 with a short last window, `Transient` 10; ends `[10,40)`, span `[10,100)`). Requirement: FR-EXP-016.
- `TestRun_SpanCalls` (D607: counts of `Begin` and `Fn` and their `Span` values; `Every` 0 gives one of each with `Span` true; `Every` 30; a branch). Requirement: FR-EXP-016.
- `TestRun_BoundaryOrderCloseStepsOpen` (a step that adds a pending total is in the opening snapshot of the next window and not in the closing one). Requirement: FR-EXP-019.
- `TestRun_CriterionOnFinalReading` (`AtMost`, `AtLeast`, `Between`; a failed or undefined run does not pass; `Pass` in criterion order). Requirement: FR-EXP-015.
- `TestRun_NoObserverWithoutAnObserverMeasure` (`testing.AllocsPerRun`: snapshot-only is below one with a recording observer; both end at one digest). Requirement: FR-EXP-017, FR-EXP-065.
- `TestRun_HarnessSeedWins` (`Options` returns `WithSeed(99)`; the digest equals that of seed 1). Requirement: FR-EXP-018.
- `TestRun_ReplayIsOneWithSeedCall` (the digest of a sweep run equals `NewWorld(Parse(base+overlays), WithSeed(seed))`, `Run`, `Checkpoint`; `RunUnit` alone equals the same; two experiments that differ in `Seed0` share nothing). Requirement: FR-EXP-010, FR-EXP-011.
- `TestRun_FailureCodes` (a faulty `Clearing` fake gives `invalid_trade` with the tick; `WithStrategy(name, nil)` gives `world`; a `Stateful` extension whose `SaveState` errors gives `checkpoint`; `unresolved`, `invalid_config`, `script`; the sweep goes on; the partial series is kept with `Keep.Series`; a failed run has no digest and no reading). Requirement: FR-EXP-050, FR-EXP-051.
- `TestClassify_Table` (unit: every row of EXP 9.1, `audit` at `NewWorld` has tick 0, a restore error that wraps `ErrAuditFailed` is `world`, `ErrCheckpointConfig` is `world`). Requirement: FR-EXP-050.
- `TestRun_MeasureErrorAndRange` (an error leaves the reading undefined, `Detail` holds the first, the run goes on; 2^40 + 1 is undefined with `ErrRange`). Requirement: FR-EXP-053, FR-EXP-047.
- `TestRun_CancelDiscardsTheRunInFlight` (cancel from a `Fn`; no record for the unit; not in `Failed`), `TestRun_PriorCompletesByteForByte` (`Render` of cancel-then-`WithPrior` equals the uninterrupted run). Requirement: FR-EXP-020, FR-EXP-052.
- `TestRun_OptionsOncePerWorld` (a counter keyed by `Key`; an `Options` that returns a fresh stateful extension per call is never seen twice). Requirement: FR-EXP-024. N-129.
- `TestRunUnit_ParallelEqualsRun` (eight goroutines in the test over the units of `Plan`, `Assemble`; under `-race`), `TestRun_DoesNotWriteTheSpec` (deep copy before and after). Requirement: FR-EXP-021, FR-EXP-023.
- `TestExecuteUnit_SkipsBaseAndPrepare`. Requirement: FR-EXP-001.
- `TestCheck_ResolvesWithoutTicks` (no `Options` call, no world; one failing point is `unresolved`), `TestCheck_RangedNarrowerThanDesign`. Requirement: FR-EXP-070, FR-EXP-071.
- `TestRun_HarnessWorkDoesNotGrowWithAgentsOrTicks` (counted calls of `Begin`, `Fn`, `Options`, `Observer`: equal for `Config.Scale` 1 and 4 (set by an overlay on the existing field) and for 20 and 40 ticks at fixed windows; a counted operation, testing.md 8.2), `BenchmarkRunUnit_HarnessShare` (`b.ReportAllocs()`, no tolerance: N-015). Requirement: NFR-EXP-003.
- `FuzzOverlayThroughTheLoader` (any path and value: never panics; `ValidPath` true means the loader accepts or rejects the document, never both). Requirement: NFR-EXP-005.
Exit check: the package passes under `go test -short -race ./experiment/...` and without `-short`.

### Phase 6: branching

Goal: D609. Files: `experiment/branch.go`. Tests (integration, as Phase 5; `SetYieldShocks` is plan 17's):
- `TestBranch_BurnInOnceRestoreEach` (a counter: `Options` is called `1 + V*C` times with the keys of D610). Requirement: FR-EXP-030, FR-EXP-024.
- `TestBranch_NoStepsNoReseedEqualsUninterrupted` (the branch digest equals the digest of the same key run from tick 0; also with steps below and at `At` in both runs). Requirement: FR-EXP-031, FR-EXP-014. Cites FR-CHK-005, FR-CHK-050, FR-WLD-063.
- `TestBranch_ReseedByContinuation` (0 and 1 pass no `WithReseed`; with 3 the salts are `Salt0..Salt0+2`, equal across variants; the record's `Salt`/`HasSalt`; digests differ across continuations and equal across none-variants of one continuation). Requirement: FR-EXP-032.
- `TestBranch_BurnInFailureInherited` (a faulty `Clearing` before `At`: every key of the unit has the burn-in's status and tick, no branch ran). Requirement: FR-EXP-033.
- `TestBranch_VariantStepBelowAtIsErrBadBranch`. Requirement: FR-EXP-034.
- `TestBranch_OptionsWithSeedGivesWorld` (every branch is `world`, `ErrOptionContext` in `Detail`). Requirement: FR-EXP-024.
- `TestBranch_ConfigCannotChange` (an `Options` with another `WithTargetSpread` gives `world`, `ErrCheckpointConfig`; a factor never branches). Requirement: FR-EXP-030.
- `TestBranch_ShockVariantDiffersFromNone` (`SetYieldShocks` at `At`: its digest differs, the none variant equals the uninterrupted run). Requirement: FR-EXP-030, FR-EXP-031.
- `TestBranch_SeriesAndFork` (the series holds the burn-in windows first; `Fork` is called for every branch including (0,0) with equal keys; a window open at `At` stays open; the sequence of hook calls shows one world at a time). Requirement: FR-EXP-016, FR-EXP-030, NFR-EXP-004.
`forkAtN` (plan 2) is a root-package test helper and cannot be imported here: this plan uses its pattern (equal bytes after N and M ticks) in the no-steps test and adds no state to a checkpoint (EXP 16).
Exit check: coverage of `branch.go` at 90%; the digests are equal across two runs of the whole package.

### Phase 7: results, statistics, `Render`, the golden and the `indicator` tests

Goal: D611, D612, D615, D618. Files: `experiment/{result,reduce,compare,sensitivity,render}.go`, `indicator/reduce_equivalence_test.go`, `indicator/experiment_measure_test.go`. Tests:
- `TestAssemble_OrderInsensitive`, `TestAssemble_UnknownDuplicateMissing`, `TestAssemble_SameAsAndDuplicatePoint`. Requirement: FR-EXP-022, FR-EXP-063, FR-EXP-071.
- `TestReduce_AgainstReduceOut` (the ten readings; the survivor and pass-share examples of 19.8: 8 000 and 7 000 bp, mean 10 778). Requirement: FR-EXP-040 to FR-EXP-043.
- `TestReduce_PlannedIdentity` (property over random failure and missing patterns), `FuzzReduce` (any readings: no panic, `Planned = Used + Failed + Undefined + Missing`). Requirement: FR-EXP-041, NFR-EXP-005.
- `TestCompare_Worked197` (Pairs 8, MeanDiff 4, MedianDiff 3, Wins 6, Losses 1, Ties 1, tail 125 000, RelativeBP 341, DiffOfMeans 4), `TestCompare_PerPointSeedsIsUnpaired`, `TestCompare_SignTailBeyond62`, `TestCompare_PairingGain` (identical settings: `Identical` true and `OK` false; a known sd gives the stated gain). Requirement: FR-EXP-044, FR-EXP-045.
- `TestMainEffects_ThreeByThree` (ranges 40 and 8, shares 8 333 and 1 667), `TestElasticity_Worked` (-5 250; `OK` false for `Y_0 <= 0`, `x_0` 0, `x_l = x_0`; `Clamped`). Requirement: FR-EXP-046.
- `TestRender_Worked199` (a hand-built `Result` renders the text of 19.9 exactly, without the invented hashes changing), `TestRender_DashRules`. Requirement: FR-EXP-060.
- `TestResult_PlainData` (reflection over `Result`: no map, pointer, func, chan, interface or float kind). Requirement: FR-EXP-061.
- `TestGolden_ExperimentOnPackOne` (D615; the digests in it are compared on two architectures by hand), `TestGolden_RenderTwiceEqual`. Requirement: NFR-EXP-001, FR-EXP-060.
- `TestRegistry_SweptPathsAreRows` (D618). Requirement: FR-EXP-072.
- The register scenarios of spec-experiments.md 17, as harness tests on toy processes (no `TestScenarioS<nn>_` name: S84 to S87 are not in plan 24's scan): `TestHarness_SpreadFallsAsRootR` (EXP-S1, S84; Requirement: FR-EXP-040, FR-EXP-042), `TestHarness_WinnersCurse` (EXP-S2, S85; FR-EXP-043), `TestHarness_TransientWindows` (EXP-S3, S86; FR-EXP-016), `TestHarness_SurvivorBias` (EXP-S4, S87; FR-EXP-041, FR-EXP-050).
- `TestRecord_WithinTheSizeFormula` (cells at most `measures * (windows + 1)`; nothing grows with ticks without `Keep.Series`). Requirement: NFR-EXP-004.
- In `indicator/` (package `indicator_test`): `TestReduceAgreesWithMeasure` (half-to-even division, nearest-rank quantile with weight-1 rows, `IsqrtNearest`, sums: equal on the same inputs, so the two implementations of one rule cannot drift). Requirement: FR-EXP-040, FR-EXP-042. `TestExperiment_TallyMeasureEqualsDirectRun` (a `Measure` whose `Observer` builds a `Tally`, `Begin` opens it and `Fn` closes it; its final reading equals that of a plain recorded run; a branch with `Fork` copying the tally equals the uninterrupted run). Requirement: FR-EXP-016, FR-EXP-030. EXP 3.4.0.
Exit check: coverage of `experiment/` at 90% or more; the golden is recorded once with `-update` after plan 8's pack and the PR explains it (testing.md 7.2).

### Phase 8: documentation and examples

Goal: NFR-EXP-006. Every exported identifier has a doc comment: new test `TestExported_HaveDocComments` (go/parser over the non-test files of `experiment/`; the root's `TestDocComments_Contracts` checks only a few fixed phrases and does not cover this). `package experiment` has a package comment and a runnable `Example` for `Plan`/`Run` (a two-point sweep of a tiny inline configuration, checked output), `Compare` and `Render`; the doc comment of `Options` states the contract of D610 and N-129. No new file under `doc/`.
Exit check: `go vet ./...`, `staticcheck ./...`, `TestExported_HaveDocComments` and `ExamplePlan`, `ExampleCompare`, `ExampleRender` pass.

### Phase 9: exit gate

`gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; R and F of plan 1 D229; `go mod tidy` with no diff; `lizard -l go -C 10 -a 8 -L 100 -w` on every new non-test file; coverage at least 90% for `experiment/`, `internal/design` and `internal/reduce`; a `gomu` pass on the two internal packages in Docker (D619); `TestHashConfig_ReferencePinned` and `testdata/golden/off/` unmoved; the registry and catalogue tests unchanged (EXP adds no row, no event). Report: `Testing` and `Observability` entries.

## Catalogue appends

| File | Appends | Position, id or tag | Spec line |
| --- | --- | --- | --- |
| `config.go`, `bounds.go`, `defaults.go`, `validate.go`, `hash.go` | none: an experiment is not configuration; its bounds are constants of `experiment/bounds.go` and no field enters the hash | none | EXP 13.1, 13.2 |
| `errors.go` (root) | none: the sentinels of EXP 14.1 are `experiment.ErrBadSchedule` and the rest; `ErrDuplicateID`, `ErrTooManyErrors`, `ErrCheckpointConfig`, `ErrOptionContext` are consumed | none | EXP 14.1 |
| `event.go`, `option.go`, `tick.go`, `snapshot.go`, `ledger.go` | none: no event, option, tick position, snapshot field or account | none | EXP 15.1 |
| checkpoint tags, log keys, metrics | none: "State contributed to a checkpoint: None"; no log key; no metric | none | EXP 16, 15.1 |
| `import_guard_test.go` | the case for files under `experiment/` (D603) | Phase 4 | architecture.md 2.7 |
| `doc.go` glossary | the terms of Phase 1 edit 6 | before code | architecture.md 3.2 |
| `testdata/golden/experiment/one.txt` | new, written with `-update` | Phase 7 | EXP 11.1 |

## Cross-plan contract

Consumes:
- spec-checkpoint.md passages of N-214 rewritten (doc/specs/spec-checkpoint.md) <- plan-2-1
- the race-aware gate command (doc/plans/plan-2-1.md) <- plan-2-1
- Checkpoint (checkpoint.go) <- plan-2-2
- (*World).Checkpoint (checkpoint.go) <- plan-2-2
- CheckpointInfo (checkpoint.go) <- plan-2-2
- RestoreWorld (checkpoint.go) <- plan-2-2
- WithReseed (option.go) <- plan-2-2
- ErrOptionContext (errors.go) <- plan-2-2
- ErrCheckpointVersion, ErrCheckpointCorrupt, ErrCheckpointConfig, ErrCheckpointExtension (errors.go) <- plan-2-2
- Stateful (checkpoint.go) <- plan-2-2 (plan 2 lists it for plan 17; the tests of N-129 and the `checkpoint` status use it)
- forkAtN, forkCase (checkpoint_fork_test.go) <- plan-2-2 (the pattern only: a root test helper cannot be imported)
- (*World).SetYieldShocks, YieldShock (integration.go) <- plan-2-17
- verdict.Check, verdict.Verdict (internal/verdict) <- plan-2-19 (the response of the FR-STY-090 test)
- Tally, NewTally, Window, Open, Close and Observe (indicator/tally.go, window.go) <- plan-2-18
- RoundDiv, Quantile and IsqrtNearest (internal/measure) <- plan-2-18
- the packs `base.toml`, `one.toml`, `warm-one.toml` as plan 8 leaves them (testdata/chains/) <- plan-2-8
- config.Parse, World.Run, World.Err, World.Snapshot, World.ConfigHash, WithSeed, WithObserver, WithAudit, ValidationError, ErrInvalidTrade, ErrAuditFailed, ErrDuplicateID, ErrTooManyErrors, MaxDocuments (config/config.go, world.go, snapshot.go, option.go, errors.go, bounds.go) <- Stage 1
- Finalise (internal/ordering), Mul (internal/checked) <- Stage 0
- importProblems, bannedUses, guardedFiles (import_guard_test.go, determinism_guard_test.go) <- Stage 1
- the scope table and the parameter-owner table (doc/specs/README.md) <- Stage 0

Plans 1, 8 and 19 are named beyond Needs (2, 17, 18): the gate command, the rebalanced packs, and the response of the FR-STY-090 test.

Provides:
- Spec, Factor, Level, Change, Step, Measure, Sample, Criterion, Branch, Variant, Keep, Design, Seeding, Op (experiment/experiment.go); MaxFactors and the other bounds of EXP 13.2 (experiment/bounds.go) -> plan-2-21
- Plan, RunUnit, ExecuteUnit, Run, Check, PointCheck, WithRecordFunc, WithPrior (experiment/plan.go, run.go, resolve.go) -> plan-2-21
- Record, Reading, Status and its constants (experiment/record.go), Assemble, Result, PointResult, VariantResult, Key, Unit, Warning (experiment/result.go, plan.go, classify.go) with the fields of EXP 3.1, 8.1 and 8.2 -> plan-2-21
- Effect, Sensitivity, Comparison with their fields (experiment/sensitivity.go, compare.go) -> (no later plan; reached through `Render`)
- Overlay(path string, value int64) ([]byte, error) and ValidPath (internal/design/overlay.go) -> plan-2-21 (the two-line document of EXP 3.3.4)
- Reduction, PassShare, Group, Compare, Comparison, MainEffects, Elasticity (experiment/reduce.go, compare.go, sensitivity.go) -> (no later plan; the consumer of `Result` reads them through `Render`)
- Render (experiment/render.go) -> plan-2-21
- Spec, Run, Render (experiment/) -> plan-2-24
- testdata/golden/experiment/one.txt (testdata/golden/experiment/) -> plan-2-24 (two architectures)

`internal/design` is consumed by `experiment/`, by the tests of `indicator/` and by plan 21's overlay writer (`Overlay`, `ValidPath`); `internal/reduce` by `experiment/` and the tests of `indicator/` only.

## Testing

Testing entry (testing.md 11), the requirements touched and the tests that cover each (full names in the phases):

| Requirement | Tests |
| --- | --- |
| FR-EXP-001, NFR-EXP-001 | `TestRunUnit_PureFunction`, `TestExecuteUnit_SkipsBaseAndPrepare`, `TestGolden_ExperimentOnPackOne` |
| FR-EXP-002, 003, 004, 005 | `TestResolve_LoaderRejections`, `TestOverlay_Bytes`, `TestValidate_*`, `TestValidPath_Syntax`, `TestGrid_IndexAndInverse`, `TestOneAtATime_Order`, `TestPlan_*` |
| FR-EXP-007 to 012 | `TestSpread_*`, `TestRangeValue_Grid`, `TestSeed_SameAndPerPoint`, `TestRun_ReplayIsOneWithSeedCall` |
| FR-EXP-013 to 019 | `TestResolve_PrepareAfterOverlaysBeforeValidate`, `TestRun_StepsMergedByTick`, `TestRun_CriterionOnFinalReading`, `TestRun_WindowsAndFinalReading`, `TestRun_NoObserverWithoutAnObserverMeasure`, `TestRun_HarnessSeedWins`, `TestRun_BoundaryOrderCloseStepsOpen` |
| FR-EXP-020 to 024 | `TestRun_CancelDiscardsTheRunInFlight`, `TestRun_PriorCompletesByteForByte`, `TestRun_DoesNotWriteTheSpec`, `TestAssemble_*`, `TestRunUnit_ParallelEqualsRun`, `TestRun_OptionsOncePerWorld` |
| FR-EXP-030 to 034 | `TestBranch_*` |
| FR-EXP-040 to 047 | `TestReduce_*`, `TestCompare_*`, `TestMainEffects_ThreeByThree`, `TestElasticity_Worked`, `Test{RDiv,IsqrtNearest,Quantile,Moments,SignTail}*`, `TestReduceAgreesWithMeasure` |
| FR-EXP-050 to 053 | `TestRun_FailureCodes`, `TestClassify_Table`, `TestRun_MeasureErrorAndRange`, `TestRun_CancelDiscardsTheRunInFlight` |
| FR-EXP-060, 061, 063 to 065 | `TestRender_*`, `TestResult_PlainData`, `TestAssemble_SameAsAndDuplicatePoint`, `TestPlan_OversizeBeforeAnything`, `TestRun_NoObserverWithoutAnObserverMeasure` |
| FR-EXP-070 to 072 | `TestCheck_*`, `TestRegistry_SweptPathsAreRows` |
| NFR-EXP-002 to 006 | `TestPackage_PurityAST`, `TestImportGuard_Experiment`, `TestRun_HarnessWorkDoesNotGrowWithAgentsOrTicks`, `TestRecord_WithinTheSizeFormula`, the fuzz targets, the coverage gate and Phase 8 |
| EXP-S1 to S4 (S84 to S87; no `TestScenarioS<nn>_` name) | `TestHarness_SpreadFallsAsRootR`, `TestHarness_WinnersCurse`, `TestHarness_TransientWindows`, `TestHarness_SurvivorBias` |
| FR-STY-090 | `TestSweep_StylisedVerdictAsResponse` (external test package of `indicator/`: a one-at-a-time sweep, default and two neighbours, of a knob with a plan 19 verdict as the response; Requirement: FR-STY-090) |
| FR-EXP-062 | retired (D602): no test |

Golden files: `testdata/golden/off/` and the pinned hash do not move (nothing in the root changes; proven by `TestHashConfig_ReferencePinned` and the off golden tests at the gate); no module-alone golden (EXP is not a module); the harness golden `testdata/golden/experiment/one.txt` (D615). Fork-at-N: not applicable, EXP contributes no state (EXP 16); the equal-digest branch test (FR-EXP-031) is its analogue. Registry and catalogue rows: none (no config field, no event), so no `notBuilt` entry is deleted. Fuzz: `FuzzRDiv`, `FuzzIsqrtNearest`, `FuzzOverlay`, `FuzzPermutation`, `FuzzOverlayThroughTheLoader`, `FuzzReduce`, each with a `Fix:` comment for any committed crash (testing.md 7.5). Invariants over seeds: a test that runs many seeds names the failing seed and point (testing.md 6.2). Not covered, by design: no band of the economy is asserted (N-027).

## Observability

Observability entry (observability-and-logging.md 9): **None, because** the harness adds no failure path inside a world (every failure is a status of a record and a returned error of the sweep, never a log line: architecture.md 8.3), no state transition, no decision point of a strategy, clearing or spawner, and no loop in a tick; its loops are over runs and are not logged. It adds no event, no `Warn`, no log key and no metric (EXP 15.1). It uses `Observer` as a consumer: it registers the observers of the measures that have one, and registers none when no measure has (FR-EXP-017), so a snapshot-only sweep constructs no event (observability-and-logging.md 4.5). Non-interference: a run with the measures' observers and a run with none end at one digest (FR-EXP-065; `TestRun_NoObserverWithoutAnObserverMeasure`).

## Risks and open items

1. **`MaxDocuments` 16 against one document per change (D604).** `config.Parse` takes at most 16 documents (`bounds.go:123`) and EXP 3.3.2 makes one for each `Change` and each `Docs` of every level, so a base of 3 documents leaves room for 13 changes, and `MaxFactors` 16 and CLB's screen of every knob (one factor per knob, level 0 included) cannot be run. Owners: EXP 13.2/3.3.4 and CFG 4.1 (N-015). Options: raise `MaxDocuments`, let a level's changes share one document (FR-EXP-003 then changes), or lower `MaxFactors`. Interim: `Validate` refuses with `ErrOversize`. Plan 21 must not assume the screen of EXP 12.4.2 runs at 16 knobs.
2. **`Options` and `Fork` with equal keys (D609).** EXP 5.5.2 gives the burn-in the key of branch (0,0), so `Observer(key)` is called twice with one key for two worlds, in sequence. Proposed text: `Fork(k, k)` for (0,0) means keep, and a consumer keying state by `Key` must treat a second `Observer(k)` as a new world. Owner: EXP 5.5.2, 6.1.1.
3. **`Pair` after ADR 0008 (D602).** ADR 0008 decision 6 says "`Pair` keeps measuring the variance reduction", which this plan reads as the always-on `PairingGain` of `Compare`. If the owner wants a flag, a divergence report (the first window at which two paired series differ) would be new text; none is specified. Removing `Spec.Pair` later is a breaking change, so the owner confirms this reading before `Spec` ships. Owner: EXP 7, ADR 0008.
4. **`Series` with `Every` 0 (D607)**, **`PointResult.Status` (D611)** and **the golden path (D615)** are clarifications owed to EXP 3.4.3, 8.2.1 and 11.1.
5. **`stale_warm_start` (D616).** N-134 says VIA 11 publishes the path list. EXP 14.4 still says "not specified here". Owner: EXP, then a plan after 21; a path list copied into the code would drift, so the rule should read CLB's `in`/`tbl` flags as data (plan 21).
6. **FR-EXP-072 across packages (D618).** The shipped experiments of STY (plan 19, in `indicator/` tests) and CLB (plan 21) need the same table check; test code cannot be shared across packages. Owner: plans 19 and 21; the cheapest answer is a short copy of the helper, or a scope table kept as data by CLB's registry.
7. **Lines other plans address to plan 20 that this plan does not consume:** plan 10 (`Modules.Subsistence` types), plan 16 (`External` types, `siteOutsideWalk`): `experiment/` builds no `Config` in Go (EXP 3.3.5) and takes no site; plan 18 (`Classes`, `Stat`, `Stats`, `Growth`, `Change`, `Point`, `YearWindow`, `SeasonWindows`): the Tally test reads one figure; plan 19 (`SR1` to `SR3`, `RenderVerdict`: seed rules and rendering are STY's, the harness reads no measure; only `verdict.Check` and `verdict.Verdict` are consumed, by the FR-STY-090 test) and its mention of a `ReplicaRun` (no such identifier exists in EXP: the equivalent is `Run` with no factor, EXP 4.1.3 and 12.2).
8. **A `Digest()` accessor.** D608 copies the encoding once per run. If `Info()` could carry the trailer (or `Checkpoint` a `Digest()`), the copy goes. Owner: CHK, plan 2; no change is requested before a measurement (NFR-EXP-003).
9. **The two-architecture run** of the golden and the digests is manual until plan 1's CI decision (plan-2-1 D236); plan 24 runs it.
10. **Notes.** N-015: leave open; learned that `MaxDocuments` binds (item 1). N-016: apply; no field, no row, FR-EXP-072. N-017: apply; the glossary terms (Phase 1), no setting vocabulary in the package. N-027: standing; no band asserted. N-103: standing; this plan edits only its own file. N-124: apply in part (D602); stays open for CLB 4.2.1 (plan 21). N-126, N-128, N-163: not applicable (no event, no saved state, the word "treasury" unused). N-127: leave open; a snapshot-only sweep registers no observer (FR-EXP-017). N-129: apply in code and docs (D610); propose closing it for EXP when the code ships, leave INT/CHK. N-134: leave open (item 5). N-162, N-179, N-207: leave open; under ADR 0008 only an ID shift, state divergence or a change of scale breaks a pair, and the harness reports no alignment. N-203: not this plan's (STY). N-206: consistent, `experiment/` imports no `indicator`. N-211: not this plan's. N-214: apply for EXP (Phase 1, D602); stays open for CLB, STB, TEC. N-217: nothing in the core moves. N-220: leave open for CLB; `design.ValidPath` is the path syntax the registry may reuse.
11. **Note proposals (text only; the supervisor edits `doc/notes.md`):**
    - Proposed N-NEW (areas EXP, CFG, CLB; CLB is kept): "A point that sets more than `16 - len(Base)` documents is refused by the loader (`MaxDocuments`): EXP 3.3.2 makes one document for each change, so a screen of every knob cannot run until the bound, the document rule or `MaxFactors` changes. Raised 2026-10-22, plan-2-20 D604."
    - Update to N-214: "EXP 7.3 and FR-EXP-062 applied by plan 2-20 D602; open for CLB 4.2.1, STB and TEC."
    - Update to N-129: "built in `experiment/` (plan-2-20 D610): `Options(key)` is called once for each world built; the library still cannot see a shared instance."
