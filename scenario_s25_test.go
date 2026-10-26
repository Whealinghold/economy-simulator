package economysim_test

import (
	"fmt"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	s25Ticks int64 = 500
	s25Year  int64 = 365
	s25Block int64 = 10
)

// s25Reserve is the reserve money of one household at the warm prices: ReserveTicks
// times the upkeep cost (3 * (2 * 96 + 628 + 179) = 3 * 999 = 2 997 in the pack).
func s25Reserve(cfg economysim.Config) int64 {
	price := map[string]int64{}
	for _, p := range cfg.WarmStart.Prices {
		price[p.Commodity] = int64(p.Price)
	}
	var u int64
	for _, it := range cfg.World.Upkeep {
		u += price[it.Commodity] * int64(it.Quantity)
	}
	return cfg.World.ReserveTicks * u
}

// s25Doc is the one-key document of an S25 row: every warm group's money per
// household set to num/den of the reserve, so the total money is num/den times the
// sum of the reserves, and the substitution exponent of the treat want.
func s25Doc(cfg economysim.Config, num, den, substitution int64) []byte {
	var b strings.Builder
	reserve := s25Reserve(cfg)
	for _, g := range cfg.WarmStart.Groups {
		fmt.Fprintf(&b, "[warm_start.region.%s.group.%s]\nmoney = %d\n", g.Region, g.Role, reserve*num/den)
	}
	fmt.Fprintf(&b, "[modules.demand.want.treat]\nsubstitution = %d\n", substitution)
	return []byte(b.String())
}

// shareSwingBP is the swing of the share of a among a and b over the series:
// the sums of both are taken over blocks of block ticks from `from`, the share of
// each block that traded is a / (a + b) in basis points, and the swing is the
// largest share less the smallest. ok is false when fewer than two blocks traded.
func shareSwingBP(a, b []int64, from, block int64) (int64, bool) {
	var lo, hi, n int64
	for i := from; i+block <= int64(min(len(a), len(b))); i += block {
		var sa, sb int64
		for j := i; j < i+block; j++ {
			sa, sb = sa+a[j], sb+b[j]
		}
		share, ok := shareBP(sa, sa+sb)
		if !ok {
			continue
		}
		if n == 0 || share < lo {
			lo = share
		}
		if n == 0 || share > hi {
			hi = share
		}
		n++
	}
	return hi - lo, n >= 2
}

// TestScenarioS25_MoneyBelowReservesAndSubstitution verifies scenario S25 of
// spec-reference-chain.md 7.7 on pack one-demand: every household's money at 0.5,
// 1 and 2 times its reserve (so the total money is that multiple of the sum of the
// reserves; one-key generated documents) with the substitution exponent of the
// treat want at 4 (its alternatives differ in price by 62%: 664 and 1 079), 500
// ticks, seeds 1 to 10. Exact: no price leaves its belief interval (FR-STB-008),
// and at the 0.5 and 1 rows, where no household has an excess over its reserve at
// tick 0, no unit of a want is funded at tick 0 (FR-DEM-020, FR-STB-028; the
// WantShortfall rows show a plan of more than 0 units, so the test cannot pass
// with no want planned). Recorded: the units planned and funded at tick 0 of
// every row, and the swing of the treat_a share of the two alternatives' volume
// in blocks of 10 ticks over the last 365 ticks (a value from a reviewed run goes
// to s25.swing_bp in bands/one-demand.toml and is then asserted). The outcomes
// are written to testdata/chains/expected/one-demand.scenarios.txt with -update.
//
// Requirement: FR-DEM-017, FR-DEM-020, FR-DEM-018, FR-STB-028, FR-STB-008
//
// Guard: a want funded by a household at or below its reserve would spend money
// the saving rule keeps for upkeep.
func TestScenarioS25_MoneyBelowReservesAndSubstitution(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S25", "one-demand"
	base := loadValidPack(t, packOneDemand...)
	swingBand, swingSet := bandIn(t, demandBandsPath, "s25.swing_bp")
	rep := newPackScenReport(pack, fmt.Sprintf("S25 one-demand money x reserves, substitution 4 seeds 1-%d, %d ticks", scenarioSeedCount(), s25Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				for _, row := range []struct {
					name     string
					num, den int64
				}{{"0.5", 1, 2}, {"1", 1, 1}, {"2", 2, 1}} {
					cfg := loadPackBytes(t, s25Doc(base, row.num, row.den, 4), packOneDemand...)
					run := runScenario(t, cfg, scenarioSpec{Seed: seed, Ticks: s25Ticks})
					label := "money " + row.name + " x reserves"
					checkExact(t, rep, scenario, pack, seed, "market series in tick order", run.Obs.badSeq == "", "%s: %s", label, run.Obs.badSeq)
					br := boundsViolations(cfg, run.Names, run.Obs, 3)
					checkExact(t, rep, scenario, pack, seed, "price within belief bounds (FR-STB-008)", len(br.Violations) == 0,
						"%s: measure price bounds: %v", label, br.Violations)
					o := run.Obs
					if len(o.wantPlanned) == 0 {
						t.Fatalf("%s pack %s seed %d %s: no WantShortfall event in %d ticks, want a want planned", scenario, pack, seed, label, s25Ticks)
					}
					rep.num(seed, "units planned at tick 0, "+label, o.wantPlanned[0])
					rep.num(seed, "units funded at tick 0, "+label, o.wantFunded[0])
					if row.num <= row.den {
						checkExact(t, rep, scenario, pack, seed, "no want funded at or below the reserve (FR-DEM-020)", o.wantPlanned[0] > 0 && o.wantFunded[0] == 0,
							"%s: planned %d, funded %d units at tick 0, want planned above 0 and funded 0", label, o.wantPlanned[0], o.wantFunded[0])
					}
					a, b := marketOf(t, run, "north", "treat_a"), marketOf(t, run, "north", "treat_b")
					if swing, ok := shareSwingBP(a.Volume, b.Volume, s25Ticks-s25Year, s25Block); ok {
						rep.num(seed, "treat_a share swing bp over the last 365 ticks, "+label, swing)
						if swingSet {
							checkExact(t, rep, scenario, pack, seed, "band s25.swing_bp", swing <= swingBand, "%s: swing %d against %d", label, swing, swingBand)
						}
					} else {
						rep.absent(seed, "treat_a share swing bp over the last 365 ticks, "+label)
					}
				}
			})
		}
	})
	rep.flush(t)
}
