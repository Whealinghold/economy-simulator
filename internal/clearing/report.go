package clearing

import "github.com/Whealinghold/economy-sim/internal/checked"

// Result is the market report of one tick (CLR 5.4, 5.5).
type Result struct {
	Price                  int64 // the published market price
	Age                    int64 // ticks since a unit last traded; 0 after a trade
	Volume                 int64 // units traded this tick
	UnmatchedBids          int64 // unfilled units of the bid side
	UnmatchedAsks          int64
	BestBid, BestAsk       int64 // highest unmatched bid, lowest unmatched ask
	HasBestBid, HasBestAsk bool  // the flags: false means none, not 0
}

// Report computes the report of one market. bids and asks are the admitted
// records, sums the units filled per order that Check left (bids first, then
// asks), matches the valid trades, prevPrice and prevAge the report of the
// previous tick (the seed price and age 0 before the first). The price is the
// volume-weighted mean rounded half to even, accumulated in 128 bits, or the
// previous price when nothing traded. ok is false only when a sum does not fit
// (impossible for valid trades within the bounds). The age saturates at
// MaxInt64 rather than wrapping, so a market that never trades keeps a valid
// report. It panics when sums is shorter than bids and asks, a violated
// programmer invariant.
func Report(bids, asks []Offer, sums []int64, matches []Match, prevPrice, prevAge int64) (r Result, ok bool) {
	if len(sums) < len(bids)+len(asks) {
		panic("clearing: Report sums shorter than the book (violated programmer invariant)")
	}
	var acc checked.Acc
	for _, m := range matches {
		// A 128-bit overflow here needs a volume above 4*MaxInt64, which the
		// Add below refuses first, so this branch is unreachable for any
		// input and is kept as a guard (testing.md 9.5).
		if !acc.AddMul(m.Price, m.Qty) {
			return Result{}, false
		}
		if r.Volume, ok = checked.Add(r.Volume, m.Qty); !ok {
			return Result{}, false
		}
	}
	r.Price = prevPrice
	if r.Age, ok = checked.Add(prevAge, 1); !ok {
		r.Age = prevAge // saturated
	}
	if r.Volume > 0 {
		// The mean never exceeds the highest price, so the quotient fits int64
		// whenever the volume does: unreachable, kept as a guard (testing.md 9.5).
		if r.Price, ok = acc.DivHalfEven(r.Volume); !ok {
			return Result{}, false
		}
		r.Age = 0
	}
	nb := len(bids)
	if r.UnmatchedBids, r.BestBid, r.HasBestBid, ok = unmatched(bids, sums[:nb], true); !ok {
		return Result{}, false
	}
	if r.UnmatchedAsks, r.BestAsk, r.HasBestAsk, ok = unmatched(asks, sums[nb:], false); !ok {
		return Result{}, false
	}
	return r, true
}

// unmatched sums the unfilled units of one side and finds its best unmatched
// limit (highest for bids, lowest for asks); ok is false if the sum overflows.
func unmatched(side []Offer, sums []int64, bid bool) (units, best int64, has, ok bool) {
	for i, o := range side {
		rem := o.Qty - sums[i]
		if rem <= 0 {
			continue
		}
		if units, ok = checked.Add(units, rem); !ok {
			return 0, 0, false, false
		}
		if !has || (bid && o.Price > best) || (!bid && o.Price < best) {
			best, has = o.Price, true
		}
	}
	return units, best, has, true
}
