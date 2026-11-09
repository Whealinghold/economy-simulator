package reduce_test

import (
	"math/big"
	"math/rand/v2"

	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// bigOf converts an Int128 to a big.Int through its decimal text, so a wrong
// String shows in every comparison that uses it.
func bigOf(v reduce.Int128) *big.Int {
	b, ok := new(big.Int).SetString(v.String(), 10)
	if !ok {
		panic("Int128.String is not a decimal number: " + v.String())
	}
	return b
}

// fromBig converts a big.Int that fits 128 bits back, by its two's complement words.
func fromBig(b *big.Int) reduce.Int128 {
	mod := new(big.Int).Lsh(big.NewInt(1), 128)
	u := new(big.Int).Mod(b, mod)
	lo := new(big.Int).And(u, new(big.Int).SetUint64(^uint64(0))).Uint64()
	hi := new(big.Int).Rsh(u, 64).Uint64()
	return reduce.FromWords(int64(hi), lo)
}

var (
	min128 = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 127))
	max128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))
)

func fits(b *big.Int) bool { return b.Cmp(min128) >= 0 && b.Cmp(max128) <= 0 }

// randInt128 draws a 128-bit value of a random width, so small, word-sized and
// full-width operands all occur. The source is seeded by the caller.
func randInt128(r *rand.Rand) reduce.Int128 {
	var hi int64
	lo := r.Uint64() >> (r.UintN(65) % 64)
	switch r.UintN(4) {
	case 0: // one word, either sign
		hi = int64(r.Uint64()>>63) * -1
	case 1: // small high word
		hi = int64(r.UintN(5)) - 2
	default:
		hi = int64(r.Uint64() >> (r.UintN(64)))
		if r.UintN(2) == 0 {
			hi = -hi
		}
	}
	return reduce.FromWords(hi, lo)
}

// refRDiv is n/d rounded half to even, by big.Int, the oracle of RDiv.
func refRDiv(n, d *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(n, d, new(big.Int))
	twice := new(big.Int).Abs(new(big.Int).Lsh(r, 1))
	c := twice.Cmp(new(big.Int).Abs(d))
	away := c > 0 || (c == 0 && q.Bit(0) == 1)
	if away {
		if (n.Sign() < 0) != (d.Sign() < 0) {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	return q
}
