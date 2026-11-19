package economysim

import (
	"math"
	"testing"
)

// Requirement: FR-CAP-012
func TestUpDiv_RoundsUpWithoutOverflow(t *testing.T) {
	cases := []struct{ a, b, want int64 }{
		{0, 5, 0}, {-3, 5, 0}, {7, 0, 0}, {7, -1, 0}, {10, 5, 2}, {11, 5, 3},
		{math.MaxInt64, 2, math.MaxInt64/2 + 1}, {math.MaxInt64, 1, math.MaxInt64},
	}
	for _, c := range cases {
		if got := upDiv(c.a, c.b); got != c.want {
			t.Errorf("upDiv(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// Requirement: FR-CAP-013
func TestRefusalClass_IgnoresWhatTheClausesCannotAnswer(t *testing.T) {
	n := &infrastructure{rec: []recipeClauses{{build: -1, repair: -1}}}
	for _, r := range []struct {
		recipe RecipeID
		reason Reason
	}{
		{0, ReasonNoFacility}, {5, ReasonNoFacility}, {-1, ReasonNoFacility},
		{0, ReasonNotUnlocked}, {0, ReasonNothingToRepair}, {5, ReasonFacilityCap}, {-1, ReasonNoTarget},
	} {
		if c, ok := n.refusalClass(0, 0, r.recipe, r.reason); ok {
			t.Errorf("refusalClass(recipe %d, %s) = %d, true; want false", r.recipe, r.reason, c)
		}
	}
}
