package household

import "github.com/Whealinghold/economy-sim/internal/checked"

// The pure rules of the demand module (spec-demand.md 4.2 to 4.7, 4.9). Like
// the rest of the package they take primitives, keep no state, draw nothing and
// allocate nothing when the caller passes slices with room. A function that
// multiplies reports ok false on an overflow or a zero divisor. A negative
// argument counts as 0, and a weight, a share or a count is never negative in a
// validated configuration.

// Carry spread constants of spec-demand.md 4.4: primes, chosen to spread the
// start of the carries. They are rule constants (architecture.md 9.6); any other
// pair would do.
const (
	carryAgentStep int64 = 7_919
	carryWantStep  int64 = 2_777
)

// WantItem is one alternative of a want at one agent: the agent's belief centre
// for the commodity, the effective reference price in its region and the weight
// of the item in basis points (spec-demand.md 4.1).
type WantItem struct {
	Centre, Ref, Weight int64
}

// Rung is one want of the ladder at the moment tier 4 is funded: the planned
// units (0 for a choked want), the shares of its items (adding to 10 000) and
// the limit price of the bid for each item. Shares and Limits are parallel and
// in commodity order (spec-demand.md 4.5).
type Rung struct {
	Units          int64
	Shares, Limits []int64
}

// PriceRatio returns the ratio of an item's price to its reference in basis
// points, floor: centre * 10 000 / ref (FR-DEM-007). ok is false when ref is
// below 1 or the product overflows.
func PriceRatio(centre, ref int64) (int64, bool) {
	if ref < 1 {
		return 0, false
	}
	return checked.MulDiv(max(centre, 0), Basis, ref)
}

// Shares divides a unit of a want among its items (spec-demand.md 4.2,
// FR-DEM-008). With x = ref * 10 000 / centre and sigma = substitution (steps of
// v = v * x / 10 000 from v = 10 000, a negative sigma counts as 0), the term of
// an item is t = weight * pow(x, sigma) / 10 000, and the share is t * 10 000 /
// sum(t) by largest remainder: each item gets the floor, and the basis points
// left go one each to the items with the greatest fractional remainders, ties to
// the lower index. The caller passes the items in commodity order, so the lower
// index is the lower commodity ID. The shares are appended to dst[:0] and add to
// 10 000. ok is false, with an empty result, for no items, a centre below 1, a
// sum of terms of 0 (all weights 0) or an overflow.
//
// The spec text hands the remainder to the item with the greatest t; its worked
// example 15.1 (5 728 and 4 272, where t gives 5 729 and 4 271) is the largest
// remainder rule, which this function follows.
func Shares(dst []int64, items []WantItem, substitution int64) ([]int64, bool) {
	var buf [8]int64 // twice the root's MaxWantItems (4): no allocation within the bound
	terms := buf[:0]
	var sum int64
	for _, it := range items {
		t, ok := term(it, substitution)
		if !ok {
			return dst[:0], false
		}
		terms = append(terms, t)
		if sum, ok = checked.Add(sum, t); !ok {
			return dst[:0], false
		}
	}
	return apportion(dst, Basis, terms, sum)
}

// term is weight * pow(ref * 10 000 / centre, sigma) / 10 000.
func term(it WantItem, sigma int64) (int64, bool) {
	if it.Centre < 1 {
		return 0, false
	}
	x, ok := checked.MulDiv(max(it.Ref, 0), Basis, it.Centre)
	if !ok {
		return 0, false
	}
	v := Basis
	for i := int64(0); i < sigma; i++ {
		if v, ok = checked.MulDiv(v, x, Basis); !ok {
			return 0, false
		}
	}
	return checked.MulDiv(max(it.Weight, 0), v, Basis)
}

// WantRatio returns the price ratio of a want: the mean of the item ratios
// weighted by the shares, sum(s * r) / 10 000, floor (spec-demand.md 4.2,
// FR-DEM-009). The slices are parallel and only the shorter prefix is read. ok
// is false on an overflow or when an item has a reference below 1.
func WantRatio(items []WantItem, shares []int64) (int64, bool) {
	var sum int64
	for i := 0; i < len(items) && i < len(shares); i++ {
		r, ok := PriceRatio(items[i].Centre, items[i].Ref)
		if !ok {
			return 0, false
		}
		if r, ok = checked.Mul(max(shares[i], 0), r); !ok {
			return 0, false
		}
		if sum, ok = checked.Add(sum, r); !ok {
			return 0, false
		}
	}
	return sum / Basis, true
}

