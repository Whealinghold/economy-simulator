// Package graph holds the algorithms of the region graph over primitive
// slices: the shortest-path tables, the next hop, the strongly connected
// components and the effective length of an edge (spec-regions.md 4.3, 4.4).
// It knows no region name: a region is its index 0 to n-1, in the caller's
// ascending order of ID, and the caller converts at the call
// (architecture.md 2.9).
//
// The tables are defined by their result (spec-regions.md 4.3.1), so any exact
// algorithm gives the same ones. This one runs a Dijkstra search from every
// region with a binary heap keyed (distance, region), reads the next hop from
// the distance table, and numbers the components from it.
package graph

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

var (
	// ErrInvalid is returned by Build for a count or an edge list that breaks
	// its contract: an end outside [0, n), a self edge, a length below 1, or
	// edges that are not in strictly ascending (From, To) order.
	ErrInvalid = errors.New("graph: invalid input")
	// ErrOverflow is returned by Build when a path sum does not fit in int64.
	ErrOverflow = errors.New("graph: path length overflows int64")
)

// MaxRegions is the most regions Build accepts: the tables take n*n values of
// 12 bytes, 200 MB at the bound. The root package's MaxRegions must not exceed it.
const MaxRegions = 4096

// basisPoints is the denominator of an edge status factor (architecture.md 3.5).
const basisPoints = 10_000

// Edge is an open directed edge between two regions with its effective
// length, at least 1.
type Edge struct {
	From, To int
	Length   int64
}

// Hop is a region and a length: the shortest distance in AppendWithin.
type Hop struct {
	Region   int
	Distance int64
}

// Effective returns the effective length of an open edge, max(1, ceil(base *
// factorBP / 10 000)), with the product held in 128 bits (spec-regions.md
// 7.2.1). ok is false when the result does not fit in int64. A base or factor
// below 1 is outside the contract and gives 1.
func Effective(base, factorBP int64) (int64, bool) {
	q, ok := checked.MulDivCeil(base, factorBP, basisPoints)
	if !ok {
		return 0, false
	}
	return max(1, q), true
}

// Tables are the derived tables of one graph. They are read-only after Build.
type Tables struct {
	n     int
	off   []int32 // the out-edges of region r are at off[r] to off[r+1]
	to    []int32 // their ends, ascending within a region
	w     []int64 // their effective lengths
	dist  []int64 // n*n, row = source; -1 when unreachable
	next  []int32 // n*n; -1 when a == b or b is unreachable
	comp  []int32
	comps int
}

// Build derives the tables of n regions from the open edges, which must be in
// strictly ascending (From, To) order. A closed edge is simply not in the list.
// n is at most MaxRegions and the list at most n*(n-1) edges, checked before
// anything is allocated. It returns ErrInvalid for an edge list that breaks its contract and ErrOverflow for a
// path sum that does not fit in int64.
func Build(n int, open []Edge) (*Tables, error) {
	t, err := newTables(n, open)
	if err != nil {
		return nil, err
	}
	if err := t.shortest(); err != nil {
		return nil, err
	}
	t.hops()
	t.components()
	return t, nil
}

// newTables validates the input and lays the open edges out by source.
func newTables(n int, open []Edge) (*Tables, error) {
	if n < 0 || n > MaxRegions {
		return nil, fmt.Errorf("%w: %d regions, at most %d", ErrInvalid, n, MaxRegions)
	}
	if len(open) > n*(n-1) { // strictly ascending pairs without a loop cannot be more
		return nil, fmt.Errorf("%w: %d edges for %d regions", ErrInvalid, len(open), n)
	}
	t := &Tables{
		n:    n,
		off:  make([]int32, n+1),
		to:   make([]int32, len(open)),
		w:    make([]int64, len(open)),
		dist: make([]int64, n*n),
		next: make([]int32, n*n),
		comp: make([]int32, n),
	}
	for i, e := range open {
		if err := checkEdge(n, open, i); err != nil {
			return nil, err
		}
		t.off[e.From+1]++
		t.to[i], t.w[i] = int32(e.To), e.Length
	}
	for r := 0; r < n; r++ {
		t.off[r+1] += t.off[r]
	}
	return t, nil
}

// checkEdge reports whether open[i] is in range, is not a self edge, has a
// length of at least 1 and sorts strictly after open[i-1].
func checkEdge(n int, open []Edge, i int) error {
	e := open[i]
	switch {
	case e.From < 0 || e.From >= n || e.To < 0 || e.To >= n:
		return fmt.Errorf("%w: edge %d (%d, %d) has an end outside [0, %d)", ErrInvalid, i, e.From, e.To, n)
	case e.From == e.To:
		return fmt.Errorf("%w: edge %d is a self edge at %d", ErrInvalid, i, e.From)
	case e.Length < 1:
		return fmt.Errorf("%w: edge %d (%d, %d) has length %d", ErrInvalid, i, e.From, e.To, e.Length)
	case i > 0 && !before(open[i-1], e):
		return fmt.Errorf("%w: edge %d (%d, %d) is not after edge %d (%d, %d)", ErrInvalid, i, e.From, e.To, i-1, open[i-1].From, open[i-1].To)
	}
	return nil
}

// before reports whether a sorts strictly before b by (From, To).
func before(a, b Edge) bool { return a.From < b.From || (a.From == b.From && a.To < b.To) }

// item is a heap entry: a tentative distance to a region.
type item struct {
	d int64
	v int32
}

func (a item) less(b item) bool { return a.d < b.d || (a.d == b.d && a.v < b.v) }

// heap is a binary min-heap of items, written by hand so a push does not
// allocate once the slice has grown.
type heap []item

