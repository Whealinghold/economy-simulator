package tenure_test

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// byRef returns the requests as a slice indexed by Ref, as the caller reads them.
func byRef(reqs []tenure.Request) []tenure.Request {
	out := make([]tenure.Request, len(reqs))
	for _, r := range reqs {
		out[r.Ref] = r
	}
	return out
}

// example48 is the example of spec-land.md 4.8: ten holders of 500 basis points
// each and a commons of 5 000 over a stock of 100 give each holder 5 units and
// the commons 50. Each holder wants 3 executions of 1 unit; forty landless
// households want 2 each. Rent is 40 and every principal is rich.
func example48() (tenure.Params, []tenure.Holder, []tenure.Request) {
	holders := make([]tenure.Holder, 10)
	var reqs []tenure.Request
	for i := range holders {
		id := int64(1 + i)
		holders[i] = tenure.Holder{ID: id, Entitled: 5}
		reqs = append(reqs, tenure.Request{Ref: len(reqs), ID: id, Principal: id, Executions: 3, Draw: 1, Key: uint64(id), Balance: 10_000})
	}
	for i := range 40 {
		id := int64(100 + i)
		reqs = append(reqs, tenure.Request{Ref: len(reqs), ID: id, Principal: id, Executions: 2, Draw: 1, Key: uint64(i * 13 % 40), Balance: 10_000})
	}
	return tenure.Params{Stock: 100, Commons: 50, Rent: 40}, holders, reqs
}

// TestTiers_OwnThenCommonsThenLet verifies the example of spec-land.md 4.8.
// Each holder takes 3 of its 5 units and has 2 unused, so the let supply is 20.
// The landless want 80 units against a commons of 50: floor(2 * 50 / 80) = 1
// each (40 units) and the 10 left go to ten of them, so ten get 2 executions
// and thirty get 1. Thirty requests of 1 execution remain against 20 let units:
// the proportional part is 0 and the 20 left go to twenty of them, ten are
// unserved. 30 + 50 + 20 = 100 units are drawn, the whole stock.
//
// Requirement: FR-LND-008, FR-LND-009, FR-LND-010
func TestTiers_OwnThenCommonsThenLet(t *testing.T) {
	t.Parallel()
	p, holders, reqs := example48()
	var s tenure.Scratch
	res, ok := s.Tiers(p, holders, reqs)
	if !ok {
		t.Fatalf("Tiers(example 4.8) not ok")
	}
	want := tenure.Result{OwnUnits: 30, CommonsUnits: 50, LetUnits: 20, Supply: 20, Unserved: 10}
	if res != want {
		t.Errorf("Tiers(example 4.8) = %+v, want %+v", res, want)
	}
	checkHolders(t, holders, reqs)
	two, tenants, unserved := countLandless(reqs)
	if two != 10 || tenants != 20 || unserved != 10 {
		t.Errorf("Tiers(example 4.8): %d with 2 commons, %d tenants, %d unserved, want 10, 20, 10", two, tenants, unserved)
	}
}

// checkHolders checks each holder of the example has 2 units unused and each
// holder's request took 3 own executions and nothing else.
func checkHolders(t *testing.T, holders []tenure.Holder, reqs []tenure.Request) {
	t.Helper()
	for _, h := range holders {
		if h.Unused != 2 {
			t.Errorf("Tiers(example 4.8) holder %d unused %d, want 2", h.ID, h.Unused)
		}
	}
	for _, r := range reqs {
		if r.ID < 100 && (r.Own != 3 || r.Commons != 0 || r.Let != 0) {
			t.Errorf("Tiers(example 4.8) holder request %+v, want 3 own", r)
		}
	}
}

// countLandless counts the landless requests (ID 100 and above) that got 2 commons
// executions, that took a let unit after 1, and that were left unserved after 1.
func countLandless(reqs []tenure.Request) (two, tenants, unserved int) {
	for _, r := range reqs {
		switch {
		case r.ID < 100:
		case r.Commons == 2 && r.Let == 0:
			two++
		case r.Commons == 1 && r.Let == 1:
			tenants++
		case r.Commons == 1:
			unserved++
		}
	}
	return two, tenants, unserved
}

