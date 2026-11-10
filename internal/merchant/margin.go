package merchant

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Cautious returns the cautious revenue per unit of a rumour (spec-merchants.md
// 5.2.1): the rumoured price when the window mean is 0 (none), else the lower of
// the price and the mean. The lower reading does not extrapolate a spike.
func Cautious(price, mean int64) int64 {
	if mean == 0 {
		return price
	}
	return min(price, mean)
}

// Net returns the revenue per unit after the share a consignee keeps and the
// expected loss of the route (spec-merchants.md 5.2.1): rhat * (10 000 - cutBP) /
// 10 000 truncated, then that * (10 000 - lossBP) / 10 000 truncated. cutBP is 0
// for a self-carried mode and lossBP 0 with the risk module off. ok is false when
// a share is outside 0 to 10 000.
func Net(rhat, cutBP, lossBP int64) (rnet int64, ok bool) {
	if cutBP < 0 || cutBP > wholeBP || lossBP < 0 || lossBP > wholeBP {
		return 0, false
	}
	r, ok := checked.MulDiv(rhat, wholeBP-cutBP, wholeBP)
	if !ok {
		return 0, false
	}
	return checked.MulDiv(r, wholeBP-lossBP, wholeBP)
}

// LifeLeft returns the ticks of life left of a cargo of shelf life shelf whose
// oldest lot is oldestAge ticks old (spec-merchants.md 5.3.1): shelf - 1 -
// max(1, oldestAge), because goods bought in a tick settle in phase 4 and cannot
// be sent before the next, so the youngest lot a merchant can ship is one tick
// old. An oldestAge of -1 means no lot is held. It is meaningful only for a
// shelf life above 0.
func LifeLeft(shelf, oldestAge int64) int64 {
	return shelf - 1 - max(1, oldestAge)
}

// Perishable reports whether a trip of k ticks is set aside for the cargo: it has
// a shelf life above 0 and k is above the life left (spec-merchants.md 5.3.1). A
// commodity with a shelf life of 0 never spoils.
func Perishable(shelf, oldestAge, k int64) bool {
	return shelf > 0 && k > LifeLeft(shelf, oldestAge)
}

// Gross returns the gross margin G = q * (rnet - pacq) of a lot (spec-merchants.md
// 5.5.1). ok is false on overflow. G is not required to be positive here: the
// caller has decided the case "no margin" before it asks.
func Gross(q, rnet, pacq int64) (g int64, ok bool) {
	d, ok := checked.Sub(rnet, pacq)
	if !ok {
		return 0, false
	}
	return checked.Mul(q, d)
}

// Pass reports whether a candidate passes the test of the fee share
// (spec-merchants.md 5.5.1): c * 10 000 <= phi * g, both sides held in 128 bits.
// A negative cost or a gross margin that is not positive does not pass.
func Pass(c, phi, g int64) bool {
	if c < 0 || phi < 0 || g <= 0 {
		return false
	}
	lhi, llo := bits.Mul64(uint64(c), wholeBP)
	rhi, rlo := bits.Mul64(uint64(phi), uint64(g))
	return lhi < rhi || (lhi == rhi && llo <= rlo)
}

// Result is what the test of the fee share leaves of a candidate: the gross
// margin, the score (the expected net gain), the margin needed per unit and the
// two prices that follow from it.
type Result struct {
	// G is the gross margin q * (rnet - pacq) and Score is G - c.
	G, Score int64
	// Needed is ceil(c * 10 000 / (phi * q)), the margin per unit the lot must
	// promise to pass.
	Needed int64
	// BreakEven is rnet - Needed, the dearest price at which the lot still passes
	// and the ceiling of the bid for the cargo; below 1 it means no bid.
	BreakEven int64
	// AskLimit is pacq + Needed, the cheapest delivered price at which the lot
	// passes and the limit of the offer.
	AskLimit int64
}

// Margin returns the gross margin, the score, the margin needed and the two prices
// of a lot of q units that costs c to carry, for a merchant of fee share phi,
// where rnet is the revenue per unit and pacq the price it buys at
// (spec-merchants.md 5.5.2, 5.5.3). The costs round up and the revenues down, so
// every rounding is against the merchant. ok is false when q or phi is below 1 or
// any product overflows.
func Margin(c, phi, q, rnet, pacq int64) (r Result, ok bool) {
	if q < 1 || phi < 1 || c < 0 {
		return Result{}, false
	}
	g, ok := Gross(q, rnet, pacq)
	if !ok {
		return Result{}, false
	}
	score, ok := checked.Sub(g, c)
	if !ok {
		return Result{}, false
	}
	pq, ok := checked.Mul(phi, q)
	if !ok {
		return Result{}, false
	}
	needed, ok := checked.MulDivCeil(c, wholeBP, pq)
	if !ok {
		return Result{}, false
	}
	be, ok1 := checked.Sub(rnet, needed)
	ask, ok2 := checked.Add(pacq, needed)
	if !ok1 || !ok2 {
		return Result{}, false
	}
	return Result{G: g, Score: score, Needed: needed, BreakEven: be, AskLimit: ask}, true
}

// HaulCeiling returns the ceiling of the bid for a haul unit (spec-merchants.md
// 5.6, FR-CAR-004): the expected margin per haul unit (g - n * toll) / (n * s)
// truncated, times phi / 10 000 truncated. n is the carrying units of the lot, s
// the haul units per carrying unit and toll the fiscal toll of a carrying unit
// (0 with the module off). ok is false when n or s is below 1 or a product
// overflows. A ceiling below 1 means no bid.
func HaulCeiling(g, toll, n, s, phi int64) (ceiling int64, ok bool) {
	if n < 1 || s < 1 || phi < 0 {
		return 0, false
	}
	tn, ok := checked.Mul(n, toll)
	if !ok {
		return 0, false
	}
	net, ok := checked.Sub(g, tn)
	if !ok {
		return 0, false
	}
	units, ok := checked.Mul(n, s)
	if !ok {
		return 0, false
	}
	per, ok := checked.MulDiv(net, 1, units)
	if !ok {
		return 0, false
	}
	return checked.MulDiv(per, phi, wholeBP)
}

// Cand is a passing candidate: its score and its place in the evaluation order,
// the destination, the cargo and the mode as the caller numbers them
// (spec-merchants.md 5.1.2).
type Cand struct {
	// Score is G - C.
	Score int64
	// Dest, Cargo and Mode are the references of the candidate in ascending order.
	Dest, Cargo, Mode int64
}

// Better reports whether a is preferred to b: the greater score, ties to the
// lower destination, then cargo, then mode (spec-merchants.md 5.1.2).
func Better(a, b Cand) bool {
	switch {
	case a.Score != b.Score:
		return a.Score > b.Score
	case a.Dest != b.Dest:
		return a.Dest < b.Dest
	case a.Cargo != b.Cargo:
		return a.Cargo < b.Cargo
	}
	return a.Mode < b.Mode
}
