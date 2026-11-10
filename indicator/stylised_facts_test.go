package indicator_test

import (
	"cmp"
	"fmt"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// The measures of the facts of group A (plan 2-19 D596): each reads a run's
// series and windows through the presets of indicator and returns plain
// integers; none touches a world. A number that does not exist carries a Why.

func marketKeys(m map[[2]int][]pinnedTick) [][2]int {
	keys := make([][2]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b [2]int) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) })
	return keys
}

// pinnedStats returns the longest run of pinned ticks and of frozen ticks of
// any market and the largest share of the last year's ticks a market was
// pinned (P02): a pinned tick has Pinned.Share at or above the working
// threshold r5.pinned_bp, a frozen tick has unmatched units on both sides and
// no volume (STuck of a one-tick window).
func pinnedStats(r *stylisedRun) (pinnedRun, frozenRun, lastYearShare stat) {
	var pRun, fRun, share int64
	for _, k := range marketKeys(r.pinned) {
		series := r.pinned[k]
		var p, f int64
		for i, tk := range series {
			if tk.Share >= workPinnedBP {
				p++
			} else {
				p = 0
			}
			if tk.Stuck {
				f++
			} else {
				f = 0
			}
			pRun, fRun = max(pRun, p), max(fRun, f)
			_ = i
		}
		from := max(0, len(series)-int(yearTicks))
		var n int64
		for _, tk := range series[from:] {
			if tk.Share >= workPinnedBP {
				n++
			}
		}
		share = max(share, n*10_000/yearTicks)
	}
	if len(r.households) < int(yearTicks) {
		why := "the run is shorter than a year"
		return stat{Why: why}, stat{Why: why}, stat{Why: why}
	}
	return stat{Value: pRun}, stat{Value: fRun}, stat{Value: share}
}

// bp is v * k / 10 000 rounded to nearest, saturating at MaxInt64.
func bp(v, k int64) int64 {
	q, ok := checked.MulDiv(v, k, 10_000)
	if !ok {
		return economysim.MaxPrice
	}
	return q
}

// refPrice returns the reference price of a commodity by its string ID.
func refPrice(cfg economysim.Config, id string) int64 {
	for _, c := range cfg.Commodities {
		if c.ID == id {
			return int64(c.ReferencePrice)
		}
	}
	return 0
}

// beliefInterval restates FR-BEL-004 from the configuration and the cost
// anchor of the tick: the floor is the larger of the reference-band floor and
// the cost floor, the ceiling the smaller of the reference-band ceiling and
// the cost ceiling, raised to the floor.
func beliefInterval(w economysim.WorldParams, ref, anchor int64) (floor, ceiling int64) {
	ref = max(ref, 1)
	refLow, refHigh := max(1, bp(ref, w.RefBandLow)), min(economysim.MaxPrice, bp(ref, w.RefBandHigh))
	costLow, costHigh := int64(0), int64(economysim.MaxPrice)
	if anchor > 0 {
		costLow, costHigh = bp(anchor, w.CostFloor), bp(anchor, w.CostCeiling)
	}
	floor = min(economysim.MaxPrice, max(refLow, costLow))
	return floor, max(floor, min(refHigh, costHigh))
}

// marketInterval is the loosest interval that every agent's belief bounds fit
// in: a trade price lies between a seller's ask and a buyer's bid, so it is
// known to lie in the reference band of an agent without a cost, widened up to
// the cost floor of a maker (FR-BEL-004's conflict).
func marketInterval(w economysim.WorldParams, ref, anchor int64) (lo, hi int64) {
	noCost, noCostHi := beliefInterval(w, ref, 0)
	mkFloor, mkHi := beliefInterval(w, ref, anchor)
	return noCost, max(noCostHi, mkFloor, mkHi)
}

// priceBounds checks STY-P01: every published price of every market and tick
// lies inside marketInterval, except where an estate sold at the tick the price
// was set (an estate's ask is priced by the markdown of EST 2.7.3, not by a
// belief). It returns the first violation, or "".
func priceBounds(r *stylisedRun) string {
	if r.obs.badSeq != "" {
		return r.obs.badSeq
	}
	keys := make([][2]int, 0, len(r.obs.markets))
	for k := range r.obs.markets {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b [2]int) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) })
	for _, k := range keys {
		m := r.obs.markets[k]
		ref := refPrice(r.cfg, r.names.Commodities[k[1]])
		for tick, p := range m.Price {
			lo, hi := marketInterval(r.cfg.World, ref, m.Anchor[tick])
			if p >= lo && p <= hi {
				continue
			}
			if set := int64(tick) - m.Age[tick]; set >= 0 && set < int64(len(m.EstateUnits)) && m.EstateUnits[set] > 0 {
				continue
			}
			return fmt.Sprintf("%s/%s tick %d price %d outside [%d, %d] (anchor %d)", r.names.Regions[k[0]], r.names.Commodities[k[1]], tick, p, lo, hi, m.Anchor[tick])
		}
	}
	return ""
}

