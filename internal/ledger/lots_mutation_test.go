package ledger

import (
	"errors"
	"testing"
)

// lotsRig is a ledger in construction with commodity 0 spoiling (shelf life 3),
// commodity 1 wearing (durability 5) and commodity 2 plain, an agent, an estate
// and the Initial and RecipeOutput boundaries of every commodity.
type lotsRig struct {
	l             *Ledger
	agent, estate int
	initial       [3]int
	output        [3]int
}

func newLotsRig(t *testing.T) *lotsRig {
	t.Helper()
	l, err := New(Config{
		Commodities: []Commodity{{ShelfLife: 3}, {Durability: 5}, {}},
		Regions:     1,
		MaxMoney:    1000,
		MaxQuantity: 100,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r := &lotsRig{l: l}
	open := func(id int, err error) int {
		t.Helper()
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		return id
	}
	for c := 0; c < 3; c++ {
		r.initial[c] = open(l.OpenBoundary(Initial, c))
		r.output[c] = open(l.OpenBoundary(RecipeOutput, c))
	}
	r.agent = open(l.Open(KindAgent, 0, 0))
	r.estate = open(l.Open(KindEstate, 0, 0))
	return r
}

// TestAppendLots_UnknownAccountIsRefused verifies that an account that does
// not exist is an error and leaves dst as it was.
//
// Guard: the error of the account lookup must reach the caller, not a nil one.
func TestAppendLots_UnknownAccountIsRefused(t *testing.T) {
	t.Parallel()
	r := newLotsRig(t)
	dst := []Lot{{Commodity: 2, Quantity: 1}}
	got, err := r.l.AppendLots(dst, 99)
	if !errors.Is(err, ErrUnknownAccount) {
		t.Fatalf("AppendLots(99) error = %v, want ErrUnknownAccount", err)
	}
	if len(got) != 1 || got[0] != dst[0] {
		t.Errorf("AppendLots(99) = %v, want dst %v unchanged", got, dst)
	}
}

// TestMoveStated_KeyOutsideTheFieldsOfACommodityThatDoesNotSpoilOrWear verifies
// that a positive born of a commodity that does not spoil and a negative
// durability of one that does not wear are ErrInvalidLot, and that a state
// refusal changes nothing.
//
// Requirement: FR-LED-043
func TestMoveStated_KeyOutsideTheFieldsOfACommodityThatDoesNotSpoilOrWear(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  Key
	}{
		// The field is 0 for the commodity: 1 is neither 0 nor in range.
		{"plain, born 1", Key{Commodity: 2, Born: 1}},
		{"wearing, born 1", Key{Commodity: 1, Born: 1, Durability: 5}},
		{"plain, durability -1", Key{Commodity: 2, Durability: -1}},
		{"spoiling, durability -1", Key{Commodity: 0, Durability: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newLotsRig(t)
			err := r.l.MoveStated(r.initial[tt.key.Commodity], r.estate, tt.key, 1, Initial)
			if !errors.Is(err, ErrInvalidLot) {
				t.Fatalf("MoveStated(%+v) = %v, want ErrInvalidLot", tt.key, err)
			}
		})
	}
}

// TestMoveStated_RefusesAReasonOtherThanInitialFromItsBoundary verifies that a
// stated key from the boundary of another reason that does fit the move (a
// recipe output) is ErrReasonMismatch, not a creation.
//
// Requirement: FR-LED-044
func TestMoveStated_RefusesAReasonOtherThanInitialFromItsBoundary(t *testing.T) {
	t.Parallel()
	r := newLotsRig(t)
	err := r.l.MoveStated(r.output[2], r.agent, Key{Commodity: 2}, 1, RecipeOutput)
	if !errors.Is(err, ErrReasonMismatch) {
		t.Fatalf("MoveStated under RecipeOutput = %v, want ErrReasonMismatch", err)
	}
	if lots, _ := r.l.AppendLots(nil, r.agent); len(lots) != 0 {
		t.Errorf("a refused MoveStated left lots %v, want none", lots)
	}
}

// TestReserve_CommodityOnePastTheLastIsUnknown verifies that the commodity
// whose index equals the number of commodities is ErrUnknownCommodity for a
// reserve and a release.
//
// Requirement: FR-LED-033
func TestReserve_CommodityOnePastTheLastIsUnknown(t *testing.T) {
	t.Parallel()
	r := newLotsRig(t)
	if err := r.l.ReserveGoods(r.agent, 3, 1); !errors.Is(err, ErrUnknownCommodity) {
		t.Errorf("ReserveGoods(commodity 3) = %v, want ErrUnknownCommodity", err)
	}
	if err := r.l.ReleaseGoods(r.agent, 3, 1); !errors.Is(err, ErrUnknownCommodity) {
		t.Errorf("ReleaseGoods(commodity 3) = %v, want ErrUnknownCommodity", err)
	}
}

// TestAdd_DoesNotHintAnAccountHoldingOnlyDurableGoods verifies that crediting
// a commodity that does not spoil leaves the expiry hint unset, and that a
// spoiling lot sets it to its last tick.
//
// Guard: a hint for a commodity with no shelf life would send Age to an
// account that has nothing to expire.
func TestAdd_DoesNotHintAnAccountHoldingOnlyDurableGoods(t *testing.T) {
	t.Parallel()
	r := newLotsRig(t)
	for _, c := range []int{1, 2} {
		if err := r.l.Move(r.initial[c], r.agent, c, 2, Initial); err != nil {
			t.Fatalf("Move c%d: %v", c, err)
		}
	}
	if a := r.l.accounts[r.agent]; a.hasHint {
		t.Fatalf("hint set (%d) after crediting only commodities without a shelf life", a.hint)
	}
	// Born in tick 0 with shelf life 3: last tick 0+3-1 = 2.
	if err := r.l.Move(r.initial[0], r.agent, 0, 2, Initial); err != nil {
		t.Fatalf("Move c0: %v", err)
	}
	if a := r.l.accounts[r.agent]; !a.hasHint || a.hint != 2 {
		t.Errorf("hint = %d (set %v), want 2 (set true)", a.hint, a.hasHint)
	}
}
