package clearing_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
)

const maxPrice = 1_000_000_000 // MaxPrice of bounds.go; the internal package cannot import the root

func offers(side []clearing.Offer) []clearing.Offer { return slices.Clone(side) }

// TestAverage_WorkedExample runs the example of CLR 11.
//
// Requirement: FR-CLR-009, FR-CLR-011, FR-CLR-012, FR-CLR-013
func TestAverage_WorkedExample(t *testing.T) {
	t.Parallel()
	bids := []clearing.Offer{{Ref: 0, Price: 12, Qty: 3}, {Ref: 1, Price: 10, Qty: 2}, {Ref: 2, Price: 7, Qty: 4}}
	asks := []clearing.Offer{{Ref: 0, Price: 6, Qty: 2}, {Ref: 1, Price: 9, Qty: 4}, {Ref: 2, Price: 11, Qty: 1}}
	// B1/A1: min(3,2)=2 at (12+6)/2=9. B1/A2: min(1,4)=1 at 10.5 -> 10.
	// B2/A2: min(2,3)=2 at 9.5 -> 10. B3 (7) against A2 (9): 7 < 9, stop.
	want := []clearing.Match{
		{BidRef: 0, AskRef: 0, Price: 9, Qty: 2},
		{BidRef: 0, AskRef: 1, Price: 10, Qty: 1},
		{BidRef: 1, AskRef: 1, Price: 10, Qty: 2},
	}
	got := clearing.Average(nil, bids, asks)
	if !slices.Equal(got, want) {
		t.Errorf("Average(%v, %v) = %v, want %v", bids, asks, got, want)
	}
}

// TestAverage_NoCross covers empty, one-sided and non-crossing books.
//
// Requirement: FR-CLR-009, FR-CLR-012
func TestAverage_NoCross(t *testing.T) {
	t.Parallel()
	one := []clearing.Offer{{Ref: 0, Price: 5, Qty: 1}}
	high := []clearing.Offer{{Ref: 0, Price: 6, Qty: 1}}
	tests := []struct {
		name       string
		bids, asks []clearing.Offer
	}{
		{"both empty", nil, nil},
		{"no bids", nil, one},
		{"no asks", one, nil},
		{"bid below ask", one, high},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := clearing.Average(nil, offers(tt.bids), offers(tt.asks))
			if len(got) != 0 {
				t.Errorf("Average(%v, %v) = %v, want none", tt.bids, tt.asks, got)
			}
		})
	}
	t.Run("equal limits cross", func(t *testing.T) {
		t.Parallel()
		got := clearing.Average(nil, offers(one), offers(one))
		want := []clearing.Match{{Price: 5, Qty: 1}}
		if !slices.Equal(got, want) {
			t.Errorf("Average(%v, %v) = %v, want %v", one, one, got, want)
		}
	})
	t.Run("appends to dst", func(t *testing.T) {
		t.Parallel()
		dst := []clearing.Match{{Price: 1, Qty: 1}}
		got := clearing.Average(dst, offers(one), offers(one))
		if len(got) != 2 || got[0] != dst[0] {
			t.Errorf("Average(%v, %v, %v) = %v, want dst kept and one appended", dst, one, one, got)
		}
	})
	t.Run("zero quantity offers are skipped", func(t *testing.T) {
		t.Parallel()
		bids := []clearing.Offer{{Ref: 0, Price: 9, Qty: 0}, {Ref: 1, Price: 8, Qty: 1}}
		asks := []clearing.Offer{{Ref: 0, Price: 1, Qty: 0}, {Ref: 1, Price: 2, Qty: 1}}
		got := clearing.Average(nil, bids, asks)
		want := []clearing.Match{{BidRef: 1, AskRef: 1, Price: 5, Qty: 1}}
		if !slices.Equal(got, want) {
			t.Errorf("Average(%v, %v) = %v, want %v", bids, asks, got, want)
		}
	})
}