// windowPrices returns the prices of a year window and the warm table of the
// pack as an indicator table (STY 4.2.3): the cell of a market the table does
// not name keeps the reference price.
func windowPrices(r *stylisedRun, w indicator.Window) (indicator.Prices, indicator.Prices, error) {
	p, err := indicator.PricesOfWindow(w)
	if err != nil {
		return p, p, err
	}
	cells := make([]economysim.Price, w.Regions*w.Commodities)
	for rg := 0; rg < w.Regions; rg++ {
		for c := 0; c < w.Commodities; c++ {
			cells[rg*w.Commodities+c] = economysim.Price(refPrice(r.cfg, r.names.Commodities[c]))
		}
	}
	if r.cfg.WarmStart != nil {
		for _, sp := range r.cfg.WarmStart.Prices {
			rg, c := slices.Index(r.names.Regions, sp.Region), slices.Index(r.names.Commodities, sp.Commodity)
			if rg >= 0 && c >= 0 {
				cells[rg*w.Commodities+c] = sp.Price
			}
		}
	}
	ref, err := indicator.NewPrices(w.ConfigHash, w.From, w.To, w.Regions, w.Commodities, cells)
	return p, ref, err
}

// rangeStats returns the largest High over Low of a market over the year
// windows and the largest growth of a range from one window to the next
// (P03). A market whose active ticks are under the working threshold of
// active_min_bp is left out; a clamped range makes the number undetermined.
func rangeStats(r *stylisedRun) (rangeMax, growthMax stat) {
	if len(r.years) < 2 {
		return stat{Why: "windows"}, stat{Why: "windows"}
	}
	prev := map[[2]int]int64{}
	var top, growth int64
	counted := false
	for _, yw := range r.years {
		res, err := indicator.PriceRange(yw.w, indicator.CellSel{})
		if err != nil {
			return stat{Why: err.Error()}, stat{Why: err.Error()}
		}
		for _, row := range res.Rows {
			if row.Ticks == 0 || row.Active*10_000 < workActiveMin*row.Ticks || !row.Range.OK {
				continue
			}
			if row.Range.Clamped {
				return stat{Why: "clamped"}, stat{Why: "clamped"}
			}
			k := [2]int{int(row.Region), int(row.Commodity)}
			v := int64(row.Range.Value)
			top, counted = max(top, v), true
			if p, ok := prev[k]; ok && p > 0 {
				if g, ok, err := indicator.Quotient((v-p)*10_000, p); err == nil && ok {
					growth = max(growth, g)
				}
			}
			prev[k] = v
		}
	}
	if !counted {
		return stat{Why: "active"}, stat{Why: "active"}
	}
	return stat{Value: top}, stat{Value: growth}
}

