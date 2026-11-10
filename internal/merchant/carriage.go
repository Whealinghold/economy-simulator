package merchant

import "github.com/Whealinghold/economy-sim/internal/checked"

// Feed is one feed entry of a self-carried mode priced at the local market: the
// quantity one carrying unit consumes per tick and the price of a unit.
type Feed struct {
	Quantity, Price int64
}

// Carriage is what the cost of a trip reads (spec-merchants.md 5.4): the carrying
// units n the lot needs and the travel time k, and then either the haul units per
// carrying unit s and the local price of a haul unit (a hired mode) or the wear,
// the vehicle, the feed and the merchant's own cost of a tick (a self-carried
// one). Toll is the money one carrying unit pays on the path with the fiscal
// module on, 0 otherwise.
type Carriage struct {
	// N is the carrying units the lot needs and K the travel time in ticks.
	N, K int64
	// Hired selects the hired cost n * s * HaulPrice; S is the haul units per
	// carrying unit.
	Hired     bool
	S         int64
	HaulPrice int64
	// WearPerTick, VehiclePrice and Durability price the wear of the vehicle; a
	// Durability of 0 means no vehicle and no wear.
	WearPerTick, VehiclePrice, Durability int64
	// Feed is the feed of one carrying unit.
	Feed []Feed
	// UpkeepCost is the cost of one tick of the agent's effective upkeep and
	// TimeCostBP the share of it that a tick away costs, in basis points.
	UpkeepCost, TimeCostBP int64
	// Toll is the money one carrying unit pays on the path.
	Toll int64
}

// Cost returns the cost of carriage C (spec-merchants.md 5.4) and whether it fits.
// Hired: n * s * HaulPrice. Self-carried: wear + feed + time, where
// wear = ceil(n * WearPerTick * k * VehiclePrice / Durability), feed = the sum of
// quantity * n * k * price and time = k * UpkeepCost * TimeCostBP / 10 000
// truncated. Every mode adds n * Toll. The costs round up, so every rounding is
// against the merchant.
func (c Carriage) Cost() (cost int64, ok bool) {
	if c.N < 1 || c.K < 0 {
		return 0, false
	}
	var base int64
	if c.Hired {
		base, ok = c.hiredCost()
	} else {
		base, ok = c.selfCost()
	}
	if !ok {
		return 0, false
	}
	toll, ok := checked.Mul(c.N, c.Toll)
	if !ok {
		return 0, false
	}
	return checked.Add(base, toll)
}

// hiredCost is n * s * HaulPrice.
func (c Carriage) hiredCost() (int64, bool) {
	ns, ok := checked.Mul(c.N, c.S)
	if !ok {
		return 0, false
	}
	return checked.Mul(ns, c.HaulPrice)
}

// selfCost is wear + feed + time.
func (c Carriage) selfCost() (int64, bool) {
	wear, ok := c.wear()
	if !ok {
		return 0, false
	}
	feed, ok := c.feed()
	if !ok {
		return 0, false
	}
	// time = k * UpkeepCost * TimeCostBP / 10 000, truncated: the first product is
	// checked and the second held in 128 bits.
	ku, ok := checked.Mul(c.K, c.UpkeepCost)
	if !ok {
		return 0, false
	}
	t, ok := checked.MulDiv(ku, c.TimeCostBP, wholeBP)
	if !ok {
		return 0, false
	}
	sum, ok := checked.Add(wear, feed)
	if !ok {
		return 0, false
	}
	return checked.Add(sum, t)
}

// wear is ceil(n * WearPerTick * k * VehiclePrice / Durability), 0 with no vehicle.
func (c Carriage) wear() (int64, bool) {
	if c.Durability == 0 {
		return 0, true
	}
	nw, ok := checked.Mul(c.N, c.WearPerTick)
	if !ok {
		return 0, false
	}
	nwk, ok := checked.Mul(nw, c.K)
	if !ok {
		return 0, false
	}
	return checked.MulDivCeil(nwk, c.VehiclePrice, c.Durability)
}

// feed is the sum over the entries of quantity * n * k * price.
func (c Carriage) feed() (int64, bool) {
	nk, ok := checked.Mul(c.N, c.K)
	if !ok {
		return 0, false
	}
	var sum int64
	for _, f := range c.Feed {
		qn, ok := checked.Mul(f.Quantity, nk)
		if !ok {
			return 0, false
		}
		p, ok := checked.Mul(qn, f.Price)
		if !ok {
			return 0, false
		}
		if sum, ok = checked.Add(sum, p); !ok {
			return 0, false
		}
	}
	return sum, true
}
