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
	sum, ok := measure.FromInt64((*rows)[i].Total).Add(measure.FromInt64(sign * p.Total))
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