// offsetStats returns the largest absolute offset of a market against the warm
// table over window w, and the unfed share of the window beside it (P04).
func offsetStats(r *stylisedRun, w runWin) (offset, unmet stat) {
	p, ref, err := windowPrices(r, w.w)
	if err != nil {
		return stat{Why: err.Error()}, stat{Why: err.Error()}
	}
	res, err := indicator.Offset(w.w, p, ref, indicator.CellSel{})
	if err != nil {
		return stat{Why: err.Error()}, stat{Why: err.Error()}
	}
	var top int64
	seen := false
	for _, row := range res.Rows {
		if !row.Diff.OK {
			continue
		}
		seen = true
		top = max(top, abs(int64(row.Diff.Value)))
	}
	offset = stat{Value: top}
	if !seen {
		offset = stat{Why: "no reference price"}
	}
	if !res.UnmetOK {
		return offset, stat{Why: "no household-ticks"}
	}
	return offset, stat{Value: int64(res.Unmet)}
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// driftStat returns the absolute growth in basis points of the Fisher index of
// the whole economy with consumed weights from the second to the third year
// window (P07: one link after the transient), with its blind-spot fields.
func driftStat(r *stylisedRun) stat {
	if len(r.years) < 3 {
		return stat{Why: "windows"}
	}
	w0, w1 := r.years[1], r.years[2]
	p0, _, err := windowPrices(r, w0.w)
	if err == nil {
		var base indicator.Base
		if base, err = indicator.BaseOf(w0.w, p0); err == nil {
			var p1 indicator.Prices
			if p1, _, err = windowPrices(r, w1.w); err == nil {
				var res indicator.IndexResult
				if res, err = indicator.PriceIndex(w1.w, base, indicator.WeightsConsumed, p1, indicator.CellSel{}); err == nil {
					switch {
					case !res.Fisher.OK:
						return stat{Why: "undefined"}
					case res.Fisher.Clamped:
						return stat{Why: "clamped"}
					case res.StaleOK && int64(res.Stale) > workStaleMax:
						return stat{Why: "stale"}
					}
					return stat{Value: abs(int64(res.Fisher.Value) - 10_000)}
				}
			}
		}
	}
	return stat{Why: err.Error()}
}

// unfedStat returns the largest unfed share of household-ticks of the year
// windows after the transient (D01): the first year is left out.
func unfedStat(r *stylisedRun) stat {
	if len(r.years) < 2 {
		return stat{Why: "windows"}
	}
	var top int64
	for _, yw := range r.years[1:] {
		res, err := indicator.Unfed(yw.w, yw.closing)
		if err != nil {
			return stat{Why: err.Error()}
		}
		if !res.Total.Share.OK {
			return stat{Why: "no household-ticks"}
		}
		top = max(top, int64(res.Total.Share.Value))
	}
	return stat{Value: top}
}

// swingStat returns the largest swing of a role's share of its region over the
// closes of the year windows (D03), a role with no households read as 0.
func swingStat(r *stylisedRun) stat {
	if len(r.years) < 2 {
		return stat{Why: "windows"}
	}
	series := map[[2]int][]indicator.Point{}
	for _, yw := range r.years {
		res, err := indicator.RoleShares(yw.w)
		if err != nil {
			return stat{Why: err.Error()}
		}
		for _, row := range res.Rows {
			k := [2]int{int(row.Region), int(row.Role)}
			series[k] = append(series[k], indicator.Point{Tick: yw.w.To, Value: int64(row.CloseShare.Value)})
		}
	}
	var top int64
	for _, s := range series {
		if sw, ok := indicator.Swing(s); ok {
			top = max(top, sw)
		}
	}
	return stat{Value: top}
}

// moneyStats returns the exact check of M01 and the largest share of money out
// of circulation of M05 at the closes of the year windows.
func moneyStats(r *stylisedRun) (m01 string, poolShare stat) {
	for i, m := range r.money {
		if m != int64(r.start.Money) {
			m01 = fmt.Sprintf("money %d at tick %d, want %d (no faucet, no sink)", m, i+1, r.start.Money)
			break
		}
	}
	var top int64
	seen := false
	for _, yw := range r.years {
		p, _, err := windowPrices(r, yw.w)
		if err != nil {
			return "windows: " + err.Error(), stat{Why: err.Error()}
		}
		res, err := indicator.Sectors(yw.w, yw.open, yw.closing, p)
		if err != nil {
			return "Sectors: " + err.Error(), stat{Why: err.Error()}
		}
		s := res.Realm
		if sum := s.MoneyHouseholds + s.MoneyPool + s.MoneyContract + s.MoneyTreasury; sum != s.Money && m01 == "" {
			m01 = fmt.Sprintf("sectors add up to %d at tick %d, want M %d", sum, yw.w.To, s.Money)
		}
		if s.Money > 0 {
			seen = true
			out, _, _ := indicator.Quotient((s.MoneyPool+s.MoneyContract+s.MoneyTreasury)*10_000, s.Money)
			top = max(top, out)
		}
	}
	if !seen {
		return m01, stat{Why: "no money"}
	}
	return m01, stat{Value: top}
}

// minHouseholds returns the least households at any tick as a share of the
// start, in basis points (G01, H02).
func minHouseholds(r *stylisedRun) stat {
	if r.start.Households <= 0 {
		return stat{Why: "no households at the start"}
	}
	low := r.start.Households
	for _, h := range r.households {
		low = min(low, h)
	}
	q, _, _ := indicator.Quotient(low*10_000, r.start.Households)
	return stat{Value: q}
}

// moneyDrift returns the largest gap of the money of the households per
// household from its start over the year closes, in basis points (H02).
func moneyDrift(r *stylisedRun) stat {
	if len(r.years) == 0 {
		return stat{Why: "windows"}
	}
	var start, top int64
	var haveStart bool
	for i, yw := range r.years {
		p, _, err := windowPrices(r, yw.w)
		if err != nil {
			return stat{Why: err.Error()}
		}
		res, err := indicator.Sectors(yw.w, yw.open, yw.closing, p)
		if err != nil {
			return stat{Why: err.Error()}
		}
		if i == 0 {
			// The start per household is the money of the first snapshot over its households:
			// the pool of a warm start is empty, so it is the households' (shortcut: a
			// warm pool would make this an upper bound; replace by Sectors at the open).
			if r.start.Households <= 0 {
				return stat{Why: "no households at the start"}
			}
			start, haveStart = int64(r.start.Money)/r.start.Households, true
		}
		if !res.Realm.PerHouseholdOK {
			return stat{Why: "no households at a close"}
		}
		if haveStart && start > 0 {
			g, _, _ := indicator.Quotient(abs(res.Realm.PerHousehold-start)*10_000, start)
			top = max(top, g)
		}
	}
	return stat{Value: top}
}

// ---- the detectors' input (G01) -----------------------------------------------

// series1 turns a slice of values into points at the given ticks.
func point(tick, value int64) verdict.Point { return verdict.Point{Tick: tick, Value: value} }

// detectorInput converts the windows and series of a run into the plain input of
// internal/verdict, inline at the call (architecture.md 2.9). The parts that need
// the agent rows of plan 2-17 (the top decile of R2) and the households per role
// per tick (R3's role clause) are left out: the detector reads an absent series
// as nothing to read.
func detectorInput(r *stylisedRun) (verdict.Input, error) {
	var in verdict.Input
	in.MInit = int64(r.start.Money)
	in.TicksPerYear = r.cfg.Calendar.TicksPerYear()
	classes := indicator.NewClasses(r.cfg)
	sels := []indicator.CellSel{{}}
	for _, name := range classes.Names() {
		cs, err := classes.Of(name)
		if err != nil {
			return in, err
		}
		sels = append(sels, indicator.CellSel{Commodities: cs})
	}
	if err := windowSeries(r, &in, sels); err != nil {
		return in, err
	}
	tickSeries(r, &in)
	return in, nil
}

// windowSeries fills the series that have one point per year window.
func windowSeries(r *stylisedRun, in *verdict.Input, sels []indicator.CellSel) error {
	links := make([]verdict.Series, len(sels))
	ranges := map[[2]int]*verdict.Series{}
	offsets := map[[2]int]*verdict.Series{}
	disp := map[[2]int][]verdict.Point{}
	swings := map[[2]int][]verdict.Point{}
	unsold := map[[2]int][]verdict.Point{}
	var prev indicator.Prices
	var prevW indicator.Window
	var active, ticks int64
	for k, yw := range r.years {
		p, ref, err := windowPrices(r, yw.w)
		if err != nil {
			return err
		}
		if k > 0 {
			base, err := indicator.BaseOf(prevW, prev)
			if err != nil {
				return err
			}
			for i, sel := range sels {
				res, err := indicator.PriceIndex(yw.w, base, indicator.WeightsConsumed, p, sel)
				if err != nil {
					return err
				}
				links[i].Points = append(links[i].Points, point(yw.w.From, int64(res.Fisher.Value)))
				links[i].Clamped = append(links[i].Clamped, res.Fisher.Clamped)
				if res.StaleOK {
					in.Stale = max(in.Stale, int64(res.Stale))
				}
			}
		}
		prev, prevW = p, yw.w
		if err := oneWindow(r, in, yw, p, ref, ranges, offsets, disp, swings, unsold, &active, &ticks); err != nil {
			return err
		}
	}
	in.Links = links
	for _, k := range sortedKeys(ranges) {
		in.Ranges = append(in.Ranges, *ranges[k])
	}
	for _, k := range sortedKeys(offsets) {
		in.Offsets = append(in.Offsets, *offsets[k])
	}
	for _, k := range sortedKeys(disp) {
		in.Dispersion = append(in.Dispersion, disp[k])
	}
	for _, k := range sortedKeys(swings) {
		in.Swings = append(in.Swings, swings[k])
	}
	for _, k := range sortedKeys(unsold) {
		in.Unsold = append(in.Unsold, unsold[k])
	}
	if ticks > 0 {
		in.Active = active * 10_000 / ticks
	}
	in.HasReference = r.cfg.WarmStart != nil
	return nil
}

func sortedKeys[V any](m map[[2]int]V) [][2]int {
	keys := make([][2]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b [2]int) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) })
	return keys
}

