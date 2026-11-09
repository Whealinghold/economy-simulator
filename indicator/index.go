package indicator

import (
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// CellSel selects (region, commodity) cells; a nil slice means all. A class is
// turned into its commodities by Classes.Of before the call (spec-indicators.md
// 5.3.1).
type CellSel struct {
	Regions     []economysim.RegionID
	Commodities []economysim.CommodityID
}

// has reports whether the cell is selected.
func (s CellSel) has(r economysim.RegionID, c economysim.CommodityID) bool {
	return (s.Regions == nil || slices.Contains(s.Regions, r)) && (s.Commodities == nil || slices.Contains(s.Commodities, c))
}

// Index is an index, a ratio or a growth in basis points. OK is false when it is
// undefined (a zero denominator) and then Value is 0; Clamped is true when the
// value was held at MaxIndexBP (spec-indicators.md 5.3.7).
type Index struct {
	Value   BP
	OK      bool
	Clamped bool
}

// ratioWide returns rdiv(10 000 * num, den) as an Index: a zero denominator is
// undefined, a value at or above MaxIndexBP is clamped and flagged, and a
// numerator past the guard 2^127 / 10 000 is ErrOverflow (5.2.1, 5.3.7).
func ratioWide(preset, field string, num, den measure.Int128) (Index, error) {
	if den.Sign() == 0 {
		return Index{}, nil
	}
	scaled, ok := measure.MulBP(num)
	if !ok {
		return Index{}, wrap(preset, field, ErrOverflow)
	}
	q, _ := measure.RoundDiv(scaled, den)
	v, clamped := measure.ClampBP(q)
	return Index{Value: BP(v), OK: true, Clamped: clamped}, nil
}

// Quantities returns the units of the window by (region, commodity) for a
// choice of weights, at region * Commodities + commodity (spec-indicators.md
// 5.3.2): the units traded, consumed or produced, the last two from the
// corrected flows.
func (w Window) Quantities(k Weights) ([]int64, error) {
	if k == WeightsTraded {
		out := make([]int64, len(w.Markets))
		for i, m := range w.Markets {
			out[i] = m.SumV
		}
		return out, nil
	}
	f, err := flowsOf(w)
	if err != nil {
		return nil, err
	}
	switch k {
	case WeightsConsumed:
		return consumedUnits(f)
	case WeightsProduced:
		return f.units(economysim.ReasonRecipeOutput)
	}
	return nil, wrap("Window.Quantities", k.String(), ErrRange)
}

// IndexResult is the price index of a set of cells against a base window, with
// the weights it used (spec-indicators.md 9.1, row 3). Stale is the share, in
// basis points of the base value of the set, of the cells whose market had no
// volume in the window, so that an index computed mostly from carried-forward
// prices says so.
type IndexResult struct {
	ConfigHash                 [32]byte
	From, To                   int64
	Weights                    Weights
	Laspeyres, Paasche, Fisher Index
	Stale                      BP
	StaleOK                    bool
	Warnings                   []Warning
}

// PriceIndex returns the Laspeyres, Paasche and Fisher indices of the cells in
// sel at the prices cur of the window w against the base, weighted by the
// quantities k names (spec-indicators.md 5.3.3, 5.3.4): Laspeyres with the
// base's quantities, Paasche with the window's. A class index and the all-goods
// index are separate calls (a class is a sel). Against its own base an index is
// 10 000. It returns ErrConfigMismatch for a base or table of another world and
// ErrRange for one of another shape.
func PriceIndex(w Window, base Base, k Weights, cur Prices, sel CellSel) (IndexResult, error) {
	for _, p := range []Prices{base.Prices, cur} {
		if err := w.checkTable("PriceIndex", p); err != nil {
			return IndexResult{}, err
		}
	}
	q0 := base.Quantities(k)
	if q0 == nil || len(q0) != len(w.Markets) {
		return IndexResult{}, wrap("PriceIndex", "weights", ErrRange)
	}
	q1, err := w.Quantities(k)
	if err != nil {
		return IndexResult{}, err
	}
	var s struct{ p1q0, p0q0, p1q1, p0q1, stale acc }
	for i := range q0 {
		r, c := economysim.RegionID(i/w.Commodities), economysim.CommodityID(i%w.Commodities)
		if !sel.has(r, c) {
			continue
		}
		p0, p1 := base.Prices.At(r, c), cur.At(r, c)
		s.p1q0.add(q0[i], p1)
		s.p0q0.add(q0[i], p0)
		s.p1q1.add(q1[i], p1)
		s.p0q1.add(q1[i], p0)
		if w.Markets[i].SumV == 0 {
			s.stale.add(q0[i], p0)
		}
	}
	res := IndexResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Weights: k}
	if err := res.fill(s.p1q0, s.p0q0, s.p1q1, s.p0q1, s.stale); err != nil {
		return IndexResult{}, err
	}
	return res, nil
}

