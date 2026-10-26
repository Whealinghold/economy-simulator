package economysim_test

import (
	"fmt"
	"testing"
)

const (
	s23Ticks int64 = 200
	s23Last  int64 = 100
	// demandBandsPath holds the band keys of the demand scenarios (S23, S25).
	demandBandsPath = "testdata/chains/bands/one-demand.toml"
	// treatRef is the reference price of treat_a in the warm document (664); the
	// two starts are 10% above and below it, 664 * 11 / 10 = 730 and
	// 664 * 9 / 10 = 597.6, rounded to 598.
	treatRef = 664
)

// s23Doc is the one-key document of an S23 run: the elasticity of the treat want
// and the starting price of treat_a.
func s23Doc(elasticity, start int64) []byte {
	return []byte(fmt.Sprintf("[modules.demand.want.treat]\nelasticity = %d\n[warm_start.region.north.price]\ntreat_a = %d\n", elasticity, start))
}

// TestScenarioS23_LuxuryRoleLabourLoop verifies scenario S23 of
// spec-reference-chain.md 7.7 on pack one-demand: the elasticity of the treat
// want at 5 000, 10 000, 15 000 and 30 000 (generated one-key documents), each
// started with the price of treat_a 10% above and 10% below the reference,
// 200 ticks, seeds 1 to 10. Exact: no published price leaves its belief
// interval (FR-STB-008) and the market series is in tick order. Recorded, not
// asserted while s23.band_bp has no value: the mean treat_a price of the last
// 100 ticks from each start, its offset from the reference in basis points and
// the gap between the two starts, which shows whether the price converges and
// where the lower edge of the basin lies (spec-demand.md 6.3, DEM-A2). The
// outcomes are written to testdata/chains/expected/one-demand.scenarios.txt
// with -update.
//
// Requirement: FR-DEM-010, FR-DEM-011, FR-STB-026, FR-STB-008
//
// Guard: a schedule that lets a price out of its belief interval, or a market
// that stops publishing, fails here whatever the elasticity.
func TestScenarioS23_LuxuryRoleLabourLoop(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S23", "one-demand"
	band, set := bandIn(t, demandBandsPath, "s23.band_bp")
	rep := newPackScenReport(pack, fmt.Sprintf("S23 one-demand elasticity x start seeds 1-%d, %d ticks", scenarioSeedCount(), s23Ticks))
	starts := []struct {
		name  string
		price int64
	}{{"above", treatRef * 11 / 10}, {"below", (treatRef*9 + 5) / 10}}
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				for _, e := range []int64{5000, 10000, 15000, 30000} {
					var mean [2]int64
					for i, st := range starts {
						cfg := loadPackBytes(t, s23Doc(e, st.price), packOneDemand...)
						run := runScenario(t, cfg, scenarioSpec{Seed: seed, Ticks: s23Ticks})
						label := fmt.Sprintf("e=%d start %s", e, st.name)
						checkExact(t, rep, scenario, pack, seed, "market series in tick order", run.Obs.badSeq == "", "%s: %s", label, run.Obs.badSeq)
						br := boundsViolations(cfg, run.Names, run.Obs, 3)
						checkExact(t, rep, scenario, pack, seed, "price within belief bounds (FR-STB-008)", len(br.Violations) == 0,
							"%s: measure price bounds: %v", label, br.Violations)
						treat := marketOf(t, run, "north", "treat_a")
						m, ok := windowMean(treat.Price, s23Ticks-s23Last, s23Ticks)
						if !ok {
							t.Fatalf("%s pack %s seed %d %s: treat_a series has %d ticks, want %d", scenario, pack, seed, label, len(treat.Price), s23Ticks)
						}
						mean[i] = m
						var traded int64
						for _, v := range treat.Volume[s23Ticks-s23Last:] {
							if v > 0 {
								traded++
							}
						}
						rep.num(seed, fmt.Sprintf("ticks with a treat_a trade in the last 100, %s", label), traded)
						rep.num(seed, fmt.Sprintf("treat_a mean last 100, %s", label), m)
						off, _ := shareBP(m-treatRef, treatRef)
						rep.num(seed, fmt.Sprintf("offset bp from reference, %s", label), off)
						if set {
							checkExact(t, rep, scenario, pack, seed, "band s23.band_bp", absInt(off) <= band, "%s: offset %d against %d", label, off, band)
						}
					}
					gap, _ := shareBP(absInt(mean[0]-mean[1]), treatRef)
					rep.num(seed, fmt.Sprintf("gap bp between the two starts, e=%d", e), gap)
				}
			})
		}
	})
	rep.flush(t)
}
