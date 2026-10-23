package economysim_test

import (
	"fmt"
	"slices"
	"testing"
)

const (
	s16Ticks    int64 = 1095
	s16LastYear int64 = 365
)

// TestScenarioS16_StapleOverhang verifies scenario S16 of
// spec-reference-chain.md 7.2: pack one + overlay/overhang (growers start with
// three times the staple), 1 095 ticks, seeds 1 to 10. Exact: no published price
// leaves its belief interval (FR-STB-008). Reported, and asserted against
// s16.pinned_share_bp only when the key has a value (otherwise a share above a
// bound is the F5 report of the scenario): the share of ticks of the last year
// in which the staple price is at a bound of its interval, with the share of
// ticks in which more than half of the belief outcomes ended at a bound beside
// it. The outcomes are written to testdata/chains/expected/one.scenarios.txt
// with -update (D181).
//
// Requirement: FR-AGT-036, FR-AGT-070, FR-BEL-004, FR-REF-020, FR-REF-022,
// FR-REF-024, FR-REF-026, FR-REF-027
//
// Guard: a belief interval that lets a price out fails here.
func TestScenarioS16_StapleOverhang(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S16", "one+overhang"
	cfg := loadValidPack(t, append(slices.Clone(packOne), "overlay/overhang.toml")...)
	rep := newScenReport(fmt.Sprintf("S16 one+overhang seeds 1-%d, 1095 ticks", scenarioSeedCount()))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				run := runScenario(t, cfg, scenarioSpec{Seed: seed, Ticks: s16Ticks})
				o := run.Obs
				checkExact(t, rep, scenario, pack, seed, "market series in tick order", o.badSeq == "", "%s", o.badSeq)
				br := boundsViolations(cfg, run.Names, o, 3)
				checkExact(t, rep, scenario, pack, seed, "price within belief bounds (FR-STB-008)", len(br.Violations) == 0,
					"measure price bounds: %v", br.Violations)
				staple := marketOf(t, run, "north", "staple")
				atBound := priceAtBoundFlags(cfg, refPrice(cfg, "staple"), staple)
				var n int64
				for _, f := range atBound[s16Ticks-s16LastYear:] {
					if f {
						n++
					}
				}
				share, _ := shareBP(n, s16LastYear)
				rep.num(seed, "staple price at a bound, share bp of last year", share)
				checkBand(t, rep, scenario, pack, seed, "s16.pinned_share_bp", share, true, func(b int64) bool { return share > b })
				var pinned int64
				for _, f := range pinnedFlags(staple, s16Ticks)[s16Ticks-s16LastYear:] {
					if f {
						pinned++
					}
				}
				pshare, _ := shareBP(pinned, s16LastYear)
				rep.num(seed, "staple belief outcomes mostly at a bound, share bp of last year", pshare)
				rep.num(seed, "households at end", run.Final.Households)
			})
		}
	})
	rep.flush(t)
}
