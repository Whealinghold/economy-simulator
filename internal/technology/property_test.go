package technology_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/technology"
)

// The property tests of Step over random inputs from fixed seeds. A failure names
// the seed, so one call of randomInput(seed) reproduces it (testing.md 6.2).

// randomInput is a valid input of one to three fields over one to five regions,
// every number inside the bounds of spec-technology.md 9.2 and a stored state that
// a world could hold: an advance in [0, cap], an experience in [0, maxExp]. Every
// field has every mechanism on, so each phase of the step has something to do.
func randomInput(seed uint64, withDiscovery bool) technology.Input {
	rng := rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))
	nf, regions := 1+rng.IntN(3), 1+rng.IntN(5)
	in := technology.Input{
		Cells:         make([]technology.Cell, nf*regions),
		Counters:      make([]int64, nf*regions),
		TrendCarry:    make([]int64, nf),
		Fields:        make([]technology.Field, nf),
		Regions:       regions,
		Tick:          int64(rng.IntN(5000)),
		TicksPerYear:  []int64{1, 12, 365, 1000}[rng.IntN(4)],
		MaxExperience: maxExp,
		Key:           rng.Uint64(),
		Dist:          matrix(randomDistances(rng, regions)),
	}
	for f := range in.Fields {
		in.Fields[f] = technology.Field{
			CapBP: 1 + rng.Int64N(100_000), Base: 1 + rng.Int64N(1_000_000), DoublingBP: rng.Int64N(10_001),
			TrendBP: rng.Int64N(10_001), CatchUpBP: rng.Int64N(10_001), Reach: 1 + rng.Int64N(500),
		}
		if withDiscovery {
			in.Fields[f].DiscoveryBP, in.Fields[f].StepBP = rng.Int64N(100_001), rng.Int64N(100_001)
		}
		in.TrendCarry[f] = rng.Int64N(in.TicksPerYear)
	}
	for i := range in.Cells {
		f := in.Fields[i%nf]
		in.Cells[i] = technology.Cell{Experience: rng.Int64N(1 << 20), Advance: rng.Int64N(f.CapBP + 1), CatchUp: rng.Int64N(10_000 * in.TicksPerYear)}
		in.Counters[i] = rng.Int64N(50) * rng.Int64N(2)
	}
	return in
}

// randomDistances is a symmetric table of distances in [1, 400], with about one
// pair in five out of contact (-1).
func randomDistances(rng *rand.Rand, n int) [][]int64 {
	d := make([][]int64, n)
	for i := range d {
		d[i] = make([]int64, n)
		d[i][i] = -1
	}
	for i := range n {
		for j := i + 1; j < n; j++ {
			v := 1 + rng.Int64N(400)
			if rng.IntN(5) == 0 {
				v = -1
			}
			d[i][j], d[j][i] = v, v
		}
	}
	return d
}

// cloneInput copies every slice the step may write.
func cloneInput(in technology.Input) technology.Input {
	in.Cells = slices.Clone(in.Cells)
	in.TrendCarry = slices.Clone(in.TrendCarry)
	return in
}

// runStep runs Step on a copy of in with the scratch given and returns the results
// and the state after.
func runStep(t *testing.T, in technology.Input, s *technology.Scratch) ([]technology.Result, technology.Input) {
	t.Helper()
	in = cloneInput(in)
	out := make([]technology.Result, len(in.Cells))
	if !technology.Step(in, s, out) {
		t.Fatalf("Step refused a valid input %+v", in)
	}
	return out, in
}

// TestStep_GainFollowsTheLaw verifies, over 300 random valid inputs, the law of
// the gain after a step: the gain is min(cap, learned(experience) + advance), the
// stored advance is inside [0, cap], the experience is the old one plus the counter
// saturated at the bound, the gain never falls and the learned part added is the
// difference of the two experiences.
//
// Requirement: FR-TEC-005, FR-TEC-006, FR-TEC-035
func TestStep_GainFollowsTheLaw(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 300; seed++ {
		in := randomInput(seed, true)
		out, after := runStep(t, in, &technology.Scratch{})
		nf := len(in.Fields)
		for i, r := range out {
			f := in.Fields[i%nf]
			x0, c := in.Cells[i].Experience, in.Counters[i]
			if want := min(x0+c, maxExp); r.Experience != want || after.Cells[i].Experience != want {
				t.Fatalf("seed %d cell %d: experience %d (stored %d), want %d", seed, i, r.Experience, after.Cells[i].Experience, want)
			}
			if r.Advance < 0 || r.Advance > f.CapBP {
				t.Fatalf("seed %d cell %d: advance %d outside [0, %d]", seed, i, r.Advance, f.CapBP)
			}
			l0, _ := technology.Learned(f.DoublingBP, f.Base, x0)
			l1, _ := technology.Learned(f.DoublingBP, f.Base, r.Experience)
			if r.Doing != l1-l0 {
				t.Fatalf("seed %d cell %d: Doing %d, want learned(%d) - learned(%d) = %d", seed, i, r.Doing, r.Experience, x0, l1-l0)
			}
			if want := min(f.CapBP, l1+r.Advance); r.After != want {
				t.Fatalf("seed %d cell %d: After %d, want min(cap %d, learned %d + advance %d) = %d", seed, i, r.After, f.CapBP, l1, r.Advance, want)
			}
			if r.After < r.Before {
				t.Fatalf("seed %d cell %d: gain fell from %d to %d with no call", seed, i, r.Before, r.After)
			}
		}
	}
}

