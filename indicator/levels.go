package indicator

import (
	"cmp"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// BasketRow is the baseline basket of one region (the world's upkeep plus the
// region's ExtraUpkeep, one household for one tick) at the published prices and
// at the reference prices (spec-indicators.md 8.4 a). Level is 10 000 exactly at
// the reference prices. Stale is the share of the basket's cost at the published
// prices of goods whose price is carried forward (it did not trade last step).
type BasketRow struct {
	Region        economysim.RegionID
	Cost, RefCost int64
	Level         Index
	Stale         BP
	StaleOK       bool
}

// BasketResult is the price level of the fixed basket of each region
// (spec-indicators.md 9.1, row 4). The basket is fixed: it does not follow what
// households buy.
type BasketResult struct {
	ConfigHash [32]byte
	Tick       int64
	Rows       []BasketRow
	Warnings   []Warning
}

// Ratio returns the price level of region of against region against, from the
// exact sums in one division: rdiv(10 000 * cost_of * ref_against, ref_of *
// cost_against). It is undefined if either region is not in the result or a
// cost or a reference cost is 0 (the level of that region is undefined).
func (r BasketResult) Ratio(of, against economysim.RegionID) (Index, error) {
	a, aok := r.row(of)
	b, bok := r.row(against)
	if !aok || !bok || a.Cost == 0 || b.Cost == 0 || a.RefCost == 0 || b.RefCost == 0 {
		return Index{}, nil
	}
	return ratioWide("BasketLevel", "ratio", measure.MulInt64(a.Cost, b.RefCost), measure.MulInt64(a.RefCost, b.Cost))
}

// row finds the row of a region.
func (r BasketResult) row(g economysim.RegionID) (BasketRow, bool) {
	for _, x := range r.Rows {
		if x.Region == g {
			return x, true
		}
	}
	return BasketRow{}, false
}

// BasketLevel returns the cost of the baseline basket of each region at the
// published prices of the snapshot over its computed cost at the region's
// effective reference prices, rdiv(10 000 * cost, Bcost_r) (spec-indicators.md
// 8.4 a). cfg is the configuration of the snapshot's world as World.Config
// returns it (ErrConfigMismatch otherwise); a region with an empty basket or a
// zero reference cost has Level undefined.
func BasketLevel(cfg economysim.Config, s economysim.Snapshot) (BasketResult, error) {
	hash := economysim.HashConfig(cfg)
	if s.ConfigHash != hash {
		return BasketResult{}, wrap("BasketLevel", "snapshot", ErrConfigMismatch)
	}
	p, err := PricesOfSnapshot(s)
	if err != nil {
		return BasketResult{}, err
	}
	ref, ids := pricesOfReference(cfg, hash), commodityIDs(cfg)
	ages := make(map[[2]int64]int64, len(s.Markets)) // looked up, never iterated
	for _, m := range s.Markets {
		ages[[2]int64{int64(m.State.Region), int64(m.State.Commodity)}] = m.State.Age
	}
	res := BasketResult{ConfigHash: s.ConfigHash, Tick: s.Tick}
	for g, region := range cfg.Regions {
		row, err := basketRow(cfg, ages, p, ref, ids, economysim.RegionID(g), region)
		if err != nil {
			return BasketResult{}, err
		}
		if row.StaleOK && row.Stale > 0 && !slices.Contains(res.Warnings, WarnStale) {
			res.Warnings = append(res.Warnings, WarnStale)
			slices.Sort(res.Warnings) // a result lists its warnings in ascending order
		}
		res.Warnings = clampWarning(res.Warnings, row.Level)
		res.Rows = append(res.Rows, row)
	}
	return res, nil
}

// basketRow prices the basket of one region.
func basketRow(cfg economysim.Config, ages map[[2]int64]int64, p, ref Prices, ids map[string]economysim.CommodityID, g economysim.RegionID, region economysim.Region) (BasketRow, error) {
	var cost, refCost, stale acc
	for _, it := range slices.Concat(cfg.World.Upkeep, region.ExtraUpkeep) {
		c, ok := ids[it.Commodity]
		if !ok {
			return BasketRow{}, wrap("BasketLevel", it.Commodity, ErrRange)
		}
		cost.add(int64(it.Quantity), p.At(g, c))
		refCost.add(int64(it.Quantity), ref.At(g, c))
		if ages[[2]int64{int64(g), int64(c)}] > 0 {
			stale.add(int64(it.Quantity), p.At(g, c))
		}
	}
	row := BasketRow{Region: g}
	var err error
	if row.Cost, err = cost.int64("BasketLevel", "cost"); err != nil {
		return BasketRow{}, err
	}
	if row.RefCost, err = refCost.int64("BasketLevel", "ref_cost"); err != nil {
		return BasketRow{}, err
	}
	if row.Level, err = ratioWide("BasketLevel", "level", cost.v, refCost.v); err != nil {
		return BasketRow{}, err
	}
	st, err := ratioWide("BasketLevel", "stale", stale.v, cost.v)
	row.Stale, row.StaleOK = st.Value, st.OK
	return row, err
}

// Lane is the delivered price of a commodity on a lane (spec-indicators.md 8.4
// b): the origin's price plus the freight per unit of cargo of the lane's hired
// mode. A lane with no hired shipment (the traveller carried it) has Hired false
// and Freight 0: its cost is the traveller's upkeep, wear and time, which is not
// a price. FreightOK is false for a hired lane that dispatched no units.
type Lane struct {
	Origin, Destination economysim.RegionID
	Mode                int64
	Commodity           economysim.CommodityID
	Delivered, Freight  int64
	HaulValue, Units    int64
	Hired, FreightOK    bool
}

// DeliveredResult is the delivered price of the commodities that arrived on each
// lane of a window (spec-indicators.md 9.1, row 5).
type DeliveredResult struct {
	ConfigHash [32]byte
	From, To   int64
	Lanes      []Lane
}

// DeliveredPrice returns, for each dispatch lane of the window and each commodity
// that arrived on it, the origin's price plus rdiv(HaulValue, Units), one
// division from the dispatch events' own sums. haul[m] is the haul commodity of
// transport mode m as the configuration gives it (NoCommodity for a mode with
// none); the freight is per unit of cargo whatever the commodity, a convention
// that is exact for a single-line shipment (8.4 b).
func DeliveredPrice(w Window, haul []economysim.CommodityID, p Prices) (DeliveredResult, error) {
	if err := w.checkTable("DeliveredPrice", p); err != nil {
		return DeliveredResult{}, err
	}
	res := DeliveredResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	for _, d := range w.Dispatches {
		lane, err := laneOf(d, haul, p)
		if err != nil {
			return DeliveredResult{}, err
		}
		for _, a := range w.Arrivals {
			if a.Origin != d.Origin || a.Destination != d.Destination {
				continue
			}
			l := lane
			l.Commodity = a.Commodity
			if l.Delivered, err = net("DeliveredPrice", "delivered", []int64{int64(p.At(d.Origin, a.Commodity)), l.Freight}); err != nil {
				return DeliveredResult{}, err
			}
			res.Lanes = append(res.Lanes, l)
		}
	}
	slices.SortStableFunc(res.Lanes, func(a, b Lane) int {
		return cmp.Or(cmp.Compare(a.Origin, b.Origin), cmp.Compare(a.Destination, b.Destination), cmp.Compare(a.Mode, b.Mode), cmp.Compare(a.Commodity, b.Commodity))
	})
	return res, nil
}

// laneOf prices the freight of one dispatch lane.
func laneOf(d DispatchTally, haul []economysim.CommodityID, p Prices) (Lane, error) {
	lane := Lane{Origin: d.Origin, Destination: d.Destination, Mode: d.Mode, Units: d.Units}
	if d.Mode < 0 || d.Mode >= int64(len(haul)) || haul[d.Mode] == economysim.NoCommodity || d.Hired == 0 {
		return lane, nil
	}
	lane.Hired = true
	var value acc
	value.add(d.HaulUnits, p.At(d.Origin, haul[d.Mode]))
	var err error
	if lane.HaulValue, err = value.int64("DeliveredPrice", "haul_value"); err != nil {
		return Lane{}, err
	}
	if lane.Units == 0 {
		return lane, nil
	}
	lane.Freight, lane.FreightOK, err = Quotient(lane.HaulValue, lane.Units)
	return lane, err
}

// GapRow is the price gap of one commodity between an origin and a destination
// region: the absolute difference of the two prices as a share of their mean
// (Spread, from the exact sums), the signed local gap in minor units, and the
// gap after freight against the delivered price where the lane has one. An
// unconnected pair is a row with Connected false: a gap with no route is not a
// failure of arbitrage (spec-indicators.md 9.1, row 6).
type GapRow struct {
	Origin, Destination economysim.RegionID
	Commodity           economysim.CommodityID
	Spread              Index
	GapLocal            int64
	// GapAfterFreight is the destination price less the delivered price and
	// GapAfterFreightBP its share of the delivered price; AfterFreightOK says the
	// lane has a delivered price.
	GapAfterFreight   int64
	GapAfterFreightBP Index
	AfterFreightOK    bool
	Connected         bool
}

// GapResult is the price gaps of a window. MeanSpread averages the spread over
// the connected pairs only, rounded half to even, and Unconnected counts the
// pairs excluded.
type GapResult struct {
	ConfigHash  [32]byte
	From, To    int64
	Rows        []GapRow
	MeanSpread  Index
	Pairs       int64
	Unconnected int64
	Warnings    []Warning
}

// PriceGap returns the gap between every ordered pair of regions for each
// commodity that has a price in both, with the delivered prices of dl (a result
// of DeliveredPrice with the same table) and the components of the graph g, in
// which a pair is connected when its regions share one. ErrRange if the graph
// has another number of regions than the window.
func PriceGap(w Window, p Prices, dl DeliveredResult, g economysim.Graph) (GapResult, error) {
	if err := w.checkTable("PriceGap", p); err != nil {
		return GapResult{}, err
	}
	if g.Len() != w.Regions {
		return GapResult{}, wrap("PriceGap", "graph", ErrRange)
	}
	if len(dl.Lanes) > 0 {
		if dl.ConfigHash != w.ConfigHash {
			return GapResult{}, wrap("PriceGap", "delivered", ErrConfigMismatch)
		}
		if dl.From != w.From || dl.To != w.To {
			return GapResult{}, wrap("PriceGap", "delivered", ErrWindowOrder)
		}
	}
	res := GapResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	var sum acc
	for o := 0; o < w.Regions; o++ {
		for d := 0; d < w.Regions; d++ {
			if err := res.addPair(w, p, dl, g, economysim.RegionID(o), economysim.RegionID(d), &sum); err != nil {
				return GapResult{}, err
			}
		}
	}
	mean, err := ratioWide("PriceGap", "mean_spread", sum.v, measure.FromInt64(res.Pairs*10_000))
	res.MeanSpread = mean
	res.Warnings = clampWarning(withWarnings(w), mean)
	for _, row := range res.Rows {
		res.Warnings = clampWarning(res.Warnings, row.Spread, row.GapAfterFreightBP)
	}
	if res.Unconnected > 0 {
		res.Warnings = append(res.Warnings, WarnNoRoute)
		slices.Sort(res.Warnings)
	}
	return res, err
}

// addPair adds the rows of every commodity between an ordered pair of regions.
func (res *GapResult) addPair(w Window, p Prices, dl DeliveredResult, g economysim.Graph, o, d economysim.RegionID, sum *acc) error {
	for c := 0; c < w.Commodities; c++ {
		row, ok, err := gapRow(p, dl, g, o, d, economysim.CommodityID(c))
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		res.Rows = append(res.Rows, row)
		switch {
		case !row.Connected:
			res.Unconnected++
		case row.Spread.OK:
			sum.addInt(measure.FromInt64(int64(row.Spread.Value)))
			res.Pairs++
		}
	}
	return nil
}

// gapRow builds the gap of one commodity between two regions; ok is false for
// the same region or a pair where either price is not above 0.
func gapRow(p Prices, dl DeliveredResult, g economysim.Graph, o, d economysim.RegionID, c economysim.CommodityID) (GapRow, bool, error) {
	po, pd := int64(p.At(o, c)), int64(p.At(d, c))
	if o == d || po <= 0 || pd <= 0 {
		return GapRow{}, false, nil
	}
	co, cok := g.Component(o)
	cd, dok := g.Component(d)
	row := GapRow{Origin: o, Destination: d, Commodity: c, Connected: cok && dok && co == cd}
	var err error
	if row.GapLocal, err = net("PriceGap", "gap", []int64{pd}, po); err != nil {
		return GapRow{}, false, err
	}
	// both prices are above 0, so the 128-bit difference and sum cannot overflow
	diff, _ := measure.FromInt64(pd).Sub(measure.FromInt64(po))
	if diff.Sign() < 0 {
		diff = diff.Neg()
	}
	twice, _ := diff.Add(diff)
	mean2, _ := measure.FromInt64(pd).Add(measure.FromInt64(po))
	if row.Spread, err = ratioWide("PriceGap", "spread", twice, mean2); err != nil {
		return GapRow{}, false, err
	}
	err = row.afterFreight(dl, pd)
	return row, true, err
}

// afterFreight sets the gap against the lowest delivered price among the lanes
// of the pair (the cheapest mode, the one an arbitrage would use; the first of
// equal prices in lane order), if any.
func (row *GapRow) afterFreight(dl DeliveredResult, pd int64) error {
	best, found := int64(0), false
	for _, l := range dl.Lanes {
		if l.Origin == row.Origin && l.Destination == row.Destination && l.Commodity == row.Commodity && (!found || l.Delivered < best) {
			best, found = l.Delivered, true
		}
	}
	if !found {
		return nil
	}
	var err error
	if row.GapAfterFreight, err = net("PriceGap", "after_freight", []int64{pd}, best); err != nil {
		return err
	}
	row.AfterFreightOK = true
	row.GapAfterFreightBP, err = ratioWide("PriceGap", "after_freight", measure.FromInt64(row.GapAfterFreight), measure.FromInt64(best))
	return err
}
