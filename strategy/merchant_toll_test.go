package strategy_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestMerchant_TollIsInTheCostOfCarriage verifies spec-merchants.md 5.4 and 6.2.1 and
// spec-fiscal.md 7.5: with the fiscal module on, the cost of carriage of a trip is the
// hired cost plus n * Toll, so the test of the fee share sees the toll, and the trip is
// not planned when the toll makes the carriage dearer than the share of the gross
// margin. A toll that leaves the test passing leaves the bid as the default asks it.
//
// Arithmetic, in the valley of example A: the lot of 150 units is n = 1 carrying unit at
// s = 6 haul units a unit, G = 13 350, the haul commodity is at 700 and the agent's fee
// share is 6 385 over 10 000. The hired cost is 6 * 700 = 4 200 plus the toll, and the
// test passes while it is at most floor(13 350 * 6 385 / 10 000) = 8 523, that is for a
// toll of up to 4 323. A toll of 4 000 passes, and its ceiling (13 350 - 4 000) / 6 =
// 1 558, times 6 385 / 10 000 = 994, is above the 685 the default asks, so the bid stays.
// A toll of 4 324 makes the cost 8 524 and nothing is planned: no cargo bid, no haul bid.
//
// Requirement: FR-FIS-034, FR-MER-011, FR-CAR-004
func TestMerchant_TollIsInTheCostOfCarriage(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		toll     int64
		want     int64 // the limit of the haul bid, 0 for none
		wantBids int64 // the bids for the cargo
	}{
		{"no toll", 0, 685, 1},
		{"a toll that leaves a ceiling above the default's", 4_000, 685, 1},
		{"the dearest toll the share carries", 4_323, 685, 1},
		{"a toll one above it", 4_324, 0, 0},
		{"a toll that takes the whole margin", 13_350, 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := valley()
			v.Routes[0].Toll = tt.toll
			p, _ := trade(t, merchantStrategy, v)
			var haulLimit, cargoBids int64
			for _, in := range p.Intents {
				switch in.Commodity {
				case haul:
					haulLimit = int64(in.Limit)
				case fuel:
					if in.Side == economysim.Bid {
						cargoBids++
					}
				}
			}
			if haulLimit != tt.want || cargoBids != tt.wantBids {
				t.Errorf("haul bid limit = %d and %d cargo bids with a toll of %d, want %d and %d", haulLimit, cargoBids, tt.toll, tt.want, tt.wantBids)
			}
		})
	}
}
