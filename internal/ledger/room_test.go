package ledger_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The commodities of the room fixture and their sizes (mass, bulk).
const (
	rPlain  = 0 // size (2, 1)
	rSpoil  = 1 // size (1, 3), shelf life 3
	rWear   = 2 // size (3, 1), durability 5
	rZero   = 3 // size (0, 0): takes no room
	rUnit   = 4 // size (1, 1)
	rNumber = 5
)

var rSizes = []ledger.Size{{Mass: 2, Bulk: 1}, {Mass: 1, Bulk: 3}, {Mass: 3, Bulk: 1}, {}, {Mass: 1, Bulk: 1}}

// rfix is a ledger of five commodities with the sizes above, two regions, the
// boundary accounts of the goods reasons a room test uses, two agents and an
// estate.
type rfix struct {
	l          *ledger.Ledger
	a0, a1, e0 int
	initial    [rNumber]int
	disposal   [rNumber]int
}

// newRoomFix builds the fixture in its construction phase and ends it if end.
// The agents are not limited: each test gives them the room it wants.
func newRoomFix(t *testing.T, maxQty int64, sizes []ledger.Size, end bool) *rfix {
	t.Helper()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{}, {ShelfLife: 3}, {Durability: 5}, {}, {}},
		Regions:     2,
		MaxMoney:    1000,
		MaxQuantity: maxQty,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := l.SetSizes(sizes); err != nil {
		t.Fatalf("SetSizes: %v", err)
	}
	f := &rfix{l: l}
	for c := 0; c < rNumber; c++ {
		for _, r := range []ledger.Reason{ledger.Initial, ledger.RecipeInput, ledger.RecipeOutput, ledger.Wear, ledger.Spoilage, ledger.Disposal} {
			id, err := l.OpenBoundary(r, c)
			if err != nil {
				t.Fatalf("setup: OpenBoundary(%d, %d): %v", r, c, err)
			}
			if r == ledger.Initial {
				f.initial[c] = id
			}
			if r == ledger.Disposal {
				f.disposal[c] = id
			}
		}
	}
	open := func(k ledger.Kind, region int) int {
		id, err := l.Open(k, 0, region)
		if err != nil {
			t.Fatalf("setup: Open: %v", err)
		}
		return id
	}
	f.a0, f.a1, f.e0 = open(ledger.KindAgent, 0), open(ledger.KindAgent, 1), open(ledger.KindEstate, 0)
	if end {
		if err := l.EndConstruction(); err != nil {
			t.Fatalf("setup: EndConstruction: %v", err)
		}
	}
	return f
}

func (f *rfix) limit(t *testing.T, id int, mass, bulk int64) {
	t.Helper()
	if err := f.l.SetRoom(id, true, mass, bulk); err != nil {
		t.Fatalf("setup: SetRoom(%d, %d, %d): %v", id, mass, bulk, err)
	}
}

// give creates qty units of c into id as a creation under Initial.
func (f *rfix) give(t *testing.T, id, c int, qty int64) {
	t.Helper()
	if err := f.l.Move(f.initial[c], id, c, qty, ledger.Initial); err != nil {
		t.Fatalf("setup: give c%d x%d: %v", c, qty, err)
	}
}

// room returns the room of id, failing the test on an error.
func (f *rfix) room(t *testing.T, id int) ledger.Room {
	t.Helper()
	r, err := f.l.RoomOf(id)
	if err != nil {
		t.Fatalf("RoomOf(%d): %v", id, err)
	}
	return r
}

// lotsLoad recomputes the load of id from its lots and the fixture's sizes.
func (f *rfix) lotsLoad(t *testing.T, id int) (mass, bulk int64) {
	t.Helper()
	lots, err := f.l.AppendLots(nil, id)
	if err != nil {
		t.Fatalf("AppendLots(%d): %v", id, err)
	}
	for _, lt := range lots {
		mass += lt.Quantity * rSizes[lt.Commodity].Mass
		bulk += lt.Quantity * rSizes[lt.Commodity].Bulk
	}
	return mass, bulk
}

// checkLoad fails the test unless the load of every limited account is the
// sum of its lots (FR-STO-006) and the audit holds (FR-STO-040 item 1).
func (f *rfix) checkLoad(t *testing.T) {
	t.Helper()
	for _, id := range []int{f.a0, f.a1} {
		r := f.room(t, id)
		if !r.Limited {
			continue
		}
		if m, b := f.lotsLoad(t, id); r.LoadMass != m || r.LoadBulk != b {
			t.Errorf("account %d: load = (%d, %d), want the lots' (%d, %d)", id, r.LoadMass, r.LoadBulk, m, b)
		}
	}
	if err := f.l.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestRoom_CreditUnmarkedRefusedWhenNotFit verifies the unmarked form of a
// credit (4.1.3): the load must fit the free room in both dimensions, else
// ErrNoRoom. A room of (10, 10) holds 5 of a (2, 1) good by mass, and a good
// of (1, 3) by bulk: 3 of it (9), not 4 (12).
//
// Requirement: FR-STO-007
func TestRoom_CreditUnmarkedRefusedWhenNotFit(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		c    int
		qty  int64
		want error
	}{
		{"mass fills exactly", rPlain, 5, nil},
		{"mass one over", rPlain, 6, ledger.ErrNoRoom},
		{"bulk fills", rSpoil, 3, nil},
		{"bulk one over", rSpoil, 4, ledger.ErrNoRoom},
		{"size zero always fits", rZero, 99, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRoomFix(t, 100, rSizes, true)
			f.limit(t, f.a0, 10, 10)
			err := f.l.Move(f.initial[tt.c], f.a0, tt.c, tt.qty, ledger.Initial)
			if !errors.Is(err, tt.want) {
				t.Errorf("Move(c%d x%d) = %v, want %v", tt.c, tt.qty, err, tt.want)
			}
		})
	}
}

