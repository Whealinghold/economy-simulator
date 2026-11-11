package indicator

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// RangeRow is the lowest and highest published price of a market in a window
// and their ratio, High * 10 000 / Low, defined where Low is above 0
// (spec-indicators.md 9.1, row 17). Active is the ticks with volume: a market
// with no trade has a range of 10 000 only because its price is carried (Active
// 0), and Low is a published price, not a limit.
type RangeRow struct {
	Region        economysim.RegionID
	Commodity     economysim.CommodityID
	Low, High     economysim.Price
	Range         Index
	Active, Ticks int64
}

// RangeResult is the price range of the markets of a window.
type RangeResult struct {
	ConfigHash [32]byte
	From, To   int64
	Rows       []RangeRow
	Warnings   []Warning
}

// checkMarkets returns ErrRange for a hand-built window whose market table
// cannot be indexed by (region, commodity): rows but no commodity count, or a
// table that is not Regions * Commodities long (the dense tables of the flows
// and snapshots are, and a longer one would be indexed past them).
func (w Window) checkMarkets(preset string) error {
	if len(w.Markets) > 0 && (w.Commodities < 1 || len(w.Markets) != w.Regions*w.Commodities) {
		return wrap(preset, "markets", ErrRange)
	}
	return nil
}

// checkBeliefs returns ErrRange unless the belief table covers the markets.
func (w Window) checkBeliefs(preset string) error {
	if len(w.Beliefs) < len(w.Markets) {
		return wrap(preset, "beliefs", ErrRange)
	}
	return nil
}

// forMarkets calls f for each market of the window in sel, in (region, commodity)
// order; f decides what to do with a market that never cleared.
func forMarkets(w Window, sel CellSel, f func(i int, r economysim.RegionID, c economysim.CommodityID) error) error {
	if err := w.checkMarkets("Window"); err != nil {
		return err
	}
	for i := range w.Markets {
		r, c := economysim.RegionID(i/w.Commodities), economysim.CommodityID(i%w.Commodities)
		if !sel.has(r, c) {
			continue
		}
		if err := f(i, r, c); err != nil {
			return err
		}
	}
	return nil
}

// PriceRange returns the lowest and highest published price of each selected
// market that cleared in the window, the ratio High over Low and the ticks with
// volume. WarnThin marks a result with a market that never traded. Whether the
// range shrinks over a series of windows is the consumer's, from the series of
// Range values (Stats).
func PriceRange(w Window, sel CellSel) (RangeResult, error) {
	res := RangeResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	thin := false
	err := forMarkets(w, sel, func(i int, r economysim.RegionID, c economysim.CommodityID) error {
		m := w.Markets[i]
		if m.Ticks == 0 {
			return nil
		}
		var rng Index
		var err error
		if m.Low > 0 { // a ratio against a price at or below 0 is undefined, not signed
			rng, err = ratioWide("PriceRange", "range", measure.FromInt64(int64(m.High)), measure.FromInt64(int64(m.Low)))
		}
		res.Rows = append(res.Rows, RangeRow{Region: r, Commodity: c, Low: m.Low, High: m.High, Range: rng, Active: m.Active, Ticks: m.Ticks})
		thin = thin || m.Active == 0
		return err
	})
	if thin {
		res.Warnings = []Warning{WarnThin}
	}
	for _, r := range res.Rows {
		res.Warnings = clampWarning(res.Warnings, r.Range) // FR-IND-023
	}
	return res, err
}

// PinnedRow is the share of belief outcomes that ended on the floor or the
// ceiling of the reference band, (AtFloor + AtCeiling) / Outcomes, and the ticks
// in which units were left unmatched on both sides with no volume (Stuck). Share
// is undefined where Outcomes is 0, which is not zero.
type PinnedRow struct {
	Region                       economysim.RegionID
	Commodity                    economysim.CommodityID
	Share                        Index
	Outcomes, AtFloor, AtCeiling int64
	Stuck                        int64
}

// PinnedResult is the pinned share of the markets of a window.
type PinnedResult struct {
	ConfigHash [32]byte
	From, To   int64
	Rows       []PinnedRow
}

