package calibration

import (
	"errors"
	"slices"
	"testing"
)

// TestAlias_Cases verifies the alias test on the worked cases of 19.7 and the
// four named in 11.3: collinear, independent, one measure and a zero vector.
//
// Requirement: FR-CLB-061
func TestAlias_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ei, ej []int64
		tol    int64
		want   bool
	}{
		// (300,-450,0).(600,-900,0) = 585 000, so dot^2 = 342 225 000 000; the norms are
		// 292 500 and 1 170 000, whose product is also 342 225 000 000: cos^2 is exactly
		// 10 000 bp, above the 9 500 of the test (19.7).
		{"collinear", []int64{300, -450, 0}, []int64{600, -900, 0}, 500, true},
		// (300,-450,0).(300,100,-200) = 45 000; cos^2 = 495 bp, far below 9 500.
		{"independent", []int64{300, -450, 0}, []int64{300, 100, -200}, 500, false},
		// One measure: (120) and (-35): dot -4 200, cos^2 = 10 000.
		{"one measure", []int64{120}, []int64{-35}, 500, true},
		{"zero vector i", []int64{0, 0}, []int64{5, 7}, 500, false},
		{"zero vector j", []int64{5, 7}, []int64{0, 0}, 500, false},
		{"both zero", nil, nil, 500, false},
		{"opposite signs are collinear", []int64{3, 4}, []int64{-6, -8}, 500, true},
		// (1,0).(1,1) = 1; 10 000 * 1 = 10 000 against 9 500 * 1 * 2 = 19 000.
		{"cos2 5000 under 9500", []int64{1, 0}, []int64{1, 1}, 500, false},
		// Same pair at tolerance 5 000: 10 000 against 5 000 * 2 = 10 000, equal passes.
		{"equality passes", []int64{1, 0}, []int64{1, 1}, 5000, true},
		{"tolerance 10000 aliases any non-zero pair", []int64{1, 0}, []int64{0, 1}, 10000, true},
		{"tolerance 0 needs exact collinearity", []int64{2, 3}, []int64{2, 3}, 0, true},
		{"tolerance 0 rejects near collinear", []int64{2, 3}, []int64{2, 4}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Aliased(tt.ei, tt.ej, tt.tol)
			if err != nil || got != tt.want {
				t.Errorf("Aliased(%v, %v, %d) = %v, %v, want %v, nil", tt.ei, tt.ej, tt.tol, got, err, tt.want)
			}
		})
	}
}

// TestAlias_LargestInputs verifies the 128-bit comparison at its bounds: sixteen
// readings of 2^26 compare without overflow, and a pair one part in 2^56 away
// from collinear is told apart, which a rounded comparison could not.
//
// Requirement: FR-CLB-061
func TestAlias_LargestInputs(t *testing.T) {
	t.Parallel()
	big := slices.Repeat([]int64{maxReading}, maxMeasures)
	if got, err := Aliased(big, big, AliasTolBP); err != nil || !got {
		t.Errorf("Aliased(16 x 2^26, same) = %v, %v, want true, nil", got, err)
	}
	neg := slices.Repeat([]int64{-maxReading}, maxMeasures)
	if got, err := Aliased(big, neg, AliasTolBP); err != nil || !got {
		t.Errorf("Aliased(16 x 2^26, 16 x -2^26) = %v, %v, want true, nil", got, err)
	}
	// Tolerance 0 at the bound: identical vectors are still aliased, and one reading
	// moved by 1 is not (cos^2 is a hair under 10 000 and the test is exact).
	near := slices.Clone(big)
	near[0]--
	if got, err := Aliased(big, big, 0); err != nil || !got {
		t.Errorf("Aliased(big, big, 0) = %v, %v, want true, nil", got, err)
	}
	if got, err := Aliased(big, near, 0); err != nil || got {
		t.Errorf("Aliased(big, big with one reading 1 lower, 0) = %v, %v, want false, nil", got, err)
	}
}

// TestAlias_Errors verifies that input outside the bounds is an error, named by
// index for a reading, and never a wrapped comparison.
//
// Requirement: FR-CLB-061
func TestAlias_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ei, ej []int64
		tol    int64
	}{
		{"reading above 2^26 in i", []int64{1, maxReading + 1}, []int64{1, 1}, 500},
		{"reading below -2^26 in j", []int64{1, 1}, []int64{1, -maxReading - 1}, 500},
		{"clamped reading", []int64{1<<31 - 1}, []int64{1}, 500},
		{"seventeen measures", slices.Repeat([]int64{1}, 17), slices.Repeat([]int64{1}, 17), 500},
		{"lengths differ", []int64{1, 2}, []int64{1}, 500},
		{"negative tolerance", []int64{1}, []int64{1}, -1},
		{"tolerance above 10000", []int64{1}, []int64{1}, 10001},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := Aliased(tt.ei, tt.ej, tt.tol); !errors.Is(err, ErrAlias) || got {
				t.Errorf("Aliased error = %v (result %v), want ErrAlias and false", err, got)
			}
		})
	}
	// The bound itself is legal.
	if _, err := Aliased([]int64{maxReading, -maxReading}, []int64{1, 1}, 500); err != nil {
		t.Errorf("Aliased at exactly 2^26 error = %v, want nil", err)
	}
}
