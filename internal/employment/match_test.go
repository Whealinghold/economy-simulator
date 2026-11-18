package employment_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/employment"
)

// open is a lane with room for every hire the tests form.
var open = employment.Lane{MaxOpen: 1000}

// refs returns the (offer, vacancy) pairs of the hires of a result.
func refs(r *employment.Result) [][2]int {
	var out [][2]int
	for _, h := range r.Hires {
		out = append(out, [2]int{h.Offer, h.Vacancy})
	}
	return out
}

// TestRound_FirstOpenVacancyAtOrAboveReserve verifies first fit: each offer, in
// order, takes the first open vacancy, in order, whose bid is at least its
// reservation, and the wage is the share of the margin.
//
// Requirement: FR-EMP-020
func TestRound_FirstOpenVacancyAtOrAboveReserve(t *testing.T) {
	t.Parallel()
	// Offers by reserve: Ref 1 at 2, Ref 2 at 5. Vacancies by bid: Ref 10 at 9
	// (1 position), Ref 11 at 6 (1 position), Ref 12 at 3.
	// Offer 1 takes the best vacancy, 10: wage = 2 + 5 000 * (9 - 2) / 10 000 = 2 + 3.5 -> 2 + 4 = 6.
	// Offer 2 finds 10 full and takes 11: wage = 5 + 5 000 * (6 - 5) / 10 000 = 5 + 0.5 -> 5 + 0 = 5.
	offers := []employment.Offer{{Ref: 2, Households: 1, Reserve: 5}, {Ref: 1, Households: 1, Reserve: 2}}
	vacancies := []employment.Vacancy{
		{Ref: 12, Positions: 1, Bid: 3, Duration: 1, ShareBP: 5000},
		{Ref: 10, Positions: 1, Bid: 9, Duration: 1, ShareBP: 5000},
		{Ref: 11, Positions: 1, Bid: 6, Duration: 1, ShareBP: 5000},
	}
	var r employment.Result
	employment.Round(&r, offers, vacancies, open, nil)
	if want := [][2]int{{1, 10}, {2, 11}}; !reflect.DeepEqual(refs(&r), want) {
		t.Fatalf("Round hires (offer, vacancy) = %v, want %v", refs(&r), want)
	}
	if got := []int64{r.Hires[0].Wage, r.Hires[1].Wage}; !slices.Equal(got, []int64{6, 5}) {
		t.Errorf("Round wages = %v, want [6 5]", got)
	}
	for _, h := range r.Hires {
		if h.Wage < h.Reserve || h.Wage > h.Bid {
			t.Errorf("hire %+v: wage outside [reserve, bid]", h)
		}
	}
}

// TestRound_OfferIndivisibleVacancyDivisible verifies that an offer is placed
// whole and a vacancy is shared: an offer of 3 skips a vacancy with 2 positions
// left, and a vacancy of 5 takes two offers of 2 and keeps 1.
//
// Requirement: FR-EMP-020
func TestRound_OfferIndivisibleVacancyDivisible(t *testing.T) {
	t.Parallel()
	offers := []employment.Offer{
		{Ref: 1, Households: 3, Reserve: 1},
		{Ref: 2, Households: 2, Reserve: 2},
		{Ref: 3, Households: 2, Reserve: 3},
	}
	vacancies := []employment.Vacancy{
		{Ref: 10, Positions: 2, Bid: 9, Duration: 1, ShareBP: 5000}, // best bid, too small for offer 1
		{Ref: 11, Positions: 5, Bid: 8, Duration: 1, ShareBP: 5000},
	}
	var r employment.Result
	employment.Round(&r, offers, vacancies, open, nil)
	// Offer 1 (3) skips 10 (2 left) and fills 11 (5 -> 2). Offer 2 (2) takes 10 (2 -> 0).
	// Offer 3 (2) finds 10 full and takes 11 (2 -> 0).
	if want := [][2]int{{1, 11}, {2, 10}, {3, 11}}; !reflect.DeepEqual(refs(&r), want) {
		t.Errorf("Round hires = %v, want %v", refs(&r), want)
	}
	for _, v := range r.Vacancies {
		if v.Left != 0 || v.Reason != employment.ReasonNone {
			t.Errorf("vacancy %+v: want Left 0 and no reason", v)
		}
	}
}

