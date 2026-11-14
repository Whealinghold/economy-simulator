package ledger_test

import (
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// roomOpNames names the entry points the seeded sequence below drives, in the
// order of its switch; the test reports a name that never succeeded.
var roomOpNames = [...]string{
	"Move", "MoveReserved", "MoveLot", "MoveLotReserved", "MoveStated",
	"MoveCredited reserved", "MoveCredited pledge", "MoveCredited free",
	"Transform", "TransformCredited pledge", "Discard", "Wear", "WearItems",
	"Age", "Relocate", "Close", "ReserveRoom", "ReleaseRoom", "SetPledge", "SetRoom",
}

// TestRoom_SeededSequencesKeepTheInvariants verifies, over sixty seeded
// sequences of eighty operations that call every entry point of the ledger
// that moves, creates, destroys, wears, ages, relocates or closes a lot, that
// after each operation the load of each limited account is the sum of its
// lots, the reserved room and the pledge are not negative, the load plus the
// reserved room is within the room, the audit holds, and an operation that
// returned an error changed nothing. It runs on every `go test`, where
// FuzzRoomOps runs only its seeds, and it drives the forms FuzzRoomOps leaves
// out (MoveReserved, MoveLot, MoveLotReserved, MoveStated, WearItems,
// Relocate, Close). Every entry point must succeed at least once over the
// seeds, or the sequence would prove nothing about it.
//
// Requirement: FR-STO-005, FR-STO-006, FR-STO-007, FR-STO-040
func TestRoom_SeededSequencesKeepTheInvariants(t *testing.T) {
	t.Parallel()
	var succeeded [len(roomOpNames)]int
	for seed := uint64(1); seed <= 60; seed++ {
		rng := rand.New(rand.NewPCG(seed, 7))
		f := newRoomFix(t, 1000, rSizes, true)
		spare, err := f.l.Open(ledger.KindAgent, 0, 0)
		if err != nil {
			t.Fatalf("seed %d: Open spare: %v", seed, err)
		}
		f.limit(t, f.a0, 120, 90)
		f.limit(t, f.a1, 60, 60)
		f.limit(t, spare, 40, 40)
		ids := [3]int{f.a0, f.a1, spare}
		for step := 0; step < 80; step++ {
			acc, other := ids[rng.IntN(3)], ids[rng.IntN(3)]
			c, n := rng.IntN(rNumber), int64(rng.IntN(6))+1
			op := rng.IntN(len(roomOpNames))
			key := ledger.Key{Commodity: c}
			before, beforeSpare := roomSnapshot(t, f), f.room(t, spare)
			var err error
			switch op {
			case 0:
				err = f.l.Move(f.initial[c], acc, c, n, ledger.Initial)
			case 1:
				if err = f.l.ReserveGoods(acc, c, n); err == nil {
					if err = f.l.MoveReserved(acc, other, c, n, ledger.Trade); err != nil {
						_ = f.l.ReleaseGoods(acc, c, n)
					}
				}
			case 2:
				err = f.l.MoveLot(acc, other, key, n, ledger.Trade)
			case 3:
				if err = f.l.ReserveLot(acc, key, n); err == nil {
					if err = f.l.MoveLotReserved(acc, other, key, n, ledger.Trade); err != nil {
						_ = f.l.ReleaseLot(acc, key, n)
					}
				}
			case 4:
				err = f.l.MoveStated(f.initial[c], acc, ledger.Key{Commodity: c, Born: -1}, n, ledger.Initial)
			case 5:
				_ = f.l.ReserveRoom(other, n*2, n*3)
				if err = f.l.ReserveGoods(acc, c, n); err == nil {
					if err = f.l.MoveCredited(acc, other, c, n, ledger.Trade, true, ledger.CreditReserved); err != nil {
						_ = f.l.ReleaseGoods(acc, c, n)
					}
				}
				_ = f.l.ReleaseRoom(other, n*2, n*3)
			case 6:
				_ = f.l.SetPledge(other, n*5, n*5)
				err = f.l.MoveCredited(f.initial[c], other, c, n, ledger.Initial, false, ledger.CreditPledge)
			case 7:
				err = f.l.MoveCredited(f.initial[c], acc, c, n, ledger.Initial, false, ledger.CreditFree)
			case 8:
				err = f.l.Transform(acc, []ledger.Line{{Commodity: c, Quantity: n}}, []ledger.Line{{Commodity: (c + 1) % rNumber, Quantity: n + 1}})
			case 9:
				_ = f.l.SetPledge(acc, n*6, n*6)
				err = f.l.TransformCredited(acc, nil, []ledger.Line{{Commodity: c, Quantity: n}}, ledger.CreditPledge)
			case 10:
				err = f.l.Discard(acc, c, n)
			case 11:
				err = f.l.Wear(acc, rWear, n)
			case 12:
				err = f.l.WearItems(acc, rWear, n, 1)
			case 13:
				f.l.AdvanceTick()
				err = f.l.Age()
			case 14:
				err = f.l.Relocate(acc, rng.IntN(2))
			case 15:
				err = f.l.Close(spare)
			case 16:
				err = f.l.ReserveRoom(acc, n, n)
			case 17:
				err = f.l.ReleaseRoom(acc, n, n)
			case 18:
				err = f.l.SetPledge(acc, n*4, n*4)
			case 19:
				err = f.l.SetRoom(acc, true, n*30, n*20)
			}
			if err == nil {
				succeeded[op]++
			} else if op != 1 && op != 3 && op != 5 && op != 6 && op != 9 {
				// The compound rows above prepare state before the call, so only
				// the single-call rows are compared whole.
				if after := roomSnapshot(t, f); !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeSpare, f.room(t, spare)) {
					t.Fatalf("seed %d step %d: %s refused with %v but changed the ledger", seed, step, roomOpNames[op], err)
				}
			}
			for _, id := range ids {
				r := f.room(t, id)
				if !r.Limited {
					if r != (ledger.Room{}) {
						t.Fatalf("seed %d step %d (%s): an unlimited account holds %+v", seed, step, roomOpNames[op], r)
					}
					continue
				}
				if m, b := f.lotsLoad(t, id); r.LoadMass != m || r.LoadBulk != b {
					t.Fatalf("seed %d step %d (%s): account %d load (%d, %d), lots weigh (%d, %d)", seed, step, roomOpNames[op], id, r.LoadMass, r.LoadBulk, m, b)
				}
				if r.ReservedMass < 0 || r.ReservedBulk < 0 || r.PledgeMass < 0 || r.PledgeBulk < 0 {
					t.Fatalf("seed %d step %d (%s): negative reserved or pledge in %+v", seed, step, roomOpNames[op], r)
				}
				if r.LoadMass+r.ReservedMass > r.Mass || r.LoadBulk+r.ReservedBulk > r.Bulk {
					t.Fatalf("seed %d step %d (%s): load plus reserved above the room in %+v", seed, step, roomOpNames[op], r)
				}
			}
			if aerr := f.l.Audit(); aerr != nil {
				t.Fatalf("seed %d step %d (%s): Audit = %v", seed, step, roomOpNames[op], aerr)
			}
		}
	}
	for op, n := range succeeded {
		// MoveStated is a construction act: after EndConstruction it must be
		// refused, which the refusal check above covers; its success is the
		// end=false row of TestRoom_LoadMaintainedByEveryOp.
		if n == 0 && op != 4 {
			t.Errorf("%s never succeeded over 60 seeds: the sequence does not exercise it", roomOpNames[op])
		}
	}
}

