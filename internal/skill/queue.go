package skill

import (
	"cmp"
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Trainer is a trainer's row for Open: the trainees it holds and the most it
// may hold, the lesser of its role's MaxTrainees and its body's when a body
// limits it (spec-skill-barriers.md 4.4).
type Trainer struct{ Held, Max int64 }

// Open returns the places open to the queue: the sum over the trainers of
// Max - Held, a trainer at or above its maximum counting 0 (an institution may
// lower the maximum under a traineeship in progress, which is not ended). The
// sum saturates at the largest int64 (spec-skill-barriers.md 4.4, FR-SKL-017).
func Open(trainers []Trainer) int64 {
	var n int64
	for _, t := range trainers {
		s, ok := checked.Add(n, max(t.Max-t.Held, 0))
		if !ok {
			return math.MaxInt64
		}
		n = s
	}
	return n
}

// Applicant is a queue entry: the agent (a Ref into the caller's slice or an
// ID), the tick it applied and the ticks it has served.
type Applicant struct {
	Agent, Applied, Served int64
}

// Less reports whether a is ahead of b in a queue of new applicants: the earlier
// application tick first, then the lower agent (spec-skill-barriers.md 4.4,
// FR-SKL-017). It is a total order over distinct agents.
func Less(a, b Applicant) bool {
	return cmp.Or(cmp.Compare(a.Applied, b.Applied), cmp.Compare(a.Agent, b.Agent)) < 0
}

// Requeue reports whether a is ahead of b when a trainer's trainees return to
// the queue: more ticks served first, then the earlier application tick, then
// the lower agent (spec-skill-barriers.md 4.6, ruling R5, FR-SKL-020). It is a
// total order over distinct agents.
func Requeue(a, b Applicant) bool {
	return cmp.Or(cmp.Compare(b.Served, a.Served), cmp.Compare(a.Applied, b.Applied), cmp.Compare(a.Agent, b.Agent)) < 0
}

// Wait returns the expected wait q_w of spec-skill-barriers.md 4.5, the
// applicants ahead times ticks over the places nmax, rounded up (FR-SKL-019).
// ok is false, and q 0, when there is no place (nmax at most 0: no trainer, so
// no application), when ahead or ticks is negative, or when the product
// overflows.
func Wait(ahead, ticks, nmax int64) (q int64, ok bool) {
	if nmax <= 0 || ahead < 0 || ticks < 0 {
		return 0, false
	}
	return checked.MulDivCeil(ahead, ticks, nmax)
}
