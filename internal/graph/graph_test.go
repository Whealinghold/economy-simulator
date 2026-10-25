package graph

import (
	"errors"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// splitmix is a local SplitMix64, so the tests draw no global randomness.
type splitmix uint64

func (s *splitmix) next() uint64 {
	*s += 0x9e3779b97f4a7c15
	z := uint64(*s)
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	return z ^ z>>31
}

func (s *splitmix) below(n int) int { return int(s.next() % uint64(n)) }

// matrix is a graph as the test writes it: w[i][j] is the length of the edge
// i to j, 0 for none.
type matrix [][]int64

func newMatrix(n int) matrix {
	m := make(matrix, n)
	for i := range m {
		m[i] = make([]int64, n)
	}
	return m
}

// edges lists the matrix in ascending (From, To), the order Build requires.
func (m matrix) edges() []Edge {
	var out []Edge
	for i := range m {
		for j := range m[i] {
			if m[i][j] > 0 {
				out = append(out, Edge{From: i, To: j, Length: m[i][j]})
			}
		}
	}
	return out
}

func (m matrix) build(t *testing.T) *Tables {
	t.Helper()
	tb, err := Build(len(m), m.edges())
	if err != nil {
		t.Fatalf("Build(%d, %v) error = %v", len(m), m.edges(), err)
	}
	return tb
}

// refBest is the definition of 4.3.1 by brute force: it enumerates every
// simple path from a to b and keeps the shortest, and among equals the
// lexicographically smallest as a sequence of region numbers. A shortest walk
// repeats no region because every length is at least 1.
func refBest(m matrix, a, b int) (int64, []int, bool) {
	var best []int
	var bestLen int64 = -1
	seen := make([]bool, len(m))
	path := []int{a}
	var walk func(at int, length int64)
	walk = func(at int, length int64) {
		if at == b {
			if bestLen < 0 || length < bestLen || (length == bestLen && slices.Compare(path, best) < 0) {
				bestLen, best = length, slices.Clone(path)
			}
			return
		}
		seen[at] = true
		for to, w := range m[at] {
			if w > 0 && !seen[to] {
				path = append(path, to)
				walk(to, length+w)
				path = path[:len(path)-1]
			}
		}
		seen[at] = false
	}
	walk(a, 0)
	return bestLen, best, bestLen >= 0
}

// refComps numbers the strongly connected components by the transitive closure
// of the edges, in ascending order of the smallest member.
func refComps(m matrix) ([]int, int) {
	n := len(m)
	reach := make([][]bool, n)
	for i := range reach {
		reach[i] = make([]bool, n)
		reach[i][i] = true
		for j, w := range m[i] {
			reach[i][j] = reach[i][j] || w > 0
		}
	}
	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				reach[i][j] = reach[i][j] || (reach[i][k] && reach[k][j])
			}
		}
	}
	comp := make([]int, n)
	for i := range comp {
		comp[i] = -1
	}
	count := 0
	for i := 0; i < n; i++ {
		if comp[i] >= 0 {
			continue
		}
		for j := i; j < n; j++ {
			if reach[i][j] && reach[j][i] {
				comp[j] = count
			}
		}
		count++
	}
	return comp, count
}