// TestRoom_EveryOpChangesTheLedger verifies that each row of the load table
// really moves a lot or a balance, so that TestRoom_LoadMaintainedByEveryOp
// cannot pass on an operation that did nothing: the snapshot of the ledger
// before and after each real entry point differs.
//
// Guard: the table of TestRoom_LoadMaintainedByEveryOp compares the load with
// the lots after the operation. An operation that silently became a no-op
// (a wrong fixture quantity, a changed precondition) would keep the load
// equal to the lots and pass; this test fails it.
//
// Requirement: FR-STO-006
func TestRoom_EveryOpChangesTheLedger(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		op   func(f *rfix) error
	}{
		{"Move", func(f *rfix) error { return f.l.Move(f.a0, f.a1, rPlain, 3, ledger.Trade) }},
		{"MoveLot", func(f *rfix) error { return f.l.MoveLot(f.a0, f.a1, ledger.Key{Commodity: rPlain}, 2, ledger.Trade) }},
		{"Transform", func(f *rfix) error {
			return f.l.Transform(f.a0, []ledger.Line{{Commodity: rPlain, Quantity: 3}}, []ledger.Line{{Commodity: rUnit, Quantity: 2}})
		}},
		{"Discard", func(f *rfix) error { return f.l.Discard(f.a0, rPlain, 2) }},
		{"Wear", func(f *rfix) error { return f.l.Wear(f.a0, rWear, 2) }},
		{"WearItems", func(f *rfix) error { return f.l.WearItems(f.a0, rWear, 3, 1) }},
		{"Age", func(f *rfix) error {
			for range 3 {
				f.l.AdvanceTick()
			}
			return f.l.Age()
		}},
		{"Relocate", func(f *rfix) error { return f.l.Relocate(f.a0, 1) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRoomFix(t, 100, rSizes, true)
			f.limit(t, f.a0, 1000, 1000)
			f.limit(t, f.a1, 1000, 1000)
			f.give(t, f.a0, rPlain, 10)
			f.give(t, f.a0, rSpoil, 4)
			f.give(t, f.a0, rWear, 3)
			before := roomSnapshot(t, f)
			if err := tt.op(f); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if after := roomSnapshot(t, f); reflect.DeepEqual(before, after) {
				t.Errorf("%s changed nothing: the load table would pass on a no-op", tt.name)
			}
		})
	}
}
