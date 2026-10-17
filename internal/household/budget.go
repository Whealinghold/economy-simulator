package household

import "github.com/Whealinghold/economy-sim/internal/checked"

// SpendCap returns the cap on the money spent in tier 3, the
// rule that saves when wealth is low (spec-agents.md 7.4.4):
//
//	H   = SpendHalfTicks * upkeepCost
//	cap = SpendMaxShare * (e * H / (e + H)) / 10 000
//
// with the products in 128 bits and each quotient truncated. With upkeepCost 0
// it is SpendMaxShare * e / 10 000. It is 0 at e 0 or below, never decreases
// with e and never exceeds e (FR-AGT-042). ok is false on an overflow of H or of
// e + H, which a validated configuration cannot reach.
func SpendCap(e, upkeepCost int64, p Params) (int64, bool) {
	if e <= 0 {
		return 0, true
	}
	share := min(max(p.SpendMaxShare, 0), Basis)
	if upkeepCost <= 0 {
		return checked.MulDiv(e, share, Basis)
	}
	h, ok := checked.Mul(max(p.SpendHalfTicks, 0), upkeepCost)
	if !ok {
		return 0, false
	}
	sum, ok := checked.Add(e, h)
	if !ok {
		return 0, false
	}
	// Untested not-ok branch: e*h/(e+h) is below both e and h, so it fits
	// whenever they do; the check keeps the function total.
	x, ok := checked.MulDiv(e, h, sum)
	if !ok {
		return 0, false
	}
	return checked.MulDiv(x, share, Basis)
}

// Excess returns max(0, avail - savingTarget), the money above the savings
// target (spec-agents.md 7.4.4). A negative savingTarget counts as 0.
func Excess(avail, savingTarget int64) int64 {
	savingTarget = max(savingTarget, 0)
	if avail <= savingTarget {
		return 0
	}
	return avail - savingTarget
}

// BuyShare returns the share of a tier-3 want that is bought, in basis points:
// FavourFloor + (10 000 - FavourFloor) * (10 000 - pos) / 10 000, with pos the
// position of the market price in the price window (EmptyPos if the window is
// empty), clamped to 0 to 10 000 (spec-agents.md 7.5.2, FR-AGT-043). The caller
// applies it to tier 3 only.
func BuyShare(pos int64, p Params) int64 {
	pos = min(max(pos, 0), Basis)
	floor := min(max(p.FavourFloor, 0), Basis)
	return floor + (Basis-floor)*(Basis-pos)/Basis
}

// Afford returns what one tier buys with the money left (spec-agents.md 7.5.3,
// 7.5.4): the quantity min(want, avail / limit) at limit, and if avail cannot
// buy one unit at limit, one unit at limitOut = avail, the limit lowered to the
// money. qty is 0 only with no money, no want or a limit below 1. The product
// qty * limitOut never exceeds avail. The draw of the limit is the caller's, and
// so is the combination of the tiers of one order: only the first tier to buy
// may take the lowered limit, and a later tier that cannot pay one unit at the
// order's limit is dropped.
func Afford(avail, limit, want int64) (limitOut, qty int64) {
	if avail < 1 || limit < 1 || want < 1 {
		return limit, 0
	}
	if avail < limit {
		return avail, 1
	}
	return limit, min(want, avail/limit)
}
