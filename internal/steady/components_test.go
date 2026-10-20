package steady_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// act builds a one-start activity at mean factor 1 (FactorSum 10 000) from
// (commodity, quantity) pairs.
func act(in, out [][2]int64) steady.Activity {
	a := steady.Activity{Starts: 1, FactorSum: 10000, Duration: 1}
	for _, p := range in {
		a.Inputs = append(a.Inputs, steady.Item{Commodity: int(p[0]), Qty: p[1]})
	}
	for _, p := range out {
		a.Outputs = append(a.Outputs, steady.Item{Commodity: int(p[0]), Qty: p[1]})
	}
	return a
}

// TestComponents verifies the strongly connected components, their canonical
// order, and that a chain of 1 000 nodes and a ring of 1 000 do not overflow the
// stack.
//
// Requirement: FR-VIA-006
func TestComponents(t *testing.T) {
	t.Parallel()
	chain := make([][]int, 1000)
	ring := make([][]int, 1000)
	for i := range chain {
		if i < 999 {
			chain[i] = []int{i + 1}
		}
		ring[i] = []int{(i + 1) % 1000}
	}
	tests := []struct {
		name string
		adj  [][]int
		want int // number of components
		head []int
	}{
		{"empty", nil, 0, nil},
		// 0 -> 1 -> 0 is one component; 2 -> 0 is its own; 3 has a self loop.
		{"small", [][]int{{1}, {0}, {0}, {3}}, 3, []int{0, 1}},
		{"chain of 1000", chain, 1000, []int{0}},
		{"ring of 1000", ring, 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := steady.Components(tt.adj)
			if len(got) != tt.want {
				t.Fatalf("Components() made %d components, want %d", len(got), tt.want)
			}
			if tt.head != nil && !slices.Equal(got[0], tt.head) {
				t.Errorf("Components()[0] = %v, want %v", got[0], tt.head)
			}
			for i := 1; i < len(got); i++ {
				if got[i][0] < got[i-1][0] {
					t.Errorf("Components() not ordered by lowest member at %d: %v", i, got)
				}
			}
		})
	}
}

