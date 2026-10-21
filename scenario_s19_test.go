package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	s19Ticks      int64 = 1095
	s19LastWindow int64 = 365
	// s19HaltOff is produce_halt_ratio of the run with the halt off: the largest
	// ratio (bounds.go MaxBasisPoints), a factor of 100 of the output target.
	s19HaltOff = economysim.MaxBasisPoints
)

// s19Result is what one run of S19 keeps.
type s19Result struct {
	end, max       map[string]int64 // unsold stock per produced good: at the end and the maximum over the run
	ratioSum, ratH map[string]int64 // sum of MeanRatioBP * RatioHouseholds and of RatioHouseholds per role, last window
	poolShareBP    int64
	poolOK         bool
	halted, haltEv int64
	volumeLast     map[string]int64 // units traded per good over the last window
	violations     []string
}

// producedGoods lists the commodities some recipe makes as a non-discard output.
func producedGoods(cfg economysim.Config) []string {
	var out []string
	for _, r := range cfg.Recipes {
		for _, o := range r.Outputs {
			if !o.Discard && !slices.Contains(out, o.Commodity) {
				out = append(out, o.Commodity)
			}
		}
	}
	slices.Sort(out)
	return out
}

// s19Run runs one world of S19 and keeps the unsold stock of each produced good
// at every tick (the households' and the estate's holdings of the market rows),
// the household-weighted mean welfare ratio per role over the last 365 ticks, the
// pool share of money at the end and the halt counts.
func s19Run(t *testing.T, cfg economysim.Config, seed uint64) s19Result {
	goods := producedGoods(cfg)
	res := s19Result{end: map[string]int64{}, max: map[string]int64{}, ratioSum: map[string]int64{}, ratH: map[string]int64{}, volumeLast: map[string]int64{}}
	run := runScenario(t, cfg, scenarioSpec{
		Seed: seed, Ticks: s19Ticks,
		Snap: func(int64) bool { return true },
		OnSnap: func(tick int64, s economysim.Snapshot) {
			north := int64(indexOf(s.Names.Regions, "north"))
			for _, m := range s.Markets {
				if int64(m.State.Region) != north {
					continue
				}
				id := s.Names.Commodities[m.State.Commodity]
				if !slices.Contains(goods, id) {
					continue
				}
				stock := int64(m.HeldByAgents + m.HeldByEstate)
				res.max[id] = max(res.max[id], stock)
				res.end[id] = stock
			}
			if tick < s19Ticks-s19LastWindow {
				return
			}
			for _, r := range s.Roles {
				if int64(r.Region) == north && r.HasMeanRatio {
					id := s.Names.Roles[r.Role]
					res.ratioSum[id] += r.MeanRatioBP * r.RatioHouseholds
					res.ratH[id] += r.RatioHouseholds
				}
			}
		},
	})
	for k, m := range run.Obs.markets {
		if k[0] != int64(indexOf(run.Names.Regions, "north")) {
			continue
		}
		id := run.Names.Commodities[k[1]]
		for _, v := range m.Volume[s19Ticks-s19LastWindow:] {
			res.volumeLast[id] += v
		}
	}
	north := indexOf(run.Names.Regions, "north")
	res.poolShareBP, res.poolOK = shareBP(int64(run.Final.Regions[north].MoneyPool), int64(run.Final.Money))
	res.halted, res.haltEv = run.Obs.halted, run.Obs.haltEv
	res.violations = boundsViolations(cfg, run.Names, run.Obs, 3).Violations
	return res
}