// checkAgainstRef compares every query of the tables with the brute force and
// reports the first difference.
func checkAgainstRef(t *testing.T, label string, m matrix) {
	t.Helper()
	tb := m.build(t)
	n := len(m)
	for a := 0; a < n; a++ {
		for b := 0; b < n; b++ {
			wantD, wantP, wantOK := refBest(m, a, b)
			gotD, gotOK := tb.Dist(a, b)
			if gotOK != wantOK || (wantOK && gotD != wantD) {
				t.Fatalf("%s: Dist(%d, %d) = %d, %v, want %d, %v (edges %v)", label, a, b, gotD, gotOK, wantD, wantOK, m.edges())
			}
			gotP, pOK := tb.AppendPath(nil, a, b)
			if pOK != wantOK || !slices.Equal(gotP, wantP) {
				t.Fatalf("%s: AppendPath(%d, %d) = %v, %v, want %v, %v (edges %v)", label, a, b, gotP, pOK, wantP, wantOK, m.edges())
			}
			hop, hopOK := tb.Next(a, b)
			if wantHop := wantOK && a != b; hopOK != wantHop || (wantHop && hop != wantP[1]) {
				t.Fatalf("%s: Next(%d, %d) = %d, %v, want first hop of %v (edges %v)", label, a, b, hop, hopOK, wantP, m.edges())
			}
		}
	}
	for a := 0; a < n; a++ {
		for _, limit := range []int64{0, 2, 4, 1 << 40} {
			var want []Hop
			for b := 0; b < n; b++ {
				if d, _, ok := refBest(m, a, b); ok && b != a && d <= limit {
					want = append(want, Hop{Region: b, Distance: d})
				}
			}
			slices.SortFunc(want, func(x, y Hop) int {
				if x.Distance != y.Distance {
					return int(x.Distance - y.Distance)
				}
				return x.Region - y.Region
			})
			if got := tb.AppendWithin(nil, a, limit); !slices.Equal(got, want) {
				t.Fatalf("%s: AppendWithin(nil, %d, %d) = %v, want %v (edges %v)", label, a, limit, got, want, m.edges())
			}
		}
	}
	wantC, wantN := refComps(m)
	if tb.Comps() != wantN {
		t.Fatalf("%s: Comps() = %d, want %d (edges %v)", label, tb.Comps(), wantN, m.edges())
	}
	for r, want := range wantC {
		if got, ok := tb.Comp(r); !ok || got != want {
			t.Fatalf("%s: Comp(%d) = %d, %v, want %d (edges %v)", label, r, got, ok, want, m.edges())
		}
	}
}

// TestBuild_MatchesEnumeration verifies that the distance, the next hop, the
// path and the components equal the definition by enumeration of every simple
// path: exhaustively on every directed graph of one to three regions with each
// edge absent, 1 or 2 long, and on 3 000 seeded graphs of four to seven
// regions with lengths 1 to 6 and a density of about 45%, so that ties,
// one-way edges and unreachable pairs are common.
//
// Requirement: FR-REG-016, FR-REG-018, FR-REG-019, FR-REG-020, FR-REG-027
func TestBuild_MatchesEnumeration(t *testing.T) {
	t.Parallel()
	t.Run("every graph of up to three regions", func(t *testing.T) {
		t.Parallel()
		for n := 1; n <= 3; n++ {
			pairs := n * (n - 1)
			total := 1
			for i := 0; i < pairs; i++ {
				total *= 3 // absent, 1, 2
			}
			for code := 0; code < total; code++ {
				m, c := newMatrix(n), code
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						if i != j {
							m[i][j] = int64(c % 3)
							c /= 3
						}
					}
				}
				checkAgainstRef(t, "n="+strconv.Itoa(n)+" code="+strconv.Itoa(code), m)
			}
		}
	})
	t.Run("seeded graphs of four to seven regions", func(t *testing.T) {
		t.Parallel()
		rng := splitmix(1)
		for g := 0; g < 3000; g++ {
			n := 4 + rng.below(4)
			m := newMatrix(n)
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					if i != j && rng.below(100) < 45 {
						m[i][j] = int64(1 + rng.below(6))
					}
				}
			}
			checkAgainstRef(t, "graph "+strconv.Itoa(g), m)
		}
	})
}

