package merchant_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// lane19 returns the round of example A (spec-merchants.md 19.1): offers of agents
// 32, 35 and 33 and takes of agents 61, 62 and 63, with their keys all equal so the
// order is the limit's.
func lane19() ([]merchant.Offer, []merchant.Bid) {
	offers := []merchant.Offer{{Ref: 32, Limit: 244, Qty: 150}, {Ref: 35, Limit: 250, Qty: 100}, {Ref: 33, Limit: 263, Qty: 150}}
	takes := []merchant.Bid{{Ref: 61, Limit: 270, Qty: 200}, {Ref: 62, Limit: 255, Qty: 150}, {Ref: 63, Limit: 240, Qty: 150}}
	return offers, takes
}

// TestMatch_FirstFitOrder verifies the round of spec-merchants.md 9.4.3 on example
// A: offers in order of limit and takes in descending order, the first take that
// fits, a divisible take left with the rest (61 keeps 50) and an offer that finds
// none (33), and that the result does not depend on the order the lists come in.
//
// Requirement: FR-MER-034
func TestMatch_FirstFitOrder(t *testing.T) {
	t.Parallel()
	want := []merchant.Pair{{Offer: 32, Bid: 61, Price: 257, Qty: 150}, {Offer: 35, Bid: 62, Price: 252, Qty: 100}}
	for _, shuffle := range []bool{false, true} {
		offers, takes := lane19()
		if shuffle {
			slices.Reverse(offers)
			slices.Reverse(takes)
		}
		merchant.SortOffers(offers)
		merchant.SortBids(takes)
		got := merchant.Match(nil, offers, takes, nil)
		if !slices.Equal(got, want) {
			t.Errorf("Match(example A, shuffled %v) = %+v, want %+v", shuffle, got, want)
		}
		if takes[0].Ref != 61 || takes[0].Qty != 50 || takes[1].Qty != 50 {
			t.Errorf("open quantities after the round = %+v, want take 61 left with 50 and take 62 with 50", takes)
		}
	}
}

// TestMatch_IndivisibleOffer_DivisibleTake verifies that an offer is never split
// across takes (the offer of 100 does not take the 50 left of the first take but
// goes to the next that holds all 100), that a take is divided among offers while
// it has room, and that a tentative match which fails formation drops the offer
// and leaves the take whole.
//
// Requirement: FR-MER-034, FR-MER-035
func TestMatch_IndivisibleOffer_DivisibleTake(t *testing.T) {
	t.Parallel()
	offers, takes := lane19()
	merchant.SortOffers(offers)
	merchant.SortBids(takes)
	var tried []merchant.Pair
	got := merchant.Match(nil, offers, takes, func(p merchant.Pair) bool {
		tried = append(tried, p)
		return p.Offer != 32 // the formation of the first offer fails
	})
	// Offer 32 is dropped and take 61 is whole; offer 35 (limit 250, 100 units) goes to take 61 at (250 + 270) / 2 = 260;
	// offer 33 (limit 263, 150 units) finds take 61 with 100 left and take 62 below its limit.
	want := []merchant.Pair{{Offer: 35, Bid: 61, Price: 260, Qty: 100}}
	if !slices.Equal(got, want) || len(tried) != 2 || takes[0].Qty != 100 {
		t.Errorf("Match with a failed first formation = %+v after trying %+v, take 61 open %d, want %+v, two tries and 100 open", got, tried, takes[0].Qty, want)
	}
	// Two offers share one take while it has room.
	o := []merchant.Offer{{Ref: 1, Limit: 10, Qty: 40}, {Ref: 2, Limit: 10, Qty: 50, Key: 1}}
	b := []merchant.Bid{{Ref: 9, Limit: 20, Qty: 100}}
	merchant.SortOffers(o)
	merchant.SortBids(b)
	if got := merchant.Match(nil, o, b, nil); len(got) != 2 || b[0].Qty != 10 {
		t.Errorf("Match(two offers, one take of 100) = %+v with %d open, want two pairs and 10 open", got, b[0].Qty)
	}
}

// TestMatch_Order verifies the total order of 9.4.2: equal limits go by key and
// then by ref, offers ascending and takes descending in limit.
//
// Requirement: FR-MER-032, FR-MER-034
func TestMatch_Order(t *testing.T) {
	t.Parallel()
	o := []merchant.Offer{{Ref: 3, Limit: 5, Key: 7}, {Ref: 2, Limit: 5, Key: 7}, {Ref: 1, Limit: 5, Key: 9}, {Ref: 4, Limit: 4, Key: 100}, {Ref: 5, Limit: 6, Key: 0}}
	merchant.SortOffers(o)
	refs := func() (out []int) {
		for _, x := range o {
			out = append(out, x.Ref)
		}
		return out
	}()
	if want := []int{4, 2, 3, 1, 5}; !slices.Equal(refs, want) {
		t.Errorf("SortOffers order = %v, want %v: limit ascending, then key, then ref", refs, want)
	}
	b := []merchant.Bid{{Ref: 3, Limit: 5, Key: 7}, {Ref: 2, Limit: 5, Key: 7}, {Ref: 1, Limit: 5, Key: 9}, {Ref: 4, Limit: 4, Key: 100}, {Ref: 5, Limit: 6, Key: 0}}
	merchant.SortBids(b)
	var got []int
	for _, x := range b {
		got = append(got, x.Ref)
	}
	if want := []int{5, 2, 3, 1, 4}; !slices.Equal(got, want) {
		t.Errorf("SortBids order = %v, want %v: limit descending, then key, then ref", got, want)
	}
}

