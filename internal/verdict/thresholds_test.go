package verdict_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// TestThresholds_OrderAndUnset verifies that the keys are in the order of STY
// 6.2 with the shared keys last, that a new list is all unset, that Set and Get
// agree, and that a value of 0 is a value.
//
// Requirement: FR-STY-070, FR-STY-060
func TestThresholds_OrderAndUnset(t *testing.T) {
	t.Parallel()
	names := verdict.ThresholdNames()
	if len(names) != 32 || names[0] != "r1.rise_bp" || !slices.Equal(names[29:], []string{"stale_max_bp", "active_min_bp", "min_windows"}) {
		t.Fatalf("ThresholdNames() = %v, want 32 keys from r1.rise_bp to the three shared keys", names)
	}
	th := verdict.NewThresholds()
	for i, k := range th {
		if k.Set || k.Name != names[i] {
			t.Errorf("NewThresholds()[%d] = %+v, want %q unset", i, k, names[i])
		}
	}
	if _, ok := th.Get("r1.rise_bp"); ok {
		t.Errorf("Get(r1.rise_bp) set on a new list")
	}
	if !th.Set("r1.rise_bp", 0) {
		t.Fatalf("Set(r1.rise_bp, 0) = false")
	}
	if v, ok := th.Get("r1.rise_bp"); !ok || v != 0 {
		t.Errorf("Get(r1.rise_bp) = %d, %v, want 0, true: a value of 0 is a value", v, ok)
	}
	if th.Set("no.such_key", 1) {
		t.Errorf("Set(no.such_key) = true, want false")
	}
	if _, ok := th.Get("no.such_key"); ok {
		t.Errorf("Get(no.such_key) = set, want not")
	}
}

// TestVerdict_Strings verifies the words of the classes and the one-line
// rendering of a verdict, all integers and words.
//
// Requirement: FR-STY-085
func TestVerdict_Strings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		v    verdict.Verdict
		want string
	}{
		{verdict.Verdict{Detector: verdict.R1, Class: verdict.None}, "R1 none"},
		{verdict.Verdict{Detector: verdict.R5a, Class: verdict.Detected, Onset: 120, Value: 80, Threshold: 60}, "R5a detected onset 120 value 80 threshold 60"},
		{verdict.Verdict{Detector: verdict.R9, Class: verdict.Unknown, Reason: verdict.ReasonWindows}, "R9 unknown (windows)"},
		{verdict.Verdict{Detector: verdict.R2, Class: verdict.Detected, Onset: -3, Value: -1, Threshold: 0}, "R2 detected onset -3 value -1 threshold 0"},
		{verdict.Verdict{Detector: verdict.R3, Class: verdict.Class(9)}, "R3 invalid"},
	} {
		if got := tc.v.String(); got != tc.want {
			t.Errorf("Verdict.String() = %q, want %q", got, tc.want)
		}
	}
	if got := verdict.Detectors(); len(got) != 10 || got[4] != verdict.R5a || got[5] != verdict.R5b {
		t.Errorf("Detectors() = %v, want ten detectors in the order of STY 6.2", got)
	}
}