// Schedule returns the share of the base quantity a household wants at a price
// ratio, in basis points (spec-demand.md 4.3, FR-DEM-010):
//
//	clamp(10 000 + trunc(elasticity * (10 000 - ratio) / 10 000), 0, ceiling)
//
// It is 10 000 at the reference (ratio 10 000), never rises with the ratio, and
// is 0 from the choke price, 10 000 + 10^8 / elasticity, rounded up to a whole
// basis point. A ceiling below 0 counts as 0. ok is false on an overflow.
func Schedule(ratio, elasticity, ceiling int64) (int64, bool) {
	d, ok := checked.Sub(Basis, ratio)
	if !ok {
		return 0, false
	}
	slope, ok := checked.MulDiv(elasticity, d, Basis)
	if !ok {
		return 0, false
	}
	g, ok := checked.Add(Basis, slope)
	if !ok {
		return 0, false
	}
	return min(max(g, 0), max(ceiling, 0)), true
}

// Units advances the carry of one want and returns the whole units wanted
// (spec-demand.md 4.4, FR-DEM-012):
//
//	need     = quantity * weight * tasteBP * regionBP * sched / (period * 10^8)
//	units    = (carry + need) / 10 000
//	newCarry = (carry + need) mod 10 000
//
// The numerator of need is held in 128 bits and divided once, so need is exact
// and the carry adds the fractions up exactly (FR-DEM-014): a cohort of weight w
// wants w times the rate and the carry loses nothing. A sched of 0 (a choked
// want) leaves the carry as it is. The caller keeps newCarry only for an agent
// whose plan it commits, and a unit that is not funded is lost (FR-DEM-015). The
// carry is in [0, 9 999]; a negative one counts as 0. ok is false for a period
// below 1 or an overflow.
func Units(carry, quantity, weight, tasteBP, regionBP, sched, period int64) (units, newCarry int64, ok bool) {
	if period < 1 {
		return 0, 0, false
	}
	n1, ok := checked.Mul(max(quantity, 0), max(weight, 0))
	if !ok {
		return 0, 0, false
	}
	if n1, ok = checked.Mul(n1, max(tasteBP, 0)); !ok {
		return 0, 0, false
	}
	n2, ok := checked.Mul(max(regionBP, 0), max(sched, 0))
	if !ok {
		return 0, 0, false
	}
	need, ok := checked.MulDivDiv(n1, n2, period, Basis*Basis)
	if !ok {
		return 0, 0, false
	}
	total, ok := checked.Add(max(carry, 0), need)
	if !ok {
		return 0, 0, false
	}
	return total / Basis, total % Basis, true
}

// Fund allocates the want budget down the ladder (spec-demand.md 4.5,
// FR-DEM-016 to FR-DEM-018) and returns the budget left. The budget is
//
//	B = min(max(0, spendCap - spent3), max(0, avail))
//
// the cap of tier 3 less the money tier 3 spent, and never more than the money
// left after tiers 1 to 3. The rungs are funded in the order given (the ladder,
// by rank then ID), so a rung of higher rank gets only what the lower ones
// leave. A rung splits its units among its items by Shares (largest remainder,
// ties to the lower index), and each item buys the lesser of its units and B
// over its limit price, then B falls by units * limit. An item whose limit is
// below 1 buys nothing, and a rung of 0 units funds nothing. There is no
// re-split to a cheaper item.
//
// out receives the units funded per item, the items of the rungs in order, so
// it needs room for the sum of len(Shares); the unfunded units of a rung are its
// Units less its funded ones. The money spent never exceeds B, so the sum of
// out * Limits is at most B. ok is false when out is too short, a rung's Shares
// and Limits differ in length or its shares add to more than 10 000; out is
// then undefined (rungs before the bad one may be written), and the caller drops
// it. Shares that add to less than 10 000 are tolerated only while the units
// still find an item to take them: otherwise apportion, and so Fund, reports false.
func Fund(spendCap, spent3, avail int64, ladder []Rung, out []int64) (left int64, ok bool) {
	budget := min(max(0, max(spendCap, 0)-max(spent3, 0)), max(avail, 0))
	off := 0
	for _, r := range ladder {
		n := len(r.Shares)
		if n != len(r.Limits) || off+n > len(out) || !sharesFit(r.Shares) {
			return 0, false
		}
		split, ok := apportion(out[off:off], max(r.Units, 0), r.Shares, Basis)
		if !ok {
			return 0, false
		}
		for j, units := range split {
			q := int64(0)
			if l := r.Limits[j]; l >= 1 {
				q = min(units, budget/l)
			}
			out[off+j] = q
			budget -= q * r.Limits[j]
		}
		off += n
	}
	return budget, true
}