// TestRound_FailureLowersFitToBlockLessOne verifies the fit: a failure of a
// block-dependent check for m households lowers the vacancy's fit to m - 1, so
// a later offer of m or more skips it without a new count, and a smaller one
// may still form.
//
// Requirement: FR-EMP-023
func TestRound_FailureLowersFitToBlockLessOne(t *testing.T) {
	t.Parallel()
	offers := []employment.Offer{
		{Ref: 1, Households: 3, Reserve: 1},
		{Ref: 2, Households: 3, Reserve: 2},
		{Ref: 3, Households: 2, Reserve: 3},
	}
	vacancies := []employment.Vacancy{{Ref: 10, Positions: 6, Bid: 9, Duration: 1, ShareBP: 5000}}
	var calls []int
	check := func(o employment.Offer, v employment.Vacancy, wage int64, formed []employment.Hire) (int, employment.Close) {
		calls = append(calls, o.Ref)
		if o.Households >= 3 {
			return employment.ReasonCheck, employment.CloseFit
		}
		return employment.ReasonNone, employment.CloseFit
	}
	var r employment.Result
	employment.Round(&r, offers, vacancies, open, check)
	// Offer 1 (3) fails: the fit falls to 2. Offer 2 (3) is skipped without a call. Offer 3 (2) forms.
	if want := []int{1, 3}; !slices.Equal(calls, want) {
		t.Errorf("check called for offers %v, want %v", calls, want)
	}
	if want := [][2]int{{3, 10}}; !reflect.DeepEqual(refs(&r), want) {
		t.Errorf("Round hires = %v, want %v", refs(&r), want)
	}
	if v := r.Vacancies[0]; v.Filled != 2 || v.Left != 4 || v.Reason != employment.ReasonNoLabour {
		t.Errorf("vacancy result = %+v, want Filled 2, Left 4, no labour (still open)", v)
	}
}

// TestRound_VacancyClosingReasonCloses verifies that a failure that does not
// depend on the block closes the vacancy, and that a fit that falls to 0 closes
// it too, each counting the positions left once with the reason.
//
// Requirement: FR-EMP-023
func TestRound_VacancyClosingReasonCloses(t *testing.T) {
	t.Parallel()
	offers := []employment.Offer{
		{Ref: 1, Households: 1, Reserve: 1},
		{Ref: 2, Households: 1, Reserve: 2},
	}
	vacancies := []employment.Vacancy{
		{Ref: 10, Positions: 4, Bid: 9, Duration: 1, ShareBP: 5000},
		{Ref: 11, Positions: 3, Bid: 8, Duration: 1, ShareBP: 5000},
	}
	const noSite, noInput = employment.ReasonCheck, employment.ReasonCheck + 1
	check := func(o employment.Offer, v employment.Vacancy, wage int64, formed []employment.Hire) (int, employment.Close) {
		switch v.Ref {
		case 10:
			return noSite, employment.CloseVacancy // closes whatever the block
		default:
			return noInput, employment.CloseFit // a block of 1 lowers the fit to 0: closes
		}
	}
	var r employment.Result
	employment.Round(&r, offers, vacancies, open, check)
	if len(r.Hires) != 0 {
		t.Fatalf("Round formed %v, want none", refs(&r))
	}
	// Offer 1 fails on 10 (closed, reason noSite) and on 11 (fit 0, closed, noInput).
	// Offer 2 then finds both closed.
	want := []employment.VacancyResult{{Ref: 10, Left: 4, Reason: noSite}, {Ref: 11, Left: 3, Reason: noInput}}
	for i, v := range r.Vacancies {
		if v.Ref != want[i].Ref || v.Left != want[i].Left || v.Reason != want[i].Reason {
			t.Errorf("vacancy %d = %+v, want ref %d left %d reason %d", i, v, want[i].Ref, want[i].Left, want[i].Reason)
		}
	}
	for _, o := range r.Offers {
		if o.Hired {
			t.Errorf("offer %+v hired, want unmatched", o)
		}
	}
}

