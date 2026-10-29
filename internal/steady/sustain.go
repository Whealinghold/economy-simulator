package steady

import "math/big"

// The shapes of an endowment as the sustainable draw reads them
// (spec-resources.md 4.2, 11.1). The package knows no resource: the caller maps
// its own shape names to these.
const (
	// Constant is a flow of Rate units a year; with Rate 0 a finite deposit.
	Constant = iota
	// Refill closes the gap between the stock and the capacity at Rate basis
	// points a year.
	Refill
	// Logistic grows in proportion to the stock and to the room left, at Rate
	// basis points a year.
	Logistic
)

// wholeBP is 10 000 basis points, one.
const wholeBP = 10000

// Endow is the data of one resource row of a programme: the sustainable draw of
// one endowment in one region (spec-resources.md 11.1). Resource is the caller's
// index of the resource, which the Draws of an activity name; Region indexes
// Model.Regions. Capacity is the effective capacity at tick 0 and Stock the
// starting stock; ScaleBP the regeneration scale in basis points (10 000 when
// not scaled); Seasons the season factor of the resource over the year as runs of
// ticks (nil for 10 000 on every tick); Horizon the years a finite stock is
// spread over.
type Endow struct {
	Resource, Region int
	Shape            int
	Capacity, Stock  int64
	Rate             int64
	PlanShareBP      int64
	ScaleBP          int64
	Seasons          []Run
	Horizon          int64
}

// seasonMean returns the mean over the year of the season factor over 10 000,
// sum(length * factor) / (year * 10 000), exact.
func seasonMean(runs []Run, year int64) *big.Rat {
	if len(runs) == 0 {
		return big.NewRat(1, 1)
	}
	sum := new(big.Int)
	for _, r := range runs {
		sum.Add(sum, new(big.Int).Mul(big.NewInt(r.Length), big.NewInt(r.FactorBP)))
	}
	return new(big.Rat).SetFrac(sum, new(big.Int).Mul(big.NewInt(year), big.NewInt(wholeBP)))
}

// regenAnnual returns the units a year the endowment regrows at the plan stock
// S_p = PlanShareBP * C / 10 000, with sbar the mean season factor (over 10 000)
// of the stretch that counts, in exact rational arithmetic:
//
//	refill    (Rate / 10^4) * (C - S_p) * (m / 10^4) * sbar
//	logistic  (Rate / 10^4) * (S_p * (C - S_p) / C) * (m / 10^4) * sbar
//	constant  Rate * (m / 10^4) * sbar
//
// The logistic growth is the exact one, not the rounded-up weight of the tick.
func (e Endow) regenAnnual(sbar *big.Rat) *big.Rat {
	c := big.NewRat(e.Capacity, 1)
	scale := big.NewRat(e.ScaleBP, wholeBP)
	if e.Shape == Constant {
		return scale.Mul(scale, new(big.Rat).Mul(big.NewRat(e.Rate, 1), sbar))
	}
	sp := new(big.Rat).Mul(c, big.NewRat(e.PlanShareBP, wholeBP))
	room := new(big.Rat).Sub(c, sp)
	base := room // refill: the gap
	if e.Shape == Logistic {
		if e.Capacity == 0 {
			return new(big.Rat)
		}
		base = new(big.Rat).Quo(new(big.Rat).Mul(sp, room), c)
	}
	out := new(big.Rat).Mul(base, big.NewRat(e.Rate, wholeBP))
	out.Mul(out, scale)
	return out.Mul(out, sbar)
}

// Phi returns the sustainable draw of the endowment in units a tick, exact: the
// regeneration at the plan stock over a year divided by the ticks of the year,
// and for a constant stock also Stock / Horizon, the reserve spread over the
// years the pack must last (spec-resources.md 11.1). A renewing stock is
// sustained by its flow; a finite one is drawn down.
func (e Endow) Phi(year int64) *big.Rat {
	a := e.regenAnnual(seasonMean(e.Seasons, year))
	if e.Shape == Constant && e.Horizon > 0 {
		a.Add(a, big.NewRat(e.Stock, e.Horizon))
	}
	return a.Quo(a, big.NewRat(year, 1))
}

// Window is a stretch [Start, End) of ticks of the year in which an activity can
// start, with the number of starts of the year that fall in it.
type Window struct {
	Start, End, Starts int64
}

// seasonMass returns the sum of the season factor (over 10 000 per tick) of the
// ticks in [from, to), the runs repeating nothing: a tick past the last run has
// factor 0 here, since the caller builds one year.
func seasonMass(runs []Run, from, to int64) int64 {
	if len(runs) == 0 {
		return (to - from) * wholeBP
	}
	var mass, at int64
	for _, r := range runs {
		lo, hi := max(at, from), min(at+r.Length, to)
		if hi > lo {
			mass += (hi - lo) * r.FactorBP
		}
		at += r.Length
	}
	return mass
}

// PeakExceeds reports whether the draws of an activity started in window w are
// more than the stock holds at its opening (spec-resources.md 11.3): the
// activity draws perTick units a tick on average, makes starts starts a year, and
// the share of them that fall in w is w.Starts / starts; the stock at the opening
// is the plan stock S_p plus what the window regrows, at most the capacity. It
// assumes starts uniform over the window, so it is an approximation, and it is
// false for an activity with no start.
func (e Endow) PeakExceeds(year int64, w Window, starts int64, perTick Frac) bool {
	if starts < 1 {
		return false
	}
	draw := new(big.Rat).Mul(frat(perTick), big.NewRat(year*w.Starts, starts))
	sbar := big.NewRat(seasonMass(e.Seasons, w.Start, w.End), year*wholeBP)
	open := new(big.Rat).Mul(big.NewRat(e.Capacity, 1), big.NewRat(e.PlanShareBP, wholeBP))
	open.Add(open, e.regenAnnual(sbar))
	if c := big.NewRat(e.Capacity, 1); open.Cmp(c) > 0 {
		open = c
	}
	return draw.Cmp(open) > 0
}

// negative reports a negative quantity in the row, a negative run length or
// season factor, or a plan share above one.
func (e Endow) negative() bool {
	for _, r := range e.Seasons {
		if r.Length < 0 || r.FactorBP < 0 {
			return true
		}
	}
	return e.Capacity < 0 || e.Stock < 0 || e.Rate < 0 || e.ScaleBP < 0 || e.PlanShareBP < 0 || e.PlanShareBP > wholeBP
}

// badShape reports a shape that is none of the three, or a constant stock with
// no horizon to spread it over.
func (e Endow) badShape() bool {
	return e.Shape < Constant || e.Shape > Logistic || e.Shape == Constant && e.Horizon < 1
}
