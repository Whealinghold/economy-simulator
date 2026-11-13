package polity

import (
	"errors"
	"fmt"
	"math/bits"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// The causes Build reports. The root maps them to its own sentinels
// (ErrJurisdictionCycle, ErrOutOfBounds); a *BuildError wraps one.
var (
	// ErrCycle is a parent chain that returns to the jurisdiction it started from,
	// a self-parent included.
	ErrCycle = errors.New("polity: parent chain has a cycle")
	// ErrDepth is a jurisdiction with maxDepth or more ancestors.
	ErrDepth = errors.New("polity: parent chain is too deep")
	// ErrIndex is a parent or a member that is neither -1 nor a jurisdiction index.
	ErrIndex = errors.New("polity: index out of range")
)

// BuildError says which cause Build found and at which index: a jurisdiction
// for ErrCycle and ErrDepth, a jurisdiction (parent) or a region (member) for
// ErrIndex. Index is the lowest one at fault, so the report does not depend on
// the order of the input rows' discovery.
type BuildError struct {
	Err   error
	Index int
	// Member is true for an ErrIndex found in the member array.
	Member bool
}

// Error implements error.
func (e *BuildError) Error() string { return fmt.Sprintf("%v at %d", e.Err, e.Index) }

// Unwrap returns the cause, so errors.Is matches ErrCycle, ErrDepth and ErrIndex.
func (e *BuildError) Unwrap() error { return e.Err }

// Tables are the derived tables of one polity: the parent and the depth of each
// jurisdiction, the direct jurisdiction and the chain of each region, and the
// children and the covered regions of each jurisdiction (spec-polity.md 4.4.1).
// They are built once and never change. A jurisdiction is an index from 0 and a
// region an index from 0; -1 means none. Every list is ascending by index except
// a chain (innermost first).
//
// A list a method returns is a view of the tables, valid for the life of the
// value and not to be modified; the root copies where architecture.md 12.4 asks
// for a fresh slice.
type Tables struct {
	parent []int32
	depth  []int32
	member []int32
	// chains is the flat store of every region's chain; chain(r) is
	// chains[chainAt[r]:chainAt[r+1]].
	chains  []int32
	chainAt []int32
	// children and covered are compressed rows by jurisdiction.
	children   []int32
	childrenAt []int32
	covered    []int32
	coveredAt  []int32
}

// Build derives the tables from parent (the parent of each jurisdiction, or -1)
// and member (the direct jurisdiction of each region, or -1). maxDepth is the
// bound on levels: a jurisdiction with maxDepth or more ancestors is ErrDepth, so
// a chain has at most maxDepth entries. The rules run in the order cycle, depth
// (an index out of range first), and the lowest index at fault is reported.
// The work is O(J + n * depth) after the arrays are given; the sort of the
// jurisdictions (the J log J) is the caller's (spec-polity.md 3.2.1, 4.4.2).
func Build(parent, member []int32, maxDepth int) (Tables, error) {
	j := len(parent)
	for i, p := range parent {
		if p < -1 || int(p) >= j {
			return Tables{}, &BuildError{Err: ErrIndex, Index: i}
		}
	}
	for r, m := range member {
		if m < -1 || int(m) >= j {
			return Tables{}, &BuildError{Err: ErrIndex, Index: r, Member: true}
		}
	}
	if i := firstCycle(parent); i >= 0 {
		return Tables{}, &BuildError{Err: ErrCycle, Index: i}
	}
	depth := depths(parent)
	for i, d := range depth {
		if int(d) >= maxDepth {
			return Tables{}, &BuildError{Err: ErrDepth, Index: i}
		}
	}
	t := Tables{parent: slices.Clone(parent), depth: depth, member: slices.Clone(member)}
	t.buildChains()
	t.buildChildren()
	t.buildCovered()
	return t, nil
}

// firstCycle returns the lowest jurisdiction that lies on a cycle of the parent
// links, or -1. It colours each jurisdiction once, so it is O(J).
func firstCycle(parent []int32) int {
	const (
		unseen = iota
		onPath
		done
	)
	state := make([]uint8, len(parent))
	lowest := -1
	var path []int32
	for start := range parent {
		path = path[:0]
		cur := int32(start)
		for cur >= 0 && state[cur] == unseen {
			state[cur] = onPath
			path = append(path, cur)
			cur = parent[cur]
		}
		if cur >= 0 && state[cur] == onPath {
			// The path from cur to its end is the cycle.
			at := slices.Index(path, cur)
			for _, m := range path[at:] {
				if lowest < 0 || int(m) < lowest {
					lowest = int(m)
				}
			}
		}
		for _, p := range path {
			state[p] = done
		}
	}
	return lowest
}

// depths returns the depth of each jurisdiction, a root being 0. parent must
// have no cycle. It resolves each jurisdiction once, so it is O(J).
func depths(parent []int32) []int32 {
	d := make([]int32, len(parent))
	for i := range d {
		d[i] = -1
	}
	var path []int32
	for start := range parent {
		path = path[:0]
		cur := int32(start)
		for cur >= 0 && d[cur] < 0 {
			path = append(path, cur)
			cur = parent[cur]
		}
		base := int32(-1)
		if cur >= 0 {
			base = d[cur]
		}
		for i := len(path) - 1; i >= 0; i-- {
			base++
			d[path[i]] = base
		}
	}
	return d
}

// buildChains fills the flat chain store: for each region the direct
// jurisdiction and then its ancestors.
func (t *Tables) buildChains() {
	t.chainAt = make([]int32, len(t.member)+1)
	total := 0
	for r, m := range t.member {
		t.chainAt[r] = int32(total)
		if m >= 0 {
			total += int(t.depth[m]) + 1
		}
	}
	t.chainAt[len(t.member)] = int32(total)
	t.chains = make([]int32, 0, total)
	for _, m := range t.member {
		for cur := m; cur >= 0; cur = t.parent[cur] {
			t.chains = append(t.chains, cur)
		}
	}
}

// buildChildren fills the children rows, ascending by jurisdiction.
func (t *Tables) buildChildren() {
	n := len(t.parent)
	t.childrenAt = make([]int32, n+1)
	for _, p := range t.parent {
		if p >= 0 {
			t.childrenAt[p+1]++
		}
	}
	for i := 0; i < n; i++ {
		t.childrenAt[i+1] += t.childrenAt[i]
	}
	t.children = make([]int32, t.childrenAt[n])
	next := slices.Clone(t.childrenAt[:n])
	for j, p := range t.parent {
		if p >= 0 {
			t.children[next[p]] = int32(j)
			next[p]++
		}
	}
}

// buildCovered fills the covered rows: region r joins the row of every
// jurisdiction in its chain. Regions are visited in ascending order, so each row
// is ascending.
func (t *Tables) buildCovered() {
	n := len(t.parent)
	t.coveredAt = make([]int32, n+1)
	for _, j := range t.chains {
		t.coveredAt[j+1]++
	}
	for i := 0; i < n; i++ {
		t.coveredAt[i+1] += t.coveredAt[i]
	}
	t.covered = make([]int32, t.coveredAt[n])
	next := slices.Clone(t.coveredAt[:n])
	for r := range t.member {
		for _, j := range t.Chain(int32(r)) {
			t.covered[next[j]] = int32(r)
			next[j]++
		}
	}
}

// Clone returns a deep copy of the tables that shares no storage with t.
func (t Tables) Clone() Tables {
	return Tables{
		parent: slices.Clone(t.parent), depth: slices.Clone(t.depth), member: slices.Clone(t.member),
		chains: slices.Clone(t.chains), chainAt: slices.Clone(t.chainAt),
		children: slices.Clone(t.children), childrenAt: slices.Clone(t.childrenAt),
		covered: slices.Clone(t.covered), coveredAt: slices.Clone(t.coveredAt),
	}
}

// Len returns the number of jurisdictions.
func (t Tables) Len() int { return len(t.parent) }

// Regions returns the number of regions the tables were built for.
func (t Tables) Regions() int { return len(t.member) }

// Parent returns the parent of j; false for a root or a j out of range.
func (t Tables) Parent(j int32) (int32, bool) {
	if j < 0 || int(j) >= len(t.parent) || t.parent[j] < 0 {
		return -1, false
	}
	return t.parent[j], true
}

// Depth returns the number of ancestors of j, a root being 0; false for a j out
// of range.
func (t Tables) Depth(j int32) (int, bool) {
	if j < 0 || int(j) >= len(t.depth) {
		return 0, false
	}
	return int(t.depth[j]), true
}

// Children returns the children of j, ascending; nil for a j out of range.
func (t Tables) Children(j int32) []int32 {
	if j < 0 || int(j) >= len(t.parent) {
		return nil
	}
	return t.children[t.childrenAt[j]:t.childrenAt[j+1]]
}

// Of returns the direct jurisdiction of region r; false when r has none or is
// out of range.
func (t Tables) Of(r int32) (int32, bool) {
	if r < 0 || int(r) >= len(t.member) || t.member[r] < 0 {
		return -1, false
	}
	return t.member[r], true
}

// Chain returns the chain of region r, innermost first; nil when r has none or
// is out of range.
func (t Tables) Chain(r int32) []int32 {
	if r < 0 || int(r) >= len(t.member) {
		return nil
	}
	return t.chains[t.chainAt[r]:t.chainAt[r+1]]
}

// Covered returns the regions j covers, ascending, the direct members of its
// descendants included; nil for a j out of range.
func (t Tables) Covered(j int32) []int32 {
	if j < 0 || int(j) >= len(t.parent) {
		return nil
	}
	return t.covered[t.coveredAt[j]:t.coveredAt[j+1]]
}

// Covers reports whether j is in the chain of r (spec-polity.md 4.1.1). It
// allocates nothing and is O(depth).
func (t Tables) Covers(j, r int32) bool {
	if j < 0 {
		return false
	}
	return slices.Contains(t.Chain(r), j)
}

// Common returns the innermost jurisdiction that covers both regions: the first
// element of the chain of a that is in the chain of b; -1 when there is none or
// an ID is out of range (spec-polity.md 4.1.1). It allocates nothing.
func (t Tables) Common(a, b int32) int32 {
	cb := t.Chain(b)
	for _, j := range t.Chain(a) {
		if slices.Contains(cb, j) {
			return j
		}
	}
	return -1
}

// Crossing returns the two lists of the ordered pair (a, b): exit, the elements
// of the chain of a that are not in the chain of b, innermost first; and enter,
// the elements of the chain of b that are not in the chain of a, outermost first
// (spec-polity.md 4.1.1). Both are fresh slices, nil when empty, and both are
// empty when a == b or either region is out of range (spec-polity.md 4.2.2).
func (t Tables) Crossing(a, b int32) (exit, enter []int32) {
	if !t.region(a) || !t.region(b) {
		return nil, nil
	}
	ca, cb := t.Chain(a), t.Chain(b)
	for _, j := range ca {
		if !slices.Contains(cb, j) {
			exit = append(exit, j)
		}
	}
	for i := len(cb) - 1; i >= 0; i-- {
		if !slices.Contains(ca, cb[i]) {
			enter = append(enter, cb[i])
		}
	}
	return exit, enter
}

// region reports whether r is a region the tables were built for.
func (t Tables) region(r int32) bool { return r >= 0 && int(r) < len(t.member) }

// EdgeBetween reports whether the crossing of (from, to) has jurisdiction a in
// its exit list and jurisdiction b in its enter list: the test of
// spec-polity.md 4.2.3 for one edge. It allocates nothing and is O(depth).
func (t Tables) EdgeBetween(from, to, a, b int32) bool {
	if a < 0 || b < 0 {
		return false
	}
	cf, ct := t.Chain(from), t.Chain(to)
	return slices.Contains(cf, a) && !slices.Contains(ct, a) &&
		slices.Contains(ct, b) && !slices.Contains(cf, b)
}

// Row is one row of a table keyed by jurisdiction: the jurisdiction index and
// the owner's value (a cap, an addition or a factor in basis points).
type Row struct {
	Jurisdiction int32
	Value        int64
}

// find returns the value of the row of j. rows are ascending by jurisdiction
// with one row each (spec-polity.md 6.1.1).
func find(rows []Row, j int32) (int64, bool) {
	i, ok := slices.BinarySearchFunc(rows, j, func(r Row, j int32) int {
		switch {
		case r.Jurisdiction < j:
			return -1
		case r.Jurisdiction > j:
			return 1
		}
		return 0
	})
	if !ok {
		return 0, false
	}
	return rows[i].Value, true
}

// Innermost returns the value of the row of the first jurisdiction of chain
// (innermost first) that has one; ok is false when none does, and the owner's
// default applies (spec-polity.md 6.1.2, mode innermost). rows are ascending by
// jurisdiction.
func Innermost(rows []Row, chain []int32) (value int64, ok bool) {
	for _, j := range chain {
		if v, found := find(rows, j); found {
			return v, true
		}
	}
	return 0, false
}

// Added returns the sum of the values of every row of the chain; zero when none
// has a row, so the owner adds it to its default (mode added, outermost first
// in the spec's wording, which a sum does not depend on). ok is false when the
// sum leaves int64, and the caller treats it as an operation that cannot be
// applied (architecture.md 3.5).
func Added(rows []Row, chain []int32) (sum int64, ok bool) {
	for _, j := range chain {
		v, found := find(rows, j)
		if !found {
			continue
		}
		if sum, ok = checked.Add(sum, v); !ok {
			return 0, false
		}
	}
	return sum, true
}

// Factors appends to dst the value of every row of the chain, in chain order,
// for Scaled.
func Factors(dst []int64, rows []Row, chain []int32) []int64 {
	for _, j := range chain {
		if v, found := find(rows, j); found {
			dst = append(dst, v)
		}
	}
	return dst
}

// WholeBP is one in basis points: the denominator of each factor of Scaled.
const WholeBP = 10_000

// MaxFactors is the most factors Scaled takes: the denominator WholeBP^k must
// fit in 128 bits, which holds up to k = 9, one more than MaxPolityDepth.
const MaxFactors = 9

// Scaled returns base * f1 * f2 * ... / 10 000^k for the k factors in basis
// points, computed exactly in 128 bits and rounded once at the end, half to
// even (spec-polity.md 6.1.2). Because the product is exact before the one
// rounding, the result does not depend on the order of the factors. With no
// factor it is base. ok is false when base or a factor is negative, there are
// more than MaxFactors factors, the exact product does not fit in 128 bits, or
// the result does not fit in int64.
//
// The helpers of internal/checked hold one 64-bit product (MulDivDiv) or a sum of
// 64-bit products (Acc); a chain of up to eight factors needs the product of
// several, so this package keeps the 128-bit product and quotient it needs and
// leaves internal/checked unchanged.
func Scaled(base int64, factorsBP []int64) (int64, bool) {
	num, den, ok := product(base, factorsBP)
	if !ok {
		return 0, false
	}
	q, r := num.divmod(den)
	// Round half to even: compare 2r with den, written as r with den - r.
	if rest := den.sub(r); r.cmp(rest) > 0 || (r.cmp(rest) == 0 && q.lo&1 == 1) {
		q = q.add1()
	}
	if q.hi != 0 || q.lo > 1<<63-1 {
		return 0, false
	}
	return int64(q.lo), true
}

// product returns the exact numerator base * f1 * ... * fk and the denominator
// 10 000^k of Scaled. ok is false for a negative input, more than MaxFactors
// factors or a numerator that does not fit in 128 bits.
func product(base int64, factorsBP []int64) (num, den u128, ok bool) {
	if base < 0 || len(factorsBP) > MaxFactors {
		return u128{}, u128{}, false
	}
	num, den = u128{lo: uint64(base)}, u128{lo: 1}
	for _, f := range factorsBP {
		if f < 0 {
			return u128{}, u128{}, false
		}
		if num, ok = num.mul(uint64(f)); !ok {
			return u128{}, u128{}, false
		}
		den, _ = den.mul(WholeBP) // at most 10^36, below 2^128
	}
	return num, den, true
}

// u128 is an unsigned 128-bit integer, enough for base times nine factors only
// when they fit; mul reports a product that does not.
type u128 struct{ hi, lo uint64 }

// mul returns x * m and whether it fit.
func (x u128) mul(m uint64) (u128, bool) {
	hi1, lo := bits.Mul64(x.lo, m)
	over, hi2 := bits.Mul64(x.hi, m) // the high word of x.hi*m is the overflow
	hi, carry := bits.Add64(hi1, hi2, 0)
	return u128{hi: hi, lo: lo}, over == 0 && carry == 0
}

// cmp compares x and y: -1, 0 or 1.
func (x u128) cmp(y u128) int {
	switch {
	case x.hi != y.hi:
		if x.hi < y.hi {
			return -1
		}
		return 1
	case x.lo < y.lo:
		return -1
	case x.lo > y.lo:
		return 1
	}
	return 0
}

// sub returns x - y for x >= y.
func (x u128) sub(y u128) u128 {
	lo, b := bits.Sub64(x.lo, y.lo, 0)
	hi, _ := bits.Sub64(x.hi, y.hi, b)
	return u128{hi: hi, lo: lo}
}

// add1 returns x + 1, which cannot wrap here: it follows a quotient below 2^127.
func (x u128) add1() u128 {
	lo, c := bits.Add64(x.lo, 1, 0)
	return u128{hi: x.hi + c, lo: lo}
}

// shl1 returns x shifted left by one bit.
func (x u128) shl1() u128 { return u128{hi: x.hi<<1 | x.lo>>63, lo: x.lo << 1} }

// divmod returns the quotient and remainder of x / d by long division, one bit
// at a time; d must not be 0. It is 128 steps of shifts and compares, which is
// cheap beside the calls that need it.
func (x u128) divmod(d u128) (q, r u128) {
	for i := 127; i >= 0; i-- {
		r = r.shl1()
		var bit uint64
		if i >= 64 {
			bit = x.hi >> uint(i-64) & 1
		} else {
			bit = x.lo >> uint(i) & 1
		}
		r.lo |= bit
		q = q.shl1()
		if r.cmp(d) >= 0 {
			r = r.sub(d)
			q.lo |= 1
		}
	}
	return q, r
}
