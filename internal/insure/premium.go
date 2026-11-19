package insure

import "github.com/Whealinghold/economy-sim/internal/checked"

// rateScale is 10^8, the denominator of the rate of a route cover in basis points,
// and premiumScale is 10^12, the denominator of its premium: the product of three
// basis-point factors is a fraction over 10^12 (spec-insurance.md 6.4.1).
const (
	rateScale    int64 = 100_000_000
	premiumScale int64 = 1_000_000_000_000
)

// Premium returns the premium of a route cover and its rate in basis points of the
// sum (spec-insurance.md 6.4.1, FR-INS-022):
//
//	premium = halfEven(sum * lossBP * coverBP * loadingBP / 10^12)
//	rate    = halfEven(lossBP * coverBP * loadingBP / 10^8)
//
// The three basis-point factors are multiplied first (they fit int64 for every
// bound of spec-insurance.md 13), then the product with the sum is divided in 128
// bits, so the rounding is one half-to-even step. ok is false for a negative input,
// an overflow or a premium above limit, the case the round counts as overflow.
func Premium(sum, lossBP, coverBP, loadingBP, limit int64) (premium, rateBP int64, ok bool) {
	if sum < 0 || lossBP < 0 || coverBP < 0 || loadingBP < 0 {
		return 0, 0, false
	}
	k, ok := checked.Mul(lossBP, coverBP)
	if !ok {
		return 0, 0, false
	}
	if k, ok = checked.Mul(k, loadingBP); !ok {
		return 0, 0, false
	}
	if premium, ok = checked.MulDivHalfEven(sum, k, premiumScale); !ok || premium > limit {
		return 0, 0, false
	}
	rateBP, ok = checked.MulDivHalfEven(k, 1, rateScale)
	return premium, rateBP, ok
}

// StorePremium returns the premium of a store cover for its one tick,
// halfEven(sum * rateBP / 10 000) (spec-insurance.md 6.4.2, FR-INS-022). ok is false
// for a negative input, an overflow or a premium above limit.
func StorePremium(sum, rateBP, limit int64) (int64, bool) {
	if sum < 0 || rateBP < 0 {
		return 0, false
	}
	p, ok := checked.MulDivHalfEven(sum, rateBP, Basis)
	if !ok || p > limit {
		return 0, false
	}
	return p, true
}

// MaxPayout returns halfEven(coverBP * sum / 10 000), the most a policy pays
// (spec-insurance.md 6.4.1, FR-INS-023). It is at most sum when coverBP is at most
// 10 000, so it fits wherever the sum does; ok is false for a negative input.
func MaxPayout(coverBP, sum int64) (int64, bool) {
	if coverBP < 0 || sum < 0 {
		return 0, false
	}
	return checked.MulDivHalfEven(coverBP, sum, Basis)
}
