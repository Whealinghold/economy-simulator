package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// townAsOneRegion folds the regions of a pack into its first region: the same
// roles and the same total households per role, the yields of the first region,
// no edge and no override. It is the control of scenario S29.
func townAsOneRegion(cfg economysim.Config) economysim.Config {
	town := cfg.Regions[0].ID
	one := cfg
	one.Regions = slices.Clone(cfg.Regions[:1])
	one.Edges, one.RegionPrices = nil, nil
	one.Yields = slices.DeleteFunc(slices.Clone(cfg.Yields), func(y economysim.YieldProfile) bool { return y.Region != town })
	one.Groups = nil
	for _, g := range cfg.Groups {
		i := slices.IndexFunc(one.Groups, func(h economysim.Group) bool { return h.Role == g.Role })
		if i < 0 {
			g.Region = town
			one.Groups = append(one.Groups, g)
			continue
		}
		one.Groups[i].Households += g.Households
	}
	return one
}

// TestScenarioS29_TownAsOneRegion verifies the graph leg of scenario S29 (REG-A2,
// spec-reference-chain.md 7.6): a town as one region and the same town as three
// regions joined by the edges of trade-three.toml (south in the middle, 100 to
// each of north and far_south, 200 between them) hold the same households per
// role; each runs under the audit for 40 ticks over the scenario seeds, conserving
// goods and money; the graph of the split town is one component and that of the
// town one region with no edge; and every market of every region is reported in
// every tick (a MarketCleared for each region and commodity, with its price age
// and unmatched units, thin or not). The dispersion of the three prices is
// logged, not asserted: its band waits for TRN and MER (the price leg), as the
// scenario row of the register says.
//
// Requirement: FR-REG-015, FR-REG-016
func TestScenarioS29_TownAsOneRegion(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	split := loadValidPack(t, withTrade("three")...)
	town := townAsOneRegion(split)
	if err := town.Validate(); err != nil {
		t.Fatalf("Validate(the town as one region) = %v, want nil", err)
	}
	households := func(c economysim.Config) int64 {
		var n int64
		for _, g := range c.Groups {
			n += g.Households
		}
		return n
	}
	if a, b := households(town), households(split); a != b {
		t.Fatalf("households of the town = %d, of the split town = %d, want equal", a, b)
	}
	for _, seed := range seedList(scenarioSeedCount()) {
		for name, cfg := range map[string]economysim.Config{"town": town, "split": split} {
			t.Run(fmt.Sprintf("%s seed %d", name, seed), func(t *testing.T) {
				t.Parallel()
				scenSem <- struct{}{}
				defer func() { <-scenSem }()
				const ticks = 40
				res := runPack(t, cfg, runSpec{Seed: seed, Ticks: ticks, Observe: true, Audit: true})
				regions, commodities := len(cfg.Regions), len(cfg.Commodities)
				cleared := 0
				var prices []economysim.Price
				for _, e := range res.Events {
					if m, ok := e.(economysim.MarketCleared); ok {
						cleared++
						if m.Tick == ticks-1 && m.Commodity == 0 {
							prices = append(prices, m.Price)
						}
					}
				}
				if want := ticks * regions * commodities; cleared != want {
					t.Errorf("%s seed %d: %d MarketCleared in %d ticks, want %d (regions %d x commodities %d)", name, seed, cleared, ticks, want, regions, commodities)
				}
				wantComps := 1
				if res.Final.Components != int64(wantComps) && regions > 1 {
					t.Errorf("%s seed %d: Components = %d, want %d", name, seed, res.Final.Components, wantComps)
				}
				if regions == 1 && res.Final.Components != 0 {
					t.Errorf("%s seed %d: Components = %d for one region, want 0", name, seed, res.Final.Components)
				}
				t.Logf("%s seed %d: prices of commodity 0 in the last tick = %v", name, seed, prices)
			})
		}
	}
}