// TestMaterialGraph verifies the edges: inputs and tools to non-discard outputs,
// no edge to a discard output, successors ascending without repeats.
//
// Requirement: FR-VIA-006
func TestMaterialGraph(t *testing.T) {
	t.Parallel()
	a := steady.Activity{Starts: 1,
		Inputs:  []steady.Item{{Commodity: 2, Qty: 1}, {Commodity: 2, Qty: 1}},
		Tools:   []steady.Tool{{Commodity: 0, Wear: 1, Durability: 1}},
		Outputs: []steady.Item{{Commodity: 3, Qty: 1}, {Commodity: 1, Qty: 1}, {Commodity: 4, Qty: 1, Discard: true}}}
	got := steady.MaterialGraph(5, []steady.Activity{a})
	want := [][]int{{1, 3}, nil, {1, 3}, nil, nil}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Errorf("MaterialGraph()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestCycles verifies V2 on mini-chains, with the fraction of each s* derived in
// the comment.
//
// Requirement: FR-VIA-006
func TestCycles(t *testing.T) {
	t.Parallel()
	in := func(c, q int64) [][2]int64 { return [][2]int64{{c, q}} }
	// half is a recipe 1 in, 2 out at mean factor 0.5, so 2 * 5000 / 10000 = 1 out.
	half := act(in(0, 1), in(0, 2))
	half.FactorSum = 5000
	tests := []struct {
		name string
		n    int
		cs   []steady.Candidate
		want []steady.CycleComponent
	}{
		// 1 seed in, 1 out: net 0, so s <= 0 in the only row: s* = 0.
		{"seed in equals seed out", 1, []steady.Candidate{{Recipe: 0, Activity: act(in(0, 1), in(0, 1))}},
			[]steady.CycleComponent{{Commodities: []int{0}, Productive: false, SBP: 0}}},
		// 1 in, 2 out: net +1 per execution, sum x = 1 so s* = 1 = 10 000 bp.
		{"seed doubles", 1, []steady.Candidate{{Recipe: 0, Activity: act(in(0, 1), in(0, 2))}},
			[]steady.CycleComponent{{Commodities: []int{0}, Productive: true, SBP: 10000}}},
		// A: c0 -> c1, B: c1 -> 2 c0. With a = share of A: c0 row 2(1-a) - a >= s,
		// c1 row a - (1-a) >= s; equal at 2 - 3a = 2a - 1, a = 3/5, s* = 1/5 = 2 000 bp.
		{"two-commodity cycle", 2, []steady.Candidate{{Recipe: 0, Activity: act(in(0, 1), in(1, 1))}, {Recipe: 1, Activity: act(in(1, 1), [][2]int64{{0, 2}})}},
			[]steady.CycleComponent{{Commodities: []int{0, 1}, Productive: true, SBP: 2000}}},
		// The same recipe in two regions: at E = 0.5 net 0, at E = 1 net 1. The best
		// mean factor counts, whichever region is listed first.
		{"best yield across regions", 1, []steady.Candidate{{Recipe: 0, Activity: half}, {Recipe: 0, Activity: act(in(0, 1), in(0, 2))}},
			[]steady.CycleComponent{{Commodities: []int{0}, Productive: true, SBP: 10000}}},
		{"best yield listed second", 1, []steady.Candidate{{Recipe: 0, Activity: act(in(0, 1), in(0, 2))}, {Recipe: 0, Activity: half}},
			[]steady.CycleComponent{{Commodities: []int{0}, Productive: true, SBP: 10000}}},
		{"half yield alone", 1, []steady.Candidate{{Recipe: 0, Activity: half}},
			[]steady.CycleComponent{{Commodities: []int{0}, Productive: false, SBP: 0}}},
		// The wild source: the cycle alone has s* = 0; the no-input recipe that makes
		// seed from nothing belongs to the component and gives net +1.
		{"cycle with a wild source", 1, []steady.Candidate{{Recipe: 0, Activity: act(in(0, 1), in(0, 1))}, {Recipe: 1, Activity: act(nil, in(0, 1))}},
			[]steady.CycleComponent{{Commodities: []int{0}, Productive: true, SBP: 10000}}},
		// No cycle: nothing is checked.
		{"acyclic chain", 2, []steady.Candidate{{Recipe: 0, Activity: act(in(0, 1), in(1, 1))}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := steady.Cycles(context.Background(), tt.n, tt.cs, 1<<20, 1000)
			if err != nil {
				t.Fatalf("Cycles(%s) error = %v", tt.name, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Cycles(%s) = %+v, want %+v", tt.name, got, tt.want)
			}
			for i, w := range tt.want {
				g := got[i]
				if !slices.Equal(g.Commodities, w.Commodities) || g.Productive != w.Productive || g.SBP != w.SBP || g.Skipped {
					t.Errorf("Cycles(%s)[%d] = %+v, want %+v", tt.name, i, g, w)
				}
			}
		})
	}
}

// TestCycles_SkippedAndCancelled verifies the size gate and the pivot cap report
// a component as skipped (D126), and that a cancelled context is an error.
//
// Requirement: FR-VIA-006
func TestCycles_SkippedAndCancelled(t *testing.T) {
	t.Parallel()
	cs := []steady.Candidate{{Recipe: 0, Activity: act([][2]int64{{0, 1}}, [][2]int64{{0, 2}})}}
	// 1 commodity: rows 2, columns 1 activity + s + 2 slacks = 4, cells 8.
	tests := []struct {
		name          string
		cells, pivots int64
	}{
		{"cells below 8", 7, 100},
		{"no pivots allowed", 8, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := steady.Cycles(context.Background(), 1, cs, tt.cells, tt.pivots)
			if err != nil || len(got) != 1 || !got[0].Skipped {
				t.Errorf("Cycles(cells %d, pivots %d) = %+v, %v, want one skipped component", tt.cells, tt.pivots, got, err)
			}
		})
	}
	t.Run("cells at the gate pass", func(t *testing.T) {
		t.Parallel()
		got, err := steady.Cycles(context.Background(), 1, cs, 8, 100)
		if err != nil || len(got) != 1 || got[0].Skipped || !got[0].Productive {
			t.Errorf("Cycles(cells 8) = %+v, %v, want one productive component", got, err)
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := steady.Cycles(ctx, 1, cs, 8, 100); !errors.Is(err, context.Canceled) {
			t.Errorf("Cycles(cancelled) error = %v, want context.Canceled", err)
		}
	})
}
