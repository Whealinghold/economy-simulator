package clearing_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
)

// TestCheck_LotGridResolvesEveryLot verifies that, over every combination of
// Named, Born and Durability (2 x 3 x 3 = 18 lots) for two owners, shuffled in
// many orders, each fill resolves to exactly the ask of its own lot.
//
// Requirement: FR-CLR-024, FR-CLR-029
//
// Guard: a lot comparator that is not antisymmetric on one field (a branch
// forced, or a result turned to 0 when only Named, Born or Durability differs)
// misplaces a lot in the sorted index, so a fill finds another ask or none.
// Eighteen lots per owner exceed the insertion-sort cutoff of the sort.
func TestCheck_LotGridResolvesEveryLot(t *testing.T) {
	t.Parallel()
	var base []clearing.Offer
	for _, owner := range []int64{3, 4} {
		for _, named := range []bool{false, true} {
			for born := int64(0); born < 3; born++ {
				for dur := int64(0); dur < 3; dur++ {
					base = append(base, clearing.Offer{Owner: owner, Price: 5, Qty: 1,
						Lot: clearing.Lot{Named: named, Born: born, Durability: dur}})
				}
			}
		}
	}
	for seed := range 30 {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			r := rand.New(rand.NewPCG(uint64(seed), 7))
			perm := r.Perm(len(base))
			asks := make([]clearing.Offer, len(base))
			for i, j := range perm {
				asks[i] = base[j]
				asks[i].Ref = i
			}
			bids := []clearing.Offer{{Ref: 0, Owner: 1, Price: 9, Qty: int64(len(base))}}
			// One fill per ask, in a second shuffled order, so a lookup is
			// not helped by the order of the index.
			fills := make([]clearing.Fill, len(asks))
			for i, k := range r.Perm(len(asks)) {
				fills[i] = fill(1, asks[k].Owner, asks[k].Lot, 6, 1)
			}
			got, code := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks, fills)
			if code != 0 || len(got) != len(fills) {
				t.Fatalf("Check(%d asks, one fill each) = (%d matches, code %d), want all and 0", len(asks), len(got), code)
			}
			for i, m := range got {
				if want := asks[perm2(fills[i], asks)].Ref; m.AskRef != want {
					t.Fatalf("fill %d (%+v) resolved to AskRef %d, want %d", i, fills[i], m.AskRef, want)
				}
			}
			// A lot nobody asks for is no order, whatever its neighbours.
			miss := fill(1, 3, clearing.Lot{Named: true, Born: 1, Durability: 7}, 6, 1)
			if _, code := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks, []clearing.Fill{miss}); code != clearing.NoOrder {
				t.Errorf("Check(fill of absent lot) code = %d, want %d", code, clearing.NoOrder)
			}
		})
	}
}

// perm2 returns the position in asks of the ask a fill names.
func perm2(f clearing.Fill, asks []clearing.Offer) int {
	for i, a := range asks {
		if a.Owner == f.Seller && a.Lot == f.Lot {
			return i
		}
	}
	return -1
}

// TestCheck_ShortScratchEitherSlice verifies that a scratch whose Sums, or
// whose Index, is one short of the book panics, even when the slice has spare
// capacity so that re-slicing it would not.
//
// Requirement: FR-CLR-024
//
// Guard: the length guard must trip on either slice alone (the book has 5 orders).
func TestCheck_ShortScratchEitherSlice(t *testing.T) {
	t.Parallel()
	bids, asks := twoSided()
	const n = 5 // 2 bids + 3 asks
	tests := []struct {
		name string
		sc   clearing.Scratch
	}{
		{"Sums short", clearing.Scratch{Sums: make([]int64, n+2)[:n-1], Index: make([]int32, n)}},
		{"Index short", clearing.Scratch{Sums: make([]int64, n), Index: make([]int32, n+2)[:n-1]}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if v, _ := recover().(string); v == "" || len(v) < 8 {
					t.Errorf("Check with %s recovered %q, want the invariant message", tt.name, v)
				}
			}()
			clearing.Check(nil, tt.sc, book, bids, asks, nil)
		})
	}
}

// TestCheck_LotsDifferingInOneFieldOnly verifies that, for one owner whose
// three asks differ only in Born (or only in Durability), in every input order,
// each fill resolves to the ask of its own lot.
//
// Requirement: FR-CLR-024, FR-CLR-029
//
// Guard: with one field differing, the comparator must return -1 for the
// smaller and 1 for the larger. A comparator that returns 1 both ways, or 0
// for the larger, leaves the index unsorted or ties lots, so the binary search
// misses the ask. Hand check, asks in order [dur 2, dur 1, dur 0]: a comparator
// that returns 0 for a>b sorts nothing, and the search for dur 2 compares
// dur 1 < dur 2, moves right and falls off the end.
func TestCheck_LotsDifferingInOneFieldOnly(t *testing.T) {
	t.Parallel()
	perms := [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	fields := []struct {
		name string
		lot  func(v int64) clearing.Lot
	}{
		{"Born", func(v int64) clearing.Lot { return clearing.Lot{Named: true, Born: v} }},
		{"Durability", func(v int64) clearing.Lot { return clearing.Lot{Named: true, Durability: v} }},
	}
	for _, f := range fields {
		for _, p := range perms {
			t.Run(fmt.Sprintf("%s %v", f.name, p), func(t *testing.T) {
				t.Parallel()
				asks := make([]clearing.Offer, 3)
				for i, v := range p {
					asks[i] = clearing.Offer{Ref: i, Owner: 3, Price: 5, Qty: 1, Lot: f.lot(int64(v))}
				}
				bids := []clearing.Offer{{Ref: 0, Owner: 1, Price: 9, Qty: 3}}
				for i, a := range asks {
					got, code := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks,
						[]clearing.Fill{fill(1, 3, a.Lot, 6, 1)})
					if code != 0 || len(got) != 1 || got[0].AskRef != i {
						t.Errorf("Check(fill of ask %d, lot %+v) = (%+v, code %d), want AskRef %d and 0", i, a.Lot, got, code, i)
					}
				}
				// A lot below every ask (value 0 against 1..3 asks shifted up)
				// is no order: the search ends on a larger lot, which must
				// not compare equal.
				for i := range asks {
					asks[i].Lot = f.lot(int64(p[i]) + 1)
				}
				_, code := clearing.Check(nil, scratchFor(bids, asks), book, bids, asks,
					[]clearing.Fill{fill(1, 3, f.lot(0), 6, 1)})
				if code != clearing.NoOrder {
					t.Errorf("Check(fill of absent smaller lot) code = %d, want %d", code, clearing.NoOrder)
				}
			})
		}
	}
}
