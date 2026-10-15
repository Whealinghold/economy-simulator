package economysim

import "github.com/Whealinghold/economy-sim/internal/checked"

// This file holds the rules of the goods that are not validation: the load of
// a set of goods, whether it fits a container, and the chargeable size and
// capacity (spec-goods.md 3.2 to 3.4). The structs are in config.go. Every
// function is pure, takes primitive values and reads nothing of a world.
//
// FR-GDS-005 to 007 are rules of the row; no module of version 1 prices by the
// chargeable size (spec-goods.md 3.3, 3.5), so chargeableSize and
// chargeableCapacity are read only by their tests.

// load returns the load of a set of goods: the sum of quantity times Mass and
// the sum of quantity times Bulk, qty indexed by CommodityID (spec-goods.md
// 3.2). ok is false when a product or a sum overflows int64, never a wrapped
// number. A quantity past the last commodity of c is a programmer error.
func load(c *Config, qty []Quantity) (mass, bulk int64, ok bool) {
	if len(qty) > len(c.Commodities) {
		panic("economysim: invariant violated: load of a quantity list longer than the commodities")
	}
	for i, q := range qty {
		cm := &c.Commodities[i]
		m, ok1 := checked.Mul(int64(q), cm.Mass)
		b, ok2 := checked.Mul(int64(q), cm.Bulk)
		if !ok1 || !ok2 {
			return 0, 0, false
		}
		if mass, ok1 = checked.Add(mass, m); !ok1 {
			return 0, 0, false
		}
		if bulk, ok2 = checked.Add(bulk, b); !ok2 {
			return 0, 0, false
		}
	}
	return mass, bulk, true
}

// fits reports whether a load is within a container: both sums at most their
// capacity (FR-GDS-005). It does not say which dimension binds; transport and
// storage define that themselves.
func fits(mass, bulk, capMass, capBulk int64) bool {
	return mass <= capMass && bulk <= capBulk
}

// chargeableSize returns the chargeable size of qty units of a commodity:
// qty times max(Mass, ceil(Bulk * bulkFactor / 10 000)) (spec-goods.md 3.3,
// FR-GDS-006). The per-unit bulk term is rounded up, toward positive infinity.
// ok is false on overflow.
func chargeableSize(c *Config, id CommodityID, qty Quantity, bulkFactor int64) (size int64, ok bool) {
	cm := &c.Commodities[id]
	charge, ok := checked.MulDivCeil(cm.Bulk, bulkFactor, wholeBP)
	if !ok {
		return 0, false
	}
	charge = max(cm.Mass, charge)
	return checked.Mul(int64(qty), charge)
}

// chargeableCapacity returns min(capMass, floor(capBulk * bulkFactor /
// 10 000)) (spec-goods.md 3.4, FR-GDS-007). The bulk term is rounded down,
// toward negative infinity, so a load of one commodity whose chargeable size is
// within it always fits. ok is false on overflow.
func chargeableCapacity(capMass, capBulk, bulkFactor int64) (int64, bool) {
	b, ok := checked.MulDivFloor(capBulk, bulkFactor, wholeBP)
	if !ok {
		return 0, false
	}
	return min(capMass, b), true
}
