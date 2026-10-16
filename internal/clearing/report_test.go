package clearing_test

import (
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
)

// TestReport_WorkedExample runs the report of CLR 11.
//
// Requirement: FR-CLR-019, FR-CLR-021
func TestReport_WorkedExample(t *testing.T) {
	t.Parallel()
	bids := []clearing.Offer{{Price: 12, Qty: 3}, {Price: 10, Qty: 2}, {Price: 7, Qty: 4}}
	asks := []clearing.Offer{{Price: 6, Qty: 2}, {Price: 9, Qty: 4}, {Price: 11, Qty: 1}}
	// Filled: B1 3, B2 2, B3 0; A1 2, A2 3, A3 0.
	sums := []int64{3, 2, 0, 2, 3, 0}
	ms := []clearing.Match{{Price: 9, Qty: 2}, {Price: 10, Qty: 1}, {Price: 10, Qty: 2}}
	// 2*9 + 1*10 + 2*10 = 48 over 5 units = 9.6 -> 10.
	got, ok := clearing.Report(bids, asks, sums, ms, 99, 4)
	want := clearing.Result{
		Price: 10, Age: 0, Volume: 5,
		UnmatchedBids: 4, UnmatchedAsks: 2,
		BestBid: 7, HasBestBid: true, BestAsk: 9, HasBestAsk: true,
	}
	// Unmatched asks: A2 has 1 left and A3 1, so 2; the lowest is A2's 9.
	if !ok || got != want {
		t.Errorf("Report(%v, %v, %v, %v, 99, 4) = %+v, %v, want %+v, true", bids, asks, sums, ms, got, ok, want)
	}
}

// TestReport_Cases covers the carried price, the age, the "none" flags and the
// half-to-even tie.
//
// Requirement: FR-CLR-019, FR-CLR-020, FR-CLR-021
func TestReport_Cases(t *testing.T) {
	t.Parallel()
	one := []clearing.Offer{{Price: 5, Qty: 2}}
	tests := []struct {
		name       string
		bids, asks []clearing.Offer
		sums       []int64
		ms         []clearing.Match
		prevP, age int64
		want       clearing.Result
	}{
		{"empty book carries the previous price and counts the age", nil, nil, nil, nil, 42, 3,
			clearing.Result{Price: 42, Age: 4}},
		{"seed carried on the first tick", nil, nil, nil, nil, 17, 0, clearing.Result{Price: 17, Age: 1}},
		{"bids only", one, nil, []int64{0}, nil, 8, 0,
			clearing.Result{Price: 8, Age: 1, UnmatchedBids: 2, BestBid: 5, HasBestBid: true}},
		{"asks only", nil, one, []int64{0}, nil, 8, 0,
			clearing.Result{Price: 8, Age: 1, UnmatchedAsks: 2, BestAsk: 5, HasBestAsk: true}},
		{"every order filled has no best", one, one, []int64{2, 2}, []clearing.Match{{Price: 5, Qty: 2}}, 8, 6,
			clearing.Result{Price: 5, Age: 0, Volume: 2}},
		// 1*2 + 1*3 = 5 over 2 units = 2.5 -> 2 (even).
		{"tie to the even neighbour down", nil, nil, nil, []clearing.Match{{Price: 2, Qty: 1}, {Price: 3, Qty: 1}}, 0, 0,
			clearing.Result{Price: 2, Volume: 2}},
		// 3 + 4 = 7 over 2 = 3.5 -> 4.
		{"tie to the even neighbour up", nil, nil, nil, []clearing.Match{{Price: 3, Qty: 1}, {Price: 4, Qty: 1}}, 0, 0,
			clearing.Result{Price: 4, Volume: 2}},
		{"best is the extreme of the unmatched", []clearing.Offer{{Price: 5, Qty: 1}, {Price: 9, Qty: 1}, {Price: 11, Qty: 1}},
			[]clearing.Offer{{Price: 4, Qty: 1}, {Price: 2, Qty: 1}}, []int64{0, 0, 1, 0, 0}, nil, 1, 0,
			clearing.Result{Price: 1, Age: 1, UnmatchedBids: 2, BestBid: 9, HasBestBid: true,
				UnmatchedAsks: 2, BestAsk: 2, HasBestAsk: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := clearing.Report(tt.bids, tt.asks, tt.sums, tt.ms, tt.prevP, tt.age)
			if !ok || got != tt.want {
				t.Errorf("Report(%v, %v, %v, %v, %d, %d) = %+v, %v, want %+v, true", tt.bids, tt.asks, tt.sums, tt.ms, tt.prevP, tt.age, got, ok, tt.want)
			}
		})
	}
}

