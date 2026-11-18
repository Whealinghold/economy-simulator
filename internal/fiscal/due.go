package fiscal

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/moneyflow"
)

// Shape is the form of a tax or of a payout: a fixed amount or a rate on a
// balance (spec-fiscal.md 5.1.2).
type Shape uint8

// The two shapes of a tax.
const (
	// Fixed is an amount per household (or per account) per period.
	Fixed Shape = iota
	// Rate is a rate in basis points per period of the balance above the
	// exemption.
	Rate
)

// Rule is the part of a tax that does not depend on the payer: its shape, its
// fields, the flow's scale and the ticks of its period (spec-fiscal.md 5.2).
type Rule struct {
	Shape    Shape
	Amount   int64 // minor units per household per period (Fixed)
	RateBP   int64 // basis points per period of the balance above the exemption (Rate)
	Exempt   int64 // minor units per household never collected
	Scale    int64 // the flow's scale in basis points
	Period   int64 // ticks of the period: 1 for a tick, TicksPerYear for a year
	MaxMoney int64 // the bound a due is cut to (spec-fiscal.md 11)
}

// Result is what a rule makes of one payer in one tick.
type Result struct {
	Due       int64 // computed from the payer's own state alone (FR-FIS-020)
	Collected int64 // what the payer could pay: min(Due, what is above the exemption)
	Carry     int64 // the carry after the tick; it advanced by Due, not by Collected
}

// Short is the part of the due the payer could not pay, which is forgiven and
// not owed (spec-fiscal.md 5.4).
func (r Result) Short() int64 { return r.Due - r.Collected }

// StartCarry returns the carry a payer starts with: a fixed tax starts at
// (id mod period) * 10^4 so that a small rate is first paid at a different tick
// by each payer, a rate tax at 0 (spec-fiscal.md 5.2). id is the account ID.
func (r Rule) StartCarry(id int64) int64 {
	if r.Shape == Rate {
		return 0
	}
	return moneyflow.StartCarry(id, r.Period)
}

// Due returns the due and the collection of one payer of weight w holding bal
// with the given carry (spec-fiscal.md 5.2):
//
//	Fixed: due = floor((w*Amount*Scale + carry) / (Period*10^4)),
//	       collected = min(due, max(0, bal - w*Exempt))
//	Rate:  base = max(0, bal - w*Exempt),
//	       due = floor((base*RateBP*Scale + carry) / (Period*10^8)),
//	       collected = min(due, base)
//
// The remainder is the new carry. ok is false for a negative argument, a period
// below 1, a carry outside [0, Period*10^4) (Fixed) or [0, Period*10^8) (Rate),
// an unknown shape, or a product that leaves int64.
func (r Rule) Due(w, bal, carry int64) (res Result, ok bool) {
	if w < 0 || bal < 0 || !r.carryInRange(carry) {
		return Result{}, false
	}
	held, hok := checked.Mul(w, r.Exempt)
	if !hok || r.Exempt < 0 {
		return Result{}, false
	}
	avail := max(0, bal-held)
	var due, next int64
	if r.Shape == Rate {
		due, next, ok = moneyflow.RateLevy(avail, r.RateBP, r.Scale, carry, r.Period, r.MaxMoney)
	} else {
		due, next, ok = moneyflow.Spread(w, r.Amount, r.Scale, carry, r.Period, r.MaxMoney)
	}
	if !ok {
		return Result{}, false
	}
	return Result{Due: due, Collected: min(due, avail), Carry: next}, true
}

// carryInRange reports whether the shape is known and the carry is in
// [0, Period*10^4) (Fixed) or [0, Period*10^8) (Rate).
func (r Rule) carryInRange(carry int64) bool {
	den := int64(10_000)
	switch r.Shape {
	case Fixed:
	case Rate:
		den *= 10_000
	default:
		return false
	}
	lim, ok := checked.Mul(r.Period, den)
	return ok && carry >= 0 && carry < lim
}

// Headroom returns how much a treasury holding bal can still take before it
// passes maxMoney, never negative (spec-fiscal.md 5.3.2).
// A negative bal or maxMoney counts as 0, so the subtraction cannot leave int64.
func Headroom(bal, maxMoney int64) int64 { return max(0, max(0, maxMoney)-max(0, bal)) }
