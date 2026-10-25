package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// Scenario S10 of spec-reference-chain.md 7.2 (plan 2-4 D286 to D288): the pack
// `one` with overlay/h200 (200 households in each of six roles, 1 200 in all) at
// world.scale 1, 4 and 10 and min_agents 10, which are 1 200, 300 and 120 agents.
// The layers are scalePacks[0]: the warm start is per household, so the totals
// are the same at every scale.

const (
	s10Ticks int64  = 365
	s10Seed  uint64 = 1
)

// s10Scales are the world scales of S10 and the agents each gives (6 groups of
// 200 households: 6 * 200 / 1, 6 * ceil(200 / 4), 6 * ceil(200 / 10); the floor
// of 10 is below every group's count).
var s10Scales = []struct{ scale, agents int64 }{{1, 1200}, {4, 300}, {10, 120}}

// TestScenarioS10_Exact verifies the exact half of S10, at construction: at each
// scale the weights of each group add up to 200, the agents are 1 200, 300 and
// 120 and the households 1 200, and the money and the units of each good held by
// the households are the households times the per-household amount, so equal at the
// three scales.
//
// Arithmetic: 6 groups * 200 households = 1 200 households; at scale 4 a group is
// ceil(200 / 4) = 50 agents of weight 4, 6 * 50 = 300; at scale 10 it is 20 agents
// of weight 10, 6 * 20 = 120.
//
// Requirement: FR-SCL-003, FR-SCL-007, FR-SCL-015, FR-SCL-018, FR-SCL-019, FR-REF-033
func TestScenarioS10_Exact(t *testing.T) {
	t.Parallel()
	var money []int64
	var goods []map[string]int64
	for _, s := range s10Scales {
		b := buildScaled(t, scalePacks[0].layers, s.scale, 10, nil)
		checkGroupsOf(t, b)
		checkTotalsOf(t, b)
		for k, g := range spawnedByGroup(b) {
			if g.households != 200 {
				t.Errorf("scale %d: group %v weights add up to %d, want 200", s.scale, k, g.households)
			}
		}
		if b.snap.Agents != s.agents || b.snap.Households != 1200 {
			t.Errorf("scale %d: snapshot %d agents, %d households, want %d and 1200", s.scale, b.snap.Agents, b.snap.Households, s.agents)
		}
		var m int64
		for _, r := range b.snap.Regions {
			m += int64(r.MoneyHouseholds)
		}
		g := map[string]int64{}
		for _, mk := range b.snap.Markets {
			g[b.names.Commodities[mk.State.Commodity]] += int64(mk.HeldByAgents)
		}
		money, goods = append(money, m), append(goods, g)
	}
	for i := 1; i < len(s10Scales); i++ {
		if money[i] != money[0] {
			t.Errorf("money at scale %d = %d, at scale 1 = %d, want equal", s10Scales[i].scale, money[i], money[0])
		}
		for c, q := range goods[0] {
			if goods[i][c] != q {
				t.Errorf("units of %s at scale %d = %d, at scale 1 = %d, want equal", c, s10Scales[i].scale, goods[i][c], q)
			}
		}
	}
}

// s10Obs keeps, per market, the sum of the published prices and the sum of the
// belief dispersion over the ticks the market had outcomes.
type s10Obs struct {
	price, disp map[[2]int64]*[2]int64 // (region, commodity) -> {sum, count}
}

func (o *s10Obs) add(m map[[2]int64]*[2]int64, k [2]int64, v int64) {
	if m[k] == nil {
		m[k] = &[2]int64{}
	}
	m[k][0] += v
	m[k][1]++
}

// Observe sums the price of each MarketCleared and the dispersion of each
// BeliefsUpdated, (HighCentre - LowCentre) * 10 000 / MeanCentre in basis points.
func (o *s10Obs) Observe(e economysim.Event) {
	switch e := e.(type) {
	case economysim.MarketCleared:
		o.add(o.price, [2]int64{int64(e.Region), int64(e.Commodity)}, int64(e.Price))
	case economysim.BeliefsUpdated:
		if bp, ok := shareBP(int64(e.HighCentre-e.LowCentre), int64(e.MeanCentre)); ok {
			o.add(o.disp, [2]int64{int64(e.Region), int64(e.Commodity)}, bp)
		}
	}
}

