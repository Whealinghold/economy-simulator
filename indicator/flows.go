package indicator

import (
	"cmp"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// Corrected returns the flows of the window by (region, reason, commodity), in
// that order: the tally's rows less the opening snapshot's pending totals plus
// the closing snapshot's (spec-indicators.md 7.1.5). A move made by a call
// before Open is in the opening balances and in the first tick's BoundaryFlow,
// and one made after the last step is in the closing balances and in no flow;
// this is the correction that makes the identities of section 7 exact for both.
// The construction bucket is not in it (7.1.4). A row whose corrected total is 0
// stays; one that no event carried but a pending total did is added. err is
// ErrOverflow for a total that does not fit int64.
func (w Window) Corrected() ([]economysim.FlowTotal, error) {
	rows := slices.Clone(w.Flows)
	for _, p := range w.ClosePending {
		if err := foldPending(&rows, p, 1); err != nil {
			return nil, err
		}
	}
	for _, p := range w.OpenPending {
		if err := foldPending(&rows, p, -1); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

// foldPending adds sign * p.Total to the row of p, inserting the row when no
// event made it.
func foldPending(rows *[]economysim.FlowTotal, p economysim.FlowTotal, sign int64) error {
	i, found := slices.BinarySearchFunc(*rows, p, func(r, k economysim.FlowTotal) int {
		return cmp.Or(cmp.Compare(r.Region, k.Region), cmp.Compare(r.Reason, k.Reason), cmp.Compare(r.Commodity, k.Commodity))
	})
	if !found {
		*rows = slices.Insert(*rows, i, economysim.FlowTotal{Region: p.Region, Reason: p.Reason, Kind: p.Kind, Commodity: p.Commodity})
	}
	delta := measure.FromInt64(p.Total)
	if sign < 0 {
		delta = delta.Neg()
	}
	sum, ok := measure.FromInt64((*rows)[i].Total).Add(delta)
	v, fits := measure.ToInt64(sum)
	if !ok || !fits {
		return wrap("Window.Corrected", string(p.Reason), ErrOverflow)
	}
	(*rows)[i].Total = v
	return nil
}

// flowSet is the corrected flows of a window with the lookups the accounts use.
type flowSet struct {
	rows                 []economysim.FlowTotal
	regions, commodities int
}

// flowsOf corrects the flows of w.
func flowsOf(w Window) (flowSet, error) {
	rows, err := w.Corrected()
	return flowSet{rows: rows, regions: w.Regions, commodities: w.Commodities}, err
}

// units returns the corrected total of a goods reason as a dense table at
// region * commodities + commodity. A row outside the table is ErrRange.
func (f flowSet) units(reason economysim.Reason) ([]int64, error) {
	out := make([]int64, f.regions*f.commodities)
	for _, r := range f.rows {
		if r.Reason != reason || r.Commodity == economysim.NoCommodity {
			continue
		}
		if r.Region < 0 || int(r.Region) >= f.regions || r.Commodity < 0 || int(r.Commodity) >= f.commodities {
			return nil, wrap("Window.Corrected", string(reason), ErrRange)
		}
		out[int(r.Region)*f.commodities+int(r.Commodity)] = r.Total
	}
	return out, nil
}

// FlowSel selects flow rows by region, reason and commodity; a nil slice means
// all (spec-indicators.md 3.6). A class is turned into its commodities by
// Classes.Of before the call.
type FlowSel struct {
	Regions     []economysim.RegionID
	Reasons     []economysim.Reason
	Commodities []economysim.CommodityID
}

// has reports whether a row is selected.
func (s FlowSel) has(r economysim.FlowTotal) bool {
	return (s.Regions == nil || slices.Contains(s.Regions, r.Region)) &&
		(s.Reasons == nil || slices.Contains(s.Reasons, r.Reason)) &&
		(s.Commodities == nil || slices.Contains(s.Commodities, r.Commodity))
}

// acc is a signed 128-bit sum of products that remembers an overflow.
type acc struct {
	v    measure.Int128
	over bool
}

// add adds q * p.
func (a *acc) add(q int64, p economysim.Price) { a.addInt(measure.MulInt64(q, int64(p))) }

// sub subtracts q * p.
func (a *acc) sub(q int64, p economysim.Price) { a.addInt(measure.MulInt64(q, int64(p)).Neg()) }

// addInt adds a wide value.
func (a *acc) addInt(m measure.Int128) {
	s, ok := a.v.Add(m)
	a.v, a.over = s, a.over || !ok
}

// merge adds another sum.
func (a *acc) merge(b acc) {
	a.addInt(b.v)
	a.over = a.over || b.over
}

// int64 returns the sum, or ErrOverflow wrapped with the preset and field.
func (a acc) int64(preset, field string) (int64, error) {
	v, ok := measure.ToInt64(a.v)
	if a.over || !ok {
		return 0, wrap(preset, field, ErrOverflow)
	}
	return v, nil
}

// net returns the sum of plus less the sum of minus, or ErrOverflow wrapped with
// the preset and field: a sum of inputs never wraps (FR-IND-009).
func net(preset, field string, plus []int64, minus ...int64) (int64, error) {
	var a acc
	for _, v := range plus {
		a.addInt(measure.FromInt64(v))
	}
	for _, v := range minus {
		a.addInt(measure.FromInt64(v).Neg())
	}
	return a.int64(preset, field)
}

// checkTable returns ErrConfigMismatch or ErrRange unless the price table is one
// of the window's world and shape.
func (w Window) checkTable(preset string, p Prices) error {
	if p.ConfigHash != w.ConfigHash {
		return wrap(preset, "prices", ErrConfigMismatch)
	}
	if p.Regions != w.Regions || p.Commodities != w.Commodities || len(p.Cells) != w.Regions*w.Commodities {
		return wrap(preset, "prices", ErrRange)
	}
	return nil
}

// Value returns the sum, in minor units, of the corrected flow totals the
// selection picks, valued at p: a goods row is its units times the price of its
// region and commodity, a money row (NoCommodity) its amount. The direction of a
// flow (faucet or sink) is not applied: it is the gross flow. A tally's flows
// are a consumer's to select, as in the sketch of spec-indicators.md 9.2.
func (w Window) Value(sel FlowSel, p Prices) (economysim.Money, error) {
	if err := w.checkTable("Window.Value", p); err != nil {
		return 0, err
	}
	rows, err := w.Corrected()
	if err != nil {
		return 0, err
	}
	var a acc
	for _, r := range rows {
		if !sel.has(r) {
			continue
		}
		if r.Commodity == economysim.NoCommodity {
			a.addInt(measure.FromInt64(r.Total))
			continue
		}
		a.add(r.Total, p.At(r.Region, r.Commodity))
	}
	v, err := a.int64("Window.Value", "total")
	return economysim.Money(v), err
}

// money returns the corrected totals of one money reason by region.
func (f flowSet) money(reason economysim.Reason) []int64 {
	out := make([]int64, f.regions)
	for _, r := range f.rows {
		if r.Reason == reason && r.Commodity == economysim.NoCommodity && r.Region >= 0 && int(r.Region) < f.regions {
			out[r.Region] = r.Total
		}
	}
	return out
}