// TestBuild_TieBreakDependsOnIDsOnly verifies the tie of example C of
// spec-regions.md 12.3: the 160 track and the 80 + 80 detour through the town
// are equally short, the track wins while the other hamlet sorts before the
// town, and relabelling so that the town sorts first changes the path.
// Permuting the order the edges were listed in does not, because the caller
// sorts them before Build.
//
// Requirement: FR-REG-021
func TestBuild_TieBreakDependsOnIDsOnly(t *testing.T) {
	t.Parallel()
	// hamlet_east 0, hamlet_west 1, town 2.
	m := newMatrix(3)
	m[2][0], m[0][2], m[2][1], m[1][2], m[0][1], m[1][0] = 80, 80, 80, 80, 160, 160
	if got, _ := m.build(t).AppendPath(nil, 0, 1); !slices.Equal(got, []int{0, 1}) {
		t.Errorf("AppendPath(hamlet_east, hamlet_west) = %v, want the direct track [0 1]", got)
	}
	// a_town 0, hamlet_east 1, hamlet_west 2: the same graph, the town renamed.
	r := newMatrix(3)
	r[0][1], r[1][0], r[0][2], r[2][0], r[1][2], r[2][1] = 80, 80, 80, 80, 160, 160
	if got, _ := r.build(t).AppendPath(nil, 1, 2); !slices.Equal(got, []int{1, 0, 2}) {
		t.Errorf("AppendPath(hamlet_east, hamlet_west) after the rename = %v, want through the town [1 0 2]", got)
	}

	want := m.build(t)
	rng := splitmix(7)
	for i := 0; i < 20; i++ {
		list := m.edges()
		for j := len(list) - 1; j > 0; j-- {
			k := rng.below(j + 1)
			list[j], list[k] = list[k], list[j]
		}
		slices.SortFunc(list, func(a, b Edge) int {
			if a.From != b.From {
				return a.From - b.From
			}
			return a.To - b.To
		})
		got, err := Build(3, list)
		if err != nil || !slices.Equal(got.dist, want.dist) || !slices.Equal(got.next, want.next) {
			t.Fatalf("Build after reordering the list (%v) = %v, %v, want the same tables", list, got, err)
		}
	}
}

// TestBuild_RejectsBrokenInput verifies the contract of Build: an end out of
// range, a self edge, a length below 1, a duplicate and an unsorted list are
// ErrInvalid, and nothing panics.
//
// Guard: the root sorts and validates before it calls, but a caller that
// passed an unsorted list would otherwise get tables that depend on the order
// of the list, which FR-REG-020 forbids.
func TestBuild_RejectsBrokenInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		n    int
		open []Edge
	}{
		{"negative count", -1, nil},
		{"count beyond int32", math.MaxInt32 + 1, nil},
		{"count beyond the bound", MaxRegions + 1, nil},
		{"more edges than ordered pairs", 2, []Edge{{0, 1, 1}, {1, 0, 1}, {1, 0, 1}}},
		{"from far out of range", 2, []Edge{{5, 0, 1}}},
		{"to far out of range", 2, []Edge{{0, 5, 1}}},
		{"from out of range", 2, []Edge{{2, 0, 1}}},
		{"to out of range", 2, []Edge{{0, 2, 1}}},
		{"negative from", 2, []Edge{{-1, 0, 1}}},
		{"negative to", 2, []Edge{{0, -1, 1}}},
		{"self edge", 2, []Edge{{1, 1, 1}}},
		{"zero length", 2, []Edge{{0, 1, 0}}},
		{"negative length", 2, []Edge{{0, 1, -3}}},
		{"duplicate", 2, []Edge{{0, 1, 1}, {0, 1, 2}}},
		{"unsorted by to", 3, []Edge{{0, 2, 1}, {0, 1, 1}}},
		{"unsorted by from", 3, []Edge{{1, 0, 1}, {0, 1, 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := Build(tt.n, tt.open); !errors.Is(err, ErrInvalid) || got != nil {
				t.Errorf("Build(%d, %v) = %v, %v, want nil and ErrInvalid", tt.n, tt.open, got, err)
			}
		})
	}
	if got, err := Build(0, nil); err != nil || got.Comps() != 0 {
		t.Errorf("Build(0, nil) = %v, %v, want an empty graph", got, err)
	}
}

// TestBuild_Components verifies the numbering of the components: a graph with
// no edge has one per region, one region has one, a one-way pair has two, a
// pair both ways has one, and the numbers ascend with the smallest member.
//
// Requirement: FR-REG-015, FR-REG-027
func TestBuild_Components(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		n     int
		open  []Edge
		comps int
		of    []int
	}{
		{"one region", 1, nil, 1, []int{0}},
		{"no edges", 4, nil, 4, []int{0, 1, 2, 3}},
		{"one-way pair", 2, []Edge{{0, 1, 1}}, 2, []int{0, 1}},
		{"two-way pair", 2, []Edge{{0, 1, 1}, {1, 0, 5}}, 1, []int{0, 0}},
		// {0, 3} and {1, 2} are cycles; {4} is alone; the number follows the smallest member.
		{"numbering by smallest member", 5,
			[]Edge{{0, 3, 1}, {1, 2, 1}, {2, 1, 1}, {3, 0, 1}, {3, 4, 1}}, 3, []int{0, 1, 1, 0, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tb, err := Build(tt.n, tt.open)
			if err != nil {
				t.Fatalf("Build(%d, %v) error = %v", tt.n, tt.open, err)
			}
			if tb.Comps() != tt.comps {
				t.Errorf("Comps() = %d, want %d", tb.Comps(), tt.comps)
			}
			for r, want := range tt.of {
				if got, ok := tb.Comp(r); !ok || got != want {
					t.Errorf("Comp(%d) = %d, %v, want %d", r, got, ok, want)
				}
			}
		})
	}
}

