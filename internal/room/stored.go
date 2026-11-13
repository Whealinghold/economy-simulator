package room

import "github.com/Whealinghold/economy-sim/internal/checked"

// StoredLoad returns the part of an account's load that sits above its base
// room, up to the capacity of its stores, per dimension (spec-storage.md 8.3.2,
// FR-STO-042):
//
//	min(storesCap, max(0, load - weight * base))
//
// a read with no attribution to commodities. A base room times the weight that
// overflows int64 is above any load, so the stored load is 0 in that
// dimension.
func StoredLoad(storesCap, load, base Pair, weight int64) Pair {
	return Pair{
		Mass: storedDim(storesCap.Mass, load.Mass, base.Mass, weight),
		Bulk: storedDim(storesCap.Bulk, load.Bulk, base.Bulk, weight),
	}
}

func storedDim(storesCap, load, base, weight int64) int64 {
	own, ok := checked.Mul(weight, base)
	if !ok {
		return 0
	}
	above, ok := checked.Sub(load, own)
	if !ok {
		// Only a negative own (a weight or base below 0, outside the
		// contract) can overflow the difference: the load is above any cap.
		return storesCap
	}
	return min(storesCap, max(0, above))
}
