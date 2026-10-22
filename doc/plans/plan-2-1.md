# Plan: Stage 2, row 1: work before any row

**Status: written (2026-10-22), not audited. Contract: [plan-2-0.md](plan-2-0.md).**
This plan builds no module code. It makes the ground true before `checkpoint.go` is written: the specs of CHK and WLD say what ADR 0008 decided, the full `-race` run finishes, the two ADRs the hash and the checkpoint rest on are accepted, the plans have a home and a life span, the order of Stage 2 is on record, CI has a decision, and the Codacy limits are a gate.

Rule tags read (the files show the same tags as `plan-2-0.md`): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1.

Specs implemented: none new. It corrects `spec-checkpoint.md` (CHK) and `spec-world.md` (WLD) to ADR 0008 and `architecture.md` 7.3; it serves NFR-WLD-004 (two architectures) and the gates of `testing.md` 9.1 and 10.5. Requirement IDs touched: FR-CHK-006, FR-CHK-025, FR-CHK-051, FR-WLD-100 (edited, not only cited), FR-WLD-101, NFR-WLD-004 (all exist in the specs).

Notes read (index, then every open entry tagged ALL, CHK or WLD), with the disposition of each:

- Apply: N-214 (Phase 1; it stays open for EXP, CLB, STB, TEC, which are not this plan's), N-243 (Phase 6), N-019 (its Stage 0 updates on ADR 0002 and CI; Phases 3 and 5), N-124 (the paired-comparison wording of CHK 13 and WLD-A5 in Phase 1; open for EXP, STB, CLB).
- Standing rules, left open (this plan adds no field, event, `Warn`, site, name or module): N-016, N-017, N-126, N-163, N-217 (it moves no golden and no hash), N-027, N-103 (this plan is one of a parallel run under plan-2-0), N-244.
- Left open for plan 2, the CHK encoding (this plan only corrects text): N-125, N-128, N-129, N-204, N-215, N-225, N-237, N-247, N-254, N-256, N-265, N-288, N-294 (the digest enters the off-only golden once), N-309.
- Left open, with N-202: ADR 0008 answered it and CHK 6.8.2 states it; it closes when TEC 4.4.1 and 4.4.6 are rewritten (TEC).
- Left open for the owner named in the note, not read further: N-144 (EMP), N-216, N-226, N-230, N-276, N-310 (the world and module rows), N-312 and N-314 (plan 8).

Decision block: D220 to D239.

## Scope

In:

- Spec text: the passages marked "stale under ADR 0008" in `spec-checkpoint.md` and `spec-world.md`, plus the worked example and models of CHK (Phase 1). The supervisor applies them (plan-2-0: a plan edits only its own file).
- Test code (not module code): `scenario_helpers_test.go`, `scenario_s01_test.go`, `scenario_s04_test.go`, `scenario_s07_test.go`, `scenario_s16_test.go`, `scenario_s19_test.go`, `determinism_test.go`, `invariants_test.go` (Phase 2). `raceEnabled` already exists in `export_phase8_race_test.go` and `export_phase8_norace_test.go` (package `economysim_test`, build tags `race` and `!race`) and is read by `allocation_test.go`.
- Text for the owner: the two ADR acceptances (Phase 3), the CI decision (Phase 5).
- Text for the supervisor: `roadmap-v1.md`, `vision.md`, `spec-roadmap.md`, `documentation_index.md`, `architecture.md` 2.6, `AGENTS.md`, `testing.md` 9.1 and 10.5 (Phases 3, 4, 6).
- Lizard and markdownlint as Codacy runs them (Phase 6).

Out: any module file; the CHK encoding (plan 2); a change to the STB, EXP, CLB or TEC passages that cite N-214 (their owners: plan 20 for EXP, plan 21 for CLB; STB and TEC are later-stage specs and N-214 stays open for them); the rebalance of pack `one` (plan 8); the run on two architectures itself (plan 24); parallel stepping.

## Decisions

- **D220.** The plan writes no module code and moves no golden and no pinned hash. Source: plan-2-0; N-217.
- **D221.** Phase order is spec, tests, ADRs, plans and order record, CI, limits, exit gate. Phase 1 comes first because `checkpoint.go` must not start while its spec contradicts the rules (plan-2-0 "Phase 1 lists every edit owed"; `AGENTS.md` "Before writing or changing a spec").
- **D222.** A stale passage is rewritten, not annotated: the "stale under ADR 0008" sentence and the old text both go, and the passage states the rule of `spec-world.md` 12 and 12.3, ADR 0008 and `architecture.md` 7.3. The authority is those three; where `spec-checkpoint.md` and they differ, CHK conforms. Source: N-214 ("apply the ADR in the next change that touches each"); `plan-2-0` ("the specs decide").
- **D223.** The World section (CHK 4.2 tag 1) holds the next agent ID and the construction flag only. The construction bucket and the pending totals that calls between ticks added are the ledger's and are written in tag 2 (CHK 4.2 and 4.4; plan 2 D244 and D245, which own the CHK layout). The tick and the effective seed are in the header (CHK 5.4.3 and 5.4). FR-WLD-100 and the `spec-world.md` 12 table name the world's own contribution as the effective seed (header), the next agent ID and the construction flag, and name the bucket and the pending totals as the ledger's contribution (Phase 1). Source: CHK 4.2; plan 2 D244. The order of fields inside the section is plan 2's.
- **D224.** `Stream` (the 16 bytes after the `"pcg:"` prefix, CHK 3.1.2) is removed from the API text, from FR-CHK-006 and from the vectors: a checkpoint has no stream state. `Info` reports the format, the configuration hash, the tick, the seed and the length. Source: ADR 0008; `spec-world.md` 12.
- **D225.** The header's seed is the effective seed and is read back (CHK 5.4.1, 6.8.3, `WithSeed` refusal row of 6.4, `WithReseed` row). Source: `spec-world.md` 12.3.
- **D226.** The race-aware reduction is by seed count and, for the determinism test, by case subset; ticks, packs, floors and every assertion stay. A `raceEnabled`-keyed function picks the count (Phase 2). Rejected: shortening ticks (S04's shock window and S19's 1095 ticks are the scenario) and skipping the tests under `-race` (the race detector would then never see the harness's parallel subtests, `scenSem` and `scenReport`, nor the allocation test's `-race -covermode=atomic` condition).
- **D227.** What the reduction keeps: every scenario (S01, S04, S07, S16, S19), every pack and floor, every tick count, every band and exact check on the seeds it runs, the allocation baseline (`TestStep_AllocationsStayAtTheBaseline` needs race and atomic), coverage of every statement the full set covers (a statement is covered by seed 1 or not at all; the list is checked in Phase 2). What it drops under `-race` only: seeds 3 to 10 of each scenario, the seeds of `TestInvariants_Packs` beyond `raceSeeds` (pack `one` 3 to 16, the others 3 to 4) and the redundant chunkings of `TestDeterminism_SameBytes`. Those run in the second run (D229).
- **D228.** `-update` is refused under `-race`: the recorded `testdata/chains/expected/one.scenarios.txt` and its titles say "seeds 1-10" and must not be written from a reduced run. The title of a report carries the seed count that ran. Source: `scenReport.flush` (`scenario_helpers_test.go`), D181 of the Stage 1 plan 1-12.
- **D229.** The Stage 2 gate is two runs. **R**: `go test -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out -timeout 60m ./...` (reduced under `-race`; the coverage thresholds of `testing.md` 9.1 are read from it). **F**: `go test -timeout 60m -run 'TestScenario|TestDeterminism|TestGolden|TestInvariants|TestCheckpoint|TestFork|TestConfigOnly|TestFixture|TestPacks|TestCensus|TestExpected|TestStylised|TestCost|TestStable' ./...` without `-race` (all ten seeds, all cases; the wide pattern makes a test that a later plan skips under `raceEnabled` run here). **Rule:** a root-package test that is skipped under `raceEnabled` MUST have a name that starts with one of these prefixes (`gateFPrefixes`, `scenario_helpers_test.go`), or it runs in neither gate; `TestRaceSkips_HaveAnFPrefix` parses the root `*_test.go` files and fails on a `Test*` function that calls `t.Skip*` under `raceEnabled` with another name. Both pass before a row is done. Until Phase 2 is done a row uses `go test -short -race ./...` and F alone, as `plan-2-0` allows.
- **D230.** The budget of R is 30 minutes wall on the reference hardware (the machine that timed out at 2 hours); the number of seeds under `-race` is set by measurement in Phase 2: start at 2, lower to 1 if the budget fails, raise only if it holds with margin. The default `go test` timeout is 10 minutes and the root package will exceed it, so the rule's command gains `-timeout 60m` (Phase 6 edit to `testing.md` 9.1).
- **D231.** If the budget fails at one seed, the fallback is to split R into `go test -race ./...` (reduced) and `go test -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...` without `-race` (coverage only, full seeds), which needs `testing.md` 9.1 reworded and the allocation test's skip condition reconsidered. The owner decides; this plan does not.
- **D232.** A plan cannot accept an ADR. The acceptances of ADR 0002 and 0009 are owner actions with the exact text in Phase 3. ADR 0002 is accepted after Phase 1 is applied, so that it is not accepted over specs that contradict it. ADR 0009 meets its own condition (Open point 3: "until the first code that imports the library"): `config/guard.go` imports `pelletier/go-toml/v2/unstable` and `config/fuzz_test.go` holds `FuzzLoad`.
- **D233.** Plans live in `doc/plans/`. During a stage they are working documents, named `plan-<stage>-<n>.md` (Stage 1 used `plan-phase-<stage>-<n>.md`; the Stage 2 name is fixed by the contract). After the stage is signed off they are deleted in one commit, as Stage 1's were (`2abbe62`), and the roadmap records that commit as their archive. A plan that is deleted leaves no link in a live document.
- **D234.** The live documents still link the deleted Stage 1 plans (`roadmap-v1.md` lines 28, 86 and the Stage 1 table rows; `documentation_index.md` (the Stage 1 plan rows, already replaced by the plan-2-N rows); `vision.md` line 108; `spec-roadmap.md`, one line). They are fixed in Phase 4: the link becomes the plan's name and "git history, `2abbe62^`". History text in `notes.md` (dated updates), three specs and ADR 0009 that names a deleted plan is not a link and is left as is: a note is removed when settled, and the dated lines are a record.
- **D235.** The Stage 2 order is the order of `plan-2-0.md`. DEM is the first module (N-314 pull-forward), the reference-pack rebalance (plan 8) follows the pool part of EST (plan 7), and plans 1 and 8 are not rows of the roadmap's Stage 2 table today. Source: owner's delegation of 2026-10-22, recorded in `plan-2-0.md`.
- **D236.** No CI exists (`.github/` is absent; the origin is GitHub). `testing.md` 9.1 says CI **MUST** run the coverage command and 7.1 and `architecture.md` 7.4 say two architectures; NFR-WLD-004 and `vision.md` 5 item 2 make the two architectures a Stage 2 exit criterion. So CI is needed for the exit. The decision is the owner's (Stage 0 plan O3: "needs an owner and a roadmap row"); Phase 5 gives the recommended content and the by-hand fallback.
- **D237.** The two-architecture requirement is met by running, on amd64 and on arm64, the tests that fix bytes: `TestGolden_*`, `TestDeterminism_*`, the hash pin test, the chain tests of `config_chain_test.go`, and `TestInvariants_Packs` (the rows `one`, `second` and `three`, and later the rows of other plans: it asserts logic, not bytes, but it is one of the four test kinds of `vision.md` 5 item 2, so the exit needs it on both architectures). They run without `-race` on arm64 if the runner has no race support. A golden that differs between architectures is a defect to fix in the code (`architecture.md` 3.5, the fused multiply-add rule), not a second golden.
- **D238.** Codacy's gate is zero new issues (N-243). Lizard: `lizard -l go -C 10 -a 8 -L 100 -w <files>` on every new or changed non-test file; test files are excluded by `.codacy.yml`. The existing `internal/checked.muldiv` (CCN 13) is split by the first change that edits it, as N-243 says; this plan records the baseline instead (Phase 6). Markdownlint uses `.markdownlint.json` (MD010 code blocks off, MD013 off, MD037 off); tables use the spaced separator row `| --- |` (MD060).
- **D239.** The gates of `plan-2-0` are amended, in the one place where they are stated, by the supervisor: the full-run line becomes R and F of D229, and the lint line gains markdownlint. Source: D229, D238.

## Phases

### Phase 1: the stale passages (CHK and WLD)

Goal: no passage of `spec-checkpoint.md` or `spec-world.md` contradicts ADR 0008 (D222). Files: those two specs, `doc/specs/models/chk/` (README.md, worked.py, worked.out, size.py, size.out), `doc/specs/README.md` only if a requirement count changes. The supervisor applies; the edits are owed, listed by the text at the time of writing (found by `grep -n "stale under ADR 0008" doc/specs/spec-checkpoint.md doc/specs/spec-world.md`).

`spec-checkpoint.md` (CHK):

| Where | Edit |
| --- | --- |
| line 23 (what was checked) | drop the PCG-layout clause; keep "non-minimal encoding" |
| lines 116, 156, 1535 (3.1.2, `Stream`, FR-CHK-006) | remove `Stream` and the `"pcg:"` prefix; `Info` as D224 |
| line 246 (4.2 tag 1) | the World section as D223: the next agent ID and the construction flag only; the bucket and the pending totals are the ledger's, in tag 2 (4.2 line 247) |
| line 451 (5.4.1 `seed`) | the effective seed, read back (D225); also line 781 and the 5.4.1 note at line 527 |
| line 470 (R4) | the raw array example is the 32-byte hash only |
| line 618 (6.2 step 8) | "Install" has no PCG state and takes no word: the next agent ID, the construction flag, the tick, then the derived tables |
| lines 672, 706, 712 (6.4 `WithSeed`, 6.7, 6.8) | `WithSeed` refused because the header holds the seed; a reseed rewrites the header's seed; the restored world has the seed of the header and no stream position |
| line 976 (8.4 item 4), lines 1026 to 1034 (8.4 heading, its "this whole section is stale" note at 1028, 8.4.1) and 8.4.2 | item 4 reads "the seed is the same, and so is every stream"; the stale note and 8.4.1 (the 20 bytes of the PCG, the positional state) are deleted and the heading drops "and the positional state"; 8.4.2 ("A restore must not draw") stays, reduced to the one statement that FR-CHK-051 holds trivially (a restore installs nothing and takes no word) |
| lines 1235, 1340 (EXP row of 13, CHK-S2 and WLD-A5) | a paired comparison is aligned in the draws by construction (`spec-world.md` 7.6); the stream fingerprint of EXP 7.3 is not read |
| lines 1372, 1405 and `models/chk/worked.py`, `worked.out`, `README.md` (17.2 worked example) | regenerate the world section without the PCG bytes; the primitives and the reseed vectors stay; `worked.py` asserts the values against a Go program's printed output as before |
| `models/chk/size.py`, `size.out` and the size table | the world section loses the 20 bytes; rerun and requote |
| lines 1554, 1580 (FR-CHK-025, FR-CHK-051) | delete the "stale" sentences; keep the requirements |
| lines 1653, 1658 (the rows owed to WLD and to `architecture.md` 7.3) | mark them applied: FR-WLD-100 and 101 and `architecture.md` 7.3 already say it |

`spec-world.md` (WLD):

| Where | Edit |
| --- | --- |
| FR-WLD-100 and the 12 table (around line 2308 and line 1814) | the world's own contribution is the effective seed (header), the next agent ID and the construction flag; the pending flow totals and the construction bucket are named as the ledger's contribution (CHK 4.2 tag 2) |
| line 587 (construction, a pair of runs) | delete the clause about `spec-technology.md` 6.3.2 being "stale": TEC owns its text, and the sentence here says only that the pair is aligned |
| line 1814 (12 table, seed row) | delete the clause "the section of CHK 4.2 that still lists a PCG state is stale" (it is true after Phase 1) |
| lines 1856 to 1858 (12.3) | delete "5.4 still says informational only, stale under ADR 0008" |
| lines 1868, 1874 (13, the CHK row and the EXP row) | the CHK row reads "the contribution of 12 (the effective seed in the header, the next agent ID, the construction flag with the bucket while in construction; the tick in the header)"; the EXP row reads that a paired comparison is aligned in the draws by construction (7.6); the "Stale under ADR 0008" paragraph goes |
| line 2065 (17, WLD-A5) | the paragraph keeps its content and loses the word "stale"; the register entries L120, WLD-D2 and WLD-S4 stay with STB, which owns them |
| line 2228 (20, the N-214 row) | "Applied for WLD 7, 12, 13, 17 and 22"; "leave open" lists only the sites outside the slice (open point 3) |
| 16.5 | verified: no marked passage and no "PCG state" remain (grep); nothing to edit |

Not in this phase: `spec-world.md` line 2469 (the `spec-stability.md` row, S63 and MIG-S3), `spec-technology.md` 6.3.2 and 4.4, `spec-experiments.md` 7.3 and FR-EXP-062, `spec-calibration.md` 4.2.1: their rows (20, 21; STB and TEC later) edit them; N-214 stays open for them.

Tests: none. Exit check: `grep -n "stale under ADR 0008" doc/specs/spec-checkpoint.md doc/specs/spec-world.md` prints nothing; `grep -n "PCG state\|20 bytes\|Stream(" doc/specs/spec-checkpoint.md` prints only history sentences that name ADR 0008 as the reason; the models rerun and their `.out` match the quoted values; markdownlint on both specs is clean.

### Phase 2: race-aware scenario and determinism tests

Goal: R of D229 finishes inside the budget of D230 and keeps what D227 says.

Files: `scenario_helpers_test.go`, `invariants_test.go`, the five `scenario_s*_test.go` files that call `seedList(scenarioSeeds)` (S01 line 45, S04 line 76, S07 line 47, S16 line 38, S19 line 130), `determinism_test.go`.

Slow tests, with the work each does (from the code): `TestScenarioS19_HaltOnAndOff` 10 seeds, two runs each, 1095 ticks; `TestScenarioS04_StapleShortage` two packs, 10 seeds, 1400 ticks; `TestScenarioS07_PerishableGlut` four cost floors, 10 seeds, 400 ticks; `TestInvariants_Packs` (`invariantTable`: `one` 16 seeds, `second` and `three` 4 seeds each, 400 ticks, one parallel subtest per seed); `TestDeterminism_SameBytes` 18 cases of 400 ticks (the table is local to the test body); `TestScenarioS01` (700 ticks) and `TestScenarioS16` (1095 ticks) are reduced by step 2 too and are timed in step 1.

Steps:

1. Measure first, on idle hardware, one test at a time: `go test -race -covermode=atomic -coverpkg=./... -run '^TestScenarioS19_HaltOnAndOff$' -v -timeout 3h .` and the same for S01, S04, S07, S16 `TestInvariants_Packs` and `TestDeterminism_SameBytes`; record wall times in the Phase 2 commit message. This fixes the starting seed count against D230.
2. Add `raceSeeds = 2` and `func scenarioSeedCount() int` to `scenario_helpers_test.go`: `scenarioSeeds` normally, `raceSeeds` when `raceEnabled`. Replace `seedList(scenarioSeeds)` with `seedList(scenarioSeedCount())` in the five files. Add `func invariantSeeds(full int) int` there too: `full` normally, `min(full, raceSeeds)` when `raceEnabled`; the loop of `TestInvariants_Packs` uses `invariantSeeds(tc.seeds)` (the row keeps its own full count `seeds int`; a row of a later plan sets its full count and calls the same function, so no row uses `scenarioSeedCount()` for its seeds). Add `gateFPrefixes` (D229). The report titles that say "seeds 1-10" are built with the count (`"seeds 1-%d"`). Seeds 1 and 2 are kept because seed 1 is the golden seed.
3. `scenReport` writes per pack: the constant `scenariosPath` is replaced by `testdata/chains/expected/<pack>.scenarios.txt`, one `scenReport` block keyed by the pack name (`one.scenarios.txt` stays the file of pack `one`), and the title of every pack file carries the seed count that ran. `scenReport.flush` refuses `-update` under `-race` through a pure helper `updateAllowed(race, updating bool) bool` (D228): when `-update` is set and `updateAllowed` is false it calls `t.Fatalf` naming the flag and `-race`. The other `-update` writers (`checkGolden`, `golden_test.go`) write the golden of seed 1 and the full pack and are not reduced, so they are not covered; `TestUpdateAllowed_NeverUnderRace` covers the helper only.
4. In `determinism_test.go`, lift the case table of `TestDeterminism_SameBytes` into a package-level function `detCases(race bool) []detCase`; the closures it uses (`step`, `run`, `seven`, `ones`) and the type `detCase` are lifted with it (`detCases` builds them itself), and the loop keeps the cold `report`. Under `-race` it returns one case per lettered group (a) to (e) plus the two chunking extremes ("Run(1)+Run(399)" and "400 Run(1)"); without it, all. `detCases` is the variant list for one world; a module or pack world is run by its own test `TestDeterminism_<Name>` that takes the cases from `detCases(true)` by name (plan 24 D689), with a name starting `TestDeterminism` and seeds from `invariantSeeds`. The assertions, `detTicks` and `solveCold` are unchanged.
5. Remeasure R as a whole. If it is over the budget, lower `raceSeeds` to 1 (D230) or open D231 with the owner.

Tests added (all in package `economysim_test`, no requirement ID: they test the harness, `testing.md` 4.4 allows the tag to be omitted):

- `TestScenarioSeedCount_RaceAware`: equals `scenarioSeeds` when `!raceEnabled`, `raceSeeds` when `raceEnabled`, at least 1, and `seedList` of it starts at 1.
- `TestInvariantSeeds_RaceAware`: `invariantSeeds(16)` and `invariantSeeds(4)` equal the full count when `!raceEnabled`, `raceSeeds` when `raceEnabled`; `invariantSeeds(1)` is 1 in both; never below 1.
- `TestRaceSkips_HaveAnFPrefix` (D229): over the root `*_test.go` files, every `Test*` function that skips under `raceEnabled` starts with a prefix of `gateFPrefixes`; a table of three names checks the matcher.
- `TestScenReport_PerPackPath`: a report for pack X writes only `<X>.scenarios.txt` and never touches another pack's file.
- `TestUpdateAllowed_NeverUnderRace`: `updateAllowed(true, true)` is false; the other three combinations are true.
- `TestDeterminism_RaceCasesKeepEveryGroup`: `detCases(true)` has at least one case in each of (a) to (e) and is a subset of `detCases(false)` by name.

Requirement served by the existing tests that are reduced, unchanged: FR-REF-024, FR-REF-025, FR-REF-030, FR-STB-008, NFR-WLD-004 (cited on `TestDeterminism_SameBytes` and the `poshash_test.go` tests; it is shown by the two-architecture run, Phase 5). Exit check: R passes on the reference hardware in the budget; F passes with all ten seeds; `go test -short -race ./...` still passes; the coverage of R is not lower than the previous non-race run by more than a statement that only a dropped seed reached (list any such statement in the commit message and either add a case or note it under `testing.md` 9.5).

### Phase 3: ADR acceptances (owner actions)

A plan cannot accept an ADR (D232). The supervisor puts these in front of the owner; the owner makes the change or tells the supervisor to. Order: after Phase 1 is applied.

| File | Line | Change |
| --- | --- | --- |
| `doc/adr/0002-persistence-by-checkpoint.md` | 3 | `- Status: Proposed` becomes `- Status: Accepted` |
| `doc/adr/0009-the-toml-library-is-pelletier-go-toml-v2.md` | 3 | `- Status: Proposed` becomes `- Status: Accepted` |
| `doc/adr/0009-the-toml-library-is-pelletier-go-toml-v2.md` | after line 204 (Open point 3) | add: `*Note, <date>.* Accepted: \`config/guard.go\` imports \`unstable\` and \`config/fuzz_test.go\` holds \`FuzzLoad\`; the condition of Open point 3 is met.` |
| `doc/adr/README.md` | the rows beginning `\| [0002]` and `\| [0009]` | `Proposed` becomes `Accepted` in the status column |
| `doc/documentation_index.md` | the ADR 0002 and ADR 0009 rows beginning `Proposed decision:` | `Proposed decision:` becomes `Accepted decision:` |

Follow-ups for the supervisor: the update of the note that said "accept ADR 0002 before Stage 2" (N-019, its Stage 0 update) is closed by a dated update; ADR 0002 keeps its dated note that ADR 0008 narrows decision 3. Exit check: `grep -n "Status:" doc/adr/0002*.md doc/adr/0009*.md` shows Accepted; the README table and the index agree.

### Phase 4: where plans live; the order on record

Goal: D233 to D235 are in the live documents. The supervisor applies; this plan edits none.

| File | Edit |
| --- | --- |
| `doc/rules/architecture.md` 2.6 (line 170, `doc/plans/`) | "implementation plans: one or more per stage of `doc/roadmap-v1.md`, split into phases; working documents, deleted after the stage's sign-off (the commit is recorded in the roadmap); a plan decides nothing: the specs, the ADRs and these rules do". Fresh RULE-TAG. |
| `AGENTS.md` Layout (line 174) | the same words, kept in step with 2.6 (`AGENTS.md` says so) |
| `doc/roadmap-v1.md` Stage 0 and Stage 1 (lines 28, 86, 158 to 169) | each `plans/plan-phase-...` link becomes the plan's name plus "git history, `2abbe62^`" (D234) |
| `doc/roadmap-v1.md` Stage 2 intro and table | add after the intro: "The order of the work is `plans/plan-2-0.md`: plan 1 (work before any row), then CHK, REG, SCL, **DEM as the first module**, MON, EST, then plan 8 (the reference-pack rebalance, N-314, after the pool part of EST), then the other modules in the order of the table (a dependency order; the hash id and the checkpoint tag of each module follow the `Modules` row of `spec-config.md` 3.1 and `spec-checkpoint.md` 4.2, not the build order: `plan-2-0.md`)." Correct the intro sentence "in the order of their `Modules` pointers, which is also their dependency order" (line 202) in the same edit, and the matching sentence of `vision.md` section 4 if it has one. Change the REG row to "the graph, distances, components, overrides, the guarded relocation (unexported `relocate`, `RelocationRefusal`) and the edge status (`Graph`, `Snapshot.Edges`)" and the INT row to "external agents, `Direct`, `Relocate`, the slot setters, the queries" (plans 3 and 17; `spec-integration.md` 3.4.3, 4.5). Add two rows, "work before any row (plan 2-1)" and "reference pack rebalance (plan 2-8)", status `plan written`; give each existing row its plan. |
| `doc/roadmap-v1.md` "What is deliberately not built" (CI bullet) | replace by the owner's CI decision (Phase 5) |
| `doc/vision.md` section 4 (the Stage 2 / version 1 bullets) and section 5 item 2 | one sentence: version 1 is built in the order of `roadmap-v1.md` Stage 2 and `plans/plan-2-0.md`; DEM first among the modules; item 2 names the CI of Phase 5 or the by-hand run |
| `doc/spec-roadmap.md` (the plan-phase line, and "Open questions", answered list, around line 298 to 314) | fix the `plans/plan-phase` mention as for the roadmap; add an answered question "In what order is Stage 2 built?": the order of `plan-2-0.md`; DEM first; the rebalance after the pool part of EST; the reason is N-314 and N-277 |
| `doc/documentation_index.md` | applied: the Stage 1 plan rows are gone and 25 `plans/plan-2-N.md` rows are present (lines 16 to 40, read-it-when "you build that row"); only check that every link resolves |

Exit check: `grep -n "plans/plan-phase" doc/roadmap-v1.md doc/vision.md doc/spec-roadmap.md doc/documentation_index.md AGENTS.md` prints only the rows that name git history (`notes.md`, the specs and ADR 0009 are excluded, D234); every link in `documentation_index.md` resolves (`ls` of each path); rule tags changed in `architecture.md` only.

### Phase 5: CI and the two architectures (O3)

Goal: the owner decides; the decision is on record either way.

Recommended, if the owner says yes: a workflow file `.github/workflows/ci.yml` (not written by this plan) with three jobs:

- **check** (amd64): `gofmt -l .` empty, `go vet ./...`, `staticcheck ./...`, `go test -short -race ./...`, `go mod tidy` with `git diff --exit-code`, `govulncheck ./...`, Lizard (D238) on changed files.
- **full** (amd64): R of D229, then F, then upload `coverage.out` to Codacy (N-243: no coverage upload exists, so Codacy reports `MissingRequirements`), and a script that reads `go tool cover -func` against the thresholds of `testing.md` 9.1.
- **arm64**: a runner of that architecture, running the tests of D237 (golden, determinism, hash pin, chain tests, `TestInvariants_Packs`).

Without CI (the owner says no, or it waits): the two architectures are run by hand once per Stage 2 row that moves a golden, and in plan 24, with `docker run --platform linux/arm64 golang:1.25 go test ...` under emulation on the amd64 host, `-run` the tests of D237 and no `-race`. To keep only the tests under emulation, cross-build the test binary on the host (`GOOS=linux GOARCH=arm64 go test -c`) and run it in an arm64 container: `docker run --rm --platform linux/arm64 --cpus=4 --memory=16g --memory-swap=16g -v "$PWD":/w -v "<scratchpad>":/s -w /w alpine /s/ec.arm64.test -test.run ...`. The roadmap then says "by hand" and names the owner of the run.

Owner actions: (1) CI yes or no; (2) who owns the workflow and the Codacy token; (3) whether an arm64 runner is available; (4) unless the owner has an arm64 runner or machine, install the binfmt handler once on the development host (`docker run --privileged --rm tonistiigi/binfmt --install arm64`). Rulings asked at the start of Stage 2, so that no later plan waits on a first question: (5) plan 8 E0 (may the seasonal-crop feature C04 move from the pack to an overlay), with E1 to E3 shown for information; (6) plan 24 R2(a) (the realm pack is `five` renamed) and R3 (which machine is the reference machine for the SCL time). (7) approve the text of FR-VIA-060, FR-REF-041 and FR-REF-042 and the two gate figures of plan 8 D365 (plan 0 forbids a plan to invent IDs; the owner approves them); (8) accept ADR 0010 when plan 5 Phase 1 drafts it, or rule plan 5 O5; (9) may the `RPG` preset of `denomination/` (gold, silver, copper) live in the core module as the one display-only exception to the `AGENTS.md` setting-agnostic bullet (plan 23 D672), or move to `examples/` with plan 24. The answers to (5) to (9) go into the `roadmap-v1.md` Stage 2 paragraph. Exit check: `roadmap-v1.md` and `vision.md` item 2 carry the decision; if yes, a CI row exists in Stage 2 (`planned`) with this plan's recommended content; `TestGolden_OneOff` passes on linux/arm64 against the committed files, with the wall time recorded, and one emulated 50-tick run of pack `one` is timed and the seconds per tick recorded (`GOOS=linux GOARCH=arm64 go test -c -o <scratchpad>/ec.arm64.test .`, outside the repository on the host, then the binary run in the arm64 container of the "Without CI" paragraph, the two commands being `-test.run '^$' -test.bench BenchmarkStep_OneNoObserver -test.benchtime 50x` and `-test.run TestGolden_OneOff`, the benchmark being in `observe_bench_test.go`; arm64 availability is shown by this run and `TestGolden_OneOff`) (plan 24 estimates the `five` and `all-on` runs from it: about 8 times the agents), or the owner has said no to both CI and emulation and the roadmap records that criterion 2 cannot be shown.

### Phase 6: Lizard and markdownlint as Codacy runs them (N-243)

1. Baseline: run `lizard -l go -C 10 -a 8 -L 100 -w .` over the non-test Go files once, before any Stage 2 file exists; record the warnings (expected: `internal/checked.muldiv`, CCN 13) in the commit message. A row's gate is "no warning on the new or changed files"; a warning that was already in the baseline and is in a file the row does not edit is not the row's.
2. Markdownlint: run it with the repository's `.markdownlint.json` on the files a row writes. Which version Codacy runs, and whether it enables MD060, is read from Codacy (`codacy_list_repository_tools` and `codacy_list_repository_tool_patterns`) and written into the Phase 6 commit; if it cannot be read, the rule stays "spaced separator row, `| --- |`", which is what the plans already use.
3. Edits owed (D239): `testing.md` 9.1 gains `-timeout 60m` (D230), the pattern of F with its prefix rule (D229) and the sentence "under `-race` the scenario and determinism tests run a reduced seed set; the full set runs without `-race` (D229)"; `testing.md` 10.5 gains the same sentence; `plan-2-0.md` gates paragraph is amended by the supervisor. Fresh RULE-TAG in `testing.md`.
4. `production_report.go`, `viability_check.go`, `viability_table.go`, `viability_report.go`, `spawner_report.go`, `strategy_default.go`, `strategy_work.go`, `strategy_view.go` and the other continuation files named in `AGENTS.md` Layout: already listed there; `documentation_index.md` has no file-level row per Go file, so N-243's "add it to the layout" item is closed by this check: `ls *.go` and the Layout list agree (Phase 7 greps it).

Exit check: the baseline is recorded; the `testing.md` edits are applied; markdownlint is clean on `doc/plans/plan-2-*.md`.

### Phase 7: exit gate

- `gofmt -l .` prints nothing; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; `go mod tidy` with no diff.
- R and F of D229 pass; the wall time of R is recorded against D230.
- Lizard on the changed Go files (the test files are excluded by `.codacy.yml`, so the gate is empty for them).
- The greps of Phases 1, 3 and 4 hold.
- `ls *.go` against the `AGENTS.md` Layout: every root file is named in the Layout.
- Supervisor's checklist: every owed edit of Phases 1, 3, 4, 6 applied or listed as an owner action; the notes proposals below applied; owner actions (1) to (9) of Phase 5 asked and the answers to (5) to (9) in the roadmap Stage 2 paragraph.

Mutation testing: not applicable (no `internal/` package is new). Coverage: no package is new; the root's coverage does not fall (Phase 2 exit check).

## Catalogue appends

| Catalogue file | Append |
| --- | --- |
| `config.go`, `bounds.go`, `defaults.go`, `validate.go`, `hash.go`, `errors.go`, `event.go`, `option.go`, `tick.go`, `snapshot.go`, `ledger.go` | none, because the plan adds no module and no field (D220) |
| checkpoint tags | none; D223 fixes only the wording of tag 1 (and names tag 2 as the ledger's), which plan 2 builds |
| log keys, metrics | none |

## Cross-plan contract

Consumes:

- `raceEnabled` (`export_phase8_race_test.go`, `export_phase8_norace_test.go`) <- Stage 1 code
- `scenReport`, `seedList`, `scenarioSeeds`, `scenSem` (`scenario_helpers_test.go`) <- Stage 1 code
- `updateFlag` (`support_test.go`) <- Stage 1 code
- `.codacy.yml` and `.markdownlint.json` <- Stage 0 and 1 files
- `spec-world.md` 12, 12.2, 12.3, FR-WLD-100, FR-WLD-101 <- spec
- `adr/0008-random-draws-are-keyed-streams.md` <- ADR

Provides:

- The CHK and WLD specs free of ADR 0008 residue (`spec-checkpoint.md`, `spec-world.md`) -> plan-2-2
- `scenarioSeedCount` (`scenario_helpers_test.go`) -> plans 3 to 7, 9 to 14, 17, 19 and 24; `updateAllowed` -> plans 4, 10, 17, 19 and 24 (the plans that name them; plan 8 uses neither)
- `invariantSeeds` and `gateFPrefixes` (`scenario_helpers_test.go`) -> `invariantSeeds` to plans 5, 7, 9, 11, 12 and 15 (the plans that add an `invariantTable` row); `gateFPrefixes` to plan 8 only. Every other plan that skips a test under `raceEnabled` (2, 4, 16, 18, 19, 24) follows the prefix rule of D229 by name, with no import
- `scenReport` per-pack path -> plans 5, 7, 12, 24
- `detCases` (`determinism_test.go`) -> every plan that adds a determinism case. `detCases` is the variant list for one world; a module or pack world is run by its own test `TestDeterminism_<Name>` that takes the cases from `detCases(true)` by name (plan 24 D689), with a name starting `TestDeterminism` and seeds from `invariantSeeds`
- Gate R and gate F of D229 -> plan-2-2 to plan-2-24
- ADR 0002 and ADR 0009 Accepted (`doc/adr/`) -> plan-2-2
- The plans convention of D233 (`architecture.md` 2.6, `AGENTS.md`) -> plan-2-2 to plan-2-24
- The Stage 2 order on record (`roadmap-v1.md`, `vision.md`, `spec-roadmap.md`) -> plan-2-24
- The Lizard baseline and the markdownlint rule of D238 -> plan-2-2 to plan-2-24
- The CI decision and, if yes, the workflow content of Phase 5 -> plan-2-24

## Testing

Per `testing.md` 11. Requirement-to-test table:

| Requirement | Test |
| --- | --- |
| none (harness) | `TestScenarioSeedCount_RaceAware`, `TestInvariantSeeds_RaceAware`, `TestRaceSkips_HaveAnFPrefix`, `TestUpdateAllowed_NeverUnderRace`, `TestDeterminism_RaceCasesKeepEveryGroup` |
| NFR-WLD-004 | `TestDeterminism_SameBytes`, the position-hash tests of `poshash_test.go`, `TestHashConfig_ReferencePinned` (existing; run on two architectures by Phase 5 or by hand) |
| FR-CHK-006, FR-CHK-025, FR-CHK-051, FR-WLD-100, FR-WLD-101 | no test: the spec text is corrected; plan 2 writes the tests (`Requirement:` lines) |

Golden files: `testdata/golden/off/` and the pinned hash do not move (D220); no module overlay, no module-alone golden, no fork-at-N test (no module). The registry and catalogue tests are unchanged. Fuzz targets: none. `testdata/chains/expected/one.scenarios.txt` is not rewritten (D228), and it is still stale (N-312, plan 8 refreshes it).

## Observability

None, because the plan adds no event, `Warn`, log key or metric (`observability-and-logging.md` 9). The only observer-visible change is none: tests are the only code touched.

## Risks and open items

- **R1. Timing is a guess until Phase 2 step 1.** The plan was written without running the suite; the starting value `raceSeeds = 2` is an estimate from the work counts of Phase 2. Owner: the Phase 2 implementer. If one seed still fails the budget, D231 goes to the owner.
- **R2. Coverage under reduced seeds.** A statement reached only by seeds 3 to 10 would drop out of R. Owner: Phase 2 exit check; fix by a unit case, not by raising the seed count.
- **R3. The allocation baseline** (`TestStep_AllocationsStayAtTheBaseline`) is measured under `-race -covermode=atomic -coverpkg=./...`; R keeps that condition, F skips it. Every row that adds per-tick allocations moves the baseline in R only.
- **R4. CHK 5.4.1 and WLD 12.3 contradict today** (informational versus read back). D225 resolves it by `spec-world.md` 12.3, ADR 0008 and `architecture.md` 7.3; the supervisor confirms with the owner if CHK's owner disagrees (this is also an owed edit in Phase 1).
- **R5. Residual N-214 markers** in EXP, CLB, STB, TEC and the `spec-stability.md` row of `spec-world.md` (line 2469) are outside this plan: plans 20 and 21 own EXP and CLB; STB and TEC later. N-214 stays open for them.
- **R6. arm64 availability** (Phase 5): without a runner or emulation that finishes, NFR-WLD-004 cannot be shown and plan 24 cannot exit. Owner: the owner.
- **R7. Codacy's markdownlint version** is not known to this plan; Phase 6 reads it.
- **R8. Every later pack and scenario is slower.** `five.toml` and the module scenarios add to R. Each module plan states its own race reduction under D226 if it adds a scenario. A plan that skips a test under `raceEnabled` names it with a prefix of D229 (guarded by `TestRaceSkips_HaveAnFPrefix`) and a plan that adds an `invariantTable` row calls `invariantSeeds`.

Note proposals (text only; the supervisor edits `doc/notes.md`):

1. New note, areas CODE, ALL: "Under `-race` the scenario and determinism tests run a reduced seed set (`raceSeeds`, `detCases(true)`); the full set runs without `-race`; `-update` is refused under `-race`; the gate is two runs (plan 2-1 D229). A row that adds a long scenario uses `scenarioSeedCount()`."
2. Update to N-214: "CHK and WLD applied (plan 2-1 Phase 1, date); open for EXP (plan 2-20), CLB (plan 2-21), STB and TEC."
3. Update to N-243: "Lizard baseline recorded (plan 2-1 Phase 6); `muldiv` still open; markdownlint version read from Codacy; coverage upload is the CI job of plan 2-1 Phase 5."
4. Update to N-019 (CI): "CI decided: <owner's answer>."
