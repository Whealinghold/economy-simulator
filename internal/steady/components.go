package steady

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"sort"
)

// Candidate is an activity offered to the cycle checks, tagged with its recipe.
// The same recipe may appear once per region with different start counts; the
// checks take the best mean factor across regions (spec-viability.md 4.3).
type Candidate struct {
	Recipe int
	Activity
}

// bestPerRecipe keeps, for each recipe, the candidate with the largest mean
// factor FactorSum/Starts (the first on a tie), in recipe order. The comparison
// is by cross-multiplication in big, since the products can exceed int64.
func bestPerRecipe(cs []Candidate) []Candidate {
	at := map[int]int{}
	var out []Candidate
	for _, c := range cs {
		i, seen := at[c.Recipe]
		if !seen {
			at[c.Recipe] = len(out)
			out = append(out, c)
			continue
		}
		l := new(big.Int).Mul(big.NewInt(c.FactorSum), big.NewInt(out[i].Starts))
		r := new(big.Int).Mul(big.NewInt(out[i].FactorSum), big.NewInt(c.Starts))
		if l.Cmp(r) > 0 {
			out[i] = c
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Recipe < out[j].Recipe })
	return out
}

// outputs reports whether the activity has a non-discard output of commodity c.
func (a Activity) outputs(c int) bool {
	for _, o := range a.Outputs {
		if o.Commodity == c && !o.Discard {
			return true
		}
	}
	return false
}

// MaterialGraph returns the material graph of 4.3 over n commodities: an edge
// from each input and each required tool of an activity to each of its
// non-discard outputs. Each successor list is ascending and without repeats.
func MaterialGraph(n int, acts []Activity) [][]int {
	adj := make([][]int, n)
	for _, a := range acts {
		var from []int
		for _, in := range a.Inputs {
			from = append(from, in.Commodity)
		}
		for _, t := range a.Tools {
			from = append(from, t.Commodity)
		}
		for _, o := range a.Outputs {
			if o.Discard {
				continue
			}
			for _, u := range from {
				adj[u] = append(adj[u], o.Commodity)
			}
		}
	}
	for u := range adj {
		slices.Sort(adj[u])
		adj[u] = slices.Compact(adj[u])
	}
	return adj
}

// frame is one level of the explicit stack of Components.
type frame struct{ node, next int }

// tarjan holds the state of one run of Tarjan's algorithm.
type tarjan struct {
	adj        [][]int
	index, low []int
	onStack    []bool
	stack      []int
	counter    int
	components [][]int
	pending    []frame
	visited    []bool
}

// visit starts a depth-first search at root without recursion, so the depth is
// not bounded by the number of nodes.
func (t *tarjan) visit(root int) {
	t.enter(root)
	for len(t.pending) > 0 {
		f := &t.pending[len(t.pending)-1]
		if f.next < len(t.adj[f.node]) {
			w := t.adj[f.node][f.next]
			f.next++
			if !t.visited[w] {
				t.enter(w)
			} else if t.onStack[w] {
				t.low[f.node] = min(t.low[f.node], t.index[w])
			}
			continue
		}
		v := f.node
		t.pending = t.pending[:len(t.pending)-1]
		if t.low[v] == t.index[v] {
			t.pop(v)
		}
		if len(t.pending) > 0 {
			p := t.pending[len(t.pending)-1].node
			t.low[p] = min(t.low[p], t.low[v])
		}
	}
}

// enter marks v discovered and pushes it.
func (t *tarjan) enter(v int) {
	t.visited[v] = true
	t.index[v], t.low[v] = t.counter, t.counter
	t.counter++
	t.stack = append(t.stack, v)
	t.onStack[v] = true
	t.pending = append(t.pending, frame{node: v})
}

// pop closes the component rooted at v.
func (t *tarjan) pop(v int) {
	var comp []int
	for {
		w := t.stack[len(t.stack)-1]
		t.stack = t.stack[:len(t.stack)-1]
		t.onStack[w] = false
		comp = append(comp, w)
		if w == v {
			break
		}
	}
	slices.Sort(comp)
	t.components = append(t.components, comp)
}

// Components returns the strongly connected components of the graph adj, each
// ascending and the list ordered by lowest member (canonical order,
// FR-VIA-006). The search is iterative.
func Components(adj [][]int) [][]int {
	n := len(adj)
	t := &tarjan{adj: adj, index: make([]int, n), low: make([]int, n), onStack: make([]bool, n), visited: make([]bool, n)}
	for v := 0; v < n; v++ {
		if !t.visited[v] {
			t.visit(v)
		}
	}
	sort.Slice(t.components, func(i, j int) bool { return t.components[i][0] < t.components[j][0] })
	return t.components
}

