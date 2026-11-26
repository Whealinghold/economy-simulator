package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestInsurance_StoreOffersLeaveATrace verifies that with the store offers present the module
// is not neutral: policies form, the cover settles at P8.1b with CoverSettled rows of stage
// store (a store policy that ends with no loss is counted and has no CoverEnded, spec-insurance.md
// 8.5, 15.1.1) and the premiums are in N_t, so the mean ratio of a role differs from the run
// with the module off.
//
// Requirement: FR-INS-001, FR-INS-005
//
// Guard: the neutrality test above holds only because its pack is idle; if the store offers
// formed nothing here, that test would pass for nothing.
func TestInsurance_StoreOffersLeaveATrace(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	on := insuranceConfig()
	off := insuranceConfig()
	off.Modules.Insurance = nil
	for i := range off.Roles {
		off.Roles[i].Cover = nil
	}
	onEvents, onSnap := riskRun(t, on, 1, 25)
	_, offSnap := riskRun(t, off, 1, 25)
	formed, settled := countCoverEvents(onEvents)
	if formed == 0 || settled == 0 {
		t.Errorf("with the store offers: %d CoverFormed and %d CoverSettled in 25 ticks, want both above 0", formed, settled)
	}
	onSnap.Modules = slices.DeleteFunc(slices.Clone(onSnap.Modules), func(m string) bool { return m == "insurance" })
	if reflect.DeepEqual(onSnap.Roles, offSnap.Roles) {
		t.Error("the role rows are equal with the module on and off, want the premiums in the mean ratio")
	}
}

// countCoverEvents returns the number of CoverFormed and of CoverSettled events.
func countCoverEvents(events []economysim.Event) (formed, settled int) {
	for _, e := range events {
		switch e.(type) {
		case economysim.CoverFormed:
			formed++
		case economysim.CoverSettled:
			settled++
		}
	}
	return formed, settled
}