// Pinned returns, for each selected market, the share of belief outcomes that
// ended on a bound and the stuck ticks (spec-indicators.md 9.1, row 18). A
// pinned price is not a stable price.
func Pinned(w Window, sel CellSel) (PinnedResult, error) {
	if err := w.checkBeliefs("Pinned"); err != nil {
		return PinnedResult{}, err
	}
	res := PinnedResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	err := forMarkets(w, sel, func(i int, r economysim.RegionID, c economysim.CommodityID) error {
		b, m := w.Beliefs[i], w.Markets[i]
		if b.Ticks == 0 && m.Stuck == 0 {
			return nil
		}
		pinned, err := net("Pinned", "pinned", []int64{b.AtFloor, b.AtCeiling})
		if err != nil {
			return err
		}
		share, err := ratioWide("Pinned", "share", measure.FromInt64(pinned), measure.FromInt64(b.Outcomes))
		res.Rows = append(res.Rows, PinnedRow{Region: r, Commodity: c, Share: share, Outcomes: b.Outcomes, AtFloor: b.AtFloor, AtCeiling: b.AtCeiling, Stuck: m.Stuck})
		return err
	})
	return res, err
}

// DispersionRow is the spread of the belief centres of a market, (HighCentre -
// LowCentre) / MeanCentre, at the last tick of the window (Spread) and its lowest
// over the window (Lowest). Both are undefined where the mean centre is 0.
type DispersionRow struct {
	Region                            economysim.RegionID
	Commodity                         economysim.CommodityID
	Spread, Lowest                    Index
	LowCentre, HighCentre, MeanCentre economysim.Price
}

// DispersionResult is the dispersion of the beliefs of a window.
type DispersionResult struct {
	ConfigHash [32]byte
	From, To   int64
	Rows       []DispersionRow
}

// Dispersion returns the spread of the belief centres of each selected market
// (spec-indicators.md 9.1, row 19): synchronisation shows as a spread that falls
// below a floor.
func Dispersion(w Window, sel CellSel) (DispersionResult, error) {
	if err := w.checkBeliefs("Dispersion"); err != nil {
		return DispersionResult{}, err
	}
	res := DispersionResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	err := forMarkets(w, sel, func(i int, r economysim.RegionID, c economysim.CommodityID) error {
		b := w.Beliefs[i]
		if b.Ticks == 0 {
			return nil
		}
		row := DispersionRow{Region: r, Commodity: c, LowCentre: b.LowCentre, HighCentre: b.HighCentre, MeanCentre: b.MeanCentre}
		var err error
		row.Spread, err = ratioWide("Dispersion", "spread", mustSub(int64(b.HighCentre), int64(b.LowCentre)), measure.FromInt64(int64(b.MeanCentre)))
		row.Lowest = Index{Value: b.Lowest, OK: b.HasLowest}
		res.Rows = append(res.Rows, row)
		return err
	})
	return res, err
}

// OffsetRow is the signed difference of a market's window price from a
// reference price, rdiv(10 000 * (p - ref), ref); undefined for a reference at
// or below 0.
type OffsetRow struct {
	Region     economysim.RegionID
	Commodity  economysim.CommodityID
	Diff       Index
	Price, Ref economysim.Price
}

// OffsetResult is the offset of a window's prices from a reference, with the
// unmet need beside it: the share of the household-ticks of the window that were
// unfed (the Unfed share), in basis points. The offset is meaningful only
// against a reference computed for the same pack.
type OffsetResult struct {
	ConfigHash [32]byte
	From, To   int64
	Rows       []OffsetRow
	Unmet      BP
	UnmetOK    bool
}

// Offset returns the offset of the prices p (usually PricesOfWindow) from the
// reference ref for the selected cells (spec-indicators.md 9.1, row 20; the
// absolute value Offset.abs of the calibration is |Diff|). Both tables must be of
// the window's world and shape.
func Offset(w Window, p, ref Prices, sel CellSel) (OffsetResult, error) {
	for _, t := range []Prices{p, ref} {
		if err := w.checkTable("Offset", t); err != nil {
			return OffsetResult{}, err
		}
	}
	res := OffsetResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	err := forMarkets(w, sel, func(i int, r economysim.RegionID, c economysim.CommodityID) error {
		pr, rf := p.At(r, c), ref.At(r, c)
		diff, err := ratioWide("Offset", "diff", mustSub(int64(pr), int64(rf)), measure.FromInt64(int64(rf)))
		if rf <= 0 {
			diff = Index{}
		}
		res.Rows = append(res.Rows, OffsetRow{Region: r, Commodity: c, Diff: diff, Price: pr, Ref: rf})
		return err
	})
	if err != nil {
		return OffsetResult{}, err
	}
	u, err := unmetShare(w)
	res.Unmet, res.UnmetOK = u.Value, u.OK
	return res, err
}

