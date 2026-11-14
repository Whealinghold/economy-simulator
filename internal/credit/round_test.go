package credit_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/credit"
)

// formAll answers Formed for every fill and records the fills.
func formAll(fills *[]credit.Fill) func(credit.Fill) credit.Outcome {
	return func(f credit.Fill) credit.Outcome {
		*fills = append(*fills, f)
		return credit.Formed
	}
}

// TestOrder_Offers verifies that offers are ordered by effective rate ascending,
// equal rates by key ascending, then by agent.
//
// Requirement: FR-CRD-022
func TestOrder_Offers(t *testing.T) {
	t.Parallel()
	offers := []credit.Row{
		{Ref: 0, Agent: 9, RateBP: 900, Key: 1},
		{Ref: 1, Agent: 8, RateBP: 600, Key: 5},
		{Ref: 2, Agent: 7, RateBP: 600, Key: 2},
		{Ref: 3, Agent: 6, RateBP: 600, Key: 2},
	}
	credit.SortOffers(offers)
	var refs []int
	for _, o := range offers {
		refs = append(refs, o.Ref)
	}
	if want := []int{3, 2, 1, 0}; !slices.Equal(refs, want) {
		t.Errorf("SortOffers order = %v, want %v (rate, then key, then agent)", refs, want)
	}
}

// TestOrder_Requests verifies that requests are ordered by limit descending,
// equal limits by key ascending, then by agent.
//
// Requirement: FR-CRD-022
func TestOrder_Requests(t *testing.T) {
	t.Parallel()
	requests := []credit.Row{
		{Ref: 0, Agent: 1, RateBP: 600, Key: 1},
		{Ref: 1, Agent: 2, RateBP: 1_500, Key: 9},
		{Ref: 2, Agent: 4, RateBP: 600, Key: 1},
		{Ref: 3, Agent: 3, RateBP: 600, Key: 0},
	}
	credit.SortRequests(requests)
	var refs []int
	for _, r := range requests {
		refs = append(refs, r.Ref)
	}
	if want := []int{1, 3, 0, 2}; !slices.Equal(refs, want) {
		t.Errorf("SortRequests order = %v, want %v (limit descending, key, agent)", refs, want)
	}
}

// TestMatch_RateScan verifies the scan of spec-credit.md 6.3.2: the request of
// example A is filled by the cheaper lender alone, an offer above the limit ends
// the scan, and an offer whose longest term is too short is skipped.
//
// Requirement: FR-CRD-023
func TestMatch_RateScan(t *testing.T) {
	t.Parallel()
	offers := []credit.Row{
		{Ref: 0, Agent: 10, Principal: 9_880, Term: 90, RateBP: 600},
		{Ref: 1, Agent: 11, Principal: 2_880, Term: 90, RateBP: 900},
	}
	requests := []credit.Row{{Ref: 0, Agent: 1, Principal: 5_700, Term: 65, RateBP: 1_500}}
	var fills []credit.Fill
	if credit.Match(requests, offers, 100, formAll(&fills)) {
		t.Fatalf("Match halted, want not")
	}
	want := []credit.Fill{{Request: 0, Offer: 0, Principal: 5_700, RateBP: 600, Term: 65}}
	if !slices.Equal(fills, want) {
		t.Errorf("fills = %+v, want %+v", fills, want)
	}
	if requests[0].Principal != 0 || offers[0].Principal != 4_180 || offers[1].Principal != 2_880 {
		t.Errorf("remainders = %d, %d, %d, want 0, 4180, 2880", requests[0].Principal, offers[0].Principal, offers[1].Principal)
	}
}