// TestRoom_CreditCountsPledgeAndReserved verifies that the free room the
// unmarked form is tested against is the room less the load, the pledge and
// the reserved room: a pledge of 4 and a reservation of 2 leave 4 of 10 mass.
//
// Requirement: FR-STO-007, FR-STO-005
func TestRoom_CreditCountsPledgeAndReserved(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	f.limit(t, f.a0, 10, 10)
	if err := f.l.SetPledge(f.a0, 4, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.l.ReserveRoom(f.a0, 2, 0); err != nil {
		t.Fatal(err)
	}
	if m, b, err := f.l.FreeRoom(f.a0); err != nil || m != 4 || b != 10 {
		t.Fatalf("FreeRoom = (%d, %d), %v, want (4, 10), nil", m, b, err)
	}
	if err := f.l.Move(f.initial[rPlain], f.a0, rPlain, 3, ledger.Initial); !errors.Is(err, ledger.ErrNoRoom) {
		t.Errorf("Move of mass 6 into a free room of 4 = %v, want ErrNoRoom", err)
	}
	if err := f.l.Move(f.initial[rPlain], f.a0, rPlain, 2, ledger.Initial); err != nil {
		t.Errorf("Move of mass 4 into a free room of 4 = %v, want nil", err)
	}
}

// settleFix is a fixture for the settlement of a bid: a1 holds 10 plain goods
// and a0, limited to (20, 20), has reserved the room of 4 of them.
func settleFix(t *testing.T) *rfix {
	t.Helper()
	f := newRoomFix(t, 100, rSizes, true)
	f.give(t, f.a1, rPlain, 10)
	f.limit(t, f.a0, 20, 20)
	if err := f.l.ReserveRoom(f.a0, 8, 4); err != nil {
		t.Fatal(err)
	}
	if err := f.l.ReserveGoods(f.a1, rPlain, 4); err != nil {
		t.Fatal(err)
	}
	return f
}

// TestRoom_FromReservedDrawsReserved verifies the settlement form: the credit
// draws the reserved room, the load rises by what the reserved room falls by,
// and the free room does not change. 4 goods of (2, 1) draw (8, 4) of the
// (8, 4) reserved.
//
// Requirement: FR-STO-007, FR-STO-009
func TestRoom_FromReservedDrawsReserved(t *testing.T) {
	t.Parallel()
	f := settleFix(t)
	freeM, freeB, _ := f.l.FreeRoom(f.a0)
	if err := f.l.MoveCredited(f.a1, f.a0, rPlain, 4, ledger.Trade, true, ledger.CreditReserved); err != nil {
		t.Fatalf("MoveCredited from reserved = %v, want nil", err)
	}
	r := f.room(t, f.a0)
	if r.ReservedMass != 0 || r.ReservedBulk != 0 || r.LoadMass != 8 || r.LoadBulk != 4 {
		t.Errorf("room after settlement = %+v, want reserved (0, 0), load (8, 4)", r)
	}
	if m, b, _ := f.l.FreeRoom(f.a0); m != freeM || b != freeB {
		t.Errorf("FreeRoom after settlement = (%d, %d), want unchanged (%d, %d)", m, b, freeM, freeB)
	}
	f.checkLoad(t)
}

// TestRoom_FromReservedExceedsIsRefused verifies that a credit marked from
// reserved that is above the reserved room is ErrReservationExceeds, not
// ErrNoRoom, and that an account that is not limited cannot take the form.
//
// Requirement: FR-STO-007
func TestRoom_FromReservedExceedsIsRefused(t *testing.T) {
	t.Parallel()
	f := settleFix(t)
	if err := f.l.ReserveGoods(f.a1, rPlain, 1); err != nil {
		t.Fatal(err)
	}
	if err := f.l.MoveCredited(f.a1, f.a0, rPlain, 5, ledger.Trade, true, ledger.CreditReserved); !errors.Is(err, ledger.ErrReservationExceeds) {
		t.Errorf("credit of 5 from a reserved room of 4 = %v, want ErrReservationExceeds", err)
	}
	if err := f.l.MoveCredited(f.a1, f.e0, rPlain, 1, ledger.EstateGoods, false, ledger.CreditReserved); !errors.Is(err, ledger.ErrNotLimited) {
		t.Errorf("credit from reserved to an account with no room = %v, want ErrNotLimited", err)
	}
}

// TestRoom_FromPledgeBoundedByRoomLeft verifies the pledge form of 4.1.3 and
// example 18.4: after a cut the room is (700, 700), the load (100, 50) and the
// pledge (940, 940). A cargo of 40 (1, 1) is credited whole, a job's output of
// (2, 2) per unit up to RoomLeft (560, 610) is 280 units, and 281 is
// ErrNoRoom although the pledge holds it. More than the pledge is
// ErrReservationExceeds.
//
// Requirement: FR-STO-007, FR-STO-013, FR-STO-015
func TestRoom_FromPledgeBoundedByRoomLeft(t *testing.T) {
	t.Parallel()
	sizes := []ledger.Size{{Mass: 2, Bulk: 1}, {}, {}, {}, {Mass: 1, Bulk: 1}}
	f := newRoomFix(t, 1000, sizes, true)
	f.limit(t, f.a0, 1200, 1200)
	f.give(t, f.a0, rPlain, 50) // after the cut the load is (100, 50)
	if err := f.l.SetPledge(f.a0, 940, 940); err != nil {
		t.Fatal(err)
	}
	f.limit(t, f.a0, 700, 700) // the cut: SetRoom does not test the pledge
	if m, b, _ := f.l.FreeRoom(f.a0); m != 0 || b != 0 {
		t.Fatalf("FreeRoom after the cut = (%d, %d), want (0, 0)", m, b)
	}
	if m, b, _ := f.l.RoomLeft(f.a0); m != 600 || b != 650 {
		t.Fatalf("RoomLeft after the cut = (%d, %d), want (600, 650)", m, b)
	}
	// The cargo: 40 of size (1, 1) from the pledge.
	if err := f.l.MoveCredited(f.initial[rUnit], f.a0, rUnit, 40, ledger.Initial, false, ledger.CreditPledge); err != nil {
		t.Fatalf("cargo credit = %v, want nil", err)
	}
	if r := f.room(t, f.a0); r.PledgeMass != 900 || r.PledgeBulk != 900 {
		t.Errorf("pledge after the cargo = (%d, %d), want (900, 900)", r.PledgeMass, r.PledgeBulk)
	}
	// The output: a good of (2, 2) cannot be created here; use two of (1, 1) per unit.
	if err := f.l.Move(f.initial[rUnit], f.a0, rUnit, 1, ledger.Initial); !errors.Is(err, ledger.ErrNoRoom) {
		t.Fatalf("a free credit into a cut room = %v, want ErrNoRoom", err)
	}
	out := []ledger.Line{{Commodity: rUnit, Quantity: 561}}
	if err := f.l.TransformCredited(f.a0, nil, out, ledger.CreditPledge); !errors.Is(err, ledger.ErrNoRoom) {
		t.Errorf("output of 561 units against RoomLeft (560, 610) = %v, want ErrNoRoom", err)
	}
	out[0].Quantity = 560
	if err := f.l.TransformCredited(f.a0, nil, out, ledger.CreditPledge); err != nil {
		t.Errorf("output of 560 units against RoomLeft (560, 610) = %v, want nil", err)
	}
	if err := f.l.TransformCredited(f.a0, nil, []ledger.Line{{Commodity: rUnit, Quantity: 1000}}, ledger.CreditPledge); !errors.Is(err, ledger.ErrReservationExceeds) && !errors.Is(err, ledger.ErrBalanceOverflow) {
		t.Errorf("output above the pledge = %v, want ErrReservationExceeds (or the holding bound first)", err)
	}
	f.checkLoad(t)
}

// TestRoom_PledgeCreditAboveThePledgeIsRefused verifies that the pledge form
// is bounded by the pledge: a pledge of (6, 6) refuses a credit of (7, 7)
// with ErrReservationExceeds even in a roomy account, and credits (6, 6).
//
// Requirement: FR-STO-007
func TestRoom_PledgeCreditAboveThePledgeIsRefused(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	f.limit(t, f.a0, 100, 100)
	if err := f.l.SetPledge(f.a0, 6, 6); err != nil {
		t.Fatal(err)
	}
	if err := f.l.MoveCredited(f.initial[rUnit], f.a0, rUnit, 7, ledger.Initial, false, ledger.CreditPledge); !errors.Is(err, ledger.ErrReservationExceeds) {
		t.Errorf("credit of 7 from a pledge of 6 = %v, want ErrReservationExceeds", err)
	}
	if err := f.l.MoveCredited(f.initial[rUnit], f.a0, rUnit, 6, ledger.Initial, false, ledger.CreditPledge); err != nil {
		t.Errorf("credit of 6 from a pledge of 6 = %v, want nil", err)
	}
}

// TestRoom_RefusedChangesNothing verifies that a credit refused for room leaves
// the whole ledger as it was (spec-ledger.md 2.6): accounts, flows, rooms.
//
// Requirement: FR-STO-007
func TestRoom_RefusedChangesNothing(t *testing.T) {
	t.Parallel()
	f := settleFix(t)
	before, roomBefore := snapshot(f.l), f.room(t, f.a0)
	attempts := map[string]error{
		"free credit":      f.l.Move(f.initial[rPlain], f.a0, rPlain, 50, ledger.Initial),
		"reserved credit":  f.l.MoveCredited(f.a1, f.a0, rPlain, 5, ledger.Trade, true, ledger.CreditReserved),
		"transform":        f.l.Transform(f.a0, nil, []ledger.Line{{Commodity: rPlain, Quantity: 50}}),
		"reserve too much": f.l.ReserveRoom(f.a0, 1000, 0),
		"release too much": f.l.ReleaseRoom(f.a0, 9, 0),
		"shrink the room":  f.l.SetRoom(f.a0, true, 1, 1),
	}
	for name, err := range attempts {
		if err == nil {
			t.Errorf("%s: want a refusal, got nil", name)
		}
	}
	if after := snapshot(f.l); !reflect.DeepEqual(before, after) {
		t.Errorf("a refused operation changed the ledger:\nbefore %+v\nafter  %+v", before, after)
	}
	if got := f.room(t, f.a0); got != roomBefore {
		t.Errorf("room after refusals = %+v, want %+v", got, roomBefore)
	}
}

// TestRoom_LoadMaintainedByEveryOp verifies that the load of a limited account
// is the sum of its lots after every operation that touches a lot: a move in
// each of its forms, a stated creation, a transform, a discard, a wear, an
// ageing, a relocation and a close. Creation and disposal are Moves with
// boundary accounts, and an estate transfer is a Move, so they are in the rows
// of Move.
//
// Requirement: FR-STO-006
func TestRoom_LoadMaintainedByEveryOp(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		end  bool
		op   func(f *rfix) error
	}{
		{"Move", true, func(f *rfix) error { return f.l.Move(f.a0, f.a1, rPlain, 3, ledger.Trade) }},
		{"Move to the estate", true, func(f *rfix) error { return f.l.Move(f.a0, f.e0, rPlain, 3, ledger.EstateGoods) }},
		{"Move to disposal", true, func(f *rfix) error {
			return f.l.Move(f.a0, mustBoundary(f, ledger.Disposal, rPlain), rPlain, 3, ledger.Disposal)
		}},
		{"MoveReserved", true, func(f *rfix) error {
			if err := f.l.ReserveGoods(f.a0, rPlain, 4); err != nil {
				return err
			}
			return f.l.MoveReserved(f.a0, f.a1, rPlain, 4, ledger.Trade)
		}},
		{"MoveLot", true, func(f *rfix) error {
			return f.l.MoveLot(f.a0, f.a1, ledger.Key{Commodity: rPlain}, 2, ledger.Trade)
		}},
		{"MoveLotReserved", true, func(f *rfix) error {
			k := ledger.Key{Commodity: rPlain}
			if err := f.l.ReserveLot(f.a0, k, 2); err != nil {
				return err
			}
			return f.l.MoveLotReserved(f.a0, f.a1, k, 2, ledger.Trade)
		}},
		{"MoveStated", false, func(f *rfix) error {
			return f.l.MoveStated(f.initial[rSpoil], f.a1, ledger.Key{Commodity: rSpoil, Born: -1}, 2, ledger.Initial)
		}},
		{"MoveCredited from reserved", true, func(f *rfix) error {
			if err := f.l.ReserveRoom(f.a1, 4, 2); err != nil {
				return err
			}
			if err := f.l.ReserveGoods(f.a0, rPlain, 2); err != nil {
				return err
			}
			return f.l.MoveCredited(f.a0, f.a1, rPlain, 2, ledger.Trade, true, ledger.CreditReserved)
		}},
		{"Transform", true, func(f *rfix) error {
			return f.l.Transform(f.a0, []ledger.Line{{Commodity: rPlain, Quantity: 3}}, []ledger.Line{{Commodity: rUnit, Quantity: 2}})
		}},
		{"TransformCredited from pledge", true, func(f *rfix) error {
			if err := f.l.SetPledge(f.a0, 10, 10); err != nil {
				return err
			}
			return f.l.TransformCredited(f.a0, nil, []ledger.Line{{Commodity: rUnit, Quantity: 4}}, ledger.CreditPledge)
		}},
		{"Discard", true, func(f *rfix) error { return f.l.Discard(f.a0, rPlain, 2) }},
		{"Wear destroys an item", true, func(f *rfix) error { return f.l.Wear(f.a0, rWear, 2) }},
		{"WearItems", true, func(f *rfix) error { return f.l.WearItems(f.a0, rWear, 3, 1) }},
		{"Age", true, func(f *rfix) error {
			for range 3 {
				f.l.AdvanceTick()
			}
			return f.l.Age()
		}},
		{"Relocate", true, func(f *rfix) error { return f.l.Relocate(f.a0, 1) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRoomFix(t, 100, rSizes, tt.end)
			f.limit(t, f.a0, 1000, 1000)
			f.limit(t, f.a1, 1000, 1000)
			k := func(c int, born, dur int64, qty int64) {
				if err := f.l.MoveStated(f.initial[c], f.a0, ledger.Key{Commodity: c, Born: born, Durability: dur}, qty, ledger.Initial); err != nil {
					t.Fatalf("setup: stated c%d: %v", c, err)
				}
			}
			if tt.end {
				// Stated creation is a construction act: create these with Move, which is the same load.
				f.give(t, f.a0, rPlain, 10)
				f.give(t, f.a0, rSpoil, 4)
				f.give(t, f.a0, rWear, 3)
				f.give(t, f.a0, rZero, 7)
				f.give(t, f.a1, rPlain, 5)
			} else {
				k(rPlain, 0, 0, 10)
			}
			f.checkLoad(t)
			if err := tt.op(f); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			f.checkLoad(t)
		})
	}
}

