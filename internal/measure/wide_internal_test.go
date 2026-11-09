package measure

import (
	"math"
	"testing"
)

// TestSticky_RemembersTheFirstOverflow verifies that each chained operation sets
// the overflow flag when it overflows and not otherwise, and that the flag stays
// set through later operations that succeed.
//
// Requirement: FR-IND-009
func TestSticky_RemembersTheFirstOverflow(t *testing.T) {
	t.Parallel()
	max := Uint128{Hi: math.MaxUint64, Lo: math.MaxUint64}
	tests := []struct {
		name string
		do   func(*sticky)
		want bool
	}{
		{"mul fits", func(s *sticky) { s.mul(Uint128{Lo: 1 << 40}, 1<<40) }, false},
		{"mul overflows", func(s *sticky) { s.mul(max, 2) }, true},
		{"add fits", func(s *sticky) { s.add(Uint128{Lo: 1}, Uint128{Lo: 2}) }, false},
		{"add overflows", func(s *sticky) { s.add(max, Uint128{Lo: 1}) }, true},
		{"mulBP fits", func(s *sticky) { s.mulBP(Uint128{Lo: 12345}) }, false},
		{"mulBP past the guard", func(s *sticky) { s.mulBP(maxTimesBP.addOne()) }, true},
		{"an overflow stays after a success", func(s *sticky) {
			s.add(max, Uint128{Lo: 1})
			s.mul(Uint128{Lo: 1}, 1)
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var s sticky
			tt.do(&s)
			if s.over != tt.want {
				t.Errorf("%s: over = %v, want %v", tt.name, s.over, tt.want)
			}
		})
	}
}

// addOne returns u + 1 for a sum that fits.
func (u Uint128) addOne() Uint128 {
	s, _ := u.Add(Uint128{Lo: 1})
	return s
}
