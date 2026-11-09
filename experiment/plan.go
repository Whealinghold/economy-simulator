package experiment

import (
	"github.com/Whealinghold/economy-sim/internal/design"
)

// A Key names one run.
type Key struct {
	// Point is the index of the point in the design.
	Point int
	// Replica counts from 0; the seed is Seed0 plus Replica (spec-experiments.md 4.6).
	Replica int
	// Variant and Cont are 0 and 0 without a branch.
	Variant, Cont int
}

// A Unit is the runs that share a burn-in: one key without a branch.
type Unit struct {
	// Point and Replica are the (point, replica) the keys share.
	Point, Replica int
	// Keys are the runs, by variant then continuation.
	Keys []Key
}

// Plan lists the units of the experiment in plan order: point ascending, then
// replica ascending, and within a unit the keys by variant then continuation
// (spec-experiments.md 5.1.1). It validates the experiment first, which checks
// every bound before anything is generated, resolves no document and builds no
// world.
func Plan(e Spec) ([]Unit, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	points, _ := e.pointCount() // Validate has checked it
	units := make([]Unit, 0, points*e.Replicas)
	for p := range points {
		for r := range e.Replicas {
			keys := e.unitKeys(p, r)
			units = append(units, Unit{Point: p, Replica: r, Keys: keys})
		}
	}
	return units, nil
}

// branchShape returns the variants and continuations of a unit: 1 and 1
// without a branch, and with one the number of variants and max(1,
// Continuations).
func (e Spec) branchShape() (variants, conts int) {
	if b := e.Branch; b != nil {
		return len(b.Variants), max(1, b.Continuations)
	}
	return 1, 1
}

// levelCounts returns the level count of every factor: its explicit levels, or
// Steps for a range factor.
func (e Spec) levelCounts() []int {
	out := make([]int, len(e.Factors))
	for i, f := range e.Factors {
		out[i] = len(f.Levels)
		if f.Path != "" {
			out[i] = f.Steps
		}
	}
	return out
}

// pointCount returns the number of points the design generates (spec-experiments.md
// 4.1): the grid's product, one at a time's 1 + sum(L - 1), or DesignN for a
// spread. An experiment with no factor has one point whatever the design says
// (4.1.3). It returns false for an overflow.
func (e Spec) pointCount() (int, bool) {
	if len(e.Factors) == 0 {
		return 1, true
	}
	switch e.Design {
	case OneAtATime:
		return design.OATCount(e.levelCounts())
	case Spread:
		return e.DesignN, e.DesignN >= 1
	}
	return design.GridCount(e.levelCounts())
}

// point returns, for the point p of a validated experiment, the level index of
// every factor (the cell, for a spread) and its value: the value of a range
// factor, else the level index that applies (spec-experiments.md 4.1.2). It
// returns false for a p outside the design.
func (e Spec) point(p int) (levels []int, values []int64, ok bool) {
	if n, good := e.pointCount(); !good || p < 0 || p >= n {
		return nil, nil, false
	}
	counts := e.levelCounts()
	if e.Design == Spread {
		return e.spreadPoint(p, counts)
	}
	switch e.Design {
	case OneAtATime:
		levels, ok = design.OATCoords(p, counts, nil)
	default:
		levels, ok = design.GridCoords(p, counts, nil)
	}
	if !ok {
		return nil, nil, false
	}
	values = make([]int64, len(levels))
	for f, l := range levels {
		values[f] = int64(l)
		if fac := e.Factors[f]; fac.Path != "" {
			if values[f], ok = design.RangeValue(l, fac.Steps, fac.Min, fac.Max); !ok {
				return nil, nil, false
			}
		}
	}
	return levels, values, true
}

// spreadPoint is point for the Spread design: the cell of each axis, and the
// level or range value its midpoint falls in.
func (e Spec) spreadPoint(p int, counts []int) ([]int, []int64, bool) {
	levels := make([]int, len(e.Factors))
	values := make([]int64, len(e.Factors))
	for f, fac := range e.Factors {
		cell, ok := design.SpreadCell(e.DesignSalt, f, e.DesignN, p)
		if !ok {
			return nil, nil, false
		}
		levels[f] = cell
		if fac.Path != "" {
			values[f], ok = design.SpreadValue(cell, e.DesignN, fac.Min, fac.Max)
		} else {
			var l int
			l, ok = design.SpreadLevel(cell, e.DesignN, counts[f])
			values[f] = int64(l)
		}
		if !ok {
			return nil, nil, false
		}
	}
	return levels, values, true
}