// fill sets the three indices, the stale share and the warnings from the sums.
func (res *IndexResult) fill(p1q0, p0q0, p1q1, p0q1, stale acc) error {
	var err error
	if res.Laspeyres, err = ratioWide("PriceIndex", "laspeyres", p1q0.v, p0q0.v); err != nil {
		return err
	}
	if res.Paasche, err = ratioWide("PriceIndex", "paasche", p1q1.v, p0q1.v); err != nil {
		return err
	}
	if res.Laspeyres.OK && res.Paasche.OK && res.Laspeyres.Value >= 0 && res.Paasche.Value >= 0 {
		// Each factor is between 0 and MaxIndexBP, so the product is below 2^62; a
		// negative index (negative corrected units) leaves Fisher undefined.
		root := measure.IsqrtNearest(uint64(res.Laspeyres.Value) * uint64(res.Paasche.Value))
		res.Fisher = Index{Value: BP(root), OK: true, Clamped: res.Laspeyres.Clamped || res.Paasche.Clamped}
	}
	st, err := ratioWide("PriceIndex", "stale", stale.v, p0q0.v)
	if err != nil {
		return err
	}
	res.Stale, res.StaleOK = st.Value, st.OK
	res.Warnings = indexWarnings(res)
	return nil
}

// indexWarnings lists the warnings of an index result in ascending order.
func indexWarnings(res *IndexResult) []Warning {
	var out []Warning
	if res.StaleOK && res.Stale > 0 {
		out = append(out, WarnStale)
	}
	if res.Laspeyres.Clamped || res.Paasche.Clamped || res.Fisher.Clamped {
		out = append(out, WarnClamped)
	}
	return out
}

// Chain links consecutive indices into a chain index: C_0 = 10 000 and C_k =
// rdiv(C_{k-1} * I_k, 10 000), each link rounded once (spec-indicators.md 5.3.5,
// FR-IND-022). It returns the points C_1 to C_n. A clamp is sticky: once a link
// or a point is clamped every later point is Clamped. A link that is undefined
// leaves the chain undefined from that point on. A direct index against a fixed
// base has one rounding and no drift; the chain is for a long series whose
// weights must move.
func Chain(links []Index) []Index {
	out := make([]Index, 0, len(links))
	prev := Index{Value: 10_000, OK: true}
	for _, l := range links {
		if !prev.OK || !l.OK {
			prev = Index{}
			out = append(out, prev)
			continue
		}
		// Both factors are at most MaxIndexBP, so the product is below 2^62 and the
		// guard of ratioWide cannot act.
		q, _ := measure.RoundDiv(measure.MulInt64(int64(prev.Value), int64(l.Value)), measure.FromInt64(10_000))
		v, clamped := measure.ClampBP(q)
		prev = Index{Value: BP(v), OK: true, Clamped: clamped || prev.Clamped || l.Clamped}
		out = append(out, prev)
	}
	return out
}

// RealResult is a nominal total, its value at base prices and the deflator
// between them (spec-indicators.md 5.3.6): Nominal = sum p * q, Real = sum
// p_base * q and Deflator = rdiv(10 000 * Nominal, Real), defined where Real is
// not 0.
type RealResult struct {
	Nominal, Real int64
	Deflator      Index
}

// Real values the quantities q of the cells in sel at the prices cur and at the
// base prices. N and R are exact sums, the deflator is the Paasche-type implicit
// price index of the total, and N = R * deflator / 10 000 up to one rounding
// (FR-IND-013). q is a table at region * Commodities + commodity, for instance
// Window.Quantities.
func Real(q []int64, base, cur Prices, sel CellSel) (RealResult, error) {
	if base.ConfigHash != cur.ConfigHash || base.Regions != cur.Regions || base.Commodities != cur.Commodities {
		return RealResult{}, wrap("Real", "prices", ErrConfigMismatch)
	}
	if len(q) != base.Regions*base.Commodities {
		return RealResult{}, wrap("Real", "quantities", ErrRange)
	}
	var nominal, real acc
	for i, n := range q {
		r, c := economysim.RegionID(i/base.Commodities), economysim.CommodityID(i%base.Commodities)
		if sel.has(r, c) {
			nominal.add(n, cur.At(r, c))
			real.add(n, base.At(r, c))
		}
	}
	var res RealResult
	var err error
	if res.Nominal, err = nominal.int64("Real", "nominal"); err != nil {
		return RealResult{}, err
	}
	if res.Real, err = real.int64("Real", "real"); err != nil {
		return RealResult{}, err
	}
	res.Deflator, err = ratioWide("Real", "deflator", measure.FromInt64(res.Nominal), measure.FromInt64(res.Real))
	return res, err
}
