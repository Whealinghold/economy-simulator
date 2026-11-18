package institution_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/institution"
)

// TestHeadroom_PlacesAndHouseholds verifies max(0, places - households), with no
// figure when places is 0 and nothing removed for being above the places.
//
// Requirement: FR-ORG-007
func TestHeadroom_PlacesAndHouseholds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		places, households int64
		want               int64
		wantNoLimit        bool
	}{
		{"room left", 12, 10, 2, false},
		{"full", 12, 12, 0, false},
		{"above the places has no room and nobody is removed", 12, 14, 0, false},
		{"no limit", 0, 14, 0, true},
		{"negative places is no limit", -1, 3, 0, true},
		{"negative households count as none", 5, -2, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, no := institution.Headroom(tt.places, tt.households)
			if got != tt.want || no != tt.wantNoLimit {
				t.Errorf("Headroom(%d, %d) = %d, %v, want %d, %v", tt.places, tt.households, got, no, tt.want, tt.wantNoLimit)
			}
		})
	}
}

// TestHeadroom_LeastOfTwo verifies that a role's headroom is the least of its class
// limit's and its body's, that a side with no limit does not bind, and that neither
// side limited gives no limit.
//
// Requirement: FR-ORG-007
func TestHeadroom_LeastOfTwo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		a           int64
		aNo         bool
		b           int64
		bNo         bool
		want        int64
		wantNoLimit bool
	}{
		// A class limit of 5 and a body's of 3 (plan 3-org Phase 9).
		{"both limited, the body's is less", 5, false, 3, false, 3, false},
		{"both limited, the class's is less", 2, false, 3, false, 2, false},
		{"zero room binds", 0, false, 3, false, 0, false},
		{"only the class limits", 4, false, 0, true, 4, false},
		{"only the body limits", 0, true, 4, false, 4, false},
		{"neither limits", 7, true, 9, true, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, no := institution.Least(tt.a, tt.aNo, tt.b, tt.bNo)
			if got != tt.want || no != tt.wantNoLimit {
				t.Errorf("Least(%d, %v, %d, %v) = %d, %v, want %d, %v", tt.a, tt.aNo, tt.b, tt.bNo, got, no, tt.want, tt.wantNoLimit)
			}
		})
	}
}
