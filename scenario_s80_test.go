package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// s80Phase is the length of each of the three phases of S80 (open, closed, reopened).
const s80Phase = 60

// TestScenarioS80_ClosedRoad is INT-S2 as the test of INT-A1 without bands
// (spec-integration.md 16): on `three` with the merchants, transport, information and
// trade overlays the public SetEdgeStatuses closes the link between south and north,
// both ways, after s80Phase ticks and reopens it after s80Phase more, with the audit on.
// Exact: the events are two EdgeStatusChanged for the closure then two for the
// reopening, in that order and stamped with the tick of the call; Slots lists the two
// edges while the road is closed and nothing after the reopening. Reported: the gap
// between the staple price of north and of south (absolute) at the end of the open and
// the closed phase and its lowest value in the reopened phase. The gap must open (the
// closed end at or above the open end) and close (the lowest reopened value at or below
// the closed end) in direction; its size is plan 19's band.
//
// Requirement: FR-INT-041, FR-INT-042, FR-INT-048, FR-REG-049
//
// Guard: a closure that was not listed by Slots, a reopening that left a slot behind, or
// a gap that did not follow the road would show here.
func TestScenarioS80_ClosedRoad(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packMerchants("three")...)
	south, north := regionNamed(t, cfg, "south"), regionNamed(t, cfg, "north")
	staple := commodityID(t, cfg, "staple")
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			scenSem <- struct{}{}
			defer func() { <-scenSem }()
			rec := &recorder{}
			opts := append(strategy.Options(), economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
			w, err := economysim.NewWorld(cfg, opts...)
			if err != nil {
				t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
			}
			gap := func() int64 {
				a, errA := w.Market(north, staple)
				b, errB := w.Market(south, staple)
				if errA != nil || errB != nil {
					t.Fatalf("seed %d: Market() errors = %v, %v, want nil", seed, errA, errB)
				}
				return max(int64(a.Price)-int64(b.Price), int64(b.Price)-int64(a.Price))
			}
			set := func(closed bool) {
				road := []economysim.EdgeStatus{{From: south, To: north, Closed: closed, FactorBP: 10_000}, {From: north, To: south, Closed: closed, FactorBP: 10_000}}
				if err := w.SetEdgeStatuses(road); err != nil {
					t.Fatalf("seed %d: SetEdgeStatuses(closed=%v) error = %v, want nil", seed, closed, err)
				}
			}
			step(t, w, s80Phase)
			open := gap()
			set(true)
			if slots, err := w.Slots(); err != nil || len(slots.EdgeStatuses) != 2 {
				t.Errorf("seed %d: Slots().EdgeStatuses while closed = %v (err %v), want the two edges", seed, slots.EdgeStatuses, err)
			}
			step(t, w, s80Phase)
			shut := gap()
			set(false)
			reopened := shut
			for range s80Phase {
				step(t, w, 1)
				reopened = min(reopened, gap())
			}
			if slots, err := w.Slots(); err != nil || len(slots.EdgeStatuses) != 0 {
				t.Errorf("seed %d: Slots().EdgeStatuses after the reopening = %v (err %v), want none", seed, slots.EdgeStatuses, err)
			}
			changes := eventsAs[economysim.EdgeStatusChanged](rec.events)
			if len(changes) != 4 || !changes[0].Closed || !changes[1].Closed || changes[2].Closed || changes[3].Closed ||
				changes[0].Tick != s80Phase || changes[2].Tick != 2*s80Phase {
				t.Errorf("seed %d: EdgeStatusChanged = %+v, want two closures at tick %d then two openings at tick %d", seed, changes, s80Phase, 2*s80Phase)
			}
			t.Logf("seed %d: staple price gap of north and south: open %d, closed %d, lowest after the reopening %d (reported)", seed, open, shut, reopened)
			if shut < open || reopened > shut {
				t.Errorf("seed %d: gap open %d, closed %d, reopened %d, want it to open (closed >= open) and close (the lowest of the reopened phase <= closed)", seed, open, shut, reopened)
			}
		})
	}
}
