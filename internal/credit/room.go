package credit

import "github.com/Whealinghold/economy-sim/internal/checked"

// Room is the exposure room of a borrower, max(0, maxDebtTicks * upkeep - debt):
// the principal it may still ask for (spec-credit.md 5.3.3). The product is
// checked and saturates at maxMoney. ok is false for a negative argument.
func Room(maxDebtTicks, upkeep, debt, maxMoney int64) (room int64, ok bool) {
	if maxDebtTicks < 0 || upkeep < 0 || debt < 0 || maxMoney < 0 {
		return 0, false
	}
	limit, fits := checked.Mul(maxDebtTicks, upkeep)
	if !fits || limit > maxMoney {
		limit = maxMoney
	}
	return max(0, limit-debt), true
}
