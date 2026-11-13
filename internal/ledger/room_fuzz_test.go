package ledger_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// roomState is everything a room operation may change: the ledger's
// observable state, the lots of the two agents and their rooms.
type roomState struct {
	S     state
	Lots  [2][]ledger.Lot
	Rooms [2]ledger.Room
}

func roomSnapshot(t *testing.T, f *rfix) roomState {
	t.Helper()
	var rs roomState
	rs.S = snapshot(f.l)
	for i, id := range []int{f.a0, f.a1} {
		lots, err := f.l.AppendLots(nil, id)
		if err != nil {
			t.Fatal(err)
		}
		rs.Lots[i] = lots
		rs.Rooms[i] = f.room(t, id)
	}
	return rs
}

// FuzzRoomOps drives two limited accounts with a random sequence of room
// operations and moves, and asserts after each: the load is the sum of the
// lots, the reserved room and the pledge are never negative, the load plus the
// reserved room is within the room, the audit holds, and a refused operation
// changes nothing.
//
// Requirement: FR-STO-005, FR-STO-006, FR-STO-007, FR-STO-040
func FuzzRoomOps(f *testing.F) {
	f.Add([]byte{0, 1, 5, 2, 1, 3, 5, 0, 2, 3, 1, 4, 4, 0, 7, 6, 1, 1, 9, 0, 3})
	f.Add([]byte{7, 0, 9, 7, 1, 2, 8, 0, 4, 10, 1, 0, 0, 0, 200, 11, 0, 3})
	f.Fuzz(func(t *testing.T, ops []byte) {
		fx := newRoomFix(t, 1000, rSizes, true)
		ids := [2]int{fx.a0, fx.a1}
		fx.limit(t, fx.a0, 300, 200)
		fx.limit(t, fx.a1, 100, 100)
		for i := 0; i+2 < len(ops) && i < 3*60; i += 3 {
			acc, other := ids[ops[i+1]%2], ids[(ops[i+1]+1)%2]
			c, n := int(ops[i+2])%rNumber, int64(ops[i+2]%7)+1
			before := roomSnapshot(t, fx)
			var err error
			switch ops[i] % 12 {
			case 0:
				err = fx.l.Move(fx.initial[c], acc, c, n, ledger.Initial)
			case 1:
				err = fx.l.Move(acc, other, c, n, ledger.Trade)
			case 2:
				err = fx.l.ReserveRoom(acc, n, n)
			case 3:
				err = fx.l.ReleaseRoom(acc, n, n)
			case 4:
				err = fx.l.SetPledge(acc, n*10, n*10)
			case 5:
				err = fx.l.SetRoom(acc, true, n*40, n*30)
			case 6:
				err = fx.l.Discard(acc, c, n)
			case 7:
				err = fx.l.Transform(acc, []ledger.Line{{Commodity: c, Quantity: n}}, []ledger.Line{{Commodity: (c + 1) % rNumber, Quantity: n}})
			case 8:
				err = fx.l.MoveCredited(fx.initial[c], acc, c, n, ledger.Initial, false, ledger.CreditReserved)
			case 9:
				err = fx.l.TransformCredited(acc, nil, []ledger.Line{{Commodity: c, Quantity: n}}, ledger.CreditPledge)
			case 10:
				fx.l.AdvanceTick()
				err = fx.l.Age()
			case 11:
				err = fx.l.Wear(acc, rWear, n)
			}
			if err != nil && !reflect.DeepEqual(before, roomSnapshot(t, fx)) {
				t.Fatalf("step %d (op %d): refused with %v but changed the ledger", i/3, ops[i]%12, err)
			}
			for _, id := range ids {
				r := fx.room(t, id)
				m, b := fx.lotsLoad(t, id)
				if r.LoadMass != m || r.LoadBulk != b {
					t.Fatalf("step %d (op %d): load (%d, %d), lots weigh (%d, %d)", i/3, ops[i]%12, r.LoadMass, r.LoadBulk, m, b)
				}
				if r.ReservedMass < 0 || r.ReservedBulk < 0 || r.PledgeMass < 0 || r.PledgeBulk < 0 {
					t.Fatalf("step %d: negative reserved or pledge in %+v", i/3, r)
				}
				if r.LoadMass+r.ReservedMass > r.Mass || r.LoadBulk+r.ReservedBulk > r.Bulk {
					t.Fatalf("step %d (op %d): load plus reserved above the room in %+v", i/3, ops[i]%12, r)
				}
			}
			if aerr := fx.l.Audit(); aerr != nil {
				t.Fatalf("step %d (op %d): Audit = %v", i/3, ops[i]%12, aerr)
			}
		}
	})
}
