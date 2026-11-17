package production_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
)

// TestEffective_TechTermOrder verifies that the technology term is the second factor of the
// effective yield, site x shock x tech x trainee x abundance, each division by 10 000
// truncating before the next factor (spec-technology.md 5.1, FR-TEC-018), and that a term
// of 10 000 changes nothing, whatever its place.
//
// Arithmetic: 12 345 x 10 333 = 127 560 885, / 10 000 = 12 756; x 7 777 = 99 203 412,
// / 10 000 = 9 920. With the trainee term first: 12 345 x 7 777 = 96 007 065, / 10 000 =
// 9 600; x 10 333 = 99 196 800, / 10 000 = 9 919, one unit less.
//
// Requirement: FR-TEC-018, FR-PRO-022
//
// Guard: the order decides the last unit of a yield, so a reordering moves every golden
// that has a gain and a trainee together and no other test would say where.
func TestEffective_TechTermOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                  string
		site, shock, tech, trainee, abundance int64
		want                                  int64
	}{
		{"tech before trainee", 12_345, 10_000, 10_333, 7_777, 10_000, 9_920},
		{"tech before trainee and abundance", 12_345, 10_000, 10_333, 7_777, 9_013, 8_940},
		{"neutral tech", 12_345, 10_000, 10_000, 7_777, 10_000, 9_600},
		{"no site factor stays 0", 0, 10_000, 100_000, 10_000, 10_000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := production.Effective(tt.site, tt.shock, tt.tech, tt.trainee, tt.abundance)
			if !ok || got != tt.want {
				t.Errorf("Effective(%d, %d, %d, %d, %d) = %d, %t, want %d, true", tt.site, tt.shock, tt.tech, tt.trainee, tt.abundance, got, ok, tt.want)
			}
		})
	}
}
