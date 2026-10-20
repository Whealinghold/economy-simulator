package steady

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
	"testing"
)

// full is an activity with unit mean factor (FactorSum = 10 000 * Starts).
func full(in, out [][2]int64) Activity {
	a := Activity{Starts: 1, FactorSum: 10000}
	for _, p := range in {
		a.Inputs = append(a.Inputs, Item{Commodity: int(p[0]), Qty: p[1]})
	}
	for _, p := range out {
		a.Outputs = append(a.Outputs, Item{Commodity: int(p[0]), Qty: p[1]})
	}
	return a
}

// TestBestPerRecipeMutation pins the pick of the best candidate per recipe, the
// first on a tie, and the recipe order of the result.
//
// Guard: gomu survivors components.go:28, 34, 38 (index bookkeeping, tie
// strictness, ordering by recipe).
func TestBestPerRecipeMutation(t *testing.T) {
	t.Parallel()
	mk := func(recipe, region int, starts, sum int64) Candidate {
		return Candidate{Recipe: recipe, Activity: Activity{Region: region, Starts: starts, FactorSum: sum}}
	}
	got := bestPerRecipe([]Candidate{
		mk(5, 0, 1, 10000), // recipe 5 first, so recipe 2 is not at index 0
		mk(2, 1, 1, 5000),
		mk(2, 2, 2, 18000), // mean 9 000: beats 5 000 only by cross-multiplication
		mk(2, 3, 1, 9000),  // mean 9 000: ties with region 2, the first stays
		mk(1, 4, 1, 100),
	})
	var regions, recipes []int
	for _, c := range got {
		regions = append(regions, c.Region)
		recipes = append(recipes, c.Recipe)
	}
	if want := []int{1, 2, 5}; !slices.Equal(recipes, want) {
		t.Errorf("recipes = %v, want %v", recipes, want)
	}
	if want := []int{4, 2, 0}; !slices.Equal(regions, want) {
		t.Errorf("regions = %v, want %v", regions, want)
	}
}

// TestMaterialGraphMutation pins that a discarded output does not stop the scan
// of the outputs after it.
//
// Guard: gomu survivor components.go:67 (continue -> break).
func TestMaterialGraphMutation(t *testing.T) {
	t.Parallel()
	a := full([][2]int64{{0, 1}}, nil)
	a.Outputs = []Item{{Commodity: 1, Qty: 1, Discard: true}, {Commodity: 2, Qty: 1}}
	got := MaterialGraph(3, []Activity{a})
	if want := [][]int{{2}, nil, nil}; !slices.EqualFunc(got, want, slices.Equal[[]int]) {
		t.Errorf("MaterialGraph = %v, want %v", got, want)
	}
}

// TestComponentsMutation compares Components with a reachability reference on
// seeded random graphs, so every branch of the iterative search is exercised.
//
// Guard: gomu survivors components.go:100-168 (low-link propagation, component
// pop, root bookkeeping).
func TestComponentsMutation(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(7, 11))
	for iter := 0; iter < 400; iter++ {
		n := 1 + rng.IntN(9)
		adj := make([][]int, n)
		reach := make([][]bool, n)
		for u := range reach {
			reach[u] = make([]bool, n)
			reach[u][u] = true
		}
		density := 1 + rng.IntN(4)
		for u := 0; u < n; u++ {
			for v := 0; v < n; v++ {
				if rng.IntN(n*density/2+1) == 0 {
					adj[u] = append(adj[u], v)
					reach[u][v] = true
				}
			}
		}
		for k := 0; k < n; k++ {
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					if reach[i][k] && reach[k][j] {
						reach[i][j] = true
					}
				}
			}
		}
		var want [][]int
		seen := make([]bool, n)
		for v := 0; v < n; v++ {
			if seen[v] {
				continue
			}
			var comp []int
			for u := 0; u < n; u++ {
				if reach[v][u] && reach[u][v] {
					comp = append(comp, u)
					seen[u] = true
				}
			}
			want = append(want, comp)
		}
		got := Components(adj)
		if !slices.EqualFunc(got, want, slices.Equal[[]int]) {
			t.Fatalf("Components(%v) = %v, want %v", adj, got, want)
		}
	}
}

// TestHasCycleMutation pins the cycle test: a pair is a cycle, a lone node is
// one only with a self edge.
//
// Guard: gomu survivor components.go:168.
func TestHasCycleMutation(t *testing.T) {
	t.Parallel()
	adj := [][]int{{1}, {0}, {2}, {0}}
	for _, tc := range []struct {
		comp []int
		want bool
	}{{[]int{0, 1}, true}, {[]int{2}, true}, {[]int{3}, false}} {
		if got := hasCycle(tc.comp, adj); got != tc.want {
			t.Errorf("hasCycle(%v) = %v, want %v", tc.comp, got, tc.want)
		}
	}
}

