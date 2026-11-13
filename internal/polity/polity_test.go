package polity_test

import (
	"errors"
	"math"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/polity"
)

// Example A of spec-polity.md 12.1. Regions by ID: capital 0, farm 1, harbour 2,
// hills 3, mill 4. Jurisdictions by ID: coast 0, crown 1, inland 2, uplands 3.
const (
	capital, farm, harbour, hills, mill = 0, 1, 2, 3, 4
	coast, crown, inland, uplands       = 0, 1, 2, 3
)

func exampleA(t *testing.T) polity.Tables {
	t.Helper()
	tb, err := polity.Build(
		[]int32{crown, -1, crown, inland}, // coast>crown, crown root, inland>crown, uplands>inland
		[]int32{crown, inland, coast, uplands, inland},
		8)
	if err != nil {
		t.Fatalf("Build(example A) = %v, want nil", err)
	}
	return tb
}

// Example B of spec-polity.md 12.2. Regions: ashford 0, brook 1, city 2, delta 3,
// far 4. Jurisdictions: east 0, west 1, both roots.
func exampleB(t *testing.T) polity.Tables {
	t.Helper()
	tb, err := polity.Build([]int32{-1, -1}, []int32{0, 1, -1, 1, -1}, 8)
	if err != nil {
		t.Fatalf("Build(example B) = %v, want nil", err)
	}
	return tb
}

// TestBuild_Chains verifies the five chains of example A, innermost first, and
// that a region with no row has an empty chain.
//
// Requirement: FR-POL-006, FR-POL-007
func TestBuild_Chains(t *testing.T) {
	t.Parallel()
	tb := exampleA(t)
	want := map[int32][]int32{
		capital: {crown},
		farm:    {inland, crown},
		harbour: {coast, crown},
		hills:   {uplands, inland, crown},
		mill:    {inland, crown},
	}
	for r, w := range want {
		if got := tb.Chain(r); !slices.Equal(got, w) {
			t.Errorf("Chain(%d) = %v, want %v", r, got, w)
		}
		for j := int32(0); j < 4; j++ {
			if got, wantCovers := tb.Covers(j, r), slices.Contains(w, j); got != wantCovers {
				t.Errorf("Covers(%d, %d) = %v, want %v", j, r, got, wantCovers)
			}
		}
	}
	free := exampleB(t)
	if got := free.Chain(2); len(got) != 0 {
		t.Errorf("Chain(city) of example B = %v, want empty", got)
	}
}

// TestBuild_Common verifies Common over all 25 ordered pairs of example A and the
// border depth the spec derives from it (4.3.1).
//
// Requirement: FR-POL-008
func TestBuild_Common(t *testing.T) {
	t.Parallel()
	tb := exampleA(t)
	// Rows of the spec's table (12.1) with the border depth, then the rest by the
	// model of the climb below.
	spec := []struct {
		a, b   int32
		common int32
		depth  int
	}{
		{hills, harbour, crown, 0},
		{harbour, hills, crown, 0},
		{farm, mill, inland, 1},
		{farm, capital, crown, 0},
		{hills, farm, inland, 1},
		{capital, mill, crown, 0},
	}
	for _, tt := range spec {
		got := tb.Common(tt.a, tt.b)
		if got != tt.common {
			t.Errorf("Common(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.common)
		}
		if d, ok := tb.Depth(got); !ok || d != tt.depth {
			t.Errorf("Depth(Common(%d, %d)) = %d, %v, want %d, true", tt.a, tt.b, d, ok, tt.depth)
		}
	}
	f := forest{parent: []int32{crown, -1, crown, inland}, member: []int32{crown, inland, coast, uplands, inland}}
	for a := 0; a < 5; a++ {
		for b := 0; b < 5; b++ {
			if got, want := tb.Common(int32(a), int32(b)), f.modelCommon(a, b); got != want {
				t.Errorf("Common(%d, %d) = %d, model %d", a, b, got, want)
			}
		}
	}
	b := exampleB(t)
	if got := b.Common(0, 1); got != -1 {
		t.Errorf("Common(ashford, brook) of example B = %d, want -1 (two roots share nothing)", got)
	}
}

// TestBuild_Regions verifies that a jurisdiction covers the direct members of its
// descendants, ascending by region.
//
// Requirement: FR-POL-009
func TestBuild_Regions(t *testing.T) {
	t.Parallel()
	tb := exampleA(t)
	want := map[int32][]int32{
		coast:   {harbour},
		crown:   {capital, farm, harbour, hills, mill}, // every region: the crown covers the tree
		inland:  {farm, hills, mill},                   // its own members and the uplands' hills
		uplands: {hills},
	}
	for j, w := range want {
		if got := tb.Covered(j); !slices.Equal(got, w) {
			t.Errorf("Covered(%d) = %v, want %v", j, got, w)
		}
	}
	if got := tb.Children(crown); !slices.Equal(got, []int32{coast, inland}) {
		t.Errorf("Children(crown) = %v, want [coast inland]", got)
	}
}

// TestCrossing_ExampleA verifies the six rows of the table of spec-polity.md 12.1.
//
// Requirement: FR-POL-010, FR-POL-033
func TestCrossing_ExampleA(t *testing.T) {
	t.Parallel()
	tb := exampleA(t)
	tests := []struct {
		name        string
		from, to    int32
		exit, enter []int32
	}{
		{"hills to harbour", hills, harbour, []int32{uplands, inland}, []int32{coast}},
		{"harbour to hills", harbour, hills, []int32{coast}, []int32{inland, uplands}},
		{"farm to mill, one province", farm, mill, nil, nil},
		{"farm to capital", farm, capital, []int32{inland}, nil},
		{"hills to farm", hills, farm, []int32{uplands}, nil},
		{"capital to mill", capital, mill, nil, []int32{inland}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exit, enter := tb.Crossing(tt.from, tt.to)
			if !slices.Equal(exit, tt.exit) || !slices.Equal(enter, tt.enter) {
				t.Errorf("Crossing(%d, %d) = %v, %v, want %v, %v", tt.from, tt.to, exit, enter, tt.exit, tt.enter)
			}
		})
	}
}

