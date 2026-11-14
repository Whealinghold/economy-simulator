package technology

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Cell is the saved state of one (region, field) pair (spec-technology.md 3.2).
type Cell struct {
	Experience int64 // executions, 0 to MaxExperience
	Advance    int64 // trend, discovery, contact and call gain, 0 to the cap
	CatchUp    int64 // carried remainder of the diffusion step
}

// Field is the rules of one technology field, as primitives (spec-technology.md
// 3.1). A rate of 0 is a mechanism that is off.
type Field struct {
	CapBP       int64
	Base        int64
	DoublingBP  int64
	TrendBP     int64
	DiscoveryBP int64
	StepBP      int64
	CatchUpBP   int64
	Reach       int64
}

// valid reports whether every rate and bound of the field is non-negative,
// which the configuration's bounds guarantee (spec-technology.md 9.2).
func (f Field) valid() bool {
	return f.CapBP >= 0 && f.Base >= 0 && f.DoublingBP >= 0 && f.TrendBP >= 0 &&
		f.DiscoveryBP >= 0 && f.StepBP >= 0 && f.CatchUpBP >= 0 && f.Reach >= 0
}

// Input is everything one step reads (spec-technology.md 4.6.4). The cells are
// in region-major order, region*len(Fields)+field, and Counters holds the
// executions of the tick in the same order. Dist is indexed by region.
type Input struct {
	Cells         []Cell
	Counters      []int64
	TrendCarry    []int64
	Fields        []Field
	Regions       int
	Tick          int64
	TicksPerYear  int64
	MaxExperience int64
	Key           uint64
	Dist          DistanceFunc
}

// Result is what one cell did in a step: the identity of spec-technology.md
// 4.6.3 and the state it ends in. After = Before + Doing + Trend + Discovery +
// CatchUp - Clamped, with Clamped never negative (FR-TEC-017).
type Result struct {
	Before, After                     int64
	Doing, Trend, Discovery, CatchUp  int64
	Clamped                           int64
	Discoveries                       int64
	Experience, Advance, CatchUpCarry int64
}

// Accrued reports whether anything accrued or was cut in the cell, which is
// when the event of spec-technology.md 11.1 is due.
func (r Result) Accrued() bool {
	return r.Doing != 0 || r.Trend != 0 || r.Discovery != 0 || r.CatchUp != 0 || r.Clamped != 0
}

// Scratch is the working memory of a step, reused between ticks so that the
// step allocates nothing once it is sized (NFR-TEC-003). The zero value is
// ready.
type Scratch struct {
	gain  []int64 // G' of every cell, after the trend and the discoveries
	snap  []int64 // the gains of one field over the regions
	carry []int64 // the trend carry after the step, per field
}

func (s *Scratch) size(cells, regions, fields int) {
	s.gain = grow(s.gain, cells)
	s.snap = grow(s.snap, regions)
	s.carry = grow(s.carry, fields)
}

func grow(b []int64, n int) []int64 {
	if cap(b) < n {
		return make([]int64, n)
	}
	return b[:n]
}

// Step composes the step of spec-technology.md 4.6.1 over every cell: fold the
// counters, the trend, discovery, the snapshot, diffusion, the clamp. It fills
// out, one Result per cell, and on success writes the new state into in.Cells
// and in.TrendCarry; on ok false (a shape mismatch or an overflow, neither of
// which the bounds of the configuration allow) it changes nothing but out and
// the scratch. It takes no random word and reads nothing but its input
// (FR-TEC-033), and the result does not depend on the order in which cells are
// visited (FR-TEC-013, NFR-TEC-002): diffusion reads one snapshot.
func Step(in Input, s *Scratch, out []Result) bool {
	if !shapeOK(in, len(out)) {
		return false
	}
	nf := len(in.Fields)
	s.size(len(out), in.Regions, nf)
	for i := range out {
		out[i] = Result{Experience: in.Cells[i].Experience, Advance: in.Cells[i].Advance, CatchUpCarry: in.Cells[i].CatchUp}
	}
	copy(s.carry, in.TrendCarry)
	if !phases(in, s, out) {
		return false
	}
	copy(in.TrendCarry, s.carry)
	for i := range out {
		in.Cells[i] = Cell{Experience: out[i].Experience, Advance: out[i].Advance, CatchUp: out[i].CatchUpCarry}
	}
	return true
}

