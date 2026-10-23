# Notes for future work

A register of things to take into account in work that has not been done yet:
constraints one spec places on another, decisions to revisit, traps found while
writing, and tasks that wait for a spec or for code. It is read **before writing
or changing a spec** (`AGENTS.md`), and it carries a read-receipt tag at the end
that an agent quotes when it starts that work.

It is not a second home for decisions or requirements. A decision lives in a spec
or an ADR, a requirement in a spec's Requirements section, and a spec's own open
questions stay in that spec. A note points at those and says *who needs to know,
and when*. If a note is only an open question of one spec, leave it there.

## How to use it

**Before writing or changing a spec.** Find the spec's three-letter area code in
the index below (the codes are in [specs/README.md](specs/README.md)). Read every
open note tagged with that code and every note tagged `ALL`. For each one, do one
of three things in your change, and say which in your summary:

- **apply** it (the spec now does what the note asks), then remove the note;
- **reject** it, with the reason, then remove the note;
- **leave it open**, if it is not yours to settle yet, and add what you learned.

**While working.** If you find something a future spec or a code change must take
into account, add a note in the same change. Do not leave it only in a chat or a
PR description.

**Rules for entries.**

- One note, one topic. Keep it to a few lines and link instead of copying.
- Tag it with the area codes of the specs that must read it. `ALL` is for every
  spec. `CODE` is for the first code change that touches the area. `EVD` is for
  the evidence file.
- Give a stable ID `N-NNN`. IDs are never reused. A settled note is removed from
  this file, not kept: git history is the record. The commit message that removes
  it says how it was settled (applied or rejected) and in which spec, ADR or code.
- Give the date it was raised (the format is `YYYY-MM-DD`) and where, so a reader
  can check it is still true. A note that names a section or a requirement may go
  stale when that changes: fix the pointer when you see it is wrong.
- Update the index in the same change, and replace the read-receipt tag at the
  end of the file with a fresh random value.

## Index by area code

| Area | Open notes |
|------|-----------|
| ALL | N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217, N-315 |
| AGT | N-043, N-067, N-087, N-101, N-111, N-137, N-139, N-142, N-144, N-153, N-156, N-164, N-188, N-196, N-214, N-228, N-236, N-237, N-244, N-245, N-247 |
| BEL | N-042, N-043, N-230, N-245 |
| CAL | N-030 |
| CAP | N-140, N-155, N-156, N-170, N-171, N-172, N-173, N-174, N-175, N-191, N-194, N-196, N-197, N-200, N-205 |
| CFG | N-016, N-061, N-075, N-093, N-123, N-132, N-165, N-182, N-218, N-221, N-276, N-281 |
| CHK | N-125, N-128, N-129, N-202, N-204, N-214, N-215, N-225, N-237, N-254, N-256, N-288, N-294, N-265, N-244, N-247, N-309 |
| CLB | N-015, N-124, N-134, N-135, N-158, N-162, N-169, N-199, N-207, N-208, N-214, N-220 |
| CLR | N-113, N-137, N-241 |
| CODE | N-019, N-089, N-127, N-198, N-214, N-215, N-216, N-217, N-218, N-222, N-224, N-228, N-243, N-276, N-277, N-294, N-295, N-300, N-244, N-245, N-247, N-254, N-290, N-309, N-310, N-311, N-312, N-315 |
| CRD | N-107, N-152, N-156, N-164, N-165, N-166, N-167, N-168, N-169, N-187, N-195, N-205, N-209, N-212, N-227 |
| DEM | N-080, N-089, N-145, N-226, N-228, N-277, N-314 |
| EMP | N-144, N-147, N-187, N-190, N-191, N-194 |
| EST | N-082, N-096, N-119, N-167, N-254, N-256, N-309, N-310 |
| EVD | N-018, N-133 |
| EXP | N-015, N-124, N-127, N-129, N-134, N-162, N-179, N-203, N-206, N-207, N-211, N-214, N-220 |
| EXT | N-108, N-112, N-113, N-214 |
| FIS | N-107, N-131, N-140, N-159, N-160, N-161, N-163, N-172, N-173, N-174, N-183, N-184, N-185, N-186, N-187, N-188, N-189, N-190, N-204 |
| GDS | N-039 |
| IND | N-130, N-159, N-178, N-201, N-206, N-208 |
| INF | N-110, N-112, N-156, N-157, N-170, N-214 |
| INS | N-176, N-179, N-213, N-227 |
| INT | N-090, N-129, N-130, N-131, N-136, N-143, N-161, N-164, N-177, N-194, N-204, N-205, N-289, N-296, N-290 |
| LED | N-021, N-125, N-130, N-159, N-225, N-226, N-290 |
| LND | N-140, N-156, N-160, N-166, N-187, N-190, N-191, N-205 |
| MER | N-120, N-122, N-136, N-173, N-176, N-180, N-189, N-190, N-210, N-214 |
| MIG | N-121, N-122, N-136, N-143, N-150, N-157, N-170, N-171, N-181, N-214, N-268, N-244 |
| MON | N-096, N-107, N-122, N-163, N-181, N-183, N-204, N-314 |
| OBS | N-125, N-127, N-130, N-159, N-288, N-289, N-294, N-300, N-309, N-310 |
| ORG | N-140, N-152, N-155, N-160, N-161, N-172, N-173, N-205, N-244 |
| PLC | N-131, N-140, N-160, N-177, N-190, N-194, N-214 |
| POP | N-096, N-097, N-109, N-132, N-153, N-192, N-268, N-244, N-290 |
| PRO | N-030, N-039, N-043, N-090, N-118, N-130, N-137, N-142, N-191, N-200, N-237, N-243 |
| REF | N-027, N-030, N-089, N-092, N-114, N-123, N-135, N-145, N-146, N-168, N-294, N-295, N-296, N-306, N-312, N-314 |
| REG | N-150, N-175, N-204 |
| RSK | N-140, N-155, N-161, N-176, N-205, N-211, N-213, N-214, N-226 |
| SAV | N-145, N-156, N-195, N-197 |
| SCL | N-030, N-198, N-222, N-300 |
| SET | N-140, N-192, N-268 |
| SKL | N-039, N-071, N-090, N-110, N-138, N-149, N-193, N-264, N-268, N-244, N-305 |
| SPN | N-043, N-089, N-096, N-118, N-122, N-138, N-148, N-153, N-172, N-192, N-193, N-264, N-265, N-267, N-268, N-305 |
| STB | N-028, N-032, N-039, N-042, N-043, N-045, N-067, N-086, N-087, N-092, N-099, N-113, N-122, N-124, N-151, N-162, N-169, N-179, N-210, N-211, N-214, N-296, N-267, N-312, N-314 |
| STO | N-155, N-160, N-166, N-181, N-186, N-191, N-194, N-205, N-221, N-244 |
| STY | N-015, N-027, N-030, N-084, N-085, N-086, N-092, N-113, N-123, N-132, N-133, N-135, N-145, N-151, N-162, N-168, N-177, N-178, N-199, N-201, N-203, N-206, N-207 |
| SUB | N-121, N-228, N-268, N-244 |
| TEC | N-214 |
| TRN | N-120, N-130, N-136, N-137, N-142, N-143, N-155, N-170, N-174, N-175, N-176, N-180, N-184, N-189, N-194, N-204, N-226, N-279 |
| VIA | N-015, N-061, N-071, N-080, N-085, N-093, N-109, N-114, N-123, N-134, N-146, N-158, N-165, N-180, N-276, N-277, N-279, N-281, N-306, N-311 |
| WEL | N-016, N-028, N-099, N-118, N-119, N-130, N-147, N-148, N-171, N-183, N-214 |
| WLD | N-124, N-125, N-144, N-204, N-214, N-216, N-226, N-230, N-276, N-288, N-265, N-309, N-310 |

## Open notes

### N-015 Placeholders to calibrate once something has run

- **Areas:** VIA, STY, CLB, EXP
- **Raised:** 2026-10-09.
- **Note:** These numbers are guesses, not results, and nothing has been run:
  `MinAgents` of 10 (`spec-scale.md` 5.2); the 10% scale-consistency tolerance
  (7.2); 200 000 agent-ticks a second (6.2); the bounds in `spec-config.md` 4.1;
  the validation time in NFR-CFG-001; the per-field bounds in `spec-goods.md` 8;
  the `TargetSpread` default of 0 (`spec-welfare-and-wages.md` 5.2). The Doran and
  Parberry paper is unread and bazaarBot has not been run, so "stable" has no
  reference yet (`bazaarbot-reference.md` section 12; `vision.md` section 6).
- **Update (2026-10-10):** the Doran and Parberry paper is read (journal version; use `ianparberry.com/pubs/econ.pdf`, the LARC URL 404s), so "stable" has a prior definition: the filter criteria of the paper's section VII, listed in `spec-beliefs.md` section 19. Still true: bazaarBot has not been run and no number has been validated. New placeholders: the 15 belief parameters of `spec-beliefs.md` section 11 (most importantly `CostFloor` 10 000, `BeliefAdjustProb` 5 000, `BeliefStep` 200 and the stock band 5 000 to 15 000), and `TraineeYield` 5 000 and the production bounds of `spec-production.md` 11.
- **Update (2026-10-10):** AGT placeholders (`spec-agents.md` 12): `CoverTicks` 3, `InputJobs` 1, `OutputCoverTicks` 1, `FavourFloor` 2 000, `ReserveTicks` 3, `SpendMaxShare` 5 000, `SpendHalfTicks` 10, `RestBand` 500, `ProduceHaltRatio` 30 000, `BasketTolerance` 1 000, `StartMoney` 10 000, `MaxUpkeepItems` 16, and the worked examples of section 16. VIA placeholders: `StockCoverTicks` 5, `MoneyBufferTicks` 10, `ResidualWarnBP` 1 000, `MaxViabilityPivots` 100 000, `MaxViabilityCells` 4 000 000 and the prototype numbers of `spec-viability.md` 15; the Go speed of the exact solver is not measured. STB adds only placeholders: every threshold of the failure regimes, the example values of its proposed parameters, and every number of its models.
- **Update (2026-10-10, round 3):** new placeholders: `CostFloor` 7 000, `BeliefEvidenceShare` 5 000, `WelfareWindow` 30, `WindowCycles` 2, `MinWorkShare` 4 000, `MaxWelfareWindow` 3 650; scale cost model constants 2 us, 1 us and 8; the rescaled example quantities of `spec-production.md` 12; the skill-barrier tolerance 200 bp and queue discount; and the bounds of `spec-config.md` 4.1 and `spec-estates.md` 3.
- **Update (2026-10-10, round 4 and 5):** new placeholders: every default of `spec-demand.md` 9, `spec-spawner.md` 11 (`EntryShare` 50, `ExitShare` 100, `Renewal` 10 000, `EntryPoolShare`), `spec-reference-chain.md` (every pack quantity, the censuses, seeds and run lengths) and the thresholds of `spec-stability.md` 7 and 11.
- **Update (2026-10-10, RES):** new placeholders: `HorizonYears` 50, `PlanShareBP` 6 000, `InitialBP` 10 000, the bounds `MaxResources` 256, `MaxEndowment` 10^15, `MaxRegenBP` 100 000, `MaxRegenScaleBP` and `MaxShockBP` 100 000, `MaxHorizonYears` 1 000, the warning thresholds 11 000 bp (`no_comparative_advantage`) and 2 000 bp a tick (`regen_per_tick_high`), every number of the worked examples (16), of the models (17) and of the five-region sizing (18).
- **Update (2026-10-10, INF):** new placeholders: `RumourErrorBP` 500, `RumourMaxAge` 60, the bounds `MaxInfoSpeed`, `MaxRumourErrorBP`, `MaxRumourAge`, `MaxRumourCells`, `MaxRumoursPerView`, the cost counts of `spec-information.md` 7.8 and every number of its models and examples.
- **Update (2026-10-10, SUB):** new placeholders: `EnterTicks` 2, `ExitTicks` 3, `RetryJobs` 10, the bounds `MaxOwnUseRecipes` 8, `MaxOwnUseTicks` 1 000 and `MaxRetryJobs` 1 000, every number of the worked examples (16) and of the model (11.2, 17).
- **Update (2026-10-10, MIG):** new placeholders: the nine defaults of `spec-migration.md` 11, the factor 3 of the warning `migration_flow_window`, and every number of the worked examples (15) and of the models (16).
- **Update (2026-10-10, MER):** new placeholders: `MaxOpen` 1 000, `LotUnits` 1, `FeeShareSpreadBP` 2 500, `MaxSourceAge` 2, `TimeCostBP` 10 000, `ConsigneeCutBP` 500, `TakeMarginBP` 1 000, the bounds `MaxLotUnits` 64, `MaxContracts` 1 000 000 and `MaxTimeCostBP` 100 000, the warning thresholds of 14.2 and every number of the models and examples.
- **Update (2026-10-11, STY):** new placeholders: every prior and threshold of `spec-stylised-facts.md` (3.3, 6.2, 11), the headrooms 2 500 and 5 000 bp, `min_windows` 3, `stale_max_bp` 1 000, `active_min_bp` 5 000 and every number of its models.
- **Update (2026-10-11, EXP):** EXP is the means to calibrate and sets no number; the bounds of `spec-experiments.md` 13.2 and the budget of 10 are labelled placeholders.
- **Update (2026-10-11, CLB):** CLB holds no value either. Its registry gives every placeholder a class (`spec-calibration.md` 4.3, 6) and says that nearly every search range is unknown (`U`: 96 of the 105 knob and damper rows); "calibrated" now means that a record names the value and the runs it rests on (8). New placeholders: the procedure constants of `spec-calibration.md` 13 (grid factor 2, explore levels -2 to +2, `screen_share_bp` 500, 10 search and 50 confirm replicas, search blocks of 100 from seed 101, three attempts before a review, `alias_tol_bp` 500, a claim set of 49 rows) and the cost of one pass (7 292 runs, 39.5 core-hours, from the placeholders of `spec-scale.md` 6). A damper knob whose default is its off value (`TargetSpread`, `HouseholdCap`, `Refuge`, a flat `Abundance`, `Limit`, `Exempt`) means the damper is not in force by default: a calibration moves it away from off when a fact shows the loop, never back (4.3.1); WEL and STB read D08 with this. `MinAgents` now has a validation bound, 1 to `MaxHouseholds` (`spec-scale.md` 5.2, FR-SCL-020), so its grid is finite; the value 10 stays a placeholder.
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** the fourteen specs add only placeholders, each labelled in its own section: SET `BandBP` 1 000, `Dwell` 30 and the bounds `MaxClasses` 16, `MaxFlags` 64, `MaxLimits` 64, `MaxBandBP` 5 000, `MaxThresholdBP` 100 000; EMP `MaxHires` 4, `WorkerShareBP` 5 000, `HireYieldBP` 10 000, `MaxOpen` 1 000, `MaxEstablishment` 1 000, `MaxHireRecords` 1 000 000; LND `RentStepBP` 0, `RentFloorBP` 0, `RentCeilingBP` 20 000, `MaxHolderShareBP` 10 000 (off), `RentDualTolBP` 5 000 and the bounds `MaxRentStepBP` 10 000, `MaxRentCeilingBP` 100 000, `MaxHoldings` 4 096, `MaxTitleTransfers` 1 024; POL `MaxJurisdictions` 1 000, `MaxPolityDepth` 8; CRD `MinPrincipal`, `ShieldTicks` 0, `WriteOffTicks` 0, `MaxClaims` and the bounds `MaxCreditRateBP` 100 000, `MaxCreditTerm` 3 650, `MaxOpenClaims` 1 000 000; CAP `MaxDurability` 1 000 000, `MaxFacilityCapacity` 100 000, `MaxBoostBP` 100 000, `MaxTotalBoostBP` 200 000, `MaxFacilities` 100 000, `MaxFacilitiesPerAgent` 64; RSK `MaxRiskRows` 4 096; SAV `SafetyTicks` 2, `HurdleSpreadBP` 20 000 (N-199), `CommitBandBP` 2 000, `PrecautionShareBP` 0, `PrecautionCapTicks` 30, `HurdleBP` 15 000, `PaybackTicks` 180, `RepairBelowBP` 5 000 and the unmeasured cost of the drawdown (N-198); TEC `Base` 100 000, `DoublingBP` 200, `CapBP` 3 000, `CatchUpBP` 2 000, `Reach` 100, `TrendBP` 40 and 50, `DiscoveryBP` 1 000, `StepBP` 300 and nine bounds (`MaxTechFields` 64, `MaxFieldRecipes` 64, `MaxTechBase` 10^12 and six more); INS `PayoutTicks` 1, `MaxCoverRequests` 4 and four bounds; every default, bound, cadence, target and gain of STO and PLC; and every number of the examples, cost tables and models of the fourteen.
- **Update (2026-10-12, CFG, ADR 0009):** new placeholders from measurement of the TOML library, not from real packs: `MaxDocuments` 16, `MaxDocumentBytes` 4 MiB (was 16 MiB), `MaxMergedBytes` 16 MiB, `MaxLineBytes` 256 KiB, `MaxNestingDepth` 16 (key segments and value nesting together, from the deepest pack key plus 2) and `MaxTableKeys` 10 000 (tied to `MaxRecipes`; the library is quadratic in the keys of one table): `spec-config.md` 4.1, ADR 0009. `MaxLabelBytes` 256 is the one label bound (FR-GDS-021).
- **Update (2026-10-13, CFG):** new placeholder: `MaxYields` = `MaxRegions * MaxRecipes` (10 000 000), the count bound of `Config.Yields` that FR-CFG-101 asks for (`spec-config.md` 4.1); derived and unmeasured. Both operands are placeholders, and a yield table of that size is not a realistic pack: lower the product when a real pack gives a size.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open, with a measurement owed. `StockCoverTicks` 5
- **Update (2026-10-18, plan 1-10 Phase 7):** the first measurement of the Go solver. Measured on the development machine (one run, `go test -bench`, Go 1.25, 32 logical CPUs, one core used by the solver): `SolveSteadyState(five, WithFreeTrade)` takes 1.37 s with 202 pivots and allocates 474 MB in 10.2 million allocations per call (the prototype took 1.7 s); `Validate` of the cold `five` takes 0.59 s and `Warnings` 0.46 s after it. All five placeholders above stay as they are; the largest tableau entry in bits was not recorded. Left open., `MoneyBufferTicks` 10, `ResidualWarnBP` 1 000, `MaxViabilityPivots` 100 000 and `MaxViabilityCells` 4 000 000 stay placeholders; Phase 7 measures the Go speed of the solver (the first ever) and records the pivots and time of `five` here. No number is claimed stable.

### N-016 Keep the parameter registry true

- **Areas:** ALL, CFG, WEL
- **Raised:** 2026-10-09, `spec-config.md` 9.3.
- **Note:** Every new configuration field needs a row in the parameter-owner table
  (architecture.md 9.5, 9.7). Known gaps: the welfare window `T` (default 30,
  `spec-welfare-and-wages.md` 2.3.1) has no row; the tolerance for matching a
  region's `ReferenceBasket` to its reference prices (`spec-currency.md` 5.3) has no
  name or default; `spec-clearing.md` calls `MaxPrice` and `MaxQuantity`
  "configured" although they are bounds in `bounds.go` (`spec-config.md` 4.1).
  A role's default `ExpectedWorkTicks` of 250 becomes invalid if the calendar gives
  fewer ticks a year (`TicksPerYear` is derived, `spec-calendar.md` 3.2;
  `spec-config.md` open question 5). The calendar fields have rows in the
  parameter-owner table since 2026-10-10.
- **Update (2026-10-10):** production added nine rows and beliefs fifteen to the parameter-owner table; no new unnamed parameter. `YieldProfile` has two owners over time (PRO now, RES later); PRO is listed as owner until RES exists.
- **Update (2026-10-10):** AGT adds 17 rows and names `BasketTolerance`, closing one gap of the known list (the unnamed basket tolerance of `spec-currency.md` 5.3); VIA adds `WarmStart`. VIA's solve options (`StockCoverTicks`, `MoneyBufferTicks`, `ResidualWarnBP`, `MaxPivots`, `Lanes`) are not `Config` fields and are NOT in the parameter-owner table, because the registry test of testing.md 6.7 requires a field behind every row; they live in `spec-viability.md` 11. STB adds no field; its four proposals (`MinWorkTicks`, `ProdStopRatio`, `EntryShare`, `RoleHysteresis`) need rows if adopted.
- **Update (2026-10-10, round 3):** `spec-config.md` 3.7 lists every field and 9.2 says a row is a `Config` field; the parameter-owner table now has a row for each (generated from 3.7, plus `WelfareWindow`, `WindowCycles`, `MinWorkShare`, `BeliefEvidenceShare`), and the solve options of `spec-viability.md` 11 are deliberately not rows. Closed: the `BasketTolerance` name, the `MaxPrice` and `MaxQuantity` wording (they are bounds), `ExpectedWorkTicks` against a derived `TicksPerYear` (an error, FR-CFG-048). Remaining: a carriage `m_bp` has no name; a test must verify the table against the code once it exists.
- **Update (2026-10-10, round 4 and 5):** the parameter-owner table gained the demand and spawner fields (`World.Spawner`, `Role.Closed`) and a row for every field of `spec-config.md` 3.7, regenerated after the new modules. No unnamed parameter was found in AGT, WEL, DEM, SPN or REF. A test must still verify the table against the code once it exists.
- **Update (2026-10-10, REG):** REG adds `Edge.From`, `.To`, `.Distance` and `RegionPrice.Region`, `.Commodity`, `.Price` to the parameter-owner table, and the presentation-only fields `Region.Label` and `World.DistanceLabel` to the allow-list of `spec-config.md` 8.3. No unnamed parameter was found. REG adds no tunable with a default (a distance and a price have none; the default of the sections is "no edges, no overrides"). The edge status of `spec-regions.md` 7 is world state, not a field, and has no row.
- **Update (2026-10-10, MON):** the fields of `MoneyFlow` have rows in the parameter-owner table; no unnamed parameter; the runtime `Scale` is state and not a row.
- **Update (2026-10-10, POP):** POP adds nine rows (`BirthStake` has an explicit name); no gap found. The registry test is still to be written.
- **Update (2026-10-10, TRN):** TRN adds `Modes` and ten fields of a mode, each with a default or "required", a bound and an owner. No unnamed parameter; it removes the unnamed carriage margin `m_bp` of `spec-carriage.md` (the haul unit's price is anchored to the producing recipe's `UnitCost`, `spec-transport.md` 8.5).
- **Update (2026-10-10, RES):** RES adds the rows of the module, 11 fields of `Resource` and 5 of `Endowment` to the parameter-owner table and changes the owner of the yield rows and of `Draws`; no unnamed parameter was found. The registry test is still to be written.
- **Update (2026-10-10, INF):** INF adds five rows (`Modules.Information`, `InfoSpeed`, `RumourErrorBP`, `RumourMaxAge`, `Role.Informed`), each with a default or "required", a bound and an owner; no unnamed parameter.
- **Update (2026-10-10, EXT):** EXT adds its rows to the parameter-owner table; no unnamed parameter. The shock, drift, carry and cumulative totals are runtime state and not rows.
- **Update (2026-10-10, SUB):** SUB adds rows for the module, its default list, the role rows, the key and list of a row, and three tunables (`spec-subsistence.md` 12); no unnamed parameter was found. The registry test is still to be written.
- **Update (2026-10-10, MIG):** MIG adds rows for the module, nine tunables, the region-keyed rows and `Role.Fixed` (`spec-migration.md` 11); no unnamed parameter was found. The registry test is still to be written.
- **Update (2026-10-10, MER):** MER adds rows for `Modules.Merchants`, `MaxOpen` and the nine fields of the block `Role.Merchant` (`spec-merchants.md` 3.2, 13); `MerchantFeeShare` keeps its row. No unnamed parameter was found.
- **Update (2026-10-11, WLD):** WLD adds no field; the one row is `World.Clearing`, owned by `spec-clearing.md` 8 as `World.Spawner` is the spawner's, now in the parameter-owner table and in `spec-config.md` 3.7 (FR-CFG-081). `WithTargetSpread` keeps its row, with the Scope cell "World (option; hashed, `spec-config.md` 8.1)".
- **Update (2026-10-11, OBS, CHK, INT, IND, STY, EXP):** none of them adds a `Config` field, a default, a bound or a row. `WithReseed` is an option and not configuration (`spec-config.md` 9.2). A path swept by EXP is a `Config` field and so already a row (`spec-experiments.md` 11.3.1); EXP's warning `no_effect` waits for a pure hash function of CFG's (14.4).
- **Update (2026-10-11, CLB):** CLB adds no field, default, bound or owner row, and adds the **registry of knobs** (`spec-calibration.md` 6), a second reader of the owner table: it classifies each owner row and each field of `spec-config.md` 3.7, and the registry test of its 11.1 checks both directions (FR-CLB-002). The solve options of `spec-viability.md` 11 are rows of that registry and not of the owner table, which settles the disagreement between VIA 11 and CFG 9.2 for CFG (`spec-viability.md` 11 and `spec-config.md` 9.2, 9.3 say so). A spec that adds a field adds its registry row in the same change, and a field that a path of `spec-viability.md` 11 names needs its flag `in` or `chk` (FR-CLB-004, FR-CLB-009). The rows `Taste.Role` and `RegionTaste` (`Region`, `Want`, `Scale`), which `spec-config.md` 3.7 listed and the owner table lacked, are now owner rows. Still to be written: the two registry tests (testing.md 6.7 and `spec-calibration.md` 11.1).
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** the fourteen specs add their rows to the parameter-owner table (the module switch, its tunables, its bounds, the role blocks and the keyed tables; each in the parameter section of its own spec) and none found an unnamed parameter. Two rules were kept: `RentDualTolBP` is a rule constant (`spec-land.md` 5.6.2) and a runtime scale, carry or directive is state and has no row. `FlowTarget` is MON's type with its three old fields; STO owns the field `Store`, FIS owns `Jurisdiction`, and no spec adds a `Label` row (CFG 9.2). Known gap: STO states no file layout for `Base` and the role rows, only `[modules.storage.class.<id>]` (`spec-storage.md` 7.3; see N-102). The registry test (testing.md 6.7) is still to be written, and so is the second registry test of CLB.
- **Update (2026-10-12, CFG):** the owner table gained rows for the element types (`Month`, `Season`, `SeasonYield`, `Item`, `Output`, `Tool`, `SeedPrice`, `SeedWage`, `WarmGroup`, `JobSpec`, `WarmPool`), split `Scale` and `Upkeep` into two rows each (World and Group; World and Role), replaced the `YieldProfile` row by the `Yields` section row, added `Estates` of `Modules`, and respelled scopes to Go type names; the registry test is staged by one explicit `notBuilt` list (FR-CFG-028, architecture.md 9.7, testing.md 6.7). The known gap "STO states no file layout for `Base` and the role rows" is closed by rule 6.2.1 of `spec-config.md` (`[modules.storage.base]` and `[modules.storage.role.<id>]`); STO confirms it (N-221). `Group.Scale` 0 means "the world's `Scale`" and is hashed as written (N-218).
- **Update (2026-10-15, plan 1-3 Phase 1):** read for the clearing row: it adds no field; the `World.Clearing` row and its registry-test line stay as they are, and `MaxPrice` and `MaxQuantity` stay bounds. Left open for the other owners.
- **Update (2026-10-16, plan 1-4 Phase 1):** the beliefs row adds no field, default, bound or validation rule (Stage 0 built all sixteen parameters); the registry test and its `notBuilt` list do not change. Left open for the other owners.
- **Update (2026-10-16, plan 1-5 Phase 1):** the production row adds no configuration field, default, bound or validation rule (Stage 0 built them all), so the registry test and its `notBuilt` list do not change. Applied for this row; left open for the other owners.
- **Update (2026-10-17, plan 1-6 Phase 1):** read for the welfare row: it adds no field and no default (every welfare field has its owner row from Stage 0) and deletes no `notBuilt` entry; `WelfareWindow` has its row since 2026-10-10. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** read for the agent row: it adds no configuration field, default or bound (every AGT field has its owner row since Stage 0 and no `notBuilt` entry is left for it); the belief-parameter group of the views (D82) is a view and not a `Config` field, so it adds no row. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read for the estate row: it adds no configuration field and no default (`EstateMarkdown` is a Stage 0 row with hash field 5; `PoolRelease` and `ReliefBelow` arrive with the module, `spec-estates.md` 3) and deletes no `notBuilt` entry. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The spawner row adds no field and no default; the registry test and its notBuilt list do not change.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. This row adds no `Config` field and no row of the parameter-owner table: the solve options are rows of CLB's registry (`spec-viability.md` 11, FR-CFG-047) and `WarmStart` has its row. Phase 5 checks the registry test is unchanged.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The row adds no configuration field. `spec-config.md` 3.7 and 9.2 now say that the section type of `Config.World` is `WorldParams` and that the scope cells of `specs/README.md` (read as Go type names) change with the rename in Phase 2, not before (a cell naming a type that does not exist fails the registry test).
- **Update (2026-10-21, anchor fix):** applied for one field: `World.AnchorHold` has its row in the parameter-owner table, its place in `spec-config.md` 3.7 and 8.2.2 (id 46), a bound and a failing-case test, and the registry row K-PRO02 of `spec-calibration.md`; the registry test passes with no `notBuilt` change. Left open as a standing rule.

### N-017 Words and names

- **Areas:** ALL
- **Raised:** 2026-10-09.
- **Note:** Names are setting-neutral (`AGENTS.md`). Traps already hit: `Weight` is
  the households an agent stands for, so a commodity has `Mass`, not weight; a
  "day" in the specs means a tick and a "year" is `TicksPerYear` ticks
  (`spec-currency.md` 2.4, now derived from the calendar); a calendar day is a
  different unit, since a tick may be a part of a day or several days
  (`spec-calendar.md` 2.5); "consumer" means the library consumer (N-001); a "pack"
  is configuration, a "module" is an optional mechanism. Add new terms to the glossary
  list in `specs/README.md`, and to `doc.go` before code uses them.
- **Update (2026-10-10, REG):** terms fixed: region, edge, route (an edge), distance (a length, never a duration), component, edge status, relocation. "Settlement" names what a region is, not a type. Setting words appear only in the flagged examples of `spec-regions.md` 12.
- **Update (2026-10-10, POP):** a birth and a death are the formation and dissolution of a household, not of a person; "stake" is the money a born household brings, not an investment; "year" is `TicksPerYear` ticks.
- **Update (2026-10-10, TRN):** the agent that moves is the *traveller*; a *carrier* is the role of `spec-carriage.md` whose households make haul units. Example mode names (`foot`, `wagon`, `coach`, `crawler`, `lift`) are examples in two invented settings and are flagged so in the spec.
- **Update (2026-10-10, RES):** terms fixed: resource (not a commodity), endowment (a resource in a region), stock, capacity, regeneration, shape, refuge, abundance, plan share, ration, sustainable draw. Setting words (land, forest, ore, ice) appear only in the flagged examples of `spec-resources.md` 16 and 18. "Draw" now has a precise meaning.
- **Update (2026-10-10, INF):** terms fixed: rumour, price record, observation, vintage, lag (of a rumour), information speed, horizon, informed (role), source.
- **Update (2026-10-10, EXT):** terms fixed: outside, outside node, gate, line, side (of a line), mid, quote, margin, counterparty (buyer, seller), Outside (account kind), outside sector, import, export, balance of trade, list value, drift, shock, availability window, capacity, cap, sweep. Setting words appear only in the flagged examples of `spec-external-sector.md` 18.
- **Update (2026-10-10, SUB):** terms fixed: subsistence floor (the whole mechanism), own-use recipe, own-use mode (the flag `OwnUse`), covered set, cover (stock in ticks of upkeep), afford, secure ticks, retry (the market test), coverage (`Cov`), `FloorBP`, blocked. The word floor has six uses in the specs (work floor, cost floor, resolution floor, `MinWorkShare`'s floor, the floor of agents, this one): SUB says "subsistence floor" for the mechanism and "own-use" for its parts. Setting words (glean, grain, common, tray) appear only in the flagged examples of `spec-subsistence.md` 16.
- **Update (2026-10-10, MIG):** terms fixed: migration (this module's moves), migrant (an agent on a journey), journey, earned ratio (the welfare ratio without relief), settled, margin and width (of a move), horizon, estimate (of a destination), pushed (household), outflow cap, pipeline cap, provisioning. "Road" is not used (a journey). Setting words appear only in the flagged examples of `spec-migration.md` 15.
- **Update (2026-10-10, MER):** terms fixed: merchant, planned trip, lane (of a merchant), candidate, lot, lot cap, cargo set, trade inputs, fee share, bar, cautious revenue, break-even, lane lock, delivery offer, delivery take, payer, deliverer, round (of contracts), delivery contract, settlement. Setting words appear only in the flagged examples of `spec-merchants.md` 19.
- **Update (2026-10-11, WLD, OBS, CHK, INT, IND, STY, EXP):** terms fixed, each in its spec's list and in the glossary of `specs/README.md` (several words have two senses, such as window, entry, key, run, step, level, gauge, record, point, tier and index, and the glossary gives each its qualified spelling): WLD: tick boundary, phase, position, entry (of a tick), slot, between ticks, cadence, rotation, bucket, Weyl value, keyed tie, failed world, word, the world's draw, creation draw, construction phase, carried remainder, window (a memory), pending totals; OBS: names table, volume class, recorder, level, delta instrument, region state, snapshot row, pending totals, construction flow, run, pattern A and B, `warnRun`, `NoCommodity`, `WantID`; CHK: checkpoint, restore, format version, section, frame, header, trailer, digest, canonical encoding, one spelling, sparse row, construction flag, extension state, `Stateful`, reseed, salt, branch, fixture; INT: agent state, forced move, market state, slot state, query class, entry (of a setter), key (of a slot); IND: indicator, preset, aggregator, tally, window (of ticks), household row, price table, base window, index (basis points of a base), Laspeyres, Paasche, Fisher, chain index, clamp, stale share, cohort share, value added (of a region), production approach, expenditure approach, income view, gross accumulation, written (goods), carriage, delivered price, freight, trade valuation, retirement basis, binding counters, reconciling item; STY: stylised fact, band, prior, setting seeds, gate seeds, hold-out set, headroom, acceptance class, exact fact, ordinal fact, banded fact, report fact, pack-asserted fact, confidence (of a band), blind-spot field, detector, verdict, unknown (verdict), onset, healthy start, provoked run, separation protocol, stability check, stability record, verdict record, tier; EXP: experiment, factor, level, change (of a path), range factor, composite factor, base pack, overlay (document), design, point, replica, seed rule, run key, unit (of work), plan, record, result, group, window (sample window), transient, series, final reading, measure, criterion, pass share, reduction, step (script step), script, branch, variant, continuation, burn-in, aligned (through a window), aligned horizon, stream fingerprint, pairing gain, effect range, elasticity (in basis points). The type of the harness is `experiment.Spec`, not `experiment.Experiment` (architecture.md 2.5). Setting words appear only in the flagged examples.
- **Update (2026-10-11, CLB):** terms fixed, each in `spec-calibration.md` 23 and in the glossary of `specs/README.md` (two have a second sense, and the glossary gives each its qualified spelling): knob, damper strength, structural choice, registry (of knobs; the parameter-owner table is a different thing), search range, evidence anchor, derived condition, base value, grid (of a knob), neighbour, edge (of a grid; REG's edge is of the region graph), screen, screened in, dormant, invisible, alias group, gauge (of a calibration; OBS's gauge is an instrument), composite (knob), search block, confirm seeds, feasible region, minimal change, grid distance, Pareto set, known miss, working target, working threshold, calibration overlay, calibration record, lineage, re-entry, stage (of a calibration), attempt, off value, structure group, scope token, overlay document, claim set, sensitivity matrix, scaling check, stage stop, program stop. Gate, hold-out, finding, stability record, elasticity and effect range are used as STY, EXP and IND define them and are not redefined. Setting words appear only in the flagged examples. The terms go to the `doc.go` glossary before code uses them.
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP):** terms fixed, each in its spec and in the glossary of `specs/README.md`: STO: store, store class, base room, room (mass room, bulk room), limited (account), load, pledge, free room, stored load, commit (room is committed at the decision); "home" is not used. EMP: employment, employer, worker, hire, vacancy, positions, labour offer, reservation (wage), worker share, labour round, escrow of a hire, unemployment (a measure), hired yield. SET: settlement class (class), enter and leave threshold, band, dwell, settlement review, flag, limit, gate, class floor (qualified: "floor" has six uses already), held change, headroom, ceiling. LND: tenure, title (a claim of ownership of a share of an endowment, never a rank or an honorific), holder, commons share, entitlement, let, lessor, tenant, tier (of the ration; not STY's tier), posted rent, rent in force, enclosure, release, apportionment; "land" is a file name and an example only. POL: jurisdiction, polity, member, root, chain, covers, crossing, border, seat, governed, treasury (the account kind and its pseudo-agent; the region's account is the pool, N-163), combining mode (innermost, added, scaled). CRD: credit, loan, request, offer, lender, borrower, posted and effective rate, rate scale, term, principal, interest, shield, waterfall, passed, written off, forgiven, exposure cap, cash gap, round (its lane is a region: "lane of the credit round"). CAP: facility, facility class, kind (site, route, ground, store), condition, decay, use wear, effect fraction, gain (of a layer), ruin, repair, build, builder, commissioning, install, pending; "durability" and "capacity" are qualified "of the class". Setting words appear only in the flagged examples of each spec.
- **Update (2026-10-13, RSK, FIS, ORG, SAV, TEC, PLC, INS):** terms fixed, each in its spec and in the glossary: RSK: risk row, route row, store row, exposure, strike, severity, common row, unit row, effective probability, loss share, kept share, expected loss, risk scale, total loss, trip account. FIS: fiscal flow, authority, tax (head tax, holding tax, remittance), duty (sales tax, purchase tax, tariff), toll, payout, standing order, base (of a duty), pro rata cut, treasury sector; qualified: window, release, payer, levy, base, scale. ORG: body, member, eligible (household), places, headroom (of a body), seniority, membership round, held completion, right, activity, prohibition, suspended, legal price, control, rate limit, legal range, clamp; qualified: place, right (not LND's title), control (not the controller of L121), cap, floor, round, need. SAV: saving (the module), saver, savings target, precaution buffer, drawdown, goal, project, margin test, hurdle, payback horizon, commit band, worth, idle money; "investment" is a build or a repair only, "hoard" only the measure, and "floor" has a seventh use (qualified). TEC: technology (not VIA's technology matrix), technology field, cell, experience, learned gain, advance, gain, technology factor, doubling, base, trend, discovery, contact, reach, frontier, catch-up, cap (of a field); "gain", "cap" and "field" are qualified. PLC: policy (the interface), policy actor, lever, mandate, reach, decision, due, cadence, mailbox, directive (of an actor; AGT's directive is another object), gauge, reading, target, deadband, span, move, rest, reaction, schedule. INS: cover, route and store policy ("cover policy" in prose), underwriter, insured, premium, sum insured, maximum payout, loading, store rate, capacity, open liability, loss value, due, payout, payout claim, shortfall, round of cover; "claim" stays the ledger's. Setting words appear only in the flagged examples of each spec. The terms go to the `doc.go` glossary before code uses them (N-019).
- **Update (2026-10-12, ADR 0007 and 0008):** terms fixed: module file, touch point, position list, fan-in (a composition of per-module steps), draw site, site constant, stream key, keyed stream, effective seed (the seed after a reseed). "The world's stream" and "positional" are retired for the draws once WLD 7 is rewritten (N-214); "word" keeps its sense within one stream.
- **Update (2026-10-13, Phase 1 of the ledger row):** terms fixed in `spec-ledger.md` 3.3.1, 3.4 and 7.2.2: built-in row (a registry row no module owns, which exists in every world), module row (declared only while its module is on), `B` (the count of the boundary accounts of the built-in group, `1 + 7 * C`), creation order (of the boundary accounts: commodity-major, not the reason-major order of `BoundaryFlow`). Add them to the glossary of `specs/README.md` and to `doc.go` before the ledger code uses them, with the terms of the plan (ledger, account, boundary account, transfer, reservation, flow totals, construction phase, construction bucket, lot, born, earmark, durable, claim, creditor, debtor, balance sheet).
- **Update (2026-10-12, Phase 1 of the Stage 0 plan):** terms to add to the glossary of `specs/README.md` and to `doc.go` before code uses them: draw site, site constant, stream key, keyed stream, effective seed (WLD 7; `word` is "word (of a stream)" and `reseed` is "reseed (of the seed)"); guard pass (the counting pass of the loader, ADR 0009), field id (of the hash), zero value as off value (the one rule of CFG 8.2 and OBS 9.5.2), pool policy and the estates module (`Modules.Estates`, the pool policy only), `notBuilt` list (the staged form of the registry and catalogue tests); `discardObserver` is unexported and `warnRun` is the Go name of the bounded `Warn` run constant (earlier text wrote `warnRun`).
- **Update (2026-10-15, plan 1-3 Phase 1):** terms of the clearing row, to be added to the `doc.go` glossary in Phase 4 before `order.go` and `clearing.go` use them (Phase 1 writes no code): market, order, bid, ask, limit price, trade, admission, reservation, available, tie-break key, crossing, market price, price age, unmatched, market report, clearing. `Market`, `Order`, `Trade` and `Clearing` leave `notYetTerms` of `glossary_test.go` in that change. "Treasury" and "pool" are used in the senses of N-163. A standing note; left open.
- **Update (2026-10-16, plan 1-4 Phase 1):** terms of the beliefs row are in the `doc.go` glossary before `internal/beliefs` uses them: price belief (`Belief`: centre and width), centre, width, effective range, belief floor and belief ceiling (`Bounds`), reference band, cost band, adjustment gate, unit of adjustment, evidence move, stock ratio, fresh price, price window, window mean, window low, window high, position in the window, seed price, target price, fill share, re-clamp. "Floor" and "ceiling" are qualified as the bounds of a belief. `limit price` and `market price` were already there (the clearing row). A standing note; left open.
- **Update (2026-10-16, plan 1-5 Phase 1):** the terms of the production row (job, executor, principal, residue, effective factor, site factor, yield shock, pending output, cost anchor, unit cost, create-and-dispose) go to the `doc.go` glossary before `internal/production` and `production.go` use them. A standing note; left open.
- **Update (2026-10-17, plan 1-6 Phase 1):** applied: the terms the welfare code uses go to the `doc.go` glossary before use (welfare ratio, window (qualified: the welfare window), working tick, working household-tick, own-use household-tick, observed wage, pooled regional wage, observed premium, earned ratio, work share, work floor, target ratio, record (of a tick; qualified from CLB's record), ring). A standing note; left open.
- **Update (2026-10-17, owed-edits job):** read; terms of the new edits (lots, `RegionAgent`, `unsupported_cause`) follow the rule above. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied for the agent row: the terms its code uses go to the `doc.go` glossary before use (Phase 2 of the plan): strategy, agent view, outcome view, trade plan, intent, purpose, job request, directive (the mailbox; not PLC's), directed strategy, fed, unfed, hunger, privation, target (the inventory target; not the welfare target ratio), keep, reserve (the units for this tick's upkeep, and the reserve money), cover (ticks of use held; not INS's), tier, budget stack, spend cap, favourability, planned recipe, production halt, work share (exists), work report. `Strategy` leaves `notYetTerms` of `glossary_test.go` in Phase 5; each term is checked against `doc.go` first (a second entry fails `TestGlossary_Vocabulary`). A standing note; left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** applied for the estate row: the terms its code uses are in the `doc.go` glossary before use (estate pseudo-agent, listing, listed price, first price, estate markdown, estate pool, intake, pool draw, held lot); "draw" is qualified because it has the random sense, and "treasury" is not used for the estate. A standing note; left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Apply. The terms of the spawner row go to the doc.go glossary before use (Phase 2); Spawner leaves notYetTerms of glossary_test.go.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied. The terms of this row go to the `doc.go` glossary at the start of Phase 2, before use: viability, steady state, warm start (already there), surplus factor, activity, start capacity, mean factor, net vector, reference programme, labour share to subsist, available work share, nominal programme, census (balanced), certificate, seed price (as the solver's), drift, residual, supply gap, lean reserve, virtual start offset, lane. "Lane" is qualified (the lane of the solver is not the lane of a credit round or of a merchant's lane) and "census" is qualified (balanced census). The note stays open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: the terms of the row (Step, Tick boundary, Phase, Position, Entry (of a tick), Failed world, In-step guard, Market record, Snapshot row, Fan-in) are in the `doc.go` glossary before the code uses them. "Run" and "Snapshot" stay in `notYetTerms` and are added in Phases 2 and 9; "World" gains `WorldParams` in Phase 2; "Executor" stays the job's. Standing; left open.

### N-018 Unverified evidence

- **Areas:** EVD
- **Raised:** 2026-10-09.
- **Note:** Several facts behind defaults are unverified summaries
  (`specs/evidence.md`). The traineeship elasticities disagree and come from unopened
  snippets (`specs/README.md`, "Still open" 8). Do not set a default from such a
  fact without checking the source.
- **Update (2026-10-13, TEC):** no source on the growth of productivity was read and no default comes from one; the experience curve and the diffusion are named from memory and flagged unread (`spec-technology.md` 1, 19).

### N-019 Housekeeping for the first code change

- **Areas:** CODE
- **Raised:** 2026-10-09.
- **Note:** When the first code lands, in the same PR:
  - update the layout in architecture.md 2.6 and `AGENTS.md` for `config.go`,
    `defaults.go`, `bounds.go`, `validate.go`, `hash.go`, `commodity.go`, `calendar.go`, the
    optional `denomination/` package (`spec-currency.md` 7), `internal/ledger` and
    `internal/clearing`;
  - add the new terms to the glossary in `doc.go` and the new options
    (`WithAudit`, `WithTargetSpread`) to the options list;
  - add the log keys of N-012 to observability-and-logging.md 3.3.2, and any new
    event to its table 4.3;
  - record an ADR choosing the TOML library, with the justification of
    architecture.md 5.3 (`spec-config.md` 6.1), and check its strict decoding;
  - amend architecture.md 4.3 and 7.7 for the checkpoint and optional extension
    state, as ADR 0002 says, with a fresh read-receipt tag.
- **Update (2026-10-10):** the log keys and events of the written specs are in `rules/observability-and-logging.md`. The rest of this note is unchanged: the architecture.md 4.3 and 7.7 amendment for the optional extension state (ADR 0002) waits for the first code.
- **Update (2026-10-10, REG):** at the first code also add `internal/graph/` (the shortest-path algorithms over primitive slices) and `regions.go` to the layout of architecture.md 2.6 and `AGENTS.md`, and the REG terms to the glossary in `doc.go` (`Region` is used by every spec and is not in the vocabulary list of architecture.md 3.2).
- **Update (2026-10-10, wave 2 integration):** the four new specs name these homes, to add to the layout of architecture.md 2.6 and `AGENTS.md` when each is first needed: `internal/graph` (REG, shortest paths over primitive slices), `internal/moneyflow` (MON), `internal/population` (POP) and `internal/transit` (TRN), each with the root file of its concept and, for TRN, the root types `Mode`, `Shipment` and `ShipmentRequest`. Each also needs its terms in the `doc.go` glossary and its sentinels mapped at the root (N-075). The Python models of the four specs are archived in `doc/specs/models/{reg,mon,pop,trn}/`.
- **Update (2026-10-10, wave 3 integration):** the three new specs name these homes, to add to the layout of architecture.md 2.6 and `AGENTS.md` when each is first needed: `internal/endowment` (RES: regeneration, abundance, the ration, the key; stdlib only), `internal/curve` (piecewise-linear evaluation, shared with the population schedules), `resources.go` (RES), `internal/rumour` and `information.go` (INF: the lag, the distortion, the ring, over primitive slices) `internal/outside` and `external.go` (EXT: the mid, the walk, the quotes and the caps, over primitive slices; the account kind Outside and the six reasons are rows of `internal/ledger`, `spec-external-sector.md` 23). Each also needs its terms in the `doc.go` glossary and its sentinels mapped at the root (N-075). The Python models of the three specs are archived in `doc/specs/models/{res,inf,ext}/`.
- **Update (2026-10-10, SUB):** at the first code add `subsistence.go` and `internal/ownuse` (the flag step, `Cov`, the ramp, over primitive slices, standard library only) to the layout of architecture.md 2.6 and `AGENTS.md`, and the SUB terms to the `doc.go` glossary. The Python models are in `doc/specs/models/sub/`.
- **Update (2026-10-10, MIG):** at the first code add `migration.go` and `internal/migration` (the estimate, the journey time, the margin and its offset, the ranking, the admission against the caps and the partition of outcomes, over primitive slices, standard library only) to the layout of architecture.md 2.6 and `AGENTS.md`, and the MIG terms to the `doc.go` glossary. The Python models are in `doc/specs/models/mig/`.
- **Update (2026-10-10, MER):** MER names these homes, to add when each is first needed: `strategy/` (it now has a second member, so it is created with the first code: `merchant.go`, `options.go`), `internal/merchant` (the pure rules over primitive slices), `contract.go` (root). The Python models are in `doc/specs/models/mer/`.
- **Update (2026-10-10, wave 4 integration, X1 and X13):** the homes of the three wave 4 specs in one list: `subsistence.go` and `internal/ownuse` (SUB), `migration.go` and `internal/migration` (MIG), `strategy/` with `Merchant` registered as "merchant", `internal/merchant` and `contract.go` (MER). One shared helper: the Weyl offset of the agent ID, `(id * 2 654 435 769) mod 2^32`, which `spec-population.md` 3.8.3 defines and MIG and MER reuse as a fixed per-agent spread with no creation draw, lives in `internal/ordering` (the package of deterministic ordering helpers) with one test of its evenness; a new per-agent spread reuses it and does not draw at creation. Each spec's terms go to the `doc.go` glossary and its sentinels are mapped at the root (N-075).
- **Update (2026-10-11, WLD, CHK, OBS, INT, IND, STY, EXP):** the homes the eight specs name, to check at the first code against the layout of `architecture.md` 2.6 and `AGENTS.md`, which already list them as the target layout (2026-10-11): `world.go`, `tick.go`, `option.go`, `random.go` (WLD) with `internal/ordering` gaining `Weyl32`, `Weyl2`, `Bucket`, `Key` and the finaliser (shared with RES); `checkpoint.go` (`Checkpoint`, `RestoreWorld`, `Stateful`, `WithReseed`) and `internal/codec`, with fixtures under `testdata/checkpoints/` per format version (CHK 9.2.1); `integration.go` (INT), where the names of the slot setters are checked against `spec-integration.md` 5.1; `indicator/` and `internal/measure/` (IND); `internal/verdict/` and the build tag `stylised_long` (STY); `experiment/`, `internal/design/` and `internal/reduce/` (EXP). `otelobserver` adds into arrays by typed ID and records each series once at `TickEnded` (NFR-OBS-005). The amendment of `architecture.md` 4.3 and 7.7 for the optional extension state is made (`Stateful`, with a fresh tag). The first consumer example must show a player (architecture.md 1.3, testing.md 7.3). The homes of CLB (`internal/calibration/`, `testdata/calibration/`) are in the layout since the CLB rows were applied (2026-10-11, below). The 128-bit checked helpers have two homes in version 1 (`internal/measure`, `internal/reduce`); the first code decides whether a shared `internal/checked` is worth it. The terms of N-017 go to the `doc.go` glossary before code uses them.
- **Update (2026-10-11, CLB):** the homes of CLB are now in the layout of `architecture.md` 2.6 and `AGENTS.md` (`internal/calibration/`, which imports the standard library and `internal/design` only, and `testdata/calibration/`, the records of a calibration per pack). At the first code that needs them create, with `spec-calibration.md` 12.1: `internal/calibration/` with the registry test (11.1) and the record checks (11.2) in the same package, reading `doc/specs` by relative path (testing.md 10.4); `testdata/calibration/` with a `README.md` of the record grammar and the fixture of the dry run (11.4), which runs in `experiment`'s external test package. No public package and no exported identifier: promoting `internal/verdict` and `internal/calibration` is CLB open question 1, to decide with the first consumer's own calibration. The CLB terms go to `doc.go` (N-017). `doc.go` does not exist yet.
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** the homes of the fourteen later-stage specs are NOT added to the layout of `architecture.md` 2.6 and `AGENTS.md` now; they are created when first needed (2.6, 2.1), as the homes of waves 2 to 4 were. The one decision taken is the shared checked 128-bit helper: `internal/checked/` (a product or a ratio that can exceed `int64`; it imports nothing and holds no type of the root) is named in one line of 2.6 and of `AGENTS.md`, and every package that needs such a product or ratio imports it instead of keeping a copy (the update of 2026-10-11 left it to the first code). The checkpoint fixtures are one all-modules-on fixture per format version, and a module's restore test builds its world from its overlay in the test. The homes, each with its root file: `storage.go` and `internal/room/` (STO), `employment.go` and `internal/employment/` (EMP; the key `K` is passed in), `settlements.go` and `internal/settlement/` (SET), `tenure.go` and `internal/tenure/` (LND; imports `internal/ordering` and `internal/endowment`), `polity.go` and `internal/polity/` (POL), `credit.go` and `internal/credit/` (CRD; imports no internal package), `infrastructure.go` (a file: 2.4 bans the directory name) and `internal/facility/` (CAP), `risk.go` and `internal/risk/` (RSK), `fiscal.go` and `internal/fiscal/` (FIS; `internal/ordering` for `K`), `institutions.go` and `internal/institution/` (ORG), `saving.go` and `internal/saving/` (SAV; the drawdown is in `internal/welfare/`, itself not in 2.6; reads `internal/ordering` for the Weyl offset), `technology.go` and `internal/technology/` (TEC; imports `internal/ordering`), `policy.go` and `internal/policy/` (PLC: the schedule's value, the rule's step, the four gauges over primitive rows, the checks and cuts, the due test with the `Weyl2` bucket of `internal/ordering`; it may import neither `internal/measure` nor `internal/reduce`) and `insurance.go` and `internal/insure/` (INS; imports `internal/credit` for the pro rata helper). The root files also touch `event.go`, `errors.go`, `bounds.go`, `validate.go` and `defaults.go`, and `option.go` (`WithPolicy`), `integration.go` (`DirectPolicy` and the calls of 5.4), `strategy.go` (the credit and cover plan and view types), `snapshot.go` and `checkpoint.go`. The terms of the fourteen go to the `doc.go` glossary before code uses them (N-017) and their sentinels are mapped at the root (N-075). The amendment of `architecture.md` 4.3 and 7.7 for `Policy` is made, and ADR 0006 narrows ADR 0001 decision 7 (N-154).
- **Update (2026-10-12, ADR 0007 and 0008):** at the first code, in the same PR: `rules/architecture.md` 2.6 gains the file-per-module rule of ADR 0007 (one root file per module that owns its state, positions, view filler, plan checker, removal step, relocation refusal, snapshot rows and checkpoint codec; `tick.go` a straight line of one-line position calls whose list is a value a test compares with `spec-world.md` 6) and `rules/testing.md` 7.1 and 10.4 gain the off-only, module-alone and all-on golden files and the test-only per-position state hash, each with a fresh tag; `random.go` holds the seed, the one PCG, the one `Rand` and the site table of ADR 0008 (N-214). The four fan-in functions (`fillAgentView`, `checkTradePlan`, `removeAgent`, `relocationRefusal`) and the checkpoint shape are built in the slice, before any module file.
- **Update (2026-10-12, applied):** the layout rows of the updates above are now in `rules/architecture.md` 2.6 (every root file, catalogue, core and module, and every `internal/` home named by the specs) and in `AGENTS.md`; the module-file rule is architecture.md 2.10; the position table test is testing.md 6.8, the three golden sets and the per-position hash testing.md 7.1.1 and 7.1.2; the delivery order is `doc/roadmap-v1.md`. Still owed at the first code: the glossary in `doc.go`, the ADR for the TOML library, the sentinel mappings (N-075) and the `random.go` of N-214.
- **Update (2026-10-12, Phase 1 of the Stage 0 plan):** the first code is planned in `plans/plan-phase-0-foundation.md`. Settled for it: (1) the TOML library is [ADR 0009](adr/0009-the-toml-library-is-pelletier-go-toml-v2.md) (Proposed; `pelletier/go-toml/v2` v2.4.3, used for `toml.Unmarshal` into `map[string]any` and for the counting guard pass of `unstable.Parser`, a package its documentation says may change, so the fallback is a byte-level scan of the same limits; the ADR stays Proposed until `config/` imports the library); the strict-decoding check is done: the loader maps the tree by hand and does not use library strict mode. (2) The `WarmStart` types live in `config.go` (architecture.md 2.6 wins; `spec-viability.md` 21 and `spec-config.md` 14 corrected), `ErrSyntax` and `ErrWrongType` are new loader sentinels in `errors.go`. (3) `random.go` holds the seed, the one PCG and `Rand`, the seven site constants of WLD 7.1.4, the stream key and the pure `reseed(seed, salt)`; `internal/ordering` holds the finaliser, `Key(t, id)` in the `+gamma` form and the stream key over primitive `uint64` and `int64`; the first tests are WLD 7.1.5 (a) to (d) and the four bare-seed rows, the allocation test covers `Seed` and a draw (checked at zero in a scratch test), and the Go constant `2 * gamma` overflows `uint64` so write `s + gamma + gamma`. (4) `Side` is declared in `event.go`, the default observer is the unexported `discardObserver` and the constant is `warnRun` (OBS 8.1.2, 7.3; architecture.md 4.5). (5) `estates.go` is a core file that also holds the `Estates` part, appended after the slice's golden runs (architecture.md 2.6, roadmap Stage 2). (6) The Go program that produced the vectors of WLD 7.1.5 is not archived: the vectors are the fixture, in the `internal/ordering` tests. (7) architecture.md 2.6 and `AGENTS.md` gained `doc/plans/`; the denomination package has a row in Stage 2 of `roadmap-v1.md` (CUR 7). **Still owed at the first code:** the glossary in `doc.go` (the terms of N-017), the sentinel mappings (N-075), `random.go` and `internal/ordering` (N-214). Not code but blocking nothing in Stage 0: no CI exists (plan O3; the commands of `AGENTS.md` are run by hand until it has an owner and a roadmap row) and ADR 0002 is still Proposed while the hash and checkpoint rules rest on it (plan O2; accept it before Stage 2). N-127 stays as it is: measure in the slice.
- **Update (2026-10-12, Stage 0 built, uncommitted):** the first code is in the tree and passes `gofmt`, `go vet`, `staticcheck`, `go test -short -race ./...`, `go mod tidy` (no diff) and `govulncheck ./...` (no vulnerabilities); statement coverage from `-coverpkg=./...` is 99.3% for the root, 99.0% for `config/`, 98.3% for `internal/checked`, 100% for `internal/ordering` (`internal/logkeys` has no statements). Settled by it: (1) `doc.go` holds the glossary of the terms the code uses, added here: audit, effective seed, estate, estate pool, finaliser, guard pass, pack, pool policy, schema version, site constant, stream key, target spread, trainee, word. (2) The sentinels the Stage 0 rules and the loader return are in `errors.go` (N-075 for the rest). (3) `internal/checked` is `Add`, `Sub`, `Mul` and `MulDiv`, `MulDivRem`, `MulDivFloor`, `MulDivCeil`, `MulDivHalfEven` over a 128-bit intermediate with an `ok` result; `internal/ordering` is `Finalise`, `Weyl32`, `Weyl2`, `Bucket`, `Key`, `StreamKey` and `StreamKeySub`. (4) `config/` imports `pelletier/go-toml/v2` v2.4.3, so ADR 0009 is now due for the owner's acceptance (it said it stays Proposed until `config/` imports the library); `go.mod` has that one requirement and `go 1.25`. (5) Every fuzz target ran for five minutes without a crasher (`FuzzLoad`, `FuzzValidate`, `FuzzMulDiv`, `FuzzBucket`, `FuzzKey`, `FuzzStream_Draw`); `FuzzLoad` is slow on some inputs (it reached about 2 800 execs per second on average against 60 000 to 130 000 for the others) in bursts that stall for seconds (the cause was not investigated; the loader is quadratic in the keys of one table up to the 10 000-key limit, ADR 0009); a candidate for a look when CI sets a fuzz time. **Still owed:** the glossary terms of each later stage before it uses them; the sentinel mappings of the later modules (N-075); the architecture.md 4.3 and 7.7 amendment for extension state (ADR 0002, Stage 2); acceptance of ADR 0009 and of ADR 0002 (plan O2); a CI that runs the commands and the timed fuzz runs (plan O3); a mutation pass over `internal/checked` and `internal/ordering` (not run: non-gating, `gomu` runs only in Docker).

- **Update (2026-10-13, minor gaps of the Stage 0 audit closed, uncommitted):** the audit's minor gaps are closed, each with a mutation check on a copy. Guards added: an import guard (stdlib and layer rules over every guarded file), a glossary guard (architecture.md 3.2 terms and the declared types against `doc.go`), a toolchain pin (`go 1.25`, no `toolchain` line), doc-comment contract phrases (`Observer`, `WithSeed`), a panic guard (a `panic(` must say "invariant"), the staged event catalogue deferral table, and a reason-constant cross-read against the specs; tests for every count bound (24 over-by-one cases, `MaxYields` new), the path cap of `ValidationError`, the declared sentinels, `FuzzKey` as a bijection per tick, the loader's read limits, the six defect overlays over cold and warm layer sets, the pinned pack hashes, and `FuzzLoad` classifying the `Validate` result. One defect found and fixed: `config.Parse` returned a partly mapped `Config` beside its errors, and now returns the zero `Config`. The five-minute fuzz runs are done (above). Nothing is committed. **Still owed**, unchanged: the items of the update of 2026-10-12 (Stage 0 built) other than the fuzz runs; N-224 lists the guards a module PR must update.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. The glossary, the sentinels and the layout of this row are done as above; the CI and the ADR acceptances stay plan 0's.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read: the layout of this row's files is already in architecture.md 2.6 and `AGENTS.md` (N-287); nothing to add. Left open.
- **Update (2026-10-23, plan 2-1 Phase 3):** the owner accepted ADR 0002 and ADR 0009 (status, `doc/adr/README.md` and `documentation_index.md` agree; ADR 0009 Open point 3 carries a dated note that its condition is met); "accept ADR 0002 before Stage 2" of the update of 2026-10-12 is closed. Still open: the CI that runs the commands and the timed fuzz runs (plan O3, now decided in the roadmap).
- **Update (2026-10-23, plan 2-1 Phase 5):** CI decided by the owner: yes, with an arm64 job; `roadmap-v1.md` carries a planned CI row and the owner's rulings, `vision.md` item 2 the decision. The workflow is not written yet. arm64 shown by hand: binfmt installed, `TestGolden_OneOff` passes under emulation (7 s), `BenchmarkStep_OneNoObserver` 15.8 ms a tick emulated against 1.76 ms native. Open until the workflow row is built.

### N-021 Anything that creates or destroys money or goods

- **Areas:** LED
- **Raised:** 2026-10-09.
- **Note:** A new faucet or sink is a row in the ledger's reason registry, with its
  kind and owner (`spec-ledger.md` 7.2); there is no other way to change a total.
  The boundary balance per reason is the measure of how much has entered or left.
  Credit will use the ledger's claims list (`spec-ledger.md` 10). A commodity marked
  `Raw` needs a real source once resources, the external sector or faucets exist
  (`spec-goods.md` open question 5).
- **Update (2026-10-10, LED):** the reason registry gained `disposal` and the goods form of `initial`; `initial` is not construction-only (an external agent mid-run, `spec-ledger.md` 7.2.1).
- **Update (2026-10-10, REG):** REG adds no faucet, no sink and no ledger reason. A relocation moves nothing between accounts; the ledger keeps relocation totals as the reconciling item of a report by region (`spec-regions.md` 6.4).
- **Update (2026-10-10, MON):** applied for money. The reason registry is the built-in rows plus one row per configured money flow (reason = flow ID), closed for a run, so a new money faucet or sink is a registry row (`spec-money-supply.md` 4.5, 10.3); any counterparty whose balance is created or destroyed outside conservation must be a Boundary account (10.1, FR-MON-043). Still open for EXT, FIS and RES: the goods side (a `Raw` commodity's real source, imports and exports, goods faucets).
- **Update (2026-10-10, POP):** POP creates and destroys no money or goods; births and deaths are internal flows between the pool and agents.
- **Update (2026-10-10, RES):** RES applies its part in `spec-resources.md` 9: a raw commodity is endowment-sourced under a stated predicate (9.2: a source recipe with `Draws` on an endowment that can start at least once), validation takes the union with source recipes and EXT's imports, the warning `raw_source_exhaustible` flags a raw good whose only sources are finite stocks, and `endowment_runway` reports the run-down time. RES adds no faucet, no sink and no ledger reason (6.2.1).
- **Update (2026-10-10, INF):** INF adds no faucet, no sink and no ledger reason.
- **Update (2026-10-10, EXT):** applied for EXT in `spec-external-sector.md`. A trade with an external counterparty is the case for built-in reasons (8.1: `outside_budget`, `outside_unspent`, `outside_receipts`, `outside_offer`, `outside_unsold`, `outside_purchases`), and the counterparty's account is a pass-through account of the kind Outside swept to zero each tick (4). The goods side: a `Raw` commodity has three real sources, a recipe with no inputs, a RES-limited sited recipe and an import line; an import line with a sell side makes a commodity sourced for `spec-config.md` 7.5 and satisfies `raw_without_source`; a commodity sourced only so draws `import_only_source` (10.1). Still open for FIS (taxes) and LED (the ledger rows and kind).
- **Update (2026-10-10, SUB):** SUB adds no faucet, no sink and no ledger reason: the output of an own-use job is created by `recipe_output` (`spec-subsistence.md` 6).
- **Update (2026-10-10, MIG):** MIG adds no faucet, no sink and no ledger reason: a move is a relocation (no transfer), and the upkeep eaten on the journey is `consumption` (`spec-migration.md` 9.1).
- **Update (2026-10-10, MER):** MER adds no faucet, no sink and no ledger reason (`spec-merchants.md` 9.6.1).
- **Update (2026-10-13, STO, EMP, LND, CRD, CAP, RSK, FIS, SAV, TEC, PLC, INS):** none of them adds a faucet, a sink or a boundary account. STO adds no ledger reason either: the levy of its stores is a MON flow (a sink whose reason is the flow ID, `spec-storage.md` 7.1). EMP adds three internal reasons (`wage_escrow`, `wage_pay`, `wage_refund`; `spec-employment.md` 7.4.3), LND two internal reasons for a rent (5.1.2), CRD three (`loan_out`, `loan_repay`, `loan_interest`; the claims list is used as this note said), FIS declares each tax, duty, toll and payout as an internal reason and a purchase and a release are `trade` (`spec-fiscal.md` 3.4), INS two (a premium and a payout; an unpaid payout is a claim of the credit list, 9). CAP adds none (the cost of a build is its inputs, consumed by `recipe_input`), RSK uses the existing sink `loss` in its goods form and leaves its money form unused (7), SAV pays no return on savings (a faucet proportional to a balance is MON 9.5's exclusion), TEC and PLC add none (PLC at position 13). A tax or a mint of a ruler that creates or destroys money is a MON grant or levy with the target kind `treasury` (FIS 4.2). Still open for LED: the rows and the claim operations of the ledger (the edits owed to `spec-ledger.md`).
- **Update (2026-10-13, Phase 1 of the ledger row):** applied for LED in `spec-ledger.md` 3.4.1 to 3.4.3, 7.2.2 and 16.1: the built-in rows are `trade`, `initial`, `recipe_input`, `recipe_output`, `disposal`, `consumption`, `spoilage`, `wear`, `estate_money`, `estate_goods`, `start_money`, `split`, `merge`, `births` and `starvation`; every other row of the registry is a module row, declared only while its module is on, with no boundary account while it is off (decision D6 of `plans/plan-phase-1-1.md`; D5 of the Stage 0 plan applied to the registry). The boundary accounts of the built-in group are `1 + 7 * C`, created `initial` (money) first and then commodity-major (64 for the pack `one`, whose estate account is then ID 64). The rows of the modules, the six claim operations (`MarkOverdue` is new, N-227) and the reasons they declare are in the table of 7.2. **Stays owed:** the rows a module declares are appended with the module (architecture.md 2.10.5); the ledger code is Phases 2 to 7 of the plan; the public sentinels that map the ledger's errors are N-075.
- **Update (2026-10-13, Phase 7 of the ledger row, built, uncommitted):** applied in code: `internal/ledger` holds the built-in registry as a fixed table and `ledger.go` builds the boundary accounts of the built-in group at construction in the order of `spec-ledger.md` 3.4.2 (money `initial` first, then commodity-major, the seven goods reasons in registry order; `B = 1 + 7 * C`, 64 and an estate account at ID 64 for the pack `one`, read from the spec by a test). `event.go` gained the eight internal reason constants (`ReasonTrade`, `ReasonEstateMoney`, `ReasonEstateGoods`, `ReasonStartMoney`, `ReasonSplit`, `ReasonMerge`, `ReasonBirths`, `ReasonStarvation`), and a guard test keeps the two ledger groups of `event.go` and the ledger's built-in rows equal both ways. Claims and the balance sheet are built with no module on and `MaxClaims` 0. **Stays owed:** a module appends its reasons to the ledger table and its names to `builtInReasons` in `ledger.go` with its own change, and the faucet or sink with no name panics as a programmer error until it does.
- **Update (2026-10-14, ledger audit):** `spec-ledger.md` 3.3.1 now lists `ErrInvalidKind` for a boundary account, which `Close` already refused; no other change.
- **Update (2026-10-16, plan 1-5 Phase 2):** read for the production row; left open. `Discard` adds no reason, no faucet, no sink and no boundary account: it moves the two boundary accounts of the built-in rows `recipe_output` and `disposal` that every world already has (`spec-ledger.md` 7.2.2, 8.2.3). The rest of the note (a module's rows) stays owed to the modules. N-233 says what the two new operations are for.

### N-027 No default macro constant is stable yet

- **Areas:** ALL, REF, STY
- **Raised:** 2026-10-10, `spec-roadmap.md` Open 4 (ADR 0001, decision 5).
- **Note:** Stable defaults cannot be claimed until `spec-reference-chain.md` and
  the acceptance bands in `spec-stylised-facts.md` exist. Until then no spec states
  a default macro constant as stable. Related: the guessed numbers in N-015.
- **Answer (2026-10-10):** this is a standing constraint, not a question. Keep the
  note open so every spec reads it; close it when REF and STY exist.
- **Update (2026-10-10, RES):** every number of `spec-resources.md` is a placeholder or structural (a finite stock, a flat curve, a full start); the models show agreement with the linearisation, not stability of any default. INF and EXT set no default macro constant; the module has no default nodes.
- **Update (2026-10-10, SUB):** every number of `spec-subsistence.md` is a placeholder (`EnterTicks` 2, `ExitTicks` 3, `RetryJobs` 10, the bounds 8 and 1 000, the example quantities); the staple model shows structure and sensitivity, not stability of any default.
- **Update (2026-10-10, MIG):** every number of `spec-migration.md` is a placeholder (`MoveReviewTicks` 30, `MoveBelow` 10 000, `MoveGain` 1 000, `MoveWidth` 5 000, `MoveCooldown` 60, `MoveHorizon` 180, `MoveOutShare` 50, `MoveInShare` 100, the factor 3 of `migration_flow_window`, every number of the examples and of the models); the models show structure and which damper works alone, not the stability of any default.
- **Update (2026-10-10, MER):** every default of `spec-merchants.md` is a labelled placeholder and the models of `doc/specs/models/mer/` show agreement with the linearised delay condition, not the stability of any default.
- **Update (2026-10-11, WLD, OBS, CHK, INT, IND, EXP):** none of them sets a default or asserts a number of the economy; the numbers of their examples, cost tables and models are placeholders or counts.
- **Update (2026-10-11, CLB):** applied in `spec-calibration.md`: no row of the registry holds a value, a default or a bound (FR-CLB-005), and "stable" is the output of STY 7.3.1 and a stability record (7.4.2, 8): a `stability` line says `in_sample no` only for a hold-out of REF 8.2's seeds 51 to 60 that no choice has read, and `in_sample yes` otherwise (FR-CLB-100); a default at the edge of a grid is `one_neighbour`, never "stable" (7.4.3; an edge clause is asked of STY 7.3.1). The closing text below is unchanged.
- **Update (2026-10-11, STY):** the closing text of this note is now fixed (`spec-stylised-facts.md` 8.6; `spec-roadmap.md` Open 4): REF and STY now exist as specifications, which is not what the constraint needs. The note is *closed when the first reviewed setting run has set the keys of the version 1 criteria (8.1.1), the gate has passed them (and, once REF adds it, the hold-out set has), and at least one shipped pack has a stability record (7.3.3) in which all three checks are STABLE at its defaults and their neighbours. Until then "stable" is not claimed, and the specs' own sentence "a default that has been shown to work" is read as "a default no run has shown to fail".*
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** none of the fourteen specs sets a default macro constant or asserts a number of the economy as stable: every default, bound, threshold and number of an example is a labelled placeholder or structural (POL: `StartMoney` is 0; TEC: every rate defaults to 0, off; PLC and RSK have no default of their own). Their models show structure and sensitivity, not the stability of a default: EMP the closed form of the circuit and the regimes of the price channel; CRD the shield (no monotone effect, its best value moves with the horizon: N-169); SAV a narrow hurdle spread does nothing and a spread of the range of the margin staggers entry (N-199); TEC agreement with the linearisation of POP 5.4 (premium within 4% at year 40); PLC properties of a rule on a linear plant with a delay; INS arithmetic and structure.
- **Update (2026-10-15, plan 1-3 Phase 1):** the clearing row sets no default and asserts no number of the economy: the worked case of `spec-clearing.md` 3.2.2 uses the bounds, which are structural. Left open.
- **Update (2026-10-16, plan 1-4 Phase 1):** the beliefs row sets no default and claims no stability: every default of `spec-beliefs.md` 11 stays a placeholder (the preamble of that spec). Left open.
- **Update (2026-10-16, plan 1-5 Phase 1):** the production row sets no default and claims no stability. A standing note; left open.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: every WEL default stays a labelled placeholder; the clamp bound `MaxInt64 / MaxWelfareWindow` is a derived proof bound and not a default. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: every default of AGT stays a labelled placeholder; no test of the agent row asserts a default as stable, and the worked examples of `spec-agents.md` 16 are hand-computed tests, not results. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: `EstateMarkdown` 500 stays a labelled placeholder (N-015); no test of the estate row asserts a default as stable. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Every SPN default stays a labelled placeholder; no test asserts one as stable.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. The table is a start, not a stability claim (`spec-viability.md` 13.1); no test of this row asserts a default as stable.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The row sets no default and asserts no number of the economy.

### N-028 Can the target-income rule make a shortage worse?

- **Areas:** WEL, STB
- **Raised:** 2026-10-10, `spec-roadmap.md` Open 5. See also N-014.
- **Note:** The rule in `spec-welfare-and-wages.md` may make labour supply fall as
  the wage rises, so a shortage could worsen itself. It is the first entry for
  `spec-stability.md`. Analyse it numerically when that spec is written, and decide
  whether the rule needs a damper or only a condition on demand. A cohort
  multiplies the lockstep of the rule (N-014).
- **Analysis (2026-10-10, derived by hand from section 5, not yet run):** days
  worked are `D = K/W`, so labour supply `N*K/W` is unit-elastic. A self-employed
  role has output `N*K/p`, the same curve in the goods price. With demand
  `Q = A*p^-e`, excess demand has slope `(N*K/p^2) * (1 - e)` at equilibrium, so a
  shortage corrects itself only if demand elasticity `e` is above 1. Staples have
  `e` below 1, so for them a shortage raises the price, days worked fall and the
  shortage deepens. Only three things stop it: the price bounds, households running
  out of money, and the one-day minimum. A spread above 0 smooths the steps but
  keeps the bend.
- **Answer (proposed):** keep the rule, because the backward bend is evidenced
  (Dyer, via Humphries and Weisdorf) for labourers. Make the damper a condition
  plus a test in `spec-stability.md`: bounded, asynchronous price steps and an
  inventory buffer (BEL), and a numeric run with a staple role that injects a
  shortage and asserts the price stays in bounds and recovers. Decide a work floor
  (a producer also works while its household's needs are short; survival before the
  target) from that run. A search snippet says inflection points in labour supply
  give multiple unstable equilibria; it is unverified (`specs/evidence.md`).
- **Update (2026-10-10, STB):** run in `spec-stability.md` 6.1 (stylised model, integer rule). The slope of excess demand is `(1 - e) Q/p`, exact to four places; stable only if `e > 1`; `e = 1` (a fixed budget) is neutral. For `e = 0.5` every seed ends pinned: at the price ceiling after a rise, or at the price where all households work the full year after a fall. An inventory buffer and `TargetSpread` 3 000 do not remove it. A work floor (minimum working days, D13) replaces the ceiling by a price set by the floor. The survival-floor reading is unit-elastic too and is not a damper by itself. The model has no income feedback, so it does not show that a real pack is unstable. **AGT (2026-10-10):** a staple's demand elasticity is 0 to 1, never above 1 (`spec-agents.md` 11.6), so the stability condition `e > 1` fails; the default strategy applies the rule with a rest band and a proportional work share for a cohort (7.1.2, 9.3). Leave open until WEL decides D13 (`MinWorkTicks`).
- **Update (2026-10-10, WEL and STB):** WEL adopted the floor as `MinWorkShare` (`spec-welfare-and-wages.md` 5.3, default 4 000 bp). `spec-stability.md` 6.1 gives the upper outcome as `P0 * (D0 / DMIN)^(1/e)`: about 1.06 times the reference for D0 = 150 days and the default floor, 2.9 for D0 = 250. The instability remains; scenario S04 records the outcome with the floor at its default and at 0. If a pinned market is still unacceptable, N-052 is the next lever. Leave open until S04 has run.
- **Update (2026-10-10, STB):** DEM answers for a luxury: stable only with schedule elasticity above 10 000 and about 10% budget headroom (`spec-demand.md` 6.3; a model), scenario S23. A staple is unchanged. Open until S04 and S23 have run.
- **Update (2026-10-10, SUB):** SUB gives the households that cannot afford a staple an own-use recipe (`spec-subsistence.md` 11). In a stylised model (`models/sub/staple.py`) it removes the shortage pin and 97% to 100% of the unfed household-ticks when the floor covers the upkeep; it does not change the elasticity of the labour supply of the households that can afford the staple, and the price falls to the lower pin (the glut) when the floor covers up to the upkeep and stays in an interior band when it covers more. It is not a damper of L14. Leave open until S04 has run on the library, with the module on and off.
- **Update (2026-10-11, STY):** read by STY: a staple's band cannot claim "not pinned" (C2, `spec-stylised-facts.md` 5.9); STY-W05 bounds the elasticity and P02 records the outcome.
- **Update (2026-10-13, EMP):** the hired case is EMP-A1, merged into L14 (`spec-employment.md` 17, 8.2.1): two regimes, vacancy-bound (the labour channel is inert) and labour-bound (L14); the elasticity of the days worked to the price of a good a worker eats is `-c / (a p - c)`, small when inputs are a small share of value, and near -1 for a good it does not eat. A hired tick counts for the floor. Open until S04 has run with a hired role.
- **Update (2026-10-13, TEC):** under the rule a gain is rest first: days worked fall as 1/T with price and output unchanged until the floor binds; at the floor, income per household changes as T^(1 - 1/e), falling for a staple (`spec-technology.md` 13.5; an analysis, `models/tec/farmer.py`). Scenario TEC-S5 = S152.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the floor is built as arithmetic with its counter and the `RestBand` form so scenario S04 can run both ways; held, its count and its denominator are now stated once (WEL 11.3). Left open for S04.

### N-030 Five regions with different climates

- **Areas:** CAL, PRO, SCL, STY, REF
- **Raised:** 2026-10-10, the first consumer's world size (`vision.md` section 2).
- **Note:** The first consumer needs five regions of varying size and resource
  richness, with different climates, so a crop grows well in some and poorly or not
  at all in others. Consequences:
  - Climate is not an entity or an enum. It is data: a region's yield for each
    gathering or growing recipe, as a basis-point factor with a bound, optionally
    with a seasonal profile from `spec-calendar.md`. A factor of 0 means the recipe
    cannot run there. Nothing branches on a crop or a climate name (architecture.md
    9.1, `AGENTS.md` setting-agnostic rule). Richness is the same kind of data: the
    capacity and regeneration of an endowment (`spec-resources.md`).
  - So RES owns the yield factor and the endowment limit, CAL owns the seasonal
    profile, and PRO's recipe format must let a region modify a recipe's yield
    without a second recipe per region (N-003). Check this against a second,
    unrelated setting, as `vision.md` section 5 item 3 requires.
  - Differences in yield produce comparative advantage, which is what drives trade,
    price convergence within transport cost and specialisation. STY and REF need a
    scenario with at least three regions whose yields differ, not only the
    two-region arbitrage test (`spec-roadmap.md`, REF row).
  - Varying size means households differ by region, so SCL's weights, `MinAgents`
    floor and cost model are checked at the smallest and largest region (N-015).
  - Size, confirmed 2026-10-10: about 1 000 households per region (5 000 in all)
    and 30 commodities, so 150 markets. At scale 1 that is 5 000 agents, within
    the placeholder budget of `spec-scale.md` 6.3 (about 25 ms a tick). Weights
    above 1 are not needed for this world, which also avoids the cohort lockstep of
    N-014 and the chance-event method of N-005 for the first consumer. SCL stays
    core because other consumers may be larger. Missing: the recipe count and the
    tick rate.
  - Region parameters override commodity reference prices (N-010), so each climate
    needs its own prices in the data pack.
  - Seasons, 2026-10-10: `spec-calendar.md` 5.3 defines one season cycle for the
    world. Regions with different climates share it through a table of yield by
    season (a reverse hemisphere reverses the table, a two-season region repeats
    values), not through a cycle of their own. RES owns that table.
- **Update (2026-10-10):** PRO applied its part in `spec-production.md` 6: the yield profile (`all` and by season, keyed by region and recipe), `Sited` for recipes that need a site, 0 = cannot run, no second recipe per region; two worked climates in 12.1 and 12.2. Remaining for RES (the table values and endowments, the `Draws` seam 6.5), REG, STY and REF (a three-region scenario). Still missing: the recipe count and tick rate of the first consumer.
- **Update (2026-10-10, VIA):** `spec-viability.md` 9 and 13.2 (A3, A5): comparative advantage from yield differences, a three-region check (prototype: autarky surplus 2.980, 2.517 and 3.124; pooled 3.124), scenario A3 handed to REF. REF must choose yield ratios large enough to show specialisation.
- **Update (2026-10-10, SCL):** with AGT, PRO and BEL describing the per-agent work, the cost model of `spec-scale.md` 6 is per tradable commodity (`c_0 + n_T * c_t`, placeholders 2 us, 1 us and a reference of 8 commodities): the first consumer's world of 5 000 agents is about 53 ms a tick (about 19 ticks a second), against the earlier 25 ms. Unmeasured; still missing: the recipe count and the tick rate.
- **Update (2026-10-10, REF):** `five` in `spec-reference-chain.md` 4.4.4: five regions with yields by season and recipe, 5 000 households, 30 commodities, 45 recipes (the missing recipe count), 85 active markets (17 per region, not 150), a mean tradable set of 7.7. Each region is viable alone; pooled 2.039. Still missing: the consumer's tick rate. Trade between regions waits for REG, TRN, MER and INF.
- **Update (2026-10-10, REG):** REG gives the region set and the graph. The five regions of `spec-reference-chain.md` 4.4.4 have no edges yet; `spec-regions.md` 12.4 proposes the edge part of `overlay/trade.toml` (a hub-and-pairs layout, 12 directed edges, distances placeholders until TRN fixes a unit's cost and speed). Yields by region keep the rules of `spec-regions.md` 5.3 (both keys exist, one row per key, canonical order). Remaining for RES, STY and REF as before.
- **Update (2026-10-10, POP):** the per-agent death carry works at weight 1, the first consumer's case; no weight above 1 is needed.
- **Update (2026-10-10, RES):** applied for RES in `spec-resources.md` 7: RES owns the VALUES of the yield table, PRO keeps the format, the lookup and the validation errors (`spec-production.md` 6); the table is core data and works with the resources module off. There is no climate entity and no function from richness to yield (rejected, 7.1.2): a climate is the rows of the table, richness is the capacity and the regeneration scale of an endowment (3.3). A new warning `no_comparative_advantage` checks the cross ratio of mean site factors (7.4). Endowments for the five-region pack are proposed in 18. Remaining: CAL, PRO, SCL, STY and REF as before.
- **Update (2026-10-10, MIG):** weight 1 is the first consumer's case, so no cohort moves; a cohort would move whole and lockstep (`spec-migration.md` 3.2, warning `migration_cohort`).
- **Update (2026-10-11, STY):** applied for STY (`spec-stylised-facts.md` 4.4, 5.1, 5.6): facts are stated at weight 1 on `five` and `three` (comparative advantage P06, T02) and on `one` for scale (P09), since the smallest groups of `five` hold 4 to 9 households. Still missing: the tick rate (REF, SCL).
- **Update (2026-10-14, plan 1-2 Phase 6):** the calendar half is built: `calendar.go` holds `Moment`, `At`, `SeasonAt` (a floor modulus, any `int64`) and the day table; the climate itself stays data in the yield table (`production.go`, `resources.go`). Still open: the three-region scenarios (STY, REF) and the tick rate.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open, with what the row learned. PRO's part (the yield profile format, `Sited`, 0 for cannot run) was applied in spec; the row builds the lookup and the check that a recipe runs in a new region by a table change alone (FR-PRO-028). The values are RES's and REF's and the three-region scenario is REF's.
- **Update (2026-10-16, plan 1-5 Phase 8):** the production half is built and gated: `internal/production` holds the site factor (`Site`, the lookup of the profile by season, 0 = cannot run) and `production.go` the yield table and the `no_site` reason; both are tested on a two-climate fixture and fuzzed (`FuzzSite`). Still open: the table values and endowments (RES), the three-region scenario (STY, REF) and the world row's end-to-end check that regions differing only in yield trade.


### N-032 Welfare window against recipe duration

- **Areas:** STB
- **Raised:** 2026-10-10, `spec-production.md` 5.2.3 and 16.
- **Note:** A job consumes inputs at its start and sells output at its end. The welfare ratio sums income over a window `T` (default 30, `spec-welfare-and-wages.md` 2.3.1), so a role whose recipe lasts `T` ticks or more shows costs and no income and looks like it is starving while healthy. That could trigger relief, migration and bankruptcy. Either the window must exceed the longest job plus selling time (PRO warns, `job_longer_than_window`) or income must be attributed to the job it came from. Credit is the real remedy for long jobs.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** `spec-welfare-and-wages.md` owns the window and the fix (it owns `T`), `spec-stability.md` records the loop. Not settled until WEL is revised.
- **Update (2026-10-10):** STB records it as STB-L19 and STB-L20 (`spec-stability.md` 4.2) with scenario S11 (a healthy producer with a job longer than the window is not flagged as starving). VIA 8.3.1 sets starting money to cover the longest job. The fix is still WEL's.
- **Update (2026-10-10, WEL):** decided in `spec-welfare-and-wages.md` 2.3.3: the window of a role is derived, `T = max(WelfareWindow, WindowCycles * longest Duration)`, `WindowCycles` default 2 (see N-062). Income attribution was rejected: it needs a valuation of work in progress or a rewrite of past records. The cost is a slower response of the ratio for long-job roles. Credit remains the real remedy (CRD); STB keeps scenario S11 (a healthy producer with a long job is not flagged as starving).
- **Update (2026-10-13, CRD):** credit is the remedy for the household's cash gap through a long job, by the default borrower's request sized to its upkeep and inputs over the job plus `SlackTicks` (`spec-credit.md` 10.2); it does not change how the ratio reads the job, which is the derived window (11.2). Open for STB until S11 has run with a borrower role (CRD-S1 = S114). CRD leaves the areas.
- **Update (2026-10-13, SAV):** a household's own remedy for the cash gap of a long job is a precaution buffer taken from its own cash cycle (`spec-saving-and-investment.md` 4.3.3: a 20-tick job with inputs 300, upkeep 100 and sales 2 500 has a drawdown of 2 200 against a flat reserve of 300); it is what AGT-A12 and AGT open question 15 ask about, in part. It is a substitute for credit, and a household with both blocks holds the buffer and asks CRD for the difference only.
- **Update (2026-10-13, CAP):** a build job is a long job with no output; the effective window of the role (WEL 2.3.3) already covers its `Duration`, and the residue (the builder looks poor to relief and to migration) is N-171.
- **Update (2026-10-13, INS):** insurance does not remedy the cash gap of a long job (credit does, CRD 10.2); a payout is income in the arrival tick of an insured loss (`spec-insurance.md` 11.1.2).

### N-039 A hard cost floor

- **Areas:** STB, GDS, PRO, SKL
- **Raised:** 2026-10-10, `spec-beliefs.md` 18.1.
- **Note:** `CostFloor` 10 000 (100% of unit cost) stops sales below cost, so perishables can spoil instead of selling cheaply and a market can freeze with buyers below the floor. The stability tests need a perishables-glut scenario. `spec-skill-barriers.md` 2.4 ("cost acts through exit") needs rewording, because cost now also acts directly on a seller's price.
- **Update (2026-10-10, STB):** `spec-stability.md` 6.2 run 5 gives numbers for the perishables-glut scenario (S07): sellers with cost 1 100, buyers capped at 1 000, shelf life 3 ticks. `CostFloor` 10 000: 0% of ticks with a trade, 99% of output spoiled; 9 000: 7% and 98%; 8 000: 97% and 69%; 5 000: 100% and 21%. The market freezes whenever the floor is above what buyers can pay (STB-L09, regime F5). BEL decides its open question 1.
- **Update (2026-10-10, BEL and STB):** `CostFloor` is now 7 000 on a cost that includes labour (`spec-beliefs.md` 3.2.2). `spec-stability.md` 6.2 run 5: 7 000 trades in every tick (45% spoiled with the model's mismatched flows); 10 000 and 9 000 still freeze, and a wage 10% to 25% above the model needs 8 000 or less. `spec-skill-barriers.md` 2.4 is reworded.
- **Update (2026-10-10, RES):** with RES the abundance is a factor of the effective yield, so a thin stock raises `UnitCost` (`spec-production.md` 9.1 divides by `E`) and with it the cost band of beliefs (`spec-resources.md` 4.7.5, RES-A6, L91). A cost floor above what buyers pay can freeze a market whose stock is thin; scenario RES-S3 (S43) with perishables is the test.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open. The cost floor is BEL's rule and STB's scenario S07. The row makes `UnitCost` count labour (9.1), which is what moves the floor, and passes `abundance` 10 000 until RES.
- **Update (2026-10-21, anchor fix):** learned: the floor is only as hard as the anchor it reads, and the anchor was 0 (so the floor fell to the reference band's 10%) for a seasonal crop outside its season and fell with the observed wage in a price crash. Both are fixed (`spec-production.md` 9.3, FR-PRO-054, FR-PRO-055): on `one` the staple floor now holds at about 40 instead of 10 and the fuel floor at about 65 instead of 16. The perishables-glut scenario S07 has not been re-run with the hold on. Left open.


### N-042 The evidence rule may offset the price

- **Areas:** BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2.
- **Note:** In a stylised model of the rules of `spec-beliefs.md` (`spec-stability.md` 6.2), the evidence component of 5.4 settles the price 11% to 13% below the price at which supply equals demand and leaves about 12% of need unmet; with the evidence off, the stock component alone settles at 1 001 of 1 000 with nothing unmet. A trace shows that unfilled bids and asks mostly move their centre down, because the target (the other side's best unmatched quote) can lie on the wrong side of a centre that the limit was drawn away from. Restricting the move to the direction of the failure removes only part of it. This is a finding about a model of the rule as written, with one quantity-sizing rule and no money loop. BEL should reproduce it or revise 5.4 (compare the evidence to the limit, not the centre); FR-STB-018 (clearing residual) is the test.
- **Update (2026-10-10, BEL and STB):** reproduced and revised. `spec-beliefs.md` 5.4 compares the evidence with the order's limit, guarded, scaled by `BeliefEvidenceShare` (FR-BEL-036, 037). In the stylised model the offset falls from 11.3% to 5.7%, with 118 of 2 000 need unmet; `spec-stability.md` 6.2 run 2b reproduces this independently. Reduced, not removed: `spec-beliefs.md` open question 12 owns the rest (candidate remedy: a unit-count imbalance signal). Test: S01 with FR-STB-018.
- **Update (2026-10-10, DEM and STB):** a DEM schedule is a second quantity-sizing rule; run 2b of `spec-stability.md` 6.2 has not been repeated with it (`spec-demand.md` open question 5; register open question 6).
- **Update (2026-10-11, STY):** read by STY: P04's prior is about 2.6 times the 5.7% offset of the revised rule, and says so (`spec-stylised-facts.md` 5.1).
- **Update (2026-10-16, plan 1-4 Phase 1):** read for the beliefs row; left open. The row implements the revised rule of 5.4 as written (the limit-based, guarded, shared evidence move, FR-BEL-036, 037) and tests the guard case (`spec-beliefs.md` 15, case D). Whether the residual offset of 5.7% remains in the library is S01 with FR-STB-018, which needs a world; the Python model (`models/pb/`) is the only evidence until then.
- **Update (2026-10-16, plan 1-4 Phase 6):** the guarded evidence rule is built and tested (FR-BEL-036, case D of `spec-beliefs.md` 15). Still open: whether the 5.7% residual remains is S01 with FR-STB-018, which needs a world; the cross-check against `models/pb/` (plan O5) was not run.
- **Update (2026-10-17, plan 1-7 Phase 1):** read for the belief parameters of the views: Phase 1 edits `spec-beliefs.md` 10.1 only for the parameter group of the `OutcomeView`; the evidence offset and scenario S01 are not this row's, and `Revise` calls `beliefs.Update` unchanged. Left open.

### N-043 Nothing anchors the price level if flows ignore price

- **Areas:** AGT, PRO, SPN, BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2 (runs 1 and 2).
- **Note:** With price-inelastic production and consumption, the belief rules have no interior attractor: an early inventory overhang ratchets the price to the floor of the reference band, a shortage to the ceiling, and a small change in a production-stop threshold decides which. A price-responsive flow must exist somewhere (a consumption rule that depends on price or money, a production choice by margin, an inventory capacity or production stop, exit through the spawner). The register proposes a production stop D24 and records the elasticity condition `es < ed` for cobwebs (6.3).
- **Answer (2026-10-10, AGT):** `spec-agents.md` 11.6 states that a staple's demand elasticity is 0 to 1, never above 1, so `e > 1` and `es < ed` fail for a staple; the responsive flows are buffer buying, recipe choice by margin, the production halt D24 (now specified with `ProduceHaltRatio`) and exit via the spawner. An elastic subsistence demand is not invented; if S04 or S16 shows a pinned market that the halt and exit do not cure, see N-052. Leave open for SPN (exit) and the scenarios.
- **Update (2026-10-10, STB):** `spec-stability.md` 6.2 run 1b: with AGT's target the default halt removes the floor pin and leaves a shortage cycle (price 1.5 to 3.1 times the reference, 12% of need unmet), so the halt bounds the glut but is not an anchor. Open for SPN (exit) and scenarios S04 and S16; see N-067.
- **Update (2026-10-10, DEM and SPN):** DEM supplies a price-responsive flow for non-staples (D30). SPN's entry and exit are slow, capped flows (`EntryShare` 50 bp, `ExitShare` 100) and are not an anchor for a pinned staple (`spec-spawner.md` 16). A pure staple is unchanged: halt, buffer buying, recipe choice, work floor. Scenarios S04 and S16 decide whether this is enough.
- **Update (2026-10-10, EXT):** an outside quote anchors the price of a traded good at the edges of its corridor, never in the interior, and weakly under the midpoint rule (`spec-external-sector.md` 6.7, EXT-A2, extended into L21). In the stylised model of 19.2 a corridor of 45 to 75 confines a cobweb that cycles between 20 and 120 alone, provided the capacity covers the swing. It is an anchor only for goods the outside deals in.
- **Update (2026-10-10, SUB):** SUB is a price-responsive flow for a staple that is not an invented demand: a household whose secure ticks fall below `EnterTicks` feeds itself and its bids shrink with its stock (`spec-subsistence.md` 11.1; SUB-A2, L102). It anchors the shortage side for the households that can take the mode and not the interior of the price. A pure staple role with no own-use recipe is unchanged.
- **Update (2026-10-11, STY):** read by STY: no absolute price level is banded; every price fact is relative to the flow-balancing price, a seed, a base window or a freight (`spec-stylised-facts.md` 4.2).
- **Update (2026-10-16, plan 1-4 Phase 1):** read for the beliefs row; left open. The rule has no anchor of its own by design (`spec-beliefs.md` 5.5: the stock rule fixes the level only where flows respond to price); the flows are the agent, demand and spawner rows' and the scenarios S04 and S16 decide.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open. Nothing in the row prices or halts; the halt is the strategy and world rows'.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: no welfare clause concerns the price anchor; "held by the floor" is defined in WEL 11.3 and does not edit AGT 7.1.3, so scenarios S04 and S16 can run the floor on and off. Left open.
- **Update (2026-10-17, owed-edits job):** the work report of the agent and beliefs edits is in `spec-agents.md` and `spec-beliefs.md`; no anchor was added.
- **Update (2026-10-17, plan 1-7 Phase 1):** read for the agent row: the flows (buffers, the halt, the recipe choice by margin) are built here as rules and the scenarios S04 and S16 decide whether they anchor a price. Learned: the work report (D88) lets S04 and S16 run the floor and the halt with their counts. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Entry and exit are slow capped flows, not an anchor; the row ships RegionCap, RoleCap and ExitCap counts so S04 and S16 can read them.
- **Update (2026-10-21, anchor fix):** the anchor hold and the season rule of `spec-production.md` 9.3 stop a price crash from lowering its own floor; they are not an anchor of the price level (the interior is still unanchored) and they do not cure the collapse of `one` (N-312, N-314). Left open.

### N-045 Dampers that act silently

- **Areas:** STB
- **Raised:** 2026-10-10, `spec-stability.md` (FR-STB-012, FR-STB-013).
- **Note:** FR-STB-013 requires every clamp, refusal, floor or limit to be counted in an event or a bounded `Warn`. Counters exist for belief clamps and order rejections (`BeliefsUpdated`, `OrderRejected`). Missing: the cap or floor on days worked, entry limits, the production stop. FR-STB-012: the regime measures must be computable from events and the snapshot alone: price, volume and unmatched per market, at-floor and at-ceiling per market, weights on spawn and bankruptcy, boundary flows, pool balance, pending output.
- **Update (2026-10-10):** counters now exist for the work floor (Snapshot and bounded `Warn`, `spec-welfare-and-wages.md` 5.3, 11.3; `spec-agents.md` 7.1.3, FR-AGT-072) and for the traineeship queue and places (`spec-skill-barriers.md` 8.3). Still missing: the production halt (a per-agent `Debug` line only, N-067) and a spawner entry limit.
- **Update (2026-10-10):** counters now exist for the demand schedule and ladder (`WantShortfall`) and for the spawner's limits (`EntryLimited`). Still missing: the production halt (N-067).
- **Update (2026-10-10, MON):** `MoneyFlowLimited` is the counter for the silent clamps of MON (a levy found too little, a grant cut at `MaxMoney`, the limit).
- **Update (2026-10-10, POP):** the silent clamps of POP are counted in the event `PopulationChanged`: `DeathsCut`, `DeathsHeld`, `BirthsCapped`, `BirthsUnfunded`, `RateClamped` (`spec-population.md` 10.1).
- **Update (2026-10-10, RES):** the cut demand on an endowment is counted: `EndowmentChanged` carries `Short`, `AgentsDenied` and `AgentsReduced` (aggregate per endowment per tick) and a bounded `Warn` "endowment exhausted" (`spec-resources.md` 6.2.2, 14). The production halt and the spawner entry limit are still as before.
- **Update (2026-10-10, INF):** the closed road, the horizon and the young run are counted by `RumoursServed` (`NoPath`, `TooOld`, `NotYet`). EXT counts its silent cuts the same way: `CutCap`, `CutClosed` and `PriceClamped` in its events (`spec-external-sector.md` 15, EXT-D2, D4, D5).
- **Update (2026-10-10, SUB):** the silent rules of SUB are counted in `OwnUseChanged` (`Entered`, `Left`, `Started`, `Blocked`, `Retried`, `Output`) and by two bounded `Warn`s (`spec-subsistence.md` 14.1, 14.2).
- **Update (2026-10-10, MIG):** every silent limit of the migration module is counted in `MigrationReviewed`: the settling rule (`Unsettled`), the gates and the caps (`NoRumour`, `TooFar`, `NoViable`, `NoGain`, `OutCap`, `InCap`, `Unprovisioned`, `Bound`), held arrivals (`Held`) and journeys ended by a removal (`Ended`); every pushed household ends in exactly one outcome (`spec-migration.md` 7.6, 13.1), and two bounded `Warn`s say when a region is blocked or holds arrivals for ten ticks.
- **Update (2026-10-10, MER):** the merchant's silent dampers (the lane lock, the lot cap, the source-age bound, the rest, the stay) are counted by `MerchantsPlanned` through the plan notes, and the contract round's limits by `ContractRound` (`spec-merchants.md` 15.1).
- **Update (2026-10-11, OBS):** every damper listed has a counter: the production halt is `ProductionHalted` (`spec-observability.md` 4.4.6) and the spawner's limits are `EntryLimited`. What remains is STB's: to run the test of FR-STB-013. OBS removed from the areas.
- **Update (2026-10-11, WLD):** the binding caps are counted by their modules' events, and WLD-A2 asks the readers to check them (`spec-world.md` 10.2.1).
- **Update (2026-10-11, CHK):** the diagnostic runs that decide when a bounded `Warn` repeats are saved as sparse rows, so a resumed run logs as the original (FR-CHK-043).
- **Update (2026-10-13, STO, EMP, SET, LND, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** every silent rule of the later stage is counted: STO by `RoomLimited` and `AgentSpawned.StoresLocked`; EMP by `LabourRound`; SET the dwell by `SettlementHeld`, the class floor by the snapshot field `ClassAtFloor` and a gate's refusals by the reader (`RoleCeiling`, `BirthsCeiling`, `NoViableRole`); LND the closure, the cut for rent and the band in `RentPaid`; CRD by `CreditRound` and `CreditSettled`; CAP by `FacilityLimited` (`Cut`, `Refused`, `Voided`, `Pending`); RSK by an event for every strike (`StoreLoss`, `ShipmentDispatched.LossBP`, `ShipmentArrived.Lost`), with no silent clamp; FIS by `FiscalFlowApplied` (`Short`, `Cut`, `Headroom`, `Exempt`) and `TreasuryOrdered` (`CutCash`, `CutHold`, `CutStock`); ORG every refusal, clamp, skip, bar, wait and hold by `MembershipChanged`, `RightRefused` and `PriceControlled`; SAV by `SavingPlanned`; TEC the cap by `ClampedBP` and `economysim.technology.clamped`; PLC every cut and refusal by `PolicyDecided`; INS by `CoverRound` and `CoverSettled`. What remains is STB's: to run the test of FR-STB-013 over the new events.
- **Update (2026-10-16, observability review of plan 1-5):** the refusal of a start (`no_input`, `no_tool`, `zero_yield`, `no_site`, `not_on_role`) was a silent limit with a per-agent `Debug` line only; it is now counted by `RecipeRefused` (`spec-observability.md` 4.4.6a, FR-OBS-095). An abandoned job is counted by `economysim.recipe.abandoned` and an output lost for room by `RecipeExecuted.Overflow`. What remains is STB's.
- **Update (2026-10-17, observability review of plan 1-7):** two silent decisions of the work report are now counted. A rest (a work share of 0, the target-income rule working as designed) and a household with no recipe worth running are `Rested` and `NoRecipe` of `ProductionHalted`; a household that requested no job because its recipe could not run is `RecipeRefused` (N-308). With the halt, the floor and the refusals this partitions the free households by outcome. What remains is STB's.
- **Update (2026-10-18, plan 1-9 Phase 1):** Reject as already settled: the spawner's limits are EntryLimited since OBS; this row builds the counts (D117, D119).
- **Update (2026-10-18, observability review of plan 1-9):** one silent limit was left: a household at rung 1 with no role to move to stayed and was counted by no cap, cooldown or band, and showed only later as `household.exits` of cause `starved`. It is now `EntryLimited.Stranded` (`spec-spawner.md` 6.4, 13.1.2, FR-SPN-061), the tenth value `stranded` of `economysim.entry_limit`. What remains is STB's.

### N-061 The load-time size gate is a guess

- **Areas:** VIA, CFG
- **Raised:** 2026-10-10, `spec-config.md` 4.1.
- **Note:** `MaxViabilityCells` and `MaxViabilityPivots` are placeholders and the speed of the exact solver in Go is unmeasured. Measure in the slice so that NFR-CFG-001 holds with the programmes on.
- **Update (2026-10-10, REF):** `five` runs 202 pivots in the Python prototype; still unmeasured in Go.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied in part. Phase 7 measures `five` pooled;
- **Update (2026-10-18, plan 1-10 Phase 7):** measured, see the update of N-015: `five` pooled is 1.37 s and 202 pivots in Go (`BenchmarkSolveFive`), `Validate` of `five` 0.59 s, so NFR-CFG-001 and NFR-VIA-002 (under one second) hold on this machine with the programmes on. `MaxViabilityCells` and `MaxViabilityPivots` stay placeholders; no work budget (cells times pivots) was added, and a hostile pack just under both caps (4 x 10^11 cell updates in the worst case) was not measured, so that residual risk of the plan stands. Left open. the gate constants stay placeholders and the note gets the measurement. NFR-CFG-001 is checked on the packs with the programmes on and skipped. Left open.

### N-067 The production halt is not an anchor and has no counter

- **Areas:** AGT, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2 run 1b.
- **Note:** With inelastic flows the default halt (30 000 bp of a one-tick target) gives a shortage cycle, not a stable price, and it is counted only by a per-agent `Debug` line. AGT and OBS: an aggregate per region and role per tick in the style of `UpkeepShortfall`. Scenario S16 must set the threshold.
- **Update (2026-10-10):** `spec-reference-chain.md` 9 repeats the request for an aggregate counter; `spec-demand.md`'s handoff proposes the name `ProductionHalted` (Region, Role, Households, Recipes). OBS decides.
- **Update (2026-10-11, OBS):** the event is `ProductionHalted`, one per region and role per tick in which the halt declined the job request of at least one free household, with the households and, per recipe, the households for which it was skipped (`spec-observability.md` 4.4.6, FR-OBS-016). A rested agent is `rest`, not a halt. OBS removed from the areas.
- **Update (2026-10-11, AGT):** the halt count is computed by the world as a pure function of the view, as the floor is, so `JobRequest` and `Work` are unchanged (`spec-agents.md` 7.1.5, 14.1). Scenario S16 sets the threshold.
- **Update (2026-10-17, owed-edits job):** the work report is in the specs (`spec-agents.md` 7.3.5 and `spec-beliefs.md`); the halt is still not an anchor and has no counter.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the halt count is the world's and is computed by the work report of `spec-agents.md` 7.1.3 and 7.1.5, a pure function of the view, so a consumer's strategy cannot hide it; the threshold is a placeholder for S16. Left open.
- **Update (2026-10-17, observability review of plan 1-7):** the rest, which the note of 2026-10-11 left out of the event ("a rested agent is `rest`, not a halt"), is now counted beside the halt: `ProductionHalted.Rested`, with `NoRecipe`, so a flat output can be told from a halt (`spec-observability.md` 4.4.6). The event keeps the name of the halt, the first of its three counts.
- **Update (2026-10-18, plan 1-9 Phase 1):** Reject as already settled: the spawner's limits are EntryLimited since OBS; this row builds the counts (D117, D119).

### N-071 One wage per region and skilled recipes

- **Areas:** VIA, SKL
- **Raised:** 2026-10-10, `spec-viability.md` open question 12.
- **Note:** One wage per region costs a skilled recipe at the common wage. Decide whether the solver reports a wage per recipe class.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for SKL. The solver reports one wage per region (`spec-viability.md` 8.2.5); no per-class wage is built. Learned: `Report` appends a per-class field if SKL decides.

### N-075 Mapping the ledger's sentinels

- **Areas:** CFG
- **Raised:** 2026-10-10, `spec-ledger.md` 5.2.4 and 8.2.1.
- **Note:** `ErrInvalidLot`, `ErrNotConstruction` and `ErrEmptyTransform` are ledger-internal; the root needs a mapping to public sentinels. Bounds for the estate fields are in `spec-estates.md` 3.
- **Update (2026-10-10):** the sentinels `ErrUnknownSpawner`, `ErrUnknownWant`, `ErrWantItems` and `ErrWantDurable` also need a root mapping.
- **Update (2026-10-10, MON):** MON adds the sentinels `ErrFlowShape` and `ErrReasonCollision`; the root needs a mapping.
- **Update (2026-10-10, REG):** REG adds the root sentinels `ErrNoRegions`, `ErrSelfEdge`, `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation` and `ErrRegionBound`; the last three of the relocation may be ledger-level if `Relocate` is in `internal/ledger`. `ErrUnknownAgent` may collide with INT's.
- **Update (2026-10-10, RES):** RES adds the root sentinels `ErrUnknownResource`, `ErrResourceShape`, `ErrCurve` and `ErrInsufficientEndowment`; the last is mapped from `internal/endowment`.
- **Update (2026-10-10, EXT):** EXT adds the sentinels `ErrOutsideNodeInhabited`, `ErrCrossedQuotes`, `ErrNotAGate` and, with REG, `ErrOutsideNode`; the root needs a mapping.
- **Update (2026-10-10, SUB):** SUB adds the sentinels `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited` and `ErrOwnUseOverlap`; the root needs a mapping.
- **Update (2026-10-10, MER):** MER adds the root sentinels `ErrUnknownMode`, `ErrMerchantShape` and `ErrMerchantWeight`; the root needs a mapping.
- **Update (2026-10-11, WLD):** `ErrUnknownClearing`, `ErrNilExtension`, `ErrWorldFailed`, `ErrInStep` and `ErrOptionContext` need a root mapping (`spec-world.md` 15.1; `spec-config.md` 7.3 now lists them).
- **Update (2026-10-11, CHK):** CHK adds the root sentinels `ErrCheckpointVersion`, `ErrCheckpointCorrupt`, `ErrCheckpointConfig`, `ErrCheckpointExtension` and `ErrOptionContext`; none is mapped from an `internal/` error except where `internal/codec` returns its own for a malformed primitive.
- **Update (2026-10-11, INT):** INT adds `ErrModuleOff`, `ErrUnknownFlow`, `ErrUnknownLine`, `ErrAgentRemoved` (wraps `ErrUnknownAgent`), `ErrNotExternal`, `ErrNotDirected`, `ErrNotOnRole` and `ErrNotTradable` (`spec-integration.md` 11.2). `ErrUnknownAgent` is one sentinel of one meaning for REG and INT (7.1.1); `AddAgent` maps the ledger's `ErrBalanceOverflow` to `ErrOverflow` (AGT 13). `ErrUnknownMode` is a root sentinel with the transport module alone (`spec-transport.md` 12.1).
- **Update (2026-10-11, EXP):** the sentinels of `experiment/` are `ErrBadSchedule`, `ErrBadFactor`, `ErrFactorOverlap`, `ErrBadDesign`, `ErrBadBranch`, `ErrUnknownMeasure`, `ErrOversize`, `ErrUnknownRun`, `ErrDuplicateRun` and `ErrRange` (`spec-experiments.md` 14.1); `ErrUnknownKey`, `ErrTooLarge`, `ErrDuplicateID` and `ErrTooManyErrors` are CFG's.
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** the root sentinels the fourteen specs add, which the root needs a mapping for: STO `ErrNoRoom`, `ErrUnknownStoreClass`, `ErrStoreShape`, `ErrRoomShape`, `ErrNotLimited`, `ErrAgentAway`, `ErrStoreLocked`, `ErrStoreInUse`; EMP `ErrEmploymentShape`, `ErrEmployerWeight`; SET `ErrUnknownSettlementClass`, `ErrUnknownSettlementFlag` (reused by STO, ORG and SPN), `ErrUnknownLimit`; LND `ErrTitleShares`, `ErrTitleCap`, `ErrNoTitle`, `ErrNotAHolder`; POL `ErrJurisdictionCycle`, `ErrSeatNotCovered`, `ErrSeatOutsideNode`, `ErrStartWithoutSeat`, `ErrUnknownJurisdiction` (reused by FIS, ORG and PLC); CRD `ErrCreditShape` (and, ledger-level, `ErrOpenClaim`, `ErrUnknownClaim`, `ErrClaimShape`; `ErrTooManyClaims` is CRD's); CAP `ErrFacilityCap`, `ErrFacilityCount`, `ErrFacilityHolder`, `ErrFacilityLocked`, `ErrFacilityNoCost`, `ErrFacilityNotTransferable`, `ErrFacilityOwner`, `ErrFacilityRegion`, `ErrFacilityShape`, `ErrFacilityTarget`, `ErrUnknownFacility`, `ErrUnknownFacilityClass`; RSK `ErrRiskShape`, `ErrUnknownRiskRow`; FIS `ErrNotGoverned`, `ErrNotCovered`, `ErrFiscalShape`; ORG `ErrBodyOverlap`, `ErrBodyShape`, `ErrControlShape`, `ErrRightShape`, `ErrUnknownBody`, `ErrUnknownControl`, `ErrUnknownRight`; SAV `ErrNotSaver`; TEC `ErrTechShape`, `ErrTechRecipeOverlap`, `ErrUnknownField`; PLC `ErrPolicyShape`, `ErrLeverNotInReach`, `ErrLeverShared`, `ErrUnknownActor`, `ErrUnknownLever`, `ErrUnknownPolicy`; INS `ErrCoverShape`. `ErrUnknownClass` is IND's and is avoided.
- **Update (2026-10-13, Phase 1 of the ledger row):** the first build of `internal/ledger` returns these sentinels, all ledger-level: `ErrInsufficientFunds`, `ErrInsufficientStock`, `ErrBalanceOverflow`, `ErrReasonMismatch`, `ErrUnknownAccount`, `ErrClosedAccount`, `ErrInvalidAmount`, `ErrBoundaryReserve`, `ErrReservationExceeds`, `ErrUnknownLot`, `ErrInvalidLot`, `ErrNotConstruction`, `ErrEmptyTransform`, `ErrOpenReservation` and `ErrNotEmpty` (new, `spec-ledger.md` 3.3.1; `ErrOpenReservation` is REG's name, `spec-regions.md` 6.1.3), and for the claims `ErrUnknownClaim`, `ErrClaimShape`, `ErrTooManyClaims` and `ErrOpenClaim`. `ErrNoRoom` waits for the storage module. **Stays owed:** which of them become public sentinels of the root, and the mapping to rejection reasons (`spec-clearing.md` 3.2), is decided with each first user (the clearing, production and world rows), not by the ledger row.
- **Update (2026-10-12, CFG):** the loader sentinels are root sentinels in `errors.go` (`config/` imports the root and returns them wrapped): `ErrSyntax` (malformed TOML), `ErrUnknownKey`, `ErrNotInteger` (a float literal where an integer is needed), `ErrWrongType` (any other wrong type, a date anywhere), `ErrMissingField`, `ErrUnsupportedSchema`, `ErrMergeConflict`, `ErrTooLarge` (also the document count, the merged size, a line, the nesting depth and the keys of one table) and `ErrTooManyErrors`. `ErrSyntax` and `ErrWrongType` are new beyond the six of the plan.
- **Update (2026-10-13, Phase 7 of the ledger row, built, uncommitted):** nothing is mapped to a public sentinel yet: no root code calls the ledger's operations, so `ledger.go` maps only the reasons (the ledger's number to the root's `Reason` and back). The seventeen ledger-level sentinels of the update of Phase 1 stay in `internal/ledger` (`errors.go` there); each becomes a public sentinel of the root, or a rejection reason of `spec-clearing.md` 3.2, with its first user (the clearing, production and world rows). `ledger.go` wraps the ledger's constructor error with `%w`, so `errors.Is(err, ledger.ErrInvalidConfig)` holds inside the module; that sentinel is not exported and is not reachable from a valid configuration.
- **Update (2026-10-15, plan 1-3 Phase 1):** the clearing row maps `ErrUnknownClearing`, `ErrNilExtension` and `ErrInvalidTrade` as root sentinels (`errors.go`, Phase 4 of the plan) and maps no ledger sentinel to a public error: every ledger refusal at settlement is a violated invariant (`spec-clearing.md` 3.2.2; D33, D34 of the plan), so `ErrBalanceOverflow` and its siblings stay in `internal/ledger`. A registration name that is not an ID is the existing `ErrInvalidID` (`spec-world.md` 4.1.1).
- **Update (2026-10-18, plan 1-9 Phase 1):** Apply for this row. ErrUnknownSpawner is declared in errors.go and returned as a *ValidationError; the ledger's sentinels are not mapped here.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied for this row's sentinels: `ErrNotViable`, `ErrNoUpkeep`, `ErrSolveLimit`, `ErrSolveInconsistent` and the type `ViabilityError` are root sentinels in `errors.go` (D124); `ErrWarmStartPrice` and `ErrWarmStartJob` exist. `errors_test.go` and the declared-sentinel test gain the four. Left open for the other owners.

### N-080 Wants in the steady-state programme use equal-price shares

- **Areas:** VIA, DEM
- **Raised:** 2026-10-10, `spec-viability.md` 3.2.
- **Note:** Wants enter the programme at the reference weights, fixed and unscaled by the surplus factor. Revisit with scenario S25 (a substitution flip changes the shares).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for DEM. Learned: the slice has no wants term (A5); the demand row adds the fixed term and the two-phase start (N-277).

### N-082 Relief and entry share the pool in time

- **Areas:** EST
- **Raised:** 2026-10-10, `spec-spawner.md` 7.3, `spec-estates.md` 2.6.
- **Note:** Relief is paid in phase 1 from what the previous tick's phase-7 entry step left, so the pool is shared in time. WLD must keep the order: phase 1 relief, phase 7 entry.
- **Update (2026-10-10, POP):** births draw from the pool in phase 7 inside `EntryPoolShare`; the POP deaths are executed before the budget is taken so the estates of the dead fund the same tick's births (`spec-population.md` 4.8). WLD must keep phase 1 relief, phase 7 entry. The pool is shared more widely now: see N-096.
- **Update (2026-10-10, SUB):** the subsistence floor draws nothing from the pool and lowers the draw on it (`spec-subsistence.md` 8.4); a floor-fed household still draws relief on a ratio that includes its receipts. Whether relief should pay a household that feeds itself is `spec-subsistence.md` open question 8 (N-119).
- **Update (2026-10-10, MIG):** the migration module draws nothing from any pool; its indirect effects are the relief a poor arrival draws from the destination's pool (L63, MIG-A4) and the money a migrant leaves to the origin's pool if it dies on the journey (`spec-migration.md` 9.2).
- **Update (2026-10-11, WLD):** applied for WLD in `spec-world.md` 6.3 and 6.9: relief is paid at P1.6 and entry at P7.1, so the order holds; money flows with a pool target run at P1.7 after relief. WLD removed from the areas.
- **Update (2026-10-18, plan 1-8 Phase 1):** read for the estate row: relief is the Stage 2 module's; the entry draw of this row (`grant`) is the only P7.1 draw on the pool, and P1.6 relief then P7.1 entry stay in the world row's order. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The order (relief at P1.6, entry at P7.1) is WLD's; the row reads only the pool balance for the advisory budget.

### N-084 Scenarios that wait for trade, money and the coarse tick

- **Areas:** STY
- **Raised:** 2026-10-10, `spec-reference-chain.md` 7.6.
- **Note:** S13 (the faucet overlay), the dynamic parts of X1 and X2 (the trade overlay) and the coarse-tick leg of S12 wait for these specs.
- **Update (2026-10-10, REG):** REG supplies the edge format and a proposal for the edge part of `overlay/trade.toml` (`spec-regions.md` 12.4), and two scenarios, S28 and S29 (in the register S28 and S29). X1 and X2 (dynamic) still wait for MER and INF.
- **Update (2026-10-10, MON):** S13 is defined: `overlay/faucet.toml` (and `overlay/faucet-sunk.toml`) are in `spec-money-supply.md` 17.6, and the pass is exact for the accounting (boundary balance = `-Total`, `M = M_init + issued`) and report-only for the price drift until STY sets a band. Still open: the dynamic parts of X1 and X2, the coarse-tick leg of S12.
- **Update (2026-10-10, TRN):** the transport mechanism for the dynamic parts of X1 and X2 exists (modes, the shipment, `Inbound` and `Routes`: `spec-transport.md` 3 to 9). An `overlay/trade.toml` needs a `modules.transport.mode` table (respelled 2026-10-12, N-102), a hired mode's haul commodity and a carrier recipe (17.4 is a model), a consignee in the destination, MER's shipper and INF's rumours. The static check can use `transport.Lanes` (9.4). Scenarios S30 to S32 are proposed in the register (TRN 15 called them S28 to S30).
- **Update (2026-10-10, wave 2 integration):** REG, MON and TRN now exist, so only EXT, MER, INF and STY are left in the areas. The overlays are N-092.
- **Update (2026-10-10, INF):** INF now supplies the rumours (`spec-information.md` 5). The overlay `overlay/trade.toml` needs a `[modules.information]` table (`info_speed`, `rumour_error_bp`, `rumour_max_age`) and an informed merchant role (`informed = true`); 12.1 there shows the keys.
- **Update (2026-10-10, EXT):** EXT exists (`spec-external-sector.md`). Its invariant scenario EXT-S5 (S55) runs the reference chain with an external overlay (an outside node on a gate region, 18.2 is a model of the file layout) over a range of seeds with `WithAudit`; REF builds the pack. EXT gives the dynamic parts of X1 and X2 the gate as the place where imports appear and the counterparty's ask as the price a shipper reads there.
- **Update (2026-10-10, wave 3 integration):** INF and EXT now exist, so only MER and STY are left in the areas. The dynamic parts of X1 and X2 wait for MER (the shipper); the overlays are N-092.
- **Update (2026-10-10, MER):** MER exists. The dynamic parts of X1 and X2, S28 to S32, S46 and S47 can run with a merchant role: `overlay/trade.toml` needs `[modules.merchants]`, a role with `strategy = "merchant"`, `informed = true`, a `[role.<id>.merchant]` block and `consigns = true` for the consignee role (`spec-merchants.md` 19.1 shows the keys), plus the Go line `economysim.NewWorld(cfg, strategy.Options()...)`.
- **Update (2026-10-11, STY):** S13 is STY-S05 (exact accounting, the drift report-only, with the reason); X1 and X2 dynamic have bands (STY-P06, STY-T02) that wait for the overlay; the coarse-tick leg of S12 stays open for CAL and REF (`spec-stylised-facts.md` 5.3, 5.6).
- **Update (2026-10-11, CHK):** X5, S60 and S72 have their checkpoint legs defined (`spec-checkpoint.md` 8.2.3).

### N-085 Nominal table against the full-labour table

- **Areas:** VIA, STY
- **Raised:** 2026-10-10, `spec-viability.md` 5.3.4.
- **Note:** When the wage dual is 0 the warm start uses the nominal programme. Run both tables on `famine` and compare.
- **Update (2026-10-11, STY):** applied as a report beside STY-H02 on `famine` (`spec-stylised-facts.md` 5.7); the comparison needs a run.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied in part. The solver builds both tables (D131) and reports `NominalSeed`; the comparison on `famine` is STY's run. Phase 7 asserts the nominal table of `famine` as `famine.toml` gives it: staple yield `spring = 3000`, nominal wage 1 624, the value of `warm-famine.toml` (the 1 826 of VIA 15 is the variant with a yield of 3 800; VIA 15 lists 1 868, 1 826, 1 624 and 755). Left open.

### N-086 Re-run the demand and spawner models under the register

- **Areas:** STB, STY
- **Raised:** 2026-10-10, `spec-stability.md` 6.5 and open question 6.
- **Note:** The numbers quoted in 6.5 are from the owners' models. STY or EXP should re-run the luxury model with DEM's schedule and the occupation model with SPN's discrete rules, and set the `entry_share_window` threshold from the run.
- **Update (2026-10-11, STY):** STY's models are of its own definitions and are no re-run of the luxury model with DEM's schedule or the occupation model with SPN's discrete rules; the runs are described by S23 and S26 and their keys (`spec-stylised-facts.md` 5.11, 10).
- **Update (2026-10-11, EXP):** the two re-runs are experiments of two of the forms of `spec-experiments.md` 12.2 (a grid over the elasticity and the headroom; a spawner cap on and off), and the harness supplies the means; STY and CLB run them (12.3).
- **Update (2026-10-11, CLB):** the two re-runs are steps of CLB's procedure (`spec-calibration.md` 7.3): the occupation model is C4c, which sets `entry_share_window` from the screen of `EntryShare`, and the luxury model is C6 (the elasticity and the headroom, I18). The runs are STY's and EXP's and the steps are CLB's; the note stays open until a run has been made.

### N-087 Wants against the production halt

- **Areas:** AGT, STB
- **Raised:** 2026-10-10, `spec-stability.md` open question 8.
- **Note:** Does the halt cut the supply of want goods in a downturn and deepen the wage and demand loop (L57)? Run S24 with the halt on and off.
- **Update (2026-10-11, OBS):** the counter by recipe exists (`ProductionHalted`, `spec-observability.md` 4.4.6). The run of S24 with the halt on and off is AGT's and STB's. OBS removed from the areas.
- **Update (2026-10-17, owed-edits job):** the work report is in the specs; the wants against the halt are not settled.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: no want exists in the slice, so S24 is a Stage 2 run. Left open.

### N-089 Model scripts of the demand, spawner and reference-chain specs

- **Areas:** DEM, SPN, REF, CODE
- **Raised:** 2026-10-10, `spec-demand.md` 14, `spec-spawner.md` 16, `spec-reference-chain.md` 14.
- **Note:** The scripts behind these specs are still in `/tmp/economy-boss/` (`dem/`, `spn/`, `ref/`). Archive them under `doc/specs/models/` like the others, and update the specs' paths.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The SPN models are not re-run; the worked cases of SPN 15 are tested as arithmetic.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open, with a learning: the REF prototype that wrote the warm documents is the one that is not archived (N-274).

### N-090 What MIG and the specs that tie obligations to a region must do with a relocation

- **Areas:** PRO, SKL, INT
- **Raised:** 2026-10-10, `spec-regions.md` 6.
- **Note:** `relocate` moves only the account's region (`spec-regions.md` 6.2). MIG decides the job (abandon it first; the abandon reason `region_changed` is proposed for `spec-production.md` 5.4, REG open question 3), the beliefs (reseed on arrival, N-038), `Privation` and the traineeship. SKL registers an `in_traineeship` refusal and CAR or MER an `open_contract` refusal (`region_bound`, 6.5.2). INT owns the public face of `relocate` for an external agent (6.6). The part for TRN is done: it registers `open_shipment` and calls `Relocate` with cause `transport` at an arrival.
- **Update (2026-10-10, SUB):** SUB's part: the mode and its counter are dropped at a relocation and at the departure of a timed move; SUB has no reason and no event for the drop; an own-use job is abandoned with `region_changed` as any job (`spec-subsistence.md` 5.8). A household on a journey or a trip is not evaluated.
- **Update (2026-10-10, MIG):** MIG's part is applied in `spec-migration.md` 7.2 and 8.2 to 8.5: the job is abandoned at departure with `region_changed` (adopted in `spec-production.md` 5.4); the beliefs are reseeded at arrival; `Privation` and `Hunger` are kept and `RoleSince` is set at arrival, which gives the ladder's grace of `spec-spawner.md` 5.2; a trainer or a trainee, an away traveller and a party to an open contract are refused by the guard, the read-only query `World.RelocationRefusal(agent, to, skip...)` (`spec-migration.md` 7.2.1; REG 6.1), run again at arrival with `skip` = `on_journey`, where a refusal HOLDS the arrival; the region then changes by the ledger-level `Relocate`, cause `migration`. MIG registers the `region_bound` refusal `on_journey`. Still open: SKL (register `in_traineeship`), INT (the public face of `relocate` for an external agent, and the cancel of a journey before a forced move, `spec-migration.md` 8.4.1).
- **Update (2026-10-10, MER):** MER registers the `region_bound` refusal `open_contract` (open for the payer and the deliverer of an open contract, `spec-merchants.md` 9.8.1). A merchant on a trip is away, is refused by `open_shipment` and `open_contract`, and is not a migrant (12.4.1). Removal of a party: 9.8.2.
- **Update (2026-10-10, wave 4 integration, X2):** the full list of region-bound refusals is `open_shipment` (TRN), `on_journey` (MIG), `open_contract` (MER) and `in_traineeship` (SKL). A timed move has the shape of TRN's traveller: the account stays in the origin while away and the region changes at arrival through the ledger-level `Relocate`; REG gains a read-only dry run of the relocate refusals with an optional `skip` of one named refusal.
- **Update (2026-10-10, supervisor, wave 4):** a rule for every later spec, found when MIG's first draft lacked it: any module that keeps an agent's account in a region while the agent is away or committed (a journey, a trip, a contract, a traineeship, a garrison, a voyage) MUST register a `region_bound` refusal for it, or another caller (INT's forced move, a later module) can relocate the account and break the module's invariant. The module's own arrival then re-runs the dry run with `skip` set to its own refusal and changes the region by the ledger-level `Relocate`. The rule stands for any later module (the update of 2026-10-13 gives the full list: RSK, CAP and ORG registered none). A module that does register one adds the new name to the list above in the same change.
- **Update (2026-10-11, INT):** the public face is `Relocate(agent, region)` for any agent that is not a pseudo-agent, with the cancel of a migration journey inside it, atomic (`spec-integration.md` 4.5); there is no public cancel. Still open: SKL registers `in_traineeship`; the rule for later modules stands.
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** the full list of `region_bound` refusals is now six: `open_shipment` (TRN), `on_journey` (MIG), `open_contract` (MER), `in_traineeship` (SKL), `open_hire` (EMP: for a principal with open hires; a worker needs none, its job is abandoned when its account changes region; `spec-employment.md` 6.5.1) and `has_store` (STO: for an owner of a store, one obligation; `spec-storage.md` 9.2). `spec-regions.md` 6.5.2 and FR-REG-060 carry the list. STO also guards the other side: a store is never added to an agent that is away or on a journey (`ErrAgentAway`, N-143). The other twelve register none, each with its reason: SET, POL (a treasury is neither away nor committed and its kind is not relocatable), CRD (a claim needs no account in a region), FIS, SAV, TEC (knowledge stays in its cell when a household moves), PLC, ORG, LND (a title does not move with a household), RSK (what it exposes is bound by `open_shipment` and `has_store`), CAP (a site, a route and a ground facility register none, with three reasons in `spec-infrastructure.md` 7.5.1; a store facility is bound by `has_store`) and INS (a route policy is bound by `open_shipment`, a store policy ends in its tick, a payout is a transfer). The rule stands for any later module that keeps an account away or committed.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open. The abandon function takes a reason and `region_changed` is appended with MIG, never inserted. A job of a self-carried trip cannot start (TRN).
- **Update (2026-10-16, plan 1-5 Phase 8):** nothing further applied. the reason is an argument of `recipeAbandoned`, the value builder of the event, and not of `job.abandon`, so MIG's `region_changed` is appended to the reasons with MIG and changes no function of the row. Still open for MIG (the call at departure), SKL (`in_traineeship`) and INT.


### N-092 Overlays and scenarios that the wave 2 specs hand to REF

- **Areas:** REF, STB, STY
- **Raised:** 2026-10-10, `spec-regions.md` 12.4, `spec-money-supply.md` 17.6, `spec-population.md` 14.4 and 18, `spec-transport.md` 15 and 17.
- **Note:** REF builds: the edge part of `overlay/trade.toml` (REG 12.4, with a `modules.transport.mode` table (respelled 2026-10-12, N-102), a hired mode's haul commodity and a carrier recipe, TRN 17.4); `overlay/faucet.toml` and `overlay/faucet-sunk.toml` (MON 17.6); a population overlay (a fixed-factor test double or RES, the demand overlay on, `Renewal` 0; a warm start's census is not the Malthusian `N*`, POP 18). The new scenarios are S28 and S29 (REG), S30 to S32 (TRN), S33 to S36 (MON) and S37 to S40 (POP); S13 is defined. STY sets the bands for natural increase, recovery time and the money-limited population.
- **Update (2026-10-10, RES):** `spec-resources.md` 18 proposes `overlay/resources.toml` (an `arable` refill resource for `grow_staple`, a finite `deposit` for `dig_mineral`, a logistic `stand` for `cut_fuel`; the line gatherers stay unlimited), a slack and a binding data set sized from the solver's flows, and the scenarios RES-S1 to RES-S5 (S41 to S45, 19.4). Capacities in 18.2 are hand-derived; REF re-derives them from the solver.
- **Update (2026-10-10, wave 3 integration):** EXT's scenarios EXT-S1 to S5 (S51 to S55) need an external overlay on a gate region of `one` or `three` (`spec-external-sector.md` 18.2); INF's S46 to S50 need `[modules.information]` and an informed merchant role; INF-S1 (S46) and the rest of INF-S2 (S47) wait for MER.
- **Update (2026-10-10, SUB):** `spec-subsistence.md` 18.3 proposes the scenarios SUB-S1 to SUB-S5 (S56 to S60). REF builds an overlay `overlay/own-use.toml` (an own-use recipe on `one`, with a yield row and, for SUB-S4, a refill endowment) and uses `famine` for SUB-S2.
- **Update (2026-10-10, MIG):** `spec-migration.md` 17.2 proposes MIG-S1 to MIG-S6 (S61 to S66) and 15.3 an overlay `overlay/migration.toml` (`[modules.information]`, `[modules.migration]`, `move_speed` chosen against the trade overlay's edge lengths so that the shortest journey is 1 or 2 ticks and the longest under `CoverTicks`). REF builds it.
- **Update (2026-10-10, MER):** the trade overlay needs a merchant role, a consignee role and `[modules.merchants]` (`spec-merchants.md` 19.1); scenarios MER-S1 to MER-S6 (S67 to S72) are proposed in 17.
- **Update (2026-10-11, STY):** the bands for natural increase (W06), recovery time (S02) and the money-limited population (W06) are STY's; the overlays stay REF's, and `overlay/sink.toml` is new (`spec-stylised-facts.md` 5.2, 5.3, 21).
- **Update (2026-10-11, CHK):** X5, S60 and S72 have their checkpoint legs defined (`spec-checkpoint.md` 8.2.3).
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** the scenarios S91 to S165 need overlay files that REF builds: `overlay/settlements.toml`, `storage.toml`, `employment.toml`, `tenure.toml`, `credit.toml`, `risk.toml`, `infrastructure.toml`, `cover.toml`, `polity.toml`, `polity-governed.toml`, `fiscal.toml`, `fiscal-governed.toml`, `institutions.toml`, `institutions-inert.toml`, `policy.toml`, `policy-directed.toml`, `saving.toml` and `technology.toml` (`spec-reference-chain.md` 4.5, with the layering order stated there), and the scenarios of 7.12.

### N-093 Inputs that the wave 2 specs offer the viability check

- **Areas:** VIA, CFG
- **Raised:** 2026-10-10, `spec-regions.md` 4.5.3, `spec-transport.md` 9.4, `spec-money-supply.md` 15.4.
- **Note:** (1) `WithFreeTrade` links every pair of regions; with edges only regions in one strongly connected component can trade (`Graph.Components`), so the pooled check may use it (REG open question 2). (2) `transport.Lanes` derives lanes from hired modes with labour only; the carriers' feed and vehicle wear are not in the programme (TRN 9.4, 20.8), and VIA may want a lane with inputs. (3) With a rate levy a class's holdings are bounded by `m* = E + g/rho` (MON 9.3); VIA may seed a hoarder class at `min(Money, m*)` (MON 15.4), optional.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for TRN, REG and MON. Item (1): the pooled check uses free trade across all regions; strongly connected components wait for the region graph. Items (2) and (3), lanes and a hoarder class, are module inputs.

### N-096 The regional pool serves births, entry, relief and money flows

- **Areas:** MON, POP, EST, SPN
- **Raised:** 2026-10-10, `spec-money-supply.md` 8.6 and open question 4, `spec-population.md` 4.3, 4.8 and 5.5.
- **Note:** Four things draw on or pay into a region's pool: relief (phase 1), money flows with the pool as a target (phase 1; a pool target needs the estates module, `spec-money-supply.md` 4.3.2; the estate account exists without it, the pool policy does not), entry and births (phase 7, inside `EntryPoolShare`, deaths first). A per-household grant couples issuance to the household count (MON-A5): POP decides whether income from a grant raises births. In a closed region the population is bounded by M / stake; a faucet or an inflow with migrants lifts the bound. MON and EXT should say how money entering a region reaches its pool.
- **Update (2026-10-10, EXT):** money that the outside pays reaches the households first (the sellers of the goods the outside buys) and the pool only through the estates (deaths, relief returns, sales by the estate); EXT adds no pool channel and no grant to a pool (`spec-external-sector.md` 11.5). A surplus therefore raises `M_h` before `M_p`, and a hoard of the outside's money is a hoard like any (STB-L31, L45; damper D42).
- **Update (2026-10-10, MIG):** the migration module draws nothing from any pool (as N-082); a migrant takes no stake and neither region's pool is touched (`spec-migration.md` 9.2).
- **Update (2026-10-13, CRD):** debt service to an estate that inherited a claim pays into the pool (`M_p`), and a removed debtor's money pays its creditors before it reaches the pool (`spec-credit.md` 9.1, 9.2).
- **Update (2026-10-13, LND):** rent paid to the pool is a fifth thing that pays into a region's pool (`spec-land.md` 7.6); it leaves households (`M_h`) and returns only by relief and entry.
- **Update (2026-10-17, spawner edits):** read; not settled by them. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: `grant` and `acceptMoney` are the one place the pool is drawn or paid by an entry, a birth or a weight reduction (`spec-estates.md` 2.3, 2.8.1); relief and a pool-targeted flow append to the same file in Stage 2. Left open for MON, POP and SPN.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The funding rule EntryPoolShare is the executor's; the row gives Funded and the advisory Budget, taken once after the module's deaths (C15).

### N-097 Migration and the population loop

- **Areas:** POP
- **Raised:** 2026-10-10, `spec-population.md` 6.5.
- **Note:** MIG is the other way households enter and leave. It must use the same stake rule for arrivals and not create money, say whether the replacement ratio is part of its signal, and share the pool budget `EntryPoolShare` with births. A closed region (no immigrant) is the behaviour tested before MIG. Relief raises the ratio and so births and lowers deaths (POP open question 4): the same effect as L40.
- **Update (2026-10-10, MIG):** applied for MIG in `spec-migration.md` 5.5 and 9: a migrant carries its account, so no household is created, no money is created and no stake is paid; neither region's pool is touched and `EntryPoolShare` is not shared (births and entry keep the whole budget); the replacement ratio is NOT part of the signal (the trigger is the household's own earned ratio against `MoveBelow`; a pack with both modules may set `MoveBelow` near `r*` by hand); the closed-region behaviour is unchanged. The money a migrant carries moves the money-limited bound `M / stake` of each region by what it carries (9.3, MIG-A5, L76). Still open for POP: its section 7 (the MIG row) and 6.5.

### N-099 A trip is a gap in income

- **Areas:** WEL, STB
- **Raised:** 2026-10-10, `spec-transport.md` 15 (TRN-A6) and 20.9.
- **Note:** The effective welfare window derives from recipes only (`spec-welfare-and-wages.md` 2.3.3), so a self-carried trip longer than the window looks like starving. The validation warning `trip_longer_than_window` is a stopgap; WEL decides whether the window counts the longest trip of a role (WEL open question 9, left open at the wave 2 integration). The working ticks of a self-carried trip run from dispatch to the tick before arrival.
- **Update (2026-10-10, MIG):** a migration journey is also a gap in income: `k` records of zero income and the origin's upkeep, not working ticks (`spec-welfare-and-wages.md` 4.3 already says so). The settling rule (`spec-migration.md` 5.2) waits a window before the record can trigger a move.
- **Update (2026-10-10, MER):** MER proposes the answer (`spec-merchants.md` 12.3): the effective window of a role with a merchant block is `max(WelfareWindow, WindowCycles * max(longest Duration, Kmax))`, `Kmax` the longest travel time over the graph of the self-carried modes of its block. Until WEL closes its open question 9, `trip_longer_than_window` is the only guard.
- **Update (2026-10-10, wave 4 integration, X4):** ruled in the one combined edit of `spec-welfare-and-wages.md`: the effective window (2.3.3) lists the recipes on the role's list, the effective own-use list and the longest trip of the role's modes; a recipe-less role with an own-use job, and a journey or a trip, are handled in 4.3; a migrant's journey is zero-income ticks in the window. The note stays open until WEL's open question 9 is closed.
- **Update (2026-10-13, EMP):** by contrast, a hired job is paid a tick at a time and is no gap in the worker's income (`spec-employment.md` 8.5).
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the merchant block does not exist in the slice, so `L` is the recipes' only; the window function takes `L` as a parameter so the merchants file appends its trip term. Left open.

### N-101 Does the default Target count Inbound?

- **Areas:** AGT
- **Raised:** 2026-10-10, `spec-transport.md` 9.1 and 20.7.
- **Note:** The field `Inbound` exists in the agent view. The default strategy never ships, but a consignee can be sent goods (TRN-A2): decide whether the default `Target` counts what is in flight to the agent.
- **Update (2026-10-10, MER):** the merchant counts its `Inbound` as a consignee (`spec-merchants.md` 8.2) and its `Outbound` as a shipper (7.1). The default strategy is never a consignee unless a consumer's strategy sends it goods, so its `Target` is still AGT's.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the view has no `Inbound` in the slice (transport is Stage 2) and the default ignores it until decided (`spec-agents.md` 21 question 20). Left open.

### N-103 Running several spec writers in parallel

- **Areas:** ALL
- **Raised:** 2026-10-10, the wave 2 run (REG, MON, POP and TRN written at once, then integrated).
- **Note:** What worked, so the next run repeats it, and what bit.
  - Each writer had its own worktree and branch and could create only its spec file and `doc/specs/models/<code>/`. It handed its edits to shared files over as a written handoff (notes, parameter rows, glossary, ledger reasons, config fields, register entries, events and keys, edits to other specs, checkpoint state, the `Testing` and `Observability` entries). The shared files collide when several writers edit them: `notes.md`, `specs/README.md`, `spec-roadmap.md`, `documentation_index.md`, `spec-ledger.md`, `spec-config.md`, `spec-stability.md` and the rules. After the writers finished, one supervisor wrote the rulings and the final numbering, and integrators, one per disjoint set of files, applied the handoffs. The merges were conflict-free.
  - Fix the interface between concurrent specs in the brief (who owns the region graph, the transit hook, the ledger reasons) and read the finished specs against each other. Two writers still disagreed with the specs around them: TRN had assumed an error that REG did not have, and POP showed that the spawner's signed hook could not work. The cheapest fix was a review message to the writer, with the owner of the interface winning. When one spec depends on another's interface, write the owner first if the schedule allows.
  - Writers use provisional IDs for register entries, dampers and scenarios (`REG-A1`, `POP-D3`, `TRN-S2`) and their proposals clash. The supervisor assigns the final loop, damper and scenario numbers (see N-046) and the note IDs, and each register entry names the provisional ID in brackets so the mapping stays traceable.
  - Checks to run mechanically after integration (the script lived in `/tmp` and is gone, so rebuild it with `perl` and `grep`, never the host Python): every `spec-x.md N.N` reference resolves (to a heading, a numbered paragraph or a table row); no requirement ID is defined twice; every file under `doc/` has a `documentation_index.md` row; the notes index equals the areas of the open notes; no `.pyc` is committed; the read-receipt tags changed only for the files that were edited. Run it on the base first: one reference was already broken (`spec-welfare-and-wages.md` to `spec-config.md` 13.5).
  - Traps. Say in the brief that models run only in Docker (three of four writers ran the host `python3` for small things anyway). A model run in Docker leaves a root-owned `__pycache__` that makes removing a worktree fail and invites a committed `.pyc`: set `PYTHONDONTWRITEBYTECODE=1`, or clean up with a container. A new worktree makes Claude Code ask whether to trust the folder, which is the user's decision. In herdr a pane ID contains a colon and an agent name must match `[a-z][a-z0-9_-]*`. On this machine `grep` is `ugrep` and rejects some patterns. Notifications from a monitor of an earlier session arrive as if new: check what the supervisor started before acting on one.
- **Update (2026-10-10, RES):** RES followed the process: own worktree, only the spec and `models/res/` created, shared-file edits handed over. Slip: the host `python3` was called twice (an empty stdin script and `--version`), nothing else; all models ran in Docker with `PYTHONDONTWRITEBYTECODE=1`.
- **Update (2026-10-10, EXT):** wave 3 followed the method. What bit: a model run first assumed a price rule that made the cobweb vanish (a midpoint with cost-based asks has a fixed point), so the cobweb model was rewritten around a uniform price and says so; a numeric claim copied from a first model run had to be recomputed when the cap was redefined on list value. Re-run the arithmetic of an example after any change of definition.
- **Update (2026-10-10, wave 3 integration):** wave 3 (RES, INF and EXT written at once, then integrated) added two lessons. (1) A seam between two concurrent specs stayed invisible to both writers: INF described a function `Informs` that EXT was to call, and EXT described an `OutsidePrice` read for INF; neither writer saw the other's side until the supervisor compared the finished specs. The supervisor's review fix removed both from EXT (no requirement ID reused or renumbered), and the decision that an outside node is not a rumour source in version 1 (N-112). (2) The fix reached only one side: after it `spec-information.md` still describes the declaration (4.5, FR-INF-002 and others), so when a review removes one half of a seam, send the message to the other writer too and grep the other spec for the removed name. RES was also revised after an independent review of its spec (commit 883eff5); the findings it accepted are written into the spec.
- **Update (2026-10-10, supervisor, wave 3):** what the supervisor learned running wave 3, for the next run.
  - Count the register IDs from the spec files, not from the handoff prose: grep each spec for its provisional `<CODE>-A`, `-D` and `-S` IDs, then decide the merges. EXT's handoff text implied four scenarios and its spec had five, which would have made two final numbers collide. The final numbers must be fixed before the integrators start, because the notes editor needs them before the stability editor has assigned them.
  - Look at a worker pane's footer before sending briefs. Claude Code prints a usage warning there ("You've used 91% of your weekly limit"). A run of three writers and four integrators is large; if usage is high, ask the user before launching. A writer that stops at the limit leaves its branch and handoff on disk, so work is not lost, only stalled.
  - Choosing integrators: reuse the writers' panes for the files whose edits need the spec's context (they finished with no conflicts), and give the long reads (`notes.md`, the roadmap and the indexes) to a fresh agent on a new branch off `main` when a writer's context is high (the RES pane was at 70%).
  - Add to the mechanical checks: every spec file on `main` has an `exists` row in the roadmap (the tier table and the existing-specs table), a row in the code table of `specs/README.md` and a documentation-index row; and read the roadmap diff, because the script cannot see a count that is out of date. The user requires the roadmap to be updated in the same change that closes a spec.
  - Re-read what an integrator wrote in the rules and in any spec it edited. In wave 2 an integrator passed text through shell quoting and left the literal sequence `'"'"'` where an apostrophe belonged, in `rules/observability-and-logging.md` and `spec-ledger.md`; it was found only on the next re-read. Grep the whole of `doc/` for it after every integration (`grep -rF "'\"'\"'" doc`).
  - Merging without extra branches (a standing preference of the user: no branches beyond `main`): merge the integrators' branches in a detached worktree (`git worktree add --detach <path> main`), run the checks there, fast-forward `main` to the verified commit, then remove the worktrees and delete the merged branches. Before removing a worktree check for uncommitted changes and for root-owned files (a Docker `__pycache__`), or the removal fails halfway and leaves an empty workspace.
  - A launcher script that splits a pane ID such as `w4:p1` on the colon sends the brief to a target that does not exist (`agent target w4 not found`); the first launch of wave 2's integrators sent nothing. Read the result file of every prompt before waiting.
- **Update (2026-10-10, wave 4 integration):** the seam check of the brief worked: the writers of SUB, MIG and MER read one another's drafts and fixed their mismatches while all three existed (MIG expected an "ending reason" of SUB that SUB does not have; MER adopted POP's Weyl offset instead of inventing a second spread). Slip: the host `python3` was called with no-op scripts; every model ran in Docker with `PYTHONDONTWRITEBYTECODE=1`.
- **Update (2026-10-13, the Stage 0 audit and its closing run):** what worked and what bit, for the next code stage. (1) Audit by phase, not by file: one read-only agent per phase of the plan (documents; types and helpers; validation and loader; events, observer and options) gave each requirement a source, a test and a severity, and the audit stopped at the first review with no blocking gap. (2) Close gaps in disjoint workstreams (validation; guards and pins; loader; documents), each owning named files, then one agent that records the plan ledger, the spec bound and the notes after the others have finished (the notes file has one editor), then two verifiers. (3) A test is not accepted until it fails when the behaviour is broken: every agent mutated a COPY of the repo under the scratchpad and ran the tests there, and the verifiers repeated it independently; one verifier found a document gap that the fixer had called closed (the old design constant in the model scripts had no comment in place). (4) Agents share one tree, so a package may fail to compile for a moment because of another agent's edit: say so in the brief and tell them to retry once and ignore files they do not own. (5) `doc/notes.md` is about 2 300 lines and over 80 000 tokens: the file reader refuses more than about 25 000 tokens a call, so read it in chunks of about 200 lines, and the tag at the end is not proof that you read it. (6) The host `python3` slip happened again (an `ast.parse` check of three scripts, which wrote nothing): use a container, or a Go or shell check, for a syntax check of a model script. (7) Nothing was committed and the user asks for the commit: the work of Stage 0 and of Phase 1 is one large uncommitted tree, so commit it (one commit per phase of the plan, A8) before Stage 1 starts, or a Stage 1 change cannot be read against it.
- **Update (2026-10-11, the eight tier 3 specs):** WLD, OBS, CHK, INT, IND, STY, EXP and CLB were written in that order, each against the finished specs before it (OBS, CHK and INT read each other's drafts at section boundaries; INT was written against the finished WLD), then integrated by eight jobs on disjoint files from one plan of rows (`<CODE>-nn`), the conflicts between rows, the rows with a wrong target and the rows the plan added, under written supervisor rulings. Slips: the host `python3` was called once by WLD, OBS and CLB, twice by IND and CHK, three times by EXP and once by STY (INT never); in the integration two jobs each called `python3 --version` once. Nothing but a version check ran on the host and every model ran in Docker with no bytecode. What bit: a row that quoted a rule text inexactly (the carve-out for a failed step, B.1 row 2), rows whose target section had moved or whose edit was already true (STALE), and one change of vocabulary (`internal/logkeys` as the home of reason constants) that touched about 40 lines of 20 specs and needed a grep list; CLB was edited three times while the plan was written, so its rows were held for a last job. Seams found by reading the finished specs against each other, not by the writers: INT and REG disagreed on the error for a pseudo-agent's ID, INT and MIG on who cancels a journey (INT won, as owner of the public call), WLD and INT on which readers work on a failed world (F1).
- **Update (2026-10-11, CLB):** CLB was written last, against the finished specs of the tier, and revised three times after reviews (22, 17 and 12 findings, all accepted) while the integration changed the working tree under it (44 files in one revision), so the owed rows that the integration had already met were removed from its edits table each time, and its rows were held for a last job that ran after the supervisor accepted it. What bit: a registry flag (`in`) first derived from a list that VIA 11 published only afterwards, so it was re-derived from the list; a README owner table that lacked `Taste.Role` and `RegionTaste`; counts that moved with every revision (the spec says "CLB's count"). Slips: the writer ran the host `python3 --version` once and `python3 -` with no script once, and no host python in the last revision; the job that applied its rows ran `python3 -` once with no script. Nothing was read or written by any of them; every model ran in Docker with `PYTHONDONTWRITEBYTECODE=1` and a read-only mount.
- **Update (2026-10-13, STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** the fourteen specs of the later stage were written as one stage with the method above (a file and `models/<code>/` each, handoffs of the shared edits, models in Docker), against the finished specs of the earlier stages and the drafts of one another, and integrated by nine jobs of one plan of rows under the written rulings F1 to F16 and R1 to R15. Slips, one line for all: the host `python3` was called by reflex by thirteen of the fourteen writers (PLC alone did not), as `--version`, `-c 1` or `-` with an empty stdin, a few times each, and by four integration jobs (J3, J5, J6, J7) once each; INS also had one command that would have run it denied by the harness. Nothing was read or written by any of them; every model ran in Docker with no bytecode. What bit: (1) the register numbers: six specs each wrote "next free L143, D99, S91", so the numbers were assigned once from the spec files after all fourteen were accepted (the entries are provisional until then); (2) a model run at generous parameters hid a structural error of the default (CRD: a slack of 5 ticks hid a one-tick error of the default horizon, found by review: run a model at the default of every parameter with the library's delays) and a model with no demand limit let capital compound (INS); (3) a first rule that the model, not the reading, showed to be wrong (EMP: profit with no outlet limits hired employment; CRD: the first run of the cycle model had every borrower able to finance its job, so credit looked inert); (4) draw IDs: RSK took D13 to D15, so TEC's is D16 and PLC's D17 (read the draw tables of the finished specs before choosing one); (5) a brief that said "IND measures" while the import rule of IND 3.1 forbids the engine to call an indicator (PLC); (6) seams found by reading the finished specs against each other and not by the writers: the removal steps (2b to 2g), the positions P1.7a and P1.7b that CRD and FIS both claimed, `PolicyFormed` against PLC's `Policy`, and whether INS's payout claim is in CRD's `CreditSettled`; (7) rows quoting a section that had moved or an edit already true (STALE) and two rows that misread the live areas of N-104 and N-108.
- **Update (2026-10-15, plan 1-3 Phase 1):** a process note, not for the clearing row. Rows of the Stage 1 slice are built one agent each on one working tree, so each re-reads a shared file (`doc.go`, `event.go`, `errors.go`, `option.go`, this file, the roadmap) immediately before editing it and never reverts a change it did not make. Left open.
- **Update (2026-10-16, plan 1-4 Phase 1):** applied in its lesson: the shared files (`doc.go`, this file, the roadmap, the index) are edited by targeted edits, and this file was read in chunks (point 5 of the update of 2026-10-13). Left open.
- **Update (2026-10-16, plan 1-5 Phase 1):** applied in its lesson: the shared files (`doc.go`, `event.go`, this file, the specs, the roadmap, the index) are edited by targeted edits after a re-read, and this file was read in chunks. Left open.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: a process note; the literal `'"'"'` search of `doc/` was run after this change and finds only the two passages that describe it. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: a process note. Applied in its lesson: the shared files (`doc.go`, `event.go`, `errors.go`, `option.go`, this file, the roadmap, `architecture.md`, `AGENTS.md`) are edited by small edits after a re-read, and this file was read in chunks. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: a process note. Applied in its lesson: the shared files (this file, the specs, the roadmap, the index) are edited by small edits after a re-read, this file was read in chunks, and the shell-quoting search of `doc/` named under Traps was run after the change. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Process note; the plan was aligned with siblings only through the coordinator.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. Process note; this plan was written beside five others and reads only their contract.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. Process note: the shared files (`doc.go`, `spec-config.md`, `spec-world.md`, this file) are edited by targeted edits after a re-read, this file was read in chunks, and the shell-quoting search of `doc/` named under Traps is run after the change.

### N-107 New money faucets and sinks start from MON

- **Areas:** FIS, CRD, MON
- **Raised:** 2026-10-10, `spec-money-supply.md` 4.5, 9.5 and 10.1.
- **Note:** Start from MON's mechanism: a grant or a levy on a class, a declared reason equal to the flow ID, closed for a run, with its boundary account created at construction, reported in `BoundaryFlow` and checked by the audit. Add a built-in ledger reason only for a mechanism that is not a grant or a levy (an interest flow, a trade with an external counterparty). MON's positions bind the later specs: no faucet reads an aggregate or grows with a balance (only a sink may be proportional to a balance), and a counterparty whose balance is created or destroyed outside conservation is a Boundary account. Tax, credit and the external sector are the likely first uses. A policy that changes a rate over time is PLC's, through the flow-scale slot (N-091).
- **Update (2026-10-10, RES):** RES adds no faucet, no sink and no ledger reason: a draw and a regeneration are not transfers and an endowment is in no account (`spec-resources.md` 6.2.1). Depletion is made visible by events and an exact per-endowment identity (6.4), which a test and the audit check.
- **Update (2026-10-10, EXT):** applied for EXT: a trade with an external counterparty is the case the note names for built-in reasons (`spec-external-sector.md` 8.1); the counterparty's account is a Boundary account's partner and holds nothing across a tick (5.3); no rule reads an aggregate (11.3); a policy that changes trade acts through the shock slot (6.5), the edge status (`spec-regions.md` 7) or a flow scale.
- **Update (2026-10-10, SUB):** applied for SUB: no faucet, no sink, no ledger reason; the form of the output is the ordinary recipe execution, and the in-kind channel was refused with reasons (`spec-subsistence.md` 6.2).
- **Update (2026-10-10, MIG):** MIG adds no faucet or sink (`spec-migration.md` 9.1).
- **Update (2026-10-10, MER):** the delivery contract moves money between non-boundary accounts and adds no faucet, no sink and no reason (`spec-merchants.md` 9.6.1); the three escrow reasons already in the registry change owner to MER.
- **Update (2026-10-13, Phase 1 of the ledger row):** applied for LED: the registry says which rows are built in and that every module row is declared with its module and has no boundary account while it is off (`spec-ledger.md` 3.4.1, 7.2.2, FR-LED-052). LED removed from the areas.
- **Update (2026-10-13, STO, EMP, LND, CRD, CAP, RSK, FIS, SAV, TEC, PLC, INS):** each starts from MON's mechanism and adds a built-in internal reason only where the counterparty is not a grant or a levy, as the second sentence allows: EMP (three wage reasons), LND (a rent is not a grant or a levy: two reasons, a tax on rent is a MON levy on the holder role or FIS's), CRD (a loan moves existing money and interest is a transfer: three reasons; a creating lender is a MON grant with its own reason and a `Limit`), FIS (the carry, the declared reason, the scale and no aggregate; a ruler's creating or destroying flow is a MON target kind `treasury`), INS (a premium and a payout). STO's levy for a store is a MON flow (`spec-storage.md` 7.1); PLC changes a rate over time through the flow-scale slot (MON 10.4) and adds no flow. POL's `StartMoney` is `initial` into a Treasury account (`spec-polity.md` 5.4.1) and the reserved MON target name for a treasury is `treasury`, which FIS adds. CAP, RSK, SAV and TEC add no flow.

### N-108 What the resources module leaves to land, subsistence, infrastructure, the outside and migration

- **Areas:** EXT
- **Raised:** 2026-10-10, `spec-resources.md` 10, 9.4 and 23.
- **Note:** Endowments are open access: any recipe with `Draws` on an endowment of its region may take from it, and the ration shares a scarce stock (5.3). LND adds the tenure through three hooks: a refusal after `no_endowment`, a per-claimant entitlement applied to the request before the ration, and a rent as a named money flow valued at the dual of the resource row (`Rent`, `spec-resources.md` 11). SUB: household production may draw on an endowment through `Draws` like any recipe. CAP: a facility that raises capacity is a layer of the effective capacity and writes no slot (`spec-infrastructure.md` 5.3; reworded 2026-10-13, it said "uses the capacity-scale slot (6.3)"). EXT: the import predicate is the second half of the sourcing of a raw good (9.4). MIG: nothing in RES moves households; the signal of a depleted region is the welfare ratio.
- **Update (2026-10-10, SUB):** applied for SUB in `spec-subsistence.md` 9: an own-use recipe draws on an endowment through `Draws` like any recipe, open access, the ration of `spec-resources.md` 5.3, no priority and no quota; its requests carry the recipe, so LND's entitlement may treat them differently, and a closed commons makes the household `Blocked` and draws `own_use_not_available` (9.5). LND, CAP and EXT stay.
- **Update (2026-10-10, MIG):** applied for MIG in `spec-migration.md` 4.4 and 6.3.3: nothing in RES moves households; the signal of a depleted region is the household's own earned ratio and measured wage; a destination's stock, endowment or abundance is state and is not read by an estimate (the estimate uses the table value of the yield). LND, CAP and EXT stay.
- **Update (2026-10-13, LND):** applied for LND in `spec-land.md`: the three hooks of RES 10.2 are the refusal (4.5), the entitlement (4.2, 4.3) and the rent (5); SUB 9.5 and MIG are answered (7.1, 7.2). A jurisdiction may be a holder through its treasury's agent ID (`spec-polity.md` 5.4.2) but `spec-land.md` 3.2.3 lists it as not offered in version 1; the holder other than an agent is LND open question 10.
- **Update (2026-10-13, CAP):** applied for CAP (`spec-infrastructure.md` 5.3): a ground facility raises an endowment's capacity as a layer multiplied into the effective capacity, `floor(Capacity * scale * (10 000 + B) / 10^8)`; it never writes the capacity-scale slot, which stays the consumer's and PLC's. The wording "uses the capacity-scale slot" above is reworded.
- **Update (2026-10-13, FIS, ORG, SAV):** a tax on rent or on titles is a FIS tax on the holder role (`spec-fiscal.md` 5) or, as a tax on a reason, N-187; a right over a recipe that draws on an endowment is read before `no_endowment` (`spec-institutions.md` 5.5.4); a ground facility is a project like any other, valued only by a `Worth` row because its gain accrues to holders and drawers (`spec-infrastructure.md` 5.3.4). LND and CAP leave the areas; EXT keeps the import predicate.

### N-109 The closed forms of the Malthusian loop need the welfare ratio at a full draw

- **Areas:** POP, VIA
- **Raised:** 2026-10-10, `spec-resources.md` 8.2, 8.5 and 23 (item 6).
- **Note:** `N* = N_bind * r_full / r*` (flat curve) and the stock share `a(sigma*) = 10 000 r* / r_full` (curve) need the welfare ratio a household reads at its full draw and full abundance, `r_full`. It is a property of the pack's economy and not a field. Decide whether a run measures it (a snapshot figure), whether VIA computes it from the programme with wants, and whether a warning `population_cannot_grow` (`r_full <= r*`) belongs to POP or VIA.
- **Update (2026-10-10, SUB):** with the module on the welfare ratio of a floor-fed household includes its receipts (`spec-subsistence.md` 7.4): `r_full` for such a household is the floor's ratio, below the replacement ratio of a market-fed one, so the schedules form fewer households and dissolve more among them.
- **Update (2026-10-11, OBS):** rejected as a snapshot field: `r_full` is a counterfactual, not a state (`spec-observability.md` 5.8). OBS removed from the areas.
- **Update (2026-10-11, IND):** no preset can compute `r_full` from a run; a proxy exists, the ratio of the households that draw at full abundance in a run with the resource unbound, as a consumer's measure over `RecipeExecuted` rows. The solver is the place for the figure (`spec-indicators.md` 21).
- **Update (2026-10-13, TEC):** a technology gain scales `r_full`: `N*` is proportional to `T` for a quota and `a(sigma*) T = 10 000 r* / r_full` for a stock (`spec-technology.md` 13.1); a logistic stock is pushed past 1/2 at `a(5 000)(10 000 + CapBP) >= a(sig) 10 000` (10.3).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for POP and RES. `r_full` is not computed in the slice (no endowment, no population module).

### N-110 Version 1 rumours prices only

- **Areas:** SKL, INF
- **Raised:** 2026-10-10, `spec-information.md` 17 (open question 4).
- **Note:** MIG's rumoured welfare ratio (`spec-welfare-and-wages.md` 5.1) and SKL's rumoured premium (`spec-skill-barriers.md` 2.5) have no source: they are regional aggregates, so whether an agent may react to one is a question for "macro is measured, never set" (roadmap). Decide in `spec-migration.md`.
- **Update (2026-10-10, MIG):** decided for MIG in `spec-migration.md` 4: a household does not react to a regional mean welfare ratio (an indicator, a controller input in disguise); it reads its own state, the structure of the world (graph, recipes, yield table, upkeep lists) and rumoured PRICES, from which it derives a destination's income (6.3). INF 17.4 is answered. The same decision applies to SKL's rumoured premium (`spec-skill-barriers.md` 2.5), which stays open for SKL.

### N-111 A view takes draws

- **Areas:** AGT
- **Raised:** 2026-10-10, `spec-information.md` 5.5.2.
- **Note:** The view of an informed agent is built immediately before its `Trade` and, with `RumourErrorBP` above 0, takes draws from the world's source in a fixed order (one per candidate). AGT's view building and WLD's phase 3 must keep that order; the draw count is a function of the configuration and of the informed agents present. `spec-agents.md` 6.1.2 (the stream a strategy sees is a function of the world's state) still holds because the order is fixed. OBS: `RumoursServed` and its metric.
- **Update (2026-10-10, MIG, X3):** a second reader. The migration pass of phase 7 reads rumours for the household under review through INF's read function, whatever the role's `Informed` flag says, and takes INF's error draws itself: among the reviewed households in ascending agent ID, one draw per source pair (candidate) whether or not a rumour is offered, none when `RumourErrorBP` is 0 or no agent is pushed (`spec-migration.md` 6.2.1, 10.2.1). They belong to the migration pass of phase 7 and never to a view of phase 3, so INF's stream (before each `Trade`) and MIG's (after the entry step) cannot interleave, and the draw count in a tick stays a function of the configuration and of which agents are pushed. `spec-information.md` (the reader, 5.5.2, 7, and the requirement that rumours reach only `Informed` views) carries the matching edit. AGT, WLD and OBS keep the fixed order of both; `RumoursServed` does not count MIG's calls.
- **Update (2026-10-11, WLD):** applied for WLD in `spec-world.md` 7.3: the view of an informed agent is built immediately before its `Trade`, its draws in ascending source `RegionID` then `CommodityID`; the migration pass takes its own at P7.2, so the two cannot interleave; the draw count is a function of the configuration and the informed agents present (D4, D11).
- **Update (2026-10-11, OBS):** `RumoursServed` counts the views of phase 3 only; MIG's calls are not in it (`spec-observability.md` 4.1.7); no new metric. OBS removed from the areas.
- **Update (2026-10-11, CHK):** a restore takes no random word; superseded by the update of 2026-10-12: a restore saves no PCG state (FR-CHK-051).
- **Update (2026-10-12, ADR 0008):** the order of D4 and D11 no longer matters for alignment: each is a stream keyed by (site, tick, agent), so the two cannot interleave by construction and a restore resumes from the seed, not from a saved PCG state (N-214). The fixed order stays for the event order of `RumoursServed`.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: no informed role exists in the slice. Learned: the view of an informed agent is built before its `Trade`, and D93 keeps the draw count of the default a function of the view (candidates are decided by the view alone, `spec-agents.md` 7.3.5). Left open.

### N-112 An outside node as a remote information source

- **Areas:** EXT, INF
- **Raised:** 2026-10-10, `spec-external-sector.md` 9.2 and open question 10, `spec-information.md` 4.5 and 17.
- **Note:** In version 1 an outside node is not a rumour source: INF records only the published price of a market and a node has no market, and EXT has no `Informs` declaration and no outside-price field (FR-EXT-033). A merchant away from a gate therefore never hears of an outside price. The minimal shape, if a setting needs it, is a second record source in INF (a quote function that EXT supplies as a pure read) and an EXT declaration of the (region, commodity) pairs it serves. The INF hook `Informs` stays reserved and unused. `spec-information.md` 4.5, 4.5.2 and FR-INF-002 still describe a declaration by EXT and must be read as reserved until INF is revised.
- **Update (2026-10-10, MER):** the market of a gate region is a rumour source like any market, so an interior merchant hears of the gate's published price (which includes the outside's trades) with the lag and the error of any rumour; only the node itself is not a source (`spec-merchants.md` 11.3).

### N-113 A quote is a limit, not a peg

- **Areas:** CLR, STB, STY, EXT
- **Raised:** 2026-10-10, `spec-external-sector.md` 6.7.
- **Note:** A counterparty's trade executes at the midpoint between its limit and the realm's, so the realm's price is pulled toward the quote and not held at it; the outside keeps the price improvement. A trade at the counterparty's own limit needs a flag on `Order` and a rule in `spec-clearing.md` 5.1 and 8. Run scenario EXT-S1 (S51: the pull of beliefs toward the quote at the default clearing); if it is too weak for a setting that wants a fixed price, CLR adds the flag and the rule.
- **Update (2026-10-11, STY):** STY-T03 is the decision input: the share of ticks near the quote (`t03.pull_bp`, `t03.quote_band_bp`); CLR adds the flag if it falls below it for a setting that wants a peg (`spec-stylised-facts.md` 5.6).
- **Update (2026-10-15, plan 1-3 Phase 1):** left open. The midpoint rule is kept and `Order` gets no "trade at my own limit" flag (`spec-clearing.md` 8). Learned: if the flag is ever added it is one field of `Order` and one branch of the price in `Average`, so the shapes of 8 and 8.3 leave room for it. STY-T03 and the external-sector row decide.

### N-114 The open layer of the steady state

- **Areas:** VIA, REF
- **Raised:** 2026-10-10, `spec-external-sector.md` 12.1.
- **Note:** VIA adds optional import and export columns per gate and side (bounded by `Capacity` times the mean window), priced at the quotes at tick 0 with the gate scale, and a trade row (imports at the quotes not above exports at the quotes); EXT supplies `Lines()` as a pure read of the configuration. An import counts as a source in the gate regions for V1. REF builds an external overlay (EXT-S5, S55; N-084).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for EXT. The programme is closed; the type keeps no import column.

### N-118 One job per agent limits the subsistence floor

- **Areas:** PRO, WEL, SPN
- **Raised:** 2026-10-10, `spec-subsistence.md` 5.7, 10.3, 8.3 and 21 (items 1, 4, 7).
- **Note:** A household in a long role job cannot take the own-use mode until the job ends, which is the lean season of a long-job role; a cohort cannot work an own-use recipe and a role recipe at once; the default spawner's rung 1 abandons a running own-use job. Decide, with the first scenario that shows the case, between a second light job slot, an interruption with an abandon reason, and a rule in the spawner that skips a household with a running own-use job.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open for the first scenario that shows it. The job type is one slot (`spec-production.md` 5.1.2).
- **Update (2026-10-17, plan 1-6 Phase 1):** read: not this row's (SUB's mode and the own-use list do not exist in the slice). Learned: the window function must count the own-use list when SUB lands (WEL 2.3.3), a one-line append; WEL 4.3 now says how an own-use job counts for a role with no recipes (the weight less the own-use `k`). Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The default's rung 1 does not read OwnUse (SUB is Stage 2).


### N-119 The receipts of a household that feeds itself

- **Areas:** WEL, EST
- **Raised:** 2026-10-10, `spec-subsistence.md` 7.5, 8.4 and 21 (items 2, 8).
- **Note:** The units of an own-use job are counted in the welfare ratio at creation, and a sale of the surplus is counted again as cash; relief may pay a household whose ratio is below the line only because its floor covers less than its upkeep. Decide a provenance, a consumption-side count, or accept the bound.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: `K^own_t` is a field of the record and stays 0 in the slice; the provenance question is SUB's and EST's. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: relief is not built in the slice; the provenance question is SUB's and EST's, for the module. Left open.

### N-120 A migrant carries its whole account for free

- **Areas:** TRN, MER
- **Raised:** 2026-10-10, `spec-migration.md` 9.4 (MIG-A9).
- **Note:** TRN limits a traveller's load by a mode's capacity (`spec-transport.md` 3.2.2); a relocation has no capacity. The default strategy keeps holdings small, so the leak is small, but a consumer's strategy that stockpiles could ride a journey for free carriage. Close it with a carry limit (mass and bulk per household) if a case appears.
- **Update (2026-10-11, INT):** a forced `Relocate` carries the whole account too and has no capacity (`spec-integration.md` 4.5.4).
- **Update (2026-10-13, STO):** applied for a limited role: an owner of a store cannot move (`has_store`) and a household's account fits its base room, so a migrant or a forced move carries at most `w` times the base room (`spec-storage.md` 9.5). A pack closes the rest by giving the role a base room. TRN and MER stay for the unlimited case; STO leaves the areas.
- **Update (2026-10-13, RSK, INS):** a migrant is not exposed to a risk row, and a journey is not a shipment (`spec-risk.md` open question 7); so it is not insurable (`spec-insurance.md` 1). N-181.

### N-121 A role with no recipe: merchants, the floor and migration

- **Areas:** MIG, SUB
- **Raised:** 2026-10-10, `spec-merchants.md` 6.4 and 12.4.
- **Note:** SUB lets a role with no recipe have an own-use list; the default `Work` a merchant inherits then starts an own-use job and the shipment mechanism refuses a trip (`job_in_progress`). MER keeps the merchant from planning a trip meanwhile and while it is in the own-use mode (the SUB qualifier on "a role with no recipe requests no job" is a supervisor erratum of `spec-merchants.md` 3.3). MIG ends the review of a no-recipe role in `no_estimate`, so no merchant is moved by MIG in version 1; if MIG gains an income estimate for such roles it must leave a merchant alone, since a merchant's place is where its trade is.

### N-122 First-come caps: where a budget serves candidates in a fixed order

- **Areas:** SPN, MIG, MON, MER, STB
- **Raised:** 2026-10-11, `spec-world.md` 10.2 (WLD-A2, L118).
- **Note:** Five places serve candidates in role, ID or region order, so a cap or a budget that binds is first-come: the entry budget and births served in role order, the caps of SPN and MIG with ties by ID, a money flow's limit cut in recipient order, and the contract round's lanes (the places O1 to O5 of `spec-world.md` 10.2). The remedies the world proposes (a keyed tie, a rotating start, a pro rata cut: 10.1.1, D87) are not adopted for them; D87 is the rule for a new spec. Each owner decides whether the unfairness of a binding cap matters for its module. Scenario S73 measures the fairness of a binding cap. The counts of a binding cap are in the owners' events (N-045), so a reader can see when one binds.
- **Update (2026-10-13, STO, EMP, SET, LND, CRD, CAP, RSK, FIS, ORG, SAV, PLC, INS):** the places of `spec-world.md` 10.2 were five and are now seven: O9 is SET's `Role.Ceiling` (entrants in need, then ID order; counted by `RoleCeiling`; "a sixth first-come cap" in SET's own words) and O10 is ORG's places of a body (seniority, then the keyed tie), registered with O1 to O5 as one hazard, WLD-A2 = L118 (10.2.1); D87 is the rule for a new spec. No other module adds a place: EMP orders the labour round by value then the stateless key and caps it per region; CRD and INS settle pro rata with the largest remainder and order a round by limit then the stateless key; CAP's `MaxFacilities` is refused in the rotation of the starts; SAV adds no cap; FIS cuts the treasury's balance pro rata with the keyed tie and applies the taxes of one payer in flow ID order (a stated rule); LND breaks every tie by the stateless key; RSK serves nobody first (its draws are exchangeable); PLC orders actors and levers by ascending ID and refuses its one key conflict in the data; STO is first come inside one room and at one consignee (STO-A4, merged into L118).
- **Update (2026-10-17, spawner edits):** read; not settled by them. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The caps of SPN 6.3 are first come in the order of need; WLD's D87 is not adopted for them and the ID is the last key. Scenario S73 measures it.

### N-123 A cold start is a lockstep

- **Areas:** CFG, VIA, REF, STY
- **Raised:** 2026-10-11, `spec-world.md` 15.2 and question 2 of 22 (WLD-A3, merged into L22).
- **Note:** A pack with no warm start, or a warm start that gives a role no job row, starts every group's first jobs together, so they complete together, a pulse of every `Duration` ticks. The informational warning `cold_start_lockstep` counts it at construction and never fails validation (`spec-config.md` 7.4.2, FR-CFG-082). REF reports the cold leg of S18 (S75 is the same leg) and does not assert it. Whether the world should stagger a cold start itself (a world-level stagger of the first jobs) is open question 2 of `spec-world.md`; the warm start's fixed stagger is VIA's (`spec-viability.md` 8.5).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. The warm start's stagger is the world row's (`spec-viability.md` 8.4.3); this row gives the offsets. The warning `cold_start_lockstep` exists (Stage 0).

### N-124 A paired comparison is exact only while aligned

- **Areas:** EXP, STB, CLB, WLD
- **Raised:** 2026-10-11, `spec-world.md` 7.6 (WLD-A5, L120), `spec-experiments.md` 7.4 and `spec-checkpoint.md` 11.
- **Note:** Two runs of one seed that differ in a parameter or a shock draw the same words only while no draw count moves: from the first tick in which a count differs, the streams are offset (the stream is positional). Until then the comparison is exact; after it, the estimate of a difference of means from seed-matched runs is still unbiased, and the matching buys only variance. EXP measures the gain (`models/exp/pairing.py`: a gain of 4.0 with 7 of 200 seeds aligned at the end, 1.2 to 1.3 in a world with a large count offset) and fingerprints the stream. A consumer must still say which comparison it means, paired or over seeds (STB D88, D90). One stream is fixed by `architecture.md` 7.3; per-purpose streams are open question 4 of `spec-world.md` 22.
- **Update (2026-10-11, CLB):** applied in part by CLB: a knob with the flag `d` (it selects how many items a term of the draw count of `spec-world.md` 7.3.1 counts) is compared as distributions over seeds and never paired; a paired comparison is used only for a knob that moves no count, and the alignment horizon is recorded where CHK offers the read (`spec-calibration.md` 4.2.1, 7.2.3, FR-CLB-024). The note stays open for EXP and WLD.
- **Update (2026-10-13, EMP, CRD, CAP, RSK, SAV, TEC, PLC, INS):** which parameters are the flag `d` of CLB: EMP, CRD and INS take no word of the world's stream (EMP's ties are by the stateless key `K(t, id)`), so none of their parameters is `d`; RSK: a pair across a scale of 0 is misaligned from there (the row takes no word at 0) and a sweep above 0 stays aligned (`spec-risk.md` 11.4, N-179); SAV takes no draw but a target above the reserve lowers the budgets that admit an order (WLD 7.3.1, D7 and D8), so every field carries `d` and a pair is exact only until some household's target exceeds its reserve (`spec-saving-and-investment.md` 9.2); TEC takes one word at construction only when a field has `DiscoveryBP` above 0 (D16), so a pair is aligned from construction except one that turns discovery on or off (N-203); PLC's built-ins draw nothing and a consumer's policy that draws is PLC-A7 (a case of L120). The modules whose presence or binding shifts a count (POL, FIS, ORG) are N-162.
- **Answer (2026-10-12, ADR 0008):** per-purpose streams are adopted, keyed by (seed, site, tick, entity): a count change moves only the later draws of its own stream, so two runs of one seed are aligned by construction and diverge through state only; the fingerprint of EXP 7.3 and the flag `d` of CLB 4.2.1 go. The note stays open until WLD 7, EXP 7 and CLB 4.2.1 are rewritten (N-214); the pairing gain is still a measured number (EXP 7.4), now of the state's divergence alone.
- **Update (2026-10-12, Phase 1):** the WLD part is applied: WLD 7.6 says paired runs are aligned in the draws by construction (model: 98.00% of later draws differ under the positional stream, 0.01% under keyed streams). Open for EXP 7.3 and 7.4 (the pairing gain is measured on the divergence of state alone) and CLB 4.2.1 (the flag `d` goes).
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open for EXP, STB and CLB. The world adds no draw (FR-WLD-062); the draws are keyed (ADR 0008).
- **Update (2026-10-23, plan 2-1 Phase 1):** the CHK part is applied: the EXP row of CHK 11 and CHK-S2 say that a paired comparison is aligned in the draws by construction (`spec-world.md` 7.6) and that the stream fingerprint of EXP 7.3 reads nothing; the WLD part was applied earlier and the 13 EXP row is now the same. Still open for EXP 7.3 and 7.4, STB and CLB 4.2.1.

### N-125 A snapshot is a function of saved state

- **Areas:** CHK, LED, WLD, OBS
- **Raised:** 2026-10-11, `spec-observability.md` 5.2 and `spec-checkpoint.md` 4.4, 4.5 (F1), 6.2 and 6.7.2.
- **Note:** Every snapshot field must be derivable from what CHK saves, so a restored world's snapshot equals the original's (FR-OBS-043). Consequences: the market report of the last clearing (volume, unmatched units, best unmatched quotes) is boundary state that `Trade`, INT's `Market` and the snapshot read, so CLR holds a state section for it (`spec-clearing.md` 16, FR-CLR-034) and the checkpoint saves it whole; a count of the last tick is an event, not a snapshot field; a checkpoint taken before the first step keeps the construction phase and its bucket, so a call between ticks on the restored world is totalled as on the original and `BoundaryFlow` agrees; the audit baseline after a restore is the restored balance less the pending totals that calls between ticks added since the last P8.6 (`spec-ledger.md` 9.1). A spec that adds a snapshot field adds its state to the inventory (N-128).
- **Update (2026-10-13, Phase 1 of the ledger row):** for LED: the audit's per-tick baseline of the boundary balances (`plans/plan-phase-1-1.md` A7) is not saved. After a restore it is the restored balances less the pending totals saved with them (`spec-ledger.md` 9.1, 11), so the checkpoint needs no field for it; the ledger records it when it clears the totals at P8.6 and a restore rebuilds it. The ledger code is Phase 6 of the plan.
- **Update (2026-10-13, STO, LND, CRD, CAP, RSK, FIS, ORG, SAV, TEC, INS):** every snapshot field the later stage adds is a function of saved state: STO's room, load and pledge (derived from the stores and holdings saved), LND's rows (13.4), CRD's `Claims` and `Credit` (the claim list and its totals), CAP's rows and queries, FIS's `HeldByTreasury` and the treasury sector, SAV's four `RoleRow` fields (`SavingHouseholds`, `GoalMoney`, `BufferMoney`, `IdleMoney`), TEC's `TechRow` (the saved cells), INS's `Policies` and `Insurance`.
- **Update (2026-10-15, plan 1-3 Phase 1):** applied in part for CLR, which leaves the areas: the market report is exactly the fields of `spec-clearing.md` 16 and nothing else (the type `marketReport` in `clearing.go`, held per market by the world row's `market.go`), so a checkpoint saves it whole and a snapshot derives from it. Stays open for CHK, LED, WLD and OBS.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open. The create-and-dispose operation and the holding read add no saved state and no snapshot field (the flow totals are cleared at P8.6 and rebuilt after a restore); the one saved field the row adds is the job's `Yield` in tag 6 (N-237), a function of saved state.
- **Update (2026-10-16, observability review of plan 1-5):** applied in a decision: the cost anchor is reported by an event field (`MarketCleared.CostAnchor`), not a snapshot field, because the anchor table is recomputed each tick from the previous tick's prices and is not saved (`spec-production.md` 15), so a snapshot of a restored world could not equal the original's, and "a count of the last tick is an event, not a snapshot field" (above). The jobs gauge uses the existing `Jobs[]` rows. Left open.
- **Update (2026-10-17, plan 1-6 Phase 1):** applied to the new field: `RoleRow.ClampedHouseholds` is derived from the saved records (the count of rings holding a field at the bound is rebuilt on restore and never saved) and the clamp run joins the diagnostic runs of `spec-checkpoint.md` 4.2 tag 7, so a restored world's snapshot and logs equal the original's (FR-OBS-043). Left open for the other modules.

- **Update (2026-10-18, observability review of plan 1-8):** applied to the new field: `Markets[].EstateAtFloor` is the units of the estate's listings priced at the resolution floor, derived from the saved listing rows (`estates.rows`, checkpoint tag 8) and the lots the estate holds, so a restored world's snapshot equals the original's. Left open for the other modules.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply for the slice: every field of the slice's snapshot (D146) is a function of saved agent, ledger, market and welfare state, and the market record's report and window are saved state that the world holds in one place (Phase 3). Left open for CHK.
- **Update (2026-10-20, observability review of the branch of plan 1-11):** applied to the field the review adds to the slice's snapshot: `MarketRow.EstateAtFloor` (appended last, a `Quantity` count, omitted at 0) is derived from the saved listing rows and the lots the estate holds, so a restored world's snapshot equals the original's once the checkpoint row saves tag 8. Left open for CHK.

### N-126 A new event or `Warn` adds a row

- **Areas:** ALL
- **Raised:** 2026-10-11, `spec-observability.md` 4.1, 7.3 (FR-OBS-003, FR-OBS-051).
- **Note:** A new event adds its row to the table of `spec-observability.md` 4.1 with its owner, position, class and gate; a new `Warn` adds its row to the table of 7.3 with its counterpart event or snapshot field. A test reads both tables (FR-OBS-003, FR-OBS-051), so an event or a `Warn` with no row fails it, and the rules' catalogue (`rules/observability-and-logging.md` 4.3) changes in the same change.
- **Update (2026-10-13, EMP, SET, CRD, SAV, TEC, INS, FIS, RSK):** where informative: the later stage adds events and few `Warn`s, and the rows are in the owners' event and key tables and in the catalogue of `rules/observability-and-logging.md` 4.3 (J1 applied them once): EMP three events and two `Warn`s (`HireFormed`, `HireEnded`, `LabourRound`), SET two events (`SettlementClassChanged`, `SettlementHeld`) and none, CRD six events and two `Warn`s ("claims overdue", "claim cap reached"), SAV one (`SavingPlanned`), TEC two (`TechnologyAdvanced`, `TechnologyAdjusted`), INS four (`CoverFormed`, `CoverEnded`, `CoverRound`, `CoverSettled`) and no `Warn`, POL none and two configuration warnings (`jurisdiction_empty`, `region_unassigned`, a row owed to `spec-config.md` 7.4.2), RSK three (`StoreLoss`, `RouteRiskScaled`, `StoreRiskScaled`), FIS two (`FiscalFlowApplied`, `TreasuryOrdered`).
- **Update (2026-10-12, Phase 1):** the catalogue test is staged (decision D3 of the Stage 0 plan): it checks the events and fields that are declared, and one explicit `notBuilt` list names the rest and fails when a listed entry exists or a row is neither declared nor listed (FR-OBS-003, 004; `rules/observability-and-logging.md` 4.3). A module's change declares its events, appends its fields at the end of an event, moves them to the end of the row in 4.1 and deletes its entries from the list in the same change.
- **Update (2026-10-15, plan 1-3 Phase 1):** the clearing row declares no event and no `Warn`: its four events exist since Stage 0 and it first emits them; the staged catalogue test is not changed. Left open.
- **Update (2026-10-16, plan 1-4 Phase 1):** the beliefs row declares no event and no `Warn`: `BeliefsUpdated` has been a row since Stage 0, the `Warn` "price beliefs at bound" is row 3 of `spec-observability.md` 7.3 and the `Debug` "belief updated" is in its 7.5 list. Its five `reason` values are log-only constants (`spec-beliefs.md` 12.3). Left open.
- **Update (2026-10-15, observability review of plan 1-3):** a reason is not an event, so no row is added to 4.1 or 7.3, but the closed set of `economysim.reason` for `economysim.market.orders.rejected` grew from ten to twelve: the `overflow` row of `spec-clearing.md` 3.2 is now three, `overflow` (the int64 product), `holding_limit` (a bid over `MaxQuantity`) and `receipt_limit` (an ask over `MaxMoney`), so that the two settlement bounds are two series. A module that credits a buyer's stock or a seller's balance between P3.2 and P4.1 (`architecture.md` 2.10.1) adds its worst case to the same two bounds and so to the same two reasons. Left open.
- **Update (2026-10-16, plan 1-5 Phase 1):** the production row declares no event and no `Warn`: `RecipeExecuted`, `RecipeAbandoned` and `ProductionHalted` and the two production `Warn`/`Debug` lines have rows since Stage 0, and the row emits none (the world row emits behind the observer flag). The reasons a slice start can be refused with are five (`spec-production.md` 4.2.1); they are log-only values of `economysim.reason`, not an event. Left open.
- **Update (2026-10-16, observability review of plan 1-5):** applied for PRO and OBS: the refusals are now an event. `RecipeRefused` is a new class A core row of `spec-observability.md` 4.1.3 and of `rules/observability-and-logging.md` 4.3 (4.4.6a, FR-OBS-095; `spec-production.md` 14.1, FR-PRO-052), declared in `event.go`, with the five reasons as `economysim.recipe.refused` values, so the five are no longer log-only. The same change appends `RecipeExecuted.Overflow` and `MarketCleared.CostAnchor` (FR-OBS-096, 097) and adds the metrics `economysim.recipe.abandoned`, `.overflow`, `.in_progress` and `economysim.production.cost_anchor`. The world row now calls them (built). Left open for the adapter.
- **Update (2026-10-17, plan 1-6 Phase 1):** applied: the welfare row adds no event and one `Warn`, "welfare record clamped", as row 26 of `spec-observability.md` 7.3 with the counterpart `RoleRow.ClampedHouseholds` in 5.3.4 (FR-WEL-042); the prose counts of 7.3.1 are updated (20 to 26, about 350 more, about 2 050). The rules' event catalogue (4.3) is unchanged. A test reads both tables, so the row is in the same change as the code (Phase 4). Left open for later rows.
- **Update (2026-10-17, observability review of plan 1-6):** applied: the welfare row adds three snapshot-fed gauges and no event and no `Warn` (`economysim.welfare.ratio.households`, `economysim.welfare.floor_held`, `economysim.wage.observed`; `rules/observability-and-logging.md` 5.3 and 5.3.1, `spec-observability.md` 6.2.2, FR-OBS-098). Each reads a snapshot field that is saved state (N-125), so none adds state; the series bound of 6.5.2 gains 300 (100 each). The `Warn` "work floor binding" fires only above half for ten ticks, so the gauge `floor_held` is the view of a role approaching or leaving it. Left open for later rows.
- **Update (2026-10-17, plan 1-7 Phase 1):** read for the agent row: it declares no event and no `Warn` (`AgentSpawned`, `UpkeepShortfall` and `ProductionHalted` exist since Stage 0 and the world row emits them; the "households unfed" `Warn` is the world's). The three reasons it adds (`rest`, `halted`, `no_recipe`) are log-only constants and not events. Left open.
- **Update (2026-10-17, observability review of plan 1-7):** applied: the agent row's data gets three appended fields of existing events and no new event. `ProductionHalted` gains `Rested` and `NoRecipe` (counts of households, so the rest and the no-recipe decline are no longer log-only: the reasons stay log-only values and the metric `economysim.work.declined` has its own attribute `economysim.work_decline`), `UpkeepShortfall` gains `Short []ShortRow` (the upkeep units needed and not held, by commodity; `economysim.upkeep.short`), and the gauge `economysim.household.privation.max` reads the existing `RoleRow.MaxPrivation`. Rows changed in `spec-observability.md` 4.1, 4.4.6, 4.4.6a, 4.4.6b, 6.2, 6.5 and FR-OBS-016, 095, 099 to 101; the rules' 4.3, 5.3, 5.3.1 and 5.4.1 changed with a fresh tag. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate row declares no event and no `Warn`: `EstateOpened` is a Stage 0 core row that the world row emits, and what `GoodsListed` counts (lots) is stated once in `spec-observability.md` 4.1.4 and the rules' catalogue 4.3 with no new row. Left open for later rows.

- **Update (2026-10-18, observability review of plan 1-8):** applied: the estate row adds no event and no `Warn` and two metrics, `economysim.estate.sold` (from `TradeExecuted` with an estate as `Seller`) and the snapshot gauge `economysim.estate.at_floor` (`rules/observability-and-logging.md` 5.3, 5.3.1; `spec-observability.md` 5.3 `EstateAtFloor` and 6.2). The ask side of `economysim.market.unmatched` includes the estate's unsold lots; both catalogues now say so. Left open for later rows.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The row declares no event and no Warn; the staged catalogue test is unchanged.
- **Update (2026-10-18, observability review of plan 1-9):** applied: the spawner row adds no event and no `Warn`, one appended field (`EntryLimited.Stranded`), three metrics (`economysim.household.entries` from `EntryLimited.Entered`, and the snapshot gauges `economysim.role.expected_ratio` and `economysim.role.viable` from the existing `Roles[].ExpectedRatioBP` and `Viable`) and one attribute value (`stranded`) (`rules/observability-and-logging.md` 4.3, 5.3, 5.3.1, 5.4.1; `spec-observability.md` 4.1, 6.2, 6.5, FR-OBS-102, 103). FR-SPN-035 now says the event is emitted when any count is above 0, as `limited()` does. Left open for later rows.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied. This row adds no event and no `Warn`: the solver logs nothing and the warnings are the existing "configuration warning" line with `rule` and `path` (CFG 7.2). Each new warning rule (`region_not_self_sufficient`, `subsistence_exceeds_expected_work`, `target_above_surplus`, `storage_gap_input`, `viability_check_skipped`, `cycle_cost_gain`; `raw_without_source` already exists) is a row of CFG 7.4.2 already. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The row adds no event and no `Warn` row (all are rows already); the catalogue test is staged and Phase 10 moves the world-emitted events out of `notBuiltEvents` only if they were listed.
- **Update (2026-10-20, observability review of the branch of plan 1-11):** applied: the review adds no event and no `Warn` and two snapshot-fed gauges that give existing `Warn`s a metric: `economysim.welfare.clamped` (`Roles[].ClampedHouseholds`, the counterpart of row 26 of `spec-observability.md` 7.3, whose `Warn` fires only on the first tick of a run and then every tenth) and `economysim.role.agents` (`Roles[].Agents` by role and region, the counterpart of "role extinct" and "group below floor", which the event-fed `economysim.agent.count` by role alone cannot show for one region). Rows in `rules/observability-and-logging.md` 5.3 and 5.3.1 (fresh tag) and `spec-observability.md` 6.2.2, FR-OBS-104 and the series bound of 6.5.2 (+200, about 7 700). Both read snapshot fields that exist; `otelobserver` (Stage 2) builds them. Left open for later rows.

### N-127 Event classes at the observer

- **Areas:** OBS, EXP, CODE
- **Raised:** 2026-10-11, `spec-observability.md` 18 (open question 1) and 8.3.
- **Note:** The order and trade classes are 92% to 99% of the events of the first consumer's world and cost a boxing allocation each. `WithObserver(o, classes...)` would build and deliver only the classes an observer wants, and with no observer wanting a class the world would build none. It changes `architecture.md` 6.2 and the options of `spec-world.md` 4.1. Not adopted: no measure shows the placeholder 6% of a tick to matter (N-015). At the first slice, measure the allocation per event, then decide. A sweep with only snapshot measures registers no observer and builds no event (`spec-experiments.md` 3.4.4), so it is not the case that needs the option.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the welfare row adds no event; the one `Warn` is not an event class. Left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. No event here.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The row measures nothing; the baseline of NFR-OBS-002 is row 12's.

### N-128 A hosted per-agent field is saved once, in the agent record

- **Areas:** ALL, CHK
- **Raised:** 2026-10-11, `spec-checkpoint.md` 4.5 (F3).
- **Note:** A per-agent field that a module hosts (`Privation`, `RoleSince`, `DeathCarry`, `OwnUse`, `OwnUseJobs`, the journey, the directive) is saved once, in the agent record (tag 3), and never again in the module's own section; the module's section lists only its own carries and runs (FR-CHK-041). A spec that adds a per-agent field says so in its checkpoint section and adds the field to the agent record's list in `spec-checkpoint.md` 4.2.
- **Update (2026-10-13, STO, EMP, LND, CRD, CAP, ORG, INS):** where informative: STO's stores are saved once, in the agent record (tag 3); ORG's membership is one field of the agent record, written only with the module; EMP, LND, CRD, CAP and INS add no per-agent field (a worker's job is PRO's record; a title, a claim, a facility and a route policy are saved once, in their module's or the ledger's section).
- **Update (2026-10-15, plan 1-3 Phase 1):** not applicable to the clearing row: the report is per market and is saved in the Markets section (`spec-clearing.md` 16), not in an agent record. Left open.
- **Update (2026-10-16, plan 1-4 Phase 1):** nothing to add for the beliefs row: the centre and width are saved once, in their own Beliefs section (tag 5, per agent and commodity of its tradable set), and the window and `BoundRun` in the Markets section (tag 4), not in the agent record (`spec-beliefs.md` 13; `spec-checkpoint.md` 4.2). Left open.
- **Update (2026-10-16, plan 1-5 Phase 1):** applied for the production row: the job and the residues are saved once, in section 6 in agent order (`spec-checkpoint.md` 4.2 tag 6), not in the agent record.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the ring is the agent's hosted field (WEL 13) and not a field of the agent record, and the clamp run is a diagnostic run per (region, role): the rule of the note does not apply to either; `spec-checkpoint.md` 4.2 tag 7 now lists the two runs. The welfare row's `Records` read and `FromRecords` rebuild give CHK one place to write the ring. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied in its lesson: `spec-agents.md` 3.1/3.2 now say a module's per-agent state lives in the module's slices (N-244) and is still saved once, in the agent record (tag 3). Left open as a standing rule.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: not applicable to the estate row: the listing prices are per region in CHK section 8 (tag 8), not an agent field, and the estate has no agent record (FR-CHK-048). Left open as a standing rule.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Privation and RoleSince are the agent's hosted fields; the renewal carry and the three runs are the world's (D118, N-265).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. The table is configuration; the solver has no state (`spec-viability.md` 18).
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The world holds no per-agent field; the ring, the job and the residues are the agent row's hosted fields.

### N-129 A stateful extension is one instance per world

- **Areas:** EXP, INT, CHK
- **Raised:** 2026-10-11, `spec-checkpoint.md` 7.2 (item 6) and 11.
- **Note:** A `Stateful` extension (architecture.md 4.3) must not be shared between worlds: `LoadState` into a shared instance changes the others. A harness that restores many worlds from one checkpoint (branching) registers a new instance of each stateful extension for every branch (`spec-experiments.md` 5.5), and a consumer that builds two worlds does the same. The library cannot detect a shared instance.

### N-130 What IND cannot measure exactly until another spec adds a field

- **Areas:** OBS, IND, WEL, LED, INT, TRN, PRO
- **Raised:** 2026-10-11, `spec-indicators.md` 21 and 22.
- **Note:** All six are asks that were not adopted for version 1, so the measure is approximate or reports `OK` false until the field exists. (1) `IncomeLast`, `ReliefLast` and `CostLast` on `RoleRow` (functions of the welfare ring, which is saved state): without them `HouseholdIncome` is `OK` false and the claim of N-054 cannot be measured from a snapshot. (2) Wear units by commodity: the ledger counts them (FR-LED-016) and no event or snapshot carries them, so depreciation is on the retirement basis only; an aggregate event `WearApplied` is the ask. (3) A bulk read of household rows: `Agents(r)` then `Agent(a)` costs about 23 reads and an allocation per agent, and a one-column `{Weight, Value}` read would be cheaper. (4) Goods lines on `AgentRelocated`: without them the regional goods identity holds in total units only (`spec-regions.md` 6.4, the reconciling item). (5) Carrying units per line of `ShipmentDispatched`: freight is per unit of cargo by convention, exact only for a one-line shipment, since the binding fill of `spec-transport.md` 3.2.1 cannot be apportioned to a line. (6) Wear units on `RecipeExecuted`: value added by recipe overstates a tool-using recipe by the wear. Each is added with its owner's spec; none blocks.
- **Update (2026-10-13, Phase 1 of the ledger row):** for item (2): the first build of `internal/ledger` counts the wear units by commodity in the flow totals (FR-LED-016, FR-LED-042) and clears them at P8.6 with the totals; it reads them by an accessor and emits nothing, as `spec-ledger.md` 5.4.2 says. No event and no snapshot field carries them, so the ask stays with OBS and IND.
- **Update (2026-10-13, EMP, CRD, FIS, INS, STO, SET, SAV, PLC, RSK):** EMP: the wage bill and the unemployment of a window are measurable from `LabourRound`, the income fields of item 1 are still an ask; CRD: interest paid and received are measurable from `CreditSettled` and `LoanEnded`, a debtor's income share still needs item 1; FIS: the income terms are inside `income_t`, so ask 1 covers them; INS: the loss ratio (payouts over premiums) is exact from `CoverSettled` and `CoverFormed`, and a premium rebuilt from a snapshot needs `Policies`, so no field is asked beyond `N_t`; STO adds one ask (`spec-storage.md` 19.1); SET: no slot sets a class, `Class` and `ClassTarget` are the snapshot fields a reader would use; SAV: the preset `Idle` needs the four `RoleRow` fields only; PLC: the gauge `mean_ratio` reads role rows and so differs from IND's mean by at most 1 bp of rounding; RSK: the loss preset reports lost units, not a rate of lost lines (N-178).
- **Update (2026-10-14, observability review of the ledger branch):** item (2) is applied in OBS and LED: `spec-observability.md` 4.1.5 declares the aggregate event `WearApplied` (`Commodity`, `Units`; class A, core, P8.2 after `BoundaryFlow`, and `Tick` -1 for the construction bucket; FR-OBS-092), `spec-ledger.md` 12.1.1 owns it, and the metric is `economysim.commodity.wear` (`rules/observability-and-logging.md` 4.3, 5.3, 5.3.1). It has no `Region`, because the ledger's wear totals are per commodity (LED 7.4); IND's table row had asked for one. `event.go` declares it and `ledger.go` builds its rows (`wearApplied`); the **world** emits it at P8.2, so the Stage 1 world plan must call it after `boundaryFlows` and at the entry of the first step. **Still open here:** IND has not adopted the accrual form of depreciation (`spec-indicators.md` 7.6.2); until it does, `HeldByAgents` values a worn durable as new. Items (1), (3), (4) and (5) are unchanged. The same review added `Markets[].Lots` (FR-OBS-093) and the gauge `economysim.ledger.lots`, whose snapshot field waits for `snapshot.go`.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row and the ledger additions; left open. Item (6), wear units on `RecipeExecuted`, is an ask not adopted: no event field or snapshot field is added. Item (2) is unchanged: the create-and-dispose operation counts no wear. `spec-indicators.md` 7.2.4 gains only a pointer to `spec-production.md` 14.1 for the meaning of `Outputs` (gross quantity created) and `Disposed` (the part never held), so value added still reads `Outputs` gross of `Disposed`.
- **Update (2026-10-16, observability review of plan 1-5):** item (6) is still not adopted. `RecipeExecuted` gains `Overflow` (the part of `Disposed` that is not a discard output), which is not wear and changes no value: `Outputs` and `Disposed` keep their meaning, so value added still reads `Outputs` gross of `Disposed` (`spec-production.md` 14.1, FR-PRO-053). Left open.
- **Update (2026-10-17, plan 1-6 Phase 1):** read, both sides: `IncomeLast`, `ReliefLast` and `CostLast` are not adopted in version 1; the welfare summary returns the three sums as unexported values so the ask costs one snapshot field if adopted. OBS 5.3.4 gains `ClampedHouseholds` only and adopts none of the six asks. Item 2 (wear units) is the ledger's. Left open.
- **Update (2026-10-17, observability review of plan 1-6):** read: the three gauges added for the welfare row (`welfare.ratio.households`, `welfare.floor_held`, `wage.observed`) read `RatioHouseholds`, `HeldByFloor` and `WageObserved`, fields that already exist; none of the six asks is adopted and the income, relief and cost sums stay declined. `economysim.wage.observed` is the per-role observed wage, not the pooled regional wage the cost anchor reads. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read: the row emits `WearApplied` at P8.2 (item 2) and adopts none of the other asks. Left open.


### N-131 Not in version 1: reopen when a consumer asks

- **Areas:** INT, FIS, PLC
- **Raised:** 2026-10-11, `spec-integration.md` 3.6 and 19.
- **Note:** INT is minimal (ADR 0003 decision 2) and does not offer: an act-now operation (an action inside a step), a timer on a shock, a bundle of slots applied as one, goods injected into a region, a capacity multiplier of an outside line (a partial embargo), a public cancel of a migration journey, or the removal of a household of the configuration by the consumer. A consumer's controller that reads measures and sets slots is outside the engine (L121): the engine guarantees conservation and determinism, not its stability. FIS and PLC are where a rule that sets taxes or rates belongs.
- **Update (2026-10-13, SET, POL, CRD, FIS, SAV, TEC, PLC, ORG, RSK, INS):** not offered in version 1, by the later stage: SET a slot that sets a class; POL a transfer of a region between jurisdictions (N-161) and a law that forbids a move across a border; CRD a loan between named agents, early repayment, a change of a claim's terms and the cancel of one claim by ID (`spec-credit.md` 13.3); FIS a rate slot, a timer, a per-region scale, a tax on a reason, a treasury that hires or buys services, a public debt and a toll by value (`spec-fiscal.md` 24); SAV a call that sets a goal, a savings account and a call that makes a household build now; TEC setting experience directly, a duration or decay of an advance and a call that teaches a region a recipe it does not run (`spec-technology.md` 7.1.6); PLC a timer, a condition, a bundle of levers across actors, a policy that reads another jurisdiction's regions, a treasury that bids under a strategy, an actor that changes a cadence and a proportional built-in rule (`spec-policy.md` 17.1, 19); ORG a charter over carriage, a store or a facility, a wage control, a named holder and a call that admits or expels (N-194); RSK and INS a controller that sets risk scales, loadings or blocks from a measure, which is outside the engine (N-177). The rule that sets taxes and rates is `spec-policy.md` (a lever of kind `flow`).

### N-132 A population with no demand module and no faucet has no healthy start

- **Areas:** CFG, POP, STY
- **Raised:** 2026-10-11, `spec-stylised-facts.md` 5.9 (C1) and `spec-population.md` 9.2.
- **Note:** A pack with the population module on, no demand module and no faucet has no healthy start at the default schedules: the closed ceiling of the mean welfare ratio is 10 000 (N-054), below the replacement ratio 12 000. Acceptance reads the warning `replacement_above_closed_ceiling` as a rejection of that pack (`spec-population.md` 9.2). Whether the warning becomes a validation error is POP's and CFG's to decide.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for POP.

### N-133 No source read gives a distribution of wealth or a wage-and-population cycle

- **Areas:** EVD, STY
- **Raised:** 2026-10-11, `spec-stylised-facts.md` 5.2 and 5.4.
- **Note:** No source read so far gives a distribution of money or wealth, nor a period of the pre-industrial wage and population cycle. Read one, and record it in `specs/evidence.md`, before STY-D02 or STY-W02(c) is given a band.

### N-134 A sweep that moves the steady state needs a regenerated warm start

- **Areas:** VIA, EXP, CLB
- **Raised:** 2026-10-11, `spec-experiments.md` 3.3.3, 14.4 and 22 (open question 7), `spec-calibration.md` 4.
- **Note:** A sweep of a parameter that moves the steady state, with a warm start taken from the base, starts every point away from its own equilibrium (EXP-A5, L138). The hook is `Prepare`, which regenerates the table per point (`spec-experiments.md` 3.3.3), and the warning `stale_warm_start` is EXP's to specify (`spec-experiments.md` 14.4) from the list of paths the programme reads, which VIA 11 now publishes. CLB's flags `in`, `tbl` and `chk` on its registry rows follow that list, and its registry test fails a data field of the list without its flag (`spec-calibration.md` 4.2.1, FR-CLB-009). A point that changes an `in` or `tbl` knob regenerates the table or says it is a perturbation (7.3.11, FR-CLB-037). Open for VIA: whether `world.belief_step` moves the table (6.4 raises a seed price to the resolution floor, and the list says `no (a check)`); `spec-viability.md` 11 now says that a `.*` names every field of its table, as CLB reads it.
- **Update (2026-10-13, TEC):** no parameter of TEC moves the table at tick 0 (the gain is 0 by construction): the path `modules.technology.*` is `no` in VIA 11's list and needs no regenerated warm start (`spec-technology.md` 3.3.2, 8.7).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for EXP. The list of paths in VIA 11 is unchanged by this row; Phase 7 reads it against the inputs the code reads (a path the code reads that the list lacks is a defect of one of them). The open question of `world.belief_step` is answered in the code: only the resolution floor of a seed price reads it (VIA 6.4).

### N-135 The hold-out set is a recommendation, not a requirement

- **Areas:** REF, STY, CLB
- **Raised:** 2026-10-11, `spec-stylised-facts.md` 7.3.1 and 8.2, `spec-calibration.md` 7.1 and `spec-experiments.md` 12.4.2.
- **Note:** A calibration chooses on search blocks (seeds from 101, outside every REF set) and confirms on REF's seeds 1 to 50, which are also the seeds the bands were set from, so a stability claim made on them is `in sample`. `spec-reference-chain.md` 8.2 recommends seeds 51 to 60 as a hold-out, a SHOULD: no band and no calibration choice is set from them, and FR-REF-024 stands as written. A claim checked on them says `in_sample no` (`spec-calibration.md` 7.4.2, FR-CLB-100); a claim not checked on them says `in sample`. The hold-out costs 10 more nightly seeds per pack and lets a claim leave `in sample` without a rule change. Whether it becomes a MUST is REF's and STY's, with the run; until it is run every claim is `in sample`. Choosing the best of many points on a few seeds fits those seeds (L135), which is why the confirm step uses another `Seed0`.

### N-136 A store ties its owner to its region

- **Areas:** MIG, TRN, INT, MER
- **Raised:** 2026-10-13, `spec-storage.md` 9.
- **Note:** An owner of a store does not migrate, travel or take a forced move until it removes the store (`has_store`, `ErrRegionBound`). A merchant that needs storage and travel holds goods at its base room and hires carriage. Revisit with the first consumer's merchants: a portable store is a vehicle with a room (STO open question 10).

### N-137 Room is committed at the decision

- **Areas:** PRO, CLR, TRN, AGT
- **Raised:** 2026-10-13, `spec-storage.md` 5.
- **Note:** A job's start is refused `no_room` and a bid is refused `no_room`; the completion never fails. PRO 5.3.2 and the update of N-002 stand for `MaxQuantity`; the rest is STO's. Tests must assert the invariant of `spec-storage.md` 4.4 over seeds.
- **Update (2026-10-15, plan 1-3 Phase 1):** left open; `no_room` is STO's. Admission keeps its place in the table of `spec-clearing.md` 3.2 as a marked position with no code, and `deferredReasons` of `TestConstants_Values` keeps `no_room` listed. How a module row reaches the ordered function is written in `architecture.md` 2.10.1 (the admission row is a field of `clearing.Facts`).
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open for STO. The execution count takes the module's limit as a pair at the place of `no_room` in the master order (`spec-production.md` 4.2.1), and the completion never fails for room (FR-PRO-029): the part that has no room is created and disposed by the ledger's create-and-dispose operation.
- **Update (2026-10-16, plan 1-5 Phase 8):** applied for the production row: the completion never fails for room (FR-PRO-029, FR-PRO-030, tested): one `Transform` of what fits, then `Discard` for each discard output and for the part above `MaxQuantity`, the room read by `Holding`; the execution count takes the module's limit as a pair at the place of `no_room`, with no code until STO. Still open for STO: the `no_room` pair and the invariant of `spec-storage.md` 4.4 over seeds (the world row).
- **Update (2026-10-17, plan 1-6 Phase 1):** read: not touched; the room rules are STO's. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the room rules are STO's; the view has no room group in the slice. Left open.


### N-138 A role change moves excess goods to the estate

- **Areas:** SPN, SKL
- **Raised:** 2026-10-13, `spec-storage.md` 5.5.2.
- **Note:** The existing `estate_goods` rule of `spec-spawner.md` 4.2 gains the condition "or above the new role's room". Measure how often it acts with scenario STO-S1 (S91).
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the SPN 4.2 edit is the welfare-history clause only and leaves the `estate_goods` rule as written; the ring resize and the move of goods are independent steps of the same role change, in either order. Left open for STO.
- **Update (2026-10-17, spawner edits):** read; the role-change goods rule is as before. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Waits for STO; the estate_goods rule of SPN 4.2 is the executor's.

### N-139 The default strategy discards, for upkeep only, when the module is on

- **Areas:** AGT
- **Raised:** 2026-10-13, `spec-storage.md` 6.3.2.
- **Note:** `spec-agents.md` 8.2 said the default never discards. With the storage module on it discards, to make room for upkeep and for that only; an account with no base room is unlimited and never discards.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the default never discards in the slice (FR-AGT-051, first half); the storage case is STO's. Left open.

### N-140 Seams STO leaves

- **Areas:** RSK, CAP, SET, FIS, PLC, LND, ORG
- **Raised:** 2026-10-13, `spec-storage.md` 3.2.3, 3.6.3, 8.3, 10.
- **Note:** RSK reads the stored load and the stores by class (8.3). CAP calls `addStore` and `removeStore` (3.6.3), and the away test of `ErrAgentAway` is in `addStore`, so the completion of a store facility meets it and keeps the facility until the arrival. SET gates a store class by `StoreClass.Needs` and `ErrStoreLocked` (3.2.3, 10.3). A buffer store over a treasury's goods (a pack's word for it is flagged in STO's header) is FIS's (`spec-fiscal.md` 9.6.1; N-186): STO may accept a treasury as a holder later (FIS open question 8), and the release is PLC's lever or FIS's standing order. A keeper and a lease are a contract over the same room (LND, ORG; 10.4).

### N-142 A job start and a dispatch leave the upkeep room free

- **Areas:** PRO, TRN, AGT
- **Raised:** 2026-10-13, `spec-storage.md` 3.4.5, 5.3.2, 5.4.1, STO-A3 (L145), STO-D6 (D103).
- **Note:** The pledge is the one thing nothing can discard, so the room in which a household buys its food is never committed by a start or a dispatch. It is a rule constant, not a tunable; the warnings `room_below_recipe` and `room_below_warm_job` carry it.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open for STO. The room limit is a pair the module appends to the count.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: not touched. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: not touched; the upkeep room is STO's. Left open.


### N-143 No store is added to an agent away or on a journey

- **Areas:** INT, TRN, MIG
- **Raised:** 2026-10-13, `spec-storage.md` 3.6.2.
- **Note:** `ErrAgentAway`: a store is not added to an agent that is away or on a journey, so no arrival is ever held for ever by `has_store` (N-090).

### N-144 A busy employer cannot post vacancies

- **Areas:** AGT, WLD, EMP
- **Raised:** 2026-10-13, `spec-employment.md` 4.1.3 and open question 4.
- **Note:** `Work` is called only for a free agent, so an employer with a job in progress cannot post a vacancy. Posting from `Trade`, or calling `Work` for its `Hire` only, would change WLD 6.4 and AGT 6.2.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: `JobRequest` has no `Offer` or `Hire` in the slice. Left open for EMP.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open (EMP, Stage 3). `Work` is called only for a free agent at P2.5, as written.

### N-145 Profit with no outlet limits hired employment

- **Areas:** DEM, STY, REF, SAV
- **Raised:** 2026-10-13, `spec-employment.md` 11.2 and `models/emp/circuit.py`.
- **Note:** With midpoint wages and employers who spend only their upkeep, hired households settle at `E u_e / ((1 - phi) a)`. REF's overlay with an employer role needs wants on the employer side; STY may state a report fact; SAV is the investment outlet and its default builds nothing for a role with no block.

### N-146 A steady state with employers is not a zero-profit state

- **Areas:** VIA, REF
- **Raised:** 2026-10-13, `spec-employment.md` 22 item 7.
- **Note:** `spec-viability.md` solves for prices at cost with labour at the wage dual; with a margin shared by `phi` the employer keeps `1 - phi` of it. A warm start has no hire in flight. A warm start has also no claims (N-165) and no rent column (N-158): three gaps, not one topic.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for CRD, EMP and LND. Learned: the table type carries no claim, hire or rent column, and this row adds none (D5).

### N-147 Pay in kind has no mechanism

- **Areas:** WEL, EMP
- **Raised:** 2026-10-13, `spec-employment.md` 22 item 1.
- **Note:** `spec-welfare-and-wages.md` 6 defines `InKind` and a mechanism for the trainee's board only. The rule that would work for a wage is in `spec-employment.md` 22 item 1; the reason `wage_in_kind` is reserved.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: `InKind` is validated and unused; no mover and no position, so pay in kind is not built. Left open.

### N-148 The employer role's observed wage includes its profit on hands

- **Areas:** SPN, WEL
- **Raised:** 2026-10-13, `spec-employment.md` (EMP-A6, merged into L36) and open question 10.
- **Note:** The spawner reads the employer role's observed wage as the occupation signal, and it includes the margin on hired hands; a rule that left the hired margin out is EMP open question 10.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: left open for EMP and SPN (Stage 3). Learned: the per-role wage function takes a selector of the roles without an employer block as a parameter (WEL 4.3), so the module appends the rule without changing the signature.
- **Update (2026-10-17, spawner edits):** read; not settled by them. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Waits for EMP; the default reads RoleView.Wage.

### N-149 Hired labour is unskilled by construction

- **Areas:** SKL
- **Raised:** 2026-10-13, `spec-employment.md` 3.3.1 and open question 9.
- **Note:** One `HireYieldBP` per employer block, not a function of the worker's skill.

### N-150 A hire is within one region

- **Areas:** REG, MIG
- **Raised:** 2026-10-13, `spec-employment.md` 5.3.2 and open question 12.
- **Note:** The lane of the labour round is the region; a commuter is not a worker, and a worker's job is abandoned when its account changes region.

### N-151 Wages do not bid up: unemployment can persist at a wage

- **Areas:** STB, STY
- **Raised:** 2026-10-13, `spec-employment.md` open question 5, EMP-D1 (D104).
- **Note:** EMP-D1 refuses a bid that rises with unfilled vacancies in version 1; a stability condition on the gain of such a rule would be needed first.

### N-152 Standing employment is not in version 1

- **Areas:** CRD, ORG
- **Raised:** 2026-10-13, `spec-employment.md` open question 3.
- **Note:** A hire is one job. A service contract with a term and a bond would reuse its record.

### N-153 Gates by settlement class: `Role.Needs` and `Role.Ceiling`

- **Areas:** SPN, POP, AGT
- **Raised:** 2026-10-13, `spec-settlements.md` 6.3, 6.4.
- **Note:** SPN and POP must implement them: a role with `Needs` is closed in a region whose class lacks the flag, as `Closed` is (no entrant, no switcher, no like-for-like replacement, no birth, no advance of the birth carry; households stay). A role with `Ceiling` has a headroom (the limit less its households); the default spawner admits only what fits it and counts a refusal (`EntryLimited.RoleCeiling`), births are capped and counted (`PopulationChanged.BirthsCeiling`), and the spawner's view carries the role's openness and headroom, never the class. A gate binds the default rules only, as `Closed` does; a world-enforced gate is SET open question 2. ORG's places are a second limit into the same headroom (N-192).
- **Update (2026-10-17, spawner edits):** read; not settled by them. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: no view carries a settlement class (`spec-agents.md` 6.4.1). Left open for SPN and POP.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Waits for SET. The view's Closed is Role.Closed in the slice; headroom and NoLimit are appended with SET (D112).

### N-155 Reserved hooks of the settlement class

- **Areas:** STO, ORG, RSK, CAP, TRN
- **Raised:** 2026-10-13, `spec-settlements.md` 6.5, 6.2 (R1 to R7).
- **Note:** A later spec that wants a size reads the class by a flag or a limit named in its own configuration, at a position before P7.2a, with `ErrModuleDependency` when the module is off, its own refusal counter and its own register entry, and adds its row to 6.3. STO's `StoreClass.Needs` and ORG's places do. TRN may want `Mode.Needs` (a terminal). A market per commodity and a recipe condition are not offered, with reasons (6.6).

### N-156 A title or a facility bought for a price needs an observable return and a seller

- **Areas:** SAV, CRD, AGT, LND, CAP, INF
- **Raised:** 2026-10-13, `spec-land.md` 6.2 and `spec-saving-and-investment.md` 6.3, 19 (one decision, two sources).
- **Note:** The decision to acquire a title is SAV's, with CRD's credit; it calls `TransferTitles` and needs a field in the plan a `Strategy` returns. It cannot be made until a holder's receipt is something a household can see: today a holder's receipts depend on others' draws and the attribution of `spec-land.md` 4.4.2, and the view carries the rent in force and no draws, and there is no seller (the pool holds and lets). The shape, if a consumer asks, is a plan field `TitleBids` matched in a round (SAV 6.3, open question 2). A facility bought for a price has the same two gaps (CAP 5.3.4: a ground facility is valued only by a `Worth` row).
- **Update (2026-10-17, plan 1-7 Phase 1):** read: `TradePlan` has `Intents` and `Discards` only and no `TitleBids`; the plan type is built so that a field is appended (architecture.md 2.10.5). Left open.

### N-157 MIG's estimate reads no rent and no title

- **Areas:** MIG, INF
- **Raised:** 2026-10-13, `spec-land.md` 7.2.2.
- **Note:** MIG's estimate of a destination's income reads no rent and no title; a rumoured rent would be a second kind of rumour (consistent with N-110 and the update of N-108 for MIG).

### N-158 The steady-state programme has one row per endowment; the warm start has no rent column

- **Areas:** VIA, CLB
- **Raised:** 2026-10-13, `spec-land.md` 12.3.
- **Note:** The split of an endowment's row among roles and the money circuit of a rent are not in the programme. Part of three warm-start gaps (N-146, N-165).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open (see N-146).

### N-159 Region-attributed treasury and fiscal flows need a reconciling item in the regional money identity

- **Areas:** FIS, IND, LED, OBS
- **Raised:** 2026-10-13, `spec-polity.md` 5.3.4 (POL-A2, L162) and `spec-fiscal.md` 4.3.2, 16.1.3.
- **Note:** A transfer from the households of region A to a treasury seated in region B is a transfer between regions that no item of the identity of `spec-indicators.md` 8.2 named. Settled as: POL's treasury sector is reported in `Sectors` (IND 9.1, with `M_t`), and FIS's flows are reported beside it as `FiscalIn_r` and `FiscalOut_r` from `FiscalFlowApplied`, with the seat from the snapshot's `Jurisdictions` rows (IND 8.2). The alternative, the region `NoRegion` for a treasury, is `spec-polity.md` 17 question 5. A new reader of the identity (the regional report of IND, OBS's rebuild) must carry both.
- **Update (2026-10-13, Phase 1 of the ledger row):** what the ledger does here, so that the readers do not assume otherwise: a flow between two non-boundary accounts is attributed to the region of the debited account (`spec-ledger.md` 3.5, FR-LED-053), so a tax paid by a household of A to a treasury seated in B is in A's flow totals and a payout from that treasury to a household of A is in B's. The ledger's audit takes the treasury term of the sectoral identity by account kind (`spec-ledger.md` 9.3; `plans/plan-phase-1-1.md` Phase 6, zero until a module opens a Treasury account). Nothing in the ledger carries the reconciling item; it stays IND's and OBS's.
- **Update (2026-10-16, plan 1-5 Phase 2):** read for the ledger additions; left open (not this row's). No treasury is involved. `Discard`'s two flows are attributed to the account's region like every other goods flow (FR-LED-053, `spec-ledger.md` 8.2.3), so a reader of the regional identity needs no extra term for them.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the new snapshot field is a count of households and not a money flow, so it takes no part in the regional money identity. Left open for FIS and IND.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open (FIS, IND). The world reports `Pending` and `Construction` as the ledger gives them.

### N-160 Readers keyed by jurisdiction use one combining rule

- **Areas:** FIS, ORG, PLC, LND, STO
- **Raised:** 2026-10-13, `spec-polity.md` 6.1, 6.2, 6.3.
- **Note:** A table keyed by (jurisdiction, subject) has one row per key, absence is the default, canonical order, resolved once into a dense array by `RegionID`, and one of three modes (innermost, added, scaled with one rounding) over `Chain(region)`; a crossing is applied in the order `Exit` then `Enter`. A reader never reads a count of households or a money supply of a jurisdiction to change a rate of decision (`spec-world.md` 9.4.1); it adds its row to the table of hooks in 6.3.

### N-161 A transfer of a region between jurisdictions is reserved

- **Areas:** INT, RSK, FIS, ORG
- **Raised:** 2026-10-13, `spec-polity.md` 7.4 and question 1.
- **Note:** Its shape is fixed (a plural setter, at the call, a sparse saved list, the event `RegionTransferred`, a seat cannot leave); every reader that caches by jurisdiction must say what a transfer does to its state before it is offered.

### N-162 A paired comparison with a module on, off or binding is not aligned

- **Areas:** EXP, CLB, STY, STB
- **Raised:** 2026-10-13, `spec-polity.md` 5.2.3, 13.2 (POL-A1), `spec-fiscal.md` 22.1, `spec-institutions.md` 14 (ORG-A5) and `spec-calibration.md` 7.2.3.
- **Note:** A world with a governed jurisdiction and one without are not a paired comparison from the first clearing: the household IDs are shifted (an ungoverned polity is exactly neutral and may be added to any pack for measurement, FR-POL-019, S112). A fiscal module off, or with every flow at scale 0, is exactly neutral (S132), but a standing order adds orders and so tie-break keys. A binding right removes an order, so the counts of D6 to D8 move; a clamp changes no count, and an inert overlay is aligned (S141). Use a comparison over seeds (N-124).
- **Update (2026-10-12, ADR 0008):** the draws are aligned whatever a module adds or removes (a stream per (site, tick, entity), N-214); what still breaks a pair is the household ID shift of a seat (POL-A1), which keys every per-agent stream by a different ID. A pair across a governed and an ungoverned polity stays a comparison over seeds; a pair across a fiscal flow, a right or a clamp is aligned in the draws and diverges through state only.

### N-163 The word "treasury"

- **Areas:** MON, FIS, ALL
- **Raised:** 2026-10-13, `spec-polity.md` 5.
- **Note:** Since POL, "treasury" is the account kind of a governed jurisdiction (and the treasury sector), not the region's estate account, which is the pool. `spec-money-supply.md` 4.3.2 and the roadmap say "pool" for a grant that goes to the estate account (F14d), and "treasury" for a MON target of kind `treasury` (FIS 4.2). Use the two words in these senses only.
- **Update (2026-10-15, plan 1-3 Phase 1):** the clearing row's code and comments say "pool" for the region's estate account and "treasury" only for the account kind. Left open.
- **Update (2026-10-16, plan 1-4 Phase 1):** the beliefs row does not use the word "treasury". Left open.
- **Update (2026-10-16, plan 1-5 Phase 1):** the production row does not use the word "treasury". Left open.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the word "treasury" is not used by the welfare row; left open for its owners.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the agent row does not use the word "treasury". Left open for its owners.
- **Update (2026-10-18, plan 1-8 Phase 1):** applied for the estate row: code and comments say "pool" for the estate account and never "treasury" for it. Left open for its owners.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The row says "pool" only.
- **Update (2026-10-18, plan 1-10 Phase 1):** not this row's: the word "treasury" is not used here. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The row says "pool" for the estate account and never "treasury".

### N-164 Credit requests and offers are plan fields

- **Areas:** AGT, CRD, INT
- **Raised:** 2026-10-13, `spec-credit.md` 5, 10, 21.2.
- **Note:** The `TradePlan` of `spec-agents.md` 6.3 gains `Borrow []CreditRequest` and `Lend []CreditOffer`, the view a `Credit` group (with the field `CreditRoom`) and the `Directive` two optional fields; the default strategy appends them for a role with a block.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the credit fields of the plan and the view are appended with CRD; the plan and view types are built so that a field is appended. Left open.

### N-165 A warm start has no claims

- **Areas:** VIA, CFG, CRD
- **Raised:** 2026-10-13, `spec-credit.md` 11.4 and open question 14.
- **Note:** A pack that wants initial debt needs a configured table. One of three warm-start gaps (N-146, N-158).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open (see N-146).

### N-166 Collateral is not in version 1

- **Areas:** CRD, LND, STO
- **Raised:** 2026-10-13, `spec-credit.md` 3.5 and open question 3.
- **Note:** The candidates are goods in custody, a title of LND and a pledge of room of STO. A title is the cheapest: a `TransferTitles` at default and a `Security` field of the claim.

### N-167 The goods of a removed debtor do not pay its creditors

- **Areas:** EST, CRD
- **Raised:** 2026-10-13, `spec-credit.md` 9.1.2 and open question 8.
- **Note:** The estate lists them at the reference price and the proceeds are the pool's; only the money of the debtor goes through the waterfall.
- **Update (2026-10-18, plan 1-8 Phase 1):** applied for the slice: the intake lists every lot at the first price and the proceeds are the pool's (`spec-estates.md` 2.1, 2.7.3); a Phase 3 test shows a removal moves the goods to the estate with no payment to any claim. Left open for CRD's waterfall, which is money only.

### N-168 A lender with no other income eats its principal while it is out

- **Areas:** CRD, REF, STY
- **Raised:** 2026-10-13, `spec-credit.md` 10.3.1, 19.2 (cases E and F: 36 and 316 unfed lender-ticks).
- **Note:** A pack with lenders gives the role an income or a `KeepTicks` of at least its longest term (warnings `lender_without_income`, `lender_keep_below_term`).

### N-169 A shield trades food now for credit later, and its best value moves with the horizon

- **Areas:** CRD, STB, CLB
- **Raised:** 2026-10-13, `spec-credit.md` 8.2.2, 19.2.
- **Note:** In the model the unfed borrower-ticks at shields 0, 1, 2, 3, 5, 8 are 727, 723, 242, 238, 231, 272 at `SlackTicks` 0 and 165, 162, 212, 208, 200, 235 at `SlackTicks` 5: the arrears a larger shield keeps bar new credit. `ShieldTicks` is a damper strength with an interior optimum that depends on the horizon, to be found by CLB.

### N-170 A route speeds goods, not news or a household's move

- **Areas:** INF, MIG, TRN, CAP
- **Raised:** 2026-10-13, `spec-infrastructure.md` 5.2.3.
- **Note:** A route facility acts on the transit factor of a mode, so the lag of a rumour (a function of the distance and `InfoSpeed`), a migration journey (`MoveSpeed`) and a shortest path are unchanged. A setting in which a road also speeds news or the move wants INF or MIG to read the facility layer, or an edge status factor set by the consumer.

### N-171 A build is booked as a cost with no return

- **Areas:** WEL, MIG, CAP
- **Raised:** 2026-10-13, `spec-infrastructure.md` 4.4.2, CAP-A3 (L169).
- **Note:** The welfare ratio nets the cash spent on inputs and a facility is not an output, so the builder's own ratio falls for the length of its job (the aggregate ratio is unaffected: N-054). Relief, a role change and migration may act on it and abandon the job, and the inputs are lost. The derived window and the settling rule damp it; a role with a buffer of money is the answer. WEL decides whether capital formed is ever credited to the ratio.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: left open for CAP (Stage 3). The record has no capital-formation field (WEL 13 has none); WEL decides then whether capital formed is ever credited.

### N-172 An idle site after its owner's death

- **Areas:** SPN, ORG, FIS, CAP
- **Raised:** 2026-10-13, `spec-infrastructure.md` 7.1, CAP-A6 (L171).
- **Note:** A site passes to the estate and idles; a successor needs a call (`TransferFacilities`) or, later, a lease or a fee (CAP open question 2 sketches the contract form).
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Not this row's.

### N-173 Use of a facility by others is not in version 1

- **Areas:** ORG, FIS, MER, CAP
- **Raised:** 2026-10-13, `spec-infrastructure.md` 5.1.5 and open question 2.
- **Note:** A fee, a lease, a charter and an open-use ration are later specs' and need a hook at P2.5 and a ration like RES 5.3. ORG's reserved charter over a facility is the same gap (N-194); the two notes are kept apart on purpose, the facility side here and the right side there.

### N-174 Edge limit, traffic wear and toll

- **Areas:** TRN, FIS, CAP
- **Raised:** 2026-10-13, `spec-infrastructure.md` open question 4.
- **Note:** TRN has no edge limit; a bridge with a capacity, a wear by traffic and a toll each need a hook at dispatch. The toll part is FIS's (N-184, N-189); the three notes are kept apart on purpose.

### N-175 A route layer applies to a direct shortest edge only

- **Areas:** TRN, REG, CAP
- **Raised:** 2026-10-13, `spec-infrastructure.md` 5.2.5 and open question 12.
- **Note:** Transport keys its factor by the pair and takes REG's shortest distance; CAP's layer is that of the pair's own direct edge and only where it is the shortest path, so a road never speeds a multi-hop shipment. A path rule (composition over REG's `Path`) wants a run with a road network.

### N-176 A shipment's fate is fixed at dispatch and is the world's fact; insurance never reads it

- **Areas:** INS, MER, TRN, RSK
- **Raised:** 2026-10-13, `spec-risk.md` 4, 10.2.2 and `spec-insurance.md` 6.1.2.
- **Note:** `LossBP` is in the record and in `ShipmentDispatched` for an observer, and no view carries it. Insurance reads `RouteLoss` and the arrival's lost units and never `LossBP`, a draw or a scale; it would insure only what is lost. FR-INS-025 is the test: two worlds that differ only in a lane's strikes form the same policies.

### N-177 No endogenous risk in version 1

- **Areas:** PLC, INT, STY
- **Raised:** 2026-10-13, `spec-risk.md` 16 (RSK-A6, L173) and 9.4.
- **Note:** A strike probability does not depend on traffic, value or price, so "loss raises price raises risk-taking" is balancing and bounded. The reinforcing version is a consumer's controller that sets a scale from a measure; the engine guarantees conservation and determinism of every setting, not its stability (N-131, L121).

### N-178 A small line's loss is a step function

- **Areas:** IND, STY
- **Raised:** 2026-10-13, `spec-risk.md` 5.3.2, 17 part 5.
- **Note:** Half to even makes a one-unit line lose it only above a share of 5 000; a measure over few small lines is noisy and biased, over lines of many sizes it is not. The loss preset IND is asked for reports lost units, not a rate of lost lines.

### N-179 A common risk row is the correlation an insurer cannot diversify

- **Areas:** INS, STB, EXP
- **Raised:** 2026-10-13, `spec-risk.md` 3.3, 10.2.3, RSK-A3.
- **Note:** A sweep through a scale of 0 is a different stream from there (the row takes no word at 0); a sweep above 0 stays aligned (`spec-risk.md` 11.4).

### N-180 The expected loss is in the shipper's margin and in the lanes

- **Areas:** VIA, TRN, MER
- **Raised:** 2026-10-13, `spec-risk.md` 8.
- **Note:** `transport.Lanes` has a loss that is no longer 0 once a pack has route rows; VIA reads it as the share of a lane's output that never arrives.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for TRN and RSK. `Lane` and its `LossBP` are not published in Stage 1 (D127); the programme reads a loss only when TRN adds them.

### N-181 Not exposed in version 1: a household's base room, a migration journey, money

- **Areas:** STO, MIG, MON
- **Raised:** 2026-10-13, `spec-risk.md` 1, 7.4 and open questions 6 to 8.
- **Note:** Each needs a decision the module does not have: an owner of the loss's gain, a journey's `LossBP`, a counterpart of money.

### N-182 Region-keyed tables of a module that sits across two others

- **Areas:** CFG
- **Raised:** 2026-10-13, `spec-risk.md` 3.2.3.
- **Note:** A route row names a mode (TRN) and an edge (REG), a store row a class (STO): validation must run after the owners' tables exist and report `ErrModuleDependency` when an owner is off.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row. Left open.

### N-183 A tax is netted from income; a MON levy is not

- **Areas:** WEL, MON, FIS
- **Raised:** 2026-10-13, `spec-fiscal.md` 12.2.
- **Note:** `income_t` gains `+P_t - T_t` for a payout and for a tax, duty or toll to a treasury, stored in the record's `G_t`, so the observed wage is gross; `spec-money-supply.md` 7.2 leaves a levy (a sink) un-netted. WEL assigns the letters and the rule (the plan's ruling F7): one rule per flow, the reason written once.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: left open for FIS (Stage 3). WEL 2.1.9 already assigns the letters, and `unearned` is the one signed field, so no field is added when FIS lands.

### N-184 Transport's "never moves money" is excepted by a toll

- **Areas:** TRN, FIS
- **Raised:** 2026-10-13, `spec-fiscal.md` 7.4.
- **Note:** `spec-transport.md` 3.3 and position 3 are edited; with the module off the position is true as it was. A toll is a fiscal flow paid at the dispatch (P3.2(c)); see N-174 and N-189.

### N-185 One order per agent, commodity, named lot and side

- **Areas:** FIS
- **Raised:** 2026-10-13, `spec-fiscal.md` 9.4.1.
- **Note:** The rule of `spec-clearing.md` 3.1.1 becomes one order per agent, commodity, named lot and side. A treasury may place a bid and an ask for one commodity with `limit(Purchase) < limit(Release)` on the integer limit prices (9.2, not on the basis points), so they never cross and a treasury never matches itself.
- **Update (2026-10-15, plan 1-3 Phase 1):** applied for CLR, which leaves the areas: the duplicate key of the book is (agent, commodity, named lot, side) (FR-CLR-004), and the one-bid-one-ask treasury case is tested with the two sides of one agent. FIS keeps the validation (`ErrFiscalShape`).

### N-186 A treasury's goods have no room limit

- **Areas:** STO, FIS
- **Raised:** 2026-10-13, `spec-fiscal.md` 9.6.1.
- **Note:** `spec-storage.md` 3.3.3 lists a treasury among the pseudo-agents that are never limited; H9 of `spec-polity.md` 6.3 (a jurisdiction as a limited holder) stays reserved.

### N-187 A tax on a reason (rent, a wage, interest) needs a hook at the transfer

- **Areas:** LND, EMP, CRD, FIS
- **Raised:** 2026-10-13, `spec-fiscal.md` 24 item 3.
- **Note:** In version 1 a tax on a reason is a tax on the holder role; a hook at the transfer would let a flow tax one ledger reason whoever pays it.

### N-188 Incidence is statutory; a duty is not in any limit price

- **Areas:** AGT, FIS
- **Raised:** 2026-10-13, `spec-fiscal.md` 6.6 and 24 item 4.
- **Note:** A view field would let the default strategy price a duty in; it is not adopted.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: no view field for a duty. Left open.

### N-189 A merchant's margin subtracts the toll

- **Areas:** MER, TRN, FIS
- **Raised:** 2026-10-13, `spec-fiscal.md` 7.5.
- **Note:** `Routes.Toll` (money per carrying unit on the shortest path at the tick) is read by MER's margin; see N-174.

### N-190 A legal price is a clamp of one side's limit

- **Areas:** LND, EMP, FIS, PLC, MER
- **Raised:** 2026-10-13, `spec-institutions.md` 6.3, 6.6 to 6.8.
- **Note:** In the admission of a market (and of the contract and credit rounds) a bid is capped and an ask floored; any later spec that forms a price from two limits (a rent, a wage, a fee, a tariff) applies the same clamp at its own admission. Never a rejection and never a rule of the strategy.

### N-191 A right is read at a start, an order and a place

- **Areas:** PRO, EMP, LND, STO, CAP
- **Raised:** 2026-10-13, `spec-institutions.md` 5.5.1.
- **Note:** Never at a continuation, a completion, a holding, a consumption or a shipment; a job in progress runs to its end.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open for ORG. `not_permitted` is a pair appended after `not_on_role` and before `zero_yield` (`spec-production.md` 4.2.1, position 2); a continuation, a completion and a holding read no right.


### N-192 A body's places are a second limit into the headroom

- **Areas:** SET, SPN, POP
- **Raised:** 2026-10-13, `spec-institutions.md` 3.4.3 (cross-reference N-153).
- **Note:** The headroom of a role is the least of its class limit's (`spec-settlements.md` 6.4.2) and its body's; the spawner's view and POP's births read it.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Not this row's.

### N-193 A completion of a traineeship is held without headroom

- **Areas:** SKL, SPN
- **Raised:** 2026-10-13, `spec-institutions.md` 4.5.2.
- **Note:** A trainee stays a trainee, and holds its trainer's place, until the target role has headroom of 1; counted `Held`.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Waits for SKL; no traineeship exists in the slice (D111).

### N-194 Reserved, not offered: a charter over carriage, a store or a facility; a wage control; a named holder; a call that admits or expels

- **Areas:** TRN, STO, CAP, EMP, PLC, INT
- **Raised:** 2026-10-13, `spec-institutions.md` 5.8, 6.8, 7.1.3.
- **Note:** A charter over carriage (`ship`), over a store or over a facility, a wage control, a holder that is a named household and a call that admits or expels a member are reserved with their shapes. The facility clause overlaps N-173, kept apart on purpose.

### N-195 Instalment claims for a credit-financed project

- **Areas:** SAV, CRD
- **Raised:** 2026-10-13, `spec-saving-and-investment.md` 6.2.2, 19 (1).
- **Note:** A bullet loan that falls due at the end of a build is the debt squeeze of CRD-A1; the default asks for no loan for a project; a claim with a schedule is the safe form.

### N-196 A `Facilities` group in the `Trade` view

- **Areas:** AGT, CAP
- **Raised:** 2026-10-13, `spec-saving-and-investment.md` (the plan and the view) and `spec-infrastructure.md` 9.1.
- **Note:** CAP 9.1 leaves the `Trade` view without facilities; SAV's goal and planned recipe are decided in `Trade`, and the clauses go in the per-recipe group.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: no `Facilities` group in the slice view. Left open for CAP and SAV.

### N-197 FR-CAP-018 and CAP 4.5 carry a carve-out for a role with a `Saver` block

- **Areas:** CAP, SAV
- **Raised:** 2026-10-13, `spec-saving-and-investment.md` (the edits owed to `spec-infrastructure.md`).
- **Note:** The default builds and repairs nothing for a role with no block; with a block it is SAV's plan that starts a build. Redundant if the edit owed to CAP is applied.

### N-198 The drawdown is `O(T)` per agent and tick

- **Areas:** SCL, CODE
- **Raised:** 2026-10-13, `spec-saving-and-investment.md` 4.1.4.
- **Note:** Measure at the first slice; an incremental form needs a monotone queue saved as state.
- **Update (2026-10-18, plan 1-10 Phase 1):** not this row's: not a solver concern. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. Phase 10 records the allocation count of a site call; nothing is measured in time.

### N-199 `HurdleSpreadBP` rests on one invented model

- **Areas:** CLB, STY
- **Raised:** 2026-10-13, `spec-saving-and-investment.md` (the default of `HurdleSpreadBP`) and `models/sav/invest.py`.
- **Note:** A spread of 5 000 does nothing and 40 000 staggers entry in the model; a calibration should find the real range.

### N-200 A gain is not a recipe

- **Areas:** PRO, CAP
- **Raised:** 2026-10-13, `spec-technology.md` 1, 8.2, 19 and open question 3.
- **Note:** An unlock (a recipe or a facility class that appears at a level) needs a refusal reason in PRO's order and a clause on the recipe or the class naming a field and a gain; not in version 1.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open. No unlock clause or refusal reason is added; the technology factor is a parameter of the effective factor, 10 000 in the slice.


### N-201 R9's volume clause is on units

- **Areas:** STY, IND
- **Raised:** 2026-10-13, `spec-technology.md` 8.8, 13.6.
- **Note:** Growth lowers the price index and raises the units traded, so R9 (deflation with distress) stays silent only while its second conjunct is a clause on units (`Turnover.Volume`, `SumV`), not on value. STY should keep the clause and run TEC-S2 (S149) with R9 and F4.
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open. `Outputs` and `Disposed` of `RecipeExecuted` are units of quantity; the sentence added to `spec-indicators.md` 7.2.4 changes no clause on value or volume.


### N-202 A module that keeps a key re-derives it at a reseed

- **Areas:** CHK
- **Raised:** 2026-10-13, `spec-technology.md` 4.4.6.
- **Note:** TEC's rule is `Key' = finalise(Key xor finalise(salt))`; a later module with a keyed hash does the same.
- **Answer (2026-10-12, ADR 0008):** generalised to the seed itself: a reseed sets `seed' = finalise(seed xor finalise(salt))` and every keyed stream and module key derives from the effective seed, so no module re-derives anything of its own. TEC 4.4.1 and 4.4.6 and CHK 6.8 are rewritten under N-214; close then.
- **Update (2026-10-12, Phase 1):** CHK 6.8.2 now states `seed' = finalise(seed xor finalise(salt))` and that no module re-derives a key at a reseed (the technology key derives from the effective seed). Close when TEC 4.4.1 and 4.4.6 are rewritten.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the welfare record keeps no key and draws nothing at a reseed (the target is a creation word); the stale-marked passages of CHK are not edited here. Left open.

### N-203 A stability claim is made with the module off

- **Areas:** STY, EXP
- **Raised:** 2026-10-13, `spec-technology.md` 8.7.
- **Note:** STY 7.3's "a warm start that stays near the steady state" assumes a steady state that does not drift; a run with growth on is held to the growth scenarios TEC-S2 to S5 (S149 to S152).

### N-204 A slot has two writers: the consumer and a policy actor

- **Areas:** WLD, INT, REG, TRN, MON, FIS, CHK
- **Raised:** 2026-10-13, `spec-policy.md` 6.4, 7.1.
- **Note:** The owners' setters have an internal form that PLC calls inside the step at P0.3a, before any position that reads a slot, with the cause `policy`; the public call keeps `ErrInStep` for a caller inside a step. The last write wins; an actor writes only the keys it returns, and a `schedule` returns its value at every decision, so it re-asserts over a consumer's call. Accepted by ruling F8(b) (`spec-world.md` 8.1.1, 3.6.3, 8.2.2).
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open. The yield-shock slot is a story shock and PLC offers no lever for it (`spec-policy.md` 6.3.3), so it has one writer: the row's setter is unexported and validates [0, `MaxYieldBP`]; the public call is INT's and the sparse shock list is written in tag 6.
- **Update (2026-10-17, config/world/scale edits):** read; the two-writer slot rule is unchanged. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The slots are Stage 2 and P0.3a is a module position; `beginStep` leaves the place.

### N-205 A new slot is a candidate lever kind; the kinds `rate`, `rent` and `legal` are reserved

- **Areas:** INT, ORG, CRD, LND, CAP, RSK, STO
- **Raised:** 2026-10-13, `spec-policy.md` 6.3, 6.3.2, 6.3.3.
- **Note:** A later spec that adds a rate or a scale an authority would hold adds a row to PLC 6.3 (kind, key, reach rule, bound, default, read position) in the change that adds the slot to INT; a shock of the story is not a lever. `SetRateScales` (CRD), `SetRentScales` (LND) and `SetLegalLevels` (ORG) are integrated, and PLC keeps the three kinds reserved. When a kind goes live, its event (`CreditRateScaled`, `RentScaled`, `LegalLevelSet`) needs a `Cause` field like `MoneyFlowScaled`.

### N-206 The engine cannot call an indicator

- **Areas:** IND, STY, EXP
- **Raised:** 2026-10-13, `spec-policy.md` 17.3 item 1.
- **Note:** IND 3.1 and FR-IND-001 hold: the built-in `rule` reads four gauges that are field reads, and a consumer's `Policy` imports `indicator/`. Whether to add a `policy/` package that imports it (amending `architecture.md` 2.7 and FR-IND-001) is PLC open question 1.

### N-207 A policy actor is not an agent and shifts no ID

- **Areas:** EXP, CLB, STY
- **Raised:** 2026-10-13, `spec-policy.md` 3.2.2.
- **Note:** An `ActorID` is its index among actors; adding an actor renumbers the actors only, so no tie-break or rotation of the agents changes. A consumer's policy that draws does shift the stream (PLC-A7, a case of L120).

### N-208 The gauge set and the IND presets must agree

- **Areas:** IND, CLB
- **Raised:** 2026-10-13, `spec-policy.md` 8.3.2, 17.2.
- **Note:** Each gauge is stated with the preset it matches; IND is asked for a region-set selector and for `PriceLevelOf`, so that a consumer's policy and a built-in rule are the same function. The `mean_ratio` gauge differs from IND's mean by at most 1 bp of rounding.

### N-209 A payout claim has an origin

- **Areas:** CRD
- **Raised:** 2026-10-13, `spec-insurance.md` 9.2.
- **Note:** The credit list holds claims of two origins, `loan` and `payout`. The credit instruments and `Warn`s, `ForgiveClaims` and `CreditSettled` count loans only; the payments of a payout claim are INS's, in `CoverSettled` (`Stage` `claim`, fields `ClaimEnded`, `ClaimPaid`, `ClaimReceived`); the welfare record counts the principal of a payout claim as income and cost, and the identity of CRD 3.6 reads `Lent + Awarded`. INS counts its own outstanding payout claims in `economysim.insurance.claims`.
- **Update (2026-10-13, Phase 1 of the ledger row):** applied for LED: the ledger's section 10 and FR-LED-063 carry the origin, `Awarded` and `Obligate` (`spec-ledger.md` 10, FR-LED-064); the first build keeps both origins in one list and checks the principal identity with `Awarded`. LED removed from the areas.

### N-210 The premium is not in the merchant's margin

- **Areas:** MER, STB
- **Raised:** 2026-10-13, `spec-insurance.md` 10.2.2, INS-A5 (a case of L19).
- **Note:** The margin of MER 5 is net of the expected loss and has no cover term, so a lot that is profitable uninsured can be negative insured. A consumer's strategy may net the recovery (INS-A1's third case: 0.624 against 0.592 and 0.790).

### N-211 Adverse selection and moral hazard need what RSK excludes

- **Areas:** STB, RSK, EXP
- **Raised:** 2026-10-13, `spec-insurance.md` 17 (head).
- **Note:** With no private information about a strike and an exogenous probability, neither exists in its full form; the quasi forms are INS-A1, A2 and A4. Experience rating (INS open question 3) is what closes the reinforcing chain and it needs a rate from an aggregate.

### N-212 A premium from borrowed money

- **Areas:** CRD
- **Raised:** 2026-10-13, `spec-insurance.md` 7.1.2, INS-A6 (a case of L11 and L45).
- **Note:** The cover round is after the credit round, so a household can pay a premium from this tick's loan.

### N-213 A store has no expected-loss read

- **Areas:** RSK, INS
- **Raised:** 2026-10-13, `spec-insurance.md` 6.4.2 and open question 4.
- **Note:** A store premium is the underwriter's posted rate, a pack's data; a read like `RouteLoss` for a store is the ask.

### N-214 Keyed random streams (ADR 0008): the edits owed

- **Areas:** WLD, CHK, EXP, CLB, STB, TEC, PLC, AGT, EXT, INF, MER, MIG, RSK, WEL, CODE
- **Raised:** 2026-10-12, `adr/0008-random-draws-are-keyed-streams.md`; `rules/architecture.md` 7.3 is already changed.
- **Note:** Randomness is a function of (seed, site, tick, entity): the one PCG is reseeded by a SplitMix64 key at every draw site, no draw depends on how many others were taken, a checkpoint holds the seed and no stream state, and `WithReseed(salt)` gives `finalise(seed xor finalise(salt))`. The specs still describe one positional stream; apply the ADR in the next change that touches each:
  - WLD 7 (one seed; the keying rule with the exact constants and the test vectors, `models/wld/seed.py` regenerated; the draw table 7.3 gains a `site` constant and an `id` per row; 7.4.2 and 7.4.3 narrow to one stream; 7.4.4 reads the seed, not the PCG state; 7.6 is no longer a hazard between runs), 12 (no PCG state), 13 (the CHK row), 16.5, 17 (WLD-A5), 22 (questions 4 and 7: answered; P4.1 may clear markets concurrently once architecture.md 12.2 has an owner), 23 (`random.go` holds the seed, the PCG, the `Rand` and the site table).
  - CHK 4.2 tag 1 (the seed is in the header; nothing else), 6.2 step 8 (no install), 6.8 (the reseed rule above), 8.4 (no positional state), FR-CHK-051 (a restore takes no word: still true, trivially), the worked example of 17 and the size table.
  - EXP 7.3 and FR-EXP-062 (the stream fingerprint has nothing to read: runs of one seed are aligned by construction and diverge through state only; `Pair` keeps measuring the gain); CLB 4.2.1, 7.2.3 and FR-CLB-024 (the flag `d` goes).
  - STB L120 (WLD-A5 becomes "a count change moves the later draws of its own stream only"), L124, D88 (draw anyway is an owner's choice inside a stream, not an alignment damper); the case PLC-A7.
  - TEC 4.4.1 and 4.4.6 (the technology key is derived from the effective seed; D16 takes no construction word; N-202 is answered by the general rule).
  - The owner of each draw row adds its `id`: AGT (D3, D5, D10, D12: the agent), BEL (D8: the agent and the order), CLR (D7: the market; whether the per-order key becomes a stateless `K` is ADR 0008 open point 2), EXT (D2: the line), INF (D4: the agent), MER (D6: the offer or take), MIG (D11: the agent), RSK (D13, D14, D15: the row, the shipment, the agent and store), WEL (D1: the agent), PLC (D17: the actor).
  - CODE: `random.go` and `internal/ordering` (the keying helper beside the finaliser); the allocation test of NFR-WLD-001 covers the reseed.
- **Update (2026-10-12, Phase 1 of the Stage 0 plan):** applied now: WLD 7 (7.1.2 to 7.1.7, 7.2 to 7.6), 4.1, 4.4, 5.1, 5.7, 6.9.1, 6.12 C22, 8.1.1, 10.1.1, 12, 14, 18, 19, 22 (questions 4 and 7), 23, FR-WLD-049, 060, 062, 065, 066, 067, 068, 069, 071, 076, 091 and 100; CHK 6.8, 17.5, FR-CHK-070 and 071; the models `wld/seed.py`, `stream.py`, `worked.py` and `chk/worked.py` regenerated; ADR 0008 open points 1 and 2 answered (WLD 7.1 and 7.3); architecture.md 7.3 gains the optional second ID of a pair (D7 and D8). Marked stale in place ("stale under ADR 0008, see N-214"): WLD 13, 16.5, 17 (WLD-A5) and the S63 row of 22; CHK's header paragraph, 3.1 `Stream`, 4.2 tag 1, 5.4, R4, 6.2 step 8, 6.4 `WithSeed`, 6.6.3, 6.7.1, 8.1 item 4, 8.4, 9.3, 11 (EXP row), 15 (CHK-A2), 16, 17.2, FR-CHK-006, 025, 051 and the edits-owed rows of 20; CHK 4.2 tag 1, 5.4, 6.2 step 8, 8.4 and 17.2 must be rewritten before `checkpoint.go` (Stage 2), and the code of Stages 0 and 1 must not implement `Checkpoint.Stream` or a PCG-state section. **Still owed:** EXP 7.3, FR-EXP-062 and the use of `0x243F6A8885A308D3` in `DesignSalt`; CLB 4.2.1, 7.2.3, FR-CLB-024; STB L120, L124, D88 and the D90 row (it states the old reseed constant); TEC 4.4.1 and 4.4.6 (D16 takes no word); the site constants and IDs of D2, D4, D6, D11, D12, D13, D14, D15 and D17, each appended by its module with its change (ADR 0008 open point 3, the key of a site with no natural entity, stays open for D2's line and D13's route row); CODE `random.go` and `internal/ordering` (N-019); the rows of `doc/specs/models/README.md` for `wld/` and `chk/`, which still describe the old seeding rule.
- **Update (2026-10-12, Stage 0 built, uncommitted):** CODE is done for `random.go` and `internal/ordering`: `random.go` holds the seed, the one `rand.PCG` and `*rand.Rand`, the seven site constants of WLD 7.1.4 (unexported, literal, spaced by at least 2^32 under a `Guard:` test), the keyed reseed of the stream, the bounded draw of WLD 7.2.1 and the pure `reseed(seed, salt)`; `internal/ordering` holds `Finalise`, `Weyl32`, `Weyl2`, `Bucket`, `Key` in the `+gamma` form, `StreamKey` and `StreamKeySub`. The vectors of WLD 7.1.5 (a) to (d), the four bare-seed rows and the reseed rows are tests; a reseed plus a draw allocates nothing (`testing.AllocsPerRun`); `FuzzStream_Draw`, `FuzzKey` and `FuzzBucket` ran five minutes each with no crasher. **Stays owed, unchanged:** the stale-marked passages of WLD 13, 16.5, 17 and of CHK (3.1 `Stream`, 4.2 tag 1, 5.4, 6.2 step 8, 8.4, 17.2, FR-CHK-006, 025, 051) before `checkpoint.go` (Stage 2); EXP 7.3, FR-EXP-062 and `DesignSalt`; CLB 4.2.1, 7.2.3 and FR-CLB-024; STB L120, L124, D88 and the D90 row; TEC 4.4.1 and 4.4.6; the site constants and IDs of D2, D4, D6, D11, D12, D13, D14, D15 and D17, each appended by its module with its change (ADR 0008 open point 3 stays open); the `wld/` and `chk/` rows of `doc/specs/models/README.md`.
- **Update (2026-10-13, minor gaps of the Stage 0 audit):** applied: the `wld/` and `chk/` rows of `doc/specs/models/README.md` now describe the keyed streams and the reseed vectors; the old reseed constant is gone from the D90 row of `spec-stability.md` (the rule `seed' = finalise(seed xor finalise(salt))`, owner CHK 6.8.2). `0x243F6A8885A308D3` stays in EXP only as the Latin-hypercube design's own default salt (`spec-experiments.md` 4.4.2, 13.3, `models/exp/*.py`); it reseeds nothing, and changing it would move every spread and the worked points of 19 (FR-EXP-008), so it was kept and the text now says so. **Still owed**, unchanged: EXP 7.3 and FR-EXP-062 (the fingerprint); CLB 4.2.1, 7.2.3 and FR-CLB-024 (the flag `d`); STB L120, L124 and D88; TEC 4.4.1 and 4.4.6; the stale-marked passages of CHK and WLD 13, 16.5 and 17; the site constants and IDs of D2, D4, D6, D11 to D15 and D17.
- **Update (2026-10-16, plan 1-4 Phase 1):** applied for BEL, which leaves the areas: `spec-beliefs.md` 4.5, 5.1.1 and FR-BEL-008 say that the outcome's word is the first word of the stream keyed by (`siteOutcome`, tick, agent, commodity), D8, so it depends on no other outcome; the order of the outcomes stays the order of the events and of the `Revise` calls (`spec-clearing.md` 7.3). The IDs are kept. The note stays open for the other areas.
- **Update (2026-10-16, plan 1-4 Phase 6):** the BEL part stays applied; the code of the beliefs row (draw by word, `Change`-based tally and reason) follows it. Open for the other areas.
- **Update (2026-10-15, plan 1-3 Phase 1):** applied for CLR, which leaves the areas: D7 is one word per order from the market's own stream, keyed by (`siteClearing`, tick, region, commodity), and `spec-clearing.md` 3.3.1, 7.2 and FR-CLR-023 no longer say that the markets share one stream in market order. ADR 0008 open point 2 (a stateless key) is answered by `spec-world.md` 7.1 and 7.3 and is not reopened. Phase 4 of plan 1-3 pins the draw with the vector of WLD 7.3 (seed 42, tick 7, markets `(r0, grain)` and `(r0, loaf)`).
- **Update (2026-10-17, plan 1-6 Phase 1):** applied for WEL and checked for WLD: WEL 5.1 and FR-WEL-024 now say the target is the first of the agent's two creation words (`spec-world.md` 5.7.1, FR-WLD-066) from its own creation stream (D1 at construction, D10 at P7.1(d), the call's stream for an external agent, WLD 7.3), and the range stays in WLD 5.7.1, which already said so and needed no edit. The other owners' edits stay open.
- **Update (2026-10-17, plan 1-11 Phase 1):** the seed text is applied in `spec-world.md` 16.5. The other edits owed by this note are not all made; left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied for AGT, left open for the rest: the keys are built (`siteWork`, `siteTrade`, `siteCreation`) and the agent row names its draws: `Work` takes none, `Trade` one per intent considered (candidates decided by the view alone, D93, N-246), the two creation draws in the order target then residue (`drawCreation`, D95). D12's site (the external agent's call) is INT's.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate draws no random word (`spec-estates.md` 2.7.4: its orders take a tie-break key in the clearing like any order), so it adds no site. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The default draws nothing (FR-SPN-029); a consumer's Decide gets the stream (siteSpawner, t, region).
- **Update (2026-10-18, plan 1-10 Phase 1):** not this row's: the solver draws nothing (FR-VIA-020, FR-VIA-040). Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply for the world's sites (D155): D1, D7, D8 and D10 are called with the keys of WLD 7.3, D5 `siteTrade` before every `Trade` and D3 `siteWork` only for a consumer's `Work`; the draws of D2, D4, D6, D11 to D15 and D17 are modules'. Left open for the other owners.
- **Update (2026-10-23, plan 2-1 Phase 1):** applied for CHK and WLD: every passage marked "stale under ADR 0008" in `spec-checkpoint.md` (header paragraph, 3.1 `Stream`, 4.2 tag 1, 5.4, R4, 6.2 step 8, 6.4, 6.6.3, 6.7.1, 6.8, 8.1 item 4, 8.4, the EXP row of 11, CHK-A2, 16, 17.2 and 17.3, FR-CHK-006, 025, 051 and the edits-owed rows of 20) and of `spec-world.md` (7.x residue at the construction paragraph, 12, 13, 17 WLD-A5, the N-214 row of 20) is rewritten; the world section (tag 1) holds the next agent ID and the construction flag, the bucket and the pending totals are the ledger's (tag 2), and the header's seed is the effective seed, read back. `models/chk/worked.py`, `size.py` and their outputs are regenerated in Docker. Left open for EXP 7.3 and FR-EXP-062 (plan 2-20), CLB 4.2.1, 7.2.3 and FR-CLB-024 (plan 2-21), STB L120, L124, D88 and the D90 row, TEC 4.4.1 and 4.4.6, the `spec-stability.md` row of `spec-world.md` 22 (S63 and MIG-S3), and the sites and IDs of the module rows.

### N-215 The reseed rule is the identity for seed 0 and salt 0

- **Areas:** CHK, CODE
- **Raised:** 2026-10-12, `spec-checkpoint.md` 6.8.2 (Phase 1 of the Stage 0 plan).
- **Note:** `seed' = finalise(seed xor finalise(salt))` and `finalise(0) = 0`, so seed 0 with salt 0 gives seed' 0: that reseed changes nothing. Distinct salts still give distinct seeds, and a salt of 0 is an ordinary reseed for any other seed. ADR 0008 decision 5 fixed the rule and CHK 6.8.2 documents the case. Decide before the first golden file records it whether to accept the identity or to change the rule (for example by adding a constant before the finaliser).
- **Update (2026-10-18, plan 1-10 Phase 1):** not this row's: no site, no reseed. Left open.

### N-216 A site constant is never a small integer

- **Areas:** WLD, CODE
- **Raised:** 2026-10-12, `spec-world.md` 7.1.4 (Phase 1 of the Stage 0 plan).
- **Note:** The first step of the key fold adds the site constant to the seed, so (seed a, site X) and (seed b, site Y) are one stream when `a - b = Y - X`. The constants are far apart for that reason (FR-WLD-067: pairwise at least 2^32) and a module that appends a site takes the first 8 bytes of SHA-256 of `economy-sim/site/<name>`, never a small number. A `Guard:` test in `random.go`'s tests enforces the distance. D2, D4, D6, D11, D12, D13, D14, D15 and D17 have no constant yet (N-214).
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the welfare row adds no site; D1 is `siteConstruction`, D10 and D12 are the callers', and `internal/welfare` takes the creation word and names no site. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate row adds no site and no constant. Left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** not this row's: no site, no reseed. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read: the row adds no site and no constant. Left open.

### N-217 D5 binds every module's change

- **Areas:** ALL, CODE
- **Raised:** 2026-10-12, decision D5 of `plans/plan-phase-0-foundation.md`; `rules/architecture.md` 2.10.5.
- **Note:** A change that adds a module (or a field to a core struct, event or snapshot row) leaves no trace while it is off: an appended configuration field's default is its zero value and it is written to the hash only when not zero (a module's fields only when the module is on, `spec-config.md` 8.2.3); an appended event or snapshot field is a count, an amount, a bool, a constant or a slice, never a typed ID, and is omitted from the golden rendering at zero (`spec-observability.md` 9.5.2, FR-OBS-091); an event type with no occurrence is not listed. So the pinned hash of the all-off reference configuration (FR-CFG-105) and `testdata/golden/off/` do not move, and a diff there is a failure of the change, not a golden to regenerate (testing.md 7.1.1). The change also deletes its entries from the `notBuilt` lists of the registry test and the catalogue test (architecture.md 9.7).
- **Update (2026-10-13, Phase 1 of the ledger row):** applied to the reason registry: a module that is off adds no row and no boundary account, so the account IDs of a world with every module off are a function of the commodity count alone (`spec-ledger.md` 3.4.1, 7.2.2). A module that is on appends its boundary accounts after the built-in group, the declared flows and the `outside_*` reasons, in the order of its pointer (`spec-world.md` 5.2 row 3a; N-226).
- **Update (2026-10-15, plan 1-3 Phase 1):** the clearing row adds no configuration field, no event field and no off-golden row; `TestHashConfig_ReferencePinned` and `testdata/golden/off/` do not move. A standing rule; left open.
- **Update (2026-10-16, plan 1-4 Phase 1):** the beliefs row appends no configuration, event or snapshot field (the doc comments of `BeliefsUpdated` are corrected in Phase 5, no field changes); the pinned hash and `testdata/golden/off/` do not move. A standing rule; left open.
- **Update (2026-10-16, plan 1-5 Phase 1):** the production row appends no configuration, event or snapshot field and its module parameters are the neutral values (10 000 or no limit); the pinned hash and `testdata/golden/off/` do not move. A standing rule; left open.
- **Update (2026-10-16, observability review of plan 1-5):** the review appends two fields to core events, `RecipeExecuted.Overflow` (a slice) and `MarketCleared.CostAnchor` (an amount), and adds one core event, `RecipeRefused`. They are core, not a module's, so they are not omitted for a module that is off; the golden rendering omits each at its zero value (a world with no overflow, no refusal and no known anchor renders as before). The pinned hash does not move (no configuration field). `testdata/golden/off/` does not exist yet; when the world row's golden files are first recorded they include these fields. Left open.
- **Update (2026-10-17, plan 1-6 Phase 1):** applied: nothing in the welfare row appends to the configuration, an event or a snapshot field before its code, and `RoleRow.ClampedHouseholds` is at 0 omitted from the golden rendering, so the pinned hash cannot move; the exit gate checks the pinned hash. Left open.
- **Update (2026-10-17, plan 1-6 exit gate):** checked: `TestHashConfig_ReferencePinned` passes after the welfare row (nothing in it touches the configuration); `testdata/golden/off/` still does not exist, so that test alone is the gate until the world row records the first golden file. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied: nothing in the agent row appends to the configuration, an event or a snapshot field, so the pinned hash and `testdata/golden/off/` cannot move; the exit gate checks `TestHashConfig_ReferencePinned`. `Strategy` is already a hashed name through `Role.Strategy`. A standing rule; left open.
- **Update (2026-10-17, observability review of plan 1-7):** three fields are appended to core events (`ProductionHalted.Rested`, `ProductionHalted.NoRecipe`, `UpkeepShortfall.Short`): counts and a slice, never a typed ID, omitted from the golden rendering at 0 and empty, so a world with nothing to report renders as before; they are not a module's, so the rule binds them as it bound `RecipeExecuted.Overflow`. The pinned hash does not move (no configuration field). Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** applied: the estate row appends nothing to the configuration, an event or a snapshot row, so the pinned hash cannot move; the exit gate checks `TestHashConfig_ReferencePinned`. `testdata/golden/off/` does not exist yet. A standing rule; left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Apply. Nothing appends to the configuration, an event or a snapshot row; the pinned hash and testdata/golden/off/ cannot move.
- **Update (2026-10-18, observability review of plan 1-9):** applied to the one field the review appended: `EntryLimited.Stranded` is a count added at the end of the event, never a typed ID, and is omitted from the golden rendering at 0, so a world with no stranded household renders as before. No configuration field is added; the pinned hash does not move.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied. The bounds and the sentinels are appended in order; no `Config` field, event or snapshot row is appended; `TestHashConfig_ReferencePinned` must not move, and Phase 7 checks it. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Applied: the row appends no configuration, event or snapshot field to anything that exists; the snapshot is new and declares slice fields only; the pinned hash and `testdata/golden/off/` do not move (the rename of D140 changes no hash input: the field id of `Config.World` is 1 and the ids of its fields are unchanged). Standing; left open.
- **Update (2026-10-21, anchor fix):** applied to a core field: `World.AnchorHold` defaults to 0 and is written to the hash only when not zero (field 46), so a configuration that never sets it hashes as before; `TestHashConfig_AnchorHoldOffLeavesNoTrace` keeps the old pin. The reference chain (`base.toml`) sets `anchor_hold = 5000`, which is a pack edit: the pins of pack `one` (`TestHashConfig_ReferencePinned`, `TestPacks_LoadValidateAndHash`, `expected/one.toml`) and `testdata/golden/off/` moved once, explained in the change under testing.md 7.2. Pack `second` is unchanged. Standing; left open.

### N-218 The Stage 0 field set and the hash rule that goes with it

- **Areas:** CFG, CODE
- **Raised:** 2026-10-12, `spec-config.md` 3.7.2, 8.2.2 (Phase 1 of the Stage 0 plan).
- **Note:** The slice declares every field no module of `Config.Modules` owns, including the skill-barrier fields and every key the packs `one` and `second` set; `Modules` is declared with no field; `Edges` and `RegionPrices` are Stage 2 (field ids 8 and 9 reserved). The hash writes a field only when not zero; a non-nil pointer is always written, so `WarmStart` nil and `&WarmStart{}` differ and a field-less module (`Estates`) is the pair that names it; the id of a module's pointer is its position in the `Modules` row counted from 1 (Estates is 25), independent of build order. A field a module adds to a core struct (`Role.Informed`, `Region.PoolRelease`, `Recipe.Builds`) is written only when the module is on. `Group.Scale` 0 means "the world's `Scale`" and is hashed as written, so a `Group.Scale` equal to `World.Scale` and a 0 behave the same and hash differently; FR-CFG-004 compares like content and accepts it. The depth rule of the loader counts key segments and value nesting together against 16, derived from the deepest pack key (`modules.external.node.<r>.line.<c>.sell.margin_bp`) plus 2: a module with a deeper key raises `MaxNestingDepth`, which is a bounds change.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. `WarmStart` is hashed as Stage 0 built it; this row changes no hash.

### N-220 The registry of knobs needs the new owner rows

- **Areas:** CLB, EXP
- **Raised:** 2026-10-12, `spec-config.md` 9.2 (Phase 1 of the Stage 0 plan).
- **Note:** `spec-calibration.md`'s registry needs rows or groups for the new owner-table rows (`Scale` of `World` against `Scale` of `Group`, `Upkeep` of `Role`, `Yields`, the element-type rows) and a group for `Modules.Estates` (S-CFG01 lists 24 names). `doc/specs/models/clb/registry.py` still holds the old path spellings (`resources.*`, `demand.*`, `module.external.*`, `modules.transport.modes.*`) and is owed a respelling by the models' owner.
- **Update (2026-10-13, minor gaps of the Stage 0 audit):** applied in part. `models/clb/registry.py` is respelled (`modules.resources.*`, `modules.demand.*`, `modules.transport.mode.*`, `modules.external.node.*`, `modules.tenure.term.*`) and re-run in Docker; the path and flag checks pass. `spec-calibration.md` S-CFG01 and 6.3 now list `Estates` as the last pointer (23 names: the owner table has no pointer row for `Demand` and `MoneySupply`, which are scopes of S-DEM01 and S-MON01), and FR-CLB-002 reads `RegionTaste` and `Taste.Role` as owner rows. **Still owed:** four coverage checks of the registry fail against the live owner table (`registry.out` records FAILED): the element-type, `Config.Edges`, `RegionPrices`, `WarmStart`, `Yields`, `World.Scale`, `JobSpec`, `Month`, `Season` rows each need a class, a stage, the flags `in` and `chk` and an owner section, which is a classification decision per field and not a respelling; the groups still use the old scope tokens; the CFG 3.7 names `Days` and `InProgressBP` and the nested `Scale` have no scope. Run the script as `docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v $PWD/doc:/d:ro -w /d/specs/models/clb python:3-slim python registry.py`.

### N-221 STO confirms the file layout of its base room and role rows

- **Areas:** STO, CFG
- **Raised:** 2026-10-12, `spec-config.md` 6.2.1 (N-102, closed).
- **Note:** `spec-storage.md` states only `[modules.storage.class.<id>]` (7.3). Rule 6.2.1 of CFG derives `[modules.storage.base]` (mass, bulk) and `[modules.storage.role.<id>]` (stores, base). STO confirms or restates them in its own section.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row. Left open.

### N-222 The cost of a reseed at every draw site is measured on paper only

- **Areas:** SCL, CODE
- **Raised:** 2026-10-12, `spec-world.md` 7.1.3 and ADR 0008 (Consequences).
- **Note:** A site call costs five finaliser evaluations (six with a fourth word) and one `PCG.Seed`, and is allocation-free (checked with `testing.AllocsPerRun`). The first slice should measure it against the cost model of `spec-scale.md` 6 and say whether the per-agent sites (D1, D3, D5, D8, D10) need a cheaper key.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate row adds no draw site. Left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** not this row's: no per-agent key. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. Phase 10 records the allocation count of a site call; nothing is measured in time.

### N-224 Stage 0 guards and what they pin

- **Areas:** CODE
- **Raised:** 2026-10-13, the closing of the minor gaps of the Stage 0 audit.
- **Note:** Stage 0 ships guard tests that fail when a later change does not update them. A module PR reads this list and updates what it names, in the same change, never by loosening a guard.
  - `import_guard_test.go`: the root, `internal/**` and `config/**` (and `strategy/`, `indicator/`, `experiment/` when present) import the standard library and the layers architecture.md 2.7 and 2.9 allow; a new third-party import is refused outside `config/`.
  - `glossary_test.go`: every term of architecture.md 3.2 is a `doc.go` glossary term or is in the `notYetTerms` list (Market, Order, Trade, Run, Strategy, Clearing, Spawner, Snapshot, Policy), and every exported type of `value.go`, `config.go` and `event.go` is named in `doc.go`. The change that builds a listed term deletes it from `notYetTerms` and adds its glossary entry.
  - `toolchain_test.go`: `go.mod` has exactly one `go` directive, `1.25`, and no `toolchain` line; raising it is a breaking change with a changelog entry (architecture.md 5.4) and moves the ordering vectors.
  - `doc_comments_test.go`: the contract phrases of the `Observer` and `WithSeed` doc comments; reword the test with the comment.
  - `determinism_guard_test.go`: the banned uses (`time.Now`, global rand, `init`, package-level mutable state) and a `panic(` whose arguments carry no string containing "invariant", over the scope function `guardedFiles` (a new guarded directory is added there).
  - `event_test.go`: the staged catalogue. `deferredCore` names the four core-gated events not built (`WorldRestored` Stage 2 `checkpoint.go`; `AgentRelocated` and `EdgeStatusChanged` `regions.go`; `YieldShocked` `integration.go`) and fails on a stale entry, so the change that declares one deletes its row; a module's events leave `notBuiltEvents` in the change that builds the module (N-126).
  - `TestConstants_Values` keeps the reason and cause constants against the owning specs (`spec-clearing.md` 3.2 and the module specs) with `deferredReasons` for `not_permitted` and `no_room`.
  - `validate_counts_test.go` (24 count bounds, each over by one) and `errors_test.go` (the 14 structural sentinels, the path cap): a new bound or sentinel adds its row. `MaxDraws`, `MaxEdges`, `MaxRegionPrices` and `MaxWants*` have no check at Stage 0 and are the module PRs'. `ErrModuleDependency` is declared with no returner; the first module rule that returns it adds the test.
  - `config/`: `TestPacks_LoadValidateAndHash` pins the hashes of packs `one` and `second` (a pack edit updates them with the reason, testing.md 7.2); `TestPacks_DefectOverlays` loads d1 to d6 over a cold and a warm layer set; `FuzzLoad` requires a loader sentinel and the zero `Config` on a rejection, and a `*ValidationError` or `ErrTooManyErrors` from `Validate` on an acceptance.
  - `internal/ordering` `FuzzKey`: `Key` is a bijection in the id for a fixed tick; a change of the finaliser or the `+gamma` form changes the vectors and is a behaviour change.
- **Known weaknesses of the guards:** (1) `FuzzLoad` does not by itself catch a `Parse` that returns a nil error and the zero `Config` for garbage, because a zero config is deterministic and `Validate` classifies it; `TestParse_SyntaxErrors`, `TestLoadFS_Limits` and `TestParse_Limits` catch it, so do not delete them as redundant. (2) `TestConstants_Values` reads the closed sets from the spec text by a plain parse, so a spec that reflows a table can fail it without a code change: fix the parse, not the constant. (3) The `Guard:` tests are Stage 0's: a module PR that adds a root file runs them all, so read their failure text before changing a list.
- **Handoff to Stage 1:** the deferred rows of the Rule ledger are the first work of `viability.go` (the five viability rules, `ViabilityError`, the solver-dependent warm-start checks and the empty-virtual-start-offsets check of `ErrWarmStartJob`, which needs the calendar's season lookup); `ErrUnknownStrategy`, `ErrUnknownSpawner` and `ErrUnknownClearing` arrive with their interfaces (D2); the world-dependent observability tests (logs, non-interference, cardinality, counterpart, golden rendering, emission order, run lifecycle, the call-site and full-`Step` allocation limits) are Stage 1's "Done when"; the glossary adds each stage's terms before it uses them (`notYetTerms`); the pinned all-off hash (`TestHashConfig_ReferencePinned`) and `testdata/golden/off/` must not move (N-217).
- **Rule ledger:** the built, deferred and module rows of CFG 7.3 and 7.4.1 are in `plans/plan-phase-0-foundation.md`, "Rule ledger".
- **Update (2026-10-18, plan 1-9 Phase 1):** Apply. Phase 2 deletes Spawner from notYetTerms; Phase 4 extends TestConstants_Values and adds the contract-phrase test for the Spawner doc comment.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied. `validate_counts_test.go` and `errors_test.go` gain the checks of the two bounds (read as constants) and the four sentinels; `import_guard_test.go` gains the rule that only `internal/steady` imports `math/big` in its non-test files (seven existing test files use it as an exact oracle; architecture.md 3.5 restricts computing, not testing); `determinism_guard_test.go` must pass for the new files. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: `guardedFiles` is a directory walk and needs no edit; `deferredCore` loses nothing (the events of the slice are built); `notYetTerms` loses `Run` in Phase 2 and `Snapshot` in Phase 9, each with its glossary entry in the same change. Left open.

### N-225 The ledger's reason constants are internal numbers

- **Areas:** LED, CHK
- **Raised:** 2026-10-13, `plans/plan-phase-1-1.md` A3 and Phase 1 (deliverable 4).
- **Note:** `internal/ledger` names a reason by a small integer constant of its own and holds no string, so that it knows no identifiers (`spec-ledger.md` 2.7) and its registry is a fixed table (16). The numbers are free to change from one build to the next, and the rows a module declares take numbers that depend on which modules are on. A checkpoint must therefore never save one raw: the pending totals are per (region, reason, thing) and CHK writes the reason by its name (the names table of `WorldCreated`) or by its position in the closed registry, which the configuration hash fixes, and the boundary balances are keyed by account ID, which the creation order of `spec-ledger.md` 3.4.2 fixes. `ledger.go` maps the ledger's constant to the root's `Reason` both ways (plan Phase 7). CHK 4.2 tag 2 and its restore check say which of the two spellings is saved before `checkpoint.go` (Stage 2).
- **Update (2026-10-16, plan 1-5 Phase 1):** read for the production row; left open for CHK (Stage 2). The create-and-dispose operation names `recipe_output` and `disposal` by the ledger's constants and the root names the five production reasons by its own `Reason` constants (a refusal is a returned value and is never saved); tag 6 saves no reason.


### N-226 The creation position of the boundary accounts of the other module reasons

- **Areas:** WLD, LED, DEM, TRN, RSK
- **Raised:** 2026-10-13, `spec-ledger.md` 3.4.2 and `spec-world.md` 5.2 (Phase 1 of the ledger row).
- **Note:** `spec-world.md` 5.2 had rows for the built-in reasons (1), the declared flows (2) and the `outside_*` reasons (3), and none for the other faucet and sink reasons that a module owns: `want_consumption` (DEM), `haul_use` and `transit_use` (TRN) and `loss` (RSK). Row 3a is added: after the `outside_*` accounts and before the estates, in the order of the `Modules` pointers, and within a module as the built-in group is ordered (money reasons first, then goods commodity-major). This is a proposal of the ledger row, because the account IDs of a world with such a module on depend on it and no module is built yet. Each of the three module specs confirms it, or says another place, in the change that builds its file (architecture.md 2.10.5: appended, never inserted); until then the position is not saved by any checkpoint. A reason that a module has in the goods form only creates one account per commodity, as the built-in group does.
- **Update (2026-10-16, plan 1-5 Phase 2):** read for the ledger additions; left open (not this row's). `Discard` uses only boundary accounts of the built-in group (`recipe_output` and `disposal`, created at construction in the order of `spec-ledger.md` 3.4.2); it adds no account and no module reason.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The slice has no module reason: the world creates the accounts in the order of WLD 5.2 rows 1, 4 and 6 and leaves rows 2, 3, 3a, 5 and 5a to the modules.

### N-227 Edits to CRD from the ledger row: `MarkOverdue` and a `Settle` of 0

- **Areas:** CRD, INS
- **Raised:** 2026-10-13, `plans/plan-phase-1-1.md` D9 and Phase 1 deliverable 1b.
- **Note:** The owner accepted two recommendations and `spec-credit.md` is amended in this change: (i) CRD 8.5 had the settlement set `Overdue` and none of the five operations of 3.2 did, so a sixth ledger operation `MarkOverdue(claim, tick)` sets it (atomic, refused with `ErrUnknownClaim`, and, beyond the plan, with `ErrClaimShape` for a `tick` below the claim's `Due`, so that `Overdue` is -1 or in `[Due, tick)` by construction, CRD 17.5; an `Overdue` already set is left as it is); (ii) a `Settle` with one part 0 skips that transfer, because a transfer of 0 is refused by `spec-ledger.md` 4.2, and both parts 0 is `ErrInvalidAmount`. Edited: CRD 3.2 and 3.2.1, 8.5 (item 2), FR-CRD-003, 032 and 034 (the rows that count the operations or set `Overdue`), the edits table and the home table; LED 10, FR-LED-063 and 16. Still owed: the models of CRD were not re-run (in Docker only): `doc/specs/models/crd/loans.py` sets `over` as a field write (line 83, which is `MarkOverdue` in the library) and `doc/specs/models/crd/alloc.py` draws random operations on a mini ledger (`Lend`, `Settle`, ...) with no `MarkOverdue` and, to check, a `Settle` that may be given a part of 0 (legal now, and both parts 0 is not); `spec-insurance.md` 9.2 and FR-INS-004 name `Obligate` only and need no edit; `spec-integration.md` 13.2 (`ForgiveClaims` uses `WriteOff`) is unchanged. The ledger code of the six operations is Phase 5 of the plan.

### N-228 A module's items in the tradable set have no touch point

- **Areas:** AGT, DEM, SUB, CODE
- **Raised:** 2026-10-13, `plans/plan-phase-1-2.md` (Phase 1, O1 and O2).
- **Note:** The tradable set of a role (`spec-agents.md` 3.4) is core data plus two module terms: the items of the wants in the role's ladder (`spec-demand.md` 3.2.3, FR-DEM-006) and the non-discard outputs of the effective own-use recipes (`spec-subsistence.md` 3.6). `rules/architecture.md` 2.10.1 does not list "its items in the tradable set" among a module file's touch points, 2.10.2 lets a module appear outside its file only in the catalogue files, and the four fan-ins of 2.10.4 do not include it. `role.go` (Stage 1) builds the set once at construction from the core terms and must leave a place for the two modules (`validate.go`'s `tradableSets` already says they are "added with those modules"). Recommended: a fifth fan-in with one entry point, a fixed-order sequence of guarded calls into the module files that append items before the set is frozen, added to 2.10.4 (and to the list of 2.10.1) with a fresh tag, in the change that builds the first of the two modules (`demand.go`, Stage 2). Not blocking: both modules are Stage 2. Second point: `spec-agents.md` 22 homes the targets, tiers, spend cap and the tradable set in `internal/household`, which is not in the layout of 2.6 or in `AGENTS.md`; the tradable set is a union of lists and lives in `role.go`, and the agent row decides whether the arithmetic needs the package (2.1) and then adds it to 2.6.
- **Update (2026-10-14, Phase 1 of plan 1-2):** applied in the specs: `spec-agents.md` 3.4, 5.2 and 22, and `spec-production.md` 3.3, say that the tradable set and the effective upkeep are derived once at construction into the role table (`role.go`), in canonical commodity order, that a module adds its items inside that constructor before it returns, and that nothing mutates the table afterwards (that is all "frozen" means); AGT 22 now homes `Role` in `config.go` and the tradable set in `role.go`. **Still open:** the fifth fan-in in `architecture.md` 2.10.1 and 2.10.4 (applied in `architecture.md` 2.10.4; `tradableItems` is built with the first module PR, `demand.go`) and whether the agent row needs `internal/household` (it then adds the package to 2.6 and `AGENTS.md`).
- **Update (2026-10-17, plan 1-6 Phase 1):** read: AGT 3.8 is the only AGT clause the welfare row edits and it concerns the length of the record; welfare adds no item to the tradable set and no fan-in, and its arithmetic is `internal/welfare`, not the `internal/household` of `spec-agents.md` 22. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied in part: D80 settles the second point (the agent row needs `internal/household` and adds it to `architecture.md` 2.6 and `AGENTS.md`). The first point (the fifth fan-in for the tradable set) is applied in `architecture.md` 2.10.4; the want items of FR-AGT-003 join the role table with the demand module. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate row adds no item to the tradable set. Left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. The solver reads the role table's tradable set only through V1's producibility, with no new touch point.

### N-230 The run of the "price beliefs at bound" `Warn` is advanced for every market every tick

- **Areas:** BEL, WLD
- **Raised:** 2026-10-16, Phase 1 of `plans/plan-phase-1-4.md` (A6, A7).
- **Note:** `BoundRun` (`spec-beliefs.md` 12.2, 13) is advanced once per market per tick, also in a tick in which the market had no outcome: such a tick does not hold the condition and resets the run (`spec-observability.md` 7.3). The world row, which owns the market record and the call, must call the run function for every market every tick, not only for a market with a `BeliefsUpdated`. Second point: a belief with `Floor` equal to `Ceiling` is at both bounds, so `AtFloor + AtCeiling` can exceed `Outcomes` and fewer than half of the outcomes can fire the `Warn` (`spec-beliefs.md` 12.1.2); the literal test and the metric `economysim.belief.at_bound` (which reads both fields) keep it on purpose. The bound the `Warn` names is the one with more beliefs, `floor` on a tie. Decide with the first scenario that pins a market (S01, S04) whether the double count matters.
- **Update (2026-10-16, observability review of plan 1-4):** applied in OBS and BEL: the metric `economysim.belief.outcomes` (the denominator of `belief.at_bound`, which the adapter holds across a tick with no event while the run resets) and the `outcomes` attribute of the `Warn` now let a reader see both points, a stale pinned gauge and a 1-of-1 pin; `economysim.belief.limited` counts `Conflicts` and `Clamped` (`spec-observability.md` 6.2.1, FR-OBS-094; `spec-beliefs.md` 12.2, 12.4). The world row must still call `beliefs.NextRun` for every market every tick and emit `BeliefsUpdated` before the `Warn`. Still open: whether the double count matters.
- **Update (2026-10-17, plan 1-11 Phase 1):** `spec-world.md` 6.1.13 now says the world advances the bounded `Warn` runs for every cell every tick (this covers the `BoundRun`; the name `beliefs.NextRun` is not written there). Still open: whether the double count matters, and the code of the world row.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the run of the "price beliefs at bound" `Warn` is the world's (row 11 calls `BoundRun` for every market every tick); `Revise` returns the belief and discards the result. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply (D156, Phase 6): `beliefs.NextRun` is called for every market every tick and `BeliefsUpdated` is emitted before the `Warn`; `spec-world.md` 6.1.13 states the call. Whether the double count of a pinned market matters stays open for S01 and S04 (row 12).

### N-236 Three halves of FR-AGT-003, 010 and 011 are the agent row's

- **Areas:** AGT
- **Raised:** 2026-10-13, audit of `plans/plan-phase-1-2.md` (Testing summary, Phase 5 row).
- **Note:** The role table of plan 1-2 covers the core terms of the tradable set (FR-AGT-003), the per-household merge of the effective upkeep (FR-AGT-011) and a role with no recipes being valid (FR-AGT-010), and nothing else of them. Three halves have no owner but the agent row, which has no plan yet (testing.md 4.5): (1) FR-AGT-011: each quantity of the effective upkeep is multiplied by the agent's weight, with checked arithmetic, in `agent.go` or `world.go`; (2) FR-AGT-010: an agent of a role with no recipes requests no job, in the default strategy and in the world's refusal; (3) FR-AGT-003: the items of the wants of the role's ladder, added with the demand module (N-228). The agent row's plan lists the first two as its requirements; the third is the demand module's PR.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the welfare cost function is the only place where the welfare cost multiplies an upkeep quantity by the weight (FR-WEL-004); the agent row's product (what the agent buys and eats, FR-AGT-011) is its own and is not handed over, so the agent row's plan confirms that no row multiplies twice. Left open for the agent row.
- **Update (2026-10-17, plan 1-7 Phase 1):** parts (1) and (2) are applied: FR-AGT-011 (the product by the weight, checked) and FR-AGT-010 (an agent of a role with no recipes requests no job) are written in `spec-agents.md`. Part (3), the items of the wants, is the demand module's PR, so the note stays open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: not touched by the estate row (it multiplies no upkeep by a weight). Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. Waits for the agent row; this row uses only the cause test Money < Upkeep * w.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open for row 7.

### N-237 A job stores its effective factor; its start tick is derived

- **Areas:** PRO, CHK, AGT
- **Raised:** 2026-10-13, audit of `plans/plan-phase-1-5.md` (the values of `RecipeExecuted`), decision D67 of plan-phase-1-5 (not the damper D67 of `spec-stability.md`).
- **Note:** `RecipeExecuted` carries `StartTick` and `EffectiveYield`, and the job of PRO 5.1.1 held neither. `StartTick` is the completing tick minus `Duration` plus 1 (a job started in `s` completes in `s + d - 1`), derived and not stored, valid after a weight fall and for a warm-start job (where it may be negative). `EffectiveYield` is the job's new `Yield` field, the effective factor of 6.3 read at the start and not recomputable at completion (the season, shock and technology may have changed; a restored world has no start-time inputs). Written into PRO 5.1.1, 14.1, 15 and FR-PRO-041 and CHK 4.2 tag 6 (`Yield` after `Principal`); AGT holds the job as `*Job` defined in PRO and needs no change. The agent row stores the field with the job; the warm-start job (VIA 8.5, FR-VIA-035) sets it from the virtual season's factor; the checkpoint row (Stage 2) writes and reads it in tag 6; a test restores mid-job across a season change and checks the reported `EffectiveYield`.
- **Update (2026-10-16, plan 1-5 Phase 1):** the lines were checked in the production row's Phase 1: `spec-production.md` 5.1.1, 14.1, 15 and FR-PRO-041 and `spec-checkpoint.md` 4.2 tag 6 still carry `Yield` after `Principal` and the derived `StartTick`, and the two sentences below the table of 14.1 are intact. Left open for the agent row (stores the field), the checkpoint row (writes tag 6) and the warm-start job (sets it).
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the welfare row edits `spec-checkpoint.md` 4.2 tag 7 and not tag 6 (the job, `Yield`). The tables are edited by several rows, so each re-reads just before the edit. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the agent holds `job` by value, `Yield` included; the world row stores it at P2.7; the warm-start job and the checkpoint are others'. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open for row 7 and Stage 2; the world stores `Yield` with the job at P2.7 and the warm-start job sets it from the virtual season (Phase 3).


### N-241 The duplicate-order scan is linear: O(n^2) in the orders of one book

- **Areas:** CLR
- **Raised:** 2026-10-15, plan 1-3 as built (`book.admit`, the `shortcut:` comment).
- **Note:** `book.admit` finds a duplicate (and the mixed ordinary and named-lot case, N-238) by scanning the entries of the book, once per admitted order, which is O(n^2) in the orders of one market. It is fine for the first consumer's world (about 1 000 households and 17 active markets per region, so at most about a thousand orders in a book and about 10^6 comparisons a book a tick, `spec-reference-chain.md` 4.4.4) and keeps the book allocation-free. **Upgrade trigger:** a book above a few thousand orders (a market of a larger world, or `Weight` 1 at the scale of `spec-scale.md` 6), or a benchmark of admission that shows it above the clearing itself; then index by (agent, lot, side) in the book's own storage (a sorted slice or an open-addressing table keyed by the agent), keeping the duplicate and mixed checks in one pass and the allocation test at zero. The plan's NFR-CLR-001 is about `Average`, not admission, so nothing is out of requirement today.
- **Update (2026-10-15, observability review of plan 1-3):** the trigger is now readable: `economysim.market.orders` gains the attribute `economysim.region` (`rules/observability-and-logging.md` 5.3, 5.4.1; `spec-observability.md` 6.2), so a book that grows toward a few thousand orders shows as the per-tick rate of one (region, commodity) series. On `five` that is about 85 active markets by 2 sides, 170 series, against 30 commodities by 2 sides, 60 series, before. Nothing else changes here; the index is still the remedy.

### N-243 Follow-up of the Codacy gate on plan 1-5

- **Areas:** CODE, PRO
- **Raised:** 2026-10-16, PR 5 (`stage-1-5`), after the Codacy findings were fixed.
- **Note:** The Codacy gate is zero new issues of any severity, so a PR must pass Lizard and markdownlint as Codacy runs them (test files are excluded by `.codacy.yml`). To do:
  - **Lizard limits** are cyclomatic complexity 10, 8 parameters and a file of about 500 lines of code. Run `lizard -l go -C 10 -a 8 -L 100 -w <files>` before a push; the plan-1-5 PR needed `UnitCost` (now `UnitCost(inputs, tools, prices, Basis)`), `Anchors`, `Discard`, `start`, `complete`, `anchors` and `refusalTally.add` split, and `production.go` split into `production.go` and `production_report.go`. The agent, world and module rows will meet the same limits; a long step is split into helpers that keep its order of checks. `internal/checked.muldiv` (CCN 13) is over the limit and untouched: split it with the first change that edits it.
  - **Markdown tables** use the spaced separator row (`| --- | --- |`); `|---|` fails MD060 (compact style).
  - **`production_report.go`** is a new root file (events, refusal tally, snapshot aggregate, cost anchors) and is in neither `AGENTS.md` nor `architecture.md` 2.6 and `documentation_index.md` has no row for it. Add it in the change that next edits those files, with a fresh tag.
  - **`production.Rescale`** now checks every pending quantity before it writes any (atomic), but its overflow branch cannot be reached for a valid input (`w < k` makes each result at most the quantity), so no test can make it fail. Keep it as the guard of a contract, with a coverage exclusion under testing.md 9.5 if the threshold needs one, or drop the branch.
  - **Coverage** shows to Codacy as `MissingRequirements` and its reviewer saw an empty report: no CI uploads `coverage.out` (plan O3, N-019). Add the upload with the CI.
  - **Plan 1-6 gate (2026-10-17):** `lizard -l go -C 10 -a 8 -L 100 -w` found one warning in the welfare files, `HeldHouseholds` with 9 parameters; fixed by passing the role's window, floor and target as `welfare.Floor`. The helper and `Floor` were then removed as unused (the root counts per agent, because each agent has its own target), with their test. `markdownlint` was not run. The coverage reading of the full run is the hand reading of plan O3; no CI uploads it.
  - **Still not built, and not this row's:** the checkpoint of jobs and yield shocks (N-237, tag 6, Stage 2). The Codacy review lists both as missing; they are the agent, world and checkpoint rows.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate row's files stay under the Lizard limits by one concept file, `estates.go`; the module half is appended after a marked comment line. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Apply to this row's files: Lizard before every push, the spaced separator in Markdown tables, the new root file in the layout.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied. The Lizard limits (complexity at most 10, at most 8 parameters, files of about 500 lines) are a design input of Phases 2 to 6; `lizard -l go -C 10 -a 8 -L 100 -w` runs before each push; `viability_check.go` and `viability_table.go` are named in the layout (architecture.md 2.6, `AGENTS.md`); markdown tables of the plan use the spaced separator. `internal/checked.muldiv` (CCN 13) is untouched unless this row edits it. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: each file of the row passes `lizard -l go -C 10 -a 8 -L 100 -w`; the Markdown tables of the row use the spaced separator row; the files are in the layout (N-287).
- **Update (2026-10-23, plan 2-1 Phase 6):** Lizard baseline recorded: 20 warnings over the non-test files (`validate.go` 16, `config/guard.go` 2, `config/config.go` 1, `internal/checked.muldiv` CCN 13), not only `muldiv` as expected; Codacy's enabled patterns are CCN 10, function NLOC 50, 8 parameters and file NLOC 500, and `-L 100` is the function length. `muldiv` stays open. Markdownlint is enabled in Codacy and uses the repository config; its version is not exposed by the API. Coverage upload is the `full` job of the planned CI row (owner's decision of 2026-10-23: CI yes).

### N-244 A module's per-agent state is in the module's slices, not in the agent type

- **Areas:** AGT, CHK, SUB, MIG, POP, ORG, STO, SKL, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-7.md` D81 (coordinator ruling).
- **Note:** `spec-agents.md` 3.1 sketches an agent that hosts `DeathCarry`, `OwnUse`, `OwnUseJobs`, `Journey`, `Member`, `Stores` and `Trainee`. `rules/architecture.md` 2.10.1 puts a module's per-agent state in the module's own slices indexed by agent slot, never as fields of the agent type, and the rules win. The agent row builds the unexported `agent` with the fields no module owns only. Each module keeps its field in its slice; the checkpoint still writes it in the agent record (N-128, `spec-checkpoint.md` 4.2 tag 3). Phase 1 of the plan edits AGT 3.1 and 3.2 to say so.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied in `spec-agents.md` 3.1 and 3.2 (module slices, never fields of the agent type). Kept open until the Phase 8 gate of plan 1-7 shows the agent row is built that way.
- **Update (2026-10-17, plan 1-7 Phase 8):** the gate shows it: `agent.go` holds only the core fields (D81, `TestAgent_HoldsNoMoneyQuantityOrRegion`) and no module field. Kept open as the standing rule for each module's PR.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read. The world holds no per-agent field of a module; row 7 builds the unexported agent with the fields no module owns, and the checkpoint writes the rest (Stage 2). Left open.

### N-245 A tunable a module's default rule reads joins the view's Tunables group

- **Areas:** AGT, BEL, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-7.md` D82 (coordinator ruling).
- **Note:** `DefaultStrategy` has no fields (AGT 7), so every number its rules read arrives in the view. The belief parameters (the fifteen of `beliefs.Params`, not `MaxPrice` or `PriceWindow`) are an exported plain-data group of `AgentView` and `OutcomeView`, filled by the world from `beliefParams` and converted inline. A module whose default rule reads a tunable (saving, credit, employment, insurance) appends it to the same group of the view, at the end, and to the spec's view table; it does not give the strategy a field.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied in `spec-agents.md` 6.4 and `spec-beliefs.md` (the Tunables group of the view). Kept open: a later module's default rule appends its tunable to it.
- **Update (2026-10-17, plan 1-7 Phase 8):** built; the group and its round trip against `beliefParams` are tested (Phases 5 and 7). Kept open for the modules.
- **Update (2026-10-17, plan 1-7 Phase 1):** checked against D82 and the Phase 1 text of `spec-agents.md` 6.4.1: the group holds the fifteen fields of `beliefs.Params` only (not `MaxPrice`, not `PriceWindow`); a test in Phase 5 asserts the group converted back equals `beliefParams` field by field, so the two cannot drift.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply for the slice: `fillCore` (row 7, called by `fillAgentView`) fills the belief group of `AgentView` from `beliefParams`, converted inline; the world builds `viewCtx` (`refreshViewCtx`, Phase 4) and fills the belief group of `OutcomeView` itself in P5.4; a module appends its tunable to the same group. Left open.

### N-247 `Directed`'s state methods and the directive in the checkpoint wait for the checkpoint row

- **Areas:** AGT, CHK, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-7.md` (Out of scope).
- **Note:** `Stateful` does not exist before `checkpoint.go` (Stage 2), so the agent row builds `Directed` and the `Directive` type without `SaveState` and `LoadState`. The checkpoint row adds them (delegating to the fallback, `spec-agents.md` 10.3), the directive in the agent record (AGT 15), the table of strategy names (AGT 15) and the assertion that `Directed` is `Stateful`.
- **Update (2026-10-17, plan 1-7 Phase 1):** read; unchanged. Left open for the checkpoint row.
- **Update (2026-10-17, plan 1-7 Phase 8):** `Directed` and `Directive` are built without the state methods, as planned. Still open for the checkpoint row.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.

### N-254 A lot that gains units keeps its listed price

- **Areas:** EST, CHK, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-8.md` D102.
- **Note:** The estate's price is per lot key (`spec-estates.md` 2.7.5, `spec-checkpoint.md` 4.2 tag 8), and two lots with one key are one lot (EST 6.1(c)). So goods that a later removal moves into a held key join the lot at its current, possibly marked-down, price. A non-spoiling, non-wearing commodity is one lot per estate, so this is the common case. If a scenario (S02, S14, STB-L38) shows it lowers the local price too much, reprice a growing lot to the higher price: a one-line change. Phase 1 of the plan writes the rule into EST 2.7.3.
- **Update (2026-10-17, spec edits):** `spec-estates.md` 2.7.3 now says "A lot that receives units keeps its price, and the joining units are listed at it" (the earlier grep missed the wording). The spec is written. Left open until the code merges and a scenario (S02, S14, STB-L38) shows whether the repricing is needed.
- **Update (2026-10-18, plan 1-8 Phase 1):** `spec-estates.md` 2.7.3 also says a row is dropped when its lot is gone at P5.3 as well as at the P3.4 walk (D102), so goods that arrive under the key of a lot sold wholly are priced at the first price, as after a restore. Left open until the code merges.
- **Update (2026-10-18, plan 1-8 Phase 5):** built in `estates.go`: one `reconcile` merge keeps a row whose lot is held (price and mark), drops a row whose lot is gone, and adds a row at the first price only in the insert-enabled walk of `asks`; `markdown` runs it filter-only, so a same-key arrival after a wholesale sale is priced at the first price (`TestEstateRound_ArrivingUnitsAndRepricing`, `TestEstateRows_StaleAndRoundTrip`). The gate's broken copy (a stale row kept) fails both. Still open: the Phase 1 to 4 code is in the working tree and the commit is pending; close it when that commit merges, unless a scenario shows the repricing is needed.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.

### N-256 The checkpoint saves listing rows of held lots only

- **Areas:** CHK, EST
- **Raised:** 2026-10-17, `plans/plan-phase-1-8.md` D109.
- **Note:** A row of the estate's listing prices (`spec-checkpoint.md` 4.2 tag 8) whose lot sold or spoiled after the last P3.4 is stale at the boundary where a checkpoint is taken. The section writes only the rows of held lots; a held lot with no row is priced at the first price at the next listing, which equals the uninterrupted run (the first price is a function of the configuration). Restore refuses a row for a lot the estate does not hold (FR-CHK-023), rows out of order or repeated, and a price outside 1 to `MaxPrice`. The Stage 2 checkpoint row builds the section from `estates.rows` and `estates.restore`.
- **Update (2026-10-17, plan 1-8 Phase 1):** written in `spec-checkpoint.md` 4.2 tag 8 and FR-CHK-023. Left open until the Stage 2 checkpoint code builds the section.
- **Update (2026-10-18, plan 1-8 Phase 1):** `estates.rows` takes the ledger to know which lots are held and is filter-only (it inserts no row and needs no configuration); `restore` checks a key by requiring it to be a held lot, with `errListingRows` for the checkpoint row to wrap in `ErrCheckpointCorrupt` (D109). Left open until the code merges.
- **Update (2026-10-18, plan 1-8 Phase 5):** built as written: `rows(l, r, dst)` has no `cfg`, omits a held lot with no row and drops a stale row; `restore(l, r, rows)` refuses a row out of order, repeated, out of range or for a lot not held with `errListingRows` (`estates_listing_internal_test.go`, `FuzzRestore`). The Stage 2 checkpoint row still has to build tag 8 from these two and wrap the error. Still open for that row; the Phase 1 to 4 commit is pending.

### N-288 What the world saves, and the diagnostic runs of its `Warn`s

- **Areas:** CHK, WLD, OBS
- **Raised:** 2026-10-17, `plans/plan-phase-1-11.md` (D145, D156).
- **Note:** The world keeps in one struct what a checkpoint must save: the effective seed, the next agent ID, the construction flag, the diagnostic runs of the bounded `Warn`s it owns (households unfed, price beliefs at bound, work floor binding, welfare record clamped, role extinct, group below floor, entry unfunded), so a resumed run logs as the original did, and the spawner's state per (region, role), which is result-affecting and not a diagnostic: the renewal carry and the "had an agent" bit (N-265, plan 1-11 D145). `spec-checkpoint.md` 4.2 places them by owner: the `BoundRun` in tag 4, the work floor and clamp runs in tag 7, the spawner's three runs and the renewal carry in tag 9; it lists no section for the run of "households unfed", which the checkpoint row decides. The tick counter is the ledger's. The row writes no encoding.
- **Update (2026-10-17, checkpoint/world edits):** consistent with `spec-checkpoint.md` 4.2 tags 4, 7 and 9 as edited. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Verified in Phase 1: the note already says the renewal carry and the "had an agent" bit are saved state and not a diagnostic run (D145). Left open for the checkpoint row.

### N-289 The slice's snapshot and `MarketState`

- **Areas:** OBS, INT
- **Raised:** 2026-10-17, `plans/plan-phase-1-11.md` (D146, D147).
- **Note:** The slice declares the snapshot sections it has state for (the header, regions, roles, markets, jobs, the boundary, pending and construction rows) and no module slice; a module appends its rows at the end. `MarketState` is declared in `snapshot.go` because the snapshot needs it now; INT's query `Market` (Stage 2) returns the same type, so the two cannot disagree.
- **Update (2026-10-20, plan 1-11 Phase 1):** Verified in Phase 1: the note matches D146 and D147. Left open until Phase 9.

### N-290 A module that adds a reduction cause appends its cause and its ledger reason together

- **Areas:** INT, POP, LED, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-11.md` (O10, Phase 7).
- **Note:** In Stage 1 the executor of the spawner's plan accepts a `DecideReduce` only for the causes the default spawner emits (`starved`, `insolvent`, ledger reason `starvation`, SPN 4.1); any other cause is refused (counted in `EntryLimited.Refused`, `Debug` "decision refused"). A reduction's ledger reason must name what it is, so no reason is guessed now. The INT change (consumer-driven removal) and the POP change (`population`, reason `deaths`) each append their cause constant and their ledger reason together in their own change (architecture.md 2.10.5, append-only), and extend the executor's accepted causes and its refusal-table case in the same change.
- **Update (2026-10-17, spawner edits):** the refusal `unsupported_cause` is written in `spec-spawner.md` 4.1.1 (line 180; checked last). Left open: INT and POP append their causes and reasons.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Verified in Phase 1: the note carries no WLD, OBS or ALL tag, so no disposition is owed here; Phase 7 builds the accepted causes and the refusal. Left open.

### N-294 The checkpoint change adds the digest to the off-only golden once

- **Areas:** CHK, REF, OBS, CODE
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md` (D172).
- **Note:** `testdata/golden/off/one.txt` of Stage 1 holds `spec-reference-chain.md` 8.5 items 1 to 5 and no item 6 (the SHA-256 of the checkpoint at tick `T`): no checkpoint exists before Stage 2 and no line is reserved. The checkpoint change adds item 6 and moves the off-only files once, with the testing.md 7.2 explanation. The checkpoint is a core file and not a module, so architecture.md 2.10.5 ("a module that is off leaves no trace") does not apply to it. After that change the files are held by the module rule again.
- **Update (2026-10-17, observability/roadmap edits):** applied in substance (REF 8.5 and the roadmap say item 6 is Stage 2). Left open for the Stage 2 checkpoint change.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read. No checkpoint item in Stage 1; the world adds none. Left open.
- **Update (2026-10-21, plan 1-12 Phase 1):** Verified against REF 8.5 item 6 ("added by the checkpoint change (Stage 2); until then the file ends with item 5") and roadmap-v1.md. No amendment. Left open for the Stage 2 checkpoint change.

### N-295 The packs of REF 7.5 and 8.2 the slice cannot run

- **Areas:** REF, CODE
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md`.
- **Note:** The invariants and determinism tables of REF 7.5 and 8.2 list `three`, `five`, `famine` (with the estates module part) and `one-demand`. They need regions, the `Estates` part and the demand module, so Stage 1 runs `one` and `second`, and `three` if the probe of Phase 5 of `plans/plan-phase-1-12.md` shows the Stage 1 world runs it (its regions do not trade, REF 16.2). The tests are table-driven: the stage that makes a pack buildable adds its row, and the rule of N-217 applies to the files it adds under `testdata/golden/all/`.
- **Update (2026-10-17, observability/roadmap edits):** REF 7.5 lists the packs; left open for the `three` probe of plan 1-12 Phase 5.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read and applied: Stage 1 runs `one` and `second` (with `two` and `warm-two` for the interleave test only); no `famine`, `five` or `three` in this row's acceptance or scenarios. Left open.
- **Update (2026-10-21, plan 1-12 Phase 1):** Verified: REF 7.5 carries the conditional paragraph (`three` only if the Phase 5 probe shows the Stage 1 world runs it). Left open for the probe.
- **Update (2026-10-21, plan 1-12 audit fixes, J1 probe outcome):** `three` IS in the invariants table (`invariantTable` of `invariants_test.go`: `three` with `base.toml`, `three.toml`, `warm-three.toml`), 400 ticks (`invTicks`), seeds 1 to 4. `go test -run 'TestInvariants_Packs/three' .` passed when run for this update (about 2 s, without `-race`), so the Stage 1 world runs `three`; the determinism and golden rows for `three` are not added. `five`, `famine` and `one-demand` still wait. Left open.

### N-296 S01 leg (b) and every visitor wait for `AddAgent` and `Direct`

- **Areas:** INT, REF, STB
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md` (O3).
- **Note:** `integration.go` (`AddAgent`, `Direct`, `SetYieldShocks`) is Stage 2. In Stage 1 a shock of `scripts/` is applied between steps by the world row's unexported yield-shock setter through `export_test.go`, an exception to FR-REF-021 and FR-STB-014 (ruled by the coordinator, reversible); the visitor of S01 (b) is a skipped subtest that carries this note as its ID. The Stage 2 integration change replaces the seam by the public calls and un-skips the leg.
- **Update (2026-10-17, observability/roadmap edits):** REF 7.1.3 states the seam. Left open until INT.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read: the shock seam is row 12's. Left open.
- **Update (2026-10-21, plan 1-12 Phase 1):** Verified: REF 7.1.3 and S01 state the seam and "waits for INT"; the wording is "ruled by the coordinator, reversible". Left open until INT.

### N-300 The first measurement of a `Step` of `one`

- **Areas:** SCL, OBS, CODE
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md` (Phase 8).
- **Note:** To be measured by Phase 8 and 9 of the plan: the allocations of a full `Step` of the pack `one` with the default logger and observer (the baseline constant of NFR-WLD-001 and NFR-OBS-002) and the time per tick. It is the measurement N-127 (the allocation per event) and N-222 (the cost of a reseed at every draw site) ask for at the first slice, and the first check of the cost model of `spec-scale.md` 6.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open for row 12. This row's allocation test asserts the call sites only and its benchmark reports (Phase 10).
- **Update (2026-10-21, plan 1-12 Phase 1):** Verified: raised empty, filled by Phases 8 and 9 ("to be measured"). Left open.
- **Update (2026-10-21, plan 1-12 audit fixes, measured baseline):** a full `Step` of `one` with the default logger and observer allocates 113 allocations under `-race -covermode=atomic -coverpkg=./...` (`stepAllocBaseline` in `allocation_test.go`). Under those flags the cardinality test (`TestCardinality_EventAttributesTakeClosedValues`) took about 171 s and the allocation test about 29 s. Time per tick is not recorded here. Left open for the time per tick and the cost-model check.

### N-276 `Validate`, `Warnings` and `NewWorld` each reach the programmes

- **Areas:** VIA, CFG, WLD, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (D125).
- **Note:** The viability programmes run in `Validate` (phase 6) and in `Warnings` (gated on a clean structure). `NewWorld` calls both, so the work is done twice. The unexported `checkViability` returns the errors and the warnings at once; the world row may call it once. Phase 7 measures the cost on `five`.
- **Update (2026-10-17, viability edits):** `spec-viability.md` 21 now has one `checkViability` returning errors and warnings together. Left open for the world row and the Phase 7 measurement.
- **Update (2026-10-18, plan 1-10 Phase 7):** measured on `five`: `Validate` 0.59 s, then `Warnings` 0.46 s (1.05 s in all; `BenchmarkValidateFive` runs the two together at 0.85 s a pair, 331 MB), against 1.37 s for one pooled `SolveSteadyState`. `NewWorld` calling both would pay the programmes twice, so the world row should call `checkViability` once. Left open for the world row.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply (A10): `NewWorld` calls `Validate` once and reads the warnings from the same outcome if row 10 exposes `checkViability`; otherwise it accepts the double work, which row 10's Phase 7 measured (0.59 s then 0.46 s on `five`). Left open for row 10.

### N-277 The programme with the wants needs a two-phase start

- **Areas:** VIA, DEM, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (A5).
- **Note:** With the demand module the goods rows are `>= lambda * D + W` and the right-hand side is negative, so the slice's simplex (every right-hand side non-negative) has no basic feasible start. `internal/steady` takes the fixed term per goods row and rejects a non-zero one until the demand row adds the first phase in its own PR. `one-demand` is DEM's pack.
- **Update (2026-10-17, viability edits):** read; unchanged. Left open for the demand row.

### N-279 `Lane` and `WithLanes` wait for the transport module

- **Areas:** VIA, TRN
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (D127).
- **Note:** Stage 1 supports only the free-trade lane set (a complete zero-cost set, collapsed to one goods row per commodity) through `WithFreeTrade`. `Lane` (`From`, `To`, `Commodity`, `Labour`, `LossBP`), `WithLanes` and `MaxLanes` are not published in Stage 1 (architecture.md 2.8); the transport module adds them with `transport.Lanes` (`spec-transport.md` 9.4, a lane of labour 0 and loss 0 being the free-trade case) in Stage 2 (ruled by the coordinator, reversible).
- **Update (2026-10-17, viability edits):** applied in `spec-viability.md` 9.4 (Stage 1 supports only free trade). Left open for the transport module (Stage 2).

### N-281 Three bounds of FR-CFG-046 were not built at Stage 0

- **Areas:** VIA, CFG
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (Phase 5).
- **Note:** `bounds.go` has `MaxSeedPrices` but not `MaxViabilityPivots`, `MaxViabilityCells` and `MaxLanes`. The solver row adds the first two (CFG 4.1 gives the values); `MaxLanes` bounds `WithLanes`, which Stage 1 does not publish, so it waits for the transport module (Stage 2). The Stage 0 plan's table that lists the viability rows as deferred also lists "group shares exceed 10 000" as deferred, but `warmRules` has it: correct the table.
- **Update (2026-10-18, plan 1-10 Phase 1):** `spec-config.md` 4.1 already gives `MaxViabilityPivots` 100 000 and `MaxViabilityCells` 4 000 000; Phase 5 adds the two constants and their tests. The Stage 0 table is corrected here, not in the plan: the sub-rule "shares of a group above 10 000" is built in `warmRules`. Left open.

### N-264 Stage 3 `skills.go` appends the queue rule and the `Apply` emission

- **Areas:** SPN, SKL
- **Raised:** 2026-10-17, `plans/plan-phase-1-9.md` (D111, coordinator ruling Q2).
- **Note:** The slice builds only the tool half of SKL: the inequality of `spec-skill-barriers.md` 2.5 (`ToolPrice`, `CapitalReturn` in the view) is in `internal/occupation`. `DecideApply` is not declared in the slice (SPN 3.3, plan 1-9 O10); a role with `TrainingTicks` above 0 is never entered by review, rung 1 or replacement, which is what SPN 6.2 says when no queue exists. `skills.go` (Stage 3) appends the queue-discounted rule of SKL 4.5 (`C_train`, `q_w`), the view fields `Surplus`, `QueueAhead`, `Trainers`, `MaxTrainees`, `InTraineeship`, the `Apply` emission for a weight-1 agent, the refusals `in_traineeship` and `needs_weight_one`, and the cause `traineeship`, all appended in module order.
- **Update (2026-10-17, spawner edits):** the tool half is written in `spec-spawner.md` and `spec-skill-barriers.md`; `DecideApply` is not declared in the slice, and SKL appends it with the `Apply` emission (O10, ruled (a)). Left open for Stage 3 `skills.go`.
- **Update (2026-10-18, plan 1-9 exit gate):** built as planned: `internal/occupation.CapitalOK` is the tool half, `DecideApply` is not declared, a role with `TrainingTicks` above 0 is never entered. Left open for Stage 3 `skills.go`.

### N-265 The renewal carry and the three runs are the world's state

- **Areas:** SPN, WLD, CHK
- **Raised:** 2026-10-17, `plans/plan-phase-1-9.md` (D118).
- **Note:** The default spawner has no state (SPN 14). The renewal carry per (region, role) is held by the world and read through `RoleView.RenewalCarry`; `Decide` sizes the renewal entries by `occupation.Renewal` on a copy and the executor calls the same function with the households it actually removed and stores the result (checkpoint tag 9, with the runs of "role extinct", "entry unfunded" and "group below floor" and the per (region, role) "had at least one agent" bit that FR-SPN-042 needs for "role extinct": `Extinct(agents, everHad, configured)`; SPN 14 does not list the bit yet, plan 1-9 Phase 1 adds it). The world row must add a `Guard:` test that the entries `Decide` planned equal the ones the executor's carry gives; a refusal of a default decision would break it and is a bug.
- **Update (2026-10-17, spawner edits):** applied for tag 9: `spec-spawner.md` 14 now lists the carry and the `everHad` bit, and `spec-checkpoint.md` 4.2 tag 9 saves them. Left open until the code merges (the `Guard:` test).
- **Update (2026-10-18, plan 1-9 exit gate):** built: `Decide` reads `RoleView.RenewalCarry`, `occupation.Renewal` and `Extinct(agents, everHad, configured)` exist, the `Guard:` test of the carry is still row 11's. Left open until the code merges.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply (D145, D192): `worldState` holds the renewal carry per (region, role), the "had an agent" bit and the three runs; the executor advances the carry with `occupation.Renewal`; the `Guard:` test the note asks for is in Phase 7. Left open for CHK.

### N-267 An agent in a role with no wage counts as earning nothing

- **Areas:** SPN, STB
- **Raised:** 2026-10-17, `plans/plan-phase-1-9.md` (D115).
- **Note:** SPN 6.2 needs `E_o` for the hysteresis comparison and gives no value when the agent's own role has no observed wage. The plan reads it as 0, so a viable role with a signal beats it and a failing role can be left; a destination still needs a signal, so a cold start with no wage anywhere moves nobody. The risk is herding into the first role that gets a wage; scenarios S06 and S26 decide, and the entry share and the cooldown are the dampers. A quotient beyond `int64` saturates by sign (D115), so no role silently loses its signal. The plan order of SPN 10.2 takes reductions before switches, so an agent at `RemovePrivation` with a viable role that waits behind the entry cap is reduced, not switched (SPN 6.4 can be read the other way); S14 and S27 decide. The reviewer's destination filter has no viability term, as SPN 6.2 reads (plan 1-9 O8). Plan 1-9 D115, D116, O8 and O9 are FINAL (ruled by the coordinator, reversible): an unrated own role earns nothing, so leaving a role requires evidence of a better one; reviewers are not held to the `EntryRatio` gate; the kind-2 replacements are admitted as a closed-form batch, tested against a per-household oracle. Leave open until S06, S14, S26 and S27 have run.
- **Update (2026-10-17, spawner edits):** the reduction-before-switch order is written in `spec-spawner.md` 6.4 and 10.2 (applied). Left open until S06, S14, S26 and S27 have run.
- **Update (2026-10-18, plan 1-9 exit gate):** built as ruled (own role without a signal is `E_o` 0, reductions before switches, no viability term for reviewers). Left open until S06, S14, S26 and S27 have run.

### N-268 Module fields of the spawner's view and decision are appended with their module

- **Areas:** SPN, SET, SUB, POP, MIG, SKL
- **Raised:** 2026-10-17, `plans/plan-phase-1-9.md` (D112).
- **Note:** The slice view and decision hold only what the slice uses. Appended by their module, in module order: SET `Role.Needs` (folded into `Closed`), `Headroom`, `NoLimit` and `EntryLimited.RoleCeiling`; SUB `OwnUse` (read-only) and the reset of `OwnUseJobs`; POP `PopBirths`, `PopDeaths`, the births and deaths entries and the effective `Renewal` of 0; MIG the refusal `on_journey` and the `RoleSince` of an arrival; SKL as N-264. The `Cause` constants `population`, `births` and `traineeship` are appended after the existing six in `event.go`, and the reasons `in_traineeship`, `on_journey`, `needs_weight_one` after the six of the slice; nothing is inserted (architecture.md 2.10.5).
- **Update (2026-10-18, plan 1-9 exit gate):** the slice view, decision, `Refusal` and `Facts` carry only slice fields; module fields are appended with their module. Left open.

### N-305 When SKL appends `DecideApply`, SPN 6.2 and 6.3.0 must be re-read

- **Areas:** SPN, SKL
- **Raised:** 2026-10-17, WP5 report of the owed-edits job.
- **Note:** `spec-spawner.md` 6.2's training rule (a role with `TrainingTicks` above 0 is never entered) and the counts of 6.3.0 hold only while no queue exists. The change that appends `DecideApply` (N-264) re-reads both and edits them in the same change.

### N-306 The census formula is pinned against `one`: the work share cancels

- **Areas:** VIA, REF
- **Raised:** 2026-10-17, WP6 report of the owed-edits job.
- **Note:** `spec-viability.md` 5.2 (line 489) says the available work share divides every term alike and cancels, so it is not in the census formula. REF 4.3 and 4.4.1 should say the same, so the census of `one` is read the same way in both specs. Check them when REF is next edited.
- **Update (2026-10-18, plan 1-10 Phase 1):** read: the census formula is written in `spec-viability.md` 5.2 against `one` (deliverable 3); Phase 6 pins it in a test. Left open.
- **Update (2026-10-21, plan 1-12 Phase 1):** Applied: REF 4.3 now says the available work share cancels and is not in the census formula (`spec-viability.md` 5.2). The census of `one` is read the same way in both specs; Phase 6 of the row that pins it is row 10's. Leave open until row 10's pin is merged, then close.

### N-309 What the estate row (plan 1-8) leaves to later rows

- **Areas:** EST, WLD, CHK, CODE, OBS
- **Raised:** 2026-10-18, the audit of `plans/plan-phase-1-8.md` (`estates.go`, built and tested without a world).
- **Note:** `estates.go` is a set of unexported functions with no caller until the world row, so these requirements are only partly covered and are completed there; each needs a test in its owner's row. (1) **World row (11):** FR-EST-010 and 011 (no `AgentSpawned`, no snapshot row, no strategy call, belief or outcome for an estate, and no agent event whose subject is an estate; the row tests only `regionOf`); FR-EST-002 (the estate that receives is the one of the region of the agent's account); FR-EST-004 (the estates open before any household, WLD 5.2 order 4); FR-EST-012 (the asks are placed at P3.4 before any market clears, with the tie-break key; the row tests only that the order carries the estate's ID); FR-EST-013 (`Regions[].MoneyPool`, `Markets[].HeldByEstate` and `EntryBudget` in the snapshot); FR-EST-014 (`w * Role.StartMoney` and "an entrant gets no goods" are the executor's, with the whole-household budget); FR-EST-015 (the `initial` money of the initial agents, and the map of `WarmPool.Region`, a string, to a `RegionID` before `seedPool`); FR-EST-021 (emit `AgentBankrupted` then `EstateOpened` behind the observer flag; `EstateOpened.Balance` is read as `pool` after `receive`, plan open item O6, and `GoodsListed` counts lots, N-255). The call list is N-258 and N-266. (2) **Checkpoint row (Stage 2):** build tag 8 from `rows` and `restore`, wrap `errListingRows` in `ErrCheckpointCorrupt`, and make the cross-check of restore step 7 (FR-CHK-023) call them (N-256). (3) **The `Estates` module part (Stage 2, after MON):** `Region.PoolRelease` and `Region.ReliefBelow`, relief at P1.6 and `ReliefPaid`, the pool as a money-flow target with `ErrModuleDependency`, FR-EST-017 to 020, and the module fields of FR-EST-023; they append to `estates.go` after the marked comment line at its end. (4) **Other modules:** FR-EST-003 (a traveller who dies in transit, a diverted cargo) is TRN's and MER's; money the estate receives as creditor or title holder is CRD's and LND's; the tool-sale rule of SKL 2.7 beyond the first price is SKL's. (5) **After the first golden file:** the first price, the markdown and the keep-the-price rule of a lot that gains units (D102, D104; N-254) enter `testdata/golden/` at the end of Stage 1, and a later change is a behaviour change with the explanation of testing.md 7.2.
- **Update (2026-10-18, test audit of the branch):** the audit against testing.md found the row tested at the unit level only, as the plan says (`estates.go` is 100% covered, every function; root package 98.8%), and left these test debts for later rows. Each needs a test in its owner's row; none can be written before there is a `World`. (a) **Public-API integration tests (testing.md 2.3), world row (11):** the only public-API test of the row is `TestEstates_PackOneAccountIDs` (FR-EST-004, 005). FR-EST-006 (a bid from an estate is `side_not_allowed`), 007 (one ask per held lot, a rejected ask is not an ask), 009 (the markdown at P5.3) and 013 (the proceeds are a `trade` transfer into the pool, `MoneyPool` equals the account) are consumer-visible and need a test through `NewWorld` and `Step` with a recording observer, beside the unit tests that exist. (b) **Restore equals uninterrupted (FR-EST-024), checkpoint row (Stage 2):** a restored world must list and price the same lots as an uninterrupted one; the unit tests cover `rows` and `restore` only (N-256). (c) **Mutation pass:** 22 hand-made mutants of `estates.go` were run; the two that survived were an equivalent one (`<=` against `<` at the floor in `markedDown`) and the clearing of the `asked` mark in `asks`, now covered by `TestEstateAsks_MarksOfTheLastTickAreCleared`. `gomu` was not run (Docker only, testing.md 9.4); it is the periodic pass for the core packages. (d) **Fuzz seeds:** `testdata/fuzz/FuzzPriceRules`, `FuzzReconcile` and `FuzzRestore` hold hand-written seeds, not crashers, so testing.md 7.5's `Fix:` rule does not apply; they must be committed with the row. (e) **The invariant branches** of `estates.go` (an unreadable account, a belief step out of bound, an overflowing first price, an admitted ask with no listing) are `Guard:` tests (`estates_errors_internal_test.go`), so no coverage exclusion under testing.md 9.5 is needed. Left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The world-row items (1) are spread over Phases 3 to 10 (the estate opened before any household, the asks at P3.4, `MoneyPool`, `HeldByEstate` and `EntryBudget` in the snapshot, `AgentBankrupted` then `EstateOpened` behind the observer flag); the plan names no test of FR-EST-006, 007 and 009 (a bid from an estate is refused, one ask per held lot, the markdown at P5.3) through `NewWorld` and `Step`, which Phase 10 should carry. Items (2) to (5) are not this row's.
- **Update (2026-10-21, plan 1-12 Phase 1):** Leave open: item (5), the first price, the markdown and the keep-the-price rule, enter `testdata/golden/off/` in Phase 3 with this row's golden; items (1) to (4) are not this row's.

### N-310 What the observability review of plan 1-8 leaves undone

- **Areas:** EST, OBS, WLD, CODE
- **Raised:** 2026-10-18, the observability review of the branch of plan 1-8 (`estates.go`, N-309 item 6).
- **Note:** The catalogue and spec rows for `economysim.estate.sold`, `economysim.estate.at_floor` and `Markets[].EstateAtFloor` are written, and nothing can emit them until the world, the snapshot and `otelobserver` exist. Owed, each with its test in the row that builds it: (1) **World/snapshot row (11):** declare `EstateAtFloor` in `snapshot.go` and fill it as the units of the estate's lots whose row in `estates.listed` is at the resolution floor (`resolutionFloor`); it is 0 with no estate stock and never above `HeldByEstate`; a test builds a world, lets an estate lot fall to the floor and compares the field; the field is a function of saved state (N-125), so the restore test (FR-OBS-043) must cover it. (2) **`otelobserver` (Stage 2):** feed `estate.sold` from `TradeExecuted` where `Seller` is below the region count (the test of `spec-observability.md` 4.3.3 item 2, `estates.regionOf`), record `estate.at_floor` from the snapshot, and add both to the adapter's cardinality test (testing 8.1: commodity and region only; about 300 series, 6.5.2). (3) **Pool overflow through a step:** the world row needs a test that an `ErrBalanceOverflow` of an estate pool fails the step (`TickEnded` with `Failed`, `economysim.world.step.failed`) and that the error text names the region and the account, as `refused` now does; the unit test covers `refused` only. (4) **Allocation test:** `estates.asks` and `markdown` have a no-allocation test in isolation (NFR-EST-001); the full-step allocation limit of observability 8.2 includes them only once the world row calls them. (5) **Reading `economysim.market.unmatched`:** its ask side includes the estate's unsold lots; if the first dashboards show it misleads, split the series by a holder attribute (doubling its ask series) instead of subtracting `HeldByEstate`; decide with data from the first golden run, not now. (6) **Not done in this review:** `doc/notes.md` was not read in full before the spec edits (only the notes tagged OBS and EST that the edits touch, and N-125, N-126, N-127, N-289 and N-309); the next change to `spec-observability.md` should read it whole.
- **Update (2026-10-20, observability review of the branch of plan 1-11):** item (1) is applied: `Markets[].EstateAtFloor` is declared in `snapshot.go` and filled in `snapRegion` from `estates.listed` (`estates.atFloor`: a held lot whose row price is at or below `resolutionFloor`; a lot with no row is not at the floor), tested by `TestSnapshot_EstateAtFloorCountsUnitsListedAtTheResolutionFloor` (fails with the check broken). The restore-equality half of the test (FR-OBS-043) is the checkpoint row's, since the field is derived from the saved listing rows (tag 8). Items (2) to (6) are unchanged; item (3), the pool-overflow step test, is still not in Phase 10. Left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. Item (1), `Markets[].EstateAtFloor` filled from `estates.listed` against the resolution floor, and item (3), the pool-overflow step test, are not in the plan's Phase 9 and Phase 10 lists: they must be added to Phase 9 (the field and its test) and Phase 10 (the failed step) when those phases are built. Items (2), (4) and (5) are not this row's.
- **Update (2026-10-21, plan 1-12 Phase 1):** Leave open: item (5) asks for the first golden run's data on `economysim.market.unmatched`; Phase 3 reports the unmatched ask units with `HeldByEstate` beside them. Items (1) to (4) and (6) are not this row's.

### N-311 Per-commodity price uniqueness is not detected by the solver

- **Areas:** VIA, CODE
- **Raised:** 2026-10-18, `plans/plan-phase-1-10.md` (Phase 6 fix, FR-VIA-026).
- **Note:** FR-VIA-026 and VIA 8.2.4 give a commodity whose price is not unique the reference price and list it in `UnpricedCommodities`. `steady.Outcome` carries only the global `Degenerate` and `Alternative` flags (FR-VIA-019), and telling which duals are non-unique needs a min and a max of each price over the optimal dual face (2n further programmes). Phase 6 builds the dual-0 half only; a non-unique non-zero dual is used as the price and the report's `Degenerate` and `AlternativeOptima` flags are the warning. Reversible: add a per-commodity flag to `steady.Outcome` and the reference-price rule in `prices()`.
- **Update (2026-10-21, plan 1-12 Phase 1):** Read: not this row's; the pack `one` report in `expected/one.toml` (Phase 6) lists `unpriced` as the solver gives it. Left open.

### N-312 The engine collapses pack `one` under its own rules: findings of the row 12 scenarios

- **Areas:** REF, STB, CODE
- **Raised:** 2026-10-21, the audit fixes of `plans/plan-phase-1-12.md` (Phases 5, 7 and 8).
- **Note:** None of this is row 12's code to fix; row 12 only adds tests, fixtures and docs, and the owner's ruling is needed on each point. (1) In the recorded golden `testdata/golden/off/one.txt` the households of `one` fall from about 600 to 81 by tick 400, and every S01, S04 and S16 run ends with 0 households. (2) The S01 O check (the mean fuel price over the shock, ticks 400 to 459, above P0, the mean over 300 to 399) fails for seeds 1, 2, 3 and 5: the shock mean is 22, 21, 94 and 104 against P0 of 39, 41, 102 and 116. (3) The price-bounds check of S01 and S04 uses the loosest maker interval (`marketInterval` in `scenario_helpers_test.go`) and exempts the ticks where an estate sold at the tick the published price was set (an estate's ask is priced by the markdown of EST 2.7.3, not by a belief); the strict per-maker violations are only reported (564 market-ticks on seed 1). Whether FR-STB-008 is the loose or the strict interval, and whether an estate sale may leave the interval, is the owner's ruling. (4) A rendering detail found on the way: the snapshot line of `events-one.txt` carries `Modules=[]` because the renderer omits a field only at `IsZero`, and a non-nil empty slice is not; treating an empty slice as zero (OBS 9.5.2) would move that one golden line, so it was not changed (testing.md 7.2).
- **Update (2026-10-21):** S01 `unmet` was 30 000 bp on seed 7 because the numerator (households unfed at the consumption phase) and the denominator (the snapshot after the exits of the tick) counted different things; the test now divides by the households at the start of each tick (seed 7: 10 000 bp, every household unfed), which is a fix to the test, not to the engine.
- **Update (2026-10-21, observability review of the branch of plan 1-12):** two metrics were proposed for the two reconstructions the scenario tests made, and both were dropped after reading the code. (a) A `MarketCleared` field for "the price was set by an estate" is ill-defined: `clearing.Report` sets the price as the volume-weighted mean of all trades of the tick, and `economysim.estate.sold` with `economysim.market.price_age` already gives the join. (b) An `economysim.household.exposed` metric cannot come from `UpkeepShortfall.Households`, which is absent when all are fed. Both traps are now written in `observability-and-logging.md` 5.3.1 (the unfed share divides by `household.count` at the end of the previous tick; the estate join). Point (3) of the note stays the owner's ruling.
- **Update (2026-10-21, owner's ruling on point 3):** applied in `spec-stability.md` FR-STB-008 and `spec-estates.md` 2.7.3: the interval is the loosest of any agent (FR-BEL-004), and in a tick in which an estate sold in that market the published price is bounded below by the resolution floor only, because an estate's ask is priced by the markdown. The strict per-maker violations stay a report, not an assertion. Points (1), (2) and (4) are open.
- **Update (2026-10-21, trace of the grower collapse, seed 1 of `one`):** the grower gap outside spring is not the cause. The warm start holds 164 930 staple at tick 0 (253 growers, each `grow_staple` 1 800 units in 60 ticks, about 30 a tick, against an upkeep of 2 a household a tick, so the census as written can supply about 6 times the 1 200 a tick that 600 households eat). The price falls 87 to 10 by tick 200 while the stock is still 18 000 to 158 000; the grower households fall 253 to 1 by tick 180 (insolvency or role change), nobody makes staple after tick 180, the stock is 0 at tick 220 and the cooks go unfed (138 of 138). Open for the owner of REF: whether the census of `one` (253 growers) and the 1 800 yield are consistent (the solver's surplus is 25 119 bp, which does not say), and why no entrant takes the grower role once the price recovers (tool_a, seed, the spring-only start).
- **Update (2026-10-21, collapse investigation):** point (1) is confirmed on every pack, not only `one`: warm `five` and `two` reach 0 households by tick about 500, `three` about 600. With the failure ladder at its maximum bound nobody is removed, yet about 450 of 600 households stay unfed for 300 ticks, the mineral, refining and tool jobs stop by tick 200 and the staple price falls to 10, so it is not a shortage of money. The cook stall (the trade side did not buy the upkeep that the work side holds back) is fixed: `Job_c` counts `U_c` for an input that is also upkeep, and the `one` golden ends at 132 households instead of 81, with five S01 seeds ending above 0. What is left: growers have no recipe outside the spring season (`ZeroYield`) for about 60 ticks and live on the staple stock of the warm start; with about 150 000 units in stock the staple price falls from 96 to 12, they become insolvent and `grow_staple` stops. The S01 O check now fails on all 10 seeds (the fuel price after the shock is about half of P0, because fuel stays in glut), where it failed on 4. Raising `tool_b` durability, the shelf life of `prepared`, the privation thresholds, `cover_ticks`, `reserve_ticks`, `input_jobs` or the output of `prepare_basic`, or removing the tool requirements, only delayed the collapse (3 seeds each).
- **Update (2026-10-21, anchor fix):** two defects of the cost anchor fed the staple and fuel crash and are fixed (`spec-production.md` 9.3, FR-PRO-054 and 055): (a) a seasonal recipe was costed at its factor of the current season, so `grow_staple` gave staple no anchor outside spring; it is now costed at its best season; (b) the anchor followed the observed wage down in a crash; `World.AnchorHold` (default 0, 5 000 in `base.toml`) holds it at a share of the starting wage. On `one`, seeds 1 to 3, households at ticks 100, 300 and 600: before 501, 196, 101 / 479, 217, 65 / 541, 148, 86; after 557, 193, 45 / 538, 182, 99 / 544, 311, 20. The floors hold (staple about 40 instead of 10 after tick 200, fuel about 65 instead of 16) and the collapse is not cured: see N-314. Points (1), (2) and (4) stay open. The probes of N-314 (2026-10-21, income and self-provisioning) qualify the trace of the grower collapse above: at the reference prices a grower earns 45% of its upkeep because the census gives it 253 households for a staple that is 19% of the basket, not only because of the spring gap.

### N-314 The slice's money has no way back from the cooks

- **Areas:** DEM, REF, STB, MON
- **Raised:** 2026-10-21, the collapse investigation of pack `one` (seeds 1 to 3, with probes outside the repository).
- **Note:** In the slice a household spends only on its upkeep and its recipe inputs, and nothing sends money back to the other roles. On `one` the 136 cook households sell `prepared`, 63% of the upkeep at the reference prices, and by tick 200 hold about 99.5% of the 24.6 million of money (11 360 a household at the start, about 180 000 at tick 200, from the cumulative trade flows), while every other role is at 0 to 600 a household. With `entry_share` and `exit_share` at 0, so that nobody moves or dies, about 460 of 600 households stay unfed from tick 300 and the staple market has no bid for 350 000 units at the price floor. The census is deliberately 1.8 to 2.5 times what a basket needs (`lambda*` 2.512, `spec-reference-chain.md` 4.4.1) and the solver's wage is a price per working tick, so at the reference prices a grower, who can start two jobs a year, earns about 300 000 a year against an upkeep of about 365 000. A smaller census (269 households), a census weighted by role income, four times the grower money, twice the `grow_staple` yield, year-round growing, no tool requirements, a doubled `prepare_basic` and the anchor fixes of N-312 each left the economy collapsing or only delayed it. The owner pulled the demand module forward (2026-10-21) as the way to return money. For whoever builds it: cooks need something to buy and a role that makes it; the warm start and the census must be solved again with the module on (N-277); run S04 with it on and off; and the pinned hash of a pack that turns it on is not the all-off pin.
- **Update (2026-10-21, income, floor and self-provisioning probes, outside the repository, seed 1 unless stated):** (a) The warm-start prices are the solver's dual: every recipe pays about 2 512 per working tick (`prepare` 2 515, `cut_fuel` 2 506, `dig_mineral` 2 511, `refine` 2 513, `make_tool_*` 2 512, `grow_staple` 2 500). At those prices, with one basket per household a tick, income per household-tick is about 1 380 for cook, digger, forester, maker and refiner (138% of the 999 upkeep) and about 445 for a grower (45%): staple is 19% of the basket's value and growers are 42% of the households, so the role's pool (about 112 000 a tick) is shared by 253. Starts read the season at the start tick, so two 60-tick jobs fit in the 92-day spring and about 137 growers would supply the staple; at full use they would earn about 82% of upkeep (two jobs of 2 500 a tick over 365 ticks), so the census of 253 and the working share both matter. The first reading of the probe assumed one job a year and is corrected here. (b) The tool chain dies before the cooks stop wanting tools: `prepare` is 35%, 22% and 4% of the cook executions in ticks 0 to 50, 50 to 100 and 100 to 150, then none; makers go 55, 11, 0 (ticks 0, 50, 100); at tick 50 `tool_b` sells at 9 796 (reference 17 105) with refined at 2 804 (3 190), a margin of about 460 a tick against an upkeep of 999. (c) A strategy that raises every staple ask to a floor: at 96 the economy ends at 127 households at tick 600 (45 without it), growers still 135 of 322 unfed at tick 100; at 190 it is gone by tick 450. (d) Growers given `cut_fuel`: 57 households at tick 600, no cure. (e) Growers given `cut_fuel` and `prepare_basic` (a data-only stand-in for self-provisioning, not SUB): 253 grower households stay fed to tick 600 on seeds 1 and 3 and 50 remain on seed 2; the cooks, foresters and tool chain are extinct by tick 150 because no one buys from them beyond upkeep. SUB as written cannot express home baking (an own-use recipe has no inputs, `spec-subsistence.md` 3.4.1, open question 3) and covers staple and fuel only. For whoever fixes the pack or builds the demand module: the census should follow the labour each role needs (each role works about 40% of its ticks at `lambda*` 2.512), a seasonal crop wants a positive yield outside spring or a second recipe, the tool chain needs a customer beyond the cooks' wear, and money must return to the non-farm roles; none of the single changes tried (a)-(e) cures the collapse.
- **Update (2026-10-22, one pack-fix attempt, outside the repository, in Go and not in the TOML files):** `grow_staple` given a positive yield in every season (spring 10 000, summer and autumn 6 000, winter 3 000), the households of each group set to the solver's balanced census for those yields, and the warm start regenerated by two `SolveSteadyState` passes. `lambda*` falls from 2.512 to 2.217 and the balanced census is cook 166, digger 38, forester 110, grower 170, maker 71, refiner 45 (growers 253 before). Households at ticks 100, 300 and 600: seed 1 527, 303, 94; seed 2 492, 284, 99; seed 3 506, 336, 117 (baseline 557, 193, 45 / 538, 182, 99 / 544, 311, 20). It delays the collapse and does not cure it: on seed 1 the tool chain is gone by tick 300, growers fall from 152 at tick 100 to 55 at tick 300, and only unfed cooks and foresters remain at tick 600. Income per role was not measured for this variant. Rebalancing the data alone does not give a stable pack, so the pack is left as it is and the rebalance is owed with the demand module and the two-phase start (N-277), which re-solve the census and the warm start anyway.

### N-315 Under -race the scenario and determinism tests run a reduced seed set

- **Areas:** CODE, ALL
- **Raised:** 2026-10-23, plan 2-1 Phase 2.
- **Note:** Under `-race` the scenario tests run `scenarioSeedCount()` seeds (`raceSeeds`, now 1), `TestInvariants_Packs` runs `invariantSeeds(tc.seeds)` and `TestDeterminism_SameBytes` runs `detCases(true)`; the full set runs without `-race`; `-update` is refused under `-race`; the gate is two runs, R (`go test -race -timeout 60m ./...`, reduced seeds) and F (the non-race atomic-coverage run, all seeds; `testing.md` 9.1). A row that adds a long scenario uses `scenarioSeedCount()`, a row of `invariantTable` calls `invariantSeeds`, and a test that skips under `raceEnabled` starts with a prefix of `gateFPrefixes`. **Settled (owner, 2026-10-23, D231):** with atomic coverage under `-race` gate R took 48 min (one seed) to 58 min (two) against the 30 min budget, because fixed-cost tests dominate; R was split from the coverage run: R is 160 s, F 15 min. The allocation test runs under `-race` alone (70 allocations per `Step` with or without coverage; the constant 113 is loose). Close this note when a row has used `scenarioSeedCount()` and `invariantSeeds`.

## Read receipt

If you are an agent about to write or change a spec, you quote the tag below in
your first message about that work, as required by
[AGENTS.md](../AGENTS.md). Reaching it means you have read this file in full. Do
not guess it or reuse one from memory.

When you edit this file (add, update or remove a note), replace the tag with a
fresh random value in the same change (for example
`od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`). A stale tag in an agent's
message means it read an older version, and may have missed a note.

NOTES-TAG: 2cab1798746dfa020da8f0e9