// hasCycle reports whether the component holds a cycle: more than one node, or
// a node with an edge to itself.
func hasCycle(comp []int, adj [][]int) bool {
	if len(comp) > 1 {
		return true
	}
	return slices.Contains(adj[comp[0]], comp[0])
}

// CycleComponent is the verdict of V2 on one strongly connected component that
// holds a cycle. SBP is floor(s* * 10 000). Skipped is set when the tableau is
// over the size gate or the solve reached the pivot cap (D126); then Productive
// and SBP mean nothing.
type CycleComponent struct {
	Commodities []int
	Productive  bool
	SBP         int64
	Skipped     bool
	Pivots      int64
}

// Cycles runs V2 (spec-viability.md 4.3, FR-VIA-006): for each component of the
// material graph that holds a cycle, the programme "maximise s with every
// commodity of the component netting at least s from a unit mix of activities",
// at each recipe's best mean factor across regions, solved exactly. The members
// of a component are the activities with a non-discard output in it, each with
// its net vector restricted to it, so a source activity belongs. maxCells
// bounds rows times columns and maxPivots the pivots; a component over either
// is reported Skipped. Only a cancelled context or a failed certificate is an
// error.
func Cycles(ctx context.Context, n int, cs []Candidate, maxCells, maxPivots int64) ([]CycleComponent, error) {
	best := bestPerRecipe(cs)
	acts := make([]Activity, len(best))
	for i, c := range best {
		acts[i] = c.Activity
	}
	adj := MaterialGraph(n, acts)
	var out []CycleComponent
	for _, comp := range Components(adj) {
		if !hasCycle(comp, adj) {
			continue
		}
		r, err := solveComponent(ctx, comp, acts, maxCells, maxPivots)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// members returns the activities with a non-discard output in the component.
func members(comp []int, acts []Activity) []Activity {
	var out []Activity
	for _, a := range acts {
		if slices.ContainsFunc(comp, a.outputs) {
			out = append(out, a)
		}
	}
	return out
}

// cycleProgramme builds "maximise s" for the component: a row per commodity,
// -net x + s <= 0, and a last row sum x <= 1 (at the optimum it holds with
// equality whenever s* is positive).
func cycleProgramme(comp []int, mem []Activity) Programme {
	pos := map[int]int{}
	for i, c := range comp {
		pos[c] = i
	}
	rows := len(comp) + 1
	p := Programme{Rows: rows, RHS: make([]Ratio, rows)}
	for i := range p.RHS {
		p.RHS[i] = Ratio{Num: 0, Den: 1}
	}
	p.RHS[rows-1] = Ratio{Num: 1, Den: 1}
	for _, a := range mem {
		col := Column{Cost: Ratio{Num: 0, Den: 1}}
		for _, ne := range a.Net() {
			if i, in := pos[ne.Commodity]; in {
				col.Entries = append(col.Entries, Entry{Row: i, Wide: new(big.Rat).Neg(ne.Val)})
			}
		}
		col.Entries = append(col.Entries, Entry{Row: rows - 1, Val: Ratio{Num: 1, Den: 1}})
		p.Cols = append(p.Cols, col)
	}
	s := Column{Cost: Ratio{Num: 1, Den: 1}}
	for i := range comp {
		s.Entries = append(s.Entries, Entry{Row: i, Val: Ratio{Num: 1, Den: 1}})
	}
	p.Cols = append(p.Cols, s)
	return p
}

// solveComponent builds and solves the programme of one component.
func solveComponent(ctx context.Context, comp []int, acts []Activity, maxCells, maxPivots int64) (CycleComponent, error) {
	res := CycleComponent{Commodities: comp}
	mem := members(comp, acts)
	rows := int64(len(comp) + 1)
	cells := new(big.Int).Mul(big.NewInt(rows), big.NewInt(int64(len(mem))+1+rows))
	if cells.Cmp(big.NewInt(maxCells)) > 0 {
		res.Skipped = true
		return res, nil
	}
	p := cycleProgramme(comp, mem)
	sol, err := Solve(ctx, p, maxPivots)
	if errors.Is(err, ErrLimit) {
		res.Skipped = true
		return res, nil
	}
	if err != nil {
		return CycleComponent{}, err
	}
	v := sol.Primal[len(p.Cols)-1]
	res.Pivots = sol.Pivots
	res.Productive = v.Num.Sign() > 0
	bp, ok := Floor(new(big.Int).Mul(v.Num, big.NewInt(10000)), v.Den, 0, maxInt64)
	if !ok {
		return CycleComponent{}, ErrOverflow
	}
	res.SBP = bp
	return res, nil
}
