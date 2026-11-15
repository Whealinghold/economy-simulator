package technology

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// Key derives the module's key from the effective seed and the site constant
// of the module: the high word of the stream key of (site, tick 0, entity 0)
// (spec-technology.md 4.4.1, ADR 0008). It draws nothing and is never saved: a
// reseed changes it by changing the seed.
func Key(seed, site uint64) uint64 {
	hi, _ := ordering.StreamKey(seed, site, 0, 0)
	return hi
}

// Hit reports whether the cell with index cell of cells discovers at tick
// (spec-technology.md 4.4.2 and 4.4.3, FR-TEC-009): the high word of
// K(t, c) * (10 000 * ticksPerYear) is below discoveryBP, with K(t, c) =
// Finalise(key + (t*cells + c + 1) * Increment) modulo 2^64. The probability
// in a tick is discoveryBP / (10 000 * ticksPerYear). It is false for a
// discoveryBP of 0 or below, a ticksPerYear below 1, a negative tick or cell
// index, or a cell index outside cells. The wrap of the modular product is the
// hash, not an overflow.
func Hit(key uint64, tick, cell, cells, discoveryBP, ticksPerYear int64) bool {
	if discoveryBP <= 0 || ticksPerYear < 1 || tick < 0 || cell < 0 || cell >= cells {
		return false
	}
	span := uint64(Basis) * uint64(ticksPerYear)
	if span/uint64(Basis) != uint64(ticksPerYear) {
		return false
	}
	idx := uint64(tick)*uint64(cells) + uint64(cell) + 1
	k := ordering.Finalise(key + idx*ordering.Increment)
	hi, _ := bits.Mul64(k, span)
	return hi < uint64(discoveryBP)
}
