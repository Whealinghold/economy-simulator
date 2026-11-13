package economysim_test

import (
	"reflect"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// warmFive is the warm five, the pack of the proof, and fivePolity the same with the
// ungoverned polity overlay layered last.
var (
	warmFive   = append(slices.Clone(packLayers["five"]), "warm-five.toml")
	fivePolity = append(slices.Clone(warmFive), "overlay/polity.toml")
)

// TestScenarioS112_Neutral verifies the neutrality proof of spec-polity.md 13.3
// (POL-S1, FR-POL-019): the warm five with overlay/polity.toml, which has no seat
// and so no treasury, against the warm five alone, over seeds 1 to 10 and 200
// ticks with the audit on. Every event and the final snapshot are identical except
// the configuration hash and the names table of the world-created event, and the
// module list and names table of the snapshot. It also covers FR-POL-020 and 021:
// had the module taken a draw, created an account or shifted an ID, the results
// would differ.
//
// Requirement: FR-POL-001, FR-POL-019, FR-POL-020, FR-POL-021
func TestScenarioS112_Neutral(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	base := loadValidPack(t, warmFive...)
	with := loadValidPack(t, fivePolity...)
	if with.Modules.Polity == nil || base.Modules.Polity != nil {
		t.Fatal("the polity overlay must turn the module on and the pack must have it off")
	}
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			spec := runSpec{Seed: seed, Ticks: 200, Observe: true, Audit: true}
			a, b := runPack(t, base, spec), runPack(t, with, spec)
			if len(a.Events) != len(b.Events) {
				t.Fatalf("seed %d: %d events with the module, %d without, want equal", seed, len(b.Events), len(a.Events))
			}
			for i := range a.Events {
				x, y := neutralEvent(a.Events[i]), neutralEvent(b.Events[i])
				if !reflect.DeepEqual(x, y) {
					t.Fatalf("seed %d: event %d differs: %#v with the module, %#v without", seed, i, y, x)
				}
			}
			x, y := neutralSnapshot(a.Final), neutralSnapshot(b.Final)
			if !reflect.DeepEqual(x, y) {
				t.Fatalf("seed %d: the final snapshots differ beyond the hash, the names and the module list", seed)
			}
			if got := b.Final.Names.Jurisdictions; len(got) != 3 {
				t.Errorf("seed %d: Names.Jurisdictions = %v, want the 3 jurisdictions", seed, got)
			}
		})
	}
}

// neutralEvent returns e without the fields that the module may change by being on
// (spec-polity.md FR-POL-019): the configuration hash and the names table of the
// world-created event.
func neutralEvent(e economysim.Event) economysim.Event {
	if wc, ok := e.(economysim.WorldCreated); ok {
		wc.ConfigHash = [32]byte{}
		wc.Names.Jurisdictions = nil
		return wc
	}
	return e
}

// neutralSnapshot returns s without the configuration hash, the names of the
// jurisdictions and the module list.
func neutralSnapshot(s economysim.Snapshot) economysim.Snapshot {
	s.ConfigHash = [32]byte{}
	s.Names.Jurisdictions = nil
	s.Modules = nil
	return s
}
