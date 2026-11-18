package main

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestRoleByID verifies that roleByID returns the index of the role with the
// given ID, and false for an ID the config does not have or an empty list.
//
// Guard: Borrower reports a pack without its player role by this lookup, so a
// miss must be false and not role 0.
func TestRoleByID(t *testing.T) {
	t.Parallel()
	cfg := economysim.Config{Roles: []economysim.Role{{ID: "a"}, {ID: "b"}}}
	tests := []struct {
		name   string
		cfg    economysim.Config
		id     string
		want   economysim.RoleID
		wantOK bool
	}{
		{"first role", cfg, "a", 0, true},
		{"second role", cfg, "b", 1, true},
		{"unknown role", cfg, "c", 0, false},
		{"no roles", economysim.Config{}, "a", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := roleByID(tt.cfg, tt.id)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("roleByID(%q) = %d, %v, want %d, %v", tt.id, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