// TestCrossing_ExampleB verifies the six rows of spec-polity.md 12.2, including
// the outside node that is in no jurisdiction and the two roots.
//
// Requirement: FR-POL-010
func TestCrossing_ExampleB(t *testing.T) {
	t.Parallel()
	const east, west = 0, 1
	const ashford, brook, city, delta, far = 0, 1, 2, 3, 4
	tb := exampleB(t)
	tests := []struct {
		name        string
		from, to    int32
		exit, enter []int32
	}{
		{"export at the gate", delta, far, []int32{west}, nil},
		{"import at the gate", far, delta, nil, []int32{west}},
		{"free city into east", city, ashford, nil, []int32{east}},
		{"two roots", ashford, brook, []int32{east}, []int32{west}},
		{"one jurisdiction", brook, delta, nil, nil},
		{"two ungoverned regions", city, far, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exit, enter := tb.Crossing(tt.from, tt.to)
			if !slices.Equal(exit, tt.exit) || !slices.Equal(enter, tt.enter) {
				t.Errorf("Crossing(%d, %d) = %v, %v, want %v, %v", tt.from, tt.to, exit, enter, tt.exit, tt.enter)
			}
		})
	}
}

// TestCrossing_Reversal verifies, over 200 random forests with a fixed seed each,
// that Crossing(b, a) is Crossing(a, b) with the lists swapped and the order
// reversed, that Exit and Enter are disjoint and that a region with itself has an
// empty crossing; it also compares every pair with the climbing model.
//
// Requirement: FR-POL-010
func TestCrossing_Reversal(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 200; seed++ {
		f := randomForest(seed, 1+int(seed%25), 1+int(seed%35), 8)
		tb, err := polity.Build(f.parent, f.member, 8)
		if err != nil {
			t.Fatalf("seed %d: Build = %v, want nil", seed, err)
		}
		for a := range f.member {
			for b := range f.member {
				exit, enter := tb.Crossing(int32(a), int32(b))
				mExit, mEnter := f.modelCrossing(a, b)
				if !slices.Equal(exit, mExit) || !slices.Equal(enter, mEnter) {
					t.Fatalf("seed %d: Crossing(%d, %d) = %v, %v, model %v, %v", seed, a, b, exit, enter, mExit, mEnter)
				}
				rExit, rEnter := tb.Crossing(int32(b), int32(a))
				// Going back, what was left is entered outermost first: the
				// reverse of the innermost-first exit list.
				if !slices.Equal(rEnter, reversed(exit)) || !slices.Equal(rExit, reversed(enter)) {
					t.Fatalf("seed %d: Crossing(%d, %d) = %v, %v is not the reversal of %v, %v", seed, b, a, rExit, rEnter, exit, enter)
				}
				for _, j := range exit {
					if slices.Contains(enter, j) {
						t.Fatalf("seed %d: jurisdiction %d in both lists of Crossing(%d, %d)", seed, j, a, b)
					}
				}
				if a == b && (len(exit) != 0 || len(enter) != 0) {
					t.Fatalf("seed %d: Crossing(%d, %d) = %v, %v, want empty", seed, a, b, exit, enter)
				}
			}
		}
	}
}

