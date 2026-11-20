package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// orgModuleOutputs zeroes what ORG-S5 allows to differ in a snapshot: the names, the hash,
// the module list and the rows of the module.
func orgModuleOutputs(s economysim.Snapshot) economysim.Snapshot {
	s.ConfigHash, s.Names, s.Modules = [32]byte{}, economysim.Names{}, nil
	s.Bodies, s.LegalLevels, s.Suspended = nil, nil, nil
	return s
}

// withoutMembership drops the module's own events from a run's events: the aggregates of
// P7.2b (the slot events are told only at a call).
func withoutMembership(events []economysim.Event) []economysim.Event {
	return slices.DeleteFunc(slices.Clone(events), func(e economysim.Event) bool {
		switch e.(type) {
		case economysim.MembershipChanged, economysim.RightRefused, economysim.PriceControlled:
			return true
		}
		return false
	})
}

// TestScenarioORG5_InertOverlayIsNeutral verifies ORG-S5 (S141): the five with
// overlay/institutions-inert.toml (a body with places 0, a produce right, two
// controls with both levels 0) against the same five without the module (the cook's course kept), seeds 1 to
// scenarioSeedCount(), 150 ticks. Everything is identical but the module's own outputs: the
// names table, the hash, Snapshot.Modules, the construction summary, the MembershipChanged
// events and the Snapshot.Bodies rows (spec-institutions.md 14.3, N-399).
//
// Requirement: FR-ORG-001, FR-ORG-017, FR-ORG-053, NFR-ORG-003
//
// Guard: a declared module that is not inert, or that draws, fails here; FR-ORG-001's strict
// form is TestEmptyModule_EqualsOff.
func TestScenarioORG5_InertOverlayIsNeutral(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed scenario: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	inert := loadValidPack(t, append(slices.Clone(fivePolity), "overlay/institutions-inert.toml")...)
	free := loadValidPack(t, fivePolity...)
	// The overlay also gives the cook its course, which is core data and not the module's: the
	// world without the module carries it too, so the two differ by the module alone.
	for i := range free.Roles {
		if free.Roles[i].ID == "cook" {
			free.Roles[i].TrainingTicks = 120
		}
	}
	const ticks = 150
	for _, seed := range seedList(scenarioSeedCount()) {
		with := runPack(t, inert, runSpec{Seed: seed, Ticks: ticks, Drive: driveStep, Observe: true, Audit: true})
		without := runPack(t, free, runSpec{Seed: seed, Ticks: ticks, Drive: driveStep, Observe: true, Audit: true})
		if got, want := orgModuleOutputs(with.Final), orgModuleOutputs(without.Final); !reflect.DeepEqual(got, want) {
			t.Errorf("seed %d: final snapshot differs from the world without the module beyond its own outputs", seed)
		}
		if got, want := len(withoutMembership(with.Events)), len(without.Events); got != want {
			t.Errorf("seed %d: %d events beyond the module's, want %d", seed, got, want)
		}
	}
}
