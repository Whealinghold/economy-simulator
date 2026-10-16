package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// countSource is a rand.Source that returns the words of next and counts them.
type countSource struct {
	n    int
	next func(n int) uint64
}

func (c *countSource) Uint64() uint64 {
	w := c.next(c.n)
	c.n++
	return w
}

// worked is the book of spec-clearing.md 11: bids B1 12x3, B2 10x2, B3 7x4 and
// asks A1 6x2, A2 9x4, A3 11x1. A2 names a lot so the lot of the ask is seen
// on its trades.
func worked() (bids, asks []economysim.Order) {
	lot := economysim.LotKey{Named: true, Born: 5, Durability: 3}
	bids = []economysim.Order{
		{Agent: 1, Commodity: 2, Side: economysim.Bid, Price: 12, Quantity: 3},
		{Agent: 2, Commodity: 2, Side: economysim.Bid, Price: 10, Quantity: 2},
		{Agent: 3, Commodity: 2, Side: economysim.Bid, Price: 7, Quantity: 4},
	}
	asks = []economysim.Order{
		{Agent: 4, Commodity: 2, Side: economysim.Ask, Price: 6, Quantity: 2},
		{Agent: 5, Commodity: 2, Side: economysim.Ask, Price: 9, Quantity: 4, Lot: lot},
		{Agent: 6, Commodity: 2, Side: economysim.Ask, Price: 11, Quantity: 1},
	}
	return bids, asks
}

// TestAverageClearing_WorkedExample verifies that the default clearing returns
// the three trades of the worked example, in order, with buyer, seller, price,
// quantity and the lot of the ask.
//
// Requirement: FR-CLR-010
func TestAverageClearing_WorkedExample(t *testing.T) {
	t.Parallel()
	bids, asks := worked()
	lot := economysim.LotKey{Named: true, Born: 5, Durability: 3}
	// B1 and A1: min(3,2)=2 at (12+6)/2=9. B1 and A2: min(1,4)=1 at 10.5, half
	// to even 10. B2 and A2: min(2,3)=2 at 9.5, half to even 10. B3 (7) is
	// below A2 (9), so the match stops.
	want := []economysim.Trade{
		{Commodity: 2, Buyer: 1, Seller: 4, Price: 9, Quantity: 2},
		{Commodity: 2, Buyer: 1, Seller: 5, Price: 10, Quantity: 1, Lot: lot},
		{Commodity: 2, Buyer: 2, Seller: 5, Price: 10, Quantity: 2, Lot: lot},
	}
	got := economysim.AverageClearing{}.Clear(bids, asks, rand.New(rand.NewPCG(1, 2)))
	if !slices.Equal(got, want) {
		t.Errorf("Clear(worked book) = %v, want %v", got, want)
	}
}

// TestAverageClearing_DrawsOneWordPerOrder verifies that Clear takes exactly
// one word per order, bids first and then asks, each in input position, by
// giving ascending and descending words to equal-limit orders and seeing which
// position wins the tie.
//
// Requirement: FR-CLR-022, FR-CLR-010
func TestAverageClearing_DrawsOneWordPerOrder(t *testing.T) {
	t.Parallel()
	mk := func(side economysim.Side, agent economysim.AgentID) economysim.Order {
		return economysim.Order{Agent: agent, Commodity: 1, Side: side, Price: 10, Quantity: 1}
	}
	// One unit bid by agent 1 against two equal asks 2 and 3: the bid draws
	// word 0 and the asks words 1 and 2, so the lower key wins the ask tie.
	// A bid pair is the mirror: two equal bids 1 and 2 draw words 0 and 1.
	asc := func(n int) uint64 { return uint64(n) + 1 }
	desc := func(n int) uint64 { return ^uint64(0) - uint64(n) }
	tests := []struct {
		name       string
		bids, asks []economysim.Order
		next       func(int) uint64
		draws      int
		seller     economysim.AgentID
		buyer      economysim.AgentID
	}{
		{"asks ascending keys, first ask wins", []economysim.Order{mk(economysim.Bid, 1)},
			[]economysim.Order{mk(economysim.Ask, 2), mk(economysim.Ask, 3)}, asc, 3, 2, 1},
		{"asks descending keys, second ask wins", []economysim.Order{mk(economysim.Bid, 1)},
			[]economysim.Order{mk(economysim.Ask, 2), mk(economysim.Ask, 3)}, desc, 3, 3, 1},
		{"bids ascending keys, first bid wins", []economysim.Order{mk(economysim.Bid, 1), mk(economysim.Bid, 2)},
			[]economysim.Order{mk(economysim.Ask, 3)}, asc, 3, 3, 1},
		{"bids descending keys, second bid wins", []economysim.Order{mk(economysim.Bid, 1), mk(economysim.Bid, 2)},
			[]economysim.Order{mk(economysim.Ask, 3)}, desc, 3, 3, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := &countSource{next: tt.next}
			got := economysim.AverageClearing{}.Clear(tt.bids, tt.asks, rand.New(src))
			if src.n != tt.draws {
				t.Errorf("Clear drew %d words, want %d (one per order)", src.n, tt.draws)
			}
			if len(got) != 1 || got[0].Buyer != tt.buyer || got[0].Seller != tt.seller {
				t.Errorf("Clear = %v, want one trade of buyer %d and seller %d", got, tt.buyer, tt.seller)
			}
		})
	}
}

// TestAverageClearing_Deterministic verifies that one seed and one input order
// give the same trades on two runs.
//
// Requirement: FR-CLR-010
func TestAverageClearing_Deterministic(t *testing.T) {
	t.Parallel()
	bids, asks := worked()
	a := economysim.AverageClearing{}.Clear(bids, asks, rand.New(rand.NewPCG(7, 7)))
	b := economysim.AverageClearing{}.Clear(bids, asks, rand.New(rand.NewPCG(7, 7)))
	if !slices.Equal(a, b) {
		t.Errorf("two runs of Clear gave %v and %v, want equal", a, b)
	}
}

// loudClearing is a consumer Clearing that delegates to the default.
type loudClearing struct{ inner economysim.AverageClearing }

func (l loudClearing) Clear(bids, asks []economysim.Order, rng *rand.Rand) []economysim.Trade {
	return l.inner.Clear(bids, asks, rng)
}

// A consumer wraps AverageClearing and builds the option that registers it.
// The world that applies the option is the world row's.
func ExampleWithClearing() {
	var c economysim.Clearing = loudClearing{}
	opt := economysim.WithClearing("loud", c)
	bids := []economysim.Order{{Agent: 1, Commodity: 0, Side: economysim.Bid, Price: 12, Quantity: 2}}
	asks := []economysim.Order{{Agent: 2, Commodity: 0, Side: economysim.Ask, Price: 6, Quantity: 1}}
	for _, tr := range c.Clear(bids, asks, rand.New(rand.NewPCG(1, 2))) {
		fmt.Println(tr.Buyer, tr.Seller, tr.Price, tr.Quantity)
	}
	fmt.Println(opt != nil)
	// Output:
	// 1 2 9 1
	// true
}
