package economysim_test

import (
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestWarnings_StoreLockedAtStart verifies store_locked_at_start (spec-storage.md 13.2):
// a role row of the storage module lists a store class whose Needs flag the class of a
// region of one of the role's groups lacks at construction. The reference chain's grower
// lists a bin and a shed; the bin needs the flag fair, which the region's class
// carries (a town of 600 households) and which it lacks when the classes are out of
// reach, and the warning counts the (region, class) pair once.
//
// Requirement: FR-SET-028, FR-STO-044
func TestWarnings_StoreLockedAtStart(t *testing.T) {
	t.Parallel()
	build := func(reach bool) economysim.Config {
		c := settlementsConfig()
		c.Modules.Storage = storageConfig().Modules.Storage
		for i := range c.Modules.Storage.Classes {
			if c.Modules.Storage.Classes[i].ID == "bin" {
				c.Modules.Storage.Classes[i].Needs = "fair"
			}
		}
		if !reach {
			c.Modules.Settlements.Classes[1].Enter, c.Modules.Settlements.Classes[2].Enter = 100_000, 200_000
		}
		return c
	}
	count := func(c economysim.Config) int {
		var n int
		for _, w := range c.Warnings() {
			if w.Rule == "store_locked_at_start" {
				n += int(w.Count)
				if p := strings.Join(w.Path, "."); p != "modules.storage.roles.grower.stores" {
					t.Errorf("store_locked_at_start path = %q, want modules.storage.roles.grower.stores", p)
				}
			}
		}
		return n
	}
	if err := build(true).Validate(); err != nil {
		t.Fatalf("Validate(reachable) = %v, want nil", err)
	}
	if got := count(build(true)); got != 0 {
		t.Errorf("store_locked_at_start with the flag carried = %d, want 0", got)
	}
	if err := build(false).Validate(); err != nil {
		t.Fatalf("Validate(out of reach) = %v, want nil", err)
	}
	if got := count(build(false)); got != 1 {
		t.Errorf("store_locked_at_start with the flag lacked = %d, want 1", got)
	}
}
