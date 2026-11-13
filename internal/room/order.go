package room

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Candidate is a commodity whose units may go when a load must fall: its ID
// (the tie-break), the size of a unit, and its units as lots in the order they
// are taken (oldest first). The discard of 6.3.2 gives one lot, the surplus.
type Candidate struct {
	// Commodity is the tie-break: the lower ID goes first.
	Commodity int
	Size      Pair
	Lots      []int64
}

// Step is one move of the order: take Units from lot Lot of candidate Cand
// (indexes into the slice given). A step of the excess order takes a whole lot.
type Step struct {
	Cand, Lot int
	Units     int64
}

// order is the rule shared by 5.5.2 and 6.3.2 (spec-storage.md 5.5.2.1). While
// some shortfall is above 0: the binding dimension is the one whose shortfall is
// largest relative to the room (cross-multiplied, ties to mass); among the
// candidates with units left and a size above 0 in it, the one with the
// greatest load of those units in it, ties by the lower commodity ID, goes: whole
// lots oldest first, or, if byUnit, the least whole units that meet the
// shortfall. Every shortfall is then recomputed and the ranking taken again.
// met is false when a shortfall remains and no candidate can lower it.
func order(shortfall, room Pair, cands []Candidate, byUnit bool) (steps []Step, met bool) {
	left := make([][]int64, len(cands))
	for i, c := range cands {
		left[i] = append([]int64(nil), c.Lots...)
	}
	for {
		d, ok := binding(shortfall, room)
		if !ok {
			return steps, true
		}
		i := pick(cands, left, d)
		if i < 0 {
			return steps, false
		}
		size := dimOf(cands[i].Size, d)
		for l := 0; l < len(left[i]) && dimOf(shortfall, d) > 0; l++ {
			n := left[i][l]
			if n == 0 {
				continue
			}
			if byUnit {
				n = min(n, ceilDiv(dimOf(shortfall, d), size))
			}
			steps = append(steps, Step{Cand: i, Lot: l, Units: n})
			left[i][l] -= n
			shortfall = lower(shortfall, cands[i].Size, n)
		}
	}
}

// binding returns the dimension (0 mass, 1 bulk) whose shortfall is above 0
// and largest relative to the room, S/max(room,1), by cross-multiplication in
// 128 bits; ties go to mass. ok is false when no shortfall is above 0.
func binding(s, room Pair) (d int, ok bool) {
	switch {
	case s.Mass <= 0 && s.Bulk <= 0:
		return 0, false
	case s.Bulk <= 0:
		return 0, true
	case s.Mass <= 0:
		return 1, true
	}
	// Mass wins unless S_bulk / R_bulk > S_mass / R_mass.
	c, ok := checked.Cmp3(s.Bulk, max(room.Mass, 1), 1, s.Mass, max(room.Bulk, 1), 1)
	if ok && c > 0 {
		return 1, true
	}
	return 0, true
}

// pick returns the index of the candidate with units left and a size above 0 in
// dimension d whose units have the greatest load in it, ties by the lower
// commodity ID, or -1.
func pick(cands []Candidate, left [][]int64, d int) int {
	best, bestLoad := -1, int64(-1)
	for i, c := range cands {
		size := dimOf(c.Size, d)
		var units int64
		for _, n := range left[i] {
			units += n
		}
		if size <= 0 || units == 0 {
			continue
		}
		load, ok := checked.Mul(units, size)
		if !ok {
			load = math.MaxInt64
		}
		if load > bestLoad || (load == bestLoad && c.Commodity < cands[best].Commodity) {
			best, bestLoad = i, load
		}
	}
	return best
}

// lower returns the shortfall after n units of size sz go.
func lower(s, sz Pair, n int64) Pair {
	return Pair{Mass: shortfallAfter(s.Mass, n, sz.Mass), Bulk: shortfallAfter(s.Bulk, n, sz.Bulk)}
}

// shortfallAfter is s - n*sz. A product or difference that does not fit int64
// removes more than any shortfall, so it saturates low (met).
func shortfallAfter(s, n, sz int64) int64 {
	m, ok := checked.Mul(n, sz)
	if ok {
		if v, ok := checked.Sub(s, m); ok {
			return v
		}
	}
	return math.MinInt64
}

func dimOf(p Pair, d int) int64 {
	if d == 0 {
		return p.Mass
	}
	return p.Bulk
}

// ceilDiv returns ceil(a/b) for a >= 0, b > 0.
func ceilDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 {
		q++
	}
	return q
}