// unmetShare is the share of the household-ticks of the window that were unfed:
// the sum of UpkeepShortfall.Unfed over the running households at the start of
// each tick, in basis points (spec-indicators.md 6.3, 9.1 row 12).
func unmetShare(w Window) (Index, error) {
	var unfed, ticks acc
	for _, p := range w.People {
		unfed.addInt(measure.FromInt64(p.Unfed))
		ticks.addInt(measure.FromInt64(p.HouseholdTicks))
	}
	return ratioWide("Offset", "unmet", unfed.v, ticks.v)
}

// CoverRow is the stock of one commodity in one region (or, for the total of a
// selection, region -1 and commodity NoCommodity) against its consumption: Held
// is the units held by agents and the estate at the close, Estate the part the
// estate holds, Consumed the units eaten and consumed as a want in the window.
// Cover is the ticks of consumption the stock covers, in basis points of one tick
// (10 000 is one tick), and Unsold the estate's share of the stock.
type CoverRow struct {
	Region                 economysim.RegionID
	Commodity              economysim.CommodityID
	Held, Estate, Consumed int64
	Cover, Unsold          Index
}

// CoverResult is the stock cover of the selected cells of a window and of the
// selection as a whole (spec-indicators.md 9.1, row 15). A glut (cover rising
// without bound) and an empty pantry (cover 0) are different failures; a cell
// with nothing consumed has its cover undefined, not infinite.
type CoverResult struct {
	ConfigHash [32]byte
	From, To   int64
	Rows       []CoverRow
	Total      CoverRow
}

// StockCover returns, for each selected cell, the units held at the close over
// the units consumed per tick, rdiv(10 000 * Held * n, Consumed) for a window of
// n ticks, and the unsold share rdiv(10 000 * Estate, Held), and the same for the
// selection as a whole.
func StockCover(w Window, closing economysim.Snapshot, sel CellSel) (CoverResult, error) {
	if err := w.checkClosing("StockCover", closing); err != nil {
		return CoverResult{}, err
	}
	st, err := stocksOf("StockCover", closing, w.Regions, w.Commodities)
	if err != nil {
		return CoverResult{}, err
	}
	f, err := flowsOf(w)
	if err != nil {
		return CoverResult{}, err
	}
	consumed, err := consumedUnits(f)
	if err != nil {
		return CoverResult{}, err
	}
	res := CoverResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	var held, estate, used acc
	err = forMarkets(w, sel, func(i int, r economysim.RegionID, c economysim.CommodityID) error {
		row, err := coverRow(w.Steps, r, c, st.held[i], st.estate[i], consumed[i])
		res.Rows = append(res.Rows, row)
		held.addInt(measure.FromInt64(st.held[i]))
		estate.addInt(measure.FromInt64(st.estate[i]))
		used.addInt(measure.FromInt64(consumed[i]))
		return err
	})
	if err != nil {
		return CoverResult{}, err
	}
	var h, e, u int64
	if h, err = held.int64("StockCover", "held"); err != nil {
		return CoverResult{}, err
	}
	if e, err = estate.int64("StockCover", "estate"); err != nil {
		return CoverResult{}, err
	}
	if u, err = used.int64("StockCover", "consumed"); err != nil {
		return CoverResult{}, err
	}
	res.Total, err = coverRow(w.Steps, -1, economysim.NoCommodity, h, e, u)
	return res, err
}

// coverRow computes the two ratios of a stock.
func coverRow(steps int64, r economysim.RegionID, c economysim.CommodityID, held, estate, consumed int64) (CoverRow, error) {
	row := CoverRow{Region: r, Commodity: c, Held: held, Estate: estate, Consumed: consumed}
	var err error
	if row.Cover, err = ratioWide("StockCover", "cover", measure.MulInt64(held, steps), measure.FromInt64(consumed)); err != nil {
		return row, err
	}
	row.Unsold, err = ratioWide("StockCover", "unsold", measure.FromInt64(estate), measure.FromInt64(held))
	return row, err
}
