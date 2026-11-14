package credit

import "github.com/Whealinghold/economy-sim/internal/checked"

// Pay returns what a debtor pays at the settlement pass: min(max(0, balance -
// shield), owed), with shield = shieldTicks * upkeep, a checked product that
// saturates at maxMoney (spec-credit.md 8.2, FR-CRD-031). balance is the one
// recorded at the entry of the pass. ok is false for a negative argument.
func Pay(balance, shieldTicks, upkeep, owed, maxMoney int64) (pay int64, ok bool) {
	if balance < 0 || shieldTicks < 0 || upkeep < 0 || owed < 0 || maxMoney < 0 {
		return 0, false
	}
	shield, fits := checked.Mul(shieldTicks, upkeep)
	if !fits || shield > maxMoney {
		shield = maxMoney
	}
	return min(max(0, balance-shield), owed), true
}

// Waterfall pays a removed debtor's claims from its whole balance with no shield
// (spec-credit.md 9.1.1, FR-CRD-040): pay = min(balance, the sum of owed), shared
// among the claims as Allocate does, and left = balance - pay is what goes to the
// estate. The shares are appended to dst[:0]. ok is false, and the shares nil,
// for a negative balance or owed, a tie of a different length or a sum of the owed
// that overflows.
func (s *Scratch) Waterfall(dst []int64, balance int64, owed []int64, tie []uint64) (shares []int64, left int64, ok bool) {
	total, _ := sumOwed(owed) // a sum or a pay that Allocate cannot take is refused there
	pay := min(balance, total)
	shares, ok = s.Allocate(dst, pay, owed, tie)
	if !ok {
		return nil, 0, false
	}
	return shares, balance - pay, true
}
