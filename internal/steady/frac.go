package steady

import "math/big"

// AtLeast reports whether f is at least num/den, exactly. It is the one
// comparison the root needs of a Frac (the cycle gain against the cost floor,
// spec-viability.md 4.3), so no big value has to leave the package. A den below
// 1 is a violated programmer invariant and panics.
func (f Frac) AtLeast(num, den int64) bool {
	if den < 1 {
		panic("steady: invariant violated: AtLeast with a denominator below 1")
	}
	l := new(big.Int).Mul(f.Num, big.NewInt(den))
	r := new(big.Int).Mul(big.NewInt(num), f.Den)
	return l.Cmp(r) >= 0
}