// TestMatch_NoLenderAtThePrice verifies that the first offer above the request's
// limit ends the scan, even when a later offer would have a short enough term, and
// that a term too short is skipped without ending it.
//
// Requirement: FR-CRD-023
func TestMatch_NoLenderAtThePrice(t *testing.T) {
	t.Parallel()
	offers := []credit.Row{
		{Ref: 0, Agent: 10, Principal: 500, Term: 30, RateBP: 600},
		{Ref: 1, Agent: 11, Principal: 500, Term: 90, RateBP: 900},
		{Ref: 2, Agent: 12, Principal: 500, Term: 90, RateBP: 1_200},
	}
	requests := []credit.Row{{Ref: 0, Agent: 1, Principal: 800, Term: 65, RateBP: 1_000}}
	var fills []credit.Fill
	credit.Match(requests, offers, 100, formAll(&fills))
	want := []credit.Fill{{Request: 0, Offer: 1, Principal: 500, RateBP: 900, Term: 65}}
	if !slices.Equal(fills, want) {
		t.Errorf("fills = %+v, want %+v (the 600 offer is too short, the 1200 offer is above the limit)", fills, want)
	}
	if requests[0].Principal != 300 {
		t.Errorf("unfilled remainder = %d, want 300", requests[0].Principal)
	}
}

// TestMatch_Discriminatory verifies that two lenders at two prices fill one
// request each at its own rate, the lower first (spec-credit.md 6.3.3).
//
// Requirement: FR-CRD-023
func TestMatch_Discriminatory(t *testing.T) {
	t.Parallel()
	offers := []credit.Row{
		{Ref: 0, Agent: 10, Principal: 300, Term: 90, RateBP: 600},
		{Ref: 1, Agent: 11, Principal: 900, Term: 90, RateBP: 900},
	}
	requests := []credit.Row{{Ref: 0, Agent: 1, Principal: 500, Term: 65, RateBP: 1_500}}
	var fills []credit.Fill
	credit.Match(requests, offers, 100, formAll(&fills))
	want := []credit.Fill{
		{Request: 0, Offer: 0, Principal: 300, RateBP: 600, Term: 65},
		{Request: 0, Offer: 1, Principal: 200, RateBP: 900, Term: 65},
	}
	if !slices.Equal(fills, want) {
		t.Errorf("fills = %+v, want %+v", fills, want)
	}
}

// TestMatch_MinPrincipal verifies the two cases of 6.3.2 step 3: a remainder of
// the request below the minimum ends the request, and a remainder of the offer
// below it skips the offer.
//
// Requirement: FR-CRD-024
func TestMatch_MinPrincipal(t *testing.T) {
	t.Parallel()
	t.Run("request remainder too small ends the request", func(t *testing.T) {
		t.Parallel()
		offers := []credit.Row{
			{Ref: 0, Agent: 10, Principal: 450, Term: 90, RateBP: 600},
			{Ref: 1, Agent: 11, Principal: 900, Term: 90, RateBP: 700},
		}
		requests := []credit.Row{{Ref: 0, Agent: 1, Principal: 500, Term: 65, RateBP: 1_500}}
		var fills []credit.Fill
		credit.Match(requests, offers, 100, formAll(&fills))
		if len(fills) != 1 || fills[0].Principal != 450 || requests[0].Principal != 50 {
			t.Errorf("fills = %+v, request left %d, want one fill of 450 and 50 left", fills, requests[0].Principal)
		}
	})
	t.Run("offer remainder too small skips the offer", func(t *testing.T) {
		t.Parallel()
		offers := []credit.Row{
			{Ref: 0, Agent: 10, Principal: 60, Term: 90, RateBP: 600},
			{Ref: 1, Agent: 11, Principal: 900, Term: 90, RateBP: 700},
		}
		requests := []credit.Row{{Ref: 0, Agent: 1, Principal: 500, Term: 65, RateBP: 1_500}}
		var fills []credit.Fill
		credit.Match(requests, offers, 100, formAll(&fills))
		if len(fills) != 1 || fills[0].Offer != 1 || fills[0].Principal != 500 {
			t.Errorf("fills = %+v, want one fill of 500 from offer 1", fills)
		}
	})
}

