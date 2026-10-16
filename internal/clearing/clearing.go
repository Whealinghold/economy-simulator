// Package clearing holds the pure matching rules of the market over primitive
// values: it knows no agent, commodity or lot, only opaque numbers (CLR 8.3,
// architecture.md 2.9). It imports the standard library only, draws no random
// numbers (the root supplies the tie-break keys) and holds no state.
package clearing

import (
	"cmp"
	"slices"
)

// Lot names the lot an ask is made from. The zero value names none. Average
// ignores it.
type Lot struct {
	Named      bool
	Born       int64
	Durability int64
}

// Offer is one admitted order as the matching sees it.
type Offer struct {
	// Ref is the index into the caller's slice and so the input position; it is
	// unique in a market, which makes the order of Average total.
	Ref   int
	Price int64  // limit price per unit, fixed point
	Qty   int64  // units, at least 1
	Key   uint64 // tie-break key of CLR 4.2, drawn by the root
	Owner int64  // opaque owner; Average ignores it
	Lot   Lot    // the lot an ask names; Average ignores it
}

// Match is one trade between a bid and an ask, by Ref.
type Match struct {
	BidRef, AskRef int
	Price, Qty     int64
}

// Midpoint returns the midpoint of two limit prices rounded half to even
// (CLR 5.2, architecture.md 3.5). It is computed as lo + (hi-lo)/2, which
// cannot overflow for non-negative prices; when hi-lo is odd the exact
// midpoint ends in one half and the even neighbour is returned. The arguments
// may come in either order.
func Midpoint(a, b int64) int64 {
	lo, hi := min(a, b), max(a, b)
	d := hi - lo
	m := lo + d/2
	if d%2 != 0 && m%2 != 0 {
		m++ // neighbours are m and m+1; m is odd, so m+1 is the even one
	}
	return m
}

// Average is the default matching (CLR 4). It sorts bids by limit descending
// and asks by limit ascending, ties by Key and then by Ref, in place (the
// slices are scratch the caller owns), pairs the best bid with the best ask
// while the bid's limit is not below the ask's, trades the smaller remaining
// quantity at Midpoint of the two limits, and appends the matches to dst. It
// does not change Qty. It does not look at Owner: the guard against a trade
// with oneself is the check of the trade list, not the matching (CLR 3.1.2).
func Average(dst []Match, bids, asks []Offer) []Match {
	slices.SortFunc(bids, func(a, b Offer) int { return compare(b.Price, a.Price, a, b) })
	slices.SortFunc(asks, func(a, b Offer) int { return compare(a.Price, b.Price, a, b) })

	i, j := 0, 0
	var rb, ra int64 // remaining of the bid and ask under the cursors
	if len(bids) > 0 {
		rb = bids[0].Qty
	}
	if len(asks) > 0 {
		ra = asks[0].Qty
	}
	for i < len(bids) && j < len(asks) {
		if rb <= 0 { // exhausted (or malformed) bid leaves its book
			if i++; i < len(bids) {
				rb = bids[i].Qty
			}
			continue
		}
		if ra <= 0 {
			if j++; j < len(asks) {
				ra = asks[j].Qty
			}
			continue
		}
		if bids[i].Price < asks[j].Price {
			break
		}
		q := min(rb, ra)
		dst = append(dst, Match{
			BidRef: bids[i].Ref, AskRef: asks[j].Ref,
			Price: Midpoint(bids[i].Price, asks[j].Price), Qty: q,
		})
		rb -= q
		ra -= q
	}
	return dst
}

// compare orders by the price difference arguments pa, pb, then key, then Ref.
func compare(pa, pb int64, a, b Offer) int {
	if pa != pb {
		return cmp.Compare(pa, pb)
	}
	if a.Key != b.Key {
		return cmp.Compare(a.Key, b.Key)
	}
	return cmp.Compare(a.Ref, b.Ref)
}
