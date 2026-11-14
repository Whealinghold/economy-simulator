package credit

import (
	"cmp"
	"slices"
)

// Row is one admitted request or offer of a round, in the primitive form the
// matching reads (spec-credit.md 6.3). The caller sets Ref to the index of the
// request or offer in its own slice; the package returns it and reads nothing else
// of it.
type Row struct {
	// Ref is the caller's index of the row, returned in a Fill.
	Ref int
	// Agent is the agent ID: the last tie-break, and the guard that an offer is
	// never matched with the request of its own agent.
	Agent int64
	// Principal is the principal still unmatched: a request's remainder to fill, an
	// offer's remainder to lend. Match lowers it in place.
	Principal int64
	// Term is a request's term or an offer's longest term, in ticks.
	Term int64
	// RateBP is a request's highest rate, or an offer's effective rate, in basis
	// points.
	RateBP int64
	// Key is the stateless tie-break key K(t, agent), smaller first.
	Key uint64
}

// SortOffers orders offers by effective rate ascending, equal rates by Key
// ascending, then by Agent (spec-credit.md 6.3.1, FR-CRD-022).
func SortOffers(offers []Row) {
	slices.SortFunc(offers, func(a, b Row) int {
		return cmp.Or(cmp.Compare(a.RateBP, b.RateBP), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Agent, b.Agent))
	})
}

// SortRequests orders requests by highest rate descending, equal limits by Key
// ascending, then by Agent (spec-credit.md 6.3.1, FR-CRD-022).
func SortRequests(requests []Row) {
	slices.SortFunc(requests, func(a, b Row) int {
		return cmp.Or(cmp.Compare(b.RateBP, a.RateBP), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Agent, b.Agent))
	})
}

// Fill is a tentative fill of the matching: the Ref of the request and of the
// offer, the principal x, the lender's effective rate and the request's term.
type Fill struct {
	// Request is the Ref of the request row.
	Request int
	// Offer is the Ref of the offer row.
	Offer int
	// Principal is x = min of the two remainders.
	Principal int64
	// RateBP is the offer's effective rate: the rate the loan is formed at.
	RateBP int64
	// Term is the request's term.
	Term int64
}

// Outcome is what the caller's formation answers for a tentative fill
// (spec-credit.md 6.4).
type Outcome int

const (
	// Formed says the loan was made: both remainders fall by the principal.
	Formed Outcome = iota
	// Dropped says formation failed with a reason that leaves the round going (an
	// overflow): the fill is dropped and the scan goes on.
	Dropped
	// Halted says formation failed with a reason that ends the round (the claims
	// limit is reached): no later fill can form either.
	Halted
)

// Match runs the matching of one lane over requests and offers already ordered by
// SortRequests and SortOffers (spec-credit.md 6.3.2, FR-CRD-023, FR-CRD-024). For
// each request in order, with p its remaining principal, it goes through the
// offers in order: it skips an offer of the request's own agent, one whose Term is
// below the request's or one with nothing left; it ends the scan at the first offer
// whose rate exceeds the request's limit (no lender at the price); it takes
// x = min(p, q) and, when x is below minPrincipal, ends the request if x = p and
// skips the offer otherwise; and it passes the fill to form, lowering both
// remainders by x when form answers Formed, dropping the fill on Dropped and
// ending the whole round on Halted. Each fill is at its offer's own rate
// (discriminatory, 6.3.3). The Principal of every row is left at its remainder,
// so the caller reads the unfilled requests and the unspent offers from the rows.
// It reports whether the round was halted. Match allocates nothing.
func Match(requests, offers []Row, minPrincipal int64, form func(Fill) Outcome) (halted bool) {
	for i := range requests {
		if fillRequest(&requests[i], offers, minPrincipal, form) {
			return true
		}
	}
	return false
}

// step is what the matching does with one offer for one request.
type step int

const (
	stepSkip step = iota // pass over the offer
	stepEnd              // end the request: no lender at the price, or its remainder is too small
	stepFill             // take a fill
)

// decide returns what the matching does with offer o for request r and, for a
// fill, its principal x = min(p, q) (spec-credit.md 6.3.2 steps 1 to 3).
func decide(r, o *Row, minPrincipal int64) (step, int64) {
	switch {
	case o.Agent == r.Agent || o.Term < r.Term || o.Principal <= 0:
		return stepSkip, 0
	case o.RateBP > r.RateBP:
		return stepEnd, 0 // ascending rates: no lender at the price
	}
	x := min(r.Principal, o.Principal)
	switch {
	case x >= minPrincipal:
		return stepFill, x
	case x == r.Principal:
		return stepEnd, 0 // the remainder of the request is too small
	}
	return stepSkip, 0 // the remainder of the offer is too small
}

// fillRequest matches one request against the ordered offers and reports whether
// the round was halted.
func fillRequest(r *Row, offers []Row, minPrincipal int64, form func(Fill) Outcome) (halted bool) {
	for j := range offers {
		if r.Principal <= 0 {
			return false
		}
		o := &offers[j]
		what, x := decide(r, o, minPrincipal)
		if what == stepEnd {
			return false
		}
		if what == stepSkip {
			continue
		}
		if out := form(Fill{Request: r.Ref, Offer: o.Ref, Principal: x, RateBP: o.RateBP, Term: r.Term}); out == Halted {
			return true
		} else if out == Formed {
			r.Principal -= x
			o.Principal -= x
		}
	}
	return false
}
