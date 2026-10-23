package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	s01Ticks     int64 = 700
	s01ShockFrom int64 = 400
	s01ShockTo   int64 = 460
	// s01ReportBandBP is the band used to report a recovery time while
	// s01.band_bp has no value; it is not a pass criterion.
	s01ReportBandBP int64 = 1000
)

// TestScenarioS01_FuelSupplyShock verifies scenario S01 leg (a) of
// spec-reference-chain.md 7.2: pack one, 700 ticks, seeds 1 to 10, the yield of
// cut_fuel at 5 000 bp from tick 400 to 459 through the script and the Stage 1
// seam. Exact: no published price leaves its belief interval (FR-STB-008). O
// check (STY-S01 "S01 sign", FR-REF-023), asserted with or without a band: the
// mean fuel price over the shock exceeds P0, the mean over ticks 300 to 399.
// Reported, and asserted only when the key has a value: the recovery time, the
// long-run offset against the seed price and the unmet need. Leg (b), a visitor
// buying, waits for AddAgent and Direct (N-296). The outcomes are written to
// testdata/chains/expected/one.scenarios.txt with -update (D181).
//
// Requirement: FR-STB-018, FR-STB-008, FR-BEL-014, FR-BEL-036, FR-BEL-034,
// FR-CLR-019, FR-CLR-020, FR-REF-020, FR-REF-022, FR-REF-023, FR-REF-024,
// FR-REF-026, FR-REF-027
//
// Guard: a belief interval that lets a price out, or a supply cut that does not
// raise the price of its product, fails here.
func TestScenarioS01_FuelSupplyShock(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	script := loadScript(t, "s01.toml")
	rep := newScenReport(fmt.Sprintf("S01 one leg-a seeds 1-%d, 700 ticks", scenarioSeedCount()))
	t.Run("leg a supply", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				s01Seed(t, cfg, script, seed, rep)
			})
		}
	})
	rep.flush(t)
	t.Run("leg b demand", func(t *testing.T) {
		t.Skip("waits for AddAgent and Direct (integration.go, Stage 2), N-296")
	})
}

// unmetCounts returns the numerator and the denominator of the unmet share of
// a window that starts at tick from: the sum of the households that went unfed
// (UpkeepShortfall.Unfed, summed per tick) and the sum of the households at the
// start of each tick of the window (startHouseholds[i] for tick from+i). Both
// count households at the moment of the consumption phase: the upkeep shortfall
// is struck in phase 5, before the exits of phase 7, so the households it counts
// are those of the snapshot after the previous step, not the snapshot after the
// tick (which has lost the households that starved in the tick and made the
// share 30 000 bp on seed 7). within is false if any tick has more unfed
// households than households at its start, which would be a defect of the
// counts, so the share never exceeds 10 000 bp when within is true.
func unmetCounts(unfed []int64, from int64, startHouseholds []int64) (num, den int64, within bool) {
	within = true
	for i, h := range startHouseholds {
		var u int64
		if t := from + int64(i); t < int64(len(unfed)) {
			u = unfed[t]
		}
		if u > h {
			within = false
		}
		num += u
		den += h
	}
	return num, den, within
}

// s01Seed runs one seed of S01 leg (a) and records and asserts its checks.
func s01Seed(t *testing.T, cfg economysim.Config, script []shockStep, seed uint64, rep *scenReport) {
	const scenario, pack = "S01", "one"
	// startHouseholds[i] is the households at the start of tick s01ShockFrom+i,
	// which is the snapshot after the step of the tick before (see unmetCounts).
	startHouseholds := make([]int64, s01ShockTo-s01ShockFrom)
	run := runScenario(t, cfg, scenarioSpec{
		Seed: seed, Ticks: s01Ticks, Script: script,
		Snap:   func(tick int64) bool { return tick >= s01ShockFrom-1 && tick < s01ShockTo-1 },
		OnSnap: func(tick int64, s economysim.Snapshot) { startHouseholds[tick+1-s01ShockFrom] = s.Households },
	})
	o := run.Obs
	checkExact(t, rep, scenario, pack, seed, "market series in tick order", o.badSeq == "", "%s", o.badSeq)
	br := boundsViolations(cfg, run.Names, o, 3)
	rep.num(seed, "market-ticks outside a maker's belief interval (reported)", br.MakerOutside)
	rep.num(seed, "market-ticks outside the band with an estate sale (exempt)", br.EstateTicks)
	checkExact(t, rep, scenario, pack, seed, "price within belief bounds (FR-STB-008)", len(br.Violations) == 0, "measure price bounds: %v", br.Violations)

	fuel := marketOf(t, run, "north", "fuel")
	p0, ok0 := windowMean(fuel.Price, 300, s01ShockFrom)
	shock, ok1 := windowMean(fuel.Price, s01ShockFrom, s01ShockTo)
	if !ok0 || !ok1 {
		t.Fatalf("%s pack %s seed %d: fuel series has %d ticks, want %d", scenario, pack, seed, len(fuel.Price), s01Ticks)
	}
	rep.num(seed, "P0 fuel mean 300-399", p0)
	rep.num(seed, "fuel mean 400-459", shock)
	checkExact(t, rep, scenario, pack, seed, "O check: shock mean fuel price > P0", shock > p0,
		"measure mean fuel price over ticks 400-459 = %d, P0 = %d, want above", shock, p0)

	band, bandSet := bandFor(t, "s01.band_bp")
	reportBand := s01ReportBandBP
	if bandSet {
		reportBand = band
	}
	if rec, ok := recoveryTicks(fuel.Price, s01ShockTo, p0, reportBand, 10); ok {
		rep.num(seed, fmt.Sprintf("recovery ticks after 460 (band %d bp)", reportBand), rec)
		if bandSet {
			checkBand(t, rep, scenario, pack, seed, "s01.recover_ticks", rec, true, func(b int64) bool { return rec > b })
		}
	} else {
		rep.absent(seed, fmt.Sprintf("recovery ticks after 460 (band %d bp)", reportBand))
		if bandSet {
			if _, set := bandFor(t, "s01.recover_ticks"); set {
				checkExact(t, rep, scenario, pack, seed, "band s01.recover_ticks", false, "measure recovery time: the price never settles in the band")
			}
		}
	}

	last100, _ := windowMean(fuel.Price, s01Ticks-100, s01Ticks)
	if seedP, ok := seedPrice(cfg, "north", "fuel"); ok {
		if off, ok := offsetBP(last100, seedP); ok {
			rep.num(seed, "offset bp last 100 vs seed price", off)
			checkBand(t, rep, scenario, pack, seed, "s01.offset_bp", off, true, func(b int64) bool { return absInt(off) > b })
		}
	}
	unfed, den, within := unmetCounts(o.unfed, s01ShockFrom, startHouseholds)
	checkExact(t, rep, scenario, pack, seed, "unfed households never exceed the households at the start of the tick", within,
		"measure unmet: a tick of %d-%d has more unfed households than households at its start (unfed %v, households %v)", s01ShockFrom, s01ShockTo, o.unfed, startHouseholds)
	if unmet, ok := shareBP(unfed, den); ok {
		rep.num(seed, "unmet bp over 400-459", unmet)
		checkBand(t, rep, scenario, pack, seed, "s01.unmet_bp", unmet, true, func(b int64) bool { return unmet > b })
	} else {
		rep.absent(seed, "unmet bp over 400-459")
	}
	rep.num(seed, "households at end", run.Final.Households)
}
