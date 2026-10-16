package economysim

import (
	"errors"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
)

// TestAverageClearing_StreamVector verifies the vectors of spec-world.md 19.2:
// with seed 42 at tick 7, the stream of the market (r0, grain) starts with
// a0043220bd826c9b and the market (r0, loaf) with 9f4b829a3880ffc9, and Clear
// takes exactly the three words of each book, so the next word of the stream
// is the fourth of a fresh one.
//
// Requirement: FR-CLR-022, FR-WLD-060
func TestAverageClearing_StreamVector(t *testing.T) {
	t.Parallel()
	ask := func(a AgentID) Order { return Order{Agent: a, Commodity: 0, Side: Ask, Price: 10, Quantity: 1} }
	bid := func(a AgentID) Order { return Order{Agent: a, Commodity: 0, Side: Bid, Price: 10, Quantity: 1} }
	tests := []struct {
		name       string
		commodity  int64
		first      uint64
		bids, asks []Order
		seller     AgentID
		buyer      AgentID
	}{
		{"grain: one bid and two asks", 0, 0xa0043220bd826c9b, []Order{bid(3)}, []Order{ask(1), ask(2)}, 0, 3},
		{"loaf: two bids and one ask", 1, 0x9f4b829a3880ffc9, []Order{bid(1), bid(2)}, []Order{ask(3)}, 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fresh := newStream(42)
			fresh.keyedPair(siteClearing, 7, 0, tt.commodity)
			words := [4]uint64{fresh.word(), fresh.word(), fresh.word(), fresh.word()}
			if words[0] != tt.first {
				t.Fatalf("first word of (siteClearing, 7, 0, %d) = %016x, want %016x", tt.commodity, words[0], tt.first)
			}
			s := newStream(42)
			s.keyedPair(siteClearing, 7, 0, tt.commodity)
			got := AverageClearing{}.Clear(tt.bids, tt.asks, s.rnd)
			if next := s.word(); next != words[3] {
				t.Errorf("word after Clear = %016x, want the fourth word %016x (three words consumed)", next, words[3])
			}
			// Equal limits and quantities: the lower key of the two on the
			// two-order side trades, bid or ask in input position.
			var winner AgentID
			if len(tt.asks) == 2 {
				winner = tt.asks[0].Agent
				if words[2] < words[1] {
					winner = tt.asks[1].Agent
				}
			} else {
				winner = tt.bids[0].Agent
				if words[1] < words[0] {
					winner = tt.bids[1].Agent
				}
			}
			if len(got) != 1 {
				t.Fatalf("Clear returned %v, want one trade", got)
			}
			if tt.seller == 0 && got[0].Seller != winner || tt.buyer == 0 && got[0].Buyer != winner {
				t.Errorf("Clear = %v, want the order of agent %d to trade (lowest key)", got, winner)
			}
		})
	}
}