// TestReport_Large accumulates trades whose products exceed int64 and expects
// the exact mean, and checks that a sum that cannot fit reports not ok.
//
// Requirement: FR-CLR-019
func TestReport_Large(t *testing.T) {
	t.Parallel()
	ms := make([]clearing.Match, 20)
	for i := range ms {
		ms[i] = clearing.Match{Price: maxPrice, Qty: maxQty}
	}
	// Each product is 10^18; twenty are 2*10^19 > MaxInt64. Volume 2*10^10.
	got, ok := clearing.Report(nil, nil, nil, ms, 1, 0)
	if !ok || got.Price != maxPrice || got.Volume != 20*maxQty {
		t.Errorf("Report(20 matches of price %d, quantity %d, 1, 0) = %+v, %v, want price %d volume %d", maxPrice, maxQty, got, ok, maxPrice, 20*maxQty)
	}
	huge := []clearing.Match{{Price: math.MaxInt64, Qty: math.MaxInt64}, {Price: math.MaxInt64, Qty: math.MaxInt64}}
	if _, ok := clearing.Report(nil, nil, nil, huge, 1, 0); ok {
		t.Errorf("Report(%+v, 1, 0) ok = true, want false: the volume does not fit int64", huge)
	}
}

// TestReport_UnmatchedOverflow asserts that Report refuses a book whose
// unmatched units on one side do not fit int64, for bids and for asks.
//
// Requirement: FR-CLR-021
func TestReport_UnmatchedOverflow(t *testing.T) {
	t.Parallel()
	// Three orders of 2^62 units sum to 3*2^62 > 2^63-1.
	side := []clearing.Offer{{Price: 5, Qty: 1 << 62}, {Price: 5, Qty: 1 << 62}, {Price: 5, Qty: 1 << 62}}
	tests := []struct {
		name       string
		bids, asks []clearing.Offer
	}{
		{"bids", side, nil},
		{"asks", nil, side},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sums := make([]int64, len(tt.bids)+len(tt.asks))
			if _, ok := clearing.Report(tt.bids, tt.asks, sums, nil, 1, 0); ok {
				t.Errorf("Report(bids %v, asks %v, no fills) ok = true, want false: the unmatched units do not fit int64", tt.bids, tt.asks)
			}
		})
	}
}

// TestReport_Bounded asserts over seeds that the published price lies between
// the lowest and the highest trade price.
//
// Requirement: FR-CLR-019
func TestReport_Bounded(t *testing.T) {
	t.Parallel()
	for seed := uint64(0); seed < 200; seed++ {
		r := rand.New(rand.NewPCG(seed, 7))
		ms := make([]clearing.Match, 1+r.IntN(8))
		lo, hi := int64(math.MaxInt64), int64(0)
		for i := range ms {
			ms[i] = clearing.Match{Price: 1 + r.Int64N(maxPrice), Qty: 1 + r.Int64N(maxQty)}
			lo, hi = min(lo, ms[i].Price), max(hi, ms[i].Price)
		}
		got, ok := clearing.Report(nil, nil, nil, ms, 1, 0)
		if !ok || got.Price < lo || got.Price > hi {
			t.Fatalf("seed %d: Report(%+v, 1, 0) = price %d (ok %v), want ok and a price in [%d, %d]", seed, ms, got.Price, ok, lo, hi)
		}
	}
}

