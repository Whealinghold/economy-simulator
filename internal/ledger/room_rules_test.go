package ledger_test

import (
	"errors"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// TestRoomRules_KRoomIsTheLargestKTheLedgerAccepts verifies, over 800 seeded
// random recipes and rooms, that the number of executions KRoom returns is
// what a real limited account takes: a Transform of k executions of the
// recipe succeeds, and one more execution (when k0 allows it) is ErrNoRoom.
// Every output here is exempt (yield 10 000 basis points, so quantity times E
// is a multiple of D), where the bound is exact.
//
// This is the seam of internal/room and the ledger: the rule reads the free
// room the ledger reports, and the ledger enforces the same room.
//
// Requirement: FR-STO-005, FR-STO-012
func TestRoomRules_KRoomIsTheLargestKTheLedgerAccepts(t *testing.T) {
	t.Parallel()
	pair := func(c int) room.Pair { return room.Pair{Mass: rSizes[c].Mass, Bulk: rSizes[c].Bulk} }
	cut, whole := 0, 0
	for seed := uint64(1); seed <= 800; seed++ {
		rng := rand.New(rand.NewPCG(seed, 11))
		inQty, outA, outB := int64(rng.IntN(4)+1), int64(rng.IntN(5)), int64(rng.IntN(3))
		if outA+outB == 0 {
			outA = 1
		}
		roomM, roomB := int64(rng.IntN(60)+20), int64(rng.IntN(60)+20)
		bystander, k0 := int64(rng.IntN(12)), int64(rng.IntN(8)+1)
		// attempt runs k executions on a fresh account and returns the free room it started
		// from, whether the seed's holdings fit its room, and the error of the Transform.
		attempt := func(k int64) (room.Pair, bool, error) {
			f := newRoomFix(t, 1000, rSizes, true)
			f.give(t, f.a0, rPlain, inQty*k0)
			if bystander > 0 {
				f.give(t, f.a0, rUnit, bystander)
			}
			if err := f.l.SetRoom(f.a0, true, roomM, roomB); err != nil {
				return room.Pair{}, false, nil // the holdings do not fit this room: skip the seed
			}
			fm, fb, err := f.l.FreeRoom(f.a0)
			if err != nil {
				t.Fatalf("seed %d: FreeRoom: %v", seed, err)
			}
			var outs []ledger.Line
			if outA > 0 {
				outs = append(outs, ledger.Line{Commodity: rUnit, Quantity: outA * k})
			}
			if outB > 0 {
				outs = append(outs, ledger.Line{Commodity: rSpoil, Quantity: outB * k})
			}
			err = f.l.Transform(f.a0, []ledger.Line{{Commodity: rPlain, Quantity: inQty * k}}, outs)
			if err == nil {
				f.checkLoad(t)
			}
			return room.Pair{Mass: fm, Bulk: fb}, true, err
		}
		free, ok, _ := attempt(1)
		if !ok {
			continue
		}
		whole++
		outputs := []room.Output{
			{Item: room.Item{Qty: outA, Size: pair(rUnit)}},
			{Item: room.Item{Qty: outB, Size: pair(rSpoil)}},
		}
		inputs := []room.Item{{Qty: inQty, Size: pair(rPlain)}}
		k := room.KRoom(inputs, outputs, room.Denominator, free, room.Pair{}, k0)
		if k > 0 {
			if _, _, err := attempt(k); err != nil {
				t.Errorf("seed %d: KRoom = %d but Transform of %d executions = %v, free %v, room (%d, %d), in %d out (%d, %d)", seed, k, k, err, free, roomM, roomB, inQty, outA, outB)
			}
		}
		if k < k0 {
			cut++
			if _, _, err := attempt(k + 1); !errors.Is(err, ledger.ErrNoRoom) {
				t.Errorf("seed %d: KRoom = %d but Transform of %d executions = %v, want ErrNoRoom, free %v, room (%d, %d), in %d out (%d, %d)", seed, k, k+1, err, free, roomM, roomB, inQty, outA, outB)
			}
		}
	}
	if whole < 200 || cut < 50 {
		t.Errorf("only %d usable seeds and %d where the room cut k: the comparison is not exercised", whole, cut)
	}
}

// TestRoomRules_ExcessOrderMakesACutRoomFit verifies the flow of a role
// change: SetRoom to a room below the load is refused, the order of
// 5.5.2.1 over the account's real lots names the moves to the estate, and
// once they are made SetRoom succeeds, the load is the lots, and no more was
// moved than the order named.
//
// Requirement: FR-STO-018, FR-STO-006
func TestRoomRules_ExcessOrderMakesACutRoomFit(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 1000, rSizes, true)
	f.limit(t, f.a0, 1000, 1000)
	f.give(t, f.a0, rPlain, 20) // (40, 20)
	f.give(t, f.a0, rUnit, 15)  // (15, 15)
	f.give(t, f.a0, rWear, 4)   // (12, 4)
	newMass, newBulk := int64(30), int64(30)
	if err := f.l.SetRoom(f.a0, true, newMass, newBulk); !errors.Is(err, ledger.ErrNoRoom) {
		t.Fatalf("SetRoom below the load = %v, want ErrNoRoom", err)
	}
	r := f.room(t, f.a0)
	lots, err := f.l.AppendLots(nil, f.a0)
	if err != nil {
		t.Fatal(err)
	}
	var cands []room.Candidate
	var keys []ledger.Key
	for _, lt := range lots {
		cands = append(cands, room.Candidate{Commodity: lt.Commodity, Size: room.Pair{Mass: rSizes[lt.Commodity].Mass, Bulk: rSizes[lt.Commodity].Bulk}, Lots: []int64{lt.Quantity}})
		keys = append(keys, ledger.Key{Commodity: lt.Commodity, Born: lt.Born, Durability: lt.Durability})
	}
	steps, met := room.ExcessOrder(room.Pair{Mass: r.LoadMass, Bulk: r.LoadBulk}, room.Pair{Mass: newMass, Bulk: newBulk}, cands)
	if !met || len(steps) == 0 {
		t.Fatalf("ExcessOrder = %v, %v, want a met order", steps, met)
	}
	for _, s := range steps {
		if err := f.l.MoveLot(f.a0, f.e0, keys[s.Cand], s.Units, ledger.EstateGoods); err != nil {
			t.Fatalf("MoveLot of step %+v: %v", s, err)
		}
	}
	f.checkLoad(t)
	if err := f.l.SetRoom(f.a0, true, newMass, newBulk); err != nil {
		t.Errorf("SetRoom after the order = %v, want nil", err)
	}
	if got := f.room(t, f.a0); got.LoadMass > newMass || got.LoadBulk > newBulk {
		t.Errorf("load after the order = (%d, %d), above the room (%d, %d)", got.LoadMass, got.LoadBulk, newMass, newBulk)
	}
	if est, _ := f.lotsLoad(t, f.e0); est == 0 {
		t.Error("the estate received nothing")
	}
}