// mustBoundary returns the boundary account of a reason and commodity, which
// the fixture opened.
func mustBoundary(f *rfix, r ledger.Reason, c int) int {
	id, ok := f.l.Boundary(r, c)
	if !ok {
		panic("fixture lacks a boundary account")
	}
	return id
}

// TestRoom_CloseDropsTheRoom verifies that an empty limited account closes and
// leaves no room behind, and that a reserved room is an open reservation, as a
// reserved balance is (ErrOpenReservation), for Close and for Relocate.
//
// Requirement: FR-STO-010
func TestRoom_CloseDropsTheRoom(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	f.limit(t, f.a0, 10, 10)
	if err := f.l.ReserveRoom(f.a0, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := f.l.Close(f.a0); !errors.Is(err, ledger.ErrOpenReservation) {
		t.Errorf("Close with reserved room = %v, want ErrOpenReservation", err)
	}
	if err := f.l.Relocate(f.a0, 1); !errors.Is(err, ledger.ErrOpenReservation) {
		t.Errorf("Relocate with reserved room = %v, want ErrOpenReservation", err)
	}
	if !f.l.HasReservation() {
		t.Error("HasReservation() = false with reserved room, want true")
	}
	if err := f.l.ReleaseRoom(f.a0, 1, 1); err != nil {
		t.Fatal(err)
	}
	if f.l.HasReservation() {
		t.Error("HasReservation() = true after the release, want false")
	}
	if err := f.l.Close(f.a0); err != nil {
		t.Fatalf("Close = %v, want nil", err)
	}
	if r := f.room(t, f.a0); r != (ledger.Room{}) {
		t.Errorf("room of a closed account = %+v, want zero", r)
	}
}

// TestRoom_ZeroSizeTakesNoRoom verifies that a commodity of mass 0 and bulk 0
// takes no room: it is credited into a full account and into one whose room
// was cut below its pledge, and the load stays 0.
//
// Requirement: FR-STO-006
func TestRoom_ZeroSizeTakesNoRoom(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	f.limit(t, f.a0, 4, 4)
	f.give(t, f.a0, rPlain, 2) // load (4, 2): mass is full
	if err := f.l.SetPledge(f.a0, 50, 50); err != nil {
		t.Fatal(err)
	}
	f.give(t, f.a0, rZero, 90)
	if r := f.room(t, f.a0); r.LoadMass != 4 || r.LoadBulk != 2 {
		t.Errorf("load after 90 units of size 0 = (%d, %d), want (4, 2)", r.LoadMass, r.LoadBulk)
	}
	if err := f.l.ReserveRoom(f.a0, 0, 0); err != nil {
		t.Errorf("ReserveRoom of the room of a size-0 good = %v, want nil", err)
	}
}

// TestRoom_SetRoomRefusedWhenLoadExceeds verifies SetRoom: a room below
// load plus reserved is ErrNoRoom in either dimension and changes nothing; the
// pledge is not tested; a negative room is ErrInvalidAmount; an account that is
// not an agent's is ErrInvalidKind; turning the room off drops it.
//
// Requirement: FR-STO-007
func TestRoom_SetRoomRefusedWhenLoadExceeds(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	f.give(t, f.a0, rPlain, 5) // load (10, 5), before the account is limited
	for _, tt := range []struct {
		name       string
		mass, bulk int64
		want       error
	}{
		{"mass below the load", 9, 100, ledger.ErrNoRoom},
		{"bulk below the load", 100, 4, ledger.ErrNoRoom},
		{"exactly the load", 10, 5, nil},
		{"negative", -1, 5, ledger.ErrInvalidAmount},
	} {
		if err := f.l.SetRoom(f.a0, true, tt.mass, tt.bulk); !errors.Is(err, tt.want) {
			t.Errorf("%s: SetRoom(%d, %d) = %v, want %v", tt.name, tt.mass, tt.bulk, err, tt.want)
		}
	}
	if r := f.room(t, f.a0); !r.Limited || r.Mass != 10 || r.LoadMass != 10 || r.LoadBulk != 5 {
		t.Errorf("room after the table = %+v, want limited (10, 5) with load (10, 5)", r)
	}
	if err := f.l.ReserveRoom(f.a0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.l.SetRoom(f.e0, true, 1, 1); !errors.Is(err, ledger.ErrInvalidKind) {
		t.Errorf("SetRoom on an estate = %v, want ErrInvalidKind", err)
	}
	if err := f.l.SetRoom(f.a0, false, 0, 0); err != nil {
		t.Fatal(err)
	}
	if r := f.room(t, f.a0); r.Limited {
		t.Errorf("room after SetRoom(false) = %+v, want unlimited", r)
	}
	if err := f.l.SetRoom(99, true, 1, 1); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("SetRoom on an unknown account = %v, want ErrUnknownAccount", err)
	}
}

// TestRoom_SetPledgeNeverRefuses verifies 4.1.2: the pledge is not tested
// against the room, so a value far above it is accepted; a negative value and
// an account with no room are the only refusals.
//
// Requirement: FR-STO-007, FR-STO-020
func TestRoom_SetPledgeNeverRefuses(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	if err := f.l.SetPledge(f.a0, 1, 1); !errors.Is(err, ledger.ErrNotLimited) {
		t.Errorf("SetPledge on an account with no room = %v, want ErrNotLimited", err)
	}
	f.limit(t, f.a0, 10, 10)
	if err := f.l.SetPledge(f.a0, 940, 940); err != nil {
		t.Errorf("SetPledge(940, 940) in a room of 10 = %v, want nil", err)
	}
	if err := f.l.SetPledge(f.a0, -1, 0); !errors.Is(err, ledger.ErrInvalidAmount) {
		t.Errorf("SetPledge(-1, 0) = %v, want ErrInvalidAmount", err)
	}
	if m, b, err := f.l.FreeRoom(f.a0); err != nil || m != 0 || b != 0 {
		t.Errorf("FreeRoom with a pledge above the room = (%d, %d), %v, want (0, 0), nil", m, b, err)
	}
	if err := f.l.SetRoom(f.a0, true, 5, 5); err != nil {
		t.Errorf("SetRoom with a pledge above the new room = %v, want nil", err)
	}
}

// TestRoom_ReserveAndRelease verifies ReserveRoom against the free room and
// ReleaseRoom against the reserved room, and the errors of both.
//
// Requirement: FR-STO-009
func TestRoom_ReserveAndRelease(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	if err := f.l.ReserveRoom(f.a0, 1, 1); !errors.Is(err, ledger.ErrNotLimited) {
		t.Errorf("ReserveRoom with no room = %v, want ErrNotLimited", err)
	}
	if err := f.l.ReleaseRoom(f.a0, 1, 1); !errors.Is(err, ledger.ErrNotLimited) {
		t.Errorf("ReleaseRoom with no room = %v, want ErrNotLimited", err)
	}
	f.limit(t, f.a0, 10, 10)
	for _, tt := range []struct {
		name       string
		reserve    bool
		mass, bulk int64
		want       error
	}{
		{"reserve within", true, 6, 6, nil},
		{"reserve over mass", true, 5, 0, ledger.ErrNoRoom},
		{"reserve over bulk", true, 0, 5, ledger.ErrNoRoom},
		{"reserve exactly the rest", true, 4, 4, nil},
		{"negative reserve", true, -1, 0, ledger.ErrInvalidAmount},
		{"release over", false, 11, 0, ledger.ErrReservationExceeds},
		{"negative release", false, 0, -1, ledger.ErrInvalidAmount},
		{"release some", false, 3, 3, nil},
	} {
		op := f.l.ReleaseRoom
		if tt.reserve {
			op = f.l.ReserveRoom
		}
		if err := op(f.a0, tt.mass, tt.bulk); !errors.Is(err, tt.want) {
			t.Errorf("%s: (%d, %d) = %v, want %v", tt.name, tt.mass, tt.bulk, err, tt.want)
		}
	}
	if r := f.room(t, f.a0); r.ReservedMass != 7 || r.ReservedBulk != 7 {
		t.Errorf("reserved after the table = (%d, %d), want (7, 7)", r.ReservedMass, r.ReservedBulk)
	}
	if err := f.l.ReserveRoom(99, 1, 1); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("ReserveRoom on an unknown account = %v, want ErrUnknownAccount", err)
	}
}