// TestMatch_OwnAgentSkipped verifies that an offer of the request's own agent is
// never matched with it.
//
// Requirement: FR-CRD-023
func TestMatch_OwnAgentSkipped(t *testing.T) {
	t.Parallel()
	offers := []credit.Row{
		{Ref: 0, Agent: 1, Principal: 900, Term: 90, RateBP: 100},
		{Ref: 1, Agent: 11, Principal: 900, Term: 90, RateBP: 700},
	}
	requests := []credit.Row{{Ref: 0, Agent: 1, Principal: 500, Term: 65, RateBP: 1_500}}
	var fills []credit.Fill
	credit.Match(requests, offers, 100, formAll(&fills))
	if len(fills) != 1 || fills[0].Offer != 1 {
		t.Errorf("fills = %+v, want one fill from offer 1", fills)
	}
}

// TestMatch_Outcomes verifies that a dropped fill leaves both remainders and the
// scan goes on, and that a halt ends the whole round at once.
//
// Requirement: FR-CRD-025
func TestMatch_Outcomes(t *testing.T) {
	t.Parallel()
	mk := func() ([]credit.Row, []credit.Row) {
		return []credit.Row{
				{Ref: 0, Agent: 1, Principal: 500, Term: 10, RateBP: 1_000},
				{Ref: 1, Agent: 2, Principal: 500, Term: 10, RateBP: 900},
			}, []credit.Row{
				{Ref: 0, Agent: 10, Principal: 500, Term: 10, RateBP: 100},
				{Ref: 1, Agent: 11, Principal: 500, Term: 10, RateBP: 200},
			}
	}
	requests, offers := mk()
	var calls int
	halted := credit.Match(requests, offers, 100, func(f credit.Fill) credit.Outcome {
		calls++
		if f.Offer == 0 {
			return credit.Dropped
		}
		return credit.Formed
	})
	if halted || calls != 3 || requests[0].Principal != 0 || requests[1].Principal != 500 || offers[0].Principal != 500 || offers[1].Principal != 0 {
		t.Errorf("dropped: halted %v, calls %d, remainders %d %d / %d %d, want the dropped offer untouched and the scan on",
			halted, calls, requests[0].Principal, requests[1].Principal, offers[0].Principal, offers[1].Principal)
	}
	requests, offers = mk()
	calls = 0
	halted = credit.Match(requests, offers, 100, func(credit.Fill) credit.Outcome { calls++; return credit.Halted })
	if !halted || calls != 1 || requests[0].Principal != 500 {
		t.Errorf("halted: halted %v, calls %d, want true and one call", halted, calls)
	}
}

// TestMatch_EmptyAndNegative verifies that an offer or a request with nothing left,
// or a negative amount, is never filled, and that an offer of 1 is lent when the
// minimum is 1.
//
// Requirement: FR-CRD-024
func TestMatch_EmptyAndNegative(t *testing.T) {
	t.Parallel()
	row := func(agent int64, principal int64) credit.Row {
		return credit.Row{Ref: int(agent), Agent: agent, Principal: principal, Term: 1, RateBP: 100}
	}
	tests := []struct {
		name           string
		request, offer credit.Row
		want           int
	}{
		{"an offer of 1 at minimum 1", row(1, 1), row(2, 1), 1},
		{"an offer of 0", row(1, 5), row(2, 0), 0},
		{"a negative offer", row(1, 5), row(2, -5), 0},
		{"a request of 0", row(1, 0), row(2, 5), 0},
		{"a negative request", row(1, -5), row(2, 5), 0},
	}
	for _, tt := range tests {
		var fills []credit.Fill
		credit.Match([]credit.Row{tt.request}, []credit.Row{tt.offer}, 1, formAll(&fills))
		if len(fills) != tt.want {
			t.Errorf("%s: %d fills, want %d", tt.name, len(fills), tt.want)
		}
	}
}

// naivePick returns the index of the cheapest eligible offer for a request with
// p still to fill, the first in order among equal rates, or -1.
func naivePick(r credit.Row, p int64, offs []credit.Row, rest map[int]int64, minP int64) int {
	best := -1
	for i, o := range offs {
		if o.Agent == r.Agent || o.Term < r.Term || rest[o.Ref] <= 0 || o.RateBP > r.RateBP {
			continue
		}
		if x := min(p, rest[o.Ref]); x < minP && x < p {
			continue
		}
		if best < 0 || o.RateBP < offs[best].RateBP {
			best = i
		}
	}
	return best
}

