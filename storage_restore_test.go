package economysim_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestRestoreWorld_StorageWorldIsRefusedNotPanicked verifies that RestoreWorld of a
// checkpoint of a world with the storage module on returns ErrCheckpointCorrupt, for a
// checkpoint taken before any step and one taken inside a job, and does not panic.
// The stores are not in the checkpoint until plan 3-sto Phase 8 (PR 5), so a restore
// would drop the stores of the role rows and change every room without a sign.
//
// Requirement: FR-STO-039, FR-CHK-050
//
// Fix: the restore of a storage world indexed the module's empty books in auditBooks
// and panicked with index out of range instead of returning an error.
func TestRestoreWorld_StorageWorldIsRefusedNotPanicked(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	for _, ticks := range []int{0, 100} {
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
		if err != nil {
			t.Fatalf("NewWorld(storage pack) error = %v, want nil", err)
		}
		for range ticks {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
		}
		ck := mustCheckpoint(t, w)
		var got *economysim.World
		var restoreErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RestoreWorld(storage world at tick %d) panicked: %v, want an error", ticks, r)
				}
			}()
			got, restoreErr = economysim.RestoreWorld(cfg, ck)
		}()
		if !errors.Is(restoreErr, economysim.ErrCheckpointCorrupt) || got != nil {
			t.Errorf("RestoreWorld(storage world at tick %d) = (%v, %v), want (nil, ErrCheckpointCorrupt)", ticks, got, restoreErr)
		}
	}
}
