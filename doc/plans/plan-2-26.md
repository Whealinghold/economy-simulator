# Plan: Stage 2, close the MER row (merchant reruns, X1 and X2 dynamic, determinism, invariants, the window term)

**Status: Proposed (2026-11-11).** Sequences the closing work of the MER row of [roadmap-v1.md](../roadmap-v1.md) Stage 2 ("built except the merchant reruns, X1 and X2 dynamic, `TestDeterminism_Merchants` and the `invariantTable` row", audit of 2026-11-11; plan 2-15 Phase 8, N-084) and the one requirement of that module with no code (FR-WEL-009's merchant term). A plan decides nothing: the specs, the ADRs and `doc/rules/` decide; the decisions below are the supervisor's picks, recorded in the memo `DEC-merchants.md` (D1 to D6 there are D700 to D705 here). The plan text of plan 2-15 is in git history (`git show 31e3a05^:doc/plans/plan-2-15.md`).

Rule tags read in full: architecture=0052d4050fd8b5aa5325ae2e, observability=e7cfbd33b1620a09dc78dff7, testing=120890253bd2f6c110a015d2. The worker of Phase 1 reads `doc/notes.md` in full and quotes its `Notes tag:` before touching a spec (AGENTS.md); the entries read for this plan are N-084, N-099, N-122, N-214, N-295, N-315, N-332, N-333, N-337.

Specs implemented or edited: [spec-reference-chain.md](../specs/spec-reference-chain.md) (REF: X1, X2, FR-REF-038, the register rows of 7.6 and 7.9), [spec-merchants.md](../specs/spec-merchants.md) 12.3, FR-MER-061, FR-MER-062; [spec-welfare-and-wages.md](../specs/spec-welfare-and-wages.md) 2.3.3, FR-WEL-009; read only: spec-transport 15 (S30 to S32), spec-information (S46, S47), STY 8.4.1.

Decision block: **D700 to D719** (D700 to D705 used).

## Scope

**Builds.**

- The merchant reruns `TestScenarioS30_WithMerchant`, `S31_WithMerchant`, `S32_WithMerchant`, `TestScenarioS46_MerchantRumouredGap` in the new `scenario_merchant_reruns_test.go`.
- `TestScenarioX1_Arbitrage` and `TestScenarioX2_ComparativeAdvantage` (the dynamic parts) in the new `scenario_x_test.go`, report-only apart from the exact checks and any band that has a value.
- `TestDeterminism_Merchants` (new `merchants_determinism_test.go`), the `invariantTable` row `two+merchants` and `TestInvariants_MerchantsScripted`.
- The merchant term of the effective welfare window: `merchantTripTicks` in `contract.go` and one guarded line in `longestDuration` (`validate.go`).
- The spec, notes, roadmap and documentation-index edits of Phase 1.

**Leaves out.** Setting any band or threshold (STY 8.4.1, plan 2-19 O1, plan 2-8; the `x1.*` and `x2.share_band_bp` keys stay without a `value`). A merchant warm-start group (plan 2-15 D516 stands). Transport lanes, stability, CI (other tasks). `testdata/golden/experiment/one.txt` re-record (plan 8). FR-MER-064 (Stage 3). A cap on `AddAgent` calls (INT, note only, Phase 6). The exit-gate runs of rows INF, MIG, EXT, IND, EXP, CLB and otel: they have no missing code or test (memo section 1) and their gates are the supervisor's. No new event, metric, log key, config field, ledger reason or catalogue entry.

## Decisions

- **D700** (memo D1) Scope of the reruns: S30, S31, S32 are new reruns with the merchant; S46 is the rumour-lag cells that `spec-reference-chain.md` 7.9 gives to S67 ("it is S46 and S30 with the merchant"); the S47 second assertion is closed by tightening S69, not by a fourth test. Source: spec 7.9; `scenario_owner_test.go`. Seam: `scenario_merchant_reruns_test.go` (delete a function, restore its `scenariosNotBuilt` entry).
- **D701** (memo D2) X1 and X2 dynamic run report-only: they assert conservation, the audit every tick and the books, log `FINDING N-314` if the households reach 0, report the gap and the occupation shares (`absent` where a series has no data), and assert a band only where `bandIn` finds a `value`. Source: STY 8.4.1, FR-REF-024, `bandIn` (`scenario_helpers_test.go`), S61 (`scenario_mig_test.go`). Seam: the `value` lines of `testdata/chains/bands/two.toml` and `three.toml`; a hard skip is one `t.Skip` line per test.
- **D702** (memo D3) Packs: the reruns use the cold `packMerchants(v)`; X1 uses a new `packMerchantsWarm("two")` and X2 `packMerchantsWarm("three")`, i.e. `tradeWarm(v)` plus the transport, information and merchants overlays. No overlay file is edited. Source: probe of the memo (the warm layering builds and runs; the households reach 0 near tick 730 with or without merchants). Seam: `packMerchantsWarm` beside `packMerchants` in `merchants_golden_test.go`.
- **D703** (memo D4) `TestDeterminism_Merchants` has two variants (the real strategy on the cold golden world; `scriptedMerchants`, the only world that forms contracts) run through `detCases(true)`; the invariants get the row `two+merchants` (4 seeds) and `TestInvariants_MerchantsScripted`, via a new `opts` field of `invariantCase` whose zero value is nil. Source: `TestDeterminism_Transport` (`idle` and `shipper`), plan 2-15 Testing section. Seam: one table row; two slice entries.
- **D704** (memo D5) The window term is built: for a role with a merchant block, `L = max(longest Duration, Kmax)`, `Kmax` the longest travel time (`transit.Ticks` at the factor 10 000, `tripExtremes`'s method) over the graph of the block's self-carried modes (`Crew`-limited as FR-MER-003 says; hired modes excluded). It lives in `contract.go` as `merchantTripTicks(c *Config, r *Role) int64`, called from `longestDuration` by one line guarded by `r.Merchant != nil` (a hook of ADR 0010 (a); no logic in `validate.go`). An invalid graph or a mode out of bound returns 0 (the other rules blame the cause first). Source: FR-WEL-009, spec-welfare 2.3.3, spec-merchants 12.3. This reverses plan 2-15 D512; the supervisor has been told. Seam: delete the guarded line; if the supervisor chooses to keep it rejected, Phase 1 swaps its edit to the two spec passages and Phase 3 is not run.
- **D705** (memo D6) The plan lives in `doc/plans/plan-2-26.md` with one row in `doc/documentation_index.md`. Seam: `git rm` both.
- **D706** FR-REF-038 is reworded, not deleted: the dynamic parts of X1 and X2 and the scenarios whose specs exist run, report-only until their bands are set; the skip-with-issue rule stays for scenarios whose spec is unwritten. Source: the requirement's own condition ("until it exists").
- **D707** Scenario tests keep one seed list from `scenarioSeedCount()`, are skipped under `-short`, take the `scenSem` semaphore, print the seed in every message (testing.md 6.2), and write their measures through `newPackScenReport` to `testdata/chains/expected/two.scenarios.txt` and `three.scenarios.txt` (`-update` once, with the testing.md 7.2 explanation in the commit).

## Phases

### Phase 1: documents (no code)

Why first: a test must cite a requirement that says what it asserts (testing.md 4).

Files: `doc/specs/spec-reference-chain.md` (FR-REF-038 text per D706; X1 and X2 paragraphs of 7.4 and the register rows for X1/X2, S30 to S32, S46, S47, S67 to S72 of 7.6 and 7.9: "run report-only until the bands are set"), `doc/specs/spec-merchants.md` 12.3 (built, FR-WEL-009 cited; the "not adopted" sentence removed; same for its decision list row 20), `doc/specs/spec-welfare-and-wages.md` 2.3.3 and open question 6 (no change if D704 stands), `doc/notes.md` (N-084: remove, applied; N-099: remove if D704, else update; fresh `NOTES-TAG`), `doc/roadmap-v1.md` (MER row to "built"), `doc/documentation_index.md` (the plan row, at the end of its group, after reading the file's rules), `scenario_trn_test.go` line 17 and `scenario_s47_test.go` header comments (text only).

Tests: none. Check: the index command at the foot of `documentation_index.md` prints no difference; `go test -run 'TestSpec|TestNotes' .` if such tests exist, else `go vet ./...`.

### Phase 2: shared test helpers

Files: `merchants_golden_test.go` (`packMerchantsWarm(variant)`), `invariants_test.go` (field `opts func() []economysim.Option` on `invariantCase`, used by `runInvariants` only when non-nil; the row `{"two+merchants", packMerchants("two"), 4, 0, ""}`).

Tests (unit): `TestPacks_MerchantsWarmLoads` (the warm pack validates, has the module and the broker role, no warning of the module's six route and share warnings) FR-MER-052, FR-REF-025; `TestInvariantTable_RowsHaveNilOptsByDefault` Guard: the new field does not change an existing row.

### Phase 3: the window term (D704)

Files: `contract.go`, `validate.go` (one guarded line), new `contract_window_internal_test.go`.

Spike first (a half-day, no commit): list the four callers of `effectiveWindow` (`role.go:83`, `validate.go:1216`, `validate.go:1667`, `migration_report.go:64`) and confirm that `NewGraph(cfg)` can be built, or its absence tolerated, at each; if one caller runs before the regions validate, `merchantTripTicks` returns 0 there. If the spike shows the term cannot be placed without logic outside `contract.go`, stop and report: D704 is then B.

Tests: unit `TestEffectiveWindow_MerchantTrip` (table: no block; `Kmax` under, equal to and over `WelfareWindow / WindowCycles`; hired-only modes ignored; two modes, the longest wins; no path; speed out of bound gives 0; the product bounded by `MaxWelfareWindow` still gives `ErrOutOfBounds` at the existing site) FR-WEL-009; `TestMerchantTripTicks_Overflow` Guard (checked arithmetic, no wrap) NFR-MER-005 style. Integration: the byte-equality proof that `testdata/golden/off/`, `testdata/golden/merchants/`, `testdata/golden/all/` and the pinned hash did not move (`TestGolden_OffUnchangedByMerchants`, `TestHashConfig_ReferencePinned`, the merchants and all goldens): with the defaults (30, 2) and the reference trips of 3 to 4 ticks nothing may move; a diff is a defect, not a golden to regenerate. A world test `TestWindow_MerchantRoleOnSnapshot` reads the effective window of the role from the `Snapshot` (FR-WEL-020) with `Kmax` forced above 15.

### Phase 4: determinism and invariants (D703)

Files: new `merchants_determinism_test.go`, `invariants_test.go`.

Tests (integration): `TestDeterminism_Merchants` (variants `merchant` and `scripted`; each: the cases of `detCases(true)` end in the snapshot and state hash of a bare `Step` run, seed 1; the scripted variant also asserts `ContractFormed` was seen, Guard: without it the variant would test nothing) FR-REF-025, FR-MER-057, FR-MER-061; the row `two+merchants` inside `TestInvariants_Packs` FR-REF-028, FR-STB-008; `TestInvariants_MerchantsScripted` (the scripted world, 40 ticks, 4 seeds, every invariant of testing.md 6.3 and the escrow identity each tick) FR-MER-044, FR-LED-035. Both skip under `-short`; the root test names start with `TestDeterminism` and `TestInvariants`, so the `gateFPrefixes` quick run selects them.

### Phase 5: the reruns (D700, D702, D707)

Files: new `scenario_merchant_reruns_test.go`; `scenario_mer_test.go` (S69); `scenario_owner_test.go` (delete the `46` entry; owners text of `{30, 32}`); `testdata/chains/expected/two.scenarios.txt`.

One helper `merRerun(t, seed, mut, ticks)` builds `packMerchants("two")`, applies `mut`, registers `strategy.Options()`, attaches `newTrnObs` and the audit, and steps. Tests (end-to-end, `-short` skipped, exact checks only; the cold pack loses its households, so gaps are reported):

- `TestScenarioS30_WithMerchant`: cells k in {1, 2, 4} (foot speed as the original) x spread {0, 2 500}; asserts lines add up, the books (dispatched = arrived + in the snapshot), the audit every tick; reports dispatched, arrived, mean and largest gap. FR-TRN-028, FR-TRN-049, FR-MER-044.
- `TestScenarioS31_WithMerchant`: `prepared` (shelf life 4) added to the broker's `trades` by the test, mode speed such that the trip is k = 2 and k = 3; asserts that with k = 3 no lot of `prepared` is planned (`MerchantsPlanned.Perishable` above 0 and no `ShipmentDispatched` of it) and with k = 2 the arrival is delivered, arithmetic in the comment: FR-MER-010 sets a lot aside iff k > S - 1 - max(1, OldestAge) = 4 - 1 - 1 = 2 for a lot of age 1, so k = 3 is `Perishable` and k = 2 passes (the originals' delivery rule, arrival 1 + k against 4 - 1, gives the same split). FR-MER-010, FR-TRN-025, FR-TRN-026.
- `TestScenarioS32_WithMerchant`: three elasticities (1, 4, 16) of the carriers' starts, brokers hire the coach; asserts lines add up, books, the haul units redeemed equal the `haul_use` flow, audit; reports early and late range of the haul price. FR-TRN-043, FR-CAR-003, FR-MER-048 (reported only, as S70).
- `TestScenarioS46_MerchantRumouredGap`: the S67 world with `information` lag in {1, 2, 4} (`info_speed`) x spread {0, 1 000, 2 500}; asserts the caps of FR-MER-025 (requests and offers at most the merchants), the audit, and that the stale count is 0 at the largest `MaxSourceAge`; reports planned trips per lag. FR-INF-017, FR-MER-025, FR-MER-024.
- S69 gains the assertion the memo found missing: at `MaxSourceAge` equal to the bound the run reports `planned` and, only if it is above 0, asserts `Stale == 0`; else it logs "cold pack, no trip" (no band invented). FR-MER-009.

The numbers in the expected file are reported with `-update` once, after the diff is read; the commit message gives the testing.md 7.2 explanation.

### Phase 6: X1 and X2 dynamic (D701, D702)

Files: new `scenario_x_test.go`; `testdata/chains/expected/two.scenarios.txt`, `three.scenarios.txt`.

Tests (end-to-end, `-short` skipped, 1 095 ticks, seeds 1 and 2):

- `TestScenarioX1_Arbitrage` on `packMerchantsWarm("two")`: asserts the exact checks (invariants of testing.md 6.3 every tick via the audit, `Escrow` back to the open contracts' sum, the books); reports households at ticks 100, 365, 730, 1 095, contracts formed and settled, the fuel price gap per year (`absent` when a market has no price); asserts `gap <= transport cost + x1.margin_bp` in the last year only if `bandIn` finds `x1.margin_bp`, `x1.worst_bp` and `x1.warmup`. FR-VIA-043 to FR-VIA-045 (read), FR-CAR-003, FR-TRN-005. `FINDING N-314` is logged when the households reach 0.
- `TestScenarioX2_ComparativeAdvantage` on `packMerchantsWarm("three")`: the same exact checks; reports the occupation shares by region against the pooled census (A3) at the end of each year; asserts `x2.share_band_bp` only if set.
- `TestScenarioX_BandsStayUnset` Guard: reads the `x1.*` and `x2.share_band_bp` keys and fails if one has a `value` without a `set_by` line (FR-REF-024), so the report-only state cannot become an unreviewed assertion.

### Phase 7: closure and gates

Files: `doc/roadmap-v1.md` (MER row "built (commit)"), a result paragraph for Phases 2 to 6 appended to this plan (what was decided, the gates run, the proof that the off set and the pin did not move), `doc/notes.md` (a note for the INT cap: "a consumer bounds its own `AddAgent` calls; a future `Integration.MaxExternal` would be the seam", areas INT, CODE, fresh tag).

Gates: `gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; the quick run of testing.md 9.1; R; F with the coverage of the root at least 90% and not lower than before; `go mod tidy` no diff; `lizard -l go -C 10 -a 8 -L 100 -w` on `contract.go` and `validate.go`; `testdata/golden/off/`, `merchants/`, `all/` and the pinned hash byte-equal. `gomu` is not run: no `internal/` package changes. The R and F runs are the supervisor's.

## Catalogue appends

None. No event, sentinel, option, bound, default, config field, checkpoint tag, ledger reason or position is added; `tick.go` and the hash do not change (Phase 3 changes a derived value, not a hashed field).

## Cross-plan contract

Consumes: `merchants_golden_test.go` (`packMerchants`, `scriptedMerchants`, `goldenSeed`), `scenario_trn_test.go` (`newTrnObs`, `trnObs`), `scenario_helpers_test.go` (`bandIn`, `newPackScenReport`, `scenarioSeedCount`, `scenSem`), `determinism_test.go` (`detCases`, `runPack`), `invariants_test.go` (`runInvariants`), `transport.go` (`tripExtremes`, `transit.Ticks`), `NewGraph`. Provides: nothing to another plan. Does not touch plan 8's re-record of `golden/experiment/one.txt` or plan 19's bands.

## Testing

Pyramid (testing.md 2): base, the unit tables of Phases 2 and 3 (`TestEffectiveWindow_MerchantTrip`, `TestMerchantTripTicks_Overflow`, the opts-default guard); middle, the integration tests of Phases 3 and 4 (`TestWindow_MerchantRoleOnSnapshot`, `TestDeterminism_Merchants` x2 variants, `two+merchants`, `TestInvariants_MerchantsScripted`); top, the six end-to-end functions of Phases 5 and 6 plus `TestScenarioX_BandsStayUnset` (cheap). Every behaviour is asserted at the lowest level that can see it: the window arithmetic at unit level, not through a scenario. Requirement map: FR-WEL-009 (Phase 3), FR-REF-025/028, FR-MER-044/057/061 (Phase 4), FR-TRN-025/026/028/043/049, FR-CAR-003, FR-MER-009/010/024/025/048, FR-INF-017 (Phase 5), FR-REF-038, FR-REF-024 (Phase 6). Each new test carries a doc comment with `Requirement:` lines and arithmetic for every literal (testing.md 4, 5). No fuzz test: nothing new parses input. Mutation check by hand (testing.md 9.3): flip the comparison in `merchantTripTicks` and see `TestEffectiveWindow_MerchantTrip` fail; delete the `Perishable` assertion path and see S31 fail.

## Observability

Per `observability-and-logging.md` 9: 1 failure paths: none new (`merchantTripTicks` returns 0, never an error). 2 state transitions: none, no event. 3 decision points: none. 4 loops: none added to the tick; the new code runs at construction. 5 events and metrics: unchanged. 6 attributes: none. 7 levels: no log line added. 8 no error is logged and returned. 9 tests: existing `TestObserver_DoesNotChangeResults` (FR-MER-057) and the determinism variants with an observer cover non-interference. "None, because the work is tests, spec text and a construction-time derivation."

## Security review

N/A: no input, endpoint, permission or trust boundary is added. The one new computation reads a validated configuration and uses checked arithmetic (architecture.md 3.5); a hostile configuration with huge distances or tiny speeds is bounded by `MaxSpeed`, `MaxTravelTicks` and `MaxWelfareWindow`, and the product overflow case has its own test.

## Compatibility and rollout

The window term can change the effective window of a role with a merchant block whose longest trip exceeds `WelfareWindow / WindowCycles` ticks (15 at the defaults). No reference pack has such a trip. A consumer's pack that does gets a longer window (a slower ratio) and may now fail `ErrOutOfBounds` if the window passes `MaxWelfareWindow`: that is the intended validation, and the commit and the changelog line say so. The configuration hash is unchanged. All other work is test and document files, so there is no rollout order beyond Phase 1 first and Phase 3 before any golden is read.

## Risks and open items

1. **Callers of `effectiveWindow` have no graph** (Phase 3 spike). Fallback: D704 becomes option B of the memo (keep it rejected, edit the specs); the other phases do not depend on it.
2. **Run time of X1 and X2** (1 095 ticks, two packs, two seeds): they run in gate F only; they use the semaphore; `TestScenarioX_*` names must start with a prefix of `gateFPrefixes` (`TestScenario`) so the quick run selects them.
3. **Report files** are shared with the transport reruns: new sections are appended under their own scenario names, and a run without `-update` only compares.
4. **A collapsed economy gives no gap series**: reported as absent, never invented (D701). When plan 8 lands a surviving pack, only the bands files and the pack name in `packMerchantsWarm` change.
5. **Reversing plan 2-15 D512** (D704) is the supervisor's to confirm before Phase 3 starts.
6. **Gates not re-run by this plan**: R, F, `gomu`, Lizard for rows INF, MIG, EXT, IND, EXP, CLB, otel (memo section 1); the supervisor's exit gate.

Note proposals (text only): N-084: remove, "X1 and X2 dynamic and the merchant reruns run report-only (plan 2-26)"; N-099: remove if D704 ("applied: `merchantTripTicks`, FR-WEL-009"); new INT note on the `AddAgent` bound (Phase 7).