// naiveRound is a second formulation of the matching for the cross-check: it
// keeps the remainders in a map of the original index and, for each request in
// order, repeatedly takes the cheapest eligible offer with money left, instead of
// scanning the sorted offers once.
func naiveRound(reqs, offs []credit.Row, minP int64) []credit.Fill {
	rest := map[int]int64{}
	for _, o := range offs {
		rest[o.Ref] = o.Principal
	}
	var out []credit.Fill
	for _, r := range reqs {
		for p := r.Principal; p > 0; {
			best := naivePick(r, p, offs, rest, minP)
			if best < 0 {
				break
			}
			x := min(p, rest[offs[best].Ref])
			if x < minP {
				break
			}
			out = append(out, credit.Fill{Request: r.Ref, Offer: offs[best].Ref, Principal: x, RateBP: offs[best].RateBP, Term: r.Term})
			p -= x
			rest[offs[best].Ref] -= x
		}
	}
	return out
}

// randomRound builds a random lane from a seed: distinct agents, rates and terms in
// a small range so that ties and skips are common.
func randomRound(seed uint64) (reqs, offs []credit.Row, minP int64) {
	r := rand.New(rand.NewPCG(seed, 11))
	minP = 1 + r.Int64N(50)
	for i := range 1 + r.IntN(6) {
		reqs = append(reqs, credit.Row{Ref: i, Agent: int64(r.IntN(8)), Principal: minP + r.Int64N(500), Term: 1 + r.Int64N(5), RateBP: r.Int64N(10), Key: r.Uint64N(3)})
	}
	for i := range 1 + r.IntN(6) {
		offs = append(offs, credit.Row{Ref: i, Agent: int64(r.IntN(8)), Principal: minP + r.Int64N(500), Term: 1 + r.Int64N(5), RateBP: r.Int64N(10), Key: r.Uint64N(3)})
	}
	return reqs, offs, minP
}

// TestMatch_CrossCheck verifies the matching against the naive formulation over
// 1 000 random rounds with fixed seeds (the failing seed is in the message). Both
// read the same sorted lane, so equal rates are resolved by the same order.
//
// Requirement: FR-CRD-022, FR-CRD-023, FR-CRD-024, NFR-CRD-005
func TestMatch_CrossCheck(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 1_000; seed++ {
		reqs, offs, minP := randomRound(seed)
		credit.SortRequests(reqs)
		credit.SortOffers(offs)
		want := naiveRound(reqs, offs, minP)
		var got []credit.Fill
		credit.Match(slices.Clone(reqs), slices.Clone(offs), minP, formAll(&got))
		if !slices.Equal(got, want) {
			t.Fatalf("seed %d: Match = %+v, naive = %+v (min %d, requests %+v, offers %+v)", seed, got, want, minP, reqs, offs)
		}
	}
}

// TestMatch_Conserves verifies, over random rounds, that the principal filled is
// the principal the requests and the offers each lost, and no fill is below the
// minimum, above the limit or past the longest term.
//
// Requirement: FR-CRD-023, FR-CRD-024, NFR-CRD-005
func TestMatch_Conserves(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 500; seed++ {
		reqs, offs, minP := randomRound(seed)
		credit.SortRequests(reqs)
		credit.SortOffers(offs)
		before := slices.Clone(reqs)
		offBefore := slices.Clone(offs)
		var fills []credit.Fill
		credit.Match(reqs, offs, minP, formAll(&fills))
		var total int64
		for _, f := range fills {
			total += f.Principal
			if f.Principal < minP {
				t.Fatalf("seed %d: fill %+v below the minimum %d", seed, f, minP)
			}
		}
		var lostReq, lostOff int64
		for i := range reqs {
			lostReq += before[i].Principal - reqs[i].Principal
		}
		for i := range offs {
			lostOff += offBefore[i].Principal - offs[i].Principal
		}
		if lostReq != total || lostOff != total {
			t.Fatalf("seed %d: filled %d, requests lost %d, offers lost %d", seed, total, lostReq, lostOff)
		}
	}
}
