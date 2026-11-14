package tenure

import "github.com/Whealinghold/economy-sim/internal/checked"

// Line is one draw of a start on an endowment with a title: Rent per unit let,
// Draw units per execution, and the executions of the job served without rent,
// Own and Commons (spec-land.md 5.3.2). An exempt request has Rent 0.
type Line struct {
	Rent    int64
	Draw    int64
	Own     int64
	Commons int64
}

// Available is the money a start can spend on rent: the principal's balance
// less what the earlier starts of the same principal reserved (spec-land.md
// 5.4.1), and 0 when the reservation is not below the balance.
func Available(balance, reserved int64) int64 {
	if balance <= 0 || reserved < 0 || reserved >= balance {
		return 0
	}
	return balance - reserved
}

// TotalRent is the total rent of k executions over all the lines,
// the sum of Rent * Draw * max(0, k - Own - Commons) (spec-land.md 5.4.1). It
// is non-decreasing in k. ok is false when the total passes int64, which the
// start treats as unaffordable, and for a negative k or a line with a negative
// field, with 0 returned.
func TotalRent(lines []Line, k int64) (total int64, ok bool) {
	for _, l := range lines {
		line, valid := lineRent(l, k)
		if !valid {
			return 0, false
		}
		if total, valid = checked.Add(total, line); !valid {
			return 0, false
		}
	}
	return total, true
}

// lineRent is the rent of k executions of one line. ok is false for a negative
// k or field. The two entitlements are taken off k one at a time, so a sum of
// them beyond int64 cannot wrap: k and both are at least 0, and so is each
// difference.
func lineRent(l Line, k int64) (int64, bool) {
	if k < 0 || l.Rent < 0 || l.Draw < 0 || l.Own < 0 || l.Commons < 0 {
		return 0, false
	}
	billed := max(0, max(0, k-l.Own)-l.Commons)
	units, ok := checked.Mul(billed, l.Draw)
	if !ok {
		return 0, false
	}
	return checked.Mul(units, l.Rent)
}

// Charge returns the largest k at most other with TotalRent(lines, k) at most
// available, found by bisection, and the rent of that k (spec-land.md 5.4.1). A
// k of 0 while other is positive is the refusal rent_funds: nothing is taken and
// nothing is carried (5.4.2). A rent that overflows is unaffordable. ok is false
// for a negative number or a line with a negative field, with 0 returned.
func Charge(available, other int64, lines []Line) (k, rent int64, ok bool) {
	if available < 0 || other < 0 {
		return 0, 0, false
	}
	if !validLines(lines) {
		return 0, 0, false
	}
	lo, hi := int64(0), other // TotalRent(lo) is affordable (0 units); find the largest affordable k
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		if r, ok := TotalRent(lines, mid); ok && r <= available {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	rent, ok = TotalRent(lines, lo)
	if !ok { // lo was found affordable, so this cannot happen; refuse rather than return a zero rent
		return 0, 0, false
	}
	return lo, rent, true
}

// validLines reports whether no field of any line is negative.
func validLines(lines []Line) bool {
	for _, l := range lines {
		if l.Rent < 0 || l.Draw < 0 || l.Own < 0 || l.Commons < 0 {
			return false
		}
	}
	return true
}

// Credit is what a payee at balance receives of amount, bounded as a credit is
// (spec-land.md 5.3.3): it receives at most limit - balance, and the rest is
// forfeited, not charged to the payer. paid + forfeited is amount. ok is false
// for a negative number or a balance above limit.
func Credit(balance, amount, limit int64) (paid, forfeited int64, ok bool) {
	if balance < 0 || amount < 0 || limit < 0 || balance > limit {
		return 0, 0, false
	}
	paid = min(amount, limit-balance)
	return paid, amount - paid, true
}