// TestCyclesMutation checks V2 end to end on small exact programmes.
//
// Guard: gomu survivors components.go:205, 220, 235, 279, 282, 283.
func TestCyclesMutation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// A makes 2 of commodity 0 from 1 of commodity 1, B makes 1 of 1 from 1 of 0.
	// Net 0 = 2a - b, net 1 = b - a, a + b <= 1: s = 1/5 at a = 2/5, b = 3/5.
	pair := []Candidate{
		{Recipe: 0, Activity: full([][2]int64{{1, 1}}, [][2]int64{{0, 2}})},
		{Recipe: 1, Activity: full([][2]int64{{0, 1}}, [][2]int64{{1, 1}})},
	}
	t.Run("fractional surplus", func(t *testing.T) {
		t.Parallel()
		got, err := Cycles(ctx, 2, pair, 18, 1000)
		if err != nil || len(got) != 1 {
			t.Fatalf("Cycles = %v, %v, want one component", got, err)
		}
		g := got[0]
		if !g.Productive || g.SBP != 2000 || g.Skipped || g.Pivots < 1 {
			t.Errorf("component = %+v, want productive, SBP 2000, not skipped, pivots >= 1", g)
		}
	})
	t.Run("cell gate is exact", func(t *testing.T) {
		t.Parallel()
		// 3 rows * (2 members + 1 + 3 rows) = 18 cells: 18 solves, 17 skips.
		if got, _ := Cycles(ctx, 2, pair, 17, 1000); len(got) != 1 || !got[0].Skipped {
			t.Errorf("maxCells 17 gave %+v, want skipped", got)
		}
	})
	t.Run("a bystander is not a member", func(t *testing.T) {
		t.Parallel()
		by := append([]Candidate{{Recipe: 2, Activity: full(nil, [][2]int64{{2, 1}})}}, pair...)
		got, err := Cycles(ctx, 3, by, 18, 1000)
		if err != nil || len(got) != 1 || got[0].Skipped {
			t.Errorf("Cycles = %+v, %v, want one solved component under the 18-cell gate", got, err)
		}
	})
	t.Run("acyclic component first", func(t *testing.T) {
		t.Parallel()
		cs := []Candidate{
			{Recipe: 0, Activity: full(nil, [][2]int64{{0, 1}})},
			{Recipe: 1, Activity: full([][2]int64{{2, 1}}, [][2]int64{{1, 2}})},
			{Recipe: 2, Activity: full([][2]int64{{1, 1}}, [][2]int64{{2, 1}})},
		}
		got, err := Cycles(ctx, 3, cs, 1000, 1000)
		if err != nil || len(got) != 1 || !slices.Equal(got[0].Commodities, []int{1, 2}) {
			t.Errorf("Cycles = %+v, %v, want the component [1 2]", got, err)
		}
	})
	t.Run("surplus overflow", func(t *testing.T) {
		t.Parallel()
		// Net 1e18 - 1 per execution: floor(s * 10 000) is past int64.
		cs := []Candidate{{Recipe: 0, Activity: full([][2]int64{{0, 1}}, [][2]int64{{0, 1_000_000_000_000_000_000}})}}
		if _, err := Cycles(ctx, 1, cs, 1000, 1000); !errors.Is(err, ErrOverflow) {
			t.Errorf("Cycles error = %v, want ErrOverflow", err)
		}
	})
}

// TestCycleGainsMutation pins the quantity sums, the single-maker filter and the
// exact edge budget of the cycle enumeration.
//
// Guard: gomu survivors cycles.go:29, 47, 48, 72, 73, 104.
func TestCycleGainsMutation(t *testing.T) {
	t.Parallel()
	gain := func(gs []Gain) string {
		if len(gs) != 1 {
			return "not one gain"
		}
		return gs[0].Ratio.Num.String() + "/" + gs[0].Ratio.Den.String()
	}
	t.Run("repeated need and output quantities add up", func(t *testing.T) {
		t.Parallel()
		// Needs 0 twice (2 + 3) and a tool of 0 (1/2); outputs 0 twice (4 + 1),
		// a discard of 0 (100) and commodity 1 (9): (2+3+1/2) / (4+1) = 11/10.
		a := full([][2]int64{{0, 2}, {0, 3}}, [][2]int64{{0, 4}, {1, 9}, {0, 1}})
		a.Outputs = append(a.Outputs, Item{Commodity: 0, Qty: 100, Discard: true})
		a.Tools = []Tool{{Commodity: 0, Wear: 1, Durability: 2}}
		got, ok := CycleGains([]int{0}, []Candidate{{Recipe: 0, Activity: a}}, 100)
		if !ok || gain(got) != "11/10" {
			t.Errorf("CycleGains = %v, %v, want 11/10", got, ok)
		}
	})
	t.Run("zero output is no maker", func(t *testing.T) {
		t.Parallel()
		a := full([][2]int64{{0, 1}}, [][2]int64{{0, 0}})
		got, ok := CycleGains([]int{0}, []Candidate{{Recipe: 0, Activity: a}}, 100)
		if !ok || len(got) != 0 {
			t.Errorf("CycleGains = %v, %v, want none", got, ok)
		}
	})
	t.Run("a commodity without a maker does not stop the scan", func(t *testing.T) {
		t.Parallel()
		// Commodity 0 has no maker; commodity 1 is its own single maker: 3/4.
		a := full([][2]int64{{1, 3}}, [][2]int64{{1, 4}})
		got, ok := CycleGains([]int{0, 1}, []Candidate{{Recipe: 0, Activity: a}}, 100)
		if !ok || gain(got) != "3/4" {
			t.Errorf("CycleGains = %v, %v, want 3/4", got, ok)
		}
	})
	t.Run("budget is exact", func(t *testing.T) {
		t.Parallel()
		// 0 makes 1 and 1 makes 0: three edge visits in all (0>1, 1>0, and the
		// refused 1>0 from the start 1).
		cs := []Candidate{
			{Recipe: 0, Activity: full([][2]int64{{1, 1}}, [][2]int64{{0, 1}})},
			{Recipe: 1, Activity: full([][2]int64{{0, 1}}, [][2]int64{{1, 1}})},
		}
		if got, ok := CycleGains([]int{0, 1}, cs, 3); !ok || len(got) != 1 {
			t.Errorf("budget 3 gave %v, %v, want one gain", got, ok)
		}
		if got, ok := CycleGains([]int{0, 1}, cs, 2); ok || got != nil {
			t.Errorf("budget 2 gave %v, %v, want nil, false", got, ok)
		}
	})
}