// shapeOK checks that the slices of the input agree with the regions and the
// fields and that every field is valid.
func shapeOK(in Input, outLen int) bool {
	nf := len(in.Fields)
	n := in.Regions * nf
	if in.Regions < 0 || len(in.Cells) != n || len(in.Counters) != n ||
		len(in.TrendCarry) != nf || outLen != n || in.TicksPerYear < 1 {
		return false
	}
	if slices.ContainsFunc(in.Fields, func(f Field) bool { return !f.valid() }) {
		return false
	}
	// a field with diffusion on reads the distances, so Dist must be set
	return in.Dist != nil || in.Regions < 2 ||
		!slices.ContainsFunc(in.Fields, func(f Field) bool { return f.CatchUpBP > 0 })
}

// phases runs the six sub-steps of 4.6.1 in order and stops at the first
// refusal.
func phases(in Input, s *Scratch, out []Result) bool {
	return foldAll(in, out) && trendAll(in, s, out) && discoverAll(in, out) &&
		snapshot(in, s, out) && diffuseAll(in, s, out) && finish(in, out)
}

// gainOf is min(CapBP, g(X) + A) for a cell of field f (spec-technology.md
// 3.3.1), with the learned part 0 when learning is off.
func gainOf(f Field, x, a int64) (int64, bool) {
	g, ok := Learned(f.DoublingBP, max(f.Base, 1), x)
	if !ok {
		return 0, false
	}
	sum, ok := checked.Add(g, a)
	if !ok {
		return 0, false
	}
	return min(f.CapBP, sum), true
}

// learnedOf is g(X) alone, the part that doing contributes.
func learnedOf(f Field, x int64) (int64, bool) {
	return Learned(f.DoublingBP, max(f.Base, 1), x)
}

// doingOf is the change of g over a step from x0 to x. g does not fall as X
// grows, so a negative change is a refusal, not a value.
func doingOf(f Field, x0, x int64) (int64, bool) {
	g0, ok0 := learnedOf(f, x0)
	g1, ok1 := learnedOf(f, x)
	if !ok0 || !ok1 {
		return 0, false
	}
	d, ok := checked.Sub(g1, g0)
	return d, ok && d >= 0
}

// foldAll is sub-step 1: X = min(MaxExperience, X + counter), Doing = the
// change of g(X). Before is the gain at the start of the step.
func foldAll(in Input, out []Result) bool {
	nf := len(in.Fields)
	for i := range out {
		f := in.Fields[i%nf]
		c := in.Counters[i]
		if c < 0 {
			return false
		}
		x0 := in.Cells[i].Experience
		if x0 < 0 || x0 > in.MaxExperience || in.Cells[i].Advance < 0 {
			return false
		}
		before, ok := gainOf(f, x0, in.Cells[i].Advance)
		if !ok {
			return false
		}
		x := in.MaxExperience
		if c < in.MaxExperience-x0 {
			x = x0 + c
		}
		doing, ok := doingOf(f, x0, x)
		if !ok {
			return false
		}
		out[i].Before = before
		out[i].Experience = x
		out[i].Doing = doing
	}
	return true
}

// trendAll is sub-step 2: per field in ascending order the carry of 4.3.1,
// and the add to every cell of the field.
func trendAll(in Input, s *Scratch, out []Result) bool {
	nf := len(in.Fields)
	for f, fd := range in.Fields {
		if fd.TrendBP == 0 {
			continue
		}
		add, carry, ok := TrendStep(s.carry[f], fd.TrendBP, in.TicksPerYear)
		if !ok {
			return false
		}
		s.carry[f] = carry
		for i := f; i < len(out); i += nf {
			a, ok := checked.Add(out[i].Advance, add)
			if !ok {
				return false
			}
			out[i].Advance, out[i].Trend = a, add
		}
	}
	return true
}