func (h *heap) push(it item) {
	*h = append(*h, it)
	s := *h
	for i := len(s) - 1; i > 0; {
		p := (i - 1) / 2
		if !s[i].less(s[p]) {
			break
		}
		s[i], s[p] = s[p], s[i]
		i = p
	}
}

func (h *heap) pop() item {
	s := *h
	top := s[0]
	last := len(s) - 1
	s[0] = s[last]
	s = s[:last]
	for i := 0; ; {
		l, small := 2*i+1, i
		if l < last && s[l].less(s[small]) {
			small = l
		}
		if l+1 < last && s[l+1].less(s[small]) {
			small = l + 1
		}
		if small == i {
			break
		}
		s[i], s[small] = s[small], s[i]
		i = small
	}
	*h = s
	return top
}

// shortest fills the distance table with one single-source search per region.
func (t *Tables) shortest() error {
	var h heap
	for s := 0; s < t.n; s++ {
		if err := t.search(s, &h); err != nil {
			return err
		}
	}
	return nil
}

// search fills row s of the distance table. Every length is at least 1, so a
// region is final when first popped at its recorded distance.
func (t *Tables) search(s int, h *heap) error {
	row := t.dist[s*t.n : (s+1)*t.n]
	for i := range row {
		row[i] = -1
	}
	row[s] = 0
	*h = (*h)[:0]
	h.push(item{0, int32(s)})
	for len(*h) > 0 {
		it := h.pop()
		if it.d > row[it.v] {
			continue // a shorter way was found after this entry was pushed
		}
		for e := t.off[it.v]; e < t.off[it.v+1]; e++ {
			nd, ok := checked.Add(it.d, t.w[e])
			if !ok {
				return fmt.Errorf("%w: searching from region %d", ErrOverflow, s)
			}
			if v := t.to[e]; row[v] < 0 || nd < row[v] {
				row[v] = nd
				h.push(item{nd, v})
			}
		}
	}
	return nil
}

// hops fills the next-hop table: for a != b with b reachable, the smallest
// out-neighbour v of a with w(a, v) + dist(v, b) = dist(a, b), found by
// scanning the out-edges in ascending order (spec-regions.md 4.3.1).
func (t *Tables) hops() {
	n := t.n
	for a := 0; a < n; a++ {
		for b := 0; b < n; b++ {
			t.next[a*n+b] = t.firstHop(a, b)
		}
	}
}

func (t *Tables) firstHop(a, b int) int32 {
	d := t.dist[a*t.n+b]
	if a == b || d < 0 {
		return -1
	}
	for e := t.off[a]; e < t.off[a+1]; e++ {
		// dist(v, b) = d - w is the same equation as w + dist(v, b) = d, and
		// the subtraction cannot overflow: d and w are positive.
		if dv := t.dist[int(t.to[e])*t.n+b]; dv >= 0 && dv == d-t.w[e] {
			return t.to[e]
		}
	}
	return -1 // unreachable: a shortest walk starts with some out-edge
}

// components numbers the strongly connected components from 0 in ascending
// order of their smallest region: two regions share one when each reaches the
// other.
func (t *Tables) components() {
	n := t.n
	for r := range t.comp {
		t.comp[r] = -1
	}
	for r := 0; r < n; r++ {
		if t.comp[r] >= 0 {
			continue
		}
		for j := r; j < n; j++ {
			if t.comp[j] < 0 && t.dist[r*n+j] >= 0 && t.dist[j*n+r] >= 0 {
				t.comp[j] = int32(t.comps)
			}
		}
		t.comps++
	}
}

// Dist returns the shortest distance from a to b, (0, true) for a == b, and
// (0, false) when b is unreachable or a region is out of range.
func (t *Tables) Dist(a, b int) (int64, bool) {
	if a < 0 || a >= t.n || b < 0 || b >= t.n {
		return 0, false
	}
	d := t.dist[a*t.n+b]
	return max(d, 0), d >= 0
}

// Next returns the first region after a on the path from a to b, and false
// when a == b, b is unreachable or a region is out of range.
func (t *Tables) Next(a, b int) (int, bool) {
	if a < 0 || a >= t.n || b < 0 || b >= t.n || t.next[a*t.n+b] < 0 {
		return 0, false
	}
	return int(t.next[a*t.n+b]), true
}

// Comp returns the number of the component of region r, and false when r is
// out of range.
func (t *Tables) Comp(r int) (int, bool) {
	if r < 0 || r >= t.n {
		return 0, false
	}
	return int(t.comp[r]), true
}

// Comps returns the number of strongly connected components.
func (t *Tables) Comps() int { return t.comps }

// AppendPath appends the path from a to b, both ends included ([a] when they
// are equal), to dst, and returns it. When b is unreachable or a region is out
// of range it returns dst unchanged and false.
func (t *Tables) AppendPath(dst []int, a, b int) ([]int, bool) {
	if _, ok := t.Dist(a, b); !ok {
		return dst, false
	}
	dst = append(dst, a)
	for a != b {
		a = int(t.next[a*t.n+b])
		dst = append(dst, a)
	}
	return dst, true
}

// AppendWithin appends to dst every region other than a whose shortest
// distance from a is at most limit, in ascending (distance, region) order. A
// negative limit or a region out of range appends nothing.
func (t *Tables) AppendWithin(dst []Hop, a int, limit int64) []Hop {
	if a < 0 || a >= t.n || limit < 0 {
		return dst
	}
	start := len(dst)
	for b, d := range t.dist[a*t.n : (a+1)*t.n] {
		if b != a && d >= 0 && d <= limit {
			dst = append(dst, Hop{Region: b, Distance: d})
		}
	}
	slices.SortFunc(dst[start:], func(x, y Hop) int {
		return cmp.Or(cmp.Compare(x.Distance, y.Distance), cmp.Compare(x.Region, y.Region))
	})
	return dst
}
