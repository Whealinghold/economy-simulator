package clearing

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Failures of Check (CLR 8.2). A fill with several faults reports the first in
// this order; a list reports the lowest code over its fills, so the code does
// not depend on the order of the fills.
const (
	BadCommodity Code = iota + 1 // the fill is for another commodity than the book's
	NoOrder                      // the buyer holds no bid or the seller no ask (of that lot)
	BadQuantity                  // quantity below 1
	BadPrice                     // outside [ask limit, bid limit]
	SameOwner                    // buyer and seller are one owner
	OverFilled                   // units summed per order above the order's
)

// Fill is one trade of a list a Clearing returned, over opaque owners.
type Fill struct {
	Commodity     int64
	Buyer, Seller int64
	Lot           Lot // the lot of the ask it filled
	Price, Qty    int64
}

// Scratch is caller-owned working memory of Check, each slice of length
// len(bids)+len(asks). Sums[i] is the units filled of order i, bids first then
// asks; Check zeroes it itself.
type Scratch struct {
	Sums  []int64
	Index []int32
}

func lotLess(a, b Lot) int {
	switch {
	case a.Named != b.Named:
		if !a.Named {
			return -1
		}
		return 1
	case a.Born != b.Born:
		if a.Born < b.Born {
			return -1
		}
		return 1
	case a.Durability != b.Durability:
		if a.Durability < b.Durability {
			return -1
		}
		return 1
	}
	return 0
}

func cmpKey(o Offer, owner int64, lot Lot) int {
	switch {
	case o.Owner < owner:
		return -1
	case o.Owner > owner:
		return 1
	}
	return lotLess(o.Lot, lot)
}

// find returns the position in side of the order of (owner, lot), or -1. idx
// holds the positions of side sorted by (owner, lot).
func find(side []Offer, idx []int32, owner int64, lot Lot) int {
	lo, hi := 0, len(idx)
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		if cmpKey(side[idx[m]], owner, lot) < 0 {
			lo = m + 1
		} else {
			hi = m
		}
	}
	if lo < len(idx) && cmpKey(side[idx[lo]], owner, lot) == 0 {
		return int(idx[lo])
	}
	return -1
}

func sortIndex(side []Offer, idx []int32) {
	for i := range idx {
		idx[i] = int32(i)
	}
	// Total order: admission allows one order per (owner, lot, side), so no
	// tie-break is needed (architecture.md 7.6).
	slices.SortFunc(idx, func(a, b int32) int {
		return cmpKey(side[a], side[b].Owner, side[b].Lot)
	})
}

// checkFill returns the first fault of one fill in the order of the failure
// codes (0 if none) and the positions of its bid and ask when they resolved.
func checkFill(f Fill, commodity int64, bids, asks []Offer, bi, ai []int32) (c Code, b, a int) {
	if f.Commodity != commodity {
		return BadCommodity, -1, -1
	}
	if b = find(bids, bi, f.Buyer, Lot{}); b < 0 {
		return NoOrder, b, -1
	}
	if a = find(asks, ai, f.Seller, f.Lot); a < 0 {
		return NoOrder, b, a
	}
	switch {
	case f.Qty < 1:
		c = BadQuantity
	case f.Price < asks[a].Price || f.Price > bids[b].Price:
		c = BadPrice
	case f.Buyer == f.Seller:
		c = SameOwner
	}
	return c, b, a
}

// Check validates a list of fills over the book's own admitted records (bids
// and asks, never the slices the Clearing was given) and the book's commodity.
// On success it appends one Match per fill, in fill order, to dst, and leaves
// the units filled per order in sc.Sums. On failure it returns dst at its
// passed length and the code; Sums is then meaningless. It writes only to dst
// and sc and allocates nothing with a warmed scratch. It panics if sc is too
// short, a violated programmer invariant.
func Check(dst []Match, sc Scratch, commodity int64, bids, asks []Offer, fills []Fill) ([]Match, Code) {
	nb, n := len(bids), len(bids)+len(asks)
	if len(sc.Sums) < n || len(sc.Index) < n {
		panic("clearing: Check scratch shorter than the book (violated programmer invariant)")
	}
	sums, bi, ai := sc.Sums[:n], sc.Index[:nb], sc.Index[nb:n]
	clear(sums)
	sortIndex(bids, bi)
	sortIndex(asks, ai)
	base := len(dst)
	var worst Code
	over := false
	for _, f := range fills {
		c, b, a := checkFill(f, commodity, bids, asks, bi, ai)
		if c != 0 {
			if worst == 0 || c < worst {
				worst = c
			}
			continue
		}
		sb, ok1 := checked.Add(sums[b], f.Qty)
		sa, ok2 := checked.Add(sums[nb+a], f.Qty)
		if !ok1 || !ok2 || sb > bids[b].Qty || sa > asks[a].Qty {
			over = true
			continue
		}
		sums[b], sums[nb+a] = sb, sa
		dst = append(dst, Match{BidRef: bids[b].Ref, AskRef: asks[a].Ref, Price: f.Price, Qty: f.Qty})
	}
	if worst == 0 && over {
		worst = OverFilled
	}
	if worst != 0 {
		return dst[:base], worst
	}
	return dst, 0
}
