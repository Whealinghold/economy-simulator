package strategy_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestMerchant_HaulCeilingNetOfToll verifies spec-merchants.md 5.6 and spec-fiscal.md
// 7.5: with the fiscal module on, the merchant's ceiling for a haul unit is the expected
// margin per haul unit net of the toll of the lot, (G - n * Toll) / (n * s) truncated,
// times its fee share of 6 383 over 10 000 truncated, and a trip whose toll takes the
// whole margin buys no haul unit at all.
//
// Arithmetic, in the valley of example A: G = 13 350 for the lot of n = 1 carrying unit
// at s = 6 haul units a unit (G / 6 = 2 225 and the ceiling 1 420 with no toll, above
// the 685 the default asks, so the bid stays at 685). A toll of 12 000 a unit leaves
// (13 350 - 12 000) / 6 = 225 and 225 * 6 383 / 10 000 = 143; a toll of 13 000 leaves
// 350 / 6 = 58 and 58 * 6 383 / 10 000 = 37; a toll of 13 350 leaves 0, below 1, and the
// bid is dropped. The bid for the cargo is the same throughout.
//
// Requirement: FR-FIS-034, FR-CAR-004
func TestMerchant_HaulCeilingNetOfToll(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		toll int64
		want int64 // the limit of the haul bid, 0 for none
	}{
		{"no toll", 0, 685},
		{"a toll that leaves a ceiling above the default's", 5_000, 685},
		{"a toll of 12 000", 12_000, 143},
		{"a toll of 13 000", 13_000, 37},
		{"a toll that takes the whole margin", 13_350, 0},
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
			if haulLimit != tt.want || cargoBids != 1 {
				t.Errorf("haul bid limit = %d and %d cargo bids with a toll of %d, want %d and 1", haulLimit, cargoBids, tt.toll, tt.want)
			}
		})
	}
}
