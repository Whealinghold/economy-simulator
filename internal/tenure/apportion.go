package tenure

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Part is one claimant of an apportionment: a holder, or the commons.
//
// Ref is the slot of the out slice where the result is written and must be
// distinct for every part. Share is the weight (basis points for a stock, units
// for a supply). ID is the agent ID and Key the stateless key of
// spec-resources.md 5.3.3, both the tie-break. Commons marks the commons part,
// which has no ID and ranks after every agent.
type Part struct {
	Ref     int
	ID      int64
	Share   int64
	Key     uint64
	Commons bool

	rem int64 // the remainder of stock * Share over the total weight
}

// Apportion shares stock units among the parts by the largest remainder
// (spec-land.md 4.2.1) and writes the units of each part to out[part.Ref]. A
// part with a Share of 0 is no part and gets 0.
//
// With W the sum of the weights, each part gets floor(stock * Share / W), and
// the units left, fewer than the number of parts with a positive weight, go one
// each to the parts of the largest remainder, ties by ascending Key, then by
// ascending ID, the commons after every agent. The order is total, so the result
// does not depend on how the parts were listed, and the units sum to stock
// exactly. Apportion sorts parts in place and allocates nothing. ok is false,
// and every slot a part names is 0, for a negative stock or Share, a Ref outside
// out, a sum of weights beyond int64, or a positive stock with no weight.
func Apportion(stock int64, parts []Part, out []int64) (ok bool) {
	total, valid := weights(stock, parts, len(out))
	if !valid {
		zeroOut(parts, out)
		return false
	}
	for _, p := range parts {
		out[p.Ref] = 0
	}
	if stock == 0 {
		return true
	}
	given := int64(0)
	for i := range parts {
		p := &parts[i]
		if p.Share == 0 {
			continue
		}
		// ok is always true: Share <= total, so the quotient is at most stock.
		q, r, _ := checked.MulDivRem(stock, p.Share, total)
		p.rem = r
		out[p.Ref] = q
		given += q // at most stock: the sum of floor(stock * s / W) over s summing to W
	}
	left := stock - given
	slices.SortFunc(parts, compareRemainder)
	for i := 0; left > 0 && i < len(parts); i++ {
		if parts[i].Share == 0 {
			break // positive weights sort first; the parts that follow have none
		}
		out[parts[i].Ref]++
		left--
	}
	return true
}

// weights returns the sum of the weights and whether the input is valid.
func weights(stock int64, parts []Part, n int) (int64, bool) {
	if stock < 0 {
		return 0, false
	}
	var total int64
	for _, p := range parts {
		if p.Share < 0 || p.Ref < 0 || p.Ref >= n {
			return 0, false
		}
		var ok bool
		if total, ok = checked.Add(total, p.Share); !ok {
			return 0, false
		}
	}
	return total, total > 0 || stock == 0
}

// zeroOut writes 0 to every slot of a part that has one.
func zeroOut(parts []Part, out []int64) {
	for _, p := range parts {
		if p.Ref >= 0 && p.Ref < len(out) {
			out[p.Ref] = 0
		}
	}
}

// compareRemainder orders the parts with a weight first, by descending
// remainder, then by agents before the commons, ascending Key and ascending ID.
func compareRemainder(a, b Part) int {
	return cmp.Or(
		boolRank(a.Share == 0)-boolRank(b.Share == 0),
		cmp.Compare(b.rem, a.rem),
		boolRank(a.Commons)-boolRank(b.Commons),
		cmp.Compare(a.Key, b.Key),
		cmp.Compare(a.ID, b.ID),
	)
}

// boolRank is 1 for true and 0 for false.
func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}
