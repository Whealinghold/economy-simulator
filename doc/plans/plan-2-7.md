# Plan: Stage 2, row 7: EST, the Estates module part (relief, the pool as a flow target)

**Status: plan written (2026-10-22), not started. Written against `plan-2-0.md`; Stage 1 is the signed-off slice (`6dd4f86`).**
Rule tags read (all three files in full, as `AGENTS.md` asks): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1 (the same as the contract's; none differs).
Specs and requirements implemented: [spec-estates.md](../specs/spec-estates.md) 2.6, 3 and FR-EST-017, 018, 019, 020, 023 (FR-EST-016 and NFR-EST-001 are re-checked with relief on); [spec-config.md](../specs/spec-config.md) 6.2.4, 8.2.1 to 8.2.4, FR-CFG-099, 104, 105; [spec-money-supply.md](../specs/spec-money-supply.md) 4.3.2, 7.4, 7.5, 12.1 (the `ErrModuleDependency` row), FR-MON-038, FR-MON-039 (`grant_to_pool_no_outlet`); [spec-ledger.md](../specs/spec-ledger.md) 7.2 (the `relief` row); [spec-observability.md](../specs/spec-observability.md) 4.1.5 (`ReliefPaid`); [spec-reference-chain.md](../specs/spec-reference-chain.md) 4.4.5, 7.2 S14, 7.5 (the `famine` rows).
Also read: [spec-spawner.md](../specs/spec-spawner.md) 4.3, 5, 7.3 (the pool shared in time), [spec-welfare-and-wages.md](../specs/spec-welfare-and-wages.md) 2.1, 2.2.0 and 4.3 (what relief is in the ratio and the wage), [spec-world.md](../specs/spec-world.md) 6.3 (P1.6, P1.7), [spec-checkpoint.md](../specs/spec-checkpoint.md) 4.2.
Notes read: the index; N-082, N-096, N-119, N-167, N-254, N-256, N-309, N-310 (tagged EST) and N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217 (ALL). The plan changes no spec and no note, so it does not quote the notes tag; the proposals are in section 9.
Decision-ID block: **D340 to D359** (D340 to D355 used).

## Scope

This plan appends the **pool policy** of `Modules.Estates` (CFG 6.2.4): relief at P1.6, the pool as a target of a money flow, the two region fields, the `relief` ledger reason, `ReliefPaid`, and the pack and scenario that exercise them (`famine`, S14). FR-EST-001 to 015, 021, 022 and 024 are Stage 1 code and tests (FR-EST-003 is tested by plan 12); FR-EST-010 gets `TestEstates_PseudoAgentNotCounted` here (not in agents, households or the snapshot rows). It also closes the Stage 2 items of N-309 and N-310 that belong to the estate (Phase 6, items 2 and 3).

| In scope | Where |
| --- | --- |
| `type Estates struct{}`, `Modules.Estates *Estates` (the 25th pointer, hash id 25, no field) | `config.go`, `hash.go` |
| `Region.PoolRelease`, `Region.ReliefBelow` (hash ids 6 and 7, written only when the module is on), their bounds and defaults | `config.go`, `hash.go`, `bounds.go`, `defaults.go`, `validate.go` |
| Loader keys `pool_release`, `relief_below`, the table `[modules.estates]` | `config/mapping.go` |
| The reason `relief` (declared, internal, estate to agent) and the public `ReasonRelief` | `internal/ledger/registry.go`, `ledger.go`, `event.go` |
| P1.6: the relief pass, appended to `estates.go` after the marked comment line at its end | `estates.go`, `tick.go`, `world.go` |
| `ReliefPaid` | `event.go` |
| The pool as a flow target: the `pool` case of plan 6's three edit points (D320, D337): the kind switch of `validateMoneySupply` with `ErrModuleDependency`, `moneyFlows.recipients`, the pool carries and their tag 13 pairs; the warning `grant_to_pool_no_outlet` | `estates.go`, `moneyflow.go`, `validate.go` |
| The `famine` pack loads, the overlay `estates` over `one`, the module-alone golden, S14, the famine invariants row | `testdata/`, tests |
| World-row test debts of N-309 (update a) and N-310 (3) | Phase 6 |

| Out of scope | Owner |
| --- | --- |
| The money flows themselves (`moneyflow.go`, `internal/moneyflow`, the flow state, `MoneyFlowLimited`, the household carries, the tag 13 frame) | plan 6 |
| Checkpoint section of the estate listings (tag 8, `estates.rows`/`restore`) and N-309 item (2) | plan 2 |
| Feeding `estate.sold` and `estate.at_floor` to OpenTelemetry (N-310 item 2); `ReliefPaid` feeds no instrument | plan 22 |
| `EstateAtFloor` in the snapshot (N-310 item 1): already declared and filled (`snapshot.go` line 443) | done in Stage 1 |
| The rebalance of the reference pack (warm start, census) and any golden it moves | plan 8 |
| Pool as the target of a tax or payout (`payout_pool_no_outlet`), rent, interest into the pool | FIS, LND, CRD (Stage 3) |
| A treasury target, a stores target of a flow | POL/FIS, STO (Stage 3) |
| Population (births, deaths from the pool) | plan 11 |

Relief has **no state of its own**: it is a function of the pool, the welfare rings and the configuration, so it adds no checkpoint section. The pool carries of a flow with a pool target are flow state and are saved in plan 6's tag 13 (D351, D355).

## Decisions

Source "owner" is the owner's delegation of 2026-10-22 (whatever order makes sense; consistency first).

- **D340.** The module part lives in `estates.go`, below the marked comment line, as unexported functions over `internal/checked`, like `firstPrice` and `markedDown`. No `internal/` package is created: the arithmetic is two truncated products, `AGENTS.md` lists no package for it, and architecture.md 2.9 puts a package only where the arithmetic is more than that. (architecture.md 2.9, 2.10.1; N-309 item 3.)
- **D341.** `Modules.Estates *Estates`, `type Estates struct{}` with no field. It is appended to `Modules` last in the Go struct, and its hash id is **25** whatever the Go field order: `pairsModules` writes each pointer under an explicit constant, never under its declaration index. The id 25 pair is written last, after every other module pair; later plans insert theirs before it. (CFG 6.2.4, 8.2.1, 8.2.2; roadmap Stage 2 intro.)
- **D342.** `Region.PoolRelease` is hash id **6** and `Region.ReliefBelow` id **7** (next free ids after `EstateMarkdown` 5; `Label` has none). They are pairs **only when `Modules.Estates` is non-nil and the value is not zero** (the first bullet of CFG 8.2.1 and FR-CFG-104, so the pinned row 4 of CFG 8.2.4 holds: a region at zero with the module on has no `0006` or `0007` pair). With the module off they are not written and not validated (CFG 8.2.1). `pairsRegion` takes the module flag from its caller. The last bullet of 8.2.1 ("whatever its value") is edited to agree (Phase 1 item 3, O1).
- **D343.** Defaults `PoolRelease` 100 and `ReliefBelow` 10 000 (spec-estates.md 3), set by `DefaultRegion`. They are not zero, which CFG 8.2.3 rule (2) forbids for a core field; they are allowed because the field is a module's and is written only when the module is on (8.2.3 rule 3, N-217). The pinned hash of the all-off configuration is unchanged by construction. Phase 1 item 3 says this once in architecture.md 2.10.5 (O1).
- **D344.** Bounds: `pool_release` 0 to `wholeBP` (10 000), `relief_below` 0 to `MaxBasisPoints` (FR-EST-023). `regionRanges(r, estatesOn)` adds the two ranges only when the module is on, so a value outside the bound with the module off is not an error (CFG 8.2.1). An out-of-bound value with it on is `ErrOutOfBounds` at `regions.<id>.pool_release` and `.relief_below`.
- **D345.** Loader: `pool_release` and `relief_below` are read in every `[region.<id>]` table whatever the module, so a pack file that states them loads with the module off (they are then inert). `[modules.estates]` is an empty table: any key under it is `ErrUnknownKey`; its presence sets `Modules.Estates = &Estates{}`. (CFG 6.2.4, FR-CFG-099.)
- **D346.** The reason `relief` is a **declared** ledger reason (spec-ledger.md 7.2.2: "a module row, declared with its module"): `ledger.Relief` is appended after `LoanInterest` in the declared group of `internal/ledger/registry.go`, with the row `{thingMoney: {true, Internal, setEstate, setAgent, false}}`; `newLedger` declares it when `cfg.Modules.Estates != nil`; the root names it `ReasonRelief = "relief"` (`event.go`, internal-reasons block) and `reasonNames` gets its row. It opens no boundary account (it is internal). `Relief` uses the mechanism of plan 5 D311 (slice offsets, `Ledger.off [NumReasons][2]int` and so the per-position state hash unchanged with the module off); `TestPositionHash_Golden` and `TestNonInterference_EstatesOff` prove it. The Go constant order differs from the order of the table of 7.2; the number is internal and never saved (ledger.go), so this is allowed and noted in O2.
- **D347.** **P1.6** is `p16Relief`, called from `tick.go` in the table order of spec-world.md 6 (after P1.5 and before P1.7 when those exist, else after `p12Anchors`), guarded by `w.cfg.Modules.Estates != nil`. Per region in ascending `RegionID`: `release = pool * PoolRelease / 10 000` (checked, truncated); a region with `release < 1` is skipped. The pool is read as it stands, that is after the previous tick's P7.1 draws and the intake of the removals (spec-estates.md 2.6, N-082).
- **D348.** Need and share, all through `checked.MulDiv` (the 128-bit product, truncated toward zero): `need_i = MulDiv(U_i, ReliefBelow - ratio_i, 10 000)` for the household agents of the region with weight at least 1 whose ratio is present and below `ReliefBelow`; `share_i = MulDiv(release, need_i, sumNeed)`; `payment_i = min(share_i, need_i)`; a payment below 1 is no transfer. `ratio_i` is `history.Ratio(U_i, Role.DefaultRatio)`, the **total** ratio (relief of earlier ticks counts as income, spec-estates.md 2.6), padded as at P2.5; an absent ratio (padded cost 0) is no need. `U_i` is `costBuf.tick(role.upkeepIn(region), valued prices, weight)`, with the prices valued by a call to the existing `valuePrices` at the start of the pass (idempotent: the published price of the previous P5.1, else the reference price while the market has never traded, welfare 2.2.0; on tick 1 a warm pool is thus valued at reference prices, never at 0). External agents and cohorts are eligible; the estate pseudo-agents are not (indices below the region count; FR-EST-011; EST open question 1a).
- **D349.** Overflow: a subtraction, product or sum that does not fit returns `ErrOverflow` wrapped with the region and the agent, and the step fails (the pattern of `production.go` and `construct_warm.go`). No value is saturated. What is not paid stays in the pool (FR-EST-018). A payment that the agent's balance cannot hold is the ledger's `ErrBalanceOverflow`, returned the same way.
- **D350.** One pass over the agents, not one per region: pass A computes `release` per region and marks the regions to serve; pass B (ascending agent ID, regions marked) computes `need` into a scratch slice by slot and adds to the region's `sumNeed`; pass C pays each agent with `need > 0` its payment through `estates.grant(ledger, region, account, payment, ledger.Relief)` and adds it into the welfare scratch of its slot (`w.welfare.at(slot).relief`, saturating with `satAdd`, so `welfare.go` is not edited). The result of an agent depends only on its need and the region's totals (FR-EST-019); the ascending order fixes only the order of ledger operations. The scratch lives in a `reliefPass` struct on `World`, sized at construction and grown only when agents are added (no allocation in a warmed step, NFR-EST-001). Cost `O(A)` for the agents of regions with a positive release (spec-world.md 11.1).
- **D351.** Relief is stateless: it keeps no remainder, no latch, nothing across ticks. The relief pass adds no checkpoint section and nothing to `Stateful`; tag 8 (the listings) is the core's and plan 2's. The only module state is the pool carries of D355, saved in tag 13. A restored world's relief equals the uninterrupted one because the pool, the rings and the configuration are restored (fork-at-N test, section 7). The welfare record of tick t holds the relief paid at P1.6 of tick t, because the scratch is pushed at P6.3 and reset after (`p63Welfare`).
- **D352.** `ReliefPaid{Tick, Region, Paid Money, Recipients int64, Households int64}` is class A. It is delivered once per region, in ascending region order, after pass C, only when `w.obs.supplied` and the region paid at least 1. It is never delivered with the module off (it has no call then). `Recipients` counts agents paid, `Households` the sum of their weights; there is no map (FR-EST-020).
- **D353.** The pool as a flow target. Plan 6 (D320) makes every `pool` target `ErrModuleDependency` and leaves three edit points (D337, its Provides). This plan owns, because they need the `Estates` pointer: (a) the `pool` case of the target-kind switch of `validateMoneySupply`: a flow whose `Target.Kind` is `pool` while `Modules.Estates` is nil stays `ErrModuleDependency` at `modules.money_supply.flows.<id>.target.kind` (MON 12.1, FR-MON-038), with the module on it is accepted; (b) the `pool` case of `moneyFlows.recipients`: the Estate account of `Target.Region`, or of every region in ascending order when empty, weight 1 (MON 4.3.2, 4.3.3, FR-MON-008), `estates.account(r)` being the account; (c) the warning `grant_to_pool_no_outlet` at the flow's path, when a grant targets a region whose `PoolRelease` is 0 and the module is on (MON 12.2, FR-MON-039); (d) the pool carry (D355). The reserved-reason rule (`reservedReasons` in `moneyflow.go`, plan 6 D325) already lists `relief`; this plan adds only the `reasonNames` row that plan 6's guard test checks against it. A pool levy collects after relief (MON 7.5; P1.6 precedes P1.7). Plan 6 owns everything else about flows; (a) to (d) plug into its seams (O3).
- **D354.** The `famine` pack is made loadable: layers `base.toml`, `famine.toml`, `overlay/estates.toml`, then `warm-famine.toml` (REF 4.5 table). The cold layer set is the first three. `expected/famine.toml` is recorded (REF 4.5, FR-REF-039), and the byte test of its warm document is added to `warmPacks` only if the document equals the solver's table; the maker's jobs differ today (`viability_solve_pack_test.go`, `TestSolveSteadyState_Famine`), so that test is plan 8's if it does not pass (O5).
- **D355.** The pool carry. A flow with a pool target and a carry shape (rate grant, fixed levy, rate levy; MON 6.2) keeps one carry per region, in region order, in plan 6's `flowState`, started with `StartCarry(regionIndex, P)` (FR-MON-013: the region index mod `P`, times 10^4; 0 for a rate levy) and advanced by the flow's due like a household carry (FR-MON-022). They are written in tag 13 after the household pairs of the same flow, as (region index, carry) pairs in ascending index (D337, MON 14), and decoded with plan 6's checks R6 and R9 (an index outside the regions or a descending pair is `ErrCheckpointCorrupt`).

## Phases

### Phase 1: edits owed before any code

Goal: the specs are true before the code starts. The supervisor applies them; this plan edits none. The pool-target and relief code all goes in `estates.go` (already in the layout); a split into a continuation file needs the same conditional layout row (architecture.md 2.6, N-243) in the same change.

| # | File and section | Edit | Why |
| --- | --- | --- | --- |
| 1 | `spec-config.md` 8.2.2 | add "`Region` ... `EstateMarkdown` 5, `PoolRelease` 6, `ReliefBelow` 7" | D342 |
| 2 | `spec-config.md` 8.2.4 | add one worked row with its bytes and digest: the one-region row with `Modules.Estates` on and the region at the defaults (`0006` 100 `0007` 10 000 inside the region, then `000c` `0019` `0000` `0000`); row 4 and the last row do not change | O1; FR-CFG-105 needs a pinned digest |
| 3 | `spec-config.md` 8.2.1 last bullet, `architecture.md` 2.10.5 and `spec-config.md` 8.2.3 | 8.2.1: "a pair only when its module is on and its value is not zero"; and one sentence: the zero-default rule applies to a field of a core struct; a field a module owns may default to a non-zero value because it is written only when the module is on. New rule tag; same 2.10.5 change as plan 11 E6; one fresh tag | D343, O1 |
| 4 | `spec-estates.md` 2.6 | state: an agent whose ratio is absent is not eligible; the ratio is the total one; overflow is `ErrOverflow` and fails the step; external agents are eligible (promote open question 1a); the prices are those of welfare 2.2.0 | D348, D349 |
| 5 | `spec-ledger.md` 7.2.2 | none here: plan 5 E7 carries the wording; relief is appended after `LoanInterest` (D346) | D346, O2 |
| 6 | `spec-money-supply.md` 12.1/12.2 | say which spec owns the pool-target rule and the warning (this plan, because it reads `Modules.Estates` and `PoolRelease`) | D353 |
| 7 | `spec-reference-chain.md` 7.2 S14, 7.1.3 | name the four scripts `scripts/s14-<m>.toml` (m = 7000, 5000, 2500, 0) and the file `bands/famine.toml` with `s14.recover_band_bp`, `s14.recover_ticks`, `s14.fall_bp` unset; say where the outcomes are recorded (`expected/famine.scenarios.txt`) | Phase 5 |
| 8 | `doc.go` glossary | add **Relief** (money the pool pays each tick to households whose welfare ratio is below the relief line) and **Relief line** (`ReliefBelow`); name the exported types and events the plan adds (`Estates`, `ReliefPaid`) so `TestGlossary_DeclaredTypesNamed` passes; all before code uses them | architecture.md 3.2 |
| 9 | `roadmap-v1.md` | the Stage 2 row marked done with its commit at the end | gate |

Exit check: the supervisor reports the nine edits applied (or ruled otherwise), and the digest of row 2 is recorded by the encoder test of Phase 2.

### Phase 2: configuration, hash, bounds, loader

Goal: the module is a configuration object and leaves no trace while off.
Files: `config.go`, `hash.go`, `bounds.go`, `defaults.go`, `validate.go`, `construct.go` (`modulesOn` appends `"estates"` after the names present when this lands: build order, E12 of plan 5), `config/mapping.go`, `registry_test.go`, the tests below.
Tests (all unit, parallel, table-driven):
- `TestHash_EstatesModule` pins `0848fee29c503bda99a17acb2c44c42fbba8b730eadc2079e3d8123b2d3317ad` (CFG 8.2.4 row 4, unchanged by D342). Requirement: FR-CFG-099, FR-CFG-104, FR-CFG-105.
- `TestHash_RegionPoolFieldsOffLeaveNoTrace` pins `ed652dfd86904c18fd76feb17175e7efeef6d4afe94bec1e3b4f5a2e2409f013` for a region with `PoolRelease` 100 and `ReliefBelow` 10 000 and the module off (the defaults do not move the hash). Requirement: FR-CFG-104, FR-CFG-105, FR-CFG-099.
- `TestHash_RegionPoolFieldsOn` pins the new row of Phase 1 item 2, shows each of the two fields changes the hash when the module is on and neither does when it is off, and that either field at 0 with the module on equals the row 4 bytes. Requirement: FR-CFG-026, FR-CFG-099.
- `TestRegionPoolFields_Bounds`: `pool_release` -1 and 10 001, `relief_below` -1 and `MaxBasisPoints+1` give `ErrOutOfBounds` at the named paths with the module on, and nothing with it off. Requirement: FR-EST-023.
- `TestDefaultRegion_PoolFields` (100 and 10 000). Requirement: FR-EST-023.
- `TestLoad_EstatesTable` in `config`: `[modules.estates]` sets the pointer; a key under it is `ErrUnknownKey`; `pool_release` and `relief_below` load with the module off. The existing cases `config_test.go` line 112 (module table of a module not built) and the fuzz seed `config/fuzz_test.go` line 40 name `estates` as the unbuilt module: they are changed to another unbuilt module (the choice follows what plans 5 and 6 left), and `config/integration_test.go` line 324 (the `estates overlay` row of the unknown-key test) is deleted. Requirement: FR-CFG-099, FR-REF-001.
- `TestRegistry_*` (existing test): delete `"Modules.Estates"`, `"Region.PoolRelease"`, `"Region.ReliefBelow"` from `notBuilt`; the rows of README Parameter owners (lines 179 to 182, 679) now resolve. Requirement: FR-CFG-099 (registry rule, architecture.md 9.7).
Exit check: `go test -short -race ./... ` on the root and `config`; `testdata/golden/off/` and `TestPacks_LoadValidateAndHash` unchanged.

### Phase 3: the `relief` reason and the relief pass

Goal: P1.6 pays relief; everything is checked without a world first.
Files: `internal/ledger/registry.go` (+ its registry test row), `ledger.go`, `event.go`, `estates.go`, `world.go` (the `reliefPass` field), `construct.go` (allocate it when the module is on), `tick.go`, `poshash_internal_test.go` (the guarded `replayRows` row for P1.6, the form plan 5 provides), `observe_internal_test.go` (the `stepSites` row, kept complete by `TestCallSites_TheTableIsTheStraightLine`), tests.
Tests:
- `internal/ledger`: `TestRegistry_Relief` (a transfer by `relief` from an estate to an agent is accepted only with the reason declared; agent to estate, agent to agent, boundary to anything are `ErrReasonMismatch`; an undeclared ledger refuses it; no boundary account opens). Requirement: FR-LED-052, FR-EST-016.
- `TestReliefRelease_Table`: pool 0, 1, 9 999, 10 000, `MaxMoney`; `PoolRelease` 0, 1, 100, 10 000; truncation; overflow impossible by `MulDiv`. Requirement: FR-EST-017.
- `TestReliefNeed_Table`: ratio at, above and below the line; ratio 0; negative ratio; `MinInt64` ratio (the subtraction overflows: `ErrOverflow`); absent ratio; truncation of the product. Requirement: FR-EST-018.
- `TestReliefShare_Table` and `TestReliefShare_Properties`: the payments never exceed their needs, the sum never exceeds the release, a release above the total need pays every need in full and leaves the rest, the remainder stays in the pool, a sum of needs of 0 pays nothing. Requirement: FR-EST-018.
- `TestReliefShare_OrderFree`: the same needs in every permutation (up to 6 agents) give the same payment for each agent. Requirement: FR-EST-019.
- `FuzzReliefShares` (seed corpus committed under `testdata/fuzz/FuzzReliefShares`): no panic, `sum(pay) <= release <= pool`, `pay_i <= need_i`, permutation-free. Requirement: FR-EST-018, FR-EST-019.
- `TestRelief_Pass` (internal, a built world with a scripted pool and rings): pool 1 000 000, `PoolRelease` 100, three households at ratios 3 000, 8 000 and 12 000 give relief to the first two only, in proportion to their needs; payments are `relief` transfers; welfare slot `relief` equals the payment; the pool falls by the total paid. Requirement: FR-EST-017, FR-EST-018, FR-WEL-040.
- `TestRelief_NoPoolNoPass`: a region with an empty pool or `PoolRelease` 0 pays nothing and emits nothing. Requirement: FR-EST-017.
- `TestRelief_EstateNeverPaid`: no estate account is a recipient even with a negative-ratio configuration. Requirement: FR-EST-011.
- `TestRelief_ExternalAndCohort`: an external agent and an agent of weight 5 are paid; the cohort's `need` is the cohort's total. Requirement: FR-EST-018 (the 6.1a decision).
- `TestEstates_PseudoAgentNotCounted` (public): the estate pseudo-agents are in no `Agents` count, no household count and no snapshot agent row, with relief on. Requirement: FR-EST-010.
- `TestRelief_NoAllocation`: a warmed pass allocates nothing. Requirement: NFR-EST-001.
- `Guard: TestRelief_OverflowFailsStep` (the pass returns `ErrOverflow` wrapping the region and agent; nothing is paid before the refusal is known: the pass computes every need before the first transfer). Requirement: FR-EST-018.
Exit check: the unit tests above; coverage of the new lines of `estates.go` 100% (the file is at 100% today, N-309 update).

### Phase 4: the world, the event, the golden, the famine pack

Goal: the module runs in a world and in the packs, and the reference configuration with it off is untouched.
Files: `tick.go` (the guarded call; `{"P1.6", "EST", line}` moved from `notBuiltPositions` to `positionList` in the table order of spec-world.md 6, after P1.5 and before P1.7 when those exist, else after `P1.2`), `event.go` (`ReliefPaid`), `estates.go` (delivery), `event_test.go` (`ReliefPaid` out of `notBuiltEvents`), `render_test.go` (out of its second list), `viability_pack_test.go` (`packLayers` gets `famine`), `config/expected_test.go` (`TestExpected_Packs` reads `expected/famine.toml`), `testdata/golden/estates/`, `testdata/chains/expected/famine.toml`, `export_test.go`, tests.
Tests:
- `TestPositions_P16` (the existing position test, tests 6.8): the table row is called and no longer listed; `replayRows` and `stepSites` hold P1.6. Requirement: FR-WLD-040, FR-WLD-004.
- `TestReliefPaid_Events` (public, recording observer, `one` + `overlay/estates` with the pool seeded by a warm pool and a household below the line): one event per region per tick in which something was paid, in ascending region order, before `MoneyFlowLimited` of the same tick when plan 6 is in; `Paid` equals the sum of the `relief` flows of the tick; `Recipients` and `Households` as counted; no event with the module off; no event when nothing was paid; no map. Requirement: FR-EST-020, FR-OBS-008, FR-CFG-099.
- `TestReliefPaid_Catalogue`: the event is a row of the table of 4.1.5 with owner EST, position P1.6, class A, gate "estates (pool policy)"; its fields in the row's order; the rendering at zero fields omits them (OBS 9.5.2). Requirement: FR-OBS-003, FR-OBS-004, FR-OBS-080, FR-OBS-091.
- `TestRelief_TimingAfterEntry` (public): a pool left by tick t's entry step is the pool relief reads at tick t+1, and relief of tick t+1 is in tick t+1's welfare record, not t's. Requirement: FR-EST-017, FR-WEL-002.
- `TestRelief_NotWage`: a household that lives on relief shows an earned ratio and observed wage without it (`EarnedRatio` below the total ratio). Requirement: FR-WEL-037, FR-WEL-040.
- `TestNonInterference_EstatesOff`: with the module off a run of `one` equals the Stage 1 run byte for byte (`testdata/golden/off/one.txt`, `events-one.txt`, `one.positions.txt` unchanged by this plan; no `-update`). Requirement: FR-CFG-099, FR-OBS-008, FR-WLD-044.
- `TestGolden_EstatesAlone`: `one` + `overlay/estates` for the length and seed of REF 8.2, recorded in `testdata/golden/estates/one.txt`, `events-one.txt`, `one.positions.txt` (the files and helper plan 5 introduced for `demand`). The explanation of testing.md 7.2 is in the PR: relief is the only difference from `off/`, and it appears only after the first removal that leaves a pool. Requirement: FR-REF-029, FR-EST-017.
- `TestPackFamine_LoadsAndHashes`: the four layers load and `Validate` returns nil, the cold hash and the warm hash are recorded in `expected/famine.toml` with `-update` (REF 4.5) and read by `TestExpected_Packs`; `packLayers` gets `famine`; the helper `famine(t)` of `viability_solve_pack_test.go` (built from `one` because the file did not load in Stage 1) is replaced by the loaded layers. Requirement: FR-REF-001, FR-REF-005, FR-REF-006, FR-REF-039.
- `TestInvariants_Famine`: the row `famine` (800 ticks, full set 1 to 4, reduced by plan 1's `invariantSeeds`, with the shock of S14 at depth 0 through `SetYieldShockForTest`) joins the invariants table of REF 7.5; all six invariants hold with the audit on and relief active. Requirement: FR-EST-016, FR-EST-022, FR-REF-025, FR-STB-005.
- `TestDeterminism_Estates`: the same seed, with and without a logger and an observer, any split of `Run`, with the module on. Requirement: FR-CFG-026, FR-OBS-023.
Exit check: the above; `testdata/golden/off/` unchanged (a diff is a failure, architecture.md 2.10.5).

### Phase 5: the pool as a flow target (after plan 6)

Goal: plan 6's three edit points get the `pool` case; a pool target works and is refused when the module is off. This phase needs plan 6's code merged.
Files: `moneyflow.go` (the `pool` case of `recipients`, the pool carries and their tag 13 pairs, D355), `estates.go` (`estatesPoolClass`), `validate.go` (the kind-switch case and the warning), `scenario_mons3_test.go`, plan 6's tag 13 test fixtures (updated for the pool pairs), tests.
Tests:
- `TestValidate_PoolTargetNeedsEstates`: a grant and a levy with `Target.Kind` "pool" and `Modules.Estates` nil give `ErrModuleDependency` at the path of the flow; with the module on, none. Requirement: FR-MON-038.
- `TestValidate_GrantToPoolNoOutlet`: warning at the flow's path when the region's `PoolRelease` is 0, none when above; it does not fail `Validate`. Requirement: FR-MON-039.
- `TestReservedReasons_CoversRelief`: plan 6's guard test (every `reasonNames` name is in `reservedReasons`) passes with `ReasonRelief` added, and a flow ID `relief` is `ErrReasonCollision`. Requirement: FR-MON-036.
- `TestPoolClass_Members`: a pool class is the estate account of the named region, or every region in ascending order, weight 1. Requirement: FR-MON-008.
- `TestPoolCarry_StartByRegionIndex` and `TestPoolCarry_AdvancesByDue`: the start is the region index mod `P` times 10^4 (0 for a rate levy), and a pool rate grant over a year sums exactly. Requirement: FR-MON-013, FR-MON-012.
- `TestCheckpoint_PoolCarryRoundTrip` and `TestCheckpoint_PoolCarryCorrupt`: tag 13 with a pool flow encodes the region pairs after the household pairs, decodes equal and is byte-canonical; an index outside the regions or a descending pair is `ErrCheckpointCorrupt`. Requirement: FR-MON-046, FR-CHK-023.
- `TestFlow_PoolGrantFeedsRelief` (public, `one` + estates + money supply): a grant to a pool raises `M_p` at P1.7; the grant reaches households at the same tick's entry step (P7.1, MON 5.1.2) and at the next tick's relief (P1.6), and by no other path (MON 7.4). Requirement: FR-MON-019, FR-MON-046.
- `TestFlow_PoolLevyAfterRelief`: a levy on a pool collects after relief was paid in the same tick, never taking money relief committed (MON 7.5). Requirement: FR-MON-019, FR-EST-017.
- `TestScenarioMONS3_PoolGrant` (tier G, `-short` skips): growing households with and without a pool grant, 8.6; reports `M_h / H` against the two thresholds, the unfed households and the pool. Requirement: FR-MON-046 (pool part), MON-S3.
Exit check: the tests; `go vet`, `staticcheck`.

### Phase 6: S14, the world-row test debts, and the gate

Goal: the scenario, the debts and the exit gate.
1. **S14** (`scenario_s14_test.go`, tier G, skipped with `-short`): pack `famine`; 1 800 ticks; `scenarioSeedCount()` seeds (the full set is 1 to 10, reduced under `-race`); four scripts `scripts/s14-7000.toml`, `s14-5000.toml`, `s14-2500.toml`, `s14-0.toml`, each `grow_staple` at `m` from tick 424 to 515 and cleared at 516 (the form of `scripts/s04.toml`). Exact: no balance or inventory negative, goods and money conserved every tick, the pool never overdrawn (FR-EST-022), no price outside its belief interval except an estate sale (FR-STB-008). Reported: households over time as a share of the start, unfed households and the longest run with any unfed, relief paid and the pool balance, removals by cause, the tick of recovery within `s14.recover_band_bp`; the depth below which the region does not recover. Asserted only when its key has a value: `s14.recover_ticks`, `s14.fall_bp` in `bands/famine.toml` (created with every key unset). The outcomes are written with `-update` to `testdata/chains/expected/famine.scenarios.txt` through plan 1's per-pack `scenReport` (the pack name picks the file; it is not the fixed `one.scenarios.txt` path). The ladder-order check ("the ladder rungs act in order", FR-SPN-013 to 015) is asserted only when `s14.fall_bp` is set. Requirement: FR-EST-014 to FR-EST-020, FR-EST-022, FR-STB-005, FR-AGT-013 to FR-AGT-017, FR-SPN-012 to FR-SPN-016, FR-SPN-024 to FR-SPN-026, FR-REF-020 to 024, FR-REF-026, FR-REF-027. The leg on `famine` with the population overlay is tier L and waits for plan 11. N-254 (a lot that gains units keeps its price) is read in this scenario: if the estate's marked-down lot depresses the staple price it reports it (O6).
2. **N-309 update (a)**, the public-API tests through `NewWorld` and `Step` with a recording observer (`estates_world_test.go`): `TestEstates_NoBidEverCarriesAnEstateID` (no `OrderPlaced` or `OrderRejected` with `Side` Bid has an estate's ID over a run with removals; FR-EST-006), `TestEstates_OneAskPerHeldLot` (the asks of a removal tick are one per held lot, a rejected ask is not marked down; FR-EST-007), `TestEstates_MarkdownAtP53` (the listed price falls by `EstateMarkdown` for each tick the lot had an admitted ask; with 0 nothing falls; FR-EST-009), `TestEstates_ProceedsArePool` (the proceeds are a `trade` transfer into the estate account and `RegionRow.MoneyPool` equals that account's money; FR-EST-013).
3. **N-310 item 3**, `TestEstates_PoolOverflowFailsStep` (`export_test.go` seam, as `phase_exit_exec_internal_test.go`): a pool seeded near `MaxMoney`, a removal that overflows it: the step fails, `TickEnded` has `Failed`, the error wraps `ErrBalanceOverflow` and names the region and the account, and the world is failed afterwards (`ErrWorldFailed`). Requirement: FR-EST-001, FR-WLD-004. It also runs once for a relief payment that overflows an agent's balance.
4. **Fork-at-N** (`TestFork_Estates`): calls `forkAtN` with a `forkCase` whose `Cfg` is `famine` with the module on and a pool grant flow (so a pool carry is mid-year), `Before` seeding a pool, and N chosen so that relief has paid and a listing holds a marked-down lot (N in {0, inside a job, inside a season, after a call between ticks} as plan 2 D258 asks). The comparison per position and the restore of a restore are `forkAtN`'s. Requirement: FR-EST-024, FR-CHK-023, FR-CHK-050, FR-MON-046.
5. **Gate** (plan-2-0): `gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; `go mod tidy` no diff; `lizard -l go -C 10 -a 8 -L 100 -w` on `estates.go`, `ledger.go`, the touched functions of `validate.go` and `hash.go`; coverage at least 90% for the root and the packages touched (`internal/ledger`, `config`); no new `internal/` package, so no new mutation pass beyond the one on `estates.go` and the registry file (Testing section); `notBuilt` and `notBuiltPositions` entries deleted; R and F exactly as plan-2-1 D229 states them.

## Catalogue appends

| File | Append | Position and id | Spec line |
| --- | --- | --- | --- |
| `config.go` | `type Estates struct{}`; `Modules.Estates *Estates`; `Region.PoolRelease`, `Region.ReliefBelow` (int64) | last pointer of `Modules`; after `EstateMarkdown` in `Region` | CFG 3.1, 6.2.4; EST 3 |
| `hash.go` | `pairsModules`: pointer written as id 25, last after every other module pair (later plans insert theirs before it), its terminator; `pairsRegion(e, r, estatesOn)`: ids 6 and 7 only when on | Modules id 25; Region ids 6, 7 | CFG 8.2.1, 8.2.2 |
| `bounds.go` | `regionRanges(r, estatesOn)`: `pool_release` 0 to 10 000, `relief_below` 0 to `MaxBasisPoints` | after `estate_markdown` | EST 3, FR-EST-023 |
| `defaults.go` | `DefaultRegion`: `PoolRelease` 100, `ReliefBelow` 10 000 | after `EstateMarkdown` | EST 3 |
| `validate.go` | region ranges with the flag; the `pool` case of plan 6's kind switch; `grant_to_pool_no_outlet` | with the module rules, after plan 6's flow rules | MON 12.1, 12.2 |
| `errors.go` | none: `ErrModuleDependency` and `ErrOverflow` exist | | |
| `event.go` | `ReliefPaid`, its `isEvent`; `ReasonRelief` in the internal-reasons block | appended after the events built before (the table order of OBS 4.1 is only in `obsEventOrder`, which already has the row) | OBS 4.1.5 |
| `option.go` | none | | |
| `tick.go` | `{"P1.6","EST",line}` in `positionList`; `"P1.6"` out of `notBuiltPositions`; the guarded call in the table order of spec-world.md 6: after P1.5 and before P1.7 when those exist, else after `p12Anchors` | table order of spec-world.md 6 | WLD 6.3 |
| `ledger.go` | `reasonNames` row for `ledger.Relief`; `newLedger` declares it with the module (`relief` is already in plan 6's `reservedReasons`) | | LED 7.2 |
| `internal/ledger/registry.go` | `Relief` after `LoanInterest`; its `table` row; the mechanism of plan 5 D311 (slice offsets, `off` unchanged), proved by `TestPositionHash_Golden` and `TestNonInterference_EstatesOff` | declared group | LED 7.2 |
| `construct.go` | `modulesOn` appends `"estates"` (its own comment asks for it); allocate `reliefPass` | after the names present when this lands (build order; E12 of plan 5) | CFG 10.1 |
| `world.go` | field `relief *reliefPass` | with the other module pointers | ADR 0007 |
| `moneyflow.go` | the `pool` case of `recipients`; pool carries in `flowState`; their pairs in the tag 13 encoder and decoder after the household pairs | plan 6's edit points | MON 4.3.2, 6.2, 14 |
| `snapshot.go`, checkpoint tags other than 13, log keys, metrics | none (D351, section 8) | | |
| `config/mapping.go` | `pool_release`, `relief_below` in `region`; `[modules.estates]` in the modules loop | | CFG 6.2.4 |
| `registry_test.go`, `event_test.go`, `render_test.go`, `poshash_internal_test.go`, `observe_internal_test.go` | three entries out of `notBuilt`; `ReliefPaid` out of `notBuiltEvents` and the second list of `render_test.go`; the `replayRows` and `stepSites` rows of P1.6 | | testing 6.8, 7.1.2 |

## Cross-plan contract

Consumes:
Plan 5 is named beyond Needs (2, 6): Modules pointer form, newLedger declared reasons, the golden helper.
- `Modules` struct and the pointer-by-explicit-id form of `pairsModules` (`config.go`, `hash.go`) <- plan-2-5
- the declared-reason list assembled by `newLedger` (`ledger.go`) <- plan-2-5
- the module-table dispatch of the loader (`config/mapping.go`) <- plan-2-5
- the module-alone golden helper and the `testdata/golden/<module>/` layout (`golden_test.go`) <- plan-2-5
- `MoneyFlow`, `FlowTarget` (`config.go`) <- plan-2-6
- `moneyFlows.recipients` (kind dispatch) and the target-kind switch of `validateMoneySupply` (`moneyflow.go`) <- plan-2-6
- the tag 13 encoder and decoder with the position of the pool carries after the household pairs (`moneyflow.go`) <- plan-2-6
- `reservedReasons` and its guard test (`moneyflow.go`) <- plan-2-6
- the guarded-call form in `tick.go` and its position test, the guarded `replayRows` row (`tick.go`, `tick_internal_test.go`, `poshash_internal_test.go`) <- plan-2-5
- `Checkpoint`, `RestoreWorld`, the estate listing section built from `estates.rows` and `estates.restore` (`checkpoint.go`) <- plan-2-2
- `estates` type, `grant`, `pool`, `account`, `heldLots`, `rows`, `restore` (`estates.go`) <- Stage 1
- `valuePrices`, `costBuf.tick`, `Ring.Ratio`, `welfareScratch.at`, `satAdd` (`phase_outcomes.go`, `welfare.go`, `internal/welfare`) <- Stage 1
- `forkAtN`, `forkCase` (`checkpoint_fork_test.go`) <- plan-2-2
- `scenarioSeedCount` (`scenario_helpers_test.go`) <- plan-2-1
- the per-pack `scenReport` (the pack name picks `expected/<pack>.scenarios.txt`) (`scenario_helpers_test.go`) <- plan-2-1
- `SetYieldShockForTest`, `runScenario`, `scenReport`, `loadScript`, `invariantTable` (`invariants_test.go`) (test seam) <- Stage 1

Provides:
- edits owed to files that plans 2 and 6 own (the `pool` case of plan 6's three edit points, the tag 13 pool pairs, plan 6's tag 13 test fixtures): applied by this plan in its Phase 1 and Phase 5, and listed there as edits to `moneyflow.go`
- `Estates`, `Modules.Estates` (`config.go`) -> plan-2-8
- `Region.PoolRelease`, `Region.ReliefBelow` (`config.go`) -> plan-2-8, plan-2-11, plan-2-16, plan-2-24
- `ledger.Relief`, `ReasonRelief` (`internal/ledger/registry.go`, `event.go`) -> plan-2-11
- `ReliefPaid` (`event.go`) -> plan-2-18, plan-2-22, plan-2-24
- `p16Relief` and position P1.6, relief reading the welfare ratio (`tick.go`, `estates.go`) -> plan-2-10 (S57 only), plan-2-14
- the `pool` case of the three edit points of plan 6 and the warning `grant_to_pool_no_outlet` (`moneyflow.go`, `estates.go`, `validate.go`); plan 6's tag 13 test fixtures are updated by this plan (Phase 5) -> no later plan
- pack `famine` loadable, `expected/famine.toml`, `bands/famine.toml`, `scripts/s14-*.toml` (`testdata/chains/`) -> plan-2-8, plan-2-19
- `testdata/golden/estates/` (module-alone golden) -> plan-2-8, plan-2-24

## Testing (testing.md 11)

| Requirement | Tests (Phase) |
| --- | --- |
| FR-EST-017 | `TestReliefRelease_Table`, `TestRelief_Pass`, `TestRelief_NoPoolNoPass`, `TestRelief_TimingAfterEntry`, S14 (3, 4, 6) |
| FR-EST-018 | `TestReliefNeed_Table`, `TestReliefShare_Table`, `TestReliefShare_Properties`, `FuzzReliefShares`, `TestRelief_Pass` (3) |
| FR-EST-019 | `TestReliefShare_OrderFree`, `FuzzReliefShares` (3) |
| FR-EST-020 | `TestReliefPaid_Events`, `TestReliefPaid_Catalogue` (4) |
| FR-EST-023 | `TestRegionPoolFields_Bounds`, `TestDefaultRegion_PoolFields` (2) |
| FR-EST-010, 011, 016, 022, NFR-EST-001 | `TestEstates_PseudoAgentNotCounted`, `TestRelief_EstateNeverPaid`, `TestInvariants_Famine`, `TestRelief_NoAllocation`, S14 (3, 4, 6) |
| FR-EST-006, 007, 009, 013 (N-309 a) | `estates_world_test.go` (6) |
| FR-CFG-099, 104, 105 | the three hash tests, `TestLoad_EstatesTable`, registry test (2) |
| FR-MON-038, 039, 008, 013, 036, 019, 046, MON-S3 | Phase 5 tests; `TestFork_Estates` (6) |

- **Golden files.** `testdata/golden/off/` unchanged (`TestNonInterference_EstatesOff`); `testdata/golden/estates/` new (module alone); `all/` does not exist yet and is not touched. The pinned hash of the all-off reference configuration is unchanged.
- **Fork-at-N.** `TestFork_Estates` (Phase 6.4).
- **Registry and catalogue.** Three `notBuilt` entries deleted; `ReliefPaid` is a row of OBS 4.1.5 and of rules 4.3 (already there) and leaves `notBuiltEvents` and the list of `render_test.go`; P1.6 is out of `notBuiltPositions` and in `replayRows` and `stepSites`.
- **Fuzz.** `FuzzReliefShares`; the loader fuzz corpus gets the `famine` files with the new keys (FR-REF-040).
- **Mutation.** `gomu` on `estates.go` and `internal/ledger/registry.go` only, in Docker (`--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`, `--workers 4`), per file; surviving equivalent mutants are listed in the PR.
- **Testing entry for the PR:** the table above, plus "off golden unchanged, estates golden recorded because relief is the only difference".

## Observability (observability-and-logging.md 9)

- **Events:** `ReliefPaid` (D352), class A, once per region per tick with a payment, only with an observer supplied, none with the module off. It is the only new event.
- **Warns:** none at run time. The configuration warning `grant_to_pool_no_outlet` is a `Warning` of `Validate` and so one `Warn` line of `NewWorld` like every configuration warning (existing path, no new log key; the path is the `Path` already logged). A relief overflow is a returned error: the world's existing `economysim.world.step.failed` path logs it.
- **Log keys:** none new (`logkeys.Region` exists).
- **Metrics:** none in the core and none in `otelobserver`: `ReliefPaid` feeds no instrument (OBS 6.3.5, line 2802; plan 22 D650); it is an event for observers.
- **Snapshot:** none. `RegionRow.MoneyPool` already shows the pool; the relief paid is in the event and in the welfare ring.
- **Allocation:** `TestRelief_NoAllocation` and the existing full-step limit (NFR-OBS-002) include the pass once the module is on in the benchmark.
- **`Info`/`Debug` in the tick:** none (observability rule: no `Info` in the tick).

## Risks and open items

| ID | Item | Owner |
| --- | --- | --- |
| O1 | CFG 8.2.1 says a field is a pair only when not zero, but its last bullet says a module's region field is a pair "whatever its value" when the module is on; that bullet would add `0006`/`0007` pairs to row 4 of 8.2.4 and break its pinned digest. The plan keeps the first bullet and FR-CFG-104 (D342) and edits the last bullet (Phase 1 item 3); a non-zero default of a module's field is allowed (D343). If the supervisor rules the other way, row 4 and every digest are re-pinned and D342 changes | supervisor |
| O2 | `ledger.Relief` takes the next Go constant after `LoanInterest`, not the place of the table of LED 7.2. Plans 5 and 6 also add declared reasons (`want_consumption`, flow reasons): the three must agree on one append order and on how `newLedger` assembles `Declared` (a fixed slice of constants cannot hold a per-flow reason). This plan assumes plan 5 makes `newLedger` build the list from the modules that are on and plan 6 extends it for dynamic reasons; the supervisor reconciles the three plans before code | supervisor, plans 5 and 6 |
| O3 | The seam with plan 6: plan 6 (D320, D337) makes `pool` `ErrModuleDependency` and leaves three edit points (kind switch, `recipients`, tag 13 pool position). This plan fills them after plan 6 merges; the names in Consumes are plan 6's own. Plan 5's Provides do not name plan 7 for the guarded forms, and plan 6's open item 1 asks for the `recipients` consume that is now stated | supervisor, plans 5 and 6 |
| O4 | `grant_to_pool_no_outlet` is assigned here (it reads `PoolRelease`); if plan 6 implements the MON warning list whole it must skip this one | supervisor, plan 6 |
| O5 | `warm-famine.toml` differs from the solver's table by the maker's jobs (`TestSolveSteadyState_Famine`), so the byte test of FR-REF-007 cannot include `famine` until the pack is rebalanced; it is added to `warmPacks` by plan 8 if it does not pass here. `expected/famine.toml` records the hashes as they are; plan 8 re-records it | plan 8 |
| O6 | S14 depends on the rebalance of N-314: pack `one` collapses today (growers earn about 45% of upkeep). `famine` is `one` with a worse staple, so S14 may collapse at every depth before the shock. The scenario records outcomes without a pass, as the Stage 1 scenarios do; the bands wait for plan 19. N-254 (a lot that gains units keeps its marked-down price) is to be judged on S14's staple price | plans 8, 19 |
| O7 | The ratio read at P1.6 is the total ratio including earlier relief, so a household held just above the line by relief stops drawing it, then falls back: a possible limit cycle (STB-L40 lists the loop). Not decided here; S14 reports the relief per tick so it is visible | STY, spec-estates owner |
| O8 | N-119: relief on a household that feeds itself counts its own-use receipts (SUB 7). This plan applies nothing; plan 10 (SUB) owns the decision | plan 10 |
| O9 | The register for a yield of 0 (`yield_bp` 0 in `s14-0.toml`): `SetYieldShockForTest` and the shock bound must accept 0; checked in Phase 6 and a spec question if not | this plan, then PRO |

**Note proposals (text only):**
- Close N-309 item (3) and N-310 item (3) when Phases 3 to 6 are merged, leave item (2) of each to plans 2 and 22, and record that item (1) of N-310 was done in Stage 1.
- N-295: `famine` now loads and joins the invariants table (REF 7.5); `five` and `one-demand` remain.
- New note, areas EST, CFG, ALL: "A module's field may default to a non-zero value because it is hashed only while the module is on; the zero-default rule of 2.10.5 binds core fields. The first case is `Region.PoolRelease` (100) and `Region.ReliefBelow` (10 000)."
- New note, areas EST, WEL: "Relief reads the total ratio, which includes the relief of earlier ticks; the first scenario that shows a relief cycle (STB-L40) decides whether it should read the earned ratio plus a fixed share."
