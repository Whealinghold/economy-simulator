# Plan: Stage 2, row 12b: transport lanes in the viability solver

**Status: Built (2026-11-11), see the closing record at the end.** Builds Phase 8 of plan 2-12, which was split out and left open (plan 2-12 is in git history: `git show 31e3a05^:doc/plans/plan-2-12.md`). Sequences [spec-viability.md](../specs/spec-viability.md) 7.1, 9.4, 9.6, FR-VIA-043, 044 and [spec-transport.md](../specs/spec-transport.md) 9.4, FR-TRN-041, under the contract of the stage plans: a plan decides nothing; the specs, the ADRs and `doc/rules/` decide, and where this plan and they disagree they win and the plan is corrected. Closes notes N-279, N-281 and N-326. The decisions below (L1 to L7) are the picks of the decision memo of the worker (`DEC-lanes.md`, in the supervisor scratchpad); Phase 1 writes them into the spec before any code, because N-326 says the spec lacks them.

Rule tags read for this plan: architecture=0052d4050fd8b5aa5325ae2e, observability=e7cfbd33b1620a09dc78dff7, testing=120890253bd2f6c110a015d2.

## Scope

**In scope.** `Lane`, `WithLanes`, `MaxLanes` in use, `Config.TransportLanes`; the lane columns and per-(region, commodity) goods rows of `internal/steady`; one programme per weak component of the lane graph; the warm-start table from a lane programme; lane validation; the tests `TestTransportLanes`, `TestSolve_LanesOracle`, `TestSolve_LaneValidation`, `FuzzSolveLanes`; the spec, notes and roadmap edits that close N-279, N-281, N-326.

