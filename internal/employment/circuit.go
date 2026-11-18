package employment

import "github.com/Whealinghold/economy-sim/internal/checked"

// Circuit holds the parameters of the closed form of spec-employment.md 11.2:
// E employers of Upkeep units a tick each, Units per execution (a), a worker
// share of the margin PhiBP (phi, in basis points), and the saving rule's cap
// SpendMaxShareBP and SpendHalfTicks (spec-agents.md 7.4.4). Places is the
// capacity E * MaxHires.
type Circuit struct {
	Employers       int64
	Upkeep          int64
	Units           int64
	PhiBP           int64
	SpendMaxShareBP int64
	SpendHalfTicks  int64
	Places          int64
}

// HiredMilli returns the closed form of the number of hired households, in
// thousandths of a household, rounded down and capped at Places:
//
//	h* = min(Places, E * u_e * (10 000 + SpendMaxShareBP * SpendHalfTicks) /
//	                 ((10 000 - PhiBP) * a))
//
// which is E u_e (1 + S H / 10 000) / ((1 - phi) a) of 11.2 with phi in basis
// points. With SpendMaxShareBP 0 it is the exact line of the first column; the
// others are the bound the saving rule's cap gives (an asymptote). It is a test
// oracle for the scenario S97 and is not called by the simulation. It reports
// false when an input is out of range (PhiBP is 0 to 9 999) or the product does
// not fit.
func HiredMilli(c Circuit) (int64, bool) {
	if c.Employers < 0 || c.Upkeep < 0 || c.Units < 1 || c.PhiBP < 0 || c.PhiBP >= WholeBP ||
		c.SpendMaxShareBP < 0 || c.SpendHalfTicks < 0 || c.Places < 0 {
		return 0, false
	}
	spendCap, ok := checked.Mul(c.SpendMaxShareBP, c.SpendHalfTicks)
	if !ok {
		return 0, false
	}
	gain, ok := checked.Add(WholeBP, spendCap)
	if !ok {
		return 0, false
	}
	supply, ok := checked.Mul(c.Employers, c.Upkeep)
	if !ok {
		return 0, false
	}
	den, ok := checked.Mul(WholeBP-c.PhiBP, c.Units)
	if !ok {
		return 0, false
	}
	// supply * gain * 1 000 / den, the product held in 128 bits.
	scaled, ok := checked.Mul(supply, 1000)
	if !ok {
		return 0, false
	}
	milli, ok := checked.MulDivFloor(scaled, gain, den)
	if !ok {
		return 0, false
	}
	places, ok := checked.Mul(c.Places, 1000)
	if !ok {
		return 0, false
	}
	return min(milli, places), true
}