// TestTiers_LetDemandClippedByCash_PricedOutCounted verifies the clip of the
// let demand (spec-land.md 4.3 step 3). A holder of 10 units with no request
// lets all 10 at a rent of 5. Three tenants want 4 executions of 1 unit: one
// has balance 100 (can pay 20, so all 4), one has 12 (floor(12 / 5) = 2) and one
// has 4 (can pay none, so 0). The units priced out are 2 + 4 = 6 and the let
// demand is 6 units, all granted.
//
// Requirement: FR-LND-010
func TestTiers_LetDemandClippedByCash_PricedOutCounted(t *testing.T) {
	t.Parallel()
	holders := []tenure.Holder{{ID: 1, Entitled: 10}}
	reqs := []tenure.Request{
		{Ref: 0, ID: 10, Principal: 10, Executions: 4, Draw: 1, Balance: 100},
		{Ref: 1, ID: 11, Principal: 11, Executions: 4, Draw: 1, Balance: 12},
		{Ref: 2, ID: 12, Principal: 12, Executions: 4, Draw: 1, Balance: 4},
	}
	var s tenure.Scratch
	res, ok := s.Tiers(tenure.Params{Stock: 10, Rent: 5}, holders, reqs)
	if !ok {
		t.Fatalf("Tiers(%+v) not ok", reqs)
	}
	var let, priced []int64
	for _, r := range byRef(reqs) {
		let, priced = append(let, r.Let), append(priced, r.PricedOut)
	}
	if !slices.Equal(let, []int64{4, 2, 0}) || !slices.Equal(priced, []int64{0, 2, 4}) {
		t.Errorf("Tiers let grants %v, priced out %v, want [4 2 0] and [0 2 4]", let, priced)
	}
	if want := (tenure.Result{LetUnits: 6, Supply: 10, PricedOut: 6}); res != want {
		t.Errorf("Tiers(...) = %+v, want %+v", res, want)
	}
}

// TestTiers_ExemptAndZeroRentAreNotClipped verifies an exempt request and a rent
// of 0 are never clipped by cash, since no rent is due (spec-land.md 5.3.4).
//
// Requirement: FR-LND-010, FR-LND-018
func TestTiers_ExemptAndZeroRentAreNotClipped(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		rent   int64
		exempt bool
	}{{"exempt", 5, true}, {"zero rent", 0, false}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			holders := []tenure.Holder{{ID: 1, Entitled: 10}}
			reqs := []tenure.Request{{Ref: 0, ID: 10, Principal: 10, Executions: 3, Draw: 2, Balance: 0, Exempt: tt.exempt}}
			var s tenure.Scratch
			res, ok := s.Tiers(tenure.Params{Stock: 10, Rent: tt.rent}, holders, reqs)
			if !ok || reqs[0].Let != 3 || res.PricedOut != 0 {
				t.Errorf("Tiers(rent %d, exempt %v) = %+v, %v, let %d, want 3 executions and nothing priced out", tt.rent, tt.exempt, res, ok, reqs[0].Let)
			}
		})
	}
}

// TestTiers_ClosedLetsNothing_HeldCounted verifies a Closed endowment lets
// nothing: the unused entitlements stay in the stock and are counted as held,
// and the demand for the let tier is not rationed or clipped.
//
// Requirement: FR-LND-010
func TestTiers_ClosedLetsNothing_HeldCounted(t *testing.T) {
	t.Parallel()
	holders := []tenure.Holder{{ID: 1, Entitled: 6}, {ID: 2, Entitled: 4}}
	reqs := []tenure.Request{{Ref: 0, ID: 1, Principal: 1, Executions: 2, Draw: 1, Balance: 9}, {Ref: 1, ID: 9, Principal: 9, Executions: 5, Draw: 1, Balance: 9}}
	var s tenure.Scratch
	res, ok := s.Tiers(tenure.Params{Stock: 10, Rent: 3, Closed: true}, holders, reqs)
	if !ok {
		t.Fatalf("Tiers(closed) not ok")
	}
	// Holder 1 uses 2 of 6 and holder 2 none: 4 + 4 = 8 held. The landless request gets nothing.
	if want := (tenure.Result{OwnUnits: 2, Held: 8}); res != want {
		t.Errorf("Tiers(closed) = %+v, want %+v", res, want)
	}
	if reqs[1].Let != 0 || reqs[1].Commons != 0 || reqs[1].PricedOut != 0 {
		t.Errorf("Tiers(closed) landless request %+v, want no grant", reqs[1])
	}
}

