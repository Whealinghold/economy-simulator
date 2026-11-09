package merchant

import "github.com/Whealinghold/economy-sim/internal/ordering"

// FeeShare returns the fee share phi(a) of the agent with the given id, in basis
// points (spec-merchants.md 5.7.1): the base share plus the spread times the
// agent's place in [-1, 1],
//
//	u   = floor(Weyl32(id) * 10 000 / 2^32)             in [0, 10 000)
//	phi = clamp(base + spread * (2u - 10 000) / 10 000, 1, 10 000)
//
// with the division truncated toward zero. The Weyl offset wraps modulo 2^32 by
// design (it is a hash of an identifier, not a quantity); nothing else here
// wraps. The product spread * (2u - 10 000) fits in int64 for any spread of at
// most 5 000.
func FeeShare(base, spread, id int64) int64 {
	u := int64(uint64(ordering.Weyl32(id)) * wholeBP >> 32)
	phi := base + spread*(2*u-wholeBP)/wholeBP
	return min(max(phi, 1), wholeBP)
}
