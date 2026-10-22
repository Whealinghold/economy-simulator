# Plan: Stage 2, row 3: REG, the region graph, the reference-price override, relocation and edge status

**Status: plan written (2026-10-22), not audited. Written in parallel with plans 2-1 to 2-24 against `plan-2-0.md`; the names it consumes from plan-2-2 follow that plan's Provides (Risks, R1).**
Sequences the second row of Stage 2 of [roadmap-v1.md](../roadmap-v1.md), "`internal/graph` and `regions.go` (the graph, distances, components, overrides, `Relocate`, the edge status)", into ten phases. It implements [spec-regions.md](../specs/spec-regions.md) (REG) FR-REG-001 to FR-REG-060 and NFR-REG-001 to NFR-REG-004, and the REG halves of FR-LED-054 (`spec-ledger.md` 7.4), FR-CFG-067 and FR-CFG-105 (`spec-config.md` 8.2).

Rule tags read for this plan (the three files in full, as `AGENTS.md` asks): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1. They equal the tags of `plan-2-0.md`.

Also read: `AGENTS.md`; `doc/roadmap-v1.md` Stage 1 and 2; `spec-regions.md` in full; the passages of `spec-world.md` (3.5 to 3.7, 5.2, 6 P1.3 and P1.5, ruling C16), `spec-ledger.md` (3.5, 7.4, 16.1), `spec-config.md` (3.1, 6.2.1, 8.2), `spec-checkpoint.md` (4.2, 5.3) and `spec-observability.md` (4.1, 5.3.7, 9.5.2) that REG touches; the Stage 1 code named in each phase.
Notes (the index and the open notes tagged REG or ALL, not the whole file; no note is edited): N-150, N-175, N-204 (REG) and N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217 (ALL); also N-075, N-090, N-218, N-243 and N-295, which the work touches. For each: N-016 apply (every new field has a README row already: Phase 3 deletes the `notBuilt` entries); N-017 apply (terms in Phase 1); N-027 apply (every bound is labelled provisional, no default is added); N-103 apply (this plan edits one file); N-126 apply (the two events have catalogue rows already; the `Warn`s have rows, Phase 1 checks 7.3 of `spec-observability.md`); N-128 leave open, standing (REG hosts no per-agent field); N-163 apply (the word "treasury" is not used); N-217 apply (Phase 8 and 10 prove `testdata/golden/off/` and the pinned hash do not move); N-075 apply for REG's sentinels (Phase 3); N-090 leave open, with this learned: the registry of obligations is built here (D270) and each module appends its name, `has_store`, `open_hire` and `in_traineeship` belong to Stage 3; N-218 leave open, with this learned: ids 8 and 9 are now used (D265); N-243 apply (Lizard on the new files); N-295 leave open, with this learned: `three` loads with `region_isolated` warnings until its trade overlay (Phase 9); N-150 and N-175 leave open (EMP and CAP, Stage 3); N-204 leave open (INT and PLC write the edge slot; REG provides the internal form, D272).
Decision IDs: D260 to D279 (this plan owns the block). Ships in two PRs: PR A = Phases 1 to 4 (graph, configuration, cutover); PR B = Phases 5 to 10 (relocation, edge status, snapshot and checkpoint, packs, gate).

## Scope

**Builds.**

| Item | File or package |
| --- | --- |
| The shortest-path tables, next hop, strongly connected components, effective length, over primitive slices | `internal/graph/graph.go` |
| `Edge`, `EdgeID`, `RegionPrice`, `Hop`, `EdgeView`, `EdgeStatus`, `Graph`, `NewGraph`, `World.Graph`, `World.ReferencePrice`; the world's own region state; the graph rules and the five warnings; the effective reference price | `regions.go` |
| `Config.Edges`, `Config.RegionPrices`, the bounds `MaxDistance`, `MaxEdges`, `MaxEdgeFactorBP`, `MaxRegionPrices`, the sentinels, the hash ids 8 and 9, the validation phases | catalogue files (table below) |
| `Ledger.Relocate` and the sparse relocation totals | `internal/ledger/relocate.go` |
| `relocate`, `moveAgent`, `RelocationRefusal`, `RegionBoundError`, the `region_bound` table and the fan-in `relocationRefusal`, `AgentRelocated` | `regions_relocate.go`, `fanin.go`, `event.go` |
| The internal `setEdgeStatuses(changes, cause)` and `EdgeStatusChanged` | `regions_status.go`, `event.go` |
| `Snapshot.Edges`, `Snapshot.Components`, checkpoint tag 11, the log keys and the construction summary | `snapshot.go`, `regions_status.go`, `internal/logkeys` |
| The loader keys `[region.<r>.edge.<to>]` and `[region.<r>.reference_price]` | `config/mapping.go` |
| The edge part of the trade overlay for `two`, `three` and `five` | `testdata/chains/overlay/trade-two.toml`, `trade-three.toml`, `trade-five.toml` |
| Scenarios S28 (REG-A1, closure and reopening of the middle link; the public-call leg is S80, plan 17, and no other plan builds S28 or S29) and S29 (REG-A2, a town as one region against a town and two hamlets) | `scenario_s28_test.go`, `scenario_s29_test.go` |

**Leaves out**, with the owner: the public `SetEdgeStatuses` and `Relocate` of the world, the cause `policy` and the closed-road shock (INT, plan 17; PLC, Stage 3); the use of the causes `transport` and `migration`, `open_shipment`, `on_journey`, `open_contract` and the calls of `moveAgent` at P1.3 and P1.5 (TRN plan 12, MIG plan 14, MER plan 15); `ErrOutsideNode`, `ReasonOutsideNode` and refusal 5a (EXT, plan 16 declares them); `has_store`, `open_hire`, `in_traineeship` (Stage 3); travel time, risk, information (TRN, RSK, INF); the metric `economysim.household.relocations` (plan 22); the checkpoint encoding itself, the frame, `Checkpoint` and `RestoreWorld` (plan 2); `Region.Label` and `World.DistanceLabel`, which exist.

