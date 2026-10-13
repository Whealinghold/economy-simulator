package economysim

import (
	"strconv"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Money is an amount of money: an int64 count of the smallest currency unit
// (a minor unit). The library has no notion of a currency or of decimal places.
type Money int64

// Price is money per unit of a commodity: an int64 count of minor units.
type Price int64

// Quantity is a count of whole units of a commodity.
type Quantity int64

// CommodityID identifies a commodity: its index in the configuration's
// commodities in ascending order of ID string. It is not stable across
// configurations; use the string ID to persist or exchange a commodity.
type CommodityID int

// RoleID identifies a role: its index in the configuration's roles in
// ascending order of ID string. It is not stable across configurations.
type RoleID int

// RecipeID identifies a recipe: its index in the configuration's recipes in
// ascending order of ID string. It is not stable across configurations.
type RecipeID int

// RegionID is the index of a region in ascending order of its ID string. It is
// not stable across configurations: adding a region can renumber the others.
// Use the string ID to persist or exchange a region.
type RegionID int

// AgentID identifies an agent. IDs are assigned in order of creation and are
// never reused within a world.
type AgentID int64

// NoCommodity means "no commodity"; where a field names a commodity it stands
// for money.
const NoCommodity CommodityID = -1

// NoRegion means "no region". A query returns it where a region does not
// exist, and it names the region of an account that is between regions.
const NoRegion RegionID = -1

// NoAgent means "no agent". It is below every agent ID and every RegionID.
const NoAgent AgentID = -1

// Add returns m + o and whether the sum fits in int64. When it does not, the
// result is 0 and the caller treats the operation as one that cannot be
// applied.
func (m Money) Add(o Money) (Money, bool) {
	s, ok := checked.Add(int64(m), int64(o))
	return Money(s), ok
}

// Sub returns m - o and whether the difference fits in int64. When it does
// not, the result is 0.
func (m Money) Sub(o Money) (Money, bool) {
	d, ok := checked.Sub(int64(m), int64(o))
	return Money(d), ok
}

// String returns the amount as a decimal count of minor units, with a leading
// minus sign when negative. It names no denomination or symbol.
func (m Money) String() string { return strconv.FormatInt(int64(m), 10) }

// Times returns the money value of q units at price p and whether the product
// fits in int64. When it does not, the result is 0.
func (p Price) Times(q Quantity) (Money, bool) {
	v, ok := checked.Mul(int64(p), int64(q))
	return Money(v), ok
}

// String returns the price as a decimal count of minor units.
func (p Price) String() string { return strconv.FormatInt(int64(p), 10) }

// Add returns q + o and whether the sum fits in int64. When it does not, the
// result is 0.
func (q Quantity) Add(o Quantity) (Quantity, bool) {
	s, ok := checked.Add(int64(q), int64(o))
	return Quantity(s), ok
}

// Sub returns q - o and whether the difference fits in int64. When it does
// not, the result is 0.
func (q Quantity) Sub(o Quantity) (Quantity, bool) {
	d, ok := checked.Sub(int64(q), int64(o))
	return Quantity(d), ok
}

// String returns the quantity as a decimal count of units.
func (q Quantity) String() string { return strconv.FormatInt(int64(q), 10) }
