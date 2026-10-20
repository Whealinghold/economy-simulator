package steady

import "math/big"

// This file is the small exact arithmetic the root needs of a Frac to build the
// warm-start table (spec-viability.md 8): a Frac is opaque to the root, which
// combines fractions with these methods and leaves them only through Floor,
// Ceil and Round, the checked boundary of round.go. Every method returns a new
// Frac and changes nothing.

// Rat returns num/den as a Frac in lowest terms. A den below 1 is a violated
// programmer invariant and panics.
func Rat(num, den int64) Frac {
	if den < 1 {
		panic("steady: invariant violated: Rat with a denominator below 1")
	}
	return fromRat(big.NewRat(num, den))
}

// fromRat copies r into a Frac.
func fromRat(r *big.Rat) Frac {
	return Frac{Num: new(big.Int).Set(r.Num()), Den: new(big.Int).Set(r.Denom())}
}

// Mul returns f * num/den. A den below 1 panics (a programmer invariant).
func (f Frac) Mul(num, den int64) Frac {
	if den < 1 {
		panic("steady: invariant violated: Mul with a denominator below 1")
	}
	return fromRat(new(big.Rat).Mul(frat(f), big.NewRat(num, den)))
}

// Times returns f * g.
func (f Frac) Times(g Frac) Frac { return fromRat(new(big.Rat).Mul(frat(f), frat(g))) }

// Add returns f + g.
func (f Frac) Add(g Frac) Frac { return fromRat(new(big.Rat).Add(frat(f), frat(g))) }

// Quo returns f / g and true, or the zero fraction and false when g is 0.
func (f Frac) Quo(g Frac) (Frac, bool) {
	if g.Num.Sign() == 0 {
		return Rat(0, 1), false
	}
	return fromRat(new(big.Rat).Quo(frat(f), frat(g))), true
}

// Cmp compares f with g: -1, 0 or +1.
func (f Frac) Cmp(g Frac) int { return frat(f).Cmp(frat(g)) }

// Sign returns -1, 0 or +1 for the sign of f.
func (f Frac) Sign() int { return f.Num.Sign() }

// Abs returns the absolute value of f.
func (f Frac) Abs() Frac { return fromRat(new(big.Rat).Abs(frat(f))) }

// Floor returns f rounded toward negative infinity, under the bound rule of the
// package function Floor.
func (f Frac) Floor(lo, hi int64) (int64, bool) { return Floor(f.Num, f.Den, lo, hi) }

// Ceil returns f rounded toward positive infinity, under the bound rule of the
// package function Ceil.
func (f Frac) Ceil(lo, hi int64) (int64, bool) { return Ceil(f.Num, f.Den, lo, hi) }

// Round returns f rounded to the nearest integer, an exact half going to the
// even one, under the bound rule of RoundHalfEven.
func (f Frac) Round(lo, hi int64) (int64, bool) { return RoundHalfEven(f.Num, f.Den, lo, hi) }

// Apportion shares total whole units among weights by the largest-remainder
// method (architecture.md 7.6): quotas total * w / sum, floors first, the
// units left to the largest remainders, a tie to the earlier index. All the
// weights 0 gives all 0. A negative total or weight is a violated programmer
// invariant and panics.
func Apportion(total int64, weights []Frac) []int64 {
	if total < 0 {
		panic("steady: invariant violated: Apportion of a negative total")
	}
	sum := new(big.Rat)
	for _, w := range weights {
		if w.Sign() < 0 {
			panic("steady: invariant violated: Apportion with a negative weight")
		}
		sum.Add(sum, frat(w))
	}
	out := make([]int64, len(weights))
	if sum.Sign() == 0 {
		return out
	}
	rem := make([]*big.Rat, len(weights))
	var given int64
	for i, w := range weights {
		q := new(big.Rat).Mul(frat(w), big.NewRat(total, 1))
		q.Quo(q, sum)
		fl := new(big.Int).Quo(q.Num(), q.Denom()) // non-negative, so truncation is the floor
		out[i] = fl.Int64()                        // at most total
		given += out[i]
		rem[i] = q.Sub(q, new(big.Rat).SetInt(fl))
	}
	for left := total - given; left > 0; left-- {
		best := 0
		for i := 1; i < len(rem); i++ {
			if rem[i].Cmp(rem[best]) > 0 { // strict: a tie keeps the earlier index
				best = i
			}
		}
		out[best]++
		rem[best] = new(big.Rat).SetInt64(-1) // taken: below every remainder
	}
	return out
}

// Uses returns the units of commodity c one execution consumes: the input
// quantities plus Wear/Durability for each tool of that commodity
// (spec-viability.md 8.4.1, the whole consumption). A Durability below 1 is a
// violated programmer invariant and panics.
func (a Activity) Uses(c int) Frac {
	sum := new(big.Rat)
	for _, in := range a.Inputs {
		if in.Commodity == c {
			sum.Add(sum, big.NewRat(in.Qty, 1))
		}
	}
	for _, tl := range a.Tools {
		if tl.Commodity != c {
			continue
		}
		if tl.Durability < 1 {
			panic("steady: invariant violated: tool durability below 1")
		}
		sum.Add(sum, big.NewRat(tl.Wear, tl.Durability))
	}
	return fromRat(sum)
}

// Yield returns the units of commodity c one execution outputs at the mean
// yield factor: FactorSum * q / (10 000 * Starts) for each non-discard output of
// it. Starts below 1 is a violated programmer invariant and panics.
func (a Activity) Yield(c int) Frac {
	if a.Starts < 1 {
		panic("steady: invariant violated: activity with no start")
	}
	var q int64
	for _, o := range a.Outputs {
		if o.Commodity == c && !o.Discard {
			q += o.Qty
		}
	}
	num := new(big.Int).Mul(big.NewInt(a.FactorSum), big.NewInt(q))
	return fromRat(new(big.Rat).SetFrac(num, new(big.Int).Mul(big.NewInt(10000), big.NewInt(a.Starts))))
}

// ValueAdded returns the value of one execution at the given prices: the sum
// over the net vector of price * net, in minor units. A commodity beyond prices
// is priced 0.
func (a Activity) ValueAdded(prices []int64) Frac {
	sum := new(big.Rat)
	for _, n := range a.Net() {
		if n.Commodity < len(prices) {
			sum.Add(sum, new(big.Rat).Mul(n.Val, big.NewRat(prices[n.Commodity], 1)))
		}
	}
	return fromRat(sum)
}