**Already built, only tested here:** the estate account and pseudo-agent of each region in ascending `RegionID` (`estates.go`, FR-REG-005, 006), `RegionID`, `NoRegion`, `ErrNoRegions` and the region ID rules, `MaxRegions`, `MaxLabelBytes`, the label rule. REG owns **no position of `tick.go`** (P1.3 is TRN's, P1.5 MIG's, P0.3a PLC's): `positionList` and `notBuiltPositions` do not change (D261).

## Decisions

Each follows a spec, a rule or a note; the plan decides nothing else.

- **D260.** REG owns `regions.go` and two continuation files declared now, because one file would hold about 20 graph methods, the rules, five warnings, relocation, status and tag 11 against the Lizard limits of N-243: `regions.go` (types, `Graph`, config rules, warnings, `refPrice`, the world's tables), `regions_relocate.go` (`relocate`, `moveAgent`, `RelocationRefusal`, `RegionBoundError`, the obligation table) and `regions_status.go` (`EdgeStatus`, `setEdgeStatuses`, snapshot rows, tag 11 `saveRegions`/`loadRegions`). Each `_test.go` carries the REG `Requirement:` lines (architecture.md 2.6). The spec's table 18 puts `relocate` and `RelocationRefusal` in `world.go`; the one-area-per-file rule wins (owed edits E1, E1b).
- **D261.** No tick position, no `Modules` pointer, no module-alone golden: REG is core. The proof of non-interference is `testdata/golden/off/` and the pinned hash unchanged (architecture.md 2.10.5, 2.10.6; spec-regions.md 2 row 11).
- **D262.** `internal/graph` receives the region count and the open edges with their effective lengths as primitive slices and returns tables; it knows no region name (NFR-REG-004, architecture.md 2.9). The root builds the lengths and converts inline at the call. One root function builds a `Graph` for `NewGraph`, `World.Graph` and the world's own tables (spec-regions.md 4.3.2, 4.4.5).
- **D263.** The tables are defined by result (4.3.1): any exact algorithm. Chosen: one Dijkstra per source with a hand-written binary heap keyed `(distance, RegionID)` (no `container/heap`, no allocation per push), next hop by scanning each source's out-edges in ascending order, components from the distance table in ascending smallest-member order (4.4.2). No map, no unstable sort (architecture.md 7.5, 7.6).
- **D264.** The effective length of an open edge is `max(1, MulDivCeil(base, factorBP, 10000))` with `internal/checked`; every path sum uses `checked.Add` and a failure is a returned error (4.2.1, 7.2.1). The bounds make it unreachable; the largest case is a test.
- **D265.** Hash: `Config.Edges` is id 8 and `Config.RegionPrices` id 9, written by `encodeList` between ids 7 and 10 (no write when empty); `Edge` has `From` 1, `To` 2, `Distance` 3; `RegionPrice` has `Region` 1, `Commodity` 2, `Price` 3 (spec-config.md 8.2.2, N-218; the row for the two structs is owed there, Phase 1). Canonical order: edges by `(From, To)`, prices by `(Region, Commodity)`, both compared as strings, which is `RegionID` order for a valid configuration.
- **D266.** The effective reference price has two readers and one rule. `Config.refPrice(r, c)` (binary search over the sorted `RegionPrices`, then the commodity's price) serves the validator and the solver, which have no world; `World` holds the dense array `refPrice[r*nc+c]`, resolved once at construction, behind `World.ReferencePrice` (5.2.4, 5.3.5, FR-REG-036). Every read of `Commodity.ReferencePrice` in Stage 1 code moves to one of the two (Phase 4 lists them); a warm-start seed price replaces the seed only (FR-REG-038).
- **D267.** `Ledger.Relocate(id, region int) error` checks unknown account, closed, not of kind Agent, region out of range, same region, open reservation or earmark, in the order of 6.1.3 (the root checks the region range first), and changes `region` only. The relocation totals are a sparse slice of `(from, to, commodity, total)` rows kept sorted by `(from, to, thing)` by insertion (money first), cleared by `ClearTotals`, never in the audit identities (a relocation moves nothing between accounts, 6.3). A dense `regions^2 * commodities` table is rejected: 10^6 cells per commodity at the bound.
- **D268.** There is one relocation path (architecture.md 2.10.4, 13). `moveAgent(a, to, cause)` does the ledger `Relocate`, the counters, the event and the `Debug` line. `relocate(id, to, cause)` is `RelocationRefusal` then `moveAgent`. TRN and MIG call `RelocationRefusal` with their own `skip` and then `moveAgent`; none calls the ledger directly (6.6, ruling C16).
- **D269.** `RelocationRefusal` maps the agent table, not the account: an ID below the region count is an estate pseudo-agent and is `not_relocatable` (its record is the zero agent and holds no account, `estates.go`); an ID at or above `state.nextAgent`, or negative, is `unknown_agent`; a removed agent (a zero record, `forget`) is `closed_account`. The counters `live.inRole` and `state.hadAgent` of the (region, role) cell are moved by `moveAgent` as `execSwitch` moves them for a role change (`phase_exit.go`).
- **D270.** The registered `region_bound` obligations are a table built at construction by the modules that are on (`w.bounds`, one `*RegionBoundError` per name, created once), so the refusal path allocates nothing (FR-REG-059). The fan-in `relocationRefusal(a *agent, skip []string) (*RegionBoundError, bool)` is a fixed-order sequence of guarded calls; empty in Stage 2 until TRN, MIG and MER append. `RegionBoundError` is a public typed error whose `Is` matches `ErrRegionBound` (architecture.md 8.4).
- **D271.** `skip` matches a name or an instance key. The textual form of a key is not in the spec; the code uses one helper, `skipHas(skip, name, key)`, that parses without allocating. The form is fixed in Phase 1 (owed edit, proposal `name:number`); no test names a form before then.
- **D272.** The edge status is a slice indexed by `EdgeID`. `setEdgeStatuses(changes, cause)` validates every change, applies all or none, rebuilds the world's tables once, then emits `EdgeStatusChanged` per changed edge in ascending `EdgeID` (7.3, 4.4.4). The public `SetEdgeStatuses` is INT's, over this function. A zero `EdgeStatus` is invalid (`FactorBP` 0 is below 1): the default is `FactorBP` 10 000.
- **D273.** The warnings live in `regionWarnings(c *Config) []Warning`, called from `warnings` in `validate.go`; `edge_not_shortest` reads the graph with every edge open at its base distance. A world of one region and no edge draws none of the first three (4.5.2).
- **D274.** `region_without_agents` is raised for a region with no group. The exception for an outside node (9.2) is a marked line that EXT fills (plan 16); REG has no outside node to exempt.
- **D275.** `Snapshot.Edges []EdgeRow` is appended (the non-default statuses, by `EdgeID`). `Snapshot.Components` is **0 when the world has one region** and the true component count otherwise (so `three` with no edge is 3): a count of 1 for the one-region reference world would render in the header of every `off` golden and move it (architecture.md 2.10.5, FR-OBS-091). Every `off` golden is the one-region pack `one`. Owed edit to `spec-observability.md` 5.3.7 and `spec-regions.md` 10.5 (Phase 1, Risks R2).
- **D276.** Checkpoint tag 11 `Regions` is present in every checkpoint. Content: the non-default edge statuses in ascending `EdgeID`, as (`From`, `To`, `Closed`, `FactorBP`). A restore rebuilds the tables from the configuration and the statuses, and the decoder refuses a row that names no configured edge, is out of order, equals the default, or has `FactorBP` outside `[1, MaxEdgeFactorBP]` (`ErrCheckpointCorrupt`, `spec-checkpoint.md` 5.3).
- **D277.** `AgentRelocated` carries `Money` (the account's balance) and `Units` (the sum of its lots' quantities), read before the move. The totals take the same figures per commodity.
- **D278.** The trade overlay is one file per variant, not one `overlay/trade.toml`: an edge table `[region.centre.edge.east]` in a pack that has no `centre` would create a bare region, which is invalid. The edges are those of `spec-regions.md` 12.4, distances placeholders. Owed edit to `spec-reference-chain.md` 7.6 (Phase 1).
- **D279.** The gate is R and F of plan 1 D229, with no temporary by-package coverage run.

## Phases

### Phase 1: edits owed before code

Goal: the specs, rules and notes are true before the first line. Files: documents only; the supervisor applies them (this plan edits none).

| # | File, section | Edit |
| --- | --- | --- |
| E1 | `spec-regions.md` 18 | `relocate`, `RelocationRefusal` and the agent lists: home `regions_relocate.go`, not `world.go` (D260) |
| E1b | `architecture.md` 2.6 and `AGENTS.md` Layout (same files in both lists; fresh rule tag on edit) | add `regions_relocate.go` and `regions_status.go` as REG continuation files beside `regions.go` (D260) |
| E2 | `spec-regions.md` 6.1.6 | state the textual form of an instance key in `skip` (D271); the proposal is `<obligation>:<number>`, a name alone matches every instance |
| E3 | `spec-regions.md` 10.5; `spec-observability.md` 5.3.7 | `Components` is 0 when the world has one region, else the count of the graph as it is now (D275) |
| E4 | `spec-config.md` 8.2.2 | add "`Edge` has `From` 1, `To` 2, `Distance` 3; `RegionPrice` has `Region` 1, `Commodity` 2, `Price` 3" |
| E5 | `spec-config.md` 7, line 1375 (N-075) | see E11; `ErrOutsideNode` is EXT's |
| E6 | `spec-checkpoint.md` 4.2 tag 11 | say that the rows are (`From`, `To`, `Closed`, `FactorBP`) in ascending `EdgeID` as `spec-regions.md` 11 does, and name the content of the pending relocation totals in tag 2 (a sorted list of (`from`, `to`, thing, total)) |
| E7 | `spec-reference-chain.md` 7.6 and `spec-regions.md` 12.4 | the edge part of the trade overlay is `overlay/trade-two.toml`, `trade-three.toml`, `trade-five.toml` (D278) |
| E8 | `doc.go` glossary (architecture.md 3.2) | add Edge, Route, Distance, Component, Edge status, Relocation and `Obligation (region-bound)` (an open fact that makes relocation refuse: `w.bounds`, `region_bound`) before any code uses them (`Region` is there); keep `Edge` as the region edge, and plan 21's "grid edge" is that term qualified, not a second `Edge` (`TestGlossary_Vocabulary` fails on a duplicate). `TestGlossary_DeclaredTypesNamed` also needs the exported types named: `Edge`, `EdgeID`, `EdgeRow`, `EdgeStatus`, `EdgeView`, `Graph`, `Hop`, `RegionPrice`, `RegionBoundError`, `RelocationRefusal`, `AgentRelocated`, `EdgeStatusChanged` |
| E10 | `spec-regions.md` FR-REG-020 | state that the enumeration test is exhaustive up to three regions and sampled to seven (Phase 2) |
| E11 | `spec-config.md` 7, line 1375 (N-075 mapping sentence) | add `ErrClosedAccount` to the REG sentinels mapped; reconcile with `spec-regions.md` 9.1 ("maps to the same value"): the mapping is by `errors.Is` (R6) |
| E13 | `spec-reference-chain.md` 7.6 | S28 and S29 are built by plan 3 (REG-A1, REG-A2); plan 12's rename of its own S28 to S30 to S30 to S32 stands |
| E9 | `spec-observability.md` 7.3 | check that the `Warn`s are rows (the five REG rules are lines of the "configuration warning" row); no new `Warn` |

Notes to append: R6 and R7 of Risks. Exit check: the supervisor confirms E1 to E8 and E10 to E13 are made; E2 and E3 are the two that change tests.

### Phase 2: `internal/graph`

Goal: the algorithms, tested alone. Files: `internal/graph/graph.go` (stdlib and `internal/checked`), `graph_test.go`, `fuzz_test.go`, `testdata/fuzz/FuzzBuild/`.

API (primitive; no region name): `type Edge struct{ From, To int; Length int64 }`; `Effective(base, factorBP int64) (int64, bool)`; `Build(n int, open []Edge) (*Tables, error)` with `open` sorted by `(From, To)`; methods `Dist(a, b int) (int64, bool)`, `Next(a, b int) (int, bool)`, `Comp(r int) (int, bool)`, `Comps() int`, `AppendPath(dst []int, a, b int) ([]int, bool)`, `AppendWithin(dst []Hop, a int, limit int64) []Hop`. Out of range gives `(0, false)`.

Tests (each with `Requirement:`):

- `TestBuild_MatchesEnumeration` over every directed graph of 1 to 3 regions with lengths 1 to 2 and 3 000 pseudo-random graphs of 4 to 7 regions (a test-local SplitMix64, no `math/rand`), lengths 1 to 6, density about 45%: `Dist`, `Next` and the path equal the shortest distance and the lexicographically smallest shortest path by enumerating every simple path (FR-REG-016, 018, 019, 020). FR-REG-020 says every graph of up to seven regions; this is exhaustive to 3 and sampled above (E10).
- `TestBuild_TieBreakDependsOnIDsOnly`: the tie of example C; relabelling so `town` sorts first changes the path; permuting the edge list does not (FR-REG-021).
- `TestBuild_Components`: numbering by smallest member, no-edge graph gives `n`, one region gives 1, a one-way pair gives two (FR-REG-015, 027).
- `TestBuild_WorkedExamples`: the tables of 12.1 (A, with the closure and the factor of 15 000), 12.2 (B, with and without `orbital > surface`) and 12.3 (C) equal the spec's (FR-REG-016, 017, 022, 049).
- `TestEffective_Rounding`: `Base` 30 at 15 000 is 45, at 10 000 is 30, `Base` 7 at 10 001 is 8, `Base` 1 at 1 is 1; a product beyond `int64` is reported (FR-REG-049, 029).
- `TestBuild_LargestCaseDoesNotOverflow`: a chain of 999 edges of `MaxDistance * MaxEdgeFactorBP / 10000` sums below 10^13 without error (FR-REG-029). `TestBuild_OverflowIsReported` on lengths near `int64` (Guard: the error path is the only checked one).
- `TestWithin_Order`, `TestQueries_OutOfRange` (FR-REG-024, 025), `TestAppendPath_FreshSlice` (FR-REG-026 at this level).
- `FuzzBuild`: never panics for any edge list and query; the triangle inequality holds; a path has length `Dist`; each consecutive pair of a path is an open edge (NFR-REG-003). Seeds from the worked examples.
- `BenchmarkBuild` with `ReportAllocs` for 100 regions and 400 edges and for the bound (NFR-REG-002, a report, N-015).

Exit: `gofmt`, `vet`, `staticcheck`, `go test -short -race ./internal/graph`, coverage 90%, Lizard clean, `gomu` in Docker (`--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS=4`, `--workers 4`) with the surviving mutants triaged.

### Phase 3: the configuration surface and validation

Goal: a `Config` can carry edges and overrides, they hash, validate, load and warn. Files: `config.go`, `bounds.go`, `errors.go`, `hash.go`, `validate.go`, `regions.go` (types, `validateRegionGraph`, `regionWarnings`, `refPrice`), `config/mapping.go`, `registry_test.go`, the tests below.

1. `config.go`: `Edge`, `RegionPrice` and `Config.Edges`, `Config.RegionPrices` appended between `Regions` and `Groups` in the declaration, so the field order stays the id order (id 8 and 9 reserved; the struct comment says so); `canonical()` sorts both (`sortedBy`) and copies them (deep, 12.4).
2. `bounds.go`: `MaxDistance` 10^9, `MaxEdges` `20 * MaxRegions`, `MaxEdgeFactorBP` 100 000, `MaxRegionPrices` `MaxRegions * MaxCommodities` (`MaxLabelBytes` exists).
3. `errors.go`, appended: `ErrSelfEdge`, `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrClosedAccount`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation`, `ErrRegionBound`, and `RegionBoundError`.
4. `hash.go`: `encodeList(e, 8, c.Edges, pairsEdge)` and `encodeList(e, 9, c.RegionPrices, pairsRegionPrice)` after id 7.
5. `validate.go`: `counts` adds the two `ErrTooMany` rows; `fields` calls `v.edgeRules()` (distance in `[1, MaxDistance]`, price in `[1, MaxPrice]`, `ErrOutOfBounds` with the path `edges[i].distance`, `region_prices[i].price`); the cross-section phase calls the graph rules (`ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrSelfEdge`, `ErrDuplicateID`). The three rules that read the reference price move to `refPrice(region, commodity)`: the resolution floor (`validate.go` around 1347, per region), `baselineCost` (around 1385, takes the region so `ErrBasketMismatch` uses the region's prices) and the warm-start band (around 1247).
6. `NewGraph(cfg)` runs the graph rules alone (FR-REG-030) and returns the first error as a `*ValidationError`.
7. `config/mapping.go`: in `region()`, `t.entities("edge", …)` appends `Edge{From: id, To: to, Distance}` with `distance` required, and `t.sub("reference_price")` appends one `RegionPrice` per sorted commodity key; unknown keys fall to the existing `ErrUnknownKey` and a non-integer to `ErrNotInteger` (FR-REG-054).
8. `registry_test.go`: delete `"Config.Edges"`, `"Config.RegionPrices"`, `"Edge"`, `"RegionPrice"` from `notBuilt`.

Tests: `TestValidate_EdgeRules` (one failing case per rule of 9.1, each breaking only that rule; FR-REG-001, 009 to 013, 033, 035); `TestValidate_RegionPriceRules` (FR-REG-033, 035); `TestValidate_EffectivePriceFloorAndBasket` (example A: `middle` without its basket is `ErrBasketMismatch`, an override below the floor is `ErrOutOfBounds`; FR-REG-037); `TestWarnings_RegionRules` (the five rules, `region_isolated` for each region of a two-region no-edge config, `edge_not_shortest` for the 90 track, none for a one-region world, `override_equals_default`, `region_without_agents`; FR-REG-007, 031, 034); `TestCanonical_EdgesAndPricesSorted` (FR-REG-002, 014, 035); `TestHashConfig_EdgesAndPrices` (changing a distance or a price changes the hash, a label does not, listing order does not, FR-REG-053; `TestHashConfig_ReferencePinned` does not move and `TestHashConfig_EdgesOffLeavesNoTrace` shows nil and empty equal; a second pinned vector for a three-region configuration with edges, recorded once); `TestLoad_EdgesAndPrices` and `TestLoad_FileAndGoAgree` for the keys, with `ErrUnknownKey` and `ErrNotInteger` cases and the same seeds in `config/fuzz_test.go`; `FuzzValidateGraph` (never panics on any edge list, NFR-REG-003); `TestSentinels_Text` rows in `errors_test.go`; `TestBounds_Constants` (FR-REG-055).

Exit: `gofmt -l .` empty, `go test -short -race . ./config`, `TestHashConfig_ReferencePinned` and the registry test green.

### Phase 4: the Graph value, the world's tables and the price cutover

Goal: queries answered, and the effective price used everywhere. Files: `regions.go`, `construct.go`, `world.go` (one field), and the read sites below.

1. `Graph`, `Hop`, `EdgeView`, `EdgeID` and the methods of 4.3.2, thin over `internal/graph`: `NewGraph` and `World.Graph()` build fresh slices; `Out`, `In`, `Within`, `Path` return fresh slices; `Len`, `Name`, `Lookup`, `Edges`, `Edge`, `EdgeAt`, `Distance`, `NextHop`, `Component`, `Components` allocate nothing (`Lookup` and `Edge` binary search the sorted slices, no map).
2. The world's own state, `regionState` (field `w.geo`): the statuses, the effective lengths and the tables, built in `builder.finish` (`construct.go`) after validation and read by TRN, INF, MIG, MER and EXT through unexported methods `distance(from, to RegionID) (int64, bool)` (the name plans 12 and 13 consume), `next`, `edge`, `out`. Estates and markets are not touched.
3. `World.ReferencePrice(r, c)` over the dense array, 0 for an index outside range (no error result, so no guard; FR-WLD-110 is tested for it and for `Graph` on a failed world).
4. The cutover, each read moved to `refPrice` (config level) or `w.refPrice` (world): `construct.go` `seedPrices` (line 207) and `seedBeliefs` (280); `strategy_view.go` (270); `estates.go` (339); `phase_exit.go` (502); `phase_outcomes.go` (125, 324); `snapshot.go` (400); `viability_table.go` `seedPrice` (243, takes the region) and the drift (274). With no override every value is the commodity's, so no run changes. Stay on the commodity price: `validate.go` 1446 (the yield-bound closure `price`, a configuration-level bound), 2106 (`tool_unaffordable`, a start-money check) and 2126 (`listCost`, used at 2090); `spec-goods.md` 6.3 lists no region use for them, and Phase 4 confirms each against it and moves any it lists.

Tests: `TestGraph_QueriesOnPacks` (`Name` and `Lookup` invert each other for every region, FR-REG-004; the shape of each pack: `one`, `second` one component and no warning; `two`, `three`, `five` with the trade overlays are one component, `five` East to West is 200 through `centre`; FR-REG-015, 016); `TestGraph_OutInOrder` (`Out` and `In` in ascending `RegionID`, closed edges absent, FR-REG-023, 049), `TestGraph_IndependentOfWorld` (a `Graph` taken before a status change answers the same after, a list it returned can be changed, a later `Graph` reflects the change; FR-REG-026, 028, 030); `TestGraph_NoAllocation` with `AllocsPerRun` on the ten non-allocating methods and on `World.ReferencePrice` (NFR-REG-001); `TestReferencePrice_Override` (a `RegionPrices` row changes the seed price, the belief band and the estate's first price in that region and nowhere else; a warm-start seed replaces only the seed; FR-REG-032, 036, 038) and `TestReferencePrice_NoOverrideIsCommodityPrice`; the existing tests of the moved sites must pass unchanged. `TestEstates_PseudoAgentIDsAndAccountOrder` and `TestEstates_ExistWhetherOrNotModuleOn` (FR-REG-005, 006; internal, reading `w.estates`), `TestRegions_SetIsFixed` (FR-REG-008), `TestRegions_IterationOrder` (FR-REG-003, on the snapshot rows and the clearing order of a three-region pack). `TestRegions_AgentLists` derives the agents of a region in ascending ID (FR-REG-045 half).

Exit: golden `off` and pinned hash unchanged; `go test -short -race .`.

### Phase 5: `Ledger.Relocate` and the relocation totals

Goal: the ledger-level operation, alone. Files: `internal/ledger/relocate.go`, `ledger.go` (the totals field and `ClearTotals`), `errors.go` (`ErrSameRegion`, `ErrNotRelocatable` as the ledger's own; `ErrOpenReservation` reused, its comment widened), `relocate_test.go`.

`CheckRelocate(id, region int) error` is the dry run (no change, no allocation) and `Relocate(id, region int) error` (D267) calls it first; they refuse unknown account, closed, kind other than Agent, region out of range, same region, reservation or earmark; on success sets `region`, adds to the totals the balance and each lot's quantity under `(from, to)`, and changes no balance, lot, flow total or boundary account. `AppendRelocations(dst []Moved) []Moved` returns the rows in ascending `from`, `to`, money before commodities in commodity order; `ClearTotals` clears them; `RestoreRelocations(rows []Moved) error` validates order and range for plan 2's decoder (D267, E6). `Units(id int) (int64, error)` returns the account's goods for the event.

Tests: `TestRelocate_OnlyRegionChanges` (balances, lots, reservations, the total of money and of each commodity, `Audit` all unchanged; FR-REG-040, FR-LED-054); `TestRelocate_Refusals` table, one case per refusal in order, each leaves the ledger byte-equal (FR-REG-039, 043); `TestRelocate_FlowsFollowTheNewRegion` (a transfer before and after is attributed to the old and then the new region, FR-REG-041); `TestRelocationTotals_OrderAndClear` (FR-REG-042); `TestRelocate_AgeingFollowsRegion` (Guard: `Age` attributes spoilage to the account's region, `ageing.go`); fuzz target `FuzzRelocate` (a random sequence of transfers and relocations keeps `Audit` nil).

Exit: coverage 90%, `gomu` on `internal/ledger/relocate.go` in Docker with the caps, Lizard clean.

### Phase 6: relocation in the world

Goal: `relocate`, `RelocationRefusal`, the registry, the event. Files: `regions_relocate.go`, `fanin.go`, `event.go`, `export_test.go`, `internal/logkeys`.

1. `event.go`, appended: `AgentRelocated{Tick, Agent AgentID, Role RoleID, From, To RegionID, Weight int64, Money Money, Units int64, Cause Cause}`, the `Reason` constants `ReasonUnknownRegion`, `ReasonClosedAccount`, `ReasonNotRelocatable`, `ReasonSameRegion`, `ReasonOpenReservation`, `ReasonRegionBound` (`ReasonUnknownAgent` is reused; `ReasonOutsideNode` is declared by plan 16, EXT) and the `Cause` constants `CauseTransport` and `CauseMigration` (declared here, once, so plans 12 and 14 do not duplicate them; `CauseConsumer` exists). Delete both events from `notBuiltEvents` and `deferredCore` in `event_test.go` and add them to `declaredEvents`.
2. `RelocationRefusal(a AgentID, to RegionID, skip ...string) (string, error)`: the order of 6.1.3 with D269 and a marked line for 5a; returns bare sentinels (no wrapping, so it allocates nothing; the reason is the string). The refusals 1 to 6 are checked by the root's own reads (region range, agent table) and `Ledger.CheckRelocate`, the seventh through the fan-in. It has no error from `guard` (FR-WLD-006, 007).
3. `relocationRefusal` in `fanin.go` changes signature (D270); `fanin_internal_test.go` follows.
4. `relocate` and `moveAgent` (D268); the `Debug` lines "agent relocated" and "relocation refused" are guarded by `Enabled`; the event only when `obs.supplied`.
5. `export_test.go` gains three seams: `RelocateForTest` (over the unexported `relocate`), `SetEdgeStatusesForTest` (D272, added in the status phase) and the func `RegisterBoundForTest` (registers a fake obligation in the table), because no module is on. `World.RelocationRefusal` is already public and needs no seam. The `ForTest` suffix is the Stage 1 one, so plan 17's public `Relocate` does not collide.

Tests: `TestRelocationRefusal_Order` (one case per row 1 to 7 with the earlier ones passing and the later ones failing too; FR-REG-039, 044, 059); `TestRelocationRefusal_PseudoAgentAndRemoved` (D269; FR-REG-043); `TestRelocationRefusal_AllocatesNothing` (NFR/FR-REG-059); `TestRelocate_AgreesWithQuery` (for every `(agent, region)` of a small world, `relocate` returns exactly the query's error, FR-REG-059); `TestRelocate_RegionBound` with a registered fake obligation: refused with `RegionBoundError` naming it, `errors.Is(err, ErrRegionBound)`, `skip` by name and by key, no other refusal skipped (FR-REG-047, 060); `TestRelocate_AgentStateUnchanged` (role, weight, strategy, job, beliefs, ring, hunger, privation, directive untouched; the `inRole` and `hadAgent` cells move; the region's agent lists reflect the move at once, FR-REG-045, 048); `TestRelocate_EventAndOrder` (one `AgentRelocated` per success, none for a refusal or a pseudo-agent, after the move is complete, `Money` and `Units` as held, no map; FR-REG-046); `TestRelocate_CountsByRegionFromEvents` (households by region rebuilt from `AgentSpawned`, `AgentBankrupted`, `WeightChanged`, `RoleChanged` and `AgentRelocated` equal the snapshot; FR-OBS-040); `TestRelocate_NextTickPlacesOrdersInNewRegion` (the agent trades in its new market; `ReasonUnknownAgent` for the old one); `TestRelocate_Audit` (`Audit` nil, money and goods conserved over a seed range with failing seed printed, testing.md 6.2, 6.3).

Exit: allocation test for the no-observer path of the relocation (observability 4.5), `go test -short -race .`.

### Phase 7: edge status

Goal: the dynamic part of the graph. Files: `regions_status.go`, `event.go`, `export_test.go`.

`EdgeStatus{From, To RegionID, Closed bool, FactorBP int64}`; `setEdgeStatuses(changes []EdgeStatus, cause Cause) error` (D272): unknown edge `ErrUnknownEdge`, factor outside `[1, MaxEdgeFactorBP]` `ErrOutOfBounds`, repeated edge `ErrDuplicateID`, none applied on any failure, a change equal to the current status accepted and silent, one rebuild of the world's tables for the batch, then `EdgeStatusChanged{Tick, From, To RegionID, Closed bool, FactorBP int64, Cause Cause}` per changed edge in ascending `EdgeID` after the tables are in place. The `Debug` line "edge status changed". The seam, a method `World.SetEdgeStatusesForTest` in `export_test.go` (INT owns the public `SetEdgeStatuses`); cause `CauseConsumer`.

Tests: `TestEdgeStatus_AtomicAndRejected` (FR-REG-050; a failed call leaves every earlier and later `Graph` unaffected); `TestEdgeStatus_ClosedAndFactor` (examples A and B with the closure and the 15 000 factor; a closed edge is absent from `Out`, `In`, `Distance`, `NextHop`, `Path`, `Within` and the components and present in `Edge` with `Closed`; FR-REG-049); `TestEdgeStatus_RebuildOnceAndEventOrder` (the rebuild counter is 1 for a batch of five, the events are in `EdgeID` order and none for an equal status; the batch's `EdgeID`s do not change; FR-REG-014, 051); `TestEdgeStatus_PartitionsComponents` (a closure splits the component count and `RelocationRefusal` is unaffected, 4.5.4); `TestScenarioS28_MiddleLinkClosure` (REG-A1, `scenario_s28_test.go`, the only S28 test; seeds from `scenarioSeedCount()`: on `five` with the trade overlay the middle link closes and reopens through the unexported `setEdgeStatuses` by the `SetEdgeStatusesForTest` seam above, with the same justification as the Stage 1 yield-shock seam (N-296): the public call is plan 17's; distances, components and `RelocationRefusal` are checked at each step, skipped under `-short`, the seed in the message; FR-REG-049 to 051); `TestEdgeStatus_NoEffectOnRun` (a run with statuses set on a pack with no trade equals a run without, the status changes answers and the event only, 7.4); `TestEdgeStatusChanged_NoMapNoObserverNoAlloc` (FR-REG-057, observability 4.5).

### Phase 8: snapshot, checkpoint, logs

Goal: the state is visible and saved. Files: `snapshot.go`, `regions_status.go`, `construct.go`, `internal/logkeys/logkeys.go`.

1. `Snapshot.Edges []EdgeRow{From, To RegionID, Closed bool, FactorBP int64}` and `Snapshot.Components` (D275), appended at the end of `Snapshot`, filled in `Snapshot()` from `w.geo`. `snapshot_test.go` gains the rows; the renderer needs no change (a nil slice and a 0 are omitted).
2. The log keys `edges`, `components`, `from_region`, `to_region`, `distance`, `closed` as constants (the rows of 3.3.2 exist; plan 2's rewritten pinned-key test needs no edit here, R5). `summary()` adds `edges` and `components` to the "world created" line; `announce` logs the five warnings through the existing "configuration warning" line, with `rule` and `path`.
3. Tag 11, by plan 2's D243 protocol: `saveRegions` and `loadRegions` in `regions_status.go` replace the zero-row skeleton row of `checkpointSections` (the skeleton is a prefix of the real layout, so a world with no non-default status has the same bytes, D244); cross-checks (each row names a configured edge) are added to `checkpointCross`; restore-failure cases are added to the restore tests. The pending relocation list already exists in `Ledger.WriteState` (count 0); this plan fills it through `AppendRelocations` and `RestoreRelocations` with no change of layout, so no checkpoint digest moves (E6).

Tests: `TestSnapshot_EdgesAndComponents` (rows only for non-default statuses, `Components` 0 for the one-region pack `one`, 3 for `three` with no edge, 1 for `five` with the trade overlay; FR-OBS rows 5.3.7); `TestConstructionLog_EdgesAndComponents` and `TestWarnLines_FiveRules` (each warning once, `Warn`, `rule`, `path`, none at `Info` in the tick; FR-REG-056); `TestCheckpoint_RegionsSection` (a world with two non-default statuses round-trips; a world with none writes an empty section; the decoder refuses an unknown edge, an out-of-order row, a default row and a factor out of range with `ErrCheckpointCorrupt` naming tag 11; FR-REG-052); `TestCheckpoint_PendingRelocationTotals` (a relocation between ticks, then a checkpoint and a restore: the next tick's totals equal the original's; FR-REG-042); `TestFork_Regions` (calls `forkAtN` of plan 2 with a `forkCase` on `three` with `trade-three.toml` and a `Before` hook that closes an edge through the `setEdgeStatuses` seam, N in the set of D258; the restored world also answers every query of 4.3.2 as the original does, FR-REG-052).

### Phase 9: the packs and the trade overlays

Goal: the packs have the graph the roadmap needs (N-295). Files: `testdata/chains/overlay/trade-two.toml`, `trade-three.toml`, `trade-five.toml`, `reference_test.go` or a new `regions_pack_test.go`.

The overlays are the tables of `spec-regions.md` 12.4: `two` has `north` and `south` both ways at 100; `three` has `south` to each of `north` and `far_south` both ways at 100 and no edge between those two; `five` has `centre` to each of `east`, `north`, `south`, `west` both ways at 100 and `north`-`east` and `south`-`west` both ways at 130. `three` without the overlay loads, validates, runs (the probe of Stage 1 stands) and draws `region_isolated` for each region and `graph_not_strongly_connected` with `components` 3.

Tests (integration, `testing.Short()` skip): `TestPacks_TradeOverlaysLoad` (each variant plus its overlay validates, is one component, hash differs from the variant alone; FR-REG-054); `TestPacks_RegionIsolatedWithoutOverlay`; `TestPacks_EdgeListOrderIndependence` (reversed edge and region lists give equal graphs, hash and run; FR-REG-002); `TestPacks_ReversedListsGiveTheSameOutcome` keeps passing with the new lists; `TestScenarioS29_TownAsOneRegion` (REG-A2, `scenario_s29_test.go`, the only S29 test; seeds from `scenarioSeedCount()`: a town as one region against a town and two hamlets on the overlay packs, skipped under `-short`, the seed in the message); `TestPacks_InvariantsAndDeterminism` (rows `two` + `trade-two.toml` and `three` + `trade-three.toml` added to `invariantTable` (REF 7.5) and one `detCases` case each (REF 8.2); testing.md 6.2); `TestWorldRun_EdgesDoNotMoveAPack` (`three` with and without the overlay, seed 1, 50 ticks: equal state hash, because no module reads the graph yet; REG 2 row 11).

### Phase 10: exit gate

Goal: the row is done. Run, once the other agents' trees are quiet: `gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; R and F of plan-2-1 D229 (coverage: root 90%, `internal/graph` 90%, `internal/ledger` 90%); `go mod tidy` no diff; `lizard -l go -C 10 -a 8 -L 100 -w` on `regions.go`, `internal/graph/graph.go`, `internal/ledger/relocate.go`, `config/mapping.go`; `gomu` in Docker for `internal/graph` and `internal/ledger/relocate.go` with the caps; `testdata/golden/off/` byte-equal to `HEAD` (`git diff --stat -- testdata/golden/off` empty) and `TestHashConfig_ReferencePinned` green; the staged tests (registry, catalogue, position table) green with the entries of this row deleted: `Config.Edges`, `Config.RegionPrices`, `Edge`, `RegionPrice` from `notBuilt`, `AgentRelocated` and `EdgeStatusChanged` from `notBuiltEvents` and `deferredCore`. The position table is unchanged (D261). A PR description with the Testing and Observability entries below.

## Catalogue appends

| File | Appends | Position and id | Fixed by |
| --- | --- | --- | --- |
| `config.go` | `Edge`, `RegionPrice`; `Config.Edges`, `Config.RegionPrices` | between `Regions` and `Groups`; hash ids 8, 9 | spec-config.md 3.1, 8.2.2 |
| `bounds.go` | `MaxDistance`, `MaxEdges`, `MaxEdgeFactorBP`, `MaxRegionPrices` | after `MaxSeedPrices`; no hash id | spec-regions.md 4.2 |
| `defaults.go` | nothing: no default (8) | | spec-regions.md 8 |
| `validate.go` | `edgeRules`, graph rules, two count rows, the `refPrice` reads, the call of `regionWarnings` | validation phases 1, 2, 4; warnings | spec-regions.md 9; spec-config.md 7.1.6 |
| `hash.go` | `pairsEdge` (From 1, To 2, Distance 3), `pairsRegionPrice` (Region 1, Commodity 2, Price 3) and two `encodeList` calls | ids 8 and 9, after 7 | spec-config.md 8.2.2, N-218 |
| `errors.go` | `ErrSelfEdge`, `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrClosedAccount`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation`, `ErrRegionBound`, `RegionBoundError` | appended after the last sentinel | spec-regions.md 9.1; spec-config.md 7 (N-075) |
| `event.go` | `AgentRelocated`, `EdgeStatusChanged`; the six `Reason` constants above and `CauseTransport`, `CauseMigration` | appended after `WearApplied` (and plan 2's events); catalogue order is `obsEventOrder`'s | spec-observability.md 4.1; observability rules 4.3 |
| `option.go` | nothing | | |
| `tick.go` | nothing (D261) | | spec-world.md 6 |
| `snapshot.go` | `EdgeRow`, `Snapshot.Edges`, `Snapshot.Components` | at the end of `Snapshot` | spec-observability.md 5.3.7 |
| `ledger.go` (root) | nothing; the totals are `internal/ledger` | | spec-ledger.md 16.1 |
| checkpoint | tag 11 `Regions`, present always; pending relocation totals in tag 2 | tag 11 after tag 10 | spec-checkpoint.md 4.2 |
| log keys | `edges`, `components`, `from_region`, `to_region`, `distance`, `closed` | `internal/logkeys` | observability rules 3.3.2 |
| metrics | `economysim.household.relocations` is fed by `AgentRelocated` (plan 22) | | observability rules 5.3 |

## Cross-plan contract

Consumes:

- `checkpointSections` (the tag 11 skeleton row this plan replaces), `notBuiltSections`, `checkpointCross` (checkpoint_sections.go, checkpoint_restore.go) <- plan-2-2
- `codec.Writer`, `codec.Reader` (internal/codec) <- plan-2-2
- `Ledger.WriteState`, `Ledger.ReadState`, with the always-present pending relocation list written as count 0 in format 1 (internal/ledger/checkpoint.go, plan-2-2 D245) <- plan-2-2
- R and F, the two-run gate (plan-2-1 D229) <- plan-2-1
- `forkAtN`, `forkCase` (checkpoint_fork_test.go, plan-2-2 D258) <- plan-2-2
- `estates.agentID`, `estates.regionOf` (estates.go) <- Stage 1
- `relocationRefusal` (fanin.go), `Config.canonical`, `encodeList` (hash.go), the validator phases (validate.go), `warnings` (validate.go), `Snapshot`, `summary` (construct.go), `execSwitch` counters (phase_exit.go) <- Stage 1
- `ledger.Account`, `ClearTotals`, `AppendLots`, `Audit` (internal/ledger), `checked.MulDivCeil`, `checked.Add` (internal/checked) <- Stage 1

Provides:

- `Edge`, `RegionPrice`, `Config.Edges`, `Config.RegionPrices`, the loader keys (config.go, config/mapping.go) -> plan-2-24; `Config.Edges` also -> plan-2-13, plan-2-16
- `Graph`, `NewGraph`, `World.Graph`, `Hop`, `EdgeView`, `EdgeID` (regions.go) -> plan-2-9, plan-2-12, plan-2-13, plan-2-14, plan-2-16
- `regionState` unexported reads `distance`, `next`, `edge`, `out`, `components` of the world's own tables (regions.go) -> plan-2-12, plan-2-13, plan-2-14, plan-2-16
- `World.ReferencePrice`, `Config.refPrice` (regions.go) -> plan-2-5, plan-2-8, plan-2-18
- `World.RelocationRefusal` (regions_relocate.go) -> plan-2-12, plan-2-14, plan-2-17
- `relocationRefusal(a *agent, skip []string)` and the obligation table `w.bounds` with its `add(name)` (fanin.go, regions_relocate.go) -> plan-2-12, plan-2-14, plan-2-15, plan-2-16
- the marked line for row 5a in `relocationRefusal`/`RelocationRefusal` (fanin.go, regions_relocate.go) and the marked `region_without_agents` exemption (validate.go) -> plan-2-16
- `moveAgent(a *agent, to RegionID, cause Cause) error` (regions_relocate.go) -> plan-2-6, plan-2-10, plan-2-12, plan-2-13, plan-2-14
- `relocate(a AgentID, to RegionID, cause Cause) error` (regions_relocate.go) -> plan-2-13, plan-2-17
- `setEdgeStatuses(changes []EdgeStatus, cause Cause) error`, `EdgeStatus` (regions_status.go) -> plan-2-17
- `ErrSelfEdge`, `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrClosedAccount`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation`, `ErrRegionBound`, `RegionBoundError` (errors.go) -> plan-2-12, plan-2-14, plan-2-17
- the `Reason` constants `ReasonUnknownRegion`, `ReasonClosedAccount`, `ReasonNotRelocatable`, `ReasonSameRegion`, `ReasonOpenReservation`, `ReasonRegionBound`, and the `Cause` constants `CauseTransport` (`transport`) and `CauseMigration` (`migration`) (event.go) -> plan-2-12, plan-2-14, plan-2-17; `ReasonUnknownRegion` and `ReasonSameRegion` also -> plan-2-15
- `AgentRelocated`, `EdgeStatusChanged` (event.go) -> plan-2-17, plan-2-18, plan-2-22
- `Ledger.CheckRelocate`, `Ledger.Relocate`, `Ledger.Units`, `Ledger.AppendRelocations`, `Ledger.RestoreRelocations` (internal/ledger/relocate.go) -> plan-2-14
- `internal/graph` `Build`, `Tables`, `Effective` (internal/graph) -> plan-2-13
- `Graph.Component`, `Graph.Components` -> plan-2-9, plan-2-18; `setEdgeStatuses` and the `export_test.go` seam `SetEdgeStatusesForTest` -> plan-2-13, plan-2-16; the log keys `from_region`, `to_region` -> plan-2-14; `Config.RegionPrices` -> plan-2-18
- `Snapshot.Edges`, `Snapshot.Components` (snapshot.go) -> plan-2-18, plan-2-22
- `testdata/chains/overlay/trade-two.toml` -> plan-2-12, plan-2-14, plan-2-15, plan-2-17, plan-2-19, plan-2-24; `trade-three.toml`, `trade-five.toml` -> plan-2-4 (the Phase 5 `five` probe), plan-2-12, plan-2-14, plan-2-15, plan-2-19, plan-2-24
- the `export_test.go` seams `RelocateForTest` -> plan-2-10, plan-2-17 and `SetEdgeStatusesForTest` -> plan-2-17 (each deleted by plan 17); `RegisterBoundForTest` stays a helper of plan 3's own tests; `scenario_s28_test.go` -> plan-2-17 (switched to the public call)

## Testing

Requirement to test (the tests are named in the phase; one row per group). All cite their IDs in a `Requirement:` line.

| Requirement | Tests (phase) |
| --- | --- |
| FR-REG-001, 009 to 013, 033, 035 | `TestValidate_EdgeRules`, `TestValidate_RegionPriceRules` (3) |
| FR-REG-002, 014, 053, 054 | `TestCanonical_EdgesAndPricesSorted`, `TestHashConfig_EdgesAndPrices`, `TestLoad_*`, `TestPacks_EdgeListOrderIndependence` (3, 9) |
| FR-REG-003, 005, 006, 008 | `TestRegions_IterationOrder`, `TestEstates_*`, `TestRegions_SetIsFixed` (4) |
| FR-REG-004, 015 to 027, 029, 030 | `TestBuild_*`, `TestWithin_Order`, `TestQueries_OutOfRange`, `TestGraph_*` (2, 4) |
| FR-REG-007, 031, 034 | `TestWarnings_RegionRules` (3) |
| FR-REG-032, 036 to 038 | `TestReferencePrice_*`, `TestValidate_EffectivePriceFloorAndBasket` (3, 4) |
| FR-REG-028 | `TestGraph_IndependentOfWorld` (4) |
| FR-REG-039 to 048, 058 (the order up to row 5 only; 5a is plan 16), 059, 060 | `TestRelocate_*`, `TestRelocationRefusal_*`, `TestRelocationTotals_*` (5, 6) |
| FR-REG-049 to 051 | `TestEdgeStatus_*`, `TestScenarioS28_MiddleLinkClosure` (7) |
| FR-REG-015, 016 (REG-A2) | `TestScenarioS29_TownAsOneRegion` (9) |
| FR-REG-052 | `TestCheckpoint_RegionsSection`, `TestCheckpoint_PendingRelocationTotals`, `TestFork_Regions` (8) |
| FR-REG-055 | `TestBounds_Constants` (3) |
| FR-REG-056, 057 | `TestConstructionLog_*`, `TestWarnLines_FiveRules`, `TestEdgeStatusChanged_NoMapNoObserverNoAlloc`, `TestRelocationRefusal_AllocatesNothing` (6, 7, 8) |
| NFR-REG-001 | `TestGraph_NoAllocation` (4) |
| NFR-REG-002 | `BenchmarkBuild`, a report (2) |
| NFR-REG-003 | `FuzzBuild`, `FuzzValidateGraph` (2, 3) |
| NFR-REG-004 | `internal/graph` imports only the standard library and `internal/checked`, `import_guard_test.go` already bans third-party and root-level imports from `internal/` generically; Phase 2 adds a case that `internal/graph` imports nothing but the standard library and `internal/checked` |

- **Golden files.** `off`: unchanged, proved in Phase 10 by `git diff` and the pinned-hash test; no `off` file is regenerated. Module alone: none, REG has no module (D261). All-on: `testdata/golden/all/` does not exist yet; the plan that creates it includes this row. Plan 2's checkpoint digest moves `off` once (N-294); this plan changes no byte of an empty tag 11 or an empty relocation list and moves no digest.
- **Fork-at-N:** `TestFork_Regions` (Phase 8, through `forkAtN`). **Registry and catalogue:** entries deleted as Phase 3 and 6 say.
- **Fuzz:** `FuzzBuild` (corpus in `internal/graph/testdata/fuzz/`), `FuzzValidateGraph` and `FuzzRelocate`; any crash input is committed with a `Fix:` comment (testing.md 7.5).
- **Test counts** follow the pyramid: the unit tests of `internal/graph` and the ledger are the majority; the world and pack tests are few.

Testing entry for the PR (testing.md 11): requirements as above; lowest level for each is the package that owns it; new boundaries are the bounds of 4.2 and the refusals of 6.1.3, each with a failing-case test; no bug fix; `Guard:` tests on ageing by region, the pseudo-agent mapping and the pinned hash; the `off` golden did not change; fuzz added; coverage not lowered; the tests were run against a deliberately broken tie-break (the smaller `RegionID` replaced by the larger) and against a relocation that skips the `inRole` move, and failed.

## Observability

Entry for the PR (observability rules 9):

1. Failure paths: every refusal of `relocate` and every rejection of a status call is returned as an error; no `Warn`. The configuration problems that are survived are the five `Warn`s of 9.2 at construction.
2. New transitions: a relocation (`AgentRelocated`) and an edge status change (`EdgeStatusChanged`); no map, fixed-point fields, constructed only when an observer was supplied.
3. Decision points: none in a strategy.
4. Loops: no `Info` in the tick; the `Debug` lines "agent relocated", "relocation refused" and "edge status changed" are guarded by `Enabled`; no `Warn` per agent; event construction allocates nothing with the no-op observer (tested).
5. No changed event, metric name or unit. `economysim.household.relocations` (series: ordered region pairs times the causes) is plan 22's; `Snapshot.Edges` and `Components` feed no metric.
6. New metric attribute: none in the core.
7. Levels: `Info` only in the existing "world created" line (two attributes added); no `Error`.
8. No error is both returned and logged.
9. Tests of observability rules 8: events and order (Phase 6, 7), logs (Phase 8), defaults and non-interference (`TestWorldRun_EdgesDoNotMoveAPack`, `TestRelocate_*` with and without observer), cardinality (the cause and region fields take configured values), restore (Phase 8).

## Risks and open items

- **R1. Names from plan-2-2.** The section table, the codec types and the ledger state methods are named as plan-2-2's Provides names them; plan-2-2's Provides decides the identifiers. Owner: supervisor reconciles before the first code of either plan. Plan-2-2 D245 reserves the relocation list (count 0) and D244 the tag 11 skeleton; if either is dropped there, the supervisor adds the owed edit to plan 2 rather than let Phase 8 move the fixtures.
- **R2. `Components` and the off golden (D275).** The spec's field is non-zero in the reference world and the renderer prints non-zero header fields, so the literal field would move `testdata/golden/off/`. The rule is 0 for a one-region world only. Owner: OBS and REG edit (E3); if the supervisor prefers another rule, D275, E3 and the Phase 8 test change together.
- **R3. Anchors before arrivals.** P1.2 computes the cost anchors, the runnable recipes and the pooled wage before P1.3 and P1.5, so an agent that arrives in a region works that tick on anchors that did not count it (`phase_start.go`). The spec defines the anchors from the previous tick, so it is allowed; TRN and MIG (plans 12 and 14) confirm it in their tests.
- **R4. Pending totals have no public reader.** The relocation totals are read-only (6.4.2) and indicators need them (`spec-indicators.md` 8.2) but no accessor is named. Owner: IND (plan 18); `Ledger.AppendRelocations` is internal, and the root offers none until IND asks.
- **R5. Pinned log-key test.** Resolved by plan 2: its rewritten `logkeys_test.go` settles the pinned list, and this plan adds its six keys without editing the test.
- **R6. `ErrClosedAccount` and `ErrUnknownAgent` text.** The root sentinels carry new texts; `errors_test.go` lists them; the mapping of the ledger's sentinels (N-075) is by `errors.Is` in `relocate`, while `spec-regions.md` 9.1 says "maps to the same value" (E11 reconciles the wording).
- **R7. Trade overlays are placeholders.** Distances are the 100 and 130 of 12.4 until TRN fixes what a unit costs (plan 12); plan 12 may change the files, not the names.
- **R10. Notes tagged VIA.** N-093 item (1): the pooled viability check does not use `Graph.Components`; `Snapshot.Components` exists but VIA keeps `WithFreeTrade`. Leave N-093 open. Other open VIA, CFG or CODE notes that Stage 2 touches (N-276, N-061, N-311, N-224, N-245) are not touched by REG's code: leave open; plan 5 and plan 8 own the VIA programme.
- **R8. Memory of the tables.** 12 MB at the bound of 1 000 regions is per `Graph` and per world; `World.Graph()` builds a second copy by design (4.4.5). NFR-REG-002's timings are unmeasured (N-015); the benchmark reports.

Note proposals (text only; the supervisor edits `doc/notes.md`):

- N-A: *REG's relocation leaves region-keyed tick scratch stale for the arrival tick* (areas WLD, TRN, MIG): `w.tk.rings`, `w.runnable`, the pooled wage and the anchors are built at P1.1 and P1.2, before the arrivals of P1.3 and P1.5; TRN and MIG state in their plans that the lag is the spec's (R3).
- N-C: *The ledger section of the checkpoint carries the relocation totals from format 1* (areas CHK, REG): R1.
- Update to N-090 (REG, 2026-10-22): the obligation table is built at construction (D270); each module appends its name in its own plan.
- Update to N-218: ids 8 and 9 are used by `Edges` and `RegionPrices` (D265).
- Update to N-295: `three` has the trade overlay `trade-three.toml` (Phase 9).