// TestStep_SameInputSameOutput verifies that Step is a function of its input: a
// fresh scratch, a scratch left dirty by a larger shape and a second run give
// identical results and identical stored state, for 100 seeds; and that two
// different keys do not give the same discoveries in every one of those seeds, so
// the key is read.
//
// Requirement: NFR-TEC-001, FR-TEC-033
func TestStep_SameInputSameOutput(t *testing.T) {
	t.Parallel()
	dirty := &technology.Scratch{}
	wide := randomInput(7, true)
	wide.Regions, wide.Cells, wide.Counters = 1, wide.Cells[:len(wide.Fields)], wide.Counters[:len(wide.Fields)]
	runStep(t, wide, dirty)
	differs := false
	for seed := uint64(1); seed <= 100; seed++ {
		in := randomInput(seed, true)
		a, sa := runStep(t, in, &technology.Scratch{})
		b, sb := runStep(t, in, dirty)
		c, sc := runStep(t, in, dirty)
		if !slices.Equal(a, b) || !slices.Equal(a, c) || !slices.Equal(sa.Cells, sb.Cells) ||
			!slices.Equal(sa.Cells, sc.Cells) || !slices.Equal(sa.TrendCarry, sb.TrendCarry) {
			t.Fatalf("seed %d: the same input gave different results with a fresh and a reused scratch", seed)
		}
		in.Key++
		d, _ := runStep(t, in, &technology.Scratch{})
		differs = differs || !slices.Equal(a, d)
	}
	if !differs {
		t.Errorf("a changed key gave the same results in all 100 seeds, want the key to move some discovery")
	}
}

// TestStep_RegionOrderIsNotRead verifies that a permutation of the regions (their
// cells, counters and the distance table) permutes the results and changes
// nothing else, with learning, trend and diffusion on: the frontier of a region is
// the maximum over the others of the snapshot, so no visiting order shows. The
// keyed discovery is left off: a hit is keyed by the position of the cell by design.
//
// Requirement: NFR-TEC-002, FR-TEC-013
func TestStep_RegionOrderIsNotRead(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 100; seed++ {
		in := randomInput(seed, false)
		nf, n := len(in.Fields), in.Regions
		rng := rand.New(rand.NewPCG(seed, 99))
		perm := rng.Perm(n)
		d := randomDistances(rand.New(rand.NewPCG(seed, 5)), n)
		in.Dist = matrix(d)
		moved := cloneInput(in)
		moved.Counters = slices.Clone(in.Counters)
		pd := make([][]int64, n)
		for a := range pd {
			pd[a] = make([]int64, n)
			for b := range pd[a] {
				pd[a][b] = d[perm[a]][perm[b]]
			}
			for f := range nf {
				moved.Cells[a*nf+f], moved.Counters[a*nf+f] = in.Cells[perm[a]*nf+f], in.Counters[perm[a]*nf+f]
			}
		}
		moved.Dist = matrix(pd)
		base, sb := runStep(t, in, &technology.Scratch{})
		got, sg := runStep(t, moved, &technology.Scratch{})
		for a := range n {
			for f := range nf {
				if got[a*nf+f] != base[perm[a]*nf+f] || sg.Cells[a*nf+f] != sb.Cells[perm[a]*nf+f] {
					t.Fatalf("seed %d perm %v: region %d field %d = %+v, want the result of region %d, %+v",
						seed, perm, a, f, got[a*nf+f], perm[a], base[perm[a]*nf+f])
				}
			}
		}
		if !slices.Equal(sb.TrendCarry, sg.TrendCarry) {
			t.Fatalf("seed %d perm %v: trend carries %v and %v differ", seed, perm, sb.TrendCarry, sg.TrendCarry)
		}
	}
}

// edge is a pool of values at and near the edges of int64 and of the bounds.
var edge = []int64{0, 1, 2, 9_999, 10_000, 1 << 31, 1 << 50, 1<<62 - 1, 1 << 62, math.MaxInt64 - 1, math.MaxInt64}

