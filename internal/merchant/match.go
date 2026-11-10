package merchant

import "slices"

// Offer is an admitted delivery offer as the round reads it (spec-merchants.md
// 9.4): an indivisible quantity at no less than Limit, with the tie-break Key
// drawn for it. Ref is the caller's index of the offer.
type Offer struct {
	Ref        int
	Limit, Qty int64
	Key        uint64
}

// Bid is an admitted delivery take as the round reads it: a divisible quantity at
// no more than Limit, with its Key. Qty is the quantity still open; Match lowers
// it as offers are matched against it. Ref is the caller's index of the take.
type Bid struct {
	Ref        int
	Limit, Qty int64
	Key        uint64
}

// Pair is a tentative or formed match: the refs of an offer and a take, the price
// and the quantity of the offer.
type Pair struct {
	Offer, Bid int
	Price, Qty int64
}

// SortOffers orders offers by Limit ascending, equal limits by Key, then by Ref
// (spec-merchants.md 9.4.2): a total order, so the result does not depend on the
// sort.
func SortOffers(o []Offer) {
	slices.SortFunc(o, func(a, b Offer) int {
		switch {
		case a.Limit != b.Limit:
			return cmp3(a.Limit < b.Limit)
		case a.Key != b.Key:
			return cmp3(a.Key < b.Key)
		}
		return cmp3(a.Ref < b.Ref)
	})
}

// SortBids orders takes by Limit descending, equal limits by Key, then by Ref
// (spec-merchants.md 9.4.2).
func SortBids(t []Bid) {
	slices.SortFunc(t, func(a, b Bid) int {
		switch {
		case a.Limit != b.Limit:
			return cmp3(a.Limit > b.Limit)
		case a.Key != b.Key:
			return cmp3(a.Key < b.Key)
		}
		return cmp3(a.Ref < b.Ref)
	})
}

// cmp3 is -1 for true and 1 for false, for a comparator whose operands differ.
func cmp3(less bool) int {
	if less {
		return -1
	}
	return 1
}

// Midpoint returns (a + b) / 2 rounded half to even (spec-merchants.md 9.4.3,
// spec-clearing.md 5.2), which lies between a and b and cannot overflow.
func Midpoint(a, b int64) int64 {
	carry := a&1 + b&1
	q := a>>1 + b>>1 + carry>>1
	if carry&1 == 1 && q&1 == 1 {
		q++
	}
	return q
}

// Match runs the round of one lane (spec-merchants.md 9.4.3): offers and takes
// must be sorted by SortOffers and SortBids. For each offer in order it finds the
// first take whose Limit is at least the offer's and whose open quantity is at
// least the offer's quantity (an offer is indivisible, a take divisible); with
// none the offer is unmatched. A found pair is tentative at the midpoint of the
// two limits and goes to form, which reports whether formation succeeded: when it
// does the take's open quantity falls by the offer's and the pair is appended to
// dst; when it does not the offer is dropped and the take is unchanged. A nil
// form always succeeds. The takes' Qty is lowered in place.
func Match(dst []Pair, offers []Offer, takes []Bid, form func(Pair) bool) []Pair {
	for _, o := range offers {
		for i := range takes {
			t := &takes[i]
			if t.Limit < o.Limit {
				break // takes are in descending limit order: none below is higher
			}
			if t.Qty < o.Qty {
				continue
			}
			p := Pair{Offer: o.Ref, Bid: t.Ref, Price: Midpoint(o.Limit, t.Limit), Qty: o.Qty}
			if form == nil || form(p) {
				t.Qty -= o.Qty
				dst = append(dst, p)
			}
			break
		}
	}
	return dst
}
