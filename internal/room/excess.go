package room

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// ExcessOrder returns the moves that bring an account's load within a new room
// after a role change (spec-storage.md 5.5.2, 5.5.2.1, FR-STO-018): goods go to
// the estate, whole lots oldest first, of the commodity with the greatest load
// in the binding dimension, until the load fits. load is the account's load and
// room the new room; cands lists the commodities held with their lots oldest
// first. met is false only when a dimension is over its room and no held
// commodity has a size in it, which the ledger's load cannot produce.
func ExcessOrder(load, room Pair, cands []Candidate) (steps []Step, met bool) {
	shortfall := Pair{Mass: excessOf(load.Mass, room.Mass), Bulk: excessOf(load.Bulk, room.Bulk)}
	return order(shortfall, room, cands, false)
}

// excessOf is load - room, saturated: a difference that does not fit int64 is
// above any quantity of goods to move, or below any shortfall.
func excessOf(load, room int64) int64 {
	if v, ok := checked.Sub(load, room); ok {
		return v
	}
	if load < 0 {
		return math.MinInt64
	}
	return math.MaxInt64
}