// TestReport_Age asserts the age after no trade, a trade, no trade.
//
// Requirement: FR-CLR-020
func TestReport_Age(t *testing.T) {
	t.Parallel()
	trade := []clearing.Match{{Price: 5, Qty: 1}}
	steps := []struct {
		ms   []clearing.Match
		want int64
	}{{nil, 1}, {trade, 0}, {nil, 1}, {nil, 2}}
	price, age := int64(9), int64(0)
	for i, s := range steps {
		r, ok := clearing.Report(nil, nil, nil, s.ms, price, age)
		if !ok || r.Age != s.want {
			t.Fatalf("tick %d: Report(nil, nil, nil, %v, %d, %d) = age %d, want %d", i, s.ms, price, age, r.Age, s.want)
		}
		price, age = r.Price, r.Age
	}
}

// TestReport_AgeSaturates asserts that an age at MaxInt64 stays there after a
// tick with no trade instead of wrapping to a negative age, and that a trade
// still resets it.
//
// Requirement: FR-CLR-020
func TestReport_AgeSaturates(t *testing.T) {
	t.Parallel()
	r, ok := clearing.Report(nil, nil, nil, nil, 5, math.MaxInt64)
	if !ok || r.Age != math.MaxInt64 {
		t.Errorf("Report(nil, nil, nil, nil, 5, MaxInt64) = age %d, %v, want MaxInt64, true", r.Age, ok)
	}
	r, ok = clearing.Report(nil, nil, nil, []clearing.Match{{Price: 5, Qty: 1}}, 5, math.MaxInt64)
	if !ok || r.Age != 0 {
		t.Errorf("Report(nil, nil, nil, [one trade], 5, MaxInt64) = age %d, %v, want 0, true", r.Age, ok)
	}
}

// TestReport_ShortSumsPanics asserts that a sums slice shorter than the book
// panics with the stated invariant message and not an index error.
//
// Guard: the caller owns sums; a short one is a programmer error that must
// name itself.
func TestReport_ShortSumsPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if v, _ := recover().(string); !strings.Contains(v, "violated programmer invariant") {
			t.Errorf("Report with sums shorter than the book recovered %q, want a violated programmer invariant message", v)
		}
	}()
	o := []clearing.Offer{{Price: 5, Qty: 1}}
	clearing.Report(o, o, []int64{0}, nil, 1, 0)
}

// TestReport_BestAndMalformed asserts the best unmatched limit when several
// orders of a side are open in either order (highest bid, lowest ask), that a
// sums slice longer than the book is accepted, and that an order filled above
// its quantity (malformed) counts as no unmatched units.
//
// Requirement: FR-CLR-019, FR-CLR-021
func TestReport_BestAndMalformed(t *testing.T) {
	t.Parallel()
	bids := []clearing.Offer{{Price: 3, Qty: 1}, {Price: 5, Qty: 1}, {Price: 4, Qty: 1}}
	asks := []clearing.Offer{{Price: 9, Qty: 1}, {Price: 7, Qty: 1}, {Price: 8, Qty: 1}}
	// The longer slice has two spare words; the report must not look at them.
	sums := make([]int64, 8)
	got, ok := clearing.Report(bids, asks, sums, nil, 1, 0)
	if !ok || got.BestBid != 5 || got.BestAsk != 7 || got.UnmatchedBids != 3 || got.UnmatchedAsks != 3 {
		t.Errorf("Report(three bids, three asks, none filled) = %+v, %v, want best bid 5, best ask 7, 3 and 3 units", got, ok)
	}
	over := []clearing.Offer{{Price: 5, Qty: 2}}
	// Filled 3 of 2: the remainder is -1, which is not an unmatched unit.
	got, ok = clearing.Report(over, nil, []int64{3}, nil, 1, 0)
	if !ok || got.UnmatchedBids != 0 || got.HasBestBid {
		t.Errorf("Report(a bid of 2 filled 3) = %+v, %v, want no unmatched bids", got, ok)
	}
}
