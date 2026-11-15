package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOneStorage is pack one with the storage module alone on: overlay/storage.toml
// (plan 3-sto D14), the module-alone golden's configuration.
var packOneStorage = append(slices.Clone(packOne), "overlay/storage.toml")

// TestWarmStart_JobPledged verifies that a warm job in progress is pledged at
// construction: every limited household of the pack with a job in progress (the
// refiners and makers) holds a pledge equal to the load of its pending output, and the audit of the world passes.
//
// Requirement: FR-STO-045, FR-STO-020
func TestWarmStart_JobPledged(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, err := economysim.NewWorld(loadValidPack(t, packOneStorage...), economysim.WithSeed(1), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	ids, err := w.Agents(0)
	if err != nil {
		t.Fatalf("Agents() = %v", err)
	}
	pledged := 0
	for _, id := range ids {
		st, err := w.Agent(id)
		if err != nil || !st.HasJob {
			continue
		}
		var load int64
		for _, p := range st.Pending {
			if w.Config().Commodities[p.Commodity].ID != "waste" { // a discard output takes no room
				load += int64(p.Quantity) // every commodity of the pack has size (1, 1)
			}
		}
		if st.Room.Pledge.Mass != load || st.Room.Pledge.Bulk != load {
			t.Errorf("agent %d: pledge %+v, want the load %d of its pending output", id, st.Room.Pledge, load)
		}
		pledged++
	}
	if pledged == 0 {
		t.Errorf("no agent of the pack has a warm job, want the refiners' and makers' jobs in progress")
	}
}

// TestConstruction_RoomBelowWarmJob verifies that a room that holds the start stock but
// not the stock and the output of a warm job is ErrRoomShape at construction, and never
// the ledger's ErrNoRoom.
//
// Arithmetic: a maker holds 24 units (fuel 5, prepared 4, refined 5, staple 10, and the
// weight's share) and its job pledges 3 refined: 27 units, above a base room of 26 that
// holds the 24 of the stock.
//
// Requirement: FR-STO-045
func TestConstruction_RoomBelowWarmJob(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	cfg.Modules.Storage.Base = &economysim.Capacity{Mass: 26, Bulk: 26}
	if _, err := economysim.NewWorld(cfg, economysim.WithSeed(1)); !errors.Is(err, economysim.ErrRoomShape) || errors.Is(err, economysim.ErrNoRoom) {
		t.Errorf("NewWorld(room 26 under a warm job of 27) = %v, want ErrRoomShape and not ErrNoRoom", err)
	}
}

// TestInvariant_RoomNeverExceeded verifies FR-STO-005, 016 and 040 over seeds on the
// pack with the module on: the audit passes every tick (the ledger's identities of the
// room and the module's pledge and reserved room), and the figures of every role row
// show the load within the room.
//
// Requirement: FR-STO-005, FR-STO-016, FR-STO-040
func TestInvariant_RoomNeverExceeded(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	for seed := uint64(1); seed <= uint64(invariantSeeds(3)); seed++ {
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit())
		if err != nil {
			t.Fatalf("seed %d: NewWorld() = %v", seed, err)
		}
		for i := range 120 {
			if err := w.Step(); err != nil {
				t.Fatalf("seed %d tick %d: Step() = %v, want nil (WithAudit)", seed, i, err)
			}
			if i%10 != 0 {
				continue
			}
			snap, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() = %v", seed, err)
			}
			for _, r := range snap.Roles {
				if r.LoadMass > r.RoomMass || r.LoadBulk > r.RoomBulk {
					t.Errorf("seed %d tick %d: role row %+v has a load above its room", seed, i, r)
				}
			}
		}
	}
}
