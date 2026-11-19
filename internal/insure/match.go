package insure

// Candidate is one policy a request asks for: a shipment of a lane request or the one
// store of a store request, with its sum insured computed once before any offer is
// scanned (spec-insurance.md 7.4). The request's own fields travel with it, so the
// matching reads one list in request order and, inside a request, candidate order.
type Candidate struct {
	// Ref is the caller's index of the candidate, returned in a Fill.
	Ref int
	// Party is the insured's ID, never matched with its own offer.
	Party int64
	// Limit is the request's highest loading or store rate, in basis points.
	Limit int64
	// MinCover is the request's lowest cover share, in basis points.
	MinCover int64
	// Sum is the candidate's sum insured.
	Sum int64
}

// Fill is a tentative fill: the Ref of the candidate and of the offer and the pair's
// maximum payout, halfEven(offer cover * sum / 10 000) (spec-insurance.md 7.3.2).
type Fill struct {
	// Candidate is the Ref of the candidate.
	Candidate int
	// Offer is the Ref of the offer.
	Offer int
	// MaxPayout is the pair's maximum payout.
	MaxPayout int64
}

// Outcome is what the caller's formation answers for a tentative fill
// (spec-insurance.md 7.4).
type Outcome int

const (
	// Formed says the policy was formed: the offer's room falls by the maximum payout
	// and the candidate is done.
	Formed Outcome = iota
	// Failed says formation failed for a reason that is the candidate's own (funds,
	// below the minimum): the candidate is counted and its scan stops.
	Failed
	// Dropped says formation failed for a reason that is the underwriter's (the
	// overflow of its balance): the fill is dropped and the scan goes on.
	Dropped
	// Halted says formation failed for a reason that ends the lane (too many
	// policies): no later fill can form either.
	Halted
)

// Match runs the matching of one lane over candidates in request order and offers
// already ordered by SortLoadings (lane requests) or SortStoreRates (store requests),
// the order that matches the Limit the candidates carry: store says which price an
// offer is compared on (spec-insurance.md 7.3.2, FR-INS-034). For each candidate it
// goes through the offers in order: it skips the candidate's own offer, an offer
// whose cover share is below the candidate's lowest, and one whose remaining room is
// below the pair's maximum payout (not any other offer's); it ends the scan at the
// first offer whose price exceeds the candidate's limit (no underwriter at the
// price); otherwise it passes the fill to form. Formed lowers the offer's Room by the
// maximum payout and ends the scan, Failed ends it, Dropped goes on, Halted ends the
// whole lane. It reports whether the lane was halted. Each policy is at its offer's own
// terms (discriminatory, spec-insurance.md 7.3.3). Match allocates nothing.
func Match(candidates []Candidate, offers []Offer, store bool, form func(Fill) Outcome) (halted bool) {
	for i := range candidates {
		if fillCandidate(&candidates[i], offers, store, form) {
			return true
		}
	}
	return false
}

// price is the price an offer is compared on.
func price(o *Offer, store bool) int64 {
	if store {
		return o.StoreRate
	}
	return o.Loading
}

// eligible returns the pair's maximum payout and whether the offer is a candidate
// for the candidate: not its own, its cover share at least the lowest asked, and room
// for the pair's maximum payout (spec-insurance.md 7.3.2 step 1).
func eligible(c *Candidate, o *Offer) (maxPayout int64, ok bool) {
	if o.Party == c.Party || o.CoverBP < c.MinCover {
		return 0, false
	}
	mp, ok := MaxPayout(o.CoverBP, c.Sum)
	return mp, ok && o.Room >= mp
}

// fillCandidate matches one candidate against the ordered offers and reports whether
// the lane was halted.
func fillCandidate(c *Candidate, offers []Offer, store bool, form func(Fill) Outcome) (halted bool) {
	for j := range offers {
		o := &offers[j]
		if price(o, store) > c.Limit {
			// Ascending prices: no later offer is cheaper, so skipping first
			// (spec-insurance.md 7.3.2 step 1) cannot change the result.
			return false
		}
		mp, ok := eligible(c, o)
		if !ok {
			continue
		}
		switch form(Fill{Candidate: c.Ref, Offer: o.Ref, MaxPayout: mp}) {
		case Halted:
			return true
		case Formed:
			o.Room -= mp
			return false
		case Failed:
			return false
		case Dropped:
		}
	}
	return false
}