// TestWithClearing_Rules verifies each rule of the registration: a name that
// is not an ID is ErrInvalidID, a built-in or repeated name is ErrDuplicateID
// and a nil extension is ErrNilExtension, checked in that order.
//
// Requirement: FR-WLD-002
func TestWithClearing_Rules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts []Option
		want error
	}{
		{"valid", []Option{WithClearing("custom", AverageClearing{})}, nil},
		{"built-in name", []Option{WithClearing("average", AverageClearing{})}, ErrDuplicateID},
		{"given twice", []Option{WithClearing("custom", AverageClearing{}), WithClearing("custom", AverageClearing{})}, ErrDuplicateID},
		{"nil extension", []Option{WithClearing("custom", nil)}, ErrNilExtension},
		{"empty name", []Option{WithClearing("", AverageClearing{})}, ErrInvalidID},
		{"upper case name", []Option{WithClearing("Custom", AverageClearing{})}, ErrInvalidID},
		{"invalid name wins over nil", []Option{WithClearing("Bad", nil)}, ErrInvalidID},
		{"built-in name wins over nil", []Option{WithClearing("average", nil)}, ErrDuplicateID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := applyOptions(tt.opts)
			if tt.want == nil && err != nil || tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("applyOptions(%s) = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// namedClearing is a Clearing distinguishable by its tag.
type namedClearing struct{ tag string }

func (namedClearing) Clear(_, _ []Order, _ *rand.Rand) []Trade { return nil }

// TestResolveClearing verifies the name check: "average" is the built-in, a
// registered name finds its extension, and an unknown or empty name is a
// *ValidationError at the path with rule ErrUnknownClearing. Two
// registrations are both found and the order given is kept.
//
// Requirement: FR-WLD-002, FR-CFG-038, FR-CFG-081
func TestResolveClearing(t *testing.T) {
	t.Parallel()
	o, err := applyOptions([]Option{
		WithClearing("zeta", namedClearing{"z"}),
		WithClearing("alpha", namedClearing{"a"}),
	})
	if err != nil {
		t.Fatalf("applyOptions: %v", err)
	}
	if want := []string{"zeta", "alpha"}; !slices.Equal(o.clearings.names, want) {
		t.Errorf("registered names = %v, want %v (order given)", o.clearings.names, want)
	}
	path := []string{"world", "clearing"}
	tests := []struct {
		name string
		in   string
		want Clearing
		fail bool
	}{
		{"built-in", "average", AverageClearing{}, false},
		{"first registered", "zeta", namedClearing{"z"}, false},
		{"second registered", "alpha", namedClearing{"a"}, false},
		{"unknown", "other", nil, true},
		{"empty", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveClearing(&o, tt.in, path)
			if !tt.fail {
				if err != nil || !reflect.DeepEqual(got, tt.want) {
					t.Errorf("resolveClearing(%q) = %v, %v, want %v, nil", tt.in, got, err, tt.want)
				}
				return
			}
			var ve *ValidationError
			if !errors.Is(err, ErrUnknownClearing) || !errors.As(err, &ve) {
				t.Fatalf("resolveClearing(%q) error = %v, want a *ValidationError of ErrUnknownClearing", tt.in, err)
			}
			if !slices.Equal(ve.Path, path) || ve.Value != tt.in || got != nil {
				t.Errorf("resolveClearing(%q) = %v, path %v, value %q, want nil, path %v, value %q", tt.in, got, ve.Path, ve.Value, path, tt.in)
			}
		})
	}
}

// TestBook_ReserveReceiptLimit verifies that an ask whose worst-case receipt,
// added to the account's tally, does not fit int64 is refused with
// ReasonReceiptLimit before the ledger is touched.
//
// Guard: admission's own test against MaxMoney should refuse first; this is the
// second line, and without it the tally would wrap and a later ask would be
// admitted against a negative receipt.
func TestBook_ReserveReceiptLimit(t *testing.T) {
	t.Parallel()
	var b book
	got, ok := b.reserve(nil, orderFacts{}, Order{Side: Ask, Quantity: 1}, math.MaxInt64)
	if ok || got != ReasonReceiptLimit {
		t.Errorf("reserve(ask 1, tally MaxInt64) = %q, %v, want %q, false", got, ok, ReasonReceiptLimit)
	}
}

// TestListFailure_UnknownCode verifies that every code Check returns has its
// own description and a code it does not return gets the fallback.
//
// Guard: a code added to Check without a case here would otherwise be reported
// as "unknown failure" with no test noticing.
func TestListFailure_UnknownCode(t *testing.T) {
	t.Parallel()
	for c := clearing.BadCommodity; c <= clearing.OverFilled; c++ {
		if listFailure(c) == "unknown failure" {
			t.Errorf("listFailure(%d) = fallback, want a description", c)
		}
	}
	if got := listFailure(clearing.Code(0)); got != "unknown failure" {
		t.Errorf("listFailure(0) = %q, want the fallback", got)
	}
}
