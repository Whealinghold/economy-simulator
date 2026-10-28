package endowment

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Request is one agent's demand on one endowment: Executions wanted, each
// drawing Draw units (spec-resources.md 5.3.1). Ref is the agent ID, and the
// slot of the caller's grants slice where the grant is written. Key is
// ordering.Key(tick, agent), the stateless tie-break key of 5.3.3.
type Request struct {
	Ref        int
	Executions int64
	Draw       int64
	Key        uint64

	phi int64 // the remainder of Executions * available over the units wanted
}

// Ration shares the available units among the requests (spec-resources.md 5.3.2)
// and writes the executions granted to grants[req.Ref] for every request.
//
// If the units wanted fit, every request is granted what it asked. Otherwise
// each is granted floor(w * A / U) executions and the units left go, one
// execution at a time, to the requests in the order of descending remainder,
// then ascending Key, then ascending Ref, each that still fits (one pass; what
// remains is not taken). The order is total, so the result does not depend on
// how the requests were listed. Ration sorts reqs in place and allocates
// nothing. The Refs must be distinct slots of grants. ok is false, and every
// grant that has a slot is 0, for a negative number, a Draw below 1, a Ref
// outside grants or a sum of units wanted beyond int64.
func Ration(available int64, reqs []Request, grants []int64) (ok bool) {
	wanted, ok := unitsWanted(available, reqs, len(grants))
	if !ok {
		for _, r := range reqs {
			if r.Ref >= 0 && r.Ref < len(grants) {
				grants[r.Ref] = 0
			}
		}
		return false
	}
	if wanted <= available {
		for _, r := range reqs {
			grants[r.Ref] = r.Executions
		}
		return true
	}
	left := available
	for i := range reqs {
		r := &reqs[i]
		// ok is always true: wanted > available >= 0 and the quotient is at most Executions.
		p, phi, _ := checked.MulDivRem(r.Executions, available, wanted)
		r.phi = phi
		grants[r.Ref] = p
		left -= p * r.Draw // at most p * d <= w * d * A / U, summing to A
	}
	slices.SortFunc(reqs, compareLeftover)
	for _, r := range reqs {
		if grants[r.Ref] < r.Executions && r.Draw <= left {
			grants[r.Ref]++
			left -= r.Draw
		}
	}
	return true
}

// unitsWanted returns the sum of Executions * Draw and whether the input is
// valid (a Ref inside the n slots of grants) and the sum fits.
func unitsWanted(available int64, reqs []Request, n int) (int64, bool) {
	if available < 0 {
		return 0, false
	}
	var total int64
	for _, r := range reqs {
		if r.Executions < 0 || r.Draw < 1 || r.Ref < 0 || r.Ref >= n {
			return 0, false
		}
		u, ok := checked.Mul(r.Executions, r.Draw)
		if !ok {
			return 0, false
		}
		if total, ok = checked.Add(total, u); !ok {
			return 0, false
		}
	}
	return total, true
}

// compareLeftover orders by descending remainder, ascending key, ascending Ref.
func compareLeftover(a, b Request) int {
	return cmp.Or(cmp.Compare(b.phi, a.phi), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Ref, b.Ref))
}