// TestRoom_SetSizesChecks verifies SetSizes: the length, the sign, and that it
// is refused once an account is limited.
//
// Requirement: FR-STO-006
func TestRoom_SetSizesChecks(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 100, rSizes, true)
	if err := f.l.SetSizes(rSizes[:2]); !errors.Is(err, ledger.ErrInvalidConfig) {
		t.Errorf("SetSizes with a short table = %v, want ErrInvalidConfig", err)
	}
	bad := append([]ledger.Size(nil), rSizes...)
	bad[1].Bulk = -1
	if err := f.l.SetSizes(bad); !errors.Is(err, ledger.ErrInvalidConfig) {
		t.Errorf("SetSizes with a negative size = %v, want ErrInvalidConfig", err)
	}
	f.limit(t, f.a0, 10, 10)
	if err := f.l.SetSizes(rSizes); !errors.Is(err, ledger.ErrInvalidConfig) {
		t.Errorf("SetSizes after an account was limited = %v, want ErrInvalidConfig", err)
	}
}

// TestRoom_Overflow verifies the checked arithmetic of a load: a quantity at
// MaxQuantity (10^12) times a size of 10^7 is 10^19, above int64. As a credit
// it does not fit any room (ErrNoRoom); turning a room on over such a holding
// is ErrBalanceOverflow; and a load of MaxQuantity times 10^6 (10^18) is
// carried.
//
// Requirement: FR-STO-006, NFR-STO-001
func TestRoom_Overflow(t *testing.T) {
	t.Parallel()
	const maxQty = int64(1e12)
	sizes := []ledger.Size{{Mass: 1e7, Bulk: 1}, {Mass: 1e6, Bulk: 1}, {}, {}, {}}
	f := newRoomFix(t, maxQty, sizes, true)
	f.limit(t, f.a0, 1<<62, 1<<62)
	if err := f.l.Move(f.initial[0], f.a0, 0, maxQty, ledger.Initial); !errors.Is(err, ledger.ErrNoRoom) {
		t.Errorf("credit whose load is 10^19 = %v, want ErrNoRoom", err)
	}
	if err := f.l.Move(f.initial[1], f.a0, 1, maxQty, ledger.Initial); err != nil {
		t.Fatalf("credit whose load is 10^18 = %v, want nil", err)
	}
	if r := f.room(t, f.a0); r.LoadMass != 1e18 {
		t.Errorf("load = %d, want 10^18", r.LoadMass)
	}
	// An unlimited account may hold what no room could, and refuses a room over it.
	for _, c := range []int{0, 1} {
		if err := f.l.Move(f.initial[c], f.a1, c, maxQty, ledger.Initial); err != nil {
			t.Fatalf("setup: unlimited credit of c%d: %v", c, err)
		}
	}
	if err := f.l.SetRoom(f.a1, true, 1<<62, 1<<62); !errors.Is(err, ledger.ErrBalanceOverflow) {
		t.Errorf("SetRoom over a load of 1.1*10^19 = %v, want ErrBalanceOverflow", err)
	}
}

