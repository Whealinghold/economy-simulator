package steady_test

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// The chains below are the prototype's (doc/specs/models/via/proto.py and
// proto3.py, spec-viability.md 15): invented recipes on the Earth calendar of 365
// ticks whose seasons start on day 59. The expected values are the exact
// fractions the prototype prints when run in Docker with
// PYTHONDONTWRITEBYTECODE=1, and the pivot counts are those of its column order:
// regions, recipes and commodities in the order listed here, not sorted by ID
// (the three-region pooled chain gives 24 pivots in this order and 13 sorted).

// pq is a quantity of a named commodity.
type pq struct {
	c string
	q int64
}

// recipe is a prototype recipe.
type recipe struct {
	name    string
	in, out []pq
	d       int64
	tools   []pq // q is the wear per use
	sited   bool
	discard string
}

// chain is a prototype chain.
type chain struct {
	recs   []recipe
	commod []string
	dur    map[string]int64
	prof   map[[2]string]map[string]int64 // (region, recipe) -> season or "all" -> factor
	upkeep []pq
	lit    bool // true: the seasons are lit (spring, summer) and dark
}

// seasonOf is the prototype's season of tick t.
func seasonOf(t int64, lit bool) string {
	d := ((t-59)%365 + 365) % 365
	s := "winter"
	switch {
	case d < 92:
		s = "spring"
	case d < 184:
		s = "summer"
	case d < 275:
		s = "autumn"
	}
	if lit {
		if s == "spring" || s == "summer" {
			return "lit"
		}
		return "dark"
	}
	return s
}

// yearOf returns the runs of recipe r in region g: one run for a recipe with no
// profile (0 if sited, 10 000 otherwise) or an "all" profile, else the factor of
// each tick's season, adjacent equal factors merged.
func (ch chain) yearOf(g string, r recipe) []steady.Run {
	prof, ok := ch.prof[[2]string{g, r.name}]
	if !ok {
		if r.sited {
			return []steady.Run{{Length: 365}}
		}
		return []steady.Run{{Length: 365, FactorBP: 10000}}
	}
	if f, all := prof["all"]; all {
		return []steady.Run{{Length: 365, FactorBP: f}}
	}
	var runs []steady.Run
	for t := int64(0); t < 365; t++ {
		f := prof[seasonOf(t, ch.lit)]
		if n := len(runs); n > 0 && runs[n-1].FactorBP == f {
			runs[n-1].Length++
		} else {
			runs = append(runs, steady.Run{Length: 1, FactorBP: f})
		}
	}
	return runs
}

// index returns the commodity index of name.
func (ch chain) index(name string) int { return slices.Index(ch.commod, name) }

// items converts quantities to items.
func (ch chain) items(ps []pq, discard string) []steady.Item {
	var out []steady.Item
	for _, p := range ps {
		out = append(out, steady.Item{Commodity: ch.index(p.c), Qty: p.q, Discard: p.c == discard && discard != ""})
	}
	return out
}

// model builds the programme model of the regions, in the order given, with the
// households h of each and the work share phi.
func (ch chain) model(regions []string, h []int64, phi steady.Ratio) steady.Model {
	m := steady.Model{Commodities: len(ch.commod), Year: 365, Phi: phi}
	for i, g := range regions {
		rg := steady.Region{Households: h[i], Upkeep: make([]int64, len(ch.commod))}
		for _, u := range ch.upkeep {
			rg.Upkeep[ch.index(u.c)] = h[i] * u.q
		}
		m.Regions = append(m.Regions, rg)
		for _, r := range ch.recs {
			sc := steady.Starts(r.d, ch.yearOf(g, r))
			if sc.Starts == 0 {
				continue
			}
			a := steady.Activity{Region: i, Duration: r.d, Starts: sc.Starts, FactorSum: sc.FactorSum,
				Inputs: ch.items(r.in, ""), Outputs: ch.items(r.out, r.discard)}
			for _, tl := range r.tools {
				a.Tools = append(a.Tools, steady.Tool{Commodity: ch.index(tl.c), Wear: tl.q, Durability: ch.dur[tl.c]})
			}
			m.Activities = append(m.Activities, a)
		}
	}
	return m
}

var full = steady.Ratio{Num: 1, Den: 1}

// breadQty are the quantities that differ between the variants of chain A.
type breadQty struct{ grain, seed, straw, flour, bran, dough, loaf, fuel, iron int64 }

