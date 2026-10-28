package economysim

import "testing"

// TestResolutionFloor_FollowsBeliefStep verifies that the resolution floor is
// ceil(10 000 / BeliefStep), computed from the configured value and not held as a
// constant: 50 at the default BeliefStep of 200, 100 at 100 and 1 000 at 10, a
// ceiling for a step that does not divide 10 000 (300 gives 34), and undefined
// outside the bound 1 to 2 500 of BeliefStep.
//
// Requirement: FR-CUR-006
func TestResolutionFloor_FollowsBeliefStep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		step   int64
		want   int64
		wantOK bool
	}{
		{200, 50, true},   // 10 000 / 200
		{100, 100, true},  // 10 000 / 100
		{10, 1_000, true}, // 10 000 / 10
		{300, 34, true},   // ceil(33.33)
		{1, 10_000, true}, // the lowest step
		{2_500, 4, true},  // the highest step
		{0, 0, false},     // below the bound
		{2_501, 0, false}, // above it
		{-200, 0, false},  // negative
	}
	for _, tt := range tests {
		w := DefaultWorldParams()
		w.BeliefStep = tt.step
		if got, ok := resolutionFloor(w); got != tt.want || ok != tt.wantOK {
			t.Errorf("resolutionFloor(BeliefStep %d) = %d, %t, want %d, %t", tt.step, got, ok, tt.want, tt.wantOK)
		}
	}
	if got, _ := resolutionFloor(DefaultWorldParams()); got != 50 {
		t.Errorf("resolutionFloor(default) = %d, want 50: the default BeliefStep is 200", got)
	}
}