// TestRoom_WorkedExamples verifies the figures of 18.1 (a barn: room (400,
// 200), grain of size (2, 1)) and 18.2 (a depot: room (6 040, 9 060), a
// ration of size (1, 2) fits 4 530 by bulk) as credits.
//
// Requirement: FR-STO-004, FR-STO-007
func TestRoom_WorkedExamples(t *testing.T) {
	t.Parallel()
	t.Run("barn", func(t *testing.T) {
		t.Parallel()
		f := newRoomFix(t, 1000, rSizes, true)
		f.limit(t, f.a0, 400, 200)
		f.give(t, f.a0, rPlain, 120) // one harvest: load (240, 120), free (160, 80)
		if m, b, _ := f.l.FreeRoom(f.a0); m != 160 || b != 80 {
			t.Errorf("FreeRoom after a harvest = (%d, %d), want (160, 80)", m, b)
		}
		if err := f.l.Move(f.initial[rPlain], f.a0, rPlain, 81, ledger.Initial); !errors.Is(err, ledger.ErrNoRoom) {
			t.Errorf("81 more grain = %v, want ErrNoRoom (162 mass, 81 bulk: 80 fit by bulk)", err)
		}
	})
	t.Run("depot", func(t *testing.T) {
		t.Parallel()
		sizes := []ledger.Size{{Mass: 1, Bulk: 2}, {}, {}, {}, {}}
		f := newRoomFix(t, 10_000, sizes, true)
		f.limit(t, f.a0, 6040, 9060)
		if err := f.l.Move(f.initial[0], f.a0, 0, 4531, ledger.Initial); !errors.Is(err, ledger.ErrNoRoom) {
			t.Errorf("4531 rations = %v, want ErrNoRoom (bulk 9062 > 9060)", err)
		}
		if err := f.l.Move(f.initial[0], f.a0, 0, 4530, ledger.Initial); err != nil {
			t.Errorf("4530 rations = %v, want nil", err)
		}
	})
}