// TestRound_LaneClosingReasonCloses verifies check 1: when the hires open in
// the region reach MaxOpen the lane is closed for the rest of the round, the
// later offers are unmatched and every vacancy still open counts its positions
// once as too many.
//
// Requirement: FR-EMP-023, FR-EMP-063
func TestRound_LaneClosingReasonCloses(t *testing.T) {
	t.Parallel()
	offers := []employment.Offer{
		{Ref: 1, Households: 1, Reserve: 1},
		{Ref: 2, Households: 1, Reserve: 2},
		{Ref: 3, Households: 1, Reserve: 3},
	}
	vacancies := []employment.Vacancy{
		{Ref: 10, Positions: 5, Bid: 9, Duration: 1, ShareBP: 5000},
		{Ref: 11, Positions: 2, Bid: 8, Duration: 1, ShareBP: 5000},
	}
	var r employment.Result
	// MaxOpen 3 with 2 already open: one hire fits.
	employment.Round(&r, offers, vacancies, employment.Lane{Open: 2, MaxOpen: 3}, nil)
	if want := [][2]int{{1, 10}}; !reflect.DeepEqual(refs(&r), want) {
		t.Fatalf("Round hires = %v, want %v", refs(&r), want)
	}
	if r.Offers[0].Hired != true || r.Offers[1].Hired || r.Offers[2].Hired {
		t.Errorf("offers = %+v, want only the first hired", r.Offers)
	}
	// 10: 5 - 1 = 4 left; 11: 2 left. Both counted once as too many.
	for i, want := range []int64{4, 2} {
		if v := r.Vacancies[i]; v.Left != want || v.Reason != employment.ReasonTooMany {
			t.Errorf("vacancy %d = %+v, want Left %d, too many", i, v, want)
		}
	}
}

// TestRound_MaxOpenPerLaneNotWorld verifies that MaxOpen is per region: two
// lanes with the same cap each form their own hires, neither first come
// against the other.
//
// Requirement: FR-EMP-063
func TestRound_MaxOpenPerLaneNotWorld(t *testing.T) {
	t.Parallel()
	one := employment.Lane{MaxOpen: 1}
	for lane := range 2 {
		offers := []employment.Offer{{Ref: 1, Households: 1, Reserve: 1}, {Ref: 2, Households: 1, Reserve: 2}}
		vacancies := []employment.Vacancy{{Ref: 10, Positions: 2, Bid: 9, Duration: 1, ShareBP: 5000}}
		var r employment.Result
		employment.Round(&r, offers, vacancies, one, nil)
		if len(r.Hires) != 1 {
			t.Errorf("lane %d formed %d hires under MaxOpen 1, want 1", lane, len(r.Hires))
		}
	}
}

// TestRound_UnmatchedCountedOnce verifies that an offer no vacancy takes is one
// unmatched row, whatever the number of vacancies it was tried against, and
// that an offer above every bid is unmatched without a check.
//
// Requirement: FR-EMP-020
func TestRound_UnmatchedCountedOnce(t *testing.T) {
	t.Parallel()
	offers := []employment.Offer{{Ref: 1, Households: 4, Reserve: 7}, {Ref: 2, Households: 1, Reserve: 20}}
	vacancies := []employment.Vacancy{
		{Ref: 10, Positions: 1, Bid: 9, Duration: 1, ShareBP: 5000}, // too few positions for offer 1
		{Ref: 11, Positions: 3, Bid: 8, Duration: 1, ShareBP: 5000}, // too few as well
	}
	calls := 0
	check := func(employment.Offer, employment.Vacancy, int64, []employment.Hire) (int, employment.Close) {
		calls++
		return employment.ReasonNone, employment.CloseFit
	}
	var r employment.Result
	employment.Round(&r, offers, vacancies, open, check)
	if len(r.Hires) != 0 || calls != 0 {
		t.Errorf("Round formed %v with %d check calls, want none and 0", refs(&r), calls)
	}
	if len(r.Offers) != 2 || r.Offers[0].Hired || r.Offers[1].Hired {
		t.Errorf("offers = %+v, want two unmatched rows", r.Offers)
	}
	// Offer 2 asks 20 against bids of 9 and 8: a pair that cannot agree is not a
	// wage below 1 and is not counted as one.
	if r.WageBelowUnit != 0 {
		t.Errorf("WageBelowUnit = %d, want 0 (no pair could agree)", r.WageBelowUnit)
	}
	// Both vacancies were never reached by an offer that fit: no labour.
	for _, v := range r.Vacancies {
		if v.Reason != employment.ReasonNoLabour {
			t.Errorf("vacancy %+v, want no labour", v)
		}
	}
}

// TestRound_ZeroWageSkipsPair verifies that a pair whose wage is below 1 is
// skipped and counted, not formed, and that the offer is unmatched.
//
// Requirement: FR-EMP-020
func TestRound_ZeroWageSkipsPair(t *testing.T) {
	t.Parallel()
	// Reserve 0, bid 1 at the midpoint: 0 + 5 000 * 1 / 10 000 = 0.5, half to even 0, so the wage is 0.
	offers := []employment.Offer{{Ref: 1, Households: 1, Reserve: 0}}
	vacancies := []employment.Vacancy{{Ref: 10, Positions: 1, Bid: 1, Duration: 1, ShareBP: 5000}}
	var r employment.Result
	employment.Round(&r, offers, vacancies, open, nil)
	if len(r.Hires) != 0 || r.WageBelowUnit != 1 || r.Offers[0].Hired {
		t.Errorf("Round = %+v, want no hire, 1 wage below unit, offer unmatched", r)
	}
}

