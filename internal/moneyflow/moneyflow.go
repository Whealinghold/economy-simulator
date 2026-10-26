// Package moneyflow is the arithmetic of the money supply module's flows
// (spec-money-supply.md 4 to 6): the due of one recipient of a grant, a levy
// or a top-up, the carry that keeps a small rate exact, and the cut of a due
// by the room of the account and the limit of the flow.
//
// Every function works on int64 primitives and imports the standard library
// and internal/checked only. It does not know what a household, a role or a
// region is: the root converts at the call (architecture.md 2.9). Division
// truncates toward zero on non-negative values (architecture.md 3.5), the
// remainder is carried and never lost, and a quotient above the caller's
// maxMoney is returned as maxMoney (spec-money-supply.md 6.4.1).
//
// ok is false only for an input outside the bounds the caller validated: a
// negative argument, a period below 1, or a product that leaves int64. The
// other results are then zero and must not be used.
package moneyflow

import "github.com/Whealinghold/economy-sim/internal/checked"

// The carry of a rate grant and of a fixed levy is in units of 1/(P*10^4) of
// a minor unit, that of a rate levy in units of 1/(P*10^8): the basis-point
// denominator once and twice (spec-money-supply.md 6.1, 6.3). They are the
// rule constants of the arithmetic, not settings.
const (
	bp  = 10_000
	bp2 = bp * bp
)

// Cut says which bound, if any, cut a due, the room bound acting first.
type Cut uint8

// The bounds that can cut a due (spec-money-supply.md 5.4, plan D328).
const (
	// CutNone: the whole due is applied.
	CutNone Cut = iota
	// CutRoom: the account's room cut it: a levy's available balance, a
	// grant's headroom below MaxMoney.
	CutRoom
	// CutLimit: the flow's remaining limit cut it, the room not having.
	CutLimit
)

// StartCarry returns the carry a recipient starts with, (id mod period) *
// 10^4: it spreads the first payments of a small rate over the ticks of the
// period with no random number (spec-money-supply.md 6.2). id is the account
// ID. It is a function of the ID alone, so starting a carry lazily at the
// first visit equals starting it at creation. A rate levy starts at 0 and
// does not call it. It returns 0 for a period below 1.
func StartCarry(id, period int64) int64 {
	if id < 0 || period < 1 {
		return 0
	}
	return id % period * bp
}

// Spread returns the due of a rate grant or a fixed levy for a recipient of
// weight w: floor((w*amount*scale + carry) / (period*10^4)), and the carry
// that remains, in [0, period*10^4). scale is the flow scale in basis points.
// The due is at most maxMoney. Over n ticks with unchanged w and scale the
// total due is floor((n*w*amount*scale + carry0) / (period*10^4)), exactly.
func Spread(w, amount, scale, carry, period, maxMoney int64) (due, newCarry int64, ok bool) {
	if w < 0 || amount < 0 || scale < 0 || carry < 0 || period < 1 || maxMoney < 0 {
		return 0, 0, false
	}
	a, aok := checked.Mul(w, amount)
	den, dok := checked.Mul(period, bp)
	if !aok || !dok {
		return 0, 0, false
	}
	// The arguments are non-negative and den is positive, so ok is true.
	q, r, ok := checked.MulAddDivSat(a, scale, carry, den)
	return min(q, maxMoney), r, ok
}

// RateLevy returns the due of a rate levy on the balance base above the
// exemption: floor((base*rateBP*scale + carry) / (period*10^8)), and the
// carry that remains, in [0, period*10^8). The due is at most maxMoney and is
// not cut at base: the caller cuts it with Clamp against the available
// balance, which counts the shortfall. The carry advances by the due, not by
// what is applied (spec-money-supply.md 6.1.2).
func RateLevy(base, rateBP, scale, carry, period, maxMoney int64) (due, newCarry int64, ok bool) {
	if base < 0 || rateBP < 0 || scale < 0 || carry < 0 || period < 1 || maxMoney < 0 {
		return 0, 0, false
	}
	rate, rok := checked.Mul(rateBP, scale)
	den, dok := checked.Mul(period, bp2)
	if !rok || !dok {
		return 0, 0, false
	}
	q, r, ok := checked.MulAddDivSat(base, rate, carry, den)
	return min(q, maxMoney), r, ok
}

// TopUp returns the due of a top-up to a level for a recipient of weight w
// holding balance: max(0, floor(w*level*scale/10^4) - balance), at most
// maxMoney. It has no period and no carry (spec-money-supply.md 4.2).
func TopUp(w, level, scale, balance, maxMoney int64) (due int64, ok bool) {
	if w < 0 || level < 0 || scale < 0 || balance < 0 || maxMoney < 0 {
		return 0, false
	}
	a, aok := checked.Mul(w, level)
	if !aok {
		return 0, false
	}
	target, _, ok := checked.MulAddDivSat(a, scale, 0, bp)
	return min(max(target-balance, 0), maxMoney), ok
}

// Clamp cuts a due to the room of the account and then to the remaining limit
// of the flow, and says which bound cut it first: a due cut by the room is
// CutRoom even if the limit would have cut the rest, so a recipient is counted
// once (plan D328). A negative room or remaining is 0.
func Clamp(due, room, remaining int64) (applied int64, cut Cut) {
	room, remaining = max(room, 0), max(remaining, 0)
	applied = due
	if applied > room {
		applied, cut = room, CutRoom
	}
	if applied > remaining {
		applied = remaining
		if cut == CutNone {
			cut = CutLimit
		}
	}
	return applied, cut
}
