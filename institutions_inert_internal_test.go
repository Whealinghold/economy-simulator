package economysim

import (
	"slices"
	"testing"
)

// TestInstitutions_TrainRightNotNeutralAtZeroPlaces documents why overlay/institutions-inert.toml
// carries no train right (N-540, spec-institutions.md 14.3): a household that becomes a weaver
// between two rounds is not a member until the next round at P1.9a (FR-ORG-010, FR-ORG-013), so
// with a train right in force it is no trainer for that tick, although places 0 admit it at the
// next round; with no train right, or with the module off, it is a trainer at once.
//
// Arithmetic: orgTrainWorld(0, 0, right) has three weavers in home (agents 4, 5, 6) and places 0,
// no limit, so the first round admits all three. A fourth weaver is added after the step: its
// membership slot is empty, so mayTrain is false under the right and true under none; one more
// step runs the round, which admits it, and mayTrain is true.
//
// Requirement: FR-ORG-010, FR-ORG-013, FR-ORG-025
//
// Guard: if a household whose role changed in a tick counted as a member at once (the change that
// would make a train right neutral at places 0), this test is the one to change with N-540.
func TestInstitutions_TrainRightNotNeutralAtZeroPlaces(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		right     string
		wantFresh bool
	}{{"member", false}, {"", true}} {
		w, _, _ := orgTrainWorld(t, 0, 0, tc.right)
		id, err := w.AddAgent(NewAgent{Region: 0, Role: skWeaver})
		if err != nil {
			t.Fatalf("right %q: AddAgent() error = %v, want nil", tc.right, err)
		}
		a := &w.agents[int(id)]
		if got := w.org.mayTrain(a, 0); got != tc.wantFresh {
			t.Errorf("right %q: a new weaver may train before the round = %v, want %v", tc.right, got, tc.wantFresh)
		}
		if got := slices.Contains(trainerIDs(w), id); got != tc.wantFresh {
			t.Errorf("right %q: a new weaver is a candidate trainer before the round = %v, want %v", tc.right, got, tc.wantFresh)
		}
		step(t, w, 1)
		if got := w.org.mayTrain(&w.agents[int(id)], 0); !got {
			t.Errorf("right %q: the new weaver may train after the next round = false, want true (places 0 admit it)", tc.right)
		}
	}
}