// table reads every Dist of the tables as a row-major slice, -1 for unreachable.
func table(tb *Tables) []int64 {
	var out []int64
	for a := 0; a < tb.n; a++ {
		for b := 0; b < tb.n; b++ {
			d, ok := tb.Dist(a, b)
			if !ok {
				d = -1
			}
			out = append(out, d)
		}
	}
	return out
}

// TestBuild_WorkedExamples verifies the tables of spec-regions.md 12.1 (A,
// with the closure of middle > lower and with the factor 15 000 on it), 12.2
// (B, with and without the edge orbital > surface) and 12.3 (C).
//
// Requirement: FR-REG-016, FR-REG-017, FR-REG-022, FR-REG-049
func TestBuild_WorkedExamples(t *testing.T) {
	t.Parallel()
	// A: lower 0, middle 1, upper 2.
	a := newMatrix(3)
	a[2][1], a[1][2], a[1][0], a[0][1], a[2][0], a[0][2] = 40, 70, 30, 55, 90, 120
	closed := newMatrix(3)
	copy(closed, a)
	closed[1] = []int64{0, 0, 70} // middle > lower closed
	factor := newMatrix(3)
	copy(factor, a)
	factor[1] = []int64{45, 0, 70} // 30 at 15 000 is 45 (TestEffective_Rounding)
	// B: belt 0, orbital 1, surface 2.
	b := newMatrix(3)
	b[2][1], b[1][2], b[1][0], b[0][1] = 9400, 2000, 3000, 3000
	oneWay := newMatrix(3)
	oneWay[2][1], oneWay[1][0], oneWay[0][1] = 9400, 3000, 3000 // orbital > surface absent
	// C: hamlet_east 0, hamlet_west 1, town 2.
	c := newMatrix(3)
	c[2][0], c[0][2], c[2][1], c[1][2], c[0][1], c[1][0] = 80, 80, 80, 80, 160, 160

	tests := []struct {
		name  string
		m     matrix
		want  []int64
		comps int
	}{
		{"A", a, []int64{0, 55, 120, 30, 0, 70, 70, 40, 0}, 1},
		// Upper reaches lower by the 90 track; middle reaches lower by upper, 70 + 90.
		{"A with middle > lower closed", closed, []int64{0, 55, 120, 160, 0, 70, 90, 40, 0}, 1},
		// Upper > middle > lower is 40 + 45 = 85, still shorter than the track.
		{"A with middle > lower at 15 000", factor, []int64{0, 55, 120, 45, 0, 70, 85, 40, 0}, 1},
		{"B", b, []int64{0, 3000, 5000, 3000, 0, 2000, 12400, 9400, 0}, 1},
		{"B without orbital > surface", oneWay, []int64{0, 3000, -1, 3000, 0, -1, 12400, 9400, 0}, 2},
		{"C", c, []int64{0, 160, 80, 160, 0, 80, 80, 80, 0}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tb := tt.m.build(t)
			if got := table(tb); !slices.Equal(got, tt.want) {
				t.Errorf("distance table = %v, want %v", got, tt.want)
			}
			if tb.Comps() != tt.comps {
				t.Errorf("Comps() = %d, want %d", tb.Comps(), tt.comps)
			}
		})
	}

	if got, _ := a.build(t).AppendPath(nil, 2, 0); !slices.Equal(got, []int{2, 1, 0}) {
		t.Errorf("A: path upper to lower = %v, want through middle [2 1 0]", got)
	}
	if got, _ := closed.build(t).AppendPath(nil, 2, 0); !slices.Equal(got, []int{2, 0}) {
		t.Errorf("A closed: path upper to lower = %v, want the track [2 0]", got)
	}
	if got, _ := b.build(t).AppendPath(nil, 2, 0); !slices.Equal(got, []int{2, 1, 0}) {
		t.Errorf("B: path surface to belt = %v, want [2 1 0]", got)
	}
	// {belt, orbital} is component 0 and {surface} component 1.
	for r, want := range []int{0, 0, 1} {
		if got, _ := oneWay.build(t).Comp(r); got != want {
			t.Errorf("B without orbital > surface: Comp(%d) = %d, want %d", r, got, want)
		}
	}
}

