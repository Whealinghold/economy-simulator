package indicator_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// The invented river valley of spec-indicators.md 18.1: two regions, four
// commodities, two roles and four recipes. The words are data; nothing branches
// on them. Every number built on it is a placeholder (N-027).
const (
	alpha, beta economysim.RegionID = 0, 1

	grain, loaf, wood, cart economysim.CommodityID = 0, 1, 2, 3

	grower, baker economysim.RoleID = 0, 1

	recGrow, recBake, recCut, recMakeCart economysim.RecipeID = 0, 1, 2, 3
)

// valleyNames is the names table of the valley.
func valleyNames() economysim.Names {
	return economysim.Names{
		Regions:     []string{"alpha", "beta"},
		Commodities: []string{"grain", "loaf", "wood", "cart"},
		Roles:       []string{"grower", "baker"},
		Recipes:     []string{"grow", "bake", "cut", "make_cart"},
	}
}

// valleyHash is the configuration hash of every valley snapshot: that of valleyConfig, which ValueAdded checks.
var valleyHash = economysim.HashConfig(valleyConfig())

// valleySnap returns a snapshot of the valley at a tick with the households of
// each (region, role) in the order alpha grower, alpha baker, beta grower, beta
// baker. The stocks are the caller's to fill.
func valleySnap(tick int64, hh [4]int64) economysim.Snapshot {
	s := economysim.Snapshot{
		Tick: tick, ConfigHash: valleyHash, Names: valleyNames(),
		Regions: make([]economysim.RegionRow, 2),
		Markets: make([]economysim.MarketRow, 8),
	}
	for r := range s.Regions {
		s.Regions[r].Region = economysim.RegionID(r)
		for c := 0; c < 4; c++ {
			s.Markets[r*4+c].State = economysim.MarketState{Region: economysim.RegionID(r), Commodity: economysim.CommodityID(c)}
		}
		for role := 0; role < 2; role++ {
			s.Roles = append(s.Roles, economysim.RoleRow{Region: economysim.RegionID(r), Role: economysim.RoleID(role), Households: hh[r*2+role], Agents: hh[r*2+role]})
		}
	}
	return s
}

// openValley opens a tally at a valley snapshot of the tick and fails the test
// on an error.
func openValley(t *testing.T, tl *indicator.Tally, tick int64, hh [4]int64) {
	t.Helper()
	if err := tl.Open(valleySnap(tick, hh)); err != nil {
		t.Fatalf("Open(valley at %d) error = %v, want nil", tick, err)
	}
}

// step feeds the tally the TickStarted and TickEnded of one tick around the
// events given (the shape of a step: events between the two).
func step(tl *indicator.Tally, tick int64, events ...economysim.Event) {
	tl.Observe(economysim.TickStarted{Tick: tick})
	for _, e := range events {
		tl.Observe(e)
	}
	tl.Observe(economysim.TickEnded{Tick: tick})
}

// flow is a BoundaryFlow event of a tick.
func flow(tick int64, r economysim.RegionID, reason economysim.Reason, c economysim.CommodityID, total int64) economysim.BoundaryFlow {
	kind := economysim.KindFaucet
	if reason == economysim.ReasonConsumption || reason == economysim.ReasonSpoilage || reason == economysim.ReasonDisposal || reason == economysim.ReasonWantConsumption {
		kind = economysim.KindSink
	}
	return economysim.BoundaryFlow{Tick: tick, Region: r, Reason: reason, Kind: kind, Commodity: c, Total: total}
}
