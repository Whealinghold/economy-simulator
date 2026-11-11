package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// TestDeterminism_Merchants verifies, for the merchants pack with the real merchant
// strategy and with the scripted merchant, that the final state does not depend on the way
// the run is driven or watched: the cases of detCases(true), seed 1, end in the snapshot
// and state hash of a bare Step run. The scripted variant is the only world that forms
// contracts (the cold pack's households die before a trip pays), so it also checks that
// a ContractFormed was seen.
//
// Requirement: FR-REF-025, FR-MER-057, FR-MER-061
//
// Guard: contracts, escrow and open shipments keep state that no other determinism test
// reaches; without the ContractFormed check the scripted variant would test a world with
// no contract and pass for the wrong reason.
func TestDeterminism_Merchants(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, variant := range []string{"merchant", "scripted"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			cfg, opts := loadValidPack(t, packMerchants("two")...), strategy.Options()
			if variant == "scripted" {
				cfg, opts = scriptedMerchants(t)
			}
			base := runPack(t, cfg, detStep(runSpec{Seed: 1, Ticks: detTicks, Extra: opts}))
			if variant == "scripted" {
				watched := runPack(t, cfg, detStep(runSpec{Seed: 1, Ticks: detTicks, Observe: true, Extra: opts}))
				if !slices.ContainsFunc(watched.Events, func(e economysim.Event) bool { _, ok := e.(economysim.ContractFormed); return ok }) {
					t.Fatal("scripted variant: no ContractFormed in 400 ticks, want at least one (the variant would test nothing)")
				}
			}
			for _, tc := range detCases(true) {
				spec := tc.spec()
				spec.Seed, spec.Ticks = 1, detTicks
				spec.Extra = append(slices.Clone(spec.Extra), opts...)
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("%s case %q: final snapshot differs from the bare run", variant, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("%s case %q: state hash differs from the bare run", variant, tc.name)
				}
			}
		})
	}
}
