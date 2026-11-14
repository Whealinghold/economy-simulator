package room

// DiscardOrder returns the discards that make room for the tier 1 bids of the
// default strategy (spec-storage.md 6.3.2, FR-STO-024): the shortfall is the
// load of the tier 1 bids above the free room, per dimension, and cands lists
// the commodities that have surplus above Keep', each with its surplus as one
// lot. Whole units go, the least that meets the binding dimension, of the
// commodity with the greatest surplus load in it, ties by the lower commodity
// ID, the ranking taken again after each commodity. A commodity with no
// surplus is never a candidate: nothing is discarded for tier 2 or beyond, or
// out of Keep'. met is false when the surplus cannot meet the shortfall.
func DiscardOrder(shortfall, room Pair, cands []Candidate) (steps []Step, met bool) {
	return order(shortfall, room, cands, true)
}
