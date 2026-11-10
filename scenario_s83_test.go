package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS83_ForcedMove is INT-S5 (spec-integration.md 16), the leg of the household
// on a journey: the body of TestRelocate_OnJourney run on the seeds of the scenarios.
// Exact: Relocate of a household on a journey drops the journey, counts its households
// in Ended and moves the household to the target of the call, with no MigrationStarted
// and no arrival; with a second obligation open the call is refused naming it and
// changes nothing; the audit holds. The leg of the open contract is
// TestRelocate_OpenContract.
//
// Requirement: FR-INT-031, FR-MIG-047
//
// Guard: a journey that survived the move would deliver the household a second time.
func TestScenarioS83_ForcedMove(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			economysim.RelocateOnJourney(t, seed)
		})
	}
}
