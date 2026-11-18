package employment

// Reasons the round itself assigns to a vacancy that ends with positions left.
// A reason returned by a Check is any value from ReasonCheck up, which the root
// maps to its constants; ReasonNone passes the check.
const (
	ReasonNone = 0
	// ReasonTooMany closes the lane: the hires open in the region reached
	// MaxOpen (spec-employment.md 5.5 check 1).
	ReasonTooMany = 1
	// ReasonNoLabour is the reason of a vacancy that stayed open to the end of
	// the round with positions left: no offer reached it (15.2).
	ReasonNoLabour = 2
	// ReasonCheck is the first value a Check may return as a reason.
	ReasonCheck = 3
)

// Close is what a failed formation check closes (spec-employment.md 5.5). Only
// the round's own check 1 closes the lane, so a Check cannot.
type Close uint8

const (
	// CloseFit lowers the vacancy's fit to the block less one: checks 4 to 8
	// and 10 to 12 depend on the size of the block.
	CloseFit Close = iota
	// CloseVacancy closes the vacancy: checks 2, 3 and 9 do not depend on the
	// size of the block.
	CloseVacancy
)

// Hire is one hire the round formed, in formation order: the offer and the
// vacancy it joins (by Ref), the households hired, and the wage, with the
// reservation and the bid that fixed it.
type Hire struct {
	Offer, Vacancy int
	Households     int64
	Wage           int64
	Reserve, Bid   int64
}

// Check runs the formation checks of the world (spec-employment.md 5.5) for the
// pair of an offer and a vacancy at a wage, in the order of the spec, and
// returns the first failure: a reason of at least ReasonCheck and what it
// closes. It returns ReasonNone when all pass. formed is every hire of the lane
// formed so far, in order, which is what the executions already committed to
// an employer are counted from. The round itself runs check 1 and the wage
// test.
type Check func(o Offer, v Vacancy, wage int64, formed []Hire) (reason int, closes Close)

// Lane is the labour lane of one region: Open is the hires open in the region
// when the round starts and MaxOpen the cap per region (13). MaxOpen is per
// region and not per world, so no lane is first come against another.
type Lane struct {
	Open, MaxOpen int64
}

// OfferResult is the outcome of one offer: Hired is true when it formed a hire.
// An offer that was not hired is unmatched (NoVacancy).
type OfferResult struct {
	Ref        int
	Households int64
	Hired      bool
}

// VacancyResult is the outcome of one vacancy at the end of the round: the
// households it took, the positions left and the reason that closed it. Reason
// is ReasonNone when nothing is left, ReasonTooMany when the lane closed,
// ReasonNoLabour when the vacancy stayed open and no offer reached it, and the
// Check's reason when its fit fell to 0 or a check closed it. The positions
// left are counted once.
type VacancyResult struct {
	Ref    int
	Filled int64
	Left   int64
	Reason int
	fit    int64
	closed bool
}

// Result is the outcome of a round of one lane. WageBelowUnit counts the pairs
// skipped because the wage is below 1 (5.4.1). Offers and Vacancies are in the
// sorted order the round worked in.
type Result struct {
	Hires         []Hire
	Offers        []OfferResult
	Vacancies     []VacancyResult
	WageBelowUnit int64
}

// reset empties the result and keeps the capacity of its slices.
func (r *Result) reset() {
	r.Hires = r.Hires[:0]
	r.Offers = r.Offers[:0]
	r.Vacancies = r.Vacancies[:0]
	r.WageBelowUnit = 0
}

