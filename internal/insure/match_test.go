package insure_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/insure"
)

// fill is one call of the formation callback, recorded by a test.
type fill struct{ cand, offer int }

// scriptedForm answers each fill by the table, recording the calls: an offer Ref
// not in the table forms.
func scriptedForm(answers map[int]insure.Outcome, calls *[]fill) func(insure.Fill) insure.Outcome {
	return func(f insure.Fill) insure.Outcome {
		*calls = append(*calls, fill{f.Candidate, f.Offer})
		if o, ok := answers[f.Offer]; ok {
			return o
		}
		return insure.Formed
	}
}

// TestMatch_SkipOwnLowCoverLowRoomPairMaximum verifies the three skips of the scan
// and that room is tested against the pair's maximum payout.
//
// Candidate sum 1 000, party 1, lowest cover 5 000, limit 20 000. Offer 0 is the
// candidate's own; offer 1 has cover 4 000 (below the lowest); offer 2 has cover
// 8 000 and room 799 (maximum payout 800, so skipped though 799 would cover a
// smaller pair); offer 3 has cover 5 000 and room 500 (maximum payout 500, fits)
// and forms. Room falls by the maximum payout.
//
// Requirement: FR-INS-034
func TestMatch_SkipOwnLowCoverLowRoomPairMaximum(t *testing.T) {
	t.Parallel()
	offers := []insure.Offer{
		{Ref: 0, Party: 1, Loading: 100, CoverBP: 8_000, Room: 9_999},
		{Ref: 1, Party: 2, Loading: 110, CoverBP: 4_000, Room: 9_999},
		{Ref: 2, Party: 3, Loading: 120, CoverBP: 8_000, Room: 799},
		{Ref: 3, Party: 4, Loading: 130, CoverBP: 5_000, Room: 500},
	}
	cands := []insure.Candidate{{Ref: 7, Party: 1, Limit: 20_000, MinCover: 5_000, Sum: 1_000}}
	var calls []fill
	halted := insure.Match(cands, offers, false, scriptedForm(nil, &calls))
	if halted || !slices.Equal(calls, []fill{{7, 3}}) {
		t.Errorf("Match = halted %v, calls %v, want false, [{7 3}]", halted, calls)
	}
	if offers[3].Room != 0 {
		t.Errorf("offer 3 room = %d, want 0 (500 less the maximum payout 500)", offers[3].Room)
	}
}

// TestMatch_ScanEndsAtLimit verifies that the scan for a candidate ends at the first
// offer whose price exceeds the limit, though a later offer has room, and that a lane
// request compares the loading and a store request the store rate.
//
// Requirement: FR-INS-034
func TestMatch_ScanEndsAtLimit(t *testing.T) {
	t.Parallel()
	for _, store := range []bool{false, true} {
		offers := []insure.Offer{
			{Ref: 0, Party: 2, Loading: 150, StoreRate: 50, CoverBP: 8_000, Room: 5},
			{Ref: 1, Party: 3, Loading: 150, StoreRate: 50, CoverBP: 8_000, Room: 5},
			{Ref: 2, Party: 4, Loading: 100, StoreRate: 200, CoverBP: 8_000, Room: 9_999},
		}
		if store {
			insure.SortStoreRates(offers)
		} else {
			insure.SortLoadings(offers)
		}
		cands := []insure.Candidate{{Ref: 1, Party: 1, Limit: 120, MinCover: 1, Sum: 1_000}}
		var calls []fill
		insure.Match(cands, offers, store, scriptedForm(nil, &calls))
		// Lane: ascending loadings 100 (ref 2), 150, 150: ref 2 is at the price and forms.
		// Store: ascending rates 50, 50 (refs 0, 1: room 5 below the maximum payout 800,
		// skipped), 200 (ref 2): above the limit 120, so the scan ends with no fill.
		want := []fill{{1, 2}}
		if store {
			want = nil
		}
		if !slices.Equal(calls, want) {
			t.Errorf("Match(store=%v) calls = %v, want %v", store, calls, want)
		}
	}
}