// TestRoomRules_StoredLoadReadsTheLedgerLoad verifies the read of 4.1: the
// room of an account built by room.Of from a weight, a base and two stores is
// what SetRoom stores, and room.StoredLoad over the ledger's own load gives
// the part of it that sits in the stores.
//
// Requirement: FR-STO-004, FR-STO-042
func TestRoomRules_StoredLoadReadsTheLedgerLoad(t *testing.T) {
	t.Parallel()
	f := newRoomFix(t, 1000, rSizes, true)
	base := room.Pair{Mass: 10, Bulk: 6}
	stores := []room.Pair{{Mass: 20, Bulk: 5}, {Mass: 15, Bulk: 10}}
	weight := int64(3)
	rm, ok := room.Of(weight, base, stores)
	if !ok {
		t.Fatal("room.Of overflowed")
	}
	f.limit(t, f.a0, rm.Mass, rm.Bulk)
	if got := f.room(t, f.a0); got.Mass != 65 || got.Bulk != 33 {
		t.Fatalf("room = (%d, %d), want (65, 33): 3*(10, 6) + (20, 5) + (15, 10)", got.Mass, got.Bulk)
	}
	f.give(t, f.a0, rPlain, 25) // load (50, 25): the home room holds (30, 18), the stores take (20, 7)
	r := f.room(t, f.a0)
	got := room.StoredLoad(room.Pair{Mass: 35, Bulk: 15}, room.Pair{Mass: r.LoadMass, Bulk: r.LoadBulk}, base, weight)
	if want := (room.Pair{Mass: 20, Bulk: 7}); got != want {
		t.Errorf("StoredLoad = %v, want %v", got, want)
	}
}
