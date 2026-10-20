package steady

import (
	"math/big"
	"slices"
)

// Gain is the input-to-output ratio around one simple cycle of the material
// graph: the product over its recipes of q_in / q_out, with the configured
// quantities and no yield factor (spec-viability.md 4.3, FR-CFG-044).
type Gain struct {
	Commodities []int
	Ratio       Frac
}

// maker is the one non-source activity that makes a commodity, with the ratio of
// each of its needs to its output of that commodity.
type maker struct {
	needs []int
	ratio map[int]*big.Rat
}

// quantities returns the configured quantity of each input and tool of the
// activity (a tool counts Wear/Durability, its flow per execution), summed per
// commodity.
func quantities(a Activity) map[int]*big.Rat {
	q := map[int]*big.Rat{}
	add := func(c int, v *big.Rat) {
		if q[c] == nil {
			q[c] = new(big.Rat)
		}
		q[c].Add(q[c], v)
	}
	for _, in := range a.Inputs {
		add(in.Commodity, new(big.Rat).SetInt64(in.Qty))
	}
	for _, t := range a.Tools {
		add(t.Commodity, big.NewRat(t.Wear, t.Durability))
	}
	return q
}

// outQty returns the summed non-discard output of c per execution.
func outQty(a Activity, c int) int64 {
	var n int64
	for _, o := range a.Outputs {
		if o.Commodity == c && !o.Discard {
			n += o.Qty
		}
	}
	return n
}

// singleMakers returns, for each commodity of the component that has exactly
// one maker (a non-source activity, one with inputs or tools, that outputs it)
// with a positive output, that maker's edge ratios.
func singleMakers(comp []int, acts []Candidate) map[int]maker {
	count := map[int]int{}
	found := map[int]Activity{}
	for _, c := range comp {
		for _, a := range acts {
			if (len(a.Inputs)+len(a.Tools) > 0) && a.outputs(c) {
				count[c]++
				found[c] = a.Activity
			}
		}
	}
	out := map[int]maker{}
	for _, c := range comp {
		a := found[c]
		q := outQty(a, c)
		if count[c] != 1 || q < 1 {
			continue
		}
		m := maker{ratio: map[int]*big.Rat{}}
		for u, qin := range quantities(a) {
			if slices.Contains(comp, u) {
				m.needs = append(m.needs, u)
				m.ratio[u] = new(big.Rat).Quo(qin, new(big.Rat).SetInt64(q))
			}
		}
		slices.Sort(m.needs)
		out[c] = m
	}
	return out
}

// walker enumerates simple cycles with a step budget.
type walker struct {
	succ   map[int][]int
	makers map[int]maker
	budget int64
	gains  []Gain
	over   bool
}

// dfs extends the path from node v; start is the lowest node of the cycles
// sought, so each cycle is found once. ratio is the product along the path.
func (w *walker) dfs(start, v int, path []int, ratio *big.Rat) {
	for _, c := range w.succ[v] {
		if w.over {
			return
		}
		if w.budget--; w.budget < 0 {
			w.over = true
			return
		}
		r := new(big.Rat).Mul(ratio, w.makers[c].ratio[v])
		switch {
		case c == start:
			w.gains = append(w.gains, Gain{Commodities: slices.Clone(path), Ratio: Frac{Num: r.Num(), Den: r.Denom()}})
		case c > start && !slices.Contains(path, c):
			w.dfs(start, c, append(path, c), r)
		}
	}
}

// CycleGains lists the simple cycles of a cyclic component whose every
// commodity has one maker in it, with the gain of each (spec-viability.md 4.3,
// JC6). A source activity is not a maker, since it supplies the commodity from
// outside the cycle. The enumeration stops after maxSteps edge visits and then
// returns ok false and no cycles: the part "cycle gain" of the skipped check.
// Cycles are ordered by their lowest commodity, then by path.
func CycleGains(comp []int, cs []Candidate, maxSteps int64) (gains []Gain, ok bool) {
	best := bestPerRecipe(cs)
	makers := singleMakers(comp, best)
	succ := map[int][]int{}
	for _, c := range comp {
		for _, u := range makers[c].needs {
			if _, single := makers[u]; single {
				succ[u] = append(succ[u], c)
			}
		}
	}
	w := &walker{succ: succ, makers: makers, budget: maxSteps}
	for _, s := range comp {
		if _, single := makers[s]; single {
			w.dfs(s, s, []int{s}, big.NewRat(1, 1))
		}
	}
	if w.over {
		return nil, false
	}
	return w.gains, true
}
