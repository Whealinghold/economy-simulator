package settlement_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/settlement"
)

// TestThresholds_CeilAndFloor verifies the effective thresholds of
// spec-settlements.md 4.2.1: the enter threshold is the ceiling of Enter times the
// scale and the leave threshold the floor of the enter threshold times the
// remaining band.
//
// Requirement: FR-SET-008
func TestThresholds_CeilAndFloor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		enter   []int64
		scale   int64
		band    int64
		wantE   []int64
		wantL   []int64
		wantOK  bool
		comment string
	}{
		// 300 * 10 000 / 10 000 = 300, L = 300 * 9 000 / 10 000 = 270; 800 gives 720; 2 000 gives 1 800.
		{"unit scale", []int64{0, 300, 800, 2000}, 10_000, 1_000, []int64{0, 300, 800, 2000}, []int64{0, 270, 720, 1800}, true, ""},
		// west: 300 * 6 000 / 10 000 = 180, L = 180 * 0.9 = 162; 800 gives 480 and 432; 2 000 gives 1 200 and 1 080.
		{"scale 6000", []int64{0, 300, 800, 2000}, 6_000, 1_000, []int64{0, 180, 480, 1200}, []int64{0, 162, 432, 1080}, true, ""},
		// 301 * 6 000 / 10 000 = 180.6 rounds up to 181; L = 181 * 0.9 = 162.9 rounds down to 162.
		{"ceil then floor", []int64{0, 301}, 6_000, 1_000, []int64{0, 181}, []int64{0, 162}, true, ""},
		// Enter 5 at 1 000 bp: L = floor(4.5) = 4, a band of one household; Enter 10 gives 9.
		{"band of one household", []int64{0, 5, 10}, 10_000, 1_000, []int64{0, 5, 10}, []int64{0, 4, 9}, true, ""},
		// Enter 3 at 5 000 bp: L = floor(1.5) = 1.
		{"odd threshold half band", []int64{0, 3}, 10_000, 5_000, []int64{0, 3}, []int64{0, 1}, true, ""},
		// No band: the leave threshold is the enter threshold.
		{"no band", []int64{0, 300}, 10_000, 0, []int64{0, 300}, []int64{0, 300}, true, ""},
		// 5 * 5 000 / 10 000 = 2.5 up to 3 and 6 * 5 000 / 10 000 = 3: the two thresholds collapse (4.2.2).
		{"equal thresholds collapse", []int64{0, 5, 6}, 5_000, 0, []int64{0, 3, 3}, []int64{0, 3, 3}, true, ""},
		{"one class", []int64{0}, 10_000, 1_000, []int64{0}, []int64{0}, true, ""},
		{"no class", nil, 10_000, 1_000, nil, nil, false, ""},
		{"zero scale", []int64{0, 5}, 0, 1_000, nil, nil, false, ""},
		{"negative band", []int64{0, 5}, 10_000, -1, nil, nil, false, ""},
		{"band above denominator", []int64{0, 5}, 10_000, 10_001, nil, nil, false, ""},
		{"negative enter", []int64{0, -5}, 10_000, 1_000, nil, nil, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e, l, ok := settlement.Thresholds(tt.enter, tt.scale, tt.band)
			if ok != tt.wantOK {
				t.Fatalf("Thresholds(%v, %d, %d) ok = %v, want %v", tt.enter, tt.scale, tt.band, ok, tt.wantOK)
			}
			if !slices.Equal(e, tt.wantE) || !slices.Equal(l, tt.wantL) {
				t.Errorf("Thresholds(%v, %d, %d) = %v, %v, want %v, %v", tt.enter, tt.scale, tt.band, e, l, tt.wantE, tt.wantL)
			}
		})
	}
}

// TestThresholds_BandNeverEmpty verifies that with a band of at least 1 basis
// point and an enter threshold of at least 1, the leave threshold is below the
// enter threshold, and that it is at least floor(e/2) for a band up to 5 000.
//
// Requirement: FR-SET-008
func TestThresholds_BandNeverEmpty(t *testing.T) {
	t.Parallel()
	for enter := int64(1); enter <= 2_000; enter++ {
		for _, scale := range []int64{1, 5_000, 10_000, 100_000} {
			for _, band := range []int64{1, 999, 1_000, 5_000} {
				e, l, ok := settlement.Thresholds([]int64{0, enter}, scale, band)
				if !ok {
					t.Fatalf("Thresholds([0 %d], %d, %d) not ok", enter, scale, band)
				}
				if l[1] > e[1]-1 || l[1] < e[1]/2-1 {
					t.Fatalf("Thresholds([0 %d], %d, %d) = e %d, l %d, want l <= e-1 and l >= e/2-1", enter, scale, band, e[1], l[1])
				}
			}
		}
	}
}

// TestThresholds_LargestCaseNoOverflow verifies that the largest validated input,
// MaxHouseholds (10^9) at MaxThresholdBP (10^5), is computed without overflow,
// and that an input whose product exceeds int64 reports it instead of wrapping.
//
// Requirement: FR-SET-032
func TestThresholds_LargestCaseNoOverflow(t *testing.T) {
	t.Parallel()
	// 10^9 * 10^5 / 10^4 = 10^10; the leave threshold with a band of 5 000 is 5 * 10^9.
	e, l, ok := settlement.Thresholds([]int64{0, 1_000_000_000}, 100_000, 5_000)
	if !ok || e[1] != 10_000_000_000 || l[1] != 5_000_000_000 {
		t.Errorf("Thresholds(10^9, 10^5, 5000) = %v, %v, %v, want [0 1e10], [0 5e9], true", e, l, ok)
	}
	// math.MaxInt64 * 10^5 / 10^4 is 10 times int64: the quotient does not fit.
	if _, _, ok := settlement.Thresholds([]int64{0, math.MaxInt64}, 100_000, 0); ok {
		t.Error("Thresholds(MaxInt64, 10^5, 0) ok = true, want false")
	}
}

// FuzzThresholds verifies that Thresholds never panics and, when it succeeds,
// returns slices of the input length with the leave threshold at most the enter
// threshold, both non-negative.
//
// Requirement: NFR-SET-004
func FuzzThresholds(f *testing.F) {
	f.Add(int64(300), int64(10_000), int64(1_000))
	f.Add(int64(5), int64(5_000), int64(0))
	f.Add(int64(math.MaxInt64), int64(100_000), int64(5_000))
	f.Fuzz(func(t *testing.T, enter, scale, band int64) {
		e, l, ok := settlement.Thresholds([]int64{0, enter}, scale, band)
		if !ok {
			return
		}
		if len(e) != 2 || len(l) != 2 || e[1] < 0 || l[1] < 0 || l[1] > e[1] {
			t.Errorf("Thresholds([0 %d], %d, %d) = %v, %v, want 2 entries with 0 <= l <= e", enter, scale, band, e, l)
		}
	})
}