// TestRoom_UnlimitedCostsOneBranch verifies that the operations on a limited or
// an unlimited account allocate nothing once their lots exist: a move back and
// forth between two accounts that keep a unit of the lot, the reads of the
// free room and the room left, and a reserve and release.
//
// Requirement: NFR-STO-001
func TestRoom_UnlimitedCostsOneBranch(t *testing.T) {
	// AllocsPerRun is not run in parallel with other tests of the package.
	f := newRoomFix(t, 100, rSizes, true)
	f.give(t, f.a0, rPlain, 10)
	f.give(t, f.a1, rPlain, 10)
	for _, tt := range []struct {
		name  string
		limit bool
	}{{"unlimited", false}, {"limited", true}} {
		if tt.limit {
			f.limit(t, f.a0, 1000, 1000)
			f.limit(t, f.a1, 1000, 1000)
		}
		allocs := testing.AllocsPerRun(100, func() {
			if err := f.l.Move(f.a0, f.a1, rPlain, 1, ledger.Trade); err != nil {
				t.Fatal(err)
			}
			if err := f.l.Move(f.a1, f.a0, rPlain, 1, ledger.Trade); err != nil {
				t.Fatal(err)
			}
		})
		if allocs != 0 {
			t.Errorf("%s: a move and back allocates %v, want 0", tt.name, allocs)
		}
	}
	allocs := testing.AllocsPerRun(100, func() {
		_, _, _ = f.l.FreeRoom(f.a0)
		_, _, _ = f.l.RoomLeft(f.a0)
		if err := f.l.ReserveRoom(f.a0, 2, 1); err != nil {
			t.Fatal(err)
		}
		if err := f.l.ReleaseRoom(f.a0, 2, 1); err != nil {
			t.Fatal(err)
		}
		if err := f.l.SetPledge(f.a0, 0, 0); err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Errorf("room reads, reserve, release and set pledge allocate %v, want 0", allocs)
	}
}

// BenchmarkMoveLimited reports the cost of a move and back on an unlimited and
// on a limited pair of accounts, for the trend (NFR-STO-001).
func BenchmarkMoveLimited(b *testing.B) {
	for _, limited := range []bool{false, true} {
		name := "unlimited"
		if limited {
			name = "limited"
		}
		b.Run(name, func(b *testing.B) {
			t := &testing.T{}
			f := newRoomFix(t, 100, rSizes, true)
			f.give(t, f.a0, rPlain, 10)
			f.give(t, f.a1, rPlain, 10)
			if limited {
				f.limit(t, f.a0, 1000, 1000)
				f.limit(t, f.a1, 1000, 1000)
			}
			b.ReportAllocs()
			for b.Loop() {
				_ = f.l.Move(f.a0, f.a1, rPlain, 1, ledger.Trade)
				_ = f.l.Move(f.a1, f.a0, rPlain, 1, ledger.Trade)
			}
		})
	}
}

// TestRoom_BoundaryDestinationRefusesRoomForms verifies a destination that is a
// boundary has no room, so the forms that draw on one are ErrNotLimited there
// as they are for an agent that is not limited (4.1.3), and nothing changes.
//
// Requirement: FR-STO-007
func TestRoom_BoundaryDestinationRefusesRoomForms(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 1000, rSizes, true)
	f.give(t, f.a0, rPlain, 5)
	for _, form := range []ledger.Credit{ledger.CreditReserved, ledger.CreditPledge} {
		if err := f.l.MoveCredited(f.a0, f.disposal[rPlain], rPlain, 1, ledger.Disposal, false, form); !errors.Is(err, ledger.ErrNotLimited) {
			t.Errorf("form %d into a boundary = %v, want ErrNotLimited", form, err)
		}
	}
	if err := f.l.MoveCredited(f.a0, f.disposal[rPlain], rPlain, 1, ledger.Disposal, false, ledger.CreditFree); err != nil {
		t.Errorf("free form into a boundary = %v, want nil", err)
	}
	f.checkLoad(t)
}

