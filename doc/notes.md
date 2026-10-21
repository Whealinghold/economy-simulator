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

- **apply** it (the spec now does what the note asks), then close the note;
- **reject** it, with the reason, then close the note;
- **leave it open**, if it is not yours to settle yet, and add what you learned.

**While working.** If you find something a future spec or a code change must take
into account, add a note in the same change. Do not leave it only in a chat or a
PR description.

**Rules for entries.**

- One note, one topic. Keep it to a few lines and link instead of copying.
- Tag it with the area codes of the specs that must read it. `ALL` is for every
  spec. `CODE` is for the first code change that touches the area. `EVD` is for
  the evidence file.
- Give a stable ID `N-NNN`. IDs are never reused. Never delete a note: close it,
  moving it to "Closed notes" with the resolution and where it was settled.
- Give the date it was raised (the format is `YYYY-MM-DD`) and where, so a reader
  can check it is still true. A note that names a section or a requirement may go
  stale when that changes: fix the pointer when you see it is wrong.
- Update the index in the same change, and replace the read-receipt tag at the
  end of the file with a fresh random value.

## Index by area code

| Area | Open notes |
|------|-----------|
| ALL | N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217, N-235, N-304 |
| AGT | N-043, N-067, N-087, N-101, N-111, N-137, N-139, N-142, N-144, N-153, N-156, N-164, N-188, N-196, N-214, N-228, N-232, N-236, N-237, N-239, N-280, N-266, N-244, N-245, N-246, N-247, N-308 |
| BEL | N-042, N-043, N-230, N-245 |
| CAL | N-030 |
| CAP | N-140, N-155, N-156, N-170, N-171, N-172, N-173, N-174, N-175, N-191, N-194, N-196, N-197, N-200, N-205 |
| CFG | N-016, N-061, N-075, N-093, N-123, N-132, N-165, N-182, N-218, N-221, N-276, N-281, N-284, N-299 |
| CHK | N-125, N-128, N-129, N-202, N-204, N-214, N-215, N-225, N-237, N-254, N-256, N-288, N-294, N-265, N-244, N-247, N-309 |
| CLB | N-015, N-124, N-134, N-135, N-158, N-162, N-169, N-199, N-207, N-208, N-214, N-220 |
| CLR | N-113, N-137, N-238, N-239, N-240, N-241, N-242 |
| CODE | N-019, N-089, N-127, N-198, N-214, N-215, N-216, N-217, N-218, N-222, N-224, N-228, N-232, N-243, N-258, N-274, N-276, N-277, N-284, N-285, N-286, N-294, N-295, N-297, N-300, N-244, N-245, N-247, N-254, N-290, N-304, N-309, N-310, N-311 |
| CRD | N-107, N-152, N-156, N-164, N-165, N-166, N-167, N-168, N-169, N-187, N-195, N-205, N-209, N-212, N-227 |
| DEM | N-080, N-089, N-145, N-226, N-228, N-277 |
| EMP | N-144, N-147, N-187, N-190, N-191, N-194 |
| EST | N-082, N-096, N-119, N-167, N-254, N-256, N-258, N-266, N-309, N-310 |
| EVD | N-018, N-133 |
| EXP | N-015, N-124, N-127, N-129, N-134, N-162, N-179, N-203, N-206, N-207, N-211, N-214, N-220 |
| EXT | N-108, N-112, N-113, N-214 |
| FIS | N-107, N-131, N-140, N-159, N-160, N-161, N-163, N-172, N-173, N-174, N-183, N-184, N-185, N-186, N-187, N-188, N-189, N-190, N-204 |
| GDS | N-031, N-039 |
| IND | N-130, N-159, N-178, N-201, N-206, N-208 |
| INF | N-110, N-112, N-156, N-157, N-170, N-214 |
| INS | N-176, N-179, N-213, N-227 |
| INT | N-090, N-129, N-130, N-131, N-136, N-143, N-161, N-164, N-177, N-194, N-204, N-205, N-289, N-296, N-290 |
| LED | N-021, N-125, N-130, N-159, N-225, N-226, N-233, N-290 |
| LND | N-140, N-156, N-160, N-166, N-187, N-190, N-191, N-205 |
| MER | N-120, N-122, N-136, N-173, N-176, N-180, N-189, N-190, N-210, N-214 |
| MIG | N-121, N-122, N-136, N-143, N-150, N-157, N-170, N-171, N-181, N-214, N-268, N-244 |
| MON | N-096, N-107, N-122, N-163, N-181, N-183, N-204 |
| OBS | N-125, N-127, N-130, N-159, N-242, N-288, N-289, N-294, N-300, N-308, N-309, N-310 |
| ORG | N-140, N-152, N-155, N-160, N-161, N-172, N-173, N-205, N-244 |
| PLC | N-131, N-140, N-160, N-177, N-190, N-194, N-214 |
| POP | N-096, N-097, N-109, N-132, N-153, N-192, N-268, N-244, N-290 |
| PRO | N-030, N-039, N-043, N-090, N-118, N-130, N-137, N-142, N-191, N-200, N-232, N-233, N-234, N-237, N-242, N-243, N-308 |
| REF | N-027, N-030, N-089, N-092, N-114, N-123, N-135, N-145, N-146, N-168, N-274, N-294, N-295, N-296, N-298, N-299, N-306, N-307 |
| REG | N-150, N-175, N-204 |
| RSK | N-140, N-155, N-161, N-176, N-205, N-211, N-213, N-214, N-226 |
| SAV | N-145, N-156, N-195, N-197 |
| SCL | N-030, N-198, N-222, N-234, N-300 |
| SET | N-140, N-192, N-268 |
| SKL | N-039, N-071, N-090, N-110, N-138, N-149, N-193, N-264, N-268, N-244, N-305 |
| SPN | N-043, N-089, N-096, N-118, N-122, N-138, N-148, N-153, N-172, N-192, N-193, N-264, N-265, N-266, N-267, N-268, N-305 |
| STB | N-028, N-032, N-039, N-042, N-043, N-045, N-067, N-086, N-087, N-092, N-099, N-113, N-122, N-124, N-151, N-154, N-162, N-169, N-179, N-210, N-211, N-214, N-296, N-267 |
| STO | N-155, N-160, N-166, N-181, N-186, N-191, N-194, N-205, N-221, N-244 |
| STY | N-015, N-027, N-030, N-084, N-085, N-086, N-092, N-113, N-123, N-132, N-133, N-135, N-145, N-151, N-162, N-168, N-177, N-178, N-199, N-201, N-203, N-206, N-207 |
| SUB | N-121, N-228, N-268, N-244 |
| TEC | N-214 |
| TRN | N-120, N-130, N-136, N-137, N-142, N-143, N-155, N-170, N-174, N-175, N-176, N-180, N-184, N-189, N-194, N-204, N-226, N-279 |
| VIA | N-015, N-061, N-071, N-080, N-085, N-093, N-109, N-114, N-123, N-134, N-146, N-158, N-165, N-180, N-274, N-276, N-277, N-279, N-280, N-281, N-306, N-307, N-311 |
| WEL | N-016, N-028, N-099, N-118, N-119, N-130, N-147, N-148, N-171, N-183, N-214, N-234 |
| WLD | N-124, N-125, N-144, N-154, N-204, N-214, N-216, N-226, N-230, N-239, N-240, N-242, N-258, N-276, N-280, N-284, N-285, N-286, N-288, N-297, N-265, N-266, N-246, N-308, N-309, N-310 |

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


### N-031 What the calendar hands to other specs

- **Areas:** GDS
- **Raised:** 2026-10-10, `spec-calendar.md`.
- **Note:**
  - RES and PRO: a season is the index `Calendar.At(tick).Season`, or -1 for none.
    A yield or a recipe condition by season is a table keyed by season ID; no logic
    names a season (architecture.md 9.1). A recipe whose production time exceeds its
    season cannot complete, and validation does not see it (`spec-calendar.md` 9);
    PRO or VIA should.
  - WLD: compute the moment once per tick, before orders are placed. It is derived,
    not state, and adds nothing to a checkpoint (`spec-calendar.md` 6.5).
  - OBS: decide whether the world emits a season event and whether the snapshot
    carries the moment (`spec-calendar.md` open question 4). Events already carry
    `Tick`.
  - INT: a consumer converts ticks to dates with `Calendar.At`; a shock by season
    (a failed harvest) needs the season index.
  - GDS and the other duration fields (`ShelfLife`, `TrainingTicks`): durations are
    in ticks and are not rescaled when the calendar changes, so a pack for another
    planet restates them (`spec-calendar.md` 6.1).
- **Update (2026-10-10):** PRO reads the season and the yield at **job start** (`spec-production.md` 6.4), so a recipe longer than its season completes; "cannot complete" does not arise for production and needs no validation. A harvest window is a perishable intermediate. INT: the yield-shock slot of 6.8 is where a seasonal failed harvest goes. RES, WLD, OBS and GDS parts unchanged.
- **Update (2026-10-10, VIA):** VIA reads the calendar as periodic (the season of tick `-o` is that of `(-o) mod TicksPerYear`) for the virtual start of initial jobs (`spec-viability.md` 8.5.2) and adds check V4 (supply gap against shelf life) for seasonal storage (4.5). Neither changes CAL; CAL may want to state the periodic extension (`At` rejects negative ticks).
- **Update (2026-10-10, CAL):** `spec-calendar.md` 4.5.1 adds the periodic helper `Calendar.SeasonAt(tick)` (the season of `floorMod(tick, TicksPerYear)`), separate from `At`; `spec-viability.md` 8.5.2 uses it. The "season shorter than a delay" entry (9) does not arise for production.
- **Update (2026-10-14, plan 1-2 Phase 6):** applied in code: `Calendar.At`, `SeasonAt` and `TicksPerYear` (`calendar.go`) and the day table the world reads once per tick (the world row, P1.1); `SeasonAt` is what VIA 8.5.2 calls. **Still open** (not this row): WLD computing the moment once per tick, OBS deciding on a season event or the moment in the snapshot.
- **Update (2026-10-10, RES):** applied for RES in `spec-resources.md` 4.5 and 7.3: the season of a tick is the index `Calendar.At(tick).Season`, the yield table and the new regeneration profile are keyed by season ID, a profile is complete (an unlisted season is 0), no logic names a season. WLD, OBS, INT and GDS parts unchanged.
- **Update (2026-10-11, WLD):** the moment is computed once at P1.1, derived, and adds nothing to a checkpoint (`spec-world.md` 6.3). WLD removed from the areas.
- **Update (2026-10-11, OBS):** no season event is emitted (`spec-observability.md` 4.4.7): `Calendar.At(tick)` is a pure function, the moment is in the snapshot header (5.3.1), and a consumer finds a season boundary by comparing `At(t)` with `At(t-1)`. OBS removed from the areas.
- **Update (2026-10-11, INT):** a consumer converts ticks to dates with `Calendar().At` (`spec-integration.md` 6.2); a shock by season is set before the window and cleared after it, since there is no season key on a slot (3.6). INT removed from the areas.
- **Update (2026-10-11, J8):** PRO's part was applied on 2026-10-10 (the season is read at job start, so a recipe longer than its season completes), so PRO leaves the areas too. What remains is GDS: durations are in ticks and are not rescaled when the calendar changes, so a pack for another planet restates them.

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
- **Update (2026-10-11, CHK):** a restore takes no random word; the stream resumes from the saved PCG state (FR-CHK-051).
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
- **Update (2026-10-15, observability review of plan 1-3):** a reason is not an event, so no row is added to 4.1 or 7.3, but the closed set of `economysim.reason` for `economysim.market.orders.rejected` grew from ten to twelve: the `overflow` row of `spec-clearing.md` 3.2 is now three, `overflow` (the int64 product), `holding_limit` (a bid over `MaxQuantity`) and `receipt_limit` (an ask over `MaxMoney`), so that the two settlement bounds are two series. A module that credits a buyer's stock or a seller's balance between P3.2 and P4.1 (N-235 item 3) adds its worst case to the same two bounds and so to the same two reasons. Left open.
- **Update (2026-10-16, plan 1-5 Phase 1):** the production row declares no event and no `Warn`: `RecipeExecuted`, `RecipeAbandoned` and `ProductionHalted` and the two production `Warn`/`Debug` lines have rows since Stage 0, and the row emits none (the world row emits behind the observer flag). The reasons a slice start can be refused with are five (`spec-production.md` 4.2.1); they are log-only values of `economysim.reason`, not an event. Left open.
- **Update (2026-10-16, observability review of plan 1-5):** applied for PRO and OBS: the refusals are now an event. `RecipeRefused` is a new class A core row of `spec-observability.md` 4.1.3 and of `rules/observability-and-logging.md` 4.3 (4.4.6a, FR-OBS-095; `spec-production.md` 14.1, FR-PRO-052), declared in `event.go`, with the five reasons as `economysim.recipe.refused` values, so the five are no longer log-only. The same change appends `RecipeExecuted.Overflow` and `MarketCleared.CostAnchor` (FR-OBS-096, 097) and adds the metrics `economysim.recipe.abandoned`, `.overflow`, `.in_progress` and `economysim.production.cost_anchor`. N-242 says what the world row must call. Left open.
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
- **Update (2026-10-16, plan 1-5 Phase 1):** applied for the production row: the job and the residues are saved once, in section 6 in agent order (`spec-checkpoint.md` 4.2 tag 6), not in the agent record; N-232 says what the agent row must do.
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
- **Update (2026-10-15, plan 1-3 Phase 1):** left open; `no_room` is STO's. Admission keeps its place in the table of `spec-clearing.md` 3.2 as a marked position with no code, and `deferredReasons` of `TestConstants_Values` keeps `no_room` listed. How a module row reaches the ordered function is held in N-235.
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

### N-154 A count of households gates a structure: the exceptions of `spec-world.md` 9.4.1

- **Areas:** WLD, STB
- **Raised:** 2026-10-13, `spec-settlements.md` 7 and `spec-land.md` 5 (the rent step; one decision, two sources).
- **Note:** Settled by ADR 0006 (`Narrows: 0001 decision 7`): `spec-world.md` 9.4.1 forbids a rate, share, probability, period or window to be a function of a count of households, and names three exceptions: a derived structural rank, a count of households of one region against a place count (the settlement class and a body's places), and a quantity local to one endowment (the rent step reads the units unserved at one endowment; BEL's unmatched units are the precedent). `spec-stability.md` 5.1.3 and FR-STB-020 carry the same sentence. The note stays so that a later spec that wants a fourth exception reads it first. The fallback for the settlement class (flags only: drop `Limits`) is `spec-settlements.md` 7.4.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: no new exception. The "work floor binding" `Warn` compares held households with half the role's households, a diagnostic threshold and not a rate, share, probability, period or window of a decision, so WLD 9.4.1 is not edited. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read: the world reads no count of households into a rate (the "work floor binding" and "households unfed" diagnostics compare counts with a threshold, which is not a decision), so WLD 9.4.1 is not edited. Left open.

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
- **Update (2026-10-14, Phase 1 of plan 1-2):** applied in the specs: `spec-agents.md` 3.4, 5.2 and 22, and `spec-production.md` 3.3, say that the tradable set and the effective upkeep are derived once at construction into the role table (`role.go`), in canonical commodity order, that a module adds its items inside that constructor before it returns, and that nothing mutates the table afterwards (that is all "frozen" means); AGT 22 now homes `Role` in `config.go` and the tradable set in `role.go`. **Still open:** the fifth fan-in in `architecture.md` 2.10.1 and 2.10.4 (held in N-235 for the first Stage 2 module PR, `demand.go`) and whether the agent row needs `internal/household` (it then adds the package to 2.6 and `AGENTS.md`).
- **Update (2026-10-17, plan 1-6 Phase 1):** read: AGT 3.8 is the only AGT clause the welfare row edits and it concerns the length of the record; welfare adds no item to the tradable set and no fan-in, and its arithmetic is `internal/welfare`, not the `internal/household` of `spec-agents.md` 22. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied in part: D80 settles the second point (the agent row needs `internal/household` and adds it to `architecture.md` 2.6 and `AGENTS.md`). The first point (the fifth fan-in for the tradable set) stays for the edit of N-235; the want items of FR-AGT-003 join the role table with the demand module. Left open.
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

### N-232 The job and the residues live on the agent; what the agent row must call