**Out of scope.**
- `LossBP` from the risk module: `Config.TransportLanes` returns 0 until RSK (N-180 stays open; the seam is `routeLossBP`).
- Lane inputs (the carriers' feed and vehicle wear): labour only, a stated limitation (spec-transport open question 8, N-093(2)).
- The infrastructure `BoostBP` on the transit factor (the factor is a parameter of one helper; the module is later).
- Self-carried modes give no lanes in version 1.
- N-321 (overlay placeholders) and N-322 (P1.3 anchor lag): left open, not lane work.
- `RelocateForTest` and `SetEdgeStatusesForTest`: dropped (L6), no work.
- Routing `WithFreeTrade` through the general form: not done; a possible later change at one call (`sets()`).

## Decisions this plan builds (decided in the memo; reversible at the named seam)

| ID | Pick | Reverse by replacing |
|----|------|----------------------|
| L1 | General lane form beside the collapsed free-trade form; the collapsed form is untouched | `internal/steady/lane.go` and `Model.Lanes`; one `if len(m.Lanes) > 0` in `Build` |
| L2 | One programme per weakly connected component of the lane graph over regions with a group; no lane is autarky | `laneGroups` in `viability_lanes.go` (return one group for the single-LP option) |
| L3 | A region's seed price of a commodity is the dual of its own row; scale is the 6.3 rule over the programme; lane flows are not exported; wants term per region | the one branch in `viability_table.go` (`newRegion`, `scale`) |
| L4 | Config-free rules in `WithLanes`; unknown region or commodity in `SolveSteadyState` after `Validate`, before any solve; `WithLanes` with `WithFreeTrade` is `ErrOptionContext`; a lane to a region with no group is ignored | `validateLanes`, `lanesOption` |
| L5 | `Config.TransportLanes() ([]Lane, error)` in `transport_lanes.go`, from `NewGraph(cfg)` at factor 10 000, `transit.Labour`, cheapest recipe by `Duration/Output` then recipe ID, sorted, bounded by `MaxLanes` | `transport_lanes.go` only |
| L6 | Drop the two test seams | none (nothing references them) |
| L7 | Spec, notes and roadmap edits are Phase 1 and Phase 6 | docs only |

## Assumptions this plan makes (flagged for override)

- A1. Weak, not strong, components (a one-way lane couples the LP). Override: `laneGroups`.
- A2. The unknown-region check happens at the solver entry, not in the option (an option cannot see the configuration). Override: `validateLanes`.
- A3. A lane touching a region with no group is ignored, with no new `Report` field. If the owner wants it reported, that is a Report addition and a spec change.
- A4. `Config.TransportLanes` returns an error and no partial list when a hired mode's haul commodity has no recipe. If `Validate` already rejects that, the branch is a defensive error path with a test.
- A5. File names: `internal/steady/lane.go`, `viability_lanes.go` (VIA continuation file, listed in architecture.md 2.6 and AGENTS.md Layout in Phase 1), `transport_lanes.go` (already listed under TRN).

Plan location: `doc/plans/` was removed from the tree by `31e3a05` (finished plans live in git history). This plan is committed there as the delivery vehicle for the worker task; the supervisor may move or drop the file after the work is done.

## Reuse and boundaries found

- `internal/transit.Labour`, `HaulPerUnit`, `Ticks` (built, tested): the lane arithmetic; no new arithmetic.
- `NewGraph(cfg)` and `Graph.Distance` (`regions.go`): distances at configuration level.
- `boundedOption`, `newSolveOptions`, `ValidationError`, `ErrOutOfBounds`, `ErrDuplicateID`, `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrOptionContext` (`viability.go`, `errors.go`).
- `steady.Build`, `Size`, `read`, `Layout`, `FixedTerm`; `wantsTerm(c, regions)` called per region; `viaModel.model`, `viaSolve.sets/programme/full`; `regionCtx`, `scale` in `viability_table.go`.
- `bounds.go: MaxLanes` exists; `export_test.go` is the home of any test seam.
- Boundaries: `internal/steady` takes and returns primitives and may use `math/big` (architecture.md 3.5, 2.9); no `math/big` type appears in a root signature; the root converts inline; `Lane` is declared once in `viability.go`; no layered package (3.1); the lanes appear only in the VIA and TRN files and `bounds.go` (2.10).

## Phase 1: the spec says it (gate for the rest)

**Why first:** N-326 and plan 2-12 E5 say the spec must answer how the table takes per-region prices and whether a lane set is one programme. Code before the text would decide it in code. Reading `doc/notes.md` is required (AGENTS.md): quote its Notes tag in the first message of this phase and apply or leave open every note tagged VIA, TRN, CFG and ALL.

- `doc/specs/spec-viability.md` 7.1, 9.4, 9.6, 11 and the pointer lines near 1759: delete the "Open" paragraph; write L2 (components), L3 (price, scale, wants, flows), L4 (validation sites and sentinels, the `ErrOptionContext` case, the groupless-region rule). Keep FR-VIA-043, 044; add an FR for the component rule and for 9.6 as a property (new IDs, next free in VIA).
- `doc/specs/spec-transport.md` 9.4: confirm the Go name, the sort order, the error cases (A4), the seam for `LossBP`.
- `doc.go` glossary: extend `Lane` (architecture.md 3.2); `doc/rules/architecture.md` 2.6 and `AGENTS.md` Layout: add `viability_lanes.go` to the VIA continuation list (both lists, same change; fresh RULE-TAG on architecture.md).
- `doc/notes.md`: N-279 and N-326 get an "Update" (settled in Phase 6, removed then), fresh NOTES-TAG.

**Tests:** none (docs). The parameter-owner table of `doc/specs/README.md` needs no row (`Lanes` is a solve option, not a `Config` field); a note in the PR says so.

**Observability and rollback:** none, because docs only; revert the commit.

**Acceptance:** `grep -n "Open (VIA" doc/specs/spec-viability.md` finds nothing; every lane rule has an FR ID that Phases 2 to 5 cite.

## Phase 2: `internal/steady` lane form (L1)

**Why now:** the root cannot test lanes until the primitive solves them; this phase has no root dependency (architecture.md 2.9).

- `internal/steady/program.go`: `Model.Lanes []Lane` (`From, To, Commodity int`, `Labour, LossBP int64`; primitives); `Layout` gains `GoodsRows` (C collapsed, R*C general) and `LabourRow` uses it; `Size`, `Build`, `lambdaColumn`, `activityColumn`, `writeFixed` take the row of (region, commodity) in the general form. `Model.Fixed` in the lane form is indexed `r*C+c`.
- `internal/steady/lane.go` (new): the lane column (`+1` at (From, c), `-(10 000-LossBP)/10 000` at (To, c), `Labour/10 000` in From's labour row, cost 0, no capacity row) and the row map. Exact rationals; `big.Int`/`big.Rat` only.
- `internal/steady/results.go`: `read` slices `Price` as GoodsRows and keeps `Wage`, `Rent`, `Flow` (activities then lanes: the lane columns sit between the activities and lambda, so `Layout.Lambda` is `len(Activities)+len(Lanes)`; lane flows are in `Flow` and not exported by the root).
- Check `components.go`, `cycles.go`, `producible.go`, `sustain.go`, `gap.go` for an assumption of one goods row per commodity; fix only where they break.

**Tests (unit, `internal/steady`):** `TestBuild_LaneColumn` (FR-VIA-043: the entries of one lane by hand, loss 5 000 gives `1/2` at To), `TestSolve_LaneHandLP` (a two-region, one-commodity LP solved by hand: price ratio equals lane cost; the certificate verifies), `TestSolve_LanesOracleModel` (FR-VIA-043, 044: the complete zero-cost lane set and the collapsed `Model` agree on `SurplusBP` and `Lambda` over a set of generated models, seed in the message), `TestSolve_NoLaneIsAutarky` (a model with `Lanes` empty builds the byte-identical programme as before), `TestLane_LabourRowCharged` (a lane of labour 10 000 per 10 000 units uses one household-tick per unit), `FuzzSteadyLanes` (never panics, deterministic, certificate verifies). The `float_guard_test.go` still passes.

**Observability and rollback:** none, because the package logs and emits nothing (architecture.md 11); rollback is deleting `lane.go` and the `Lanes` field (L1 seam).

**Acceptance:** `go test ./internal/steady/...` passes with coverage of the package at least 90%; the Stage 1 tests of the package are unchanged and pass without edits.

## Phase 3: the root solver takes lanes (L2, L3, L4)

**Why now:** needs Phase 2 for the programme and Phase 1 for the rules.

- `viability.go`: `Lane` (declared once; fields as spec 9.4), `WithLanes([]Lane) SolveOption` (copies the slice; config-free rules: self-lane, duplicate, `Labour < 0`, `LossBP` outside 0..9 999, more than `MaxLanes`; `ValidationError` with path `solve_options.lanes[i]`), `solveOptions.lanes`; `WithFreeTrade` and `WithLanes` together: `ErrOptionContext`; the doc comment of `WithFreeTrade` drops "Stage 1 realises only".
- `viability_lanes.go` (new, VIA continuation): `validateLanes(cfg, lanes)` (unknown region, unknown commodity; called in `SolveSteadyState` after `cold.Validate()`), `laneGroups` (weak components over the regions with a group; sorted by first region), the lane list of a group in canonical order (by From, To, Commodity, Labour, LossBP).
- `viability_check.go`: `viaModel.model` builds the lane `steady.Model` for a group with lanes (activities as now; wants term per region via `wantsTerm(c, []RegionID{r})`); `sets()` in `viability.go` returns `laneGroups` when lanes are given. A group of one region is the autarky path unchanged.
- `viability_table.go`: the lane branch in `newRegion` and `scale` (L3); `viability_report.go`: check that `PairRatio` and `Advantage` do not assume one price vector for a lane programme and read the regions' own prices; `Pivots`, `Degenerate`, `Alternative` accumulate as for any programme.

**Tests:** `TestSolve_LanesOracle` (FR-VIA-043, 044: root level, the pack `two` with the complete zero-cost set equals `WithFreeTrade` on `SurplusBP` and on every region's seed prices; no lane equals the autarky result byte for byte; a lane of loss 5 000 halves the delivered column), `TestSolve_LaneValidation` (one case per rule of L4, `errors.Is` on the sentinel; `ErrOutOfBounds` above `MaxLanes` before any work; `WithLanes` plus `WithFreeTrade` is `ErrOptionContext`), `TestSolve_LaneComponents` (lanes {A,B} and C alone: C equals its autarky solve; the groups are what `laneGroups` says), `TestSolve_LanePricesWithinLaneCost` (FR-VIA-043, spec 9.6: for every lane the seed prices satisfy `p[To]*(1-loss) <= p[From] + wage*Labour/10 000` in the table, exact integers with the rounding rule stated), `TestSolve_LaneGrouplessRegion` (a lane to a region with no group is ignored), `FuzzSolveLanes` (testing.md 7.4: random lane lists over a fixture, never panics, deterministic, an error is a sentinel; seeds with `f.Add`; any crash input committed under `testdata/fuzz/FuzzSolveLanes/` with a `Fix:` comment), the Guard test at `viability_solve_internal_test.go:336` is replaced by these. `golden/off` and the pinned configuration hash are re-run and must not move.

**Observability and rollback:** none, because `SolveSteadyState` is pure and logs and emits nothing (spec-viability 7.2); no new log key, event or metric. Rollback: revert the commit; without `WithLanes` the code path is the old one.

**Acceptance:** the oracle, validation and component tests pass; coverage of the root at least 90% (gate F); `testdata/golden/off/` and the pinned hash are unchanged in the diff; the `internal/steady` Stage 1 tests pass unedited, and the only existing root test edited is the Guard test at `viability_solve_internal_test.go:336`, replaced by the lane tests.

## Phase 4: `Config.TransportLanes` (L5)

**Why now:** a consumer needs the list to pass to Phase 3; independent of Phase 3's internals, so it may run beside it after Phase 1.

- `transport_lanes.go` (new, TRN continuation, already in the layout): `func (c Config) TransportLanes() ([]Lane, error)`. Transport off: `nil, nil`. For each hired mode, ordered pair with a path in `NewGraph(c)` and a travel time within `MaxTravelTicks` (the same test as `World.travel`), each commodity: `transit.Labour(mass, bulk, unitMass, unitBulk, span, distance, duration, output)` with the cheapest recipe making the haul commodity (lowest `Duration/Output` at E = 10 000 by cross-multiplication, ties by recipe ID); `LossBP` from `routeLossBP(...)` returning 0 (the RSK seam). Result sorted (From, To, Commodity, Labour), deterministic, at most `MaxLanes` (else `ErrOutOfBounds`). Overflow or no recipe: a wrapped error, no partial list.

**Tests:** `TestTransportLanes` (FR-TRN-041: the lanes of `two` + `trade-two.toml` + `transport-two.toml`; a self-carried mode gives none; transport off gives none; labour equals the hand figure of `doc/specs/models/trn/worked.py` recorded in the test comment with the arithmetic; the tie by recipe ID; a commodity with Mass and Bulk 0 gives labour 0), `TestTransportLanes_Deterministic` (two calls equal, no map order), `TestTransportLanes_Errors` (no recipe for the haul commodity; above `MaxLanes`), `TestTransportLanes_FeedsSolver` (integration: the list passed to `WithLanes` solves and the table passes `Validate`; no world is built, so the check stays independent of the module).

**Observability and rollback:** none, because it is a pure function; revert the file.

**Acceptance:** the four tests pass; `transport_lanes.go` Lizard clean (`lizard -l go -C 10 -a 8 -L 100 -w`).

## Phase 5: integration, determinism and gates

- Pack check (`viability_pack_test.go` style): `two` with the derived lanes: the table validates and a world builds from it and runs a short scripted run; seeds reported in messages (testing.md 6.2). Determinism of the table across two calls and two seeds of the option order (lane order in the option does not change the result: sorted internally).
- Exit gates: `gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; gate R; gate F with the thresholds of testing.md 9.1 (90% root and `internal/steady`); `go mod tidy` no diff; Lizard on the changed files; `gomu` only in Docker, `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4 --workers 4`, on `internal/steady/lane.go`, `viability_lanes.go`, `transport_lanes.go`, equivalents listed with a reason.

**Tests:** as above; mutation survivors in the new files are killed or listed.

**Observability and rollback:** none; nothing runs in a tick.

**Acceptance:** all gates green; the PR text has the `Testing` entry (requirement IDs FR-VIA-043, 044, the new component FR, FR-TRN-041 and the tests above) and the `Observability` entry ("None, because the solver and `TransportLanes` are pure and emit nothing").

## Phase 6: closure

- `doc/notes.md`: remove N-279, N-326; N-281 gets its last update (`MaxLanes` in use, all three bounds built) and is removed; the commit message says applied, in which spec and code; update the index; fresh NOTES-TAG.
- `doc/roadmap-v1.md` row 12: "built except the lanes" becomes "built"; `doc/documentation_index.md` if a doc file changed.
- Record that `RelocateForTest` and `SetEdgeStatusesForTest` are dropped (L6): one line in the roadmap row or the PR text; no code.

**Tests:** the registry and catalogue tests still pass (no row added or removed).

**Observability and rollback:** none; docs only.

**Acceptance:** `grep -rn "N-279\|N-326" doc/` finds only history-free text (no open entry); N-281 gone.

## Security review

The lane list is caller data to a pure function. Worst case is resource exhaustion or a wrong answer, not exposure: (1) size: `MaxLanes` bounds the list before any work; the row count R*C is gated by the existing `MaxViabilityCells` and `MaxViabilityPivots` (a skipped programme is reported as today); (2) hostile input: negative labour, loss at or above 10 000 (a division by zero or negative gain), self-lanes, duplicates, unknown IDs are all refused with sentinels (L4); the fuzz test covers them; (3) overflow: `internal/steady` uses `big`, and each conversion back to `int64` is checked as today; `transit.Labour` returns `ok` on overflow; (4) determinism: no map order, lanes sorted; (5) no network, file, goroutine, global state or randomness is added. There is no authentication surface (library).

## Compatibility and rollout

Additive: `Lane`, `WithLanes`, `Config.TransportLanes` are new exported names (architecture.md 6.6: a minor addition; `MaxLanes` exported already). `WithFreeTrade`, the autarky path, the collapsed form, the table of every existing pack, `golden/off` and the pinned hash do not change; the proof is the unedited Stage 1 tests (but for that one Guard test) and the unmoved goldens (testing.md 7.1.1). No `Config` field, so the configuration hash and the checkpoint format do not move. A consumer that wants lanes calls `cfg.TransportLanes()` and passes the result; nothing calls it implicitly (`NewWorld` never solves). Rollback is per phase (the seams of the decisions table).

## Risks

| ID | Risk | Owner |
|----|------|-------|
| O1 | Row indexing in `internal/steady` (`Commodities` assumed as the goods row count in `LabourRow`, `read`, `lambdaColumn`): a missed use gives a wrong price silently. Mitigation: the no-lane byte-identical test and the model-level oracle. | Phase 2 |
| O2 | R*C rows exceed the cell gate on a large realm: the programme is skipped. Mitigation: the existing skipped path and report; no new behaviour. | Phase 3 |
| O3 | Plan 5 edits `internal/steady` too (N-277); rebase before Phase 2. | Phase 2 |
| O4 | The new FR IDs and the VIA spec edit need the owner of VIA; if the owner picks one programme (L2 option A), only `laneGroups` changes. | Phase 1 |

## Requirement traceability

| Requirement | Test |
|-------------|------|
| FR-VIA-043 | `TestBuild_LaneColumn`, `TestSolve_LaneHandLP`, `TestSolve_LanesOracle`, `TestSolve_NoLaneIsAutarky`, `TestSolve_LanePricesWithinLaneCost` |
| FR-VIA-044 | `TestSolve_LanesOracleModel`, `TestSolve_LanesOracle` |
| new FR (components, Phase 1) | `TestSolve_LaneComponents`, `TestSolve_LaneGrouplessRegion` |
| lane validation (spec-viability 9.4) | `TestSolve_LaneValidation`, `FuzzSolveLanes` |
| FR-TRN-041 | `TestLabour_Exact` (built), `TestTransportLanes`, `TestTransportLanes_Errors`, `TestTransportLanes_FeedsSolver` |

## Closing record (2026-11-11)

Built in `stage-2-done-lanes`: Phase 1 `0d96f3e`; Phase 2 `bc2ff62`; Phase 3 `5d26555`, `102b636`; Phase 4 `Config.TransportLanes` in `transport_lanes.go` (`7cdc2ec`); Phase 5 the pack test `TestTransportLanes_PackTwoWorld`; Phase 6 this closure.

- **Notes:** N-279, N-281 and N-326 are applied and removed: `spec-viability.md` 7.1, 9.4, 9.6 (FR-VIA-061 to 063), `spec-transport.md` 9.4, code in `internal/steady/lane.go`, `viability_lanes.go`, `transport_lanes.go`. N-180 (the risk loss) stays open at the seam `routeLossBP`.
- **Roadmap:** the row of `transport.go` and `internal/transit` reads built.
- **Dropped (L6):** `RelocateForTest` and `SetEdgeStatusesForTest`; nothing referenced them.
- **Goldens:** no golden file or pinned hash was touched or regenerated.
- **Tests added in Phase 4 and 5:** `TestTransportLanes`, `TestTransportLanes_Deterministic`, `TestTransportLanes_Errors`, `TestTransportLanes_FeedsSolver`, `TestTransportLanes_PackTwoWorld`, `TestCheapestHaulRecipe`, `TestTooManyLanes` (FR-TRN-041, FR-VIA-043, FR-VIA-061). The `MaxLanes` bound is tested through `tooManyLanes(n, limit)`.
- **Mutation:** `gomu` runs only on `internal/*`; `transport_lanes.go` is in the root and was not mutated.
