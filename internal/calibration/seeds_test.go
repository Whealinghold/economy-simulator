package calibration

import (
	"errors"
	"math"
	"testing"
)

// TestSeeds_Blocks verifies that search block b is the seeds 100b+1 to 100b+100,
// disjoint from the stylised seeds 1 to 60, and that a block below 1 or one that
// leaves int64 is an error.
//
// Requirement: FR-CLB-020, FR-CLB-021
func TestSeeds_Blocks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		b           int
		first, last uint64
	}{
		{1, 101, 200}, // 100*1 + 1 to 100*1 + 100
		{2, 201, 300},
		{10, 1001, 1100},
	}
	for _, tt := range tests {
		first, last, err := Block(tt.b)
		if err != nil || first != tt.first || last != tt.last {
			t.Errorf("Block(%d) = %d, %d, %v, want %d, %d, nil", tt.b, first, last, err, tt.first, tt.last)
		}
		if InSTYRange(first) || InSTYRange(last) {
			t.Errorf("Block(%d) = %d..%d overlaps the stylised seeds 1 to 60", tt.b, first, last)
		}
	}
}

// TestSeeds_BlockErrors verifies that a block below 1, or one whose seeds leave
// int64, is an error, and that the last block whose end fits is not.
//
// Requirement: FR-CLB-020
func TestSeeds_BlockErrors(t *testing.T) {
	t.Parallel()
	for _, b := range []int{0, -1, math.MaxInt} {
		if _, _, err := Block(b); !errors.Is(err, ErrBlock) {
			t.Errorf("Block(%d) error = %v, want ErrBlock", b, err)
		}
	}
	// The last block whose end fits: 100b+100 <= MaxInt64.
	const lastFit = (math.MaxInt64 - 100) / 100
	if _, last, err := Block(lastFit); err != nil || last > math.MaxInt64 {
		t.Errorf("Block(%d) = %d, %v, want a block that fits", lastFit, last, err)
	}
	if _, _, err := Block(lastFit + 1); !errors.Is(err, ErrBlock) {
		t.Errorf("Block(%d) error = %v, want ErrBlock", lastFit+1, err)
	}
}

// TestSeeds_Confirm verifies the confirm seeds 1 to 50, and 1 to 10 at tier L, and
// the stylised range that no search uses.
//
// Requirement: FR-CLB-021
func TestSeeds_Confirm(t *testing.T) {
	t.Parallel()
	if f, l := ConfirmSeeds(false); f != 1 || l != 50 {
		t.Errorf("ConfirmSeeds(false) = %d, %d, want 1, 50", f, l)
	}
	if f, l := ConfirmSeeds(true); f != 1 || l != 10 {
		t.Errorf("ConfirmSeeds(true) = %d, %d, want 1, 10", f, l)
	}
	for seed, want := range map[uint64]bool{0: false, 1: true, 60: true, 61: false, 101: false} {
		if got := InSTYRange(seed); got != want {
			t.Errorf("InSTYRange(%d) = %v, want %v", seed, got, want)
		}
	}
}

// TestSeeds_Threshold verifies the pass threshold R - floor(R/10), exact integer
// arithmetic and not half to even: 25 seeds give 23 (25 - 2), where ceil(9*25/10)
// = ceil(22.5) = 23 and half to even would make 22.
//
// Requirement: FR-CLB-013, FR-CLB-106
func TestSeeds_Threshold(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ r, want int }{
		{25, 23}, {10, 9}, {50, 45}, {1, 1}, {9, 9}, {11, 10}, {100, 90}, {0, 0}, {-3, 0},
	} {
		if got := Threshold(tt.r); got != tt.want {
			t.Errorf("Threshold(%d) = %d, want %d", tt.r, got, tt.want)
		}
	}
}

// TestSeeds_ErrorValues verifies that a failed Block returns zero seeds, not a
// partial block.
//
// Requirement: FR-CLB-020
func TestSeeds_ErrorValues(t *testing.T) {
	t.Parallel()
	for _, b := range []int{0, math.MaxInt, (math.MaxInt64-100)/100 + 1} {
		if first, last, err := Block(b); err == nil || first != 0 || last != 0 {
			t.Errorf("Block(%d) = %d, %d, %v, want 0, 0 and an error", b, first, last, err)
		}
	}
}
