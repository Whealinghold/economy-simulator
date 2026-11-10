// Package calibration holds the pure rules of a calibration of the economy
// (spec-calibration.md): the grid of a knob and its neighbours, the seed blocks,
// the plan of the screen, the explore and the confirm, the choice among the
// points, the alias test, the calibration record and the overlay of a choice.
//
// It is standard library, internal/checked and internal/design only (it imports
// none of the root, config/, strategy/, indicator/ or experiment/,
// architecture.md 2.9). It holds no state, no float, no clock, no random source
// and no goroutine (NFR-CLB-003): a failure is a returned error and a decision
// is a line of the record. The procedure constants are placeholders, each named
// with the section of the spec that gives it (N-015).
package calibration

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// calError is the type of the sentinel errors of the package. They are constants
// of that type, not package variables (NFR-CLB-003); match them with errors.Is.
type calError string

func (e calError) Error() string { return string(e) }

const (
	// ErrNoGrid is returned for a `geo` knob whose base is 0 (a `z` row, whose owner
	// default is 0): it has no grid, is not searched, and the record says
	// `unsearched at_off` (spec-calibration.md 5.2, FR-CLB-108). A `lin` row at 0
	// does get a grid.
	ErrNoGrid = calError("calibration: the knob has no grid at a base of 0")
	// ErrBase is returned for a base outside its bounds, a rule that cannot make
	// a grid (a step below 1, an empty set) or bounds that are not ordered.
	ErrBase = calError("calibration: base, bounds or rule cannot make a grid")
	// ErrOverflow is returned by the checked multiply of the grid when a level
	// would leave int64 (FR-CLB-013): the level is an error, never a wrapped value.
	ErrOverflow = calError("calibration: a level leaves int64")
)

// Factor of a geometric grid: 2 is a placeholder (P) of spec-calibration.md 5.2.
const geoFactor = 2

// maxLevels bounds the levels of one grid, so that a step of 1 over the range of
// an int64 is an ErrBase and not an allocation of the memory of the machine. It
// is a limit of this code, not of the spec: the widest bound of the registry is
// far below it.
const maxLevels = 1 << 16

// Kind is the kind of a grid rule (spec-calibration.md 5.2 step 1).
type Kind uint8

// The kinds of grid: geometric (`geo`), linear (`lin<s>`) and a listed set
// (`set:<a/b/c>`).
const (
	Geo Kind = iota
	Lin
	Set
)

// Rule is the grid rule of a knob: Step is the `s` of `lin<s>`, Set the listed
// values of `set:`, in the order given.
type Rule struct {
	Kind Kind
	Step int64
	Set  []int64
}

// Bounds is the validation bound `[Lo, Hi]` of the owner, evaluated with the
// other fields as configured.
type Bounds struct{ Lo, Hi int64 }

// Off is the off value of a D row, resolved by the caller (`off=min` and
// `off=max` are read from the owner's bound and never copied into the row).
type Off struct {
	Has   bool
	Value int64
}

// Edge says why a side of a grid has no neighbour (spec-calibration.md 5.2
// step 4): None for a side that has one.
type Edge uint8

// The edges: the next level was outside the bounds or below the floor of a
// `geo` grid (`edge:bound`), was the off value or beyond it and was cut
// (`edge:off`), or was invalid (`edge:rule`).
const (
	None Edge = iota
	Bound
	OffEdge
	RuleEdge
)

// Side is a neighbour of the base: Present with its Level, or an Edge.
type Side struct {
	Present bool
	Level   int64
	Edge    Edge
}

// Grid is the kept valid levels of a knob, ascending for `geo` and `lin` and in
// the order given for `set`, with the base always among them (Base is its
// index) and the two neighbours of the base.
type Grid struct {
	Levels       []int64
	Base         int
	Lower, Upper Side
}

// Build makes the grid of a knob in the four steps of spec-calibration.md 5.2:
// the levels of the kind in bounds, the cut of the off value of a D row (the base
// is never cut), the skip of the levels that valid rejects, and the neighbours.
// valid may be nil, meaning every level is valid. A `geo` knob whose base is 0
// (the `z` rows) returns ErrNoGrid.
func Build(r Rule, base int64, b Bounds, off Off, valid func(level int64) bool) (Grid, error) {
	if b.Lo > b.Hi || base < b.Lo || base > b.Hi {
		return Grid{}, ErrBase
	}
	if base == 0 && r.Kind == Geo {
		return Grid{}, ErrNoGrid
	}
	all, err := levelsOf(r, base, b)
	if err != nil {
		return Grid{}, err
	}
	kept := cutOff(all, base, off, r.Kind == Set)
	return neighbours(all, kept, base, valid), nil
}

// levelsOf is step 1: the levels of the kind, in bounds, with the base among them.
func levelsOf(r Rule, base int64, b Bounds) ([]int64, error) {
	switch r.Kind {
	case Geo:
		if base < 1 {
			return nil, ErrBase
		}
		return geoLevels(base, b)
	case Lin:
		if r.Step < 1 {
			return nil, ErrBase
		}
		return linLevels(base, r.Step, b)
	case Set:
		return setLevels(r.Set, base, b)
	}
	return nil, ErrBase
}

