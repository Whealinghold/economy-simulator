package room

import "github.com/Whealinghold/economy-sim/internal/checked"

// Pair is a mass and a bulk, the two dimensions of a room, a load or a size
// (spec-goods.md 3.2). Neither is compared with the other.
type Pair struct {
	Mass, Bulk int64
}

// Of returns the room of a limited account of the given weight: weight times
// the base room per household plus the capacity of each store, per dimension,
// with checked arithmetic. A store's capacity is not multiplied by the weight
// (spec-storage.md 3.3.2, FR-STO-004). ok is false when a dimension overflows
// int64.
func Of(weight int64, base Pair, stores []Pair) (r Pair, ok bool) {
	var m, b bool
	r.Mass, m = checked.Mul(weight, base.Mass)
	r.Bulk, b = checked.Mul(weight, base.Bulk)
	ok = m && b
	for _, s := range stores {
		var okm, okb bool
		r.Mass, okm = checked.Add(r.Mass, s.Mass)
		r.Bulk, okb = checked.Add(r.Bulk, s.Bulk)
		ok = ok && okm && okb
	}
	return r, ok
}