// bread is chain A of spec-production 12.1 completed as the prototype does:
// grain, seed and straw per grow_grain execution, and the quantities of the
// other steps given by the arguments.
func bread(q breadQty) chain {
	grain, seed, straw, flour, bran, dough, loaf, fuel, iron := q.grain, q.seed, q.straw, q.flour, q.bran, q.dough, q.loaf, q.fuel, q.iron
	return chain{
		recs: []recipe{
			{name: "grow_grain", in: []pq{{"seed", seed}}, out: []pq{{"grain", grain}, {"straw", straw}}, d: 120, tools: []pq{{"hoe", 1}}, sited: true},
			{name: "save_seed", in: []pq{{"grain", 1}}, out: []pq{{"seed", 1}}, d: 1},
			{name: "mill_flour", in: []pq{{"grain", 3}}, out: []pq{{"flour", flour}, {"bran", bran}}, d: 1, tools: []pq{{"millstone", 1}}},
			{name: "knead_dough", in: []pq{{"flour", 1}}, out: []pq{{"dough", dough}}, d: 1},
			{name: "bake_loaf", in: []pq{{"dough", 2}, {"fuel", 1}}, out: []pq{{"loaf", loaf}}, d: 2, tools: []pq{{"oven", 1}}},
			{name: "cut_wood", out: []pq{{"fuel", fuel}}, d: 1},
			{name: "mine_iron", out: []pq{{"iron", iron}}, d: 1, sited: true},
			{name: "make_hoe", in: []pq{{"iron", 1}}, out: []pq{{"hoe", 1}}, d: 1},
			{name: "make_millstone", in: []pq{{"iron", 2}}, out: []pq{{"millstone", 1}}, d: 1},
			{name: "make_oven", in: []pq{{"iron", 2}}, out: []pq{{"oven", 1}}, d: 1},
		},
		commod: []string{"seed", "grain", "straw", "flour", "bran", "dough", "loaf", "fuel", "iron", "hoe", "millstone", "oven"},
		dur:    map[string]int64{"hoe": 2000, "millstone": 2000, "oven": 2000},
		prof: map[[2]string]map[string]int64{
			{"north", "grow_grain"}:     {"spring": 10000},
			{"south", "grow_grain"}:     {"spring": 12000, "autumn": 7000},
			{"far_south", "grow_grain"}: {"autumn": 10000},
			{"north", "mine_iron"}:      {"all": 10000}, {"south", "mine_iron"}: {"all": 5000}, {"far_south", "mine_iron"}: {"all": 15000},
			{"north", "cut_wood"}: {"all": 10000}, {"south", "cut_wood"}: {"all": 6000}, {"far_south", "cut_wood"}: {"all": 12000},
		},
		upkeep: []pq{{"loaf", 2}, {"fuel", 1}},
	}
}

// space is chain B of spec-production 12.2 completed and rescaled as the
// prototype does, with the lit and dark seasons.
func space() chain {
	return chain{
		recs: []recipe{
			{name: "cut_ice", out: []pq{{"ice", 30}}, d: 1, tools: []pq{{"drill", 2}}, sited: true},
			{name: "melt", in: []pq{{"ice", 1}}, out: []pq{{"water", 6}}, d: 1},
			{name: "grow_algae", in: []pq{{"water", 4}}, out: []pq{{"biomass", 120}}, d: 12, sited: true},
			{name: "dig_regolith", out: []pq{{"regolith", 30}}, d: 1},
			{name: "refine_wafer", in: []pq{{"regolith", 3}}, out: []pq{{"wafer", 1}}, d: 1},
			{name: "refine_solder", in: []pq{{"regolith", 2}}, out: []pq{{"solder", 1}}, d: 1},
			{name: "fab_board", in: []pq{{"wafer", 2}, {"solder", 1}}, out: []pq{{"board", 1}, {"slag", 1}}, d: 3, tools: []pq{{"fabricator", 1}}, discard: "slag"},
			{name: "make_drill", in: []pq{{"board", 1}}, out: []pq{{"drill", 1}}, d: 1},
			{name: "make_fabricator", in: []pq{{"board", 3}, {"solder", 1}}, out: []pq{{"fabricator", 1}}, d: 1},
		},
		commod: []string{"ice", "water", "biomass", "regolith", "wafer", "solder", "board", "slag", "drill", "fabricator"},
		dur:    map[string]int64{"drill": 500, "fabricator": 1000},
		prof: map[[2]string]map[string]int64{
			{"crater_rim", "cut_ice"}:     {"all": 6000},
			{"crater_rim", "grow_algae"}:  {"lit": 10000, "dark": 2500},
			{"deep_crater", "grow_algae"}: {"all": 0},
			{"deep_crater", "cut_ice"}:    {"all": 10000},
		},
		upkeep: []pq{{"biomass", 2}, {"water", 1}},
		lit:    true,
	}
}