func reversed(s []int32) []int32 {
	out := slices.Clone(s)
	slices.Reverse(out)
	return out
}

// TestBuild_Rejects verifies the cycle, self-parent, depth and index rules and
// the exact depth bound: a chain of maxDepth entries builds and one more is
// rejected, the lowest jurisdiction at fault being reported.
//
// Requirement: FR-POL-003, FR-POL-004
func TestBuild_Rejects(t *testing.T) {
	t.Parallel()
	chain := func(n int) []int32 { // n jurisdictions in a line, the last the deepest
		p := make([]int32, n)
		for i := range p {
			p[i] = int32(i - 1)
		}
		return p
	}
	tests := []struct {
		name   string
		parent []int32
		member []int32
		want   error // nil builds
		at     int
	}{
		{"two-cycle", []int32{1, 0}, nil, polity.ErrCycle, 0},
		{"self-parent", []int32{-1, 1}, nil, polity.ErrCycle, 1},
		{"cycle behind a tail", []int32{2, 0, 1, 1}, nil, polity.ErrCycle, 0},
		{"eight entries build", chain(8), []int32{7}, nil, 0},
		{"nine entries are too deep", chain(9), []int32{8}, polity.ErrDepth, 8},
		{"parent out of range", []int32{5}, nil, polity.ErrIndex, 0},
		{"member out of range", []int32{-1}, []int32{-2}, polity.ErrIndex, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tb, err := polity.Build(tt.parent, tt.member, 8)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Build = %v, want nil", err)
				}
				if got := tb.Chain(0); len(got) != 8 {
					t.Errorf("len(Chain(0)) = %d, want 8", len(got))
				}
				return
			}
			var be *polity.BuildError
			if !errors.Is(err, tt.want) || !errors.As(err, &be) || be.Index != tt.at {
				t.Errorf("Build = %v, want %v at %d", err, tt.want, tt.at)
			}
		})
	}
}

// TestBuild_BoundSize builds a forest of the bound size (1 000 jurisdictions, 1 000
// regions, 8 levels) and compares every chain, covered list and a sample of
// crossings with the climbing model.
//
// Requirement: FR-POL-004
func TestBuild_BoundSize(t *testing.T) {
	t.Parallel()
	f := randomForest(7, 1000, 1000, 8)
	tb, err := polity.Build(f.parent, f.member, 8)
	if err != nil {
		t.Fatalf("Build(bound size) = %v, want nil", err)
	}
	deepest := 0
	for r := range f.member {
		got, want := tb.Chain(int32(r)), f.modelChain(r)
		if !slices.Equal(got, want) {
			t.Fatalf("Chain(%d) = %v, model %v", r, got, want)
		}
		deepest = max(deepest, len(want))
	}
	if deepest != 8 {
		t.Errorf("deepest chain = %d entries, want 8 (the seed was chosen to reach the bound)", deepest)
	}
	for j := range f.parent {
		if got, want := tb.Covered(int32(j)), f.modelCovered(int32(j)); !slices.Equal(got, want) {
			t.Fatalf("Covered(%d) = %v, model %v", j, got, want)
		}
	}
	for a := 0; a < 1000; a += 37 {
		for b := 0; b < 1000; b += 41 {
			exit, enter := tb.Crossing(int32(a), int32(b))
			mExit, mEnter := f.modelCrossing(a, b)
			if !slices.Equal(exit, mExit) || !slices.Equal(enter, mEnter) {
				t.Fatalf("Crossing(%d, %d) = %v, %v, model %v, %v", a, b, exit, enter, mExit, mEnter)
			}
		}
	}
	// One level more is refused: the same forest with maxDepth 7.
	if _, err := polity.Build(f.parent, f.member, 7); !errors.Is(err, polity.ErrDepth) {
		t.Errorf("Build(bound size, maxDepth 7) = %v, want ErrDepth", err)
	}
}

