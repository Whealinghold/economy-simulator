package transit

import (
	"cmp"
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// valid reports whether a load and a unit capacity are usable: nothing
// negative and at least one capacity above 0.
func valid(mass, bulk, unitMass, unitBulk int64) bool {
	return min(mass, bulk, unitMass, unitBulk) >= 0 && max(unitMass, unitBulk) > 0
}

// UnitsNeeded returns n = max(1, ceil(mass / unitMass), ceil(bulk / unitBulk)),
// the carrying units a load needs (spec-transport.md 3.2.1); a term whose unit
// capacity is 0 is left out. ok is false when a load is negative, a capacity is
// negative or both capacities are 0.
func UnitsNeeded(mass, bulk, unitMass, unitBulk int64) (n int64, ok bool) {
	if !valid(mass, bulk, unitMass, unitBulk) {
		return 0, false
	}
	n = 1
	if unitMass > 0 {
		n = max(n, ceilDiv(mass, unitMass))
	}
	if unitBulk > 0 {
		n = max(n, ceilDiv(bulk, unitBulk))
	}
	return n, true
}

// Fits reports whether a load of (mass, bulk) is within the pooled capacity of
// units carrying units, (units * unitMass, units * unitBulk) (spec-transport.md
// 3.2.1, spec-goods.md 3.2): both sums within their capacity, a capacity of 0
// leaving its dimension unlimited. ok is false when an argument is negative,
// both capacities are 0, or a pooled capacity leaves int64.
func Fits(mass, bulk, units, unitMass, unitBulk int64) (fits, ok bool) {
	if units < 0 || !valid(mass, bulk, unitMass, unitBulk) {
		return false, false
	}
	m, mok := within(mass, units, unitMass)
	b, bok := within(bulk, units, unitBulk)
	if !mok || !bok {
		return false, false
	}
	return m && b, true
}

// within reports whether load is at most units * capacity, a capacity of 0
// being unlimited; ok is false when the pooled capacity leaves int64.
func within(load, units, capacity int64) (fits, ok bool) {
	if capacity == 0 {
		return true, true
	}
	c, ok := checked.Mul(units, capacity)
	return load <= c, ok
}

// ceilDiv returns ceil(a / b) for a >= 0 and b > 0 without forming a + b.
func ceilDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 {
		q++
	}
	return q
}

// mulCmp compares a * b with c * d as unsigned 128-bit products: -1, 0 or 1.
func mulCmp(a, b, c, d uint64) int {
	h1, l1 := bits.Mul64(a, b)
	h2, l2 := bits.Mul64(c, d)
	return cmp.Or(cmp.Compare(h1, h2), cmp.Compare(l1, l2))
}