// frac renders the surplus factor of an outcome.
func lambdaOf(o steady.Outcome) string { return str(o.Lambda) }

// dualProblems checks an outcome against the dual of the programme in exact
// arithmetic: for every activity the value added at the prices, sum p n, is at
// most the wage for its duration plus its capacity rent, with equality when it is
// active; and sum p D = 1 when lambda* is positive (FR-VIA-013).
func dualProblems(m steady.Model, o steady.Outcome) []string {
	var out []string
	for k, a := range m.Activities {
		va := new(big.Rat)
		for _, n := range a.Net() {
			va.Add(va, new(big.Rat).Mul(fracRat(o.Price[n.Commodity]), n.Val))
		}
		bound := new(big.Rat).Mul(big.NewRat(a.Duration, 1), fracRat(o.Wage[a.Region]))
		bound.Add(bound, fracRat(o.Rent[k]))
		if c := va.Cmp(bound); c > 0 || (c < 0 && o.Active[k]) {
			out = append(out, fmt.Sprintf("activity %d (active %v): value added %v against wage and rent %v", k, o.Active[k], va, bound))
		}
	}
	if fracRat(o.Lambda).Sign() > 0 {
		if norm := basketValue(m, o); norm.Cmp(big.NewRat(1, 1)) != 0 {
			out = append(out, fmt.Sprintf("sum p D = %v, want 1", norm))
		}
	}
	return out
}

// fracRat is the exact value of f.
func fracRat(f steady.Frac) *big.Rat { return new(big.Rat).SetFrac(f.Num, f.Den) }

// basketValue is sum over commodities of p(c) * D(c), D the upkeep of all regions.
func basketValue(m steady.Model, o steady.Outcome) *big.Rat {
	norm := new(big.Rat)
	for c := 0; c < m.Commodities; c++ {
		d := int64(0)
		for _, rg := range m.Regions {
			if c < len(rg.Upkeep) {
				d += rg.Upkeep[c]
			}
		}
		norm.Add(norm, new(big.Rat).Mul(fracRat(o.Price[c]), big.NewRat(d, 1)))
	}
	return norm
}

// TestSolveModel_PrototypeChains verifies the reference programme on the
// prototype's chains: lambda* as an exact fraction, its floor in basis points,
// the labour share where defined and the pivot count, and the dual feasibility of
// every activity. Values are the committed proto.py and proto3.py's, run in
// Docker (spec-viability.md 15, REF 7.3).
//
// Expected, e.g. chain A rescaled: lambda* = 11197440000/3756940819 = 2.98047,
// SurplusBP = floor(29804.7) = 29804, labour share = ceil(10^8 / 29804) = 3356
// (33.56%, the 33.6% of 15).
//
// Requirement: FR-VIA-003, FR-VIA-012, FR-VIA-013, FR-VIA-043
func TestSolveModel_PrototypeChains(t *testing.T) {
	t.Parallel()
	asWritten := bread(breadQty{6, 1, 2, 2, 1, 2, 3, 2, 3})
	stepsRescaled := bread(breadQty{6, 1, 2, 12, 6, 12, 18, 12, 18})
	rescaled := bread(breadQty{1500, 60, 100, 12, 6, 12, 18, 12, 18})
	north := []string{"north"}
	three := []string{"north", "south", "far_south"}
	tests := []struct {
		name    string
		ch      chain
		regions []string
		h       []int64
		lambda  string
		bp      int64
		share   int64 // 0: not asserted; -1: asserted absent
		pivots  int64
	}{
		// 1/73: chain A as written needs 73 times the labour.
		{"A as written", asWritten, north, []int64{1000}, "1/73", 136, -1, 11},
		{"A steps rescaled", stepsRescaled, north, []int64{1000}, "116640000/52135357", 22372, 4470, 11},
		{"A rescaled", rescaled, north, []int64{1000}, "11197440000/3756940819", 29804, 3356, 11},
		{"A south autarky", rescaled, []string{"south"}, []int64{1000}, "42456960/16867483", 25170, 3973, 11},
		{"A far south autarky", rescaled, []string{"far_south"}, []int64{1000}, "4199040000/1344044047", 31241, 3201, 11},
		// The three regions pooled: the best region sets the bound, 24 pivots in the
		// prototype's order.
		{"A three regions pooled", rescaled, three, []int64{1000, 1000, 1000}, "4199040000/1344044047", 31241, 0, 24},
		{"B rescaled", space(), []string{"crater_rim"}, []int64{1000}, "119621306250/60834283481", 19663, 5086, 10},
		{"B two regions pooled", space(), []string{"crater_rim", "deep_crater"}, []int64{600, 400}, "237/124", 19112, 5233, 18},
		// Deep crater cannot grow algae: lambda* = 0 in 3 pivots, no labour share.
		{"B deep crater alone", space(), []string{"deep_crater"}, []int64{400}, "0", 0, 0, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.ch.model(tt.regions, tt.h, full)
			o, err := steady.SolveModel(context.Background(), m, 10000)
			if err != nil {
				t.Fatalf("SolveModel(%s) error = %v, want nil", tt.name, err)
			}
			for _, p := range append(protoProblems(tt.lambda, tt.bp, tt.share, tt.pivots, o), dualProblems(m, o)...) {
				t.Errorf("SolveModel(%s): %s", tt.name, p)
			}
		})
	}
}

