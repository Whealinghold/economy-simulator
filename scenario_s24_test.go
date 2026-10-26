package economysim_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	s24Ticks int64 = 250
	s24Last  int64 = 100
	// s24Target is the TargetRatio of the run above the ceiling: 2.0 against the
	// expected ceiling of about 1.32 (spec-demand.md 7.1, the 20 000 row).
	s24Target = 20_000
	// s24GlutFactor is the stated bound of the glut: with the target above the
	// ceiling no produced good's stock exceeds this many times the larger maximum
	// of the same good in the wants-off and default wants-on runs of the seed.
	s24GlutFactor = 2
)

// s24Bound is the hard bound 1 + Wmax / U of spec-demand.md 7.1 in basis points,
// the largest over the roles: U is the upkeep cost at the reference prices and
// Wmax a household's spending with every want at its schedule ceiling,
// sum over its ladder of Ref * Quantity * S * Ceiling / (Period * 10^8) with
// Ref the weight-weighted mean reference price of the items and S the taste in
// basis points (no region taste row in the pack, so the region scale is 10 000).
func s24Bound(cfg economysim.Config) int64 {
	var u int64
	for _, it := range cfg.World.Upkeep {
		u += refPrice(cfg, it.Commodity) * int64(it.Quantity)
	}
	var best int64
	for _, ro := range cfg.Roles {
		var wmax int64
		for _, ta := range cfg.Modules.Demand.Tastes {
			if ta.Role != ro.ID {
				continue
			}
			for _, w := range cfg.Modules.Demand.Wants {
				if w.ID != ta.Want {
					continue
				}
				var sum, wt int64
				for _, it := range w.Items {
					sum += refPrice(cfg, it.Commodity) * it.Weight
					wt += it.Weight
				}
				wmax += sum / wt * w.Quantity * ta.Scale * w.Ceiling / (w.Period * 100_000_000)
			}
		}
		best = max(best, (u+wmax)*10_000/u)
	}
	return best
}

// s24TargetDoc raises the TargetRatio of every role of cfg to s24Target.
func s24TargetDoc(cfg economysim.Config) []byte {
	var b strings.Builder
	for _, ro := range cfg.Roles {
		fmt.Fprintf(&b, "[role.%s]\ntarget_ratio = %d\n", ro.ID, s24Target)
	}
	return []byte(b.String())
}

// s24Result is what one run of S24 keeps.
type s24Result struct {
	ratioBP        int64
	ratioOK        bool
	households     int64
	end, max       map[string]int64
	halted, haltEv int64
	violations     []string
	badSeq         string
}

// s24Run runs one world of S24: the household-weighted mean welfare ratio over
// the last 100 ticks (all roles of north) and the unsold stock per produced good.
func s24Run(t *testing.T, cfg economysim.Config, seed uint64) s24Result {
	goods := producedGoods(cfg)
	res := s24Result{end: map[string]int64{}, max: map[string]int64{}}
	var sum, hh int64
	run := runScenario(t, cfg, scenarioSpec{
		Seed: seed, Ticks: s24Ticks,
		Snap: func(int64) bool { return true },
		OnSnap: func(tick int64, s economysim.Snapshot) {
			north := int64(indexOf(s.Names.Regions, "north"))
			for _, m := range s.Markets {
				id := s.Names.Commodities[m.State.Commodity]
				if int64(m.State.Region) != north || !slices.Contains(goods, id) {
					continue
				}
				stock := int64(m.HeldByAgents + m.HeldByEstate)
				res.max[id] = max(res.max[id], stock)
				res.end[id] = stock
			}
			if tick < s24Ticks-s24Last {
				return
			}
			for _, r := range s.Roles {
				if int64(r.Region) == north && r.HasMeanRatio {
					sum += r.MeanRatioBP * r.RatioHouseholds
					hh += r.RatioHouseholds
				}
			}
		},
	})
	res.ratioBP, res.ratioOK = divHE(sum, hh)
	res.households = run.Final.Households
	res.halted, res.haltEv = run.Obs.halted, run.Obs.haltEv
	res.violations = boundsViolations(cfg, run.Names, run.Obs, 3).Violations
	res.badSeq = run.Obs.badSeq
	return res
}