// sharesFit reports whether the shares (a negative one counts as 0) add to at
// most 10 000, which apportion alone does not see when the units are few.
func sharesFit(shares []int64) bool {
	var sum int64
	for _, s := range shares {
		sum += max(s, 0)
		if sum > Basis {
			return false
		}
	}
	return true
}

// apportion divides total in proportion to w / den, by largest remainder:
// dst[j] = floor(total * w[j] / den), then the units left go one each to the
// items with the greatest remainders of total * w[j] mod den, ties to the lower
// index. The result is appended to dst[:0] and adds to total when the weights
// add to den. ok is false for a den of 0, an overflow or weights that add to
// more than den.
func apportion(dst []int64, total int64, w []int64, den int64) ([]int64, bool) {
	dst = dst[:0]
	left := total
	for _, wj := range w {
		q, ok := checked.MulDiv(total, max(wj, 0), den)
		if !ok {
			return dst[:0], false
		}
		dst = append(dst, q)
		left -= q
	}
	if left < 0 {
		return dst[:0], false
	}
	// Each pass takes the next item in the order (remainder descending, index
	// ascending), so no scratch marks the items already given a unit.
	prevRem, prev := int64(0), -1
	for ; left > 0; left-- {
		best, bestRem, ok := nextRemainder(total, w, den, prevRem, prev)
		if !ok {
			return dst[:0], false
		}
		dst[best]++
		prevRem, prev = bestRem, best
	}
	return dst, true
}

// nextRemainder returns the item with the greatest remainder of total * w[j]
// mod den that comes after the item prev (with remainder prevRem) in the order
// remainder descending, index ascending; prev -1 means the first. ok is false on
// an overflow or when no item is left.
func nextRemainder(total int64, w []int64, den, prevRem int64, prev int) (best int, bestRem int64, ok bool) {
	best = -1
	for j, wj := range w {
		_, r, ok := checked.MulDivRem(total, max(wj, 0), den)
		if !ok {
			return -1, 0, false
		}
		if prev >= 0 && (r > prevRem || (r == prevRem && j <= prev)) {
			continue
		}
		if best < 0 || r > bestRem {
			best, bestRem = j, r
		}
	}
	return best, bestRem, best >= 0
}

// Consumed returns the units of a commodity a household consumes for its wants
// in the consumption phase, after upkeep (spec-demand.md 4.7, D32, FR-DEM-025):
// min(want, max(0, held - upkeep)). The subtraction is the guard that a want
// never eats into the next tick's upkeep of the same commodity. A negative
// argument counts as 0.
func Consumed(want, held, upkeep int64) int64 {
	return min(max(want, 0), max(0, max(held, 0)-max(upkeep, 0)))
}

// InitialCarry returns the carry of a want at the creation of an agent or at the
// change of its role (spec-demand.md 4.4, FR-DEM-013):
//
//	(agentIndex * 7 919 + position * 2 777) mod 10 000
//
// agentIndex is the 0-based position of the agent in its region's agent list and
// position the 0-based position of the want in the agent's ladder. It is a
// deterministic spread and draws nothing. ok is false on an overflow.
func InitialCarry(agentIndex, position int64) (int64, bool) {
	a, ok := checked.Mul(max(agentIndex, 0), carryAgentStep)
	if !ok {
		return 0, false
	}
	p, ok := checked.Mul(max(position, 0), carryWantStep)
	if !ok {
		return 0, false
	}
	s, ok := checked.Add(a, p)
	if !ok {
		return 0, false
	}
	return s % Basis, true
}

// KeepCarries forms the carries of an agent whose role changed (spec-demand.md
// 4.9, FR-DEM-042). oldIDs and oldCarries are the want IDs and carries of the
// old ladder, newIDs the want IDs of the new ladder in ladder order. A want in
// both keeps its carry (matched by ID, so a new rank does not matter), a want
// only in the new ladder starts at InitialCarry with its position in the new
// ladder and the agent's index, and a want only in the old ladder is dropped.
// The carries are appended to dst[:0] in the order of newIDs; dst must not
// share its array with oldIDs or oldCarries. ok is false when
// oldIDs and oldCarries differ in length or InitialCarry overflows.
func KeepCarries(dst, oldIDs, oldCarries, newIDs []int64, agentIndex int64) ([]int64, bool) {
	dst = dst[:0]
	if len(oldIDs) != len(oldCarries) {
		return dst, false
	}
	for pos, id := range newIDs {
		c, kept := int64(0), false
		for i, old := range oldIDs {
			if old == id {
				c, kept = oldCarries[i], true
				break
			}
		}
		if !kept {
			var ok bool
			if c, ok = InitialCarry(agentIndex, int64(pos)); !ok {
				return dst[:0], false
			}
		}
		dst = append(dst, c)
	}
	return dst, true
}
