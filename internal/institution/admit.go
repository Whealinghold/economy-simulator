package institution

import (
	"cmp"
	"slices"
)

// Lapsed says why a membership ended at a round (spec-institutions.md 4.2.1).
type Lapsed uint8

// The values of Lapsed. Keep means the membership stands.
const (
	// Keep is a membership that is still valid: the household is on the list, in the
	// scope and in the place it was admitted in.
	Keep Lapsed = iota
	// LapsedList is a household that is no longer on the body's list.
	LapsedList
	// LapsedScope is a household that is out of the body's scope or in a different
	// place from the one it was admitted in.
	LapsedScope
)

// Lapse classifies a membership at a round (FR-ORG-011, spec-institutions.md
// 4.2.1): onList and inScope say whether the household is eligible at its place now,
// and admittedAt and at are the place it was admitted in and the place it is in. The
// list is tested first, so a household that is off the list and out of the scope is
// counted as off the list; a household that is eligible but has moved is
// LapsedScope, whatever the new place is.
func Lapse(onList, inScope bool, admittedAt, at int32) Lapsed {
	switch {
	case !onList:
		return LapsedList
	case !inScope, admittedAt != at:
		return LapsedScope
	}
	return Keep
}

// Candidate is an eligible non-member of one (body, place) at a round. ID is the
// household agent's ID and the caller's reference to it; Weight is its households;
// Since is the tick it took its role; Key is the stateless key of the tick and the
// agent from internal/ordering.
type Candidate struct {
	ID     int32
	Weight int64
	Since  int64
	Key    uint64
}

// Outcome counts, in households (the sum of weights), what a round did to one
// (body, place) (the counts of `MembershipChanged`, spec-institutions.md 11.1).
type Outcome struct {
	// Admitted is the households admitted.
	Admitted int64
	// LapsedList and LapsedScope are the households whose membership lapsed.
	LapsedList, LapsedScope int64
	// Skipped is the candidates that did not fit when their turn came.
	Skipped int64
	// Barred is the candidates of a place whose need is not met: nobody is admitted.
	Barred int64
	// Waiting is the eligible non-members left after the round.
	Waiting int64
}

// Lapsed counts a household of weight w whose membership lapsed for the reason k.
// Keep counts nothing.
func (o *Outcome) Lapsed(k Lapsed, w int64) {
	switch k {
	case LapsedList:
		o.LapsedList += w
	case LapsedScope:
		o.LapsedScope += w
	}
}

// Admit admits candidates into the free places of one (body, place) in the total
// order of FR-ORG-011: by Since ascending, then Key ascending, then ID ascending
// (architecture.md 7.6). A candidate is admitted when its weight is at most the
// free places, and the free places then fall by its weight; a candidate that does not
// fit is counted Skipped and the round goes on, so a smaller one behind it may be
// admitted (the first-fit of FR-ORG-012, a cohort being whole or not at all). With
// limited false there is no limit and every candidate is admitted. With barred true
// (the body's need is not met in the place) nobody is admitted and every candidate is
// counted Barred. free is the headroom of the place before the round, from Headroom.
//
// Admit sorts cands in place, appends the IDs of the admitted candidates, in the
// order admitted, to dst and returns it with the counts of Admitted, Skipped,
// Barred and Waiting (the lapse counts are the caller's). The result is a function of
// the candidates as a set, not of their order in the slice (FR-ORG-019), and a second
// round with the admitted ones now counted in the used places admits nobody, because a
// candidate that did not fit could not fit in fewer places.
func Admit(dst []int32, cands []Candidate, free int64, limited, barred bool) ([]int32, Outcome) {
	var out Outcome
	slices.SortFunc(cands, byOrder)
	if barred {
		for _, c := range cands {
			out.Barred += c.Weight
		}
		out.Waiting = out.Barred
		return dst, out
	}
	for _, c := range cands {
		if limited && c.Weight > free {
			out.Skipped += c.Weight
			out.Waiting += c.Weight
			continue
		}
		free -= c.Weight
		out.Admitted += c.Weight
		dst = append(dst, c.ID)
	}
	return dst, out
}

// byOrder is the total order of admission: Since, then Key, then ID.
func byOrder(a, b Candidate) int {
	return cmp.Or(cmp.Compare(a.Since, b.Since), cmp.Compare(a.Key, b.Key), cmp.Compare(a.ID, b.ID))
}
