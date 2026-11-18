package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// roomCuts is what the RoomLimited events of a run add up to.
type roomCuts struct {
	bidsCut, bidsRejected, startsCut, startsRefused, discarded, shipmentsRefused int64
}

// addRoomCuts folds the RoomLimited events of a run.
func addRoomCuts(events []economysim.Event) roomCuts {
	var c roomCuts
	for _, e := range events {
		if rl, ok := e.(economysim.RoomLimited); ok {
			c.bidsCut, c.bidsRejected, c.startsCut, c.startsRefused = c.bidsCut+rl.BidsCut, c.bidsRejected+rl.BidsRejected, c.startsCut+rl.StartsCut, c.startsRefused+rl.StartsRefused
			c.discarded, c.shipmentsRefused = c.discarded+rl.Discarded, c.shipmentsRefused+rl.ShipmentsRefused
		}
	}
	return c
}

// reportRoomCuts records the counts of the RoomLimited events of a run under seed.
func reportRoomCuts(rep *scenReport, seed uint64, c roomCuts) {
	rep.num(seed, "bids cut for room", c.bidsCut)
	rep.num(seed, "bids rejected for room", c.bidsRejected)
	rep.num(seed, "starts cut for room", c.startsCut)
	rep.num(seed, "starts refused for room", c.startsRefused)
	rep.num(seed, "units discarded for room", c.discarded)
}

// TestScenarioS91_LimitedStapleHouseholdsAndProducers is STO-S1 (spec-storage.md 16): pack
// one with overlay/storage.toml, whose households have a base room just above one tick's
// upkeep plus one job's load, 400 ticks, seeds 1 to 10, audit on. Exact: the audit holds at
// every tick (the pledge, the reserved room and the load of every account), and the load of
// every role row is within its room after every tick. Recorded, not asserted: how often the
// room cut a bid, a start or a stock, and the households at the end of the run beside those
// of the same seed with the module off (the pack collapses under its own rules, N-352, so
// the effect of a binding room on it cannot be told from the baseline: plan 3-sto R1).
//
// Requirement: FR-STO-005, FR-STO-011, FR-STO-016, FR-STO-022, FR-STO-023, FR-STO-040
//
// Guard: a commitment that let the load pass the room for one tick would be repaired by the
// next and seen only by a check after every tick.
func TestScenarioS91_LimitedStapleHouseholdsAndProducers(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	on, off := loadValidPack(t, packOneStorage...), loadValidPack(t, packOne...)
	stoScenario(t, "one", "S91 one+storage limited households and producers, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		res := runPack(t, on, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapEvery})
		broke := ""
		for _, s := range res.Samples {
			for _, r := range s.Roles {
				if broke == "" && (r.LoadMass > r.RoomMass || r.LoadBulk > r.RoomBulk) {
					broke = fmt.Sprintf("role row %+v at tick %d", r, s.Tick)
				}
			}
		}
		checkExact(t, rep, "S91", "one", seed, "the load of every role is within its room after every tick", broke == "", "%s", broke)
		reportRoomCuts(rep, seed, addRoomCuts(res.Events))
		rep.num(seed, "households at end, module on", res.Final.Households)
		rep.num(seed, "households at end, module off", runPack(t, off, runSpec{Seed: seed, Ticks: goldenTicks}).Final.Households)
	})
}
