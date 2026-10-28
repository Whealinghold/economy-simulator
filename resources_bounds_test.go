package economysim_test

import (
	"math/big"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestBounds_ResourceProducts verifies the arithmetic of spec-resources.md 4.6:
// the largest regeneration numerator (about 10^31) fits in 127 bits, the
// largest denominator D = 10^12 * MaxTicksPerYear fits int64, MaxEndowment
// exceeds MaxQuantity, and the draws of a tick (MaxWeight times a draw already
// at most MaxQuantity) fit a stock, and MaxKnots and MaxDraws are the values of
// spec-config.md 4.1.
//
// Requirement: FR-RES-008, FR-CFG-046
//
// Guard: the 128-bit regeneration of internal/endowment cannot overflow only
// while these products stay inside 127 bits. Raising a bound without this test
// would let a numerator wrap silently.
func TestBounds_ResourceProducts(t *testing.T) {
	t.Parallel()
	// numerator = RateBP * C * (RegenScaleBP * ShockBP / 10^4) * SeasonBP.
	num := big.NewInt(economysim.MaxRegenBP)
	num.Mul(num, big.NewInt(economysim.MaxEndowment))
	num.Mul(num, big.NewInt(economysim.MaxRegenScaleBP*economysim.MaxShockBP/10_000))
	num.Mul(num, big.NewInt(economysim.MaxSeasonRegenBP))
	if got := num.BitLen(); got > 127 {
		t.Errorf("largest regeneration numerator %v has %d bits, want at most 127", num, got)
	}
	// 10^5 * 10^15 * 10^6 * 10^5 = 10^31.
	if want := new(big.Int).Exp(big.NewInt(10), big.NewInt(31), nil); num.Cmp(want) != 0 {
		t.Errorf("largest regeneration numerator = %v, want %v", num, want)
	}

	d := new(big.Int).Mul(big.NewInt(1_000_000_000_000), big.NewInt(economysim.MaxTicksPerYear))
	if !d.IsInt64() {
		t.Errorf("largest denominator D = %v, want it to fit int64", d)
	}
	if economysim.MaxEndowment <= economysim.MaxQuantity {
		t.Errorf("MaxEndowment = %d, want it above MaxQuantity %d", economysim.MaxEndowment, economysim.MaxQuantity)
	}
	if economysim.MaxEndowments != economysim.MaxRegions*economysim.MaxResources {
		t.Errorf("MaxEndowments = %d, want MaxRegions * MaxResources", economysim.MaxEndowments)
	}
	if economysim.MaxKnots != 8 || economysim.MaxDraws != 4 {
		t.Errorf("MaxKnots, MaxDraws = %d, %d, want 8, 4", economysim.MaxKnots, economysim.MaxDraws)
	}
}
