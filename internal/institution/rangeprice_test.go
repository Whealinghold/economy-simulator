package institution_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/institution"
)

// The bounds of the root (bounds.go), restated here because an internal package
// never imports the root: MaxPrice 10^9 and MaxBasisPoints 10^6.
const (
	maxPrice       = 1_000_000_000
	maxBasisPoints = 1_000_000
)

// TestRange_TruncatedBoundedChecked verifies the legal range of FR-ORG-028: the
// reference price times the level in basis points, truncated, at least 1 and at most
// MaxPrice, a level of 0 is none, and an input outside the domain is refused.
//
// Requirement: FR-ORG-028
func TestRange_TruncatedBoundedChecked(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		ref, capBP, floorBP int64
		wantCap, wantFloor  int64
		wantOK              bool
	}{
		// 1 000 * 12 000 / 10 000 = 1 200 (spec-institutions.md 13.2, example B).
		{"example B cap", 1000, 12000, 0, 1200, 0, true},
		// 999 * 5 000 / 10 000 = 499.5 truncated to 499.
		{"truncated toward zero", 999, 5000, 0, 499, 0, true},
		// 1 * 1 / 10 000 = 0 truncated, raised to 1.
		{"a level below one unit is one", 1, 1, 1, 1, 1, true},
		// 10^9 * 10^6 / 10^4 = 10^11, bounded to MaxPrice.
		{"bounded by MaxPrice", maxPrice, maxBasisPoints, maxBasisPoints, maxPrice, maxPrice, true},
		{"no level is none", 1000, 0, 0, 0, 0, true},
		{"a floor alone", 1000, 0, 8000, 0, 800, true},
		{"cap and floor", 1000, 12000, 8000, 1200, 800, true},
		{"reference below 1", 0, 1, 1, 0, 0, false},
		{"negative level", 1000, -1, 0, 0, 0, false},
		{"max price below 1", 1000, 1, 1, 0, 0, false},
		// math.MaxInt64 * 20 000 does not fit int64 before the division? it fits 128 bits
		// and the quotient 1.8e19 does not fit: refused, never wrapped.
		{"overflow of the quotient", math.MaxInt64, 20000, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mp := int64(maxPrice)
			if tt.name == "max price below 1" {
				mp = 0
			}
			c, f, ok := institution.Range(tt.ref, tt.capBP, tt.floorBP, mp)
			if ok != tt.wantOK || (ok && (c != tt.wantCap || f != tt.wantFloor)) {
				t.Errorf("Range(%d, %d, %d, %d) = %d, %d, %v, want %d, %d, %v",
					tt.ref, tt.capBP, tt.floorBP, mp, c, f, ok, tt.wantCap, tt.wantFloor, tt.wantOK)
			}
		})
	}
}

// TestRange_GridMonotoneAndInBounds verifies, over a grid of reference prices and
// levels up to the bounds of the root, that the range never fails, stays in
// [1, MaxPrice] when a level is set, and is monotone in the level: equal levels give
// equal prices and a floor at most the cap in basis points is at most it in minor
// units (the grid of spec-institutions.md 6.2.1; MaxPrice times MaxBasisPoints is
// 10^15 and fits int64).
//
// Requirement: FR-ORG-028
func TestRange_GridMonotoneAndInBounds(t *testing.T) {
	t.Parallel()
	refs := []int64{1, 2, 3, 7, 99, 1000, 12345, maxPrice - 1, maxPrice}
	levels := []int64{1, 2, 99, 5000, 9999, 10000, 10001, 20000, maxBasisPoints - 1, maxBasisPoints}
	for _, ref := range refs {
		for _, hi := range levels {
			for _, lo := range levels[:slices.Index(levels, hi)+1] {
				c, f, ok := institution.Range(ref, hi, lo, maxPrice)
				if !ok || !inBounds(c) || !inBounds(f) || f > c {
					t.Fatalf("Range(%d, %d, %d, MaxPrice) = %d, %d, %v, want ok, both in [1, MaxPrice] and floor <= cap", ref, hi, lo, c, f, ok)
				}
			}
		}
	}
}

// inBounds reports whether a price is in [1, maxPrice].
func inBounds(p int64) bool { return p >= 1 && p <= maxPrice }

// FuzzRange checks that Range never panics and that an accepted range stays inside
// [1, maxPrice] (or is 0, none) with no wrapped product.
func FuzzRange(f *testing.F) {
	f.Add(int64(1000), int64(12000), int64(8000), int64(maxPrice))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(1), int64(math.MaxInt64))
	f.Fuzz(func(t *testing.T, ref, capBP, floorBP, mp int64) {
		c, fl, ok := institution.Range(ref, capBP, floorBP, mp)
		if !ok {
			return
		}
		for _, p := range []int64{c, fl} {
			if p != 0 && (p < 1 || p > mp) {
				t.Errorf("Range(%d, %d, %d, %d) gave %d outside [1, %d]", ref, capBP, floorBP, mp, p, mp)
			}
		}
	})
}
