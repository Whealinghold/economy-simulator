package economysim

import "testing"

// Equivalent mutants of config.go that no test can kill (gomu, one-line why each):
//   - sortedBy: `len(s) == 0` -> `len(s) <= 0`: a length is never negative.
//
// The mutant `len(s) == 0` -> `len(s) < 0` is not equivalent: slices.Clone
// keeps a non-nil empty slice non-nil, so it is killed below.

// TestSortedBy_EmptyInputGivesNil pins that an empty, non-nil slice comes back
// as nil, so a canonical Config never carries an empty non-nil list that
// differs from the nil one.
//
// Requirement: FR-CFG-005
func TestSortedBy_EmptyInputGivesNil(t *testing.T) {
	t.Parallel()
	if got := sortedBy([]int{}, func(a, b int) int { return a - b }); got != nil {
		t.Errorf("sortedBy([]int{}) = %#v, want nil", got)
	}
	got := Config{Commodities: []Commodity{}}.canonical()
	if got.Commodities != nil {
		t.Errorf("canonical Commodities = %#v, want nil", got.Commodities)
	}
}
