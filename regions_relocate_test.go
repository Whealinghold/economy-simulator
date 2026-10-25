package economysim_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestRelocate_RegionBound verifies, through the public query and the test seam
// of the relocation, that a registered obligation refuses with a
// *RegionBoundError that names it and matches ErrRegionBound, that skipping it
// by name or by every instance key lifts the refusal, that skipping one of two
// instances does not, and that no other refusal is skipped.
//
// Requirement: FR-REG-047, FR-REG-060
func TestRelocate_RegionBound(t *testing.T) {
	t.Parallel()
	c := withRegions(referenceConfig(), "south")
	c.Groups = append(c.Groups, economysim.Group{Region: "south", Role: "cook", Households: 10})
	w, err := economysim.NewWorld(c, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	// The agent holds two instances of the obligation, keys 3 and 4.
	holder := economysim.AgentID(len(c.Regions)) // the first agent of the first region
	economysim.RegisterBoundForTest(w, "open_contract", func(a economysim.AgentID, skip []string) bool {
		return a == holder && (!economysim.SkipHasForTest(skip, "open_contract", 3) || !economysim.SkipHasForTest(skip, "open_contract", 4))
	})
	to := economysim.RegionID(1)
	tests := []struct {
		name string
		skip []string
		want bool
	}{
		{"no skip", nil, true},
		{"skip by name", []string{"open_contract"}, false},
		{"skip by both keys", []string{"open_contract:3", "open_contract:4"}, false},
		{"skip by one key", []string{"open_contract:3"}, true},
		{"skip of another obligation", []string{"on_journey"}, true},
	}
	for _, tt := range tests {
		reason, err := w.RelocationRefusal(holder, to, tt.skip...)
		var rb *economysim.RegionBoundError
		got := errors.Is(err, economysim.ErrRegionBound) && errors.As(err, &rb) && rb.Obligation == "open_contract" && reason == string(economysim.ReasonRegionBound)
		if got != tt.want || (!tt.want && (reason != "" || err != nil)) {
			t.Errorf("%s: RelocationRefusal(%d, %d, %v) = %q, %v, want a region_bound refusal naming open_contract: %v", tt.name, holder, to, tt.skip, reason, err, tt.want)
		}
	}
	if err := economysim.RelocateForTest(w, holder, to, economysim.CauseConsumer); !errors.Is(err, economysim.ErrRegionBound) {
		t.Errorf("RelocateForTest = %v, want ErrRegionBound: the guarded path takes no skip", err)
	}
}

// TestRelocate_ThroughAPackRun verifies, on three with its trade overlay and over
// a range of seeds, the relocation as a consumer of the library meets it: the
// query allows the move, the account moves between two ticks, one AgentRelocated
// event reports it with the household weight, the snapshot moves exactly that
// weight from the old region to the new one, the same move is then refused as
// same_region, the run goes on, and a checkpoint taken just after the move
// restores to a world whose checkpoint after 20 more ticks equals the original's.
//
// Requirement: FR-REG-039, FR-REG-040, FR-REG-045, FR-REG-046, FR-REG-052, FR-REG-059
// Guard: the unit tests move an account in a mill; this is the only test that
// does it in a pack run with the real clearing, spawner and checkpoint.
func TestRelocate_ThroughAPackRun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, tradeWarm("three")...)
	for _, seed := range seedList(invariantSeeds(3)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			var moves []economysim.AgentRelocated
			obs := economysim.ObserverFunc(func(e economysim.Event) {
				if m, ok := e.(economysim.AgentRelocated); ok {
					moves = append(moves, m)
				}
			})
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(obs))
			if err != nil {
				t.Fatalf("seed %d: NewWorld() = %v, want nil", seed, err)
			}
			step := func(w *economysim.World, n int) {
				for range n {
					if err := w.Step(); err != nil {
						t.Fatalf("seed %d: Step() at tick %d = %v, want nil", seed, w.Tick(), err)
					}
				}
			}
			step(w, 2)
			// The first agent after the estates, one per region, belongs to region 0.
			holder, from, to := economysim.AgentID(len(cfg.Regions)), economysim.RegionID(0), economysim.RegionID(1)
			if reason, err := w.RelocationRefusal(holder, to); reason != "" || err != nil {
				t.Fatalf("seed %d: RelocationRefusal(%d, %d) = %q, %v, want no refusal", seed, holder, to, reason, err)
			}
			before, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() = %v, want nil", seed, err)
			}
			if err := economysim.RelocateForTest(w, holder, to, economysim.CauseConsumer); err != nil {
				t.Fatalf("seed %d: relocate(%d, %d) = %v, want nil", seed, holder, to, err)
			}
			if len(moves) != 1 || moves[0].Agent != holder || moves[0].From != from || moves[0].To != to {
				t.Fatalf("seed %d: AgentRelocated events = %+v, want one for agent %d from %d to %d", seed, moves, holder, from, to)
			}
			after, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() = %v, want nil", seed, err)
			}
			wt := moves[0].Weight
			if got, want := after.Regions[from].Households, before.Regions[from].Households-wt; got != want {
				t.Errorf("seed %d: households of region %d = %d after the move, want %d", seed, from, got, want)
			}
			if got, want := after.Regions[to].Households, before.Regions[to].Households+wt; got != want {
				t.Errorf("seed %d: households of region %d = %d after the move, want %d", seed, to, got, want)
			}
			if reason, _ := w.RelocationRefusal(holder, to); reason != string(economysim.ReasonSameRegion) {
				t.Errorf("seed %d: RelocationRefusal(%d, %d) after the move = %q, want %q", seed, holder, to, reason, economysim.ReasonSameRegion)
			}
			ck, err := w.Checkpoint()
			if err != nil {
				t.Fatalf("seed %d: Checkpoint() = %v, want nil", seed, err)
			}
			r, err := economysim.RestoreWorld(cfg, ck)
			if err != nil {
				t.Fatalf("seed %d: RestoreWorld() = %v, want nil", seed, err)
			}
			step(w, 20)
			step(r, 20)
			if !bytes.Equal(mustMarshal(t, mustCheckpoint(t, w)), mustMarshal(t, mustCheckpoint(t, r))) {
				t.Errorf("seed %d: the restored world's checkpoint differs from the original's after 20 ticks", seed)
			}
		})
	}
}
