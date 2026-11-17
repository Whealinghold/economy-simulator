package economysim

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
	tnr "github.com/Whealinghold/economy-sim/internal/tenure"
)

// The unit tests of the limits the tenure module adds to a start (appendTenureLimits,
// capRent, the reasons of the limit codes), of the open units and the charge per unit of
// a request, and of the rent term of the cost anchor. Each calls one function on a built
// world, with the scratch planted, and runs no tick.

// TestAppendTenureLimits_NoTitleThenRentFunds verifies spec-land.md 4.5.1: a request
// closed by rule adds a limit of 0 reasoned no_title, a rent cap adds its own limit
// reasoned rent_funds after it, and a request with neither adds nothing.
//
// Requirement: FR-LND-014
func TestAppendTenureLimits_NoTitleThenRentFunds(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		q    startReq
		want []production.Limit
	}{
		{"neither", startReq{}, nil},
		{"closed", startReq{noTitle: true}, []production.Limit{{Cap: 0, Reason: limitNoTitle}}},
		{"rent cap", startReq{hasRentCap: true, rentCap: 3}, []production.Limit{{Cap: 3, Reason: limitRentFunds}}},
		{"both", startReq{noTitle: true, hasRentCap: true, rentCap: 3},
			[]production.Limit{{Cap: 0, Reason: limitNoTitle}, {Cap: 3, Reason: limitRentFunds}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := appendTenureLimits(nil, &tt.q)
			if len(got) != len(tt.want) {
				t.Fatalf("appendTenureLimits(%+v) = %+v, want %+v", tt.q, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("appendTenureLimits(%+v)[%d] = %+v, want %+v", tt.q, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestLimitCount_TenureReasonsAfterNoEndowment verifies the order of the master list of
// spec-production.md 4.2.1 for the two reasons of the tenure: with the endowment's limit
// at 0 as well, no_endowment is the reason, and with only a tenure limit at 0 that limit's
// reason is, no_title before rent_funds, so a refusal for rule is never reported as a thin
// stock; a rent cap above 0 lowers k without a reason.
//
// Requirement: FR-LND-014
func TestLimitCount_TenureReasonsAfterNoEndowment(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		endowment  int64
		q          startReq
		wantK      int64
		wantReason Reason
	}{
		{"endowment 0 wins", 0, startReq{noTitle: true}, 0, ReasonNoEndowment},
		{"no title", 5, startReq{noTitle: true}, 0, ReasonNoTitle},
		{"no title before rent funds", 5, startReq{noTitle: true, hasRentCap: true}, 0, ReasonNoTitle},
		{"rent funds", 5, startReq{hasRentCap: true}, 0, ReasonRentFunds},
		{"rent cap lowers k", 5, startReq{hasRentCap: true, rentCap: 2}, 2, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			limits := []production.Limit{{Cap: tt.endowment, Reason: limitNoEndowment}}
			limits = appendTenureLimits(limits, &tt.q)
			k, code := production.Count(5, 5, limits)
			if reason := limitReason(code); k != tt.wantK || reason != tt.wantReason {
				t.Errorf("Count(5, 5, %+v) = %d, %q, want %d, %q", limits, k, reason, tt.wantK, tt.wantReason)
			}
		})
	}
}

// TestCapRent_LargestKTheBalanceCovers verifies spec-land.md 5.4.1 on planted lines: the
// rent cap is the largest k at most the weight whose rent the spendable balance covers,
// less what the earlier starts reserved; a request that pays no rent has no rent cap, and
// one priced out with nothing granted has a cap of 0.
//
// Arithmetic: one line of 3 units at a rent of 2 costs 6 an execution. With the balance of
// 10 000 and a weight of 5 all 5 executions cost 30, so k = 5. With 9 990 reserved the
// spendable balance is 10, so floor(10 / 6) = 1.
//
// Requirement: FR-LND-016
func TestCapRent_LargestKTheBalanceCovers(t *testing.T) {
	t.Parallel()
	paying := []startLine{{line: tnr.Line{Rent: 2, Draw: 3}}}
	free := []startLine{{line: tnr.Line{Rent: 0, Draw: 3}}}
	for _, tt := range []struct {
		name      string
		lines     []startLine
		reserved  int64
		pricedOut bool
		has       bool
		want      int64
	}{
		{"balance covers all", paying, 0, false, true, 5},
		{"reserved leaves one", paying, 9_990, false, true, 1},
		{"reserved leaves none", paying, 10_000, false, true, 0},
		{"no rent", free, 0, false, false, 0},
		{"priced out", nil, 0, true, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
			w.tenure.reserve(1, tt.reserved)
			w.tenure.rt.lines = tt.lines
			w.tenure.rt.rentCell = -1
			q := startReq{principal: 1, weight: 5}
			w.tenure.capRent(&q, tt.pricedOut)
			if q.hasRentCap != tt.has || q.rentCap != tt.want {
				t.Errorf("capRent() set hasRentCap %v, rentCap %d, want %v, %d", q.hasRentCap, q.rentCap, tt.has, tt.want)
			}
		})
	}
}

// TestOpenAndCharge_ClosedExemptAndRentInForce verifies the two readers of a request: the
// units open to a principal are the whole snapshot stock for an open endowment and its own
// entitlement plus the commons for a closed one, and the charge per unit is the rent in
// force, 0 for an exempt request.
//
// Requirement: FR-LND-014, FR-LND-018
func TestOpenAndCharge_ClosedExemptAndRentInForce(t *testing.T) {
	t.Parallel()
	for _, closed := range []bool{false, true} {
		w := newMill(t, rationConfig(10, 3_000, closed), WithSeed(1))
		toSnapshot(t, w)
		// pool 3 000 bp of 10 = 3 units, commons 7: a household holds no title.
		want := int64(10)
		if closed {
			want = 7
		}
		if got := w.tenure.open(0, 1); got != want {
			t.Errorf("open(closed %v) = %d, want %d", closed, got, want)
		}
	}
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	q := &resRequest{cell: 0, principal: 1, recipe: 0}
	if got := w.tenure.chargeRent(q); got != 2 {
		t.Errorf("chargeRent() = %d, want the rent in force 2", got)
	}
	w.tenure.ownUseExempt = true
	w.roles.row(w.agents[1].role).ownUse = []RecipeID{0}
	if got := w.tenure.chargeRent(q); got != 0 {
		t.Errorf("chargeRent() of an exempt request = %d, want 0", got)
	}
}

// TestRentTerms_SaturatesOnOverflowAndZeroWithNoRents verifies the rent term of the cost
// anchor (spec-land.md 5.7.1): the sum of draw times rent, the largest int64 when the
// product does not fit so the unit cost reads as unknown, and 0 when no rents are set.
//
// Guard: a wrapped sum would put a negative rent in the anchor and make an unaffordable
// recipe look cheap.
func TestRentTerms_SaturatesOnOverflowAndZeroWithNoRents(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	w.p12Anchors() // loads the anchor recipes into the scratch
	s := w.prod
	s.rents = []int64{2}
	s.rentTerms(0)
	if got := s.anchor[0].Rent; got != 6 {
		t.Errorf("rentTerms rent = %d, want 6 (3 units at 2)", got)
	}
	s.rents = []int64{math.MaxInt64 / 2}
	s.rentTerms(0)
	if got := s.anchor[0].Rent; got != math.MaxInt64 {
		t.Errorf("rentTerms rent with an overflowing product = %d, want MaxInt64", got)
	}
	s.rents = nil
	s.rentTerms(0)
	if got := s.anchor[0].Rent; got != 0 {
		t.Errorf("rentTerms rent with no rents = %d, want 0", got)
	}
}