// TestRoom_SetRoomResizeKeepsPledgeAndReserved pins that SetRoom on a limited
// account changes the size only.
//
// Requirement: FR-STO-004
func TestRoom_SetRoomResizeKeepsPledgeAndReserved(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 1000, rSizes, true)
	f.limit(t, f.a0, 100, 100)
	if err := f.l.SetPledge(f.a0, 7, 8); err != nil {
		t.Fatalf("setup: SetPledge: %v", err)
	}
	f.give(t, f.a0, rPlain, 3)
	f.limit(t, f.a0, 200, 150)
	r := f.room(t, f.a0)
	if r.Mass != 200 || r.Bulk != 150 || r.PledgeMass != 7 || r.PledgeBulk != 8 || r.LoadMass != 6 || r.LoadBulk != 3 {
		t.Errorf("room after resize = %+v, want size (200, 150), pledge (7, 8), load (6, 3)", r)
	}
	f.checkLoad(t)
}

// TestRoom_OverflowingCreditKeepsItsFormsRefusal verifies a credit whose load
// does not fit int64 is refused as its form says (4.1.3): a reserved credit is
// ErrReservationExceeds, an unmarked one ErrNoRoom.
//
// Requirement: FR-STO-007
func TestRoom_OverflowingCreditKeepsItsFormsRefusal(t *testing.T) {
	t.Parallel()
	const maxQty = int64(1e12)
	sizes := []ledger.Size{{Mass: 1e7, Bulk: 1}, {}, {}, {}, {}}
	f := newRoomFix(t, maxQty, sizes, true)
	f.limit(t, f.a0, 1<<62, 1<<62)
	for form, want := range map[ledger.Credit]error{ledger.CreditReserved: ledger.ErrReservationExceeds, ledger.CreditFree: ledger.ErrNoRoom} {
		if err := f.l.MoveCredited(f.initial[0], f.a0, 0, maxQty, ledger.Initial, false, form); !errors.Is(err, want) {
			t.Errorf("form %d = %v, want %v", form, err, want)
		}
	}
}