// TestMidpoint covers even and odd differences, ties and the largest limits.
//
// Requirement: FR-CLR-013, FR-CLR-014
func TestMidpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b int64
		want int64
	}{
		{"equal", 7, 7, 7},
		{"even difference", 6, 12, 9},
		{"half goes down to even", 9, 12, 10}, // 10.5: neighbours 10, 11
		{"half goes up to even", 9, 10, 10},   // 9.5: neighbours 9, 10
		{"half at 1 and 2", 1, 2, 2},          // 1.5 -> 2
		{"half at 2 and 3", 2, 3, 2},          // 2.5 -> 2
		{"reversed arguments", 12, 9, 10},
		{"max and max-1", maxPrice, maxPrice - 1, maxPrice}, // MaxPrice-0.5, MaxPrice is even
		{"max and 1", maxPrice, 1, 500_000_000},             // (1e9+1)/2 = 500000000.5 -> even
		{"max and max", maxPrice, maxPrice, maxPrice},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := clearing.Midpoint(tt.a, tt.b); got != tt.want {
				t.Errorf("Midpoint(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestAverage_Order checks the total order: equal limits by key, equal keys by
// position, and independence from the slice order of the input.
//
// Requirement: FR-CLR-010
func TestAverage_Order(t *testing.T) {
	t.Parallel()
	mk := func(perm []int) ([]clearing.Offer, []clearing.Offer) {
		b := []clearing.Offer{{Ref: 0, Price: 10, Qty: 1, Key: 5}, {Ref: 1, Price: 10, Qty: 1, Key: 2}, {Ref: 2, Price: 10, Qty: 1, Key: 2}}
		a := []clearing.Offer{{Ref: 0, Price: 10, Qty: 1, Key: 9}}
		out := make([]clearing.Offer, len(b))
		for i, p := range perm {
			out[i] = b[p]
		}
		return out, a
	}
	// Keys: bid 1 and 2 share key 2, so position decides: 1, then 2, then 0.
	want := []clearing.Match{{BidRef: 1, AskRef: 0, Price: 10, Qty: 1}}
	for _, perm := range [][]int{{0, 1, 2}, {2, 1, 0}, {1, 0, 2}} {
		t.Run(fmt.Sprint(perm), func(t *testing.T) {
			t.Parallel()
			b, a := mk(perm)
			got := clearing.Average(nil, b, a)
			if !slices.Equal(got, want) {
				t.Errorf("Average(%v, %v) = %v, want %v", b, a, got, want)
			}
		})
	}
	t.Run("asks by key", func(t *testing.T) {
		t.Parallel()
		b := []clearing.Offer{{Ref: 0, Price: 10, Qty: 1}}
		a := []clearing.Offer{{Ref: 0, Price: 4, Qty: 1, Key: 8}, {Ref: 1, Price: 4, Qty: 1, Key: 3}, {Ref: 2, Price: 3, Qty: 1, Key: 9}}
		got := clearing.Average(nil, b, a)
		want := []clearing.Match{{BidRef: 0, AskRef: 2, Price: 6, Qty: 1}} // lowest limit first
		if !slices.Equal(got, want) {
			t.Errorf("Average(bid {10, 1}, %v) = %v, want %v", a, got, want)
		}
		a = []clearing.Offer{{Ref: 0, Price: 4, Qty: 1, Key: 8}, {Ref: 1, Price: 4, Qty: 1, Key: 3}}
		got = clearing.Average(nil, []clearing.Offer{{Ref: 0, Price: 10, Qty: 1}}, a)
		want = []clearing.Match{{BidRef: 0, AskRef: 1, Price: 7, Qty: 1}} // equal limits: lower key
		if !slices.Equal(got, want) {
			t.Errorf("Average(bid {10, 1}, %v) = %v, want %v", a, got, want)
		}
	})
}

// crossing computes the units that cross from the sorted limits, one entry per
// unit, independently of the matching walk.
func crossing(bids, asks []clearing.Offer) int64 {
	var bu, au []int64
	for _, o := range bids {
		for range o.Qty {
			bu = append(bu, o.Price)
		}
	}
	for _, o := range asks {
		for range o.Qty {
			au = append(au, o.Price)
		}
	}
	slices.SortFunc(bu, func(x, y int64) int { return int(y - x) })
	slices.Sort(au)
	var n int64
	for n < int64(len(bu)) && n < int64(len(au)) && bu[n] >= au[n] {
		n++
	}
	return n
}

// TestAverage_Properties checks the invariants over random books.
//
// Requirement: FR-CLR-009, FR-CLR-014, FR-CLR-015, FR-CLR-017
func TestAverage_Properties(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 300; seed++ {
		r := rand.New(rand.NewPCG(seed, seed*7919))
		gen := func(n int) []clearing.Offer {
			o := make([]clearing.Offer, n)
			for i := range o {
				o[i] = clearing.Offer{Ref: i, Price: 1 + r.Int64N(12), Qty: 1 + r.Int64N(5), Key: r.Uint64N(4)}
			}
			return o
		}
		bids, asks := gen(r.IntN(12)), gen(r.IntN(12))
		wantUnits := crossing(bids, asks)
		bl, al := map[int]clearing.Offer{}, map[int]clearing.Offer{}
		for _, o := range bids {
			bl[o.Ref] = o
		}
		for _, o := range asks {
			al[o.Ref] = o
		}
		got := clearing.Average(nil, bids, asks)
		var units int64
		bf, af := map[int]int64{}, map[int]int64{}
		for _, m := range got {
			b, a := bl[m.BidRef], al[m.AskRef]
			units += m.Qty
			bf[m.BidRef] += m.Qty
			af[m.AskRef] += m.Qty
			if b.Price < a.Price {
				t.Fatalf("seed %d: match %v has bid limit %d below ask limit %d", seed, m, b.Price, a.Price)
			}
			if m.Price < a.Price || m.Price > b.Price {
				t.Fatalf("seed %d: match %v price outside [%d, %d]", seed, m, a.Price, b.Price)
			}
		}
		if units != wantUnits {
			t.Fatalf("seed %d: matched %d units, want %d crossing", seed, units, wantUnits)
		}
		for ref, f := range bf {
			if f > bl[ref].Qty {
				t.Fatalf("seed %d: bid %d filled %d of %d", seed, ref, f, bl[ref].Qty)
			}
		}
		for ref, f := range af {
			if f > al[ref].Qty {
				t.Fatalf("seed %d: ask %d filled %d of %d", seed, ref, f, al[ref].Qty)
			}
		}
	}
}

// TestAverage_PairingFair verifies that with two bids and two asks at one
// limit and random keys, each bid is first about equally often.
//
// Requirement: FR-CLR-010
//
// Guard: pairing among equal limits must be fair (STB-L12). If the tie-break
// favoured an input position, one bid would always trade first and the
// systematic advantage would go unnoticed by every other test.
func TestAverage_PairingFair(t *testing.T) {
	t.Parallel()
	const draws = 4000
	r := rand.New(rand.NewPCG(42, 99))
	first := [2]int{}
	for range draws {
		bids := []clearing.Offer{{Ref: 0, Price: 10, Qty: 1, Key: r.Uint64()}, {Ref: 1, Price: 10, Qty: 1, Key: r.Uint64()}}
		asks := []clearing.Offer{{Ref: 0, Price: 10, Qty: 1, Key: r.Uint64()}, {Ref: 1, Price: 10, Qty: 1, Key: r.Uint64()}}
		first[clearing.Average(nil, bids, asks)[0].BidRef]++
	}
	// Binomial(4000, 1/2) has sd about 32; 200 is over six sd, so a fair
	// source passes for any fixed seed.
	if d := first[0] - draws/2; d < -200 || d > 200 {
		t.Errorf("bid 0 first %d of %d times, want about %d (tolerance 200)", first[0], draws, draws/2)
	}
}

// BenchmarkAverage measures Average over books of increasing size.
//
// Requirement: NFR-CLR-001, NFR-CLR-003
func BenchmarkAverage(b *testing.B) {
	for _, n := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			r := rand.New(rand.NewPCG(1, 2))
			bids, asks := make([]clearing.Offer, n/2), make([]clearing.Offer, n/2)
			for i := range bids {
				bids[i] = clearing.Offer{Ref: i, Price: 50 + r.Int64N(100), Qty: 1 + r.Int64N(5), Key: r.Uint64()}
				asks[i] = clearing.Offer{Ref: i, Price: 50 + r.Int64N(100), Qty: 1 + r.Int64N(5), Key: r.Uint64()}
			}
			bs, as := make([]clearing.Offer, n/2), make([]clearing.Offer, n/2)
			dst := make([]clearing.Match, 0, n)
			b.ReportAllocs()
			for b.Loop() {
				copy(bs, bids)
				copy(as, asks)
				dst = clearing.Average(dst[:0], bs, as)
			}
		})
	}
}

