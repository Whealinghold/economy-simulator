package ledger_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestRoom_ReadsOfAnAccountThatHasNoRoom verifies the error and zero forms of
// the reads: an id outside the table is ErrUnknownAccount for every read, an
// account that is not limited is ErrNotLimited for the free room and the room
// left, and RoomOf returns the zero Room for it without an error.
//
// Requirement: FR-STO-003
func TestRoom_ReadsOfAnAccountThatHasNoRoom(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	for _, id := range []int{-1, 1 << 20} {
		if _, err := f.l.RoomOf(id); !errors.Is(err, ledger.ErrUnknownAccount) {
			t.Errorf("RoomOf(%d) = %v, want ErrUnknownAccount", id, err)
		}
		if _, _, err := f.l.FreeRoom(id); !errors.Is(err, ledger.ErrUnknownAccount) {
			t.Errorf("FreeRoom(%d) = %v, want ErrUnknownAccount", id, err)
		}
		if _, _, err := f.l.RoomLeft(id); !errors.Is(err, ledger.ErrUnknownAccount) {
			t.Errorf("RoomLeft(%d) = %v, want ErrUnknownAccount", id, err)
		}
	}
	if r, err := f.l.RoomOf(f.a0); err != nil || r != (ledger.Room{}) {
		t.Errorf("RoomOf(unlimited agent) = %+v, %v, want the zero Room, nil", r, err)
	}
	if _, _, err := f.l.FreeRoom(f.a0); !errors.Is(err, ledger.ErrNotLimited) {
		t.Errorf("FreeRoom(unlimited agent) = %v, want ErrNotLimited", err)
	}
	if _, _, err := f.l.RoomLeft(f.a0); !errors.Is(err, ledger.ErrNotLimited) {
		t.Errorf("RoomLeft(unlimited agent) = %v, want ErrNotLimited", err)
	}
	if err := f.l.ReserveRoom(f.a0, 1, 1); !errors.Is(err, ledger.ErrNotLimited) {
		t.Errorf("ReserveRoom(unlimited agent) = %v, want ErrNotLimited", err)
	}
	if err := f.l.SetPledge(f.a0, 1, 1); !errors.Is(err, ledger.ErrNotLimited) {
		t.Errorf("SetPledge(unlimited agent) = %v, want ErrNotLimited", err)
	}
}

// TestRoom_LedgerWithoutSizesLoadsNothing verifies that a ledger whose sizes
// were never set treats every commodity as size 0: a limited account takes any
// quantity, its load stays (0, 0), and the audit holds.
//
// Guard: the module off leaves the sizes unset. A limited account in that
// ledger (a test double of a consumer) must not read a missing table as an
// index error or as a unit of room.
//
// Requirement: FR-STO-006
func TestRoom_LedgerWithoutSizesLoadsNothing(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: 100, MaxQuantity: 100})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	b, _ := l.OpenBoundary(ledger.Initial, 0)
	a, _ := l.Open(ledger.KindAgent, 0, 0)
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	if err := l.SetRoom(a, true, 1, 1); err != nil {
		t.Fatalf("SetRoom: %v", err)
	}
	if err := l.Move(b, a, 0, 100, ledger.Initial); err != nil {
		t.Fatalf("Move of 100 units without sizes = %v, want nil", err)
	}
	if r, _ := l.RoomOf(a); r.LoadMass != 0 || r.LoadBulk != 0 {
		t.Errorf("load = (%d, %d), want (0, 0)", r.LoadMass, r.LoadBulk)
	}
	if err := l.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestRoom_CloseOfAnAccountThatHoldsGoodsKeepsItsLoad verifies that Close of a
// limited account that still holds goods is ErrNotEmpty and leaves the room,
// the load and the lots as they were.
//
// Requirement: FR-STO-006
func TestRoom_CloseOfAnAccountThatHoldsGoodsKeepsItsLoad(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	f.limit(t, f.a0, 100, 100)
	f.give(t, f.a0, rPlain, 5)
	before := roomSnapshot(t, f)
	if err := f.l.Close(f.a0); !errors.Is(err, ledger.ErrNotEmpty) {
		t.Fatalf("Close of an account with goods = %v, want ErrNotEmpty", err)
	}
	if after := roomSnapshot(t, f); !reflect.DeepEqual(before, after) {
		t.Error("a refused Close changed the ledger")
	}
	f.checkLoad(t)
}