// TestMatch_OutcomesSteerTheScan verifies what each answer of the formation does: a
// failure ends the candidate, a drop goes on to the next offer, a halt ends the lane,
// and an offer is used by at most one candidate until its room is gone.
//
// Requirement: FR-INS-034, FR-INS-035
func TestMatch_OutcomesSteerTheScan(t *testing.T) {
	t.Parallel()
	mk := func() []insure.Offer {
		return []insure.Offer{
			{Ref: 0, Party: 2, Loading: 100, CoverBP: 10_000, Room: 1_000},
			{Ref: 1, Party: 3, Loading: 110, CoverBP: 10_000, Room: 1_000},
		}
	}
	cands := []insure.Candidate{
		{Ref: 10, Party: 1, Limit: 500, MinCover: 1, Sum: 1_000},
		{Ref: 11, Party: 1, Limit: 500, MinCover: 1, Sum: 1_000},
	}
	tests := []struct {
		name       string
		answers    map[int]insure.Outcome
		want       []fill
		wantHalted bool
	}{
		{"formed offers are used up", nil, []fill{{10, 0}, {11, 1}}, false},
		{"a failure ends the candidate", map[int]insure.Outcome{0: insure.Failed}, []fill{{10, 0}, {11, 0}}, false},
		{"a drop goes on to the next offer", map[int]insure.Outcome{0: insure.Dropped}, []fill{{10, 0}, {10, 1}, {11, 0}}, false},
		{"a halt ends the lane", map[int]insure.Outcome{0: insure.Halted}, []fill{{10, 0}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls []fill
			halted := insure.Match(cands, mk(), false, scriptedForm(tt.answers, &calls))
			if halted != tt.wantHalted || !slices.Equal(calls, tt.want) {
				t.Errorf("Match = halted %v, calls %v, want %v, %v", halted, calls, tt.wantHalted, tt.want)
			}
		})
	}
}

// TestMatch_NoOffers verifies that a lane with no offer and a lane with no candidate
// call nothing and are not halted.
//
// Requirement: FR-INS-034
func TestMatch_NoOffers(t *testing.T) {
	t.Parallel()
	var calls []fill
	form := scriptedForm(nil, &calls)
	if insure.Match([]insure.Candidate{{Ref: 1, Party: 1, Limit: 1, Sum: 1}}, nil, false, form) || len(calls) != 0 {
		t.Errorf("Match with no offers called %v, want nothing", calls)
	}
	if insure.Match(nil, []insure.Offer{{Ref: 0, Party: 2, CoverBP: 1, Room: 1}}, false, form) || len(calls) != 0 {
		t.Errorf("Match with no candidates called %v, want nothing", calls)
	}
}

// TestCheck_OrderOfFailures verifies the checks of the formation in their order: the
// first failing one is the reason, and a premium that did not fit is an overflow and
// not a premium below 1.
//
// Requirement: FR-INS-036
func TestCheck_OrderOfFailures(t *testing.T) {
	t.Parallel()
	ok := insure.Form{Open: 1, MaxPolicies: 2, Sum: 100, MinSum: 50, Premium: 10, PremiumOK: true, Available: 10, Balance: 5, Limit: 1_000}
	tests := []struct {
		name   string
		mutate func(*insure.Form)
		want   insure.Failure
	}{
		{"passes", func(*insure.Form) {}, insure.None},
		{"too many wins over every later check", func(f *insure.Form) { f.Open = 2; f.Sum = 0; f.Available = 0; f.Balance = 1_000 }, insure.TooManyPolicies},
		{"sum below the minimum", func(f *insure.Form) { f.Sum = 49 }, insure.BelowMinimum},
		{"premium below 1", func(f *insure.Form) { f.Premium = 0 }, insure.BelowMinimum},
		{"below the minimum wins over funds", func(f *insure.Form) { f.Premium = 0; f.Available = 0 }, insure.BelowMinimum},
		{"premium above the available money", func(f *insure.Form) { f.Available = 9 }, insure.Funds},
		{"premium exactly the available money", func(f *insure.Form) { f.Available = 10 }, insure.None},
		{"funds wins over overflow", func(f *insure.Form) { f.Available = 9; f.Balance = 1_000 }, insure.Funds},
		{"the underwriter's balance would pass the bound", func(f *insure.Form) { f.Balance = 991 }, insure.Overflow},
		{"the balance reaches the bound exactly", func(f *insure.Form) { f.Balance = 990 }, insure.None},
		{"a premium that did not fit", func(f *insure.Form) { f.PremiumOK = false; f.Premium = 0 }, insure.Overflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := ok
			tt.mutate(&f)
			if got := insure.Check(f); got != tt.want {
				t.Errorf("Check(%+v) = %d, want %d", f, got, tt.want)
			}
		})
	}
}
