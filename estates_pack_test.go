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

// TestPackFamine_LoadsAndHashes verifies that the pack famine of REF 4.4.5 loads
// in its cold layer set and with its warm-start document, validates, has the
// estates module on, the default relief fields and a positive starting wage, and
// that the warm start changes the configuration hash (it is a part of it).
//
// Requirement: FR-REF-001, FR-REF-005, FR-REF-006, FR-REF-013, FR-REF-039
func TestPackFamine_LoadsAndHashes(t *testing.T) {
	t.Parallel()
	cold := loadValidPack(t, packLayers["famine"]...)
	warm := loadValidPack(t, append(slices.Clone(packLayers["famine"]), "warm-famine.toml")...)
	if cold.Modules.Estates == nil || warm.Modules.Estates == nil {
		t.Errorf("Modules.Estates = %v (cold), %v (warm), want the module on in both", cold.Modules.Estates, warm.Modules.Estates)
	}
	if r := cold.Regions[0]; r.PoolRelease != 100 || r.ReliefBelow != 10_000 {
		t.Errorf("region %s PoolRelease %d and ReliefBelow %d, want 100 and 10 000", r.ID, r.PoolRelease, r.ReliefBelow)
	}
	if cold.WarmStart != nil || warm.WarmStart == nil || len(warm.WarmStart.Wages) == 0 || warm.WarmStart.Wages[0].Wage <= 0 {
		t.Errorf("warm start cold %v, warm %+v, want none cold and a positive starting wage warm", cold.WarmStart, warm.WarmStart)
	}
	if economysim.HashConfig(cold) == economysim.HashConfig(warm) {
		t.Errorf("HashConfig of the cold and the warm set are equal, want the warm start to be a part of the hash")
	}
}
