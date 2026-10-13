// Package ordering holds the deterministic spreads and mixes of the library:
// the SplitMix64 finaliser, the Weyl values and the rotation bucket
// (spec-world.md 7.1.2 and 9.1.1). Nothing here draws a random number or keeps
// state; each function is a pure function of its arguments.
package ordering

import "math/bits"

// Increment is the SplitMix64 increment, the odd constant (2^64 divided by the
// golden ratio) that the seeding rule of spec-world.md 7.1.2 adds before each
// finalisation.
const Increment uint64 = 0x9E3779B97F4A7C15

const (
	finaliseMul1 uint64 = 0xBF58476D1CE4E5B9
	finaliseMul2 uint64 = 0x94D049BB133111EB

	weyl32Mul uint32 = 2654435769 // 0x9E3779B9, the golden ratio times 2^32
	weyl2Mul  uint32 = 0x6A09E667 // the fractional part of sqrt(2) times 2^32
)

// Finalise is the SplitMix64 output function, a bijection of uint64:
//
//	z = (z xor (z >> 30)) * 0xBF58476D1CE4E5B9
//	z = (z xor (z >> 27)) * 0x94D049BB133111EB
//	return z xor (z >> 31)
//
// All products are modulo 2^64.
func Finalise(z uint64) uint64 {
	z = (z ^ (z >> 30)) * finaliseMul1
	z = (z ^ (z >> 27)) * finaliseMul2
	return z ^ (z >> 31)
}

// Weyl32 returns (id * 2654435769) mod 2^32, the fixed per-agent spread for
// carries and margins (spec-world.md 9.1.1). A negative id is taken modulo
// 2^32 like any other.
func Weyl32(id int64) uint32 {
	return uint32(id) * weyl32Mul
}

// Weyl2 returns (id * 0x6A09E667) mod 2^32, the second Kronecker sequence that
// feeds Bucket. A rotation must use Weyl2 and not Weyl32, so that two spreads
// acting on one decision do not share a value (spec-world.md 9.1.1).
func Weyl2(id int64) uint32 {
	return uint32(id) * weyl2Mul
}

// Bucket returns (Weyl2(id) * p) >> 32, a value in [0, p) for every id and
// every p of at least 1. An agent acts on the ticks where
// (tick + Bucket(id, p)) mod p is 0. A p below 1 has no bucket and returns 0.
func Bucket(id int64, p int64) int64 {
	if p < 1 {
		return 0
	}
	// The 96-bit product (a 32-bit Weyl2 by a 63-bit p) shifted right by 32.
	hi, lo := bits.Mul64(uint64(Weyl2(id)), uint64(p))
	return int64(hi<<32 | lo>>32)
}

// Key returns K(t, id) = Finalise(t*Increment + id + Increment) modulo 2^64,
// the stateless key of spec-world.md 7.1.7 and spec-resources.md 5.3.3. A
// signed value enters as its two's-complement uint64. It draws nothing and
// keeps no state.
func Key(t, id int64) uint64 {
	return Finalise(uint64(t)*Increment + uint64(id) + Increment)
}

// StreamKey returns the PCG state (hi, lo) of the stream named by the seed and
// the words (site, tick, id), spec-world.md 7.1.3: s = seed; for each word w,
// s = Finalise(s + w + Increment); hi = Finalise(s + Increment);
// lo = Finalise(s + Increment + Increment). A signed word enters as its
// two's-complement uint64. The caller passes (hi, lo) to PCG.Seed.
func StreamKey(seed, site uint64, tick, id int64) (hi, lo uint64) {
	s := Finalise(seed + site + Increment)
	s = Finalise(s + uint64(tick) + Increment)
	s = Finalise(s + uint64(id) + Increment)
	return streamState(s)
}

// StreamKeySub is StreamKey for a row that names a pair: the fourth word sub
// (spec-world.md 7.1.3) is folded after id.
func StreamKeySub(seed, site uint64, tick, id, sub int64) (hi, lo uint64) {
	s := Finalise(seed + site + Increment)
	s = Finalise(s + uint64(tick) + Increment)
	s = Finalise(s + uint64(id) + Increment)
	s = Finalise(s + uint64(sub) + Increment)
	return streamState(s)
}

func streamState(s uint64) (hi, lo uint64) {
	return Finalise(s + Increment), Finalise(s + Increment + Increment)
}