// Round runs the matching of one lane (spec-employment.md 5.4, 5.5): it sorts
// offers and vacancies in place (SortOffers, SortVacancies), then takes each
// offer in order and offers it the first open vacancy for which Bid is at
// least Reserve, the positions left cover the block, the block is within the
// vacancy's fit and the wage is at least 1. The pair then meets check 1 (the
// hires open in the lane against MaxOpen) and then the caller's Check. A pass
// forms the hire; a failure closes what its kind says and the offer goes on to
// the next vacancy. An offer is indivisible and a vacancy is not (5.4.4).
//
// The round draws nothing: the same inputs give the same result whatever the
// order they were passed in, provided the Refs are distinct. res is reused, so
// a caller that keeps a result across rounds copies it. A nil check passes
// every pair.
//
// Inputs are admitted by the root: Positions at least 0, ShareBP from 0 to
// 10 000, Reserve at least 0, and a Check reason of ReasonNone or at least
// ReasonCheck. The round does not re-validate them: a pair whose Wage is not
// defined is counted in WageBelowUnit with the pairs whose wage is below 1.
func Round(res *Result, offers []Offer, vacancies []Vacancy, lane Lane, check Check) {
	res.reset()
	SortOffers(offers)
	SortVacancies(vacancies)
	for _, v := range vacancies {
		res.Vacancies = append(res.Vacancies, VacancyResult{Ref: v.Ref, Left: v.Positions, fit: v.Positions})
	}
	closed := false
	for _, o := range offers {
		hired := false
		if !closed {
			var lane2 bool
			hired, lane2 = place(res, o, vacancies, lane, check)
			closed = lane2
		}
		res.Offers = append(res.Offers, OfferResult{Ref: o.Ref, Households: o.Households, Hired: hired})
	}
	for i := range res.Vacancies {
		finish(&res.Vacancies[i], closed)
	}
}

// place offers o the vacancies in order and reports whether a hire formed and
// whether the lane closed.
func place(res *Result, o Offer, vacancies []Vacancy, lane Lane, check Check) (hired, laneClosed bool) {
	for i, v := range vacancies {
		if v.Bid < o.Reserve {
			return false, false // the bids only fall from here
		}
		s := &res.Vacancies[i]
		if !s.admits(o.Households) {
			continue
		}
		wage, ok := Wage(o.Reserve, v.Bid, v.ShareBP)
		if !ok || wage < 1 {
			res.WageBelowUnit++
			continue
		}
		if lane.Open+int64(len(res.Hires)) >= lane.MaxOpen {
			return false, true
		}
		reason, closes := ReasonNone, CloseFit
		if check != nil {
			reason, closes = check(o, v, wage, res.Hires)
		}
		if reason == ReasonNone {
			res.Hires = append(res.Hires, Hire{Offer: o.Ref, Vacancy: v.Ref, Households: o.Households, Wage: wage, Reserve: o.Reserve, Bid: v.Bid})
			s.Filled += o.Households
			s.Left -= o.Households
			return true, false
		}
		fail(s, o.Households, reason, closes)
	}
	return false, false
}

// admits reports whether the vacancy is open and the block fits both the
// positions left and the fit.
func (s *VacancyResult) admits(block int64) bool {
	return !s.closed && s.Left >= block && block <= s.fit
}

// fail applies a failed check to a vacancy: a vacancy-wide kind closes it, a
// block-dependent one lowers the fit to the block less one and closes the
// vacancy when the fit reaches 0.
func fail(s *VacancyResult, block int64, reason int, closes Close) {
	switch closes {
	case CloseVacancy:
		s.closed, s.Reason = true, reason
	default:
		s.fit = min(s.fit, block-1)
		if s.fit <= 0 {
			s.closed, s.Reason = true, reason
		}
	}
}

// finish gives a vacancy its closing reason at the end of the round: a lane
// that closed marks every vacancy not closed before it, a vacancy that stayed
// open with positions left is ReasonNoLabour, and one with none is ReasonNone.
func finish(s *VacancyResult, laneClosed bool) {
	switch {
	case s.Left == 0:
		s.Reason = ReasonNone
	case s.closed:
	case laneClosed:
		s.Reason = ReasonTooMany
	default:
		s.Reason = ReasonNoLabour
	}
}
