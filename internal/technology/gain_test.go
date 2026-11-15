package technology_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/technology"
)

// TestGain_CappedAndMonotone verifies G = min(CapBP, g(X) + A): the advance adds to
// the learned part, the cap cuts the sum, and G does not fall as X or A grow.
//
// Requirement: FR-TEC-005
func TestGain_CappedAndMonotone(t *testing.T) {
	t.Parallel()
	f := technology.Field{CapBP: 1000, Base: 100, DoublingBP: 300}
	tests := []struct {
		name string
		x, a int64
		want int64
	}{
		{name: "nothing", want: 0},
		{name: "advance alone", a: 400, want: 400},
		{name: "one doubling adds 300", x: 100, want: 300},
		{name: "learned plus advance", x: 100, a: 400, want: 700},
		{name: "cap cuts the sum", x: 100, a: 900, want: 1000},
	}
	for _, tt := range tests {
		got, ok := technology.Gain(f, tt.x, tt.a)
		if !ok || got != tt.want {
			t.Errorf("%s: Gain(x=%d, a=%d) = %d, %v, want %d", tt.name, tt.x, tt.a, got, ok, tt.want)
		}
	}
	prev := int64(0)
	for x := int64(0); x < 5000; x += 7 {
		g, _ := technology.Gain(f, x, 50)
		if g < prev {
			t.Fatalf("gain fell from %d to %d at x=%d", prev, g, x)
		}
		prev = g
	}
}