// TestRoom_OverflowingTransformKeepsItsFormsRefusal verifies a Transform whose
// output load does not fit int64 is refused as Move is (4.1.3): a reserved or
// pledge credit is ErrReservationExceeds, a free one ErrNoRoom.
//
// Requirement: FR-STO-007
func TestRoom_OverflowingTransformKeepsItsFormsRefusal(t *testing.T) {
	t.Parallel()
	const maxQty = int64(1e12)
	sizes := []ledger.Size{{Mass: 1e7, Bulk: 1}, {}, {}, {}, {}}
	f := newRoomFix(t, maxQty, sizes, true)
	f.limit(t, f.a0, 1<<62, 1<<62)
	out := []ledger.Line{{Commodity: 0, Quantity: maxQty}}
	for form, want := range map[ledger.Credit]error{ledger.CreditReserved: ledger.ErrReservationExceeds, ledger.CreditPledge: ledger.ErrReservationExceeds, ledger.CreditFree: ledger.ErrNoRoom} {
		if err := f.l.TransformCredited(f.a0, nil, out, form); !errors.Is(err, want) {
			t.Errorf("form %d = %v, want %v", form, err, want)
		}
	}
}

// TestRoom_TransformDoesNotForgiveAPledgeAboveTheRoom verifies the floor at 0
// of the free room is taken after the freed inputs are added (3.4.4): in a room
// of (10, 10) with load 5 and pledge 8 the free room is -3, so inputs of 5
// free 2, and an output of 5 is ErrNoRoom while an output of 2 fits.
//
// Fix: review of PR 1 (d8ec0f7). The free room was floored at 0 before the
// freed inputs were added, so a pledge above the room was forgiven by the load
// a Transform frees and an output that did not fit was accepted.
//
// Requirement: FR-STO-005
func TestRoom_TransformDoesNotForgiveAPledgeAboveTheRoom(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 1000, rSizes, true)
	f.limit(t, f.a0, 10, 10)
	f.give(t, f.a0, rUnit, 5)
	if err := f.l.SetPledge(f.a0, 8, 8); err != nil {
		t.Fatalf("setup: SetPledge: %v", err)
	}
	in := []ledger.Line{{Commodity: rUnit, Quantity: 5}}
	if err := f.l.Transform(f.a0, in, []ledger.Line{{Commodity: rPlain, Quantity: 2}}); !errors.Is(err, ledger.ErrNoRoom) {
		t.Errorf("output (4, 2) from freed (5, 5) with free -3 = %v, want ErrNoRoom", err)
	}
	if err := f.l.Transform(f.a0, in, []ledger.Line{{Commodity: rUnit, Quantity: 2}}); err != nil {
		t.Errorf("output (2, 2) from freed (5, 5) with free -3 = %v, want nil", err)
	}
	f.checkLoad(t)
}

// TestRoom_OverflowingCreditIsRefusedInARoomOfMaxInt64 verifies a credit whose
// load does not fit int64 is refused with its form's refusal even where the
// free room is the largest int64: the load is above any room, so it must not
// be saturated and compared with the room.
//
// Fix: DP3-STO-1. The load was saturated to MaxInt64 and fitted a free room of
// MaxInt64, so the credit passed the check and loadAdd then panicked on the
// overflow it was meant to rule out.
//
// Requirement: FR-STO-007
func TestRoom_OverflowingCreditIsRefusedInARoomOfMaxInt64(t *testing.T) {
	t.Parallel()
	const maxQty = int64(1e12)
	sizes := []ledger.Size{{Mass: 1e7, Bulk: 1}, {}, {}, {}, {}}
	f := newRoomFix(t, maxQty, sizes, true)
	f.limit(t, f.a0, math.MaxInt64, math.MaxInt64)
	for form, want := range map[ledger.Credit]error{ledger.CreditReserved: ledger.ErrReservationExceeds, ledger.CreditPledge: ledger.ErrReservationExceeds, ledger.CreditFree: ledger.ErrNoRoom} {
		if err := f.l.MoveCredited(f.initial[0], f.a0, 0, maxQty, ledger.Initial, false, form); !errors.Is(err, want) {
			t.Errorf("MoveCredited form %d = %v, want %v", form, err, want)
		}
	}
	if r := f.room(t, f.a0); r.LoadMass != 0 || r.LoadBulk != 0 {
		t.Errorf("load after the refusals = (%d, %d), want (0, 0)", r.LoadMass, r.LoadBulk)
	}
}
