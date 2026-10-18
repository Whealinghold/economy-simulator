package economysim

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestRefusalTally_EdgeCells verifies the first cell of the tally is flushed,
// that a pair one past the end of the table and one far beyond it both panic
// with the word invariant, and that an uncounted reason names itself in the
// message.
//
// Arithmetic: a (2 regions x 2 recipes) tally has 2*2*5 = 20 counters; region
// 2, recipe 0, column 0 is index (2*2+0)*5+0 = 20, which is len(counts).
//
// Requirement: FR-PRO-052
func TestRefusalTally_EdgeCells(t *testing.T) {
	t.Parallel()
	tl := newRefusalTally(2, 2)
	tl.add(0, 0, ReasonNotOnRole, 2)
	want := []RecipeRefused{{Tick: 4, Region: 0, Recipe: 0, NotOnRole: 2}}
	if got := flushed(tl, 4); !reflect.DeepEqual(got, want) {
		t.Errorf("rows for cell (0, 0) = %+v, want %+v", got, want)
	}
	tests := []struct {
		name   string
		region RegionID
		reason Reason
		want   string
	}{
		{"first index past the counters", 2, ReasonNotOnRole, "invariant"},
		{"far past the counters", 9, ReasonNotOnRole, "invariant"},
		{"uncounted reason", 0, Reason("no_room"), "no_room, which the first slice does not count"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := panicMessage(func() { newRefusalTally(2, 2).add(tt.region, 0, tt.reason, 1) })
			if !strings.Contains(msg, tt.want) {
				t.Errorf("add(%d, 0, %q, 1) panic = %q, want it to contain %q", tt.region, tt.reason, msg, tt.want)
			}
		})
	}
}

// TestAddPending_NegativeExecutions verifies a job with negative executions
// adds nothing, like a cleared one.
//
// Requirement: FR-PRO-042
func TestAddPending_NegativeExecutions(t *testing.T) {
	t.Parallel()
	r := newAnchorRig(flourConfig())
	pending, jobs := make([]int64, 4), make([]int64, 1)
	j := job{recipe: 0, executions: -1, pending: []int64{5, 6}}
	if !r.s.addPending(&j, pending, jobs) || !slices.Equal(pending, []int64{0, 0, 0, 0}) || jobs[0] != 0 {
		t.Errorf("addPending(executions -1) left pending %v and jobs %v, want all zero", pending, jobs)
	}
}

// TestAnchors_RegionsAndRunnableEdges verifies each region reads its own
// price and wage slices, that a runnable list shorter than the regions leaves
// the later regions unrunnable, and that recipe ids outside the table are
// skipped.
//
// Arithmetic: grain 400, wage 300, E = 10 000: cost = 3*400 + 10 + 300 =
// 1 510, anchor ceil(1510*10000/20000) = 755. Grain 200: 910, anchor 455.
//
// Requirement: FR-PRO-037, FR-PRO-049
func TestAnchors_RegionsAndRunnableEdges(t *testing.T) {
	t.Parallel()
	prices := slices.Concat(
		[]int64{0, 0, 200, 20_000}, []int64{0, 0, 200, 20_000},
		[]int64{0, 0, 200, 20_000}, []int64{0, 0, 400, 20_000})
	wages := []int64{300, 300, 300, 300}
	tests := []struct {
		name     string
		runnable [][]RecipeID
		want     []int64
	}{
		{"every region", [][]RecipeID{{0}, {0}, {0}, {0}},
			[]int64{0, 455, 0, 0, 0, 569, 0, 0, 0, 455, 0, 0, 0, 755, 0, 0}},
		{"none listed", nil, make([]int64, 16)},
		{"one listed of four", [][]RecipeID{{0}}, []int64{0, 455, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"ids outside the table", [][]RecipeID{{-1, 0, 1, 7}}, []int64{0, 455, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := newAnchorRig(flourConfig()).run(prices, wages, tt.runnable); !slices.Equal(got, tt.want) {
				t.Errorf("anchors(%v) = %v, want %v", tt.runnable, got, tt.want)
			}
		})
	}
}

// TestUnitCost_PastTheRow verifies a commodity at or beyond the row width is 0
// even where the cell it would alias holds a cost.
//
// Arithmetic: 4 commodities, so (region 0, commodity 5) aliases (region 1,
// commodity 1), the poor flour anchor 569; (region 3, commodity 4) is index 16,
// one past the 16 cells.
//
// Requirement: FR-PRO-049
func TestUnitCost_PastTheRow(t *testing.T) {
	t.Parallel()
	r := newAnchorRig(flourConfig())
	prices := repeat([]int64{0, 0, 200, 20_000}, 4)
	r.run(prices, []int64{300, 300, 300, 300}, [][]RecipeID{{0}, {0}, {0}, {0}})
	for _, c := range []struct {
		g RegionID
		c CommodityID
	}{{0, 5}, {3, 4}} {
		if got := r.s.unitCost(c.g, c.c); got != 0 {
			t.Errorf("unitCost(%d, %d) = %d, want 0", c.g, c.c, got)
		}
	}
}
