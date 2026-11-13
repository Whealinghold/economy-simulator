package room

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Denominator is D of spec-storage.md 5.3.2: yields are in basis points, so a
// quantity times a yield is divided by it. A rule constant (architecture.md
// 9.6), not a tunable.
const Denominator int64 = 10_000

// Item is a quantity of a commodity and the size of one unit of it.
type Item struct {
	Qty  int64
	Size Pair
}

// Output is an output line of a recipe: the quantity per execution and the
// size of a unit, the executor's residue of the output (spec-production.md
// 6.6) and whether it is a discard output, which is created and disposed at
// the completion and takes no room (spec-storage.md 5.3.2).
type Output struct {
	Item
	Residue int64
	Discard bool
}

// KRoom returns the largest number of executions k, at most k0, whose pending
// output fits the room that the start leaves free (spec-storage.md 5.3.2,
// FR-STO-012). free and upkeep are the principal's free room and upkeep room,
// and yieldBP is E, the effective yield in basis points. Per dimension, with
// I the load of the inputs of one execution, A the load of the outputs times D
// and c = A - D*I, R = D*(free - upkeep) - the sum of size * r' (r' is the
// residue, or 0 for an output whose quantity times E is a multiple of D), the
// output of k executions fits once the inputs are taken if k*c <= R in both
// dimensions. The inequality is linear in k, so the answer is found without a
// search: an upper bound floor(R/c) where c > 0, a lower bound ceil(-R/-c)
// where c < 0 and R < 0 (an input larger than the output frees room as it
// goes), and k = 0 when the bounds do not meet.
//
// The bound is sound and conservative by less than the load of a unit of each
// output. An input that overflows int64 makes the answer 0: no execution
// fits, which is the safe side.
func KRoom(inputs []Item, outputs []Output, yieldBP int64, free, upkeep Pair, k0 int64) int64 {
	if k0 < 1 {
		return 0
	}
	rc := recipe{in: inputs, out: outputs, yieldBP: yieldBP}
	lo, hi := int64(0), k0
	for _, d := range [2]func(Pair) int64{pairMass, pairBulk} {
		l, h, ok := rc.bound(d, d(free)-d(upkeep))
		if !ok {
			return 0
		}
		lo, hi = max(lo, l), min(hi, h)
	}
	if lo > hi {
		return 0
	}
	return hi
}

func pairMass(p Pair) int64 { return p.Mass }
func pairBulk(p Pair) int64 { return p.Bulk }

// recipe is the lines of one recipe and its effective yield.
type recipe struct {
	in      []Item
	out     []Output
	yieldBP int64
}

// remainder is R of 5.3.2 written as D*f2 + add with 0 < add <= D, so that a
// division has a non-negative numerator whenever R is not negative.
type remainder struct {
	f2, add, s0 int64
}

// remainderOf returns R = D*free - s, with free the free room less the upkeep
// room and s the residue term.
func remainderOf(free, s int64) remainder {
	return remainder{f2: free - s/Denominator - 1, add: Denominator - s%Denominator, s0: s % Denominator}
}

// negative reports R < 0.
func (r remainder) negative() bool {
	return r.f2 < -1 || (r.f2 == -1 && r.add != Denominator)
}

// bound is the range of k that fits in one dimension: lo is the lowest k (0 for
// none) and hi the highest (the saturated maximum for no bound). free is the
// free room less the upkeep room in that dimension.
func (rc recipe) bound(dim func(Pair) int64, free int64) (lo, hi int64, ok bool) {
	c, s, ok := rc.coefficients(dim)
	if !ok {
		return 0, 0, false
	}
	r := remainderOf(free, s)
	if c >= 0 {
		hi, ok = r.upper(c)
		return 0, hi, ok
	}
	return r.lower(-c)
}

// upper is the highest k with k*c <= R for c >= 0: none of 1 or more where
// R < 0, or R = 0 with c > 0; any where c = 0.
func (r remainder) upper(c int64) (int64, bool) {
	switch {
	case r.negative(), c > 0 && r.f2 < 0:
		return 0, true
	case c == 0:
		return math.MaxInt64, true
	}
	q, _, ok := checked.MulAddDivSat(r.f2, Denominator, r.add, c)
	return q, ok
}

// lower returns the range of k with k*c <= R for c = -absC < 0: every k where
// R >= 0, else k >= ceil(|R| / absC), with |R| = D*m + s0.
func (r remainder) lower(absC int64) (lo, hi int64, ok bool) {
	if !r.negative() {
		return 0, math.MaxInt64, true
	}
	// absC is at least 1 and at most MaxInt64: c is not below -MaxInt64 (A is
	// not negative and D*I fits).
	q, rest, ok := checked.MulAddDivSat(-r.f2-1, Denominator, r.s0, absC)
	if rest > 0 && q < math.MaxInt64 {
		q++
	}
	return q, math.MaxInt64, ok
}

// coefficients returns c = A - D*I and s, the sum over the outputs that carry
// a remainder of size * residue, in one dimension. ok is false on overflow.
func (rc recipe) coefficients(dim func(Pair) int64) (c, s int64, ok bool) {
	var in sum
	for _, it := range rc.in {
		in.addMul(it.Qty, dim(it.Size))
	}
	a, rest := rc.outputLoad(dim)
	di, fits := checked.Mul(Denominator, in.v)
	c, sub := checked.Sub(a.v, di)
	if in.bad || a.bad || rest.bad || !fits || !sub {
		return 0, 0, false
	}
	return c, rest.v, true
}

// outputLoad returns A, the load of the outputs of one execution times D, and
// the residue term, over the outputs that are not discarded.
func (rc recipe) outputLoad(dim func(Pair) int64) (a, rest sum) {
	for _, o := range rc.out {
		if o.Discard {
			continue
		}
		qe, fits := checked.Mul(o.Qty, rc.yieldBP)
		a.addMul(qe, dim(o.Size))
		a.bad = a.bad || !fits
		if qe%Denominator != 0 {
			rest.addMul(dim(o.Size), o.Residue)
		}
	}
	return a, rest
}

// sum is a running checked sum of products that remembers an overflow.
type sum struct {
	v   int64
	bad bool
}

// addMul adds a*b, or marks the sum bad if the product or the sum overflows.
func (s *sum) addMul(a, b int64) {
	p, ok := checked.Mul(a, b)
	if !ok {
		s.bad = true
		return
	}
	if s.v, ok = checked.Add(s.v, p); !ok {
		s.bad = true
	}
}
