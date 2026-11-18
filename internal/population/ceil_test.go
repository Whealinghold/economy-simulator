package population_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/population"
)

// TestAllow_RoleHeadroom verifies that the role ceiling drops the births beyond
// the headroom after the household cap has acted, and that the two counts stay
// apart.
//
// Arithmetic: 6 births due, a region cap leaving 5 (Allow: allowed 5, dropped 1),
// then a headroom of 3 keeps 3 and drops 2: kept + ceiling dropped = 5 and the
// cap's 1 is not counted again. A room of 0 is a closed gate; a negative room is
// no ceiling.
//
// Requirement: FR-POP-041, FR-SET-023
func TestAllow_RoleHeadroom(t *testing.T) {
	t.Parallel()
	allowed, capped := population.Allow(6, 95, 100)
	if allowed != 5 || capped != 1 {
		t.Fatalf("Allow(6, 95, 100) = %d, %d, want 5, 1", allowed, capped)
	}
	tests := []struct {
		name                 string
		births, room         int64
		wantKept, wantCutOff int64
	}{
		{"room below the births", allowed, 3, 3, 2},
		{"room above the births", allowed, 9, 5, 0},
		{"room equal to the births", allowed, 5, 5, 0},
		{"room 0 closes the role", allowed, 0, 0, 5},
		{"negative room is no ceiling", allowed, -1, 5, 0},
		{"negative births count as 0", -4, 2, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			kept, dropped := population.Ceil(tt.births, tt.room)
			if kept != tt.wantKept || dropped != tt.wantCutOff {
				t.Errorf("Ceil(%d, %d) = %d, %d, want %d, %d", tt.births, tt.room, kept, dropped, tt.wantKept, tt.wantCutOff)
			}
		})
	}
}
