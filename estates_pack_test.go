package economysim_test

import (
	"os"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// TestEstates_PackOneAccountIDs verifies that in pack one (nine commodities,
// so B = 1 + 7 * 9 = 64 boundary accounts) the estate of the region is account
// 64 and the first household account would be 65.
//
// Requirement: FR-EST-004, FR-EST-005
func TestEstates_PackOneAccountIDs(t *testing.T) {
	t.Parallel()
	raw, err := config.LoadFS(os.DirFS("testdata/chains"), "base.toml", "one.toml", "warm-one.toml")
	if err != nil {
		t.Fatalf("LoadFS error = %v, want nil", err)
	}
	if err := raw.Validate(); err != nil {
		t.Fatalf("Validate error = %v, want nil", err)
	}
	cfg := economysim.Canonical(raw)
	accts, next, err := economysim.EstateAccountIDs(cfg)
	if err != nil {
		t.Fatalf("EstateAccountIDs error = %v, want nil", err)
	}
	if len(cfg.Regions) != 1 || !slices.Equal(accts, []int{64}) || next != 65 {
		t.Errorf("estate accounts %v, next %d over %d regions, want [64], 65 over 1", accts, next, len(cfg.Regions))
	}
}