// TestMatch_Boundaries verifies the two comparisons of the first fit at equality:
// a take whose limit equals the offer's limit matches (at that price), and a take
// whose open quantity equals the offer's quantity matches and is left with 0.
//
// Requirement: FR-MER-034
func TestMatch_Boundaries(t *testing.T) {
	t.Parallel()
	o := []merchant.Offer{{Ref: 1, Limit: 100, Qty: 100}}
	b := []merchant.Bid{{Ref: 2, Limit: 100, Qty: 100}}
	got := merchant.Match(nil, o, b, nil)
	if want := []merchant.Pair{{Offer: 1, Bid: 2, Price: 100, Qty: 100}}; !slices.Equal(got, want) || b[0].Qty != 0 {
		t.Errorf("Match(equal limits, equal quantities) = %+v with %d open, want %+v with 0 open", got, b[0].Qty, want)
	}
	o = []merchant.Offer{{Ref: 1, Limit: 101, Qty: 1}}
	b = []merchant.Bid{{Ref: 2, Limit: 100, Qty: 5}}
	if got := merchant.Match(nil, o, b, nil); len(got) != 0 {
		t.Errorf("Match(take limit one below the offer's) = %+v, want none", got)
	}
}

// TestMatch_KeyAndRefBreakTies verifies that equal limits are ordered by key and
// equal keys by ref, whatever order the lists come in: keys that run against refs
// put the lowest key first, and keys that are equal go by ref.
//
// Requirement: FR-MER-032
func TestMatch_KeyAndRefBreakTies(t *testing.T) {
	t.Parallel()
	o := []merchant.Offer{{Ref: 1, Limit: 5, Key: 30}, {Ref: 2, Limit: 5, Key: 20}, {Ref: 3, Limit: 5, Key: 10}}
	merchant.SortOffers(o)
	if o[0].Ref != 3 || o[1].Ref != 2 || o[2].Ref != 1 {
		t.Errorf("SortOffers with keys against refs = %+v, want refs 3, 2, 1", o)
	}
	o = []merchant.Offer{{Ref: 9, Limit: 5, Key: 1}, {Ref: 7, Limit: 5, Key: 1}, {Ref: 8, Limit: 5, Key: 1}}
	merchant.SortOffers(o)
	if o[0].Ref != 7 || o[1].Ref != 8 || o[2].Ref != 9 {
		t.Errorf("SortOffers with equal keys = %+v, want refs 7, 8, 9", o)
	}
	b := []merchant.Bid{{Ref: 1, Limit: 5, Key: 30}, {Ref: 2, Limit: 5, Key: 20}, {Ref: 3, Limit: 5, Key: 10}}
	merchant.SortBids(b)
	if b[0].Ref != 3 || b[1].Ref != 2 || b[2].Ref != 1 {
		t.Errorf("SortBids with keys against refs = %+v, want refs 3, 2, 1", b)
	}
	b = []merchant.Bid{{Ref: 9, Limit: 5, Key: 1}, {Ref: 7, Limit: 5, Key: 1}, {Ref: 8, Limit: 5, Key: 1}}
	merchant.SortBids(b)
	if b[0].Ref != 7 || b[1].Ref != 8 || b[2].Ref != 9 {
		t.Errorf("SortBids with equal keys = %+v, want refs 7, 8, 9", b)
	}
}

// TestMidpoint_HalfEven_BetweenLimits verifies the midpoint rounded half to even
// (252.5 is 252, 3.5 is 4, 4.5 is 4, -3.5 is -4, -4.5 is -4), that it lies between
// the limits, and that it does not overflow near the ends of int64.
//
// Requirement: FR-MER-035
func TestMidpoint_HalfEven_BetweenLimits(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ a, b, want int64 }{
		{244, 270, 257}, {250, 255, 252}, {3, 4, 4}, {4, 5, 4}, {5, 6, 6}, {-3, -4, -4}, {-4, -5, -4},
		{0, 0, 0}, {7, 7, 7}, {0, 1, 0}, {1, 2, 2}, {-1, 0, 0},
		{1<<63 - 1, 1<<63 - 1, 1<<63 - 1}, {-1 << 63, -1 << 63, -1 << 63}, {1<<63 - 1, 1<<63 - 2, 1<<63 - 2},
	} {
		got := merchant.Midpoint(tt.a, tt.b)
		if got != tt.want || got < min(tt.a, tt.b) || got > max(tt.a, tt.b) {
			t.Errorf("Midpoint(%d, %d) = %d, want %d between the limits", tt.a, tt.b, got, tt.want)
		}
		if back := merchant.Midpoint(tt.b, tt.a); back != got {
			t.Errorf("Midpoint(%d, %d) = %d, want the symmetric %d", tt.b, tt.a, back, got)
		}
	}
}