// TestRound_WageIndependentOfCounts verifies that the wage of a pair does not
// move with the number of other offers and vacancies.
//
// Requirement: FR-EMP-022
func TestRound_WageIndependentOfCounts(t *testing.T) {
	t.Parallel()
	pair := func(extraOffers, extraVacancies int) int64 {
		offers := []employment.Offer{{Ref: 1, Households: 1, Reserve: 4}}
		for i := range extraOffers {
			offers = append(offers, employment.Offer{Ref: 100 + i, Households: 1, Reserve: 1000}) // never matched
		}
		vacancies := []employment.Vacancy{{Ref: 10, Positions: 1, Bid: 10, Duration: 1, ShareBP: 5000}}
		for i := range extraVacancies {
			vacancies = append(vacancies, employment.Vacancy{Ref: 200 + i, Positions: 1, Bid: 2, Duration: 1, ShareBP: 5000}) // below the reserve
		}
		var r employment.Result
		employment.Round(&r, offers, vacancies, open, nil)
		if len(r.Hires) != 1 {
			t.Fatalf("Round(%d, %d) formed %d hires, want 1", extraOffers, extraVacancies, len(r.Hires))
		}
		return r.Hires[0].Wage
	}
	// 4 + 5 000 * (10 - 4) / 10 000 = 4 + 3 = 7 whatever the crowd.
	for _, n := range [][2]int{{0, 0}, {50, 0}, {0, 50}, {50, 50}} {
		if got := pair(n[0], n[1]); got != 7 {
			t.Errorf("wage with %d extra offers and %d extra vacancies = %d, want 7", n[0], n[1], got)
		}
	}
}

// TestRound_CheckSeesHiresFormedSoFar verifies that the check receives the hires
// formed so far in the lane in order, which is what the committed executions of
// an employer are counted from.
//
// Requirement: FR-EMP-025
func TestRound_CheckSeesHiresFormedSoFar(t *testing.T) {
	t.Parallel()
	offers := []employment.Offer{{Ref: 1, Households: 1, Reserve: 1}, {Ref: 2, Households: 1, Reserve: 2}, {Ref: 3, Households: 1, Reserve: 3}}
	vacancies := []employment.Vacancy{{Ref: 10, Positions: 5, Bid: 9, Duration: 1, ShareBP: 5000}}
	var seen []int
	check := func(o employment.Offer, v employment.Vacancy, wage int64, formed []employment.Hire) (int, employment.Close) {
		seen = append(seen, len(formed))
		return employment.ReasonNone, employment.CloseFit
	}
	var r employment.Result
	employment.Round(&r, offers, vacancies, open, check)
	if want := []int{0, 1, 2}; !slices.Equal(seen, want) {
		t.Errorf("check saw %v hires before each pair, want %v", seen, want)
	}
}

// TestRound_BadCountsDoNoHarm verifies that an offer of fewer than 1 household
// is left unmatched and a vacancy of fewer than 0 positions counts as 0, so the
// round never forms an empty or negative hire.
//
// Requirement: FR-EMP-020
func TestRound_BadCountsDoNoHarm(t *testing.T) {
	t.Parallel()
	offers := []employment.Offer{{Ref: 1, Households: 0, Reserve: 1}, {Ref: 2, Households: -3, Reserve: 2}}
	vacancies := []employment.Vacancy{{Ref: 10, Positions: -5, Bid: 9, Duration: 1, ShareBP: 5000}}
	var r employment.Result
	employment.Round(&r, offers, vacancies, open, nil)
	if len(r.Hires) != 0 {
		t.Errorf("Round formed %v, want no hire", refs(&r))
	}
	if len(r.Offers) != 2 || r.Offers[0].Hired || r.Offers[1].Hired {
		t.Errorf("offers = %+v, want two unmatched rows", r.Offers)
	}
	if v := r.Vacancies[0]; v.Left != 0 || v.Filled != 0 || v.Reason != employment.ReasonNone {
		t.Errorf("vacancy = %+v, want nothing left and no reason", v)
	}
}
