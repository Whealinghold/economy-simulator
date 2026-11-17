package economysim_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// storesLevyDoc is a money flow that charges 1 minor unit a tick for each store, over
// the storage pack: the growers of pack one hold a barn each from the start, so the
// levy has owners with no call to AddStore.
const storesLevyDoc = `
[modules.money_supply.flow.upkeep]
kind = "levy"
amount = 1
target = { kind = "stores", store = "barn" }
`

// storesOf is the stores of every class in the snapshot, the owners' count of the next
// P1.7 when nothing changes between the boundary and the step.
func storesOf(snap economysim.Snapshot) (n int64) {
	for _, r := range snap.Stores {
		n += r.Stores
	}
	return n
}

// shortOf is the money that the flows of the events of one tick did not apply: the due
// less the applied of each MoneyFlowLimited of the reason, which is also what the flow
// was due in all when it was limited.
func shortOf(events []economysim.Event, tick int64) (n int64) {
	for _, e := range events {
		if l, ok := e.(economysim.MoneyFlowLimited); ok && l.Tick == tick && l.Flow == "upkeep" {
			n += int64(l.Due) - int64(l.Applied)
		}
	}
	return n
}

// sinkOf is the total of the BoundaryFlow sink rows of a reason in the events of one
// tick, over every region.
func sinkOf(events []economysim.Event, tick int64, reason economysim.Reason) (n int64) {
	for _, e := range events {
		if b, ok := e.(economysim.BoundaryFlow); ok && b.Tick == tick && b.Reason == reason && b.Kind == economysim.KindSink {
			n += b.Total
		}
	}
	return n
}

// TestStoresLevy_ChargesEachStoreEveryTick verifies FR-STO-026 and FR-STO-027 through
// the public API, from a pack file to a running world: a levy of 1 a tick on the
// stores of the class barn, layered over the storage pack by a file, is charged on the
// stores the snapshot lists before each step, the sink row of its flow ID carries
// exactly that many minor units, no flow of its own reason is needed for the module
// (the ledger's reason is the flow's), and the audit holds for 150 ticks, across the
// role change at about tick 118 that ends stores and so lowers the charge.
//
// Requirement: FR-STO-026, FR-STO-027, FR-MON-008
//
// Guard: the unit tests call the flow at P1.7 alone; a stores target that was read
// from a stale owner list, or charged per household instead of per store, would pass
// them and charge the wrong sum once stores come and go in a stepping world.
func TestStoresLevy_ChargesEachStoreEveryTick(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(storesLevyDoc), packOneStorage...)
	rec := &stepRecorder{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithAudit(), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld(storage pack with a stores levy) error = %v, want nil", err)
	}
	const ticks = 150
	var charged, last int64
	for range ticks {
		tick, want := w.Tick(), storesOf(mustSnap(t, w))
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d = %v, want nil", tick, err)
		}
		// An owner with no money cannot pay: the flow reports it as short (agent 25 at
		// tick 77 of seed 1 held 0 and went insolvent), so the sink is the stores less it.
		if got, short := sinkOf(rec.events, tick, "upkeep"), shortOf(rec.events, tick); got+short != want {
			t.Fatalf("tick %d: the levy charged %d and left %d short, want %d (1 for each of the %d stores listed before the step)", tick, got, short, want, want)
		}
		charged, last = charged+want, want
	}
	if charged == 0 || last == 0 {
		t.Errorf("stores charged in %d ticks = %d, at the end = %d, want the growers' barns charged throughout", ticks, charged, last)
	}
}

// mustSnap is the snapshot of w at its boundary.
func mustSnap(t *testing.T, w *economysim.World) economysim.Snapshot {
	t.Helper()
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	return snap
}

// stepRecorder keeps the events of a run.
type stepRecorder struct{ events []economysim.Event }

// Observe keeps e.
func (r *stepRecorder) Observe(e economysim.Event) { r.events = append(r.events, e) }

// TestStorage_DefaultStrategyNeverFillsTheRoomWithABidTheMarketRejects verifies, over
// 400 ticks of the storage pack for each seed, what the default strategy under a room
// promises (FR-STO-023, FR-STO-024, FR-STO-036): no bid is rejected for room (the
// default sizes each bid to the free room, so the cuts and the discards are its only
// effect), every RoomLimited row has at least one of its six counts above 0, the rows
// come in ascending region then role order within a tick, the limited households of a
// row are at least its full households, and the units it discarded are no more than
// the units of goods that reached the disposal boundary in the tick.
//
// Requirement: FR-STO-023, FR-STO-024, FR-STO-036
//
// Guard: a bid that the market rejects for room would be the strategy's fault; a golden
// that records one would pass its byte comparison, and only a property over seeds says
// it must not occur.
func TestStorage_DefaultStrategyNeverFillsTheRoomWithABidTheMarketRejects(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	rows := 0
	for _, seed := range seedList(scenarioSeedCount()) {
		res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true})
		rows += checkRoomLimitedRun(t, seed, res.Events)
	}
	// The room binds in seed 1 (the golden's) and not in every seed, so the run proves
	// the module acted over the seeds together and not in each one.
	if rows == 0 {
		t.Errorf("no RoomLimited in %d ticks over any seed, want a room that binds", goldenTicks)
	}
}

// checkRoomLimitedRun asserts the properties of the room events of one run and returns
// the number of RoomLimited rows it saw.
func checkRoomLimitedRun(t *testing.T, seed uint64, events []economysim.Event) int {
	t.Helper()
	var prev *economysim.RoomLimited
	disposal, discarded := map[int64]int64{}, map[int64]int64{}
	rows := 0
	for _, e := range events {
		switch e := e.(type) {
		case economysim.OrderRejected:
			if e.Reason == economysim.ReasonNoRoom {
				t.Errorf("seed %d tick %d: the default strategy placed a bid rejected for room: %+v", seed, e.Tick, e)
			}
		case economysim.BoundaryFlow:
			if e.Commodity != economysim.NoCommodity && e.Reason == economysim.ReasonDisposal {
				disposal[e.Tick] += e.Total
			}
		case economysim.RoomLimited:
			rows++
			if msg := roomRowProblem(e, prev); msg != "" {
				t.Errorf("seed %d tick %d: RoomLimited %+v: %s", seed, e.Tick, e, msg)
			}
			discarded[e.Tick] += e.Discarded
			prev = &e
		}
	}
	for _, tick := range slices.Sorted(maps.Keys(discarded)) {
		if discarded[tick] > disposal[tick] {
			t.Errorf("seed %d tick %d: RoomLimited discarded %d units, only %d reached the disposal boundary", seed, tick, discarded[tick], disposal[tick])
		}
	}
	return rows
}

// roomRowProblem says what is wrong with a RoomLimited row, or "" when nothing is: no
// rejected bid, one of the six counts above 0, no more full households than limited
// ones, and a place after the row before it in the same tick (ascending region, then
// role).
func roomRowProblem(e economysim.RoomLimited, prev *economysim.RoomLimited) string {
	switch {
	case e.BidsRejected != 0:
		return "has a bid rejected for room"
	case e.StartsCut+e.StartsRefused+e.BidsCut+e.ShipmentsRefused+e.Discarded == 0:
		return "has no count above 0"
	case e.FullHouseholds > e.LimitedHouseholds:
		return "has more full households than limited ones"
	case prev != nil && prev.Tick == e.Tick && (e.Region < prev.Region || (e.Region == prev.Region && e.Role <= prev.Role)):
		return fmt.Sprintf("is out of order after %+v", *prev)
	}
	return ""
}