// TestEdgeBetween verifies the edge-pair test on example A: the edges harbour>mill
// (from the coast into the inland province) and mill>harbour, and the nested and
// equal pairs that have none (spec-polity.md 12.1, 4.2.3).
//
// Requirement: FR-POL-011
func TestEdgeBetween(t *testing.T) {
	t.Parallel()
	tb := exampleA(t)
	tests := []struct {
		name     string
		from, to int32
		a, b     int32
		want     bool
	}{
		{"coast to inland, harbour>mill", harbour, mill, coast, inland, true},
		{"inland to coast, mill>harbour", mill, harbour, inland, coast, true},
		{"wrong direction", harbour, mill, inland, coast, false},
		{"coast to crown are nested", harbour, mill, coast, crown, false},
		{"a jurisdiction with itself", harbour, mill, coast, coast, false},
		{"unknown jurisdiction", harbour, mill, -1, inland, false},
		{"unknown region", 99, mill, coast, inland, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tb.EdgeBetween(tt.from, tt.to, tt.a, tt.b); got != tt.want {
				t.Errorf("EdgeBetween(%d, %d, %d, %d) = %v, want %v", tt.from, tt.to, tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestQueries_OutOfRange verifies that every query answers with the zero result
// and does not panic for -1, the count and a huge ID.
//
// Requirement: FR-POL-012
func TestQueries_OutOfRange(t *testing.T) {
	t.Parallel()
	tb := exampleA(t)
	for _, id := range []int32{-1, -2, 4, 5, math.MaxInt32, math.MinInt32} {
		if _, ok := tb.Parent(id); ok && id != coast && id != inland && id != uplands {
			t.Errorf("Parent(%d) ok, want false", id)
		}
		if id < 0 || id > 3 {
			if _, ok := tb.Depth(id); ok {
				t.Errorf("Depth(%d) ok, want false", id)
			}
			if got := tb.Children(id); got != nil {
				t.Errorf("Children(%d) = %v, want nil", id, got)
			}
			if got := tb.Covered(id); got != nil {
				t.Errorf("Covered(%d) = %v, want nil", id, got)
			}
			if tb.Covers(id, 0) {
				t.Errorf("Covers(%d, 0) = true, want false", id)
			}
		}
		if id < 0 || id > 4 {
			if _, ok := tb.Of(id); ok {
				t.Errorf("Of(%d) ok, want false", id)
			}
			if got := tb.Chain(id); got != nil {
				t.Errorf("Chain(%d) = %v, want nil", id, got)
			}
			if got := tb.Common(id, 0); got != -1 {
				t.Errorf("Common(%d, 0) = %d, want -1", id, got)
			}
			if e, n := tb.Crossing(id, 0); e != nil || n != nil {
				t.Errorf("Crossing(%d, 0) = %v, %v, want nil, nil", id, e, n)
			}
			if tb.Covers(0, id) {
				t.Errorf("Covers(0, %d) = true, want false", id)
			}
		}
	}
	var zero polity.Tables
	if zero.Len() != 0 || zero.Regions() != 0 || zero.Chain(0) != nil || zero.Covered(0) != nil || zero.Common(0, 1) != -1 {
		t.Error("the zero Tables must answer none to every query")
	}
}

// rowsA are the three tables of the worked combining example (12.1), ascending by
// jurisdiction.
var (
	levy  = []polity.Row{{Jurisdiction: crown, Value: 200}, {Jurisdiction: inland, Value: 100}, {Jurisdiction: uplands, Value: 50}}
	cap80 = []polity.Row{{Jurisdiction: crown, Value: 90}, {Jurisdiction: inland, Value: 80}}
	scale = []polity.Row{{Jurisdiction: crown, Value: 9000}, {Jurisdiction: inland, Value: 8000}}
)

// TestCombine_ExampleA verifies the three rows of the combining table of
// spec-polity.md 12.1: added, innermost and scaled, over the five regions in ID
// order.
//
// Requirement: FR-POL-023
func TestCombine_ExampleA(t *testing.T) {
	t.Parallel()
	tb := exampleA(t)
	wantAdded := []int64{200, 300, 200, 350, 300}
	wantInner := []int64{90, 80, 90, 80, 80}
	// Scaled over a base of 1 000: farm is 1 000 * 9 000 * 8 000 / 10 000^2 = 720.
	wantScaled := []int64{900, 720, 900, 720, 720}
	for r := int32(0); r < 5; r++ {
		chain := tb.Chain(r)
		if got, ok := polity.Added(levy, chain); !ok || got != wantAdded[r] {
			t.Errorf("Added(levy, chain %d) = %d, %v, want %d, true", r, got, ok, wantAdded[r])
		}
		if got, ok := polity.Innermost(cap80, chain); !ok || got != wantInner[r] {
			t.Errorf("Innermost(cap, chain %d) = %d, %v, want %d, true", r, got, ok, wantInner[r])
		}
		factors := polity.Factors(nil, scale, chain)
		if got, ok := polity.Scaled(1000, factors); !ok || got != wantScaled[r] {
			t.Errorf("Scaled(1000, %v) = %d, %v, want %d, true", factors, got, ok, wantScaled[r])
		}
	}
	// An ungoverned region (an empty chain) gets the default in all three modes.
	if v, ok := polity.Innermost(cap80, nil); ok || v != 0 {
		t.Errorf("Innermost(cap, empty chain) = %d, %v, want 0, false", v, ok)
	}
	if v, ok := polity.Added(levy, nil); !ok || v != 0 {
		t.Errorf("Added(levy, empty chain) = %d, %v, want 0, true", v, ok)
	}
	if v, ok := polity.Scaled(1000, nil); !ok || v != 1000 {
		t.Errorf("Scaled(1000, no factor) = %d, %v, want 1000, true", v, ok)
	}
}

// TestScaled_SingleRounding verifies that the product is rounded once and not
// order dependent, and the edge cases of the arithmetic.
//
// Requirement: FR-POL-023
//
// Guard: two rounding steps give a different answer and depend on the order of the
// chain; rounding each step is the likely mistake of a reader that keeps its own
// copy.
func TestScaled_SingleRounding(t *testing.T) {
	t.Parallel()
	// 5 * 3 000 * 3 000 / 10 000^2 = 0.45, which rounds to 0. Rounding after each
	// factor gives 5 * 0.3 = 1.5 -> 2, then 2 * 0.3 = 0.6 -> 1.
	if got, ok := polity.Scaled(5, []int64{3000, 3000}); !ok || got != 0 {
		t.Errorf("Scaled(5, [3000 3000]) = %d, %v, want 0, true", got, ok)
	}
	tests := []struct {
		name    string
		base    int64
		factors []int64
		want    int64
		ok      bool
	}{
		{"tie goes to the even quotient, down", 5, []int64{5000}, 2, true},              // 2.5 -> 2
		{"tie goes to the even quotient, up", 7, []int64{5000}, 4, true},                // 3.5 -> 4
		{"just above a tie", 1, []int64{5001}, 1, true},                                 // 0.5001 -> 1
		{"zero factor", 123, []int64{9000, 0}, 0, true},                                 //
		{"negative base", -1, []int64{10000}, 0, false},                                 //
		{"negative factor", 1, []int64{-1}, 0, false},                                   //
		{"too many factors", 1, make([]int64, polity.MaxFactors+1), 0, false},           //
		{"nine factors of one", 1, slicesOf(9, 10_000), 1, true},                        // 10^36 fits in 128 bits
		{"nine factors of one, base too big", 1_000_000, slicesOf(9, 10_000), 0, false}, // 10^42 does not
		{"result above int64", math.MaxInt64, []int64{20_000}, 0, false},                // 2 * 2^63 - 2
		{"numerator above 128 bits", math.MaxInt64, slicesOf(9, math.MaxInt64), 0, false},
		{"largest result", math.MaxInt64, []int64{10_000}, math.MaxInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := polity.Scaled(tt.base, tt.factors)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Scaled(%d, %v) = %d, %v, want %d, %v", tt.base, tt.factors, got, ok, tt.want, tt.ok)
			}
		})
	}
	// The order of the factors does not change the result.
	f := []int64{9000, 8123, 7777, 10_500}
	want, _ := polity.Scaled(987_654_321, f)
	for _, p := range [][]int{{3, 2, 1, 0}, {1, 0, 3, 2}, {2, 0, 1, 3}} {
		var g []int64
		for _, i := range p {
			g = append(g, f[i])
		}
		if got, _ := polity.Scaled(987_654_321, g); got != want {
			t.Errorf("Scaled order %v = %d, want %d (the order of the chain must not matter)", p, got, want)
		}
	}
}

func slicesOf(n int, v int64) []int64 {
	s := make([]int64, n)
	for i := range s {
		s[i] = v
	}
	return s
}

// TestAdded_Overflow verifies that a sum that leaves int64 is reported and not
// wrapped.
//
// Requirement: FR-POL-023
func TestAdded_Overflow(t *testing.T) {
	t.Parallel()
	rows := []polity.Row{{Jurisdiction: 0, Value: math.MaxInt64}, {Jurisdiction: 1, Value: 1}}
	if got, ok := polity.Added(rows, []int32{1, 0}); ok || got != 0 {
		t.Errorf("Added overflow = %d, %v, want 0, false", got, ok)
	}
}

// FuzzBuild verifies that Build never panics on random parent and member arrays,
// that a cyclic input errors with ErrCycle at a jurisdiction that is on a cycle,
// and that an accepted forest agrees with the climbing model.
func FuzzBuild(f *testing.F) {
	f.Add([]byte{255, 0, 0}, []byte{0, 1})
	f.Add([]byte{1, 0}, []byte{0})
	f.Add([]byte{255, 0, 1, 2, 3, 4, 5, 6, 7, 8}, []byte{9, 0, 255})
	f.Fuzz(func(t *testing.T, p, m []byte) {
		if len(p) > 64 || len(m) > 64 {
			return
		}
		// A byte of 255 is "none"; any other byte is an index, which may be out of range.
		conv := func(b []byte) []int32 {
			out := make([]int32, len(b))
			for i, v := range b {
				out[i] = int32(v)
				if v == 255 {
					out[i] = -1
				}
			}
			return out
		}
		parent, member := conv(p), conv(m)
		tb, err := polity.Build(parent, member, 8)
		if err != nil {
			if !errors.Is(err, polity.ErrCycle) && !errors.Is(err, polity.ErrDepth) && !errors.Is(err, polity.ErrIndex) {
				t.Fatalf("Build(%v, %v) = %v, want a known cause", parent, member, err)
			}
			return
		}
		fo := forest{parent: parent, member: member}
		for r := range member {
			if got, want := tb.Chain(int32(r)), fo.modelChain(r); !slices.Equal(got, want) {
				t.Fatalf("Chain(%d) = %v, model %v for %v %v", r, got, want, parent, member)
			}
		}
	})
}

// TestScaled_AgainstBig compares Scaled with an exact big.Int computation rounded
// half to even, over 5 000 seeded cases of up to nine factors, and checks that a
// case the exact value does not fit in int64 is refused.
//
// Requirement: FR-POL-023
//
// Guard: the 128-bit product and quotient are hand written; a carry or word-order
// slip shows only with large operands, which the worked examples do not have.
func TestScaled_AgainstBig(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(11, 22))
	for i := 0; i < 5000; i++ {
		base := rng.Int64N(1 << uint(1+rng.IntN(62)))
		factors := make([]int64, rng.IntN(polity.MaxFactors+1))
		for j := range factors {
			factors[j] = rng.Int64N(30_000)
		}
		num := big.NewInt(base)
		den := big.NewInt(1)
		for _, f := range factors {
			num.Mul(num, big.NewInt(f))
			den.Mul(den, big.NewInt(polity.WholeBP))
		}
		q, r := new(big.Int).QuoRem(num, den, new(big.Int))
		twice := new(big.Int).Lsh(r, 1)
		if c := twice.Cmp(den); c > 0 || (c == 0 && q.Bit(0) == 1) {
			q.Add(q, big.NewInt(1))
		}
		got, ok := polity.Scaled(base, factors)
		fits := q.IsInt64() && num.BitLen() <= 128
		switch {
		case fits && (!ok || got != q.Int64()):
			t.Fatalf("case %d: Scaled(%d, %v) = %d, %v, want %v, true", i, base, factors, got, ok, q)
		case !fits && ok:
			t.Fatalf("case %d: Scaled(%d, %v) = %d, true, want a refusal (exact %v)", i, base, factors, got, q)
		}
	}
}