// edgeInput is an input whose every number is drawn from edge, shaped validly, so
// that the step meets overflow in each of its phases.
func edgeInput(seed uint64) technology.Input {
	rng := rand.New(rand.NewPCG(seed, 0xED6E))
	pick := func() int64 { return edge[rng.IntN(len(edge))] }
	in := randomInput(seed, true)
	in.TicksPerYear = max(1, pick())
	in.Tick = pick()
	for f := range in.Fields {
		in.Fields[f] = technology.Field{CapBP: pick(), Base: pick(), DoublingBP: pick(), TrendBP: pick(),
			DiscoveryBP: pick(), StepBP: pick(), CatchUpBP: pick(), Reach: pick()}
		in.TrendCarry[f] = pick()
	}
	for i := range in.Cells {
		in.Cells[i] = technology.Cell{Experience: pick() % (maxExp + 1), Advance: pick(), CatchUp: pick()}
		in.Counters[i] = pick()
	}
	return in
}

// exactSum is a + b + c + d + e - f in unbounded integers.
func exactSum(r technology.Result) *big.Int {
	sum := big.NewInt(r.Before)
	for _, v := range []int64{r.Doing, r.Trend, r.Discovery, r.CatchUp} {
		sum.Add(sum, big.NewInt(v))
	}
	return sum.Sub(sum, big.NewInt(r.After))
}

// TestStep_NoWrapAtTheInt64Edges verifies, over 2 000 inputs whose numbers sit at
// and near the edges of int64, that Step never panics and never wraps: it either
// refuses and leaves the cells and the carries as they were, or accepts and every
// number it writes is not negative, the stored advance is inside the cap and the
// clamp identity holds in unbounded integers. The refusals of Learned, of the
// trend, of the catch-up and of the stored advance are the ones that keep an
// accepted result exact.
//
// Requirement: NFR-TEC-005, FR-TEC-017, FR-TEC-035
func TestStep_NoWrapAtTheInt64Edges(t *testing.T) {
	t.Parallel()
	accepted := 0
	for seed := uint64(1); seed <= 2000; seed++ {
		in := edgeInput(seed)
		before := cloneInput(in)
		out := make([]technology.Result, len(in.Cells))
		if !technology.Step(in, &technology.Scratch{}, out) {
			if !slices.Equal(before.Cells, in.Cells) || !slices.Equal(before.TrendCarry, in.TrendCarry) {
				t.Fatalf("seed %d: a refusal changed the state", seed)
			}
			continue
		}
		accepted++
		for i, r := range out {
			vals := []int64{r.Before, r.After, r.Doing, r.Trend, r.Discovery, r.CatchUp, r.Clamped, r.Experience, r.Advance, r.CatchUpCarry}
			if slices.ContainsFunc(vals, func(v int64) bool { return v < 0 }) {
				t.Fatalf("seed %d cell %d: a negative number in %+v (a wrap)", seed, i, r)
			}
			if r.Advance > in.Fields[i%len(in.Fields)].CapBP || r.Experience > maxExp {
				t.Fatalf("seed %d cell %d: advance or experience past its bound in %+v", seed, i, r)
			}
			if exactSum(r).Cmp(big.NewInt(r.Clamped)) != 0 {
				t.Fatalf("seed %d cell %d: before + additions - after = %v, want Clamped %d", seed, i, exactSum(r), r.Clamped)
			}
		}
	}
	if accepted == 0 || accepted == 2000 {
		t.Errorf("accepted %d of 2000 edge inputs, want some of each so that both paths are exercised", accepted)
	}
}

// TestLearned_NeverFalls verifies that the learned gain does not fall as the
// experience grows, for 50 fields over 200 ascending experiences each, so the
// difference of two learned gains that Step books as Doing is never negative.
//
// Requirement: FR-TEC-005, FR-TEC-006
func TestLearned_NeverFalls(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 50; seed++ {
		rng := rand.New(rand.NewPCG(seed, 3))
		doubling, base := rng.Int64N(10_001), 1+rng.Int64N(1_000_000)
		var x, prev int64
		for range 200 {
			x += rng.Int64N(1 << 30)
			g, ok := technology.Learned(doubling, base, x)
			if !ok || g < prev {
				t.Fatalf("seed %d: Learned(%d, %d, %d) = %d, %v after %d, want a gain that does not fall", seed, doubling, base, x, g, ok, prev)
			}
			prev = g
		}
	}
}

// TestCatchUp_RefusesEachOverflow verifies that the catch-up refuses, and wraps in
// none of them, when the year length times 10 000, the gap times the rate or the
// carry plus that product does not fit int64.
//
// The arithmetic: 10 000 * MaxInt64 overflows the divisor; 1<<62 * 4 overflows the
// product; MaxInt64 carry plus a product of 1 overflows the sum.
//
// Requirement: NFR-TEC-005
func TestCatchUp_RefusesEachOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                      string
		carry, gap, rate, perYear int64
	}{
		{"divisor", 0, 1, 1, math.MaxInt64},
		{"product", 0, 1 << 62, 4, 365},
		{"sum", math.MaxInt64, 1, 1, 365},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if add, c, ok := technology.CatchUp(tt.carry, tt.gap, tt.rate, tt.perYear); ok {
				t.Errorf("CatchUp(%d, %d, %d, %d) = %d, %d, true, want a refusal", tt.carry, tt.gap, tt.rate, tt.perYear, add, c)
			}
		})
	}
}