// TestTiers_CohortOwnTierIsItsShare verifies a cohort holder is one request of w
// executions whose own tier is its entitlement: a cohort of 6 executions of 2
// units against an entitlement of 7 takes floor(7 / 2) = 3 own executions and
// has 1 unit unused.
//
// Requirement: FR-LND-008
func TestTiers_CohortOwnTierIsItsShare(t *testing.T) {
	t.Parallel()
	holders := []tenure.Holder{{ID: 5, Entitled: 7}}
	reqs := []tenure.Request{{Ref: 0, ID: 5, Principal: 5, Executions: 6, Draw: 2, Balance: 0}}
	var s tenure.Scratch
	if _, ok := s.Tiers(tenure.Params{Stock: 7, Rent: 1}, holders, reqs); !ok {
		t.Fatalf("Tiers(cohort) not ok")
	}
	if reqs[0].Own != 3 || holders[0].Unused != 1 {
		t.Errorf("Tiers(cohort) own %d unused %d, want 3 and 1", reqs[0].Own, holders[0].Unused)
	}
}

// TestTiers_PrincipalWithSeveralRequests verifies the entitlement of a holder
// is rationed among the requests of its principal (spec-land.md 4.7.3): two
// jobs of 3 executions of 1 unit against an entitlement of 4 share it 2 and 2.
//
// Requirement: FR-LND-008, FR-LND-012
func TestTiers_PrincipalWithSeveralRequests(t *testing.T) {
	t.Parallel()
	holders := []tenure.Holder{{ID: 1, Entitled: 4}}
	reqs := []tenure.Request{
		{Ref: 0, ID: 20, Principal: 1, Executions: 3, Draw: 1, Key: 1, Balance: 100},
		{Ref: 1, ID: 21, Principal: 1, Executions: 3, Draw: 1, Key: 2, Balance: 100},
	}
	var s tenure.Scratch
	if _, ok := s.Tiers(tenure.Params{Stock: 4, Rent: 1}, holders, reqs); !ok {
		t.Fatalf("Tiers(two jobs) not ok")
	}
	got := byRef(reqs)
	if got[0].Own != 2 || got[1].Own != 2 || holders[0].Unused != 0 {
		t.Errorf("Tiers(two jobs) own %d %d, unused %d, want 2 2 0", got[0].Own, got[1].Own, holders[0].Unused)
	}
}

// TestTiers_GrantNeverExceedsRequest_UnitsNeverExceedStock_IndependentOfListingOrder
// checks over random endowments that a grant never exceeds its request, that the
// units of all tiers never exceed the stock, and that a shuffled listing of the
// holders and the requests gives the same grants and results.
//
// Requirement: FR-LND-011, NFR-LND-003
func TestTiers_GrantNeverExceedsRequest_UnitsNeverExceedStock_IndependentOfListingOrder(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(3, 4))
	for seed := range 200 {
		p, holders, reqs := randomEndowment(rng)
		var s tenure.Scratch
		res, ok := s.Tiers(p, slices.Clone(holders), reqs)
		if !ok {
			t.Fatalf("seed %d: Tiers(%+v) not ok", seed, p)
		}
		checkTiers(t, p, res, reqs)
		shuffledH, shuffledR := slices.Clone(holders), slices.Clone(reqs)
		rng.Shuffle(len(shuffledH), func(i, j int) { shuffledH[i], shuffledH[j] = shuffledH[j], shuffledH[i] })
		rng.Shuffle(len(shuffledR), func(i, j int) { shuffledR[i], shuffledR[j] = shuffledR[j], shuffledR[i] })
		again, ok := s.Tiers(p, shuffledH, shuffledR)
		if !ok || again != res || !slices.Equal(byRef(shuffledR), byRef(reqs)) {
			t.Fatalf("seed %d: shuffled listing gave %+v, want %+v", seed, again, res)
		}
	}
}

// randomEndowment builds a random endowment whose entitlements sum to the stock.
func randomEndowment(rng *rand.Rand) (tenure.Params, []tenure.Holder, []tenure.Request) {
	stock := int64(rng.IntN(300))
	nh := rng.IntN(6)
	holders := make([]tenure.Holder, nh)
	left := stock
	for i := range holders {
		e := int64(rng.IntN(int(left) + 1))
		holders[i] = tenure.Holder{ID: int64(i + 1), Entitled: e}
		left -= e
	}
	commons := left
	var reqs []tenure.Request
	for i := range 2 + rng.IntN(12) {
		id := int64(i + 1)
		principal := id
		if rng.IntN(3) == 0 {
			principal = int64(1 + rng.IntN(7)) // a holder, or a principal with no title
		}
		reqs = append(reqs, tenure.Request{
			Ref: i, ID: id, Principal: principal, Executions: int64(rng.IntN(9)), Draw: int64(1 + rng.IntN(3)),
			Key: uint64(rng.IntN(5)), Balance: int64(rng.IntN(60)), Exempt: rng.IntN(8) == 0,
		})
	}
	return tenure.Params{Stock: stock, Commons: commons, Rent: int64(rng.IntN(6)), Closed: rng.IntN(5) == 0}, holders, reqs
}