// s10Run steps the world of one scale for s10Ticks from tick 0 and returns the
// observer and the final snapshot. It holds a slot of scenSem.
func s10Run(t *testing.T, scale int64) (*s10Obs, economysim.Snapshot) {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	cfg := loadValidPack(t, scalePacks[0].layers...)
	cfg.World.Scale, cfg.World.MinAgents = scale, 10
	o := &s10Obs{price: map[[2]int64]*[2]int64{}, disp: map[[2]int64]*[2]int64{}}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(s10Seed), economysim.WithObserver(o))
	if err != nil {
		t.Fatalf("NewWorld(scale %d) error = %v, want nil", scale, err)
	}
	for i := int64(0); i < s10Ticks; i++ {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d (scale %d) error = %v, want nil", i, scale, err)
		}
	}
	fin, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot (scale %d) error = %v, want nil", scale, err)
	}
	return o, fin
}

// TestScenarioS10_PricesAcrossScales reports, for pack one with 200 households per
// role at scales 1, 4 and 10, the time-averaged price of each commodity and its gap
// to scale 1 in basis points, and the time-averaged dispersion of the belief
// centres of each market, and asserts the largest gap against s10.tolerance_bp and
// the smallest dispersion against s10.dispersion_min_bp only when the keys have a
// value. They have none (D287), and the pack collapses (N-312, N-314), so a gap here
// measures the collapse and no pass of FR-SCL-017 is claimed (D288). The report is
// written to testdata/chains/expected/one.scenarios.txt with -update.
//
// Requirement: FR-SCL-017, FR-SCL-010, FR-SCL-016, FR-AGT-052
func TestScenarioS10_PricesAcrossScales(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	if raceEnabled {
		t.Skip("S10 has one seed and is slow under -race; it runs in gate F (plan 2-4 D286)")
	}
	const scenario, pack = "S10", "one+h200"
	rep := newScenReport(fmt.Sprintf("S10 one+h200 scales 1/4/10, %d ticks, seed %d", s10Ticks, s10Seed))
	obs := make([]*s10Obs, len(s10Scales))
	var names economysim.Names
	for i, s := range s10Scales {
		o, fin := s10Run(t, s.scale)
		obs[i], names = o, fin.Names
		rep.num(s10Seed, fmt.Sprintf("agents at end scale %d", s.scale), fin.Agents)
		rep.num(s10Seed, fmt.Sprintf("households at end scale %d", s.scale), fin.Households)
	}
	var worstGap, minDisp int64
	haveDisp := false
	for k, p1 := range obs[0].price {
		c := names.Commodities[k[1]]
		base, ok := divHE(p1[0], p1[1])
		for i, s := range s10Scales {
			avg, aok := divHE(obs[i].price[k][0], obs[i].price[k][1])
			rep.num(s10Seed, fmt.Sprintf("average price %s scale %d", c, s.scale), avg)
			if d := obs[i].disp[k]; d != nil && d[1] > 0 {
				disp, _ := divHE(d[0], d[1])
				rep.num(s10Seed, fmt.Sprintf("dispersion bp %s scale %d", c, s.scale), disp)
				if !haveDisp || disp < minDisp {
					minDisp, haveDisp = disp, true
				}
			}
			if i == 0 || !ok || !aok || base == 0 {
				continue
			}
			gap, _ := divHE((avg-base)*10_000, base)
			rep.num(s10Seed, fmt.Sprintf("gap bp %s scale %d vs 1", c, s.scale), gap)
			worstGap = max(worstGap, absInt(gap))
		}
	}
	rep.num(s10Seed, "largest absolute gap bp", worstGap)
	checkBand(t, rep, scenario, pack, s10Seed, "s10.tolerance_bp", worstGap, true, func(b int64) bool { return worstGap > b })
	checkBand(t, rep, scenario, pack, s10Seed, "s10.dispersion_min_bp", minDisp, haveDisp, func(b int64) bool { return minDisp < b })
	rep.flush(t)
}
