package steady_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// TestActivity_Net verifies the net vector of one execution: the mean factor
// times each non-discard output, less the inputs, less the wear of each tool, as
// exact fractions in commodity order with zero entries dropped.
//
// Worked case: FactorSum 18 000 over Starts 2 is a mean factor of 9 000 basis
// points; an output of 12 gives 9 000 * 12 / 10 000 = 54/5; an input of 4 of the
// same commodity leaves 54/5 - 4 = 34/5; a tool with wear 3 and durability 2 000
// costs 3/2 000 per execution.
//
// Requirement: FR-VIA-001
func TestActivity_Net(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    steady.Activity
		want map[int]string
	}{
		{"output input tool and discard", steady.Activity{Duration: 1, Starts: 2, FactorSum: 18000,
			Inputs:  []steady.Item{{Commodity: 1, Qty: 4}},
			Outputs: []steady.Item{{Commodity: 1, Qty: 12}, {Commodity: 3, Qty: 100, Discard: true}},
			Tools:   []steady.Tool{{Commodity: 2, Wear: 3, Durability: 2000}}},
			map[int]string{1: "34/5", 2: "-3/2000"}},
		// 10^10 * 10^9 = 10^19 is above MaxInt64 (9.22 * 10^18); over 10 000 * 3 starts
		// it is 10^19 / 30 000 = 10^15 / 3.
		{"product beyond int64", steady.Activity{Duration: 1, Starts: 3, FactorSum: 10_000_000_000,
			Outputs: []steady.Item{{Commodity: 0, Qty: 1_000_000_000}}},
			map[int]string{0: "1000000000000000/3"}},
		// An output of 5 at factor 10 000 less an input of 5 is 0 and is dropped.
		{"zero entry dropped", steady.Activity{Duration: 1, Starts: 1, FactorSum: 10000,
			Inputs: []steady.Item{{Commodity: 0, Qty: 5}}, Outputs: []steady.Item{{Commodity: 0, Qty: 5}}},
			map[int]string{}},
		// Two outputs of one commodity add.
		{"outputs add", steady.Activity{Duration: 1, Starts: 1, FactorSum: 5000,
			Outputs: []steady.Item{{Commodity: 2, Qty: 4}, {Commodity: 2, Qty: 6}}},
			map[int]string{2: "5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.a.Net()
			if len(got) != len(tt.want) {
				t.Errorf("Net() has %d entries %v, want %d", len(got), got, len(tt.want))
			}
			last := -1
			for _, n := range got {
				if n.Commodity <= last {
					t.Errorf("Net() entry for commodity %d follows %d, want ascending order", n.Commodity, last)
				}
				last = n.Commodity
				if s := n.Val.RatString(); s != tt.want[n.Commodity] {
					t.Errorf("Net()[%d] = %s, want %s", n.Commodity, s, tt.want[n.Commodity])
				}
			}
		})
	}
}

// TestActivity_NetPanicsOnBrokenInvariant verifies that an activity with no
// start, or a tool with durability below 1, panics instead of dividing by zero.
//
// Guard: both would be a zero denominator in the exact arithmetic; the root never
// builds an activity with J = 0 (FR-VIA-002) or a tool with no durability.
func TestActivity_NetPanicsOnBrokenInvariant(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    steady.Activity
	}{
		{"no start", steady.Activity{Starts: 0}},
		{"no durability", steady.Activity{Starts: 1, Tools: []steady.Tool{{Commodity: 0, Wear: 1, Durability: 0}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if recover() == nil {
					t.Errorf("Net() on %+v did not panic", tt.a)
				}
			}()
			tt.a.Net()
		})
	}
}
