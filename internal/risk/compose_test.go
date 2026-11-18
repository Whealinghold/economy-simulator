package risk_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/risk"
)

// TestPhi_RefusesAQuotientThatLeavesInt64 verifies that Phi reports ok false,
// and not a wrapped fraction, when either of its two products leaves int64.
//
// Guard: the 128-bit product is checked only through the ok of MulDiv; a
// refactor that dropped the check would wrap and expose a store to a wrong
// share without any error.
//
// Requirement: FR-RSK-017, FR-RSK-040
func TestPhi_RefusesAQuotientThatLeavesInt64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                       string
		stored, sumCap, capK, load int64
	}{
		// stored * capK / sumCap = MaxInt64^2 / 1, far above int64.
		{"first quotient too large", math.MaxInt64, 1, math.MaxInt64, 1},
		// x = 10^15, then x * 10 000 / load = 10^19 / 1 > MaxInt64 (about 9.2 * 10^18).
		{"second quotient too large", 1_000_000_000_000_000, 1, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := risk.Phi(tt.stored, tt.sumCap, tt.capK, tt.load); ok || got != 0 {
				t.Errorf("Phi(%d, %d, %d, %d) = %d, %v, want 0, false", tt.stored, tt.sumCap, tt.capK, tt.load, got, ok)
			}
		})
	}
}

// TestExpected_EqualsCombineWhenEveryRowIsCertain verifies that the expected
// loss of a route whose rows all have probability 10 000 is the loss share the
// dispatch fold gives when every row strikes, for every pair and triple of
// severities on a grid.
//
// Two pieces compose here: Expected folds f = 10 000 - rhe(10 000 * s,
// 10 000) = 10 000 - s into K, which is Fold(kept, s). A drift between the two
// roundings would make the table the strategies read disagree with the loss a
// shipment takes in the certain case. Example: severities 2 500 and 4 000 give
// kept 10 000 -> 7 500 -> 4 500, loss 5 500.
//
// Guard: the expected-loss table (FR-RSK-024) and the dispatch fold
// (FR-RSK-009) are two implementations of one rounding rule; nothing else ties
// them.
//
// Requirement: FR-RSK-009, FR-RSK-024
func TestExpected_EqualsCombineWhenEveryRowIsCertain(t *testing.T) {
	t.Parallel()
	grid := []int64{0, 1, 999, 2_500, 4_000, 5_000, 5_001, 9_999, 10_000}
	for _, a := range grid {
		for _, b := range grid {
			for _, c := range grid {
				sev := []int64{a, b, c}
				rows := []risk.Edge{{P: 10_000, SeverityBP: a}, {P: 10_000, SeverityBP: b}, {P: 10_000, SeverityBP: c}}
				want, wok := risk.Combine(sev)
				got, ok := risk.Expected(rows)
				if !ok || !wok || got != want {
					t.Fatalf("Expected(certain rows %v) = %d, %v, want Combine = %d, %v", sev, got, ok, want, wok)
				}
			}
		}
	}
}

// TestExpected_NeverFallsWhenARowIsAdded verifies that the expected loss of a
// route does not fall when a row is appended, and stays 0 to 10 000, over a
// grid of probabilities and severities.
//
// Guard: a route with one more hazard cannot be safer; the merchant nets its
// revenue by this number, so a fall would make a riskier lane look better.
//
// Requirement: FR-RSK-024
func TestExpected_NeverFallsWhenARowIsAdded(t *testing.T) {
	t.Parallel()
	grid := []int64{0, 1, 500, 3_000, 10_000}
	var rows []risk.Edge
	prev := int64(0)
	for _, p := range grid {
		for _, s := range grid {
			rows = append(rows, risk.Edge{P: p, SeverityBP: s})
			got, ok := risk.Expected(rows)
			if !ok || got < prev || got > 10_000 {
				t.Fatalf("Expected(%d rows, last %+v) = %d, %v after %d, want from %d to 10000", len(rows), rows[len(rows)-1], got, ok, prev, prev)
			}
			prev = got
		}
	}
}