- **Areas:** PRO, AGT, CODE
- **Raised:** 2026-10-16, Phase 1 of `plans/plan-phase-1-5.md` (D50).
- **Note:** `production.go` defines the `job` type, the residue layout (one residue per output item of each recipe on the role's list, then of the effective own-use list, discards included; a function of the role and the recipe table, never saved) and the steps over them, and takes a pointer to a job and a slice of residues, so it decides no storage. The agent row keeps `Job` and `Residues` as the fields `spec-agents.md` 3.1 and 3.2 give the `Agent`, and writes them once, in section 6 in agent order (`spec-checkpoint.md` 4.2 tag 6, N-128). The agent and world rows must: (1) store the fields and call the steps (start, continue, complete, abandon, reweigh, the job of a warm start) at the positions of `spec-world.md` 6.4; (2) draw the one residue word at creation (FR-PRO-026, `spec-world.md` 5.7.1) and set every residue to it, residue 0 for a warm-start job; (3) reset the residues to 0 at a role change (`spec-agents.md` 3.7, `spec-spawner.md` 4.2) with the layout function; (4) emit `RecipeExecuted` and `RecipeAbandoned` from the values the steps return, behind the "observer supplied" flag, log the `Debug` and `Warn` lines and call the welfare scratch with the returned facts (D51: the production row emits nothing and calls no other row); (5) store `Yield` with the job (N-237). If the agent row prefers slices indexed by agent slot (`architecture.md` 2.10.1) the same functions serve.
- **Update (2026-10-17, plan 1-6 Phase 1):** read for AGT 3.8: a role change resets the residues (point 3) and also resizes the welfare ring (WEL 2.3.3, AGT 3.8, SPN 4.2); the agent and spawner rows do both in the one role-change step, in either order. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied for point (2): the residue word at creation is the agent row's (`drawCreation`, which calls the existing `setResidues` and `residueLen`). Point (3), the reset at a role change, is the executor's (row 11 composes `reseedBeliefs`, `setResidues`, `history.Resize` and `job.abandon` inline; this row builds no role-change helper); points (1), (4) and (5) are the world row's; the agent keeps the `job` by value and writes `Yield` with it (N-237). Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate row touches neither the job nor the residues; a role change's goods move by `acceptGoods` (whole lots), independent of the residue reset. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The role change resets the residues in the executor's one role-change step.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for the agent row. `virtualStartOffsets` and `warmJob` are what the world row calls.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open for row 7 (it stores the fields); the world calls the steps at the positions of WLD 6.4.

### N-233 `Discard` and `Holding` exist: what each is for

- **Areas:** PRO, LED
- **Raised:** 2026-10-16, Phase 2 of `plans/plan-phase-1-5.md` (D54, D58).
- **Note:** `internal/ledger` has two additions in `discard.go`, a sibling of `Transform` and not a change to it. `Discard(account, commodity, quantity)` creates the quantity under `recipe_output` and sinks it under `disposal` in one step, with no change to the account's lots, holding or earmarks, no room check and no stock taken (`spec-ledger.md` 8.2.3, FR-LED-066); the completion of a job calls it for each discard output and for each part of an output above `MaxQuantity`, after one `Transform` of what fits. `Holding(account, commodity)` returns the units held, earmarked ones included, and the earmarked ones, with zero allocation (5.2.5, FR-LED-067); the completion reads it for the room (`MaxQuantity` less the held units), and plan 1-3's clearing row may use it in place of a list of lots. `Discard` adds no reason, no faucet, no sink and no boundary account (N-021 and N-226 are untouched). A later row that creates and sinks goods in one step (an own-use job's discard output, for one) uses `Discard` and does not widen `Transform`; a row that needs room from a limited account (storage) reads its own room, not `Holding`, and its pair appears in the count of `spec-production.md` 4.2.1, not in the ledger (N-137).

### N-234 The wear cost takes no second factor of the weight

- **Areas:** WEL, PRO, SCL
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-6.md` (D66).
- **Note:** WEL 2.1.4 and FR-WEL-003 said the wear cost was summed over executions "with the weight of the executor". Read literally beside FR-SCL-007 (the tools a job requires are the per-household amounts times the weight) that counts a cohort of weight 3 running 2 executions as 6. Both are now written once: one ceiling per execution, then the sum over the executions, with no second factor (an execution is one household's and the `k` executions of a job are at most the weight). The owner may rule the other reading (the weight is a factor and `n` is per household): then `n` is 1 in the welfare scratch and only that row changes. Check with the production row's wear facts (`plans/plan-phase-1-5.md` D51), which hand over `n` executions started, and with `spec-production.md` 4.4.2 (each execution wears its own item).
- **Update (2026-10-17, plan 1-6 exit gate):** settled in code: `welfare.WearCost(wearPerUse, price, durability, executions)` takes the executions started and no weight, and `welfareScratch.addWear` merges rows by (commodity, wear per use), so the reading above is what is built; the other reading would change only the `n` the world row hands to `addWear`. Left open for the owner's ruling and for the world row's call.
- **Also read in the same change and left open, not touched by the welfare edits:** N-229, N-230, N-239, N-240 (clearing and beliefs registration, runs and books), N-242 (the production observability the world row emits), N-243 (the Codacy gate: a PR from the welfare rows passes Lizard and uses the spaced separator row in a Markdown table).
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply as built: the world hands `addWear` the executions started (D158).

### N-235 One combined architecture.md edit for the fan-in and admission touch points

- **Areas:** ALL
- **Raised:** 2026-10-13, cross-plan consistency review of `plans/plan-phase-1-2.md` (O1) and `plans/plan-phase-1-3.md` (D38, O2); ruled by the coordinator.
- **Note:** `architecture.md` 2.10.4 gains a fifth fan-in for tradable items (plan 1-2 O1; N-228) and the admission-table touch point for module rows (plan 1-3 D38/O2) in ONE combined edit, with a fresh RULE-TAG, before the first module PR (Stage 2, `demand.go`). The same edit fixes the example of 2.9, which omits the tie-break key and the random source of `clearing.Average` (plan 1-3 D31). No Stage 1 plan makes any of these edits; the sequencing owner is the first Stage 2 module PR. Not blocking Stage 1.
- **Update (2026-10-15, plan 1-3 Phase 1):** the items of plan 1-3 for that one edit, now written where the module PRs will read them. (1) D31: the example of 2.9 omits the tie-break key and the random source that the clearing's matching takes; `spec-clearing.md` 8.3 now has the final shapes (`Offer` with `Key`, `Owner` and `Lot`; `Average(dst, bids, asks)`). (2) D38: the admission table of `spec-clearing.md` 3.2 is one ordered function (`Admit`) whose module rows (`not_permitted`, `no_room`) and legal clamp have marked positions and no code; `not_permitted` is also checked while orders are built, beside `not_tradable`, so the one order of FR-CLR-031 spans two functions. How a module row reaches the function is open item O2 of the plan, settled by the first module PR that needs one: 2.10.2 and 2.10.4 allow a module outside its file only in the catalogue files and the four fan-ins. (3) The D34 window: the overflow rule of `spec-clearing.md` 3.2.2 holds only while no other credit reaches a seller's balance or a buyer's stock between P3.2 and P4.1, so a module that credits there (credit P3.3b, insurance P3.3c, fiscal) adds its worst case to the `receiptTally` or keeps its own headroom. Still open.
- **Update (2026-10-16, plan 1-4 Phase 1):** left open, not the beliefs row's: it adds no fan-in, no admission and no touch point (`beliefs.go` is a core file, not a module file), so nothing here waits for the edit or changes it.
- **Update (2026-10-16, plan 1-5 Phase 1):** left open (owner: the first Stage 2 module PR). The production row adds no fan-in, no admission and no touch point: the limit pairs of the execution count are a parameter of a function, not a touch point, and the tradable set is plan 1-2's.
- **Update (2026-10-17, plan 1-6 Phase 1):** read: the welfare row adds no fan-in and no admission touch point (`welfare.go` is a core concept file; its scratch is called from other rows' positions), so the `architecture.md` edit still waits for the first Stage 2 module PR. Left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: the agent row adds no fan-in and no admission touch point. The core view filler and the intent conversion are called by the world's `fillAgentView` and `checkTradePlan`, and the admission of `not_tradable` and `duplicate_order` is the clearing row's. The edit of 2.6 for `internal/household` (D80) is separate from the combined edit this note holds. Left open for the first Stage 2 module PR.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate row adds no fan-in and no admission touch point: the estate reaches the admission as a plain pseudo-agent through `orderFacts`, and `removeAgent` is the world row's. Learned: the estate is the first caller of a named lot, tested through the existing `book.admit`. Left open for the first Stage 2 module PR.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The spawner adds no fan-in and no admission touch point; the edit waits for the first Stage 2 module PR.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open. Not this row's: no fan-in, no admission. The layout edit of this row is separate and small.

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


### N-238 One agent's ordinary and named-lot asks of one commodity must not share a book

- **Areas:** CLR
- **Raised:** 2026-10-15, plan 1-3 as built (`clearing.go`, `book.admit`).
- **Note:** The ledger earmarks the units of an ordinary ask by commodity, oldest lot first (`ReserveGoods`), so an ordinary ask and a named-lot ask (`ReserveLot`) of one agent and one commodity in one book could earmark the same units. `book.admit` therefore panics (a violated programmer invariant, never a rejection `reason`) when an agent places both kinds. Only the estate names a lot and it names one always, so nothing builds the pair today. `spec-clearing.md` 3.1.2 now says so. A later pseudo-agent that names lots (a store facility, a treasury's release of a held lot) must name a lot on every ask of a commodity or none; if one needs both, the ledger's earmark rule changes first (`spec-ledger.md` 6.7), not the book.

### N-239 `AverageClearing` can produce a self-trade; the check fails the step

- **Areas:** CLR, AGT, WLD
- **Raised:** 2026-10-15, plan 1-3 as built (`internal/clearing.Average`, `Check`).
- **Note:** `Average` does not look at the owner of an offer (plan 1-3 Phase 2 and D31), so one agent holding a crossing bid and a crossing ask in one book is paired with itself, and the check of `spec-clearing.md` 8.2 fails the step with `ErrInvalidTrade`. The guard is deliberately in the check and not in the matching: skipping the pair would change which units trade and the keys drawn per order. Who must avoid it: the strategy rows (an ordinary agent places one side per commodity, `spec-agents.md` 6.5.2, 7.3.3, so it cannot), the world's builders of pseudo-agent orders (the estate places asks only, a counterparty one side, and the treasury of `spec-fiscal.md` 9.4 places a bid strictly below its ask, FR-FIS-053, validated at construction with `ErrFiscalShape`). A module that lets one agent place a bid and an ask of one commodity adds the strict ordering of the limits to its own validation, and a test with the two sides of one agent through `clearMarket`. `spec-clearing.md` 8.2 says so.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied: the default places at most one order per commodity (`spec-agents.md` 7.3.3: a bid if there is a gap, else an ask), so it cannot trade with itself; Phase 6 of the plan asserts one order per commodity for every state of a table and by a fuzz. Left open for the pseudo-agents and the modules.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: the step fails with `ErrInvalidTrade` and the world is failed (D151); the test uses a faulty `Clearing` fake (Phases 5 and 10).

### N-240 The book holds its entries and filled units; the world's scratch does not

- **Areas:** CLR, WLD
- **Raised:** 2026-10-15, plan 1-3 as built (plan A2 said the scratch holds them).
- **Note:** The admitted entries of a market and the units filled per order live in the `book` (`entries`, `fill`), and `clearScratch` holds only what one market step needs and discards (the split sides, the fills, the matches, the lot read, `Check`'s `Sums` and `Index`). Reason: `spec-world.md` P3.2 admits every market before P4.1 clears any, so many books hold entries at once, and a scratch that held them would be overwritten by the next market's admission; and the outcomes of `spec-clearing.md` 7.3 read the filled units after every market has cleared. The world row owns one `clearScratch` per world and one `book` per market, and keeps each book's `fill` until the outcomes are delivered. `spec-clearing.md` 8.3 says so.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: the world owns one `clearScratch` and one `book` per market and keeps each `fill` until the outcomes are delivered (D154, Phase 6).

### N-241 The duplicate-order scan is linear: O(n^2) in the orders of one book

- **Areas:** CLR
- **Raised:** 2026-10-15, plan 1-3 as built (`book.admit`, the `shortcut:` comment).
- **Note:** `book.admit` finds a duplicate (and the mixed ordinary and named-lot case, N-238) by scanning the entries of the book, once per admitted order, which is O(n^2) in the orders of one market. It is fine for the first consumer's world (about 1 000 households and 17 active markets per region, so at most about a thousand orders in a book and about 10^6 comparisons a book a tick, `spec-reference-chain.md` 4.4.4) and keeps the book allocation-free. **Upgrade trigger:** a book above a few thousand orders (a market of a larger world, or `Weight` 1 at the scale of `spec-scale.md` 6), or a benchmark of admission that shows it above the clearing itself; then index by (agent, lot, side) in the book's own storage (a sorted slice or an open-addressing table keyed by the agent), keeping the duplicate and mixed checks in one pass and the allocation test at zero. The plan's NFR-CLR-001 is about `Average`, not admission, so nothing is out of requirement today.
- **Update (2026-10-15, observability review of plan 1-3):** the trigger is now readable: `economysim.market.orders` gains the attribute `economysim.region` (`rules/observability-and-logging.md` 5.3, 5.4.1; `spec-observability.md` 6.2), so a book that grows toward a few thousand orders shows as the per-tick rate of one (region, commodity) series. On `five` that is about 85 active markets by 2 sides, 170 series, against 30 commodities by 2 sides, 60 series, before. Nothing else changes here; the index is still the remedy.

### N-242 What the world row calls for the production observability

- **Areas:** PRO, WLD, OBS, CLR
- **Raised:** 2026-10-16, the observability review of plan 1-5 (`production.go`, `clearing.go`).
- **Note:** The production row builds the values and emits nothing; the world row emits them, behind the "observer supplied" flag. It must: (1) make one `refusalTally` per world (`newRefusalTally(regions, recipes)`), call `add(region, recipe, reason, weight)` where a start returns `k = 0` with a reason, for a free household's job request only (not a halted request, a trainee, an own-use start or a hire), and call `flush(tick, emit)` once at P2.8a after `ProductionHalted`; a later module's reason is a new column of the tally and a new count field of `RecipeRefused`, appended in the master order, and `add` panics on a reason it does not count until then; (2) set `book.anchor` from `productionScratch.unitCost(region, commodity)` before each market's clearing, so `MarketCleared.CostAnchor` is the anchor in force for the tick; (3) build `RecipeExecuted` with `recipeExecuted(outs, disp, over, ...)` and feed the `Warn` "recipe output disposed" from `Overflow`, not from `Disposed`; (4) keep the allocation test of the production phase at zero with the tally. The adapter derives `economysim.recipe.in_progress` from `Jobs[].Executions`, which the snapshot row builds from `addPending`.
- **Update (2026-10-17, plan 1-11 Phase 1):** `spec-world.md` 6.1.13 now lists the `refusalTally` flush at P2.8a, the cost anchor set on the book before each P4.1 clearing, and the advance of the bounded `Warn` runs (points (1) and (2) in part); points (3) (`recipeExecuted` and the `Overflow`-fed `Warn`) and (4) (the allocation test) are not in that list. Left open until the world row merges.
- **Update (2026-10-17, observability review of plan 1-7):** point (1) is widened (N-308): the world also adds to the tally each free household whose work report names a blocked recipe (`workReportBuf.blocked`, the first recipe of the view that could not run, with the view's reason), once, under that recipe and reason, because the default strategy never requests a recipe the view says cannot run and so never reaches the refusal of a start. The same pass fills `Rested` and `NoRecipe` of `ProductionHalted` from `rest` and `reason == no_recipe`. Left open until the world row merges.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply in part (D157, D158, Phases 4 and 5): the `refusalTally` per world, `add` at a refused free household's request and `flush` at P2.8a (and the blocked recipe of the work report, N-308), `book.anchor` set before each clearing, `RecipeExecuted` built with `recipeExecuted(outs, disp, over, ...)` and the "recipe output disposed" `Warn` fed from `Overflow`, and the allocation test of the production phase at zero with the tally. Points (3) and (4) are not yet in `spec-world.md` 6.1.13 (the list there has points (1) and (2)); left open until the world row merges.

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
  - **Still not built, and not this row's:** production in the agent lifecycle, the observer emission and the tick (N-232, N-242), and the checkpoint of jobs and yield shocks (N-237, tag 6, Stage 2). The Codacy review lists both as missing; they are the agent, world and checkpoint rows.
- **Update (2026-10-18, plan 1-8 Phase 1):** read: the estate row's files stay under the Lizard limits by one concept file, `estates.go`; the module half is appended after a marked comment line. Left open.
- **Update (2026-10-18, plan 1-9 Phase 1):** Apply to this row's files: Lizard before every push, the spaced separator in Markdown tables, the new root file in the layout.
- **Update (2026-10-18, plan 1-10 Phase 1):** applied. The Lizard limits (complexity at most 10, at most 8 parameters, files of about 500 lines) are a design input of Phases 2 to 6; `lizard -l go -C 10 -a 8 -L 100 -w` runs before each push; `viability_check.go` and `viability_table.go` are named in the layout (architecture.md 2.6, `AGENTS.md`); markdown tables of the plan use the spaced separator. `internal/checked.muldiv` (CCN 13) is untouched unless this row edits it. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: each file of the row passes `lizard -l go -C 10 -a 8 -L 100 -w`; the Markdown tables of the row use the spaced separator row; the files are in the layout (N-287).

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

### N-246 What the world row must do for the strategy streams and the draw count

- **Areas:** AGT, WLD
- **Raised:** 2026-10-17, `plans/plan-phase-1-7.md` D93.
- **Note:** The default `Work` takes no word. The default `Trade` takes one `rng.Uint64()` per intent it considers, in canonical commodity order: a bid candidate is `Gap > 0`, an ask candidate is `Gap == 0` with `Surplus > 0` or `Expiring - Reserve > 0`, decided by the view alone and never by affordability (a tool the role does not make is not a candidate; plan 1-7 D93, JC2), before any money is allocated, so the count is a function of the view. The world keys the stream before each call: `siteWork` for a consumer's `Work` only (the default takes no word, so the key is skipped), `siteTrade` for `Trade`, both with the tick and the agent; for `Revise` it takes the outcome's word `Draw` from `(siteOutcome, tick, agent, commodity)` first and passes the same `*rand.Rand`, so a consumer's own words follow the world's (WLD 7.3 D3, D5, D8). The creation draws are `drawCreation` (the target word, then the residue word) on the agent's creation stream.
- **Update (2026-10-17, plan 1-7 Phase 1):** `spec-agents.md` 7.3.5 now carries the D93 rules: the candidate rules (bid `Gap > 0`; ask `Gap == 0` with `Surplus > 0` or `Expiring - Reserve > 0`), decided by the view alone and never by affordability, and a tool the role does not make is not an ask candidate. The world's keying of the streams is the world row's code; left open.
- **Update (2026-10-17, plan 1-7 Phase 1):** checked against D93: the ask quantity is `max(Surplus, max(0, Expiring - Reserve))` (the expiring units are part of the stock, so the two are not added), and a bid candidate whose limit cannot buy one unit spends its word and forms no ask; the stream is keyed `(siteTrade, tick, agent)` before the call and the default reads it through the `*rand.Rand`, one source word per draw. The tier combination of D98 (only the first tier to buy may lower the order's limit) is written in `spec-agents.md` 7.5.3 and needs nothing from the world.
- **Update (2026-10-17, plan 1-7 Phase 8):** the default is built to D93 and D98 (`TestTrade_CostAndDrawsDoNotDependOnTheWeight` shows the word count is the same at weights 1, 7 and `MaxWeight`; the many-state property of Phase 7 shows no admission rejection). Still open for the world row: the keying of the streams and the call position of `workReport`.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply (D155): the stream is keyed before each call to an extension, `siteTrade` before every `Trade`, `siteWork` only for a consumer's `Work`, and for `Revise` the outcome's word `Draw` is taken first from `(siteOutcome, tick, agent, commodity)` and the same `*rand.Rand` is passed on; the creation draws are `drawCreation`. Left open for row 7.

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

### N-258 What the world row calls on the estate

- **Areas:** EST, WLD, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-8.md` Cross-plan contract.
- **Note:** The estate is reached only through `estates.go`: `openEstates` at WLD 5.2 order 4, `seedPool` at 5.6 item 5, `asks` then `book.admit` with `facts(region, Ask, tally)` then `admitted` at P3.4 (after the agents' orders, WLD 6.5.1), `markdown` at P5.3, `receive` at removal steps 3 and 4, `grant`, `acceptMoney` and `acceptUnits` at the entry step and at weight changes. The world must not call the ledger on the estate account for money in any other way, so the minimum of FR-EST-014 and the bound of FR-EST-022 have one home. An estate's order has no outcome and takes no belief word (WLD C21).
- **Update (2026-10-17, plan 1-8 D106):** `acceptUnits` (the goods of a weight change, with `occupation.Remaining`) and `acceptGoods` (the intake of a removal or role change, counting lots) are the two goods entries; `receive` is the common intake. Left open until the world row is built.
- **Update (2026-10-18, plan 1-8 Phase 1):** the list gains `acceptUnits` for a weight reduction's goods and the stale-row drop at P5.3 (D102, D106); the world calls `admitted` only for an ask that `book.admit` accepted, so a rejected ask is not marked down (D103). Left open for row 11.
- **Update (2026-10-18, plan 1-9 Phase 1):** Leave open. The calls the executor makes are listed in Consumes; N-266 lists what row 11 calls for the spawner.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: the estate is reached only through `estates.go`, at the positions of the note (Phases 3, 5, 6 and 7); a text-search guard in Phase 7 fails a second ledger call on the estate account for money. Left open until the world row merges.

### N-284 The type `World` is the configuration section and the simulation

- **Areas:** WLD, CFG, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-11.md` (D140).
- **Note:** `config.go` declares `type World struct` (the world-level parameters, `Config.World`), and architecture.md 2.2, 3.2, 6.1, 12.1 and `spec-world.md` 3.1 require the simulation type to be `World`. The plan renames the section type to `WorldParams` (and `DefaultWorld` to `DefaultWorldParams`) in its Phase 2 and keeps the field `Config.World`, the TOML table `[world]`, the hash ids and the owner-table scope cell "World" (the registry test maps the scope to the type). Phase 1 writes it into `spec-config.md` 3.7 and the glossary. **Answer (2026-10-17, coordinator): accepted** (the rules beat `spec-config.md` 3.7; the module is unreleased; no hash input changes). Still owner-confirmable (plan 1-11 O1).
- **Update (2026-10-18, plan 1-10 Phase 1):** left open, with what this row learned: row 11 renames the section to `WorldParams`; this row reads it through `Config.World` only (see the Consumes table of the plan).
- **Update (2026-10-20, plan 1-11 Phase 1):** Phase 1 wrote the decision into `spec-config.md` 3.1, 3.7, 3.7.2, 5.1, 8.2.2 and 9.2 (the section type `WorldParams`, the constructor `DefaultWorldParams`, the field `Config.World`, the table `[world]` and the field ids unchanged) and into the `doc.go` glossary entry of "World" (Phase 2, with the type). Left open until Phase 2 renames the type and the README scope cells.

### N-285 The position table test is staged

- **Areas:** WLD, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-11.md` (D141).
- **Note:** `testing.md` 6.8 and architecture.md 2.10.3 compare `tick.go`'s positions with the table of `spec-world.md` 6, which has about 70 positions of which most belong to modules built later. The plan gives the test the staged form of 6.7 and 9.7: an explicit `notBuiltPositions` list, failing on a stale entry and on a position that is neither called nor listed. A module's change deletes its entries and appends its guarded call. Phase 1 edits the two rules with fresh tags.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Verified in Phase 1: `testing.md` 6.8 and `architecture.md` 2.10.3 already carry the staged form (`notBuiltPositions`, the `where` field, the owner rule, P0.2, the `go/parser` check of the straight line) and `spec-world.md` 6.1.12 states the same, so no rule file is edited and no tag changes. Left open until the test is written (Phase 4).

### N-286 The step is replayable from a test file

- **Areas:** WLD, CODE
- **Raised:** 2026-10-17, coordinator ruling 2 and `plans/plan-phase-1-11.md` (D166).
- **Note:** No hook in the production `Step`: every position is an unexported method of `World` and the non-position parts (the guards, `TickStarted`, `TickEnded`, the marking of a failed world, the tick advance) are unexported methods (`beginStep`, `endStep`) that `Step` and a replay in a `_test.go` file both call, so the per-position state hash of testing.md 7.1.2 needs no exported back door (architecture.md 10.2). A module's position method is appended with its guarded call.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Verified in Phase 1: `spec-world.md` 6.1.12 and `testing.md` 7.1.2 carry the replay seam (`beginStep`, `endStep`, the position methods). Left open until Phase 4 builds it and row 12 replays it.

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

### N-295 The packs of REF 7.5 and 8.2 the slice cannot run

- **Areas:** REF, CODE
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md`.
- **Note:** The invariants and determinism tables of REF 7.5 and 8.2 list `three`, `five`, `famine` (with the estates module part) and `one-demand`. They need regions, the `Estates` part and the demand module, so Stage 1 runs `one` and `second`, and `three` if the probe of Phase 5 of `plans/plan-phase-1-12.md` shows the Stage 1 world runs it (its regions do not trade, REF 16.2). The tests are table-driven: the stage that makes a pack buildable adds its row, and the rule of N-217 applies to the files it adds under `testdata/golden/all/`.
- **Update (2026-10-17, observability/roadmap edits):** REF 7.5 lists the packs; left open for the `three` probe of plan 1-12 Phase 5.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read and applied: Stage 1 runs `one` and `second` (with `two` and `warm-two` for the interleave test only); no `famine`, `five` or `three` in this row's acceptance or scenarios. Left open.

### N-296 S01 leg (b) and every visitor wait for `AddAgent` and `Direct`

- **Areas:** INT, REF, STB
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md` (O3).
- **Note:** `integration.go` (`AddAgent`, `Direct`, `SetYieldShocks`) is Stage 2. In Stage 1 a shock of `scripts/` is applied between steps by the world row's unexported yield-shock setter through `export_test.go`, an exception to FR-REF-021 and FR-STB-014 (ruled by the coordinator, reversible); the visitor of S01 (b) is a skipped subtest that carries this note as its ID. The Stage 2 integration change replaces the seam by the public calls and un-skips the leg.
- **Update (2026-10-17, observability/roadmap edits):** REF 7.1.3 states the seam. Left open until INT.
- **Update (2026-10-20, plan 1-11 Phase 1):** Read: the shock seam is row 12's. Left open.

### N-297 The per-position hash replays the step through unexported methods

- **Areas:** WLD, CODE
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md` (D174; ADR 0007 open point 2).
- **Note:** The test-only state hash of testing.md 7.1.2 adds no hook to production code (architecture.md 10.2). The world row keeps every position of `tick.go` an unexported method of `World` and factors the non-position parts of `Step` (the guard, the tick advance, the `TickStarted`/`TickEnded` bracketing, the failed-world marking) into unexported methods that `Step` and the replay in a `_test.go` file both call. A module that adds a position adds its row to the table and one entry to the test's row-to-method map; a position with no method, or a method with no row, fails the test. A guard test compares the replay with a plain `Run`.
- **Update (2026-10-17, config/world edits):** the WLD half is applied (the position test and the `beginStep`/`endStep` seam are in `spec-world.md` and testing.md 6.8). Left open for row 12.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply (D166): every position is an unexported method, `beginStep` and `endStep(err)` are the non-position parts, no hook. Left open for row 12.

### N-298 Which files of `expected/`, `bands/` and `scripts/` exist, and who creates them

- **Areas:** REF
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md`.
- **Note:** REF 4.1 lists `expected/<pack>.toml`, `bands/<pack>.toml`, `bands/regimes.toml` and `scripts/<scenario>.toml`; none exists. Row 12 creates `bands/one.toml` and `bands/second.toml` (every key unset, REF 7.1.5), `scripts/s01.toml`, `scripts/s04.toml` and `expected/one.scenarios.txt` (a record, not compared). `expected/one.toml` and `expected/second.toml` (hash, surplus, pivots, unpriced, warnings) are created and asserted by row 12 (Phase 6; coordinator ruling J2, reversible); other packs' files are created by the first change that asserts them. `bands/regimes.toml` is STY's.

### N-299 The defect-file tests flip with the viability rules

- **Areas:** CFG, REF
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md` (A8).
- **Note:** `TestPacks_DefectOverlays` asserts `Validate` is nil for d1 to d5 "at Stage 0". The viability row's change adds the rules and must flip that line to `errors.Is(err, ErrNotViable)` in its own PR so the suite stays green. Row 12 then adds the rule name per defect, "that rule only" for d3, d4 and d5, the warm layer set, and the case `d1` with the wild source on (`labour_short`, not `cycle_not_productive`).
- **Update (2026-10-18, plan 1-10 Phase 1):** applied for this row: Phase 5 flips the d1 to d5 line of `TestPacks_DefectOverlays` from nil to `errors.Is(err, ErrNotViable)` in the PR that adds phase 6; plan 1-12 adds the rule names and the "that rule only" assertions. Left open for row 12.

### N-300 The first measurement of a `Step` of `one`

- **Areas:** SCL, OBS, CODE
- **Raised:** 2026-10-17, Phase 1 of `plans/plan-phase-1-12.md` (Phase 8).
- **Note:** To be measured by Phase 8 and 9 of the plan: the allocations of a full `Step` of the pack `one` with the default logger and observer (the baseline constant of NFR-WLD-001 and NFR-OBS-002) and the time per tick. It is the measurement N-127 (the allocation per event) and N-222 (the cost of a reseed at every draw site) ask for at the first slice, and the first check of the cost model of `spec-scale.md` 6.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open for row 12. This row's allocation test asserts the call sites only and its benchmark reports (Phase 10).

### N-274 The producer of the warm documents is not in the repository

- **Areas:** VIA, REF, CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (D133).
- **Note:** `testdata/chains/warm-*.toml` were written by REF's prototype (`/tmp/economy-boss/ref/lp.py`), which is not archived (`doc/specs/models/via/` gives lambda and prices only, not money, stock or jobs). Once the Go solver passes the agreement run of Phase 7 it is the reference: a document that does not load equal to the solver's table is regenerated with the explanation of testing.md 7.2, or the code is fixed, by the owner's ruling (O2 of the plan). Do not archive anything from `/tmp`.
- **Update (2026-10-17, viability edits):** `spec-viability.md` 15 now cites the unarchived `lp.py`. Left open.
- **Update (2026-10-18, plan 1-10 Phase 7):** the Go solver is now the reference. Loaded equality (D133) of `warm-one`, `warm-two`, `warm-three`, `warm-five` and `warm-second` with the solver's table holds, and each validates. Regenerated with the testing.md 7.2 explanation: `warm-five.toml` as a whole (prices and wage on the exact scale of VIA 6.3, the file's old scale being 0.9909 to 1.0129 of it; `semi_a` and `semi_d` lifted from 50 to 51 and 52, into the band; the three seasonal job rows and the centre grower reserve 1 092 for 1 087, D123); the seasonal `grow_staple` job rows of `warm-two` (south 3 341 to 3 110) and `warm-three` (far_south 1 736 to 3 250, south 3 464 to 3 224), which follow VIA 8.5.1 as written (D134 (9)); and the `grow_algae` row of `warm-second` (5 163 to 5 232, D134 (8): the producer is unarchived, so the cause is not known). The hash pin of `second` in `config/integration_test.go` moved with it. `warm-famine.toml` is unchanged (Stage 2). Left open.

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

### N-280 One function gives the virtual start offsets

- **Areas:** VIA, WLD, AGT
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (Provides).
- **Note:** `virtualStartOffsets` (8.5.2, by `Calendar.SeasonAt`) is called by the empty-set rule of `validate.go` and, at construction, by the world row's jobs, so the rule and the construction cannot disagree (FR-VIA-053). The world row owns the stagger and the application of the table.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: not this row's; the world's job construction calls `virtualStartOffsets`. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: Phase 3 deliverable 3e calls `virtualStartOffsets`, so the rule of `validate.go` and the construction cannot disagree (FR-VIA-053); the stagger and the application of the table are this row's. Left open.

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

### N-266 What the world row calls for the spawner

- **Areas:** SPN, WLD, AGT, EST
- **Raised:** 2026-10-17, `plans/plan-phase-1-9.md` (Cross-plan contract).
- **Note:** The spawner row builds the plan and the pure support and emits nothing; the executor is the world's (SPN 20, WLD 6.9 P7.1(d), coordinator Q1). Row 11 must: (1) build the `RegionView` of each region (roles in canonical order, agents in ascending ID, no pseudo-agent; `Wage` and `HasWage` from the welfare summary, `Upkeep` per household at the published prices, `PoolMoney` from the estate, `RenewalCarry`), set `Privation` at P6.1 with `household.NextPrivation` (plan 1-7), and give the view the one scratch of the world; (2) call `Decide` with the keyed stream `(siteSpawner, t, region)`; (3) execute the plan in order, checking each decision with `occupation.Refuse` against live state, taking the funded number from `occupation.Funded` and the placement from `occupation.Place`, sending a reduction's money through `acceptMoney` and its goods, per commodity, through `acceptUnits` with `occupation.Remaining`, and a role change's goods through `acceptGoods` (plan 1-8), counting `GoodsListed` in lots; (4) merge the executor's counts with the counts of the plan into one `EntryLimited` per region per tick (`entryTally.limited`), emit the `Debug` lines and the three bounded `Warn`s from `extinctRun`, `unfundedRun` and `belowFloorRun`; (5) call `resolveSpawner` once in `NewWorld` after `Validate`. The executor is row 11's: the role change and the weight change are composed there from the primitives of rows 5, 6 and 7 (`job.abandon`, `job.reweigh`, `setResidues`, `history.Resize`, `reseedBeliefs`) and the target carried over by `occupation.Retarget`; plan 1-7 builds no combined step (consistency pass, 2026-10-17).
- **Update (2026-10-17, plan 1-8 D106):** the executor sends a reduction's goods through `acceptUnits`, and a role change's through `acceptGoods`; both count `GoodsListed` in lots (N-255). Left open until row 11 is built.
- **Update (2026-10-17, plan 1-7 Phase 1):** read: row 11 calls `consume` and `household.NextPrivation` through it; the agent row builds no combined step. Learned: `consume` sets `privation` and the spawner's view reads it. Left open.
- **Update (2026-10-18, plan 1-8 Phase 1):** checked against Provides: `acceptMoney`, `acceptUnits` (per commodity), `acceptGoods` (a role change's whole lots), `receive` and `grant` have the signatures of the Cross-plan contract of plan 1-8. Left open for row 11.
- **Update (2026-10-18, plan 1-9 exit gate):** the support functions of the Cross-plan contract exist with the planned names (`Refuse`, `Funded`, `Place`, `Remaining`, `Retarget`, `entryTally`, the three runs). Left open for row 11.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply (Phase 7): the five items of the note are Phase 7's deliverables 2 to 4 and the `resolveSpawner` call of Phase 2; `GoodsListed` counts lots. Left open until the world row merges.

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

### N-304 The file lists of architecture.md 2.6 and AGENTS.md must stay identical

- **Areas:** ALL, CODE
- **Raised:** 2026-10-17, WP1 report of the owed-edits job.
- **Note:** `rules/architecture.md` 2.6 lists the continuation files (`construct*.go`, `fanin.go`, `phase_*.go`, `viability_*.go`, `production_report.go`); the layout in `AGENTS.md` repeats them. `AGENTS.md` says the architecture list wins. Any change to one lists the same files in the other in the same change, with a fresh tag for architecture.md.
- **Update (2026-10-17, plan 1-7 Phase 1):** applied: this phase's layout edits (`agent.go`, `strategy.go`, `strategy_default.go`, `strategy_work.go`, `strategy_view.go`, `internal/household`) are in `architecture.md` 2.6 and in `AGENTS.md` with the same file lists, checked line by line, and `architecture.md` has a fresh tag. A standing rule; left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** read: `viability_check.go` and `viability_table.go` are in both lists (architecture.md 2.6 and `AGENTS.md`), identical. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Apply: the continuation files of this row (`construct.go`, `construct_warm.go`, `fanin.go`, `phase_start.go`, `phase_production.go`, `phase_trade.go`, `phase_outcomes.go`, `phase_exit.go`) and `production_report.go` are in `architecture.md` 2.6 and in `AGENTS.md` with identical lists, checked line by line in Phase 1; no layout edit was needed, so no tag changed. A standing rule; left open.

### N-305 When SKL appends `DecideApply`, SPN 6.2 and 6.3.0 must be re-read

- **Areas:** SPN, SKL
- **Raised:** 2026-10-17, WP5 report of the owed-edits job.
- **Note:** `spec-spawner.md` 6.2's training rule (a role with `TrainingTicks` above 0 is never entered) and the counts of 6.3.0 hold only while no queue exists. The change that appends `DecideApply` (N-264) re-reads both and edits them in the same change.

### N-306 The census formula is pinned against `one`: the work share cancels

- **Areas:** VIA, REF
- **Raised:** 2026-10-17, WP6 report of the owed-edits job.
- **Note:** `spec-viability.md` 5.2 (line 489) says the available work share divides every term alike and cancels, so it is not in the census formula. REF 4.3 and 4.4.1 should say the same, so the census of `one` is read the same way in both specs. Check them when REF is next edited.
- **Update (2026-10-18, plan 1-10 Phase 1):** read: the census formula is written in `spec-viability.md` 5.2 against `one` (deliverable 3); Phase 6 pins it in a test. Left open.

### N-307 The `second` `grow_algae` row: 5 163 against 5 232

- **Areas:** REF, VIA
- **Raised:** 2026-10-17, WP6 report of the owed-edits job.
- **Note:** The `grow_algae` row of the pack `second` differs between the prototype (5 232) and the Go solver (5 163). Settle it in plan 1-10 Phase 7 (the agreement run) and then fix the number in REF or in the solver, with the testing.md 7.2 explanation. N-274 applies to any regenerated warm document.
- **Update (2026-10-18, plan 1-10 Phase 1):** left open for Phase 7 (the agreement run), which settles the number and gives the testing.md 7.2 explanation.

### N-308 The refusal tally never fires for the default strategy unless the work report names the blocked recipe

- **Areas:** AGT, PRO, OBS, WLD
- **Raised:** 2026-10-17, the observability review of the branch of plan 1-7 (`strategy_work.go`, `spec-production.md` 14.1, N-242).
- **Note:** The default `Work` passes over every recipe whose view says `MaxExecutions` 0, so it never requests a recipe production would refuse, and the `refusalTally` of N-242 (filled where a start returns 0) would stay empty for it: `economysim.recipe.refused` would read 0 on a chain stalled for want of an input or a tool, the case the event was added for. The work report now carries `blocked`, the index of the first recipe of the role that could not run when the share is above 0 and the halt skipped none (FR-AGT-086, `spec-agents.md` 7.1.4), and the world adds that household to the tally under the recipe and the view's reason. A household with several blocked recipes is counted once, under the first in list order, so the counts add to the households refused; the reason is the first applicable, as 4.2.1 says. A consumer's strategy that requests a blocked recipe is counted by the refusal of the start as before, and a household is never counted by both. **Open:** the world row must make the call (N-242 point 1), and the golden of a stalled chain must show a non-zero `RecipeRefused` under the default strategy; a later module's reason (`not_permitted`, `no_room`, `no_endowment`) arrives in the view as it does in the tally, a new column appended with the module.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The plan does not yet list the call this note asks (the work report's blocked recipe added to the tally at P2.5, with `Rested` and `NoRecipe` of `ProductionHalted`): Phase 4 must make it, with N-242 point 1, and the golden of a stalled chain (row 12) must show a non-zero `RecipeRefused` under the default strategy.

### N-309 What the estate row (plan 1-8) leaves to later rows

- **Areas:** EST, WLD, CHK, CODE, OBS
- **Raised:** 2026-10-18, the audit of `plans/plan-phase-1-8.md` (`estates.go`, built and tested without a world).
- **Note:** `estates.go` is a set of unexported functions with no caller until the world row, so these requirements are only partly covered and are completed there; each needs a test in its owner's row. (1) **World row (11):** FR-EST-010 and 011 (no `AgentSpawned`, no snapshot row, no strategy call, belief or outcome for an estate, and no agent event whose subject is an estate; the row tests only `regionOf`); FR-EST-002 (the estate that receives is the one of the region of the agent's account); FR-EST-004 (the estates open before any household, WLD 5.2 order 4); FR-EST-012 (the asks are placed at P3.4 before any market clears, with the tie-break key; the row tests only that the order carries the estate's ID); FR-EST-013 (`Regions[].MoneyPool`, `Markets[].HeldByEstate` and `EntryBudget` in the snapshot); FR-EST-014 (`w * Role.StartMoney` and "an entrant gets no goods" are the executor's, with the whole-household budget); FR-EST-015 (the `initial` money of the initial agents, and the map of `WarmPool.Region`, a string, to a `RegionID` before `seedPool`); FR-EST-021 (emit `AgentBankrupted` then `EstateOpened` behind the observer flag; `EstateOpened.Balance` is read as `pool` after `receive`, plan open item O6, and `GoodsListed` counts lots, N-255). The call list is N-258 and N-266. (2) **Checkpoint row (Stage 2):** build tag 8 from `rows` and `restore`, wrap `errListingRows` in `ErrCheckpointCorrupt`, and make the cross-check of restore step 7 (FR-CHK-023) call them (N-256). (3) **The `Estates` module part (Stage 2, after MON):** `Region.PoolRelease` and `Region.ReliefBelow`, relief at P1.6 and `ReliefPaid`, the pool as a money-flow target with `ErrModuleDependency`, FR-EST-017 to 020, and the module fields of FR-EST-023; they append to `estates.go` after the marked comment line at its end. (4) **Other modules:** FR-EST-003 (a traveller who dies in transit, a diverted cargo) is TRN's and MER's; money the estate receives as creditor or title holder is CRD's and LND's; the tool-sale rule of SKL 2.7 beyond the first price is SKL's. (5) **After the first golden file:** the first price, the markdown and the keep-the-price rule of a lot that gains units (D102, D104; N-254) enter `testdata/golden/` at the end of Stage 1, and a later change is a behaviour change with the explanation of testing.md 7.2.
- **Update (2026-10-18, test audit of the branch):** the audit against testing.md found the row tested at the unit level only, as the plan says (`estates.go` is 100% covered, every function; root package 98.8%), and left these test debts for later rows. Each needs a test in its owner's row; none can be written before there is a `World`. (a) **Public-API integration tests (testing.md 2.3), world row (11):** the only public-API test of the row is `TestEstates_PackOneAccountIDs` (FR-EST-004, 005). FR-EST-006 (a bid from an estate is `side_not_allowed`), 007 (one ask per held lot, a rejected ask is not an ask), 009 (the markdown at P5.3) and 013 (the proceeds are a `trade` transfer into the pool, `MoneyPool` equals the account) are consumer-visible and need a test through `NewWorld` and `Step` with a recording observer, beside the unit tests that exist. (b) **Restore equals uninterrupted (FR-EST-024), checkpoint row (Stage 2):** a restored world must list and price the same lots as an uninterrupted one; the unit tests cover `rows` and `restore` only (N-256). (c) **Mutation pass:** 22 hand-made mutants of `estates.go` were run; the two that survived were an equivalent one (`<=` against `<` at the floor in `markedDown`) and the clearing of the `asked` mark in `asks`, now covered by `TestEstateAsks_MarksOfTheLastTickAreCleared`. `gomu` was not run (Docker only, testing.md 9.4); it is the periodic pass for the core packages. (d) **Fuzz seeds:** `testdata/fuzz/FuzzPriceRules`, `FuzzReconcile` and `FuzzRestore` hold hand-written seeds, not crashers, so testing.md 7.5's `Fix:` rule does not apply; they must be committed with the row. (e) **The invariant branches** of `estates.go` (an unreadable account, a belief step out of bound, an overflowing first price, an admitted ask with no listing) are `Guard:` tests (`estates_errors_internal_test.go`), so no coverage exclusion under testing.md 9.5 is needed. Left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. The world-row items (1) are spread over Phases 3 to 10 (the estate opened before any household, the asks at P3.4, `MoneyPool`, `HeldByEstate` and `EntryBudget` in the snapshot, `AgentBankrupted` then `EstateOpened` behind the observer flag); the plan names no test of FR-EST-006, 007 and 009 (a bid from an estate is refused, one ask per held lot, the markdown at P5.3) through `NewWorld` and `Step`, which Phase 10 should carry. Items (2) to (5) are not this row's.

### N-310 What the observability review of plan 1-8 leaves undone

- **Areas:** EST, OBS, WLD, CODE
- **Raised:** 2026-10-18, the observability review of the branch of plan 1-8 (`estates.go`, N-309 item 6).
- **Note:** The catalogue and spec rows for `economysim.estate.sold`, `economysim.estate.at_floor` and `Markets[].EstateAtFloor` are written, and nothing can emit them until the world, the snapshot and `otelobserver` exist. Owed, each with its test in the row that builds it: (1) **World/snapshot row (11):** declare `EstateAtFloor` in `snapshot.go` and fill it as the units of the estate's lots whose row in `estates.listed` is at the resolution floor (`resolutionFloor`); it is 0 with no estate stock and never above `HeldByEstate`; a test builds a world, lets an estate lot fall to the floor and compares the field; the field is a function of saved state (N-125), so the restore test (FR-OBS-043) must cover it. (2) **`otelobserver` (Stage 2):** feed `estate.sold` from `TradeExecuted` where `Seller` is below the region count (the test of `spec-observability.md` 4.3.3 item 2, `estates.regionOf`), record `estate.at_floor` from the snapshot, and add both to the adapter's cardinality test (testing 8.1: commodity and region only; about 300 series, 6.5.2). (3) **Pool overflow through a step:** the world row needs a test that an `ErrBalanceOverflow` of an estate pool fails the step (`TickEnded` with `Failed`, `economysim.world.step.failed`) and that the error text names the region and the account, as `refused` now does; the unit test covers `refused` only. (4) **Allocation test:** `estates.asks` and `markdown` have a no-allocation test in isolation (NFR-EST-001); the full-step allocation limit of observability 8.2 includes them only once the world row calls them. (5) **Reading `economysim.market.unmatched`:** its ask side includes the estate's unsold lots; if the first dashboards show it misleads, split the series by a holder attribute (doubling its ask series) instead of subtracting `HeldByEstate`; decide with data from the first golden run, not now. (6) **Not done in this review:** `doc/notes.md` was not read in full before the spec edits (only the notes tagged OBS and EST that the edits touch, and N-125, N-126, N-127, N-289 and N-309); the next change to `spec-observability.md` should read it whole.
- **Update (2026-10-20, observability review of the branch of plan 1-11):** item (1) is applied: `Markets[].EstateAtFloor` is declared in `snapshot.go` and filled in `snapRegion` from `estates.listed` (`estates.atFloor`: a held lot whose row price is at or below `resolutionFloor`; a lot with no row is not at the floor), tested by `TestSnapshot_EstateAtFloorCountsUnitsListedAtTheResolutionFloor` (fails with the check broken). The restore-equality half of the test (FR-OBS-043) is the checkpoint row's, since the field is derived from the saved listing rows (tag 8). Items (2) to (6) are unchanged; item (3), the pool-overflow step test, is still not in Phase 10. Left open.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row. Left open.
- **Update (2026-10-20, plan 1-11 Phase 1):** Leave open. Item (1), `Markets[].EstateAtFloor` filled from `estates.listed` against the resolution floor, and item (3), the pool-overflow step test, are not in the plan's Phase 9 and Phase 10 lists: they must be added to Phase 9 (the field and its test) and Phase 10 (the failed step) when those phases are built. Items (2), (4) and (5) are not this row's.

### N-311 Per-commodity price uniqueness is not detected by the solver

- **Areas:** VIA, CODE
- **Raised:** 2026-10-18, `plans/plan-phase-1-10.md` (Phase 6 fix, FR-VIA-026).
- **Note:** FR-VIA-026 and VIA 8.2.4 give a commodity whose price is not unique the reference price and list it in `UnpricedCommodities`. `steady.Outcome` carries only the global `Degenerate` and `Alternative` flags (FR-VIA-019), and telling which duals are non-unique needs a min and a max of each price over the optimal dual face (2n further programmes). Phase 6 builds the dual-0 half only; a non-unique non-zero dual is used as the price and the report's `Degenerate` and `AlternativeOptima` flags are the warning. Reversible: add a per-commodity flag to `steady.Outcome` and the reference-price rule in `prices()`.

## Closed notes

When a note is closed, move it here and add **Closed:** with the date, the
resolution (applied or rejected, and why) and where it was settled.
- **Update (2026-10-18, observability review of plan 1-8):** (6) **World row (11) and the adapter:** fill `Markets[].EstateAtFloor` from `estates.listed` against the resolution floor (units of lots whose row price equals it), and the adapter feeds `economysim.estate.sold` from `TradeExecuted` where `Seller` is below the region count (`regionOf`); each needs a test in that row. The errors of `receive`, `acceptUnits` and `acceptGoods` now name the region and the account, so a pool that overflows names itself in the failed step's error.

### N-223 `doc/roadmap.md` is `doc/spec-roadmap.md`

- **Areas:** ALL
- **Raised:** 2026-10-12, Phase 1 of the Stage 0 plan.
- **Note:** The file was renamed (the code's order is `roadmap-v1.md`). The links and names were corrected in `vision.md`, the ADRs, `documentation_index.md` and `spec-observability.md`. A bare `roadmap.md` remains in `overview.md`, `specs/README.md`, `specs/evidence.md` and the specs `spec-checkpoint`, `credit`, `experiments`, `fiscal`, `indicators`, `insurance`, `integration`, `land`, `policy`, `risk`, `saving-and-investment`, `settlements`, `stability`, `stylised-facts`, `technology` and `world`: each reads `spec-roadmap.md` (a mechanical rename, by the owner of each file). Close when a text search finds none.
- **Closed:** 2026-10-12. Applied (the `/code-review` run after Stage 0 was built): every bare `roadmap.md` in `doc/` now reads `spec-roadmap.md`, in the live references (the "Raised: `roadmap.md` Open N" lines of N-024 to N-029, whose numbers are those of the "Open" list of `spec-roadmap.md`; `spec-stability.md`; `spec-stylised-facts.md`; `spec-settlements.md`) and in the handoff tables of the specs that name the file they edited. A text search over `doc/` (outside `doc/plans/` and the closed notes that describe the rename) finds none.

### N-102 Unify the module table names before the loader is written

- **Areas:** CFG
- **Raised:** 2026-10-10, the wave 2 integration (the specs of that wave were not asked to share one spelling).
- **Note:** The specs name the table of a module differently in the file: `[module.estates]`, `[demand]`, `[module.money_supply]`, `[population]` and transport's own. `spec-config.md` records each as its spec says and leaves this as an open question. Unify before the loader is written.
- **Update (2026-10-10, RES):** RES writes its module tables as `[resources]`, `[resources.resource.<id>]` and `[resources.endowment.<region>.<resource>]`, the way `[demand.want.<id>]` is written (`spec-resources.md` 3.3.2).
- **Update (2026-10-10, INF):** INF writes its table as `[modules.information]` (as transport's `modules.transport.modes`) and the role flag as `informed` in the role table.
- **Update (2026-10-10, SUB):** SUB writes `[modules.subsistence]` and, for a role row, `[modules.subsistence.role.<id>]` (`spec-subsistence.md` 3.2).
- **Update (2026-10-10, MER):** MER writes `[modules.merchants]` and the block `[role.<id>.merchant]` (`spec-merchants.md` 19.1), the form of `[modules.information]`.
- **Update (2026-10-13, STO, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS):** the later modules write their tables in the form of `[modules.information]` and `[modules.merchants]`: `[modules.storage.class.<id>]` (STO; the layout of its base room and role rows is not stated yet and is STO's to state), `[modules.settlements]` with `.class.<id>`, `.class.<id>.limits` and `.region.<id>`, `[modules.tenure]` with `.term.<region>.<resource>` and `.holding.<region>.<resource>.<holder>`, `[modules.polity]` with `.jurisdiction.<id>` and `.member`, `[modules.credit]`, `[modules.infrastructure]` with `.class.<id>` and `.initial`, `[modules.risk]` with `.route.<from>.<to>.<mode>` and `.store.<region>.<class>`, `[modules.fiscal]` with `.tax.<id>`, `.toll.<id>` and the other kinds, `[modules.institutions]` with `.body.<id>`, `.right.<id>`, `.control.<id>` and `.rate_limit.<id>`, `[modules.saving]`, `[modules.technology]` with `.field.<id>`, `[modules.policy]` with `.actor.<id>`, `.lever.<id>` and `.rule.<id>`, and `[modules.insurance]`; role blocks follow the form of `[role.<id>.merchant]` (SAV `[role.<id>.saver]`). The loader of CFG is written against the spelling of `spec-config.md`, which records each as its spec says.
- **Closed:** 2026-10-12. Applied (Phase 1 of the Stage 0 plan): one rule, `[modules.<name>]` with `<name>` the snake case of the field of `Config.Modules`; the module is on iff its table is in the merged tree (an empty table counts); a module's entities are tables one level below keyed by ID and named by the singular of the list field (`want`, `flow`, `mode`, `node`); a block added to a role or recipe is a sub-table `[role.<id>.<block>]`; regional rows are `[modules.<name>.region.<id>]` (`spec-config.md` 6.2.1, FR-CFG-098). `[modules.estates]` maps to `Modules.Estates`, a field-less struct appended as the 25th pointer, and what it switches on is the pool policy only (6.2.4, FR-CFG-099); `ReliefPaid` is not a slice event. Every spec and the fixtures `testdata/chains/demand.toml` and `overlay/estates.toml` are respelled; the old forms are `ErrUnknownKey`. STO's base room and role rows follow the rule (N-221).

### N-023 Calendar

- **Areas:** CAL, WEL
- **Raised:** 2026-10-09.
- **Note:** `spec-calendar.md` builds on `TicksPerYear` and does not redefine it
  (`spec-currency.md` 2.4). Per-tick quantities (upkeep, the reference basket) are
  not rescaled when the tick length changes. `ExpectedWorkTicks` is an expectation
  for calibration and the skill entry rule, and must be at most `TicksPerYear`
  (`spec-welfare-and-wages.md` 3.3, 4.2).
- **Closed:** 2026-10-10. Applied in part, rejected in part. Applied: per-tick
  quantities and durations are not rescaled (`spec-calendar.md` 6.1, 6.2) and
  `ExpectedWorkTicks` is at most `TicksPerYear` (6.3). Rejected: the first sentence.
  To make time configurable, the calendar defines `TicksPerYear` as a derived
  quantity (`spec-calendar.md` 3.2) and `spec-currency.md` 2.4 now says so, so it is
  no longer a separate world parameter.

### N-003 Non-tradable goods, disposal and the recipe format

- **Areas:** PRO, GDS, LED
- **Raised:** 2026-10-09, `spec-goods.md` open question 4 and `vision.md` goal 5.
- **Note:** Open for the production spec: a byproduct with no buyer, a waste
  product, whether a `Tradable` flag and a disposal sink are needed. Quality
  grades are separate commodities (`spec-goods.md` 7.1), so the recipe format must
  not assume a grade. Still undecided: variable yield, and conditions on a recipe
  (a season, a building), which touch `spec-calendar.md` and
  `spec-infrastructure.md`. Decide the recipe format against a second, unrelated
  setting, as `vision.md` section 5, item 3 requires.
- **Closed:** 2026-10-10. Applied in `spec-production.md` 7 and 3.2: no `Tradable` flag (rejected for version 1); an output may be `Discard` (created and disposed in one step, measured in flow totals); a `disposal` goods sink is needed in the ledger registry (N-033); byproducts are outputs with `CostShare` 0. Variable yield and conditions: deterministic yield factor by region and season, season read at job start; building conditions and random yield deferred with costs in 6.7. The recipe format assumes no grade and was checked against two unrelated settings (section 12). `spec-goods.md` open question 4 is answered by `spec-production.md` 7.5.

### N-004 What production takes on from other specs

- **Areas:** PRO
- **Raised:** 2026-10-09, collected while writing GDS, SCL and LED.
- **Note:** The production spec inherits these, already stated elsewhere:
  - it owns `Requires` on a recipe and `wear_per_use`; GDS owns `Durability`
    (`spec-goods.md` 5.3), and wear falls on the most worn item first
    (`spec-ledger.md` 5.4);
  - a recipe execution is one all-or-nothing ledger transform; a recipe with no
    inputs is a source (`spec-ledger.md` 8.1, 8.2; `spec-goods.md` 6.2);
  - an agent of weight `w` runs up to `w` executions, and `k` if inputs cover
    only `k` (`spec-scale.md` 4.2, FR-SCL-007, FR-SCL-008);
  - a durable no recipe requires must warn (FR-GDS-023);
  - which recipes are trainee-eligible and how large their output is sets
    whether training pays (`specs/README.md`, "Still open" 1).
- **Closed:** 2026-10-10. `Requires` and `WearPerUse` are owned in `spec-production.md` 4.4 (wear on distinct items, most worn first; it asks the ledger for `WearItems`, N-033); one all-or-nothing transform and source recipes in 4.3 and 3.1.2; weight `w` runs up to `w`, `k` if inputs cover `k`, in 4.2; the durable-not-required warning is FR-PRO-040; the trainee-eligible lever is in section 8. The chain's actual values stay open (`specs/README.md`, "Still open" 1).

### N-005 How chance events resolve for a cohort

- **Areas:** PRO, SKL
- **Raised:** 2026-10-09, `spec-scale.md` open question 3.
- **Note:** A per-household chance event for an agent of weight `w` must have the
  right expected count with integer arithmetic and the world's random source
  (`spec-scale.md` 4.6, FR-SCL-012). The recommended method is a carried
  remainder with a random starting offset. Tool breakage is no longer random (it
  is a durability counter, `spec-skill-barriers.md` 2.1), so this may only matter
  for other events. Decide whether the method is used at every weight or only
  above 1.
- **Closed:** 2026-10-10. Settled for production and skill barriers in `spec-production.md` 6.6: yield fractions use a deterministic carried remainder at every weight (including 1), started from one seeded draw per agent at creation; production draws no chance trials; tool breakage is a counter (`spec-skill-barriers.md` 2.1). The belief gate is a decision with one draw per outcome whatever the weight, not a per-household chance event (`spec-beliefs.md` 9.3). A future spec with a true per-household chance uses the helper of `spec-scale.md` 4.6. This also settles `spec-scale.md` open question 3 for production.

### N-006 What price beliefs must do with clearing's outputs

- **Areas:** BEL
- **Raised:** 2026-10-09, `spec-clearing.md` sections 5.4, 5.5, 7.3 and 13.
- **Note:** `spec-beliefs.md` must:
  - use the **price age** and the **best unmatched quotes** to move off a stale
    price, since clearing publishes a carried-forward price when nothing trades;
  - own the **belief step** that sets the resolution floor
    (`spec-currency.md` 3.1; named in `spec-config.md` 7.4), bound its steps, and
    adjust asynchronously;
  - fix the bazaarBot defects assigned to it (`bazaarbot-reference.md` section 11):
    7 (the lookback window), 9 (a threshold that depends on the price scale; use
    basis points of the mean), 10 (the range can invert; define the invariant) and
    11 (update per order outcome, once, not per match; `spec-clearing.md` 7.3).
  - the Doran and Parberry paper has not been read (N-015).
- **Closed:** 2026-10-10. Applied in `spec-beliefs.md`. Price age and the best unmatched quotes move a belief off a stale price (5.4, FR-BEL-011, FR-BEL-012, FR-BEL-034). `BeliefStep` is the belief step and sets the resolution floor `ceil(10 000 / BeliefStep)` (5.3, 11, FR-BEL-029); steps are bounded to `2 * BeliefStep` and asynchronous (5.2, 5.6). Defect 7: a per-market window, no per-agent history (7). Defect 9: `BeliefSignificance` in basis points of the centre (5.4). Defect 10: centre and half-width, invariant in 3.3. Defect 11: one update per admitted order (5.1.2). The Doran and Parberry paper is now read (journal version, `specs/evidence.md`); it uses no window for an agent's own range and its pseudocode has no crossing check.

### N-002 Does an agent have an inventory capacity?

- **Areas:** AGT, LED
- **Raised:** 2026-10-09, writing `spec-goods.md` (open question 3).
- **Note:** bazaarBot bounded an agent's whole inventory (`max_size`) and made it
  discard goods when full (`bazaarbot-reference.md` sections 3 and 10). GDS gives
  `Mass` and `Bulk`; whether an agent has a capacity in version 1 is undecided.
  If it does, a discard needs a goods sink reason in the ledger registry
  (`spec-ledger.md` 7.2) and an event, and the choice of what to discard is a
  strategy decision, not a rule.
- **Learned (2026-10-10, `spec-production.md` 5.3.2):** production never refuses an output for lack of room; overflow beyond `MaxQuantity` is disposed with reason `disposal`. If AGT adds an inventory capacity, a full inventory is the strategy's problem on the next tick, not a reason to lose output at completion. `disposal` is available to AGT for discards.
- **Closed:** 2026-10-10. Applied in `spec-agents.md` 8: no inventory capacity in version 1 (bazaarBot's `max_size` is the source of defects 6 and 8, storage is later, production never refuses output, overstock is limited by spoilage, the buffer cap, the production halt and the belief stock rule). A discard is a strategy decision executed as a move to the `disposal` boundary, so it is counted in `BoundaryFlow`; the default never discards. `spec-ledger.md` needs nothing beyond `disposal` (N-033).
- **Update (2026-10-13, STO):** "production never refuses an output for lack of room" stands for `MaxQuantity` and for every completion; the one exception is a cut of the room: with the storage module on the room is committed at the decision, so a job's start is refused `no_room` and the completion never fails (`spec-storage.md` 5.3, 5.3.4). The default strategy, which "never discards" above, discards for upkeep only when the module is on (N-139).

### N-007 What the strategy takes on

- **Areas:** AGT
- **Raised:** 2026-10-09, `spec-clearing.md` section 3 and `spec-scale.md` section 4.
- **Note:** Order sizing is the strategy's job: the market rejects an unaffordable
  order whole and never clamps it (`spec-clearing.md` 3.2). One order per
  commodity per tick (3.1.1). A hard liquidity constraint, and a consumption rule
  that saves when wealth is low (roadmap, AGT row). Define the tradable set
  explicitly and a missing ideal stock as 0 (`bazaarbot-reference.md` item 5).
  The default strategy receives the world's random source and may not keep it
  (architecture.md 7.7). Quantities of an agent of weight `w` are `w` times the
  household amounts, and a limit price is never scaled (`spec-scale.md` 4.4).
- **Closed:** 2026-10-10. Applied in `spec-agents.md`: order sizing by the strategy with a budget that never gets an order rejected (7.4, FR-AGT-044); one order per commodity per tick (7.3.3); the hard liquidity constraint and the saving rule (7.4); the tradable set explicit (3.4) and a missing ideal as target 0 (7.3.1); the strategy gets the world's random source and may not keep it (6.1.2); quantities scale by the weight and a limit price never (9.1, 9.2).

### N-014 Stability register entries to collect

- **Areas:** STB
- **Raised:** 2026-10-09; each spec left a section for it.
- **Note:** Entries are waiting in `spec-clearing.md` section 13,
  `spec-scale.md` section 10, `spec-config.md` section 12, `spec-goods.md`
  section 10 and `spec-ledger.md` section 14. The roadmap's first entries are the
  target-income labour loop, the occupation-choice cobweb, and the delays in
  production, travel and traineeship (roadmap, STB row). Also record that a cohort
  multiplies the lockstep of the target-income rule (`spec-scale.md` 5.1) and that
  whole-lot spoilage makes a sawtooth (`spec-goods.md` 4.5).
- **Entries ready (2026-10-10):** `spec-beliefs.md` section 17 has nine (price/belief/limit loop, stale price loop, cobweb, herding and lockstep including the cohort gate, cost-push ratchet, shortage with no supply, width collapse, hard floor with perishables, reference band). `spec-production.md` section 16 has its own. The register itself is not written.
- **Closed:** 2026-10-10. Applied in `spec-stability.md`: all entries waiting in `spec-clearing.md` 13, `spec-scale.md` 10, `spec-config.md` 12, `spec-goods.md` 10, `spec-ledger.md` 14, `spec-beliefs.md` 17 (nine), `spec-production.md` 16 and `spec-calendar.md` 9 are in section 4, merged where two specs named one loop; loops the other specs leave unlabelled (WEL, SKL, EST, CAR) are added; the cohort lockstep of the target-income rule (L34) and the whole-lot spoilage sawtooth (L25) are recorded. The roadmap's first entries are L14 (target-income), L36 (occupation cobweb) and L21 (production, travel and traineeship delays). Future specs add entries by the checklist of section 9 (FR-STB-002, FR-STB-003). The thirteen entries of `spec-agents.md` 18 (AGT-A1 to A13) still have to be merged into `spec-stability.md` 4.7 (see N-059).

### N-037 What the strategy must supply to beliefs

- **Areas:** AGT, PRO
- **Raised:** 2026-10-10, `spec-beliefs.md` 4.4 and 10.
- **Note:** Beliefs need three inputs per order from the strategy or production: `Stock`, `Target` (units the agent wants to hold, from its recent demand over a cover period; 0 means none) and `UnitCost` (money per unit, 0 unknown). AGT must define `Target` and the cover period. PRO supplies `UnitCost` per (region, commodity) through `internal/production.UnitCost` (`spec-production.md` 9.4: input cost of one unit, rounded up, the minimum over recipes that can run there, byproducts and labour excluded) and the root hands beliefs a plain `int64`; the two specs agree on this. AGT creates a belief for every commodity in the tradable set at creation (`spec-beliefs.md` 3.1), draws limit prices with `beliefs.Draw` or its own rule, and uses the market price window (7) for quantity favourability. Watch: "a missing ideal stock as 0" (N-007) now has two readings, a zero ideal quantity for sizing and `Target = 0` meaning no stock signal for beliefs.
- **Closed:** 2026-10-10. Applied in `spec-agents.md` 7.3: `Target` is the units to hold after this tick's upkeep, `U * CoverTicks` plus the next job's inputs for a commodity used, `ceil(Make * OutputCoverTicks)` for one only made, 0 otherwise; cover periods `CoverTicks`, `InputJobs`, `OutputCoverTicks`; the `Stock` passed to `Revise` is stock after the trades less this tick's upkeep reserve (6.4.4); `UnitCost` is passed only for a commodity the role can make (7.3.6); beliefs for the whole tradable set at creation (3.4.1). The two readings of a missing ideal are one rule (7.3.1).

### N-033 Ledger changes production needs

- **Areas:** LED
- **Raised:** 2026-10-10, `spec-production.md` 4.4.2, 7.3, 15.
- **Note:** (1) add reason `disposal` (goods sink, agent to boundary, owner PRO); (2) allow `Transform` with an empty input list or an empty output list, not both (start step = inputs only, completion = outputs only); (3) add `WearItems(account, commodity, n, units)` that wears the `n` most worn distinct items by `units` each, destroying those that reach 0, with the same semantics as `Wear` otherwise (needed for a cohort of `k` executions); (4) the invariant "goods conserved except by recipes" holds with work in progress: pending output is in no account, so the snapshot reports it.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** accept all four. Not yet applied to `spec-ledger.md`; do it in the next change that touches that spec.
- **Closed:** 2026-10-10. Applied in `spec-ledger.md`: the `disposal` reason and the goods form of `initial` (7.2), a `Transform` with an empty side (8.2.1), `WearItems` (5.4.3) and work in progress (8.2.2); FR-LED-037 to 049.

### N-053 Ledger query for expiring lots

- **Areas:** LED
- **Raised:** 2026-10-10, `spec-agents.md` 7.3.5.
- **Note:** The default `Trade` offers units that spoil at the end of the tick. It needs a read-only query `Expiring(account, commodity, tick)`; `spec-ledger.md` 5.3.2 keeps an earliest-expiry hint per account but no per-commodity query.
- **Closed:** 2026-10-10. Applied in `spec-ledger.md` 5.3.3: the read-only `Expiring(account, commodity, tick)`.

### N-050 Spending classification for the welfare income

- **Areas:** WEL, AGT
- **Raised:** 2026-10-10, `spec-agents.md` FR-AGT-057.
- **Note:** `spec-welfare-and-wages.md` 2.1 nets inputs out of income. AGT's `Intent.Purpose` (`Upkeep`, `Input`, `Tool`, `Other`) classifies each bid; the world totals money received and money spent by purpose per agent per tick. A commodity that is both an input and upkeep is classed as an input. WEL decides what is netted (and whether tools are capex or recipe cost) and amends its 4.3 per N-035.
- **Closed:** 2026-10-10. Applied in `spec-welfare-and-wages.md` 2.1: `income_t` nets money spent on bids with the purpose `Input` and the wear cost of tools; upkeep, tool purchases and other spending are not netted; a tool is capital whose use is netted as wear (2.1.3); relief and in-kind count as income, a starting-money grant does not; a commodity that is both input and upkeep is classed as input and the ratio is understated by at most the upkeep bought of it (2.1.5, conservative).

### N-055 The belief cost ceiling against the equilibrium price

- **Areas:** BEL, PRO, STB
- **Raised:** 2026-10-10, `spec-viability.md` 8.2.3 and 15.
- **Note:** `UnitCost` (`spec-production.md` 9.1) excludes labour. At zero-profit prices the seed is 3.7, 9.7, 8.4 and 19 times unit cost for flour, dough, loaf and a tool in the prototype, against `CostCeiling` of 3 times (`spec-beliefs.md` 3.2.2). BEL clamps a belief to the ceiling, so sellers could not price at the wage and the seed would be clamped at creation. Options: count recipe labour at the wage in `UnitCost`; raise `CostCeiling`; drop the ceiling where the labour share is high. The solver reports `SeedAboveCostCeiling` meanwhile.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** count labour in `UnitCost` at the observed wage (`spec-welfare-and-wages.md` already defines an observed wage), measured from the previous period so there is no circularity. Reason: Lengnick's price bounds are multiples of marginal cost, and his marginal cost includes wages; the ceiling then stays near the equilibrium price instead of far below it. Cost: with labour counted, `CostFloor` 10 000 binds at the zero-profit price, which makes the freeze of N-039 more likely, so the floor must be re-examined together. Not applied: it changes `spec-production.md` 9 and `spec-beliefs.md` 3.2.2, and needs a revision pass of both specs.
- **Closed:** 2026-10-10. Resolved in design. `spec-production.md` 9 counts labour at the pooled observed wage of the previous tick (FR-PRO-035 to 037, 047 to 049); `spec-beliefs.md` 3.2.2 sets `CostFloor` 7 000 and `CostCeiling` 30 000 on the full cost, because a 100% floor freezes markets with perishables (`spec-stability.md` 6.2 run 5); `spec-viability.md` 8.2.3 and 15 re-ran the seed check against the full unit cost: ratio 1.000, minimum 0.991 (whole-unit rounding). A starting wage per region closes the first-ticks gap (`WarmStart.Wages`, `spec-viability.md` 8.2.5). Residual: the wage loop has gain `CostFloor / 10 000` times the labour share (STB-L10).

### N-057 The worked examples of `spec-production.md` 12 are not viable as written

- **Areas:** PRO, REF
- **Raised:** 2026-10-10, `spec-viability.md` 15.
- **Note:** Example A (6 grain per 120-tick job) needs 73 times the available labour; example B needs 12 times. They are packs with placeholder numbers, but FR-PRO-046 says they must "run to a stable state". Rescale them before REF or any test uses them. A set that passes (with invented completing recipes) is in `spec-viability.md` 15. Rule of thumb from the runs: the output per labour tick of each step of the chain must exceed the household's need by the depth of the chain.
- **Closed:** 2026-10-10. Applied in `spec-production.md` 12: both examples rescaled to the viable set of `spec-viability.md` 15 (invented completing recipes labelled as such); `spec-viability.md` confirms both pass check V3.

### N-059 Merge the AGT entries into the stability register

- **Areas:** STB, AGT
- **Raised:** 2026-10-10, `spec-agents.md` 18 and `spec-stability.md` 4.7.
- **Note:** The thirteen entries `AGT-A1` to `AGT-A13` of `spec-agents.md` 18 were written against the checklist of `spec-stability.md` 9 while that spec was being written. Merge them into `spec-stability.md` 4.7 and remove the AGT hook row. D24 and D09 become specified (`spec-agents.md` 7.1.5 and 7.1.2); D26 (survival rule, not a damper of L14), D27 (budget stack and saving rule) and D28 (proportional work share) are new; scenarios S19 and S20 are new. D13 (`MinWorkTicks`) is not adopted; WEL decides.
- **Closed:** 2026-10-10. Applied in `spec-stability.md` 4.7: the thirteen AGT entries merged (L44 to L51; five merged into existing loops), D24 and D09 recorded as specified, D26 to D28 added, S19 and S20 added; D13 is adopted by WEL as `MinWorkShare` and D29 is the derived window.

### N-044 The stock signal alone is an integrator

- **Areas:** BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.1.
- **Note:** A price rule driven only by the stock ratio gives a sustained oscillation of 6% to 29% of the price in the model; the amplitude scales with `BeliefStep`. The evidence rule is the proportional term that stops this, and the combination is stable in the model, so keep both and re-test when the evidence rule is revised (N-042). Test: S01, S04.
- **Closed:** 2026-10-10. Applied in `spec-beliefs.md` 5.5: both signals are needed. Re-tested with the revised evidence rule in `spec-stability.md` 6.2 run 2b (stock-only recovers in 98 ticks, combined in 30).

### N-049 Order reasons the strategy contract adds

- **Areas:** CLR, PRO
- **Raised:** 2026-10-10, `spec-agents.md` 6.5.
- **Note:** `not_tradable` (an intent outside the agent's tradable set) joins the table of `spec-clearing.md` 3.2; `not_on_role` (a job request off the role's list) joins the refusal reasons of `spec-production.md` 4.2, with a `Debug` log. Both are constants in `internal/logkeys`.
- **Closed:** 2026-10-10. Applied: `not_tradable` in `spec-clearing.md` 3.2 (FR-CLR-030, checked after `unknown_commodity`, pseudo-agents exempt) and `not_on_role` in `spec-production.md` 4.2 and 14.2 (FR-PRO-008, FR-PRO-048, checked first).
- **Update (2026-10-12):** the reasons are exported constants of the root package (`event.go`), not constants of `internal/logkeys`, which holds the key strings only (`spec-observability.md` 3.2.1; architecture.md 2.9). The line above said "Both are constants in `internal/logkeys`" and is read that way.

### N-048 The tradable set gives "tradable" a meaning

- **Areas:** GDS, CUR, CFG
- **Raised:** 2026-10-10, `spec-agents.md` 3.4.
- **Note:** `spec-goods.md` 6.4 and `spec-currency.md` 3.1 speak of the cheapest "tradable" commodity without defining it. `spec-agents.md` 3.4 defines it: a commodity in the tradable set of at least one role. Edit both specs' wording and FR-GDS-002.
- **Closed:** 2026-10-10. Applied: "tradable" is defined by `spec-agents.md` 3.4 and used in `spec-goods.md` 6.4 and FR-GDS-002, `spec-currency.md` 3.1 and `spec-config.md` 7.4.1.

### N-041 Edits to other specs owed by PRO and BEL

- **Areas:** LED, CFG, CLR, CUR, GDS, SKL, WEL, SCL, CAL, AGT, WLD, EST, PRO, BEL, STB, VIA
- **Raised:** 2026-10-10, handoffs of the workers that wrote `spec-production.md` and `spec-beliefs.md`.
- **Note:** The two new specs need changes in specs that already exist. None is made yet; make each in the next change that touches that spec, and close this note when all are done.
  - `spec-ledger.md`: 7.2 add the `disposal` reason; 8.1 allow an empty side on `Transform`; 5.4 add `WearItems` (N-033).
  - `spec-welfare-and-wages.md`: 4.3 working tick = job in progress (N-035); a note on the window against job length (N-032).
  - `spec-skill-barriers.md`: 2.1 and 4.2 refer to `spec-production.md` 4.4 and 8 (trainee-eligible recipes have no tool; trainee output = `TraineeYield`; breakage is not random); 2.4 reword (N-039).
  - `spec-config.md`: 4.1 bounds `MaxRecipeTicks`, `MaxYieldBP`, `MaxRequires`, `MaxDraws` (provisional); 4.2 the recipe checks of `spec-production.md` 10.3; 3.1 a `Yields` section; 7.3 and 7.4 the sentinels `ErrCostShares`, `ErrToolNotDurable`, `ErrEligibleHasTool`, `ErrUnknownSeason` and the PRO rules, and the 15 BEL parameters and rules (FR-BEL-027 to 029); 7.5 a `Sited` source with `Draws` counts as sourced; 6.2 the example keys; 7.4 name `BeliefStep`.
  - `spec-clearing.md`: 5.4 the seed price; 7.3 and FR-CLR-025 the outcome goes to beliefs once per admitted order; 5.4 or 7.1 who keeps the price window; open question 1 is also true of the paper's pseudocode; 9.2 cross-reference to the BEL `Warn`.
  - `spec-currency.md`: 3.1 name `BeliefStep` (floor `ceil(10 000 / BeliefStep)`; the default 200 gives 50).
  - `spec-goods.md`: 6.3 the effective reference price is also the default seed price and the centre of the reference band; open question 4 is answered.
  - `spec-scale.md`: open question 3 is settled for production.
  - `spec-calendar.md`: 9 the "season shorter than a delay" entry does not arise for production.
  - Future specs: AGT (N-034, N-037), WLD (N-009, N-036), OBS (N-012, N-035, N-040), STB (N-014), VIA (N-036, N-038).
  - `rules/observability-and-logging.md` 4.3: `RecipeExecuted` at completion with new fields, `RecipeAbandoned`, and later `BeliefsUpdated`. A rules change: it needs a new read-receipt tag and is not made here.
- **Added by round 2 (2026-10-10).** Further edits owed, none yet made:
  - `spec-clearing.md` 3.2, 9: the rejection reason `not_tradable` (N-049); 7.3 and FR-CLR-025: the outcome goes to `Revise` with the market report once per admitted order.
  - `spec-production.md` 3.3, 4.1, 4.2, 14.2: the strategy names the recipe through `JobRequest`; refusal reason `not_on_role`; `Role.Recipes` is owned by `spec-agents.md` 3.3; 12 and 18: rescale the worked examples (N-057), question 2 is answered by `spec-viability.md` 8.5; 9: the decision on labour in `UnitCost` (N-055).
  - `spec-beliefs.md` 3.2.2 (cost ceiling, N-055), 4.4, 10.1, 10.5 (`Target` and `Stock` are defined in `spec-agents.md` 7.3 and 6.4.4; the method that receives the outcome is `Revise`), 5.4 (the evidence move, N-042), 5.5 and 17 (the stock rule alone is an integrator, N-044; section 17 can point to STB-L01, L03, L21, L13, L10, L05, L07, L09, L08), 8.4 (the table is `Config.WarmStart.Prices`).
  - `spec-ledger.md` 7.2, 5.2.3, 7.4, 8.3: the goods form of `initial` at construction, lots with negative `Born`, a construction bucket in the flow totals (N-056); the query `Expiring` (N-053).
  - `spec-estates.md` 2.3, 3: `StartMoney` is `Role.StartMoney`; a later agent receives no goods; removal mechanics are `spec-agents.md` 3.6; pool seeding by `WarmStart.Pools` (`spec-viability.md` 8.3.3).
  - `spec-scale.md` 4.1, 4.4, 4.8, 7.2: cross-references to `spec-agents.md` 3.7, 5, 9; the scale-consistency check can use a warm start.
  - `spec-welfare-and-wages.md` 2.1, 4.3, 5, 5.1, 7: `cost_t` is the effective upkeep of `spec-agents.md` 5.2; income netting by `Purpose` (N-050); 5.1's "left to the strategy" is answered by `spec-agents.md` 7.1.2; WEL decides D13; the role economics check fails in a closed upkeep-only economy (N-054); note that labour supply is unit-elastic (`spec-stability.md` 6.1).
  - `spec-skill-barriers.md` 4.2: a trainee's upkeep is consumed from the trainer's account (`spec-agents.md` 5.7).
  - `spec-currency.md` 5.1, 5.3: the effective upkeep and `BasketTolerance`. `spec-goods.md` 6.4, FR-GDS-002, FR-GDS-003: "tradable" is defined by `spec-agents.md` 3.4 (N-048).
  - `spec-config.md` 3.1, 4.1, 4.2, 6.2, 7.3 to 7.6, 8, 9.2: the `WarmStart` section and its hash, bounds `MaxViabilityPivots`, `MaxViabilityCells`, `MaxLanes`, `MaxUpkeepItems`, the sentinels `ErrNotViable`, `ErrWarmStartPrice`, `ErrWarmStartJob`, `ErrUnknownStrategy`, `ErrBasketMismatch`, the viability rules and warnings, FR-CFG-033 satisfied by FR-VIA-004 to 008 and 011, the cycle-gain check (FR-STB-017).
  - `spec-calendar.md` 4.5, 9: state the periodic extension used by `spec-viability.md` 8.5.2 (optional).
  - The sources' own register sections (`spec-clearing.md` 13, `spec-scale.md` 10, `spec-config.md` 12, `spec-goods.md` 10, `spec-ledger.md` 14, `spec-calendar.md` 9, `spec-production.md` 16): a line "merged into `spec-stability.md` section 4" at the next touch.
  - `rules/architecture.md` 3.5 (allow `math/big` in `internal/steady`, N-058) and 4.3 (the `Strategy` row could say "Work, Trade and Revise") and `rules/observability-and-logging.md` 4.3 (`UpkeepShortfall`, `AgentSpawned` fields): rules changes, each needs a fresh read-receipt tag; not made.
- **Closed:** 2026-10-10. All owed edits were made in round 3: ledger, estates, scale and calendar; clearing, goods, currency and carriage; welfare and wages and skill barriers; config; production and beliefs; stability; and the agents and viability sweeps. Rules changes remain and are N-077.

### N-060 A starting wage for the cost anchor

- **Areas:** VIA, PRO, AGT, WEL
- **Raised:** 2026-10-10, `spec-production.md` 9.3 and 18.11, `spec-agents.md` 19.
- **Note:** The cost anchor is 0 while a region has no observed wage (`spec-production.md` 9.1.1), so beliefs have only the reference band in the first ticks.
- **Closed:** 2026-10-10. `spec-viability.md` 8.2.5 adds `WarmStart.Wages` (a starting wage per region from the labour-row dual, used only until the first working household-tick is observed; FR-VIA-049 to 052); `spec-welfare-and-wages.md` 4.3 exposes the observed wage per working household-tick.
- **Update (2026-10-11, J5):** "used until the first working household-tick is observed" is now a pure function of the welfare rings (the starting wage is used whenever the pooled figure is unknown), not a latch that no section would save (`spec-production.md` 9.3, `spec-viability.md` 8.2.5, FR-VIA-052; `spec-checkpoint.md` 4.5 F5).

### N-062 The derived welfare window

- **Areas:** WEL, CFG, PRO, STB
- **Raised:** 2026-10-10, `spec-welfare-and-wages.md` 2.3.3.
- **Note:** The effective window is `T = max(WelfareWindow, WindowCycles * longest Duration)`. CFG needs `WindowCycles`, `MaxWelfareWindow` and the effective window in the overflow check; PRO's `job_longer_than_window` becomes obsolete; STB's L19 damper is the window.
- **Closed:** 2026-10-10. Applied in `spec-config.md` (3.7, 4.1, 4.2, 7.4; FR-CFG-050 to 054), `spec-production.md` 10.4 and FR-PRO-040 (the warning is replaced by the informational `window_extended`) and `spec-stability.md` (D29).

### N-063 Estate ask anchor

- **Areas:** EST, SKL
- **Raised:** 2026-10-10, `spec-skill-barriers.md` 2.7.
- **Note:** `spec-skill-barriers.md` 2.7 anchored the estate's ask to the region's recent average price while `spec-estates.md` 2.7.3 anchors it to the effective reference price.
- **Closed:** 2026-10-10. Decided by the coordinating agent: the effective reference price, times remaining over full durability for a durable, less the per-tick markdown. It is deterministic and does not couple the estate to belief state. A market-anchored rule (the price-window mean, then the cost floor, then the reference price) is an experiment for later. Recorded in `specs/README.md` decisions.

### N-066 The cycle-gain warning formula

- **Areas:** CFG, STB
- **Raised:** 2026-10-10, `spec-stability.md` 4.1 (L10).
- **Note:** `spec-config.md` 7.6 item 3 computed the cost-loop gain from inputs only, but labour is now in the unit cost.
- **Closed:** 2026-10-10. Restated in `spec-config.md` 7.6 (FR-CFG-044): the material loop gain is `CostFloor / 10 000` times the full-cost share of the cycle's own inputs, which at the solver's prices is `q_in / q_out`; the wage loop is flagged by `cost_floor_at_full_cost`. `spec-stability.md` FR-STB-017 follows.

### N-069 Warning rule names

- **Areas:** CFG, CAL, SKL
- **Raised:** 2026-10-10, `spec-config.md` 7.4.2.
- **Note:** Three warnings were stated by other specs without a name: `inexact_hour`, `season_shorter_than_tick` (`spec-calendar.md` 7.3) and `tool_unaffordable` (`spec-skill-barriers.md` 2.3).
- **Closed:** 2026-10-10. Adopted in `spec-calendar.md` 7.3 and `spec-skill-barriers.md` 2.3 under the names given by `spec-config.md` 7.4.2.

### N-001 Households are the consumers; avoid the word

- **Areas:** DEM
- **Raised:** 2026-10-09, in a question about simulating consumers.
- **Note:** There is no separate consumer population. Every agent is a household
  that earns, pays upkeep and buys wants (`spec-currency.md` 5.5; overview).
  `spec-agents.md` owns upkeep orders and the consumption rule. `spec-demand.md`
  (version 1) must decide whether it needs households that do not produce
  (people on unearned income) or whether classes are only roles. The wage and
  demand loop (roadmap, "Macro is measured, never set") depends on this.
- **Watch for:** "consumer" in these docs means the *library consumer* (a game or
  server). Call the household a household or the demand side.
- **Update (2026-10-10, AGT):** a class of non-producing households is a role with an empty `Recipes` list (`spec-agents.md` 3.3.1). Upkeep, the consumption rule and the saving rule are in `spec-agents.md` 5 and 7. Spending beyond upkeep does not exist in version 1 (11.7); DEM must supply it (see N-054).
- **Closed:** 2026-10-10. Applied: a class of non-producing households is a role with an empty `Recipes` list (`spec-agents.md` 3.3.1, `spec-demand.md` 3.3); its income is a transfer named by another spec (`spec-demand.md` 7.3, warning `class_without_income`).

### N-052 A price-responsive flow for a staple

- **Areas:** STB, DEM, STY
- **Raised:** 2026-10-10, `spec-agents.md` 11.6.
- **Note:** A staple's demand elasticity is 0 to 1. If S04 or S16 shows a pinned market that the halt and exit do not cure, add the rationing of tier 1 above a dear price (11.6.1) or an elastic demand in DEM.
- **Closed:** 2026-10-10. Rejected as framed (`spec-demand.md` 17): tier-1 rationing makes hunger a choice and is an unsupported guess. DEM supplies elastic demand for discretionary goods, including an elastic increment on a staple listed as a want item. Reopen condition: S04 or S16 shows a pinned staple market that the halt, exit and the work floor do not cure; then raise a new note for AGT (upkeep alternatives).

### N-054 The closed economy fixes the mean welfare ratio at 1

- **Areas:** WEL, DEM, STB, MON
- **Raised:** 2026-10-10, `spec-viability.md` 4.7.
- **Note:** With no money faucet or sink and upkeep as the only purchase, household money is conserved, so total income equals total upkeep spending and the household-weighted mean welfare ratio is 1 whatever `TargetRatio` is. The default 15 000 cannot be met by all roles; the target-income rule never rests; the surplus of the steady state is produced and unsold. The role economics check of `spec-welfare-and-wages.md` 7 fails for every role in such a world. DEM, MON or EXT must supply the spending or money that lets a ratio above 1 exist. Same loop as N-028 from the other side. STB: record it, with a test that runs the default target with no DEM and asserts a bounded glut.
- **Answer (2026-10-10, AGT):** version 1 has no spending beyond upkeep; the glut is accepted and bounded by the production halt (`spec-agents.md` 11.7, FR-AGT-070). Open for WEL: whether the default `TargetRatio` stays 15 000 in a world without DEM, MON or EXT.
- **Update (2026-10-10):** WEL made the role-economics check a report, never an error (`spec-welfare-and-wages.md` 7.1). Decided by the coordinating agent: the default `TargetRatio` stays 15 000. In a world whose only demand is upkeep the mean ratio is 1, agents work at the floor-bounded maximum and the glut is bounded by the production halt; lowering the default would hide the missing demand, money-supply and external-sector specs. Revisit when DEM, MON or EXT exist. STB records it as STB-L46 with scenario S19.
- **Closed:** 2026-10-10. Answered by `spec-demand.md` 7: with wants the closed economy's mean welfare ratio is `1 + W/U` (model: 1.53 at target 15 000, against exactly 1.000 with upkeep only), up to `1 + Wmax/U`; no money faucet is needed; the money needed is the reserve plus about `e*` per household. The default `TargetRatio` stays 15 000 with the module on or off (`spec-welfare-and-wages.md` 7.2); the warning `target_above_demand` compares a pack's target with its own ceiling. STB-L46 stays for a pack without the demand module.
- **Update (2026-10-10, MON):** with MON the answer stands: a balanced pair leaves the mean ratio unchanged, an unbalanced faucet raises it only while the supply grows (`spec-money-supply.md` 8.2, 9.1); the default `TargetRatio` stays 15 000, so "revisit when DEM, MON or EXT exist" is answered for MON.
- **Update (2026-10-10, POP):** the closed ceiling matters for the population: with upkeep as the only demand a fed household reads a mean ratio near 10 000, so the Malthusian fixed point is at chronic hunger or the population dies out (`spec-population.md` 5.3, POP-A7).
- **Update (2026-10-11, IND):** the claim is stated on the aggregate ratio (the sum of income over the sum of cost), not on the mean of ratios, and the two differ by the covariance of the ratio with the cost (`spec-indicators.md` 7.7.2, 18.9). It is not measurable from a snapshot until the income fields of N-130 exist, so `HouseholdIncome` reports `OK` false. STY states the identity as M03 and the ceiling as M04 (`spec-stylised-facts.md` 5.5).
- **Update (2026-10-13, EMP, CRD, LND, SAV, INS, CAP):** the words are scoped. A transfer between households redistributes and creates nothing, so the aggregate identity of the note holds: interest (CRD 11.1), rent (LND 7.5), a premium and a payout (INS 11.1) and a wage (EMP 11.4.3) change the ratios of the two sides, not the aggregate. Saving is not a transfer and is not income: a goal or a buffer withholds buffers and wants and so lowers the mean ratio (`1 + W / U` over a window of unchanged stocks as DEM 7.1 and MON 8.2 give it, `1 + (W + Other) / U` with SAV's own term for a window of changing stocks; `spec-saving-and-investment.md` 8.2.1). A project's inputs are netted at the buyer and are income at the seller, so they add nothing to the aggregate ratio; the builder's own ratio is another matter, it falls for the length of its job (N-171). With employers the split by role is EMP-A3: the workers' ratio is bounded by the employers' appetite, `h* = E u_e / ((1 - phi) a)` with employers who eat only their upkeep. `N_t` of the insurance spec is income and is excluded from the observed wage. The letters of the welfare record are WEL's (EMP `Y_t`, `V_t`; CRD `I^in`, `I^out`; FIS `X_t`; interest received, `N_t` and rent received are folded into the signed `G_t`), one rule per flow.

### N-047 What the reference chain must contain

- **Areas:** REF, STY, VIA, STB
- **Raised:** 2026-10-10, `spec-stability.md` 8.
- **Note:** To build scenarios S04 to S20 the chain needs a self-employed staple role with demand elasticity below 1, a perishable, a recipe cycle (for the cost-push gain), a long recipe or trip (the cobweb), a durable with a replacement cycle, a role with a job longer than the welfare window, and a region that can lose its income (pool, relief, failure ladder). STY sets the bands and the regime thresholds; VIA staggers the initial jobs (S18).
- **Update (2026-10-10):** scenarios now run to S22 (S19 closed economy, S20 reserve, S21 seeds against the cost band, S22 warm against cold start). The chain also needs a world with upkeep as the only demand, a role with a job longer than the base window, and a warm start. The three-region run of VIA used wood yields by region that `spec-production.md` 12.1 does not list (N-070).
- **Closed:** 2026-10-10. Applied in `spec-reference-chain.md` 3 and 4: features C01 to C17 (a self-employed staple role, a perishable, a seed cycle, a 60-tick job, durables, a job longer than the window, a famine variant with estates, a warm start, an upkeep-only world, trainee recipes above and below break-even); the scenarios are assigned to packs in `spec-stability.md` 8. Bands are STY's.

### N-070 Regional yields of the three-region run

- **Areas:** VIA, REF
- **Raised:** 2026-10-10, `spec-viability.md` 15.
- **Note:** The three-region run used wood yields by region that `spec-production.md` 12.1 does not list. REF adds them when it builds scenario A3.
- **Closed:** 2026-10-10. The yields of the three-region run are listed in `spec-reference-chain.md` 4.4.3.

### N-034 Role-to-recipe membership and the choice of recipe

- **Areas:** SPN
- **Raised:** 2026-10-10, `spec-production.md` 3.3, 4.1, 5.1.2.
- **Note:** `Role.Recipes` (sorted list of recipe IDs) is a field AGT owns. An agent has one job at a time, starts only when free, and the `Strategy` names the recipe and an optional maximum number of executions. A role change abandons the job (reason `role_changed`). The default rule for choosing among executable recipes (for example by expected net value using `UnitCost` and the expected price) is AGT's. The spawner must expect a job to be lost when it moves an agent to another role.
- **Update (2026-10-10, AGT):** `Role.Recipes` is in `spec-agents.md` 3.3; the default recipe choice is the eligible recipe with the greatest net value per tick at the agent's belief centres, ties to the lower ID, net value positive or a source (7.1.4); the world refuses a request off the role's list with `not_on_role` (6.5.1). The spawner must expect a job to be lost when it moves an agent to another role.
- **Closed:** 2026-10-10. Applied: `Role.Recipes` in `spec-agents.md` 3.3; the default recipe choice in 7.1.4; a role change abandons the job with `role_changed` (`spec-spawner.md` 4.2).

### N-064 Traineeship is for weight 1

- **Areas:** SPN, SKL, SCL
- **Raised:** 2026-10-10, `spec-skill-barriers.md` 4.1.1.
- **Note:** A trainer and a trainee are agents of weight 1; a role with `TrainingTicks` above 0 needs groups of scale 1 (validation `traineeship_needs_weight_one`). SPN: the entry rule must not offer a traineeship to a cohort.
- **Closed:** 2026-10-10. Applied in `spec-spawner.md` 4.4 and FR-SPN-009 (the world refuses `Apply` for weight above 1, `needs_weight_one`) and `spec-skill-barriers.md` 4.1.1.

### N-073 A weight rise the pool cannot fund

- **Areas:** SPN, POP, EST
- **Raised:** 2026-10-10, `spec-estates.md` 2.8.1 and open question 2.
- **Note:** A weight rise draws money from the estate pool; refuse it or apply it partially when the pool is short. Goods drawn for a rise are unspecified.
- **Closed:** 2026-10-10. Applied for entry in `spec-spawner.md` 4.3 and 7.3 and `spec-estates.md` 2.8.1: partial, in whole households, money only, within the whole-pool budget `EntryPoolShare`; entrants get no goods. POP confirms for births.
- **Update (2026-10-10, POP):** confirmed for births with one change: each born household brings `Role.StartMoney * BirthStake / 10 000` (`spec-population.md` 4.3), with the ledger reason `births` for a new agent as well as for households added to an existing one. In a closed region the stake caps the population at M / stake (`spec-population.md` 5.5, model 2).

### N-058 Exact big integers in the solver

- **Areas:** ALL, CODE
- **Raised:** 2026-10-10, `spec-viability.md` 6 and open question 7.
- **Note:** `internal/steady` uses `math/big` (standard library). architecture.md 3.5 bans floats and requires checked arithmetic where `int64` can overflow; unbounded integers cannot overflow. A sentence in 3.5 would remove doubt. That is a rules change and needs a fresh read-receipt tag; not made.
- **Closed:** 2026-10-10. Applied in `rules/architecture.md` 3.5: `math/big` is allowed in `internal/steady` only, with the conversion rules and no `math/big` type in anything exported.

### N-077 Rules changes owed by the specs

- **Areas:** ALL, CODE
- **Raised:** 2026-10-10, handoffs of rounds 2 and 3.
- **Note:** None is made; each needs a fresh read-receipt tag. (1) `rules/architecture.md` 3.5: allow `math/big` in `internal/steady` (N-058). (2) 4.3: the `Strategy` row could say "`Work`, `Trade` and `Revise`". (3) `rules/observability-and-logging.md` 4.3: `RecipeExecuted` at completion with new fields, `RecipeAbandoned`, `UpkeepShortfall`, `AgentSpawned` fields and the removal event's `Cause`, and later `BeliefsUpdated`; 3.3.2: the log keys `path`, `rule`, `config_hash`, `region`, `bound`, `agents`, `households`, `unfed`.
- **Closed:** 2026-10-10. Applied in two edits: `rules/architecture.md` 3.5 and 4.3, and `rules/observability-and-logging.md` 3.3.2 (log keys), 4.3 (events including `WeightChanged`, `RoleChanged`, `EntryLimited`, `WantShortfall`) and 5.3 (metrics including `household.unfed`, `role.changes`, `want.unfunded`), with fresh read-receipt tags. Remaining conflicts of the rules edit are N-088.

### N-068 Model scripts are outside the repository

- **Areas:** STB, VIA, CODE
- **Raised:** 2026-10-10, `spec-stability.md` 15.
- **Note:** The numbers of `spec-stability.md` 6, `spec-viability.md` 15 and the belief revision come from small Docker Python models in `/tmp/economy-boss/` (`stb/`, `via/`, `pb/`), which will vanish with that directory. Decision for the human: archive them under `doc/specs/` (with a documentation index row; the project would then hold Python) or accept that the specs' descriptions are the record.
- **Closed:** 2026-10-10. Decided by the human: archive. The scripts are in `doc/specs/models/` (`stb/`, `via/`, `pb/`, `agt/`) with a README and a documentation index row; the specs cite the new paths. The scripts of the demand, spawner and reference-chain specs are still in `/tmp/economy-boss/` (see N-089).

### N-024 Which of FIS, INT and PLC are in the first release

- **Areas:** FIS, INT, PLC
- **Raised:** 2026-10-10, `spec-roadmap.md` Open 1.
- **Note:** `spec-fiscal.md`, `spec-integration.md` and `spec-policy.md` are
  candidates to defer past the first release, and the decision is open. CHK is
  core (ADR 0002). SCL and DEM change how agents work and are costly to retrofit,
  so they stay in the core. Settle it against `vision.md` section 6 (version 1
  scope). Note that INT carries the player agent (N-022) and external shocks, so
  deferring it also defers those.
- **Answer (proposed, 2026-10-10):** follow `vision.md` section 4. FIS and PLC are
  later. INT is in version 1 only as shocks and a player agent, because STY needs a
  shock (a failed harvest) and the first consumer needs a player. A game can inject
  taxes as shocks until FIS exists. It binds only when the vision is accepted and
  recorded in an ADR; close this note then.
- **Closed:** 2026-10-10. Decided by the human (ADR 0003): FIS and PLC are later; INT is in version 1 only as shocks and a player agent.

### N-008 Weight changes, cohort failure and the estate

- **Areas:** POP
- **Raised:** 2026-10-09, `spec-scale.md` 4.8 and open question 4, `spec-ledger.md`.
- **Note:** A cohort whose money reaches 0 fails whole. Recommended: reduce its
  weight by the share it cannot support before removing it (`spec-scale.md` 5.1),
  inside a failure ladder (change role, relief, then death). A weight change is a
  ledger operation with reason `split`, `merge`, `births` or `starvation`, scaling
  holdings in proportion with the remainder going to the estate
  (`spec-scale.md` 4.8; `spec-ledger.md` 7.2). Account IDs are never reused, so a
  run with many deaths accumulates closed accounts (`spec-ledger.md` open
  question 6; N-020). The estate pseudo-agent is not an agent: it has no weight and
  is not counted in agents or households (`spec-estates.md` 2.7.4).
- **Update (2026-10-10, AGT):** a cohort does not fail by itself; an agent with no money still exists and goes unfed (`spec-agents.md` 3.6.1). AGT supplies `Unfed` and `Hunger` (5.4, 5.5) and the effect of a weight change on agent state (3.7). The ladder and the partial weight reduction by the unfed share are SPN and POP's.
- **Update (2026-10-10, EST):** weight changes pass money through the estate (`spec-estates.md` 2.8.1); a rise the pool cannot fund is open (N-073). The ladder is SPN and POP's.
- **Update (2026-10-10, SPN and EST):** applied for SPN and EST in `spec-spawner.md` 5 (the ladder, reduction by the unfed share, the estate path) and `spec-estates.md` 2.8. POP owns the demographic rates (`PopDelta` hook, `spec-spawner.md` 8; the ledger reason `deaths`).
- **Update (2026-10-10, POP):** applied for POP in `spec-population.md` 4.7: the ledger reasons are `births` (extended to every household born through the module) and `deaths` (new, a partial reduction; a death that empties an agent is a removal with `estate_money` and `estate_goods` and cause `population`). The ladder's `starvation` reduction stays SPN's.
**Closed:** 2026-10-10. Applied in `spec-population.md` 4.7 (reasons) and 4.5 (no household counted by two rules).
- **Update (2026-10-13, CRD):** a reduction of weight changes no claim: the survivors owe what the agent owed, and the waterfall of claims is at a removal only (`spec-credit.md` 9.3, CRD-A7 = L165).

### N-010 Regions

- **Areas:** REG, MIG
- **Raised:** 2026-10-09, from GDS, EST and LED.
- **Note:** `spec-regions.md` owns the per-region override of a commodity's
  reference price (`spec-goods.md` 6.3) and decides whether a region holds one
  settlement or several (roadmap, Open 3). Each region creates its estate
  pseudo-agent and account (`spec-estates.md` 2.7). Changing an account's region
  is an explicit operation (`spec-ledger.md` 3.5) and its interaction with goods in
  transit is open (`spec-ledger.md` open question 8; `spec-migration.md`).
- **Update (2026-10-10, TRN):** goods in transit are in Transit ledger accounts, so relocating an agent moves no cargo (`spec-transport.md` 7; `spec-regions.md` 6.5.1). TRN registers the `region_bound` refusal `open_shipment` (`spec-regions.md` 6.5.2) when the module is on: an away traveller, or the shipper or consignee of a shipment in flight, is refused by the public relocation with `ErrRegionBound`. At arrival TRN changes the traveller account's region by the ledger-level `Relocate` with cause `transport` and emits REG's `AgentRelocated`; it does not use `NoRegion`, `depart` or `arrive` in version 1 (the traveller stays in the origin region while away). It uses REG's `Distance(from, to)` (`(int64, bool)`, 4.3). Answers `spec-ledger.md` open question 8 for transport (`spec-transport.md` 1.2, 5.5.3).
**Closed:** 2026-10-10. Applied in `spec-regions.md` (REG): the reference-price override is `RegionPrices` (5.2), one case of a single override mechanism (5.1); one settlement per region (3.1); each region's estate account and pseudo-agent are created in region order before any household, the pseudo-agent's agent ID equal to the `RegionID` (3.6); changing an account's region is `relocate` (6): ordered refusals, no transfer and no reason, relocation totals in the ledger, the event `AgentRelocated`. The hook for goods in transit is 6.5: cargo is never the agent's account, a spec that ties an obligation to a region registers a `region_bound` refusal (TRN registers `open_shipment`), and `NoRegion`, `depart` and `arrive` are reserved and unused in version 1. What MIG, PRO, SKL and CAR must do with a relocation is N-090.

### N-026 One settlement per region, or several?

- **Areas:** REG, SET
- **Raised:** 2026-10-10, `spec-roadmap.md` Open 3. See also N-010.
- **Note:** One settlement per region is simpler and makes `spec-settlements.md` a
  thin layer; several lets a region have a large settlement and its smaller ones.
  Decide when writing `spec-regions.md`, before `spec-settlements.md`, because it
  fixes where market, estate and size class live.
- **Answer (decided, 2026-10-10):** one settlement per region, and a region is one
  market. A town and its hamlets are separate regions joined by short edges, which
  costs little because clearing cost grows with regions and commodities, not
  households (`spec-scale.md`). `spec-settlements.md` (later) then only gives a
  region a size class from its households, which is the thin layer its roadmap row
  describes. Victoria 3 treats the market as the unit and varies a state's access to
  it instead of splitting prices inside a state (dev diary 9, snippet only). Apply
  this in `spec-regions.md` and close the note there.
- **Update (2026-10-10):** ADR 0003 decision 4 records the answer. Apply it in `spec-regions.md` and close this note there.
**Closed:** 2026-10-10. Applied in `spec-regions.md` 3.1.3 and 3.4: a region is one settlement and one market, and a town and its hamlets are separate regions joined by short edges (example C, 12.3). The cost is recorded as REG-A2 in the register (a small market is a thin market, and each hamlet has its own pool). `spec-settlements.md` (later) only adds a size class to a region.

### N-029 How deep a review of other economy libraries?

- **Areas:** POP, EVD
- **Raised:** 2026-10-10, `spec-roadmap.md` Open 6. See also N-015, N-018.
- **Note:** Open before `spec-agents.md` and `spec-population.md`. Only bazaarBot
  has been read in full. abcEconomics, Prehistoric, the Space Project, ASSUME and
  Crusoe are known from search snippets, and two documentation pages failed to
  load. Record what is read in `specs/evidence.md`, and do not set a default from
  an unread source.
- **Answer (proposed, 2026-10-10):** a targeted review, not a survey. Opened today:
  the abcEconomics index (Firm and Household archetypes; the platform moves goods,
  as our ledger does; message passing) and ASSUME's custom-unit example (a unit plus
  a bidding strategy with one `calculate_bids()` reused across actor types; dispatch
  stays in the unit). Both support a single `Strategy` and physical handling outside
  it. Still to read before AGT: the Doran and Parberry bazaarBot paper (N-015), the
  abcEconomics Firm and Household pages, ASSUME's base strategy class, and the rest
  of Lengnick (only pages 3 to 7 are read). Before POP: Clark's own Malthusian
  source, not the VoxEU summary. Skip Prehistoric, the Space Project and Crusoe
  unless a specific question needs them. The evidence rows are added to
  `specs/evidence.md`.
- **Update (2026-10-10):** the Doran and Parberry journal paper is read in full (text only). Lengnick's firm price rule, Table 1 and the price-flexibility experiment are read from extracted text with the Greek symbols lost. ASSUME's custom-unit example re-read; its bidding-agents page (404) and the ZIP paper (403) failed to load. Factorio, Victoria 3 and abcEconomics production pages were opened through a summarising fetch only. Still to read before AGT: the abcEconomics Firm and Household pages and ASSUME's base strategy class.
- **Update (2026-10-10, AGT):** the targeted review for AGT is done. Read: Lengnick's household section (equations 1 to 4, 11, 12, Tables 1 and 2, the search part of 2.2; text extracted from the PDF, symbols partly lost); ASSUME's `base.py` (`BaseStrategy`, `BaseUnit`); abcEconomics `Household`, `Firm` and `Trade` pages through a summarising fetch. ASSUME's bidding-agents page still 404. Still to read before POP: Clark's own source.
- **Update (2026-10-10, POP):** Clark's own source read; see the closing line. Evidence rows added to `specs/evidence.md`; the VoxEU row is superseded for the structure of the model.
**Closed:** 2026-10-10. Applied for POP in `spec-population.md` 16: Clark's own source read (chapter 2 of *A Farewell to Alms*, author's chapter PDF, text extracted, figures not seen; draft chapters 4 and 5 of the October 2006 draft of the whole book, draft pp. 90-93 and 100-103). Chapter 3 and the published edition not read. It gives shapes (birth rate flat or rising with income, death rate falling, living standards falling with population) and no usable rates; no default is set from it. The evidence rows are in `specs/evidence.md`.

### N-076 A carriage market is not a goods market

- **Areas:** CAR, MER, TRN, CLR, LED
- **Raised:** 2026-10-10, `spec-carriage.md` 4.2.
- **Note:** `spec-clearing.md` admits asks only for stock and settles by moving goods; hired carriage moves none and uses escrow. The contract, transport and merchant specs must pick: the carriage unit as a produced good (a trip is a job), or a contract-aware admission and settlement that keeps the guarantees of `spec-clearing.md` 2.3. The carriage unit's price may fall below the resolution floor (an ask of 24 against a floor of 50), so choose a coarser unit.
- **Update (2026-10-10, REG):** a route is an edge, keyed by `EdgeID`, in ascending (From, To) (`spec-regions.md` 4.7).
- **Update (2026-10-10, TRN):** decided in `spec-transport.md` 8: option 1, the carriage unit is a produced good, the *haul unit* of a hired mode, made by an ordinary recipe of a carrier role, traded in the origin region's ordinary market and redeemed by the shipment at dispatch. `spec-clearing.md` needs no change and keeps the guarantees of 2.3. The unit price in the worked example is 1 195 against a floor of 50 (17.4), so the coarser unit is a data choice. Costs: capacity is produced ahead and unsold units spoil (a cobweb, TRN-A3); no escrow and no carrier custody for the market; a bilateral contract with escrow stays MER's.
**Closed:** 2026-10-10. Decided by TRN in `spec-transport.md` 8 (option 1, the haul unit; accepted at the wave 2 integration). `spec-clearing.md` is unchanged. `spec-carriage.md` is edited to match: `FR-CAR-001` and `FR-CAR-002` move to the future contract spec keeping their IDs, `FR-CAR-005` to `FR-CAR-007` are superseded by `FR-TRN-009`, `FR-TRN-010` and `FR-TRN-012`, and the unnamed margin `m_bp` is deleted; `spec-ledger.md` and `spec-goods.md` are edited as `spec-transport.md` 8.5 lists. A bilateral contract with escrow and the backhaul double count stay MER's (N-100).

### N-078 What the demand module needs from the money and income specs

- **Areas:** MON, EXT, EMP, LND, FIS
- **Raised:** 2026-10-10, `spec-demand.md` 7.3.
- **Note:** A non-producing class needs a recurring named transfer of money and a paired sink or return flow. In the model a stipend of 300 a tick to 50 households with no sink left household money at 47 700 against 2 700 at the start.
- **Update (2026-10-10, MON):** applied in `spec-money-supply.md`: the recurring named transfer is a grant (4.2) to a role with an empty `Recipes` list (7.3), counted as income by default (7.2); the paired sink or return flow is a levy, with the validation check `pair_unbalanced` for fixed pairs (9.2, 12.2) and a derived bound for a rate levy (9.3, `m* = E + g/rho`). The model of DEM (a stipend of 300 to 50 households with no sink) is the unbalanced case (9.1). `spec-demand.md` 10.2 `class_without_income` gets its edit in the integration.
**Closed:** 2026-10-10. Applied in `spec-money-supply.md` 4.2, 7.2, 7.3, 9.1 to 9.3 and 12.2: a grant and a levy as data, counted as income by default (the welfare edit of the wave 2 integration, `spec-welfare-and-wages.md` 2.1), with a derived bound for the rate levy.

### N-083 Population adds the `deaths` reason and decides renewal against births

- **Areas:** POP, LED
- **Raised:** 2026-10-10, `spec-spawner.md` 4.1 and 8.2.
- **Note:** POP adds the ledger reason `deaths` and decides `Renewal` against births (warning `renewal_with_population`).
**Closed:** 2026-10-10. Applied in `spec-population.md` 4.6 and 4.7: `deaths` is added to the ledger registry (money and goods, internal, agent to estate, a partial reduction) and `births` is extended to every household born through the module; `Renewal` is ignored (effective 0) while the population module is on, with the warning `renewal_with_population` only for a non-default positive `Renewal` and the informational `renewal_ignored` at the default (4.6.1); births are never netted.

### N-095 The external sector's hooks in money and clearing

- **Areas:** EXT, MON, CLR
- **Raised:** 2026-10-10, `spec-money-supply.md` 10.1 and 10.2.
- **Note:** MON reserves the target kind `external`, the top-up plus sweep as the pass-through counterparty, and the rule that a counterparty whose balance is not conserved is a Boundary account. EXT decides between a boundary-backed admission in CLR and the pass-through account (MON open question 1).
- **Closed:** 2026-10-10. Decided in `spec-external-sector.md` 4.1: a pass-through account (an account of the kind Outside, topped up before admission, swept after the markets) with six built-in reasons. `spec-clearing.md` changes by one wording only (the counterparty beside the estate in 3.1.2 and the `side_not_allowed` row of 3.2); a boundary-backed admission was rejected because the ledger refuses a boundary reservation and two specs would change. MON's target kind `external` is released (`spec-money-supply.md` 10.2, open question 1). The decision is recorded in ADR 0004.

### N-098 The Malthusian feedback needs a fixed factor from RES

- **Areas:** RES, POP
- **Raised:** 2026-10-10, `spec-population.md` 5.2.
- **Note:** The third Malthusian assumption (the ratio falls as households rise) can be provided only by an endowment that recipes draw from (`Draws`, `spec-production.md` 6.5), which fails validation while RES is off. Until RES exists POP's loop has no feedback; the only bounds are the pool's money (M / stake) and `HouseholdCap`. RES must provide a limited, regenerating endowment and say how it makes the ratio fall with households, so RES is written before the POP scenarios can run (recorded in the roadmap). A test double (a ratio function of households) is used until then.
- **Update (2026-10-10, RES):** applied in `spec-resources.md` 8: an endowment that binds is the fixed factor. With a flat abundance curve output per household is the endowment's maximum flow over the households above `N_bind` (an elasticity of -1, a kink); with a curve the stock and the population have closed forms; the stability condition is the characteristic equation `lambda^2 - F' lambda + Gamma = 0`, stable iff `F' < 0`, which gives: `constant` neutral, `refill` always stable and monotone above a rate (0.16 a year at the default population gain), `logistic` stable only above the maximum-yield point and in a narrow basin. Models in `doc/specs/models/res/` agree with the linearisation.
- **Closed:** 2026-10-10. Applied in `spec-resources.md` 8 (the mechanism, the closed forms, the stability condition by shape, the validator's warnings `resource_overshoot` and `resource_unstable_plan`) and in `spec-population.md` 5.2 and 9.2 (the third assumption is provided by RES; `no_fixed_factor` reads RES). The POP scenarios S37 to S40 may now use RES; nothing has run. The closed forms need the welfare ratio at a full draw (N-109).
- **Update (2026-10-13, TEC):** a technology gain scales `r_full` and moves the fixed point, not the loop gain (`spec-technology.md` 13.1; N-109).

### N-038 Seed prices, newcomers and migrants

- **Areas:** MIG
- **Raised:** 2026-10-10, `spec-beliefs.md` 8.
- **Note:** A single seed price per region and commodity starts both the beliefs and the market's initial price (`spec-beliefs.md` 8.1, 8.4); VIA may supply a table. CLR: the price before the first trade is that seed price (`spec-clearing.md` 5.4). SPN: replacements and newborns start from the window mean or the market price (8.3). MIG decides whether a migrant keeps or reseeds its beliefs (default reseed).
- **Update (2026-10-10, VIA):** the seed table exists: `Config.WarmStart.Prices` with groups for money, stock and jobs (`spec-viability.md` 8.1, 8.2); newcomers use the window (8.8). It is a start, not a standing anchor.
- **Update (2026-10-10):** the CLR part is applied (`spec-clearing.md` 5.4: the price before the first trade is the seed price). The BEL part is applied (`spec-beliefs.md` 8.1, 8.4).
- **Update (2026-10-10, SPN):** newcomers use `spec-beliefs.md` 8.3, not the warm-start table; role changers keep common beliefs (`spec-spawner.md` 4.2, 7.4). MIG decides whether a migrant keeps or reseeds its beliefs (default reseed).
- **Update (2026-10-10, REG):** `relocate` leaves a migrant's beliefs untouched (`spec-regions.md` 6.3), so MIG's default of reseeding on arrival is MIG's to execute. A warm-start seed replaces the seed price only; the reference band stays on the effective reference price (5.2.2).
- **Update (2026-10-10, POP):** a born household takes the newcomer's beliefs of `spec-beliefs.md` 8.3 (`spec-spawner.md` 7.4); MIG still decides for a migrant. A traveller of the transport module keeps its beliefs across a trip (`spec-transport.md` 9.3).
- **Update (2026-10-10, INF):** a migrant has no rumour state to keep or reseed: its rumours are those of its new region from its next view (`spec-information.md` 7.3).
- **Update (2026-10-10, MER):** a self-carried traveller reseeds its beliefs on arrival by the rule of `spec-beliefs.md` 8.3, taking no draw (`spec-merchants.md` 12.2), as MIG does for a migrant. One helper should serve both.
- **Closed:** 2026-10-10. Applied in `spec-migration.md` 8.4 and 8.5: a migrant is reseeded at arrival as any newcomer (`spec-beliefs.md` 8.3): for every commodity of its tradable set the centre is the clamp of the window mean of the new market (the market price if the window is empty) and the width is `BeliefWidthInit`; it has no rumour state to keep or reseed. A traveller of the transport module is reseeded by the same rule (`spec-merchants.md` 12.2, MER).

### N-100 What transport leaves to MER and INF

- **Areas:** MER, CAR
- **Raised:** 2026-10-10, `spec-transport.md` 8.4, 9.3, 20.2 and 20.5.
- **Note:** A hired shipment needs a consignee in the destination region (otherwise `consignee_elsewhere`); a diverted cargo goes to the destination estate; a traveller keeps its beliefs and remembers nothing else; the backhaul double count of `spec-carriage.md` 5.1 stands; a bilateral contract with escrow is MER's and does not use the book. The shipper's decision is MER's and rumours are INF's.
- **Update (2026-10-10, wave 2 integration):** a shipper that responds to a price gap with a transit delay of `k` ticks is stable in the linearised model only if its gain per tick is below `2 sin(pi / (2 (2k + 1)))` (1.0 at `k` = 1, 0.35 at `k` = 4), unless it counts `Inbound` or a cap bounds what it dispatches per tick, in which case the limit is about 2 whatever `k` is (`spec-transport.md` 15, 16; L84, L85). A model of a linear rule, not of the library: MER should count `Inbound` and test the margin with scenario S30.
- **Update (2026-10-10, INF):** rumours are defined (`spec-information.md` 5). A traveller keeps its beliefs and remembers nothing else; a trip carries no news back in version 1 and an arrival changes no record and no rumour (7.5, FR-INF-028); a per-traveller memory is MER's to ask for (`spec-information.md` 17, open question 1). A shipper's reading of a rumour and the condition on its gain are in INF-A1 and INF-A2 (13; L84, L85 and L93).
- **Update (2026-10-10, MIG):** a migrant is not a traveller and a traveller that is away is not a migrant; the migration guard respects `open_shipment` and `open_contract` (`spec-migration.md` 3.3.1, 7.2).
- **Closed:** 2026-10-10. Applied in `spec-merchants.md`: the consignee of a hired shipment is the payer of a delivery contract (8, 9); a diverted cargo counts as reached and the estate of the destination stands in for a removed payer (9.7); a traveller reseeds its beliefs on arrival (12.2); the backhaul double count stands and is measured (10); the bilateral contract with escrow is built on the shipment and does not use the book (9.1); the shipper's stability rules are in 7 and the model in `doc/specs/models/mer/`. The matching of delivery offers and takes is a first-fit round after the last plan of phase 3, not `Clearing` ([ADR 0005](adr/0005-delivery-contracts-are-matched-by-a-first-fit-round.md)).

### N-105 Instant or timed migration

- **Areas:** MIG, TRN
- **Raised:** 2026-10-10, `spec-regions.md` 6.5.3, `spec-transport.md` 5.5.
- **Note:** `relocate` is instantaneous (`spec-regions.md` 6). A migration that takes time has two ways. It can use TRN's traveller: the account stays in the origin region while away and changes region at arrival, and TRN's `open_shipment` refusal means MIG cannot relocate that agent meanwhile. Or it can use REG's reserved `NoRegion`, `depart` and `arrive`, which 6.5.3 binds to three rules: the departure region is recorded, a death leaves the money to the region departed from, and an account with no region is in no market. MIG picks one and states what a death, a ladder rung and the welfare window mean for a migrant on the road (the window problem of N-099 applies).
- **Closed:** 2026-10-10. Decided in `spec-migration.md` 8.1: a timed journey in the shape of TRN's traveller, the account staying in the region of departure until arrival and changing region at arrival through the ledger-level `Relocate` with the cause `migration`, after a dry run of every other refusal; MIG registers the `region_bound` refusal `on_journey`; REG's `NoRegion`, `depart` and `arrive` stay reserved and unused. A death on the journey leaves the money to the origin's pool, the ladder, relief and POP act as on any agent (8.3, 8.3.1); the window records `k` zero-income, non-working ticks (8.6) and the settling rule (5.2) keeps them from triggering a second move. TRN is unchanged except the wording of `spec-transport.md` 10.5.

### N-106 What a strategy may see of other regions

- **Areas:** MER
- **Raised:** 2026-10-10, `spec-transport.md` 9.1 and 15 (TRN-A1, L84).
- **Note:** The agent view carries `Routes` (for each other region and mode: whether there is a route, the travel time, and for a hired mode the haul units per carrying unit) and `Inbound` (goods on the way to this agent). No view carries a remote price or a remote stock, and none should until INF defines rumours with a delay and an error: otherwise a shipper sees the gap it is meant to estimate, and the transit-delay loop is hidden. MER's shipper reads rumoured prices and `Routes` and counts `Inbound`.
- **Update (2026-10-10, RES):** the agent view carries the endowments of its OWN region only (`spec-resources.md` 5.5.1); no view carries another region's stock.
- **Update (2026-10-10, INF):** the view field is `Rumours` of the `Trade` view of a role with `Informed` (`spec-information.md` 6.2). No view carries a true remote price: a rumour is at least 2 ticks old (the local report is 1) and blurred by an independent error, whatever the speed (5.2.2, 5.5, FR-INF-018). What MER's shipper must do with it: read `Rumours` with `Routes`; count `Inbound` (L84) and cap what it dispatches per tick (D39); use `SourceAge`, since a rumour of a carried-forward price is not news (INF-A3, L94); not difference rumours of successive ticks, because a status change steps the vintage (INF-A4, L95); give shippers a fixed spread of their own thresholds, drawn at creation: in the model of `spec-information.md` 11.2 a spread of 0.8 times the freight settles the gap (0.04 f) where an independent information error of the same size leaves 0.6 f and identical shippers 1.8 f (a model of an invented rule, not of MER's). Condition on the gain: INF-A1 and INF-A2 (L84, L85, L93).
- **Update (2026-10-10, EXT):** applied: no view carries an outside price (`spec-external-sector.md` 9.1, 9.2). In a gate region the price is learned from ordinary orders; elsewhere not at all in version 1; an outside node is not a rumour source (INF records only a market's published price). The shape of a source is EXT open question 10 (N-112).
- **Update (2026-10-10, MIG):** the migration pass reads rumours through INF's function for the agent under review, not through a view, and shows a strategy nothing (`spec-migration.md` 4.3.1); MER's shipper is unchanged.
- **Closed:** 2026-10-10. Applied in `spec-merchants.md` 4, 5, 7: the merchant reads `Rumours` with `Routes`, counts its `Inbound` as a consignee and its own `Outbound` as a shipper, caps the dispatch, spreads its thresholds by agent ID, acts only on a `SourceAge` within a bound and differences nothing. No view carries a remote price.

### N-009 Decisions the world and tick phases must make

- **Areas:** WLD
- **Raised:** 2026-10-09, from CLR, LED, CFG and SCL.
- **Note:** `spec-world.md` fixes, and several specs already assume:
  - all orders of a tick are placed before any market clears, and markets clear in
    region then commodity order (`spec-clearing.md` 7.2);
  - estate pseudo-agents are created with their regions, before any household
    (`spec-estates.md` 2.7.6);
  - ageing runs at the end of the tick (`spec-ledger.md` 5.3);
  - `Step` surfaces `ErrInvalidTrade` and, with `WithAudit`, `ErrAuditFailed`
    (`spec-clearing.md` open question 5; `spec-ledger.md` 12.4);
  - a checkpoint is taken between ticks, with no reservation open
    (`spec-ledger.md` 6.5);
  - options that change results (such as `WithTargetSpread`) are part of the
    effective configuration and its hash (`spec-config.md` 8.1);
  - decisions run at different rates (roadmap, WLD row).
- **Update (2026-10-10, AGT):** AGT assumes the phase order of `spec-agents.md` 4: start (moment, cost anchors, relief and starting money, directive), production (continue and complete, then `Work`, then starts), `Trade` and orders, clearing, outcomes and `Revise`, consumption, entry and exit, end (ageing). Creation draws in the order target ratio then residue (3.5.2). VIA: the warm start is applied before the agents' creation draws (`spec-viability.md` 8.6 item 5).
- **Update (2026-10-10):** a construction phase precedes tick 0 and ends at the first `Step` (`spec-ledger.md` 5.2.4, N-074).
- **Update (2026-10-10):** AGT phase 6 includes `Privation` and want consumption; phase 7 is detailed in `spec-spawner.md` 10.2 (relief in phase 1 from what the previous tick's entry step left, `spec-estates.md` 2.6; see N-082).
- **Update (2026-10-10, REG):** WLD takes from `spec-regions.md`: estate accounts and pseudo-agents are created with the regions, in ascending `RegionID`, before any household (3.6.1; the order against the ledger's boundary accounts is WLD's and the ledger's); every iteration over regions is ascending `RegionID` (3.3.2); a relocation needs no reservation open, so it runs after settlement and before the next admission, and the recommended phase is entry and exit (6.1.2); `SetEdgeStatuses` is a between-ticks call (7.3).
- **Update (2026-10-10, MON):** the money flows run in phase 1 after relief and starting money and before production (`spec-money-supply.md` 5.1).
- **Update (2026-10-10, TRN):** transport assumes: arrivals in phase 1, first, after the moment and before relief and starting money, in shipment ID order; dispatch in phase 3 for each agent in ID order after `Trade` returns, in this order: discards, shipments, intents; an agent that is away is skipped in phases 2 and 3 (`spec-transport.md` 5.3, 5.5); `Inbound` is fixed at the start of phase 3.
- **Update (2026-10-10, POP):** in phase 7 the POP deaths run before the entry budget is taken and births come last (`spec-population.md` 4.8).
- **Update (2026-10-10, wave 2 integration):** the phase 1 order is now: the moment and the cost anchors; transport arrivals; relief and starting money; money flows; the directive. Phase 7: ladder reductions and removals, POP deaths, switches and applies, births last. WLD fixes both.
- **Update (2026-10-10, wave 2 integration):** the order of account creation at construction is now fixed by the specs together and WLD adopts it: the built-in boundary accounts, then the boundary accounts of the declared money-flow reasons in flow-ID order (`spec-ledger.md` FR-LED-052), then each region's estate account in ascending `RegionID` (`spec-regions.md` 3.6.1, FR-REG-005), then the agents. The agent IDs of the estate pseudo-agents are `0..n-1`, so a pseudo-agent's ID equals its `RegionID`.
- **Update (2026-10-10, RES):** RES assumes: phase 2 takes a snapshot of every stock after continuations and completions and before `Work`, calls `Work` for every free agent, rations the draws per endowment in (RegionID, ResourceID) order, then starts in the rotation order; phase 8 ends with regeneration of every endowment in the same order, then the events `EndowmentChanged`, before the optional audit. Shock slots act between ticks, before phase 1 (`spec-resources.md` 4.1, 5.2).
- **Update (2026-10-10, INF):** WLD takes from INF: in phase 3 the view of an informed agent is built, with its error draws, immediately before its `Trade`, agents in ID order; `RumoursServed` is emitted after the last plan of phase 3, before phase 4; in phase 5 the price record is appended after the price windows are pushed and before the first `Revise` (`spec-information.md` 7.1).
- **Update (2026-10-10, EXT):** EXT asks WLD for three positions (`spec-external-sector.md` 5): phase 1, after the money flows and before the directive, the walk draws (one per walking line, ascending node then commodity); phase 3, after the agents' plans and the estates' asks, the counterparties' top-ups and orders in ascending agent ID; phase 4, after every market has cleared and every reservation is settled or released and before outcomes, the sweep of every counterparty account. Agent IDs: the estates first, then the counterparties (ascending node, gate, buy before sell), then the households. Account creation: after the estate accounts, the Outside accounts in agent-ID order; the boundary accounts of the six reasons after those of the declared flows (money reasons first, then goods reasons by commodity).
- **Update (2026-10-10, wave 3 integration):** the positions asked by RES, INF and EXT, in one list with the earlier ones (no two claim the same slot; where two act in one phase the order is the one listed). Phase 1: the moment and the cost anchors; transport arrivals; relief and starting money; money flows; the outside's walk (EXT); the directive. Phase 2: continuations and completions; the snapshot of every endowment's stock and abundance; `Work` for every free agent; the ration of each endowment that is drawn on; the starts in the rotation order (RES, W2: a two-pass start). Phase 3: for each agent in ID order, the rumours of an informed agent's view are built (INF, with the error draws) immediately before its `Trade`, then its plan is executed; the estates' asks; the counterparties' top-ups and orders in ascending agent ID (EXT); then `RumoursServed`, before phase 4. Phase 4: every market clears; then the sweep of every counterparty account (EXT). Phase 5: price windows are pushed, then the price record is appended (INF), then `Revise`. Phases 6 and 7 as before. Phase 8: ageing and spoilage, flow totals, regeneration of every endowment, the events `EndowmentChanged`, then the optional audit (RES). Construction order: the built-in boundary accounts, the boundary accounts of the declared money flows, those of EXT's six reasons, each region's estate account, the Outside accounts, then the agents. WLD fixes all of it.
- **Update (2026-10-10, SUB):** SUB assumes in phase 2: continuations and completions, the snapshot of the endowments (RES), then the flag step of SUB for every eligible agent in agent-ID order, then `Work` for every free agent, the ration, the starts; the event `OwnUseChanged` follows the starts, in (RegionID, role) order (`spec-subsistence.md` 5.4, 14.1). SUB draws nothing.
- **Update (2026-10-10, MIG):** MIG asks WLD for two positions (`spec-migration.md` 10.1): in phase 1, the arrivals of the migration journeys, in ascending agent ID, after the arrivals of the transport module and before relief and starting money; in phase 7, one migration pass for the whole world, after the entry step (the spawner and the population module) of EVERY region has run, with its error draws (6.2.1, 10.2.1). An agent on a journey is passed to neither `Work` nor `Trade` (as an away agent).
- **Update (2026-10-10, MER):** MER asks: phase 1, after transport's arrivals and before relief and starting money, the settlement of contracts in contract ID order, and the reseed of an arriving traveller's beliefs; phase 3, after the last agent's plan, the contract round, then the events `ContractRound` and `MerchantsPlanned`, then the estates' asks and the counterparties' orders, then `RumoursServed` (`spec-merchants.md` 12.5).
- **Update (2026-10-10, wave 4 integration, X12):** the positions asked by SUB, MIG and MER, in one list with the earlier ones. Where two specs act at the same point of a phase the order is the one listed; WLD fixes all of it. Phase 1: the moment and the cost anchors; transport arrivals (`ShipmentArrived`) and, at a traveller's arrival, the reseed of its beliefs (MER 12.2); the settlement of delivery contracts whose shipment arrived, in contract ID order, after the `ShipmentArrived` events (MER); the arrivals of migration journeys in ascending agent ID (MIG; MER's settlement and MIG's arrivals claim the same slot after transport's arrivals, and the order here is settlement first, since a party to an open contract cannot be on a journey, `spec-migration.md` 7.2); relief and starting money; money flows; the outside's walk (EXT); the directive. Phase 2: continuations and completions; the snapshot of every endowment's stock and abundance (RES); the flag step of the subsistence module for every eligible agent in agent-ID order (SUB, a pure function, no draw); `Work` for every free agent (an agent that is away or on a journey is skipped); the ration of each endowment that is drawn on; the starts in the rotation order (RES, a two-pass start); then the event `OwnUseChanged` in (RegionID, role) order. Phase 3: for each agent in ID order, the rumours of an informed agent's view are built (INF, with the error draws) immediately before its `Trade`, then its plan is executed; after the last agent's plan the contract round (MER: admission, first-fit matching, formation, with its draws), then `ContractRound` and `MerchantsPlanned` in ascending `RegionID`; the estates' asks; the counterparties' top-ups and orders in ascending agent ID (EXT); then `RumoursServed`, before phase 4. Phase 4: every market clears; then the sweep of every counterparty account (EXT). Phase 5: price windows are pushed, then the price record is appended (INF), then `Revise`. Phase 6 as before. Phase 7: ladder reductions and removals, POP deaths, switches and applies, births last (the entry step of every region), then ONE migration pass for the world with its error draws in ascending agent ID (MIG), taking its counts once at its start, then `MigrationStarted` and `MigrationReviewed`. Phase 8: ageing and spoilage, flow totals, regeneration of every endowment, the events `EndowmentChanged`, then the optional audit (RES). Construction order as in the wave 3 entry.
- **Closed:** 2026-10-11. Applied in `spec-world.md`: one tick of eight phases with every position (6), the wordings that disagreed ruled (6.12), the errors of `Step` (3.3.3), the options and the configuration hash (4), the checkpoint position (8.4) and the rates of decision (9). The positions asked by waves 1 to 4 are consolidated there, so the updates above are history; a new position takes a letter after the position it follows (FR-WLD-040).

### N-012 Events, metrics and log keys to define

- **Areas:** OBS
- **Raised:** 2026-10-09, collected from LED, SCL, CFG and EST.
- **Note:** `spec-observability.md` must define or accept:
  - `BoundaryFlow`, one aggregated event per region, reason and commodity per tick
    (`spec-ledger.md` 12.1);
  - `Weight` on `AgentSpawned` and `AgentBankrupted`, and agents and households
    reported separately in the snapshot (`spec-scale.md` 9);
  - the configuration hash on `WorldCreated` (`spec-config.md` 10.3);
  - `EstateOpened` and `ReliefPaid` (`spec-estates.md` 2.5, 2.6);
  - a household-count metric, `economysim.household.count`, as a change to the
    metric catalogue (`spec-scale.md` 9.4);
  - the log keys `path`, `rule` and `config_hash`, and the ledger reasons as the
    values of the `reason` key.
  Estate pseudo-agents must not count in `economysim.agent.count`.
- **Update (2026-10-10):** AGT adds `UpkeepShortfall` (one per region and role per tick, `spec-agents.md` 14.1); `AgentSpawned` gains `Role`, `StartMoney`, `External`; the removal event gains `Cause`; log keys `households` and `unfed` (14.4); a metric `economysim.household.unfed` is proposed (14.5). STB asks for counters on dampers that act silently (N-045).
- **Update (2026-10-10):** OBS also decides the name of the Snapshot field for the households held to work by the floor, and whether the construction bucket is a `BoundaryFlow` with `Tick` -1 (N-072).
- **Update (2026-10-10):** the events, keys and metrics of the written specs are now in the rules (`rules/observability-and-logging.md` 3.3.2, 4.3, 5.3). Still open for OBS: the Snapshot fields (N-065, N-081), `Tick` -1 for the construction bucket (N-072), the snapshot-fed gauges, and the remaining conflicts of the rules edit (N-088).
- **Update (2026-10-10, REG):** REG adds the events `AgentRelocated` and `EdgeStatusChanged`, the log keys `edges`, `components`, `from_region`, `to_region`, `distance` and `closed`, the metric `economysim.household.relocations` (attributes `economysim.region.from` and `.to`), snapshot fields for the edges with a non-default status and the component count, and a question for OBS: events carry the `RegionID` or the string ID (`spec-regions.md` 10, open question 4).
- **Update (2026-10-10, MON):** three events (`MoneyFlowLimited`, `MoneyFlowEnded`, `MoneyFlowScaled`), three log keys (`flow`, `due`, `applied`) and two proposed metrics (`economysim.money.flow`, `economysim.money.flow.unapplied`) in `spec-money-supply.md` 13; Snapshot fields for the money supply by sector, the money boundary balance per reason and each flow's state in 13.4 (names are OBS's).
- **Update (2026-10-10, POP):** POP adds the event `PopulationChanged` (one per region and role per tick, never when all counts are zero), the log keys `births` and `deaths`, and two proposed metrics (`economysim.household.births`, `economysim.household.deaths`) (`spec-population.md` 10).
- **Update (2026-10-10, TRN):** transport adds the events `ShipmentDispatched`, `ShipmentArrived` and `ShipmentRejected`, the log keys `shipment`, `mode`, `dest` and `modes`, and the metrics `economysim.transport.shipments`, `economysim.transport.in_transit` and `economysim.transport.cargo.spoiled` with a new metric attribute `economysim.mode` (`spec-transport.md` 13). Snapshot: the shipments in flight, the away agents and the non-default transit factors (13.5).
- **Update (2026-10-10, wave 2 integration):** the events, log keys and metrics of REG, MON, POP and TRN are now in `rules/observability-and-logging.md` 3.3.2, 4.3 and 5.3. Still open for OBS: the Snapshot fields (N-094), whether events carry the `RegionID` or the string ID, and the conflicts of N-088.
- **Update (2026-10-10, RES):** RES adds the events `EndowmentChanged` and `EndowmentAdjusted`, the log keys `resource`, `stock` and `short`, four proposed metrics (`economysim.endowment.drawn`, `.regenerated`, `.short`, `.stock`) and two new metric attributes `economysim.resource` and `economysim.region` (`spec-resources.md` 14). Whether events carry the string ID or the integer index is OBS's (as for REG).
- **Update (2026-10-10, INF):** INF adds the event `RumoursServed` (one per reader region per tick with an informed view), the log keys `rumours` and `lag`, and the metric `economysim.information.rumours` with a new attribute `economysim.outcome` (`offered`, `no_path`, `too_old`, `not_yet`) (`spec-information.md` 10).
- **Update (2026-10-10, EXT):** EXT adds the events `OutsideTraded`, `OutsideEnded` and `OutsideShocked`, the log keys `node`, `value`, `nodes`, `counterparties` and `drift`, and three metrics `economysim.external.units`, `.value` and `.price` with the attribute `economysim.node` (`spec-external-sector.md` 15). Snapshot fields in 15.4 (names are OBS's).
- **Update (2026-10-10, SUB):** SUB adds the event `OwnUseChanged` (one per region and role per tick, never when all counts are zero), the log key `own_use`, and two proposed metrics (`economysim.household.own_use`, a Gauge, and `economysim.own_use.units`, a Counter) with no new metric attribute (`spec-subsistence.md` 14).
- **Update (2026-10-10, MIG):** MIG adds the events `MigrationStarted` (per departure) and `MigrationReviewed` (one per region per tick in which any household was pushed or held back, any arrival was held or any journey ended by a removal), three proposed metrics (`economysim.migration.started`, `.limited`, `.on_journey`) and a new metric attribute `economysim.limit` with twelve constants (`spec-migration.md` 13). No new log key. Arrivals are REG's `AgentRelocated` with the cause `migration`.
- **Update (2026-10-10, MER):** MER adds the events `ContractFormed`, `ContractSettled`, `ContractRound` and `MerchantsPlanned`, the log keys `contract`, `offers`, `takes` and `merchants`, the metrics `economysim.contract.formed`, `.value`, `.escrow` and `economysim.merchant.decisions`, and a new metric attribute `economysim.decision` (`spec-merchants.md` 15). Snapshot: the contracts in flight and the money in escrow (15.4); names are OBS's.
- **Update (2026-10-11, WLD):** OBS accepted `Failed` on `RunEnded` and `TickEnded` and the events `YieldShocked`, `TransitFactorChanged` and `PopulationScaled` for the slots that had no event (`spec-world.md` 16).
- **Update (2026-10-11, INT):** INT asked OBS for those three events and adds the value `consumer` of the key `cause`; no new key and no metric (`spec-integration.md` 12).
- **Update (2026-10-11, CHK):** CHK adds one event, `WorldRestored` (its fields are `spec-observability.md` 4.4.5), the log message "world restored" and the keys `format` and `bytes`; no metric. `Checkpoint` emits no event.
- **Closed:** 2026-10-11. Applied in `spec-observability.md`: the events and fields of every written spec are one catalogue (4), whether an event carries the index or the string is answered by the typed integer ID and the names table (3.2), the snapshot fields are named (5), the metrics and the log keys are in 6 and 7, and estate pseudo-agents are in no count (5.1.2, 6.1.5). The rules carry the catalogue (`rules/observability-and-logging.md` 3.3.2, 4.3, 5.3). A new event or `Warn` adds a row to its tables (N-126).

### N-035 Working tick with multi-tick jobs

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-production.md` 5.2.1.
- **Note:** `spec-welfare-and-wages.md` 4.3 defines a working tick by the `RecipeExecuted` event. With jobs, `RecipeExecuted` comes at completion and every tick of a job is a working tick, so the root reads working ticks from the agent's job state. Change the wording of 4.3 to "had a job in progress in the tick (started, continued or completed)". OBS: `RecipeExecuted` gains `StartTick`, `Executions`, outputs, `Disposed`, `EffectiveYield`; `RecipeAbandoned` is new. This is a change to the event table in `rules/observability-and-logging.md` 4.3, which needs a new read-receipt tag when made.
- **Update (2026-10-10, WEL):** applied for WEL in `spec-welfare-and-wages.md` 4.3: a working tick is a tick with a job in progress (started, continued or completed); the root reads it from the agent's job state. OBS: `RecipeExecuted` fields and `RecipeAbandoned` remain.
- **Closed:** 2026-10-11. Applied: `RecipeExecuted` (start tick, executions, outputs, disposed quantities, effective yield) and `RecipeAbandoned` are in `spec-observability.md` 4.1.3, with `Region` added, and in `rules/observability-and-logging.md` 4.3; the working tick of `spec-welfare-and-wages.md` 4.3 was changed on 2026-10-10.

### N-040 Belief event, log keys and snapshot

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-beliefs.md` 12 and 18.9.
- **Note:** `BeliefsUpdated` (one aggregated event per market per tick, 12.1) and the log keys `region`, `bound` and `agents` (12.3) join the list of N-012. Whether `Snapshot` carries beliefs is open.
- **Closed:** 2026-10-11. Applied: `BeliefsUpdated` is in `spec-observability.md` 4.1.2; the snapshot carries beliefs as aggregates per market (5.3.5) and per agent through INT's query (5.4); the keys `region`, `bound` and `agents` are in the rules 3.3.2.

### N-065 Silent dampers need counters

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-agents.md` 7.1.3, `spec-welfare-and-wages.md` 5.3.
- **Note:** The work floor is counted by the world (a pure function of the welfare record, so a consumer's strategy cannot hide it): a Snapshot field and a bounded `Warn` "work floor binding" (`spec-agents.md` FR-AGT-072). The traineeship queue has its own `Warn`. OBS names the Snapshot fields. The production halt and a spawner entry limit are still uncounted (N-067).
- **Update (2026-10-10, POP):** the silent clamps of POP are counted in `PopulationChanged` (`spec-population.md` 10.1); the production halt is still uncounted (N-067).
- **Update (2026-10-10, INF):** the closed road, the horizon and the young run are counted by `RumoursServed` (`spec-information.md` 10.1).
- **Update (2026-10-10, SUB):** as N-045: the mode, the retry and the blocked households are counted by an aggregate event per region and role.
- **Update (2026-10-10, MIG):** as N-045: every pushed household ends in one counted outcome of `MigrationReviewed`.
- **Update (2026-10-10, MER):** as N-045: the merchant's dampers are counted by `MerchantsPlanned` and `ContractRound`.
- **Closed:** 2026-10-11. Applied: the work floor is `RoleRow.HeldByFloor` (`spec-observability.md` 5.3.4) with the bounded `Warn`; the production halt is `ProductionHalted` (4.4.6, N-067); the spawner's limits are `EntryLimited`; the other silent clamps are counted by their modules' events (N-045).
- **Update (2026-10-18, plan 1-9 Phase 1):** Reject as already settled: the spawner's limits are EntryLimited since OBS; this row builds the counts (D117, D119).

### N-072 The construction bucket as an event

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-ledger.md` 7.4.1 and open question 9.
- **Note:** Decide whether the construction bucket is emitted as a `BoundaryFlow` with `Tick` -1 (recommended in the ledger).
- **Update (2026-10-11, CHK):** a world restored before its first step is in construction with its bucket and emits the bucket's `BoundaryFlow` at its first step as the original would; one restored after it emits none (`spec-checkpoint.md` 6.7.2).
- **Closed:** 2026-10-11. Applied: `BoundaryFlow` with `Tick` -1 at the entry of the first step, before `TickStarted` (`spec-observability.md` 4.4.2, FR-OBS-015; `spec-ledger.md` 7.4.1; rules 4.3). An event at the end of `NewWorld` was the alternative and stays open question 2 of `spec-observability.md` 18.

### N-081 Snapshot figures the scenarios need

- **Areas:** OBS, WEL, AGT
- **Raised:** 2026-10-10, `spec-reference-chain.md` 9.
- **Note:** (1) the mean welfare ratio per role and region (`spec-welfare-and-wages.md` 11.2, FR-WEL-036: specified); (2) unsold stock per commodity; (3) a production-halt counter (N-067).
- **Closed:** 2026-10-11. Applied: (1) the mean welfare ratio per role and region is `RoleRow` with `RatioHouseholds` (`spec-observability.md` 5.3.4; read by `spec-indicators.md` 9, `WelfareDistribution`); (2) the unsold stock per commodity and region, agents and estate apart, is `MarketRow` (5.3.5, `StockCover` in IND); (3) the production halt is `ProductionHalted` (4.4.6).

### N-088 Conflicts left by the rules edits

- **Areas:** OBS
- **Raised:** 2026-10-10, `rules2` report.
- **Note:** (1) `household.unfed` is an up-down counter fed by an event that is absent when everyone is fed: the adapter keeps per-region and role state and resets at the end of the tick; (2) the construction bucket as a `BoundaryFlow` with `Tick` -1 is undecided (N-072); (3) the snapshot-fed gauges (production work in progress, the beliefs' mean centre) have no name and conflict with 5.3.1 ("derivable from events"); (4) `agent.bankruptcies` counts every cause of `AgentBankrupted`, including `consumer` and `population`, though the name says bankruptcies; (5) whether the estate's `ReliefPaid` and `EstateOpened` belong to a module that can be off.
- **Closed:** 2026-10-11. Applied, all five items, in `spec-observability.md` 6.3: (1) the exact rule for `household.unfed`, (2) the construction bucket as a `BoundaryFlow` with `Tick` -1, (3) the snapshot-fed gauges are named and `belief.centre` is fed by events, (4) `agent.removals` with the attribute `cause`, (5) `EstateOpened` is core and `ReliefPaid` belongs to the pool policy. The rules carry the result (5.3, 5.3.1).

### N-094 Snapshot fields of the wave 2 modules

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-regions.md` 10.5, `spec-money-supply.md` 13.4, `spec-population.md` 10.4, `spec-transport.md` 13.5.
- **Note:** OBS names the fields. REG: the edges with a non-default status and the component count. MON: the money supply by sector, the money boundary balance per reason and each flow's state. POP: per region and role the mean birth and death rates, the natural increase, the birth carry and the unfed households of the last tick; per module `r*` and the gain. TRN: the shipments in flight, the away agents and the non-default transit factors.
- **Update (2026-10-10, RES):** snapshot fields per endowment: stock, effective capacity, abundance, `ExhaustedRun`, and the shock scales when non-default; with the population module on OBS may add `N_bind` where VIA has computed it (`spec-resources.md` 14.4). Names are OBS's. EXT's fields are in `spec-external-sector.md` 15.4 and INF has none beyond the event.
- **Update (2026-10-10, SUB):** snapshot fields: per region and role the households in the mode and the households blocked in the last tick; per agent `OwnUse`, `OwnUseJobs`, `Cover` and `Secure` (`spec-subsistence.md` 14.4). Names are OBS's.
- **Closed:** 2026-10-11. Applied: every field of every module is named in `spec-observability.md` 5.3 and 5.8 and in INT's queries (`spec-integration.md` 6). The exceptions are events and not snapshot fields: the last tick's counts of SPN, MIG, DEM and SUB (5.2.2, 5.8). The income and wear fields that IND would want are a future ask (N-130).

### N-115 The boundary flows of the outside are gross

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-external-sector.md` 5.2.3 and 8.1.
- **Note:** `BoundaryFlow` shows the budget and the part returned, not the net. The net is derived (`import_c = outside_offer - outside_unsold`, `export_c = outside_purchases`, `export_value = outside_budget - outside_unspent`, `import_value = outside_receipts`). OBS decides whether to add a net event or metric of the balance of trade and names the Snapshot fields of `spec-external-sector.md` 15.4.
- **Closed:** 2026-10-11. Applied: no net event and no net metric. The balance of trade is derived from `OutsideTraded` (`spec-observability.md` 4.4.7; the cumulative figures are 5.3.9), and `TradeBalance` of `spec-indicators.md` 7.3.3 reports gross and net, in units and in money.

### N-013 What indicators must not get wrong

- **Areas:** IND
- **Raised:** 2026-10-09, from SCL, LED and GDS.
- **Note:** Inequality within a cohort is zero by construction, so an inequality
  measure sees only differences between agents (`spec-scale.md` 5.1). Population
  indicators count households (the sum of weights), never agents, and exclude
  pseudo-agents (FR-SCL-015). Balance sheets value goods at a price the caller
  supplies and are measurement only (`spec-ledger.md` 9.2). Wear units are counted
  by commodity and could back a depreciation measure (`spec-ledger.md` 5.4.2, open
  question 7). Commodity classes are the grouping for price indices and value
  added (`spec-goods.md` 6.1).
- **Update (2026-10-10, wave 2 integration):** a report by region does not balance from flows alone: a household that relocates takes its money and goods with it, and the ledger's relocation totals are the reconciling item (`spec-regions.md` 6.4). Rebuild the count of households by region from `AgentSpawned`, `AgentBankrupted`, `WeightChanged`, `RoleChanged` and `AgentRelocated`. The money supply by sector and the boundary balance per reason (one per money flow) come from the Snapshot fields of N-094. A price index of goods that travel should say whether it is the local price or the delivered price, which includes the freight (`spec-transport.md` 15, TRN-A4).
- **Closed:** 2026-10-11. Applied in `spec-indicators.md`: within-cohort inequality is zero by construction and is reported (`CohortShare`, 6.2); population indicators count households and exclude pseudo-agents (6.1); balance sheets value goods at the caller's price and are measurement only (4.3, 7.5.3); wear units back a depreciation measure only through an ask (7.6, N-130); classes are the grouping, and class value added is not defined from output and input (7.2.3); a report by region reconciles by the relocation totals (8.2) and households by region are rebuilt from events (6.3); the money supply by sector and the boundary balance per reason are `Sectors` (9); a price index says whether it is local or delivered (8.4).

### N-020 What each spec contributes to a checkpoint

- **Areas:** CHK
- **Raised:** 2026-10-09.
- **Note:** Every spec lists the state it adds (ADR 0002). Known so far: the ledger
  (`spec-ledger.md` 11) and the estate's listing prices (`spec-estates.md` 2.7.5).
  Not saved: reservations, flow totals and the derived expiry hints. The
  configuration hash includes result-affecting options (`spec-config.md` 8.1). Open:
  compaction of closed accounts (N-008), the extension-state interface, and the
  compatibility policy after 1.0 (ADR 0002, open points).
- **Update (2026-10-10, LED):** `Born` is signed in the saved state; the construction bucket and pending output are not in the ledger state (`spec-ledger.md` 11). The agent state AGT contributes is in `spec-agents.md` 20, the beliefs' in `spec-beliefs.md` 14.
- **Update (2026-10-10):** the spawner's state (`Privation`, `RoleSince`, the renewal carry, two diagnostic runs) is in `spec-spawner.md` 14; the demand module's carries are in `spec-demand.md` 12.
- **Update (2026-10-10, REG):** REG saves only the edges whose status is not the default (From, To, Closed, FactorBP), in `EdgeID` order (`spec-regions.md` 11). The graph tables, the resolved override arrays and the relocation totals are derived and not saved.
- **Update (2026-10-10, MON):** state in `spec-money-supply.md` 14 (per flow `Total`, `Scale`, `Ended`; carries per open account; pool carries per region).
- **Update (2026-10-10, POP):** POP contributes a death carry per agent, a birth carry per (region, role), the effective `BirthScale` and `DeathScale` per region when a shock set them, and two diagnostic runs (`spec-population.md` 11). Deaths close one account per removed agent and none for a partial reduction; with POP on the list of closed accounts grows by one slot per death in a weight-1 world (ledger open question 6).
- **Update (2026-10-10, TRN):** the transport module's state is the next shipment ID, the shipments in flight and the non-default transit factors (`spec-transport.md` 14); away flags and the free lists of Transit accounts are derived.
- **Update (2026-10-10, RES):** RES contributes per endowment `Stock`, the regeneration `Carry` and `ExhaustedRun`, and the non-default shock scales (`spec-resources.md` 15); effective values, the abundance snapshot, grants and keys are derived.
- **Update (2026-10-10, INF):** INF saves the price record: per source in ascending order the observations held, oldest first (`spec-information.md` 14). The lag table, sources and rumours are derived; the error needs no state.
- **Update (2026-10-10, EXT):** EXT saves per line the drift (with a walk) and the shock, per side the carry, the cumulative units and list value and the ended flag; the counterparty accounts are empty at a boundary and are not special state (`spec-external-sector.md` 16).
- **Update (2026-10-10, SUB):** SUB contributes per agent `OwnUse` and `OwnUseJobs`, and per (region, role) two diagnostic runs for its `Warn`s (`spec-subsistence.md` 15). The welfare ring gains two integers per tick (WEL).
- **Update (2026-10-10, MIG):** MIG contributes per agent on a journey its destination and arrival tick, and per region two diagnostic runs and a pending count of journeys ended by a removal (`spec-migration.md` 14). `RoleSince` is SPN's state. The welfare ring needs one more field per tick (relief).
- **Update (2026-10-10, MER):** MER saves the next contract ID and the contracts in flight in ID order; the merchant strategy has no state; the Escrow accounts are the ledger's (`spec-merchants.md` 16).
- **Update (2026-10-11, WLD):** the world contributes the PCG state, the next agent ID and the construction flag; the tick is written in the header (`spec-world.md` 12).
- **Update (2026-10-11, INT):** INT contributes, per external agent of an informed role, the tick of its last view and the rumours given to it (`spec-integration.md` 4.7.3, 13); no other state.
- **Closed:** 2026-10-11. Applied in `spec-checkpoint.md` 4: one inventory of the state every spec contributes (23 sections in dependency order), what is derived, what is deliberately not saved, and eleven findings (F1 to F11) where specs list state twice, incompletely or ambiguously, each with a ruling and an edit. The four things the note left open are settled: compaction of closed accounts (never in version 1, 9.4), the extension-state interface (`Stateful`, 7), the compatibility policy (9) and the hash (5.1, 6.2). A new spec adds its state to the inventory (4.2) and a per-agent field is saved once, in the agent record (N-128).

### N-022 The player and other external agents

- **Areas:** INT
- **Raised:** 2026-10-09.
- **Note:** An agent added through the integration interface has weight 1
  (`spec-scale.md` 4.7). It has an ordinary agent account and is bound by the same
  admission and reservation rules as any agent (`spec-clearing.md` 3).
- **Update (2026-10-10, AGT):** `spec-agents.md` 10. An external agent has weight 1, an `External` flag, and a strategy that is either a non-blocking consumer `Strategy` or the library's `Directed`, which reads a `Directive` left in the agent's mailbox between steps (applied to the next tick, discarded unless `Sticky`, saved in a checkpoint). No action happens inside a step; an "act now" operation is not provided, and INT may add one with its own rules. INT owns `AddAgent` and `Direct`. The spawner must not remove or replace an external agent unless the consumer asks.
- **Update (2026-10-10, SPN):** an external agent is refused by the world for removal or replacement unless it is `Mortal` (`spec-spawner.md` 9, FR-SPN-010; `spec-agents.md` 10.6).
- **Update (2026-10-10, MIG):** the default migration module never reviews, moves or draws for an external agent; INT moves one by the guarded `relocate` (`spec-migration.md` 3.3, 10.3).
- **Update (2026-10-11, WLD):** an external agent is created at the call with the next ID and the two creation draws (`spec-world.md` 8.3, D12).
- **Closed:** 2026-10-11. Applied in `spec-integration.md` 4: `AddAgent` (`NewAgent`: region, role, strategy, money, stock, `Mortal`), `Direct` and the mailbox, `RemoveAgent` (external agents only) and `Relocate`; what the player is told and the directive in the checkpoint are there too. What is not offered is N-131.

### N-079 External agents are mortal only when the consumer says so

- **Areas:** INT
- **Raised:** 2026-10-10, `spec-agents.md` 10.6, `spec-spawner.md` 9.
- **Note:** `AddAgent` carries a `Mortal` flag (default false); the world refuses decisions on an external agent unless it is mortal.
- **Update (2026-10-10, MIG):** as N-022: a mortal external agent is not moved by the default module.
- **Update (2026-10-11, WLD):** the removal call acts between ticks (`spec-world.md` 8.3).
- **Closed:** 2026-10-11. Applied in `spec-integration.md` 4.1, 4.4 and 4.6: `NewAgent.Mortal`, default false; the library acts on a mortal external agent only through a consumer's spawner or call; `RemoveAgent` ignores the flag.

### N-091 The world slots INT must expose

- **Areas:** INT
- **Raised:** 2026-10-10, `spec-regions.md` 7, `spec-money-supply.md` 10.4, `spec-population.md` 3.6 and 6.6, `spec-transport.md` 9.1 and 10.3.
- **Note:** Several mechanisms are world slots set between ticks, saved in a checkpoint, with a public face that belongs to INT (decided at the wave 2 integration: the mechanism belongs to the spec that owns it, the public call to INT): the yield shock (`spec-production.md` 6.8), the edge status, closed or slowed (`SetEdgeStatuses`, `spec-regions.md` 7), the transit factor (`spec-transport.md` 10.3), the flow scale (`SetFlowScale`, `spec-money-supply.md` 10.4) and the birth and death scales (`BirthScale`, `DeathScale`). INT also owns `relocate` for an external agent. For each, INT states the public name, its error values and its bound, between steps only. Each slot emits its own event and none is configuration.
- **Update (2026-10-10, RES):** RES adds two slots for INT: per endowment a capacity scale and a regeneration scale (basis points, 0 to `MaxShockBP`, default 10 000), and a one-off signed adjustment of the stock (`spec-resources.md` 6.3). Both are set between ticks, saved in a checkpoint, emit `EndowmentAdjusted`. The failed harvest stays the yield shock.
- **Update (2026-10-10, INF):** INF has no world slot. A blackout shock (slow or cut the news) would be one, set between ticks (`spec-information.md` 17 (open question 6)).
- **Update (2026-10-10, EXT):** one more slot: the outside price shock per (node, commodity), a factor in basis points, set between steps by `SetOutsideShock` (name provisional), saved, not hashed, with the event `OutsideShocked` (`spec-external-sector.md` 6.5). An embargo is the edge status of `spec-regions.md` 7. A capacity slot is left open (EXT open question 6). See N-116.
- **Update (2026-10-10, MIG):** MIG has no slot. A forced move (a flight, an expulsion) is INT's call of the guarded `relocate` with a cause of its own (`spec-regions.md` 6.6, `spec-migration.md` 10.3); for a household on a migration journey INT must cancel the journey first (MIG has an internal cancel, `spec-migration.md` 8.4.1; its public name is INT's), because MIG registers the refusal `on_journey`.
- **Update (2026-10-10, MER):** MER has no world slot.
- **Update (2026-10-11, WLD):** the slots are one table with the position each is read at (`spec-world.md` 8.2), and the calls that act on agents are in 8.3.
- **Update (2026-10-11, CHK):** every slot's state is saved by its owner and is listed in the inventory (`spec-checkpoint.md` 4.2); the sequence of calls is not saved.
- **Closed:** 2026-10-11. Applied in `spec-integration.md` 5.1: `SetYieldShocks`, `SetEdgeStatuses`, `SetTransitFactors`, `SetFlowScales`, `SetBirthScales`, `SetDeathScales`, `SetCapacityScales`, `SetRegenScales`, `AdjustEndowments` and `SetOutsideShocks`, each with its public name, errors, bound, event, position and saved state, between steps only; `Slots()` reads them back. The setters are plural slice forms and the cause is fixed by the setter (5.1.2): the singular names `SetFlowScale` and `SetOutsideShock` are retired, and the public `SetEdgeStatuses(changes)` has no `cause` argument (the internal form keeps it). A later spec that adds a slot adds a row.
- **Update (2026-10-13, STO, LND, CRD, CAP, RSK, ORG, TEC, PLC, FIS, SAV, POL, SET, INS):** the public faces added to `spec-integration.md` 5.1 and 5.4. Slots: `SetRateScales` (CRD; the entry type is `RegionScale`), `SetRentScales` (LND; it reuses `EndowmentScale`), `SetRouteRiskScales` and `SetStoreRiskScales` (RSK), `SetLegalLevels` and `SetRightSuspensions` (ORG), and the existing `SetFlowScales`, which accepts a fiscal flow ID (FIS, one lever). Calls that are not slots (5.4): `AddStore` and `RemoveStore` (STO, 4.9), `ForgiveClaims` (CRD, 4.10; claims of origin `loan` only), `TransferTitles` (LND), `TransferFacilities` and `AddFacilities` (CAP), `AdjustTechnology` (TEC; a one-off adjustment of module state, not listed by `Slots()`) and `DirectPolicy` (PLC). POL, SET, SAV (one query, `Saving`) and INS have no slot and no call; a transfer of a region between jurisdictions is reserved (N-161). PLC is a second writer of the existing slots at P0.3a (N-204).

### N-116 The outside's slots

- **Areas:** INT
- **Raised:** 2026-10-10, `spec-external-sector.md` 6.5 and 7.5.
- **Note:** INT names `SetOutsideShock` (node, commodity, basis points, between steps) and uses `SetEdgeStatuses` for an embargo. A capacity multiplier (a partial embargo) is not provided; decide when a consumer asks. Part of the slots of N-091.
- **Closed:** 2026-10-11. Applied in `spec-integration.md` 5.1 and 3.6: `SetOutsideShocks`; an embargo is `SetEdgeStatuses`; a capacity multiplier of a line is not provided (open question 8, N-131).

### N-036 Production's draws, phases and shocks in the world

- **Areas:** WLD, INT
- **Raised:** 2026-10-10, `spec-production.md` 5.5, 6.5, 6.6.2, 6.8.
- **Note:** WLD: production runs before orders; continuations and completions before starts; starts rotate by tick modulo the region's agent count (count taken at the start of the phase); at agent creation one draw in [0, 10 000) for production's residues, after the other creation draws in an order WLD fixes. INT: shocks are set and cleared between ticks through the yield-shock slot (a multiplier by region and recipe, default 10 000). RES: provide `Endowment(region, resource)` and `TakeEndowment(region, resource, n)`, own the resource definitions, depletion and regeneration, and make depletion visible by event since the ledger does not see it.
- **Update (2026-10-10, VIA):** applied for VIA in `spec-viability.md` 8.5, which staggers initial jobs with a fixed rule and answers `spec-production.md` open question 2; WLD must apply the warm start before the agents' creation draws (8.6 item 5).
- **Update (2026-10-10, REG):** the yield-shock slot and RES's `Endowment(region, resource)` key on `RegionID` (`spec-regions.md` 3.3). The edge status of `spec-regions.md` 7 is a second slot of the same shape (world state, set between ticks) for INT's closed-road shock; the other slots are listed in N-091.
- **Update (2026-10-10, RES):** applied for RES in `spec-resources.md` 5, 6 and 14: the seam is `Endowment` (public read) and an internal all-or-nothing take; RES owns the resource definitions, depletion and regeneration; depletion is visible by the events `EndowmentChanged` and `EndowmentAdjusted`, a bounded `Warn` and the counter `Short`. WLD: take a snapshot of every stock at the start of the production phase, after continuations and completions, ration the draws before the starts (5.2), and run regeneration as the last step of phase 8 before the optional audit (4.1). INT: two more slots for N-091, the scales and the adjustment (6.3). The ration replaces the rotation of `spec-production.md` 5.5.1 for contention on an endowment (decided at the wave 3 integration); the rotation still orders the starts.
- **Update (2026-10-11, INT):** the public call of the yield-shock slot is `SetYieldShocks` (`spec-integration.md` 5.1), read at P1.2, P2.5 and P2.7 and cleared by setting 10 000; it has no timer (5.2.2).
- **Closed:** 2026-10-11. Applied for WLD in `spec-world.md` 5.7, 6.1.4, 6.4 and 8.2 (production before orders, continuations and completions before starts, the rotation by `t mod n_r` with the count taken at the start of the phase, the creation draws in the order target then residue, the slot read positions) and for INT in `spec-integration.md` 5.1 (the public call). The RES and VIA parts were applied on 2026-10-10.

### N-051 What the Strategy contract needs from other specs

- **Areas:** INT, CHK, WLD
- **Raised:** 2026-10-10, `spec-agents.md` 6, 10.
- **Note:** INT: `AddAgent`, `Direct` and the mailbox. CHK: the strategy names are hashed, the code is not (`spec-agents.md` open question 4), and the directive is state. WLD: the phase order of `spec-agents.md` 4, `Work` between continuations and starts, `Trade` after production, `Revise` per outcome with the world's draw.
- **Update (2026-10-10, SPN):** SPN needs from WLD the phase-7 position, regions in order and the creation draws after all `Decide` calls (`spec-spawner.md` 10.2); from INT the `Mortal` flag of `AddAgent` (N-079); from CHK the spawner name hashed and its state pair.
- **Update (2026-10-11, WLD):** applied for WLD in `spec-world.md` 6: the phase order of `spec-agents.md` 4 stands (6 supersedes its table); `Work` runs between completions and starts and always in two passes (6.4.1); `Trade` after production; `Revise` per outcome with the world's draw (P5.4); the spawner after consumption with its draws per region (6.9.1).
- **Update (2026-10-11, CHK):** applied for CHK in `spec-checkpoint.md` 4.5 (F8) and 7: the strategy names are hashed and the code is not; the agent record carries a table of the strategy names agents use, which covers an external agent's name that the hash does not, and every name must be registered at restore; the directive is state (AGT 15); the identity of the code is the version of the `Stateful` pair (7.3.3), which answers AGT open question 4.
- **Update (2026-10-11, INT):** applied for INT: `AddAgent`, `Direct` and the mailbox are in `spec-integration.md` 4.
- **Closed:** 2026-10-11. All three areas applied (WLD, CHK, INT; the SPN part on 2026-10-10).

### N-056 What the ledger and world need for a warm start

- **Areas:** WLD
- **Raised:** 2026-10-10, `spec-viability.md` 8.6.
- **Note:** (1) `initial` for goods into any account at construction, including estate accounts; (2) lots with a negative `Born` and durable items with remaining durability below full at construction; (3) a construction bucket in the flow totals before tick 0, included in the audit; (4) a job created at construction with `Remaining < Duration` and pending output fixed from a virtual past start; (5) the warm start is applied before the agent creation draws (N-009, N-036).
- **Update (2026-10-10, LED):** items 1 to 3 are applied in `spec-ledger.md` 5.2.4, 7.4.1 and 9.1 (stated lot key at construction, signed `Born`, construction bucket in the audit). Items 4 and 5 stay with WLD (a job created at construction; the warm start before the creation draws); see N-074.
- **Closed:** 2026-10-11. Items 1 to 3 were applied in `spec-ledger.md` on 2026-10-10. Items 4 and 5 are applied in `spec-world.md` 5.6: a job created at construction with `Remaining < Duration` and its pending output fixed with a residue of 0 (5.6.1 step 4), and the warm start applied before the creation draws (5.6, 5.7; FR-WLD-033).

### N-074 The construction phase ends at the first Step

- **Areas:** WLD
- **Raised:** 2026-10-10, `spec-ledger.md` 5.2.4.
- **Note:** A stated lot key (negative `Born`, a worn durable) is allowed only during construction; WLD ends the phase at the first `Step`. With N-056 items 4 and 5: a job created at construction, and the warm start before the creation draws.
- **Closed:** 2026-10-11. Applied in `spec-world.md` 5.5: the construction phase begins at the first line of `NewWorld` and ends when the first `Step` begins, the consumer's calls in between are inside it, and only the warm start may state a lot key (FR-WLD-035). A world restored from a checkpoint taken before its first step is in construction (`spec-checkpoint.md` 6.7.2).

### N-117 The agent ID space and the account order with an outside

- **Areas:** AGT, EST, WLD, LED
- **Raised:** 2026-10-10, `spec-external-sector.md` 4.2.
- **Note:** With the module on the estates take the lowest agent IDs (`0..n-1`, equal to the `RegionID`), then the counterparties (ascending node, gate, buy before sell), then the households, so adding a node or a gate shifts every household's ID, tie-break keys and checkpoints, as adding a region does. WLD fixes the construction order (the estate accounts, the Outside accounts, then the agents) and the positions of N-009; AGT and EST restate FR-AGT-001 and FR-EST-005.
- **Closed:** 2026-10-11. Applied in `spec-world.md` 5.2 and 5.3: the estates are agents `0..n-1` with the ID equal to the `RegionID`, then the counterparties, then the households; accounts are created as built-in boundaries, the declared flows' reasons, the outside's six, the estates, the Outside accounts, then the agents (FR-WLD-030, FR-WLD-031). FR-AGT-001 and FR-EST-005 say the same.
- **Update (2026-10-13, POL, PLC, ORG):** the treasury pseudo-agents of POL take the IDs after the counterparties and before the households, and the Treasury accounts follow the Outside accounts and precede the agent accounts (`spec-polity.md` 5.2; `spec-world.md` 5.1 row 5 and 5.3). A policy actor has no agent ID and shifts none (N-207), and a body is not an agent (ORG).

### N-011 Transport, storage and carriage

- **Areas:** STO
- **Raised:** 2026-10-09, `spec-goods.md` section 3.
- **Note:** A vehicle's and a store's capacity is a pair, a mass capacity and a
  bulk capacity, and a load fits only if both are within it
  (`spec-goods.md` 3.2). Pricing uses the chargeable size (`BulkFactor`, 3.3, 3.4),
  which over-charges well-mixed cargo (3.5). Measure how much on the reference
  chain, and add a per-mode factor if the error matters. Goods keep aging in
  transit and in transit accounts; cargo lost is a `loss` sink
  (`spec-ledger.md` 7.2). Contract money is held in an Escrow account, not a
  reservation (`spec-ledger.md` 6.6). `spec-carriage.md` has no Requirements
  section yet, and its open items (`specs/README.md`, "Still open" 5 and 6) belong
  to the contract and merchant specs.
- **Update (2026-10-10, CAR):** `spec-carriage.md` now has Requirements (FR-CAR-001 to 013), restates its open items 5 and 6 with the reason they wait, and records a finding in 4.2 (see N-076).
- **Update (2026-10-10, TRN):** applied for TRN in `spec-transport.md`. A mode's capacity is a pair per carrying unit and a load fits only if both sums are within it (3.2). Transport does not use `BulkFactor` or the chargeable size: a hired load buys `n = max(ceil(mass / UnitMass), ceil(bulk / UnitBulk))` carrying units by its binding fill, which removes the over-charge of `spec-goods.md` 3.5 for transport and is the per-mode factor the note asked for (3.2.1, 8.3). Goods keep ageing in Transit accounts and a lot spoils whole from them (5.6); cargo lost is the reason `loss`, reserved for RSK (10.1). Still open for STO (the chargeable size for storage), CAR and MER (a bilateral contract and its escrow, `spec-transport.md` 8.4).
- **Update (2026-10-10, MER):** applied for MER in `spec-merchants.md` 9: contract money is held in an Escrow account by transfers, not in a reservation (9.6); the open items of CAR about the bilateral contract and the merchant's margin are settled there (9, 5.6, 10). Still open for STO (the chargeable size for storage).
- **Update (2026-10-13, RSK):** cargo lost is the sink `loss`, used with the risk module on (`spec-risk.md` 5, 7).
- **Update (2026-10-13, INS):** cargo lost is `loss`; a payout for it is an internal transfer and replaces no goods (`spec-insurance.md` 3.4, 3.5).
- **Closed:** 2026-10-13. Applied in `spec-storage.md` 8.4 and 19.1: storage uses neither the chargeable size nor `BulkFactor`; a store's capacity is an exact pair (mass room and bulk room) and its cost is per store, so the over-charge of `spec-goods.md` 3.5 does not arise for storage, as TRN settled it for transport. The other parts of the note were closed by TRN and MER (updates above), and cargo lost is the `loss` sink of the ledger (RSK, INS).

### N-046 Register entries owed by unwritten specs

- **Areas:** EMP, SAV, CRD, STO, RSK, FIS, PLC, LND
- **Raised:** 2026-10-10, `spec-stability.md` 4.7 and 9.
- **Note:** Each adds its entries by the checklist of `spec-stability.md` 9 and states its dampers by `STB-Dnn`. The expected entries are in section 4.7 (SPN: the failure ladder, entry limits, replacement by the most profitable role; POP: the Malthusian fixed point; and so on). A spec that adds a loop and no entry is not done (FR-STB-003). Undamped now: see `spec-stability.md` section 10.
- **Update (2026-10-10):** the DEM and SPN entries are merged into the register (L35, L36, L45, L46, L47, L50, L54 and L57 to L65).
- **Update (2026-10-10, REG):** REG has its entries (REG-A1 to A3, `spec-regions.md` 13), now L66 to L68; the TRN, MER, MIG and INF rows of `spec-stability.md` 4.8 depend on the distances of `spec-regions.md` 4.3.
- **Update (2026-10-10, MON):** MON-A1 to MON-A7, dampers MON-D1 to MON-D5 and scenarios S13 (defined) and MON-S1 to MON-S4 are in `spec-money-supply.md` 16; in the register MON-A1 is L69, MON-A2 merges into L30, MON-A3 to A7 are L70 to L74, the dampers are D41 to D45 and the scenarios S33 to S36.
- **Update (2026-10-10, POP):** the POP entries are in `spec-population.md` 14 (POP-A1 to A9, dampers POP-D1 to D7, scenarios POP-S1 to S4); in the register they are L75 to L83, D46 to D52 and S37 to S40. POP's row in 4.8 can be removed.
- **Update (2026-10-10, TRN):** TRN's entries are TRN-A1 to TRN-A8 in `spec-transport.md` 15 (price-gap shipping with a transit delay, the invisible pipeline, the haul capacity cobweb, freight and the delivered price, perishables that cannot cross, a trip as a gap in income, vehicles as capital, diversion to the pool); in the register TRN-A1, A2 and A5 are L84 to L86, A3 merges into L21, A4 into L17, A6 into L19, A7 into L26 and A8 into L63; the dampers are D38 to D40 and the scenarios S30 to S32.
- **Update (2026-10-10, wave 2 integration):** the entries of REG, MON, POP and TRN are merged into the register, so POP, MON and TRN leave the areas of this note. Still owed: MIG, INF, EMP, EXT, SAV, CRD, MER, STO, RSK, FIS, PLC, RES, LND, SUB, WLD.
- **Update (2026-10-10, wave 2 integration):** the register now has 86 loops, 52 dampers and 39 scenarios (S01 to S40, S15 unused). The next free numbers are **L87, D53 and S41**. A writer uses provisional IDs (`<CODE>-A1`, `<CODE>-D1`, `<CODE>-S1`) in its spec, and the supervisor assigns the final numbers when the entries are merged (N-103); do not reuse a number a spec quotes provisionally.
- **Update (2026-10-10, wave 3 integration):** the entries of RES, INF and EXT (`spec-resources.md` 19, `spec-information.md` 13, `spec-external-sector.md` 17) are merged into the register, so RES, INF and EXT leave the areas of this note. New loops: RES-A1 to A3 are L87 to L89 and RES-A5 to A7 are L90 to L92 (RES-A4 merges into L29); INF-A2 to A4 are L93 to L95 (INF-A1 merges into L84 and L85 as a case, INF-A5 into L23); EXT-A1 is L96 and EXT-A3 to A7 are L97 to L101 (EXT-A2 extends L21, EXT-A8 merges into L84 as a case). Dampers: RES-D1 to D6 are D53 to D58, INF-D1 to D3 are D59 to D61, EXT-D1 to D5 are D62 to D66. Scenarios: RES-S1 to S5 are S41 to S45, INF-S1 to S5 are S46 to S50, EXT-S1 to S5 are S51 to S55. The register now has 101 loops, 66 dampers and 54 scenarios (S01 to S55, S15 unused). The next free numbers are **L102, D67 and S56**. Still owed: MIG, EMP, SAV, CRD, MER, STO, RSK, FIS, PLC, LND, SUB, WLD.
- **Update (2026-10-10, SUB):** SUB-A1 to SUB-A7, dampers SUB-D1 to SUB-D5 and scenarios SUB-S1 to SUB-S5 are in `spec-subsistence.md` 18. SUB-A1 merges into L11 (and names L39 and L63 as beneficiaries), SUB-A7 into L18, L40 and L46; SUB-A2 to SUB-A6 are new loops and hazards (L102 to L106).
- **Update (2026-10-10, MIG):** MIG-A1 to MIG-A9, dampers MIG-D1 to MIG-D7 and scenarios MIG-S1 to MIG-S6 are in `spec-migration.md` 17. MIG-A4 merges into L40 (relief and migration), MIG-A5 into L68 and L76, MIG-A6 into L19; MIG-A1, A2, A3, A7, A8 and A9 are new (L107 to L109, L110 to L112).
- **Update (2026-10-10, MER):** MER's entries are MER-A1 to MER-A7 in `spec-merchants.md` 17 (the merchant's response to a price gap, the consignee's take loop, the merchant at its target income, derived demand for haul units with a choke price, the escrow, the drift of merchants between regions, a stranded preparation); MER-A1 merges into L84, L85 and L93, MER-A3 into L14 and L16, MER-A4 into L16 and L21, and MER-A2, A5, A6 and A7 are new (L113 to L116). Dampers MER-D1 to MER-D8, scenarios MER-S1 to MER-S6. The MER hook row of `spec-stability.md` 4.8 can be removed.
- **Update (2026-10-10, wave 4 integration):** the entries of SUB, MIG and MER are merged into the register, so SUB, MIG and MER leave the areas of this note. Dampers: SUB-D1 to D5 are D67 to D71, MIG-D1 to D7 are D72 to D78, MER-D1 to D8 are D79 to D86. Scenarios: SUB-S1 to S5 are S56 to S60, MIG-S1 to S6 are S61 to S66, MER-S1 to S6 are S67 to S72. The register now has 116 loops, 86 dampers and 71 scenarios (S01 to S72, S15 unused). The next free numbers are **L117, D87 and S73**. Still owed: EMP, SAV, CRD, STO, RSK, FIS, PLC, LND, WLD.
- **Update (2026-10-11, wave 5 integration):** the entries of WLD, OBS, CHK, IND, INT, STY, EXP and CLB are merged into the register (`spec-stability.md` 4.12, 5, 8), so WLD leaves the areas of this note. Loops: WLD-A1 is L117, WLD-A2 L118, WLD-A3 merges into L22 as the case of a world with no warm start, WLD-A4 is L119, WLD-A5 L120; OBS-A1 is L121 with IND-A1 and INT-A1 as its cases (F4), OBS-A2 L122; CHK-A1 to A3 are L123 to L125; IND-A2 and A3 are L126 and L127; INT-A2 and A3 are L128 and L129; STY-A1 to A4 are L130 to L133; EXP-A1 to A5 are L134 to L138; CLB-A1 is L139, CLB-A2 merges into L135, CLB-A3 is L140, CLB-A4 merges into L138, CLB-A5 and A6 are L141 and L142. Dampers: WLD-D1 is D87 (proposed, not adopted for its five places; the rule for a new spec), WLD-D2 is D88, D25 moves from proposed to specified, CHK-D1 and D2 are D89 and D90, INT-D1 is D91, EXP-D1 to D3 are D92 to D94, CLB-D1 is D95, CLB-D2 merges into D93, CLB-D3 to D5 are D96 to D98. Scenarios: WLD-S1 to S4 are S73 to S76, CHK-S1 and S2 are S77 and S78, INT-S1 to S5 are S79 to S83, EXP-S1 to S4 are S84 to S87, CLB-S1 to S3 are S88 to S90; the nine that test the library's own procedures against toy models (S77, S78, S84 to S90) are of the class "harness test" and are outside the band rule of FR-STY-098. STY adds the regime F9, deflation with distress, to section 7 (a proposal, no loop). The register now has 142 loops, 98 dampers and 90 scenarios (S01 to S90, S15 unused). The next free numbers are **L143, D99 and S91**. The CLB entries were merged from the live section 17 of `spec-calibration.md` and are provisional until that spec is accepted. Still owed: EMP, SAV, CRD, STO, RSK, FIS, PLC, LND.
- **Update (2026-10-13, STO):** STO-A1 to STO-A7, dampers STO-D1 to STO-D6 and scenarios STO-S1 to STO-S6 are in `spec-storage.md` 16. The register numbers: STO-A1 L143, A2 L144, A3 L145, A6 L146 (a constraint: the hard room), A7 L147; STO-A4 merges into L118 and STO-A5 into L71 and L69; STO-D1 to D3 are D99 to D101, D5 and D6 are D102 and D103, and STO-D4 (the hard room) is a constraint with no D number; STO-S1 to S6 are S91 to S96.
- **Update (2026-10-13, EMP):** EMP-A1 to EMP-A6, dampers EMP-D1 to EMP-D7 and scenarios EMP-S1 to EMP-S5 are in `spec-employment.md` 17. EMP-A3 is L148, A4 L149, A5 L150; EMP-A1 merges into L14, EMP-A2 into L21 with L22 and EMP-A6 into L36; EMP-D1 to D7 are D104 to D110; EMP-S1 to S5 are S97 to S101.
- **Update (2026-10-13, SET):** SET-A1 to SET-A5, dampers SET-D1 to SET-D4 and scenarios SET-S1 to SET-S5 are in `spec-settlements.md` 13. SET-A1 to A4 are L151 to L154 (A4 a constraint: the exception of `spec-settlements.md` 7); SET-A5 (the role ceiling, "a sixth first-come cap") merges into L118 and is the place O9; SET-D1, D2 and D4 are D111, D112 and D114, and SET-D3 (the class floor, a bound) has no D number; SET-S1 to S5 are S102 to S106.
- **Update (2026-10-13, LND):** LND-A1 to LND-A8, dampers LND-D1 to LND-D6 and scenarios LND-S1 to LND-S5 are in `spec-land.md` 16. LND-A1, A2, A4, A5, A6 and A8 are L155 to L160; LND-A3 merges into L14 and LND-A7 into L31 and L45; LND-D1 to D6 are D115 to D120; LND-S1 to S5 are S107 to S111.
- **Update (2026-10-13, POL):** POL-A1 (the seat shifts household IDs) and POL-A2 are L161 and L162 (`spec-polity.md` 13); POL has no damper; POL-S1 and S2 are S112 and S113 (harness tests).
- **Update (2026-10-13, CRD):** CRD-A1 to CRD-A10, dampers CRD-D1 to CRD-D8 and scenarios CRD-S1 to CRD-S5 are in `spec-credit.md` 18. CRD-A3, A5, A7, A8 and A9 are L163 to L167; CRD-A1 merges into L11 and L45, CRD-A2 into L14, CRD-A4 into L22, CRD-A6 into L31 and L45 and CRD-A10 into L39, L63 and L40; CRD-D1 to D8 are D121 to D128; CRD-S1 to S5 are S114 to S118.
- **Update (2026-10-13, CAP):** CAP-A1 to CAP-A8, dampers CAP-D1 to CAP-D5 and scenarios CAP-S1 to CAP-S7 are in `spec-infrastructure.md` 16. CAP-A2, A3, A5 and A6 are L168 to L171; CAP-A1 merges into L21 (full text) and L36, CAP-A4 into L75, CAP-A7 into L84 and L85 and CAP-A8 (`MaxFacilities`, refused in the rotation of the starts) into L118; CAP-D1 to D5 are D129 to D133; CAP-S1 to S7 are S119 to S125.
- **Update (2026-10-13, RSK):** RSK-A1 to RSK-A6, dampers RSK-D1 to RSK-D3 and scenarios RSK-S1 to RSK-S6 are in `spec-risk.md` 16. RSK-A2 is L172 and RSK-A6 (no endogenous risk, a constraint) is L173; RSK-A1 is a case of L84, L85 and L93 (the loop is more stable with a loss and the shipper poorer), RSK-A3 of L93 and L34, RSK-A4 of L19, L40 and L63 and RSK-A5 of L21 and L16; RSK-D1 to D3 are D134 to D136 (D2 and D3 constraints); RSK-S1 to S6 are S126 to S131.
- **Update (2026-10-13, FIS):** FIS-A1 to FIS-A8, dampers FIS-D1 to FIS-D6 and scenarios FIS-S1 to FIS-S5 are in `spec-fiscal.md` 19. FIS-A1, A2 and A4 to A8 are L174 to L180 (A3 absent: it merges into L14 as an amplifier); FIS-D1, D2 and D4 to D6 are D137, D138 and D140 to D142, and FIS-D3 (the rule of WLD D87) is folded into D87; FIS-S1 to S5 are S132 to S136.
- **Update (2026-10-13, ORG):** ORG-A1 to ORG-A8, dampers ORG-D1 to ORG-D5 and scenarios ORG-S1 to ORG-S5 are in `spec-institutions.md` 14. ORG-A1 to A4, A6 and A7 are L181 to L186; ORG-A5 (seniority is a first-come rule among the equal) merges into L118 and is the place O10, and ORG-A8 into L121; ORG-D1 to D5 are D143 to D147; ORG-S1 to S5 are S137 to S141.
- **Update (2026-10-13, SAV):** SAV-A1 to SAV-A7, dampers SAV-D1 to SAV-D6 and scenarios SAV-S1 to SAV-S6 are in `spec-saving-and-investment.md` 14. SAV-A2, A4, A5 and A6 are L187 to L190; SAV-A1 merges into L45, SAV-A3 (the decision side of CAP-A1) into L21 and L36 and SAV-A7 into L31 and L45; SAV-D1 to D6 are D148 to D153; SAV-S1 to S6 are S142 to S147.
- **Update (2026-10-13, TEC):** TEC-A1 to TEC-A7, dampers TEC-D1 to TEC-D4 and scenarios TEC-S1 to TEC-S7 are in `spec-technology.md` 14. TEC-A1, A2, A5 and A6 are L191 to L194; TEC-A3 merges into L88, TEC-A4 into L14 and TEC-A7 into L21 and L22; TEC-D1 to D4 are D154 to D157; TEC-S1 to S7 are S148 to S154.
- **Update (2026-10-13, PLC):** PLC-A1 (a reactive rule is a controller), A2 (windup on a stale reading), A3 (two actors, one economy), A4 (a bound hides a runaway) and A5 (a schedule is open loop) are L195 to L199 (`spec-policy.md` 15); PLC-A6 (a directive waits for its cadence) merges into L121 and PLC-A7 (a consumer's policy that draws) into L120; PLC-D1 to D6 are D158 to D163; PLC-S1 to S5 are S155 to S159 (S157 is a report).
- **Update (2026-10-13, INS):** INS-A1 to INS-A7, dampers INS-D1 to INS-D7 and scenarios INS-S1 to INS-S6 are in `spec-insurance.md` 17. INS-A3, A4 and A7 are L200 to L202 (A7 a link of RSK-A4's chain); INS-A1 merges into L84, L85 and L93 (a case, with RSK-A1), INS-A2 into L93, L34 and L163, INS-A5 into L19 (and L14) and INS-A6 into L11 and L45; INS-D1 is D164, D2 is D121 (CRD-D1: not a new row), D3 to D7 are D165 to D169 (D6 a constraint; D7 the case of D122 for a payout claim); INS-S1 to S6 are S160 to S165. Moral hazard and adverse selection in their full forms need a probability the insured moves and private information, which RSK excludes: the quasi forms are INS-A1, A2 and A4 (N-211).
- **Closed:** 2026-10-13. Applied: the entries of all fourteen later-stage specs (STO, EMP, SET, LND, POL, CRD, CAP, RSK, FIS, ORG, SAV, TEC, PLC, INS) are merged into the register, `spec-stability.md` 4.13 (STO, EMP, SET, LND: L143 to L160), 4.14 (POL, CRD, CAP, RSK, FIS: L161 to L180) and 4.15 (ORG, SAV, TEC, PLC, INS: L181 to L202), each with its "merged into" table; 36 of the 96 provisional loop entries merge into existing loops. The register has 202 loops, 169 damper numbers (D01 to D169; D113 and D139 are not issued, as S15) and 165 scenarios (S01 to S165, S15 unused). The harness tests among the new scenarios (outside the band rule of FR-STY-098) are S101, S105, S106, S112, S113, S119, S125, S132, S141, S142, S147, S148, S153, S155, S158 and S159. The next free numbers are **L203, D170 and S166**. The hook rows of `spec-stability.md` 4.8 that named these specs are gone. A new spec adds its entries by the checklist of `spec-stability.md` 9 (FR-STB-002, FR-STB-003).

### N-104 Region-keyed tables use the one override mechanism

- **Areas:** LND
- **Raised:** 2026-10-10, `spec-regions.md` 5.1 and 5.3.
- **Note:** A new table keyed by region (a yield, a price, a scale, a cap, a rate) uses the mechanism of `spec-regions.md` 5.1: one row per key in the owner's table, combined with the default by replace, add or scale, in canonical order by `RegionID`, with the errors of 5.3, and a row in the list of 5.1.3. `RegionPrices` and POP's `Regions[]` are the two examples. Do not invent a per-region field with its own rules. The same holds for SET, CAP and later specs.
- **Update (2026-10-10, RES):** applied for RES in `spec-resources.md` 3.3.1: `Endowments` is a region-keyed table of the mechanism of `spec-regions.md` 5.1, one row per (region, resource), `Capacity` and `Initial` by replace, `RegenScaleBP` by scale, absence is the default (a resource's default capacity is 0). The row in the list of `spec-regions.md` 5.1.3 is added at the wave 3 integration. SUB, MIG and LND stay.
- **Update (2026-10-10, INF):** INF adds no region-keyed table. The speed of news is one world value; a slower region is expressed by longer distances; a per-region speed would be a row of `spec-regions.md` 5.1.3 in scale mode keyed by the reader's region (`spec-information.md` 17 (open question 9)).
- **Update (2026-10-10, EXT):** EXT's tables follow the one mechanism: the node rows are keyed by region, the line rows by (region, commodity), one row per key, in canonical order, with the errors of `spec-regions.md` 5.3; the gate scale (`OutsideGate.ScaleBP`) is a **scale** row (`spec-external-sector.md` 3.2.4). Rows are added to the list of `spec-regions.md` 5.1.3 at the wave 3 integration.
- **Update (2026-10-10, SUB):** applied for SUB in `spec-subsistence.md` 3.2.1 and 3.5: the module has no region-keyed table and no per-region field. Its role-keyed rows follow the replace mode (one row per role, absence is the default, an empty row means none); regional differences are rows of the yield table and the endowment rows, which already follow the mechanism.
- **Update (2026-10-10, MIG):** applied for MIG in `spec-migration.md` 11.2: `Regions` rows with `OutShare` and `InShare`, one row per region, by replace, in region order, with the errors of `spec-regions.md` 5.3; a row is added to the list of `spec-regions.md` 5.1.3.
- **Update (2026-10-10, MER):** MER adds no region-keyed table. A lane is not a row. LND stays.
- **Update (2026-10-13, SET):** `Modules.Settlements.Regions` is one row per region, `ThresholdBP` by scale and `Floor` by replace, with the errors of `spec-regions.md` 5.3 (`spec-settlements.md` 3.5); two rows join the list of 5.1.3.
- **Update (2026-10-13, LND):** `Terms` and `Holdings` are region-keyed tables of the one mechanism, replace modes, one row per key, the errors of `spec-regions.md` 5.3 (`spec-land.md` 3.3.1); two rows for 5.1.3.
- **Update (2026-10-13, POL):** `Modules.Polity.Members` is one row per region, replace mode, with the errors of 5.3 (`spec-polity.md` 3.4.1); one row joins the list of 5.1.3.
- **Update (2026-10-13, CAP):** the initial rows are a table keyed by (region, class, toward, role), one row per key, canonical order, the errors of 5.3 (`spec-infrastructure.md` 3.4.2); a row joins the list of 5.1.3.
- **Update (2026-10-13, RSK):** `Modules.Risk.Routes` and `.Stores` are region-keyed tables of the one mechanism, replace mode (`spec-risk.md` 3.2.3); rows join the list of 5.1.3. They name a mode, an edge and a store class of two other owners, so validation runs after those tables exist and reports `ErrModuleDependency` when an owner is off (N-182).
- **Update (2026-10-13, ORG):** rights and controls are rows keyed by region or jurisdiction; a row is added to 5.1.3.
- **Update (2026-10-13, EMP, FIS, TEC, SAV, CRD, INS, PLC):** none of them adds a region-keyed table (a regional difference is a yield-table or endowment row; FIS names one authority and optionally one region; PLC's lever names a key).
- **Closed:** 2026-10-13. Applied: the rule is the mechanism of `spec-regions.md` 5.1 (one row per key in the owner's table, combined with the default by replace, add or scale, in canonical order by `RegionID`, with the errors of 5.3, and a row in the list of 5.1.3), and every later table follows it: RES, EXT, MIG, SET, LND, POL, CAP, RSK and ORG say so in their own sections (updates above). It lives in `spec-regions.md` 5.1 and not in a note; a spec that adds a table keyed by region repeats it in its own section and adds its row to 5.1.3.

### N-025 Do policy actors reuse `Strategy`?

- **Areas:** PLC
- **Raised:** 2026-10-10, `spec-roadmap.md` Open 2.
- **Note:** Open: whether a policy actor (a ruler choosing tax rates, minting,
  granary release, price controls) reuses the `Strategy` extension point or needs
  its own interface. A new interface means a change to the table in
  architecture.md 4.3, which also requires a default implementation and a
  compile-time assertion (4.5), and receives its randomness from the world (7.7).
  Decide when writing `spec-policy.md`.
- **Answer (proposed, 2026-10-10):** a separate interface, written when PLC is.
  ASSUME reuses one `calculate_bids()` interface for generators, demand and custom
  units, which supports one `Strategy` for every actor that *bids*. A policy actor
  does not bid: it reads measures and returns settings (rates, release amounts), so
  the signature and the returned type differ. A ruler that also trades (buying grain
  for a granary) uses `Strategy` for its orders and the policy interface for its
  settings. Do not create the interface before PLC (architecture.md 4.2).
- **Update (2026-10-10):** ADR 0003 decision 3 records the answer (its own interface, written with `spec-policy.md`). Close this note when PLC is written.
- **Closed:** 2026-10-13. No: a policy actor has its own interface, `Policy`, with one method, `Decide(PolicyView, *rand.Rand) Settings` (`spec-policy.md` 4). The table of `architecture.md` 4.3 gains it as a fifth row, and 4.3's extension-state paragraph, 6.2 (`WithPolicy`) and 7.7 name it, with a fresh tag. A ruler that also trades is two objects: the treasury's standing orders (`spec-fiscal.md` 9) and an external agent with `Directed` (`spec-policy.md` 9.2); this narrows the second sentence of ADR 0003 decision 3, which a dated note there records, and ADR 0001 open point 1 is answered. FIS adds no interface.

### N-249 The privation step is planned twice

- **Areas:** AGT, SPN, CODE
- **Raised:** 2026-10-17, the consistency pass over plans 1-7 and 1-9.
- **Note:** `household.NextPrivation` (plan 1-7 Phase 2, called by `consume` at P6.1; AGT 5.4.1, FR-AGT-075) and `household.NextPrivation` (plan 1-9 Phase 2; SPN 5.1, FR-SPN-012) are the same step, `prev + 1` if unfed else `max(0, prev - 1)`, in two packages, and N-266 has the world set `Privation` with the second while `consume` sets it with the first. One must call the other or one must go before either is merged; until then the first row to land adds a root test that the two agree. Left open for the coordinator.
- **Closed:** 2026-10-17. Applied by coordinator ruling (consistency pass): one implementation, `household.NextPrivation` in `internal/household` (plan 1-7; AGT 5.4.1, AGT 22). Plan 1-9 drops `occupation.PrivationStep`; FR-SPN-012 and SPN 15.6 are tested through plan 1-7's function; SPN 20 loses "the privation update" from the `internal/occupation` row (plan 1-9 Phase 1, deliverable 10a).

### N-219 Edits owed to EST and MON by the estates answer

- **Areas:** EST, MON
- **Raised:** 2026-10-12, `spec-config.md` 6.2.4 (Phase 1 of the Stage 0 plan).
- **Note:** `spec-estates.md`'s layer line ("module for the regional pool, starting money and relief") says "module for relief": the estate account, the pseudo-agent, the listings and the use of the pool to fund an entrant exist in every world, and `[modules.estates]` switches on the pool policy only (relief, and the pool as a money-flow target). `spec-money-supply.md` open question 8 is answered by `spec-config.md` 6.2.4: the account exists without the module, the pool target needs it, `ErrModuleDependency` stays.
- **Closed:** 2026-10-17. Applied in the spec edits of the owed-edits job: the layer line of `spec-estates.md` (line 9) says core for the estate account, pseudo-agent, listings and the pool's funding of an entrant, module for relief and the pool as a money-flow target; `spec-money-supply.md` open question 8 is marked answered (N-219).

### N-255 What `GoodsListed` counts

- **Areas:** EST, OBS, SPN, CAP
- **Raised:** 2026-10-17, `plans/plan-phase-1-8.md` D101.
- **Note:** `AgentBankrupted.GoodsListed`, `EstateOpened.GoodsListed` and `RoleChanged.GoodsListed` say "the number of goods passed to the estate"; `FacilityChanged.GoodsListed` says "the count of lots moved" (`spec-infrastructure.md` 700, FR-CAP-047). The plan counts lots (one per distinct key of the agent), made by the one function every caller shares, so the four agree; the coordinator's ruling was asked on 2026-10-17. A ruling of units changes the return value of `receive` and `acceptGoods` and the doc comments. **Answer (2026-10-17, coordinator): lots.** Units travel separately in `WeightChanged.Units` (FR-OBS-019). Plan 1-8 Phase 1 states it where the field is defined and Phase 3 rewords the `event.go` comments to "number of lots"; the note closes when Phase 1 is merged.
- **Closed:** 2026-10-17. Applied: lots, in `spec-estates.md` 2.1 (line 52) and 2.5 (FR-EST-021), `spec-agents.md` 3.6 and 14.1, `spec-spawner.md` 13.1 (table and the sentence below it), `spec-world.md` 6.9.0, `spec-observability.md` 4.1.4 (line 366), `rules/observability-and-logging.md` (line 361) and `spec-infrastructure.md` (FR-CAP-047 already said lots). `spec-storage.md` lines 678 and 1749 say only "the number of goods listed" / "as that field counts them"; they do not contradict lots and were left unedited (a one-word fix, 'lots', may be made when STO is next edited). Plan 1-8 Phase 3 still rewords the `event.go` comments to "number of lots".
- **Update (2026-10-18, plan 1-9 Phase 1):** Apply for this row. GoodsListed counts lots; the executor emits it from the lot count; this row computes none.

### N-257 FR-EST-010 asks for a snapshot marker that OBS 5.1.2 does not have

- **Areas:** EST, OBS
- **Raised:** 2026-10-17, `plans/plan-phase-1-8.md` D107.
- **Note:** FR-EST-010 says the snapshot MUST mark the estate as a pseudo-agent. `spec-observability.md` 5.1.2 puts pseudo-agents only where a row is about them, REG 3.6.1 fixes the estate's agent ID as its `RegionID` and FR-CHK-048 gives it no agent row; the pool is `Regions[].MoneyPool`. Phase 1 of the plan rewords FR-EST-010 to match OBS. **Answer (2026-10-17, coordinator):** reword FR-EST-010 to match OBS 5.1.2 and FR-OBS-007; the estate has no per-agent snapshot row, so there is nothing to mark and no marker is added. The note closes when Phase 1 of plan 1-8 is merged.
- **Closed:** 2026-10-17. Applied: FR-EST-010 now says the snapshot has no row for the estate and no marker (its identity is the `RegionID` of the region row, its money `MoneyPool`), and `spec-observability.md` 5.1.2/5.3.9 and FR-OBS-007 agree.

### N-229 Registration names: one rule for the four options, and no second reserved clearing name

- **Areas:** CLR, WLD
- **Raised:** 2026-10-15, Phase 1 of `plans/plan-phase-1-3.md` (the coordinator's answer of 2026-10-13, Q3).
- **Note:** `spec-clearing.md` 8 listed a second reserved clearing name, `default`, that `spec-world.md` 4.1 and FR-WLD-002 never reserved: the library reserves `average` for a clearing and nothing else, and CLR 8 is corrected. The registration rule is written once, in `spec-world.md` 4.1.1, for `WithStrategy`, `WithSpawner`, `WithClearing` and `WithPolicy`: a name satisfies `validID` (`ErrInvalidID`, empty included), a duplicate or built-in name is `ErrDuplicateID`, a nil extension is `ErrNilExtension`; an empty or unregistered name in the configuration is the field's `ErrUnknown*`, a `*ValidationError` with its path (`spec-config.md` 7.1.4). FR-WLD-002 and 4.3 carry `ErrInvalidID`. To read: the strategy (`spec-agents.md` 6.6), spawner (`spec-spawner.md` 3.1.3) and policy (`spec-policy.md` 4.5) specs may restate the duplicate and nil rules or an empty name and should say "as `spec-world.md` 4.1.1"; the strategy and spawner rows reuse the one generic `registry[T]` of the clearing row (D36) unchanged. Check that no other spec reserves a name that the table of 4.1 does not list.
- **Closed:** 2026-10-17. Applied: `spec-clearing.md` 8 no longer reserves `default`; the registration rule is once in `spec-world.md` 4.1.1 (`ErrInvalidID`, `ErrDuplicateID`, `ErrNilExtension`, the field's `ErrUnknown*`), and `spec-spawner.md` 3.1.3 (line 93) says "as `spec-world.md` 4.1.1" and `ErrUnknownSpawner` is returned by `NewWorld`; `spec-config.md` 7.1.4 and FR-CFG-038/063 carry the four name checks. No other spec reserves a name beyond `average`.

### N-248 The spawner's per-agent group is `RegionAgent`, not `AgentView`

- **Areas:** SPN, AGT, CODE
- **Raised:** 2026-10-17, the consistency pass over `plans/plan-phase-1-7.md` to `plan-phase-1-12.md`.
- **Note:** Plan 1-7 exports `AgentView`, the strategy's view of one agent (AGT 6.4). The first text of plan 1-9 called the per-agent group of the spawner's `RegionView` `AgentView` too, and two types cannot share a name in the root package. `spec-spawner.md` 3.2 names no Go type for the group; plan 1-9 now says `RegionAgent`, and its Phase 1 writes the name into SPN 3.2 so that the spec and the code agree. A later view or row type is checked against the exported names of the root before it is declared.
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Closed:** 2026-10-17. Applied: `spec-spawner.md` 3.2 names the per-agent group `RegionAgent` and lists it among the exported names, so spec and plan 1-9 agree. The standing advice (check a new view or row type against the root's exported names) belongs to the plans.

### N-275 The supply gap and the lean reserve

- **Areas:** VIA, REF
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (D122, D123).
- **Note:** VIA 4.5 says the gap uses "the same start windows as `J`", which is one household's greedy sequence, and the plan keeps that reading (round 1 audit): it gives REF's gaps (304 for `one`: starts 59 and 119, completions 118 and 178; 123; 184), where counting every open start tick would give 273 for the 92-day spring. The lean reserve of 8.4.1 has no formula in the spec; the plan states it (largest cumulative shortfall over a year from tick 0 of the whole consumption, upkeep plus the inputs of the active recipes: 644 plus the 20 `staple` of `save_seed`'s input is the 664 of `warm-one.toml`) and REF's "use per tick times the ticks to the first harvest" is its special case. Both are written into VIA in Phase 1 of the plan. An activity open on every tick completes every tick (8.5.2 staggers the group), so it has no gap and no reserve (ruled by the coordinator, reversible; option (a) of O10); a start counts at every tick of the open window (the prototype's `t < YEAR` reading, 7 starts of a 60-day job; ruled, reversible), the completion ticks wrap cyclically for the gap, but a start whose completion tick is at or after the year end adds nothing to the reserve and is not wrapped, while `N_open` still counts its tick (`two` south: 789 in `warm-two.toml`, 510 if wrapped).
- **Closed:** 2026-10-17. Applied in `spec-viability.md` 4.5 (the supply gap on the same start windows as the jobs, the cyclic wrap, no gap for an activity open on every tick) and 8.4.1 (the lean reserve formula and the unwrapped start at or after the year end, the `N_open` count), settled by the owed-edits job. The numbers 304, 123, 184 and 664 are REF's, not changed.
- **Update (2026-10-18, plan 1-10 Phase 7):** the code reproduces the gaps at the file level: with the `staple` shelf life lowered to 100 ticks `storage_gap` reports 304 for `one`, 304 and 123 for the north and south of `two`, 123 for the south of `three`, and 304, 123 and 184 in `five` (`TestSolveSteadyState_StorageGapsOfTheReferenceChain`).

### N-278 The viability files are split for the Lizard limits

- **Areas:** CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (Phase 1, N-243).
- **Note:** `viability.go`, `viability_check.go` and `viability_table.go` are one concept in three root files (architecture.md 2.6 and `AGENTS.md` name them in Phase 1); a module that appends a column, a row or a report field appends to these files and keeps complexity 10, 8 parameters and about 500 lines.
- **Closed:** 2026-10-17. Applied: `viability.go`, `viability_check.go` and `viability_table.go` are named in `rules/architecture.md` 2.6 and `AGENTS.md`.

### N-282 A recipe on the lists of several roles divides its executions equally per household

- **Areas:** VIA, REF
- **Raised:** 2026-10-17, `plans/plan-phase-1-10.md` (D121).
- **Note:** For the jobs of the warm start the executions in progress of a recipe are divided by the households of all groups of the region that list it, each group getting the same share; a group over 10 000 is scaled down and listed in the census. REF's packs have no such recipe (checked in Phase 5 of the plan); the rule is for later packs and VIA 5.2 already flags the census ambiguous.
- **Closed:** 2026-10-17. Applied in `spec-viability.md` 8.5.1 and 9.4 (the executions of a recipe on the lists of several roles are divided equally per household of the region; a group over 10 000 is scaled and listed in the census). No pack uses it yet.

### N-287 The files of the world row

- **Areas:** CODE
- **Raised:** 2026-10-17, `plans/plan-phase-1-11.md` (D144).
- **Note:** The row adds root files `construct.go`, `construct_warm.go` (the application of the warm start), `fanin.go`, `phase_start.go`, `phase_production.go`, `phase_trade.go`, `phase_outcomes.go` and `phase_exit.go` beside the listed `world.go`, `tick.go`, `market.go` and `snapshot.go`, to stay under the Lizard limits of N-243. They are listed in architecture.md 2.6 and `AGENTS.md` in Phase 1 (the omission of `production_report.go` in N-243 is the lesson).
- **Update (2026-10-18, plan 1-10 Phase 1):** read, not touched by this row: it concerns the agent, estate, world, checkpoint or golden rows; no symbol this row consumes or provides changes. Left open.
- **Closed:** 2026-10-17. Applied: `construct.go`, `construct_warm.go`, `fanin.go` and the `phase_*.go` files are in `rules/architecture.md` 2.6 and `AGENTS.md`.

## Read receipt

If you are an agent about to write or change a spec, you quote the tag below in
your first message about that work, as required by
[AGENTS.md](../AGENTS.md). Reaching it means you have read this file in full. Do
not guess it or reuse one from memory.

When you edit this file (add, update or close a note), replace the tag with a
fresh random value in the same change (for example
`od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`). A stale tag in an agent's
message means it read an older version, and may have missed a note.

NOTES-TAG: e3d54093697f1844a53e3c91