// oneWindow adds the points of one year window to the series of the input.
func oneWindow(r *stylisedRun, in *verdict.Input, yw runWin, p, ref indicator.Prices,
	ranges, offsets map[[2]int]*verdict.Series, disp, swings, unsold map[[2]int][]verdict.Point, active, ticks *int64) error {
	from := yw.w.From
	rg, err := indicator.PriceRange(yw.w, indicator.CellSel{})
	if err != nil {
		return err
	}
	for _, row := range rg.Rows {
		k := [2]int{int(row.Region), int(row.Commodity)}
		if ranges[k] == nil {
			ranges[k] = &verdict.Series{}
		}
		if row.Range.OK {
			ranges[k].Points = append(ranges[k].Points, point(from, int64(row.Range.Value)))
			ranges[k].Clamped = append(ranges[k].Clamped, row.Range.Clamped)
		}
		*active, *ticks = *active+row.Active, *ticks+row.Ticks
	}
	of, err := indicator.Offset(yw.w, p, ref, indicator.CellSel{})
	if err != nil {
		return err
	}
	for _, row := range of.Rows {
		k := [2]int{int(row.Region), int(row.Commodity)}
		if offsets[k] == nil {
			offsets[k] = &verdict.Series{}
		}
		if row.Diff.OK {
			offsets[k].Points = append(offsets[k].Points, point(from, int64(row.Diff.Value)))
			offsets[k].Clamped = append(offsets[k].Clamped, row.Diff.Clamped)
		}
	}
	if of.UnmetOK {
		in.Unmet = append(in.Unmet, point(from, int64(of.Unmet)))
	}
	if err := moneyAndTrade(r, in, yw, p, unsold); err != nil {
		return err
	}
	ds, err := indicator.Dispersion(yw.w, indicator.CellSel{})
	if err != nil {
		return err
	}
	for _, row := range ds.Rows {
		if row.Lowest.OK {
			k := [2]int{int(row.Region), int(row.Commodity)}
			disp[k] = append(disp[k], point(from, int64(row.Lowest.Value)))
		}
	}
	rs, err := indicator.RoleShares(yw.w)
	if err != nil {
		return err
	}
	for _, row := range rs.Rows {
		k := [2]int{int(row.Region), int(row.Role)}
		swings[k] = append(swings[k], point(from, abs(int64(row.CloseShare.Value)-int64(row.OpenShare.Value))))
	}
	return nil
}

