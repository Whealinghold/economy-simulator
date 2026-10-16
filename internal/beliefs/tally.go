package beliefs

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Tally accumulates the counts of BeliefsUpdated for one market in one tick
// (spec-beliefs.md 12.1). Every count is read from the Change of an outcome, so
// a strategy that ignores the default rule is counted the same way. The zero
// value is an empty tally. A belief whose Floor equals its Ceiling counts in
// both AtFloor and AtCeiling (12.1.2).
type Tally struct {
	Outcomes  int64 // outcomes processed
	Adjusted  int64 // gates open
	Raised    int64 // centres above the stored centre
	Lowered   int64 // centres below the stored centre
	AtFloor   int64 // centres on the floor
	AtCeiling int64 // centres on the ceiling
	Conflicts int64 // outcomes whose bounds have the conflict flag
	Clamped   int64 // beliefs the clamp changed

	sum       checked.Acc
	low, high int64
}

// Add counts one outcome.
func (t *Tally) Add(c Change) {
	t.Outcomes++
	t.Adjusted += b2i(c.Open)
	t.Raised += b2i(c.Raised())
	t.Lowered += b2i(c.Lowered())
	t.AtFloor += b2i(c.AtFloor())
	t.AtCeiling += b2i(c.AtCeiling())
	t.Conflicts += b2i(c.Conflict())
	t.Clamped += b2i(c.Clamped)
	centre := c.After.Centre
	if t.Outcomes == 1 {
		t.low, t.high = centre, centre
	}
	t.low, t.high = min(t.low, centre), max(t.high, centre)
	t.sum.AddMul(max(centre, 0), 1) // 128 bits: cannot overflow in practice
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// Mean returns the mean centre after the outcomes, rounded half to even, and
// false for an empty tally, which the caller does not emit.
func (t *Tally) Mean() (int64, bool) {
	if t.Outcomes == 0 {
		return 0, false
	}
	return t.sum.DivHalfEven(t.Outcomes)
}

// Low returns the lowest centre after the outcomes, 0 for an empty tally.
func (t *Tally) Low() int64 { return t.low }

// High returns the highest centre after the outcomes, 0 for an empty tally.
func (t *Tally) High() int64 { return t.high }

// Run is the diagnostic run of the Warn "price beliefs at bound" for one market
// after one tick (spec-beliefs.md 12.2).
type Run struct {
	Count    int64 // consecutive ticks on a bound, the BoundRun to keep
	Warn     bool  // emit the Warn this tick
	Ceiling  bool  // the bound to name is the ceiling, not the floor
	Agents   int64 // the number of beliefs at that bound
	Outcomes int64 // the outcomes of the tick, the denominator of the test
}

// NextRun advances the run of a market by one tick from the tick's tally and
// the run before. A tick is pinned when more than half of the outcomes ended on
// a bound, AtFloor + AtCeiling > Outcomes / 2 (integer division), so a tick with
// no outcome is not pinned and resets the run. It warns at every every-th
// consecutive pinned tick, naming the bound with more beliefs and the floor on
// a tie. Call it once per market per tick, also when the market had no outcome.
func NextRun(before int64, t *Tally, every int64) Run {
	if t.Outcomes == 0 || t.AtFloor+t.AtCeiling <= t.Outcomes/2 {
		return Run{}
	}
	r := Run{Count: before, Outcomes: t.Outcomes}
	if r.Count < math.MaxInt64 {
		r.Count++
	}
	r.Ceiling = t.AtCeiling > t.AtFloor
	r.Agents = t.AtFloor
	if r.Ceiling {
		r.Agents = t.AtCeiling
	}
	r.Warn = every > 0 && r.Count%every == 0
	return r
}