// TestScenarioS24_ClosedEconomyLadder verifies scenario S24 of
// spec-reference-chain.md 7.7: pack one (the module off) and pack one-demand
// with the default TargetRatio and with every role's TargetRatio at 20 000 (above
// the ceiling), 250 ticks, seeds 1 to 10. Exact: no price leaves its belief
// interval in any run (FR-STB-008), and the household-weighted mean welfare ratio
// of the last 100 ticks of a run with the wants is at most 1 + Wmax / U
// (FR-DEM-031, restated in s24Bound from the configuration). Recorded, not
// asserted, because HouseholdIncome is not OK in version 1 (D319): the mean
// ratio of each run (near 10 000 without the wants, above it with them is
// FR-DEM-030 and FR-STB-027), the households at the end, the unsold stock and the
// ProductionHalted counts of each run (N-087, the halt comparison is reported).
// The glut is asserted bounded (FR-STB-027): with the target above the ceiling
// the maximum stock of each produced good stays within s24GlutFactor times the
// larger maximum of the same good in the other two runs; a value from a reviewed
// run replaces the factor when OBS adds the income fields. The outcomes are written to
// testdata/chains/expected/one-demand.scenarios.txt with -update.
//
// Requirement: FR-DEM-030, FR-DEM-031, FR-AGT-070, FR-STB-027, FR-STB-008
//
// Guard: a ratio above the identity bound means want spending created money.
func TestScenarioS24_ClosedEconomyLadder(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S24", "one-demand"
	off := loadValidPack(t, packOne...)
	on := loadValidPack(t, packOneDemand...)
	above := loadPackBytes(t, s24TargetDoc(on), packOneDemand...)
	bound := s24Bound(on)
	rep := newPackScenReport(pack, fmt.Sprintf("S24 one-demand wants off, on, target above seeds 1-%d, %d ticks", scenarioSeedCount(), s24Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				rep.num(seed, "bound 1+Wmax/U bp", bound)
				var results []s24Result
				for _, r := range []struct {
					name  string
					cfg   economysim.Config
					wants bool
				}{{"wants off (pack one)", off, false}, {"wants on", on, true}, {"wants on, target 20000", above, true}} {
					res := s24Run(t, r.cfg, seed)
					results = append(results, res)
					checkExact(t, rep, scenario, pack, seed, "market series in tick order", res.badSeq == "", "%s: %s", r.name, res.badSeq)
					checkExact(t, rep, scenario, pack, seed, "price within belief bounds (FR-STB-008)", len(res.violations) == 0,
						"%s: measure price bounds: %v", r.name, res.violations)
					rep.num(seed, "households at end, "+r.name, res.households)
					rep.num(seed, "ProductionHalted events, "+r.name, res.haltEv)
					for _, g := range producedGoods(r.cfg) {
						rep.num(seed, fmt.Sprintf("stock max %s, %s", g, r.name), res.max[g])
					}
					if !res.ratioOK {
						rep.absent(seed, "mean ratio bp, "+r.name)
						continue
					}
					rep.num(seed, "mean ratio bp, "+r.name, res.ratioBP)
					if r.wants {
						checkExact(t, rep, scenario, pack, seed, "mean ratio at most 1+Wmax/U (FR-DEM-031)", res.ratioBP <= bound,
							"%s: mean ratio %d bp above the bound %d", r.name, res.ratioBP, bound)
					}
				}
				for _, g := range producedGoods(above) {
					worst := max(results[0].max[g], results[1].max[g])
					checkExact(t, rep, scenario, pack, seed, "glut with the target above the ceiling is bounded (FR-STB-027)", results[2].max[g] <= s24GlutFactor*worst,
						"%s: maximum stock %d with target %d, above %d times the larger maximum of the other two runs (%d)", g, results[2].max[g], s24Target, s24GlutFactor, worst)
				}
			})
		}
	})
	rep.flush(t)
}