// TestScenarioS19_HaltOnAndOff verifies scenario S19 of
// spec-reference-chain.md 7.2 for the Stage 1 measures (D175): pack one with the
// production halt on (the default) and off (produce_halt_ratio 1 000 000 by a
// one-key document layered last), 1 095 ticks, seeds 1 to 10. Exact: no price
// leaves its belief interval in either run (FR-STB-008). O check (FR-REF-023),
// asserted without a band: for at least one produced good g the end stock with
// the halt off exceeds the maximum stock of g over the halt-on run, so the
// measure can see the glut. Reported: the household-weighted mean welfare ratio
// per role over the last 365 ticks (RoleRow.MeanRatioBP; the aggregate-ratio band
// is reported only, HouseholdIncome is not OK in version 1), the share of money in
// the pool (MoneyPool over Money; the escrow term is 0 in Stage 1), the unsold
// stock per produced good and the ProductionHalted counts. Asserted on the
// halt-on run only when the key has a value: m05.pool_share_bp, and
// s19.stock_bound_bp against the halt threshold, taken as ProduceHaltRatio times
// the units traded per tick of the last 365 ticks (the output target is not in
// the events or the snapshot). The outcomes are written to
// testdata/chains/expected/one.scenarios.txt with -update (D181).
//
// Requirement: FR-STB-021, FR-AGT-070, FR-WEL-032, FR-WEL-031, FR-AGT-066,
// FR-REF-020, FR-REF-022, FR-REF-023, FR-REF-024, FR-REF-026, FR-REF-027
//
// Guard: a halt that does not stop the build-up of a glut, or a measure that
// cannot see one, fails the O check.
func TestScenarioS19_HaltOnAndOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S19", "one"
	on := loadValidPack(t, packOne...)
	off := loadPackBytes(t, []byte(fmt.Sprintf("[world]\nproduce_halt_ratio = %d\n", s19HaltOff)), packOne...)
	goods := producedGoods(on)
	rep := newScenReport("S19 one halt on/off seeds 1-10, 1095 ticks")
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeeds) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				ron, roff := s19Run(t, on, seed), s19Run(t, off, seed)
				checkExact(t, rep, scenario, pack, seed, "price within belief bounds, halt on (FR-STB-008)", len(ron.violations) == 0,
					"measure price bounds: %v", ron.violations)
				checkExact(t, rep, scenario, pack, seed, "price within belief bounds, halt off (FR-STB-008)", len(roff.violations) == 0,
					"measure price bounds: %v", roff.violations)
				var glut []string
				for _, g := range goods {
					rep.num(seed, "stock max halt-on "+g, ron.max[g])
					rep.num(seed, "stock end halt-on "+g, ron.end[g])
					rep.num(seed, "stock end halt-off "+g, roff.end[g])
					if roff.end[g] > ron.max[g] {
						glut = append(glut, g)
					}
				}
				checkExact(t, rep, scenario, pack, seed, "O check: halt-off end stock exceeds halt-on maximum for some good", len(glut) > 0,
					"measure unsold stock: no good with a halt-off end stock above the halt-on maximum (on max %v, off end %v)", ron.max, roff.end)
				rep.num(seed, "goods where the halt-off stock exceeds the halt-on maximum", int64(len(glut)))
				for _, role := range sortedRoleKeys(ron.ratH) {
					if v, ok := divHE(ron.ratioSum[role], ron.ratH[role]); ok {
						rep.num(seed, "mean ratio bp halt-on "+role, v)
					}
				}
				for _, role := range sortedRoleKeys(roff.ratH) {
					if v, ok := divHE(roff.ratioSum[role], roff.ratH[role]); ok {
						rep.num(seed, "mean ratio bp halt-off "+role, v)
					}
				}
				if ron.poolOK {
					rep.num(seed, "pool share bp of money, halt-on", ron.poolShareBP)
					checkBand(t, rep, scenario, pack, seed, "m05.pool_share_bp", ron.poolShareBP, true, func(b int64) bool { return ron.poolShareBP > b })
				}
				rep.num(seed, "ProductionHalted events halt-on", ron.haltEv)
				rep.num(seed, "ProductionHalted households halt-on", ron.halted)
				rep.num(seed, "ProductionHalted events halt-off", roff.haltEv)
				if bound, set := bandFor(t, "s19.stock_bound_bp"); set {
					for _, g := range goods {
						perTick, ok := divHE(ron.volumeLast[g], s19LastWindow)
						if !ok || perTick == 0 {
							continue
						}
						threshold, _ := divHE(perTick*on.World.ProduceHaltRatio, 10_000)
						if share, ok := shareBP(ron.end[g], threshold); ok {
							rep.num(seed, "stock end as bp of the halt threshold (proxy) "+g, share)
							checkBand(t, rep, scenario, pack, seed, "s19.stock_bound_bp", share, true, func(int64) bool { return share > bound })
						}
					}
				}
			})
		}
	})
	rep.flush(t)
}

// sortedRoleKeys returns the keys of m in order.
func sortedRoleKeys(m map[string]int64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
