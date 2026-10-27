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

// TestPackFamine_LoadsAndHashes verifies that the pack famine of REF 4.4.5 loads in
// its cold layer set, validates, has the estates module on and the default relief
// fields, and that its warm-start document (layered last) is still the one the
// maker's job shares make invalid (they sum to 10 794 against 10 000): the
// rebalance is plan 8's (plan 2-7 O5), and this test fails when it lands so the
// warm layer is added to the pack's tests then.
//
// Requirement: FR-REF-001, FR-REF-005, FR-REF-006
func TestPackFamine_LoadsAndHashes(t *testing.T) {
	t.Parallel()
	cold := loadValidPack(t, packLayers["famine"]...)
	if cold.Modules.Estates == nil {
		t.Error("Modules.Estates = nil, want the module on")
	}
	if r := cold.Regions[0]; r.PoolRelease != 100 || r.ReliefBelow != 10_000 {
		t.Errorf("region %s PoolRelease %d and ReliefBelow %d, want 100 and 10 000", r.ID, r.PoolRelease, r.ReliefBelow)
	}
	if cold.WarmStart != nil {
		t.Errorf("cold WarmStart = %+v, want none", cold.WarmStart)
	}
	warm := loadPack(t, append(slices.Clone(packLayers["famine"]), "warm-famine.toml")...)
	if err := warm.Validate(); err == nil {
		t.Error("Validate(famine with warm-famine.toml) = nil, want the job-share error of plan 2-7 O5: plan 8 has rebalanced the pack, add the warm layer to the tests")
	}
}
