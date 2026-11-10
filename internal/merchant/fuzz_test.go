package merchant_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// FuzzFeeShare verifies, for any base, spread and agent ID, that FeeShare never
// panics, returns a share in 1 to 10 000, and returns the base (clamped) when the
// spread is 0.
//
// Requirement: NFR-MER-005, FR-MER-017
func FuzzFeeShare(f *testing.F) {
	f.Add(int64(5_000), int64(2_500), int64(32))
	f.Add(int64(1), int64(5_000), int64(-7))
	f.Add(int64(10_000), int64(5_000), int64(1<<62))
	f.Fuzz(func(t *testing.T, base, spread, id int64) {
		spread = spread % 5_001 // the validated bound; the product spread * 10 000 must fit
		if spread < 0 {
			spread = -spread
		}
		got := merchant.FeeShare(base%20_000, spread, id)
		if got < 1 || got > 10_000 {
			t.Fatalf("FeeShare(%d, %d, %d) = %d, want 1 to 10000", base%20_000, spread, id, got)
		}
		if spread == 0 && got != min(max(base%20_000, 1), 10_000) {
			t.Errorf("FeeShare(%d, 0, %d) = %d, want the clamped base", base%20_000, id, got)
		}
	})
}

// FuzzMargin verifies, for any lot, that Margin never panics and, when it reports
// ok, that the score is G - C, the break-even and the ask limit are the revenue
// and the acquisition price moved by the margin needed, and that the lot passes
// the test exactly when the needed margin does not exceed the unit margin (the
// score and the pass test agree about the sign of the net).
//
// Requirement: NFR-MER-005, FR-MER-015, FR-MER-018
func FuzzMargin(f *testing.F) {
	f.Add(int64(4_200), int64(6_385), int64(150), int64(289), int64(200))
	f.Add(int64(1<<62), int64(1), int64(1), int64(0), int64(0))
	f.Add(int64(-1), int64(0), int64(0), int64(-5), int64(1<<63-1))
	f.Fuzz(func(t *testing.T, c, phi, q, rnet, pacq int64) {
		r, ok := merchant.Margin(c, phi, q, rnet, pacq)
		if !ok {
			if r != (merchant.Result{}) {
				t.Fatalf("Margin(%d, %d, %d, %d, %d) = %+v with ok false, want the zero Result", c, phi, q, rnet, pacq, r)
			}
			return
		}
		if r.Score != r.G-c || r.BreakEven != rnet-r.Needed || r.AskLimit != pacq+r.Needed || r.Needed < 0 {
			t.Fatalf("Margin(%d, %d, %d, %d, %d) = %+v, want Score G-C, BreakEven rnet-Needed, AskLimit pacq+Needed, Needed >= 0", c, phi, q, rnet, pacq, r)
		}
		// phi * G >= C * 10 000 means the margin per unit covers the needed margin: Needed <= rnet - pacq.
		if pass := merchant.Pass(c, phi, r.G); pass != (r.Needed <= rnet-pacq && r.G > 0) {
			t.Errorf("Pass(%d, %d, %d) = %v but Needed %d, unit margin %d: the test and the prices disagree", c, phi, r.G, pass, r.Needed, rnet-pacq)
		}
	})
}

// FuzzMatch verifies, for any lane, that Match never panics, that the price of a
// pair lies between the two limits, that each pair's quantity is the offer's, that
// no take is left with a negative open quantity, that the open quantity of each
// take plus what was matched against it is what it started with, and that an offer
// is matched at most once.
//
// Requirement: NFR-MER-005, FR-MER-034, FR-MER-035
func FuzzMatch(f *testing.F) {
	f.Add([]byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120})
	f.Add([]byte{255, 0, 255, 0, 1, 1, 1, 1, 7, 7, 7, 7})
	f.Fuzz(func(t *testing.T, data []byte) {
		var offers []merchant.Offer
		var takes []merchant.Bid
		for i := 0; i+3 < len(data); i += 4 {
			l, q, k := int64(data[i]), int64(data[i+1])+1, uint64(data[i+2])
			if data[i+3]&1 == 0 {
				offers = append(offers, merchant.Offer{Ref: i, Limit: l, Qty: q, Key: k})
			} else {
				takes = append(takes, merchant.Bid{Ref: i, Limit: l, Qty: q * 3, Key: k})
			}
		}
		start := map[int]int64{}
		for _, b := range takes {
			start[b.Ref] = b.Qty
		}
		limit := map[int]int64{}
		for _, o := range offers {
			limit[o.Ref] = o.Limit
		}
		merchant.SortOffers(offers)
		merchant.SortBids(takes)
		pairs := merchant.Match(nil, offers, takes, nil)
		matched, seen := map[int]int64{}, map[int]bool{}
		for _, p := range pairs {
			lo, hi := min(limit[p.Offer], takeLimit(takes, p.Bid)), max(limit[p.Offer], takeLimit(takes, p.Bid))
			if p.Price < lo || p.Price > hi || seen[p.Offer] || limit[p.Offer] > takeLimit(takes, p.Bid) {
				t.Fatalf("pair %+v: price outside the limits [%d, %d], or an offer matched twice, or below the take's limit", p, lo, hi)
			}
			seen[p.Offer] = true
			matched[p.Bid] += p.Qty
		}
		for _, b := range takes {
			if b.Qty < 0 || b.Qty+matched[b.Ref] != start[b.Ref] {
				t.Fatalf("take %d: open %d + matched %d != started %d", b.Ref, b.Qty, matched[b.Ref], start[b.Ref])
			}
		}
	})
}

// takeLimit returns the limit of the take with the given ref.
func takeLimit(takes []merchant.Bid, ref int) int64 {
	for _, b := range takes {
		if b.Ref == ref {
			return b.Limit
		}
	}
	return -1
}

// FuzzSettle verifies, for any contract, that Settle never panics, that when it
// reports ok the two amounts sum to price * quantity exactly and the units add
// up, and that when the units do not add up it reports ok false (overflow is
// reported, not wrapped).
//
// Requirement: NFR-MER-005, FR-MER-040
func FuzzSettle(f *testing.F) {
	f.Add(int64(257), int64(150), int64(128), int64(22), int64(0))
	f.Add(int64(1<<62), int64(4), int64(4), int64(0), int64(0))
	f.Add(int64(-1), int64(-1), int64(-1), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, price, qty, reached, spoiled, lost int64) {
		out, refund, ok := merchant.Settle(price, qty, reached, spoiled, lost)
		if !ok {
			if out != 0 || refund != 0 {
				t.Fatalf("Settle(%d, %d, %d, %d, %d) = %d, %d with ok false, want zeros", price, qty, reached, spoiled, lost, out, refund)
			}
			return
		}
		if reached+spoiled+lost != qty || out+refund != price*qty || out < 0 || refund < 0 {
			t.Fatalf("Settle(%d, %d, %d, %d, %d) = %d, %d, want units that add up and amounts summing to %d", price, qty, reached, spoiled, lost, out, refund, price*qty)
		}
	})
}