// protoProblems compares an outcome with the expected exact lambda*, its basis
// points, the labour share (0: not checked, -1: must be absent behind a tight
// capacity row) and the pivot count.
func protoProblems(lambda string, bp, share, pivots int64, o steady.Outcome) []string {
	var out []string
	if got := lambdaOf(o); got != lambda {
		out = append(out, fmt.Sprintf("Lambda = %s, want %s", got, lambda))
	}
	if o.SurplusBP != bp {
		out = append(out, fmt.Sprintf("SurplusBP = %d, want %d", o.SurplusBP, bp))
	}
	if o.Pivots != pivots {
		out = append(out, fmt.Sprintf("Pivots = %d, want %d", o.Pivots, pivots))
	}
	switch {
	case share < 0 && (o.HasLabourShare || !slices.Contains(o.CapTight, true)):
		out = append(out, fmt.Sprintf("labour share (%d, %v) with CapTight %v, want it absent behind a tight capacity row", o.LabourShareBP, o.HasLabourShare, o.CapTight))
	case share > 0 && (!o.HasLabourShare || o.LabourShareBP != share):
		out = append(out, fmt.Sprintf("labour share = (%d, %v), want (%d, true)", o.LabourShareBP, o.HasLabourShare, share))
	}
	return out
}

// TestSolveModel_SortedColumnOrderChangesPivots verifies that the order of the
// columns is part of the input: the three-region pooled chain with regions,
// recipes and commodities sorted by ID takes 13 pivots, not the 24 of the
// prototype's order, and reaches the same lambda*.
//
// Guard: Bland's rule makes the pivot count depend on the indexing; the pivot
// counts of REF are claims about one order (A2). If the package silently sorted
// its input, the 24 would be lost without any visible change of lambda*.
func TestSolveModel_SortedColumnOrderChangesPivots(t *testing.T) {
	t.Parallel()
	ch := bread(breadQty{1500, 60, 100, 12, 6, 12, 18, 12, 18})
	slices.Sort(ch.commod)
	slices.SortFunc(ch.recs, func(a, b recipe) int { return strings.Compare(a.name, b.name) })
	m := ch.model([]string{"far_south", "north", "south"}, []int64{1000, 1000, 1000}, full)
	o, err := steady.SolveModel(context.Background(), m, 10000)
	if err != nil {
		t.Fatalf("SolveModel(sorted regions) error = %v, want nil", err)
	}
	if got := lambdaOf(o); got != "4199040000/1344044047" {
		t.Errorf("SolveModel(sorted regions).Lambda = %s, want 4199040000/1344044047", got)
	}
	if o.Pivots != 13 {
		t.Errorf("SolveModel(sorted regions).Pivots = %d, want 13", o.Pivots)
	}
}