// TestTiers_Refusals verifies the inputs Tiers cannot apply: a negative number,
// a Draw below 1, a repeated ID, entitlements above the stock and a sum beyond
// int64, each reporting not ok with every output 0.
//
// Requirement: FR-LND-011
func TestTiers_Refusals(t *testing.T) {
	t.Parallel()
	good := func() (tenure.Params, []tenure.Holder, []tenure.Request) {
		return tenure.Params{Stock: 10, Commons: 4, Rent: 1},
			[]tenure.Holder{{ID: 1, Entitled: 6}},
			[]tenure.Request{{Ref: 0, ID: 1, Principal: 1, Executions: 2, Draw: 1, Balance: 5}}
	}
	tests := []struct {
		name string
		edit func(*tenure.Params, *[]tenure.Holder, *[]tenure.Request)
	}{
		{"negative stock", func(p *tenure.Params, _ *[]tenure.Holder, _ *[]tenure.Request) { p.Stock = -1 }},
		{"negative commons", func(p *tenure.Params, _ *[]tenure.Holder, _ *[]tenure.Request) { p.Stock, p.Commons = 4, -1 }},
		{"negative rent", func(p *tenure.Params, _ *[]tenure.Holder, _ *[]tenure.Request) { p.Rent = -1 }},
		{"entitlements above stock", func(p *tenure.Params, _ *[]tenure.Holder, _ *[]tenure.Request) { p.Commons = 5 }},
		{"negative entitlement", func(_ *tenure.Params, h *[]tenure.Holder, _ *[]tenure.Request) { (*h)[0].Entitled = -1 }},
		{"repeated holder", func(_ *tenure.Params, h *[]tenure.Holder, _ *[]tenure.Request) { *h = append(*h, (*h)[0]) }},
		{"zero draw", func(_ *tenure.Params, _ *[]tenure.Holder, r *[]tenure.Request) { (*r)[0].Draw = 0 }},
		{"negative executions", func(_ *tenure.Params, _ *[]tenure.Holder, r *[]tenure.Request) { (*r)[0].Executions = -1 }},
		{"repeated executor", func(_ *tenure.Params, _ *[]tenure.Holder, r *[]tenure.Request) { *r = append(*r, (*r)[0]) }},
		{"units overflow", func(_ *tenure.Params, _ *[]tenure.Holder, r *[]tenure.Request) {
			(*r)[0].Executions, (*r)[0].Draw = math.MaxInt64, 2
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, h, r := good()
			tt.edit(&p, &h, &r)
			var s tenure.Scratch
			res, ok := s.Tiers(p, h, r)
			if ok || res != (tenure.Result{}) {
				t.Fatalf("Tiers(%s) = %+v, %v, want refused", tt.name, res, ok)
			}
			for _, rq := range r {
				if rq.Own != 0 || rq.Commons != 0 || rq.Let != 0 || rq.PricedOut != 0 {
					t.Errorf("Tiers(%s) left an output %+v", tt.name, rq)
				}
			}
			for _, hd := range h {
				if hd.Unused != 0 {
					t.Errorf("Tiers(%s) left holder %+v unused", tt.name, hd)
				}
			}
		})
	}
}

// TestTiers_UnaffordableRentProduct verifies a rent times draw beyond int64 is
// unaffordable: the demand is priced out whole rather than wrapped
// (spec-land.md 5.2.3).
//
// Requirement: FR-LND-010
func TestTiers_UnaffordableRentProduct(t *testing.T) {
	t.Parallel()
	holders := []tenure.Holder{{ID: 1, Entitled: 10}}
	reqs := []tenure.Request{{Ref: 0, ID: 2, Principal: 2, Executions: 2, Draw: 4, Balance: math.MaxInt64}}
	var s tenure.Scratch
	res, ok := s.Tiers(tenure.Params{Stock: 10, Rent: math.MaxInt64 / 2}, holders, reqs)
	if !ok || res.PricedOut != 8 || reqs[0].Let != 0 {
		t.Errorf("Tiers(huge rent) = %+v, %v, let %d, want 8 units priced out and no grant", res, ok, reqs[0].Let)
	}
}
