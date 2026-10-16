package clearing_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
)

// TestAverage_NegativeBidQtyIsSkipped verifies that a bid with a negative
// quantity leaves the book like an exhausted one and never trades, so the next
// bid is matched in its place.
//
// Requirement: FR-CLR-010
//
// Guard: the ask side has the same rule; without the bid-side test a check of
// `rb == 0` instead of `rb <= 0` would let a malformed bid emit a match of
// negative quantity.
func TestAverage_NegativeBidQtyIsSkipped(t *testing.T) {
	t.Parallel()
	// Both bids share limit 10 and key 0, so Ref orders them: the malformed
	// bid 0 is first. Bid 1 meets the ask at Midpoint(10, 5): lo 5, d 5,
	// m = 5+2 = 7, d odd and m odd, so 8; quantity min(5, 5) = 5.
	bids := []clearing.Offer{{Ref: 0, Price: 10, Qty: -1}, {Ref: 1, Price: 10, Qty: 5}}
	asks := []clearing.Offer{{Ref: 0, Price: 5, Qty: 5}}
	want := []clearing.Match{{BidRef: 1, AskRef: 0, Price: 8, Qty: 5}}
	if got := clearing.Average(nil, bids, asks); !slices.Equal(got, want) {
		t.Errorf("Average(%v, %v) = %v, want %v", bids, asks, got, want)
	}
}