// moneyAndTrade adds the money, the volume and the unsold shares of a window.
func moneyAndTrade(r *stylisedRun, in *verdict.Input, yw runWin, p indicator.Prices, unsold map[[2]int][]verdict.Point) error {
	from := yw.w.From
	sc, err := indicator.Sectors(yw.w, yw.open, yw.closing, p)
	if err != nil {
		return err
	}
	s := sc.Realm
	in.M = append(in.M, point(from, s.Money))
	if s.Money > 0 {
		q, _, _ := indicator.Quotient((s.MoneyPool+s.MoneyContract+s.MoneyTreasury)*10_000, s.Money)
		in.MoneyOut = append(in.MoneyOut, point(from, q))
	}
	tv, err := indicator.Turnover(yw.w, yw.closing, r.cfg.Calendar.TicksPerYear(), indicator.CellSel{})
	if err != nil {
		return err
	}
	in.Volume = append(in.Volume, point(from, tv.Volume))
	cv, err := indicator.StockCover(yw.w, yw.closing, indicator.CellSel{})
	if err != nil {
		return err
	}
	for _, row := range cv.Rows {
		if row.Consumed > 0 && row.Unsold.OK {
			k := [2]int{int(row.Region), int(row.Commodity)}
			unsold[k] = append(unsold[k], point(from, int64(row.Unsold.Value)))
		}
	}
	return nil
}

// tickSeries fills the one-tick series: households, pinned shares, frozen ticks
// and the completions of the first ticks.
func tickSeries(r *stylisedRun, in *verdict.Input) {
	hh := make([]verdict.Point, len(r.households))
	for i, h := range r.households {
		hh[i] = point(int64(i), h)
	}
	in.Households = [][]verdict.Point{hh}
	for _, k := range marketKeys(r.pinned) {
		var pin, frz []verdict.Point
		for i, tk := range r.pinned[k] {
			share := max(tk.Share, 0)
			if tk.Share >= 0 {
				in.Outcomes++
			}
			pin = append(pin, point(int64(i), share))
			frz = append(frz, point(int64(i), b2i(tk.Stuck)))
		}
		in.Pinned, in.Frozen = append(in.Pinned, pin), append(in.Frozen, frz)
	}
	recipes := make([]economysim.RecipeID, 0, len(r.obs.complete))
	for id := range r.obs.complete {
		recipes = append(recipes, id)
	}
	slices.Sort(recipes)
	for _, id := range recipes {
		c := r.obs.complete[id]
		var pts []verdict.Point
		for i := 0; i < min(len(c), 60); i++ {
			pts = append(pts, point(int64(i), c[i]))
		}
		in.Completions = append(in.Completions, pts)
	}
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