// TestAverage_EqualLimitsAndMalformed asserts that a bid equal to an ask
// matches at that price, and that an offer with no quantity is skipped without
// stopping the match behind it.
//
// Requirement: FR-CLR-010
func TestAverage_EqualLimitsAndMalformed(t *testing.T) {
	t.Parallel()
	got := clearing.Average(nil,
		[]clearing.Offer{{Ref: 0, Price: 5, Qty: 2}},
		[]clearing.Offer{{Ref: 0, Price: 5, Qty: 2}})
	want := []clearing.Match{{BidRef: 0, AskRef: 0, Price: 5, Qty: 2}}
	if !slices.Equal(got, want) {
		t.Errorf("Average(bid 5x2, ask 5x2) = %v, want %v", got, want)
	}
	// The top bid and the cheapest ask have no quantity; each is skipped.
	got = clearing.Average(nil,
		[]clearing.Offer{{Ref: 0, Price: 9, Qty: 0}, {Ref: 1, Price: 8, Qty: 1}},
		[]clearing.Offer{{Ref: 0, Price: 2, Qty: -1}, {Ref: 1, Price: 4, Qty: 1}})
	want = []clearing.Match{{BidRef: 1, AskRef: 1, Price: 6, Qty: 1}}
	if !slices.Equal(got, want) {
		t.Errorf("Average(empty top offers) = %v, want %v", got, want)
	}
}