// geoLevels are base*2^k for k >= 0 and rdiv(base, 2^-k) for k < 0, in
// [max(Lo, 1), Hi]: no geo level is below 1 (0 is usually an off value).
func geoLevels(base int64, b Bounds) ([]int64, error) {
	floor := max(b.Lo, 1)
	out := []int64{base}
	for v := base; ; {
		next, err := double(v)
		if err != nil || next > b.Hi {
			break // an overflowing product is above Hi: not a level, never a wrapped one
		}
		out = append(out, next)
		v = next
	}
	for k := uint(1); ; k++ {
		v := halve(base, k)
		if v < floor {
			break
		}
		out = append(out, v)
	}
	slices.Sort(out)
	return slices.Compact(out), nil // the halvings repeat the floor: rdiv(5, 8) = 1 = rdiv(5, 4)
}

// double is the checked multiply of the geometric grid: it reports ErrOverflow
// where the product leaves int64.
func double(v int64) (int64, error) {
	p, ok := checked.Mul(v, geoFactor)
	if !ok {
		return 0, ErrOverflow
	}
	return p, nil
}

// halve is rdiv(v, 2^k), half to even, for v >= 1.
func halve(v int64, k uint) int64 {
	if k >= 63 { // v < 2^63: v / 2^63 is above one half only for v > 2^62
		if k == 63 && v > 1<<62 {
			return 1
		}
		return 0
	}
	q, _ := checked.MulDivHalfEven(v, 1, 1<<k)
	return q
}

// linLevels are base + k*step in bounds. The counts below and above the base
// are computed first, in unsigned arithmetic, so that no level wraps and a grid
// of more than maxLevels levels is refused instead of built.
func linLevels(base, step int64, b Bounds) ([]int64, error) {
	s := uint64(step)
	below := (uint64(base) - uint64(b.Lo)) / s
	above := (uint64(b.Hi) - uint64(base)) / s
	if below+above >= maxLevels { // no wrap: below+above is at most (Hi-Lo)/s, below 2^64
		return nil, ErrBase
	}
	var out []int64
	for k := below; k > 0; k-- {
		out = append(out, base-int64(k*s)) // k*s <= base-Lo: in int64, no wrap
	}
	out = append(out, base)
	for k := uint64(1); k <= above; k++ {
		out = append(out, base+int64(k*s))
	}
	return out, nil
}

// setLevels are the listed values in the order given, in bounds, without
// duplicates; the base must be one of them.
func setLevels(listed []int64, base int64, b Bounds) ([]int64, error) {
	if len(listed) > maxLevels {
		return nil, ErrBase
	}
	var out []int64
	seen := make(map[int64]bool, len(listed))
	for _, v := range listed {
		if v >= b.Lo && v <= b.Hi && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	if !slices.Contains(out, base) {
		return nil, ErrBase
	}
	return out, nil
}

// cutOff is step 2: in a D row, the level equal to the off value and every level
// beyond it, on that side of the base, is removed, except the base. A sorted
// grid cuts by value; a set cuts by position (the order given).
func cutOff(all []int64, base int64, off Off, byPosition bool) []int64 {
	if !off.Has || off.Value == base {
		return slices.Clone(all)
	}
	bi := slices.Index(all, base)
	oi := offIndex(all, bi, off.Value, byPosition)
	out := make([]int64, 0, len(all))
	for i, v := range all {
		if v == base || !((oi < bi && i <= oi) || (oi > bi && i >= oi)) {
			out = append(out, v)
		}
	}
	return out
}

// offIndex is the position the cut starts from: for a sorted grid, the first
// level on the off side at or beyond the off value; for a set, the position of
// the off value itself, or the base's when the set does not list it.
func offIndex(all []int64, bi int, off int64, byPosition bool) int {
	if byPosition {
		if i := slices.Index(all, off); i >= 0 {
			return i
		}
		return bi
	}
	if off < all[bi] {
		return slices.IndexFunc(all, func(v int64) bool { return v > off }) - 1
	}
	return slices.IndexFunc(all, func(v int64) bool { return v >= off }) // -1: nothing is cut
}

// neighbours is steps 3 and 4: the kept levels that valid accepts, the base
// always kept, and the nearest of them on each side of the base.
func neighbours(all, kept []int64, base int64, valid func(int64) bool) Grid {
	var g Grid
	for _, v := range kept {
		if v == base || valid == nil || valid(v) {
			g.Levels = append(g.Levels, v)
		}
	}
	g.Base = slices.Index(g.Levels, base)
	g.Lower = Side{Edge: sideEdge(all, kept, base, true)}
	g.Upper = Side{Edge: sideEdge(all, kept, base, false)}
	if g.Base > 0 {
		g.Lower = Side{Present: true, Level: g.Levels[g.Base-1]}
	}
	if g.Base+1 < len(g.Levels) {
		g.Upper = Side{Present: true, Level: g.Levels[g.Base+1]}
	}
	return g
}

// sideEdge says why a side has no neighbour, from the uncut levels: no level on
// that side is a bound; the nearest cut (it is the off value or lies beyond it) is
// an off edge; the nearest kept but invalid is a rule edge.
func sideEdge(all, kept []int64, base int64, below bool) Edge {
	bi := slices.Index(all, base)
	var near int64
	switch {
	case below && bi > 0:
		near = all[bi-1]
	case !below && bi+1 < len(all):
		near = all[bi+1]
	default:
		return Bound
	}
	if !slices.Contains(kept, near) { // the off value, or a level beyond it, was cut
		return OffEdge
	}
	return RuleEdge
}

// Distance is the grid distance between two levels of a grid: the number of
// index steps between them (spec-calibration.md 7.2.3). It counts steps over
// kept valid levels, so a skipped level is not a step.
// An index outside the grid gives -1.
func Distance(g Grid, i, j int) int64 {
	if i < 0 || j < 0 || i >= len(g.Levels) || j >= len(g.Levels) {
		return -1
	}
	return int64(max(i, j) - min(i, j))
}
