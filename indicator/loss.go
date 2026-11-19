package indicator

import (
	"cmp"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// Source is where a loss came from: a cargo in transit or a store.
type Source string

// The sources of a loss (spec-indicators.md 9.1 row 25).
const (
	// SourceRoute is the units lost in transit: the Lost of each ShipmentArrived line
	// and the AccountLost of a self-carried trip.
	SourceRoute Source = "route"
	// SourceStore is the units lost from stores: the Lost of StoreLoss.
	SourceStore Source = "store"
)

// LossRow is the units lost by source, region and commodity, valued at the price
// table of the call at the cell of the region of the account that held them: the
// origin for a cargo in transit (spec-indicators.md 8.1), the owner's region for a
// store.
type LossRow struct {
	Source    Source
	Region    economysim.RegionID
	Commodity economysim.CommodityID
	Units     int64
	Value     int64
}

// StruckRow is the strikes of a source in a region: the shipments dispatched with a
// loss share above 0 (by origin) or the stores struck. A strike that took no unit is
// counted here and has no LossRow.
type StruckRow struct {
	Source Source
	Region economysim.RegionID
	Count  int64
}

// LossesResult is the units lost to strikes in a window and their value, by source,
// region and commodity, with the strikes counted apart and the check against the
// boundary flow loss.
type LossesResult struct {
	ConfigHash [32]byte
	From, To   int64
	// Rows are the cells with units lost, by source (route, then store), region and
	// commodity; Struck are the strikes by source and region.
	Rows   []LossRow
	Struck []StruckRow
	// Units and Value are the totals of the rows.
	Units, Value int64
	// Boundary is the units of the boundary flow loss of the window and Gap is Units
	// less Boundary: 0 in a window whose events and flows are the same ticks.
	Boundary, Gap int64
}

// Losses returns the units lost to strikes in a window, valued at p, by source,
// region and commodity (spec-indicators.md 9.1 row 25, FR-IND-059). The units of the
// contracts' Lost are those of the shipment lines and are not added again. The total
// is checked against the boundary flow loss of the window: Gap is 0 when every
// strike's units went to the boundary in the window.
func Losses(w Window, p Prices) (LossesResult, error) {
	const name = "Losses"
	if err := w.checkTable(name, p); err != nil {
		return LossesResult{}, err
	}
	route, store, err := lossCells(w)
	if err != nil {
		return LossesResult{}, err
	}
	res := LossesResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	var units, value acc
	for _, tbl := range [...]struct {
		src   Source
		cells []int64
	}{{SourceRoute, route}, {SourceStore, store}} {
		for i, n := range tbl.cells {
			if n == 0 {
				continue
			}
			r, c := economysim.RegionID(i/w.Commodities), economysim.CommodityID(i%w.Commodities)
			var v acc
			v.add(n, p.At(r, c))
			row := LossRow{Source: tbl.src, Region: r, Commodity: c, Units: n}
			if row.Value, err = v.int64(name, "value"); err != nil {
				return LossesResult{}, err
			}
			res.Rows = append(res.Rows, row)
			units.addInt(measure.FromInt64(n))
			value.merge(v)
		}
	}
	if res.Units, err = units.int64(name, "units"); err != nil {
		return LossesResult{}, err
	}
	if res.Value, err = value.int64(name, "value"); err != nil {
		return LossesResult{}, err
	}
	if res.Struck, err = struckRows(w); err != nil {
		return LossesResult{}, err
	}
	err = res.reconcile(w)
	return res, err
}

// lossCells sums the units lost in transit and in stores into dense tables at region
// * commodities + commodity. A row outside the table is ErrRange.
func lossCells(w Window) (route, store []int64, err error) {
	if err = w.checkDims("Losses"); err != nil {
		return nil, nil, err
	}
	n := w.Regions * w.Commodities
	route, store = make([]int64, n), make([]int64, n)
	for _, a := range w.Arrivals {
		i, ok := w.lossCell(a.Origin, a.Commodity)
		if !ok {
			return nil, nil, wrap("Losses", "arrivals", ErrRange)
		}
		if route[i], err = net("Losses", "route", []int64{route[i], a.Lost, a.AccountLost}); err != nil {
			return nil, nil, err
		}
	}
	for _, l := range w.StoreLosses {
		i, ok := w.lossCell(l.Region, l.Commodity)
		if !ok {
			return nil, nil, wrap("Losses", "store_losses", ErrRange)
		}
		if store[i], err = net("Losses", "store", []int64{store[i], l.Units}); err != nil {
			return nil, nil, err
		}
	}
	return route, store, nil
}

// lossCell returns the index of the (region, commodity) cell of the dense tables.
func (w Window) lossCell(r economysim.RegionID, c economysim.CommodityID) (int, bool) {
	if r < 0 || int(r) >= w.Regions || c < 0 || int(c) >= w.Commodities {
		return 0, false
	}
	return int(r)*w.Commodities + int(c), true
}

// struckRows returns the strikes by source and region: the shipments struck by origin,
// then the stores struck, in ascending region.
func struckRows(w Window) ([]StruckRow, error) {
	var out []StruckRow
	add := func(src Source, r economysim.RegionID, n int64) error {
		if r < 0 || int(r) >= w.Regions {
			return wrap("Losses", "struck", ErrRange)
		}
		if n == 0 {
			return nil
		}
		for i := range out {
			if out[i].Source == src && out[i].Region == r {
				var err error
				out[i].Count, err = net("Losses", "struck", []int64{out[i].Count, n})
				return err
			}
		}
		out = append(out, StruckRow{Source: src, Region: r, Count: n})
		return nil
	}
	for _, d := range w.Dispatches {
		if err := add(SourceRoute, d.Origin, d.Struck); err != nil {
			return nil, err
		}
	}
	for _, s := range w.StoreStrikes {
		if err := add(SourceStore, s.Region, s.Stores); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(out, func(a, b StruckRow) int {
		return cmp.Or(cmp.Compare(a.Source, b.Source), cmp.Compare(a.Region, b.Region))
	})
	return out, nil
}

// reconcile sets Boundary, the units of the boundary flow loss of the window, and Gap.
func (res *LossesResult) reconcile(w Window) error {
	f, err := flowsOf(w)
	if err != nil {
		return err
	}
	cells, err := f.units(economysim.ReasonLoss)
	if err != nil {
		return err
	}
	if res.Boundary, err = net("Losses", "boundary", cells); err != nil {
		return err
	}
	res.Gap, err = net("Losses", "gap", []int64{res.Units}, res.Boundary)
	return err
}
