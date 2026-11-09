package indicator

import (
	economysim "github.com/Whealinghold/economy-sim"
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
	if regions < 0 || commodities < 0 || len(cells) != regions*commodities || to < from {
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
	if r < 0 || int(r) >= p.Regions || c < 0 || int(c) >= p.Commodities || len(p.Cells) != p.Regions*p.Commodities {
		return 0
	}
	return p.Cells[int(r)*p.Commodities+int(c)]
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
	return Prices{ConfigHash: economysim.HashConfig(cfg), Regions: regions, Commodities: commodities, Cells: cells}
}
