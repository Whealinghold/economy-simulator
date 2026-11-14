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
// ascending, then by Agent (spec-credit.md 6.3.1, FR-CRD-022). Rows equal in all
// three, which the round never builds, end in Ref order so the order is total
// (architecture.md 7.6).
func SortOffers(offers []Row) {
	slices.SortFunc(offers, func(a, b Row) int {
		return cmp.Or(cmp.Compare(a.RateBP, b.RateBP), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Agent, b.Agent), cmp.Compare(a.Ref, b.Ref))
	})
}

// SortRequests orders requests by highest rate descending, equal limits by Key
// ascending, then by Agent (spec-credit.md 6.3.1, FR-CRD-022), and by Ref as
// SortOffers does.
func SortRequests(requests []Row) {
	slices.SortFunc(requests, func(a, b Row) int {
		return cmp.Or(cmp.Compare(b.RateBP, a.RateBP), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Agent, b.Agent), cmp.Compare(a.Ref, b.Ref))
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
// x = min(p, q) and, when x is below minPrincipal, skips the offer (when x = p the
// request's remainder is too small for every later offer too, which is spec-credit.md
// 6.3.2's end of the request); and it passes the fill to form, lowering both
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
	stepEnd              // end the request: no lender at the price
	stepFill             // take a fill
)

// verdict is a step and, for a fill, its principal.
type verdict struct {
	what step
	x    int64
}

// decide returns what the matching does with offer o for request r
// (spec-credit.md 6.3.2 steps 1 to 3): skip it, end the request, or fill
// x = min(p, q). A fill below minPrincipal is skipped: when x = p the remainder of
// the request is below the minimum for every later offer too, so skipping to the end
// of the scan is the same as ending the request, and when x = q the offer is skipped.
func decide(r, o *Row, minPrincipal int64) verdict {
	switch {
	case o.Agent == r.Agent || o.Term < r.Term || o.Principal <= 0:
		return verdict{what: stepSkip}
	case o.RateBP > r.RateBP:
		return verdict{what: stepEnd} // ascending rates: no lender at the price
	}
	if x := min(r.Principal, o.Principal); x >= minPrincipal {
		return verdict{what: stepFill, x: x}
	}
	return verdict{what: stepSkip}
}

// fillRequest matches one request against the ordered offers and reports whether
// the round was halted.
func fillRequest(r *Row, offers []Row, minPrincipal int64, form func(Fill) Outcome) (halted bool) {
	for j := range offers {
		if r.Principal <= 0 {
			return false
		}
		o := &offers[j]
		v := decide(r, o, minPrincipal)
		switch v.what {
		case stepEnd:
			return false
		case stepSkip:
			continue
		case stepFill:
		}
		switch form(Fill{Request: r.Ref, Offer: o.Ref, Principal: v.x, RateBP: o.RateBP, Term: r.Term}) {
		case Halted:
			return true
		case Formed:
			r.Principal -= v.x
			o.Principal -= v.x
		case Dropped:
		}
	}
	return false
}
