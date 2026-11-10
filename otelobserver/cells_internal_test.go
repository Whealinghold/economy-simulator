package otelobserver

import (
	"math"
	"testing"
)

// TestRatioBP_TruncatesAndDoesNotWrap verifies the basis-point ratio: n * 10 000
// / d truncated toward zero, computed in 128 bits so that a large n does not
// wrap, and 0 where the ratio is undefined.
//
// Requirement: FR-OBS-098
func TestRatioBP_TruncatesAndDoesNotWrap(t *testing.T) {
	tests := []struct {
		name string
		n, d int64
		want int64
	}{
		{"one third truncates", 1, 3, 3333},
		{"all is 10 000", 8, 8, 10000},
		{"zero numerator", 0, 5, 0},
		{"zero denominator", 5, 0, 0},
		{"negative numerator", -1, 5, 0},
		{"large numerator does not wrap (just under a half)", math.MaxInt64 / 2, math.MaxInt64, 4999},
		{"ratio beyond int64 saturates", math.MaxInt64, 1, math.MaxInt64},
		{"quotient beyond int64 saturates", 1_000_000_000_000_000_000, 1000, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ratioBP(tt.n, tt.d); got != tt.want {
				t.Errorf("ratioBP(%d, %d) = %d, want %d", tt.n, tt.d, got, tt.want)
			}
		})
	}
}

// TestIx_RejectsNegativeAndHuge verifies that an ID outside the dense range maps
// to -1, which no table holds, so the sample is dropped.
//
// Requirement: FR-OBS-052
func TestIx_RejectsNegativeAndHuge(t *testing.T) {
	if got := ix(int64(-3)); got != -1 {
		t.Errorf("ix(-3) = %d, want -1", got)
	}
	if got := ix(int64(1) << 40); got != -1 {
		t.Errorf("ix(1<<40) = %d, want -1", got)
	}
	if got := ix(int32(7)); got != 7 {
		t.Errorf("ix(7) = %d, want 7", got)
	}
}
