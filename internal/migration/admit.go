package migration

import (
	"cmp"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Agent is a pushed household with a ranking, as admission sees it
// (spec-migration.md 7). Ref is the caller's reference, returned in the Result.
// ID is the agent ID, the tie-break of the order of need. Origin indexes the
// household's region in Caps.Out. Bound says the guard refused it (7.2). HoldTicks
// is the longest journey, in ticks, that its stock provisions: the least over
// the upkeep commodities of floor(held / (weight * per-household quantity)),
// which is "holds at least w * u * k of each" (7.4); the root passes MaxInt64
// when the upkeep is empty. Ranked is the ranking of Rank, never empty (an empty
// one is returned as NoDestination).
type Agent struct {
	Ref       int
	ID        int64
	Weight    int64
	EarnedBP  int64
	Origin    int
	Bound     bool
	HoldTicks int64
	Ranked    []Ranked
}

// Caps are the limits of one pass, indexed by the caller's region index, which
// is the Origin of an Agent and the Ref of a Ranked entry. Out[i] is the outflow
// cap of region i (7.3). In[i] is its pipeline cap (7.5) and Pipeline[i] the
// households already on a journey to it. The slices are read only.
type Caps struct {
	Out, In, Pipeline []int64
}

// Result is the end of one pushed household in the pass. Outcome is Started
// with Dest and K when it was admitted, else the reason it was not.
type Result struct {
	Ref     int
	Outcome Outcome
	Dest    int
	K       int64
}

// Scratch holds the buffers of Admit so that a pass allocates nothing once they
// have grown. The zero value is ready.
type Scratch struct {
	order   []int
	out, in []int64
}

// CapOf returns ceil(households * shareBP / 10 000), the outflow or pipeline cap
// of a region (spec-migration.md 7.3, 7.5). ok is false on overflow.
func CapOf(households, shareBP int64) (int64, bool) {
	return checked.MulDivCeil(households, shareBP, wholeBP)
}

// Hold returns floor(held / (weight * perHousehold)), the ticks of one upkeep
// commodity an agent holds (7.4), and ok false when the divisor overflows or is
// below 1 (a commodity that is not upkeep needs no provisioning, which the caller
// leaves out).
func Hold(held, weight, perHousehold int64) (ticks int64, ok bool) {
	need, ok := checked.Mul(weight, perHousehold)
	if !ok || need < 1 {
		return 0, false
	}
	return max(held, 0) / need, true
}

// Admit considers the agents in the order of need, the earned ratio ascending
// and then the agent ID (7.1), and appends one Result per agent to dst[:0], in
// that order. For each agent the tests are, in this fixed order: the guard, the
// outflow cap of its origin, then for each candidate in its ranking the
// provisioning and the pipeline cap; the first candidate that passes both is the
// destination (7.6). A cap admits while the running count is below it, so an
// agent that makes the count reach or pass the cap is admitted and one considered
// after it is refused; the count grows by the agent's weight. The running outflow
// starts at 0 and the running pipeline at caps.Pipeline, and both include the
// agents admitted earlier in the pass. The outcome of an agent that was not
// admitted is Bound, OutCap, Unprovisioned (every candidate failed provisioning)
// or InCap (at least one passed it and every such candidate was full). It draws
// nothing and keeps nothing of the caller's slices.
func Admit(dst []Result, agents []Agent, caps Caps, s *Scratch) []Result {
	s.order = s.order[:0]
	for i := range agents {
		s.order = append(s.order, i)
	}
	slices.SortFunc(s.order, func(a, b int) int {
		x, y := &agents[a], &agents[b]
		return cmp.Or(cmp.Compare(x.EarnedBP, y.EarnedBP), cmp.Compare(x.ID, y.ID), cmp.Compare(x.Ref, y.Ref))
	})
	s.out = slices.Grow(s.out[:0], len(caps.Out))[:len(caps.Out)]
	clear(s.out)
	s.in = append(s.in[:0], caps.Pipeline...)
	dst = dst[:0]
	for _, i := range s.order {
		dst = append(dst, s.consider(&agents[i], caps))
	}
	return dst
}

// consider runs the tests of Admit for one agent against the running counts.
func (s *Scratch) consider(a *Agent, caps Caps) Result {
	r := Result{Ref: a.Ref}
	switch {
	case len(a.Ranked) == 0:
		r.Outcome = NoDestination
		return r
	case a.Bound:
		r.Outcome = Bound
		return r
	case s.out[a.Origin] >= caps.Out[a.Origin]:
		r.Outcome = OutCap
		return r
	}
	provisioned := false
	for _, c := range a.Ranked {
		if c.K > a.HoldTicks {
			continue
		}
		provisioned = true
		if s.in[c.Ref] >= caps.In[c.Ref] {
			continue
		}
		s.out[a.Origin] = addSat(s.out[a.Origin], a.Weight)
		s.in[c.Ref] = addSat(s.in[c.Ref], a.Weight)
		r.Outcome, r.Dest, r.K = Started, c.Ref, c.K
		return r
	}
	r.Outcome = Unprovisioned
	if provisioned {
		r.Outcome = InCap
	}
	return r
}

// addSat adds two non-negative counts, saturating at MaxInt64: a running count
// past int64 is far past every cap, so the saturated count refuses the same.
func addSat(a, b int64) int64 {
	if s, ok := checked.Add(a, b); ok {
		return s
	}
	return math.MaxInt64
}
