# Plan: Stage 2, row 25: the stable reference chain (collapse fix, plan 2-8 acceptance, first setting run, PR D)

**Status: Proposed (2026-11-11).** Sequences `vision.md` 5 item 1 ("the reference chain runs to a stable state from a warm start and passes the STY bands") into phases. It is the plan that plans 2-8 (stopped by D369), 2-19 (bands unset, O1) and 2-24 (PR D, 6a) each wait on. Input: the decision memo `DEC-stability.md` of the worker session (summarised in the Decisions below, so the plan is self-contained); the old plans are in git history (`git show 31e3a05^:doc/plans/plan-2-8.md`, `plan-2-19.md`, `plan-2-24.md`). The plan sequences; where it needs a ruling it says so (Risks, R1 to R4). Decision-ID block: **D700 to D719**.

Rule tags read (all three files, in full): architecture=0052d4050fd8b5aa5325ae2e, observability=e7cfbd33b1620a09dc78dff7, testing=120890253bd2f6c110a015d2. Notes tag: bb1aecc236a36f9a48ffd6a2.

Specs and requirement IDs touched: `spec-reference-chain.md` (REF) 4.4, 7.4 (X4), 8.4, 8.7, FR-REF-007, 009, 024, 039, 041, 042; `spec-viability.md` (VIA) FR-VIA-060, 4.7; `spec-stability.md` (STB) FR-STB-015; `spec-stylised-facts.md` (STY) 8.1.1, 8.4, 8.6, 18, FR-STY-006, 077 to 079, 087; `spec-spawner.md` (SPN) SPN-A8, FR-SPN-042 (only on ruling R2); `vision.md` 5.1; `roadmap-v1.md` Stage 2 table. Notes read (index entries ALL, REF, STY, DEM, STB, MON): N-015 (open, placeholders), N-027 (apply: no default is called stable), N-030 (open), N-045, N-067, N-087 (open, STB counters exist), N-114, N-123 (cold start lockstep, open), N-134 (apply as a runbook step), N-135 (apply: hold-out 51 to 60 is a report), N-203 (apply: stability claim with `Modules.Technology` off), N-217 (apply: no module, no trace), N-306, N-312 (apply: closed by the outcome), N-314 and N-325 (apply: closed or re-pointed by the outcome), N-335 to N-345 (N-336, N-337, N-341, N-342 leave open; N-338, N-339, N-340, N-344 apply; N-343 leave open, arm64 is not this plan's; N-345 read, no new field planned).

## Scope

**Builds.**
1. A survival-screen harness in test code (`scenario_screen_test.go`): runs a pack with a list of overlay layers and prints, at fixed ticks, households, heads and unfed per role, the money share per role, the price of each market against its reference, and the X4 criteria. It reuses `runPack`, `incomeObs` (`scenario_income_test.go`), `stablePredicate` and `newStableRun` (`scenario_stable_test.go`). Flags follow the pattern of `indicator/stylised_helpers_test.go` (`flag.*`, no environment variable, N-340).
2. A diagnosis (note and log) of why a household population goes extinct on `one-demand`, with the memo's hypotheses H1 to H4 (D703) tested one by one.
3. A bounded trial ladder over pack data and existing config knobs (D704), kept in `testdata/chains/overlay/stable.toml` and, if a rung needs it, region documents. Accepted rung or stop (D708).
4. Only on ruling R2: a spec amendment for an engine rule (D705). No engine code is written under this plan; a separate plan follows a ruling.
5. If a rung is accepted: one regeneration commit (warm documents, `expected/*.toml` including the new `expected/famine.toml`, hash pins, fixtures, goldens, `examples/realm/pack/`), D709.
6. Plan 2-8 acceptance: `TestPacks_IncomeBalance` and the `one-demand` X4 test un-skipped; scenario records re-recorded.
7. The first reviewed setting run (STY 8.4, 8.6) and the `value` and `set_by` lines for the keys of STY 8.1.1, D710 to D712.
8. PR D: `TestScenarioStable_FiveWarm` (on the claim pack, D702), `expected/five.scenarios.txt`, `golden/stylised/five.txt` and `five.stability.txt`, `x3.ms_per_tick`, the hold-out report, the examples refresh.
9. Close-out: spec wording (`vision.md` 5.1, FR-REF-041/042 status), roadmap rows, notes.

**Does not build.** No module and no catalogue append (no event, error, option, tick position or `Config` field is added, so `tick.go`, `event.go`, `config.go` are untouched and `golden/off` moves only for D709's pack-data reason, architecture 2.10.5, testing 7.1.1). No engine rule (SPN, AGT, BEL, WEL): R2. No CI workflow and no arm64 run (N-343). No Stage 3 module. No band set on a run that collapses (STY 8.4.1). No hand edit of a warm document (FR-REF-007).

## Decisions

- **D700 (what is wrong, as measured 2026-11-11, seed 1, households at ticks 100/300/600/1100).** `one` 557/193/45/0; `one-demand` 597/335/0/0; `five` warm 4344/1215/0/0; the all-on pack (`allLayers`) 2240/88/0/0. Every pack goes extinct; the modules merged since plan 2-8 did not cure it (the all-on pack is worse, with 99 households unfed at tick 1). Source: scratch probe of the worker session, not kept; Phase 1 re-measures it with the harness.
- **D701 (probe evidence on `one-demand`, seeds 1 and 2).** (a) By tick 200 cooks hold 83% of the money (18.6 M of 22.4 M). (b) A rate levy on cooks plus level top-ups to the five other roles tripled the money (22 M to 61 M) and delayed, not prevented, the end (557 households at tick 300, 0 by 600). (c) The grower role, the only staple producer, is extinct by tick 300 in every run; then seed and staple stock are 0 and every role is unfed. (d) Closing `cook`, or closing all non-farm roles, leaves growers dying (278 to 7). (e) `entry_share = exit_share = 0` freezes the head count at 600 (the ladder is off: a control, not a fix). (f) `role_hysteresis` 5000, `role_cooldown` 120, `entry_share` 20: growers 223 to 4. (g) Staple yield at 50% and 30%: the slowest decline, still extinct by tick 600. (h) The staple trades at 42 to 55 against a reference of 96, with 11 725 units in stock at tick 100. SPN-A8 makes the extinction absorbing.
- **D702 (which pack carries the claim).** Development on `one-demand` (about 1 s for 1 100 ticks). The claim is on the all-on pack (`allLayers`, five regions, 5 000 households), because `five` is upkeep-only and a glut by construction (VIA 4.7, FR-REF-018, ruling E1 of 2026-10-28). `vision.md` 5.1 and FR-REF-042 are reworded to say so (Phase 8); the bands are the same. Seam: one constant `stablePack` (a layers slice) read by `TestScenarioStable_FiveWarm` and the stylised `five` row. Reversible by changing the constant and the sentence.
- **D703 (hypotheses the diagnosis tests, in order).** H1: the staple is overbuilt (census `lambda*` 2.5, start capacity three times the labour term, VIA 5.2), so its price falls below the growers' cost and the role starves (D701 (h)). H2: the cold start is a lockstep (N-123): the first cohort of growers harvests together, a glut follows, and the delay of the 60-tick job is a cobweb. H3: money concentration in cooks removes the bids of the other roles (D701 (a)); alone it is not enough (D701 (b)). H4: extinction is absorbing (SPN-A8), so any dip below one grower is final. The test of each is a harness run with one cause removed, never a guess: H1 by a census scaled to demand, H2 by staggered starts, H3 by a recycling pair, H4 by a seeded grower in a read-only experiment (a `Direct` or `AddAgent` call from test code, `integration.go`).
- **D704 (the ladder).** At most **12 trials**; stop after **3 consecutive non-improving**; at most two values per knob (the value and one bisection). Each trial is a layer file in a scratch directory outside `testdata/` until accepted. Knobs allowed: pack data (census, yields, start money, start stock, tool durability), the existing config fields of `Config.World` (`entry_share`, `exit_share`, `role_hysteresis`, `role_cooldown`, `entry_ratio`, `change_privation`, `remove_privation`, `renewal`, `entry_pool_share`), module flows (money supply grants and levies, `Modules.Estates` `pool_release`, `relief_below`), and the demand module's wants. Not allowed: an engine rule (R2). Objective, screen: no role extinct and households at least `x4.households_bp` prior (9 000) of the start over ticks 235 to 1 095, on search seeds 101 to 104 (STY 8.2 search block, never a gate seed). Acceptance: the same on seeds 1 to 10 and X4 (a) to (f) on seeds 1 to 4, then the gate facts of D711. Seeds 51 to 60 are never an input.
- **D705 (engine rule, gated).** If Phase 1 shows that no data or knob rung can pass because the cause is a rule (for example SPN-A8, the absence of a staple ask floor, N-314 (c)), the plan stops the ladder, writes the proposed spec amendment as text (SPN or STB, with the config key that would hold it, default equal to the current behaviour per architecture 2.10.5), and asks R2. No code. Seam: a config key defaulting to off.
- **D706 (famine pack).** `expected/famine.toml` and its `TestExpected_Packs` row are created by re-solving `warm-famine.toml` with the solver (FR-REF-007; today it fails validation, maker `jobs` 10 794 against the bound 10 000, plan 7 O5). If it still fails, one bisection: the famine census halved. Only `famine.toml` and `warm-famine.toml` are touched. The famine band is a bound on the shape of the fall (STY-S02, S06 keys), not "no collapse".
- **D707 (band method).** STY 8.4 as written: setting seeds 1 to 50 (tier L 1 to 10), headroom 2 500 bp (5 000 for fewer than 40 seeds), two significant digits rounded outward, never looser than a regime threshold. Two report-only columns are added to the proposal text: the share of post-warm-up ticks where a price gap exceeds the transport cost (parity bounds, Baulch 1997, for P06) and the analytic supply-over-demand slope ratio of a staple-like chain (Ezekiel, for P03). Neither is a key or an assertion. The source status is "search summary only, not opened in full" and STY 18 says so.
- **D708 (stop rule).** If the ladder stops without acceptance: Phases 3 and 4 (regeneration, acceptance tests) are skipped, no pack, golden, pin or fixture moves, the log table and the diagnosis go to the owner as R1 (record criterion 1 as unmet and carry it, plan 2-8 E3), and Phase 6 sets no band on a collapsing run (STY 8.4.1). Phases 1, 2 and 7 still run, and Phase 6 runs only its cost record and hold-out report.
- **D709 (regeneration once).** One commit, in this order: warm documents, `expected/`, Go literal `referenceConfig()` and the two hash pins, checkpoint fixtures (`format-1-*.bin/.txt`, `-update`), goldens (`off/`, `all/`, module sets that render a changed pack, `events-*.txt`, `*.positions.txt`), `examples/realm/pack/` through `rename.sh`, scenario records. The PR carries the testing.md 7.2 text: what changed (the accepted rung), the requirement it serves (FR-REF-041, 042, N-312, N-314), and that no module code changed. The off set moves because a pack changed (plan 2-8 D371 precedent), not because a module left a trace.
- **D710 (reviewer, O1).** The supervisor, as the owner's delegate, reviews the first setting run; an agent prepares the proposal with `-stylised.propose` and never commits a `value` line in the same commit as code. The review commit records, per miss, whether the band, the code or the fact was wrong (STY 8.4.5, testing 7.2). Nobody approves a band for code they wrote.
- **D711 (keys that must be set).** The keys of STY 8.1.1: C-BP (P01, P02), C-NC (P03), P06, S01, S02, S06, C-WS and G01, in `bands/regimes.toml` and the band files of the packs those facts name; every other B key may stay unset (STY 8.1.1).
- **D712 (cost figure).** `x3.ms_per_tick` in `bands/five.toml` is the figure of `cost_five_test.go` on a named machine, set by the reviewer's commit, never asserted (testing.md 8.2, 10.2).
- **D713 (what the exit says).** The words are those of STY 7.3: "the claim pack passes the X4 predicate and the version 1 bands in the recorded run", never "stable" of a pack or a default (N-027).

## Phases

### Phase 1: harness and diagnosis (test code, notes, no pack change)

Why first: every later phase needs a cheap, repeatable survival measure, and the ladder is only worth running if the cause is data-shaped (D705).
Goal: the harness exists and H1 to H4 are each confirmed or rejected with a log.
Files: `scenario_screen_test.go` (new), `doc/notes.md` (the diagnosis note, supervisor edits).
Items:
1. `TestScreen_Packs` (flag `-screen.pack`, `-screen.layers`, `-screen.seeds`, `-screen.ticks`, `-screen.at`): skipped unless a flag is given, so it never runs in a gate; with `-short` always skipped. It prints the table of D700/D701 and the role waterfall (gross sales, recipe-input cost, margin as a share of upkeep, from `incomeObs`).
2. `TestScreen_Reproduces` (runs by default, not `-short`): the households of `one` at ticks 100/300/600 on seeds 1 to 3 equal the series recorded in `expected/one.scenarios.txt` (557, 193, 45 on seed 1, the "wants off (pack one)" rows of `expected/one-demand.scenarios.txt`; `expected/one.scenarios.txt` is stale) so a harness that drifts is seen. Requirement: FR-REF-039.
3. Run H1 to H4 (D703) on `one-demand` seeds 101 to 104; log the table in this plan's Audit section and in the note.
4. Decide, and write down, the branch: ladder (D704) or R2 (D705).
Tests: unit level for the harness helpers (a table test of the share and waterfall arithmetic on a hand-built snapshot, with the arithmetic in the comment, testing 5.5); the reproduce test is integration level.
Exit: the log exists; R2 is asked if D705 applies; `go test -short -race ./...` clean; coverage not lower.

### Phase 2: the trial ladder (scratch layers, then one overlay)

Why second: it needs Phase 1's harness and its branch decision.
Goal: an accepted rung or a stop (D708).
Files: scratch directory outside the repository for trials; on acceptance `testdata/chains/overlay/stable.toml` (new) and, if a rung needs it, `testdata/chains/one-demand.toml`, `testdata/chains/five.toml` region documents (nothing in `base.toml`, `lines.toml` unless the rung needs it, then the plan-2-8 D369 "closed after one ladder" rule applies and it is a ruling item).
Items:
1. Run the ladder of D704 on `one-demand`, search seeds 101 to 104; record each trial in a table (change, households at 100/300/600/1100, extinct roles, score).
2. Port the best rung to the claim pack (`allLayers` plus the overlay) and screen it (D704 objective); at most 2 of the 12 trials may be spent on the port.
3. Acceptance: X4 (a) to (f) on seeds 1 to 4 for 1 095 ticks, the invariants of REF 7.5 every tick (audit on), S01, S04, S19 results recorded, and no role extinct on seeds 1 to 10.
Tests: the acceptance run is an integration test at the top of the pyramid (testing 2); the overlay's own validity is a unit-level `Validate` test (`TestPacks_PassTheViabilityRules` extended with the claim pack).
Exit: the table is in the plan and in N-325's successor note; an accepted rung (go to Phase 3) or a stop (D708, skip Phase 3).

### Phase 3: regeneration, once (only after acceptance)

Why third: pins and goldens must move once, after the data is final (D709).
Files: `testdata/chains/warm-*.toml`, `testdata/chains/expected/*.toml` (+ new `expected/famine.toml`), `reference_test.go`, `hash_test.go` pins, `config/expected_test.go` (famine row), `testdata/checkpoints/format-1-*`, `testdata/golden/off/`, `testdata/golden/all/`, module golden sets that render a changed pack, `examples/realm/pack/` (`rename.sh`), `testdata/chains/expected/*.scenarios.txt`.
Items: regenerate in the D709 order with `-update` on amd64; `famine` per D706; re-run the runbook of REF 8.7 (income gate, screen, warm start, goldens once, scenario record).
Tests: `TestExpected_Packs` (with `famine`), `TestWarmDocuments_Bytes` (FR-REF-007), `TestGolden_*`, `TestHashConfig_*`, `TestFixture_*`, `TestExamples_PackCopiesAgree`, `TestAllPack_*`; the module-alone goldens and the proof that the module-off hash does not move except by the pack change (testing 7.1.1).
Exit: the full gate F passes; the PR text of testing 7.2 is written.

### Phase 4: plan 2-8 acceptance tests

Files: `scenario_stable_test.go`, `viability_pack_test.go` (or the file of `TestPacks_IncomeBalance`), `testdata/chains/expected/one.scenarios.txt`, `one-demand.scenarios.txt`.
Items: un-skip `TestPacks_IncomeBalance` (FR-REF-041, the gate of plan 2-8 D365: lowest `IncomeBP` 8 000, highest at most 1.5 times the lowest, with the listed exceptions) and give `one-demand` its X4 test (FR-REF-042); `TestScenarioStable_OneWarm` stays the upkeep-only control with its reason updated (FR-REF-018). Re-record the scenario files with their flags. Spec status of FR-REF-041 and 042 changes from "not met" to the outcome.
Tests: those named; each un-skipped test names its `Requirement:` line and, if it was a `Fix:`, the bug it pins (testing 5.2, 5.3).
Exit: no skip carries the N-312/N-314 reference; N-312, N-314, N-325 closed or re-pointed.

### Phase 5: first reviewed setting run

Why fifth: bands need a pack that does not collapse (STY 8.4.1) and the module plans are merged.
Files: `testdata/chains/bands/regimes.toml`, `bands/five.toml` and the band files of the packs the D711 facts name, `testdata/golden/stylised/` (new records).
Items:
1. Preconditions of 8.4.1 checked and logged (commit, config hash, `expected/` present, no failed step, every X and O fact passes on every setting seed).
2. `go test ./indicator/ -run TestStylised_Propose -v -count=1 -timeout 240m -stylised.tier=n -stylised.propose` (seeds 1 to 50; tier L 1 to 10); the proposal prints the two report columns of D707.
3. The reviewer (D710) commits `value` and `set_by = "commit=<id>;config=<hash>;seeds=1-50"` lines for the keys of D711 only, with the three-way decision in the PR.
4. `x3.ms_per_tick` (D712).
Tests: `TestRegistry_Spec`, `TestBandFiles_Order`, `TestBandFile_Grammar` stay green; the key set now has values (the registry test that asserted "unset" is updated in the same commit with the reason, testing 5.4); `TestSetBand_HeadroomAndRounding` etc. unchanged.
Exit: every key of D711 has a value, or the finding for the one that cannot is recorded and the plan reports criterion 1 as not met (D708).

### Phase 6: PR D (the exit run)

Files: `scenario_stable_test.go` (`TestScenarioStable_FiveWarm`), `testdata/chains/expected/five.scenarios.txt` (new), `testdata/golden/stylised/five.txt`, `five.stability.txt` (new), `examples/realm/` and `examples/colony/` tests, `bands/five.toml`.
Items: `TestScenarioStable_FiveWarm` on `stablePack` (D702), 1 095 ticks, seeds 1 to 4, audit on, thresholds read from `bands/five.toml` where set (priors otherwise, logged as priors); the STY release gate `-stylised.release` on seeds 1 to 10; the hold-out seeds 51 to 60 as a report (`-stylised.report`, never asserted); record files with `-update` on amd64; the examples refresh builds and runs (`TestExamples_*`).
Tests: named above; race: the 5 000-agent tests skip under `raceEnabled` and run in gate F (names start with a prefix of `gateFPrefixes`, `TestRaceSkips_HaveAnFPrefix`).
Exit: the recorded run exists; the sentence of D713 is the only claim.

### Phase 7: close-out and exit gate

Files: `vision.md` (5.1 wording, D702), `spec-reference-chain.md` (FR-REF-041/042 status, 7.4 claim pack), `spec-stylised-facts.md` 18 (the two sources, status "search summary only"), `roadmap-v1.md` (rows: rebalance, STY, examples; sign-off paragraph), `doc/documentation_index.md` (if a doc is added), `doc/specs/README.md` (only if a range changes), `doc/notes.md` (close N-312, N-314, N-325, N-338, N-344, N-339 per outcome; add the diagnosis note; replace the tag).
Items: edit the specs and roadmap in the same change as the outcome; run the gates of `AGENTS.md` (gofmt, vet, staticcheck, `go test -short -race ./...`, gate R, gate F, `go mod tidy`); state in the sign-off that criterion 2 (arm64) is still N-343.
Exit: all gates pass, or the failing one is named with its output.

## Security and trust boundaries

N/A: no input, endpoint, permission or parser is added. The harness reads checked-in files through `os.DirFS("testdata/chains")` and flags in test code only; the library is not touched (architecture 1.1, 11.1).

## Compatibility and rollback

- Contract: no exported identifier, config field, event or checkpoint format changes. A checkpoint fixture changes only because its pack data changed (D709), and `format-1-*` keeps its format version.
- Rollout: Phase 3 is one commit, so the revert is `git revert` of that commit; the overlay `stable.toml` and the `stablePack` constant are the only seams (D702, D704). A band is data: a wrong `value` line is reverted by deleting it, and the key reports again (STY 8.4.5).
- Dependency failure: a missing optional file (`expected/famine.toml` before Phase 3) is handled by the existing staged lists (`notBuilt`, architecture 9.7); a failing solver re-solve stops Phase 3 before any file moves (D706).

## Catalogue appends

None: no event, sentinel, option, tick position, bound or default is added (architecture 2.10.2, 2.10.5). The only data appended are band `value` lines and overlay/expected/golden files. If R2 is ruled and a plan follows, that plan owns its catalogue entries.

## Cross-plan contract

- Consumes: `internal/verdict`, the `stylised_*` helpers and flags (plan 2-19); `incomeObs`, `stablePredicate`, `newStableRun`, `x4Priors` (plan 2-8 Phase 2); `cost_five_test.go`, `allLayers`, `examples/realm` (plan 2-24); `internal/calibration`/`experiment` only if a trial needs a grid (plan 2-21); `Direct`/`AddAgent` (plan 2-17) for the H4 experiment.
- Provides: the accepted overlay and regenerated data; `bands/*.toml` values; `expected/five.scenarios.txt`; the stylised records; the closed Stage 2 criterion 1 or its recorded non-closure.
- Conflicts: any other branch editing `testdata/chains/*`, `golden/*` or `reference_test.go` before Phase 3 forces a second regeneration (plan 2-8 Risks 5): the supervisor serialises.

## Testing (testing.md 11)

- Requirements touched and covering tests: FR-REF-039 (`TestScreen_Reproduces`, scenario records), FR-REF-041 (`TestPacks_IncomeBalance`), FR-REF-042 (`one-demand` X4 test), FR-REF-007 (`TestWarmDocuments_Bytes`), FR-REF-024 and FR-STY-077 to 079 (the review rule, PR record), FR-STY-087 (stylised record tests), FR-STB-015 (scenario files), FR-VIA-060 (`TestPacks_IncomeBalance` reads the column).
- Lowest level: harness arithmetic is unit; pack validity is `Validate`; survival is necessarily an integration run (the top of the pyramid has a handful of tests; this plan adds three: the one-demand X4, `FiveWarm`, `Screen_Reproduces`).
- Goldens: Phase 3 moves them once, with the 7.2 explanation; `golden/off` moves for a pack-data reason only.
- Fuzz: no new parser or validator. Coverage must not drop (9.2); the harness is test code.
- Each un-skipped test is run against a deliberately broken input (a pack with a grower census of 0) to see it fail (9.3).

## Observability (observability-and-logging.md 9)

None, because no failure path, state transition, strategy decision, loop or metric is added to the library: the plan changes data, tests and specs. Existing counters and events (`UpkeepShortfall`, `EntryLimited`, `ProductionHalted`, the "role extinct" `Warn`) are what the harness reads; no new key is introduced. The harness prints through `t.Logf`.

## Risks and open items (rulings)

- **R1 (owner or supervisor).** If the ladder stops (D708), criterion 1 is recorded unmet and carried to a later stage. Options: E3 as plan 2-8 named it; or authorise R2.
- **R2 (owner or supervisor).** An engine rule (D705): amend SPN-A8/FR-SPN-042 (re-seed an extinct role with unmet demand), or add a staple ask floor. Both change golden output and are spec changes first.
- **R3 (supervisor).** The claim pack (D702) and the rewording of `vision.md` 5.1.
- **R4 (supervisor).** O1 reviewer (D710) is assumed to be the supervisor; if the owner wants to review, only the commit author changes.
- **Cost.** Phase 5 setting run: 50 seeds on several packs, `-timeout 240m`; Phase 6 `FiveWarm` about a minute a seed; gate F needs about 120 minutes (testing 9.1). The ladder runs without `-race`.
- **Equivalence.** A rung that only freezes the head count (`entry_share = exit_share = 0`, D701 (e)) is not an acceptance: the ladder is switched off, so the objective also requires the failure ladder and entry to be on (D704).
- **Source quality.** The outside references of D707 were read through search summaries only; they justify the form of a column, not a number.
- **Note proposals (text; the supervisor edits `doc/notes.md`).** (1) REF, STY, STB: the diagnosis (D700, D701) as the successor of N-325. (2) STY: the claim pack is the all-on pack (D702). (3) REF, ALL: adding goods re-runs the rebalance (plan 2-8 D370 stands).
