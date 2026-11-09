package indicator

import (
	"fmt"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// Prices is a table of a price per (region, commodity), held region by region
// (spec-indicators.md 4.3). A price is always an argument of a measure, never
// read from the world, so that GDP at base prices and at today's are two calls.
//
// It carries the configuration hash and the window it was taken from, so a
// table from another world is ErrConfigMismatch. A table of a snapshot has
// From == To == the snapshot's tick; one of a reference price has both 0.
// A table built from a world has no zero price; a consumer's own may hold one,
// and the value of a quantity at it is 0 (spec-indicators.md 4.3.7). The zero
// value is an empty table whose At is 0.
type Prices struct {
	ConfigHash           [32]byte
	From, To             int64
	Regions, Commodities int
	// Cells has Regions * Commodities prices: the price of commodity c in region
	// r is Cells[r*Commodities+c]. A cell of an outside node, which has no
	// market, is 0.
	Cells []economysim.Price
}

// NewPrices builds a table from cells in region-by-commodity order and checks
// its shape: ErrRange for a negative count, a price below 0, or a length that is
// not regions * commodities. The cells are copied.
func NewPrices(hash [32]byte, from, to int64, regions, commodities int, cells []economysim.Price) (Prices, error) {
	if regions < 0 || commodities < 0 || len(cells) != regions*commodities || (regions != 0 && len(cells)/regions != commodities) || to < from {
		return Prices{}, wrap("NewPrices", "shape", ErrRange)
	}
	for _, c := range cells {
		if c < 0 {
			return Prices{}, wrap("NewPrices", "price", ErrRange)
		}
	}
	return Prices{ConfigHash: hash, From: from, To: to, Regions: regions, Commodities: commodities, Cells: append([]economysim.Price(nil), cells...)}, nil
}

// At returns the price of commodity c in region r, and 0 for a pair outside the
// table.
func (p Prices) At(r economysim.RegionID, c economysim.CommodityID) economysim.Price {
	if !p.has(r, c) {
		return 0
	}
	return p.Cells[int(r)*p.Commodities+int(c)]
}

// has reports whether the cell is inside the table.
func (p Prices) has(r economysim.RegionID, c economysim.CommodityID) bool {
	return r >= 0 && int(r) < p.Regions && c >= 0 && int(c) < p.Commodities && len(p.Cells) == p.Regions*p.Commodities
}

// Compatible returns ErrConfigMismatch when the table was not taken from a world
// of the given configuration hash. A measure that takes a table calls it first.
func (p Prices) Compatible(hash [32]byte) error {
	if p.ConfigHash != hash {
		return wrap("Prices", "config_hash", ErrConfigMismatch)
	}
	return nil
}

// PricesOfSnapshot returns the published price of each market at a boundary,
// MarketRow.State.Price, which is the price of the last step
// (spec-indicators.md 4.3.3). A (region, commodity) with no market row is 0.
// err is ErrRange for a row whose region or commodity is outside the names
// table.
func PricesOfSnapshot(s economysim.Snapshot) (Prices, error) {
	regions, commodities := len(s.Regions), len(s.Names.Commodities)
	cells := make([]economysim.Price, regions*commodities)
	for _, m := range s.Markets {
		r, c := m.State.Region, m.State.Commodity
		if r < 0 || int(r) >= regions || c < 0 || int(c) >= commodities {
			return Prices{}, wrap("PricesOfSnapshot", "market", ErrRange)
		}
		cells[int(r)*commodities+int(c)] = m.State.Price
	}
	return Prices{ConfigHash: s.ConfigHash, From: s.Tick, To: s.Tick, Regions: regions, Commodities: commodities, Cells: cells}, nil
}

// PricesOfReference returns the effective reference price of each commodity in
// each region of a configuration as World.Config returns it: the commodity's
// ReferencePrice, or the region's override in RegionPrices where it has one
// (spec-indicators.md 4.3.4). It is the price of a commodity that never trades.
func PricesOfReference(cfg economysim.Config) Prices {
	return pricesOfReference(cfg, economysim.HashConfig(cfg))
}

// pricesOfReference is PricesOfReference for a caller that has the hash.
func pricesOfReference(cfg economysim.Config, hash [32]byte) Prices {
	regions, commodities := len(cfg.Regions), len(cfg.Commodities)
	override := make(map[[2]string]economysim.Price, len(cfg.RegionPrices)) // looked up, never iterated
	for _, rp := range cfg.RegionPrices {
		override[[2]string{rp.Region, rp.Commodity}] = rp.Price
	}
	cells := make([]economysim.Price, regions*commodities)
	for r, region := range cfg.Regions {
		for c, cm := range cfg.Commodities {
			p, ok := override[[2]string{region.ID, cm.ID}]
			if !ok {
				p = cm.ReferencePrice
			}
			cells[r*commodities+c] = p
		}
	}
	return Prices{ConfigHash: hash, Regions: regions, Commodities: commodities, Cells: cells}
}

// PricesOfWindow returns the unit value of each market over a window: the
// volume-weighted mean of the published prices of its ticks, rdiv(SumPV, SumV)
// half to even, and where no unit traded the last published price carried
// forward (spec-indicators.md 4.3.2, FR-IND-028). It carries the window's
// configuration hash and ticks. A (region, commodity) the window saw no
// clearing for has the price 0. err is ErrOverflow for a value past int64 and
// ErrRange for a negative volume sum, a negative price or a market table that is not Regions * Commodities long.
func PricesOfWindow(w Window) (Prices, error) {
	if w.Regions < 0 || w.Commodities < 0 || len(w.Markets) != w.Regions*w.Commodities { // a table of another shape would read as 0 everywhere
		return Prices{}, wrap("PricesOfWindow", "markets", ErrRange)
	}
	cells := make([]economysim.Price, len(w.Markets))
	for i, m := range w.Markets {
		p, err := m.unitValue()
		if err != nil {
			return Prices{}, err
		}
		cells[i] = p
	}
	return Prices{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Regions: w.Regions, Commodities: w.Commodities, Cells: cells}, nil
}

// unitValue is the unit value of one market: the volume-weighted mean of its prices, or the carried price.
func (m MarketTally) unitValue() (economysim.Price, error) {
	if m.SumV == 0 {
		if m.Price < 0 {
			return 0, wrap("PricesOfWindow", "price", ErrRange)
		}
		return m.Price, nil
	}
	if m.SumV < 0 { // a hand-built Window; a Tally never keeps a negative volume sum
		return 0, wrap("PricesOfWindow", "volume", ErrRange)
	}
	q, ok := measure.RoundDivU(m.SumPV.u(), measure.Uint128{Lo: uint64(m.SumV)})
	wide, small := measure.FromUint128(q)
	v, fits := measure.ToInt64(wide)
	if !ok || !small || !fits {
		return 0, wrap("PricesOfWindow", "price", ErrOverflow)
	}
	return economysim.Price(v), nil
}

// Weights selects the quantities an index is weighted by (spec-indicators.md
// 5.3.2). The set is closed. A fixed basket (the basket of a region) is not a
// choice of this type: BasketLevel takes it from the configuration.
type Weights int

// The weights of an index.
const (
	// WeightsTraded is the units traded, SumV of the window's markets.
	WeightsTraded Weights = iota + 1
	// WeightsConsumed is the units consumed: consumption plus want_consumption.
	WeightsConsumed
	// WeightsProduced is the units made: recipe_output.
	WeightsProduced
)

// String returns the name of the weights.
func (k Weights) String() string {
	switch k {
	case WeightsTraded:
		return "traded"
	case WeightsConsumed:
		return "consumed"
	case WeightsProduced:
		return "produced"
	}
	return fmt.Sprintf("weights_%d", int(k))
}

// Base is the prices and the quantities of a base window, frozen from its closed
// tally: Prices is the table (the constant prices of real measures) and each of
// Traded, Consumed and Produced the units by (region, commodity), at region *
// Commodities + commodity, for one choice of Weights (spec-indicators.md 4.3.5,
// 10.4). It is a value the consumer stores.
type Base struct {
	Prices                     Prices
	Traded, Consumed, Produced []int64
}

// Quantities returns the base quantities for a choice of weights, or nil.
func (b Base) Quantities(k Weights) []int64 {
	switch k {
	case WeightsTraded:
		return b.Traded
	case WeightsConsumed:
		return b.Consumed
	case WeightsProduced:
		return b.Produced
	}
	return nil
}

// BaseOf freezes the prices p and the quantities of a closed window as a base. p
// must be a table of the window's world and shape (ErrConfigMismatch, ErrRange);
// it is usually PricesOfWindow(w) (spec-indicators.md 4.3.5).
func BaseOf(w Window, p Prices) (Base, error) {
	if err := w.checkTable("BaseOf", p); err != nil {
		return Base{}, err
	}
	f, err := flowsOf(w)
	if err != nil {
		return Base{}, err
	}
	b := Base{Prices: p, Traded: make([]int64, len(w.Markets))}
	for i, m := range w.Markets {
		b.Traded[i] = m.SumV
	}
	if b.Consumed, err = consumedUnits(f); err != nil {
		return Base{}, err
	}
	if b.Produced, err = f.units(economysim.ReasonRecipeOutput); err != nil {
		return Base{}, err
	}
	return b, nil
}

// consumedUnits is consumption plus want_consumption by (region, commodity).
func consumedUnits(f flowSet) ([]int64, error) {
	cons, err := f.units(economysim.ReasonConsumption)
	if err != nil {
		return nil, err
	}
	want, err := f.units(economysim.ReasonWantConsumption)
	if err != nil {
		return nil, err
	}
	for i := range cons {
		sum, ok := measure.FromInt64(cons[i]).Add(measure.FromInt64(want[i]))
		v, fits := measure.ToInt64(sum)
		if !ok || !fits {
			return nil, wrap("BaseOf", "consumed", ErrOverflow)
		}
		cons[i] = v
	}
	return cons, nil
}