// TestEffective_Rounding verifies the effective length of an edge: the
// examples of spec-regions.md 7.2.2, the floor of 1, and the report of a
// product beyond int64.
//
// Requirement: FR-REG-049, FR-REG-029
func TestEffective_Rounding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		base, factor int64
		want         int64
		ok           bool
	}{
		{"30 at 15 000 is 45", 30, 15000, 45, true},
		{"30 at 10 000 is 30", 30, 10000, 30, true},
		{"7 at 10 001 rounds up to 8", 7, 10001, 8, true},
		{"1 at 1 is 1", 1, 1, 1, true},
		{"a shortening never reaches 0", 3, 1, 1, true},
		{"a zero base is outside the contract and gives 1", 0, 10000, 1, true},
		{"the largest edge", 1_000_000_000, 100_000, 10_000_000_000, true},
		{"a quotient beyond int64 is reported", math.MaxInt64, 100_000, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := Effective(tt.base, tt.factor); got != tt.want || ok != tt.ok {
				t.Errorf("Effective(%d, %d) = %d, %v, want %d, %v", tt.base, tt.factor, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestBuild_LargestCaseDoesNotOverflow verifies the bound of
// spec-regions.md 4.2: a chain of 999 edges of the largest effective length
// (MaxDistance 10^9 at MaxEdgeFactorBP 100 000, so 10^10) is 999 * 10^10
// long, below 10^13, and builds without error.
//
// Requirement: FR-REG-029
func TestBuild_LargestCaseDoesNotOverflow(t *testing.T) {
	t.Parallel()
	const n = 1000
	length, ok := Effective(1_000_000_000, 100_000)
	if !ok {
		t.Fatalf("Effective of the largest edge reported overflow")
	}
	open := make([]Edge, 0, n-1)
	for i := 0; i+1 < n; i++ {
		open = append(open, Edge{From: i, To: i + 1, Length: length})
	}
	tb, err := Build(n, open)
	if err != nil {
		t.Fatalf("Build(%d regions, chain of %d edges of %d) error = %v", n, len(open), length, err)
	}
	const want = 999 * 10_000_000_000
	if d, ok := tb.Dist(0, n-1); !ok || d != want || d >= 10_000_000_000_000 {
		t.Errorf("Dist(0, %d) = %d, %v, want %d, below 10^13", n-1, d, ok, want)
	}
	if _, ok := tb.Dist(n-1, 0); ok {
		t.Errorf("Dist(%d, 0) is defined on a one-way chain", n-1)
	}
}

// TestBuild_OverflowIsReported verifies that a path sum beyond int64 is
// ErrOverflow and not a wrapped distance.
//
// Guard: the bounds make overflow unreachable, so this error path is checked
// by no other test; without it a wrap-around in a future change would pass.
func TestBuild_OverflowIsReported(t *testing.T) {
	t.Parallel()
	half := int64(math.MaxInt64/2 + 1)
	tests := []struct {
		name string
		open []Edge
	}{
		{"two edges whose sum overflows", []Edge{{0, 1, half}, {1, 2, half}}},
		{"one edge at the limit and one more", []Edge{{0, 1, math.MaxInt64}, {1, 2, 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := Build(3, tt.open); !errors.Is(err, ErrOverflow) || got != nil {
				t.Errorf("Build(3, %v) = %v, %v, want nil and ErrOverflow", tt.open, got, err)
			}
		})
	}
	if tb, err := Build(2, []Edge{{0, 1, math.MaxInt64}}); err != nil {
		t.Errorf("Build with one edge of MaxInt64 error = %v, want none: no sum overflows", err)
	} else if d, _ := tb.Dist(0, 1); d != math.MaxInt64 {
		t.Errorf("Dist(0, 1) = %d, want MaxInt64", d)
	}
}

// TestWithin_Order verifies AppendWithin: every region other than the origin
// at most limit away, in ascending (distance, region), nothing for a negative
// limit, the origin itself never, and the appended part only (dst keeps its
// start).
//
// Requirement: FR-REG-024
func TestWithin_Order(t *testing.T) {
	t.Parallel()
	// Region 0 reaches 3 at 5, 1 at 5 (tie, region 1 first), 2 at 7 and 4 at 9; 5 is unreachable.
	m := newMatrix(6)
	m[0][3], m[0][1], m[1][2], m[3][4], m[4][0] = 5, 5, 2, 4, 1
	tb := m.build(t)
	tests := []struct {
		name  string
		limit int64
		want  []Hop
	}{
		{"all reachable", 100, []Hop{{1, 5}, {3, 5}, {2, 7}, {4, 9}}},
		{"cut by the limit", 7, []Hop{{1, 5}, {3, 5}, {2, 7}}},
		{"the limit is inclusive", 5, []Hop{{1, 5}, {3, 5}}},
		{"below the nearest", 4, nil},
		{"zero", 0, nil},
		{"negative", -1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tb.AppendWithin(nil, 0, tt.limit); !slices.Equal(got, tt.want) {
				t.Errorf("AppendWithin(nil, 0, %d) = %v, want %v", tt.limit, got, tt.want)
			}
		})
	}
	prefix := []Hop{{99, 99}}
	if got := tb.AppendWithin(prefix, 0, 7); !slices.Equal(got, []Hop{{99, 99}, {1, 5}, {3, 5}, {2, 7}}) {
		t.Errorf("AppendWithin(prefix, 0, 7) = %v, want the prefix kept and the rest sorted", got)
	}
}

// TestQueries_OutOfRange verifies that a region number outside [0, n) in any
// query gives the zero result and false and never panics, and that Dist(a, a)
// is (0, true).
//
// Requirement: FR-REG-025, FR-REG-016
func TestQueries_OutOfRange(t *testing.T) {
	t.Parallel()
	// A cycle 0 > 1 > 2 > 0 with an extra chord, so that every pair is reachable
	// and an index that wrapped into the next row would read a defined value.
	m := newMatrix(3)
	m[0][1], m[1][2], m[2][0], m[0][2] = 1, 1, 1, 5
	tb := m.build(t)
	for _, bad := range []int{-1, 3, math.MaxInt32, math.MinInt32} {
		for _, other := range []int{0, 1, bad} {
			if d, ok := tb.Dist(bad, other); d != 0 || ok {
				t.Errorf("Dist(%d, %d) = %d, %v, want 0, false", bad, other, d, ok)
			}
			if d, ok := tb.Dist(other, bad); d != 0 || ok {
				t.Errorf("Dist(%d, %d) = %d, %v, want 0, false", other, bad, d, ok)
			}
			if v, ok := tb.Next(bad, other); v != 0 || ok {
				t.Errorf("Next(%d, %d) = %d, %v, want 0, false", bad, other, v, ok)
			}
			if v, ok := tb.Next(other, bad); v != 0 || ok {
				t.Errorf("Next(%d, %d) = %d, %v, want 0, false", other, bad, v, ok)
			}
			if p, ok := tb.AppendPath(nil, bad, other); p != nil || ok {
				t.Errorf("AppendPath(nil, %d, %d) = %v, %v, want nil, false", bad, other, p, ok)
			}
			if p, ok := tb.AppendPath(nil, other, bad); p != nil || ok {
				t.Errorf("AppendPath(nil, %d, %d) = %v, %v, want nil, false", other, bad, p, ok)
			}
		}
		if c, ok := tb.Comp(bad); c != 0 || ok {
			t.Errorf("Comp(%d) = %d, %v, want 0, false", bad, c, ok)
		}
		if got := tb.AppendWithin(nil, bad, 100); got != nil {
			t.Errorf("AppendWithin(nil, %d, 100) = %v, want nothing", bad, got)
		}
	}
	for r := 0; r < 3; r++ {
		if d, ok := tb.Dist(r, r); d != 0 || !ok {
			t.Errorf("Dist(%d, %d) = %d, %v, want 0, true", r, r, d, ok)
		}
		if v, ok := tb.Next(r, r); v != 0 || ok {
			t.Errorf("Next(%d, %d) = %d, %v, want 0, false: a region has no next hop to itself", r, r, v, ok)
		}
		if p, ok := tb.AppendPath(nil, r, r); !ok || !slices.Equal(p, []int{r}) {
			t.Errorf("AppendPath(nil, %d, %d) = %v, %v, want [%d], true", r, r, p, ok, r)
		}
	}
	one := newMatrix(3)
	one[0][1], one[1][2] = 1, 1
	ow := one.build(t)
	if v, ok := ow.Next(2, 0); v != 0 || ok {
		t.Errorf("Next(2, 0) = %d, %v, want 0, false: 0 is unreachable from 2", v, ok)
	}
	if p, ok := ow.AppendPath([]int{7}, 2, 0); ok || !slices.Equal(p, []int{7}) {
		t.Errorf("AppendPath([7], 2, 0) = %v, %v, want [7] unchanged, false", p, ok)
	}
}

// TestAppendPath_FreshSlice verifies that the path is appended to the
// caller's slice and shares no storage with the tables: changing it changes
// no later answer.
//
// Requirement: FR-REG-026
func TestAppendPath_FreshSlice(t *testing.T) {
	t.Parallel()
	m := newMatrix(3)
	m[0][1], m[1][2] = 1, 1
	tb := m.build(t)
	first, _ := tb.AppendPath(nil, 0, 2)
	first[0], first[1], first[2] = 9, 9, 9
	if second, _ := tb.AppendPath(nil, 0, 2); !slices.Equal(second, []int{0, 1, 2}) {
		t.Errorf("AppendPath(nil, 0, 2) after changing a returned path = %v, want [0 1 2]", second)
	}
	hops := tb.AppendWithin(nil, 0, 10)
	hops[0] = Hop{Region: 9, Distance: 9}
	if again := tb.AppendWithin(nil, 0, 10); !slices.Equal(again, []Hop{{1, 1}, {2, 2}}) {
		t.Errorf("AppendWithin(nil, 0, 10) after changing a returned list = %v, want [{1 1} {2 2}]", again)
	}
}

// TestImports_StdlibAndChecked verifies that internal/graph imports the
// standard library and internal/checked and nothing else.
//
// Requirement: NFR-REG-004
func TestImports_StdlibAndChecked(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("Glob(*.go) = %v, %v", files, err)
	}
	const checked = `"github.com/Whealinghold/economy-sim/internal/checked"`
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			if p := imp.Path.Value; p != checked && strings.Contains(strings.Split(strings.Trim(p, `"`), "/")[0], ".") {
				t.Errorf("%s imports %s, want the standard library and internal/checked only", name, p)
			}
		}
	}
}

// TestHeap_PopsInOrder verifies that the hand-written heap pops items in
// ascending (distance, region) order, with equal items allowed.
//
// Guard: the search stays correct for any pop order, because a shorter
// distance found later is pushed and relaxed again, so a broken heap would
// pass every other test and only cost time. This pins the order itself.
func TestHeap_PopsInOrder(t *testing.T) {
	t.Parallel()
	rng := splitmix(11)
	for round := 0; round < 200; round++ {
		var h heap
		var want []item
		for i, n := 0, 1+rng.below(40); i < n; i++ {
			it := item{d: int64(rng.below(8)), v: int32(rng.below(5))}
			h.push(it)
			want = append(want, it)
		}
		slices.SortFunc(want, func(a, b item) int {
			if a.d != b.d {
				return int(a.d - b.d)
			}
			return int(a.v - b.v)
		})
		for i, w := range want {
			if got := h.pop(); got != w {
				t.Fatalf("round %d: pop %d = %v, want %v", round, i, got, w)
			}
		}
		if len(h) != 0 {
			t.Fatalf("round %d: %d items left", round, len(h))
		}
	}
}