// discoverAll is sub-step 3: a cell that practises (its counter is above 0) in
// a field with DiscoveryBP above 0 discovers when Hit says so, in cell order.
func discoverAll(in Input, out []Result) bool {
	nf := len(in.Fields)
	cells := int64(len(out))
	for i := range out {
		f := in.Fields[i%nf]
		if f.DiscoveryBP == 0 || in.Counters[i] <= 0 {
			continue
		}
		if !Hit(in.Key, in.Tick, int64(i), cells, f.DiscoveryBP, in.TicksPerYear) {
			continue
		}
		a, ok := checked.Add(out[i].Advance, f.StepBP)
		if !ok {
			return false
		}
		out[i].Advance = a
		out[i].Discovery = f.StepBP
		out[i].Discoveries = 1
	}
	return true
}

// snapshot is sub-step 4: G' of every cell, from the experience after the fold
// and the advance after the trend and the discoveries.
func snapshot(in Input, s *Scratch, out []Result) bool {
	nf := len(in.Fields)
	for i := range out {
		g, ok := gainOf(in.Fields[i%nf], out[i].Experience, out[i].Advance)
		if !ok {
			return false
		}
		s.gain[i] = g
	}
	return true
}

// diffuseAll is sub-step 5: per field with CatchUpBP above 0, the frontier of
// every cell from the snapshot, then the catch-up. Every frontier is read
// before any catch-up is written, and the catch-ups write only the cell's own
// advance and carry, so the order of the cells does not matter.
func diffuseAll(in Input, s *Scratch, out []Result) bool {
	nf := len(in.Fields)
	for f, fd := range in.Fields {
		if fd.CatchUpBP == 0 {
			continue
		}
		for r := 0; r < in.Regions; r++ {
			s.snap[r] = s.gain[r*nf+f]
		}
		for r := 0; r < in.Regions; r++ {
			if !catchUpCell(in, s, out, fd, r*nf+f, r) {
				return false
			}
		}
	}
	return true
}

func catchUpCell(in Input, s *Scratch, out []Result, fd Field, i, r int) bool {
	front := Frontier(r, s.snap, fd.Reach, in.Dist)
	add, carry, ok := CatchUp(out[i].CatchUpCarry, front-s.gain[i], fd.CatchUpBP, in.TicksPerYear)
	if !ok {
		return false
	}
	a, ok := checked.Add(out[i].Advance, add)
	if !ok {
		return false
	}
	out[i].Advance, out[i].CatchUp, out[i].CatchUpCarry = a, add, carry
	return true
}

// finish is sub-step 6: A = min(CapBP, A), G = min(CapBP, g(X) + A), and the
// part the cap cut, defined by the identity (4.6.3). A Clamped below 0 would
// break the rules and is refused.
func finish(in Input, out []Result) bool {
	nf := len(in.Fields)
	for i := range out {
		f := in.Fields[i%nf]
		a := min(f.CapBP, out[i].Advance)
		g, ok := gainOf(f, out[i].Experience, a)
		if !ok {
			return false
		}
		r := &out[i]
		r.Advance, r.After = a, g
		r.Clamped, ok = clampedOf(*r)
		if !ok {
			return false
		}
	}
	return true
}

// clampedOf is Before + Doing + Trend + Discovery + CatchUp - After, the part
// the cap cut, with every sum checked. ok is false on an overflow or a
// negative result, which the rules never produce.
func clampedOf(r Result) (int64, bool) {
	sum := r.Before
	for _, v := range [...]int64{r.Doing, r.Trend, r.Discovery, r.CatchUp} {
		var ok bool
		if sum, ok = checked.Add(sum, v); !ok {
			return 0, false
		}
	}
	c, ok := checked.Sub(sum, r.After)
	return c, ok && c >= 0
}
